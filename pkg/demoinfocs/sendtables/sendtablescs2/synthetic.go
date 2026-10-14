package sendtablescs2

import (
	"fmt"
	"slices"
	"sync"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stbridge"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// Synthetic entities are real *Entity values whose class has a flat serializer of simple fields,
// each named with the full (dotted) property name. Names resolve through the exact-name lookup of
// serializer.getFieldPathForName, so all Entity methods work unchanged.
// They are only used by tests, via internal/stfake.

func init() {
	stbridge.NewParser = func(classes []string) any {
		return newSyntheticParser(classes)
	}
	stbridge.NewEntity = func(parser any, class string, id, serial int, names []string, values []any) any {
		p, _ := parser.(*Parser)

		return newSyntheticEntity(p, class, id, serial, names, values)
	}
	stbridge.Create = func(entity any) {
		syntheticCreate(entity.(*Entity))
	}
	stbridge.Set = func(entity any, names []string, values []any, fire, finish bool) {
		syntheticSet(entity.(*Entity), names, values, fire, finish)
	}
	stbridge.Destroy = func(entity any) {
		syntheticDestroy(entity.(*Entity))
	}
}

// syntheticPositionProps are declared by every synthetic entity, Position() needs them.
var syntheticPositionProps = []string{propCellX, propCellY, propCellZ, propVecX, propVecY, propVecZ}

type syntheticInfo struct {
	parser *Parser
	class  *class // the parser's class, whose created-handlers fire on creation; nil without parser
}

// syntheticInfos maps a synthetic *Entity to its *syntheticInfo.
var syntheticInfos sync.Map

func newSyntheticClass(id int32, name string) *class {
	return &class{
		classId:     id,
		name:        name,
		serializer:  newSerializer(name, 0),
		fpNameCache: &fpNameTreeCache{},
		fpFlatCache: make(map[uint64]string),
	}
}

func newSyntheticParser(classes []string) *Parser {
	p := NewParser(nil)

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

func newSyntheticEntity(p *Parser, className string, id, serial int, names []string, values []any) *Entity {
	if len(names) != len(values) {
		_panicf("synthetic entity: %d property names but %d values", len(names), len(values))
	}

	info := &syntheticInfo{parser: p}
	classID := int32(-1)

	if p != nil {
		info.class = p.classesByName[className]
		if info.class == nil {
			_panicf("synthetic entity: unknown server-class %q", className)
		}

		classID = info.class.classId
	}

	// Each entity gets its own class: all props must be declared before the first lookup,
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

	syntheticSet(e, names, values, false, false)
	syntheticInfos.Store(e, info)

	return e
}

func (c *class) addSyntheticField(name string) {
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

func syntheticInfoOf(e *Entity) *syntheticInfo {
	info, ok := syntheticInfos.Load(e)
	if !ok {
		panic(fmt.Sprintf("entity %d is not a synthetic entity", e.index))
	}

	return info.(*syntheticInfo)
}

func (info *syntheticInfo) fireEntityHandlers(e *Entity, op st.EntityOp) {
	if info.parser == nil {
		return
	}

	for _, h := range info.parser.entityHandlers {
		if err := h(e, op); err != nil {
			panic(fmt.Sprintf("entity handler returned an error for entity %d (%v): %v", e.index, op, err))
		}
	}
}

// syntheticCreate mirrors Parser.OnPacketEntities for a newly created entity.
func syntheticCreate(e *Entity) {
	info := syntheticInfoOf(e)

	createdHandlers := e.class.createdHandlers
	if info.class != nil {
		createdHandlers = info.class.createdHandlers
	}

	for _, h := range createdHandlers {
		h(e)
	}

	for _, f := range e.onCreateFinished {
		f()
	}

	info.fireEntityHandlers(e, st.EntityOpCreated|st.EntityOpEntered)

	e.fireInitialUpdateHandlers()
	e.fireUpdateFinished()
}

// syntheticSet sets the given values in field-path order, like Entity.readFields does.
// With finish, it mirrors Parser.OnPacketEntities for an updated entity.
func syntheticSet(e *Entity, names []string, values []any, fire, finish bool) {
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

	op := st.EntityOpUpdated

	if finish && !e.active {
		e.active = true
		op |= st.EntityOpEntered
	}

	fp := newFieldPath()
	defer fp.release()

	for _, u := range updates {
		fp.path[0] = u.index
		e.state.set(fp, u.value)

		if fire && e.hasHandlers {
			e.dispatchUpdate(fp, u.value)
		}
	}

	if !finish {
		return
	}

	e.fireUpdateFinished()
	syntheticInfoOf(e).fireEntityHandlers(e, op)
}

// syntheticDestroy mirrors Parser.OnPacketEntities for a deleted entity.
func syntheticDestroy(e *Entity) {
	if !e.active {
		return // entity has already been destroyed
	}

	e.Destroy()
	syntheticInfoOf(e).fireEntityHandlers(e, st.EntityOpLeft|st.EntityOpDeleted)
}
