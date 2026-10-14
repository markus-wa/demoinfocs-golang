package sendtablescs2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReader_ReadBytes_Aligned covers the byte-aligned fast path, including
// reading exactly the remaining bytes and the bounds panic past them.
func TestReader_ReadBytes_Aligned(t *testing.T) {
	t.Parallel()

	data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A}
	r := newReader(data)

	assert.Equal(t, []byte{0x12, 0x34}, r.readBytes(2))
	assert.Equal(t, []byte{0x56, 0x78, 0x9A}, r.readBytes(3))

	// Exact boundary: reading the remaining bytes must not panic.
	assert.Equal(t, []byte{}, r.readBytes(0))

	assert.Panics(t, func() {
		r.readBytes(1)
	})
}

// TestReader_ReadBytes_Unaligned covers the bit-packed path, including the
// exact boundary and bit-correct output after an initial misalignment.
func TestReader_ReadBytes_Unaligned(t *testing.T) {
	t.Parallel()

	// Consume the low 4 bits of the first byte, shifting the stream by 4 bits.
	assert.Equal(t, uint32(0x2), newReader([]byte{0x12}).readBits(4))

	data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A}
	r := newReader(data)

	assert.Equal(t, uint32(0x2), r.readBits(4))
	assert.Equal(t, []byte{0x41, 0x63, 0x85}, r.readBytes(3))
	// The remaining stream still lines up correctly.
	assert.Equal(t, uint32(0xA7), r.readBits(8))

	assert.Panics(t, func() {
		r.readBytes(1)
	})
}

// TestReader_ReadBytes_CorruptLength ensures a wire-decoded length that
// exceeds the remaining buffer is rejected before any allocation.
// Before the bounds check was hoisted, the unaligned path allocated up to
// uint32-max bytes (OOM) before panicking.
func TestReader_ReadBytes_CorruptLength(t *testing.T) {
	t.Parallel()

	aligned := newReader(make([]byte, 8))
	requirePanicContaining(t, "readBytes: insufficient buffer", func() {
		aligned.readBytes(0xFFFFFFFF)
	})

	unaligned := newReader(make([]byte, 8))
	unaligned.readBits(3)
	requirePanicContaining(t, "readBytes: insufficient buffer", func() {
		unaligned.readBytes(0xFFFFFFFF)
	})
}

func requirePanicContaining(t *testing.T, contains string, f func()) {
	t.Helper()

	defer func() {
		r := recover()
		require.NotNil(t, r)
		msg, ok := r.(string)
		require.True(t, ok)
		assert.Contains(t, msg, contains)
	}()

	f()
}
