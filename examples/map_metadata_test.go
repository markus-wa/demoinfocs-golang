package examples_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v4/examples"
)

func TestGetMapMetadata(t *testing.T) {
	t.Skip()

	meta := examples.GetMapMetadata("de_cache", 1901448379)

	assert.Equal(t, examples.Map{
		PosX:  -2000,
		PosY:  3250,
		Scale: 5.5,
	}, meta)
}
