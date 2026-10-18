package common

import (
	"fmt"
	"sort"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/quickhull-go/v2"

	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// Inferno is a list of Fires with helper functions.
// Also contains already extinguished fires.
//
// See also: Inferno.Active() and Fire.IsBurning
type Inferno struct {
	Entity st.Entity

	// uniqueID is used to distinguish different infernos (which potentially have the same, reused entityID) from each other.
	uniqueID         UniqueID
	demoInfoProvider demoInfoProvider
	thrower          *Player

	// fireCountProp & fireProps cache the property handles Fires() reads (m_fireCount and the per-fire properties
	// by index): consumers tracking positions call it for every inferno on every frame, formatting the property
	// names and looking them up each time was most of its cost. The entity never changes for an inferno.
	fireCountProp st.Property
	fireProps     []fireProps
}

// fireProps are the property handles of one fire of an inferno, see Inferno.Fires().
type fireProps struct {
	isBurning st.Property
	position  st.Property // m_firePositions.X, nil on demos where the fires are networked as deltas to the origin
	xDelta    st.Property
	yDelta    st.Property
	zDelta    st.Property
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

// UniqueID returns the unique id of the inferno, see [UniqueID] for its guarantees.
func (inf *Inferno) UniqueID() UniqueID {
	return inf.uniqueID
}

// Thrower returns the player who threw the fire grenade.
// Could be nil if the player disconnected after throwing it.
func (inf *Inferno) Thrower() *Player {
	if inf.thrower != nil {
		return inf.thrower
	}

	handleProp := inf.Entity.Property("m_hOwnerEntity").Value()
	if inf.demoInfoProvider.IsSource2() {
		return inf.demoInfoProvider.FindPlayerByPawnHandle(handleProp.Handle())
	}

	return inf.demoInfoProvider.FindPlayerByHandle(uint64(handleProp.Int()))
}

// Fires returns all fires (past + present).
// Some are currently active and some have extinguished (see Fire.IsBurning).
func (inf *Inferno) Fires() Fires {
	entity := inf.Entity

	if inf.fireCountProp == nil {
		inf.fireCountProp = entity.Property("m_fireCount")
	}

	// m_fireCount is the total number of fire positions ever spawned for this inferno, not the number
	// currently burning - it only ever increases and never returns to 0. Use Fires().Active() (gated
	// on m_bFireIsBurning) for the fires burning right now.
	nFires := mustPropertyValue(inf.fireCountProp, "m_fireCount").Int()
	fires := make([]Fire, 0, nFires)

	var (
		origin         r3.Vector
		originResolved bool
	)

	for i := 0; i < nFires; i++ {
		if i >= len(inf.fireProps) {
			inf.fireProps = append(inf.fireProps, inf.resolveFireProps(i))
		}

		props := inf.fireProps[i]

		fire := Fire{
			IsBurning: mustPropertyValue(props.isBurning, "m_bFireIsBurning").BoolVal(),
		}

		if props.position != nil {
			fire.Vector = props.position.Value().R3Vec()
		} else {
			if !originResolved {
				origin = entity.Position()
				originResolved = true
			}

			offset := r3.Vector{
				X: float64(mustPropertyValue(props.xDelta, "m_fireXDelta").Int()),
				Y: float64(mustPropertyValue(props.yDelta, "m_fireYDelta").Int()),
				Z: float64(mustPropertyValue(props.zDelta, "m_fireZDelta").Int()),
			}
			fire.Vector = origin.Add(offset)
		}

		fires = append(fires, fire)
	}

	return Fires{s: fires}
}

// resolveFireProps looks up the property handles of the inferno's fire i.
func (inf *Inferno) resolveFireProps(i int) fireProps {
	iFormat := "%03d"
	if inf.demoInfoProvider.IsSource2() {
		iFormat = "%04d"
	}

	iStr := fmt.Sprintf(iFormat, i)
	entity := inf.Entity

	return fireProps{
		isBurning: entity.Property("m_bFireIsBurning." + iStr),
		position:  entity.Property("m_firePositions." + iStr),
		xDelta:    entity.Property("m_fireXDelta." + iStr),
		yDelta:    entity.Property("m_fireYDelta." + iStr),
		zDelta:    entity.Property("m_fireZDelta." + iStr),
	}
}

// mustPropertyValue returns the property's value, panicking like Entity.PropertyValueMust() if it's nil.
func mustPropertyValue(prop st.Property, name string) st.PropertyValue {
	if prop == nil {
		panic(fmt.Sprintf("property '%s' not found", name))
	}

	return prop.Value()
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

// List returns fires a list of the raw Fire entities. This can be useful
// if you need to do custom calculations on the fires.
func (f Fires) List() []Fire {
	return f.s
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
	det := float64((a.X-s.center.X)*(b.Y-s.center.Y)) - float64((b.X-s.center.X)*(a.Y-s.center.Y))
	if det < 0 {
		return true
	}

	if det > 0 {
		return false
	}

	// points a and b are on the same line from the s.center
	// check which point is closer to the s.center
	d1 := float64((a.X-s.center.X)*(a.X-s.center.X)) + float64((a.Y-s.center.Y)*(a.Y-s.center.Y))
	d2 := float64((b.X-s.center.X)*(b.X-s.center.X)) + float64((b.Y-s.center.Y)*(b.Y-s.center.Y))

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
// The id must come from the creating Parser's unique-id source so that it is unique within that
// parse and reproducible across parses of the same demo.
//
// Intended for internal use only.
func NewInferno(demoInfoProvider demoInfoProvider, entity st.Entity, thrower *Player, id UniqueID) *Inferno {
	return &Inferno{
		Entity:           entity,
		uniqueID:         id,
		demoInfoProvider: demoInfoProvider,
		thrower:          thrower,
	}
}
