package st

import (
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"sync"
)

type Entity struct {
	Id          int
	ServerClass *ServerClass
	props       []PropertyEntry
}

func (e *Entity) Props() []PropertyEntry {
	return e.props
}

func (e *Entity) FindProperty(name string) *PropertyEntry {
	var prop *PropertyEntry
	for i, _ := range e.props {
		if e.props[i].entry.name == name {
			if prop != nil {
				panic("More than one property with name " + name + " found")
			}
			prop = &e.props[i]
		}
	}
	if prop == nil {
		panic("Could not find property with name " + name)
	}
	return prop
}

type entrySliceBacker struct {
	slice []*PropertyEntry
}

var entrySliceBackerPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return &entrySliceBacker{make([]*PropertyEntry, 0, 8)}
	},
}

func (e *Entity) ApplyUpdate(reader bs.BitReader) {
	idx := -1

	backer := entrySliceBackerPool.Get().(*entrySliceBacker)

	newWay := reader.ReadBit()

	for idx = e.readFileIndex(reader, idx, newWay); idx != -1; idx = e.readFileIndex(reader, idx, newWay) {
		backer.slice = append(backer.slice, &e.props[idx])
	}

	for _, prop := range backer.slice {
		prop.FirePropertyUpdateEvent(propDecoder.decodeProp(prop.entry, reader), e)
	}

	// Reset to 0 length before pooling
	backer.slice = backer.slice[:0]
	// Defer has quite the overhead so we just fill the pool here
	entrySliceBackerPool.Put(backer)
}

func (e *Entity) readFileIndex(reader bs.BitReader, lastIndex int, newWay bool) int {
	if newWay {
		// New way
		if reader.ReadBit() {
			return lastIndex + 1
		}
	}
	res := 0
	if newWay && reader.ReadBit() {
		res = int(reader.ReadInt(3))
	} else {
		res = int(reader.ReadInt(7))
		switch res & (32 | 64) {
		case 32:
			// Cast might be too late, should maybe be before bitshift
			res = (res & ^96) | (int(reader.ReadInt(2) << 5))
		case 64:
			res = (res & ^96) | (int(reader.ReadInt(4) << 5))
		case 96:
			res = (res & ^96) | (int(reader.ReadInt(7) << 5))
		}
	}

	// end marker
	if res == 0xfff {
		return -1
	}

	return lastIndex + 1 + res
}

func (e *Entity) CollectProperties(ppBase *[]*RecordedPropertyUpdate) {
	adder := func(event PropertyUpdateEvent) {
		*ppBase = append(*ppBase, event.Record())
	}

	for i, _ := range e.props {
		e.props[i].RegisterPropertyUpdateHandler(adder)
	}
}

func NewEntity(id int, serverClass *ServerClass) *Entity {
	props := make([]PropertyEntry, 0, len(serverClass.FlattenedProps))
	for i, _ := range serverClass.FlattenedProps {
		props = append(props, NewPropertyEntry(&serverClass.FlattenedProps[i], i))
	}
	return &Entity{Id: id, ServerClass: serverClass, props: props}
}

type PropertyEntry struct {
	index         int
	entry         *FlattenedPropEntry
	eventHandlers []PropertyUpdateHandler
}

func (pe *PropertyEntry) Entry() *FlattenedPropEntry {
	return pe.entry
}

func (pe *PropertyEntry) FirePropertyUpdateEvent(value PropValue, entity *Entity) {
	for _, h := range pe.eventHandlers {
		if h != nil {
			h(PropertyUpdateEvent{value: value, entity: entity, property: pe})
		}
	}
}

func (pe *PropertyEntry) RegisterPropertyUpdateHandler(handler PropertyUpdateHandler) {
	// TODO: Use godispatch internally? Might be slower, needs testing
	pe.eventHandlers = append(pe.eventHandlers, handler)
}

type PropertyUpdateEvent struct {
	value    PropValue
	entity   *Entity
	property *PropertyEntry
}

func (e *PropertyUpdateEvent) Value() PropValue {
	return e.value
}

func (e *PropertyUpdateEvent) Entity() *Entity {
	return e.entity
}

func (e *PropertyUpdateEvent) Property() *PropertyEntry {
	return e.property
}

func (e *PropertyUpdateEvent) Record() *RecordedPropertyUpdate {
	return &RecordedPropertyUpdate{propIndex: e.property.index, value: e.value}
}

type RecordedPropertyUpdate struct {
	propIndex int
	value     PropValue
}

func (r *RecordedPropertyUpdate) PropIndex() int {
	return r.propIndex
}

func (r *RecordedPropertyUpdate) Value() PropValue {
	return r.value
}

type PropertyUpdateHandler func(PropertyUpdateEvent)

func NewPropertyEntry(entry *FlattenedPropEntry, index int) PropertyEntry {
	return PropertyEntry{index: index, entry: entry}
}
