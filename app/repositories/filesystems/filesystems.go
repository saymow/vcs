package filesystems

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
	Path "path/filepath"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/directories"
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
	REFS_FILE_NAME         = "refs"

	INITIAL_REF_NAME = "master"
)

type Save struct {
	Id          string
	Checkpoints []*Checkpoint
}

type Checkpoint struct {
	Id        string
	Message   string
	CreatedAt time.Time
	Parent    string
	Changes   []*directories.Change
}

type FileSystem struct {
	Root string
}

type Refs map[string]string

func Create(root string) *FileSystem {
	err := os.Mkdir(Path.Join(root, REPOSITORY_FOLDER_NAME), 0644)
	errors.Check(err)

	indexFile, err := os.Create(Path.Join(root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
	errors.Check(err)
	defer indexFile.Close()

	_, err = indexFile.Write([]byte("Tracked files:\r\n\r\n"))
	errors.Check(err)

	refsFile, err := os.Create(Path.Join(root, REPOSITORY_FOLDER_NAME, REFS_FILE_NAME))
	errors.Check(err)
	defer refsFile.Close()

	_, err = refsFile.Write([]byte(fmt.Sprintf("Refs:\n\n%s\n\n", INITIAL_REF_NAME)))
	errors.Check(err)

	headFile, err := os.Create(Path.Join(root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME))
	errors.Check(err)
	defer headFile.Close()

	_, err = headFile.Write([]byte(INITIAL_REF_NAME))
	errors.Check(err)

	err = os.Mkdir(Path.Join(root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME), 0644)
	errors.Check(err)

	err = os.Mkdir(Path.Join(root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME), 0644)
	errors.Check(err)

	return &FileSystem{Root: root}
}

func Open(root string) *FileSystem {
	return &FileSystem{Root: root}
}

func (save *Save) Contains(otherSave *Save) bool {
	otherSaveCheckpoint := otherSave.Checkpoints[len(otherSave.Checkpoints)-1]

	for idx := len(save.Checkpoints) - 1; idx >= 0; idx-- {
		if save.Checkpoints[idx].Id == otherSaveCheckpoint.Id {
			return true
		}
	}

	return false
}

func (save *Save) Checkpoint() *Checkpoint {
	return save.Checkpoints[len(save.Checkpoints)-1]
}

func (save *Save) FindFirstCommonCheckpointParent(otherSave *Save) *Checkpoint {
	seen := make(map[string]*Checkpoint)

	for idx := len(otherSave.Checkpoints) - 2; idx >= 0; idx-- {
		seen[otherSave.Checkpoints[idx].Id] = otherSave.Checkpoints[idx]
	}

	for idx := len(save.Checkpoints) - 2; idx >= 0; idx-- {
		if checkpoint, ok := seen[save.Checkpoints[idx].Id]; ok {
			return checkpoint
		}
	}

	return nil
}

func (fileSystem *FileSystem) SaveIndex(index []*directories.Change) {
	file, err := os.OpenFile(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte("Tracked files:\n\n"))
	errors.Check(err)

	for _, change := range index {
		if change.ChangeType == directories.Modification {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directories.MODIFIED_CHANGE, change.File.ObjectName)))
		} else if change.ChangeType == directories.Creation {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directories.CREATED_CHANGE, change.File.ObjectName)))
		} else {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n", change.Removal.Filepath, directories.REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}
}

func (fileSystem *FileSystem) parseIndex(file *os.File) []*directories.Change {
	var index []*directories.Change
	scanner := bufio.NewScanner(file)

	// Skip file header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		change := directories.Change{}

		changeHeader := strings.Split(scanner.Text(), "\t")

		if len(changeHeader) != 2 {
			errors.Error("Invalid index format.")
		}

		if changeHeader[1] == directories.MODIFIED_CHANGE || changeHeader[1] == directories.CREATED_CHANGE {
			if changeHeader[1] == directories.MODIFIED_CHANGE {
				change.ChangeType = directories.Modification
			} else {
				change.ChangeType = directories.Creation
			}

			change.File = &directories.File{}
			change.File.Filepath = changeHeader[0]
			scanner.Scan()
			change.File.ObjectName = scanner.Text()
		} else {
			change.ChangeType = directories.Removal
			change.Removal = &directories.FileRemoval{}
			change.Removal.Filepath = changeHeader[0]
		}

		index = append(index, &change)
	}

	return index
}

