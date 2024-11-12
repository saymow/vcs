package repositories

import (
	"bufio"
	"os"
	"path"
	fp "path/filepath"
	"saymow/version-manager/app/pkg/errors"
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
	REPOSITORY_FOLDER_NAME = ".repository"
	OBJECTS_FOLDER_NAME    = "objects"
	INDEX_FILE_NAME        = "index"
)

func CreateRepository(dir string) *Repository {
	err := os.Mkdir(fp.Join(dir, REPOSITORY_FOLDER_NAME), 0755)
	errors.Check(err)

	stageFile, err := os.Create(fp.Join(dir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
	errors.Check(err)

	_, err = stageFile.Write([]byte("Tracked files:\r\n\r\n"))
	errors.Check(err)

	errors.Check(err)

	err = os.Mkdir(fp.Join(dir, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME), 0755)
	errors.Check(err)

	return &Repository{
		rootDir:      dir,
		indexObjects: []*Object{},
	}
}

func GetRepository(dir string) *Repository {
	stageFile, err := os.OpenFile(path.Join(dir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_RDONLY, 0755)
	errors.Check(err)

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
