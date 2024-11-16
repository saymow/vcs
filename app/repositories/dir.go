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
	subdirName := segments[0]

	if _, ok := root.children[subdirName]; ok {
		node = root.children[subdirName]
	} else {
		node = &Node{
			nodeType: DirType,
			dir: &Dir{
				path:     fp.Join(root.path, subdirName),
				children: make(map[string]*Node),
			},
		}
		root.children[subdirName] = node
	}

	node.dir.addNodeHelper(segments[1:], change)
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
	segments := strings.Split(path, string(fp.Separator))

	return root.findNodeHelper(segments)
}

func (root *Dir) findFile(path string) *File {
	node := root.findNode(path)

	if node == nil || node.nodeType == DirType {
		return nil
	}

	return node.file
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

func (root *Dir) collectFiles(path string) []*File {
	node := root.findNode(path)
	files := []*File{}

	if node != nil {
		if node.nodeType == DirType {
			files = node.dir.collectAllFiles()
		} else {
			files = append(files, node.file)
		}
	}

	return files
}
