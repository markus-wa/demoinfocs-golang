package st

import (
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
)

func ParsePacket(r bs.BitReader) {
	tables := int(r.ReadByte())
	for i := 0; i < tables; i++ {
		tableName := r.ReadString()
		parseStringTable(r, tableName)
	}
}

func parseStringTable(r bs.BitReader, name string) {
	// FIXME: Do parsing
	getItOverWith(r, name)
}

func getItOverWith(r bs.BitReader, name string) {
	strings := r.ReadSignedInt(16)
	for i := 0; i < strings; i++ {
		stringName := r.ReadString()
		if len(stringName) >= 100 {
			panic("Someone said that Roy said I should panic")
		}
		if r.ReadBit() {
			userDataSize := r.ReadSignedInt(16)
			r.ReadBytes(userDataSize)
			switch name {
			case "userinfo":
				// FIXME: Parse player info here
			case "instancebaseline":
				// FIXME: Do instancebaseline stuff
			case "modelprecache":
			// FIXME: Do modelprecache stuff
			default:
				// FIXME: Unknown table
			}
		}
	}
	// Client side stuff, dgaf
	if r.ReadBit() {
		strings2 := r.ReadSignedInt(16)
		for i := 0; i < strings2; i++ {
			r.ReadString()
			if r.ReadBit() {
				r.ReadBytes(r.ReadSignedInt(16))
			}
		}
	}
}
