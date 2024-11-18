package filesystem

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
	path "path/filepath"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/directory"
	"slices"
	"strings"
	"time"
)

const (
	REPOSITORY_FOLDER_NAME = ".repository"
	OBJECTS_FOLDER_NAME    = "objects"
	SAVES_FOLDER_NAME      = "saves"
	INDEX_FILE_NAME        = "index"
	HEAD_FILE_NAME         = "head"
)

type Save struct {
	Id          string
	Checkpoints []*CheckPoint
}

type CheckPoint struct {
	Id        string
	Message   string
	CreatedAt time.Time
	Parent    string
	Changes   []*directory.Change
}

type FileSystem struct {
	Root string
}

func Create(root string) *FileSystem {
	err := os.Mkdir(path.Join(root, REPOSITORY_FOLDER_NAME), 0644)
	errors.Check(err)

	indexFile, err := os.Create(path.Join(root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
	errors.Check(err)
	defer indexFile.Close()

	_, err = indexFile.Write([]byte("Tracked files:\r\n\r\n"))
	errors.Check(err)

	headFile, err := os.Create(path.Join(root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME))
	errors.Check(err)
	defer headFile.Close()

	err = os.Mkdir(path.Join(root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME), 0644)
	errors.Check(err)

	err = os.Mkdir(path.Join(root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME), 0644)
	errors.Check(err)

	return &FileSystem{Root: root}
}

func Open(root string) *FileSystem {
	return &FileSystem{Root: root}
}

func (fileSystem *FileSystem) SaveIndex(index []*directory.Change) {
	file, err := os.OpenFile(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte("Tracked files:\n\n"))
	errors.Check(err)

	for _, change := range index {
		if change.ChangeType == directory.Modification {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directory.MODIFIED_CHANGE, change.File.ObjectName)))
		} else if change.ChangeType == directory.Creation {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directory.CREATED_CHANGE, change.File.ObjectName)))
		} else {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n", change.Removal.Filepath, directory.REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}
}

func (fileSystem *FileSystem) parseIndex(file *os.File) []*directory.Change {
	var index []*directory.Change
	scanner := bufio.NewScanner(file)

	// Skip file header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		change := directory.Change{}

		changeHeader := strings.Split(scanner.Text(), "\t")

		if len(changeHeader) != 2 {
			errors.Error("Invalid index format.")
		}

		if changeHeader[1] == directory.MODIFIED_CHANGE || changeHeader[1] == directory.CREATED_CHANGE {
			if changeHeader[1] == directory.MODIFIED_CHANGE {
				change.ChangeType = directory.Modification
			} else {
				change.ChangeType = directory.Creation
			}

			change.File = &directory.File{}
			change.File.Filepath = changeHeader[0]
			scanner.Scan()
			change.File.ObjectName = scanner.Text()
		} else {
			change.ChangeType = directory.Removal
			change.Removal = &directory.FileRemoval{}
			change.Removal.Filepath = changeHeader[0]
		}

		index = append(index, &change)
	}

	return index
}

func (fileSystem *FileSystem) ReadIndex() []*directory.Change {
	file, err := os.OpenFile(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_RDONLY, 0644)
	errors.Check(err)
	defer file.Close()

	return fileSystem.parseIndex(file)
}

func (fileSystem *FileSystem) WriteHead(name string) {
	file, err := os.OpenFile(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0644)
	errors.Check(err)

	_, err = file.Write([]byte(name))
	errors.Check(err)
}

func (fileSystem *FileSystem) parseHead(file *os.File) string {
	buffer := make([]byte, 128)
	n, err := file.Read(buffer)

	if err != nil && err != io.EOF {
		errors.Error(err.Error())
	}

	return string(buffer[:n])
}

func (fileSystem *FileSystem) ReadHead() string {
	file, err := os.OpenFile(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_RDONLY, 0644)
	errors.Check(err)
	defer file.Close()

	return fileSystem.parseHead(file)
}

func (fileSystem *FileSystem) ReadDir(head string) directory.Dir {
	dir := directory.Dir{Path: fileSystem.Root, Children: make(map[string]*directory.Node)}
	changes := []directory.Change{}
	saveName := head

	for saveName != "" {
		file, err := os.Open(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
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
			change := directory.Change{}

			changeHeader := strings.Split(scanner.Text(), "\t")

			if len(changeHeader) != 2 {
				errors.Error("Invalid save format.")
			}

			if changeHeader[1] == directory.MODIFIED_CHANGE || changeHeader[1] == directory.CREATED_CHANGE {
				if changeHeader[1] == directory.MODIFIED_CHANGE {
					change.ChangeType = directory.Modification
				} else {
					change.ChangeType = directory.Creation
				}

				change.File = &directory.File{}
				change.File.Filepath = changeHeader[0]
				scanner.Scan()
				change.File.ObjectName = scanner.Text()
			} else {
				change.ChangeType = directory.Removal
				change.Removal = &directory.FileRemoval{}
				change.Removal.Filepath = changeHeader[0]
			}

			changes = append(changes, change)
		}

		file.Close()
	}

	slices.Reverse(changes)

	for _, change := range changes {
		var normalizedPath string

		if change.ChangeType == directory.Removal {
			normalizedPath = change.Removal.Filepath[len(fileSystem.Root)+1:]
		} else {
			normalizedPath = change.File.Filepath[len(fileSystem.Root)+1:]
		}

		dir.AddNode(normalizedPath, &change)
	}

	return dir
}

func (fileSystem *FileSystem) WriteObject(filepath string, file *os.File) *directory.File {
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
	objectFile, err := os.Create(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, objectName))
	errors.Check(err)
	defer objectFile.Close()

	compressor := gzip.NewWriter(objectFile)
	_, err = compressor.Write(buffer.Bytes())
	errors.Check(err)
	compressor.Close()

	return &directory.File{Filepath: filepath, ObjectName: objectName}
}

