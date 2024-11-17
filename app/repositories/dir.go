package repositories

import (
	fp "path/filepath"
	"strings"
)

func (root *Dir) addNodeHelper(segments []string, change *Change) {
	if len(segments) == 1 {
		if change.changeType == Removal {
			delete(root.children, segments[0])
		} else {
			root.children[segments[0]] = &Node{
				nodeType: FileType,
				file:     change.modified,
			}
		}

		return
	}

	var node *Node
	dirNodeName := segments[0]

	if _, ok := root.children[dirNodeName]; ok {
		node = root.children[dirNodeName]
	} else {
		node = &Node{
			nodeType: DirType,
			dir: &Dir{
				path:     fp.Join(root.path, dirNodeName),
				children: make(map[string]*Node),
			},
		}
		root.children[dirNodeName] = node
	}

	node.dir.addNodeHelper(segments[1:], change)

	if len(node.dir.children) == 0 {
		// If we remove all entries from a directory, then we dont need it anymore.
		// This is ensure we dont restore an empty directory.
		delete(root.children, dirNodeName)
	}
}

func (root *Dir) addNode(path string, change *Change) {
	segments := strings.Split(path, string(fp.Separator))

	root.addNodeHelper(segments, change)
}

func (root *Dir) findNodeHelper(segments []string) *Node {
	if len(segments) == 1 {
		node, ok := root.children[segments[0]]
		if !ok {
			return nil
		}

		return node
	}

	subdirName := segments[0]
	node, ok := root.children[subdirName]
	if !ok {
		return nil
	}

	return node.dir.findNodeHelper(segments[1:])
}

func (root *Dir) findNode(path string) *Node {
	if path == "" || path == string(fp.Separator) {
		// edge case to handle all paths for this dir
		return &Node{nodeType: DirType, dir: root}
	}

	segments := strings.Split(path, string(fp.Separator))

	return root.findNodeHelper(segments)
}

func (root *Dir) collectFilesHelper(files *[]*File) {
	for _, node := range root.children {
		if node.nodeType == DirType {
			node.dir.collectFilesHelper(files)
		} else {
			*files = append(*files, node.file)
		}
	}
}

func (root *Dir) collectAllFiles() []*File {
	files := []*File{}

	root.collectFilesHelper(&files)

	return files
}

func (root *Dir) preOrderTraversalHelper(nodes *[]*Node) {
	for _, node := range root.children {
		*nodes = append(*nodes, node)

		if node.nodeType == DirType {
			node.dir.preOrderTraversalHelper(nodes)
		}
	}
}

func (root *Dir) preOrderTraversal() []*Node {
	nodes := []*Node{{nodeType: DirType, dir: root}}

	root.preOrderTraversalHelper(&nodes)

	return nodes
}
