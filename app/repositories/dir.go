package repositories

import (
	fp "path/filepath"
	"strings"
)

func (root *Dir) addNodeHelper(segments []string, object *Object) {
	if len(segments) == 1 {
		root.children[segments[0]] = Node{
			nodeType: FileType,
			file:     *object,
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

	node.dir.addNodeHelper(segments[1:], object)
}

func (root *Dir) addNode(path string, object *Object) {
	segments := strings.Split(path, string(fp.Separator))

	root.addNodeHelper(segments, object)
}
