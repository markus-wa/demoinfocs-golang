package bitstream

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"strconv"
)

const (
	bufferSize = 2048 + sled
	sled       = 4
)

type BitReader interface {
	LazyGlobalPosition() uint
	ActualGlobalPosition() uint
	ReadBits(bits uint) []byte
	ReadBit() bool
	ReadByte() byte
	ReadCString(int) string
	ReadSignedInt(uint) int
	ReadFloat() float32
}

type bitReader struct {
	underlying         io.Reader
	buffer             []byte
	offset             uint
	bitsInBuffer       uint
	lazyGlobalPosition uint
}

func (r *bitReader) LazyGlobalPosition() uint {
	return r.lazyGlobalPosition
}

func (r *bitReader) ActualGlobalPosition() uint {
	return r.lazyGlobalPosition + uint(r.offset)
}

func (r *bitReader) ReadBits(bits uint) []byte {
	b := make([]byte, (bits+7)/8)
	r.underlying.Read(b)
	r.advance(bits)
	return b
}

func (r *bitReader) ReadBit() bool {
	res := (r.buffer[(r.offset+1)/8] >> (r.offset % 8)) == 1
	r.advance(1)
	return res
}

func (r *bitReader) advance(bits uint) {
	r.offset += bits
	if r.offset > r.bitsInBuffer {
		// Refill if we reached the sled
		r.refillBuffer()
	}
}

func (r *bitReader) refillBuffer() {
	r.offset -= r.bitsInBuffer // sled bits used remain in offset
	r.lazyGlobalPosition += uint(r.bitsInBuffer)
	// Copy sled to beginning
	copy(r.buffer[0:sled], r.buffer[r.bitsInBuffer:r.bitsInBuffer+sled])
	newBytes, _ := r.underlying.Read(r.buffer[sled:])
	if newBytes != 0 {
		r.bitsInBuffer = uint(newBytes) * 8
	} else {
		// we're done, consume sled
		r.bitsInBuffer = sled * 8
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
	if bits > sled*8 {
		panic("Can't read more than " + strconv.Itoa(sled*8) + " bits for int, sled too small")
	}
	// FIXME buffer pos in statshelix deminf=int(r.offset)/8 & ~3
	val := binary.LittleEndian.Uint64(r.buffer[int(r.offset)/8:])
	return uint(val << (64 - bits) >> (64 - bits))
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

// ReadSignedInt is like ReadInt but returns signed int
func (r *bitReader) ReadSignedInt(bits uint) int {
	if bits > sled*8 {
		panic("Can't read more than " + strconv.Itoa(sled*8) + " bits for int, sled too small")
	}
	val := binary.LittleEndian.Uint64(r.buffer[int(r.offset)/8:])
	r.advance(bits)
	// Cast to int64 before right shift
	return int(int64(val<<(64-bits)) >> (64 - bits))
}

func (r *bitReader) ReadFloat() float32 {
	bits := r.ReadInt(32)
	return math.Float32frombits(uint32(bits))
}

func NewBitReader(underlying io.Reader) BitReader {
	br := &bitReader{underlying: underlying, buffer: make([]byte, bufferSize)}
	br.refillBuffer()
	br.offset = sled * 8
	return BitReader(br)
}
