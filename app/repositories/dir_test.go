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

func TestFindFile(t *testing.T) {
	dir := &Dir{
		children: map[string]*Node{
			"a.txt": &Node{
				nodeType: FileType,
				file: &File{
					"/home/project/a.txt",
					"object-a",
				},
			},
			"b.txt": &Node{
				nodeType: FileType,
				file: &File{
					"/home/project/b.txt",
					"object-b",
				},
			},
		},
	}

	assert.Equal(t, dir.findFile("a.txt"), &File{"/home/project/a.txt", "object-a"})
	assert.Equal(t, dir.findFile("b.txt"), &File{"/home/project/b.txt", "object-b"})
	assert.Nil(t, dir.findFile("c.txt"))
}

func TestFindFileNestedPath(t *testing.T) {
	dir := &Dir{
		children: map[string]*Node{
			"a.txt": &Node{
				nodeType: FileType,
				file: &File{
					"/home/project/a.txt",
					"object-a",
				},
			},
			"b.txt": &Node{
				nodeType: FileType,
				file: &File{
					"/home/project/b.txt",
					"object-b",
				},
			},
			"subdir": &Node{
				nodeType: DirType,
				dir: &Dir{
					children: map[string]*Node{
						"a.txt": &Node{
							nodeType: FileType,
							file: &File{
								"/home/project/subdir/a.txt",
								"object-subdir-a",
							},
						},
						"c.txt": &Node{
							nodeType: FileType,
							file: &File{
								"/home/project/subdir/c.txt",
								"object-subdir-c",
							},
						},
						"nested-subdir": &Node{
							nodeType: DirType,
							dir: &Dir{
								children: map[string]*Node{
									"b.txt": &Node{
										nodeType: FileType,
										file: &File{
											"/home/project/subdir/nested-subdir/b.txt",
											"object-subdir-nested-subdir-b",
										},
									},
									"d.txt": &Node{
										nodeType: FileType,
										file: &File{
											"/home/project/subdir/nested-subdir/d.txt",
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

	assert.Equal(t, dir.findFile("a.txt"), &File{"/home/project/a.txt", "object-a"})
	assert.Equal(t, dir.findFile("b.txt"), &File{"/home/project/b.txt", "object-b"})
	assert.Nil(t, dir.findFile("subdir"))
	assert.Equal(t, dir.findFile(fmt.Sprintf("subdir%sa.txt", PATH_SEPARATOR)), &File{"/home/project/subdir/a.txt", "object-subdir-a"})
	assert.Equal(t, dir.findFile(fmt.Sprintf("subdir%sc.txt", PATH_SEPARATOR)), &File{"/home/project/subdir/c.txt", "object-subdir-c"})
	assert.Nil(t, dir.findFile(fmt.Sprintf("subdir%snested-subdir", PATH_SEPARATOR)))
	assert.Equal(t, dir.findFile(fmt.Sprintf("subdir%snested-subdir%sb.txt", PATH_SEPARATOR, PATH_SEPARATOR)), &File{"/home/project/subdir/nested-subdir/b.txt", "object-subdir-nested-subdir-b"})
	assert.Equal(t, dir.findFile(fmt.Sprintf("subdir%snested-subdir%sd.txt", PATH_SEPARATOR, PATH_SEPARATOR)), &File{"/home/project/subdir/nested-subdir/d.txt", "object-subdir-nested-subdir-d"})
}

func TestCollectFiles(t *testing.T) {
	dir := &Dir{
		children: map[string]*Node{
			"a.txt": &Node{
				nodeType: FileType,
				file: &File{
					"/home/project/a.txt",
					"1",
				},
			},
			"b.txt": &Node{
				nodeType: FileType,
				file: &File{
					"/home/project/b.txt",
					"2",
				},
			},
			"subdir": &Node{
				nodeType: DirType,
				dir: &Dir{
					children: map[string]*Node{
						"a.txt": &Node{
							nodeType: FileType,
							file: &File{
								"/home/project/subdir/a.txt",
								"3",
							},
						},
						"c.txt": &Node{
							nodeType: FileType,
							file: &File{
								"/home/project/subdir/c.txt",
								"4",
							},
						},
						"nested-subdir": &Node{
							nodeType: DirType,
							dir: &Dir{
								children: map[string]*Node{
									"b.txt": &Node{
										nodeType: FileType,
										file: &File{
											"/home/project/subdir/nested-subdir/b.txt",
											"5",
										},
									},
									"d.txt": &Node{
										nodeType: FileType,
										file: &File{
											"/home/project/subdir/nested-subdir/d.txt",
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

	received := dir.collectFiles()

	sort.Slice(received, func(i, j int) bool { return received[i].objectName < received[j].objectName })
	assert.Equal(t,
		received,
		[]*File{
			&File{
				"/home/project/a.txt",
				"1",
			},
			&File{
				"/home/project/b.txt",
				"2",
			},
			&File{
				"/home/project/subdir/a.txt",
				"3",
			},
			&File{
				"/home/project/subdir/c.txt",
				"4",
			},
			&File{
				"/home/project/subdir/nested-subdir/b.txt",
				"5",
			},
			&File{
				"/home/project/subdir/nested-subdir/d.txt",
				"6",
			},
		},
	)
}
