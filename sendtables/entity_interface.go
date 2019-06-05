// DO NOT EDIT: Auto generated

package sendtables

import (
	"github.com/golang/geo/r3"
	bit "github.com/markus-wa/demoinfocs-golang/bitread"
)

// IEntity is an auto-generated interface for Entity, intended to be used when mockability is needed.
// Entity stores a entity in the game (e.g. players etc.) with its properties.
type IEntity interface {
	// ServerClass returns the entity's server-class.
	ServerClass() *ServerClass
	// ID returns the entity's ID.
	ID() int
	// Properties is deprecated, use PropertiesI() instead.
	Properties() (out []Property)
	// PropertiesI returns all properties of the Entity.
	PropertiesI() (out []IProperty)
	// FindProperty is deprecated, use FindPropertyI() instead.
	FindProperty(name string) (prop *Property)
	// FindPropertyI finds a property on the Entity by name.
	//
	// Returns nil if the property wasn't found.
	//
	// Panics if more than one property with the same name was found.
	FindPropertyI(name string) (prop IProperty)
	// BindProperty combines FindPropertyI() & Property.Bind() into one.
	// Essentially binds a property's value to a pointer.
	// See the docs of the two individual functions for more info.
	BindProperty(name string, variable interface{}, valueType PropertyValueType)
	// ApplyUpdate reads an update to an Enitiy's properties and
	// triggers registered PropertyUpdateHandlers if values changed.
	//
	// Intended for internal use only.
	ApplyUpdate(reader *bit.BitReader)
	// Position returns the entity's position in world coordinates.
	Position() r3.Vector
	// OnPositionUpdate registers a handler for the entity's position update.
	// The handler is called with the new position every time a position-relevant property is updated.
	//
	// See also Position()
	OnPositionUpdate(h func(pos r3.Vector))
	// BindPosition binds the entity's position to a pointer variable.
	// The pointer is updated every time a position-relevant property is updated.
	//
	// See also OnPositionUpdate()
	BindPosition(pos *r3.Vector)
	// OnDestroy registers a function to be called on the entity's destruction.
	OnDestroy(delegate func())
	// Destroy triggers all via OnDestroy() registered functions.
	//
	// Intended for internal use only.
	Destroy()
	// OnCreateFinished registers a function to be called once the entity is fully created -
	// i.e. once all property updates have been sent out.
	OnCreateFinished(delegate func())
}
