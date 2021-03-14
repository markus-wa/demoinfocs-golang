package sendtables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPropertyValue_BoolVal(t *testing.T) {
	t.Parallel()

	assert.True(t, PropertyValue{IntVal: 1}.BoolVal())
	assert.False(t, PropertyValue{IntVal: 0}.BoolVal())
}

func TestPropertyValue_IsSet(t *testing.T) {
	t.Parallel()

	assert.True(t, PropertyValue{isSet: true}.IsSet())
	assert.False(t, PropertyValue{isSet: false}.IsSet())
}

func TestDecodeProp_UnknownType(t *testing.T) {
	t.Parallel()

	prop := &property{entry: &flattenedPropEntry{prop: &sendTableProperty{rawType: -1}}}

	f := func() {
		propDecoder.decodeProp(prop, nil)
	}

	assert.Panics(t, f)
}
