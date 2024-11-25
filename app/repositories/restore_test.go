package repositories

import (
	path "path/filepath"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	"github.com/stretchr/testify/assert"
	fsAssert "gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestInvalidRestore(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	assert.EqualError(t, repository.Restore("", "."), "Validation Error: invalid ref.")
	assert.EqualError(t, repository.Restore("def invalid", "."), "Validation Error: invalid ref.")
	assert.EqualError(t, repository.Restore("___", "."), "Validation Error: invalid ref.")

	assert.EqualError(t, repository.Restore("HEAD", "def-invalid-folder"), "Validation Error: invalid path.")
	assert.EqualError(t, repository.Restore("HEAD", "def-invalid-folder"), "Validation Error: invalid path.")
	assert.EqualError(t, repository.Restore("HEAD", "def-invalid-folder"), "Validation Error: invalid path.")
}

func TestRestoreHeadSingleFile(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check for file changed
	{
		// Setup
		{
			fixtures.WriteFile(dir.Join("1.txt"), []byte("the original content."))
			repository.IndexFile("1.txt")
			repository.SaveIndex()
			repository.CreateSave("initial save")

			repository = GetRepository(dir.Path())
			fixtures.WriteFile(dir.Join("1.txt"), []byte("not the original content. Saved on the index"))
			repository.IndexFile("1.txt")
			repository.SaveIndex()

			fixtures.WriteFile(dir.Join("1.txt"), []byte("someone messed up!"))
		}

		// Test
		{
			// 1) Ensure index priority (and remove files from it)
			repository = GetRepository(dir.Path())
			assert.Equal(t, len(repository.index), 1)
			assert.Equal(t, repository.index[0].File.Filepath, dir.Join("1.txt"))
			repository.Restore("HEAD", "1.txt")
			repository.SaveIndex()

			assert.Equal(t, len(repository.index), 0)
			assert.Equal(t, fixtures.ReadFile(dir.Join("1.txt")), "not the original content. Saved on the index")

			// 1) When no index files, use history file
			repository = GetRepository(dir.Path())
			// should be indempontent now
			repository.Restore("HEAD", "1.txt")
			repository.Restore("HEAD", "1.txt")
			repository.Restore("HEAD", "1.txt")
			repository.SaveIndex()

			assert.Equal(t, fixtures.ReadFile(dir.Join("1.txt")), "the original content.")
		}
	}

	// Check for file removed (should remove removal from index)
	{
		// Setup
		{
			fixtures.WriteFile(dir.Join("2.txt"), []byte("the original content."))
			repository.IndexFile("2.txt")
			repository.SaveIndex()
			repository.CreateSave("initial save")
		}

		// Test
		{
			repository = GetRepository(dir.Path())
			repository.RemoveFile("2.txt")

			assert.Equal(t, len(repository.index), 1)
			assert.Equal(t, repository.index[0].Removal.Filepath, dir.Join("2.txt"))
			assert.False(t, fixtures.FileExists(dir.Join("2.txt")))

			repository.Restore("HEAD", "2.txt")
			repository.SaveIndex()

			assert.Equal(t, len(repository.index), 0)
			assert.True(t, fixtures.FileExists(dir.Join("2.txt")))
			assert.Equal(t, fixtures.ReadFile(dir.Join("2.txt")), "the original content.")
		}
	}
}

