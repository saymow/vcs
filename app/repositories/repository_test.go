package repositories

import (
	"fmt"
	Path "path/filepath"
	"saymow/version-manager/app/repositories/directories"
	"saymow/version-manager/app/repositories/filesystems"
	"testing"

	"github.com/stretchr/testify/assert"
	fsAssert "gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestInitRepository(t *testing.T) {
	dir := fs.NewDir(t, "project")
	defer dir.Remove()

	CreateRepository(dir.Path())

	fsAssert.Assert(
		t,
		fs.Equal(
			dir.Path(),
			fs.Expected(
				t,
				fs.WithDir(
					filesystems.REPOSITORY_FOLDER_NAME,
					fs.WithFile(filesystems.REFS_FILE_NAME, fmt.Sprintf("Refs:\n\n%s\n\n", filesystems.INITAL_REF_NAME)),
					fs.WithFile(filesystems.HEAD_FILE_NAME, filesystems.INITAL_REF_NAME),
					fs.WithFile(filesystems.INDEX_FILE_NAME, "Tracked files:\r\n\r\n"),
					fs.WithDir(filesystems.SAVES_FOLDER_NAME),
					fs.WithDir(filesystems.OBJECTS_FOLDER_NAME),
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

	fsAssert.Equal(t, repository.fs.Root, dir.Path())
	fsAssert.Equal(t, repository.head, filesystems.INITAL_REF_NAME)
	assert.EqualValues(
		t,
		repository.index,
		[]*directories.Change{
			{
				ChangeType: directories.Creation,
				File: &directories.File{
					Filepath:   dir.Join("4.txt"),
					ObjectName: "814f15a360c1a700342d1652e3bd8b9c954ee2ad9c974f6ec88eb92ff2d6b3b3",
				},
			},
			{
				ChangeType: directories.Removal,
				Removal: &directories.FileRemoval{
					Filepath: dir.Join("2.txt"),
				},
			},
		},
	)
	fsAssert.Equal(t, len(repository.dir.Children), 3)
	fsAssert.Equal(t, repository.dir.Children["1.txt"].File.Filepath, dir.Join("1.txt"))
	fsAssert.Equal(t, repository.dir.Children["1.txt"].File.ObjectName, "6f6367cbecfac86af4e749156e1b1046524eff9afbd8a29c964c3b46ebdf7fc2")
	fsAssert.Equal(t, repository.dir.Children["2.txt"].File.Filepath, dir.Join("2.txt"))
	fsAssert.Equal(t, repository.dir.Children["2.txt"].File.ObjectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	fsAssert.Equal(t, repository.dir.Children["3.txt"].File.Filepath, dir.Join("3.txt"))
	fsAssert.Equal(t, repository.dir.Children["3.txt"].File.ObjectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
}

func TestResolvePath(t *testing.T) {
	dir, repository := fixtureGetBaseProject(t)
	defer dir.Remove()

	path, err := repository.resolvePath("a.txt")
	assert.Nil(t, err)
	assert.Equal(t, path, "a.txt")

	path, err = repository.resolvePath(dir.Join("a.txt"))
	assert.Nil(t, err)
	assert.Equal(t, path, "a.txt")

	path, err = repository.resolvePath(dir.Join("folder", ".."))
	assert.Nil(t, err)
	assert.Equal(t, path, "")

	path, err = repository.resolvePath(dir.Join("folder", "b.txt"))
	assert.Nil(t, err)
	assert.Equal(t, path, Path.Join("folder", "b.txt"))

	path, err = repository.resolvePath(dir.Join("folder", "..", "b.txt"))
	assert.Nil(t, err)
	assert.Equal(t, path, "b.txt")

	path, err = repository.resolvePath(dir.Join("a", "b", "c", "..", "..", "b.txt"))
	assert.Nil(t, err)
	assert.Equal(t, path, Path.Join("a", "b.txt"))

	path, err = repository.resolvePath(dir.Join("a", "b", "c"))
	assert.Nil(t, err)
	assert.Equal(t, path, Path.Join("a", "b", "c"))

	_, err = repository.resolvePath(dir.Join(".."))
	assert.Error(t, err, "invalid path.")
}
