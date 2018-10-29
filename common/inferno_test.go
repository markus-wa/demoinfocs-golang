package common

import (
	"testing"

	r3 "github.com/golang/geo/r3"
	s2 "github.com/golang/geo/s2"
	assert "github.com/stretchr/testify/assert"
)

func TestInfernoUniqueID(t *testing.T) {
	assert.NotEqual(t, NewInferno().UniqueID(), NewInferno().UniqueID(), "UniqueIDs of different infernos should be different")
}

func TestInfernoActive(t *testing.T) {
	inf := Inferno{
		Fires: []*Fire{
			{
				IsBurning: false,
				Vector:    r3.Vector{X: 1, Y: 2, Z: 3},
			},
		},
	}

	assert.Empty(t, inf.Active().Fires, "Inferno should have no active fires")

	activeFires := []*Fire{
		{
			IsBurning: true,
			Vector:    r3.Vector{X: 4, Y: 5, Z: 6},
		},
		{
			IsBurning: true,
			Vector:    r3.Vector{X: 7, Y: 8, Z: 9},
		},
	}
	inf.Fires = append(inf.Fires, activeFires...)

	assert.Equal(t, activeFires, inf.Active().Fires, "Active inferno should contain active fires")
}

func TestInfernoConvexHull2D(t *testing.T) {
	// Construct a  Inferno that looks roughly like this.
	// D should be inside the 2D Convex Hull but a corner of the 3D Convex Hull
	//
	//         C
	//       /   \
	//     /   D   \
	//   /           \
	// A - - - - - - - B
	//
	inf := Inferno{
		Fires: []*Fire{
			{
				Vector: r3.Vector{X: 1, Y: 2, Z: 3},
			},
			{
				Vector: r3.Vector{X: 4, Y: 7, Z: 6},
			},
			{
				Vector: r3.Vector{X: 7, Y: 2, Z: 9},
			},
			{
				Vector: r3.Vector{X: 4, Y: 4, Z: 12}, // This fire is inside the 2D hull
			},
		},
	}

	expectedHull := s2.LoopFromPoints([]s2.Point{
		{Vector: r3.Vector{X: 4, Y: 7, Z: 1}},
		{Vector: r3.Vector{X: 1, Y: 2, Z: 1}},
		{Vector: r3.Vector{X: 7, Y: 2, Z: 1}},
	})

	assert.ElementsMatch(t, expectedHull.Vertices(), inf.ConvexHull2D().Vertices(), "ConvexHull2D should be as expected")
	assert.True(t, inf.ConvexHull2D().BoundaryEqual(expectedHull), "Boundary of expected and actual should be equal")

	// 3D-hull should be different
	assert.NotEqual(t, expectedHull.NumVertices(), inf.ConvexHull3D().NumVertices(), "3D hull should contain the vertex 'D'")
}
