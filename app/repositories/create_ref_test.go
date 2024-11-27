package repositories

import (
	"saymow/version-manager/app/pkg/fixtures"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidRef(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// Cannot create refs when there is no save history
	{
		assert.Error(t, repository.CreateRef("feat/a"), "cannot create refs when there is no save history.")
		assert.EqualValues(t, repository.GetRefs().Refs, map[string]string{
			"master": "",
		})
	}

	// Name already in use
	{
		fixtures.WriteFile(dir.Join("new.txt"), []byte("it does not matter."))

		repository.IndexFile("new.txt")
		repository.SaveIndex()
		save0, _ := repository.CreateSave("save message")

		repository = GetRepository(dir.Path())

		repository.CreateRef("feat/a")

		assert.EqualValues(t, repository.GetRefs().Refs, map[string]string{
			"master": save0.Id,
			"feat/a": save0.Id,
		})

		fixtures.WriteFile(dir.Join("new.txt"), []byte("it does not matter even more."))

		repository.IndexFile("new.txt")
		repository.SaveIndex()
		save1, _ := repository.CreateSave("save message")

		assert.Error(t, repository.CreateRef("master"), "name already in use.")
		assert.EqualValues(t, repository.GetRefs().Refs, map[string]string{
			"master": save0.Id,
			"feat/a": save1.Id,
		})
	}
}
