package repositories

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	path "path/filepath"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/directory"
	"saymow/version-manager/app/repositories/filesystem"
	"testing"
	"time"

	testifyAssert "github.com/stretchr/testify/assert"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestInitRepository(t *testing.T) {
	dir := fs.NewDir(t, "project")
	defer dir.Remove()

	CreateRepository(dir.Path())

	assert.Assert(
		t,
		fs.Equal(
			dir.Path(),
			fs.Expected(
				t,
				fs.WithDir(
					".repository",
					fs.WithFile("head", ""),
					fs.WithFile("index", "Tracked files:\r\n\r\n"),
					fs.WithDir("saves"),
					fs.WithDir("objects"),
				),
			),
		),
	)
}

func TestGetRepository(t *testing.T) {
	dir := fs.NewDir(t, "project")
	defer dir.Remove()

	fs.Apply(
		t,
		dir,
		fixtureMakeBasicRepositoryFs(dir),
	)

	repository := GetRepository(dir.Path())

	assert.Equal(t, repository.fs.Root, dir.Path())
	assert.Equal(t, repository.head, "3f674c71a3596db8f24fd31a85c503ae600898cc03810fcc171781d4f35531d2")
	testifyAssert.EqualValues(
		t,
		repository.index,
		[]*directory.Change{
			{
				ChangeType: directory.Creation,
				File: &directory.File{
					Filepath:   dir.Join("4.txt"),
					ObjectName: "814f15a360c1a700342d1652e3bd8b9c954ee2ad9c974f6ec88eb92ff2d6b3b3",
				},
			},
			{
				ChangeType: directory.Removal,
				Removal: &directory.FileRemoval{
					Filepath: dir.Join("2.txt"),
				},
			},
		},
	)
	assert.Equal(t, len(repository.dir.Children), 3)
	assert.Equal(t, repository.dir.Children["1.txt"].File.Filepath, dir.Join("1.txt"))
	assert.Equal(t, repository.dir.Children["1.txt"].File.ObjectName, "6f6367cbecfac86af4e749156e1b1046524eff9afbd8a29c964c3b46ebdf7fc2")
	assert.Equal(t, repository.dir.Children["2.txt"].File.Filepath, dir.Join("2.txt"))
	assert.Equal(t, repository.dir.Children["2.txt"].File.ObjectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	assert.Equal(t, repository.dir.Children["3.txt"].File.Filepath, dir.Join("3.txt"))
	assert.Equal(t, repository.dir.Children["3.txt"].File.ObjectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
}

func TestIndexFile(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check index file base case
	{
		repository.IndexFile("1.txt")

		hasher := sha256.New()
		hasher.Write([]byte("1 content"))
		fileHash := hex.EncodeToString(hasher.Sum(nil))

		var buffer bytes.Buffer
		compressor := gzip.NewWriter(&buffer)
		compressor.Write([]byte("1 content"))
		compressor.Close()

		assert.Assert(
			t,
			fs.Equal(
				dir.Join(filesystem.REPOSITORY_FOLDER_NAME),
				fs.Expected(
					t,
					fs.WithFile(filesystem.HEAD_FILE_NAME, ""),
					fs.WithFile(filesystem.INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
					fs.WithDir(filesystem.SAVES_FOLDER_NAME),
					fs.WithDir(
						filesystem.OBJECTS_FOLDER_NAME,
						fs.WithFile(fileHash, buffer.String()),
					),
				),
			),
		)

		testifyAssert.EqualValues(
			t,
			repository.index,
			[]*directory.Change{
				{
					ChangeType: directory.Creation,
					File: &directory.File{
						Filepath:   dir.Join("1.txt"),
						ObjectName: fileHash,
					},
				},
			})

		// Check indempotence
		{
			repository.IndexFile("1.txt")
			repository.IndexFile("1.txt")
			repository.IndexFile("1.txt")

			assert.Assert(
				t,
				fs.Equal(
					dir.Join(filesystem.REPOSITORY_FOLDER_NAME),
					fs.Expected(
						t,
						fs.WithFile(filesystem.HEAD_FILE_NAME, ""),
						fs.WithFile(filesystem.INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
						fs.WithDir(filesystem.SAVES_FOLDER_NAME),
						fs.WithDir(
							filesystem.OBJECTS_FOLDER_NAME,
							fs.WithFile(fileHash, buffer.String()),
						),
					),
				),
			)

			testifyAssert.EqualValues(
				t,
				repository.index,
				[]*directory.Change{
					{
						ChangeType: directory.Creation,
						File: &directory.File{
							Filepath:   dir.Join("1.txt"),
							ObjectName: fileHash,
						},
					},
				},
			)
		}
	}

	// Check Update file index object
	{
		file, err := os.OpenFile(dir.Join("1.txt"), os.O_WRONLY|os.O_TRUNC, 0644)
		errors.Check(err)

		_, err = file.Write([]byte("1 new content"))
		errors.Check(err)

		repository.IndexFile("1.txt")

		hasher := sha256.New()
		hasher.Write([]byte("1 new content"))
		fileHash := hex.EncodeToString(hasher.Sum(nil))

		var buffer bytes.Buffer
		compressor := gzip.NewWriter(&buffer)
		compressor.Write([]byte("1 new content"))
		compressor.Close()

		assert.Assert(
			t,
			fs.Equal(
				dir.Join(filesystem.REPOSITORY_FOLDER_NAME),
				fs.Expected(
					t,
					fs.WithFile(filesystem.HEAD_FILE_NAME, ""),
					fs.WithFile(filesystem.INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
					fs.WithDir(filesystem.SAVES_FOLDER_NAME),
					fs.WithDir(
						filesystem.OBJECTS_FOLDER_NAME,
						fs.WithFile(fileHash, buffer.String()),
					),
				),
			),
		)

		testifyAssert.EqualValues(
			t,
			repository.index,
			[]*directory.Change{
				{
					ChangeType: directory.Creation,
					File: &directory.File{
						Filepath:   dir.Join("1.txt"),
						ObjectName: fileHash,
					},
				},
			},
		)
	}
}

func TestIndexFileComplexCases(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	// Check try to index unchaged tree file
	{
		repository.IndexFile("1.txt")
		repository.IndexFile("1.txt")
		repository.IndexFile("1.txt")

		changeIdx := collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
			return change.ChangeType == directory.Modification && change.File.Filepath == dir.Join("1.txt")
		})

		testifyAssert.Equal(t, changeIdx, -1)
	}

	// Check IndexFile corner case
	// IndexFile is used to remove file from the index. If you index an unchanged tree file
	// it should remove any changes of the file stored in the index.
	{
		// Initial change reference
		var change *directory.Change

		// It should index the change flawlessly
		{
			file, err := os.OpenFile(dir.Join("1.txt"), os.O_WRONLY|os.O_TRUNC, 0644)
			errors.Check(err)

			_, err = file.Write([]byte("1 new content"))
			errors.Check(err)

			repository.IndexFile("1.txt")

			changeIdx := collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
				return change.ChangeType == directory.Modification && change.File.Filepath == dir.Join("1.txt")
			})

			testifyAssert.NotEqual(t, changeIdx, -1)
			change = repository.index[changeIdx]
		}

		// When updating the file to the tree file content, IndexFile should be used to remove
		// existing index. It should also remove the object from the fs.
		{
			file, err := os.OpenFile(dir.Join("1.txt"), os.O_WRONLY|os.O_TRUNC, 0644)
			errors.Check(err)

			_, err = file.Write([]byte("1 content"))
			errors.Check(err)

			repository.IndexFile("1.txt")

			changeIdx := collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
				return change.ChangeType == directory.Modification && change.File.Filepath == dir.Join("1.txt")
			})

			testifyAssert.Equal(t, changeIdx, -1)
			testifyAssert.False(t, fixtures.FileExists(dir.Join(filesystem.REPOSITORY_FOLDER_NAME, filesystem.OBJECTS_FOLDER_NAME, change.File.ObjectName)))
		}
	}
}

