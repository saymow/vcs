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
)

type RepositoryStatus struct {
	Staged struct {
		CreatedFilesPaths []string
		ModifiedFilePaths []string
	}
	WorkingDir struct {
		ModifiedFilePaths  []string
		UntrackedFilePaths []string
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

	gzipWriter := gzip.NewWriter(objectFile)
	_, err = gzipWriter.Write(buffer.Bytes())
	errors.Check(err)

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
	stagedObject := repository.findStagedObject(filepath)
	savedObject := repository.findSavedObject(filepath)

	if savedObject != nil && savedObject.objectName == object.objectName {
		// No changes at all
		if stagedObject != nil {
			// Undo index changes if any
			repository.RemoveFile(filepath)
		}
	} else if stagedObject != nil {
		if stagedObject.objectName != object.objectName {
			// Update stage object if an update really exists
			repository.removeObject(stagedObject.objectName)
			stagedObject.objectName = object.objectName
		}
	} else {
		// Create stage object
		repository.index = append(repository.index, object)
	}
}

func (repository *Repository) RemoveFile(filepath string) {
	if !path.IsAbs(filepath) {
		filepath = path.Join(repository.root, filepath)
	}
	if !strings.HasPrefix(filepath, repository.root) {
		log.Fatal("Invalid file path.")
	}

	stagedObjectIdx := collections.FindIndex(repository.index, func(object *File, _ int) bool {
		return object.filepath == filepath
	})
	savedObject := repository.findSavedObject(filepath)

	if stagedObjectIdx == -1 && savedObject == nil {
		// Nothing to be removed
		return
	}

	if stagedObjectIdx != -1 {
		if repository.index[stagedObjectIdx].objectName == "" {
			// Index entry is already meant for removal
			return
		}

		// Remove from the index
		repository.removeObject(repository.index[stagedObjectIdx].objectName)
		repository.index = slices.Delete(repository.index, stagedObjectIdx, stagedObjectIdx+1)
	}
	if savedObject != nil {
		// Create Index file removal entry
		repository.index = append(repository.index, &File{objectName: "", filepath: filepath})
	}
}

func (repository *Repository) SaveIndex() {
	file, err := os.OpenFile(path.Join(repository.root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte("Tracked files:\r\n\r\n"))
	errors.Check(err)

	for _, object := range repository.index {
		_, err = file.Write([]byte(fmt.Sprintf("%s\r\n%s\r\n", object.filepath, object.objectName)))
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
		files:     repository.index,
		createdAt: time.Now(),
	}

	name := repository.writeSave(&save)
	repository.clearIndex()
	repository.writeHead(name)

	return &save
}

func (repository *Repository) writeSave(save *Save) string {
	var stringBuilder strings.Builder

	_, err := stringBuilder.Write([]byte(fmt.Sprintf("%s\r\n", save.message)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\r\n", save.parent)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\r\n\r\n", save.createdAt.Format(time.Layout))))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte("Please do not edit the lines below.\r\n\r\nFiles:\r\n"))
	errors.Check(err)

	for _, object := range save.files {
		_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\r\n%s\r\n", object.filepath, object.objectName)))
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
	repository.index = []*File{}
	repository.SaveIndex()
}

func (repository *Repository) writeHead(name string) {
	file, err := os.OpenFile(path.Join(repository.root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte(name))
	errors.Check(err)
}

func (repository *Repository) findStagedObject(filepath string) *File {
	objectIdx := collections.FindIndex(repository.index, func(item *File, _ int) bool {
		return item.filepath == filepath
	})

	if objectIdx == -1 {
		return nil
	}

	return repository.index[objectIdx]
}

func (repository *Repository) findSavedObject(filepath string) *File {
	normalizedPath := filepath[len(repository.root)+1:]
	return repository.dir.findFile(normalizedPath)
}

func (repository *Repository) GetStatus() *RepositoryStatus {
	status := RepositoryStatus{}

	path.Walk(repository.root, func(filepath string, info fs.FileInfo, err error) error {
		errors.Check(err)
		if repository.root == filepath || strings.HasPrefix(filepath, path.Join(repository.root, REPOSITORY_FOLDER_NAME)) {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		savedObject := repository.findSavedObject(filepath)
		stagedObject := repository.findStagedObject(filepath)

		if savedObject == nil && stagedObject == nil {
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
		hash := hex.EncodeToString(hasher.Sum(nil))

		if stagedObject != nil {
			if savedObject == nil {
				status.Staged.CreatedFilesPaths = append(status.Staged.CreatedFilesPaths, filepath)
			} else if stagedObject.objectName != savedObject.objectName {
				status.Staged.ModifiedFilePaths = append(status.Staged.ModifiedFilePaths, filepath)
			}

			if stagedObject.objectName != hash {
				status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
			}
		} else {
			if savedObject.objectName != hash {
				status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
			}
		}

		return nil
	})

	return &status
}
