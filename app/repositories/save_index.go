package repositories

func (repository *Repository) SaveIndex() error {
	if repository.isDetachedMode() {
		return &ValidationError{"cannot make changes in detached mode."}
	}

	repository.fs.SaveIndex(repository.index)

	return nil
}
