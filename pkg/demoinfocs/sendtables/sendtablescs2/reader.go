package sendtablescs2

import (
	"encoding/binary"
	"math"
	"sync"
)

// f32CacheEntry caches a pre-boxed interface{} for a given float32 bit pattern.
type f32CacheEntry struct {
	bits  uint32
	boxed interface{}
}

// reader performs read operations against a buffer
type reader struct {
	buf      []byte
	size     uint32
	pos      uint32
	bitVal   uint64 // value of the remaining bits in the current byte
	bitCount uint32 // number of remaining bits in the current byte
	strBuf   []byte // reusable buffer for readString
	// f32Cache is a direct-mapped cache of pre-boxed float32 interface{} values.
	// Indexed by (bits & f32CacheMask); same bits == same float32 value so reuse is safe.
	// Retained across pool reuse to maximise hit rate; no reset needed on newReader.
	f32Cache [512]f32CacheEntry
}

const f32CacheMask = uint32(512 - 1) // must match f32Cache array size

// cachedFloat32 returns a pre-boxed interface{} for the given float32 bit pattern,
// allocating and caching on miss. Zero is handled by the caller where possible.
func (r *reader) cachedFloat32(bits uint32) interface{} {
	// Mix high bits (exponent+sign) into the index to reduce collisions for
	// clusters of similar values (e.g., nearby positions, velocities).
	idx := (bits ^ (bits >> 16)) & f32CacheMask
	e := &r.f32Cache[idx]
	if e.bits == bits && e.boxed != nil {
		return e.boxed
	}
	v := interface{}(math.Float32frombits(bits))
	e.bits = bits
	e.boxed = v
	return v
}

var readerPool = sync.Pool{
	New: func() any { return &reader{} },
}

// newReader returns a reader for buf, reusing pooled instances where possible.
func newReader(buf []byte) *reader {
	r := readerPool.Get().(*reader)
	r.buf = buf
	r.size = uint32(len(buf)) //nolint:gosec
	r.pos = 0
	r.bitVal = 0
	r.bitCount = 0
	// strBuf is intentionally kept to reuse its backing array
	return r
}

// release returns the reader to the pool for reuse.
func (r *reader) release() {
	r.buf = nil
	readerPool.Put(r)
}

// remBytes calculates the number of unread bytes in the buffer
func (r *reader) remBytes() uint32 {
	return r.size - r.pos
}

// nextByte reads the next byte from the buffer.
// The panic is in a separate noinline function so this hot path can be inlined.
func (r *reader) nextByte() byte {
	if r.pos >= r.size {
		r.nextBytePanic()
	}
	x := r.buf[r.pos]
	r.pos++
	return x
}

//go:noinline
func (r *reader) nextBytePanic() {
	_panicf("nextByte: insufficient buffer (%d of %d)", r.pos, r.size)
}

// readBits returns the uint32 value for the given number of sequential bits
func (r *reader) readBits(n uint32) uint32 {
	for n > r.bitCount {
		r.bitVal |= uint64(r.nextByte()) << r.bitCount
		r.bitCount += 8
	}

	x := (r.bitVal & ((1 << n) - 1))
	r.bitVal >>= n
	r.bitCount -= n

	return uint32(x) //nolint:gosec
}

// readByte reads a single byte
func (r *reader) readByte() byte {
	// Fast path if we're byte aligned
	if r.bitCount == 0 {
		return r.nextByte()
	}

	return byte(r.readBits(8))
}

// readBytes reads the given number of bytes
func (r *reader) readBytes(n uint32) []byte {
	// Fast path if we're byte aligned
	if r.bitCount == 0 {
		r.pos += n

		if r.pos > r.size {
			_panicf("readBytes: insufficient buffer (%d of %d)", r.pos, r.size)
		}

		return r.buf[r.pos-n : r.pos]
	}

	buf := make([]byte, n)

	for i := uint32(0); i < n; i++ {
		buf[i] = byte(r.readBits(8))
	}

	return buf
}

// readLeUint32 reads an little-endian uint32
func (r *reader) readLeUint32() uint32 {
	// Fast path if we're byte aligned
	if r.bitCount == 0 {
		return binary.LittleEndian.Uint32(r.readBytes(4))
	}

	return r.readBits(32)
}

// readLeUint64 reads a little-endian uint64
func (r *reader) readLeUint64() uint64 {
	return binary.LittleEndian.Uint64(r.readBytes(8))
}

