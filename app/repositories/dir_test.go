package repositories

import (
	"fmt"
	path "path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

const PATH_SEPARATOR = string(path.Separator)

func TestAddNode(t *testing.T) {
	dir := &Dir{
		path:     path.Join("home", "project"),
		children: make(map[string]*Node),
	}

	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "home/project/a.txt"}})

	assert.Equal(t, dir.path, path.Join("home", "project"))
	assert.Equal(t, len(dir.children), 1)
	assert.Equal(t, dir.children["a.txt"].file, &File{filepath: "home/project/a.txt"})
}

func TestAddNodeNestedPath(t *testing.T) {
	dir := &Dir{
		path:     path.Join("home", "project"),
		children: make(map[string]*Node),
	}

	dir.addNode("1.txt", &Change{changeType: Modified, modified: &File{filepath: "home/project/1.txt"}})
	dir.addNode(fmt.Sprintf("a%s2.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/2.txt"}})
	dir.addNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/3.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s4.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/b/4.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/b/5.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s6.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/b/6.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s7.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/c/7.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/c/8.txt"}})

	assert.Equal(t, dir.path, path.Join("home", "project"))
	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, dir.children["a"].dir.path, path.Join("home", "project", "a"))
	assert.Equal(t, len(dir.children["a"].dir.children), 4)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "home/project/a/2.txt"})
	assert.Equal(t, dir.children["a"].dir.children["3.txt"].file, &File{filepath: "home/project/a/3.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.path, path.Join("home", "project", "a", "b"))
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "home/project/a/b/4.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["5.txt"].file, &File{filepath: "home/project/a/b/5.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.path, path.Join("home", "project", "a", "c"))
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "home/project/a/c/7.txt"})
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["8.txt"].file, &File{filepath: "home/project/a/c/8.txt"})
}

func TestAddNodeRemovalChanges(t *testing.T) {
	dir := &Dir{
		path:     path.Join("home", "project"),
		children: make(map[string]*Node),
	}

	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "home/project/a.txt"}})
	dir.addNode("b.txt", &Change{changeType: Modified, modified: &File{filepath: "home/project/b.txt"}})
	dir.addNode("a.txt", &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/a.txt"}})
	dir.addNode("c.txt", &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/c.txt"}})

	assert.Equal(t, dir.path, path.Join("home", "project"))
	assert.Equal(t, len(dir.children), 1)
	assert.Equal(t, dir.children["b.txt"].file, &File{filepath: "home/project/b.txt"})
}

func TestAddNodeOverrideRemovalChanges(t *testing.T) {
	dir := &Dir{
		path:     path.Join("home", "project"),
		children: make(map[string]*Node),
	}

	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "home/project/a.txt", objectName: "old-version"}})
	dir.addNode("a.txt", &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/a.txt"}})
	dir.addNode("a.txt", &Change{changeType: Modified, modified: &File{filepath: "home/project/a.txt", objectName: "newer-version"}})

	assert.Equal(t, dir.path, path.Join("home", "project"))
	assert.Equal(t, len(dir.children), 1)
	assert.Equal(t, dir.children["a.txt"].file, &File{filepath: "home/project/a.txt", objectName: "newer-version"})
}

