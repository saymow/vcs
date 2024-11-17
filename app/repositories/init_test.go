package repositories

import (
	"testing"

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

	assert.Equal(t, repository.root, dir.Path())
	assert.Equal(t, repository.head, "3f674c71a3596db8f24fd31a85c503ae600898cc03810fcc171781d4f35531d2")
	testifyAssert.EqualValues(
		t,
		repository.index,
		[]*Change{
			{
				changeType: Creation,
				file: &File{
					filepath:   dir.Join("4.txt"),
					objectName: "814f15a360c1a700342d1652e3bd8b9c954ee2ad9c974f6ec88eb92ff2d6b3b3",
				},
			},
			{
				changeType: Removal,
				removal: &FileRemoval{
					filepath: dir.Join("2.txt"),
				},
			},
		},
	)
	assert.Equal(t, len(repository.dir.children), 3)
	assert.Equal(t, repository.dir.children["1.txt"].file.filepath, dir.Join("1.txt"))
	assert.Equal(t, repository.dir.children["1.txt"].file.objectName, "6f6367cbecfac86af4e749156e1b1046524eff9afbd8a29c964c3b46ebdf7fc2")
	assert.Equal(t, repository.dir.children["2.txt"].file.filepath, dir.Join("2.txt"))
	assert.Equal(t, repository.dir.children["2.txt"].file.objectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	assert.Equal(t, repository.dir.children["3.txt"].file.filepath, dir.Join("3.txt"))
	assert.Equal(t, repository.dir.children["3.txt"].file.objectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
}
