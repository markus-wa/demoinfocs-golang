package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFloat_Nil(t *testing.T) {
	assert.Zero(t, getFloat(nil, "test"))
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
