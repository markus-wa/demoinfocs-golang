package sendtables

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

// TestFindDecoder_Fixed8ReadsRawBytes verifies that "fixed8" fields are read as 8 raw bits. Read as
// a varint, 0xFF would also consume the next byte (CS2 update of 2026-11-11, m_recentKillQueue).
func TestFindDecoder_Fixed8ReadsRawBytes(t *testing.T) {
	t.Parallel()

	f := field{varName: "m_recentKillQueue", varType: "uint8[8]", encoder: "fixed8", fieldType: newFieldType("uint8")}
	dec := findDecoder(&f)
	r := newReader([]byte{0xFF, 0x01, 0x80})

	assert.Equal(t, uint64(0xFF), dec(r))
	assert.Equal(t, uint64(0x01), dec(r))
	assert.Equal(t, uint64(0x80), dec(r))
	assert.Zero(t, r.remBytes())
}

// TestFindDecoder_Fixed16And32 verifies the wider fixedN encoders read little-endian raw bits.
func TestFindDecoder_Fixed16And32(t *testing.T) {
	t.Parallel()

	f16 := field{encoder: "fixed16", fieldType: newFieldType("uint16")}
	assert.Equal(t, uint64(0xBEEF), findDecoder(&f16)(newReader([]byte{0xEF, 0xBE})))

	f32 := field{encoder: "fixed32", fieldType: newFieldType("uint32")}
	assert.Equal(t, uint64(0xDEADBEEF), findDecoder(&f32)(newReader([]byte{0xEF, 0xBE, 0xAD, 0xDE})))
}

// TestFindDecoder_Uint8WithoutEncoderStaysVarint guards the default: no encoder is still a varint.
func TestFindDecoder_Uint8WithoutEncoderStaysVarint(t *testing.T) {
	t.Parallel()

	f := field{varName: "m_x", varType: "uint8", fieldType: newFieldType("uint8")}

	assert.Equal(t, uint64(300), findDecoder(&f)(newReader([]byte{0xAC, 0x02})))
}

// TestFindDecoder_FixedSignedKeepsInt32 verifies that signed fixedN fields are sign-extended to
// int32, the type signedDecoder returns, so PropertyValue.Int() (e.g. Player.RankType() on
// m_iCompetitiveRankType, an int8 with "fixed8") doesn't panic.
func TestFindDecoder_FixedSignedKeepsInt32(t *testing.T) {
	t.Parallel()

	f8 := field{varName: "m_iCompetitiveRankType", varType: "int8", encoder: "fixed8", fieldType: newFieldType("int8")}
	dec := findDecoder(&f8)
	r := newReader([]byte{0xFF, 0x0B, 0x80, 0x7F})

	assert.Equal(t, int32(-1), dec(r))
	assert.Equal(t, int32(11), dec(r))
	assert.Equal(t, int32(-128), dec(r))
	assert.Equal(t, int32(127), dec(r))
	assert.Zero(t, r.remBytes())

	assert.Equal(t, -1, PropertyValue{Any: findDecoder(&f8)(newReader([]byte{0xFF}))}.Int())

	f16 := field{encoder: "fixed16", fieldType: newFieldType("int16")}
	assert.Equal(t, int32(-2), findDecoder(&f16)(newReader([]byte{0xFE, 0xFF})))

	f32 := field{encoder: "fixed32", fieldType: newFieldType("int32")}
	assert.Equal(t, int32(-3), findDecoder(&f32)(newReader([]byte{0xFD, 0xFF, 0xFF, 0xFF})))
}

// TestFindDecoderByBaseType_Fixed8VectorElements verifies the elements of a uint8 vector with
// "fixed8" (e.g. m_networkAnimTiming) are read as raw bytes.
func TestFindDecoderByBaseType_Fixed8VectorElements(t *testing.T) {
	t.Parallel()

	f := field{varName: "m_networkAnimTiming", encoder: "fixed8", fieldType: newFieldType("CNetworkUtlVectorBase< uint8 >")}
	dec := findDecoderByBaseType(&f)
	r := newReader([]byte{0xFF, 0x80})

	assert.Equal(t, uint64(0xFF), dec(r))
	assert.Equal(t, uint64(0x80), dec(r))
	assert.Zero(t, r.remBytes())
}
