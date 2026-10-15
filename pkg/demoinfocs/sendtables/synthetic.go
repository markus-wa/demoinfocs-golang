package sendtables

import (
	"fmt"
	"slices"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stbridge"
)

// Synthetic entities are real *Entity values whose class has a flat serializer of simple fields,
// each named with the full (dotted) property name. Names resolve through the exact-name lookup of
// serializer.getFieldPathForName, so all Entity methods work unchanged.
// They are only used by tests, via internal/stfake, which also keeps track of each entity's parser.

func init() {
	stbridge.NewParser = func(classes []string) any {
		return newSyntheticParser(classes)
	}
	stbridge.NewEntity = func(parser any, class string, id, serial int, names []string, values []any) any {
		return newSyntheticEntity(parser.(*Parser), class, id, serial, names, values)
	}
	stbridge.Create = func(entity, parser any) {
		syntheticCreate(entity.(*Entity), parser.(*Parser))
	}
	stbridge.Set = func(entity, parser any, names []string, values []any, fire, finish bool) {
		syntheticSet(entity.(*Entity), parser.(*Parser), names, values, fire, finish)
	}
	stbridge.Destroy = func(entity, parser any) {
		syntheticDestroy(entity.(*Entity), parser.(*Parser))
	}
}

// syntheticPositionProps are declared by every synthetic entity, Position() needs them.
var syntheticPositionProps = []string{propCellX, propCellY, propCellZ, propVecX, propVecY, propVecZ}

func newSyntheticClass(id int32, name string) *ServerClass {
	return &ServerClass{
		classId:     id,
		name:        name,
		serializer:  newSerializer(name, 0),
		fpNameCache: &fpNameTreeCache{},
		fpFlatCache: make(map[uint64]string),
	}
}

func newSyntheticParser(classes []string) *Parser {
	p := NewParser()

	for i, name := range classes {
		if p.classesByName[name] != nil {
			_panicf("synthetic parser: duplicate server-class %q", name)
		}

		c := newSyntheticClass(int32(i), name)
		p.classesById[c.classId] = c
		p.classesByName[name] = c
	}

	return p
}

// newSyntheticEntity returns a new synthetic entity. p may be nil.
func newSyntheticEntity(p *Parser, className string, id, serial int, names []string, values []any) *Entity {
	classID := int32(-1)

	if p != nil {
		registered := p.classesByName[className]
		if registered == nil {
			_panicf("synthetic entity: unknown server-class %q", className)
		}

		classID = registered.classId
	}

	// Each entity gets its own copy of the class: all props must be declared before the first lookup,
	// lookups cache negative results (serializer.fieldNameChecks, Entity.fpNoop).
	c := newSyntheticClass(classID, className)

	for _, name := range names {
		c.addSyntheticField(name)
	}

	for _, name := range syntheticPositionProps {
		if c.serializer.fieldIndexes[name] == nil {
			c.addSyntheticField(name)
		}
	}

	e := newEntity(int32(id), int32(serial), c)

	syntheticSet(e, nil, names, values, false, false)

	return e
}

func (c *ServerClass) addSyntheticField(name string) {
	if c.serializer.fieldIndexes[name] != nil {
		_panicf("synthetic entity: property %q declared twice", name)
	}

	c.serializer.addField(&field{
		varName:          name,
		name:             name,
		model:            fieldModelSimple,
		polySerializerID: -1,
	})
}

func fireSyntheticEntityHandlers(p *Parser, e *Entity, op EntityOp) {
	if p == nil {
		return
	}

	for _, h := range p.entityHandlers {
		if err := h(e, op); err != nil {
			panic(fmt.Sprintf("entity handler returned an error for entity %d (%v): %v", e.index, op, err))
		}
	}
}

// syntheticCreate mirrors Parser.OnPacketEntities for a newly created entity.
// The created-handlers of p's class (if p isn't nil) fire, not those of the entity's own class copy.
func syntheticCreate(e *Entity, p *Parser) {
	createdHandlers := e.class.createdHandlers
	if p != nil {
		createdHandlers = p.classesByName[e.class.name].createdHandlers
	}

	for _, h := range createdHandlers {
		h(e)
	}

	for _, f := range e.onCreateFinished {
		f()
	}

	fireSyntheticEntityHandlers(p, e, EntityOpCreated|EntityOpEntered)

	e.fireInitialUpdateHandlers()
	e.fireUpdateFinished()
}

// syntheticSet sets the given values in field-path order, like Entity.readFields does.
// With finish, it mirrors Parser.OnPacketEntities for an updated entity.
func syntheticSet(e *Entity, p *Parser, names []string, values []any, fire, finish bool) {
	if len(names) != len(values) {
		_panicf("synthetic entity: %d property names but %d values", len(names), len(values))
	}

	type update struct {
		index int
		value any
	}

	updates := make([]update, 0, len(names))

	for i, name := range names {
		fi := e.class.serializer.fieldIndexes[name]
		if fi == nil {
			_panicf("synthetic entity %d: property %q isn't declared", e.index, name)
		}

		updates = append(updates, update{index: fi.index, value: values[i]})
	}

	// stable: the last value wins if a property is set more than once
	slices.SortStableFunc(updates, func(a, b update) int {
		return a.index - b.index
	})

	// Like Parser.OnPacketEntities, an update re-activates a destroyed entity before its fields are read,
	// also without finish (so Destroy works again afterwards). Only finish fires the entity handlers (Entered).
	op := EntityOpUpdated

	if !e.active {
		e.active = true
		op |= EntityOpEntered
	}

	fp := newFieldPath()
	defer fp.release()

	for _, u := range updates {
		fp.path[0] = u.index
		_ = e.state.set(fp, u.value)

		if fire && e.hasHandlers {
			e.dispatchUpdate(fp, u.value)
		}
	}

	if !finish {
		return
	}

	e.fireUpdateFinished()
	fireSyntheticEntityHandlers(p, e, op)
}

// syntheticDestroy mirrors Parser.OnPacketEntities for a deleted entity.
func syntheticDestroy(e *Entity, p *Parser) {
	if !e.active {
		return // entity has already been destroyed
	}

	e.destroy()
	fireSyntheticEntityHandlers(p, e, EntityOpLeft|EntityOpDeleted)
}
