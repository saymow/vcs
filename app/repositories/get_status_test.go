package repositories

import (
	path "path/filepath"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/directories"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStatus(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Base case
	{
		repository.IndexFile("1.txt")
		repository.IndexFile(path.Join("a", "4.txt"))
		repository.IndexFile(path.Join("a", "b", "6.txt"))
		repository.IndexFile(path.Join("c", "8.txt"))
		repository.IndexFile(path.Join("c", "9.txt"))
		repository.SaveIndex()
		repository.CreateSave("initial save")

		repository = GetRepository(dir.Path())

		repository.IndexFile("2.txt")
		fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("4 new content"))
		repository.IndexFile(path.Join("a", "4.txt"))
		repository.RemoveFile(path.Join("a", "b", "6.txt"))
		repository.SaveIndex()

		repository = GetRepository(dir.Path())

		fixtures.WriteFile(dir.Join("c", "8.txt"), []byte("8 new content"))
		fixtures.RemoveFile(dir.Join("c", "9.txt"))

		status := repository.GetStatus()

		assert.EqualValues(t, status.Staged.CreatedFilesPaths, []string{dir.Join("2.txt")})
		assert.EqualValues(t, status.Staged.ModifiedFilePaths, []string{dir.Join("a", "4.txt")})
		assert.EqualValues(t, status.Staged.RemovedFilePaths, []string{dir.Join("a", "b", "6.txt")})
		assert.EqualValues(t, status.WorkingDir.ModifiedFilePaths, []string{dir.Join("c", "8.txt")})
		assert.EqualValues(t, status.WorkingDir.RemovedFilePaths, []string{dir.Join("c", "9.txt")})
		assert.EqualValues(
			t,
			status.WorkingDir.UntrackedFilePaths,
			[]string{dir.Join("3.txt"), dir.Join("a", "5.txt"), dir.Join("a", "b", "7.txt")},
		)
	}

	// Conflicted index
	{
		// To make it easier to test, i'm updating the index in place
		repository.index = append(repository.index, &directories.Change{
			ChangeType: directories.Conflict,
			Conflict: &directories.FileConflict{
				Filepath:   dir.Join("1.txt"),
				Message:    "Conflict.",
				ObjectName: "definitely-not-a-real-hash",
			},
		})

		fixtures.WriteFile(dir.Join("1.txt"), []byte("it is definitely gonna fix the conflict."))

		status := repository.GetStatus()

		assert.EqualValues(t, len(status.Staged.ConflictedFilesPaths), 1)
		assert.EqualValues(t, status.Staged.ConflictedFilesPaths[0].Filepath, dir.Join("1.txt"))
		assert.EqualValues(t, status.Staged.ConflictedFilesPaths[0].Message, "Conflict.")
		assert.EqualValues(t, status.Staged.CreatedFilesPaths, []string{dir.Join("2.txt")})
		assert.EqualValues(t, status.Staged.ModifiedFilePaths, []string{dir.Join("a", "4.txt")})
		assert.EqualValues(t, status.Staged.RemovedFilePaths, []string{dir.Join("a", "b", "6.txt")})
		assert.EqualValues(t, status.WorkingDir.ModifiedFilePaths, []string{dir.Join("1.txt"), dir.Join("c", "8.txt")})
		assert.EqualValues(t, status.WorkingDir.RemovedFilePaths, []string{dir.Join("c", "9.txt")})
		assert.EqualValues(
			t,
			status.WorkingDir.UntrackedFilePaths,
			[]string{dir.Join("3.txt"), dir.Join("a", "5.txt"), dir.Join("a", "b", "7.txt")},
		)
	}
}
