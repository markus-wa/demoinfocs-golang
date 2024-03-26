package demoinfocs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
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
	if player.UserID == math.MaxUint16 && p.rawPlayers[index] != nil {
		player.UserID = p.rawPlayers[index].UserID
	}

	p.rawPlayers[index] = &player

	p.updatePlayerFromRawIfExists(index, player)

	p.eventDispatcher.Dispatch(events.PlayerInfo{
		Index: index,
		Info:  player,
	})
}

func (p *parser) handleUpdateStringTable(tab *msgs2.CSVCMsg_UpdateStringTable, s2 bool) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	if len(p.stringTables) <= int(tab.GetTableId()) {
		return // FIXME: We never got a proper CreateStringTable for this table ...
	}

	cTab := p.stringTables[tab.GetTableId()]

	switch cTab.GetName() {
	case stNameUserInfo:
		fallthrough
	case stNameModelPreCache:
		fallthrough
	case stNameInstanceBaseline:
		// Only handle updates for the above types
		// Create fake CreateStringTable and handle it like one of those
		p.processStringTable(createStringTable{
			CSVCMsg_CreateStringTable: &msgs2.CSVCMsg_CreateStringTable{
				Name:                 cTab.Name,
				NumEntries:           tab.NumChangedEntries,
				UserDataFixedSize:    cTab.UserDataFixedSize,
				UserDataSize:         cTab.UserDataSize,
				UserDataSizeBits:     cTab.UserDataSizeBits,
				Flags:                cTab.Flags,
				StringData:           tab.StringData,
				UsingVarintBitcounts: cTab.UsingVarintBitcounts,
			},
			isS2:         s2,
			s1MaxEntries: cTab.s1MaxEntries,
		})
	}
}

func (p *parser) handleUpdateStringTableS2(tab *msgs2.CSVCMsg_UpdateStringTable) {
	p.handleUpdateStringTable(tab, true)
}

func (p *parser) handleCreateStringTable(tab createStringTable) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	switch tab.GetName() {
	case stNameUserInfo:
		fallthrough
	case stNameModelPreCache:
		fallthrough
	case stNameInstanceBaseline:
		p.processStringTable(tab)
	}

	p.stringTables = append(p.stringTables, tab)

	p.eventDispatcher.Dispatch(events.StringTableCreated{TableName: tab.GetName()})
}

func (p *parser) handleCreateStringTableS2(tab *msgs2.CSVCMsg_CreateStringTable) {
	p.handleCreateStringTable(createStringTable{
		CSVCMsg_CreateStringTable: tab,
		isS2:                      true,
	})
}

func (p *parser) processStringTableS1(tab createStringTable, br *bit.BitReader) {
	hist := make([]string, 0)
	idx := -1

	nEntryBits := int(math.Ceil(math.Log2(float64(*tab.s1MaxEntries))))

	for i := 0; i < int(tab.GetNumEntries()); i++ {
		if br.ReadBit() {
			idx++
		} else {
			idx = int(br.ReadInt(nEntryBits))
		}

		if idx < 0 || idx >= int(*tab.s1MaxEntries) {
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

			if p.header.ClientName == player.Name {
				p.recordingPlayerSlot = idx
				p.eventDispatcher.Dispatch(events.POVRecordingPlayerDetected{PlayerSlot: idx, PlayerInfo: player})
			}

			p.setRawPlayer(idx, player)

		case stNameInstanceBaseline:
			classID, err := strconv.Atoi(entry)
			if err != nil {
				panic(errors.Wrap(err, "failed to parse serverClassID"))
			}

			p.stParser.SetInstanceBaseline(classID, userdata)

		case stNameModelPreCache:
			p.modelPreCache[idx] = entry
		}
	}
}

// Holds and maintains a single entry in a string table.
type stringTableItem struct {
	Index int32
	Key   string
	Value []byte
}

const (
	stringtableKeyHistorySize = 32
)

