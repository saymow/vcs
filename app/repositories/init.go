package repositories

import (
	"bufio"
	"io"
	"os"
	"path"
	fp "path/filepath"
	"saymow/version-manager/app/pkg/errors"
	"slices"
	"strings"
	"time"
)

type File struct {
	filepath   string
	objectName string
}

type FileRemoval struct {
	filepath string
}

type ChangeType int

const (
	Modified ChangeType = iota
	Removal
)

type Change struct {
	changeType ChangeType
	modified   *File
	removal    *FileRemoval
}

type Save struct {
	id        string
	message   string
	createdAt time.Time
	parent    string
	changes   []*Change
}

type NodeType int

const (
	FileType NodeType = iota
	DirType
)

type Node struct {
	nodeType NodeType
	file     *File
	dir      *Dir
}

type Dir struct {
	children map[string]*Node
}

type Repository struct {
	root  string
	head  string
	index []*Change
	dir   Dir
}

const (
	REPOSITORY_FOLDER_NAME = ".repository"
	OBJECTS_FOLDER_NAME    = "objects"
	SAVES_FOLDER_NAME      = "saves"
	INDEX_FILE_NAME        = "index"
	HEAD_FILE_NAME         = "head"

	MODIFIED_CHANGE = "(modified)"
	REMOVAL_CHANGE  = "(removed)"
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
		index: []*Change{},
		dir:   Dir{},
	}
}

func readIndex(file *os.File) []*Change {
	var index []*Change
	scanner := bufio.NewScanner(file)

	// Skip file header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		change := Change{}

		changeHeader := strings.Split(scanner.Text(), "\t")

		if len(changeHeader) != 2 {
			errors.Error("Invalid index format.")
		}

		if changeHeader[1] == MODIFIED_CHANGE {
			change.changeType = Modified
			change.modified = &File{}
			change.modified.filepath = changeHeader[0]
			scanner.Scan()
			change.modified.objectName = scanner.Text()
		} else {
			change.changeType = Removal
			change.removal = &FileRemoval{}
			change.removal.filepath = changeHeader[0]
		}

		index = append(index, &change)
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
	dir := Dir{make(map[string]*Node)}
	changes := []Change{}
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
		// Skip newline
		scanner.Scan()
		// Skip header message
		scanner.Scan()
		// Skip newline
		scanner.Scan()

		for scanner.Scan() {
			change := Change{}

			changeHeader := strings.Split(scanner.Text(), "\t")

			if len(changeHeader) != 2 {
				errors.Error("Invalid save format.")
			}

			if changeHeader[1] == MODIFIED_CHANGE {
				change.changeType = Modified
				change.modified = &File{}
				change.modified.filepath = changeHeader[0]
				scanner.Scan()
				change.modified.objectName = scanner.Text()
			} else {
				change.changeType = Removal
				change.removal = &FileRemoval{}
				change.removal.filepath = changeHeader[0]
			}

			changes = append(changes, change)
		}

		file.Close()
	}

	slices.Reverse(changes)

	for _, change := range changes {
		var normalizedPath string

		if change.changeType == Modified {
			normalizedPath = change.modified.filepath[len(root)+1:]
		} else {
			normalizedPath = change.removal.filepath[len(root)+1:]
		}

		dir.addNode(normalizedPath, &change)
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
