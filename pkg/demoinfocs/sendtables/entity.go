package sendtables

import (
	"sync"

	"github.com/golang/geo/r3"

	bit "github.com/markus-wa/demoinfocs-golang/v2/internal/bitread"
)

//go:generate ifacemaker -f entity.go -s entity -i Entity -p sendtables -D -y "Entity is an auto-generated interface for entity, intended to be used when mockability is needed." -c "DO NOT EDIT: Auto generated" -o entity_interface.go
//go:generate ifacemaker -f entity.go -s property -i Property -p sendtables -D -y "Property is an auto-generated interface for property, intended to be used when mockability is needed." -c "DO NOT EDIT: Auto generated" -o property_interface.go

// entity stores a entity in the game (e.g. players etc.) with its properties.
type entity struct {
	serverClass *ServerClass
	id          int
	props       []property

	onCreateFinished []func()
	onDestroy        []func()
	position         func() r3.Vector
}

// ServerClass returns the entity's server-class.
func (e *entity) ServerClass() *ServerClass {
	return e.serverClass
}

// ID returns the entity's ID.
func (e *entity) ID() int {
	return e.id
}

// Properties returns all properties of the entity.
func (e *entity) Properties() (out []Property) {
	for i := range e.props {
		out = append(out, &e.props[i])
	}

	return out
}

func (e *entity) property(name string) *property {
	i, ok := e.serverClass.propNameToIndex[name]
	if !ok {
		return nil
	}

	return &e.props[i]
}

// Property finds a property on the entity by name.
//
// Returns nil if the property wasn't found.
func (e *entity) Property(name string) Property {
	prop := e.property(name)
	if prop == nil {
		// See https://stackoverflow.com/questions/13476349/check-for-nil-and-nil-interface-in-go
		return nil
	}

	return prop
}

// BindProperty combines Property() & Property.Bind() into one.
// Essentially binds a property's value to a pointer.
// See the docs of the two individual functions for more info.
func (e *entity) BindProperty(name string, variable interface{}, valueType PropertyValueType) {
	e.Property(name).Bind(variable, valueType)
}

// PropertyValue finds a property on the entity by name and returns its value.
//
// Returns false as second value if the property was not found.
func (e *entity) PropertyValue(name string) (PropertyValue, bool) {
	prop := e.property(name)
	if prop == nil {
		// See https://stackoverflow.com/questions/13476349/check-for-nil-and-nil-interface-in-go
		return PropertyValue{}, false
	}

	return prop.value, true
}

