package demoinfocs

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	bit "github.com/markus-wa/demoinfocs-golang/v2/internal/bitread"
	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg"
)

const (
	stNameInstanceBaseline = "instancebaseline"
	stNameUserInfo         = "userinfo"
	stNameModelPreCache    = "modelprecache"
)

type playerInfo struct {
	version     int64
	xuid        uint64
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

func (p *parser) parseStringTables() {
	p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)

	tables := int(p.bitReader.ReadSingleByte())
	for i := 0; i < tables; i++ {
		tableName := p.bitReader.ReadString()
		p.parseSingleStringTable(tableName)
	}

	p.processModelPreCacheUpdate()
	p.bitReader.EndChunk()
}

func (p *parser) updatePlayerFromRawIfExists(index int, raw *playerInfo) {
	pl := p.gameState.playersByEntityID[index+1]
	if pl == nil {
		return
	}

	oldName := pl.Name
	newName := raw.name
	nameChanged := !pl.IsBot && !raw.isFakePlayer && raw.guid != "BOT" && oldName != newName

	pl.Name = raw.name
	pl.SteamID64 = raw.xuid
	pl.IsBot = raw.isFakePlayer

	p.gameState.indexPlayerBySteamID(pl)

	if nameChanged {
		p.eventDispatcher.Dispatch(events.PlayerNameChange{
			Player:  pl,
			OldName: oldName,
			NewName: newName,
		})
	}
}

func (p *parser) parseSingleStringTable(name string) {
	nStrings := p.bitReader.ReadSignedInt(16)
	for i := 0; i < nStrings; i++ {
		stringName := p.bitReader.ReadString()

		const roysMaxStringLength = 100
		if len(stringName) >= roysMaxStringLength {
			panic("Someone said that Roy said I should panic")
		}

		if p.bitReader.ReadBit() {
			userDataSize := p.bitReader.ReadSignedInt(16)
			data := p.bitReader.ReadBytes(userDataSize)

			switch name {
			case stNameUserInfo:
				player := parsePlayerInfo(bytes.NewReader(data))

				playerIndex, err := strconv.Atoi(stringName)
				if err != nil {
					panic(errors.Wrap(err, "couldn't parse playerIndex from string"))
				}

				p.rawPlayers[int(playerIndex)] = player

				p.updatePlayerFromRawIfExists(playerIndex, player)

			case stNameInstanceBaseline:
				classID, err := strconv.Atoi(stringName)
				if err != nil {
					panic(errors.Wrap(err, "couldn't parse serverClassID from string"))
				}

				p.stParser.SetInstanceBaseline(int(classID), data)

			case stNameModelPreCache:
				p.modelPreCache = append(p.modelPreCache, stringName)

			default: // Irrelevant table
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

func (p *parser) handleUpdateStringTable(tab *msg.CSVCMsg_UpdateStringTable) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

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

func (p *parser) handleCreateStringTable(tab *msg.CSVCMsg_CreateStringTable) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	p.processStringTable(tab)

	p.stringTables = append(p.stringTables, tab)

	p.eventDispatcher.Dispatch(events.StringTableCreated{TableName: tab.Name})
}

//nolint:funlen,gocognit
func (p *parser) processStringTable(tab *msg.CSVCMsg_CreateStringTable) {
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

		if entryIndex < 0 || entryIndex >= int(tab.MaxEntries) {
			panic("Something went to shit")
		}

		var entry string
		if br.ReadBit() { //nolint:wsl
			if br.ReadBit() {
				idx := br.ReadInt(5)
				bytes2cp := int(br.ReadInt(5))
				entry = hist[idx][:bytes2cp]

				entry += br.ReadString()
			} else {
				entry = br.ReadString()
			}
		}

		const maxHistoryLength = 31
		if len(hist) > maxHistoryLength {
			hist = hist[1:]
		}

		hist = append(hist, entry)

		var userdata []byte
		if br.ReadBit() { //nolint:wsl
			if tab.UserDataFixedSize {
				// Should always be < 8 bits => use faster ReadBitsToByte() over ReadBits()
				userdata = []byte{br.ReadBitsToByte(int(tab.UserDataSizeBits))}
			} else {
				const nUserdataBits = 14
				userdata = br.ReadBytes(int(br.ReadInt(nUserdataBits)))
			}
		}

		if len(userdata) == 0 {
			continue
		}

		switch tab.Name {
		case stNameUserInfo:
			player := parsePlayerInfo(bytes.NewReader(userdata))
			p.rawPlayers[entryIndex] = player

			p.updatePlayerFromRawIfExists(entryIndex, player)

		case stNameInstanceBaseline:
			classID, err := strconv.Atoi(entry)
			if err != nil {
				panic(errors.Wrap(err, "failed to parse serverClassID"))
			}

			p.stParser.SetInstanceBaseline(classID, userdata)

		case stNameModelPreCache:
			p.modelPreCache[entryIndex] = entry
		}
	}

	if tab.Name == stNameModelPreCache {
		p.processModelPreCacheUpdate()
	}

	p.poolBitReader(br)
}

func parsePlayerInfo(reader io.Reader) *playerInfo {
	br := bit.NewSmallBitReader(reader)

	const (
		playerNameMaxLength = 128
		guidLength          = 33
	)

	res := &playerInfo{
		version:     int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		xuid:        binary.BigEndian.Uint64(br.ReadBytes(8)),
		name:        br.ReadCString(playerNameMaxLength),
		userID:      int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		guid:        br.ReadCString(guidLength),
		friendsID:   int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		friendsName: br.ReadCString(playerNameMaxLength),

		isFakePlayer: br.ReadSingleByte() != 0,
		isHltv:       br.ReadSingleByte() != 0,

		customFiles0: int(br.ReadInt(32)),
		customFiles1: int(br.ReadInt(32)),
		customFiles2: int(br.ReadInt(32)),
		customFiles3: int(br.ReadInt(32)),

		filesDownloaded: br.ReadSingleByte(),
	}

	br.Pool()

	return res
}

var modelPreCacheSubstringToEq = map[string]common.EquipmentType{
	"flashbang":         common.EqFlash,
	"fraggrenade":       common.EqHE,
	"smokegrenade":      common.EqSmoke,
	"molotov":           common.EqMolotov,
	"incendiarygrenade": common.EqIncendiary,
	"decoy":             common.EqDecoy,
	// @micvbang TODO: add all other weapons too.
}

func (p *parser) processModelPreCacheUpdate() {
	for i, name := range p.modelPreCache {
		for eqName, eq := range modelPreCacheSubstringToEq {
			if strings.Contains(name, eqName) {
				p.grenadeModelIndices[i] = eq
			}
		}
	}
}
