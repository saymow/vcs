package repositories

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	Path "path/filepath"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
	"time"
)

func (repository *Repository) createConflictFile(refFile *directories.File, incomingFile *directories.File, refName, incomingName string) *directories.FileConflict {
	refFileContent := repository.fs.ReadDirFile(refFile)
	incomingFileContent := repository.fs.ReadDirFile(incomingFile)

	var buffer bytes.Buffer

	_, err := buffer.Write([]byte(fmt.Sprintf("<%s>\n", refName)))
	errors.Check(err)

	_, err = buffer.Write(refFileContent.Bytes())
	errors.Check(err)

	_, err = buffer.Write([]byte(fmt.Sprintf("\n</%s>\n", refName)))
	errors.Check(err)

	_, err = buffer.Write([]byte(fmt.Sprintf("<%s>\n", incomingName)))
	errors.Check(err)

	_, err = buffer.Write(incomingFileContent.Bytes())
	errors.Check(err)

	_, err = buffer.Write([]byte(fmt.Sprintf("\n</%s>\n", incomingName)))
	errors.Check(err)

	hasher := sha256.New()
	_, err = hasher.Write(buffer.Bytes())
	errors.Check(err)
	hash := hasher.Sum(nil)

	objectName := hex.EncodeToString(hash)
	objectFile, err := os.Create(Path.Join(repository.fs.Root, filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, objectName))
	errors.Check(err)
	defer objectFile.Close()

	compressor := gzip.NewWriter(objectFile)
	_, err = compressor.Write(buffer.Bytes())
	errors.Check(err)
	errors.Check(compressor.Close())

	return &directories.FileConflict{
		Filepath:   refFile.Filepath,
		ObjectName: objectName,
		Message:    "Conflict.",
	}
}

func (repository *Repository) handleMergeSave(refSave *filesystems.Save, incomingSave *filesystems.Save, ref, incoming string) *filesystems.Save {
	commonCheckpoint := refSave.FindFirstCommonCheckpointParent(incomingSave)
	ancestorSave := repository.getSave(commonCheckpoint.Id)
	dir := buildDir(repository.fs.Root, ancestorSave)
	refCommonAncestorIdx := collections.FindIndex(refSave.Checkpoints, func(checkpoint *filesystems.Checkpoint, _ int) bool {
		return checkpoint.Id == commonCheckpoint.Id
	})
	incomingAncestorIdx := collections.FindIndex(incomingSave.Checkpoints, func(checkpoint *filesystems.Checkpoint, _ int) bool {
		return checkpoint.Id == commonCheckpoint.Id
	})

	refChangesMap := make(map[string]*directories.Change)

	for _, checkpoint := range refSave.Checkpoints[refCommonAncestorIdx+1:] {
		for _, change := range checkpoint.Changes {
			refChangesMap[change.GetPath()] = change

			normalizedPath, err := dir.NormalizePath(change.GetPath())
			errors.Check(err)

			dir.AddNode(normalizedPath, change)
		}
	}

	conflictedChanges := []*directories.Change{}

	for _, checkpoint := range incomingSave.Checkpoints[incomingAncestorIdx+1:] {
		for _, incomingChange := range checkpoint.Changes {
			normalizedPath, err := dir.NormalizePath(incomingChange.GetPath())
			errors.Check(err)

			refChange, ok := refChangesMap[incomingChange.GetPath()]

			if !ok || !refChange.Conflicts(incomingChange) {
				// No conflict found, go ahead

				dir.AddNode(normalizedPath, incomingChange)
				continue
			}
			// Otherwise, create conflict change

			var change *directories.Change

			switch {
			case refChange.ChangeType == directories.Removal:
				change = &directories.Change{
					ChangeType: directories.Conflict,
					Conflict: &directories.FileConflict{
						Filepath:   incomingChange.File.Filepath,
						ObjectName: incomingChange.File.ObjectName,
						Message:    fmt.Sprintf("Removed at \"%s\" but modified at \"%s\".", ref, incoming),
					},
				}
			case incomingChange.ChangeType == directories.Removal:
				change = &directories.Change{
					ChangeType: directories.Conflict,
					Conflict: &directories.FileConflict{
						Filepath:   refChange.File.Filepath,
						ObjectName: refChange.File.ObjectName,
						Message:    fmt.Sprintf("Removed at \"%s\" but modified at \"%s\".", incoming, ref),
					},
				}
			default:
				change = &directories.Change{
					ChangeType: directories.Conflict,
					Conflict:   repository.createConflictFile(refChange.File, incomingChange.File, ref, incoming),
				}
			}

			conflictedChanges = append(conflictedChanges, change)
			dir.AddNode(normalizedPath, change)
		}
	}

	// Apply changes on the working directory
	repository.applyDir(dir)

	// Append the incoming Checkpoints to the end of the refSave, to keep the incoming save history correct
	incomingCheckpoints := incomingSave.Checkpoints[incomingAncestorIdx+1:]
	leafCheckpointId := refSave.Id

	for len(incomingCheckpoints) > 0 {
		// Rebuild each checkpoint accordingly

		incomingCheckpoint := incomingCheckpoints[0]

		checkpoint := filesystems.Checkpoint{
			Parent:    leafCheckpointId,
			Message:   incomingCheckpoint.Message,
			CreatedAt: time.Now(),
			Changes:   incomingCheckpoint.Changes,
		}
		leafCheckpointId = repository.fs.WriteCheckpoint(&checkpoint)

		incomingCheckpoints = incomingCheckpoints[1:]
	}

	if len(conflictedChanges) > 0 {
		// Then populate the index with conflicting changes and let the user resolve the merge.

		repository.setRef(repository.head, leafCheckpointId)
		repository.index = conflictedChanges
		repository.SaveIndex()

		return repository.getSave(leafCheckpointId)
	}

	// Otherwise, append merge checkpoint at the end
	checkpoint := filesystems.Checkpoint{
		Message:   fmt.Sprintf("Merge \"%s\" at \"%s\".", incoming, ref),
		Parent:    leafCheckpointId,
		CreatedAt: time.Now(),
		Changes:   []*directories.Change{},
	}
	checkpoint.Id = repository.fs.WriteCheckpoint(&checkpoint)
	repository.setRef(repository.head, checkpoint.Id)

	return repository.getSave(checkpoint.Id)
}

func (repository *Repository) Merge(ref string) (*filesystems.Save, error) {
	if repository.isDetachedMode() {
		return nil, &ValidationError{"cannot make changes in detached mode."}
	}

	if len(repository.index) > 0 {
		return nil, &ValidationError{"unsaved changes."}
	}

	workingDirStatus := repository.GetStatus().WorkingDir
	if len(workingDirStatus.ModifiedFilePaths)+len(workingDirStatus.RemovedFilePaths)+len(workingDirStatus.UntrackedFilePaths) > 0 {
		return nil, &ValidationError{"unsaved changes."}
	}

	refSave := repository.getSave(repository.getCurrentSaveName())
	incomingSave := repository.getSave(ref)
	if incomingSave == nil {
		return nil, &ValidationError{"invalid ref."}
	}

	if incomingSave.Contains(refSave) {
		// Fast forward

		dir := buildDir(repository.fs.Root, incomingSave)

		repository.applyDir(dir)
		repository.setRef(repository.head, incomingSave.Id)
		return incomingSave, nil
	}

	save := repository.handleMergeSave(refSave, incomingSave, repository.head, ref)

	return save, nil
}