func (fileSystem *FileSystem) ReadIndex() []*directories.Change {
	file, err := os.OpenFile(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_RDONLY, 0644)
	errors.Check(err)
	defer file.Close()

	return fileSystem.parseIndex(file)
}

func (fileSystem *FileSystem) ReadRefs() *Refs {
	refs := Refs{}

	file, err := os.Open(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, REFS_FILE_NAME))
	errors.Check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip file header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		key := scanner.Text()

		if !scanner.Scan() {
			errors.Error("Invalid refs format.")
		}

		refs[key] = scanner.Text()
	}

	return &refs
}

func (fileSystem *FileSystem) WriteRefs(refs *Refs) {
	file, err := os.OpenFile(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, REFS_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0644)
	errors.Check(err)
	defer file.Close()

	_, err = file.Write([]byte("Refs:\n\n"))
	errors.Check(err)

	for branchName, saveName := range *refs {
		_, err = file.Write([]byte(fmt.Sprintf("%s\n%s\n", branchName, saveName)))
		errors.Check(err)
	}
}

func (fileSystem *FileSystem) WriteHead(name string) {
	file, err := os.OpenFile(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0644)
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
	file, err := os.OpenFile(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_RDONLY, 0644)
	errors.Check(err)
	defer file.Close()

	return fileSystem.parseHead(file)
}

func (fileSystem *FileSystem) ReadDir(saveName string) directories.Dir {
	dir := directories.Dir{Path: fileSystem.Root, Children: make(map[string]*directories.Node)}
	changes := []directories.Change{}

	for saveName != "" {
		file, err := os.Open(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
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
			change := directories.Change{}

			changeHeader := strings.Split(scanner.Text(), "\t")

			if len(changeHeader) != 2 {
				errors.Error("Invalid save format.")
			}

			if changeHeader[1] == directories.MODIFIED_CHANGE || changeHeader[1] == directories.CREATED_CHANGE {
				if changeHeader[1] == directories.MODIFIED_CHANGE {
					change.ChangeType = directories.Modification
				} else {
					change.ChangeType = directories.Creation
				}

				change.File = &directories.File{}
				change.File.Filepath = changeHeader[0]
				scanner.Scan()
				change.File.ObjectName = scanner.Text()
			} else {
				change.ChangeType = directories.Removal
				change.Removal = &directories.FileRemoval{}
				change.Removal.Filepath = changeHeader[0]
			}

			changes = append(changes, change)
		}

		file.Close()
	}

	slices.Reverse(changes)

	for _, change := range changes {
		normalizedPath, err := dir.NormalizePath(change.GetPath())
		errors.Check(err)

		dir.AddNode(normalizedPath, &change)
	}

	return dir
}

func (fileSystem *FileSystem) WriteObject(filepath string, file *os.File) *directories.File {
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
	objectFile, err := os.Create(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, objectName))
	errors.Check(err)
	defer objectFile.Close()

	compressor := gzip.NewWriter(objectFile)
	_, err = compressor.Write(buffer.Bytes())
	errors.Check(err)
	compressor.Close()

	return &directories.File{Filepath: filepath, ObjectName: objectName}
}

func (fileSystem *FileSystem) RemoveObject(name string) {
	err := os.Remove(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, name))
	errors.Check(err)
}

