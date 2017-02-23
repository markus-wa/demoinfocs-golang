package bitstream

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"sync"
)

const (
	SmallBuffer       = 1024 * 2
	LargeBuffer       = 1024 * 128
	sled              = 4
	kMaxVarintBytes   = 10
	kMaxVarint32Bytes = 5
)

type BitReader interface {
	LazyGlobalPosition() int
	ActualGlobalPosition() int
	ReadBit() bool
	ReadBits(bits uint) []byte
	ReadSingleByte() byte
	ReadBytes(int) []byte
	ReadBytesInto(b *[]byte, bytes int)
	ReadString() string
	ReadCString(int) string
	ReadSignedInt(uint) int
	ReadInt(uint) uint
	ReadVarInt32() uint32
	ReadSignedVarInt32() int32
	ReadFloat() float32
	ReadUBitInt() uint
	BeginChunk(int)
	EndChunk()
	ChunkFinished() bool
	Close()
}

// A simple int stack
type stack []int

func (s stack) Push(v int) stack {
	return append(s, v)
}

// Pop returns the stack without the last added item as well as said item (seperately)
// Attention: panics when the stack is empty
func (s stack) Pop() (stack, int) {
	// TODO: CBA to handle empty stacks rn
	l := len(s)
	return s[:l-1], s[l-1]
}

func (s stack) Top() int {
	return s[len(s)-1]
}

type bitReader struct {
	underlying         io.Reader
	buffer             []byte
	offset             int
	bitsInBuffer       int
	lazyGlobalPosition int
	chunkTargets       stack
}

func (r *bitReader) LazyGlobalPosition() int {
	return r.lazyGlobalPosition
}

func (r *bitReader) ActualGlobalPosition() int {
	return r.lazyGlobalPosition + r.offset
}

func (r *bitReader) ReadBits(bits uint) []byte {
	b := make([]byte, (bits+7)>>3)
	for i := uint(0); i < bits>>3; i++ {
		b[i] = r.ReadSingleByte()
	}
	if bits&7 != 0 {
		b[bits>>3] = r.ReadBitsToByte(bits & 7)
	}
	return b
}

func (r *bitReader) ReadBit() bool {
	res := (r.buffer[r.offset>>3] & (1 << uint(r.offset&7))) != 0
	r.advance(1)
	return res
}

func (r *bitReader) advance(bits uint) {
	r.offset += int(bits)
	if r.offset >= r.bitsInBuffer {
		// Refill if we reached the sled
		r.refillBuffer()
	}
}

func (r *bitReader) refillBuffer() {
	// Copy sled to beginning
	copy(r.buffer[0:sled], r.buffer[r.bitsInBuffer>>3:(r.bitsInBuffer>>3)+sled])

	r.offset -= r.bitsInBuffer // sled bits used remain in offset
	r.lazyGlobalPosition += r.bitsInBuffer

	newBytes, _ := r.underlying.Read(r.buffer[sled:])

	r.bitsInBuffer = newBytes << 3
	if newBytes < len(r.buffer)-(sled<<1) {
		// we're done here, consume sled
		r.bitsInBuffer += sled << 3
	}
}

func (r *bitReader) ReadSingleByte() byte {
	return r.readByteInternal(true)
}

func (r *bitReader) readByteInternal(bitLevel bool) byte {
	if !bitLevel {
		res := r.buffer[r.offset>>3]
		r.advance(8)
		return res
	}
	return r.ReadBitsToByte(8)
}

func (r *bitReader) ReadBitsToByte(bits uint) byte {
	if bits > 8 {
		panic("Can't read more than 8 bits into byte")
	}
	return byte(r.ReadInt(bits))
}

func (r *bitReader) ReadInt(bits uint) uint {
	res := r.peekInt(bits)
	r.advance(bits)
	return res
}

func (r *bitReader) peekInt(bits uint) uint {
	if bits > 32 {
		panic("Can't read more than 32 bits for uint")
	}
	val := binary.LittleEndian.Uint64(r.buffer[r.offset>>3&^3:])
	return uint(val << (64 - (uint(r.offset) & 31) - bits) >> (64 - bits))
}

func (r *bitReader) ReadBytes(bytes int) []byte {
	b := make([]byte, 0, bytes)
	r.ReadBytesInto(&b, bytes)
	return b
}

