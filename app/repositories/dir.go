package repositories

import (
	fp "path/filepath"
	"strings"
)

func (root *Dir) addNodeHelper(segments []string, file *File) {
	if len(segments) == 1 {
		root.children[segments[0]] = Node{
			nodeType: FileType,
			file:     *file,
		}

		return
	}

	var node Node
	subdirName := segments[0]

	if _, ok := root.children[subdirName]; ok {
		node = root.children[subdirName]
	} else {
		node = Node{
			nodeType: DirType,
			dir:      Dir{make(map[string]Node)},
		}
		root.children[subdirName] = node
	}

	node.dir.addNodeHelper(segments[1:], file)
}

func (root *Dir) addNode(path string, file *File) {
	segments := strings.Split(path, string(fp.Separator))

	root.addNodeHelper(segments, file)
}

func (root *Dir) findNodeHelper(segments []string) *File {
	if len(segments) == 1 {
		node, ok := root.children[segments[0]]

		if !ok || node.nodeType != FileType {
			return nil
		}

		return &node.file
	}

	subdirName := segments[0]
	node, ok := root.children[subdirName]

	if !ok || node.nodeType != DirType {
		return nil
	}

	return node.dir.findNodeHelper(segments[1:])
}

func (root *Dir) findFile(path string) *File {
	segments := strings.Split(path, string(fp.Separator))

	return root.findNodeHelper(segments)
}
