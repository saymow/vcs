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
	var changeType ChangeType

	if savedObject != nil {
		changeType = Modification
	} else {
		changeType = Creation
	}

	if savedObject != nil && savedObject.objectName == object.objectName {
		// No changes at all

		if stagedChangeIdx != -1 {
			if repository.index[stagedChangeIdx].changeType != Removal {
				// Remove change file object
				repository.removeObject(repository.index[stagedChangeIdx].file.objectName)
			}

			// Undo index existing change
			repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		}
	} else if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].changeType != Removal &&
			repository.index[stagedChangeIdx].file.objectName != object.objectName {
			// Remove change file object
			repository.removeObject(repository.index[stagedChangeIdx].file.objectName)
		}

		// Undo index existing change
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
		// Index change
		repository.index = append(repository.index, &Change{changeType: changeType, file: object})

	} else {
		// Index change
		repository.index = append(repository.index, &Change{changeType: changeType, file: object})
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
		repository.removeObject(repository.index[stagedChangeIdx].file.objectName)
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
		if change.changeType == Modification {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.file.filepath, MODIFIED_CHANGE, change.file.objectName)))
		} else if change.changeType == Creation {
			_, err = file.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.file.filepath, CREATED_CHANGE, change.file.objectName)))
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
		if change.changeType == Modification {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.file.filepath, MODIFIED_CHANGE, change.file.objectName)))
		} else if change.changeType == Creation {
			_, err = stringBuilder.Write([]byte(fmt.Sprintf("%s\t%s\n%s\n", change.file.filepath, CREATED_CHANGE, change.file.objectName)))
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
		if item.changeType == Removal {
			return item.removal.filepath == filepath
		}

		return item.file.filepath == filepath
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
		if change.changeType == Removal {
			trackedPaths.Insert(change.removal.filepath)
		} else {
			trackedPaths.Insert(change.file.filepath)
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
			if stagedChange.changeType == Removal {
				status.Staged.RemovedFilePaths = append(status.Staged.RemovedFilePaths, stagedChange.removal.filepath)
			} else {
				if savedFile == nil {
					status.Staged.CreatedFilesPaths = append(status.Staged.CreatedFilesPaths, filepath)
				} else {
					status.Staged.ModifiedFilePaths = append(status.Staged.ModifiedFilePaths, filepath)
				}

				if stagedChange.file.objectName != fileHash {
					status.WorkingDir.ModifiedFilePaths = append(status.WorkingDir.ModifiedFilePaths, filepath)
				}
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

		if changeHeader[1] == MODIFIED_CHANGE || changeHeader[1] == CREATED_CHANGE {
			if changeHeader[1] == MODIFIED_CHANGE {
				change.changeType = Modification
			} else {
				change.changeType = Creation
			}

			change.file = &File{}
			change.file.filepath = changeHeader[0]
			scanner.Scan()
			change.file.objectName = scanner.Text()
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

			if change.changeType == Removal {
				normalizedPath = change.removal.filepath[len(root)+1:]
			} else {
				normalizedPath = change.file.filepath[len(root)+1:]
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

// Safely remove a directory
//
// This helper prevents the .repository dir to be removed
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
//
//  2. Load all the content saved up until all checkpoints created.
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

	filesRemovedFromIndex := []*File{}

	if ref == "HEAD" {
		// Index files have higher priority over tree files to be restored

		if node.nodeType == FileType {
			// Check if there is a index modification for the node

			stagedChangeIdx := repository.findStagedChangeIdx(node.file.filepath)

			if stagedChangeIdx != -1 {
				// If so, remove the change from the index

				if repository.index[stagedChangeIdx].changeType != Removal {
					// If it is a file modification, the index modification is applied instead of the tree one.

					node = &Node{nodeType: FileType, file: repository.index[stagedChangeIdx].file}
					filesRemovedFromIndex = append(filesRemovedFromIndex, repository.index[stagedChangeIdx].file)
				}

				repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
			}
		} else {
			// For directories we should iterate over the index and rebuild the node dir, if necessary,
			// applying the index priority.

			for _, change := range slices.Clone(repository.index) {
				var filepath string

				if change.changeType == Removal {
					filepath = change.removal.filepath
				} else {
					filepath = change.file.filepath
				}

				if !strings.HasPrefix(filepath, node.dir.path) {
					continue
				}

				if change.changeType != Removal {
					normalizedPath := filepath[len(node.dir.path)+1:]
					node.dir.addNode(normalizedPath, change)

					filesRemovedFromIndex = append(filesRemovedFromIndex, change.file)
				}

				repository.index = collections.Remove(repository.index, func(item *Change, _ int) bool {
					return item == change
				})
			}
		}
	}

	if node.nodeType == DirType {
		nodes := node.dir.preOrderTraversal()

		if node.dir == rootDir {
			// if we are traversing the root dir, the root-dir-file is included in the response.

			nodes = nodes[1:]
		}

		repository.safeRemoveDir(node.dir)

		for _, node := range nodes {
			repository.applyNode(node)
		}
	} else {
		repository.applyNode(node)
	}

	for _, fileRemoved := range filesRemovedFromIndex {
		repository.removeObject(fileRemoved.objectName)
	}

	return nil
}