func (r *bitReader) ReadBytesInto(b *[]byte, bytes int) {
	bitLevel := r.offset&7 != 0
	if !bitLevel && r.offset+(bytes<<3) < r.bitsInBuffer {
		// Shortcut if all bytes are already buffered
		*b = append(*b, r.buffer[r.offset>>3:r.offset>>3+bytes]...)
		r.advance(uint(bytes) << 3)
	} else {
		for i := 0; i < bytes; i++ {
			*b = append(*b, r.readByteInternal(bitLevel))
		}
	}
}

func (r *bitReader) ReadCString(chars int) string {
	b := r.ReadBytes(chars)
	end := bytes.IndexByte(b, 0)
	if end < 0 {
		end = chars
	}
	return string(b[:end])
}

// ReadString reads a varaible length string
func (r *bitReader) ReadString() string {
	// Valve also uses this sooo
	return r.readStringLimited(4096, false)
}

func (r *bitReader) readStringLimited(limit int, endOnNewLine bool) string {
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

// ReadSignedInt is like ReadInt but returns signed int
func (r *bitReader) ReadSignedInt(bits uint) int {
	if bits > 32 {
		panic("Can't read more than 32 bits for int")
	}
	val := binary.LittleEndian.Uint64(r.buffer[r.offset>>3&^3:])
	// Cast to int64 before right shift & use offset before advance
	res := int(int64(val<<(64-(uint(r.offset)&31)-bits)) >> (64 - bits))
	r.advance(bits)
	return res
}

func (r *bitReader) ReadFloat() float32 {
	bits := r.ReadInt(32)
	return math.Float32frombits(uint32(bits))
}

func (r *bitReader) ReadVarInt32() uint32 {
	var res uint32 = 0
	var b uint32 = 0x80

	// TODO: This seems strange (Maybe check statshelix implementation)
	for count := uint(0); b&0x80 != 0; count++ {
		if count == kMaxVarint32Bytes {
			return res
		}
		b = uint32(r.ReadSingleByte())
		res |= (b & 0x7f) << (7 * count)
	}
	return res
}

func (r *bitReader) ReadSignedVarInt32() int32 {
	res := r.ReadVarInt32()
	return int32((res >> 1) ^ -(res & 1))
}

func (r *bitReader) ReadUBitInt() uint {
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

func (r *bitReader) BeginChunk(length int) {
	r.chunkTargets = r.chunkTargets.Push(r.ActualGlobalPosition() + length)
}

func (r *bitReader) EndChunk() {
	var target int
	r.chunkTargets, target = r.chunkTargets.Pop()
	delta := target - r.ActualGlobalPosition()
	if delta < 0 {
		panic("Someone read beyond a chunk boundary, what a dick")
	} else if delta > 0 {
		// We must seek for peace (or the end of the boundary, for a start)
		seeker, ok := r.underlying.(io.Seeker)
		if ok {
			bufferBits := r.bitsInBuffer - r.offset
			if delta > bufferBits+sled<<3 {
				unbufferedSkipBits := delta - bufferBits
				seeker.Seek(int64((unbufferedSkipBits>>3)-sled), io.SeekCurrent)

				newBytes, _ := r.underlying.Read(r.buffer)

				r.bitsInBuffer = (newBytes - sled) << 3
				if newBytes <= sled {
					// TODO: Maybe do this even if newBytes is <= bufferSize - sled like in refillBuffer
					// Consume sled
					// Shouldn't really happen unless we reached the end of the stream
					// In that case bitsInBuffer should be 0 after this line (newBytes=0 - sled + sled)
					r.bitsInBuffer += sled << 3
				}

				r.offset = unbufferedSkipBits & 7
				r.lazyGlobalPosition = target - r.offset
			} else {
				// no seek necessary
				r.advance(uint(delta))
			}
		} else {
			// Canny seek, do it manually
			r.advance(uint(delta))
		}
	}
}

func (r *bitReader) ChunkFinished() bool {
	return r.chunkTargets.Top() == r.ActualGlobalPosition()
}

func (r *bitReader) Close() {
	r.underlying = nil
	r.offset = 0
	r.bitsInBuffer = 0
	r.chunkTargets = r.chunkTargets[:0]
	r.lazyGlobalPosition = 0
	bitReaderPool.Put(r)
}

var bitReaderPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(bitReader)
	},
}

func NewBitReader(underlying io.Reader, bufferSize int) BitReader {
	br := bitReaderPool.Get().(*bitReader)
	br.underlying = underlying
	bufferSize += sled
	if len(br.buffer) != bufferSize {
		br.buffer = make([]byte, bufferSize)
	}
	br.refillBuffer()
	br.offset = sled << 3
	return BitReader(br)
}
