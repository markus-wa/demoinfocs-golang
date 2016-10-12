package st

import ()

type SendTable struct {
	properties []*SendTableProperty
	Name       string
	IsEnd      bool
}

func (st *SendTable) Properties() []*SendTableProperty {
	return st.properties
}

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
