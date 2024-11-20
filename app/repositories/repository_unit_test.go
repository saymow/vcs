package repositories

import (
	Path "path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
