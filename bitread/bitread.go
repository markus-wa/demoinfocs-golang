package bitread

import (
	"github.com/markus-wa/gobitread"
	"io"
	"math"
	"sync"
)

const (
	smallBuffer      = 512
	largeBuffer      = 1024 * 128
	maxVarInt32Bytes = 5
)

type BitReader struct {
	bitread.BitReader
	buffer []byte
}

// ReadString reads a variable length string
func (r *BitReader) ReadString() string {
	// Valve also uses this sooo
	return r.readStringLimited(4096, false)
}

func (r *BitReader) readStringLimited(limit int, endOnNewLine bool) string {
	result := make([]byte, 0, 512)
	for i := 0; i < limit; i++ {
		b := r.ReadSingleByte()
		if b == 0 || (endOnNewLine && b == '\n') {
			break
		}
		result = append(result, b)
	}
	return string(result)
}

func (r *BitReader) ReadFloat() float32 {
	return math.Float32frombits(uint32(r.ReadInt(32)))
}

func (r *BitReader) ReadVarInt32() uint32 {
	var res uint32
	var b uint32 = 0x80
	for count := uint(0); b&0x80 != 0 && count != maxVarInt32Bytes; count++ {
		b = uint32(r.ReadSingleByte())
		res |= (b & 0x7f) << (7 * count)
	}
	return res
}

func (r *BitReader) ReadSignedVarInt32() int32 {
	res := r.ReadVarInt32()
	return int32((res >> 1) ^ -(res & 1))
}

func (r *BitReader) ReadUBitInt() uint {
	res := r.ReadInt(6)
	switch res & (16 | 32) {
	case 16:
		res = (res & 15) | (r.ReadInt(4) << 4)
	case 32:
		res = (res & 15) | (r.ReadInt(8) << 4)
	case 48:
		res = (res & 15) | (r.ReadInt(32-4) << 4)
	}
	return res
}

var bitReaderPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(BitReader)
	},
}

func (r *BitReader) Pool() {
	r.Close()
	if len(r.buffer) == smallBuffer {
		smallBufferPool.Put(r.buffer)
	}
	r.buffer = nil
	bitReaderPool.Put(r)
}

func newBitReader(underlying io.Reader, buffer []byte) *BitReader {
	br := bitReaderPool.Get().(*BitReader)
	br.buffer = buffer
	br.OpenWithBuffer(underlying, buffer)
	return br
}

var smallBufferPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, smallBuffer)
	},
}

func NewSmallBitReader(underlying io.Reader) *BitReader {
	return newBitReader(underlying, smallBufferPool.Get().([]byte))
}

func NewLargeBitReader(underlying io.Reader) *BitReader {
	return newBitReader(underlying, make([]byte, largeBuffer))
}
