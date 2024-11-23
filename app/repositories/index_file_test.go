package repositories

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	"github.com/stretchr/testify/assert"
	fsAssert "gotest.tools/v3/assert"
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
		errors.Check(compressor.Close())

		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Join(filesystems.REPOSITORY_FOLDER_NAME),
				fs.Expected(
					t,
					fs.WithFile(filesystems.REFS_FILE_NAME, fmt.Sprintf("Refs:\n\n%s\n\n", filesystems.INITIAL_REF_NAME)),
					fs.WithFile(filesystems.HEAD_FILE_NAME, filesystems.INITIAL_REF_NAME),
					fs.WithFile(filesystems.INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
					fs.WithDir(filesystems.SAVES_FOLDER_NAME),
					fs.WithDir(
						filesystems.OBJECTS_FOLDER_NAME,
						fs.WithFile(fileHash, buffer.String()),
					),
				),
			),
		)

		assert.EqualValues(
			t,
			repository.index,
			[]*directories.Change{
				{
					ChangeType: directories.Creation,
					File: &directories.File{
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

			fsAssert.Assert(
				t,
				fs.Equal(
					dir.Join(filesystems.REPOSITORY_FOLDER_NAME),
					fs.Expected(
						t,
						fs.WithFile(filesystems.REFS_FILE_NAME, fmt.Sprintf("Refs:\n\n%s\n\n", filesystems.INITIAL_REF_NAME)),
						fs.WithFile(filesystems.HEAD_FILE_NAME, filesystems.INITIAL_REF_NAME),
						fs.WithFile(filesystems.INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
						fs.WithDir(filesystems.SAVES_FOLDER_NAME),
						fs.WithDir(
							filesystems.OBJECTS_FOLDER_NAME,
							fs.WithFile(fileHash, buffer.String()),
						),
					),
				),
			)

			assert.EqualValues(
				t,
				repository.index,
				[]*directories.Change{
					{
						ChangeType: directories.Creation,
						File: &directories.File{
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
		defer errors.CheckFn(file.Close)

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

		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Join(filesystems.REPOSITORY_FOLDER_NAME),
				fs.Expected(
					t,
					fs.WithFile(filesystems.REFS_FILE_NAME, fmt.Sprintf("Refs:\n\n%s\n\n", filesystems.INITIAL_REF_NAME)),
					fs.WithFile(filesystems.HEAD_FILE_NAME, filesystems.INITIAL_REF_NAME),
					fs.WithFile(filesystems.INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
					fs.WithDir(filesystems.SAVES_FOLDER_NAME),
					fs.WithDir(
						filesystems.OBJECTS_FOLDER_NAME,
						fs.WithFile(fileHash, buffer.String()),
					),
				),
			),
		)

		assert.EqualValues(
			t,
			repository.index,
			[]*directories.Change{
				{
					ChangeType: directories.Creation,
					File: &directories.File{
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

		changeIdx := collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
			return change.ChangeType == directories.Modification && change.File.Filepath == dir.Join("1.txt")
		})

		assert.Equal(t, changeIdx, -1)
	}

	// Check IndexFile corner case
	// IndexFile is used to remove file from the index. If you index an unchanged tree file
	// it should remove any changes of the file stored in the index.
	{
		// Initial change reference
		var change *directories.Change

		// It should index the change flawlessly
		{
			file, err := os.OpenFile(dir.Join("1.txt"), os.O_WRONLY|os.O_TRUNC, 0644)
			errors.Check(err)
			defer errors.CheckFn(file.Close)

			_, err = file.Write([]byte("1 new content"))
			errors.Check(err)

			repository.IndexFile("1.txt")

			changeIdx := collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
				return change.ChangeType == directories.Modification && change.File.Filepath == dir.Join("1.txt")
			})

			assert.NotEqual(t, changeIdx, -1)
			change = repository.index[changeIdx]
		}

		// When updating the file to the tree file content, IndexFile should be used to remove
		// existing index. It should also remove the object from the fs.
		{
			file, err := os.OpenFile(dir.Join("1.txt"), os.O_WRONLY|os.O_TRUNC, 0644)
			errors.Check(err)
			defer errors.CheckFn(file.Close)

			_, err = file.Write([]byte("1 content"))
			errors.Check(err)

			repository.IndexFile("1.txt")

			changeIdx := collections.FindIndex(repository.index, func(change *directories.Change, _ int) bool {
				return change.ChangeType == directories.Modification && change.File.Filepath == dir.Join("1.txt")
			})

			assert.Equal(t, changeIdx, -1)
			assert.False(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, change.File.ObjectName)))
		}
	}
}

func TestIndexConflictedFile(t *testing.T) {
	dir, repository := fixtureNewProject(t)

	// to make it easier to test, index is updated in place

	// Test temporary conflict object (for some reason unchanged, so it becomes permanent)
	{
		// SETUP

		tempObjectName := "cae2c9816a64843de42b6eaea3fd3f690e529e771d7491d8f409b7687960f82f"
		fixtures.WriteFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName), []byte("x"))

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
		fixtures.WriteFile(dir.Join("a.txt"), []byte("<ref>content a.</ref><incoming>content b.</incoming>"))
		repository.IndexFile(dir.Join("a.txt"))

		assert.Equal(t, len(repository.index), 1)
		assert.Equal(t, repository.index[0].ChangeType, directories.Creation)
		assert.Equal(t, repository.index[0].File.Filepath, dir.Join("a.txt"))
		assert.Equal(t, repository.index[0].File.ObjectName, tempObjectName)
		assert.True(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName)))
	}

	// Test temporary conflict object
	{
		// SETUP

		tempObjectName := "cae2c9816a64843de42b6eaea3fd3f690e529e771d7491d8f409b7687960f82f"
		fixtures.WriteFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName), []byte("x"))

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
		fixtures.WriteFile(dir.Join("a.txt"), []byte("content a.\n"))
		repository.IndexFile(dir.Join("a.txt"))

		assert.Equal(t, len(repository.index), 1)
		assert.Equal(t, repository.index[0].ChangeType, directories.Creation)
		assert.Equal(t, repository.index[0].File.Filepath, dir.Join("a.txt"))
		assert.NotEqual(t, repository.index[0].File.ObjectName, tempObjectName)
		assert.False(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, tempObjectName)))
	}

	// Test permanent object (part of the history and therefore MUST NOT BE removed if the content changes)
	{
		// SETUP

		permanentObjectName := "cae2c9816a64843de42b6eaea3fd3f690e529e771d7491d8f409b7687960f82f"
		fixtures.WriteFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, permanentObjectName), []byte("x"))

		repository.index = []*directories.Change{
			{
				ChangeType: directories.Conflict,
				Conflict: &directories.FileConflict{
					Filepath:   dir.Join("a.txt"),
					ObjectName: permanentObjectName,
					Message:    "Removed at \"incoming\" but modified at \"ref\".",
				},
			},
		}
		repository.SaveIndex()

		// TEST
		fixtures.WriteFile(dir.Join("a.txt"), []byte("definetely the hash is going to change."))
		repository.IndexFile(dir.Join("a.txt"))

		assert.Equal(t, len(repository.index), 1)
		assert.Equal(t, repository.index[0].ChangeType, directories.Creation)
		assert.Equal(t, repository.index[0].File.Filepath, dir.Join("a.txt"))
		assert.NotEqual(t, repository.index[0].File.ObjectName, permanentObjectName)
		assert.True(t, fixtures.FileExists(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.OBJECTS_FOLDER_NAME, permanentObjectName)))
	}
}