func (fileSystem *FileSystem) WriteSave(save *Checkpoint) string {
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
		if change.ChangeType == directories.Modification {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directories.MODIFIED_CHANGE, change.File.ObjectName)))
		} else if change.ChangeType == directories.Creation {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.File.Filepath, directories.CREATED_CHANGE, change.File.ObjectName)))
		} else {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n", change.Removal.Filepath, directories.REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}

	saveContent := stringBuilder.String()

	hasher := sha256.New()
	_, err = hasher.Write([]byte(saveContent))
	errors.Check(err)
	hash := hasher.Sum(nil)

	saveName := hex.EncodeToString(hash)

	file, err := os.Create(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
	errors.Check(err)
	defer file.Close()

	_, err = file.Write([]byte(saveContent))
	errors.Check(err)

	return saveName
}

func (fileSystem *FileSystem) ParseCheckpoint(id string, file *os.File) *Checkpoint {
	checkpoint := &Checkpoint{}
	scanner := bufio.NewScanner(file)

	checkpoint.Id = id

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
		change := &directories.Change{}

		changeHeader := strings.Split(scanner.Text(), "\t")

		if len(changeHeader) != 2 {
			errors.Error("Invalid save format.")
		}

		if changeHeader[1] == directories.MODIFIED_CHANGE || changeHeader[1] == directories.CREATED_CHANGE {
			if changeHeader[1] == directories.MODIFIED_CHANGE {
				change.ChangeType = directories.Modification
			} else {
				change.ChangeType = directories.Creation
			}

			change.File = &directories.File{}
			change.File.Filepath = changeHeader[0]
			scanner.Scan()
			change.File.ObjectName = scanner.Text()
		} else {
			change.ChangeType = directories.Removal
			change.Removal = &directories.FileRemoval{}
			change.Removal.Filepath = changeHeader[0]
		}

		checkpoint.Changes = append(checkpoint.Changes, change)
	}

	return checkpoint
}

func (fileSystem *FileSystem) ReadSave(checkpointId string) *Save {
	save := &Save{Id: checkpointId}

	checkpointFile, err := os.Open(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, checkpointId))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		errors.Error(err.Error())
	}
	defer checkpointFile.Close()

	save.Checkpoints = append(save.Checkpoints, fileSystem.ParseCheckpoint(checkpointId, checkpointFile))

	for save.Checkpoints[len(save.Checkpoints)-1].Parent != "" {
		checkpointId = save.Checkpoints[len(save.Checkpoints)-1].Parent
		checkpointFile, err = os.Open(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, checkpointId))
		errors.Check(err)
		save.Checkpoints = append(save.Checkpoints, fileSystem.ParseCheckpoint(checkpointId, checkpointFile))
		checkpointFile.Close()
	}

	slices.Reverse(save.Checkpoints)

	return save
}

func (fileSystem *FileSystem) createFile(file *directories.File) {
	sourceFile, err := os.OpenFile(file.Filepath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			errors.Error(err.Error())
		}

		sourceFile, err = os.Create(file.Filepath)
		errors.Check(err)
	}
	defer sourceFile.Close()

	objectFile, err := os.Open(Path.Join(fileSystem.Root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, file.ObjectName))
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

func (fileSystem *FileSystem) CreateNode(node *directories.Node) {
	if node.NodeType == directories.FileType {
		fileSystem.createFile(node.File)
		return
	}

	err := os.Mkdir(node.Dir.Path, 0644)
	errors.Check(err)
}

// Safely remove a directory
//
// This helper prevents the .repository dir to be removed
func (fileSystem *FileSystem) SafeRemoveWorkingDir(path string) {
	if path != fileSystem.Root {
		err := os.RemoveAll(path)
		errors.Check(err)
		return
	}

	entries, err := os.ReadDir(fileSystem.Root)
	errors.Check(err)

	for _, entry := range entries {
		if entry.Name() == REPOSITORY_FOLDER_NAME {
			continue
		}

		filepath := Path.Join(fileSystem.Root, entry.Name())

		if entry.IsDir() {
			err := os.RemoveAll(filepath)
			errors.Check(err)
		} else {
			err := os.Remove(filepath)
			errors.Check(err)
		}
	}
}
