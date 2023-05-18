package sendtables2

import (
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2"
)

// EntityOp is a bitmask representing the type of operation performed on an Entity
type EntityOp int

const (
	EntityOpNone           EntityOp = 0x00
	EntityOpCreated        EntityOp = 0x01
	EntityOpUpdated        EntityOp = 0x02
	EntityOpDeleted        EntityOp = 0x04
	EntityOpEntered        EntityOp = 0x08
	EntityOpLeft           EntityOp = 0x10
	EntityOpCreatedEntered EntityOp = EntityOpCreated | EntityOpEntered
	EntityOpUpdatedEntered EntityOp = EntityOpUpdated | EntityOpEntered
	EntityOpDeletedLeft    EntityOp = EntityOpDeleted | EntityOpLeft
)

var entityOpNames = map[EntityOp]string{
	EntityOpNone:           "None",
	EntityOpCreated:        "Created",
	EntityOpUpdated:        "Updated",
	EntityOpDeleted:        "Deleted",
	EntityOpEntered:        "Entered",
	EntityOpLeft:           "Left",
	EntityOpCreatedEntered: "Created+Entered",
	EntityOpUpdatedEntered: "Updated+Entered",
	EntityOpDeletedLeft:    "Deleted+Left",
}

// Flag determines whether an EntityOp includes another. This is primarily
// offered to prevent bit flag errors in downstream clients.
func (o EntityOp) Flag(p EntityOp) bool {
	return o&p != 0
}

// String returns a human identifiable string for the EntityOp
func (o EntityOp) String() string {
	return entityOpNames[o]
}

// EntityHandler is a function that receives Entity updates
type EntityHandler func(*Entity, EntityOp) error

// Entity represents a single game entity in the replay
type Entity struct {
	index   int32
	serial  int32
	class   *class
	active  bool
	state   *fieldState
	fpCache map[string]*fieldPath
	fpNoop  map[string]bool
}

// newEntity returns a new entity for the given index, serial and class
func newEntity(index, serial int32, class *class) *Entity {
	return &Entity{
		index:   index,
		serial:  serial,
		class:   class,
		active:  true,
		state:   newFieldState(),
		fpCache: make(map[string]*fieldPath),
		fpNoop:  make(map[string]bool),
	}
}

// String returns a human identifiable string for the Entity
func (e *Entity) String() string {
	return fmt.Sprintf("%d <%s>", e.index, e.class.name)
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

// Internal Callback for OnCSVCMsg_PacketEntities.
func (p *Parser) OnPacketEntities(m *msgs2.CSVCMsg_PacketEntities) error {
	r := newReader(m.GetEntityData())

	var (
		index   = int32(-1)
		updates = int(m.GetUpdatedEntries())
		cmd     uint32
		classId int32
		serial  int32
		e       *Entity
	)

	if !m.GetIsDelta() {
		if p.entityFullPackets > 0 {
			return nil
		}
		p.entityFullPackets++
	}

	type tuple struct {
		e  *Entity
		op EntityOp
	}
	tuples := make([]tuple, 0, updates)

	for ; updates > 0; updates-- {
		next := index + int32(r.readUBitVar()) + 1
		index = next

		var op EntityOp

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
				readFields(newReader(baseline), class.serializer, e.state)
				readFields(r, class.serializer, e.state)

				op = EntityOpCreated | EntityOpEntered
			} else {
				if e = p.entities[index]; e == nil {
					_panicf("unable to find existing entity %d", index)
				}

				op = EntityOpUpdated
				if !e.active {
					e.active = true
					op |= EntityOpEntered
				}

				readFields(r, e.class.serializer, e.state)
			}
		} else {
			if e = p.entities[index]; e == nil {
				_panicf("unable to find existing entity %d", index)
			}

			if !e.active {
				_panicf("entity %d (%s) ordered to leave, already inactive", e.class.classId, e.class.name)
			}

			op = EntityOpLeft
			if cmd&0x02 != 0 {
				op |= EntityOpDeleted
				p.entities[index] = nil
			}
		}

		tuples = append(tuples, tuple{e, op})
	}

	if r.remBytes() > 1 || r.bitCount > 7 {
		// FIXME: maybe we should panic("didn't consume all data")
	}

	for _, h := range p.entityHandlers {
		for _, t := range tuples {
			if err := h(t.e, t.op); err != nil {
				return err
			}
		}
	}

	return nil
}

// OnEntity registers an EntityHandler that will be called when an entity
// is created, updated, deleted, etc.
func (p *Parser) OnEntity(h EntityHandler) {
	p.entityHandlers = append(p.entityHandlers, h)
}
