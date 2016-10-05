package dt

import (
	"errors"
)

type SendPropertyType byte
type SendPropertyFlags int

const (
	SPT_Int SendPropertyType = iota
	SPT_Float
	SPT_Vector
	SPT_VectorXY
	SPT_String
	SPT_Array
	SPT_Data_Table
	SPT_Int64
)

const (
	SPF_Unsigned SendPropertyFlags = (1 << iota)
	SPF_Coord
	SPF_NoScale
	SPF_RoundDown
	SPF_RoundUp
	SPF_Normal
	SPF_Exclude
	SPF_XYZE
	SPF_InsideArray
	SPF_ProxyAlwaysYes
	SPF_IsVectorElement
	SPF_Collapsible
	SPF_CoordMp
	SPF_CoordMpLowPrecision
	SPF_CoordMpIntegral
	SPF_CellCoord
	SPF_CellCoordLowPrecision
	SPF_CellCoordIntegral
	SPF_ChangesOften
	SPF_VarInt
)

type SendTableProperty struct {
	RawFlags         int
	Name             string
	DataTableName    string
	LowValue         float32
	HighValue        float32
	NumberOfBits     int
	NumberOfElements int
	Priority         int
	RawType          int
}

func (stp *SendTableProperty) Flags() SendPropertyFlags {
	return SendPropertyFlags(stp.RawFlags)
}

func (stp *SendTableProperty) Type() SendPropertyType {
	return SendPropertyType(stp.RawType)
}

type Entity struct {
	Id          int
	ServerClass *ServerClass
	props       []*PropertyEntry
}

func (e *Entity) Props() []*PropertyEntry {
	return e.props
}

func (e *Entity) FindProperty(name string) (*PropertyEntry, error) {
	for _, p := range e.props {
		if p.entry.name == name {
			return p, nil
		}
	}
	return &PropertyEntry{}, errors.New("Could not find property with name" + name)
}

func NewEntity(id int, serverClass *ServerClass) *Entity {
	props := make([]*PropertyEntry, len(serverClass.FlattenedProps))
	for i, p := range serverClass.FlattenedProps {
		props[i] = NewPropertyEntry(p, int(i))
	}
	return &Entity{Id: id, ServerClass: serverClass, props: props}
}

type PropertyEntry struct {
	index int
	entry *FlattenedPropEntry
}

func NewPropertyEntry(entry *FlattenedPropEntry, index int) *PropertyEntry {
	return &PropertyEntry{index: index, entry: entry}
}
