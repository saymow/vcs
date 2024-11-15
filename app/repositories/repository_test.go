package repositories

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"saymow/version-manager/app/pkg/collections"
	"saymow/version-manager/app/pkg/errors"
	"testing"

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
					fs.WithFile("head", ""),
					fs.WithFile("index", "Tracked files:\r\n\r\n"),
					fs.WithDir("saves"),
					fs.WithDir(
						"objects",
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
						fs.WithFile("head", ""),
						fs.WithFile("index", "Tracked files:\r\n\r\n"),
						fs.WithDir("saves"),
						fs.WithDir(
							"objects",
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
					fs.WithFile("head", ""),
					fs.WithFile("index", "Tracked files:\r\n\r\n"),
					fs.WithDir("saves"),
					fs.WithDir(
						"objects",
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
