package common

import (
	"math/rand"

	quickhull "github.com/markus-wa/quickhull-go"

	r3 "github.com/golang/geo/r3"
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

// UniqueID returns the unique id of the inferno.
// The unique id is a random int generated internally by this library and can be used to differentiate
// infernos from each other. This is needed because demo-files reuse entity ids.
func (inf Inferno) UniqueID() int64 {
	return inf.uniqueID
}

// Active returns an Inferno containing only the active fires of the original.
// The returned Inferno will have the same Unique-ID as the original.
func (inf Inferno) Active() Inferno {
	res := Inferno{
		uniqueID: inf.uniqueID,
	}

	res.Fires = make([]*Fire, 0, len(inf.Fires))
	for _, f := range inf.Fires {
		if f.IsBurning {
			res.Fires = append(res.Fires, f)
		}
	}

	return res
}

// ConvexHull2D returns the vertices making up the 2D convex hull of all the fires in the inferno.
// Useful for drawing on 2D maps.
func (inf Inferno) ConvexHull2D() []r3.Vector {
	pointCloud := make([]r3.Vector, 0, len(inf.Fires))

	for _, f := range inf.Fires {
		pointCloud = append(pointCloud, r3.Vector{
			X: f.Vector.X,
			Y: f.Vector.Y,
			Z: 0,
		})
	}

	return quickhull.ConvexHull(pointCloud)
}

// ConvexHull3D returns the vertices making up the 3D convex hull of all the fires in the inferno.
func (inf Inferno) ConvexHull3D() []r3.Vector {
	pointCloud := make([]r3.Vector, len(inf.Fires))

	for i, f := range inf.Fires {
		pointCloud[i] = f.Vector
	}

	return quickhull.ConvexHull(pointCloud)
}

// NewInferno creates a inferno and sets the Unique-ID.
//
// Intended for internal use only.
func NewInferno() *Inferno {
	return &Inferno{uniqueID: rand.Int63()}
}
