package repositories

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	Path "path/filepath"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"slices"
	"strings"
	"time"

	"saymow/version-manager/app/repositories/directory"
	"saymow/version-manager/app/repositories/filesystem"

	"github.com/golang-collections/collections/set"
)

type Repository struct {
	fs    *filesystem.FileSystem
	refs  *filesystem.Refs
	head  string
	index []*directory.Change
	dir   directory.Dir
}

type Log struct {
	Checkpoint *filesystem.Checkpoint
}

type Status struct {
	Staged struct {
		CreatedFilesPaths []string
		ModifiedFilePaths []string
		RemovedFilePaths  []string
	}
	WorkingDir struct {
		ModifiedFilePaths  []string
		UntrackedFilePaths []string
		RemovedFilePaths   []string
	}
}

type ValidationError struct {
	message string
}

func CreateRepository(root string) *Repository {
	fileSystem := filesystem.Create(root)

	return &Repository{
		fs:    fileSystem,
		refs:  &filesystem.Refs{filesystem.INITAL_REF_NAME: ""},
		head:  filesystem.INITAL_REF_NAME,
		index: []*directory.Change{},
		dir:   directory.Dir{Path: root, Children: make(map[string]*directory.Node)},
	}
}

func GetRepository(root string) *Repository {
	repository := &Repository{}

	repository.fs = filesystem.Open(root)
	repository.index = repository.fs.ReadIndex()
	repository.refs = repository.fs.ReadRefs()
	repository.head = repository.fs.ReadHead()
	repository.dir = repository.fs.ReadDir(repository.getCurrentSaveName())

	return repository
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", err.message)
}

func (repository *Repository) getCurrentSaveName() string {
	if repository.isDetachedMode() {
		return repository.head
	}

	return (*repository.refs)[repository.head]
}

func (repository *Repository) isDetachedMode() bool {
	if _, ok := (*repository.refs)[repository.head]; ok {
		// Then head is a reference

		return false
	}
	// Otherwise, head is a saveName

	return true
}

func (repository *Repository) IndexFile(filepath string) error {
	if repository.isDetachedMode() {
		return &ValidationError{"cannot make changes in detached mode."}
	}

	filepath, err := repository.dir.AbsPath(filepath)
	if err != nil {
		return &ValidationError{err.Error()}
	}

	file, err := os.Open(filepath)
	errors.Check(err)
	defer file.Close()

	object := repository.fs.WriteObject(filepath, file)
	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedFile(filepath)
	var ChangeType directory.ChangeType

	if savedObject != nil {
		ChangeType = directory.Modification
	} else {
		ChangeType = directory.Creation
	}

	if savedObject != nil && savedObject.ObjectName == object.ObjectName {
		// No changes at all

		if stagedChangeIdx != -1 {
			if repository.index[stagedChangeIdx].ChangeType != directory.Removal {
				// Remove change file object
				repository.fs.RemoveObject(repository.index[stagedChangeIdx].File.ObjectName)
			}

			// Undo index existing change
			repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		}
	} else if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].ChangeType != directory.Removal &&
			repository.index[stagedChangeIdx].File.ObjectName != object.ObjectName {
			// Remove change file object
			repository.fs.RemoveObject(repository.index[stagedChangeIdx].File.ObjectName)
		}

		// Undo index existing change
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		// Index change
		repository.index = append(repository.index, &directory.Change{ChangeType: ChangeType, File: object})

	} else {
		// Index change
		repository.index = append(repository.index, &directory.Change{ChangeType: ChangeType, File: object})
	}

	return nil
}

func (repository *Repository) RemoveFile(filepath string) error {
	if repository.isDetachedMode() {
		return &ValidationError{"cannot make changes in detached mode."}
	}

	filepath, err := repository.dir.AbsPath(filepath)
	if err != nil {
		return &ValidationError{err.Error()}
	}

	// Remove from working dir
	err = os.Remove(filepath)
	if err != nil && !os.IsNotExist(err) {
		errors.Error(err.Error())
	}

	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedFile(filepath)

	if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].ChangeType == directory.Removal {
			// Index entry is already meant for removal
			return nil
		}

		// Remove existing change from the index
		repository.fs.RemoveObject(repository.index[stagedChangeIdx].File.ObjectName)
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
	}

	if savedObject != nil {
		// Create Index file removal entry
		repository.index = append(repository.index, &directory.Change{ChangeType: directory.Removal, Removal: &directory.FileRemoval{Filepath: filepath}})
	}

	return nil
}

func (repository *Repository) SaveIndex() error {
	if repository.isDetachedMode() {
		return &ValidationError{"cannot make changes in detached mode."}
	}

	repository.fs.SaveIndex(repository.index)

	return nil
}