func TestRestoreHeadDir(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check for files changed and removed - subdir (untracked files should be deleted)
	{
		// Setup
		{
			fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 original content."))
			fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 original content."))

			repository.IndexFile(dir.Join("a", "4.txt"))
			repository.IndexFile(dir.Join("a", "b", "6.txt"))
			repository.SaveIndex()
			repository.CreateSave("initial save")

			fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 updated content."))

			repository = GetRepository(dir.Path())
			repository.Restore("HEAD", "a")
		}

		// Test
		{
			fsAssert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
			fsAssert.Assert(
				t,
				fs.Equal(
					dir.Join("a"),
					fs.Expected(
						t,
						fs.WithFile("4.txt", "file 4 original content."),
						fs.WithDir("b",
							fs.WithFile("6.txt", "file 6 original content."),
						),
					),
				),
			)
		}
	}

	// Check for files changed, removed and created - root (untracked files should be deleted)
	{
		// Setup
		{
			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 original content."))
			fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 original content."))
			fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 original content."))
			fixtures.WriteFile(dir.Join("c", "8.txt"), []byte("file 8 original content."))

			repository.IndexFile(dir.Join("1.txt"))
			repository.IndexFile(dir.Join("a", "4.txt"))
			repository.IndexFile(dir.Join("a", "b", "6.txt"))
			repository.IndexFile(dir.Join("c", "8.txt"))
			repository.SaveIndex()
			repository.CreateSave("initial save")

			fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 updated content."))
			fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 updated content."))
			fixtures.WriteFile(dir.Join("newfile.txt"), []byte("new file content."))
			fixtures.MakeDirs(dir.Join("dir1"), dir.Join("dir1", "dir2"), dir.Join("dir1", "dir2", "dir3"))
			fixtures.WriteFile(dir.Join("dir1", "dir2", "dir3", "10.txt"), []byte("file 10 original content."))

			repository = GetRepository(dir.Path())
			repository.IndexFile(path.Join("dir1", "dir2", "dir3", "10.txt"))
			repository.RemoveFile(dir.Join("c", "8.txt"))
			repository.SaveIndex()

			repository = GetRepository(dir.Path())
			repository.Restore("HEAD", ".")
		}

		// Test
		{
			fsAssert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
			fsAssert.Equal(t, len(repository.index), 0)
			fsAssert.Assert(
				t,
				fs.Equal(
					dir.Path(),
					fs.Expected(
						t,
						fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
						fs.WithFile("1.txt", "file 1 original content."),
						fs.WithDir(
							"a",
							fs.WithFile("4.txt", "file 4 original content."),
							fs.WithDir("b",
								fs.WithFile("6.txt", "file 6 original content."),
							)),
						fs.WithDir("c", fs.WithFile("8.txt", "file 8 original content.")),
						fs.WithDir(
							"dir1",
							fs.WithDir(
								"dir2",
								fs.WithDir(
									"dir3",
									fs.WithFile("10.txt", "file 10 original content."),
								),
							),
						),
					),
				),
			)
		}
	}
}

func TestRestoreHeadNoHistory(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// SETUP

	repository.IndexFile(dir.Join("1.txt"))
	repository.IndexFile(dir.Join("2.txt"))
	repository.IndexFile(dir.Join("a", "4.txt"))
	repository.SaveIndex()

	fixtures.WriteFile(dir.Join("1.txt"), []byte("1 updated content"))
	fixtures.WriteFile(dir.Join("2.txt"), []byte("2 updated content"))
	fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("4 updated content"))

	repository = GetRepository(dir.Path())

	repository.Restore("HEAD", ".")

	fsAssert.Assert(
		t,
		fs.Equal(
			dir.Path(),
			fs.Expected(
				t,
				fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
				fs.WithFile("1.txt", "1 content"),
				fs.WithFile("2.txt", "2 content"),
				fs.WithDir(
					"a",
					fs.WithFile("4.txt", "4 content"),
				),
			),
		),
	)

}

