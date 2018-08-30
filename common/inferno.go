package common

import (
	"math/rand"

	r3 "github.com/golang/geo/r3"
	s2 "github.com/golang/geo/s2"
)

// Inferno is a list of Fires with helper functions.
// Also contains already extinguished fires.
//
// See also: Inferno.Active() and Fire.IsBurning
type Inferno struct {
	EntityID int
	Fires    []*Fire

	// uniqueID is used to distinguish different infernos (which potentially have the same, reused entityID) from each other.
	uniqueID int64
}

// Fire is a component of an Inferno.
type Fire struct {
	r3.Vector

	IsBurning bool
}

// Active returns an Inferno containing only the active fires of the original.
// The returned Inferno will have the same Unique-ID as the original.
func (inf Inferno) Active() Inferno {
	res := Inferno{
		uniqueID: inf.uniqueID,
	}

	res.Fires = make([]*Fire, 0, len(inf.Fires))
	for _, f := range inf.Fires {
		res.Fires = append(res.Fires, f)
	}

	return res
}

// ConvexHull2D returns the 2D convex hull of all the fires in the inferno.
// Useful for drawing on 2D maps.
func (inf Inferno) ConvexHull2D() *s2.Loop {
	q := s2.NewConvexHullQuery()

	for _, f := range inf.Fires {
		q.AddPoint(s2.Point{
			Vector: r3.Vector{
				X: f.Vector.X,
				Y: f.Vector.Y,
				Z: 1,
			},
		})
	}

	return q.ConvexHull()
}

// ConvexHull3D returns the 3D convex hull of all the fires in the inferno.
func (inf Inferno) ConvexHull3D() *s2.Loop {
	q := s2.NewConvexHullQuery()

	for _, f := range inf.Fires {
		q.AddPoint(s2.Point{Vector: f.Vector})
	}

	return q.ConvexHull()
}

// NewInferno creates a inferno and sets the Unique-ID.
//
// Intended for internal use only.
func NewInferno() *Inferno {
	return &Inferno{uniqueID: rand.Int63()}
}
