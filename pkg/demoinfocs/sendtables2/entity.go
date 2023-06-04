package sendtables2

import (
	"fmt"
	"strings"

	"github.com/golang/geo/r3"

	bit "github.com/markus-wa/demoinfocs-golang/v3/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/sendtables"
)

// Entity represents a single game entity in the replay
type Entity struct {
	index   int32
	serial  int32
	class   *class
	active  bool
	state   *fieldState
	fpCache map[string]*fieldPath
	fpNoop  map[string]bool

	onCreateFinished []func()
	onDestroy        []func()
	updateHandlers   map[string][]st.PropertyUpdateHandler
}

func (e *Entity) ServerClass() st.ServerClass {
	return e.class
}

func (e *Entity) ID() int {
	return int(e.index)
}

func (e *Entity) SerialNum() int {
	return int(e.serial)
}

func (e *Entity) Properties() (out []st.Property) {
	for _, fp := range e.class.getFieldPaths(newFieldPath(), e.state) {
		out = append(out, e.Property(e.class.getNameForFieldPath(fp)))
	}

	return
}

type property struct {
	entity *Entity
	name   string
}

func (p property) Name() string {
	return p.name
}

func (p property) Value() st.PropertyValue {
	return st.PropertyValue{
		VectorVal: r3.Vector{},
		IntVal:    0,
		Int64Val:  0,
		ArrayVal:  nil,
		StringVal: "",
		FloatVal:  0,
		Any:       p.entity.Get(p.name),
		S2:        true,
	}
}

func (p property) Type() st.PropertyType {
	panic("not implemented")
}

func (p property) ArrayElementType() st.PropertyType {
	panic("not implemented")
}

func (p property) OnUpdate(handler st.PropertyUpdateHandler) {
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], handler)
}

type bindFactory func(variable any) st.PropertyUpdateHandler

var bindFactoryByType = map[st.PropertyValueType]bindFactory{
	st.ValTypeVector: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*r3.Vector) = v.VectorVal // FIXME this won't work
		}
	},
	st.ValTypeInt: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*int) = v.IntVal
		}
	},
	st.ValTypeArray: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*[]st.PropertyValue) = v.ArrayVal
		}
	},
	st.ValTypeString: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*string) = v.StringVal
		}
	},
	st.ValTypeBoolInt: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*bool) = v.IntVal != 0
		}
	},
	st.ValTypeFloat32: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*float32) = v.FloatVal
		}
	},
	st.ValTypeFloat64: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*float64) = float64(v.FloatVal)
		}
	},
}

func (p property) Bind(variable any, t st.PropertyValueType) {
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], bindFactoryByType[t](variable))
}

func (e *Entity) Property(name string) st.Property {
	ok := e.class.serializer.getFieldPathForName(newFieldPath(), name)
	if !ok {
		return nil
	}

	return property{
		entity: e,
		name:   name,
	}
}

func (e *Entity) BindProperty(prop string, variable any, t st.PropertyValueType) {
	e.Property(prop).Bind(variable, t)
}

func (e *Entity) PropertyValue(name string) (st.PropertyValue, bool) {
	prop := e.Property(name)
	if prop == nil {
		return st.PropertyValue{S2: true}, false
	}

	v := prop.Value()

	return v, true
}

func (e *Entity) PropertyValueMust(name string) st.PropertyValue {
	val, ok := e.PropertyValue(name)
	if !ok {
		panic(fmt.Sprintf("property '%s' not found", name))
	}

	return val
}

func (e *Entity) ApplyUpdate(reader *bit.BitReader) {
	panic("not implemented")
}

const (
	serverClassPlayer = "CCSPlayerPawn"

	maxCoordInt = 16384

	propCellBits          = "m_cellbits"
	propCellX             = "CBodyComponent.m_cellX"
	propCellY             = "CBodyComponent.m_cellY"
	propCellZ             = "CBodyComponent.m_cellZ"
	propVecOrigin         = "m_vecOrigin"
	propVecOriginPlayerXY = "cslocaldata.m_vecOrigin"
	propVecOriginPlayerZ  = "cslocaldata.m_vecOrigin[2]"
)

func (e *Entity) isPlayer() bool {
	return e.class.name == serverClassPlayer
}

// Returns a coordinate from a cell + offset
func coordFromCell(cell, cellWidth int, offset float64) float64 {
	return float64(cell*cellWidth-maxCoordInt) + offset
}

func (e *Entity) Position() r3.Vector {
	return r3.Vector{} // FIXME: implement

	if e.isPlayer() {
		// FIXME: POV demo support
		xyProp := e.Property(propVecOriginPlayerXY)
		zProp := e.Property(propVecOriginPlayerZ)

		xy := xyProp.Value().VectorVal
		z := float64(zProp.Value().FloatVal)

		return r3.Vector{
			X: xy.X,
			Y: xy.Y,
			Z: z,
		}
	}

	cellBitsProp := e.Property(propCellBits)
	cellXProp := e.Property(propCellX)
	cellYProp := e.Property(propCellY)
	cellZProp := e.Property(propCellZ)
	offsetProp := e.Property(propVecOrigin)

	cellWidth := 1 << uint(cellBitsProp.Value().IntVal)
	cellX := cellXProp.Value().IntVal
	cellY := cellYProp.Value().IntVal
	cellZ := cellZProp.Value().IntVal
	offset := offsetProp.Value().VectorVal

	return r3.Vector{
		X: coordFromCell(cellX, cellWidth, offset.X),
		Y: coordFromCell(cellY, cellWidth, offset.Y),
		Z: coordFromCell(cellZ, cellWidth, offset.Z),
	}
}

