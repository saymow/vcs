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

func (repository *Repository) handleMergeSave(refSave *filesystems.Save, incomingSave *filesystems.Save, ref, incoming string) *filesystems.Checkpoint {
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
	changes := []*directories.Change{}
	hasConflict := false

	for _, checkpoint := range refSave.Checkpoints[refCommonAncestorIdx+1:] {
		for _, change := range checkpoint.Changes {
			refChangesMap[change.GetPath()] = change

			normalizedPath, err := dir.NormalizePath(change.GetPath())
			errors.Check(err)

			dir.AddNode(normalizedPath, change)
		}
	}

	for _, checkpoint := range incomingSave.Checkpoints[incomingAncestorIdx+1:] {
		for _, incomingChange := range checkpoint.Changes {
			normalizedPath, err := dir.NormalizePath(incomingChange.GetPath())
			errors.Check(err)

			var change *directories.Change

			if refChange, ok := refChangesMap[incomingChange.GetPath()]; ok && refChange.Conflicts(incomingChange) {
				hasConflict = true

				if refChange.ChangeType == directories.Removal {
					change = &directories.Change{
						ChangeType: directories.Conflict,
						Conflict: &directories.FileConflict{
							Filepath:   incomingChange.File.Filepath,
							ObjectName: incomingChange.File.ObjectName,
							Message:    fmt.Sprintf("Removed at \"%s\" but modified at \"%s\".", ref, incoming),
						},
					}
				} else if incomingChange.ChangeType == directories.Removal {
					change = &directories.Change{
						ChangeType: directories.Conflict,
						Conflict: &directories.FileConflict{
							Filepath:   refChange.File.Filepath,
							ObjectName: refChange.File.ObjectName,
							Message:    fmt.Sprintf("Removed at \"%s\" but modified at \"%s\".", incoming, ref),
						},
					}
				} else {
					change = &directories.Change{
						ChangeType: directories.Conflict,
						Conflict:   repository.createConflictFile(refChange.File, incomingChange.File, ref, incoming),
					}
				}
			} else {
				change = incomingChange
			}

			changes = append(changes, change)
			dir.AddNode(normalizedPath, change)
		}
	}

	repository.applyDir(dir)

	if hasConflict {
		repository.index = changes
		repository.SaveIndex()

		return nil
	}

	save := filesystems.Checkpoint{
		Message:   fmt.Sprintf("Merge \"%s\" at \"%s\".", incoming, ref),
		Parent:    refSave.Id,
		CreatedAt: time.Now(),
		Changes:   changes,
	}
	save.Id = repository.fs.WriteSave(&save)
	repository.setRef(repository.head, save.Id)

	return &save
}

func (repository *Repository) Merge(ref string) (*filesystems.Checkpoint, error) {
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
		return incomingSave.Checkpoint(), nil
	}

	save := repository.handleMergeSave(refSave, incomingSave, repository.head, ref)

	if save == nil {
		// No merge save is created, changes applied in the working dir + index

		return refSave.Checkpoint(), nil
	}

	return save, nil
}
