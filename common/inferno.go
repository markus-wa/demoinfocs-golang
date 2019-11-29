package common

import (
	"math/rand"
	"sort"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/quickhull-go/v2"

	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// Inferno is a list of Fires with helper functions.
// Also contains already extinguished fires.
//
// See also: Inferno.Active() and Fire.IsBurning
type Inferno struct {
	Entity   st.IEntity
	EntityID int // Same as Entity.ID(), use Entity.ID() instead
	Fires    []*Fire

	// uniqueID is used to distinguish different infernos (which potentially have the same, reused entityID) from each other.
	uniqueID         int64
	demoInfoProvider demoInfoProvider
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

// ConvexHull2D returns clockwise sorted corner points making up the 2D convex hull of all the fires in the inferno.
// Useful for drawing on 2D maps.
func (inf Inferno) ConvexHull2D() []r2.Point {
	pointCloud := make([]r3.Vector, len(inf.Fires))
	for i, f := range inf.Fires {
		pointCloud[i] = f.Vector
		pointCloud[i].Z = 0
	}

	vertices := convexHull(convexHull(pointCloud).Vertices).Vertices
	points := make([]r2.Point, len(vertices))
	for i, v := range vertices {
		points[i] = r2.Point{X: v.X, Y: v.Y}
	}
	sortPointsClockwise(points)

	return points
}

// pointsClockwiseSorter implements the Sort interface for slices of Point
// with a comparator for sorting points in clockwise order around their center.
type pointsClockwiseSorter struct {
	center r2.Point
	points []r2.Point
}

func (s pointsClockwiseSorter) Len() int { return len(s.points) }

func (s pointsClockwiseSorter) Swap(i, j int) { s.points[i], s.points[j] = s.points[j], s.points[i] }

func (s pointsClockwiseSorter) Less(i, j int) bool {
	a, b := s.points[i], s.points[j]

	if a.X-s.center.X >= 0 && b.X-s.center.X < 0 {
		return true
	}
	if a.X-s.center.X < 0 && b.X-s.center.X >= 0 {
		return false
	}
	if a.X-s.center.X == 0 && b.X-s.center.X == 0 {
		if a.Y-s.center.Y >= 0 || b.Y-s.center.Y >= 0 {
			return a.Y > b.Y
		}
		return b.Y > a.Y
	}

	// compute the cross product of vectors (s.center -> a) X (s.center -> b)
	det := (a.X-s.center.X)*(b.Y-s.center.Y) - (b.X-s.center.X)*(a.Y-s.center.Y)
	if det < 0 {
		return true
	}
	if det > 0 {
		return false
	}

	// points a and b are on the same line from the s.center
	// check which point is closer to the s.center
	d1 := (a.X-s.center.X)*(a.X-s.center.X) + (a.Y-s.center.Y)*(a.Y-s.center.Y)
	d2 := (b.X-s.center.X)*(b.X-s.center.X) + (b.Y-s.center.Y)*(b.Y-s.center.Y)
	return d1 > d2
}

func sortPointsClockwise(points []r2.Point) {
	sorter := pointsClockwiseSorter{
		center: r2.RectFromPoints(points...).Center(),
		points: points,
	}
	sort.Sort(sorter)
}

// ConvexHull3D returns the 3D convex hull of all the fires in the inferno.
func (inf Inferno) ConvexHull3D() quickhull.ConvexHull {
	pointCloud := make([]r3.Vector, len(inf.Fires))

	for i, f := range inf.Fires {
		pointCloud[i] = f.Vector
	}

	return convexHull(pointCloud)
}

// Owner returns the player who threw the fire grenade.
// Could be nil if the player disconnected after throwing it.
//
// Deprecated: Owner() exists for historical compatibility
// and should not be used. Use Thrower() instead.
func (inf Inferno) Owner() *Player {
	return inf.demoInfoProvider.FindPlayerByHandle(inf.Entity.FindPropertyI("m_hOwnerEntity").Value().IntVal)
}

// Thrower is a more consistent name than Owner
func (inf Inferno) Thrower() *Player {
	return inf.demoInfoProvider.FindPlayerByHandle(inf.Entity.FindPropertyI("m_hOwnerEntity").Value().IntVal)
}

func convexHull(pointCloud []r3.Vector) quickhull.ConvexHull {
	return new(quickhull.QuickHull).ConvexHull(pointCloud, false, false, 0)
}

// NewInferno creates a inferno and sets the Unique-ID.
//
// Intended for internal use only.
func NewInferno(demoInfoProvider demoInfoProvider, entity st.IEntity) *Inferno {
	return &Inferno{
		Entity:           entity,
		EntityID:         entity.ID(),
		uniqueID:         rand.Int63(),
		demoInfoProvider: demoInfoProvider,
	}
}