func (fileSystem *FileSystem) RemoveObject(name string) {
	err := os.Remove(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, name))
	errors.Check(err)
}

func (fileSystem *FileSystem) WriteSave(save *CheckPoint) string {
	var stringBuilder strings.Builder

	_, err := stringBuilder.Write([]byte(fmt.Sprintf("%s\n", save.Message)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\n", save.Parent)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\n\n", save.CreatedAt.Format(time.Layout))))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte("Please do not edit the lines below.\n\n\nFiles:\n\n"))
	errors.Check(err)

	for _, change := range save.Changes {
		if change.ChangeType == directory.Modification {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directory.MODIFIED_CHANGE, change.File.ObjectName)))
		} else if change.ChangeType == directory.Creation {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directory.CREATED_CHANGE, change.File.ObjectName)))
		} else {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n", change.Removal.Filepath, directory.REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}

	saveContent := stringBuilder.String()

	hasher := sha256.New()
	_, err = hasher.Write([]byte(saveContent))
	errors.Check(err)
	hash := hasher.Sum(nil)

	saveName := hex.EncodeToString(hash)

	file, err := os.Create(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
	errors.Check(err)
	defer file.Close()

	_, err = file.Write([]byte(saveContent))
	errors.Check(err)

	return saveName
}

func (fileSystem *FileSystem) ParseCheckpoint(file *os.File) *CheckPoint {
	checkpoint := &CheckPoint{}
	scanner := bufio.NewScanner(file)

	scanner.Scan()
	checkpoint.Message = scanner.Text()

	scanner.Scan()
	checkpoint.Parent = scanner.Text()

	scanner.Scan()
	createdAt, err := time.Parse(time.Layout, scanner.Text())
	errors.Check(err)
	checkpoint.CreatedAt = createdAt

	// skip newline
	scanner.Scan()
	// skip warn message
	scanner.Scan()
	// skip newline
	scanner.Scan()
	// skip newline
	scanner.Scan()
	// skip header message
	scanner.Scan()
	// skip newline
	scanner.Scan()

	for scanner.Scan() {
		change := &directory.Change{}

		changeHeader := strings.Split(scanner.Text(), "\t")

		if len(changeHeader) != 2 {
			errors.Error("Invalid save format.")
		}

		if changeHeader[1] == directory.MODIFIED_CHANGE || changeHeader[1] == directory.CREATED_CHANGE {
			if changeHeader[1] == directory.MODIFIED_CHANGE {
				change.ChangeType = directory.Modification
			} else {
				change.ChangeType = directory.Creation
			}

			change.File = &directory.File{}
			change.File.Filepath = changeHeader[0]
			scanner.Scan()
			change.File.ObjectName = scanner.Text()
		} else {
			change.ChangeType = directory.Removal
			change.Removal = &directory.FileRemoval{}
			change.Removal.Filepath = changeHeader[0]
		}

		checkpoint.Changes = append(checkpoint.Changes, change)
	}

	return checkpoint
}

func (fileSystem *FileSystem) ReadSave(checkpointId string) *Save {
	save := &Save{Id: checkpointId}

	checkpointFile, err := os.Open(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, checkpointId))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		errors.Error(err.Error())
	}
	defer checkpointFile.Close()

	save.Checkpoints = append(save.Checkpoints, fileSystem.ParseCheckpoint(checkpointFile))

	for save.Checkpoints[len(save.Checkpoints)-1].Parent != "" {
		checkpointId = save.Checkpoints[len(save.Checkpoints)-1].Parent
		checkpointFile, err = os.Open(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, checkpointId))
		errors.Check(err)
		save.Checkpoints = append(save.Checkpoints, fileSystem.ParseCheckpoint(checkpointFile))
		checkpointFile.Close()
	}

	slices.Reverse(save.Checkpoints)

	return save
}

func (fileSystem *FileSystem) createFile(file *directory.File) {
	sourceFile, err := os.OpenFile(file.Filepath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			errors.Error(err.Error())
		}

		sourceFile, err = os.Create(file.Filepath)
		errors.Check(err)
	}

	errors.Check(err)
	defer sourceFile.Close()

	objectFile, err := os.Open(path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, file.ObjectName))
	errors.Check(err)
	defer objectFile.Close()

	decompressor, err := gzip.NewReader(objectFile)
	errors.Check(err)

	buffer := make([]byte, 256)

	for {
		n, err := decompressor.Read(buffer)

		if err != nil && err != io.EOF {
			errors.Error(err.Error())
		}
		if n == 0 {
			break
		}

		_, err = sourceFile.Write(buffer[:n])
		errors.Check(err)
	}
}

func (fileSystem *FileSystem) CreateNode(node *directory.Node) {
	if node.NodeType == directory.FileType {
		fileSystem.createFile(node.File)
		return
	}

	err := os.Mkdir(node.Dir.Path, 0644)
	errors.Check(err)
}

// Safely remove a directory
//
// This helper prevents the .repository dir to be removed
func (fileSystem *FileSystem) SafeRemoveDir(dir *directory.Dir) {
	if dir.Path != fileSystem.Root {
		err := os.RemoveAll(dir.Path)
		errors.Check(err)
		return
	}

	entries, err := os.ReadDir(fileSystem.Root)
	errors.Check(err)

	for _, entry := range entries {
		if entry.Name() == REPOSITORY_FOLDER_NAME {
			continue
		}

		filepath := path.Join(fileSystem.Root, entry.Name())

		if entry.IsDir() {
			err := os.RemoveAll(filepath)
			errors.Check(err)
		} else {
			err := os.Remove(filepath)
			errors.Check(err)
		}
	}
}
