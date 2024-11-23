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

func TestRemoveConflictedFile(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	// to make it easier to test, index is updated in place

	// Test temporary conflict object
	{
		// SETUP

		// Save file in history

		fixtures.WriteFile(dir.Join("a.txt"), []byte("content a."))

		repository.IndexFile(dir.Join("a.txt"))
		repository.SaveIndex()
		repository.CreateSave("s0")

		repository = GetRepository(dir.Path())

		// Mock a merge conflict

		tempObjectName := "cae2c9816a64843de42b6eaea3fd3f690e529e771d7491d8f409b7687960f82f"
		fixtures.WriteFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName), []byte("x"))
		fixtures.WriteFile(dir.Join("a.txt"), []byte("<ref>content a.</ref><incoming>content b.</incoming>"))

		repository.index = []*directories.Change{
			{
				ChangeType: directories.Conflict,
				Conflict: &directories.FileConflict{
					Filepath:   dir.Join("a.txt"),
					ObjectName: tempObjectName,
					Message:    "Conflict.",
				},
			},
		}
		repository.SaveIndex()

		// TEST

		repository.RemoveFile(dir.Join("a.txt"))

		assert.Equal(t, len(repository.index), 1)
		assert.Equal(t, repository.index[0].ChangeType, directories.Removal)
		assert.Equal(t, repository.index[0].Removal.Filepath, dir.Join("a.txt"))
		assert.False(t, fixtures.FileExists(dir.Join("a.txt")))
		assert.False(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName)))
	}

	// Test permanent conflict object (MUST NOT BE REMOVED)
	{
		// SETUP

		// Mock a merge conflict

		tempObjectName := "b221ff6068cbe899726392b3e24f71ca743107b2e986601fa94429835509f662"
		fixtures.WriteFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName), []byte("x"))
		fixtures.WriteFile(dir.Join("a.txt"), []byte("incoming content."))

		repository.index = []*directories.Change{
			{
				ChangeType: directories.Conflict,
				Conflict: &directories.FileConflict{
					Filepath:   dir.Join("a.txt"),
					ObjectName: tempObjectName,
					Message:    "Removed at \"ref\" but modified at \"incoming\".",
				},
			},
		}
		repository.SaveIndex()

		// TEST

		repository.RemoveFile(dir.Join("a.txt"))

		assert.Equal(t, len(repository.index), 1)
		assert.Equal(t, repository.index[0].ChangeType, directories.Removal)
		assert.Equal(t, repository.index[0].Removal.Filepath, dir.Join("a.txt"))
		assert.False(t, fixtures.FileExists(dir.Join("a.txt")))
		assert.True(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName)))
	}
}