func TestRemoveFile(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	// Check it should only remove from working directory if file is not being tracked
	{
		// Test indempontence along
		repository.RemoveFile(path.Join("a", "5.txt"))
		repository.RemoveFile(path.Join("a", "5.txt"))
		repository.RemoveFile(path.Join("a", "5.txt"))

		changeIdx := collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
			return change.ChangeType == directory.Removal && change.Removal.Filepath == dir.Join("a", "5.txt")
		})
		testifyAssert.Equal(t, changeIdx, -1)
		testifyAssert.False(t, fixtures.FileExists(dir.Join("a", "5.txt")))
	}

	// Check remove file base case (existing only on the tree and working dir)
	{
		// Test indempontence along
		repository.RemoveFile("1.txt")
		repository.RemoveFile("1.txt")
		repository.RemoveFile("1.txt")

		changeIdx := collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
			return change.ChangeType == directory.Removal && change.Removal.Filepath == dir.Join("1.txt")
		})
		testifyAssert.NotEqual(t, changeIdx, -1)
		testifyAssert.False(t, fixtures.FileExists(dir.Join("1.txt")))
	}

	// Check remove file base case (existing only on the index and working dir)
	{
		repository.IndexFile(path.Join("a", "4.txt"))

		idx := collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
			return change.ChangeType == directory.Creation && change.File.Filepath == dir.Join("a", "4.txt")
		})

		testifyAssert.NotEqual(t, idx, -1)
		creationChange := repository.index[idx]

		// Test indempontence along
		repository.RemoveFile(path.Join("a", "4.txt"))
		repository.RemoveFile(path.Join("a", "4.txt"))
		repository.RemoveFile(path.Join("a", "4.txt"))

		// Check modification change is removed from the index
		testifyAssert.Equal(
			t,
			collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
				return change.ChangeType == directory.Modification && change.File.Filepath == dir.Join("a", "4.txt")
			}),
			-1,
		)
		// Check file is deleted
		testifyAssert.False(t, fixtures.FileExists(dir.Join("a", "4.txt")))
		// Check object is deleted
		testifyAssert.False(t, fixtures.FileExists(dir.Join(filesystem.REPOSITORY_FOLDER_NAME, filesystem.OBJECTS_FOLDER_NAME, creationChange.File.ObjectName)))
	}

	// Check remove file existing on the index, working filesystem.dir and tree
	{
		repository.IndexFile(path.Join("3.txt"))

		idx := collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
			return change.ChangeType == directory.Modification && change.File.Filepath == dir.Join("3.txt")
		})

		testifyAssert.NotEqual(t, idx, -1)
		modificationChange := repository.index[idx]

		// Test indempontence along
		repository.RemoveFile(path.Join("3.txt"))
		repository.RemoveFile(path.Join("3.txt"))
		repository.RemoveFile(path.Join("3.txt"))

		// Check modification change is removed from the index
		testifyAssert.Equal(
			t,
			collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
				return change.ChangeType == directory.Modification && change.File.Filepath == dir.Join("3.txt")
			}),
			-1,
		)
		// Check file is deleted
		testifyAssert.False(t, fixtures.FileExists(dir.Join("3.txt")))
		// Check object is deleted
		testifyAssert.False(t, fixtures.FileExists(dir.Join(filesystem.REPOSITORY_FOLDER_NAME, filesystem.OBJECTS_FOLDER_NAME, modificationChange.File.ObjectName)))
		// Check removal change is added to the index
		testifyAssert.NotEqual(
			t,
			collections.FindIndex(repository.index, func(change *directory.Change, _ int) bool {
				return change.ChangeType == directory.Removal && change.Removal.Filepath == dir.Join("3.txt")
			}),
			-1,
		)
	}
}

