package common

import (
	"testing"

	r3 "github.com/golang/geo/r3"
	assert "github.com/stretchr/testify/assert"
)

func TestInfernoUniqueID(t *testing.T) {
	assert.NotEqual(t, NewInferno().UniqueID(), NewInferno().UniqueID(), "UniqueIDs of different infernos should be different")
}

func TestInfernoActive(t *testing.T) {
	inf := Inferno{
		Fires: []*Fire{
			&Fire{
				IsBurning: false,
				Vector:    r3.Vector{X: 1, Y: 2, Z: 3},
			},
		},
	}

	assert.Empty(t, inf.Active().Fires, "Inferno should have no active fires")

	activeFires := []*Fire{
		&Fire{
			IsBurning: true,
			Vector:    r3.Vector{X: 4, Y: 5, Z: 6},
		},
		&Fire{
			IsBurning: true,
			Vector:    r3.Vector{X: 7, Y: 8, Z: 9},
		},
	}
	inf.Fires = append(inf.Fires, activeFires...)

	assert.Equal(t, activeFires, inf.Active().Fires, "Active inferno should contain active fires")
}