// readVarUint64 reads an unsigned 32-bit varint
func (r *reader) readVarUint32() uint32 {
	var x, s uint32
	for {
		b := uint32(r.readByte())
		x |= (b & 0x7F) << s
		s += 7
		if ((b & 0x80) == 0) || (s == 35) {
			break
		}
	}

	return x
}

// readVarInt64 reads a signed 32-bit varint
func (r *reader) readVarInt32() int32 {
	ux := r.readVarUint32()
	x := int32(ux >> 1) //nolint:gosec
	if ux&1 != 0 {
		x = ^x
	}
	return x
}

// readVarUint64 reads an unsigned 64-bit varint
func (r *reader) readVarUint64() uint64 {
	var x, s uint64
	for i := 0; ; i++ {
		b := r.readByte()
		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				_panicf("read overflow: varint overflows uint64")
			}
			return x | uint64(b)<<s
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
}

// readBoolean reads and interprets a single bit as true or false.
// Implemented as a direct bit extraction rather than calling readBits(1)
// so that this hot function can be inlined by the compiler.
// The refill is extracted into a noinline helper to keep the budget low.
func (r *reader) readBoolean() bool {
	if r.bitCount == 0 {
		r.refillByte()
	}
	b := r.bitVal&1 == 1
	r.bitVal >>= 1
	r.bitCount--
	return b
}

//go:noinline
func (r *reader) refillByte() {
	if r.pos >= r.size {
		r.nextBytePanic()
	}
	r.bitVal = uint64(r.buf[r.pos])
	r.pos++
	r.bitCount = 8
}

// readUBitVar reads a variable length uint32 with encoding in last to bits of 6 bit group
func (r *reader) readUBitVar() uint32 {
	ret := r.readBits(6)

	switch ret & 0x30 {
	case 16:
		ret = (ret & 15) | (r.readBits(4) << 4)

	case 32:
		ret = (ret & 15) | (r.readBits(8) << 4)

	case 48:
		ret = (ret & 15) | (r.readBits(28) << 4)
	}

	return ret
}

// readUBitVarFP reads a variable length uint32 encoded using fieldpath encoding
func (r *reader) readUBitVarFP() uint32 {
	if r.readBoolean() {
		return r.readBits(2)
	}
	if r.readBoolean() {
		return r.readBits(4)
	}
	if r.readBoolean() {
		return r.readBits(10)
	}
	if r.readBoolean() {
		return r.readBits(17)
	}
	return r.readBits(31)
}

func (r *reader) readUBitVarFieldPath() int {
	return int(r.readUBitVarFP())
}

// readString reads a null terminated string
func (r *reader) readString() string {
	r.strBuf = r.strBuf[:0]
	for {
		b := r.readByte()
		if b == 0 {
			break
		}
		r.strBuf = append(r.strBuf, b)
	}

	return string(r.strBuf)
}

// readCoord reads a coord as a float32
func (r *reader) readCoord() float32 {
	value := float32(0.0)

	intval := r.readBits(1)
	fractval := r.readBits(1)
	signbit := false

	if intval != 0 || fractval != 0 {
		signbit = r.readBoolean()

		if intval != 0 {
			intval = r.readBits(14) + 1
		}

		if fractval != 0 {
			fractval = r.readBits(5)
		}

		value = float32(intval) + float32(fractval)*(1.0/(1<<5))

		// Fixup the sign if negative.
		if signbit {
			value = -value
		}
	}

	return value
}

// readAngle reads a bit angle of the given size
func (r *reader) readAngle(n uint32) float32 {
	return float32(r.readBits(n)) * float32(360.0) / float32(int(1<<n))
}

// readNormal reads a normalized float vector
func (r *reader) readNormal() float32 {
	isNeg := r.readBoolean()
	len := r.readBits(11) //nolint:revive
	ret := float32(len) * float32(1.0/(float32(1<<11)-1.0))

	if isNeg {
		return -ret
	} else { //nolint:revive
		return ret
	}
}

// read3BitNormal reads a normalized float vector
func (r *reader) read3BitNormal() [3]float32 {
	var ret [3]float32

	hasX := r.readBoolean()
	hasY := r.readBoolean()

	if hasX {
		ret[0] = r.readNormal()
	}

	if hasY {
		ret[1] = r.readNormal()
	}

	negZ := r.readBoolean()
	prodsum := ret[0]*ret[0] + ret[1]*ret[1]

	if prodsum < 1.0 {
		ret[2] = float32(math.Sqrt(float64(1.0 - prodsum)))
	} else {
		ret[2] = 0.0
	}

	if negZ {
		ret[2] = -ret[2]
	}

	return ret
}