func TestRestoreHistoryUnsavedChangesRootDir(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()
	var save *filesystems.Checkpoint

	// Setup
	{
		// SAVE 0
		{
			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("c", "8.txt"), []byte("file 8 (SAVE 0)."))

			repository.IndexFile(dir.Join("1.txt"))
			repository.IndexFile(dir.Join("2.txt"))
			repository.IndexFile(dir.Join("a", "4.txt"))
			repository.IndexFile(dir.Join("a", "b", "6.txt"))
			repository.IndexFile(dir.Join("c", "8.txt"))
			repository.SaveIndex()
			save, _ = repository.CreateSave("SAVE 0")
		}

		// SAVE 1
		{
			repository = GetRepository(dir.Path())

			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0) (SAVE 1)."))
			fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0) (SAVE 1)."))
			repository.IndexFile(dir.Join("1.txt"))
			repository.IndexFile(dir.Join("2.txt"))
			repository.SaveIndex()
			repository.CreateSave("SAVE 1")
		}

		// create
		fixtures.WriteFile(dir.Join("9.txt"), []byte("new file content."))
		fixtures.WriteFile(dir.Join("10.txt"), []byte("new file content."))

		// update
		fixtures.WriteFile(dir.Join("1.txt"), []byte("file updated content."))
		fixtures.WriteFile(dir.Join("2.txt"), []byte("file updated content."))

		// delete
		fixtures.RemoveFile(dir.Join("c", "8.txt"))

		repository = GetRepository(dir.Path())

		repository.IndexFile(dir.Join("9.txt"))
		repository.IndexFile(dir.Join("2.txt"))
		repository.RemoveFile(dir.Join("a", "4.txt"))
		repository.SaveIndex()
	}

	// Test Save
	{
		repository = GetRepository(dir.Path())
		repository.Restore(save.Id, ".")

		repository = GetRepository(dir.Path())

		fsAssert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
		// should keep index changes, since we are not restoring HEAD.
		fsAssert.Equal(t, len(repository.index), 3)
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "file 1 (SAVE 0)."),
					fs.WithFile("2.txt", "file 2 (SAVE 0)."),
					fs.WithDir(
						"a",
						fs.WithFile("4.txt", "file 4 (SAVE 0)."),
						fs.WithDir("b",
							fs.WithFile("6.txt", "file 6 (SAVE 0)."),
						)),
					fs.WithDir("c", fs.WithFile("8.txt", "file 8 (SAVE 0).")),
				),
			),
		)
	}

}

func TestRestoreHistoryUnsavedChangesSubdir(t *testing.T) {
	dir, repository := fixtureNewProject(t)
	defer dir.Remove()
	var save0 *filesystems.Checkpoint

	// Setup
	{
		// SAVE 0
		{
			fixtures.WriteFile(dir.Join("0.txt"), []byte("file 0 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0)."))
			fixtures.MakeDirs(dir.Join("c"))
			fixtures.WriteFile(dir.Join("c", "8.txt"), []byte("file 8 (SAVE 0)."))
			fixtures.MakeDirs(dir.Join("a"), dir.Join("a", "b"))
			fixtures.WriteFile(dir.Join("a", "3.txt"), []byte("file 3 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("a", "5.txt"), []byte("file 5 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 (SAVE 0)."))

			repository.IndexFile(dir.Join("0.txt"))
			repository.IndexFile(dir.Join("1.txt"))
			repository.IndexFile(dir.Join("2.txt"))
			repository.IndexFile(dir.Join("a", "3.txt"))
			repository.IndexFile(dir.Join("a", "4.txt"))
			repository.IndexFile(dir.Join("a", "5.txt"))
			repository.IndexFile(dir.Join("a", "b", "6.txt"))
			repository.IndexFile(dir.Join("c", "8.txt"))
			repository.SaveIndex()
			save0, _ = repository.CreateSave("SAVE 0")
		}

		// SAVE 1
		{
			repository = GetRepository(dir.Path())

			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0) (SAVE 1)."))
			fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0) (SAVE 1)."))
			repository.IndexFile(dir.Join("1.txt"))
			repository.IndexFile(dir.Join("2.txt"))
			repository.SaveIndex()
			repository.CreateSave("SAVE 1")
		}

		// Changes outside of "a" dir

		{
			// create
			fixtures.WriteFile(dir.Join("9.txt"), []byte("new file 9 content."))
			fixtures.WriteFile(dir.Join("10.txt"), []byte("new file 10 content."))

			// update
			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 updated content."))
			fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 updated content."))

			// delete
			fixtures.RemoveFile(dir.Join("c", "8.txt"))
		}

		// Changes inside "a" dir

		{
			// create
			fixtures.WriteFile(dir.Join("a", "11.txt"), []byte("new file 11 content."))
			fixtures.WriteFile(dir.Join("a", "12.txt"), []byte("new file 12 content."))

			// update
			fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 updated content."))
			fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 updated content."))

			// delete
			fixtures.RemoveFile(dir.Join("a", "5.txt"))
		}

		// Apply index
		repository = GetRepository(dir.Path())

		repository.IndexFile(dir.Join("9.txt"))
		repository.IndexFile(dir.Join("2.txt"))
		repository.RemoveFile(dir.Join("0.txt"))

		repository.IndexFile(dir.Join("a", "11.txt"))
		repository.IndexFile(dir.Join("a", "4.txt"))
		repository.RemoveFile(dir.Join("a", "3.txt"))

		repository.SaveIndex()
	}

	// Test Restore
	{
		repository = GetRepository(dir.Path())

		repository.Restore(save0.Id, "a")

		repository = GetRepository(dir.Path())

		fsAssert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
		fsAssert.Equal(t, len(repository.index), 6)
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "file 1 updated content."),
					fs.WithFile("2.txt", "file 2 updated content."),
					fs.WithDir(
						"a",
						fs.WithFile("3.txt", "file 3 (SAVE 0)."),
						fs.WithFile("4.txt", "file 4 (SAVE 0)."),
						fs.WithFile("5.txt", "file 5 (SAVE 0)."),
						fs.WithDir("b",
							fs.WithFile("6.txt", "file 6 (SAVE 0)."),
						)),
					fs.WithFile("9.txt", "new file 9 content."),
					fs.WithFile("10.txt", "new file 10 content."),
					fs.WithDir("c"),
				),
			),
		)
	}
}

