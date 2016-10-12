package st

import (
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"reflect"
)

type Entity struct {
	Id          int
	ServerClass *ServerClass
	props       []*PropertyEntry
}

func (e *Entity) Props() []*PropertyEntry {
	return e.props
}

func (e *Entity) FindProperty(name string) *PropertyEntry {
	var prop *PropertyEntry
	for _, p := range e.props {
		if p.entry.name == name {
			if prop != nil {
				panic("More than one property with name " + name + " found")
			}
			prop = p
		}
	}
	if prop == nil {
		panic("Could not find property with name " + name)
	}
	return prop
}

func (e *Entity) ApplyUpdate(reader bs.BitReader) {
	idx := -1
	entries := make([]*PropertyEntry, 0, 8)

	newWay := reader.ReadBit()

	for idx = e.readFileIndex(reader, idx, newWay); idx != -1; idx = e.readFileIndex(reader, idx, newWay) {
		entries = append(entries, e.props[idx])
	}

	for _, prop := range entries {
		prop.FirePropertyUpdateEvent(propDecoder.decodeProp(prop.entry, reader), e)
	}
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

func NewEntity(id int, serverClass *ServerClass) *Entity {
	props := make([]*PropertyEntry, len(serverClass.FlattenedProps))
	for i, p := range serverClass.FlattenedProps {
		props[i] = NewPropertyEntry(p, i)
	}
	return &Entity{Id: id, ServerClass: serverClass, props: props}
}

type PropertyEntry struct {
	index         int
	entry         *FlattenedPropEntry
	eventHandlers map[reflect.Type]PropertyUpdateHandler
}

func (pe *PropertyEntry) Entry() *FlattenedPropEntry {
	return pe.entry
}

func (pe *PropertyEntry) FirePropertyUpdateEvent(value interface{}, entity *Entity) {
	h := pe.eventHandlers[reflect.TypeOf(value)]
	if h != nil {
		h(&PropertyUpdateEvent{value: value, entity: entity, property: pe})
	}
}

func (pe *PropertyEntry) RegisterPropertyUpdateHandler(valueType reflect.Type, handler PropertyUpdateHandler) {
	if pe.eventHandlers == nil {
		pe.eventHandlers = make(map[reflect.Type]PropertyUpdateHandler)
	}
	pe.eventHandlers[valueType] = handler
}

type PropertyUpdateEvent struct {
	value    interface{}
	entity   *Entity
	property *PropertyEntry
}

func (e *PropertyUpdateEvent) Value() interface{} {
	return e.value
}

func (e *PropertyUpdateEvent) Entity() *Entity {
	return e.entity
}

func (e *PropertyUpdateEvent) Property() *PropertyEntry {
	return e.property
}

type PropertyUpdateHandler func(*PropertyUpdateEvent)

func NewPropertyEntry(entry *FlattenedPropEntry, index int) *PropertyEntry {
	return &PropertyEntry{index: index, entry: entry}
}