func TestAddNodeRemovalChangesRemovesEmptyDir(t *testing.T) {
	dir := &Dir{
		path: path.Join("home", "project"),
		children: map[string]*Node{
			"dir": {
				nodeType: DirType,
				dir: &Dir{
					path: path.Join("home", "project", "dir"),
					children: map[string]*Node{
						"a.txt": {
							nodeType: FileType,
							file: &File{
								"home/project/dir/a.txt",
								"object-a",
							},
						},
						"b.txt": {
							nodeType: FileType,
							file: &File{
								"home/project/dir/b.txt",
								"object-b",
							},
						},
					},
				},
			},
		},
	}

	dir.addNode(path.Join("dir", "a.txt"), &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/dir/a.txt"}})
	assert.Equal(t, len(dir.children), 1)
	assert.Equal(t, dir.children["dir"].nodeType, DirType)
	dir.addNode(path.Join("dir", "b.txt"), &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/dir/b.txt"}})
	assert.Equal(t, len(dir.children), 0)
}

func TestAddNodeRemovalChangesNestedPath(t *testing.T) {
	dir := &Dir{
		path:     path.Join("home", "project"),
		children: make(map[string]*Node),
	}

	dir.addNode("1.txt", &Change{changeType: Modified, modified: &File{filepath: "home/project/1.txt"}})
	dir.addNode(fmt.Sprintf("a%s2.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/2.txt"}})
	dir.addNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/3.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s4.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/b/4.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/b/5.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s6.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/b/6.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s7.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/c/7.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/c/8.txt"}})

	assert.Equal(t, dir.path, path.Join("home", "project"))
	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children), 4)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "home/project/a/2.txt"})
	assert.Equal(t, dir.children["a"].dir.children["3.txt"].file, &File{filepath: "home/project/a/3.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.path, path.Join("home", "project", "a", "b"))
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "home/project/a/b/4.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["5.txt"].file, &File{filepath: "home/project/a/b/5.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.path, path.Join("home", "project", "a", "c"))
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "home/project/a/c/7.txt"})
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["8.txt"].file, &File{filepath: "home/project/a/c/8.txt"})

	dir.addNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/a/3.txt"}})
	dir.addNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/a/b/5.txt"}})
	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Removal, removal: &FileRemoval{filepath: "home/project/a/c/8.txt"}})

	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "home/project/a/2.txt"})
	_, ok := dir.children["a"].dir.children["3.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "home/project/a/b/4.txt"})
	_, ok = dir.children["a"].dir.children["b"].dir.children["5.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 1)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "home/project/a/c/7.txt"})
	_, ok = dir.children["a"].dir.children["c"].dir.children["8.txt"]
	assert.False(t, ok)

	dir.addNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{changeType: Modified, modified: &File{filepath: "home/project/a/c/8.txt", objectName: "newer-version"}})

	assert.Equal(t, len(dir.children), 2)
	assert.Equal(t, dir.children["1.txt"].file, &File{filepath: "home/project/1.txt"})
	assert.Equal(t, dir.children["a"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children), 3)
	assert.Equal(t, dir.children["a"].dir.children["2.txt"].file, &File{filepath: "home/project/a/2.txt"})
	_, ok = dir.children["a"].dir.children["3.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].nodeType, DirType)
	assert.Equal(t, len(dir.children["a"].dir.children["b"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["4.txt"].file, &File{filepath: "home/project/a/b/4.txt"})
	_, ok = dir.children["a"].dir.children["b"].dir.children["5.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.children["a"].dir.children["b"].dir.children["6.txt"].file, &File{filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.children["a"].dir.children["c"].dir.children), 2)
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["7.txt"].file, &File{filepath: "home/project/a/c/7.txt"})
	assert.Equal(t, dir.children["a"].dir.children["c"].dir.children["8.txt"].file, &File{filepath: "home/project/a/c/8.txt", objectName: "newer-version"})
}

func TestFindNode(t *testing.T) {
	dir := &Dir{
		path: path.Join("home", "project"),
		children: map[string]*Node{
			"a.txt": {
				nodeType: FileType,
				file: &File{
					"home/project/a.txt",
					"object-a",
				},
			},
			"b.txt": {
				nodeType: FileType,
				file: &File{
					"home/project/b.txt",
					"object-b",
				},
			},
		},
	}

	assert.Equal(t, dir.findNode("").nodeType, DirType)
	assert.Equal(t, dir.findNode("a.txt").nodeType, FileType)
	assert.Equal(t, dir.findNode("a.txt").file, &File{"home/project/a.txt", "object-a"})
	assert.Equal(t, dir.findNode("b.txt").nodeType, FileType)
	assert.Equal(t, dir.findNode("b.txt").file, &File{"home/project/b.txt", "object-b"})
}

func TestFindNodeNestedPath(t *testing.T) {
	dir := &Dir{
		children: map[string]*Node{
			"a.txt": {
				nodeType: FileType,
				file: &File{
					"home/project/a.txt",
					"object-a",
				},
			},
			"b.txt": {
				nodeType: FileType,
				file: &File{
					"home/project/b.txt",
					"object-b",
				},
			},
			"subdir": {
				nodeType: DirType,
				dir: &Dir{
					children: map[string]*Node{
						"a.txt": {
							nodeType: FileType,
							file: &File{
								"home/project/subdir/a.txt",
								"object-subdir-a",
							},
						},
						"c.txt": {
							nodeType: FileType,
							file: &File{
								"home/project/subdir/c.txt",
								"object-subdir-c",
							},
						},
						"nested-subdir": {
							nodeType: DirType,
							dir: &Dir{
								children: map[string]*Node{
									"b.txt": {
										nodeType: FileType,
										file: &File{
											"home/project/subdir/nested-subdir/b.txt",
											"object-subdir-nested-subdir-b",
										},
									},
									"d.txt": {
										nodeType: FileType,
										file: &File{
											"home/project/subdir/nested-subdir/d.txt",
											"object-subdir-nested-subdir-d",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// File nodes

	assert.Equal(t, dir.findNode("").nodeType, DirType)
	assert.Equal(t, dir.findNode("a.txt").nodeType, FileType)
	assert.Equal(t, dir.findNode("a.txt").file, &File{"home/project/a.txt", "object-a"})
	assert.Equal(t, dir.findNode("b.txt").nodeType, FileType)
	assert.Equal(t, dir.findNode("b.txt").file, &File{"home/project/b.txt", "object-b"})
	assert.Equal(t, dir.findNode("subdir").nodeType, DirType)
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%sa.txt", PATH_SEPARATOR)).nodeType, FileType)
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%sa.txt", PATH_SEPARATOR)).file, &File{"home/project/subdir/a.txt", "object-subdir-a"})
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%sc.txt", PATH_SEPARATOR)).nodeType, FileType)
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%sc.txt", PATH_SEPARATOR)).file, &File{"home/project/subdir/c.txt", "object-subdir-c"})
	assert.Equal(t, dir.findNode(path.Join("subdir", "nested-subdir")).nodeType, DirType)
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%snested-subdir%sb.txt", PATH_SEPARATOR, PATH_SEPARATOR)).nodeType, FileType)
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%snested-subdir%sb.txt", PATH_SEPARATOR, PATH_SEPARATOR)).file, &File{"home/project/subdir/nested-subdir/b.txt", "object-subdir-nested-subdir-b"})
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%snested-subdir%sd.txt", PATH_SEPARATOR, PATH_SEPARATOR)).nodeType, FileType)
	assert.Equal(t, dir.findNode(fmt.Sprintf("subdir%snested-subdir%sd.txt", PATH_SEPARATOR, PATH_SEPARATOR)).file, &File{"home/project/subdir/nested-subdir/d.txt", "object-subdir-nested-subdir-d"})

}

func TestAllCollectFiles(t *testing.T) {
	dir := &Dir{
		children: map[string]*Node{
			"a.txt": {
				nodeType: FileType,
				file: &File{
					"home/project/a.txt",
					"1",
				},
			},
			"b.txt": {
				nodeType: FileType,
				file: &File{
					"home/project/b.txt",
					"2",
				},
			},
			"subdir": {
				nodeType: DirType,
				dir: &Dir{
					children: map[string]*Node{
						"a.txt": {
							nodeType: FileType,
							file: &File{
								"home/project/subdir/a.txt",
								"3",
							},
						},
						"c.txt": {
							nodeType: FileType,
							file: &File{
								"home/project/subdir/c.txt",
								"4",
							},
						},
						"nested-subdir": {
							nodeType: DirType,
							dir: &Dir{
								children: map[string]*Node{
									"b.txt": {
										nodeType: FileType,
										file: &File{
											"home/project/subdir/nested-subdir/b.txt",
											"5",
										},
									},
									"d.txt": {
										nodeType: FileType,
										file: &File{
											"home/project/subdir/nested-subdir/d.txt",
											"6",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	received := dir.collectAllFiles()

	sort.Slice(received, func(i, j int) bool { return received[i].objectName < received[j].objectName })
	assert.Equal(t,
		received,
		[]*File{
			{
				"home/project/a.txt",
				"1",
			},
			{
				"home/project/b.txt",
				"2",
			},
			{
				"home/project/subdir/a.txt",
				"3",
			},
			{
				"home/project/subdir/c.txt",
				"4",
			},
			{
				"home/project/subdir/nested-subdir/b.txt",
				"5",
			},
			{
				"home/project/subdir/nested-subdir/d.txt",
				"6",
			},
		},
	)
}

// Since it's we cannot rely on the sequence the of the map iteration, the test becomes
// hard. This is the reason there are only file nodes on the last dir node.
// This teste ensure that the nodes are indeed collected in pre order.
func TestPreOrderTraversal(t *testing.T) {
	dir := &Dir{
		path: "",
		children: map[string]*Node{
			"subdir": {
				nodeType: DirType,
				dir: &Dir{
					path: "subdir",
					children: map[string]*Node{
						"nested-subdir": {
							nodeType: DirType,
							dir: &Dir{
								path: path.Join("subdir", "nested-subdir"),
								children: map[string]*Node{
									"b.txt": {
										nodeType: FileType,
										file: &File{
											"home/project/subdir/nested-subdir/b.txt",
											"5",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	received := dir.preOrderTraversal()

	assert.EqualValues(t, len(received), 4)
	assert.EqualValues(t, received[0].nodeType, DirType)
	assert.EqualValues(t, received[0].dir.path, "")
	assert.EqualValues(t, received[1].nodeType, DirType)
	assert.EqualValues(t, received[1].dir.path, "subdir")
	assert.EqualValues(t, received[2].nodeType, DirType)
	assert.EqualValues(t, received[2].dir.path, path.Join("subdir", "nested-subdir"))
	assert.EqualValues(t,
		received[3],
		&Node{
			nodeType: FileType,
			file: &File{
				"home/project/subdir/nested-subdir/b.txt",
				"5",
			},
		},
	)
}
