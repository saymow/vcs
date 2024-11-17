package directory

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
		Path:     path.Join("home", "project"),
		Children: make(map[string]*Node),
	}

	dir.AddNode("a.txt", &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a.txt"}})

	assert.Equal(t, dir.Path, path.Join("home", "project"))
	assert.Equal(t, len(dir.Children), 1)
	assert.Equal(t, dir.Children["a.txt"].File, &File{Filepath: "home/project/a.txt"})
}

func TestAddNodeNestedPath(t *testing.T) {
	dir := &Dir{
		Path:     path.Join("home", "project"),
		Children: make(map[string]*Node),
	}

	dir.AddNode("1.txt", &Change{ChangeType: Modification, File: &File{Filepath: "home/project/1.txt"}})
	dir.AddNode(fmt.Sprintf("a%s2.txt", PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/2.txt"}})
	dir.AddNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/3.txt"}})
	dir.AddNode(fmt.Sprintf("a%sb%s4.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/b/4.txt"}})
	dir.AddNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/b/5.txt"}})
	dir.AddNode(fmt.Sprintf("a%sb%s6.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/b/6.txt"}})
	dir.AddNode(fmt.Sprintf("a%sc%s7.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/c/7.txt"}})
	dir.AddNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/c/8.txt"}})

	assert.Equal(t, dir.Path, path.Join("home", "project"))
	assert.Equal(t, len(dir.Children), 2)
	assert.Equal(t, dir.Children["1.txt"].File, &File{Filepath: "home/project/1.txt"})
	assert.Equal(t, dir.Children["a"].NodeType, DirType)
	assert.Equal(t, dir.Children["a"].Dir.Path, path.Join("home", "project", "a"))
	assert.Equal(t, len(dir.Children["a"].Dir.Children), 4)
	assert.Equal(t, dir.Children["a"].Dir.Children["2.txt"].File, &File{Filepath: "home/project/a/2.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["3.txt"].File, &File{Filepath: "home/project/a/3.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].NodeType, DirType)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Path, path.Join("home", "project", "a", "b"))
	assert.Equal(t, len(dir.Children["a"].Dir.Children["b"].Dir.Children), 3)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["4.txt"].File, &File{Filepath: "home/project/a/b/4.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["5.txt"].File, &File{Filepath: "home/project/a/b/5.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["6.txt"].File, &File{Filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Path, path.Join("home", "project", "a", "c"))
	assert.Equal(t, len(dir.Children["a"].Dir.Children["c"].Dir.Children), 2)
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Children["7.txt"].File, &File{Filepath: "home/project/a/c/7.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Children["8.txt"].File, &File{Filepath: "home/project/a/c/8.txt"})
}

func TestAddNodeRemovalChanges(t *testing.T) {
	dir := &Dir{
		Path:     path.Join("home", "project"),
		Children: make(map[string]*Node),
	}

	dir.AddNode("a.txt", &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a.txt"}})
	dir.AddNode("b.txt", &Change{ChangeType: Modification, File: &File{Filepath: "home/project/b.txt"}})
	dir.AddNode("a.txt", &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/a.txt"}})
	dir.AddNode("c.txt", &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/c.txt"}})

	assert.Equal(t, dir.Path, path.Join("home", "project"))
	assert.Equal(t, len(dir.Children), 1)
	assert.Equal(t, dir.Children["b.txt"].File, &File{Filepath: "home/project/b.txt"})
}

func TestAddNodeOverrideRemovalChanges(t *testing.T) {
	dir := &Dir{
		Path:     path.Join("home", "project"),
		Children: make(map[string]*Node),
	}

	dir.AddNode("a.txt", &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a.txt", ObjectName: "old-version"}})
	dir.AddNode("a.txt", &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/a.txt"}})
	dir.AddNode("a.txt", &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a.txt", ObjectName: "newer-version"}})

	assert.Equal(t, dir.Path, path.Join("home", "project"))
	assert.Equal(t, len(dir.Children), 1)
	assert.Equal(t, dir.Children["a.txt"].File, &File{Filepath: "home/project/a.txt", ObjectName: "newer-version"})
}

func TestAddNodeRemovalChangesRemovesEmptyDir(t *testing.T) {
	dir := &Dir{
		Path: path.Join("home", "project"),
		Children: map[string]*Node{
			"dir": {
				NodeType: DirType,
				Dir: &Dir{
					Path: path.Join("home", "project", "dir"),
					Children: map[string]*Node{
						"a.txt": {
							NodeType: FileType,
							File: &File{
								"home/project/dir/a.txt",
								"object-a",
							},
						},
						"b.txt": {
							NodeType: FileType,
							File: &File{
								"home/project/dir/b.txt",
								"object-b",
							},
						},
					},
				},
			},
		},
	}

	dir.AddNode(path.Join("dir", "a.txt"), &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/dir/a.txt"}})
	assert.Equal(t, len(dir.Children), 1)
	assert.Equal(t, dir.Children["dir"].NodeType, DirType)
	dir.AddNode(path.Join("dir", "b.txt"), &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/dir/b.txt"}})
	assert.Equal(t, len(dir.Children), 0)
}

func TestAddNodeRemovalChangesNestedPath(t *testing.T) {
	dir := &Dir{
		Path:     path.Join("home", "project"),
		Children: make(map[string]*Node),
	}

	dir.AddNode("1.txt", &Change{ChangeType: Modification, File: &File{Filepath: "home/project/1.txt"}})
	dir.AddNode(fmt.Sprintf("a%s2.txt", PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/2.txt"}})
	dir.AddNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/3.txt"}})
	dir.AddNode(fmt.Sprintf("a%sb%s4.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/b/4.txt"}})
	dir.AddNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/b/5.txt"}})
	dir.AddNode(fmt.Sprintf("a%sb%s6.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/b/6.txt"}})
	dir.AddNode(fmt.Sprintf("a%sc%s7.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/c/7.txt"}})
	dir.AddNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/c/8.txt"}})

	assert.Equal(t, dir.Path, path.Join("home", "project"))
	assert.Equal(t, len(dir.Children), 2)
	assert.Equal(t, dir.Children["1.txt"].File, &File{Filepath: "home/project/1.txt"})
	assert.Equal(t, dir.Children["a"].NodeType, DirType)
	assert.Equal(t, len(dir.Children["a"].Dir.Children), 4)
	assert.Equal(t, dir.Children["a"].Dir.Children["2.txt"].File, &File{Filepath: "home/project/a/2.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["3.txt"].File, &File{Filepath: "home/project/a/3.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].NodeType, DirType)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Path, path.Join("home", "project", "a", "b"))
	assert.Equal(t, len(dir.Children["a"].Dir.Children["b"].Dir.Children), 3)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["4.txt"].File, &File{Filepath: "home/project/a/b/4.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["5.txt"].File, &File{Filepath: "home/project/a/b/5.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["6.txt"].File, &File{Filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].NodeType, DirType)
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Path, path.Join("home", "project", "a", "c"))
	assert.Equal(t, len(dir.Children["a"].Dir.Children["c"].Dir.Children), 2)
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Children["7.txt"].File, &File{Filepath: "home/project/a/c/7.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Children["8.txt"].File, &File{Filepath: "home/project/a/c/8.txt"})

	dir.AddNode(fmt.Sprintf("a%s3.txt", PATH_SEPARATOR), &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/a/3.txt"}})
	dir.AddNode(fmt.Sprintf("a%sb%s5.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/a/b/5.txt"}})
	dir.AddNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Removal, Removal: &FileRemoval{Filepath: "home/project/a/c/8.txt"}})

	assert.Equal(t, len(dir.Children), 2)
	assert.Equal(t, dir.Children["1.txt"].File, &File{Filepath: "home/project/1.txt"})
	assert.Equal(t, dir.Children["a"].NodeType, DirType)
	assert.Equal(t, len(dir.Children["a"].Dir.Children), 3)
	assert.Equal(t, dir.Children["a"].Dir.Children["2.txt"].File, &File{Filepath: "home/project/a/2.txt"})
	_, ok := dir.Children["a"].Dir.Children["3.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].NodeType, DirType)
	assert.Equal(t, len(dir.Children["a"].Dir.Children["b"].Dir.Children), 2)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["4.txt"].File, &File{Filepath: "home/project/a/b/4.txt"})
	_, ok = dir.Children["a"].Dir.Children["b"].Dir.Children["5.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["6.txt"].File, &File{Filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.Children["a"].Dir.Children["c"].Dir.Children), 1)
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Children["7.txt"].File, &File{Filepath: "home/project/a/c/7.txt"})
	_, ok = dir.Children["a"].Dir.Children["c"].Dir.Children["8.txt"]
	assert.False(t, ok)

	dir.AddNode(fmt.Sprintf("a%sc%s8.txt", PATH_SEPARATOR, PATH_SEPARATOR), &Change{ChangeType: Modification, File: &File{Filepath: "home/project/a/c/8.txt", ObjectName: "newer-version"}})

	assert.Equal(t, len(dir.Children), 2)
	assert.Equal(t, dir.Children["1.txt"].File, &File{Filepath: "home/project/1.txt"})
	assert.Equal(t, dir.Children["a"].NodeType, DirType)
	assert.Equal(t, len(dir.Children["a"].Dir.Children), 3)
	assert.Equal(t, dir.Children["a"].Dir.Children["2.txt"].File, &File{Filepath: "home/project/a/2.txt"})
	_, ok = dir.Children["a"].Dir.Children["3.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].NodeType, DirType)
	assert.Equal(t, len(dir.Children["a"].Dir.Children["b"].Dir.Children), 2)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["4.txt"].File, &File{Filepath: "home/project/a/b/4.txt"})
	_, ok = dir.Children["a"].Dir.Children["b"].Dir.Children["5.txt"]
	assert.False(t, ok)
	assert.Equal(t, dir.Children["a"].Dir.Children["b"].Dir.Children["6.txt"].File, &File{Filepath: "home/project/a/b/6.txt"})
	assert.Equal(t, len(dir.Children["a"].Dir.Children["c"].Dir.Children), 2)
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Children["7.txt"].File, &File{Filepath: "home/project/a/c/7.txt"})
	assert.Equal(t, dir.Children["a"].Dir.Children["c"].Dir.Children["8.txt"].File, &File{Filepath: "home/project/a/c/8.txt", ObjectName: "newer-version"})
}

func TestFindNode(t *testing.T) {
	dir := &Dir{
		Path: path.Join("home", "project"),
		Children: map[string]*Node{
			"a.txt": {
				NodeType: FileType,
				File: &File{
					"home/project/a.txt",
					"object-a",
				},
			},
			"b.txt": {
				NodeType: FileType,
				File: &File{
					"home/project/b.txt",
					"object-b",
				},
			},
		},
	}

	assert.Equal(t, dir.FindNode("").NodeType, DirType)
	assert.Equal(t, dir.FindNode("a.txt").NodeType, FileType)
	assert.Equal(t, dir.FindNode("a.txt").File, &File{"home/project/a.txt", "object-a"})
	assert.Equal(t, dir.FindNode("b.txt").NodeType, FileType)
	assert.Equal(t, dir.FindNode("b.txt").File, &File{"home/project/b.txt", "object-b"})
}

func TestFindNodeNestedPath(t *testing.T) {
	dir := &Dir{
		Children: map[string]*Node{
			"a.txt": {
				NodeType: FileType,
				File: &File{
					"home/project/a.txt",
					"object-a",
				},
			},
			"b.txt": {
				NodeType: FileType,
				File: &File{
					"home/project/b.txt",
					"object-b",
				},
			},
			"subdir": {
				NodeType: DirType,
				Dir: &Dir{
					Children: map[string]*Node{
						"a.txt": {
							NodeType: FileType,
							File: &File{
								"home/project/subdir/a.txt",
								"object-subdir-a",
							},
						},
						"c.txt": {
							NodeType: FileType,
							File: &File{
								"home/project/subdir/c.txt",
								"object-subdir-c",
							},
						},
						"nested-subdir": {
							NodeType: DirType,
							Dir: &Dir{
								Children: map[string]*Node{
									"b.txt": {
										NodeType: FileType,
										File: &File{
											"home/project/subdir/nested-subdir/b.txt",
											"object-subdir-nested-subdir-b",
										},
									},
									"d.txt": {
										NodeType: FileType,
										File: &File{
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

	assert.Equal(t, dir.FindNode("").NodeType, DirType)
	assert.Equal(t, dir.FindNode("a.txt").NodeType, FileType)
	assert.Equal(t, dir.FindNode("a.txt").File, &File{"home/project/a.txt", "object-a"})
	assert.Equal(t, dir.FindNode("b.txt").NodeType, FileType)
	assert.Equal(t, dir.FindNode("b.txt").File, &File{"home/project/b.txt", "object-b"})
	assert.Equal(t, dir.FindNode("subdir").NodeType, DirType)
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%sa.txt", PATH_SEPARATOR)).NodeType, FileType)
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%sa.txt", PATH_SEPARATOR)).File, &File{"home/project/subdir/a.txt", "object-subdir-a"})
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%sc.txt", PATH_SEPARATOR)).NodeType, FileType)
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%sc.txt", PATH_SEPARATOR)).File, &File{"home/project/subdir/c.txt", "object-subdir-c"})
	assert.Equal(t, dir.FindNode(path.Join("subdir", "nested-subdir")).NodeType, DirType)
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%snested-subdir%sb.txt", PATH_SEPARATOR, PATH_SEPARATOR)).NodeType, FileType)
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%snested-subdir%sb.txt", PATH_SEPARATOR, PATH_SEPARATOR)).File, &File{"home/project/subdir/nested-subdir/b.txt", "object-subdir-nested-subdir-b"})
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%snested-subdir%sd.txt", PATH_SEPARATOR, PATH_SEPARATOR)).NodeType, FileType)
	assert.Equal(t, dir.FindNode(fmt.Sprintf("subdir%snested-subdir%sd.txt", PATH_SEPARATOR, PATH_SEPARATOR)).File, &File{"home/project/subdir/nested-subdir/d.txt", "object-subdir-nested-subdir-d"})

}

func TestAllCollectFiles(t *testing.T) {
	dir := &Dir{
		Children: map[string]*Node{
			"a.txt": {
				NodeType: FileType,
				File: &File{
					"home/project/a.txt",
					"1",
				},
			},
			"b.txt": {
				NodeType: FileType,
				File: &File{
					"home/project/b.txt",
					"2",
				},
			},
			"subdir": {
				NodeType: DirType,
				Dir: &Dir{
					Children: map[string]*Node{
						"a.txt": {
							NodeType: FileType,
							File: &File{
								"home/project/subdir/a.txt",
								"3",
							},
						},
						"c.txt": {
							NodeType: FileType,
							File: &File{
								"home/project/subdir/c.txt",
								"4",
							},
						},
						"nested-subdir": {
							NodeType: DirType,
							Dir: &Dir{
								Children: map[string]*Node{
									"b.txt": {
										NodeType: FileType,
										File: &File{
											"home/project/subdir/nested-subdir/b.txt",
											"5",
										},
									},
									"d.txt": {
										NodeType: FileType,
										File: &File{
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

	received := dir.CollectAllFiles()

	sort.Slice(received, func(i, j int) bool { return received[i].ObjectName < received[j].ObjectName })
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
		Path: "",
		Children: map[string]*Node{
			"subdir": {
				NodeType: DirType,
				Dir: &Dir{
					Path: "subdir",
					Children: map[string]*Node{
						"nested-subdir": {
							NodeType: DirType,
							Dir: &Dir{
								Path: path.Join("subdir", "nested-subdir"),
								Children: map[string]*Node{
									"b.txt": {
										NodeType: FileType,
										File: &File{
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

	received := dir.PreOrderTraversal()

	assert.EqualValues(t, len(received), 4)
	assert.EqualValues(t, received[0].NodeType, DirType)
	assert.EqualValues(t, received[0].Dir.Path, "")
	assert.EqualValues(t, received[1].NodeType, DirType)
	assert.EqualValues(t, received[1].Dir.Path, "subdir")
	assert.EqualValues(t, received[2].NodeType, DirType)
	assert.EqualValues(t, received[2].Dir.Path, path.Join("subdir", "nested-subdir"))
	assert.EqualValues(t,
		received[3],
		&Node{
			NodeType: FileType,
			File: &File{
				"home/project/subdir/nested-subdir/b.txt",
				"5",
			},
		},
	)
}
