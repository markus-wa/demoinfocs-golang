package sendtables2

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestQAngleFactory_Noscale32Bit verifies that a QAngle with 32-bit components is decoded as
// full-precision noscale float32s (raw IEEE bits) rather than as scaled bit-angles mapped into
// [0,360). See the m_aimPunchAngle / m_pAimPunchServices.m_predictableBaseAngle cases.
func TestQAngleFactory_Noscale32Bit(t *testing.T) {
	t.Parallel()

	bc := int32(32)
	f := field{bitCount: &bc}

	dec := qangleFactory(&f)

	raw := []byte{}
	for _, v := range []float32{1.5, -4.25, 350.0} {
		bits := math.Float32bits(v)
		for i := 0; i < 4; i++ {
			raw = append(raw, byte(bits>>(8*i))) // little-endian
		}
	}

	val := dec(newReader(raw)).([3]float32)

	assert.Equal(t, [3]float32{1.5, -4.25, 350.0}, val)
}

// TestQAngleFactory_ScaledBitAngle still reads scaled bit-angles for sub-32-bit fields, so the
// noscale guard doesn't regress the common smaller encodings.
func TestQAngleFactory_ScaledBitAngle(t *testing.T) {
	t.Parallel()

	bc := int32(20)
	f := field{bitCount: &bc}

	dec := qangleFactory(&f)

	// A raw 20-bit value of 0x80000 (largest magnitude) maps to ~359.65 degrees.
	val := dec(newReader([]byte{0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x08, 0x00})).([3]float32)

	assert.Less(t, val[0], float32(360.0))
	assert.Less(t, val[1], float32(360.0))
	assert.Less(t, val[2], float32(360.0))
}
