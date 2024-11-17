package repositories

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	Path "path/filepath"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"slices"
	"strings"
	"time"

	"github.com/golang-collections/collections/set"
)

type Status struct {
	Staged struct {
		CreatedFilesPaths []string
		ModifiedFilePaths []string
		RemovedFilePaths  []string
	}
	WorkingDir struct {
		ModifiedFilePaths  []string
		UntrackedFilePaths []string
		RemovedFilePaths   []string
	}
}

type ValidationError struct {
	message string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", err.message)
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
	objectFile, err := os.Create(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, objectName))
	errors.Check(err)
	defer objectFile.Close()

	compressor := gzip.NewWriter(objectFile)
	_, err = compressor.Write(buffer.Bytes())
	errors.Check(err)
	compressor.Close()

	return &File{filepath, objectName}
}

func (repository *Repository) removeObject(name string) {
	err := os.Remove(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, name))
	errors.Check(err)
}

func (repository *Repository) IndexFile(filepath string) {
	if !Path.IsAbs(filepath) {
		filepath = Path.Join(repository.root, filepath)
	}
	if !strings.HasPrefix(filepath, repository.root) {
		log.Fatal("Invalid file path.")
	}

	file, err := os.Open(filepath)
	errors.Check(err)
	defer file.Close()

	object := repository.writeObject(filepath, file)
	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedFile(filepath)

	if savedObject != nil && savedObject.objectName == object.objectName {
		// No changes at all

		if stagedChangeIdx != -1 {
			if repository.index[stagedChangeIdx].changeType == Modified {
				// Remove change file object
				repository.removeObject(repository.index[stagedChangeIdx].modified.objectName)
			}

			// Undo index existing change
			repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		}
	} else if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].changeType == Modified &&
			repository.index[stagedChangeIdx].modified.objectName != object.objectName {
			// Remove change file object
			repository.removeObject(repository.index[stagedChangeIdx].modified.objectName)
		}

		// Undo index existing change
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		// Index change
		repository.index = append(repository.index, &Change{changeType: Modified, modified: object})

	} else {
		// Index change
		repository.index = append(repository.index, &Change{changeType: Modified, modified: object})
	}
}

func (repository *Repository) RemoveFile(filepath string) {
	if !Path.IsAbs(filepath) {
		filepath = Path.Join(repository.root, filepath)
	}
	if !strings.HasPrefix(filepath, repository.root) {
		log.Fatal("Invalid file path.")
	}

	// Remove from working dir
	err := os.Remove(filepath)
	if err != nil && !os.IsNotExist(err) {
		errors.Error(err.Error())
	}

	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedFile(filepath)

	if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].changeType == Removal {
			// Index entry is already meant for removal
			return
		}

		// Remove existing change from the index
		repository.removeObject(repository.index[stagedChangeIdx].modified.objectName)
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
	}

	if savedObject != nil {
		// Create Index file removal entry
		repository.index = append(repository.index, &Change{changeType: Removal, removal: &FileRemoval{filepath}})
	}
}

func (repository *Repository) SaveIndex() {
	file, err := os.OpenFile(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte("Tracked files:\n\n"))
	errors.Check(err)

	for _, change := range repository.index {
		if change.changeType == Modified {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.modified.filepath, MODIFIED_CHANGE, change.modified.objectName)))
		} else {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n", change.removal.filepath, REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}
}

func (repository *Repository) CreateSave(message string) *CheckPoint {
	if len(repository.index) == 0 {
		errors.Error("Cannot save empty index.")
	}

	save := CheckPoint{
		message:   message,
		parent:    repository.head,
		changes:   repository.index,
		createdAt: time.Now(),
	}

	save.id = repository.writeSave(&save)
	repository.clearIndex()
	repository.writeHead(save.id)

	return &save
}

func (repository *Repository) writeSave(save *CheckPoint) string {
	var stringBuilder strings.Builder

	_, err := stringBuilder.Write([]byte(fmt.Sprintf("%s\n", save.message)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\n", save.parent)))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\n\n", save.createdAt.Format(time.Layout))))
	errors.Check(err)

	_, err = stringBuilder.Write([]byte("Please do not edit the lines below.\n\n\nFiles:\n\n"))
	errors.Check(err)

	for _, change := range save.changes {
		if change.changeType == Modified {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.modified.filepath, MODIFIED_CHANGE, change.modified.objectName)))
		} else {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n", change.removal.filepath, REMOVAL_CHANGE)))
		}
		errors.Check(err)
	}

	saveContent := stringBuilder.String()

	hasher := sha256.New()
	_, err = hasher.Write([]byte(saveContent))
	errors.Check(err)
	hash := hasher.Sum(nil)

	saveName := hex.EncodeToString(hash)

	file, err := os.Create(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, saveName))
	errors.Check(err)
	defer file.Close()

	_, err = file.Write([]byte(saveContent))
	errors.Check(err)

	return saveName
}

