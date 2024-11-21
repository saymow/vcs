package repositories

import (
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/directories"
	"slices"
	"strings"
)

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
//   - When applied to directory, Restore will remove all existing changes in the directory (forever) and
//     restore the Save or HEAD + index.
//   - You can use Restore to recover a deleted file from the index or from a Save.
//   - The HEAD is not changed during Restore.
func (repository *Repository) Restore(ref string, path string) error {
	resolvedPath, err := repository.resolvePath(path)
	if err != nil {
		return err
	}

	save := repository.getSave(ref)
	if save == nil {
		return &ValidationError{"invalid ref."}
	}

	node := buildDir(repository.fs.Root, save).FindNode(resolvedPath)
	if node == nil {
		return &ValidationError{"invalid path."}
	}

	filesRemovedFromIndex := []*directories.File{}

	if node.NodeType == directories.FileType {
		// Check if there is a index modification for the node
		stagedChangeIdx := repository.findStagedChangeIdx(node.File.Filepath)

		if stagedChangeIdx != -1 {
			// If so, remove the change from the index

			if repository.index[stagedChangeIdx].ChangeType != directories.Removal {
				// If it is a file modification, should also remove its object

				if ref == "HEAD" {
					// If restoring to HEAD, index is a priority

					node = &directories.Node{NodeType: directories.FileType, File: repository.index[stagedChangeIdx].File}
				}

				filesRemovedFromIndex = append(filesRemovedFromIndex, repository.index[stagedChangeIdx].File)
			}

			repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		}
	} else {
		// For directories we should iterate over the index and rebuild the node dir, if necessary,
		// applying the index priority.

		repository.index = collections.Filter(repository.index, func(change *directories.Change, _ int) bool {
			filepath := change.GetPath()

			if !strings.HasPrefix(filepath, node.Dir.Path) {
				// If index change is in other directory, keep file

				return true
			}

			if change.ChangeType != directories.Removal {
				// If it is a file modification, should also remove its object

				if ref == "HEAD" {
					// If restoring to HEAD, index is a priority

					normalizedPath, err := node.Dir.NormalizePath(filepath)
					errors.Check(err)

					node.Dir.AddNode(normalizedPath, change)
				}

				filesRemovedFromIndex = append(filesRemovedFromIndex, change.File)
			}

			return false
		})
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