func TestSaveIndex(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check empty index
	{
		repository.SaveIndex()

		fileContent := fixtures.ReadFile(dir.Join(filesystem.REPOSITORY_FOLDER_NAME, filesystem.INDEX_FILE_NAME))

		assert.Equal(t, fileContent, "Tracked files:\n\n")
	}

	// Check non empty index
	{
		repository.index = append(
			repository.index,
			&directory.Change{ChangeType: directory.Modification, File: &directory.File{Filepath: dir.Join("1.txt"), ObjectName: "1.txt-object"}},
			&directory.Change{ChangeType: directory.Modification, File: &directory.File{Filepath: dir.Join("a", "b", "6.txt"), ObjectName: "6.txt-object"}},
			&directory.Change{ChangeType: directory.Removal, Removal: &directory.FileRemoval{Filepath: dir.Join("a", "b", "5.txt")}},
			&directory.Change{ChangeType: directory.Modification, File: &directory.File{Filepath: dir.Join("a", "b", "7.txt"), ObjectName: "7.txt-object"}},
			&directory.Change{ChangeType: directory.Modification, File: &directory.File{Filepath: dir.Join("a", "b", "c", "8.txt"), ObjectName: "8.txt-object"}},
			&directory.Change{ChangeType: directory.Removal, Removal: &directory.FileRemoval{Filepath: dir.Join("a", "b", "c", "9.txt")}},
		)

		repository.SaveIndex()

		received := fixtures.ReadFile(dir.Join(filesystem.REPOSITORY_FOLDER_NAME, filesystem.INDEX_FILE_NAME))
		expected := `Tracked files:

%s	(modified)
1.txt-object
%s	(modified)
6.txt-object
%s	(removed)
%s	(modified)
7.txt-object
%s	(modified)
8.txt-object
%s	(removed)
`

		assert.Equal(
			t,
			received,
			fmt.Sprintf(
				expected,
				dir.Join("1.txt"),
				dir.Join("a", "b", "6.txt"),
				dir.Join("a", "b", "5.txt"),
				dir.Join("a", "b", "7.txt"),
				dir.Join("a", "b", "c", "8.txt"),
				dir.Join("a", "b", "c", "9.txt"),
			),
		)

		// Check index updates
		{
			repository.index = []*directory.Change{repository.index[0], repository.index[2], repository.index[4]}

			repository.SaveIndex()

			received := fixtures.ReadFile(dir.Join(filesystem.REPOSITORY_FOLDER_NAME, filesystem.INDEX_FILE_NAME))
			expected := `Tracked files:

%s	(modified)
1.txt-object
%s	(removed)
%s	(modified)
8.txt-object
`

			assert.Equal(
				t,
				received,
				fmt.Sprintf(
					expected,
					dir.Join("1.txt"),
					dir.Join("a", "b", "5.txt"),
					dir.Join("a", "b", "c", "8.txt"),
				),
			)
		}
	}
}

