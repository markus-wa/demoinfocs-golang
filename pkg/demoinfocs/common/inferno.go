package common

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/quickhull-go/v2"

	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
)

// Inferno is a list of Fires with helper functions.
// Also contains already extinguished fires.
//
// See also: Inferno.Active() and Fire.IsBurning
type Inferno struct {
	Entity st.Entity

	// uniqueID is used to distinguish different infernos (which potentially have the same, reused entityID) from each other.
	uniqueID         int64
	demoInfoProvider demoInfoProvider
}

// Fire is a component of an Inferno.
type Fire struct {
	r3.Vector

	IsBurning bool
}

// Fires is a collection of fires that provides utility functions for things like calculation of 2D & 3D convex hulls.
// See also Inferno.
type Fires struct {
	s []Fire
}

// UniqueID returns the unique id of the inferno.
// The unique id is a random int generated internally by this library and can be used to differentiate
// infernos from each other. This is needed because demo-files reuse entity ids.
func (inf *Inferno) UniqueID() int64 {
	return inf.uniqueID
}

// Thrower returns the player who threw the fire grenade.
// Could be nil if the player disconnected after throwing it.
func (inf *Inferno) Thrower() *Player {
	return inf.demoInfoProvider.FindPlayerByHandle(inf.Entity.Property("m_hOwnerEntity").Value().IntVal)
}

// Fires returns all fires (past + present).
// Some are currently active and some have extinguished (see Fire.IsBurning).
func (inf *Inferno) Fires() Fires {
	entity := inf.Entity
	origin := entity.Position()
	nFires := entity.PropertyValueMust("m_fireCount").IntVal
	fires := make([]Fire, 0, nFires)

	for i := 0; i < nFires; i++ {
		iStr := fmt.Sprintf("%03d", i)
		offset := r3.Vector{
			X: float64(entity.PropertyValueMust("m_fireXDelta." + iStr).IntVal),
			Y: float64(entity.PropertyValueMust("m_fireYDelta." + iStr).IntVal),
			Z: float64(entity.PropertyValueMust("m_fireZDelta." + iStr).IntVal),
		}

		fire := Fire{
			Vector:    origin.Add(offset),
			IsBurning: entity.PropertyValueMust("m_bFireIsBurning."+iStr).IntVal == 1,
		}

		fires = append(fires, fire)
	}

	return Fires{s: fires}
}

// Active returns all currently active fires (only Fire.IsBurning == true).
func (f Fires) Active() Fires {
	allFires := f.s
	active := make([]Fire, 0, len(allFires))

	for _, f := range allFires {
		if f.IsBurning {
			active = append(active, f)
		}
	}

	return Fires{s: active}
}

// ConvexHull2D returns clockwise sorted corner points making up the 2D convex hull of all the fires in the inferno.
// Useful for drawing on 2D maps.
func (f Fires) ConvexHull2D() []r2.Point {
	pointCloud := make([]r3.Vector, len(f.s))
	for i, f := range f.s {
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
func (f Fires) ConvexHull3D() quickhull.ConvexHull {
	pointCloud := make([]r3.Vector, len(f.s))

	for i, f := range f.s {
		pointCloud[i] = f.Vector
	}

	return convexHull(pointCloud)
}

func convexHull(pointCloud []r3.Vector) quickhull.ConvexHull {
	return new(quickhull.QuickHull).ConvexHull(pointCloud, false, false, 0)
}

// NewInferno creates a inferno and sets the Unique-ID.
//
// Intended for internal use only.
func NewInferno(demoInfoProvider demoInfoProvider, entity st.Entity) *Inferno {
	return &Inferno{
		Entity:           entity,
		uniqueID:         rand.Int63(),
		demoInfoProvider: demoInfoProvider,
	}
}
