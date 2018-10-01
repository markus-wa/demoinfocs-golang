package common

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func TestInfernoUniqueID(t *testing.T) {
	assert.NotEqual(t, NewInferno().UniqueID(), NewInferno().UniqueID(), "UniqueIDs of different infernos should be different")
}
