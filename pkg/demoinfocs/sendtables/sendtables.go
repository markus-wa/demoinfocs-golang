// Package sendtables contains code related to decoding sendtables.
// Mostly used internally but can be interesting for direct access to server-classes and entities.
package sendtables

import (
	"fmt"

	"github.com/golang/geo/r3"
)

// PropertyValueType specifies the type of PropertyValue
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

// PropertyType identifies the data type of property.
type PropertyType int

// PropertyValue stores parsed & decoded send-table values.
// For instance player health, location etc.
type PropertyValue struct {
	Any any
}

func (v PropertyValue) R3Vec() r3.Vector {
	fs := v.Any.([]float32)

	return r3.Vector{
		X: float64(fs[0]),
		Y: float64(fs[1]),
		Z: float64(fs[2]),
	}
}

func (v PropertyValue) R3VecOrNil() *r3.Vector {
	if v.Any == nil {
		return nil
	}

	fs := v.Any.([]float32)

	return &r3.Vector{
		X: float64(fs[0]),
		Y: float64(fs[1]),
		Z: float64(fs[2]),
	}
}

func (v PropertyValue) Int() int {
	return int(v.Any.(int32))
}

func (v PropertyValue) Int64() int64 {
	return v.Any.(int64)
}

func (v PropertyValue) S2UInt64() uint64 {
	return v.Any.(uint64)
}

func (v PropertyValue) S2Array() []any {
	return v.Any.([]any)
}

func (v PropertyValue) S2UInt32() uint32 {
	return v.Any.(uint32)
}

func (v PropertyValue) Handle() uint64 {
	return v.S2UInt64()
}

func (v PropertyValue) Float() float32 {
	return v.Any.(float32)
}

func (v PropertyValue) Str() string {
	return v.Any.(string)
}

func (v PropertyValue) String() string {
	return fmt.Sprint(v.Any)
}

// BoolVal returns true if IntVal > 0.
func (v PropertyValue) BoolVal() bool {
	return v.Any.(bool)
}

// PropertyUpdateHandler is the interface for handlers that are interested in property changes.
type PropertyUpdateHandler func(PropertyValue)

type PropertyEntry struct {
	Name    string
	IsArray bool
	Type    PropertyType
}

// EntityCreatedHandler is the interface for handlers that are interested in EntityCreatedEvents.
type EntityCreatedHandler func(Entity)

// ServerClasses is a searchable list of ServerClasses.
type ServerClasses interface {
	All() []ServerClass
	FindByName(name string) ServerClass
}

// EntityOp is a bitmask representing the type of operation performed on an Entity
type EntityOp int

const (
	EntityOpNone           EntityOp = 0x00
	EntityOpCreated        EntityOp = 0x01
	EntityOpUpdated        EntityOp = 0x02
	EntityOpDeleted        EntityOp = 0x04
	EntityOpEntered        EntityOp = 0x08
	EntityOpLeft           EntityOp = 0x10
	EntityOpCreatedEntered EntityOp = EntityOpCreated | EntityOpEntered
	EntityOpUpdatedEntered EntityOp = EntityOpUpdated | EntityOpEntered
	EntityOpDeletedLeft    EntityOp = EntityOpDeleted | EntityOpLeft
)

var entityOpNames = map[EntityOp]string{
	EntityOpNone:           "None",
	EntityOpCreated:        "Created",
	EntityOpUpdated:        "Updated",
	EntityOpDeleted:        "Deleted",
	EntityOpEntered:        "Entered",
	EntityOpLeft:           "Left",
	EntityOpCreatedEntered: "Created+Entered",
	EntityOpUpdatedEntered: "Updated+Entered",
	EntityOpDeletedLeft:    "Deleted+Left",
}

// Flag determines whether an EntityOp includes another. This is primarily
// offered to prevent bit flag errors in downstream clients.
func (o EntityOp) Flag(p EntityOp) bool {
	return o&p != 0
}

// String returns a human identifiable string for the EntityOp
func (o EntityOp) String() string {
	return entityOpNames[o]
}

// EntityHandler is a function that receives Entity updates
type EntityHandler func(Entity, EntityOp) error
