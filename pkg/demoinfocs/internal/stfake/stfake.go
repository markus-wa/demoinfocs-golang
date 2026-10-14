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
package stfake

import (
	"math"

	"github.com/golang/geo/r3"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stbridge"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables/sendtablescs2"
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
	cellWidth = 1 << 9 // see sendtablescs2.coordFromCell()
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
	Parser *sendtablescs2.Parser
	// Class is the name of the entity's server-class.
	Class  string
	ID     int
	Serial int
	// Props are all properties of the entity, in field-path order - the order in which an update fires
	// their handlers. Declaring a property twice panics.
	Props []Prop
}

// NewParser returns a send-tables parser that knows server-classes with the given names, and nothing else.
func NewParser(classes ...string) *sendtablescs2.Parser {
	return stbridge.NewParser(classes).(*sendtablescs2.Parser)
}

// New returns a new synthetic entity.
//
// The entity isn't created yet - its handlers don't fire until Create is called.
func New(spec Spec) *sendtablescs2.Entity {
	names, values := flatten(spec.Props)

	return stbridge.NewEntity(spec.Parser, spec.Class, spec.ID, spec.Serial, names, values).(*sendtablescs2.Entity)
}

// NewEntity returns a new synthetic entity with the given ID and properties, without parser & server-class.
func NewEntity(id int, props ...Prop) *sendtablescs2.Entity {
	return New(Spec{ID: id, Props: props})
}

// Create fires what an entity packet that creates the entity fires, in the same order:
// the server-class's created-handlers, OnCreateFinished delegates, the parser's entity handlers
// (Created|Entered), all update handlers with the current values (in registration order)
// and finally the OnUpdateFinished delegates.
func Create(e *sendtablescs2.Entity) {
	stbridge.Create(e)
}

// Set sets the given property values without firing any handlers.
// Setting a property that wasn't declared panics.
func Set(e *sendtablescs2.Entity, props ...Prop) {
	names, values := flatten(props)
	stbridge.Set(e, names, values, false, false)
}

// Update applies the given property values like one entity packet updating the entity does:
// each property is set and its update handlers fire before the next one is set, in field-path
// (declaration) order regardless of the order given. Then the OnUpdateFinished delegates
// and the parser's entity handlers (Updated) fire.
// Setting a property that wasn't declared panics.
func Update(e *sendtablescs2.Entity, props ...Prop) {
	names, values := flatten(props)
	stbridge.Set(e, names, values, true, true)
}

// UpdateUnfinished is like Update, but stops after the update handlers fired:
// neither the OnUpdateFinished delegates nor the parser's entity handlers are called.
func UpdateUnfinished(e *sendtablescs2.Entity, props ...Prop) {
	names, values := flatten(props)
	stbridge.Set(e, names, values, true, false)
}

// Destroy deletes the entity like an entity packet does: the OnDestroy delegates fire,
// then the parser's entity handlers (Left|Deleted). Destroying an entity again does nothing.
func Destroy(e *sendtablescs2.Entity) {
	stbridge.Destroy(e)
}
