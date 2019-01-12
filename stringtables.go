package demoinfocs

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"
	"strings"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

const (
	stNameInstanceBaseline = "instancebaseline"
	stNameUserInfo         = "userinfo"
	stNameModelPreCache    = "modelprecache"
)

type playerInfo struct {
	version     int64
	xuid        int64
	name        string
	userID      int
	guid        string
	friendsID   int
	friendsName string
	// Custom files stuff (CRC)
	customFiles0 int
	customFiles1 int
	customFiles2 int
	customFiles3 int
	// Amount of downloaded files from the server
	filesDownloaded byte
	// Bots
	isFakePlayer bool
	// HLTV Proxy
	isHltv bool
}

func (p *Parser) parseStringTables() {
	p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
	tables := int(p.bitReader.ReadSingleByte())
	for i := 0; i < tables; i++ {
		tableName := p.bitReader.ReadString()
		p.parseSingleStringTable(tableName)
	}
	p.processModelPreCacheUpdate()
	p.bitReader.EndChunk()
}

func (p *Parser) parseSingleStringTable(name string) {
	nStrings := p.bitReader.ReadSignedInt(16)
	for i := 0; i < nStrings; i++ {
		stringName := p.bitReader.ReadString()
		if len(stringName) >= 100 {
			panic("Someone said that Roy said I should panic")
		}
		if p.bitReader.ReadBit() {
			userDataSize := p.bitReader.ReadSignedInt(16)
			data := p.bitReader.ReadBytes(userDataSize)
			switch name {
			case stNameUserInfo:
				player := parsePlayerInfo(bytes.NewReader(data))
				playerIndex, err := strconv.ParseInt(stringName, 10, 64)
				if err != nil {
					panic("Couldn't parse playerIndex from string")
				}
				p.rawPlayers[int(playerIndex)] = player

			case stNameInstanceBaseline:
				classID, err := strconv.ParseInt(stringName, 10, 64)
				if err != nil {
					panic("Couldn't parse id from string")
				}
				p.stParser.SetInstanceBaseline(int(classID), data)

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
				p.bitReader.Skip(p.bitReader.ReadSignedInt(16))
			}
		}
	}
}

func (p *Parser) handleUpdateStringTable(tab *msg.CSVCMsg_UpdateStringTable) {
	// No need for recoverFromUnexpectedEOF here as we do that in processStringTable already

	cTab := p.stringTables[tab.TableId]
	switch cTab.Name {
	case stNameUserInfo:
		fallthrough
	case stNameModelPreCache:
		fallthrough
	case stNameInstanceBaseline:
		// Only handle updates for the above types
		// Create fake CreateStringTable and handle it like one of those
		cTab.NumEntries = tab.NumChangedEntries
		cTab.StringData = tab.StringData
		p.processStringTable(cTab)
	}
}

func (p *Parser) handleCreateStringTable(tab *msg.CSVCMsg_CreateStringTable) {
	// No need for recoverFromUnexpectedEOF here as we do that in processStringTable already

	p.processStringTable(tab)

	p.stringTables = append(p.stringTables, tab)

	p.eventDispatcher.Dispatch(events.StringTableCreated{TableName: tab.Name})
}

func (p *Parser) processStringTable(tab *msg.CSVCMsg_CreateStringTable) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	if tab.Name == stNameModelPreCache {
		for i := len(p.modelPreCache); i < int(tab.MaxEntries); i++ {
			p.modelPreCache = append(p.modelPreCache, "")
		}
	}

	br := bit.NewSmallBitReader(bytes.NewReader(tab.StringData))

	if br.ReadBit() {
		panic("Can't decode")
	}

	nTmp := tab.MaxEntries
	nEntryBits := 0

	for nTmp != 0 {
		nTmp >>= 1
		nEntryBits++
	}
	if nEntryBits > 0 {
		nEntryBits--
	}

	hist := make([]string, 0)
	lastEntry := -1
	for i := 0; i < int(tab.NumEntries); i++ {
		entryIndex := lastEntry + 1
		if !br.ReadBit() {
			entryIndex = int(br.ReadInt(nEntryBits))
		}

		lastEntry = entryIndex

		var entry string
		if entryIndex < 0 || entryIndex >= int(tab.MaxEntries) {
			panic("Something went to shit")
		}
		if br.ReadBit() {
			if br.ReadBit() {
				idx := br.ReadInt(5)
				bytes2cp := int(br.ReadInt(5))
				entry = hist[idx][:bytes2cp]

				entry += br.ReadString()
			} else {
				entry = br.ReadString()
			}
		}

		if len(hist) > 31 {
			hist = hist[1:]
		}
		hist = append(hist, entry)

		var userdata []byte
		if br.ReadBit() {
			if tab.UserDataFixedSize {
				// Should always be < 8 bits => use faster ReadBitsToByte() over ReadBits()
				userdata = []byte{br.ReadBitsToByte(int(tab.UserDataSizeBits))}
			} else {
				userdata = br.ReadBytes(int(br.ReadInt(14)))
			}
		}

		if len(userdata) == 0 {
			continue
		}

		switch tab.Name {
		case stNameUserInfo:
			p.rawPlayers[entryIndex] = parsePlayerInfo(bytes.NewReader(userdata))

		case stNameInstanceBaseline:
			classID, err := strconv.ParseInt(entry, 10, 64)
			if err != nil {
				panic("WTF VOLVO PLS")
			}
			p.stParser.SetInstanceBaseline(int(classID), userdata)

		case stNameModelPreCache:
			p.modelPreCache[entryIndex] = entry
		}
	}

	if tab.Name == stNameModelPreCache {
		p.processModelPreCacheUpdate()
	}

	br.Pool()
}

func parsePlayerInfo(reader io.Reader) *playerInfo {
	br := bit.NewSmallBitReader(reader)

	res := &playerInfo{
		version:     int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		xuid:        int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		name:        br.ReadCString(128),
		userID:      int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		guid:        br.ReadCString(33),
		friendsID:   int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		friendsName: br.ReadCString(128),

		isFakePlayer: br.ReadSingleByte()&0xff != 0,
		isHltv:       br.ReadSingleByte()&0xff != 0,

		customFiles0: int(br.ReadInt(32)),
		customFiles1: int(br.ReadInt(32)),
		customFiles2: int(br.ReadInt(32)),
		customFiles3: int(br.ReadInt(32)),

		filesDownloaded: br.ReadSingleByte(),
	}

	br.Pool()
	return res
}

var modelPreCacheSubstringToEq = map[string]common.EquipmentElement{
	"flashbang_dropped":         common.EqFlash,
	"fraggrenade_dropped":       common.EqHE,
	"smokegrenade_thrown":       common.EqSmoke,
	"molotov_dropped":           common.EqMolotov,
	"incendiarygrenade_dropped": common.EqIncendiary,
	"decoy_dropped":             common.EqDecoy,
	// @micvbang TODO: add all other weapons too.
}

func (p *Parser) processModelPreCacheUpdate() {
	for i, name := range p.modelPreCache {
		for eqName, eq := range modelPreCacheSubstringToEq {
			if strings.Contains(name, eqName) {
				p.grenadeModelIndices[i] = eq
			}
		}
	}
}
