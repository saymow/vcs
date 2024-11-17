package repositories

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	head  string
	index []*directory.Change
	dir   directory.Dir
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
		index: []*directory.Change{},
		dir:   directory.Dir{},
	}
}

func GetRepository(root string) *Repository {
	fileSystem := filesystem.Open(root)
	index := fileSystem.ReadIndex()
	head := fileSystem.ReadHead()
	dir := fileSystem.ReadDir(head)

	return &Repository{
		fs:    fileSystem,
		index: index,
		head:  head,
		dir:   dir,
	}
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", err.message)
}

func (repository *Repository) IndexFile(filepath string) {
	if !Path.IsAbs(filepath) {
		filepath = Path.Join(repository.fs.Root, filepath)
	}
	if !strings.HasPrefix(filepath, repository.fs.Root) {
		log.Fatal("Invalid file path.")
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
}

func (repository *Repository) RemoveFile(filepath string) {
	if !Path.IsAbs(filepath) {
		filepath = Path.Join(repository.fs.Root, filepath)
	}
	if !strings.HasPrefix(filepath, repository.fs.Root) {
		log.Fatal("Invalid file path.")
	}

	// Remove from working dir
	err := os.Remove(filepath)
	if err != nil && !os.IsNotExist(err) {
		errors.Error(err.Error())
	}

	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedFile(filepath)

	if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].ChangeType == directory.Removal {
			// Index entry is already meant for removal
			return
		}

		// Remove existing change from the index
		repository.fs.RemoveObject(repository.index[stagedChangeIdx].File.ObjectName)
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
	}

	if savedObject != nil {
		// Create Index file removal entry
		repository.index = append(repository.index, &directory.Change{ChangeType: directory.Removal, Removal: &directory.FileRemoval{Filepath: filepath}})
	}
}

func (repository *Repository) SaveIndex() {
	repository.fs.SaveIndex(repository.index)
}

func (repository *Repository) CreateSave(message string) *filesystem.CheckPoint {
	if len(repository.index) == 0 {
		errors.Error("Cannot save empty index.")
	}

	save := filesystem.CheckPoint{
		Message:   message,
		Parent:    repository.head,
		Changes:   repository.index,
		CreatedAt: time.Now(),
	}

	save.Id = repository.fs.WriteSave(&save)
	repository.clearIndex()
	repository.fs.WriteHead(save.Id)

	return &save
}

func (repository *Repository) clearIndex() {
	repository.index = []*directory.Change{}
	repository.fs.SaveIndex(repository.index)
}

func (repository *Repository) findStagedChangeIdx(filepath string) int {
	return collections.FindIndex(repository.index, func(item *directory.Change, _ int) bool {
		if item.ChangeType == directory.Removal {
			return item.Removal.Filepath == filepath
		}

		return item.File.Filepath == filepath
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
	normalizedPath := filepath[len(repository.fs.Root)+1:]
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
		if change.ChangeType == directory.Removal {
			trackedPaths.Insert(change.Removal.Filepath)
		} else {
			trackedPaths.Insert(change.File.Filepath)
		}
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
	var checkpointId string

	if ref == "HEAD" {
		checkpointId = repository.head
	} else {
		checkpointId = ref
	}

	if ref == "" {
		return nil
	}

	return repository.fs.ReadSave(checkpointId)
}

func buildDirFromSave(root string, save *filesystem.Save) *directory.Dir {
	dir := &directory.Dir{Path: root, Children: map[string]*directory.Node{}}

	for _, checkpoint := range save.Checkpoints {
		for _, change := range checkpoint.Changes {
			var normalizedPath string

			if change.ChangeType == directory.Removal {
				normalizedPath = change.Removal.Filepath[len(root)+1:]
			} else {
				normalizedPath = change.File.Filepath[len(root)+1:]
			}

			dir.AddNode(normalizedPath, change)
		}
	}

	return dir
}

// Load cover 2 usecases:
//
//  1. Restore HEAD + index changes (...and remove the index change).
//
//     It can be used to restore the current head + index changes. Index changes
//     have higher priorities.
//
//  2. Load all the content saved up until all checkpoints created.
func (repository *Repository) Load(ref string, path string) error {
	save := repository.getSave(ref)
	if save == nil {
		return &ValidationError{fmt.Sprintf("\"%s\" is an invalid ref.", ref)}
	}

	rootDir := buildDirFromSave(repository.fs.Root, save)

	node := rootDir.FindNode(path)
	if node == nil {
		return &ValidationError{fmt.Sprintf("\"%s\" is a invalid path.", ref)}
	}

	filesRemovedFromIndex := []*directory.File{}

	if ref == "HEAD" {
		// Index files have higher priority over tree files to be restored

		if node.NodeType == directory.FileType {
			// Check if there is a index modification for the node

			stagedChangeIdx := repository.findStagedChangeIdx(node.File.Filepath)

			if stagedChangeIdx != -1 {
				// If so, remove the change from the index

				if repository.index[stagedChangeIdx].ChangeType != directory.Removal {
					// If it is a file modification, the index modification is applied instead of the tree one.

					node = &directory.Node{NodeType: directory.FileType, File: repository.index[stagedChangeIdx].File}
					filesRemovedFromIndex = append(filesRemovedFromIndex, repository.index[stagedChangeIdx].File)
				}

				repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
			}
		} else {
			// For directories we should iterate over the index and rebuild the node dir, if necessary,
			// applying the index priority.

			for _, change := range slices.Clone(repository.index) {
				var filepath string

				if change.ChangeType == directory.Removal {
					filepath = change.Removal.Filepath
				} else {
					filepath = change.File.Filepath
				}

				if !strings.HasPrefix(filepath, node.Dir.Path) {
					continue
				}

				if change.ChangeType != directory.Removal {
					normalizedPath := filepath[len(node.Dir.Path)+1:]
					node.Dir.AddNode(normalizedPath, change)

					filesRemovedFromIndex = append(filesRemovedFromIndex, change.File)
				}

				repository.index = collections.Remove(repository.index, func(item *directory.Change, _ int) bool {
					return item == change
				})
			}
		}
	}

	if node.NodeType == directory.DirType {
		nodes := node.Dir.PreOrderTraversal()

		if node.Dir == rootDir {
			// if we are traversing the root dir, the root-dir-file is included in the response.

			nodes = nodes[1:]
		}

		repository.fs.SafeRemoveDir(node.Dir)

		for _, node := range nodes {
			repository.fs.ApplyNode(node)
		}
	} else {
		repository.fs.ApplyNode(node)
	}

	for _, fileRemoved := range filesRemovedFromIndex {
		repository.fs.RemoveObject(fileRemoved.ObjectName)
	}

	return nil
}
