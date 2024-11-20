package repositories

import (
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	"github.com/stretchr/testify/assert"
	fsAssert "gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestInvalidLoad(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	assert.EqualError(t, repository.Load(""), "Validation Error: invalid ref.")
	assert.EqualError(t, repository.Load("____"), "Validation Error: invalid ref.")
	assert.EqualError(t, repository.Load("invalid"), "Validation Error: invalid ref.")

	fixtures.WriteFile(dir.Join("1.txt"), []byte("1 updated content."))
	fixtures.WriteFile(dir.Join("2.txt"), []byte("2 updated content."))
	fixtures.RemoveFile(dir.Join("a", "4.txt"))

	assert.EqualError(t, repository.Load("9a35bd416196f27e40f4f9e4768496ef29c1922f0ab5e2651a218e4d4cb09688"), "Validation Error: unsaved changes.")
}

func TestLoad(t *testing.T) {
	dir, repository := fixtureNewProject(t)
	defer dir.Remove()

	// Setup

	// Save 0

	fixtures.WriteFile(dir.Join("1.txt"), []byte("1 content."))
	fixtures.WriteFile(dir.Join("2.txt"), []byte("2 content."))
	fixtures.WriteFile(dir.Join("3.txt"), []byte("3 content."))
	fixtures.MakeDirs(dir.Join("a"))
	fixtures.WriteFile(dir.Join("a", "1.txt"), []byte("1 content."))
	fixtures.WriteFile(dir.Join("a", "2.txt"), []byte("2 content."))
	fixtures.WriteFile(dir.Join("a", "3.txt"), []byte("3 content."))
	fixtures.MakeDirs(dir.Join("b"))
	fixtures.WriteFile(dir.Join("b", "1.txt"), []byte("1 content."))
	fixtures.WriteFile(dir.Join("b", "2.txt"), []byte("2 content."))
	fixtures.WriteFile(dir.Join("b", "3.txt"), []byte("3 content."))

	repository.IndexFile(dir.Join("1.txt"))
	repository.IndexFile(dir.Join("2.txt"))
	repository.IndexFile(dir.Join("3.txt"))
	repository.IndexFile(dir.Join("a", "1.txt"))
	repository.IndexFile(dir.Join("a", "2.txt"))
	repository.IndexFile(dir.Join("a", "3.txt"))
	repository.IndexFile(dir.Join("b", "1.txt"))
	repository.IndexFile(dir.Join("b", "2.txt"))
	repository.IndexFile(dir.Join("b", "3.txt"))
	repository.SaveIndex()
	save0, _ := repository.CreateSave("save0")

	// Save 1

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("1.txt"), []byte("1 updated content."))
	fixtures.MakeDirs(dir.Join("c"))
	fixtures.WriteFile(dir.Join("c", "1.txt"), []byte("1 content."))
	fixtures.WriteFile(dir.Join("c", "2.txt"), []byte("2 content."))
	fixtures.WriteFile(dir.Join("c", "3.txt"), []byte("3 content."))

	repository.IndexFile(dir.Join("1.txt"))
	repository.IndexFile(dir.Join("c", "1.txt"))
	repository.IndexFile(dir.Join("c", "2.txt"))
	repository.IndexFile(dir.Join("c", "3.txt"))
	repository.RemoveFile(dir.Join("a", "1.txt"))
	repository.RemoveFile(dir.Join("a", "2.txt"))
	repository.RemoveFile(dir.Join("a", "3.txt"))
	repository.SaveIndex()
	save1, _ := repository.CreateSave("save2")

	// Save 2

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("2.txt"), []byte("2 updated content."))
	fixtures.MakeDirs(dir.Join("d"))
	fixtures.WriteFile(dir.Join("d", "1.txt"), []byte("1 content."))
	fixtures.WriteFile(dir.Join("d", "2.txt"), []byte("2 content."))
	fixtures.WriteFile(dir.Join("d", "3.txt"), []byte("3 content."))

	repository.IndexFile(dir.Join("2.txt"))
	repository.IndexFile(dir.Join("d", "1.txt"))
	repository.IndexFile(dir.Join("d", "2.txt"))
	repository.IndexFile(dir.Join("d", "3.txt"))
	repository.RemoveFile(dir.Join("b", "1.txt"))
	repository.RemoveFile(dir.Join("b", "2.txt"))
	repository.RemoveFile(dir.Join("b", "3.txt"))
	repository.SaveIndex()
	save2, _ := repository.CreateSave("save2")

	// Save 2

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("3.txt"), []byte("3 updated content."))
	fixtures.MakeDirs(dir.Join("e"))
	fixtures.WriteFile(dir.Join("e", "1.txt"), []byte("1 content."))
	fixtures.WriteFile(dir.Join("e", "2.txt"), []byte("2 content."))
	fixtures.WriteFile(dir.Join("e", "3.txt"), []byte("3 content."))

	repository.IndexFile(dir.Join("3.txt"))
	repository.IndexFile(dir.Join("e", "1.txt"))
	repository.IndexFile(dir.Join("e", "2.txt"))
	repository.IndexFile(dir.Join("e", "3.txt"))
	repository.RemoveFile(dir.Join("c", "1.txt"))
	repository.RemoveFile(dir.Join("c", "2.txt"))
	repository.RemoveFile(dir.Join("c", "3.txt"))
	repository.SaveIndex()
	repository.CreateSave("save3")

	// Tests

	// Load Save 0
	{

		repository = GetRepository(dir.Path())
		repository.Load(save0.Id)

		assert.Equal(t, repository.head, save0.Id)
		assert.True(t, repository.isDetachedMode())
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "1 content."),
					fs.WithFile("2.txt", "2 content."),
					fs.WithFile("3.txt", "3 content."),
					fs.WithDir(
						"a",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
					fs.WithDir(
						"b",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
				),
			),
		)
	}

	// Load Save 1
	{

		repository = GetRepository(dir.Path())
		repository.Load(save1.Id)

		assert.Equal(t, repository.head, save1.Id)
		assert.True(t, repository.isDetachedMode())
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "1 updated content."),
					fs.WithFile("2.txt", "2 content."),
					fs.WithFile("3.txt", "3 content."),
					fs.WithDir(
						"b",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
					fs.WithDir(
						"c",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
				),
			),
		)
	}

	// Load Save 2
	{

		repository = GetRepository(dir.Path())
		repository.Load(save2.Id)

		assert.Equal(t, repository.head, save2.Id)
		assert.True(t, repository.isDetachedMode())
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "1 updated content."),
					fs.WithFile("2.txt", "2 updated content."),
					fs.WithFile("3.txt", "3 content."),
					fs.WithDir(
						"c",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
					fs.WithDir(
						"d",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
				),
			),
		)
	}

	// Load Save 3 (using ref)
	{

		repository = GetRepository(dir.Path())
		repository.Load(filesystems.INITAL_REF_NAME)

		assert.Equal(t, repository.head, filesystems.INITAL_REF_NAME)
		assert.False(t, repository.isDetachedMode())
		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("1.txt", "1 updated content."),
					fs.WithFile("2.txt", "2 updated content."),
					fs.WithFile("3.txt", "3 updated content."),
					fs.WithDir(
						"d",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
					fs.WithDir(
						"e",
						fs.WithFile("1.txt", "1 content."),
						fs.WithFile("2.txt", "2 content."),
						fs.WithFile("3.txt", "3 content."),
					),
				),
			),
		)
	}
}