func (e *Entity) OnPositionUpdate(h func(pos r3.Vector)) {
	return // FIXME: implement

	pos := new(r3.Vector)
	firePosUpdate := func(st.PropertyValue) {
		newPos := e.Position()
		if *pos != newPos {
			h(newPos)
			*pos = newPos
		}
	}

	if e.isPlayer() {
		e.Property(propVecOriginPlayerXY).OnUpdate(firePosUpdate) // FIXME: POV demos use different property names
		e.Property(propVecOriginPlayerZ).OnUpdate(firePosUpdate)
	} else {
		e.Property(propCellX).OnUpdate(firePosUpdate)
		e.Property(propCellY).OnUpdate(firePosUpdate)
		e.Property(propCellZ).OnUpdate(firePosUpdate)
		e.Property(propVecOrigin).OnUpdate(firePosUpdate)
	}
}

func (e *Entity) OnDestroy(delegate func()) {
	e.onDestroy = append(e.onDestroy, delegate)
}

func (e *Entity) Destroy() {
	for _, delegate := range e.onDestroy {
		delegate()
	}
}

func (e *Entity) OnCreateFinished(delegate func()) {
	e.onCreateFinished = append(e.onCreateFinished, delegate)
}

// newEntity returns a new entity for the given index, serial and class
func newEntity(index, serial int32, class *class) *Entity {
	return &Entity{
		index:            index,
		serial:           serial,
		class:            class,
		active:           true,
		state:            newFieldState(),
		fpCache:          make(map[string]*fieldPath),
		fpNoop:           make(map[string]bool),
		onCreateFinished: nil,
		onDestroy:        nil,
		updateHandlers:   make(map[string][]st.PropertyUpdateHandler),
	}
}

// String returns a human identifiable string for the Entity
func (e *Entity) String() string {
	paths := e.class.getFieldPaths(newFieldPath(), e.state)
	props := make([]string, len(paths))

	for _, fp := range paths {
		props = append(props, fmt.Sprintf("%s: %v", e.class.getNameForFieldPath(fp), e.state.get(fp)))
	}

	return fmt.Sprintf("%d <%s>\n %s", e.index, e.class.name, strings.Join(props, "\n "))
}

// Map returns a map of current entity state as key-value pairs
func (e *Entity) Map() map[string]interface{} {
	values := make(map[string]interface{})
	for _, fp := range e.class.getFieldPaths(newFieldPath(), e.state) {
		values[e.class.getNameForFieldPath(fp)] = e.state.get(fp)
	}
	return values
}

// Get returns the current value of the Entity state for the given key
func (e *Entity) Get(name string) interface{} {
	if fp, ok := e.fpCache[name]; ok {
		return e.state.get(fp)
	}
	if e.fpNoop[name] {
		return nil
	}

	fp := newFieldPath()
	if !e.class.getFieldPathForName(fp, name) {
		e.fpNoop[name] = true
		fp.release()
		return nil
	}
	e.fpCache[name] = fp

	return e.state.get(fp)
}

// Exists returns true if the given key exists in the Entity state
func (e *Entity) Exists(name string) bool {
	return e.Get(name) != nil
}

// GetInt32 gets given key as an int32
func (e *Entity) GetInt32(name string) (int32, bool) {
	x, ok := e.Get(name).(int32)
	return x, ok
}

// GetUint32 gets given key as a uint32
func (e *Entity) GetUint32(name string) (uint32, bool) {
	if v := e.Get(name); v != nil {
		switch x := v.(type) {
		case uint32:
			return x, true
		case uint64:
			return uint32(x), true
		}
	}
	return 0, false
}

// GetUint64 gets given key as a uint64
func (e *Entity) GetUint64(name string) (uint64, bool) {
	x, ok := e.Get(name).(uint64)
	return x, ok
}

// GetFloat32 gets given key as an float32
func (e *Entity) GetFloat32(name string) (float32, bool) {
	x, ok := e.Get(name).(float32)
	return x, ok
}

// GetString gets given key as a string
func (e *Entity) GetString(name string) (string, bool) {
	x, ok := e.Get(name).(string)
	return x, ok
}

// GetBool gets given key as a bool
func (e *Entity) GetBool(name string) (bool, bool) {
	x, ok := e.Get(name).(bool)
	return x, ok
}

// GetSerial return the serial of the class associated with this Entity
func (e *Entity) GetSerial() int32 {
	return e.serial
}

// GetClassId returns the id of the class associated with this Entity
func (e *Entity) GetClassId() int32 {
	return e.class.classId
}