func TestRestoreHistoryFileUnsavedChanges(t *testing.T) {
	dir, repository := fixtureNewProject(t)
	defer dir.Remove()
	var save *filesystems.Checkpoint

	// Setup
	{
		// SAVE 0
		{
			fixtures.WriteFile(dir.Join("0.txt"), []byte("file 0 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0)."))
			fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0)."))

			repository.IndexFile(dir.Join("0.txt"))
			repository.IndexFile(dir.Join("1.txt"))
			repository.IndexFile(dir.Join("2.txt"))
			repository.SaveIndex()
			save, _ = repository.CreateSave("SAVE 0")
		}

		// SAVE 1
		{
			repository = GetRepository(dir.Path())

			fixtures.WriteFile(dir.Join("0.txt"), []byte("file 0 (SAVE 0) (SAVE 1)."))
			fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0) (SAVE 1)."))
			fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0) (SAVE 1)."))
			repository.IndexFile(dir.Join("0.txt"))
			repository.IndexFile(dir.Join("1.txt"))
			repository.IndexFile(dir.Join("2.txt"))
			repository.SaveIndex()
			repository.CreateSave("SAVE 1")
		}

		// Create
		fixtures.WriteFile(dir.Join("3.txt"), []byte("file 3."))
		fixtures.WriteFile(dir.Join("4.txt"), []byte("file 4."))

		// Update
		fixtures.WriteFile(dir.Join("2.txt"), []byte("updated file 2."))

		// Remove
		fixtures.RemoveFile(dir.Join("0.txt"))

		// Apply index
		repository = GetRepository(dir.Path())
		repository.IndexFile(dir.Join("3.txt"))
		repository.IndexFile(dir.Join("2.txt"))
		repository.RemoveFile(dir.Join("1.txt"))
		repository.SaveIndex()
	}

	// Test Save
	{
		repository = GetRepository(dir.Path())

		fsAssert.Equal(t, len(repository.index), 3)

		GetRepository(dir.Path()).Restore(save.Id, "0.txt")
		GetRepository(dir.Path()).Restore(save.Id, "2.txt")
		GetRepository(dir.Path()).Restore(save.Id, "1.txt")

		repository = GetRepository(dir.Path())

		fsAssert.Equal(t, len(repository.index), 3)
		fsAssert.Equal(t, repository.index[0].File.Filepath, dir.Join("3.txt"))
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("0.txt", "file 0 (SAVE 0)."),
					fs.WithFile("1.txt", "file 1 (SAVE 0)."),
					fs.WithFile("2.txt", "file 2 (SAVE 0)."),
					fs.WithFile("3.txt", "file 3."),
					fs.WithFile("4.txt", "file 4."),
				),
			),
		)
	}
}