func (repository *Repository) clearIndex() {
	repository.index = []*Change{}
	repository.SaveIndex()
}

func (repository *Repository) writeHead(name string) {
	file, err := os.OpenFile(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, HEAD_FILE_NAME), os.O_WRONLY|os.O_TRUNC, 0755)
	errors.Check(err)

	_, err = file.Write([]byte(name))
	errors.Check(err)
}

func (repository *Repository) findStagedChangeIdx(filepath string) int {
	return collections.FindIndex(repository.index, func(item *Change, _ int) bool {
		if item.changeType == Modified {
			return item.modified.filepath == filepath
		}

		return item.removal.filepath == filepath
	})
}

func (repository *Repository) findStagedChange(filepath string) *Change {
	idx := repository.findStagedChangeIdx(filepath)

	if idx == -1 {
		return nil
	}

	return repository.index[idx]
}

func (repository *Repository) findSavedFile(filepath string) *File {
	normalizedPath := filepath[len(repository.root)+1:]
	node := repository.dir.findNode(normalizedPath)

	if node == nil || node.nodeType != FileType {
		return nil
	}

	return node.file
}

func (repository *Repository) GetStatus() *Status {
	status := Status{}
	seenPaths := set.New()
	trackedPaths := set.New()

	for _, file := range repository.dir.collectAllFiles() {
		trackedPaths.Insert(file.filepath)
	}

	for _, change := range repository.index {
		if change.changeType == Modified {
			trackedPaths.Insert(change.modified.filepath)
		} else {
			trackedPaths.Insert(change.removal.filepath)
		}
	}

	Path.Walk(repository.root, func(filepath string, info fs.FileInfo, err error) error {
		errors.Check(err)
		if repository.root == filepath || strings.HasPrefix(filepath, Path.Join(repository.root, REPOSITORY_FOLDER_NAME)) {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		seenPaths.Insert(filepath)

		savedFile := repository.findSavedFile(filepath)
		stagedChange := repository.findStagedChange(filepath)

		if savedFile == nil && stagedChange == nil {
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
		fileHash := hex.EncodeToString(hasher.Sum(nil))

		if stagedChange != nil {
			if stagedChange.changeType == Modified {
				if savedFile == nil {
					status.Staged.CreatedFilesPaths = append(status.Staged.CreatedFilesPaths, filepath)
				} else {
					status.Staged.ModifiedFilePaths = append(status.Staged.ModifiedFilePaths, filepath)
				}

				if stagedChange.modified.objectName != fileHash {
					status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
				}
			} else {
				status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, stagedChange.removal.filepath)
			}
		} else {
			if savedFile.objectName != fileHash {
				status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
			}
		}

		return nil
	})

	trackedPaths.Difference(seenPaths).Do(func(i interface{}) {
		filepath := i.(string)

		if repository.findStagedChange(filepath) != nil {
			status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, filepath)
		} else {
			status.WorkingDir.RemovedFilePaths = append(status.WorkingDir.RemovedFilePaths, filepath)
		}
	})

	return &status
}

func readCheckpoint(file *os.File) *CheckPoint {
	checkpoint := &CheckPoint{}
	scanner := bufio.NewScanner(file)

	scanner.Scan()
	checkpoint.message = scanner.Text()

	scanner.Scan()
	checkpoint.parent = scanner.Text()

	scanner.Scan()
	createdAt, err := time.Parse(time.Layout, scanner.Text())
	errors.Check(err)
	checkpoint.createdAt = createdAt

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
		change := &Change{}

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

		checkpoint.changes = append(checkpoint.changes, change)
	}

	return checkpoint
}

func (repository *Repository) getSave(ref string) *Save {
	save := &Save{}
	var checkpointId string

	if ref == "HEAD" {
		checkpointId = repository.head
	} else {
		checkpointId = ref
	}

	if ref == "" {
		return nil
	}

	checkpointFile, err := os.Open(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, checkpointId))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		errors.Error(err.Error())
	}
	defer checkpointFile.Close()

	save.checkpoints = append(save.checkpoints, readCheckpoint(checkpointFile))

	for save.checkpoints[len(save.checkpoints)-1].parent != "" {
		checkpointId = save.checkpoints[len(save.checkpoints)-1].parent
		checkpointFile, err = os.Open(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, SAVES_FOLDER_NAME, checkpointId))
		errors.Check(err)
		save.checkpoints = append(save.checkpoints, readCheckpoint(checkpointFile))
		checkpointFile.Close()
	}

	slices.Reverse(save.checkpoints)

	return save
}