// GetClassName returns the name of the class associated with this Entity
func (e *Entity) GetClassName() string {
	return e.class.name
}

// GetIndex returns the index of this Entity
func (e *Entity) GetIndex() int32 {
	return e.index
}

// FindEntity finds a given Entity by index
func (p *Parser) FindEntity(index int32) *Entity {
	return p.entities[index]
}

const (
	// SOURCE2
	indexBits  uint64 = 14
	handleMask uint64 = (1 << indexBits) - 1
)

func handle2idx(handle uint64) int32 {
	return int32(handle & handleMask)
}

func serialForHandle(handle uint64) int32 {
	return int32(handle >> indexBits)
}

// FindEntityByHandle finds a given Entity by handle
func (p *Parser) FindEntityByHandle(handle uint64) *Entity {
	idx := handle2idx(handle)
	e := p.FindEntity(idx)
	if e != nil && e.GetSerial() != serialForHandle(handle) {
		return nil
	}
	return e
}

// FilterEntity finds entities by callback
func (p *Parser) FilterEntity(fb func(*Entity) bool) []*Entity {
	entities := make([]*Entity, 0, 0)
	for _, et := range p.entities {
		if fb(et) {
			entities = append(entities, et)
		}
	}
	return entities
}

func (e *Entity) readFields(r *reader) {
	fps := readFieldPaths(r)

	for _, fp := range fps {
		decoder := e.class.serializer.getDecoderForFieldPath(fp, 0)

		val := decoder(r)
		e.state.set(fp, val)

		upd := e.updateHandlers[e.class.getNameForFieldPath(fp)]
		if upd != nil {
			for _, h := range upd {
				h(st.PropertyValue{
					VectorVal: r3.Vector{},
					IntVal:    0,
					Int64Val:  0,
					ArrayVal:  nil,
					StringVal: "",
					FloatVal:  0,
					Any:       val,
					S2:        true,
				})
			}
		}

		fp.release()
	}
}

// Internal Callback for OnCSVCMsg_PacketEntities.
func (p *Parser) OnPacketEntities(m *msgs2.CSVCMsg_PacketEntities) error {
	r := newReader(m.GetEntityData())

	var (
		index   = int32(-1)
		updates = int(m.GetUpdatedEntries())
		cmd     uint32
		classId int32
		serial  int32
	)

	if !m.GetIsDelta() {
		if p.entityFullPackets > 0 {
			return nil
		}
		p.entityFullPackets++
	}

	type tuple struct {
		ent *Entity
		op  st.EntityOp
	}

	var tuples []tuple

	for ; updates > 0; updates-- {
		var (
			e  *Entity
			op st.EntityOp
		)

		next := index + int32(r.readUBitVar()) + 1
		index = next

		cmd = r.readBits(2)
		if cmd&0x01 == 0 {
			if cmd&0x02 != 0 {
				classId = int32(r.readBits(p.classIdSize))
				serial = int32(r.readBits(17))
				r.readVarUint32()

				class := p.classesById[classId]
				if class == nil {
					_panicf("unable to find new class %d", classId)
				}

				baseline := p.classBaselines[classId]
				if baseline == nil {
					_panicf("unable to find new baseline %d", classId)
				}

				e = newEntity(index, serial, class)
				p.entities[index] = e

				e.readFields(newReader(baseline))
				e.readFields(r)

				// Fire created-handlers so update-handlers can be registered
				for _, h := range class.createdHandlers {
					h(e)
				}

				// Fire all post-creation actions
				for _, f := range e.onCreateFinished {
					f()
				}

				op = st.EntityOpCreated | st.EntityOpEntered
			} else {
				if e = p.entities[index]; e == nil {
					_panicf("unable to find existing entity %d", index)
				}

				op = st.EntityOpUpdated
				if !e.active {
					e.active = true
					op |= st.EntityOpEntered
				}

				e.readFields(r)
			}
		} else {
			if e = p.entities[index]; e == nil {
				_panicf("unable to find existing entity %d", index)
			}

			if !e.active {
				_panicf("entity %d (%s) ordered to leave, already inactive", e.class.classId, e.class.name)
			}

			op = st.EntityOpLeft
			if cmd&0x02 != 0 {
				op |= st.EntityOpDeleted
				delete(p.entities, index)
			}
		}

		tuples = append(tuples, tuple{e, op})
	}

	for _, t := range tuples {
		e := t.ent

		for _, h := range p.entityHandlers {
			if err := h(e, t.op); err != nil {
				return err
			}
		}

		if t.op&st.EntityOpCreated != 0 {
			for prop, hs := range e.updateHandlers {
				v := e.PropertyValueMust(prop)

				for _, h := range hs {
					h(v)
				}
			}
		}
	}

	if r.remBytes() > 1 || r.bitCount > 7 {
		// FIXME: maybe we should panic("didn't consume all data")
	}

	return nil
}

// OnEntity registers an EntityHandler that will be called when an entity
// is created, updated, deleted, etc.
func (p *Parser) OnEntity(h st.EntityHandler) {
	p.entityHandlers = append(p.entityHandlers, h)
}
