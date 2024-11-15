package repositories

import (
	"testing"

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
