package directories

import (
	Path "path/filepath"
	"strings"
)

type NodeType int

const (
	FileType NodeType = iota
	DirType
)

type FileRemoval struct {
	Filepath string
}

type FileConflict struct {
	Filepath   string
	ObjectName string
	Message    string
}

type ChangeType int

const (
	Creation ChangeType = iota
	Modification
	Removal
	Conflict
)

type Change struct {
	ChangeType ChangeType
	File       *File
	Removal    *FileRemoval
	Conflict   *FileConflict
}

type File struct {
	Filepath   string
	ObjectName string
}

type Node struct {
	NodeType NodeType
	File     *File
	Dir      *Dir
}

type Dir struct {
	Path     string
	Children map[string]*Node
}

const (
	MODIFIED_CHANGE = "(modified)"
	CREATED_CHANGE  = "(created)"
	REMOVAL_CHANGE  = "(removed)"
	CONFLICT_CHANGE = "(conflicted)"
)

type DirError struct {
	message string
}

func (err *DirError) Error() string {
	return err.message
}

func (change *Change) GetPath() string {
	if change.ChangeType == Removal {
		return change.Removal.Filepath
	}
	if change.ChangeType == Conflict {
		return change.Conflict.Filepath
	}

	return change.File.Filepath
}

func (change *Change) GetHash() string {
	if change.ChangeType == Removal {
		return ""
	}
	if change.ChangeType == Conflict {
		return change.Conflict.ObjectName
	}

	return change.File.ObjectName
}

func (change *Change) Conflicts(otherChange *Change) bool {
	if change.GetPath() != otherChange.GetPath() {
		return false
	}
	if change.ChangeType == Removal && otherChange.ChangeType == Removal {
		return false
	}

	return change.GetHash() != otherChange.GetHash()
}

func (root *Dir) addNodeHelper(segments []string, change *Change) {
	if len(segments) == 1 {
		if change.ChangeType == Removal {
			delete(root.Children, segments[0])
		} else {
			root.Children[segments[0]] = &Node{
				NodeType: FileType,
				File:     change.File,
			}
		}

		return
	}

	var node *Node
	dirNodeName := segments[0]

	if _, ok := root.Children[dirNodeName]; ok {
		node = root.Children[dirNodeName]
	} else {
		node = &Node{
			NodeType: DirType,
			Dir: &Dir{
				Path:     Path.Join(root.Path, dirNodeName),
				Children: make(map[string]*Node),
			},
		}
		root.Children[dirNodeName] = node
	}

	node.Dir.addNodeHelper(segments[1:], change)

	if len(node.Dir.Children) == 0 {
		// If we remove all entries from a directory, then we dont need it anymore.
		// This is ensure we dont restore an empty directory.
		delete(root.Children, dirNodeName)
	}
}

func (root *Dir) AddNode(path string, change *Change) {
	segments := strings.Split(path, string(Path.Separator))

	root.addNodeHelper(segments, change)
}

func (root *Dir) findNodeHelper(segments []string) *Node {
	if len(segments) == 1 {
		node, ok := root.Children[segments[0]]
		if !ok {
			return nil
		}

		return node
	}

	subdirName := segments[0]
	node, ok := root.Children[subdirName]
	if !ok {
		return nil
	}

	return node.Dir.findNodeHelper(segments[1:])
}

func (root *Dir) FindNode(path string) *Node {
	if path == "" || path == string(Path.Separator) {
		// edge case to handle all paths for this dir
		return &Node{NodeType: DirType, Dir: root}
	}

	segments := strings.Split(path, string(Path.Separator))

	return root.findNodeHelper(segments)
}

func (root *Dir) collectFilesHelper(files *[]*File) {
	for _, node := range root.Children {
		if node.NodeType == DirType {
			node.Dir.collectFilesHelper(files)
		} else {
			*files = append(*files, node.File)
		}
	}
}

func (root *Dir) CollectAllFiles() []*File {
	files := []*File{}

	root.collectFilesHelper(&files)

	return files
}

func (root *Dir) preOrderTraversalHelper(nodes *[]*Node) {
	for _, node := range root.Children {
		*nodes = append(*nodes, node)

		if node.NodeType == DirType {
			node.Dir.preOrderTraversalHelper(nodes)
		}
	}
}

func (root *Dir) PreOrderTraversal() []*Node {
	nodes := []*Node{{NodeType: DirType, Dir: root}}

	root.preOrderTraversalHelper(&nodes)

	return nodes
}

func (root *Dir) isSubpath(path string) bool {
	rootParts := strings.Split(root.Path, string((Path.Separator)))
	parts := strings.Split(path, string((Path.Separator)))

	if len(rootParts) > len(parts) {
		return false
	}

	idx := 0
	for idx < len(rootParts) {
		if rootParts[idx] != parts[idx] {
			return false
		}
		idx++
	}

	return true
}

func (root *Dir) NormalizePath(path string) (string, error) {
	path, err := root.AbsPath(path)
	if err != nil {
		return "", err
	}
	if path == root.Path {
		// it seems a path is subpath of it self
		return "", nil
	}

	return path[len(root.Path)+1:], nil
}

func (root *Dir) AbsPath(path string) (string, error) {
	if !Path.IsAbs(path) {
		path = Path.Join(root.Path, path)
	}
	if !root.isSubpath(path) {
		return "", &DirError{"invalid path."}
	}

	return path, nil
}
