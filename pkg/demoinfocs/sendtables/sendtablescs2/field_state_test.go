package sendtablescs2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldStateSetGetRoundTrip(t *testing.T) {
	s := newFieldState()

	leaf := &fieldPath{path: []int{45}, last: 0}
	s.set(leaf, "leaf")

	assert.Equal(t, "leaf", s.get(leaf))

	nested := &fieldPath{path: []int{3, 129, 7}, last: 2}
	s.set(nested, 42)

	assert.Equal(t, 42, s.get(nested))
}

// Real values observed on a desynced CS2 bitstream; unguarded, the first one
// makes set() allocate ~32 GB in a single call.
func TestFieldStateSetRejectsCorruptIndex(t *testing.T) {
	paths := [][]int{
		{2142459506},
		{3, 2142459506},
		{0, 7, 1829614422},
		{-1},
	}

	for _, path := range paths {
		fp := &fieldPath{path: path, last: len(path) - 1}

		assert.Panics(t, func() {
			newFieldState().set(fp, "x")
		}, "set(%v) should panic", path)
	}
}
