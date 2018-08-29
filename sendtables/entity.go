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

	onCreateFinished []func()
	onDestroy        []func()
}

// Props returns all properties (PropertyEntry) for the Entity.
func (e *Entity) Props() []PropertyEntry {
	return e.props
}

// FindProperty finds a property on the Entity by name.
//
// Returns nil if the property wasn't found.
//
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

// BindProperty combines FindProperty() & PropertyEntry.Bind() into one.
// Essentially binds a property's value to a pointer.
// See the docs of the two individual functions for more info.
func (e *Entity) BindProperty(name string, variable interface{}, valueType propertyValueType) {
	e.FindProperty(name).Bind(variable, valueType)
}

var updatedPropIndicesPool = sync.Pool{
	New: func() interface{} {
		s := make([]int, 0, 8)
		return &s
	},
}

// ApplyUpdate reads an update to an Enitiy's properties and
// triggers registered PropertyUpdateHandlers if values changed.
//
// Intended for internal use only.
func (e *Entity) ApplyUpdate(reader *bit.BitReader) {
	idx := -1
	newWay := reader.ReadBit()
	updatedPropIndices := updatedPropIndicesPool.Get().(*[]int)

	for idx = readFieldIndex(reader, idx, newWay); idx != -1; idx = readFieldIndex(reader, idx, newWay) {
		*updatedPropIndices = append(*updatedPropIndices, idx)
	}

	for _, idx := range *updatedPropIndices {
		propDecoder.decodeProp(&e.props[idx], reader)
		e.props[idx].firePropertyUpdate()
	}

	// Reset length to 0 before pooling
	*updatedPropIndices = (*updatedPropIndices)[:0]
	// Defer has quite the overhead so we just fill the pool here
	updatedPropIndicesPool.Put(updatedPropIndices)
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
//
// Intended for internal use only.
func (e *Entity) InitializeBaseline(r *bit.BitReader) map[int]PropValue {
	baseline := make(map[int]PropValue)
	for i := range e.props {
		i2 := i // Copy for the adder
		adder := func(val PropValue) {
			baseline[i2] = val
		}

		e.props[i].OnUpdate(adder)
	}

	e.ApplyUpdate(r)

	for i := range e.props {
		e.props[i].updateHandlers = nil
	}

	return baseline
}

// ApplyBaseline baseline applies a previously collected baseline.
//
// Intended for internal use only.
func (e *Entity) ApplyBaseline(baseline map[int]PropValue) {
	for idx := range baseline {
		e.props[idx].value = baseline[idx]
	}
}

const (
	maxCoordInt = 16384

	propCellBits          = "m_cellbits"
	propCellX             = "m_cellX"
	propCellY             = "m_cellY"
	propCellZ             = "m_cellZ"
	propVecOrigin         = "m_vecOrigin"
	propVecOriginPlayerXY = "cslocaldata.m_vecOrigin"
	propVecOriginPlayerZ  = "cslocaldata.m_vecOrigin[2]"

	serverClassPlayer = "CCSPlayer"
)

// Position returns the entity's position in world coordinates.
func (e *Entity) Position() r3.Vector {
	// Player positions are calculated differently
	if e.isPlayer() {
		return e.positionPlayer()
	}

	return e.positionDefault()
}

func (e *Entity) isPlayer() bool {
	return e.ServerClass.Name == serverClassPlayer
}

func (e *Entity) positionDefault() r3.Vector {
	cellWidth := 1 << uint(e.FindProperty(propCellBits).value.IntVal)
	cellX := e.FindProperty(propCellX).value.IntVal
	cellY := e.FindProperty(propCellY).value.IntVal
	cellZ := e.FindProperty(propCellZ).value.IntVal
	offset := e.FindProperty(propVecOrigin).value.VectorVal

	return r3.Vector{
		X: coordFromCell(cellX, cellWidth, offset.X),
		Y: coordFromCell(cellY, cellWidth, offset.Y),
		Z: coordFromCell(cellZ, cellWidth, offset.Z),
	}
}

func (e *Entity) positionPlayer() r3.Vector {
	xy := e.FindProperty(propVecOriginPlayerXY).value.VectorVal
	z := float64(e.FindProperty(propVecOriginPlayerZ).value.FloatVal)
	return r3.Vector{
		X: xy.X,
		Y: xy.Y,
		Z: z,
	}
}

// Returns a coordinate from a cell + offset
func coordFromCell(cell, cellWidth int, offset float64) float64 {
	return float64(cell*cellWidth-maxCoordInt) + offset
}

// OnDestroy registers a function to be called on the entity's destruction.
func (e *Entity) OnDestroy(delegate func()) {
	e.onDestroy = append(e.onDestroy, delegate)
}

// Destroy triggers all via OnDestroy() registered functions.
//
// Intended for internal use only.
func (e *Entity) Destroy() {
	for _, f := range e.onDestroy {
		f()
	}
}

// OnCreateFinished registers a function to be called once the entity is fully created -
// i.e. once all property updates have been sent out.
func (e *Entity) OnCreateFinished(delegate func()) {
	e.onCreateFinished = append(e.onCreateFinished, delegate)
}

// NewEntity creates a new Entity with a given id and ServerClass and returns it.
func NewEntity(id int, serverClass *ServerClass) *Entity {
	propCount := len(serverClass.FlattenedProps)
	props := make([]PropertyEntry, propCount)
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

// Trigger all the registered PropertyUpdateHandlers on this entry.
func (pe *PropertyEntry) firePropertyUpdate() {
	for _, h := range pe.updateHandlers {
		if h != nil {
			h(pe.value)
		}
	}
}

// RegisterPropertyUpdateHandler registers a handler for updates of the PropertyEntry's value.
// Deprecated: Use OnUpdate instead.
func (pe *PropertyEntry) RegisterPropertyUpdateHandler(handler PropertyUpdateHandler) {
	pe.OnUpdate(handler)
}

// OnUpdate registers a handler for updates of the PropertyEntry's value.
func (pe *PropertyEntry) OnUpdate(handler PropertyUpdateHandler) {
	pe.updateHandlers = append(pe.updateHandlers, handler)
}

type propertyValueType int

// Possible types of property values.
//
// See PropertyEntry.Bind()
const (
	ValTypeInt propertyValueType = iota
	ValTypeFloat32
	ValTypeFloat64
	ValTypeString
	ValTypeVector
	ValTypeArray
	ValTypeBoolInt // Int that is treated as bool (1 -> true, != 1 -> false)
)

/*
Bind binds a property's value to a pointer.

Example:
	var i int
	PropertyEntry.Bind(&i, ValTypeInt)

This will bind the property's value to i so every time it's updated i is updated as well.

The valueType indicates which field of the PropValue to use for the binding.
*/
func (pe *PropertyEntry) Bind(variable interface{}, valueType propertyValueType) {
	var binder PropertyUpdateHandler
	switch valueType {
	case ValTypeInt:
		binder = func(val PropValue) {
			*(variable.(*int)) = val.IntVal
		}
	case ValTypeBoolInt:
		binder = func(val PropValue) {
			*(variable.(*bool)) = val.IntVal == 1
		}

	case ValTypeFloat32:
		binder = func(val PropValue) {
			*(variable.(*float32)) = val.FloatVal
		}

	case ValTypeFloat64:
		binder = func(val PropValue) {
			*(variable.(*float64)) = float64(val.FloatVal)
		}

	case ValTypeString:
		binder = func(val PropValue) {
			*(variable.(*string)) = val.StringVal
		}

	case ValTypeVector:
		binder = func(val PropValue) {
			*(variable.(*r3.Vector)) = val.VectorVal
		}

	case ValTypeArray:
		binder = func(val PropValue) {
			*(variable.(*[]PropValue)) = val.ArrayVal
		}
	}
	pe.OnUpdate(binder)
}

// PropertyUpdateHandler is the interface for handlers that are interested in PropertyEntry changes.
type PropertyUpdateHandler func(PropValue)
