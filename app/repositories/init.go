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

type NodeType int

const (
	FileType NodeType = iota
	DirType
)

type Node struct {
	nodeType NodeType
	file     Object
	dir      Dir
}

type Dir struct {
	children map[string]Node
}

type Repository struct {
	rootDir string
	head    string
	index   []*Object
	wd      Dir
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
		wd:      Dir{},
	}
}

func readIndex(file *os.File) []*Object {
	var index []*Object
	scanner := bufio.NewScanner(file)

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

func addDirNodeHelper(segments []string, dir *Dir, object *Object) {
	if len(segments) == 1 {
		dir.children[segments[0]] = Node{
			nodeType: FileType,
			file:     *object,
		}

		return
	}

	subdirName := segments[0]
	var node Node

	if _, ok := dir.children[subdirName]; ok {
		node = dir.children[subdirName]
	} else {
		node = Node{
			nodeType: DirType,
			dir:      Dir{make(map[string]Node)},
		}
		dir.children[subdirName] = node
	}

	addDirNodeHelper(segments[1:], &node.dir, object)
}

func addDirNode(rootDir string, dir *Dir, object *Object) {
	// len(rootDir)+1 to skip initial /. E.g, turn "base/a/b/c/file" to "a/b/c/file"
	normalizedPath := object.filepath[len(rootDir)+1:]
	segments := strings.Split(normalizedPath, string(fp.Separator))

	addDirNodeHelper(segments, dir, object)
}

func buildDir(rootDir string, head string) Dir {
	dir := Dir{make(map[string]Node)}
	objects := []Object{}
	saveName := head

	for saveName != "" {
		file, err := os.Open(path.Join(rootDir, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
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
			object := Object{}

			object.filepath = scanner.Text()
			scanner.Scan()
			object.name = scanner.Text()
			objects = append(objects, object)
		}

		file.Close()
	}

	slices.Reverse(objects)

	for _, object := range objects {
		addDirNode(rootDir, &dir, &object)
	}

	return dir
}

func GetRepository(dir string) *Repository {
	indexFile, err := os.OpenFile(path.Join(dir, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_RDONLY, 0755)
	errors.Check(err)
	defer indexFile.Close()

	index := readIndex(indexFile)

	headFile, err := os.OpenFile(path.Join(dir, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_RDONLY, 0755)
	errors.Check(err)
	defer headFile.Close()

	head := readHead(headFile)
	wd := buildDir(dir, head)

	return &Repository{
		rootDir: dir,
		index:   index,
		head:    head,
		wd:      wd,
	}
}
