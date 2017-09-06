// Package st contains sendtable specific magic and should really be better documented (TODO).
package st

type SendTable struct {
	properties []SendTableProperty
	Name       string
	IsEnd      bool
}

type SendTableProperty struct {
	Flags            SendPropertyFlags
	Name             string
	DataTableName    string
	LowValue         float32
	HighValue        float32
	NumberOfBits     int
	NumberOfElements int
	Priority         int
	RawType          int
}
