package repositories

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	path "path/filepath"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"slices"
	"strings"
	"time"

	"github.com/golang-collections/collections/set"
)

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

func (repository *Repository) writeObject(filepath string, file *os.File) *File {
	var buffer bytes.Buffer
	chunkBuffer := make([]byte, 1024)

	for {
		n, err := file.Read(chunkBuffer)

		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		if n == 0 {
			break
		}

		_, err = buffer.Write(chunkBuffer[:n])
		errors.Check(err)
	}

	hasher := sha256.New()
	_, err := hasher.Write(buffer.Bytes())
	errors.Check(err)
	hash := hasher.Sum(nil)

	objectName := hex.EncodeToString(hash)
	objectFile, err := os.Create(path.Join(repository.root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, objectName))
	errors.Check(err)
	defer objectFile.Close()

	compressor := gzip.NewWriter(objectFile)
	_, err = compressor.Write(buffer.Bytes())
	errors.Check(err)
	compressor.Close()

	return &File{filepath, objectName}
}

func (repository *Repository) removeObject(name string) {
	err := os.Remove(path.Join(repository.root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, name))
	errors.Check(err)
}

func (repository *Repository) IndexFile(filepath string) {
	if !path.IsAbs(filepath) {
		filepath = path.Join(repository.root, filepath)
	}
	if !strings.HasPrefix(filepath, repository.root) {
		log.Fatal("Invalid file path.")
	}

	file, err := os.Open(filepath)
	errors.Check(err)
	defer file.Close()

	object := repository.writeObject(filepath, file)
	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedObject(filepath)

	if savedObject != nil && savedObject.objectName == object.objectName {
		// No changes at all

		if stagedChangeIdx != -1 {
			if repository.index[stagedChangeIdx].changeType == Modified {
				// Remove change file object
				repository.removeObject(repository.index[stagedChangeIdx].modified.objectName)
			}

			// Undo index existing change
			repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		}
	} else if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].changeType == Modified &&
			repository.index[stagedChangeIdx].modified.objectName != object.objectName {
			// Remove change file object
			repository.removeObject(repository.index[stagedChangeIdx].modified.objectName)
		}

		// Undo index existing change
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		// Index change
		repository.index = append(repository.index, &Change{changeType: Modified, modified: object})

	} else {
		// Index change
		repository.index = append(repository.index, &Change{changeType: Modified, modified: object})
	}
}

func (repository *Repository) RemoveFile(filepath string) {
	if !path.IsAbs(filepath) {
		filepath = path.Join(repository.root, filepath)
	}
	if !strings.HasPrefix(filepath, repository.root) {
		log.Fatal("Invalid file path.")
	}

	// Remove from working dir
	err := os.Remove(filepath)
	if err != nil && !os.IsNotExist(err) {
		errors.Error(err.Error())
	}

	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedObject(filepath)

	if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].changeType == Removal {
			// Index entry is already meant for removal
			return
		}

		// Remove existing change from the index
		repository.removeObject(repository.index[stagedChangeIdx].modified.objectName)
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
	}

	if savedObject != nil {
		// Create Index file removal entry
		repository.index = append(repository.index, &Change{changeType: Removal, removal: &FileRemoval{filepath}})
	}
}

func (repository *Repository) SaveIndex() {
	file, err := os.OpenFile(path.Join(repository.root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte("Tracked files:\n\n"))
	errors.Check(err)

	for _, change := range repository.index {
		if change.changeType == Modified {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.modified.filepath, MODIFIED_CHANGE, change.modified.objectName)))
		} else {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n", change.removal.filepath, REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}
}

func (repository *Repository) CreateSave(message string) *Save {
	if len(repository.index) == 0 {
		errors.Error("Cannot save empty index.")
	}

	save := Save{
		message:   message,
		parent:    repository.head,
		changes:   repository.index,
		createdAt: time.Now(),
	}

	save.id = repository.writeSave(&save)
	repository.clearIndex()
	repository.writeHead(save.id)

	return &save
}

func (repository *Repository) writeSave(save *Save) string {
	var stringBuilder strings.Builder

	_, err := stringBuilder.Write([]byte(fmt.Sprintf("%s\n", save.message)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\n", save.parent)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\n\n", save.createdAt.Format(time.Layout))))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte("Please do not edit the lines below.\n\n\nFiles:\n\n"))
	errors.Check(err)

	for _, change := range save.changes {
		if change.changeType == Modified {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.modified.filepath, MODIFIED_CHANGE, change.modified.objectName)))
		} else {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n", change.removal.filepath, REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}

	saveContent := stringBuilder.String()

	hasher := sha256.New()
	_, err = hasher.Write([]byte(saveContent))
	errors.Check(err)
	hash := hasher.Sum(nil)

	saveName := hex.EncodeToString(hash)

	file, err := os.Create(path.Join(repository.root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
	errors.Check(err)
	defer file.Close()

	_, err = file.Write([]byte(saveContent))
	errors.Check(err)

	return saveName
}

func (repository *Repository) clearIndex() {
	repository.index = []*Change{}
	repository.SaveIndex()
}

func (repository *Repository) writeHead(name string) {
	file, err := os.OpenFile(path.Join(repository.root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte(name))
	errors.Check(err)
}

func (repository *Repository) findStagedChangeIdx(filepath string) int {
	return collections.FindIndex(repository.index, func(item *Change, _ int) bool {
		if item.changeType == Modified {
			return item.modified.filepath == filepath
		}

		return item.removal.filepath == filepath
	})
}

func (repository *Repository) findStagedChange(filepath string) *Change {
	idx := repository.findStagedChangeIdx(filepath)

	if idx == -1 {
		return nil
	}

	return repository.index[idx]
}

func (repository *Repository) findSavedObject(filepath string) *File {
	normalizedPath := filepath[len(repository.root)+1:]
	return repository.dir.findFile(normalizedPath)
}

func (repository *Repository) GetStatus() *Status {
	status := Status{}
	seenPaths := set.New()
	trackedPaths := set.New()

	for _, file := range repository.dir.collectFiles() {
		trackedPaths.Insert(file.filepath)
	}

	for _, change := range repository.index {
		if change.changeType == Modified {
			trackedPaths.Insert(change.modified.filepath)
		} else {
			trackedPaths.Insert(change.removal.filepath)
		}
	}

	path.Walk(repository.root, func(filepath string, info fs.FileInfo, err error) error {
		errors.Check(err)
		if repository.root == filepath || strings.HasPrefix(filepath, path.Join(repository.root, REPOSITORY_FOLDER_NAME)) {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		seenPaths.Insert(filepath)

		savedObject := repository.findSavedObject(filepath)
		stagedChange := repository.findStagedChange(filepath)

		if savedObject == nil && stagedChange == nil {
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
			if stagedChange.changeType == Modified {
				if savedObject == nil {
					status.Staged.CreatedFilesPaths = append(status.Staged.CreatedFilesPaths, filepath)
				} else {
					status.Staged.ModifiedFilePaths = append(status.Staged.ModifiedFilePaths, filepath)
				}

				if stagedChange.modified.objectName != fileHash {
					status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
				}
			} else {
				status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, stagedChange.removal.filepath)
			}
		} else {
			if savedObject.objectName != fileHash {
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
