package repositories

import (
	"bufio"
	"io"
	"os"
	"path"
	fp "path/filepath"
	"saymow/version-manager/app/pkg/errors"
	"slices"
	"time"
)

type File struct {
	filepath   string
	objectName string
}

type Save struct {
	message   string
	createdAt time.Time
	parent    string
	files     []*File
}

type NodeType int

const (
	FileType NodeType = iota
	DirType
)

type Node struct {
	nodeType NodeType
	file     File
	dir      Dir
}

type Dir struct {
	children map[string]Node
}

type Repository struct {
	root  string
	head  string
	index []*File
	dir   Dir
}

const (
	REPOSITORY_FOLDER_NAME = ".repository"
	OBJECTS_FOLDER_NAME    = "objects"
	SAVES_FOLDER_NAME      = "saves"
	INDEX_FILE_NAME        = "index"
	HEAD_FILE_NAME         = "head"
)

func CreateRepository(root string) *Repository {
	err := os.Mkdir(fp.Join(root, REPOSITORY_FOLDER_NAME), 0755)
	errors.Check(err)

	indexFile, err := os.Create(fp.Join(root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
	errors.Check(err)
	defer indexFile.Close()

	_, err = indexFile.Write([]byte("Tracked files:\r\n\r\n"))
	errors.Check(err)

	headFile, err := os.Create(fp.Join(root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME))
	errors.Check(err)
	defer headFile.Close()

	err = os.Mkdir(fp.Join(root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME), 0755)
	errors.Check(err)

	err = os.Mkdir(fp.Join(root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME), 0755)
	errors.Check(err)

	return &Repository{
		root:  root,
		index: []*File{},
		dir:   Dir{},
	}
}

func readIndex(file *os.File) []*File {
	var index []*File
	scanner := bufio.NewScanner(file)

	// Skip file header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		object := File{}

		object.filepath = scanner.Text()
		scanner.Scan()
		object.objectName = scanner.Text()

		index = append(index, &object)
	}

	return index
}

func readHead(file *os.File) string {
	buffer := make([]byte, 128)
	n, err := file.Read(buffer)

	if err != nil && err != io.EOF {
		errors.Error(err.Error())
	}

	return string(buffer[:n])
}

func buildDir(root string, head string) Dir {
	dir := Dir{make(map[string]Node)}
	objects := []File{}
	saveName := head

	for saveName != "" {
		file, err := os.Open(path.Join(root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
		errors.Check(err)

		scanner := bufio.NewScanner(file)

		// Skip save message
		scanner.Scan()

		// Scan parent hash
		scanner.Scan()
		saveName = scanner.Text()

		// Skip createdAt
		scanner.Scan()
		// Skip newline
		scanner.Scan()
		// Skip warn message
		scanner.Scan()
		// Skip newline
		scanner.Scan()
		// Skip header message
		scanner.Scan()

		for scanner.Scan() {
			object := File{}

			object.filepath = scanner.Text()
			scanner.Scan()
			object.objectName = scanner.Text()
			objects = append(objects, object)
		}

		file.Close()
	}

	slices.Reverse(objects)

	for _, object := range objects {
		// len(rootDir)+1 to skip initial /. E.g, turn "base/a/b/c/file" to "a/b/c/file"
		normalizedPath := object.filepath[len(root)+1:]
		dir.addNode(normalizedPath, &object)
	}

	return dir
}

func GetRepository(root string) *Repository {
	indexFile, err := os.OpenFile(path.Join(root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_RDONLY, 0755)
	errors.Check(err)
	defer indexFile.Close()

	index := readIndex(indexFile)

	headFile, err := os.OpenFile(path.Join(root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_RDONLY, 0755)
	errors.Check(err)
	defer headFile.Close()

	head := readHead(headFile)
	dir := buildDir(root, head)

	return &Repository{
		root:  root,
		index: index,
		head:  head,
		dir:   dir,
	}
}
