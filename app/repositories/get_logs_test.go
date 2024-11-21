package repositories

import (
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetLogs(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	// History empty

	assert.EqualValues(
		t,
		repository.GetLogs(),
		&Log{Head: filesystems.INITIAL_REF_NAME, History: []*SaveLog{}},
	)

	// After Save 0

	fixtures.WriteFile(dir.Join("1.txt"), []byte("file 1 original content."))

	repository.IndexFile(dir.Join("1.txt"))
	repository.SaveIndex()
	save0, _ := repository.CreateSave("save0")

	log := repository.GetLogs()
	assert.Equal(t, log.Head, filesystems.INITIAL_REF_NAME)
	assert.Equal(t, len(log.History), 1)
	assert.Equal(t, len(log.History[0].Refs), 1)
	assert.Equal(t, log.History[0].Refs[0], filesystems.INITIAL_REF_NAME)
	assert.Equal(t, log.History[0].Checkpoint.Id, save0.Id)
	assert.Equal(t, log.History[0].Checkpoint.Message, save0.Message)
	assert.Equal(t, log.History[0].Checkpoint.Parent, save0.Parent)
	// When saving the time in the file, using the Layout format, we lose the ms precision.
	// Therefore this is needed to compare times
	assert.Equal(t, log.History[0].Checkpoint.CreatedAt.Format(time.Layout), save0.CreatedAt.Format(time.Layout))
	assert.EqualValues(t, log.History[0].Checkpoint.Changes, save0.Changes)

	// Create Ref

	repository.CreateRef("a")

	// After Save 1

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("2.txt"), []byte("file 2 original content."))

	repository.IndexFile(dir.Join("2.txt"))
	repository.SaveIndex()
	save1, _ := repository.CreateSave("save1")

	log = repository.GetLogs()
	assert.Equal(t, log.Head, "a")
	assert.Equal(t, len(log.History), 2)
	assert.Equal(t, len(log.History[0].Refs), 1)
	assert.Equal(t, log.History[0].Refs[0], "a")
	assert.Equal(t, log.History[0].Checkpoint.Id, save1.Id)
	assert.Equal(t, log.History[0].Checkpoint.Message, save1.Message)
	assert.Equal(t, log.History[0].Checkpoint.Parent, save1.Parent)
	// When saving the time in the file, using the Layout format, we lose the ms precision.
	// Therefore this is needed to compare times
	assert.Equal(t, log.History[0].Checkpoint.CreatedAt.Format(time.Layout), save1.CreatedAt.Format(time.Layout))
	assert.EqualValues(t, log.History[0].Checkpoint.Changes, save1.Changes)

	assert.Equal(t, len(log.History[1].Refs), 1)
	assert.Equal(t, log.History[1].Refs[0], filesystems.INITIAL_REF_NAME)
	assert.Equal(t, log.History[1].Checkpoint.Id, save0.Id)
	assert.Equal(t, log.History[1].Checkpoint.Message, save0.Message)
	assert.Equal(t, log.History[1].Checkpoint.Parent, save0.Parent)
	// When saving the time in the file, using the Layout format, we lose the ms precision.
	// Therefore this is needed to compare times
	assert.Equal(t, log.History[1].Checkpoint.CreatedAt.Format(time.Layout), save0.CreatedAt.Format(time.Layout))
	assert.EqualValues(t, log.History[1].Checkpoint.Changes, save0.Changes)

	// After Save 2

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("3.txt"), []byte("file 3 original content."))

	repository.IndexFile(dir.Join("3.txt"))
	repository.SaveIndex()
	save2, _ := repository.CreateSave("save2")

	// Create refs

	repository.CreateRef("b")
	repository.CreateRef("c")

	log = repository.GetLogs()
	assert.Equal(t, log.Head, "c")
	assert.Equal(t, len(log.History), 3)
	assert.Equal(t, len(log.History[0].Refs), 3)
	assert.Equal(t, log.History[0].Checkpoint.Id, save2.Id)
	assert.Equal(t, log.History[0].Checkpoint.Message, save2.Message)
	assert.Equal(t, log.History[0].Checkpoint.Parent, save2.Parent)
	// When saving the time in the file, using the Layout format, we lose the ms precision.
	// Therefore this is needed to compare times
	assert.Equal(t, log.History[0].Checkpoint.CreatedAt.Format(time.Layout), save2.CreatedAt.Format(time.Layout))
	assert.EqualValues(t, log.History[0].Checkpoint.Changes, save2.Changes)

	assert.Equal(t, len(log.History[1].Refs), 0)
	assert.Equal(t, log.History[1].Checkpoint.Id, save1.Id)
	assert.Equal(t, log.History[1].Checkpoint.Message, save1.Message)
	assert.Equal(t, log.History[1].Checkpoint.Parent, save1.Parent)
	// When saving the time in the file, using the Layout format, we lose the ms precision.
	// Therefore this is needed to compare times
	assert.Equal(t, log.History[1].Checkpoint.CreatedAt.Format(time.Layout), save1.CreatedAt.Format(time.Layout))
	assert.EqualValues(t, log.History[1].Checkpoint.Changes, save1.Changes)

	assert.Equal(t, len(log.History[2].Refs), 1)
	assert.Equal(t, log.History[2].Refs[0], filesystems.INITIAL_REF_NAME)
	assert.Equal(t, log.History[2].Checkpoint.Id, save0.Id)
	assert.Equal(t, log.History[2].Checkpoint.Message, save0.Message)
	assert.Equal(t, log.History[2].Checkpoint.Parent, save0.Parent)
	// When saving the time in the file, using the Layout format, we lose the ms precision.
	// Therefore this is needed to compare times
	assert.Equal(t, log.History[2].Checkpoint.CreatedAt.Format(time.Layout), save0.CreatedAt.Format(time.Layout))
	assert.EqualValues(t, log.History[2].Checkpoint.Changes, save0.Changes)
}