func (repository *Repository) clearIndex() {
	repository.index = []*directory.Change{}
	repository.fs.SaveIndex(repository.index)
}

func (repository *Repository) setRef(name, saveName string) {
	(*repository.refs)[name] = saveName
	repository.fs.WriteRefs(repository.refs)
}

func (repository *Repository) SetHead(newHead string) {
	repository.head = newHead
	repository.fs.WriteHead(repository.head)
}

func (repository *Repository) CreateSave(message string) (*filesystem.Checkpoint, error) {
	if repository.isDetachedMode() {
		return nil, &ValidationError{"cannot make changes in detached mode."}
	}
	if len(repository.index) == 0 {
		return nil, &ValidationError{"cannot save empty index."}
	}

	save := filesystem.Checkpoint{
		Message:   message,
		Parent:    repository.getCurrentSaveName(),
		Changes:   repository.index,
		CreatedAt: time.Now(),
	}

	save.Id = repository.fs.WriteSave(&save)
	repository.clearIndex()
	repository.setRef(repository.head, save.Id)

	return &save, nil
}

func (repository *Repository) findStagedChangeIdx(filepath string) int {
	return collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
		return change.GetPath() == filepath
	})
}

func (repository *Repository) findStagedChange(filepath string) *directory.Change {
	idx := repository.findStagedChangeIdx(filepath)
	if idx == -1 {
		return nil
	}

	return repository.index[idx]
}

func (repository *Repository) findSavedFile(filepath string) *directory.File {
	normalizedPath, err := repository.dir.NormalizePath(filepath)
	errors.Check(err)

	node := repository.dir.FindNode(normalizedPath)
	if node == nil || node.NodeType != directory.FileType {
		return nil
	}

	return node.File
}

func (repository *Repository) GetStatus() *Status {
	status := Status{}
	seenPaths := set.New()
	trackedPaths := set.New()

	for _, file := range repository.dir.CollectAllFiles() {
		trackedPaths.Insert(file.Filepath)
	}

	for _, change := range repository.index {
		trackedPaths.Insert(change.GetPath())
	}

	Path.Walk(repository.fs.Root, func(filepath string, info fs.FileInfo, err error) error {
		errors.Check(err)
		if repository.fs.Root == filepath || strings.HasPrefix(filepath, Path.Join(repository.fs.Root, filesystem.REPOSITORY_FOLDER_NAME)) {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		seenPaths.Insert(filepath)

		savedFile := repository.findSavedFile(filepath)
		stagedChange := repository.findStagedChange(filepath)

		if savedFile == nil && stagedChange == nil {
			status.WorkingDir.UntrackedFilePaths = append(status.WorkingDir.UntrackedFilePaths, filepath)
			return nil
		}

		file, err := os.Open(filepath)
		errors.Check(err)
		defer file.Close()

		var buffer bytes.Buffer
		chunkBuffer := make([]byte, 1024)

		for {
			n, err := file.Read(chunkBuffer)

			if err != nil && err != io.EOF {
				errors.Error(err.Error())
			}
			if n == 0 {
				break
			}

			_, err = buffer.Write(chunkBuffer[:n])
			errors.Check(err)
		}

		hasher := sha256.New()
		_, err = hasher.Write(buffer.Bytes())
		errors.Check(err)
		fileHash := hex.EncodeToString(hasher.Sum(nil))

		if stagedChange != nil {
			if stagedChange.ChangeType == directory.Removal {
				status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, stagedChange.Removal.Filepath)
			} else {
				if savedFile == nil {
					status.Staged.CreatedFilesPaths = append(status.Staged.CreatedFilesPaths, filepath)
				} else {
					status.Staged.ModifiedFilePaths = append(status.Staged.ModifiedFilePaths, filepath)
				}

				if stagedChange.File.ObjectName != fileHash {
					status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
				}
			}
		} else {
			if savedFile.ObjectName != fileHash {
				status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
			}
		}

		return nil
	})

	trackedPaths.Difference(seenPaths).Do(func(i interface{}) {
		filepath := i.(string)

		if repository.findStagedChange(filepath) != nil {
			status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, filepath)
		} else {
			status.WorkingDir.RemovedFilePaths = append(status.WorkingDir.RemovedFilePaths, filepath)
		}
	})

	return &status
}

