package repositories

import (
	"fmt"
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
		fs.WithDir(
			".repository",
			fs.WithDir(
				"saves",
				fs.WithFile(
					"9a35bd416196f27e40f4f9e4768496ef29c1922f0ab5e2651a218e4d4cb09688",
					fmt.Sprintf(`initial save

11/15 04:08:58PM '24 -0300

Please do not edit the lines below.


Files:

%s	(modified)
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
%s	(modified)
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
`, dir.Join("a.txt"), dir.Join("b.txt")),
				),
				fs.WithFile(
					"3f674c71a3596db8f24fd31a85c503ae600898cc03810fcc171781d4f35531d2",
					fmt.Sprintf(`second save
9a35bd416196f27e40f4f9e4768496ef29c1922f0ab5e2651a218e4d4cb09688
11/15 04:09:54PM '24 -0300

Please do not edit the lines below.


Files:

%s	(modified)
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
%s	(modified)
18b7cb099a9ea3f50ba899b5ba81e0d377a5f3b16f8f6eeb8b3e58cd4692b993`, dir.Join("c.txt"), dir.Join("a.txt")),
				),
			),
			fs.WithDir(
				"objects",
				fs.WithFile("6ac93242553e35a043104765a33828117479f12ae8333a65a2f0b0ce6dcc0263", ""),
				fs.WithFile("18b7cb099a9ea3f50ba899b5ba81e0d377a5f3b16f8f6eeb8b3e58cd4692b993", ""),
				fs.WithFile("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", ""),
			),
			fs.WithFile("head", "3f674c71a3596db8f24fd31a85c503ae600898cc03810fcc171781d4f35531d2"),
			fs.WithFile("index", fmt.Sprintf(`Tracked files:

%s	(modified)
6ac93242553e35a043104765a33828117479f12ae8333a65a2f0b0ce6dcc0263
%s	(removed)
`, dir.Join("d.txt"), dir.Join("b.txt"))),
		),
	)

	repository := GetRepository(dir.Path())

	assert.Equal(t, repository.root, dir.Path())
	assert.Equal(t, repository.head, "3f674c71a3596db8f24fd31a85c503ae600898cc03810fcc171781d4f35531d2")
	testifyAssert.EqualValues(
		t,
		repository.index,
		[]*Change{
			{
				changeType: Modified,
				modified: &File{
					filepath:   dir.Join("d.txt"),
					objectName: "6ac93242553e35a043104765a33828117479f12ae8333a65a2f0b0ce6dcc0263",
				},
			},
			{
				changeType: Removal,
				removal: &FileRemoval{
					filepath: dir.Join("b.txt"),
				},
			},
		},
	)
	assert.Equal(t, len(repository.dir.children), 3)
	assert.Equal(t, repository.dir.children["a.txt"].file.filepath, dir.Join("a.txt"))
	assert.Equal(t, repository.dir.children["a.txt"].file.objectName, "18b7cb099a9ea3f50ba899b5ba81e0d377a5f3b16f8f6eeb8b3e58cd4692b993")
	assert.Equal(t, repository.dir.children["b.txt"].file.filepath, dir.Join("b.txt"))
	assert.Equal(t, repository.dir.children["b.txt"].file.objectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	assert.Equal(t, repository.dir.children["c.txt"].file.filepath, dir.Join("c.txt"))
	assert.Equal(t, repository.dir.children["c.txt"].file.objectName, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
}
