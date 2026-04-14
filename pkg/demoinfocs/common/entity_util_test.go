package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func TestGetFloat_Nil(t *testing.T) {
	assert.Zero(t, getFloat(nil, "test"))
}

func TestGetFloatIfExists_Nil(t *testing.T) {
	value, ok := getFloatIfExists(nil, "test")
	assert.Zero(t, value)
	assert.False(t, ok)
}

func TestGetFloatIfExists_Missing(t *testing.T) {
	value, ok := getFloatIfExists(entityWithoutProperty("test"), "test")
	assert.Zero(t, value)
	assert.False(t, ok)
}

func TestGetFloatIfExists_NilValue(t *testing.T) {
	value, ok := getFloatIfExists(entityWithProperty("test", st.PropertyValue{Any: nil}), "test")
	assert.Zero(t, value)
	assert.False(t, ok)
}

func TestGetFloatIfExists_WrongType(t *testing.T) {
	value, ok := getFloatIfExists(entityWithProperty("test", st.PropertyValue{Any: int32(12)}), "test")
	assert.Zero(t, value)
	assert.False(t, ok)
}

func TestGetFloatIfExists(t *testing.T) {
	value, ok := getFloatIfExists(entityWithProperty("test", st.PropertyValue{Any: float32(12.5)}), "test")
	assert.Equal(t, float32(12.5), value)
	assert.True(t, ok)
}

func TestGetInt_Nil(t *testing.T) {
	assert.Zero(t, getInt(nil, "test"))
}

func TestGetString_Nil(t *testing.T) {
	assert.Empty(t, getString(nil, "test"))
}

func TestGetBool_Nil(t *testing.T) {
	assert.Empty(t, getBool(nil, "test"))
}
