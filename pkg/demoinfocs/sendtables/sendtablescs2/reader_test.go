package sendtablescs2

import "testing"

type mockReader struct {
	data     []byte
	bitVal   uint64
	bitCount uint32
	pos      int
}

func (r *mockReader) nextByte() byte {
	if r.pos >= len(r.data) {
		return 0
	}
	b := r.data[r.pos]
	r.pos++
	return b
}

func (r *mockReader) readBits(n uint32) uint32 {
	for n > r.bitCount {
		r.bitVal |= uint64(r.nextByte()) << r.bitCount
		r.bitCount += 8
	}
	x := (r.bitVal & ((1 << n) - 1))
	r.bitVal >>= n
	r.bitCount -= n
	return uint32(x)
}

func (r *mockReader) readBoolean() bool {
	return r.readBits(1) == 1
}

func TestBooleanRead(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected struct{ X, Y, Z bool }
	}{
		{"All true", []byte{0b00000111}, struct{ X, Y, Z bool }{true, true, true}},
		{"All false", []byte{0b00000000}, struct{ X, Y, Z bool }{false, false, false}},
		{"Mixed", []byte{0b00000100}, struct{ X, Y, Z bool }{false, false, true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &mockReader{data: tt.data}

			// Original approach
			hasX := r.readBoolean()
			hasY := r.readBoolean()
			hasZ := r.readBoolean()

			if hasX != tt.expected.X || hasY != tt.expected.Y || hasZ != tt.expected.Z {
				t.Errorf("Original: got (%v, %v, %v), want (%v, %v, %v)",
					hasX, hasY, hasZ, tt.expected.X, tt.expected.Y, tt.expected.Z)
			}

			// Reset reader
			r = &mockReader{data: tt.data}
			bits := r.readBits(3)

			// Proposed approach
			hasX = bits&1 != 0
			hasY = bits&2 != 0
			hasZ = bits&4 != 0

			if hasX != tt.expected.X || hasY != tt.expected.Y || hasZ != tt.expected.Z {
				t.Errorf("Proposed: got (%v, %v, %v), want (%v, %v, %v)",
					hasX, hasY, hasZ, tt.expected.X, tt.expected.Y, tt.expected.Z)
			}
		})
	}
}
