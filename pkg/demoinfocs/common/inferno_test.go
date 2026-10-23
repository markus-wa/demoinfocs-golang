package common

import (
	"testing"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

func TestInferno_UniqueID(t *testing.T) {
	entity := entityWithID(1)
	assert.NotEqual(t, NewInferno(nil, entity, nil, UniqueID(1)).UniqueID(), NewInferno(nil, entity, nil, UniqueID(2)).UniqueID(), "UniqueIDs of different infernos should be different")
}

func TestFires_Active(t *testing.T) {
	inf := Fires{
		s: []Fire{
			{
				IsBurning: false,
				Vector:    r3.Vector{X: 1, Y: 2, Z: 3},
			},
		},
	}

	assert.Empty(t, inf.Active().s, "Inferno should have no active fires")

	activeFires := []Fire{
		{
			IsBurning: true,
			Vector:    r3.Vector{X: 4, Y: 5, Z: 6},
		},
		{
			IsBurning: true,
			Vector:    r3.Vector{X: 7, Y: 8, Z: 9},
		},
	}
	inf.s = append(inf.s, activeFires...)

	assert.Equal(t, activeFires, inf.Active().s, "Active inferno should contain active fires")
}

func TestInferno_ConvexHull2D(t *testing.T) {
	// Construct a Inferno that looks roughly like this.
	// D should be inside the 2D Convex Hull but a corner of the 3D Convex Hull
	//
	//         C
	//       /   \
	//     /   D   \
	//   /           \
	// A - - - - - - - B
	//
	inf := Fires{
		s: []Fire{
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

	expectedHull := []r2.Point{
		{X: 1, Y: 2},
		{X: 4, Y: 7},
		{X: 7, Y: 2},
	}

	assert.ElementsMatch(t, expectedHull, inf.ConvexHull2D(), "ConvexHull2D should be as expected")

	// 3D-hull should be different
	assert.NotEqual(t, len(expectedHull), len(inf.ConvexHull3D().Vertices), "3D hull should contain the vertex 'D'")
}

// Just check that all fires are passed to quickhull.ConvexHull()
func TestInferno_ConvexHull3D(t *testing.T) {
	inf := Fires{
		s: []Fire{
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
				Vector: r3.Vector{X: 4, Y: 4, Z: 12},
			},
		},
	}

	expectedHull := []r3.Vector{
		{X: 1, Y: 2, Z: 3},
		{X: 4, Y: 7, Z: 6},
		{X: 7, Y: 2, Z: 9},
		{X: 4, Y: 4, Z: 12},
	}

	assert.ElementsMatch(t, expectedHull, inf.ConvexHull3D().Vertices, "ConvexHull3D should contain all fire locations")
}

func TestInferno_Thrower(t *testing.T) {
	entity := entityWithProperty("m_hOwnerEntity", st.PropertyValue{Any: uint64(1)})

	player := new(Player)
	provider := demoInfoProviderMock{
		playersByHandle: map[uint64]*Player{1: player},
	}

	assert.Equal(t, player, NewInferno(provider, entity, nil, UniqueID(1)).Thrower())
}

func TestInferno_Thrower_UnreadableOwner(t *testing.T) {
	provider := demoInfoProviderMock{
		playersByHandle: map[uint64]*Player{1: new(Player)},
	}

	entities := map[string]*st.Entity{
		"missing":     entityWithoutProperty("m_hOwnerEntity"),
		"not decoded": stfake.NewEntity(1, stfake.P("m_hOwnerEntity", nil)),
		"wrong type":  entityWithProperty("m_hOwnerEntity", st.PropertyValue{Any: int32(1)}),
	}

	for name, entity := range entities {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, NewInferno(provider, entity, nil, UniqueID(1)).Thrower())
		})
	}
}

func TestInferno_Thrower_Cached(t *testing.T) {
	thrower := new(Player)

	assert.Same(t, thrower, NewInferno(nil, nil, thrower, UniqueID(1)).Thrower())
}

func TestInferno_Fires_UndecodedProperties(t *testing.T) {
	t.Run("missing count", func(t *testing.T) {
		assert.Empty(t, NewInferno(nil, entityWithID(1), nil, UniqueID(1)).Fires().List())
	})

	t.Run("undecoded count", func(t *testing.T) {
		entity := stfake.NewEntity(1, stfake.P("m_fireCount", nil))

		assert.Empty(t, NewInferno(nil, entity, nil, UniqueID(1)).Fires().List())
	})

	t.Run("undecoded fire properties", func(t *testing.T) {
		entity := stfake.NewEntity(1,
			stfake.P("m_fireCount", int32(3)),
			stfake.P("m_bFireIsBurning.0000", nil),
			stfake.P("m_firePositions.0000", [3]float32{1, 2, 3}),
			stfake.P("m_bFireIsBurning.0001", true),
			stfake.P("m_firePositions.0001", nil),
			stfake.P("m_bFireIsBurning.0002", true),
			stfake.P("m_firePositions.0002", [3]float32{4, 5, 6}),
		)

		expected := []Fire{
			{Vector: r3.Vector{X: 1, Y: 2, Z: 3}},
			{Vector: r3.Vector{X: 4, Y: 5, Z: 6}, IsBurning: true},
		}

		assert.Equal(t, expected, NewInferno(nil, entity, nil, UniqueID(1)).Fires().List())
	})
}

func TestInferno_List(t *testing.T) {
	expected := []Fire{
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
			Vector: r3.Vector{X: 4, Y: 4, Z: 12},
		},
	}
	fires := Fires{
		s: expected,
	}

	got := fires.List()
	assert.ElementsMatch(t, expected, got, "List() should return the fires contained in Fires")
}
