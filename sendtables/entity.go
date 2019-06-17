package sendtables

import (
	"fmt"
	"sync"

	"github.com/golang/geo/r3"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
)

//go:generate ifacemaker -f entity.go -s Entity -i IEntity -p sendtables -D -y "IEntity is an auto-generated interface for Entity, intended to be used when mockability is needed." -c "DO NOT EDIT: Auto generated" -o entity_interface.go
//go:generate ifacemaker -f entity.go -s Property -i IProperty -p sendtables -D -y "IProperty is an auto-generated interface for Property, intended to be used when mockability is needed." -c "DO NOT EDIT: Auto generated" -o property_interface.go

// Entity stores a entity in the game (e.g. players etc.) with its properties.
type Entity struct {
	serverClass *ServerClass
	id          int
	props       []Property

	onCreateFinished []func()
	onDestroy        []func()
	position         func() r3.Vector
}

// ServerClass returns the entity's server-class.
func (e *Entity) ServerClass() *ServerClass {
	return e.serverClass
}

// ID returns the entity's ID.
func (e *Entity) ID() int {
	return e.id
}

// Properties is deprecated, use PropertiesI() instead which returns a slice of interfaces.
func (e *Entity) Properties() (out []Property) {
	return e.props
}

// PropertiesI returns all properties of the Entity.
func (e *Entity) PropertiesI() (out []IProperty) {
	for i := range e.props {
		out = append(out, &e.props[i])
	}
	return
}

// FindProperty is deprecated, use FindPropertyI() instead which returns an interface.
func (e *Entity) FindProperty(name string) (prop *Property) {
	for i := range e.props {
		if e.props[i].entry.name == name {
			if prop != nil {
				panic(fmt.Sprintf("More than one property with name %q found", name))
			}
			prop = &e.props[i]
		}
	}
	return
}

// FindPropertyI finds a property on the Entity by name.
//
// Returns nil if the property wasn't found.
//
// Panics if more than one property with the same name was found.
func (e *Entity) FindPropertyI(name string) IProperty {
	prop := e.FindProperty(name)
	if prop == nil {
		// See https://stackoverflow.com/questions/13476349/check-for-nil-and-nil-interface-in-go
		return nil
	}
	return prop
}

// BindProperty combines FindPropertyI() & Property.Bind() into one.
// Essentially binds a property's value to a pointer.
// See the docs of the two individual functions for more info.
func (e *Entity) BindProperty(name string, variable interface{}, valueType PropertyValueType) {
	e.FindPropertyI(name).Bind(variable, valueType)
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

	// end marker
	if res == 0xfff {
		return -1
	}

	return lastIndex + 1 + int(res)
}

// Collects an initial baseline for a server-class
func (e *Entity) initializeBaseline(r *bit.BitReader) map[int]PropertyValue {
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
func (e *Entity) applyBaseline(baseline map[int]PropertyValue) {
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

// Sets up the Entity.Position() function
// Necessary because FindPropertyI() is fairly slow
// This way we only need to find the necessary properties once
func (e *Entity) initialize() {
	// Player positions are calculated differently
	if e.isPlayer() {
		xyProp := e.FindPropertyI(propVecOriginPlayerXY)
		zProp := e.FindPropertyI(propVecOriginPlayerZ)

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
		cellBitsProp := e.FindPropertyI(propCellBits)
		cellXProp := e.FindPropertyI(propCellX)
		cellYProp := e.FindPropertyI(propCellY)
		cellZProp := e.FindPropertyI(propCellZ)
		offsetProp := e.FindPropertyI(propVecOrigin)

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

func (e *Entity) isPlayer() bool {
	return e.serverClass.name == serverClassPlayer
}

// Position returns the entity's position in world coordinates.
func (e *Entity) Position() r3.Vector {
	return e.position()
}

// OnPositionUpdate registers a handler for the entity's position update.
// The handler is called with the new position every time a position-relevant property is updated.
//
// See also Position()
func (e *Entity) OnPositionUpdate(h func(pos r3.Vector)) {
	pos := new(r3.Vector)
	firePosUpdate := func(PropertyValue) {
		newPos := e.Position()
		if *pos != newPos {
			h(newPos)
			*pos = newPos
		}
	}

	if e.isPlayer() {
		e.FindPropertyI(propVecOriginPlayerXY).OnUpdate(firePosUpdate)
		e.FindPropertyI(propVecOriginPlayerZ).OnUpdate(firePosUpdate)
	} else {
		e.FindPropertyI(propCellX).OnUpdate(firePosUpdate)
		e.FindPropertyI(propCellY).OnUpdate(firePosUpdate)
		e.FindPropertyI(propCellZ).OnUpdate(firePosUpdate)
		e.FindPropertyI(propVecOrigin).OnUpdate(firePosUpdate)
	}
}

// BindPosition binds the entity's position to a pointer variable.
// The pointer is updated every time a position-relevant property is updated.
//
// See also OnPositionUpdate()
func (e *Entity) BindPosition(pos *r3.Vector) {
	e.OnPositionUpdate(func(newPos r3.Vector) {
		*pos = newPos
	})
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

// Property wraps a flattenedPropEntry and allows registering handlers
// that can be triggered on a update of the property.
type Property struct {
	entry          *flattenedPropEntry
	updateHandlers []PropertyUpdateHandler
	value          PropertyValue
}

// Name returns the property's name.
func (pe *Property) Name() string {
	return pe.entry.name
}

// Value returns current value of the property.
func (pe *Property) Value() PropertyValue {
	return pe.value
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

// PropertyUpdateHandler is the interface for handlers that are interested in Property changes.
type PropertyUpdateHandler func(PropertyValue)

// OnUpdate registers a handler for updates of the Property's value.
//
// The handler will be called with the current value upon registration.
func (pe *Property) OnUpdate(handler PropertyUpdateHandler) {
	handler(pe.value)
	pe.updateHandlers = append(pe.updateHandlers, handler)
}

// Trigger all the registered PropertyUpdateHandlers on this entry.
func (pe *Property) firePropertyUpdate() {
	for _, h := range pe.updateHandlers {
		h(pe.value)
	}
}

/*
Bind binds a property's value to a pointer.

Example:
	var i int
	Property.Bind(&i, ValTypeInt)

This will bind the property's value to i so every time it's updated i is updated as well.

The valueType indicates which field of the PropertyValue to use for the binding.
*/
func (pe *Property) Bind(variable interface{}, valueType PropertyValueType) {
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