func buildDirFromSave(root string, save *Save) *Dir {
	dir := &Dir{path: root, children: map[string]*Node{}}

	for _, checkpoint := range save.checkpoints {
		for _, change := range checkpoint.changes {
			var normalizedPath string

			if change.changeType == Modified {
				normalizedPath = change.modified.filepath[len(root)+1:]
			} else {
				normalizedPath = change.removal.filepath[len(root)+1:]
			}

			dir.addNode(normalizedPath, change)
		}
	}

	return dir
}

func (repository *Repository) applyFile(file *File) {
	sourceFile, err := os.OpenFile(file.filepath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			errors.Error(err.Error())
		}

		sourceFile, err = os.Create(file.filepath)
		errors.Check(err)
	}

	errors.Check(err)
	defer sourceFile.Close()

	objectFile, err := os.Open(Path.Join(repository.root, REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, file.objectName))
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

func (repository *Repository) applyNode(node *Node) {
	if node.nodeType == FileType {
		repository.applyFile(node.file)
		return
	}

	err := os.Mkdir(node.dir.path, 0644)
	errors.Check(err)
}

func (repository *Repository) safeRemoveDir(dir *Dir) {
	if dir.path != repository.root {
		err := os.RemoveAll(dir.path)
		errors.Check(err)
		return
	}

	entries, err := os.ReadDir(repository.root)
	errors.Check(err)

	for _, entry := range entries {
		if entry.Name() == REPOSITORY_FOLDER_NAME {
			continue
		}

		filepath := Path.Join(repository.root, entry.Name())

		if entry.IsDir() {
			err := os.RemoveAll(filepath)
			errors.Check(err)
		} else {
			err := os.Remove(filepath)
			errors.Check(err)
		}
	}

}

// Load cover 2 usecases:
//
//  1. Restore HEAD + index changes (...and remove the index change).
//
//     It can be used to restore the current head + index changes. Index changes
//     have higher priorities.
func (repository *Repository) Load(ref string, path string) error {
	save := repository.getSave(ref)
	if save == nil {
		return &ValidationError{fmt.Sprintf("\"%s\" is an invalid ref.", ref)}
	}

	rootDir := buildDirFromSave(repository.root, save)
	node := rootDir.findNode(path)
	if node == nil {
		return &ValidationError{fmt.Sprintf("\"%s\" is a invalid path.", ref)}
	}

	nodes := []*Node{}
	filesRemovedFromIndex := []*File{}

	if node.nodeType == DirType {
		nodes = node.dir.preOrderTraversal()

		if node.dir == rootDir {
			// if we are traversing the root dir, the root-dir-file is included in the response.
			// this removes it, since we dont want to recreate the entire dir.

			nodes = nodes[1:]
		}
	} else {
		nodes = append(nodes, node)
	}

	if ref == "HEAD" {
		// index files have higher priority over tree files to be restored

		for idx, change := range slices.Clone(repository.index) {
			// might the time of O(nm) become a problem?
			// idk, we reading and writing a lot on the disk, this is irrelevant.

			var filepath string

			if change.changeType == Modified {
				filepath = change.modified.filepath
			} else {
				filepath = change.removal.filepath
			}

			nodeIdx := collections.FindIndex(nodes, func(node *Node, _ int) bool {
				if node.nodeType == DirType {
					return false
				}

				return node.file.filepath == filepath
			})

			if nodeIdx == -1 {
				continue
			}

			if change.changeType == Modified {
				// should override history file and remove modification from the index

				nodes[nodeIdx] = &Node{nodeType: FileType, file: change.modified}
				filesRemovedFromIndex = append(filesRemovedFromIndex, change.modified)
				repository.index = slices.Delete(repository.index, idx, idx+1)
			} else {
				// should remove removal change from the index

				repository.index = slices.Delete(repository.index, idx, idx+1)
			}
		}
	}

	if node.nodeType == DirType {
		repository.safeRemoveDir(node.dir)
	}
	for _, node := range nodes {
		repository.applyNode(node)
	}
	for _, fileRemoved := range filesRemovedFromIndex {
		repository.removeObject(fileRemoved.objectName)
	}

	return nil
}
