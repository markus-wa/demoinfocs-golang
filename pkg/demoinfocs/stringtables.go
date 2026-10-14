package demoinfocs

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/snappy"
	"google.golang.org/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/v6/internal/bitread"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

const (
	stNameInstanceBaseline = "instancebaseline"
	stNameUserInfo         = "userinfo"
	stNameModelPreCache    = "modelprecache"
)

// updatePlayerFromRawIfExists applies raw player info (userinfo string table) to the player bound to
// the slot's controller, if the info belongs to it. Raw info only maps the player to its user-ID -
// name and SteamID64 come from the controller (see getOrCreatePlayerFromControllerEntity).
func (p *Parser) updatePlayerFromRawIfExists(index int) {
	pl := p.gameState.playersByEntityID[index+1]
	if pl == nil || pl.Entity == nil {
		return
	}

	// The info may belong to a different person than the one currently bound to the slot's
	// controller, e.g. the GOTV proxy's state dumps contain the next occupant of a slot before
	// they connect (or the previous one after a reconnect).
	if p.matchRawPlayerToController(index, pl.IsBot, pl.Entity) == nil {
		return
	}

	pl = p.getOrCreatePlayerFromControllerEntity(pl.Entity)

	p.eventDispatcher.Dispatch(events.StringTablePlayerUpdateApplied{
		Player: pl,
	})
}

// setRawPlayer stores raw player info for a slot (or a legacy user-ID, see playerConnect) and
// dispatches PlayerInfo - only if the info actually changed, as the GOTV proxy & periodic
// CDemoStringTables dumps re-send identical info.
func (p *Parser) setRawPlayer(index int, player common.PlayerInfo) {
	existing := p.rawPlayers[index]

	if player.UserID == math.MaxUint16 && existing != nil {
		player.UserID = existing.UserID
	}

	if existing != nil && *existing == player {
		return
	}

	p.rawPlayers[index] = &player

	p.updatePlayerFromRawIfExists(index)

	p.eventDispatcher.Dispatch(events.PlayerInfo{
		Index: index,
		Info:  player,
	})
}

func (p *Parser) handleUpdateStringTable(tab *msg.CSVCMsg_UpdateStringTable) {
	if p.aborted() {
		return
	}

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
		p.processStringTable(&msg.CSVCMsg_CreateStringTable{
			Name:                 cTab.Name,
			NumEntries:           tab.NumChangedEntries,
			UserDataFixedSize:    cTab.UserDataFixedSize,
			UserDataSize:         cTab.UserDataSize,
			UserDataSizeBits:     cTab.UserDataSizeBits,
			Flags:                cTab.Flags,
			StringData:           tab.StringData,
			UsingVarintBitcounts: cTab.UsingVarintBitcounts,
		})
	}
}

func (p *Parser) handleCreateStringTable(tab *msg.CSVCMsg_CreateStringTable) {
	if p.aborted() {
		return
	}

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
//
//nolint:funlen,gocognit
func (p *Parser) parseStringTable(
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

	// The items parsed before a failure are still applied.
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		// e.g. a ParserWarn handler, see poolBitReader()
		if classifyPanic(r) == panics.KindUser {
			panic(panics.Capture(r))
		}

		err := newDecodeError(panics.NewFailure(r), p.currentFrame)

		p.eventDispatcher.Dispatch(events.ParserWarn{
			Type:    events.WarnTypeStringTableParsingFailure,
			Message: fmt.Sprintf("failed to parse stringtable %q properly: %v", name, err.Cause),
			Err:     err,
		})
	}()

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
		// skips ahead. Skips are relative to the previous index: the varint is the
		// number of entries skipped minus one (i.e. index += varint + 2).
		// Treating it as an absolute index put userinfo updates into the wrong
		// player slots in demos that skip entries (e.g. Esportal GOTV).
		incr := r.ReadBit()
		if incr {
			index++
		} else {
			index += int32(r.ReadVarInt32()) + 2
		}

		// Some values have keys, some don't.
		hasKey := r.ReadBit()
		if hasKey { //nolint:nestif
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
		}

		// Some entries have a value.
		hasValue := r.ReadBit()
		//nolint:nestif
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

	p.poolBitReader(r)

	return items
}

var instanceBaselineKeyRegex = regexp.MustCompile(`^\d+:\d+$`)

func (p *Parser) processStringTable(tab *msg.CSVCMsg_CreateStringTable) {
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

	items := p.parseStringTable(tab.StringData, tab.GetNumEntries(), tab.GetName(), tab.GetUserDataFixedSize(), tab.GetUserDataSize(), tab.GetFlags(), tab.GetUsingVarintBitcounts())

	for _, item := range items {
		switch tab.GetName() {
		case stNameInstanceBaseline:
			if item.Key == "" || instanceBaselineKeyRegex.MatchString(item.Key) {
				continue
			}

			classID, err := strconv.Atoi(item.Key)
			if err != nil {
				panic(fmt.Errorf("failed to parse serverClassID: %w", err))
			}

			p.stParser.SetInstanceBaseline(classID, item.Value)
		case stNameUserInfo:
			p.parseUserInfo(item.Value, int(item.Index))
		}
	}

	if tab.GetName() == stNameModelPreCache {
		p.processModelPreCacheUpdate()
	}
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

func (p *Parser) processModelPreCacheUpdate() {
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
func (p *Parser) handleStringTables(msg *msg.CDemoStringTables) {
	if p.aborted() {
		return
	}

	for _, tab := range msg.GetTables() {
		if tab.GetTableName() == stNameInstanceBaseline {
			for _, item := range tab.GetItems() {
				key := item.GetStr()

				if instanceBaselineKeyRegex.MatchString(key) {
					continue
				}

				classID, err := strconv.Atoi(key)
				if err != nil {
					panic(fmt.Errorf("failed to parse serverClassID: %w", err))
				}

				p.stParser.SetInstanceBaseline(classID, item.GetData())
			}
		} else if tab.GetTableName() == stNameUserInfo {
			for _, item := range tab.GetItems() {
				playerIndex, err := strconv.Atoi(item.GetStr())
				if err != nil {
					panic(fmt.Errorf("failed to parse playerIndex: %w", err))
				}

				p.parseUserInfo(item.GetData(), playerIndex)
			}
		}
	}
}

// parseUserInfo applies a userinfo string-table entry for the given player slot.
//
// Entries are applied even for slots we already have info for (needed to fix up user-ID mappings
// when parsing broadcasts from mid-match, see #655). Stale entries - the GOTV proxy keeps the
// pre-reconnect info for re-used slots (#692) - can't affect players' names or SteamIDs, as those
// come from the player controller.
func (p *Parser) parseUserInfo(data []byte, playerIndex int) {
	var userInfo msg.CMsgPlayerInfo

	err := proto.Unmarshal(data, &userInfo)
	if err != nil {
		panic(fmt.Errorf("failed to parse CMsgPlayerInfo msg: %w", err))
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
