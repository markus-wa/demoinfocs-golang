package sendtablescs2

import (
	"container/heap"
)

// fpHuffNode is a node in a flat, array-backed huffman tree.
// When left == -1, the node is a leaf and value holds the op index.
// When left >= 0, the node is internal and left/right hold child indices.
type fpHuffNode struct {
	left  int16
	right int16
	value int16
}

// fpHuffNodes is the pre-built flat huffman tree used by readFieldPaths.
// The root is always at index 0.
var fpHuffNodes []fpHuffNode

func init() {
	fpHuffNodes = buildFlatHuffmanTree(newHuffmanTree())
}

// buildFlatHuffmanTree converts an interface-based huffman tree into a flat
// slice representation, eliminating interface dispatch during traversal.
// The root node is always placed at index 0 (pre-order layout).
func buildFlatHuffmanTree(t huffmanTree) []fpHuffNode {
	nodes := make([]fpHuffNode, 0, 128)
	var build func(t huffmanTree) int16
	build = func(t huffmanTree) int16 {
		idx := int16(len(nodes))
		nodes = append(nodes, fpHuffNode{})
		if t.IsLeaf() {
			nodes[idx].left = -1
			nodes[idx].value = int16(t.Value())
		} else {
			leftIdx := build(t.Left())
			rightIdx := build(t.Right())
			nodes[idx].left = leftIdx
			nodes[idx].right = rightIdx
		}
		return idx
	}
	build(t)
	return nodes
}

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
