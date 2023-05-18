package sendtables2

import (
	"container/heap"
	"fmt"
)

// Interface for the tree, only implements Weight
type huffmanTree interface {
	Weight() int
	IsLeaf() bool
	Value() int
	Left() huffmanTree
	Right() huffmanTree
}

// A leaf, contains encoded value
type huffmanLeaf struct {
	weight int
	value  int
}

// A node with potential left / right nodes or leafs
type huffmanNode struct {
	weight int
	value  int
	left   huffmanTree
	right  huffmanTree
}

// Return weight for leaf
func (self huffmanLeaf) Weight() int {
	return self.weight
}

// Return leaf state
func (self huffmanLeaf) IsLeaf() bool {
	return true
}

// Return value for leaf
func (self huffmanLeaf) Value() int {
	return self.value
}

func (self huffmanLeaf) Right() huffmanTree {
	_panicf("huffmanLeaf doesn't have right node")
	return nil
}

func (self huffmanLeaf) Left() huffmanTree {
	_panicf("huffmanLeaf doesn't have left node")
	return nil
}

// Return weight for node
func (self huffmanNode) Weight() int {
	return self.weight
}

// Return leaf state
func (self huffmanNode) IsLeaf() bool {
	return false
}

// Return value for node
func (self huffmanNode) Value() int {
	return self.value
}

func (self huffmanNode) Left() huffmanTree {
	return huffmanTree(self.left)
}

func (self huffmanNode) Right() huffmanTree {
	return huffmanTree(self.right)
}

type treeHeap []huffmanTree

// Returns the amount of nodes in the tree
func (th treeHeap) Len() int {
	return len(th)
}

// Weight compare function
func (th treeHeap) Less(i int, j int) bool {
	if th[i].Weight() == th[j].Weight() {
		return th[i].Value() >= th[j].Value()
	} else {
		return th[i].Weight() < th[j].Weight()
	}
}

// Append item, required for heap
func (th *treeHeap) Push(ele interface{}) {
	*th = append(*th, ele.(huffmanTree))
}

// Remove item, required for heap
func (th *treeHeap) Pop() (popped interface{}) {
	popped = (*th)[len(*th)-1]
	*th = (*th)[:len(*th)-1]
	return
}

// Swap two items, required for heap
func (th treeHeap) Swap(i, j int) {
	th[i], th[j] = th[j], th[i]
}

// Construct a tree from a map of weight -> item
func buildHuffmanTree(symFreqs []int) huffmanTree {
	var trees treeHeap
	for v, w := range symFreqs {
		if w == 0 {
			w = 1
		}

		trees = append(trees, &huffmanLeaf{w, v})
	}

	n := 40

	heap.Init(&trees)
	for trees.Len() > 1 {
		a := heap.Pop(&trees).(huffmanTree)
		b := heap.Pop(&trees).(huffmanTree)

		heap.Push(&trees, &huffmanNode{a.Weight() + b.Weight(), n, a, b})
		n++
	}

	return heap.Pop(&trees).(huffmanTree)
}

// Swap two nodes based on the given path
func swapNodes(tree huffmanTree, path uint32, len uint32) {
	for len > 0 {
		// get current bit
		len--
		one := path & 1
		path = path >> 1

		// check if we are correct
		if tree.IsLeaf() {
			_panicf("Touching leaf in node swap, %d left in path", len)
		}

		// switch on the type
		if one == 1 {
			tree = tree.Right()
		} else {
			tree = tree.Left()
		}
	}

	node := tree.(*huffmanNode)
	node.left, node.right = node.right, node.left
}

// Print computed tree order
func printCodes(tree huffmanTree, prefix []byte) {
	if tree == nil {
		return
	}

	if tree.IsLeaf() {
		node := tree.(*huffmanLeaf)
		fmt.Printf("%v\t%d\t%d\t%s\n", node.Value(), node.Weight(), len(prefix), string(prefix))
	} else {
		prefix = append(prefix, '0')
		printCodes(tree.Left(), prefix)
		prefix = prefix[:len(prefix)-1]

		prefix = append(prefix, '1')
		printCodes(tree.Right(), prefix)
		prefix = prefix[:len(prefix)-1]
	}
}

// Used to create a huffman tree by hand
// path: Numeric representation of path to follow
// value: Value for given path
// value_default: Default value set for empty branches / leafs
func addNode(tree huffmanTree, path int, path_len int, value int) huffmanTree {
	root := tree
	for path_len > 1 {
		if tree.IsLeaf() {
			_panicf("Trying to add node to leaf")
		}

		// get the current bit
		path_len--
		one := path & 1
		path = path >> 1

		// add node / leaf
		if one == 1 {
			if tree.Right() != nil {
				tree = tree.Right()
			} else {
				tree.(*huffmanNode).right = &huffmanNode{0, 0, nil, nil}
				tree = tree.Right()
			}
		} else {
			if tree.Left() != nil {
				tree = tree.Left()
			} else {
				tree.(*huffmanNode).left = &huffmanNode{0, 0, nil, nil}
				tree = tree.Left()
			}
		}
	}

	// set value
	one := path & 1
	path = path >> 1

	if one == 1 {
		tree.(*huffmanNode).right = huffmanLeaf{0, value}
	} else {
		tree.(*huffmanNode).left = huffmanLeaf{0, value}
	}

	return root
}
