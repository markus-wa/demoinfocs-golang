package demoinfocs

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	bit "github.com/markus-wa/demoinfocs-golang/v3/internal/bitread"
	common "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg"
)

const (
	stNameInstanceBaseline = "instancebaseline"
	stNameUserInfo         = "userinfo"
	stNameModelPreCache    = "modelprecache"
)

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

func (p *parser) updatePlayerFromRawIfExists(index int, raw common.PlayerInfo) {
	pl := p.gameState.playersByEntityID[index+1]
	if pl == nil {
		return
	}

	oldName := pl.Name
	newName := raw.Name
	nameChanged := !pl.IsBot && !raw.IsFakePlayer && raw.GUID != "BOT" && oldName != newName

	pl.Name = raw.Name
	pl.SteamID64 = raw.XUID
	pl.IsBot = raw.IsFakePlayer

	p.gameState.indexPlayerBySteamID(pl)

	if nameChanged {
		p.eventDispatcher.Dispatch(events.PlayerNameChange{
			Player:  pl,
			OldName: oldName,
			NewName: newName,
		})
	}

	p.eventDispatcher.Dispatch(events.StringTablePlayerUpdateApplied{
		Player: pl,
	})
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

				p.setRawPlayer(playerIndex, player)

			case stNameInstanceBaseline:
				classID, err := strconv.Atoi(stringName)
				if err != nil {
					panic(errors.Wrap(err, "couldn't parse serverClassID from string"))
				}

				p.stParser.SetInstanceBaseline(classID, data)

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
func (p *parser) setRawPlayer(index int, player common.PlayerInfo) {
	p.rawPlayers[index] = &player

	p.updatePlayerFromRawIfExists(index, player)

	p.eventDispatcher.Dispatch(events.PlayerInfo{
		Index: index,
		Info:  player,
	})
}

func (p *parser) handleUpdateStringTable(tab *msg.CSVCMsg_UpdateStringTable) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	cTab := p.stringTables[tab.GetTableId()]
	switch cTab.GetName() {
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

	p.eventDispatcher.Dispatch(events.StringTableCreated{TableName: tab.GetName()})
}

//nolint:funlen,gocognit
func (p *parser) processStringTable(tab *msg.CSVCMsg_CreateStringTable) {
	if tab.GetName() == stNameModelPreCache {
		for i := len(p.modelPreCache); i < int(tab.GetMaxEntries()); i++ {
			p.modelPreCache = append(p.modelPreCache, "")
		}
	}

	br := bit.NewSmallBitReader(bytes.NewReader(tab.StringData))

	if br.ReadBit() {
		panic("Can't decode")
	}

	nTmp := tab.GetMaxEntries()
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

	for i := 0; i < int(tab.GetNumEntries()); i++ {
		entryIndex := lastEntry + 1
		if !br.ReadBit() {
			entryIndex = int(br.ReadInt(nEntryBits))
		}

		lastEntry = entryIndex

		if entryIndex < 0 || entryIndex >= int(tab.GetMaxEntries()) {
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
			if tab.GetUserDataFixedSize() {
				// Should always be < 8 bits => use faster ReadBitsToByte() over ReadBits()
				userdata = []byte{br.ReadBitsToByte(int(tab.GetUserDataSizeBits()))}
			} else {
				const nUserdataBits = 14
				userdata = br.ReadBytes(int(br.ReadInt(nUserdataBits)))
			}
		}

		if len(userdata) == 0 {
			continue
		}

		switch tab.GetName() {
		case stNameUserInfo:
			player := parsePlayerInfo(bytes.NewReader(userdata))

			p.setRawPlayer(entryIndex, player)

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

	if tab.GetName() == stNameModelPreCache {
		p.processModelPreCacheUpdate()
	}

	p.poolBitReader(br)
}

func parsePlayerInfo(reader io.Reader) common.PlayerInfo {
	br := bit.NewSmallBitReader(reader)

	const (
		playerNameMaxLength = 128
		guidLength          = 33
	)

	res := common.PlayerInfo{
		Version:     int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		XUID:        binary.BigEndian.Uint64(br.ReadBytes(8)),
		Name:        br.ReadCString(playerNameMaxLength),
		UserID:      int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		GUID:        br.ReadCString(guidLength),
		FriendsID:   int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		FriendsName: br.ReadCString(playerNameMaxLength),

		IsFakePlayer: br.ReadSingleByte() != 0,
		IsHltv:       br.ReadSingleByte() != 0,

		CustomFiles0: int(br.ReadInt(32)),
		CustomFiles1: int(br.ReadInt(32)),
		CustomFiles2: int(br.ReadInt(32)),
		CustomFiles3: int(br.ReadInt(32)),

		FilesDownloaded: br.ReadSingleByte(),
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
