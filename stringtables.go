package demoinfocs

import (
	"bytes"
	"strconv"

	common "github.com/markus-wa/demoinfocs-golang/common"
)

func (p *Parser) parseStringTables() {
	p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
	tables := int(p.bitReader.ReadSingleByte())
	for i := 0; i < tables; i++ {
		tableName := p.bitReader.ReadString()
		p.parseSingleStringTable(tableName)
	}
	p.bitReader.EndChunk()
}

func (p *Parser) parseSingleStringTable(name string) {
	strings := p.bitReader.ReadSignedInt(16)
	for i := 0; i < strings; i++ {
		stringName := p.bitReader.ReadString()
		if len(stringName) >= 100 {
			panic("Someone said that Roy said I should panic")
		}
		if p.bitReader.ReadBit() {
			userDataSize := p.bitReader.ReadSignedInt(16)
			data := p.bitReader.ReadBytes(userDataSize)
			switch name {
			case stNameUserInfo:
				player := common.ParsePlayerInfo(bytes.NewReader(data))
				pid, err := strconv.ParseInt(stringName, 10, 64)
				if err != nil {
					panic("Couldn't parse id from string")
				}
				p.rawPlayers[int(pid)] = player

			case stNameInstanceBaseline:
				pid, err := strconv.ParseInt(stringName, 10, 64)
				if err != nil {
					panic("Couldn't parse id from string")
				}
				p.instanceBaselines[int(pid)] = data

			case stNameModelPreCache:
				p.modelPreCache = append(p.modelPreCache, stringName)

			default:
				// Irrelevant table
			}
		}
	}
	// Client side stuff, dgaf
	if p.bitReader.ReadBit() {
		strings2 := p.bitReader.ReadSignedInt(16)
		for i := 0; i < strings2; i++ {
			p.bitReader.ReadString()
			if p.bitReader.ReadBit() {
				p.bitReader.ReadBytes(p.bitReader.ReadSignedInt(16))
			}
		}
	}
}
