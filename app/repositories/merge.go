package repositories

import (
	"fmt"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/repositories/filesystems"
	"time"
)

func (repository *Repository) handleMergeSave(refSave *filesystems.Save, incomingSave *filesystems.Save, ref, incoming string) *filesystems.Checkpoint {
	commonCheckpoint := refSave.FindFirstCommonCheckpointParent(incomingSave)
	ancestorSave := repository.getSave(commonCheckpoint.Id)
	dir := buildDirFromSave(repository.fs.Root, ancestorSave)
	refCommonAncestorIdx := collections.FindIndex(refSave.Checkpoints, func(checkpoint *filesystems.Checkpoint, _ int) bool {
		return checkpoint.Id == commonCheckpoint.Id
	})
	incomingAncestorIdx := collections.FindIndex(incomingSave.Checkpoints, func(checkpoint *filesystems.Checkpoint, _ int) bool {
		return checkpoint.Id == commonCheckpoint.Id
	})

	save := filesystems.Checkpoint{
		Message:   fmt.Sprintf("Merge \"%s\" at \"%s\".", incoming, ref),
		Parent:    refSave.Id,
		CreatedAt: time.Now(),
	}

	for _, checkpoint := range refSave.Checkpoints[refCommonAncestorIdx+1:] {
		for _, change := range checkpoint.Changes {
			normalizedPath, err := dir.NormalizePath(change.GetPath())
			errors.Check(err)

			dir.AddNode(normalizedPath, change)
		}
	}

	for _, checkpoint := range incomingSave.Checkpoints[incomingAncestorIdx+1:] {
		for _, change := range checkpoint.Changes {
			normalizedPath, err := dir.NormalizePath(change.GetPath())
			errors.Check(err)

			save.Changes = append(save.Changes, change)
			dir.AddNode(normalizedPath, change)
		}
	}

	save.Id = repository.fs.WriteSave(&save)
	repository.setRef(repository.head, save.Id)

	repository.fs.SafeRemoveWorkingDir(dir)

	nodes := dir.PreOrderTraversal()
	nodes = nodes[1:]

	for _, node := range nodes {
		repository.fs.CreateNode(node)
	}

	return &save
}

func (repository *Repository) Merge(ref string) (*filesystems.Checkpoint, error) {
	if repository.isDetachedMode() {
		return nil, &ValidationError{"cannot make changes in detached mode."}
	}

	incomingSave := repository.getSave(ref)
	if incomingSave == nil {
		return nil, &ValidationError{"invalid ref."}
	}

	refSave := repository.getSave(repository.getCurrentSaveName())

	workingDirStatus := repository.GetStatus().WorkingDir
	if len(workingDirStatus.ModifiedFilePaths)+len(workingDirStatus.RemovedFilePaths)+len(workingDirStatus.UntrackedFilePaths) > 0 {
		return nil, &ValidationError{"unsaved changes."}
	}

	// Fast forward
	if incomingSave.Contains(refSave) {

		dir := buildDirFromSave(repository.fs.Root, incomingSave)

		nodes := dir.PreOrderTraversal()
		// if we are traversing the root dir, the root-dir-file should be removed from the response.
		nodes = nodes[1:]

		repository.fs.SafeRemoveWorkingDir(dir)

		for _, node := range nodes {
			repository.fs.CreateNode(node)
		}

		repository.setRef(repository.head, incomingSave.Id)

		return incomingSave.Checkpoint(), nil
	}

	save := repository.handleMergeSave(refSave, incomingSave, repository.head, ref)

	return save, nil
}
