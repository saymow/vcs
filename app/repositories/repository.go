package repositories

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	path "path/filepath"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"slices"
	"strings"
	"time"
)

func (repository *Repository) writeObject(filepath string, file *os.File) *Object {
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
	objectFile, err := os.Create(path.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, objectName))
	errors.Check(err)
	defer objectFile.Close()

	gzipWriter := gzip.NewWriter(objectFile)
	_, err = gzipWriter.Write(buffer.Bytes())
	errors.Check(err)

	return &Object{filepath, objectName}
}

func (repository *Repository) removeObject(name string) {
	err := os.Remove(path.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, name))
	errors.Check(err)
}

func (repository *Repository) IndexFile(filepath string) {
	if !path.IsAbs(filepath) {
		filepath = path.Join(repository.rootDir, filepath)
	}
	if !strings.HasPrefix(filepath, repository.rootDir) {
		log.Fatal("Invalid file path.")
	}

	file, err := os.Open(filepath)
	errors.Check(err)
	defer file.Close()

	object := repository.writeObject(filepath, file)
	stageObjectIdx := collections.FindIndex(repository.index, func(stageObject *Object, _ int) bool {
		return stageObject.filepath == filepath
	})

	if stageObjectIdx != -1 {
		// Update existing stage object name
		if repository.index[stageObjectIdx].name != object.name {
			repository.removeObject(repository.index[stageObjectIdx].name)
			repository.index[stageObjectIdx].name = object.name
		}
	} else {
		// Create stage object
		repository.index = append(repository.index, object)
	}
}

func (repository *Repository) RemoveFileIndex(filepath string) {
	if !path.IsAbs(filepath) {
		filepath = path.Join(repository.rootDir, filepath)
	}
	if !strings.HasPrefix(filepath, repository.rootDir) {
		log.Fatal("Invalid file path.")
	}

	objectIdx := collections.FindIndex(repository.index, func(object *Object, _ int) bool {
		return object.filepath == filepath
	})

	if objectIdx == -1 {
		return
	}

	repository.removeObject(repository.index[objectIdx].name)
	repository.index = slices.Delete(repository.index, objectIdx, objectIdx+1)
}

func (repository *Repository) SaveIndex() {
	file, err := os.OpenFile(path.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte("Tracked files:\r\n\r\n"))
	errors.Check(err)

	for _, object := range repository.index {
		_, err = file.Write([]byte(fmt.Sprintf("%s\r\n%s\r\n", object.filepath, object.name)))
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
		objects:   repository.index,
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

	for _, object := range save.objects {
		_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\r\n%s\r\n", object.filepath, object.name)))
		errors.Check(err)
	}

	saveContent := stringBuilder.String()

	hasher := sha256.New()
	_, err = hasher.Write([]byte(saveContent))
	errors.Check(err)
	hash := hasher.Sum(nil)

	saveName := hex.EncodeToString(hash)

	file, err := os.Create(path.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
	errors.Check(err)
	defer file.Close()

	_, err = file.Write([]byte(saveContent))
	errors.Check(err)

	return saveName
}

func (repository *Repository) clearIndex() {
	repository.index = []*Object{}
	repository.SaveIndex()
}

func (repository *Repository) writeHead(name string) {
	file, err := os.OpenFile(path.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte(name))
	errors.Check(err)
}
