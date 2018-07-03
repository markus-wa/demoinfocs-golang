package sendtables

import (
	"fmt"
	"sync"

	"github.com/golang/geo/r3"

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
// Returns nil if the property wasn't found.
// Panics if more than one property with the same name was found.
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

// InitializeBaseline applies an update and collects a baseline (default values) from the update.
func (e *Entity) InitializeBaseline(r *bit.BitReader) map[int]PropValue {
	baseline := make(map[int]PropValue)
	for i := range e.props {
		i2 := i // Copy for the adder
		adder := func(val PropValue) {
			baseline[i2] = val
		}

		e.props[i].RegisterPropertyUpdateHandler(adder)
	}

	e.ApplyUpdate(r)

	for i := range e.props {
		e.props[i].updateHandlers = nil
	}

	return baseline
}

const maxCoordInt = 16384

// Position returns the entity's position in world coordinates.
func (e *Entity) Position() r3.Vector {
	cellWidth := 1 << uint(e.FindProperty("m_cellbits").value.IntVal)
	cellX := e.FindProperty("m_cellX").value.IntVal
	cellY := e.FindProperty("m_cellY").value.IntVal
	cellZ := e.FindProperty("m_cellZ").value.IntVal
	offset := e.FindProperty("m_vecOrigin").value.VectorVal

	return r3.Vector{
		X: coordFromCell(cellX, cellWidth, offset.X),
		Y: coordFromCell(cellY, cellWidth, offset.Y),
		Z: coordFromCell(cellZ, cellWidth, offset.Z),
	}
}

// Returns a coordinate from a cell + offset
func coordFromCell(cell, cellWidth int, offset float64) float64 {
	return float64(cell*cellWidth-maxCoordInt) + offset
}

// NewEntity creates a new Entity with a given id and ServerClass and returns it.
func NewEntity(id int, serverClass *ServerClass) *Entity {
	props := make([]PropertyEntry, len(serverClass.FlattenedProps))
	for i := range serverClass.FlattenedProps {
		props[i] = PropertyEntry{entry: &serverClass.FlattenedProps[i]}
	}
	return &Entity{ID: id, ServerClass: serverClass, props: props}
}

// PropertyEntry wraps a FlattenedPropEntry and allows registering handlers
// that can be triggered on a update of the property.
type PropertyEntry struct {
	entry          *FlattenedPropEntry
	updateHandlers []PropertyUpdateHandler
	value          PropValue
}

// Entry returns the underlying FlattenedPropEntry.
func (pe *PropertyEntry) Entry() *FlattenedPropEntry {
	return pe.entry
}

// Value returns current value of the property.
func (pe *PropertyEntry) Value() PropValue {
	return pe.value
}

// FirePropertyUpdate triggers all registered PropertyUpdateHandler
// on the PropertyEntry with the given PropValue.
func (pe *PropertyEntry) FirePropertyUpdate(value PropValue) {
	pe.value = value
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
