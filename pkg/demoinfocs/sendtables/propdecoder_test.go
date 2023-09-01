package sendtables

import (
	"bytes"
	"encoding/binary"
	"testing"

	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
	"github.com/stretchr/testify/assert"
)

func TestPropertyValue_BoolVal(t *testing.T) {
	assert.True(t, PropertyValue{IntVal: 1}.BoolVal())
	assert.False(t, PropertyValue{IntVal: 0}.BoolVal())
}

func TestPropertyValue_Int64Val(t *testing.T) {
	expected := int64(76561198000697560)
	prop := &property{entry: &flattenedPropEntry{prop: &sendTableProperty{rawType: propTypeInt64, flags: propFlagUnsigned, numberOfBits: 64}}}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(expected))
	r := bit.NewSmallBitReader(bytes.NewReader(b))

	propDecoder.decodeProp(prop, r)

	assert.Equal(t, expected, prop.value.Int64Val)
}

func TestDecodeProp_UnknownType(t *testing.T) {
	prop := &property{entry: &flattenedPropEntry{prop: &sendTableProperty{rawType: -1}}}

	f := func() {
		propDecoder.decodeProp(prop, nil)
	}

	assert.Panics(t, f)
}
