package examples_test

import (
	"testing"

	"github.com/golang/geo/r2"
	"github.com/markus-wa/demoinfocs-golang/v2/examples"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	"github.com/stretchr/testify/assert"
)

func TestGetMapMetadata(t *testing.T) {
	meta := examples.GetMapMetadata("de_cache", 1901448379)

	assert.Equal(t, metadata.Map{
		Name: "de_cache",
		PZero: r2.Point{
			X: -2000,
			Y: 3250,
		},
		Scale: 5.5,
	}, meta)
}
