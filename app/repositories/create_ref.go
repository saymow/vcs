package repositories

func (repository *Repository) CreateRef(name string) error {
	currentSaveName := repository.getCurrentSaveName()

	if currentSaveName == "" {
		return &ValidationError{"cannot create refs when there is no save history."}
	}
	if saveName, found := (*repository.refs)[name]; found && saveName != currentSaveName {
		return &ValidationError{"name already in use."}
	}

	repository.setRef(name, repository.getCurrentSaveName())
	repository.setHead(name)
	return nil
}
