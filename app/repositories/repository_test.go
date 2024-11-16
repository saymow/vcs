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
	"testing"
	"time"

	testifyAssert "github.com/stretchr/testify/assert"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

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
				dir.Join(REPOSITORY_FOLDER_NAME),
				fs.Expected(
					t,
					fs.WithFile(HEAD_FILE_NAME, ""),
					fs.WithFile(INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
					fs.WithDir(SAVES_FOLDER_NAME),
					fs.WithDir(
						OBJECTS_FOLDER_NAME,
						fs.WithFile(fileHash, buffer.String()),
					),
				),
			),
		)

		testifyAssert.EqualValues(
			t,
			repository.index,
			[]*Change{
				{
					changeType: Modified,
					modified: &File{
						filepath:   dir.Join("1.txt"),
						objectName: fileHash,
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
					dir.Join(REPOSITORY_FOLDER_NAME),
					fs.Expected(
						t,
						fs.WithFile(HEAD_FILE_NAME, ""),
						fs.WithFile(INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
						fs.WithDir(SAVES_FOLDER_NAME),
						fs.WithDir(
							OBJECTS_FOLDER_NAME,
							fs.WithFile(fileHash, buffer.String()),
						),
					),
				),
			)

			testifyAssert.EqualValues(
				t,
				repository.index,
				[]*Change{
					{
						changeType: Modified,
						modified: &File{
							filepath:   dir.Join("1.txt"),
							objectName: fileHash,
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
				dir.Join(REPOSITORY_FOLDER_NAME),
				fs.Expected(
					t,
					fs.WithFile(HEAD_FILE_NAME, ""),
					fs.WithFile(INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
					fs.WithDir(SAVES_FOLDER_NAME),
					fs.WithDir(
						OBJECTS_FOLDER_NAME,
						fs.WithFile(fileHash, buffer.String()),
					),
				),
			),
		)

		testifyAssert.EqualValues(
			t,
			repository.index,
			[]*Change{
				{
					changeType: Modified,
					modified: &File{
						filepath:   dir.Join("1.txt"),
						objectName: fileHash,
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

		changeIdx := collections.FindIndex(repository.index, func(change *Change, _ int) bool {
			return change.changeType == Modified && change.modified.filepath == dir.Join("1.txt")
		})

		testifyAssert.Equal(t, changeIdx, -1)
	}

	// Check IndexFile corner case
	// IndexFile is used to remove file from the index. If you index an unchanged tree file
	// it should remove any changes of the file stored in the index.
	{
		// Initial change reference
		var change *Change

		// It should index the change flawlessly
		{
			file, err := os.OpenFile(dir.Join("1.txt"), os.O_WRONLY|os.O_TRUNC, 0644)
			errors.Check(err)

			_, err = file.Write([]byte("1 new content"))
			errors.Check(err)

			repository.IndexFile("1.txt")

			changeIdx := collections.FindIndex(repository.index, func(change *Change, _ int) bool {
				return change.changeType == Modified && change.modified.filepath == dir.Join("1.txt")
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

			changeIdx := collections.FindIndex(repository.index, func(change *Change, _ int) bool {
				return change.changeType == Modified && change.modified.filepath == dir.Join("1.txt")
			})

			testifyAssert.Equal(t, changeIdx, -1)
			testifyAssert.False(t, fixtureFileExists(dir.Join(REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, change.modified.objectName)))
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

		changeIdx := collections.FindIndex(repository.index, func(change *Change, _ int) bool {
			return change.changeType == Removal && change.removal.filepath == dir.Join("a", "5.txt")
		})
		testifyAssert.Equal(t, changeIdx, -1)
		testifyAssert.False(t, fixtureFileExists(dir.Join("a", "5.txt")))
	}

	// Check remove file base case (existing only on the tree and working dir)
	{
		// Test indempontence along
		repository.RemoveFile("1.txt")
		repository.RemoveFile("1.txt")
		repository.RemoveFile("1.txt")

		changeIdx := collections.FindIndex(repository.index, func(change *Change, _ int) bool {
			return change.changeType == Removal && change.removal.filepath == dir.Join("1.txt")
		})
		testifyAssert.NotEqual(t, changeIdx, -1)
		testifyAssert.False(t, fixtureFileExists(dir.Join("1.txt")))
	}

	// Check remove file base case (existing only on the index and working dir)
	{
		repository.IndexFile(path.Join("a", "4.txt"))

		idx := collections.FindIndex(repository.index, func(change *Change, _ int) bool {
			return change.changeType == Modified && change.modified.filepath == dir.Join("a", "4.txt")
		})

		testifyAssert.NotEqual(t, idx, -1)
		modificationChange := repository.index[idx]

		// Test indempontence along
		repository.RemoveFile(path.Join("a", "4.txt"))
		repository.RemoveFile(path.Join("a", "4.txt"))
		repository.RemoveFile(path.Join("a", "4.txt"))

		// Check modification change is removed from the index
		testifyAssert.Equal(
			t,
			collections.FindIndex(repository.index, func(change *Change, _ int) bool {
				return change.changeType == Modified && change.modified.filepath == dir.Join("a", "4.txt")
			}),
			-1,
		)
		// Check file is deleted
		testifyAssert.False(t, fixtureFileExists(dir.Join("a", "4.txt")))
		// Check object is deleted
		testifyAssert.False(t, fixtureFileExists(dir.Join(REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, modificationChange.modified.objectName)))
	}

	// Check remove file existing on the index, working dir and tree
	{
		repository.IndexFile(path.Join("3.txt"))

		idx := collections.FindIndex(repository.index, func(change *Change, _ int) bool {
			return change.changeType == Modified && change.modified.filepath == dir.Join("3.txt")
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
			collections.FindIndex(repository.index, func(change *Change, _ int) bool {
				return change.changeType == Modified && change.modified.filepath == dir.Join("3.txt")
			}),
			-1,
		)
		// Check file is deleted
		testifyAssert.False(t, fixtureFileExists(dir.Join("3.txt")))
		// Check object is deleted
		testifyAssert.False(t, fixtureFileExists(dir.Join(REPOSITORY_FOLDER_NAME, OBJECTS_FOLDER_NAME, modificationChange.modified.objectName)))
		// Check removal change is added to the index
		testifyAssert.NotEqual(
			t,
			collections.FindIndex(repository.index, func(change *Change, _ int) bool {
				return change.changeType == Removal && change.removal.filepath == dir.Join("3.txt")
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

		fileContent := fixtureReadFile(dir.Join(REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))

		assert.Equal(t, fileContent, "Tracked files:\n\n")
	}

	// Check non empty index
	{
		repository.index = append(
			repository.index,
			&Change{changeType: Modified, modified: &File{filepath: dir.Join("1.txt"), objectName: "1.txt-object"}},
			&Change{changeType: Modified, modified: &File{filepath: dir.Join("a", "b", "6.txt"), objectName: "6.txt-object"}},
			&Change{changeType: Removal, removal: &FileRemoval{filepath: dir.Join("a", "b", "5.txt")}},
			&Change{changeType: Modified, modified: &File{filepath: dir.Join("a", "b", "7.txt"), objectName: "7.txt-object"}},
			&Change{changeType: Modified, modified: &File{filepath: dir.Join("a", "b", "c", "8.txt"), objectName: "8.txt-object"}},
			&Change{changeType: Removal, removal: &FileRemoval{filepath: dir.Join("a", "b", "c", "9.txt")}},
		)

		repository.SaveIndex()

		received := fixtureReadFile(dir.Join(REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
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
			repository.index = []*Change{repository.index[0], repository.index[2], repository.index[4]}

			repository.SaveIndex()

			received := fixtureReadFile(dir.Join(REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME))
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

	indexFilepath := dir.Join(REPOSITORY_FOLDER_NAME, INDEX_FILE_NAME)
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
	fixtureWriteFile(indexFilepath, []byte(index))

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
		firstSave.message,
		firstSave.createdAt.Format(time.Layout),
		firstSave.changes[0].modified.filepath,
		firstSave.changes[0].modified.objectName,
		firstSave.changes[1].modified.filepath,
		firstSave.changes[1].modified.objectName,
		firstSave.changes[2].modified.filepath,
		firstSave.changes[2].modified.objectName,
	)

	testifyAssert.Equal(t, firstSave.message, "first save")
	testifyAssert.Equal(t, firstSave.parent, "")
	testifyAssert.EqualValues(
		t,
		firstSave.changes,
		[]*Change{
			{
				changeType: Modified,
				modified: &File{
					filepath:   dir.Join("1.txt"),
					objectName: "1.txt-object",
				},
			},
			{
				changeType: Modified,
				modified: &File{
					filepath:   dir.Join("a", "4.txt"),
					objectName: "4.txt-object",
				},
			},
			{
				changeType: Modified,
				modified: &File{
					filepath:   dir.Join("a", "b", "6.txt"),
					objectName: "6.txt-object",
				},
			},
		},
	)
	assert.Assert(
		t,
		fs.Equal(
			dir.Join(REPOSITORY_FOLDER_NAME),
			fs.Expected(
				t,
				fs.WithFile(HEAD_FILE_NAME, firstSave.id),
				fs.WithFile(INDEX_FILE_NAME, "Tracked files:\n\n"),
				fs.WithDir(SAVES_FOLDER_NAME,
					fs.WithFile(firstSave.id, expectedFirstSaveFileContent),
				),
				fs.WithDir(OBJECTS_FOLDER_NAME),
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
	fixtureWriteFile(indexFilepath, []byte(index))

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
		secondSave.message,
		secondSave.parent,
		secondSave.createdAt.Format(time.Layout),
		secondSave.changes[0].removal.filepath,
		secondSave.changes[1].removal.filepath,
		secondSave.changes[2].modified.filepath,
		secondSave.changes[2].modified.objectName,
	)

	testifyAssert.Equal(t, secondSave.message, "second save")
	testifyAssert.Equal(t, secondSave.parent, firstSave.id)
	testifyAssert.EqualValues(
		t,
		secondSave.changes,
		[]*Change{
			{
				changeType: Removal,
				removal:    &FileRemoval{dir.Join("1.txt")},
			},
			{
				changeType: Removal,
				removal:    &FileRemoval{dir.Join("a", "4.txt")},
			},
			{
				changeType: Modified,
				modified: &File{
					filepath:   dir.Join("a", "b", "c", "8.txt"),
					objectName: "8.txt-object",
				},
			},
		},
	)
	assert.Assert(
		t,
		fs.Equal(
			dir.Join(REPOSITORY_FOLDER_NAME),
			fs.Expected(
				t,
				fs.WithFile(HEAD_FILE_NAME, secondSave.id),
				fs.WithFile(INDEX_FILE_NAME, "Tracked files:\n\n"),
				fs.WithDir(SAVES_FOLDER_NAME,
					fs.WithFile(firstSave.id, expectedFirstSaveFileContent),
					fs.WithFile(secondSave.id, expectedSecondSaveFileContent),
				),
				fs.WithDir(OBJECTS_FOLDER_NAME),
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
	fixtureWriteFile(dir.Join("a", "4.txt"), []byte("4 new content"))
	repository.IndexFile(path.Join("a", "4.txt"))
	repository.RemoveFile(path.Join("a", "b", "6.txt"))
	repository.SaveIndex()

	repository = GetRepository(dir.Path())

	fixtureWriteFile(dir.Join("c", "8.txt"), []byte("8 new content"))
	fixtureRemoveFile(dir.Join("c", "9.txt"))

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

func TestRestoreInvalidRef(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	testifyAssert.EqualError(t, repository.Restore("", "."), "Validation Error: \"\" is an invalid ref.")
	testifyAssert.EqualError(t, repository.Restore("def invalid", "."), "Validation Error: \"def invalid\" is an invalid ref.")
	testifyAssert.EqualError(t, repository.Restore("___", "."), "Validation Error: \"___\" is an invalid ref.")
}

func TestRestoreSingleFile(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	testifyAssert.EqualError(t, repository.Restore("", "."), "Validation Error: \"\" is an invalid ref.")
	testifyAssert.EqualError(t, repository.Restore("def invalid", "."), "Validation Error: \"def invalid\" is an invalid ref.")
	testifyAssert.EqualError(t, repository.Restore("___", "."), "Validation Error: \"___\" is an invalid ref.")
}
