package sendtablescs2

import (
	"testing"

	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

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

	assert.Equal(t, -1, st.PropertyValue{Any: findDecoder(&f8)(newReader([]byte{0xFF}))}.Int())

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

// TestFindDecoder_FixedKeepsVarintType verifies that fixedN fields keep the Go type the varint
// decoder of their base type returns, including base types outside the int8/16/32 and uint8/16/32
// basics: signed enums and indices (signedDecoder), unsigned handles (unsignedDecoder) and
// unmapped enums such as EKillTypes_t (defaultDecoder, uint32).
func TestFindDecoder_FixedKeepsVarintType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		baseType string
		in       []byte
		want     any
	}{
		{baseType: "CEntityIndex", in: []byte{0xFF}, want: int32(-1)},
		{baseType: "AmmoIndex_t", in: []byte{0x80}, want: int32(-128)},
		{baseType: "AttachmentHandle_t", in: []byte{0xFF}, want: uint64(0xFF)},
		{baseType: "EKillTypes_t", in: []byte{0xFF}, want: uint32(0xFF)},
	}

	for _, tc := range tests {
		f := field{varName: "m_x", varType: tc.baseType, encoder: "fixed8", fieldType: newFieldType(tc.baseType)}
		r := newReader(tc.in)

		assert.Equal(t, tc.want, findDecoder(&f)(r), tc.baseType)
		assert.Zero(t, r.remBytes(), tc.baseType)
	}
}

// TestFindDecoder_FixedIgnoredForNonIntegers verifies that a fixedN encoder doesn't replace the
// decoder of non-integer base types.
func TestFindDecoder_FixedIgnoredForNonIntegers(t *testing.T) {
	t.Parallel()

	fBool := field{varName: "m_b", varType: "bool", encoder: "fixed8", fieldType: newFieldType("bool")}
	assert.Equal(t, true, findDecoder(&fBool)(newReader([]byte{0x01})))

	fFloat := field{varName: "m_f", varType: "float32", encoder: "fixed32", fieldType: newFieldType("float32")}
	assert.Equal(t, float32(1), findDecoder(&fFloat)(newReader([]byte{0x00, 0x00, 0x80, 0x3F})))
}
