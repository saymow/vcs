package repositories

import (
	"saymow/version-manager/app/repositories/filesystems"
	"time"
)

func (repository *Repository) CreateSave(message string) (*filesystems.Checkpoint, error) {
	if repository.isDetachedMode() {
		return nil, &ValidationError{"cannot make changes in detached mode."}
	}
	if len(repository.index) == 0 {
		return nil, &ValidationError{"cannot save empty index."}
	}

	save := filesystems.Checkpoint{
		Message:   message,
		Parent:    repository.getCurrentSaveName(),
		Changes:   repository.index,
		CreatedAt: time.Now(),
	}

	save.Id = repository.fs.WriteSave(&save)
	repository.clearIndex()
	repository.setRef(repository.head, save.Id)

	return &save, nil
}
