package examples_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v5/examples"
)

func TestGetMapMetadata(t *testing.T) {
	meta := examples.GetMapMetadata("de_dust2")

	assert.Equal(t, examples.Map{
		PosX:  -2476,
		PosY:  3239,
		Scale: 4.4,
	}, meta)
}
