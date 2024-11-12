package handlers

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	fp "path/filepath"
	"slices"
	"strings"
)

type Object struct {
	filepath string
	name     string
}

type Repository struct {
	rootDir      string
	indexObjects []*Object
}

const (
	REPOSITORY_FOLDER_NAME = "repository"
	OBJECTS_FOLDER_NAME    = "objects"
	INDEX_FILE_NAME        = "index"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func FindIndex[T any](arr []T, callback func(T, int) bool) int {
	for idx, element := range arr {
		if callback(element, idx) {
			return idx
		}
	}

	return -1
}

func createRepository(dir string) *Repository {
	err := os.Mkdir(fp.Join(dir, REPOSITORY_FOLDER_NAME), 0755)
	check(err)

	stageFile, err := os.Create(fp.Join(dir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
	check(err)

	_, err = stageFile.Write([]byte("Tracked files:\r\n\r\n"))
	check(err)

	check(err)

	err = os.Mkdir(fp.Join(dir, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME), 0755)
	check(err)

	return &Repository{
		rootDir:      dir,
		indexObjects: []*Object{},
	}
}

func getRepository(dir string) *Repository {
	stageFile, err := os.OpenFile(path.Join(dir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_RDONLY, 0755)
	check(err)

	var stageObjects []*Object
	scanner := bufio.NewScanner(stageFile)

	// Skip file header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		object := Object{}

		object.filepath = scanner.Text()
		scanner.Scan()
		object.name = scanner.Text()

		stageObjects = append(stageObjects, &object)
	}

	return &Repository{
		rootDir:      dir,
		indexObjects: stageObjects,
	}
}

func (repository *Repository) writeObject(filepath string, file *os.File) Object {
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
		check(err)
	}

	hasher := sha256.New()
	_, err := hasher.Write(buffer.Bytes())
	check(err)
	hash := hasher.Sum(nil)

	objectName := hex.EncodeToString(hash)
	objectFile, err := os.Create(path.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, objectName))
	check(err)
	defer objectFile.Close()

	gzipWriter := gzip.NewWriter(objectFile)
	_, err = gzipWriter.Write(buffer.Bytes())
	check(err)

	return Object{filepath, objectName}
}

func (repository *Repository) removeObject(name string) {
	err := os.Remove(fp.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, name))
	check(err)
}

func (repository *Repository) IndexFile(filepath string) {
	if !fp.IsAbs(filepath) {
		filepath = fp.Join(repository.rootDir, filepath)
	}
	if !strings.HasPrefix(filepath, repository.rootDir) {
		log.Fatal("Invalid file path.")
	}

	file, err := os.Open(filepath)
	check(err)
	defer file.Close()

	object := repository.writeObject(filepath, file)
	stageObjectIdx := FindIndex(repository.indexObjects, func(stageObject *Object, _ int) bool {
		return stageObject.filepath == filepath
	})

	if stageObjectIdx != -1 {
		// Update existing stage object name
		if repository.indexObjects[stageObjectIdx].name != object.name {
			repository.removeObject(repository.indexObjects[stageObjectIdx].name)
			repository.indexObjects[stageObjectIdx].name = object.name
		}
	} else {
		// Create stage object
		repository.indexObjects = append(repository.indexObjects, &object)
	}
}

func (repository *Repository) RemoveFileIndex(filepath string) {
	if !fp.IsAbs(filepath) {
		filepath = fp.Join(repository.rootDir, filepath)
	}
	if !strings.HasPrefix(filepath, repository.rootDir) {
		log.Fatal("Invalid file path.")
	}

	objectIdx := FindIndex(repository.indexObjects, func(object *Object, _ int) bool {
		return object.filepath == filepath
	})

	if objectIdx == -1 {
		return
	}

	repository.removeObject(repository.indexObjects[objectIdx].name)
	repository.indexObjects = slices.Delete(repository.indexObjects, objectIdx, objectIdx+1)
}

func (repository *Repository) SaveIndex() {
	file, err := os.OpenFile(fp.Join(repository.rootDir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	check(err)

	_, err = file.Write([]byte("Tracked files:\r\n\r\n"))
	check(err)

	for _, object := range repository.indexObjects {
		_, err = file.Write([]byte(fmt.Sprintf("%s\r\n%s\r\n", object.filepath, object.name)))
		check(err)
	}
}
