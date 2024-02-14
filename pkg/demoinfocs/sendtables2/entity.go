package sendtables2

import (
	"fmt"
	"strings"

	"github.com/golang/geo/r3"

	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
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
	propCache        map[string]st.Property
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
			*variable.(*r3.Vector) = v.R3Vec()
		}
	},
	st.ValTypeInt: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*int) = v.Int()
		}
	},
	st.ValTypeArray: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*[]st.PropertyValue) = v.ArrayVal
		}
	},
	st.ValTypeString: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*string) = v.String()
		}
	},
	st.ValTypeBoolInt: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*bool) = v.BoolVal()
		}
	},
	st.ValTypeFloat32: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*float32) = v.Float()
		}
	},
	st.ValTypeFloat64: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*float64) = float64(v.Float())
		}
	},
}

func (p property) Bind(variable any, t st.PropertyValueType) {
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], bindFactoryByType[t](variable))
}

func (e *Entity) Property(name string) st.Property {
	if e.propCache[name] == nil {
		ok := e.class.serializer.checkFieldName(name)
		if !ok {
			return nil
		}

		e.propCache[name] = property{
			entity: e,
			name:   name,
		}
	}

	return e.propCache[name]
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

	propCellX = "CBodyComponent.m_cellX"
	propCellY = "CBodyComponent.m_cellY"
	propCellZ = "CBodyComponent.m_cellZ"
	propVecX  = "CBodyComponent.m_vecX"
	propVecY  = "CBodyComponent.m_vecY"
	propVecZ  = "CBodyComponent.m_vecZ"
)

func (e *Entity) isPlayer() bool {
	return e.class.name == serverClassPlayer
}

// Returns a coordinate from a cell + offset
func coordFromCell(cell uint64, offset float32) float64 {
	const (
		cellBits    = 9
		maxCoordInt = 16384
	)
	cellCoord := float64(cell)*float64(1<<cellBits) - maxCoordInt

	return cellCoord + float64(offset)
}

func (e *Entity) Position() r3.Vector {
	cellXProp := e.Property(propCellX)
	cellYProp := e.Property(propCellY)
	cellZProp := e.Property(propCellZ)
	offsetXProp := e.Property(propVecX)
	offsetYProp := e.Property(propVecY)
	offsetZProp := e.Property(propVecZ)

	cellX := cellXProp.Value().S2UInt64()
	cellY := cellYProp.Value().S2UInt64()
	cellZ := cellZProp.Value().S2UInt64()
	offsetX := offsetXProp.Value().Float()
	offsetY := offsetYProp.Value().Float()
	offsetZ := offsetZProp.Value().Float()

	return r3.Vector{
		X: coordFromCell(cellX, offsetX),
		Y: coordFromCell(cellY, offsetY),
		Z: coordFromCell(cellZ, offsetZ),
	}
}

func (e *Entity) OnPositionUpdate(h func(pos r3.Vector)) {
	pos := new(r3.Vector)
	firePosUpdate := func(st.PropertyValue) {
		newPos := e.Position()
		if *pos != newPos {
			h(newPos)
			*pos = newPos
		}
	}

	e.Property(propCellX).OnUpdate(firePosUpdate)
	e.Property(propCellY).OnUpdate(firePosUpdate)
	e.Property(propCellZ).OnUpdate(firePosUpdate)
	e.Property(propVecX).OnUpdate(firePosUpdate)
	e.Property(propVecY).OnUpdate(firePosUpdate)
	e.Property(propVecZ).OnUpdate(firePosUpdate)
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
		propCache:        map[string]st.Property{},
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

func handle2idx(handle uint64) int32 {
	return int32(handle & constants.EntityHandleIndexMaskSource2)
}

func serialForHandle(handle uint64) int32 {
	return int32(handle >> constants.MaxEdictBitsSource2)
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
	entities := make([]*Entity, 0)

	for _, et := range p.entities {
		if fb(et) {
			entities = append(entities, et)
		}
	}

	return entities
}

func (e *Entity) readFields(r *reader, paths *[]*fieldPath) {
	readFieldPaths(r, paths)

	for _, fp := range *paths {
		decoder := e.class.serializer.getDecoderForFieldPath(fp, 0)

		val := decoder(r)
		e.state.set(fp, val)

		for _, h := range e.updateHandlers[e.class.getNameForFieldPath(fp)] {
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

		fp.release()
	}
}

const (
	updateFlagDelete       = 0b10
	updateFlagVisibleInPVS = 0b10000
)

type updateType = uint32

const (
	updateTypeEnter updateType = iota
	updateTypeLeave
	updateTypeDelta
	updateTypePreserve
)

func readEntityUpdateType(r *reader, hasVisBits uint32) (updateType, uint32) {
	flags := r.readBits(2)
	if flags&0x01 != 0 {
		return updateTypeLeave, flags
	}
	if flags&0x02 != 0 {
		return updateTypeEnter, flags
	}
	if hasVisBits != 0 {
		flags = r.readBits(2) << 3
		if flags&0x08 != 0 {
			return updateTypePreserve, flags
		}
	}

	return updateTypeDelta, flags
}

// Internal Callback for OnCSVCMsg_PacketEntities.
func (p *Parser) OnPacketEntities(m *msgs2.CSVCMsg_PacketEntities) error {
	r := newReader(m.GetEntityData())

	var (
		index   = int32(-1)
		updates = int(m.GetUpdatedEntries())
		classId int32
		serial  int32
	)

	if !m.GetLegacyIsDelta() {
		if p.entityFullPackets > 0 {
			return nil
		}
		p.entityFullPackets++
	}

	type tuple struct {
		ent *Entity
		op  st.EntityOp
	}

	var (
		tuples []tuple
		paths  = make([]*fieldPath, 0)
	)

	hasVisBits := m.GetHasPvsVisBits()
	for ; updates > 0; updates-- {
		var (
			e  *Entity
			op st.EntityOp
		)

		next := index + int32(r.readUBitVar()) + 1
		index = next

		updateType, flags := readEntityUpdateType(r, hasVisBits)
		switch updateType {
		case updateTypeEnter:
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

			e.readFields(newReader(baseline), &paths)
			paths = paths[:0]

			e.readFields(r, &paths)
			paths = paths[:0]

			// Fire created-handlers so update-handlers can be registered
			for _, h := range class.createdHandlers {
				h(e)
			}

			// Fire all post-creation actions
			for _, f := range e.onCreateFinished {
				f()
			}

			op = st.EntityOpCreated | st.EntityOpEntered
		case updateTypeDelta:
			if e = p.entities[index]; e == nil {
				_panicf("unable to find existing entity %d", index)
			}

			op = st.EntityOpUpdated
			if !e.active {
				e.active = true
				op |= st.EntityOpEntered
			}

			e.readFields(r, &paths)
			paths = paths[:0]
			// todo: handle visibility
			// visibleInPVS := hasVisBits == 0 || flags&updateFlagVisibleInPVS != 0
			// fmt.Println("visible in pvs", visibleInPVS)
		case updateTypePreserve:
			// visibleInPVS := hasVisBits == 0 || flags&updateFlagVisibleInPVS != 0
		case updateTypeLeave:
			e = p.entities[index]
			if e == nil {
				continue
				_panicf("unable to find existing entity %d", index)
			}

			if !e.active {
				_panicf("entity %d (%s) ordered to leave, already inactive", e.class.classId, e.class.name)
			}

			op = st.EntityOpLeft
			if flags&updateFlagDelete != 0 {
				op |= st.EntityOpDeleted

				e.Destroy()

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