// PropertyValueMust finds a property on the entity by name and returns its value.
//
// Panics with nil pointer dereference error if the property was not found.
func (e *entity) PropertyValueMust(name string) PropertyValue {
	return e.property(name).value
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
func (e *entity) ApplyUpdate(reader *bit.BitReader) {
	idx := -1
	newWay := reader.ReadBit()
	updatedPropIndices := updatedPropIndicesPool.Get().(*[]int)

	for idx = readFieldIndex(reader, idx, newWay); idx != -1; idx = readFieldIndex(reader, idx, newWay) {
		*updatedPropIndices = append(*updatedPropIndices, idx)
	}

	for _, idx := range *updatedPropIndices {
		propDecoder.decodeProp(&e.props[idx], reader)
	}

	// Fire update only after all properties have been updated
	// That way data that is made up of multiple properties won't be wrong
	// For instance the entity's position
	for _, idx := range *updatedPropIndices {
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

	const fileIndexEndMarker = 0xfff
	if res == fileIndexEndMarker {
		return -1
	}

	return lastIndex + 1 + int(res)
}

// Collects an initial baseline for a server-class
func (e *entity) initializeBaseline(r *bit.BitReader) map[int]PropertyValue {
	baseline := make(map[int]PropertyValue)

	for i := range e.props {
		i2 := i // Copy for the adder
		adder := func(val PropertyValue) {
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

// Apply a previously via initializeBaseline collected baseline
func (e *entity) applyBaseline(baseline map[int]PropertyValue) {
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

// Sets up the entity.Position() function
// Necessary because Property() is fairly slow
// This way we only need to find the necessary properties once
func (e *entity) initialize() {
	// Player positions are calculated differently
	if e.isPlayer() {
		xyProp := e.Property(propVecOriginPlayerXY)
		zProp := e.Property(propVecOriginPlayerZ)

		e.position = func() r3.Vector {
			xy := xyProp.Value().VectorVal
			z := float64(zProp.Value().FloatVal)

			return r3.Vector{
				X: xy.X,
				Y: xy.Y,
				Z: z,
			}
		}
	} else {
		cellBitsProp := e.Property(propCellBits)
		cellXProp := e.Property(propCellX)
		cellYProp := e.Property(propCellY)
		cellZProp := e.Property(propCellZ)
		offsetProp := e.Property(propVecOrigin)

		e.position = func() r3.Vector {
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
	}
}

func (e *entity) isPlayer() bool {
	return e.serverClass.name == serverClassPlayer
}

// Position returns the entity's position in world coordinates.
func (e *entity) Position() r3.Vector {
	return e.position()
}

// OnPositionUpdate registers a handler for the entity's position update.
// The handler is called with the new position every time a position-relevant property is updated.
//
// See also Position()
func (e *entity) OnPositionUpdate(h func(pos r3.Vector)) {
	pos := new(r3.Vector)
	firePosUpdate := func(PropertyValue) {
		newPos := e.Position()
		if *pos != newPos {
			h(newPos)
			*pos = newPos
		}
	}

	if e.isPlayer() {
		e.Property(propVecOriginPlayerXY).OnUpdate(firePosUpdate)
		e.Property(propVecOriginPlayerZ).OnUpdate(firePosUpdate)
	} else {
		e.Property(propCellX).OnUpdate(firePosUpdate)
		e.Property(propCellY).OnUpdate(firePosUpdate)
		e.Property(propCellZ).OnUpdate(firePosUpdate)
		e.Property(propVecOrigin).OnUpdate(firePosUpdate)
	}
}

// Returns a coordinate from a cell + offset
func coordFromCell(cell, cellWidth int, offset float64) float64 {
	return float64(cell*cellWidth-maxCoordInt) + offset
}

// OnDestroy registers a function to be called on the entity's destruction.
func (e *entity) OnDestroy(delegate func()) {
	e.onDestroy = append(e.onDestroy, delegate)
}

// Destroy triggers all via OnDestroy() registered functions.
//
// Intended for internal use only.
func (e *entity) Destroy() {
	for _, f := range e.onDestroy {
		f()
	}
}

// OnCreateFinished registers a function to be called once the entity is fully created -
// i.e. once all property updates have been sent out.
func (e *entity) OnCreateFinished(delegate func()) {
	e.onCreateFinished = append(e.onCreateFinished, delegate)
}

// property wraps a flattenedPropEntry and allows registering handlers
// that can be triggered on a update of the property.
type property struct {
	entry          *flattenedPropEntry
	updateHandlers []PropertyUpdateHandler
	value          PropertyValue
}

// Name returns the property's name.
func (pe *property) Name() string {
	return pe.entry.name
}

// Value returns the current value of the property.
func (pe *property) Value() PropertyValue {
	return pe.value
}

// Type returns the data type of the property.
func (pe *property) Type() PropertyType {
	return PropertyType(pe.entry.prop.rawType)
}

// ArrayElementType returns the data type of array entries, if Property.Type() is PropTypeArray.
func (pe *property) ArrayElementType() PropertyType {
	return PropertyType(pe.entry.arrayElementProp.rawType)
}

// PropertyValueType specifies the type of a PropertyValue
type PropertyValueType int

// Possible types of property values.
// See Property.Bind()
const (
	ValTypeInt PropertyValueType = iota
	ValTypeFloat32
	ValTypeFloat64 // Like ValTypeFloat32 but with additional cast to float64
	ValTypeString
	ValTypeVector
	ValTypeArray
	ValTypeBoolInt // Int that is treated as bool (1 -> true, != 1 -> false)
)

// PropertyUpdateHandler is the interface for handlers that are interested in property changes.
type PropertyUpdateHandler func(PropertyValue)

// OnUpdate registers a handler for updates of the property's value.
//
// The handler will be called with the current value upon registration.
func (pe *property) OnUpdate(handler PropertyUpdateHandler) {
	handler(pe.value)
	pe.updateHandlers = append(pe.updateHandlers, handler)
}

// Trigger all the registered PropertyUpdateHandlers on this entry.
func (pe *property) firePropertyUpdate() {
	for _, h := range pe.updateHandlers {
		h(pe.value)
	}
}

/*
Bind binds a property's value to a pointer.

Example:
	var i int
	property.Bind(&i, ValTypeInt)

This will bind the property's value to i so every time it's updated i is updated as well.

The valueType indicates which field of the PropertyValue to use for the binding.
*/
func (pe *property) Bind(variable interface{}, valueType PropertyValueType) {
	var binder PropertyUpdateHandler

	switch valueType {
	case ValTypeInt:
		binder = func(val PropertyValue) {
			*(variable.(*int)) = val.IntVal
		}
	case ValTypeBoolInt:
		binder = func(val PropertyValue) {
			*(variable.(*bool)) = val.IntVal == 1
		}

	case ValTypeFloat32:
		binder = func(val PropertyValue) {
			*(variable.(*float32)) = val.FloatVal
		}

	case ValTypeFloat64:
		binder = func(val PropertyValue) {
			*(variable.(*float64)) = float64(val.FloatVal)
		}

	case ValTypeString:
		binder = func(val PropertyValue) {
			*(variable.(*string)) = val.StringVal
		}

	case ValTypeVector:
		binder = func(val PropertyValue) {
			*(variable.(*r3.Vector)) = val.VectorVal
		}

	case ValTypeArray:
		binder = func(val PropertyValue) {
			*(variable.(*[]PropertyValue)) = val.ArrayVal
		}
	}

	pe.OnUpdate(binder)
}