func (repository *Repository) getSave(ref string) *filesystem.Save {
	if ref == "" {
		return nil
	}
	if ref == filesystem.INITAL_REF_NAME {
		if saveName, ok := (*repository.refs)[ref]; ok && saveName == "" {
			// This is expect to only happen for repositories with not saves

			return nil
		}
	}
	if ref == "HEAD" {
		ref = repository.head
	}

	var checkpointId string

	if saveName, ok := (*repository.refs)[ref]; ok {
		checkpointId = saveName
	} else {
		checkpointId = ref
	}

	return repository.fs.ReadSave(checkpointId)
}

func buildDirFromSave(root string, save *filesystem.Save) *directory.Dir {
	dir := &directory.Dir{Path: root, Children: map[string]*directory.Node{}}

	for _, checkpoint := range save.Checkpoints {
		for _, change := range checkpoint.Changes {
			normalizedPath, err := dir.NormalizePath(change.GetPath())
			errors.Check(err)

			dir.AddNode(normalizedPath, change)
		}
	}

	return dir
}

func (repository *Repository) resolvePath(path string) (string, *ValidationError) {
	normalizedPath, err := repository.dir.NormalizePath(path)

	if err != nil {
		return "", &ValidationError{err.Error()}
	}

	return normalizedPath, nil
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
//   - When applied to directory, Restore will remove all existing changes in the directory (forever) and
//     restore the Save or HEAD + index.
//   - You can use Restore to recover a deleted file from the index or from a Save.
//   - The HEAD is not changed during Restore.
func (repository *Repository) Restore(ref string, path string) *ValidationError {
	resolvedPath, err := repository.resolvePath(path)
	if err != nil {
		return err
	}

	save := repository.getSave(ref)
	if save == nil {
		return &ValidationError{fmt.Sprintf("\"%s\" is an invalid ref.", ref)}
	}

	node := buildDirFromSave(repository.fs.Root, save).FindNode(resolvedPath)
	if node == nil {
		return &ValidationError{fmt.Sprintf("\"%s\" is a invalid path.", path)}
	}

	filesRemovedFromIndex := []*directory.File{}

	if node.NodeType == directory.FileType {
		// Check if there is a index modification for the node
		stagedChangeIdx := repository.findStagedChangeIdx(node.File.Filepath)

		if stagedChangeIdx != -1 {
			// If so, remove the change from the index

			if repository.index[stagedChangeIdx].ChangeType != directory.Removal {
				// If it is a file modification, should also remove its object

				if ref == "HEAD" {
					// If restoring to HEAD, index is a priority

					node = &directory.Node{NodeType: directory.FileType, File: repository.index[stagedChangeIdx].File}
				}

				filesRemovedFromIndex = append(filesRemovedFromIndex, repository.index[stagedChangeIdx].File)
			}

			repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		}
	} else {
		// For directories we should iterate over the index and rebuild the node dir, if necessary,
		// applying the index priority.

		repository.index = collections.Filter(repository.index, func(change *directory.Change, _ int) bool {
			filepath := change.GetPath()

			if !strings.HasPrefix(filepath, node.Dir.Path) {
				// If index change is in other directory, keep file

				return true
			}

			if change.ChangeType != directory.Removal {
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

	if node.NodeType == directory.DirType {
		nodes := node.Dir.PreOrderTraversal()

		if node.Dir.Path == repository.fs.Root {
			// if we are traversing the root dir, the root-dir-file should be removed from the response.

			nodes = nodes[1:]
		}

		repository.fs.SafeRemoveDir(node.Dir)

		for _, node := range nodes {
			repository.fs.CreateNode(node)
		}
	} else {
		repository.fs.CreateNode(node)
	}

	for _, fileRemoved := range filesRemovedFromIndex {
		repository.fs.RemoveObject(fileRemoved.ObjectName)
	}

	repository.SaveIndex()

	return nil
}

func (repository *Repository) GetLogs() []*Log {
	save := repository.getSave(repository.head)

	if save == nil {
		return []*Log{}
	}

	// By default the save checkpoints is ordered by createdAt in ascending order.
	// The other way around is better for logging.
	slices.Reverse(save.Checkpoints)

	return collections.Map(save.Checkpoints, func(checkpoint *filesystem.Checkpoint, _ int) *Log {
		return &Log{Checkpoint: checkpoint}
	})
}

func (repository *Repository) GetRefs() filesystem.Refs {
	return *repository.refs
}

func (repository *Repository) CreateRef(name string) *ValidationError {
	currentSaveName := repository.getCurrentSaveName()

	if currentSaveName == "" {
		return &ValidationError{"cannot create refs when there is no save history."}
	}
	if saveName, found := (*repository.refs)[name]; found && saveName != currentSaveName {
		return &ValidationError{"name already in use."}
	}

	repository.setRef(name, repository.getCurrentSaveName())
	repository.SetHead(name)
	return nil
}
