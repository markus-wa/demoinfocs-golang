package st

import (
	bs "github.com/markus-wa/demoinfocs-golang/bitread"
	"sync"
)

type Entity struct {
	ID          int
	ServerClass *ServerClass
	props       []PropertyEntry
}

func (e *Entity) Props() []PropertyEntry {
	return e.props
}

func (e *Entity) FindProperty(name string) *PropertyEntry {
	var prop *PropertyEntry
	for i := range e.props {
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

// Wrapping the slice in a struct causes far fewer object allocations for some reason
type entrySliceBacker struct {
	slice []*PropertyEntry
}

var entrySliceBackerPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return &entrySliceBacker{make([]*PropertyEntry, 0, 8)}
	},
}

func (e *Entity) ApplyUpdate(reader *bs.BitReader) {
	idx := -1
	newWay := reader.ReadBit()
	backer := entrySliceBackerPool.Get().(*entrySliceBacker)

	// TODO: Use index slice instead?
	for idx = readFieldIndex(reader, idx, newWay); idx != -1; idx = readFieldIndex(reader, idx, newWay) {
		backer.slice = append(backer.slice, &e.props[idx])
	}

	for _, prop := range backer.slice {
		prop.FirePropertyUpdate(propDecoder.decodeProp(prop.entry, reader))
	}

	// Reset to 0 length before pooling
	backer.slice = backer.slice[:0]
	// Defer has quite the overhead so we just fill the pool here
	entrySliceBackerPool.Put(backer)
}

func readFieldIndex(reader *bs.BitReader, lastIndex int, newWay bool) int {
	if newWay && reader.ReadBit() {
		// NewWay A
		return lastIndex + 1
	}
	var res uint
	if newWay && reader.ReadBit() {
		// NewWay B
		res = reader.ReadInt(3)
	} else {
		res = reader.ReadInt(7)
		switch res & (32 | 64) {
		case 32:
			res = (res & ^uint(96)) | (reader.ReadInt(2) << 5)
		case 64:
			res = (res & ^uint(96)) | (reader.ReadInt(4) << 5)
		case 96:
			res = (res & ^uint(96)) | (reader.ReadInt(7) << 5)
		}
	}

	// end marker
	if res == 0xfff {
		return -1
	}

	return lastIndex + 1 + int(res)
}

func (e *Entity) CollectProperties(ppBase *map[int]PropValue) {
	for i := range e.props {
		adder := func(val PropValue) {
			(*ppBase)[e.props[i].index] = val
		}

		e.props[i].RegisterPropertyUpdateHandler(adder)
	}
}

func NewEntity(id int, serverClass *ServerClass) *Entity {
	props := make([]PropertyEntry, 0, len(serverClass.FlattenedProps))
	for i := range serverClass.FlattenedProps {
		props = append(props, NewPropertyEntry(&serverClass.FlattenedProps[i], i))
	}
	return &Entity{ID: id, ServerClass: serverClass, props: props}
}

type PropertyEntry struct {
	index          int
	entry          *FlattenedPropEntry
	updateHandlers []PropertyUpdateHandler
}

func (pe *PropertyEntry) Entry() *FlattenedPropEntry {
	return pe.entry
}

func (pe *PropertyEntry) FirePropertyUpdate(value PropValue) {
	for _, h := range pe.updateHandlers {
		if h != nil {
			h(value)
		}
	}
}

func (pe *PropertyEntry) RegisterPropertyUpdateHandler(handler PropertyUpdateHandler) {
	pe.updateHandlers = append(pe.updateHandlers, handler)
}

type PropertyUpdateHandler func(PropValue)

func NewPropertyEntry(entry *FlattenedPropEntry, index int) PropertyEntry {
	return PropertyEntry{index: index, entry: entry}
}
