package repositories

import (
	"os"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/directories"
	"slices"
)

func (repository *Repository) RemoveFile(filepath string) error {
	if repository.isDetachedMode() {
		return &ValidationError{"cannot make changes in detached mode."}
	}

	filepath, err := repository.dir.AbsPath(filepath)
	if err != nil {
		return &ValidationError{err.Error()}
	}

	// Remove from working dir
	err = os.Remove(filepath)
	if err != nil && !os.IsNotExist(err) {
		errors.Error(err.Error())
	}

	stagedChangeIdx := repository.findStagedChangeIdx(filepath)
	savedObject := repository.findSavedFile(filepath)

	if stagedChangeIdx != -1 {
		if repository.index[stagedChangeIdx].ChangeType == directories.Removal {
			// Index entry is already meant for removal
			return nil
		}

		if !(repository.index[stagedChangeIdx].ChangeType == directories.Conflict && !repository.index[stagedChangeIdx].Conflict.IsObjectTemporary()) {
			// Remove change object unless it is a conflict permanent object.

			repository.fs.RemoveObject(repository.index[stagedChangeIdx].GetHash())
		}
		// Remove existing change from the index
		repository.index = slices.Delete(repository.index, stagedChangeIdx, stagedChangeIdx+1)
	}

	if savedObject != nil {
		// Create Index file removal entry
		repository.index = append(repository.index, &directories.Change{ChangeType: directories.Removal, Removal: &directories.FileRemoval{Filepath: filepath}})
	}

	return nil
}
