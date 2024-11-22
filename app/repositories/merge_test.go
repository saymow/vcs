package repositories

import (
	"fmt"
	Path "path/filepath"
	"saymow/version-manager/app/pkg/fixtures"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	"github.com/stretchr/testify/assert"
	fsAssert "gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

type BaseRepositoryMeta struct {
	s0      *filesystems.Checkpoint
	s1      *filesystems.Checkpoint
	s2      *filesystems.Checkpoint
	refName string
}

func makeBaseRepository(t *testing.T) (*fs.Dir, *Repository, *BaseRepositoryMeta) {
	dir, repository := fixtureNewProject(t)

	fixtures.WriteFile(dir.Join("a.txt"), []byte("a.txt content."))
	fixtures.WriteFile(dir.Join("b.txt"), []byte("b.txt content."))

	repository.IndexFile("a.txt")
	repository.IndexFile("b.txt")
	repository.SaveIndex()
	s0, _ := repository.CreateSave("s0")
	repository.CreateRef("ref")

	repository = GetRepository(dir.Path())

	fixtures.MakeDirs(dir.Join("a"))
	fixtures.WriteFile(dir.Join("a", "a.txt"), []byte("a/a.txt content."))

	repository.IndexFile(Path.Join("a", "a.txt"))
	repository.SaveIndex()
	s1, _ := repository.CreateSave("s1")

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("b.txt"), []byte("b.txt updated content."))
	fixtures.WriteFile(dir.Join("a", "b.txt"), []byte("b/b.txt content."))

	repository.IndexFile(Path.Join("b.txt"))
	repository.IndexFile(Path.Join("a", "b.txt"))
	repository.RemoveFile(Path.Join("a.txt"))
	repository.SaveIndex()
	s2, _ := repository.CreateSave("s1")

	return dir,
		GetRepository(dir.Path()),
		&BaseRepositoryMeta{s0: s0, s1: s1, s2: s2, refName: "ref"}
}

func TestInvalidMerge(t *testing.T) {
	dir, repository, meta := makeBaseRepository(t)
	defer dir.Remove()

	repository.Load(meta.s0.Id)

	_, err := repository.Merge(meta.refName)
	assert.Error(t, err, "Validaton Error: cannot make changes in detached mode.")

	repository = GetRepository(dir.Path())
	repository.Load(filesystems.INITIAL_REF_NAME)

	_, err = repository.Merge("undefined")
	assert.Error(t, err, "Validaton Error: invalid ref.")

	fixtures.WriteFile(dir.Join("new_file.txt"), []byte("new file original content."))

	_, err = repository.Merge(meta.refName)
	assert.Error(t, err, "Validaton Error: unsaved changes.")

	repository.IndexFile(dir.Join("new_file.txt"))

	_, err = repository.Merge(meta.refName)
	assert.Error(t, err, "Validaton Error: unsaved changes.")
}

func TestFastForwardMerge(t *testing.T) {
	dir, repository, meta := makeBaseRepository(t)
	defer dir.Remove()

	repository.Load(filesystems.INITIAL_REF_NAME)

	repository = GetRepository(dir.Path())

	saveCheckpoint, err := repository.Merge(meta.refName)
	refs := repository.GetRefs()

	assert.Nil(t, err)
	assert.Equal(t, saveCheckpoint.Id, meta.s2.Id)
	assert.Equal(t, repository.head, filesystems.INITIAL_REF_NAME)
	assert.Equal(t, refs[filesystems.INITIAL_REF_NAME], refs[meta.refName])
	assert.Equal(t, refs[meta.refName], meta.s2.Id)
	fsAssert.Assert(
		t,
		fs.Equal(
			dir.Path(),
			fs.Expected(
				t,
				fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
				fs.WithFile("b.txt", "b.txt updated content."),
				fs.WithDir(
					"a",
					fs.WithFile("a.txt", "a/a.txt content."),
					fs.WithFile("b.txt", "b/b.txt content."),
				),
			),
		),
	)
}