func TestCreateSave(t *testing.T) {
	dir, _ := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check initial save

	indexFilepath := dir.Join(filesystem.REPOSITORY_FOLDER_NAME, filesystem.INDEX_FILE_NAME)
	index := fmt.Sprintf(`Tracked files:
	
%s	(modified)
1.txt-object
%s	(modified)
4.txt-object
%s	(modified)
6.txt-object
`, dir.Join("1.txt"),
		dir.Join("a", "4.txt"),
		dir.Join("a", "b", "6.txt"),
	)
	fixtures.WriteFile(indexFilepath, []byte(index))

	repository := GetRepository(dir.Path())
	firstSave := repository.CreateSave("first save")
	expectedFirstSaveFileContent := fmt.Sprintf(`%s

%s

Please do not edit the lines below.


Files:

%s	(modified)
%s
%s	(modified)
%s
%s	(modified)
%s
`,
		firstSave.Message,
		firstSave.CreatedAt.Format(time.Layout),
		firstSave.Changes[0].File.Filepath,
		firstSave.Changes[0].File.ObjectName,
		firstSave.Changes[1].File.Filepath,
		firstSave.Changes[1].File.ObjectName,
		firstSave.Changes[2].File.Filepath,
		firstSave.Changes[2].File.ObjectName,
	)

	testifyAssert.Equal(t, firstSave.Message, "first save")
	testifyAssert.Equal(t, firstSave.Parent, "")
	testifyAssert.EqualValues(
		t,
		firstSave.Changes,
		[]*directory.Change{
			{
				ChangeType: directory.Modification,
				File: &directory.File{
					Filepath:   dir.Join("1.txt"),
					ObjectName: "1.txt-object",
				},
			},
			{
				ChangeType: directory.Modification,
				File: &directory.File{
					Filepath:   dir.Join("a", "4.txt"),
					ObjectName: "4.txt-object",
				},
			},
			{
				ChangeType: directory.Modification,
				File: &directory.File{
					Filepath:   dir.Join("a", "b", "6.txt"),
					ObjectName: "6.txt-object",
				},
			},
		},
	)
	assert.Assert(
		t,
		fs.Equal(
			dir.Join(filesystem.REPOSITORY_FOLDER_NAME),
			fs.Expected(
				t,
				fs.WithFile(filesystem.HEAD_FILE_NAME, firstSave.Id),
				fs.WithFile(filesystem.INDEX_FILE_NAME, "Tracked files:\n\n"),
				fs.WithDir(filesystem.SAVES_FOLDER_NAME,
					fs.WithFile(firstSave.Id, expectedFirstSaveFileContent),
				),
				fs.WithDir(filesystem.OBJECTS_FOLDER_NAME),
			),
		),
	)

	// Check second save

	index = fmt.Sprintf(`Tracked files:
	
%s	(removed)
%s	(removed)
%s	(modified)
8.txt-object
`, dir.Join("1.txt"),
		dir.Join("a", "4.txt"),
		dir.Join("a", "b", "c", "8.txt"),
	)
	fixtures.WriteFile(indexFilepath, []byte(index))

	repository = GetRepository(dir.Path())
	secondSave := repository.CreateSave("second save")
	expectedSecondSaveFileContent := fmt.Sprintf(`%s
%s
%s

Please do not edit the lines below.


Files:

%s	(removed)
%s	(removed)
%s	(modified)
%s
`,
		secondSave.Message,
		secondSave.Parent,
		secondSave.CreatedAt.Format(time.Layout),
		secondSave.Changes[0].Removal.Filepath,
		secondSave.Changes[1].Removal.Filepath,
		secondSave.Changes[2].File.Filepath,
		secondSave.Changes[2].File.ObjectName,
	)

	testifyAssert.Equal(t, secondSave.Message, "second save")
	testifyAssert.Equal(t, secondSave.Parent, firstSave.Id)
	testifyAssert.EqualValues(
		t,
		secondSave.Changes,
		[]*directory.Change{
			{
				ChangeType: directory.Removal,
				Removal:    &directory.FileRemoval{Filepath: dir.Join("1.txt")},
			},
			{
				ChangeType: directory.Removal,
				Removal:    &directory.FileRemoval{Filepath: dir.Join("a", "4.txt")},
			},
			{
				ChangeType: directory.Modification,
				File: &directory.File{
					Filepath:   dir.Join("a", "b", "c", "8.txt"),
					ObjectName: "8.txt-object",
				},
			},
		},
	)
	assert.Assert(
		t,
		fs.Equal(
			dir.Join(filesystem.REPOSITORY_FOLDER_NAME),
			fs.Expected(
				t,
				fs.WithFile(filesystem.HEAD_FILE_NAME, secondSave.Id),
				fs.WithFile(filesystem.INDEX_FILE_NAME, "Tracked files:\n\n"),
				fs.WithDir(filesystem.SAVES_FOLDER_NAME,
					fs.WithFile(firstSave.Id, expectedFirstSaveFileContent),
					fs.WithFile(secondSave.Id, expectedSecondSaveFileContent),
				),
				fs.WithDir(filesystem.OBJECTS_FOLDER_NAME),
			),
		),
	)
}

