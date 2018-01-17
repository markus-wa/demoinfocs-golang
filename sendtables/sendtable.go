// Package sendtables contains sendtable specific magic and should really be better documented (TODO).
package sendtables

type sendTable struct {
	properties []sendTableProperty
	name       string
	isEnd      bool
}

type sendTableProperty struct {
	flags            sendPropertyFlags
	name             string
	dataTableName    string
	lowValue         float32
	highValue        float32
	numberOfBits     int
	numberOfElements int
	priority         int
	rawType          int
}
