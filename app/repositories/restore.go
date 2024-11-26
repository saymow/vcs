package repositories

import (
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/directories"
	"slices"
)

func (repository *Repository) getIndexDir() *directories.Node {
	dir := directories.Dir{
		Path:     repository.fs.Root,
		Children: make(map[string]*directories.Node),
	}

	for _, change := range repository.index {
		if change.ChangeType != directories.Removal {
			normalizedPath, err := dir.NormalizePath(change.GetPath())
			errors.Check(err)

			dir.AddNode(normalizedPath, change)
		}
	}

	return &directories.Node{NodeType: directories.DirType, Dir: &dir}
}

// Restore cover 2 usecases:
//
//  1. Restore HEAD + index changes (...and remove the index change).
//
//     It can be used to restore the current head + index changes. Index changes have higher priorities.
//     Initialy Restore will look for your change in the index, if found, the index change is applied. Otherwise,
//     Restore will apply the HEAD changes.
//
//  2. Restore Save
//
//     It can be used to restore existing Saves to the current working directory.
//
// Caveats:
//
//   - Restore will remove the existing changes in the path (forever) and restore reference.
//   - You can use Restore to recover a deleted file from the index or from a Save.
//   - The HEAD is not changed during Restore.
func (repository *Repository) Restore(ref string, path string) error {
	resolvedPath, err := repository.resolvePath(path)
	if err != nil {
		return err
	}

	var node *directories.Node

	if repository.hasEmptySaveHistory() {
		node = repository.getIndexDir().Dir.FindNode(resolvedPath)
	} else {
		save := repository.getSave(ref)

		if save == nil {
			return &ValidationError{"invalid ref."}
		}

		dir := buildDir(repository.fs.Root, save)

		if ref == "HEAD" {
			dir.Merge(repository.getIndexDir().Dir)
		}

		node = dir.FindNode(resolvedPath)
	}

	if node == nil {
		return &ValidationError{"invalid path."}
	}

	filesRemovedFromIndex := []*directories.File{}

	if ref == "HEAD" {
		// Should correctly cleanup applied  index changes

		if node.NodeType == directories.FileType {
			// Check if there is a index modification for the node
			stagedChangeIdx := repository.findStagedChangeIdx(node.File.Filepath)

			if stagedChangeIdx != -1 {
				// If defined, we are restoring the index change

				switch {
				// Should remove the object
				case repository.index[stagedChangeIdx].ChangeType == directories.Creation || repository.index[stagedChangeIdx].ChangeType == directories.Modification:
					filesRemovedFromIndex = append(filesRemovedFromIndex, repository.index[stagedChangeIdx].File)
				case repository.index[stagedChangeIdx].ChangeType == directories.Conflict && repository.index[stagedChangeIdx].Conflict.IsObjectTemporary():
					filesRemovedFromIndex = append(filesRemovedFromIndex, &directories.File{
						Filepath:   repository.index[stagedChangeIdx].Conflict.Filepath,
						ObjectName: repository.index[stagedChangeIdx].Conflict.ObjectName,
					})
				}

				repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
			}
		} else {
			// For directories we should iterate over the index and remove entries contained in that dir

			repository.index = collections.Filter(repository.index, func(change *directories.Change, _ int) bool {
				if !node.Dir.IsSubpath(change.GetPath()) {
					// If index change is in other directory, keep index change

					return true
				}

				// Otherwise, we are restoring the index change

				switch {
				// Should remove the object
				case change.ChangeType == directories.Creation || change.ChangeType == directories.Modification:
					filesRemovedFromIndex = append(filesRemovedFromIndex, change.File)
				case change.ChangeType == directories.Conflict && change.Conflict.IsObjectTemporary():
					filesRemovedFromIndex = append(filesRemovedFromIndex, &directories.File{
						Filepath:   change.Conflict.Filepath,
						ObjectName: change.Conflict.ObjectName,
					})
				}

				return false
			})
		}
	}

	if node.NodeType == directories.DirType {
		repository.applyDir(node.Dir)
	} else {
		repository.fs.CreateNode(node)
	}

	for _, fileRemoved := range filesRemovedFromIndex {
		repository.fs.RemoveObject(fileRemoved.ObjectName)
	}

	repository.SaveIndex()

	return nil
}