func TestGetStatus(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

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

	testifyAssert.EqualValues(t, status.Staged.CreatedFilesPaths, []string{dir.Join("2.txt")})
	testifyAssert.EqualValues(t, status.Staged.ModifiedFilePaths, []string{dir.Join("a", "4.txt")})
	testifyAssert.EqualValues(t, status.Staged.RemovedFilePaths, []string{dir.Join("a", "b", "6.txt")})
	testifyAssert.EqualValues(t, status.WorkingDir.ModifiedFilePaths, []string{dir.Join("c", "8.txt")})
	testifyAssert.EqualValues(t, status.WorkingDir.RemovedFilePaths, []string{dir.Join("c", "9.txt")})
	testifyAssert.EqualValues(
		t,
		status.WorkingDir.UntrackedFilePaths,
		[]string{dir.Join("3.txt"), dir.Join("a", "5.txt"), dir.Join("a", "b", "7.txt")},
	)
}

func TestLoadInvalidRef(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	testifyAssert.EqualError(t, repository.Load("", "."), "Validation Error: \"\" is an invalid ref.")
	testifyAssert.EqualError(t, repository.Load("def invalid", "."), "Validation Error: \"def invalid\" is an invalid ref.")
	testifyAssert.EqualError(t, repository.Load("___", "."), "Validation Error: \"___\" is an invalid ref.")
}

