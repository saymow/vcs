package repositories

import (
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRefs(t *testing.T) {
	dir, repository := fixtureGetCustomProject(t, fixtureMakeBasicRepositoryFs)
	defer dir.Remove()

	// Multiples refs to the same save
	// Setup
	fixtures.WriteFile(dir.Join("new.txt"), []byte("it does matter."))

	repository.IndexFile("new.txt")
	repository.SaveIndex()
	save0, _ := repository.CreateSave("save message")

	repository.CreateRef("feat/a")
	repository.CreateRef("feat/b")

	// Test
	repository = GetRepository(dir.Path())
	assert.Equal(
		t,
		fixtures.ReadFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.HEAD_FILE_NAME)),
		"feat/b",
	)
	assert.EqualValues(t, repository.GetRefs(), map[string]string{
		"master": save0.Id,
		"feat/a": save0.Id,
		"feat/b": save0.Id,
	})

	// Save (move current save as a side effect) and create refs
	{
		// Setup
		repository = GetRepository(dir.Path())

		fixtures.WriteFile(dir.Join("new.txt"), []byte("it does not matter."))

		repository.IndexFile("new.txt")
		repository.SaveIndex()
		save1, _ := repository.CreateSave("save message")

		// Test
		repository.CreateRef("feat/c")

		assert.Equal(
			t,
			fixtures.ReadFile(dir.Join(filesystems.REPOSITORY_FOLDER_NAME, filesystems.HEAD_FILE_NAME)),
			"feat/c",
		)
		assert.EqualValues(t, repository.GetRefs(), map[string]string{
			"master": save0.Id,
			"feat/a": save0.Id,
			"feat/b": save1.Id,
			"feat/c": save1.Id,
		})

		// Setup
		repository = GetRepository(dir.Path())

		fixtures.WriteFile(dir.Join("new.txt"), []byte("it does not matter 2.0."))

		repository.IndexFile("new.txt")
		repository.SaveIndex()
		lastSave, _ := repository.CreateSave("save message")

		assert.EqualValues(t, repository.GetRefs(), map[string]string{
			"master": save0.Id,
			"feat/a": save0.Id,
			"feat/b": save1.Id,
			"feat/c": lastSave.Id,
		})
	}
}