// Parse a string table data blob, returning a list of item updates.
func parseStringTable(
	buf []byte,
	numUpdates int32,
	name string,
	userDataFixed bool,
	userDataSize int32,
	flags int32,
	variantBitCount bool) (items []*stringTableItem) {
	items = make([]*stringTableItem, 0)
	// Some tables have no data
	if len(buf) == 0 {
		return items
	}

	// Create a reader for the buffer
	r := bit.NewSmallBitReader(bytes.NewReader(buf))

	// Start with an index of -1.
	// If the first item is at index 0 it will use a incr operation.
	index := int32(-1)
	keys := make([]string, 0, stringtableKeyHistorySize+1)

	// Loop through entries in the data structure
	//
	// Each entry is a tuple consisting of {index, missing m_iItemDefinitionIndex property key, value}
	//
	// Index can either be incremented from the previous position or
	// overwritten with a given entry.
	//
	// Key may be omitted (will be represented here as "")
	//
	// Value may be omitted
	for i := 0; i < int(numUpdates); i++ {
		key := ""
		var value []byte

		// Read a boolean to determine whether the operation is an increment or
		// has a fixed index position. A fixed index position of zero should be
		// the last data in the buffer, and indicates that all data has been read.
		incr := r.ReadBit()
		if incr {
			index++
		} else {
			index = int32(r.ReadVarInt32()) + 1
		}

		// Some values have keys, some don't.
		hasKey := r.ReadBit()
		if hasKey {
			// Some entries use reference a position in the key history for
			// part of the key. If referencing the history, read the position
			// and size from the buffer, then use those to build the string
			// combined with an extra string read (null terminated).
			// Alternatively, just read the string.
			useHistory := r.ReadBit()
			if useHistory {
				pos := r.ReadInt(5)
				size := r.ReadInt(5)

				if int(pos) >= len(keys) {
					key += r.ReadString()
				} else {
					s := keys[pos]
					if int(size) > len(s) {
						key += s + r.ReadString()
					} else {
						key += s[0:size] + r.ReadString()
					}
				}
			} else {
				key = r.ReadString()
			}

			keys = append(keys, key)

			if len(keys) > stringtableKeyHistorySize {
				keys = keys[1:]
			}

			// Some entries have a value.
			hasValue := r.ReadBit()
			if hasValue {
				bitSize := uint(0)
				isCompressed := false

				if userDataFixed {
					bitSize = uint(userDataSize)
				} else {
					if (flags & 0x1) != 0 {
						isCompressed = r.ReadBit()
					}

					if variantBitCount {
						bitSize = r.ReadUBitInt() * 8
					} else {
						bitSize = r.ReadInt(17) * 8
					}
				}

				value = r.ReadBits(int(bitSize))

				if isCompressed {
					tmp, err := snappy.Decode(nil, value)
					if err != nil {
						panic(fmt.Sprintf("unable to decode snappy compressed stringtable item (%s, %d, %s): %s", name, index, key, err))
					}

					value = tmp
				}
			}

			items = append(items, &stringTableItem{index, key, value})
		}
	}

	return items
}

var instanceBaselineKeyRegex = regexp.MustCompile(`^\d+:\d+$`)

func (p *parser) processStringTableS2(tab createStringTable) {
	items := parseStringTable(tab.StringData, tab.GetNumEntries(), tab.GetName(), tab.GetUserDataFixedSize(), tab.GetUserDataSize(), tab.GetFlags(), tab.GetUsingVarintBitcounts())

	for _, item := range items {
		switch tab.GetName() {
		case stNameInstanceBaseline:
			if item.Key == "" || instanceBaselineKeyRegex.MatchString(item.Key) {
				continue
			}

			classID, err := strconv.Atoi(item.Key)
			if err != nil {
				panic(errors.Wrap(err, "failed to parse serverClassID"))
			}

			p.stParser.SetInstanceBaseline(classID, item.Value)
		case stNameUserInfo:
			p.parseUserInfo(item.Value, int(item.Index))
		}
	}
}

