package repositories

import (
	"bufio"
	"io"
	"os"
	"path"
	fp "path/filepath"
	"saymow/version-manager/app/pkg/errors"
	"time"
)

type Object struct {
	filepath string
	name     string
}

type Save struct {
	message   string
	createdAt time.Time
	parent    string
	objects   []*Object
}

type Repository struct {
	rootDir string
	head    string
	index   []*Object
}

const (
	REPOSITORY_FOLDER_NAME = ".repository"
	OBJECTS_FOLDER_NAME    = "objects"
	SAVES_FOLDER_NAME      = "saves"
	INDEX_FILE_NAME        = "index"
	HEAD_FILE_NAME         = "head"
)

func CreateRepository(dir string) *Repository {
	err := os.Mkdir(fp.Join(dir, REPOSITORY_FOLDER_NAME), 0755)
	errors.Check(err)

	indexFile, err := os.Create(fp.Join(dir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
	errors.Check(err)
	defer indexFile.Close()

	_, err = indexFile.Write([]byte("Tracked files:\r\n\r\n"))
	errors.Check(err)

	headFile, err := os.Create(fp.Join(dir, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME))
	errors.Check(err)
	defer headFile.Close()

	err = os.Mkdir(fp.Join(dir, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME), 0755)
	errors.Check(err)

	err = os.Mkdir(fp.Join(dir, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME), 0755)
	errors.Check(err)

	return &Repository{
		rootDir: dir,
		index:   []*Object{},
	}
}

func GetRepository(dir string) *Repository {
	stageFile, err := os.OpenFile(path.Join(dir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_RDONLY, 0755)
	errors.Check(err)
	defer stageFile.Close()

	var index []*Object
	scanner := bufio.NewScanner(stageFile)

	// Skip file header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		object := Object{}

		object.filepath = scanner.Text()
		scanner.Scan()
		object.name = scanner.Text()

		index = append(index, &object)
	}

	headFile, err := os.OpenFile(path.Join(dir, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_RDONLY, 0755)
	errors.Check(err)
	defer headFile.Close()

	head := ""
	buffer := make([]byte, 128)
	n, err := headFile.Read(buffer)

	if err != nil && err != io.EOF {
		errors.Error(err.Error())
	}

	head = string(buffer[:n])

	return &Repository{
		rootDir: dir,
		index:   index,
		head:    head,
	}
}
