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

const (
	// fpHuffBits is the number of bits peeked per fpHuffLUT lookup.
	// Must cover the majority of code lengths in one lookup; the observed
	// max code length for the field-path table is 17.
	fpHuffBits = 10
	// fpHuffLeafMarker marks fpHuffLUT entries that resolve to a leaf.
	// node then stores ^op so the marker cannot collide with a tree index.
	fpHuffLeafMarker = int16(-0x8000)
)

// fpHuffLUTEntry is one fpHuffLUT slot: the tree node to continue from
// (fpHuffLeafMarker|~op for leaves) and the number of bits consumed.
type fpHuffLUTEntry struct {
	consumed uint32
	node     int16
}

// fpHuffLUT is a peek table built from fpHuffNodes; indexed by the next
// fpHuffBits bits of the stream. The reader's bit order is LSB-first
// (next bit = LSB of bitVal), so stream bit i lives at index bit i.
var fpHuffLUT [1 << fpHuffBits]fpHuffLUTEntry

// buildFpHuffLUT fills fpHuffLUT from fpHuffNodes.
// For every tree leaf at depth <= fpHuffBits, all table indices whose low
// `depth` bits are the leaf's code (first-read bit = LSB) are filled with
// that leaf. Deeper codes resolve to the tree node reached after fpHuffBits
// bits (node >= 0), which readFieldPaths continues from with the per-bit walk.
func buildFpHuffLUT() {
	var fill func(node int16, depth int, code uint32)

	fill = func(node int16, depth int, code uint32) {
		n := fpHuffNodes[node]

		if n.left < 0 {
			entry := fpHuffLUTEntry{consumed: uint32(depth), node: fpHuffLeafMarker | ^n.value}

			// leaf: replicate across the remaining (high) padding
			shift := uint32(fpHuffBits - depth)
			for tail := uint32(0); tail < 1<<shift; tail++ {
				fpHuffLUT[code|(tail<<depth)] = entry
			}

			return
		}

		if depth == fpHuffBits {
			entry := fpHuffLUTEntry{consumed: fpHuffBits, node: node}
			fpHuffLUT[code] = entry

			return
		}

		// first-read bit = LSB: left (0) keeps the bit unset, right (1) sets it
		fill(n.left, depth+1, code)
		fill(n.right, depth+1, code|1<<depth)
	}

	fill(0, 0, 0)
}

func init() {
	fpHuffNodes = buildFlatHuffmanTree(newHuffmanTree())

	buildFpHuffLUT()
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
func (l huffmanLeaf) Weight() int {
	return l.weight
}

// Return leaf state
func (l huffmanLeaf) IsLeaf() bool {
	return true
}

// Return value for leaf
func (l huffmanLeaf) Value() int {
	return l.value
}

func (huffmanLeaf) Right() huffmanTree {
	_panicf("huffmanLeaf doesn't have right node")
	return nil
}

func (huffmanLeaf) Left() huffmanTree {
	_panicf("huffmanLeaf doesn't have left node")
	return nil
}

// Return weight for node
func (n huffmanNode) Weight() int {
	return n.weight
}

// Return leaf state
func (huffmanNode) IsLeaf() bool {
	return false
}

// Return value for node
func (n huffmanNode) Value() int {
	return n.value
}

func (n huffmanNode) Left() huffmanTree {
	return huffmanTree(n.left) //nolint:unconvert
}

func (n huffmanNode) Right() huffmanTree {
	return huffmanTree(n.right) //nolint:unconvert
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
	} else { //nolint:revive
		return th[i].Weight() < th[j].Weight()
	}
}

// Append item, required for heap
func (th *treeHeap) Push(ele any) {
	*th = append(*th, ele.(huffmanTree))
}

// Remove item, required for heap
func (th *treeHeap) Pop() (popped any) {
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
	var trees treeHeap //nolint:prealloc

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