func (p *parser) processStringTable(tab createStringTable) {
	if tab.GetName() == stNameModelPreCache {
		for i := len(p.modelPreCache); i < int(tab.GetNumEntries()); i++ {
			p.modelPreCache = append(p.modelPreCache, "")
		}
	}

	if tab.GetDataCompressed() {
		tmp := make([]byte, tab.GetUncompressedSize())

		b, err := snappy.Decode(tmp, tab.StringData)
		if err != nil {
			panic(err)
		}

		tab.StringData = b
	}

	if tab.isS2 {
		p.processStringTableS2(tab)
	} else {
		br := bit.NewSmallBitReader(bytes.NewReader(tab.StringData))

		if br.ReadBit() {
			panic("unknown stringtable format")
		}

		p.processStringTableS1(tab, br)

		p.poolBitReader(br)
	}

	if tab.GetName() == stNameModelPreCache {
		p.processModelPreCacheUpdate()
	}
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

// Manta says:
// These appear to be periodic state dumps and appear every 1800 outer ticks.
// XXX TODO: decide if we want to at all integrate these updates,
// or trust create/update entirely. Let's ignore them for now.
func (p *parser) handleStringTables(msg *msgs2.CDemoStringTables) {
	p.msgDispatcher.SyncAllQueues()

	for _, tab := range msg.GetTables() {
		if tab.GetTableName() == stNameInstanceBaseline {
			for _, item := range tab.GetItems() {
				key := item.GetStr()

				if instanceBaselineKeyRegex.MatchString(key) {
					continue
				}

				classID, err := strconv.Atoi(key)
				if err != nil {
					panic(errors.Wrap(err, "failed to parse serverClassID"))
				}

				p.stParser.SetInstanceBaseline(classID, item.GetData())
			}
		} else if tab.GetTableName() == stNameUserInfo {
			for _, item := range tab.GetItems() {
				playerIndex, err := strconv.Atoi(item.GetStr())
				if err != nil {
					panic(errors.Wrap(err, "failed to parse playerIndex"))
				}

				p.parseUserInfo(item.GetData(), playerIndex)
			}
		}
	}
}

func (p *parser) handleUpdateStringTableS1(tab *msg.CSVCMsg_UpdateStringTable) {
	p.handleUpdateStringTable(&msgs2.CSVCMsg_UpdateStringTable{
		TableId:           tab.TableId,
		NumChangedEntries: tab.NumChangedEntries,
		StringData:        tab.StringData,
	}, false)
}

func (p *parser) handleCreateStringTableS1(tab *msg.CSVCMsg_CreateStringTable) {
	size := int32(len(tab.StringData))
	compressed := false

	p.handleCreateStringTable(createStringTable{
		CSVCMsg_CreateStringTable: &msgs2.CSVCMsg_CreateStringTable{
			Name:              tab.Name,
			NumEntries:        tab.NumEntries,
			UserDataFixedSize: tab.UserDataFixedSize,
			UserDataSize:      tab.UserDataSize,
			UserDataSizeBits:  tab.UserDataSizeBits,
			Flags:             tab.Flags,
			StringData:        tab.StringData,
			UncompressedSize:  &size,
			DataCompressed:    &compressed,
		},
		s1MaxEntries: tab.MaxEntries,
	})
}

func (p *parser) parseUserInfo(data []byte, playerIndex int) {
	if _, exists := p.rawPlayers[playerIndex]; exists {
		return
	}

	var userInfo msgs2.CMsgPlayerInfo
	err := proto.Unmarshal(data, &userInfo)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse CMsgPlayerInfo msg"))
	}

	xuid := userInfo.GetXuid() // TODO: what to do with userInfo.GetSteamid()? (seems to be the same, but maybe not in China?)
	name := userInfo.GetName()
	// When the userinfo ST is created its data contains 1 message for each possible player slot (up to 64).
	// We ignore messages that are empty, i.e. not related to a real player, a BOT or GOTV.
	if xuid == 0 && name == "" {
		return
	}

	playerInfo := common.PlayerInfo{
		XUID:         xuid,
		Name:         name,
		UserID:       int(userInfo.GetUserid()),
		IsFakePlayer: userInfo.GetFakeplayer(),
		IsHltv:       userInfo.GetIshltv(),
		// Fields not available with CS2 demos
		Version:         0,
		GUID:            "",
		FriendsID:       0,
		FriendsName:     "",
		CustomFiles0:    0,
		CustomFiles1:    0,
		CustomFiles2:    0,
		CustomFiles3:    0,
		FilesDownloaded: 0,
	}
	p.setRawPlayer(playerIndex, playerInfo)

	povDemoDetected := p.recordingPlayerSlot == -1 && p.header.ClientName == playerInfo.Name
	if povDemoDetected {
		p.recordingPlayerSlot = playerIndex
		p.eventDispatcher.Dispatch(events.POVRecordingPlayerDetected{PlayerSlot: playerIndex, PlayerInfo: playerInfo})
	}
}
