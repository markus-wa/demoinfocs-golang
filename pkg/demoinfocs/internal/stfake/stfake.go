// Package stfake creates synthetic send-table entities for tests.
//
// Synthetic entities are real entities of the send-tables parser, whose server-class declares
// exactly the given properties as flat fields named with the full property name.
// All entity methods (Property, PropertyValue, OnUpdate, Bind, Position, OnDestroy, ...) work like
// they do for parsed entities. Properties that aren't declared don't exist - declare every property
// the code under test reads.
//
// Every entity additionally declares the CBodyComponent cell & offset properties used by
// Entity.Position() (unset, i.e. position (0,0,0), unless Pos is given).
//
// Each synthetic entity has its own copy of its server-class, declaring exactly that entity's
// properties. So e.ServerClass() is never the *ServerClass registered with the Spec's Parser, but its
// Name() and ID() match. A per-parser class with the union of all its entities' properties isn't
// used: entities of the same class would get each other's properties, and the class caches
// negative property lookups, so properties declared after the first lookup wouldn't be found.
// Handlers registered on the parser's class (OnEntityCreated) fire on Create; handlers registered
// on e.ServerClass() only do if the entity has no Parser.
package stfake

import (
	"math"
	"sync"

	"github.com/golang/geo/r3"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stbridge"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// Prop is one or more property values, see P and Pos.
type Prop struct {
	names  []string
	values []any
}

// P returns a property with the given name and value.
// The value must have the type the parser decodes it to (e.g. uint64 for handles, int32 for ints),
// nil means the property is declared but has no value.
// A st.PropertyValue is unwrapped.
func P(name string, value any) Prop {
	if pv, ok := value.(st.PropertyValue); ok {
		value = pv.Any
	}

	return Prop{names: []string{name}, values: []any{value}}
}

const (
	cellWidth = 1 << 9 // see sendtables.coordFromCell()
	maxCoord  = 16384
)

func cellAndOffset(coord float64) (uint64, float32) {
	cell := math.Floor((coord + maxCoord) / cellWidth)

	return uint64(cell), float32(coord + maxCoord - cell*cellWidth)
}

// Pos returns the CBodyComponent cell & offset properties for the given position.
func Pos(pos r3.Vector) Prop {
	cellX, vecX := cellAndOffset(pos.X)
	cellY, vecY := cellAndOffset(pos.Y)
	cellZ, vecZ := cellAndOffset(pos.Z)

	return Prop{
		names: []string{
			"CBodyComponent.m_cellX", "CBodyComponent.m_cellY", "CBodyComponent.m_cellZ",
			"CBodyComponent.m_vecX", "CBodyComponent.m_vecY", "CBodyComponent.m_vecZ",
		},
		values: []any{cellX, cellY, cellZ, vecX, vecY, vecZ},
	}
}

func flatten(props []Prop) (names []string, values []any) {
	for _, p := range props {
		names = append(names, p.names...)
		values = append(values, p.values...)
	}

	return names, values
}

// Spec specifies a synthetic entity.
type Spec struct {
	// Parser is optional. If set, Class must be one of its server-classes; creating the entity fires the
	// class's created-handlers and the parser's entity handlers (Parser.OnEntity).
	Parser *st.Parser
	// Class is the name of the entity's server-class.
	Class  string
	ID     int
	Serial int
	// Props are all properties of the entity, in field-path order - the order in which an update fires
	// their handlers. Declaring a property twice panics.
	Props []Prop
}

// parsers maps each synthetic *st.Entity that has a Spec.Parser to that parser.
// Entities without a parser have no entry. Entries are never removed, so parser-bound entities
// live as long as the test binary - acceptable for tests.
var parsers sync.Map // *st.Entity -> *st.Parser

// parserOf returns e's Spec.Parser, nil if it has none.
func parserOf(e *st.Entity) *st.Parser {
	p, _ := parsers.Load(e)
	parser, _ := p.(*st.Parser)

	return parser
}

// NewParser returns a send-tables parser that knows server-classes with the given names, and nothing else.
func NewParser(classes ...string) *st.Parser {
	return stbridge.NewParser(classes).(*st.Parser)
}

// New returns a new synthetic entity.
//
// The entity isn't created yet - its handlers don't fire until Create is called.
func New(spec Spec) *st.Entity {
	names, values := flatten(spec.Props)

	e := stbridge.NewEntity(spec.Parser, spec.Class, spec.ID, spec.Serial, names, values).(*st.Entity)
	if spec.Parser != nil {
		parsers.Store(e, spec.Parser)
	}

	return e
}

// NewEntity returns a new synthetic entity with the given ID and properties, without parser & server-class.
func NewEntity(id int, props ...Prop) *st.Entity {
	return New(Spec{ID: id, Props: props})
}

// Create fires what an entity packet that creates the entity fires, in the same order:
// the server-class's created-handlers, OnCreateFinished delegates, the parser's entity handlers
// (Created|Entered), all update handlers with the current values (in registration order)
// and finally the OnUpdateFinished delegates.
func Create(e *st.Entity) {
	stbridge.Create(e, parserOf(e))
}

// Set sets the given property values without firing any handlers.
// Like an update, it re-activates a destroyed entity and brings back an entity that left the PVS
// (without firing anything).
// Setting a property that wasn't declared panics.
func Set(e *st.Entity, props ...Prop) {
	names, values := flatten(props)
	stbridge.Set(e, parserOf(e), names, values, false, false)
}

// Update applies the given property values like one entity packet updating the entity does:
// each property is set and its update handlers fire before the next one is set, in field-path
// (declaration) order regardless of the order given. Then the OnUpdateFinished delegates
// and the parser's entity handlers (Updated, plus Entered if the entity had been destroyed or left the PVS) fire.
// If the entity had left the PVS (see Leave), its OnEnterPVS delegates fire before the entity handlers.
// Setting a property that wasn't declared panics.
func Update(e *st.Entity, props ...Prop) {
	names, values := flatten(props)
	stbridge.Set(e, parserOf(e), names, values, true, true)
}

// UpdateUnfinished is like Update, but stops after the update handlers fired:
// neither the OnUpdateFinished and OnEnterPVS delegates nor the parser's entity handlers are called.
func UpdateUnfinished(e *st.Entity, props ...Prop) {
	names, values := flatten(props)
	stbridge.Set(e, parserOf(e), names, values, true, false)
}

// Destroy deletes the entity like an entity packet does: the OnDestroy delegates fire,
// then the parser's entity handlers (Left|Deleted). Destroying an entity again does nothing,
// unless it was re-activated by Set, Update or UpdateUnfinished in between.
func Destroy(e *st.Entity) {
	stbridge.Destroy(e, parserOf(e))
}

// Leave makes the entity leave the PVS without deleting it, like an entity packet does:
// the OnLeavePVS delegates fire, then the parser's entity handlers (Left).
// Leaving again does nothing, unless the entity came back with Set, Update or UpdateUnfinished in between.
// Destroying it afterwards works like for an entity in the PVS.
func Leave(e *st.Entity) {
	stbridge.Leave(e, parserOf(e))
}