func TestRestoreHistory(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	var save0 *filesystems.Checkpoint
	var save3 *filesystems.Checkpoint
	var save5 *filesystems.Checkpoint

	// Setup
	{
		// Save 0 Changes

		fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0)."))
		fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0)."))
		fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 (SAVE 0)."))
		fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 (SAVE 0)."))
		fixtures.WriteFile(dir.Join("c", "8.txt"), []byte("file 8 (SAVE 0)."))

		fixtures.RemoveFile(dir.Join("3.txt"))
		fixtures.RemoveFile(dir.Join("a", "5.txt"))
		fixtures.RemoveFile(dir.Join("a", "b", "7.txt"))
		fixtures.RemoveFile(dir.Join("c", "9.txt"))

		repository.IndexFile(dir.Join("1.txt"))
		repository.IndexFile(dir.Join("2.txt"))
		repository.IndexFile(dir.Join("a", "4.txt"))
		repository.IndexFile(dir.Join("a", "b", "6.txt"))
		repository.IndexFile(dir.Join("c", "8.txt"))
		repository.SaveIndex()
		save0, _ = repository.CreateSave("SAVE 0")

		// Save 1 Changes

		repository = GetRepository(dir.Path())

		fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0) (SAVE 1)."))

		repository.IndexFile(dir.Join("2.txt"))
		repository.SaveIndex()
		repository.CreateSave("SAVE 1")

		// Save 2 Changes

		repository = GetRepository(dir.Path())

		fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 (SAVE 0) (SAVE 2)."))

		repository.IndexFile(dir.Join("a", "4.txt"))
		repository.SaveIndex()
		repository.CreateSave("SAVE 2")

		// Save 3 Changes

		repository = GetRepository(dir.Path())

		fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0) (SAVE 1) (SAVE 3)."))
		fixtures.MakeDirs(dir.Join("dir1"), dir.Join("dir1", "dir2"), dir.Join("dir1", "dir2", "dir3"), dir.Join("dir1", "dir2", "dir3", "dir4"))
		fixtures.WriteFile(dir.Join("dir1", "10.txt"), []byte("file 10 (SAVE 3)."))
		fixtures.WriteFile(dir.Join("dir1", "dir2", "11.txt"), []byte("file 11 (SAVE 3)."))
		fixtures.WriteFile(dir.Join("dir1", "dir2", "dir3", "12.txt"), []byte("file 12 (SAVE 3)."))
		fixtures.WriteFile(dir.Join("dir1", "dir2", "dir3", "dir4", "13.txt"), []byte("file 13 (SAVE 3)."))

		repository.IndexFile(dir.Join("2.txt"))
		repository.IndexFile(dir.Join("dir1", "10.txt"))
		repository.IndexFile(dir.Join("dir1", "dir2", "11.txt"))
		repository.IndexFile(dir.Join("dir1", "dir2", "dir3", "12.txt"))
		repository.IndexFile(dir.Join("dir1", "dir2", "dir3", "dir4", "13.txt"))
		repository.SaveIndex()
		save3, _ = repository.CreateSave("SAVE 3")

		// Save 4 Changes

		repository = GetRepository(dir.Path())

		fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 (SAVE 0) (SAVE 4)."))
		fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 (SAVE 0) (SAVE 1) (SAVE 3) (SAVE 4)."))
		fixtures.WriteFile(dir.Join("a", "4.txt"), []byte("file 4 (SAVE 0) (SAVE 2) (SAVE 4)."))
		fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 (SAVE 0) (SAVE 4)."))
		fixtures.WriteFile(dir.Join("c", "8.txt"), []byte("file 8 (SAVE 0) (SAVE 4)."))

		repository.RemoveFile(dir.Join("dir1", "10.txt"))
		repository.RemoveFile(dir.Join("dir1", "dir2", "11.txt"))
		repository.RemoveFile(dir.Join("dir1", "dir2", "dir3", "12.txt"))
		repository.RemoveFile(dir.Join("dir1", "dir2", "dir3", "dir4", "13.txt"))
		repository.IndexFile(dir.Join("1.txt"))
		repository.IndexFile(dir.Join("2.txt"))
		repository.IndexFile(dir.Join("a", "4.txt"))
		repository.IndexFile(dir.Join("a", "b", "6.txt"))
		repository.IndexFile(dir.Join("c", "8.txt"))
		repository.SaveIndex()
		repository.CreateSave("SAVE 4")

		// Save 5 Changes

		repository = GetRepository(dir.Path())

		repository.RemoveFile(dir.Join("1.txt"))
		repository.RemoveFile(dir.Join("2.txt"))
		repository.RemoveFile(dir.Join("a", "4.txt"))
		repository.SaveIndex()
		save5, _ = repository.CreateSave("SAVE 5")
	}

	// Test Save 3
	{
		repository = GetRepository(dir.Path())
		repository.Restore(save3.Id, ".")

		fsAssert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "file 1 (SAVE 0)."),
					fs.WithFile("2.txt", "file 2 (SAVE 0) (SAVE 1) (SAVE 3)."),
					fs.WithDir(
						"a",
						fs.WithFile("4.txt", "file 4 (SAVE 0) (SAVE 2)."),
						fs.WithDir("b",
							fs.WithFile("6.txt", "file 6 (SAVE 0)."),
						)),
					fs.WithDir(
						"c",
						fs.WithFile("8.txt", "file 8 (SAVE 0)."),
					),
					fs.WithDir(
						"dir1",
						fs.WithFile("10.txt", "file 10 (SAVE 3)."),
						fs.WithDir(
							"dir2",
							fs.WithFile("11.txt", "file 11 (SAVE 3)."),
							fs.WithDir(
								"dir3",
								fs.WithFile("12.txt", "file 12 (SAVE 3)."),
								fs.WithDir(
									"dir4",
									fs.WithFile("13.txt", "file 13 (SAVE 3)."),
								),
							),
						),
					),
				),
			),
		)
	}

	// Test Save 0
	{
		repository = GetRepository(dir.Path())
		repository.Restore(save0.Id, ".")

		fsAssert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "file 1 (SAVE 0)."),
					fs.WithFile("2.txt", "file 2 (SAVE 0)."),
					fs.WithDir(
						"a",
						fs.WithFile("4.txt", "file 4 (SAVE 0)."),
						fs.WithDir("b",
							fs.WithFile("6.txt", "file 6 (SAVE 0)."),
						)),
					fs.WithDir(
						"c",
						fs.WithFile("8.txt", "file 8 (SAVE 0)."),
					),
				),
			),
		)
	}

	// Test Save 5
	{
		repository = GetRepository(dir.Path())
		repository.Restore(save5.Id, ".")

		fsAssert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithDir(
						"a",
						fs.WithDir("b",
							fs.WithFile("6.txt", "file 6 (SAVE 0) (SAVE 4)."),
						)),
					fs.WithDir(
						"c",
						fs.WithFile("8.txt", "file 8 (SAVE 0) (SAVE 4)."),
					),
				),
			),
		)
	}
}
