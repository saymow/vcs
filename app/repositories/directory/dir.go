package directory

import (
	fp "path/filepath"
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

type ChangeType int

const (
	Creation ChangeType = iota
	Modification
	Removal
)

type Change struct {
	ChangeType ChangeType
	File       *File
	Removal    *FileRemoval
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
)

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
				Path:     fp.Join(root.Path, dirNodeName),
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
	segments := strings.Split(path, string(fp.Separator))

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
	if path == "" || path == string(fp.Separator) {
		// edge case to handle all paths for this dir
		return &Node{NodeType: DirType, Dir: root}
	}

	segments := strings.Split(path, string(fp.Separator))

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
