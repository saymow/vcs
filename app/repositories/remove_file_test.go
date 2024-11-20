package repositories

import (
	path "path/filepath"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveFile(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	// Check it should only remove from working directory if file is not being tracked
	{
		// Test indempontence along
		repository.RemoveFile(path.Join("a", "5.txt"))
		repository.RemoveFile(path.Join("a", "5.txt"))
		repository.RemoveFile(path.Join("a", "5.txt"))

		changeIdx := collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
			return change.ChangeType == directories.Removal && change.Removal.Filepath == dir.Join("a", "5.txt")
		})
		assert.Equal(t, changeIdx, -1)
		assert.False(t, fixtures.FileExists(dir.Join("a", "5.txt")))
	}

	// Check remove file base case (existing only on the tree and working dir)
	{
		// Test indempontence along
		repository.RemoveFile("1.txt")
		repository.RemoveFile("1.txt")
		repository.RemoveFile("1.txt")

		changeIdx := collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
			return change.ChangeType == directories.Removal && change.Removal.Filepath == dir.Join("1.txt")
		})
		assert.NotEqual(t, changeIdx, -1)
		assert.False(t, fixtures.FileExists(dir.Join("1.txt")))
	}

	// Check remove file base case (existing only on the index and working dir)
	{
		repository.IndexFile(path.Join("a", "4.txt"))

		idx := collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
			return change.ChangeType == directories.Creation && change.File.Filepath == dir.Join("a", "4.txt")
		})

		assert.NotEqual(t, idx, -1)
		creationChange := repository.index[idx]

		// Test indempontence along
		repository.RemoveFile(path.Join("a", "4.txt"))
		repository.RemoveFile(path.Join("a", "4.txt"))
		repository.RemoveFile(path.Join("a", "4.txt"))

		// Check modification change is removed from the index
		assert.Equal(
			t,
			collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
				return change.ChangeType == directories.Modification && change.File.Filepath == dir.Join("a", "4.txt")
			}),
			-1,
		)
		// Check file is deleted
		assert.False(t, fixtures.FileExists(dir.Join("a", "4.txt")))
		// Check object is deleted
		assert.False(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, creationChange.File.ObjectName)))
	}

	// Check remove file existing on the index, working filesystem.dir and tree
	{
		repository.IndexFile(path.Join("3.txt"))

		idx := collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
			return change.ChangeType == directories.Modification && change.File.Filepath == dir.Join("3.txt")
		})

		assert.NotEqual(t, idx, -1)
		modificationChange := repository.index[idx]

		// Test indempontence along
		repository.RemoveFile(path.Join("3.txt"))
		repository.RemoveFile(path.Join("3.txt"))
		repository.RemoveFile(path.Join("3.txt"))

		// Check modification change is removed from the index
		assert.Equal(
			t,
			collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
				return change.ChangeType == directories.Modification && change.File.Filepath == dir.Join("3.txt")
			}),
			-1,
		)
		// Check file is deleted
		assert.False(t, fixtures.FileExists(dir.Join("3.txt")))
		// Check object is deleted
		assert.False(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, modificationChange.File.ObjectName)))
		// Check removal change is added to the index
		assert.NotEqual(
			t,
			collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
				return change.ChangeType == directories.Removal && change.Removal.Filepath == dir.Join("3.txt")
			}),
			-1,
		)
	}
}
