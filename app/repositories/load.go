package repositories

func (repository *Repository) Load(ref string) error {
	save := repository.getSave(ref)
	if save == nil {
		return &ValidationError{"invalid ref."}
	}

	workingDir := repository.GetStatus().WorkingDir
	if len(workingDir.ModifiedFilePaths)+len(workingDir.RemovedFilePaths)+len(workingDir.UntrackedFilePaths) > 0 {
		return &ValidationError{"unsaved changes."}
	}

	dir := buildDirFromSave(repository.fs.Root, save)

	nodes := dir.PreOrderTraversal()
	// if we are traversing the root dir, the root-dir-file should be removed from the response.
	nodes = nodes[1:]

	repository.fs.SafeRemoveWorkingDir(dir)

	for _, node := range nodes {
		repository.fs.CreateNode(node)
	}

	repository.setHead(ref)

	return nil
}
