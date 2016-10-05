package bitstream

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

const (
	bufferSize = 2048 + sled
	sled       = 4
)

type BitReader interface {
	LazyGlobalPosition() int
	ActualGlobalPosition() int
	ReadBit() bool
	ReadByte() byte
	ReadBytes(int) []byte
	ReadString() string
	ReadCString(int) string
	ReadSignedInt(uint) int
	ReadInt(uint) uint
	ReadFloat() float32
	BeginChunk(int)
	EndChunk()
}

type stack []int

func (s stack) Push(v int) stack {
	return append(s, v)
}

func (s stack) Pop() (stack, int) {
	// FIXME: CBA to handle empty stacks rn

	l := len(s)
	return s[:l-1], s[l-1]
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
	b := make([]byte, (bits+7)/8)
	r.underlying.Read(b)
	r.advance(bits)
	return b
}

func (r *bitReader) ReadBit() bool {
	res := (r.buffer[r.offset/8] & (1 << uint(r.offset&7))) != 0
	r.advance(1)
	return res
}

func (r *bitReader) advance(bits uint) {
	r.offset += int(bits)
	for r.offset >= r.bitsInBuffer {
		// Refill if we reached the sled
		r.refillBuffer()
	}
}

func (r *bitReader) refillBuffer() {
	r.offset -= r.bitsInBuffer // sled bits used remain in offset
	r.lazyGlobalPosition += r.bitsInBuffer
	// Copy sled to beginning
	copy(r.buffer[0:sled], r.buffer[r.bitsInBuffer/8:r.bitsInBuffer/8+sled])
	newBytes, _ := r.underlying.Read(r.buffer[sled:])

	r.bitsInBuffer = newBytes * 8
	if newBytes < bufferSize-sled {
		// we're done here, consume sled
		r.bitsInBuffer += sled * 8
	}
}

func (r *bitReader) ReadByte() byte {
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
	val := binary.LittleEndian.Uint64(r.buffer[r.offset/8&^3:])
	return uint(val << (64 - (uint(r.offset) % 32) - bits) >> (64 - bits))
}

func (r *bitReader) ReadBytes(bytes int) []byte {
	res := make([]byte, 0, bytes)
	for i := 0; i < bytes; i++ {
		b := r.ReadByte()
		res = append(res, b)
	}
	return res
}

func (r *bitReader) ReadCString(chars int) string {
	b := r.ReadBytes(chars)
	return string(b[:bytes.IndexByte(b, 0)])
}

// ReadString reads a varaible length string
func (r *bitReader) ReadString() string {
	// Valve also uses this sooo
	return r.readStringLimited(4096, false)
}

func (r *bitReader) readStringLimited(limit int, endOnNewLine bool) string {
	result := make([]byte, 0, 512)
	for i := 0; i < limit; i++ {
		b := r.ReadByte()
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
	val := binary.LittleEndian.Uint64(r.buffer[r.offset/8&^3:])
	res := int(int64(val<<(64-(uint(r.offset)%32)-bits)) >> (64 - bits))
	r.advance(bits)
	// Cast to int64 before right shift
	return res
}

func (r *bitReader) ReadFloat() float32 {
	bits := r.ReadInt(32)
	return math.Float32frombits(uint32(bits))
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
			if delta > bufferBits+sled*8 {
				unbufferedSkipBits := delta - bufferBits
				seeker.Seek(int64((unbufferedSkipBits>>3)-sled), io.SeekCurrent)

				newBytes, _ := r.underlying.Read(r.buffer)

				r.bitsInBuffer = 8 * (newBytes - sled)
				if newBytes <= sled {
					// FIXME: Maybe do this even if newBytes is <= bufferSize - sled like in refillBuffer
					// Consume sled
					// Shouldn't really happen unless we reached the end of the stream
					// In that case bitsInBuffer should be 0 after this line (newBytes=0 - sled + sled)
					r.bitsInBuffer += sled * 8
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

func NewBitReader(underlying io.Reader) BitReader {
	br := &bitReader{underlying: underlying, buffer: make([]byte, bufferSize)}
	br.refillBuffer()
	br.offset = sled * 8
	return BitReader(br)
}
