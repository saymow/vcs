package repositories

import (
	"fmt"
	path "path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const PATH_SEPARATOR = string(path.Separator)

func TestAddNode(t *testing.T) {
	dir := &Dir{
		make(map[string]*Node),
	}

	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "/home/project/a.txt"}})

	assert.Equal(t, len(dir.children), 1)
	assert.Equal(t, dir.children["a.txt"].file, &File{filepath: "/home/project/a.txt"})
}

func TestAddNodeNestedPath(t *testing.T) {
	dir := &Dir{
		make(map[string]*Node),
	}

	dir.addNode("1.txt", &Change{changeType: Modified, modified: &File{filepath: "/home/project/1.txt"}})
	dir.addNode(fmt.Sprintf("a%s2.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/2.txt"}})
	dir.addNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/3.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s4.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/b/4.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/b/5.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s6.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/b/6.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s7.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/c/7.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/c/8.txt"}})

	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "/home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children), 4)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "/home/project/a/2.txt"})
	assert.Equal(t, dir.children["a"].dir.children["3.txt"].file, &File{filepath: "/home/project/a/3.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "/home/project/a/b/4.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["5.txt"].file, &File{filepath: "/home/project/a/b/5.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "/home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "/home/project/a/c/7.txt"})
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["8.txt"].file, &File{filepath: "/home/project/a/c/8.txt"})
}

func TestAddNodeRemovalChanges(t *testing.T) {
	dir := &Dir{
		make(map[string]*Node),
	}

	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "/home/project/a.txt"}})
	dir.addNode("b.txt", &Change{changeType: Modified, modified: &File{filepath: "/home/project/b.txt"}})
	dir.addNode("a.txt", &Change{changeType: Removal, removal: &FileRemoval{filepath: "/home/project/a.txt"}})
	dir.addNode("c.txt", &Change{changeType: Removal, removal: &FileRemoval{filepath: "/home/project/c.txt"}})

	assert.Equal(t, len(dir.children), 1)
	assert.Equal(t, dir.children["b.txt"].file, &File{filepath: "/home/project/b.txt"})
}

func TestAddNodeOverrideRemovalChanges(t *testing.T) {
	dir := &Dir{
		make(map[string]*Node),
	}

	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "/home/project/a.txt", objectName: "old-version"}})
	dir.addNode("a.txt", &Change{changeType: Removal, removal: &FileRemoval{filepath: "/home/project/a.txt"}})
	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "/home/project/a.txt", objectName: "newer-version"}})

	assert.Equal(t, len(dir.children), 1)
	assert.Equal(t, dir.children["a.txt"].file, &File{filepath: "/home/project/a.txt", objectName: "newer-version"})
}

func TestAddNodeRemovalChangesNestedPath(t *testing.T) {
	dir := &Dir{
		make(map[string]*Node),
	}

	dir.addNode("1.txt", &Change{changeType: Modified, modified: &File{filepath: "/home/project/1.txt"}})
	dir.addNode(fmt.Sprintf("a%s2.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/2.txt"}})
	dir.addNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/3.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s4.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/b/4.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/b/5.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s6.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/b/6.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s7.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/c/7.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/c/8.txt"}})

	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "/home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children), 4)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "/home/project/a/2.txt"})
	assert.Equal(t, dir.children["a"].dir.children["3.txt"].file, &File{filepath: "/home/project/a/3.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "/home/project/a/b/4.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["5.txt"].file, &File{filepath: "/home/project/a/b/5.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "/home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "/home/project/a/c/7.txt"})
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["8.txt"].file, &File{filepath: "/home/project/a/c/8.txt"})

	dir.addNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{changeType: Removal, removal: &FileRemoval{filepath: "/home/project/a/3.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Removal, removal: &FileRemoval{filepath: "/home/project/a/b/5.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Removal, removal: &FileRemoval{filepath: "/home/project/a/c/8.txt"}})

	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "/home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "/home/project/a/2.txt"})
	_, ok := dir.children["a"].dir.children["3.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "/home/project/a/b/4.txt"})
	_, ok = dir.children["a"].dir.children["b"].dir.children["5.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "/home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 1)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "/home/project/a/c/7.txt"})
	_, ok = dir.children["a"].dir.children["c"].dir.children["8.txt"]
	assert.False(t, ok)

	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "/home/project/a/c/8.txt", objectName: "newer-version"}})

	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "/home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "/home/project/a/2.txt"})
	_, ok = dir.children["a"].dir.children["3.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "/home/project/a/b/4.txt"})
	_, ok = dir.children["a"].dir.children["b"].dir.children["5.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "/home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "/home/project/a/c/7.txt"})
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["8.txt"].file, &File{filepath: "/home/project/a/c/8.txt", objectName: "newer-version"})
}
