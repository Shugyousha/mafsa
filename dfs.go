package mafsa

import "sort"

// depthFirst sends all items on the tree in lexicographical order to its channel.
type depthFirst struct {
	tree    *MinTree
	channel chan string
}

func newDepthFirst(tree *MinTree) *depthFirst {
	return &depthFirst{
		tree:    tree,
		channel: make(chan string),
	}
}

func (df *depthFirst) start() {
	df.search(df.tree.Root, "")
	close(df.channel)
}

func (df *depthFirst) search(node *MinTreeNode, word string) {
	if node.Final {
		df.channel <- string(word)
	} else {
		for _, char := range sortKeys(node.Edges) {
			df.search(node.Edges[char], word+string(char))
		}
	}
}

func sortKeys(m map[rune]*MinTreeNode) (sorted []rune) {
	for r := range m {
		sorted = append(sorted, r)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}