func TestNewSaveMerge(t *testing.T) {
	dir, repository, meta := makeBaseRepository(t)
	defer dir.Remove()
	incoming := "incoming"

	// Setup

	// Create incoming ref
	repository.CreateRef(incoming)

	// s1

	fixtures.WriteFile(dir.Join("c.txt"), []byte("c.txt incoming content."))

	repository.IndexFile(dir.Join("c.txt"))
	repository.SaveIndex()
	repository.CreateSave("s1")

	// s2

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("a", "c.txt"), []byte("a/c.txt incoming content."))
	fixtures.WriteFile(dir.Join("a", "b.txt"), []byte("a/b.txt incoming updated content."))
	fixtures.MakeDirs(dir.Join("a", "b"))
	fixtures.WriteFile(dir.Join("a", "b", "c.txt"), []byte("a/b/c.txt incoming content."))
	fixtures.WriteFile(dir.Join("a", "b", "d.txt"), []byte("a/b/d.txt incoming content."))

	repository.IndexFile(dir.Join("a", "c.txt"))
	repository.IndexFile(dir.Join("a", "b.txt"))
	repository.IndexFile(dir.Join("a", "b", "c.txt"))
	repository.IndexFile(dir.Join("a", "b", "d.txt"))
	repository.SaveIndex()
	repository.CreateSave("s2")

	// s3

	repository = GetRepository(dir.Path())

	fixtures.MakeDirs(dir.Join("c"))
	fixtures.WriteFile(dir.Join("c", "a.txt"), []byte("c/a.txt incoming content."))

	repository.IndexFile(dir.Join("c", "a.txt"))
	repository.SaveIndex()
	s3, _ := repository.CreateSave("s2")

	// Load ref

	repository = GetRepository(dir.Path())

	repository.Load(meta.refName)

	// s1'

	repository = GetRepository(dir.Path())

	fixtures.WriteFile(dir.Join("a.txt"), []byte("a.txt ref content."))
	fixtures.WriteFile(dir.Join("b.txt"), []byte("b.txt ref updated content."))
	fixtures.WriteFile(dir.Join("a", "a.txt"), []byte("a/a.txt ref updated content."))

	repository.IndexFile(dir.Join("a.txt"))
	repository.IndexFile(dir.Join("b.txt"))
	repository.IndexFile(dir.Join("a", "a.txt"))
	repository.SaveIndex()
	s1Prime, _ := repository.CreateSave("s1'")

	// Test

	repository = GetRepository(dir.Path())
	saveCheckpoint, err := repository.Merge(incoming)
	refs := repository.GetRefs()

	assert.Nil(t, err)
	assert.NotEqual(t, saveCheckpoint.Id, s3)
	assert.NotEqual(t, saveCheckpoint.Id, s1Prime.Id)
	assert.Equal(t, saveCheckpoint.Message, fmt.Sprintf("Merge \"%s\" at \"%s\".", incoming, meta.refName))
	assert.Equal(t, repository.head, meta.refName)
	assert.Equal(t, refs[incoming], s3.Id)
	assert.Equal(t, refs[meta.refName], saveCheckpoint.Id)
	assert.Equal(t, saveCheckpoint.Parent, s1Prime.Id)
	fsAssert.Assert(
		t,
		fs.Equal(
			dir.Path(),
			fs.Expected(
				t,
				fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
				fs.WithFile("a.txt", "a.txt ref content."),
				fs.WithFile("b.txt", "b.txt ref updated content."),
				fs.WithFile("c.txt", "c.txt incoming content."),
				fs.WithDir(
					"a",
					fs.WithFile("a.txt", "a/a.txt ref updated content."),
					fs.WithFile("b.txt", "a/b.txt incoming updated content."),
					fs.WithFile("c.txt", "a/c.txt incoming content."),
					fs.WithDir(
						"b",
						fs.WithFile("c.txt", "a/b/c.txt incoming content."),
						fs.WithFile("d.txt", "a/b/d.txt incoming content."),
					),
				),
				fs.WithDir(
					"c",
					fs.WithFile("a.txt", "c/a.txt incoming content."),
				),
			),
		),
	)

	// Check if the file tree is not corrupted
	{
		repository = GetRepository(dir.Path())

		// Load older versions
		repository.Load(meta.s0.Id)

		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("a.txt", "a.txt content."),
					fs.WithFile("b.txt", "b.txt content."),
				),
			),
		)

		repository = GetRepository(dir.Path())

		// Load merge save
		repository.Load(saveCheckpoint.Id)

		fsAssert.Assert(
			t,
			fs.Equal(
				dir.Path(),
				fs.Expected(
					t,
					fs.WithDir(filesystems.REPOSITORY_FOLDER_NAME, fs.MatchExtraFiles),
					fs.WithFile("a.txt", "a.txt ref content."),
					fs.WithFile("b.txt", "b.txt ref updated content."),
					fs.WithFile("c.txt", "c.txt incoming content."),
					fs.WithDir(
						"a",
						fs.WithFile("a.txt", "a/a.txt ref updated content."),
						fs.WithFile("b.txt", "a/b.txt incoming updated content."),
						fs.WithFile("c.txt", "a/c.txt incoming content."),
						fs.WithDir(
							"b",
							fs.WithFile("c.txt", "a/b/c.txt incoming content."),
							fs.WithFile("d.txt", "a/b/d.txt incoming content."),
						),
					),
					fs.WithDir(
						"c",
						fs.WithFile("a.txt", "c/a.txt incoming content."),
					),
				),
			),
		)
	}
}
