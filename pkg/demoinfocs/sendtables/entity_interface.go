// DO NOT EDIT: Auto generated

package sendtables

import (
	"github.com/golang/geo/r3"
	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
)

// Entity is an auto-generated interface for entity, intended to be used when mockability is needed.
// entity stores an entity in the game (e.g. players etc.) with its properties.
type Entity interface {
	// ServerClass returns the entity's server-class.
	ServerClass() ServerClass
	// ID returns the entity's ID.
	ID() int
	// SerialNum returns the entity's serial number.
	SerialNum() int
	// Properties returns all properties of the entity.
	Properties() (out []Property)
	// Property finds a property on the entity by name.
	//
	// Returns nil if the property wasn't found.
	Property(name string) Property
	// BindProperty combines Property() & Property.Bind() into one.
	// Essentially binds a property's value to a pointer.
	// See the docs of the two individual functions for more info.
	BindProperty(name string, variable any, valueType PropertyValueType) int64
	// PropertyValue finds a property on the entity by name and returns its value.
	//
	// Returns false as second value if the property was not found.
	PropertyValue(name string) (PropertyValue, bool)
	// PropertyValueMust finds a property on the entity by name and returns its value.
	//
	// Panics with nil pointer dereference error if the property was not found.
	PropertyValueMust(name string) PropertyValue
	// ApplyUpdate reads an update to an Enitiy's properties and
	// triggers registered PropertyUpdateHandlers if values changed.
	//
	// Intended for internal use only.
	ApplyUpdate(reader *bit.BitReader)
	// Position returns the entity's position in world coordinates.
	Position() r3.Vector
	// OnPositionUpdateWithId registers a handler for the entity's position update with the given id.
	// The handler is called with the new position every time a position-relevant property is updated.
	//
	// See also Position()
	OnPositionUpdateWithId(h func(pos r3.Vector), id int64)
	// OnPositionUpdate registers a handler for the entity's position update and returns a randomly-generated handler id.
	// The handler is called with the new position every time a position-relevant property is updated.
	//
	// See also Position()
	OnPositionUpdate(h func(pos r3.Vector))
	// OnDestroyWithId registers a function to be called on the entity's destruction with a given id.
	OnDestroyWithId(delegate func(), id int64)
	// OnDestroy registers a function to be called on the entity's destruction and returns a randomly-generated handler id.
	OnDestroy(delegate func()) (delegateId int64)
	// Destroy triggers all via OnDestroy() registered functions.
	//
	// Intended for internal use only.
	Destroy()
	// OnCreateFinished registers a function to be called once the entity is fully created -
	// i.e. once all property updates have been sent out.
	OnCreateFinished(delegate func())
}