func TestLoadHeadSingleFile(t *testing.T) {
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
			testifyAssert.Equal(t, len(repository.index), 1)
			testifyAssert.Equal(t, repository.index[0].File.Filepath, dir.Join("1.txt"))
			repository.Load("HEAD", "1.txt")
			repository.SaveIndex()

			testifyAssert.Equal(t, len(repository.index), 0)
			testifyAssert.Equal(t, fixtures.ReadFile(dir.Join("1.txt")), "not the original content. Saved on the index")

			// 1) When no index files, use history file
			repository = GetRepository(dir.Path())
			// should be indempontent now
			repository.Load("HEAD", "1.txt")
			repository.Load("HEAD", "1.txt")
			repository.Load("HEAD", "1.txt")
			repository.SaveIndex()

			testifyAssert.Equal(t, fixtures.ReadFile(dir.Join("1.txt")), "the original content.")
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

			testifyAssert.Equal(t, len(repository.index), 1)
			testifyAssert.Equal(t, repository.index[0].Removal.Filepath, dir.Join("2.txt"))
			testifyAssert.False(t, fixtures.FileExists(dir.Join("2.txt")))

			repository.Load("HEAD", "2.txt")
			repository.SaveIndex()

			testifyAssert.Equal(t, len(repository.index), 0)
			testifyAssert.True(t, fixtures.FileExists(dir.Join("2.txt")))
			testifyAssert.Equal(t, fixtures.ReadFile(dir.Join("2.txt")), "the original content.")
		}
	}
}

func TestLoadHeadDir(t *testing.T) {
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
			repository.Load("HEAD", "a")
		}

		// Test
		{
			assert.Assert(
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
			repository.Load("HEAD", ".")
		}

		// Test
		{
			assert.Equal(t, len(repository.index), 0)
			assert.Assert(
				t,
				fs.Equal(
					dir.Path(),
					fs.Expected(
						t,
						fs.WithDir(filesystem.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
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

func TestLoadHistoryUnsavedChanges(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()
	var save *filesystem.CheckPoint

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
			save = repository.CreateSave("SAVE 0")
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

		fixtures.WriteFile(dir.Join("a", "b", "6.txt"), []byte("file 6 updated content."))
		fixtures.WriteFile(dir.Join("newfile.txt"), []byte("new file content."))
	}

	// Test Save
	{
		repository = GetRepository(dir.Path())
		testifyAssert.EqualError(t, repository.Load(save.Id, "."), "Validation Error: you have unsaved changes in the working dir, you have to save them before loading a save.")
	}
}

func TestLoadHistory(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Check for files changed and removed - root (untracked files should be deleted)
	{
		var save0 *filesystem.CheckPoint
		var save3 *filesystem.CheckPoint
		var save5 *filesystem.CheckPoint

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
			save0 = repository.CreateSave("SAVE 0")

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
			save3 = repository.CreateSave("SAVE 3")

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
			save5 = repository.CreateSave("SAVE 5")
		}

		// Test Save 3
		{
			repository = GetRepository(dir.Path())
			repository.Load(save3.Id, ".")

			assert.Equal(t, repository.head, save3.Id)
			assert.Assert(
				t,
				fs.Equal(
					dir.Path(),
					fs.Expected(
						t,
						fs.WithDir(filesystem.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
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
			repository.Load(save0.Id, ".")

			assert.Equal(t, repository.head, save0.Id)
			assert.Assert(
				t,
				fs.Equal(
					dir.Path(),
					fs.Expected(
						t,
						fs.WithDir(filesystem.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
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
			repository.Load(save5.Id, ".")

			assert.Equal(t, repository.head, save5.Id)
			assert.Assert(
				t,
				fs.Equal(
					dir.Path(),
					fs.Expected(
						t,
						fs.WithDir(filesystem.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
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
}
