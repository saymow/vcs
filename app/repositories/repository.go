package repositories

import (
	"fmt"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"

	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
)

type Repository struct {
	fs    *filesystems.FileSystem
	refs  *filesystems.Refs
	head  string
	index []*directories.Change
	dir   directories.Dir
}

type SaveLog struct {
	Refs       []string
	Checkpoint *filesystems.Checkpoint
}

type Log struct {
	Head    string
	History []*SaveLog
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
	Message string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", err.Message)
}

func CreateRepository(root string) *Repository {
	fileSystem := filesystems.Create(root)

	return &Repository{
		fs:    fileSystem,
		refs:  &filesystems.Refs{filesystems.INITAL_REF_NAME: ""},
		head:  filesystems.INITAL_REF_NAME,
		index: []*directories.Change{},
		dir:   directories.Dir{Path: root, Children: make(map[string]*directories.Node)},
	}
}

func GetRepository(root string) *Repository {
	repository := &Repository{}

	repository.fs = filesystems.Open(root)
	repository.index = repository.fs.ReadIndex()
	repository.refs = repository.fs.ReadRefs()
	repository.head = repository.fs.ReadHead()
	repository.dir = repository.fs.ReadDir(repository.getCurrentSaveName())

	return repository
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

func (repository *Repository) clearIndex() {
	repository.index = []*directories.Change{}
	repository.fs.SaveIndex(repository.index)
}

func (repository *Repository) setRef(name, saveName string) {
	(*repository.refs)[name] = saveName
	repository.fs.WriteRefs(repository.refs)
}

func (repository *Repository) setHead(newHead string) {
	repository.head = newHead
	repository.fs.WriteHead(repository.head)
}

func (repository *Repository) findStagedChangeIdx(filepath string) int {
	return collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
		return change.GetPath() == filepath
	})
}

func (repository *Repository) findStagedChange(filepath string) *directories.Change {
	idx := repository.findStagedChangeIdx(filepath)
	if idx == -1 {
		return nil
	}

	return repository.index[idx]
}

func (repository *Repository) findSavedFile(filepath string) *directories.File {
	normalizedPath, err := repository.dir.NormalizePath(filepath)
	errors.Check(err)

	node := repository.dir.FindNode(normalizedPath)
	if node == nil || node.NodeType != directories.FileType {
		return nil
	}

	return node.File
}

func (repository *Repository) getSave(ref string) *filesystems.Save {
	if ref == "" {
		return nil
	}
	if ref == filesystems.INITAL_REF_NAME {
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

func buildDirFromSave(root string, save *filesystems.Save) *directories.Dir {
	dir := &directories.Dir{Path: root, Children: map[string]*directories.Node{}}

	for _, checkpoint := range save.Checkpoints {
		for _, change := range checkpoint.Changes {
			normalizedPath, err := dir.NormalizePath(change.GetPath())
			errors.Check(err)

			dir.AddNode(normalizedPath, change)
		}
	}

	return dir
}

func (repository *Repository) resolvePath(path string) (string, error) {
	normalizedPath, err := repository.dir.NormalizePath(path)

	if err != nil {
		return "", &ValidationError{err.Error()}
	}

	return normalizedPath, nil
}
