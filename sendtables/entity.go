package sendtables

import (
	"fmt"
	"sync"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
)

// Entity stores a entity in the game (e.g. players etc.) with its properties.
type Entity struct {
	ID          int
	ServerClass *ServerClass
	props       []PropertyEntry
}

// Props returns all properties (PropertyEntry) for the Entity.
func (e *Entity) Props() []PropertyEntry {
	return e.props
}

// FindProperty finds a property on the Entity by name.
func (e *Entity) FindProperty(name string) *PropertyEntry {
	var prop *PropertyEntry
	for i := range e.props {
		if e.props[i].entry.name == name {
			if prop != nil {
				panic(fmt.Sprintf("More than one property with name %q found", name))
			}
			prop = &e.props[i]
		}
	}
	if prop == nil {
		panic(fmt.Sprintf("Could not find property with name %q", name))
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

// ApplyUpdate reads an update to an Enitiy's properties and
// triggers registered PropertyUpdateHandlers if values changed.
func (e *Entity) ApplyUpdate(reader *bit.BitReader) {
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

func readFieldIndex(reader *bit.BitReader, lastIndex int, newWay bool) int {
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

// CollectProperties registers PropertyUpdateHandlers on the Entitiy
// that will update a 'preprocessedBasleine' map with the most recent values.
// This is used if entities leave and re-enter the PVS as entities will be re-created at that point
// and we can then take the map as baseline for the new entity.
func (e *Entity) CollectProperties(ppBase *map[int]PropValue) {
	for i := range e.props {
		i2 := i // Copy for the adder
		adder := func(val PropValue) {
			(*ppBase)[e.props[i2].index] = val
		}

		e.props[i].RegisterPropertyUpdateHandler(adder)
	}
}

// NewEntity creates a new Entity with a given id and ServerClass and returns it.
func NewEntity(id int, serverClass *ServerClass) *Entity {
	props := make([]PropertyEntry, 0, len(serverClass.FlattenedProps))
	for i := range serverClass.FlattenedProps {
		props = append(props, NewPropertyEntry(&serverClass.FlattenedProps[i], i))
	}
	return &Entity{ID: id, ServerClass: serverClass, props: props}
}

// PropertyEntry wraps a FlattenedPropEntry and allows registering handlers
// that can be triggered on a update of the property.
type PropertyEntry struct {
	index          int
	entry          *FlattenedPropEntry
	updateHandlers []PropertyUpdateHandler
}

// Entry returns the underlying FlattenedPropEntry.
func (pe *PropertyEntry) Entry() *FlattenedPropEntry {
	return pe.entry
}

// FirePropertyUpdate triggers all registered PropertyUpdateHandler
// on the PropertyEntry with the given PropValue.
func (pe *PropertyEntry) FirePropertyUpdate(value PropValue) {
	for _, h := range pe.updateHandlers {
		if h != nil {
			h(value)
		}
	}
}

// RegisterPropertyUpdateHandler registers a PropertyUpdateHandler.
// The handler will be triggered on every FirePropertyUpdate call.
func (pe *PropertyEntry) RegisterPropertyUpdateHandler(handler PropertyUpdateHandler) {
	pe.updateHandlers = append(pe.updateHandlers, handler)
}

// PropertyUpdateHandler is the interface for handlers that are interested in PropertyEntry changes.
type PropertyUpdateHandler func(PropValue)

// NewPropertyEntry creates a new PropertyEntry from a FlattenedPropEntry and index
func NewPropertyEntry(entry *FlattenedPropEntry, index int) PropertyEntry {
	return PropertyEntry{index: index, entry: entry}
}
