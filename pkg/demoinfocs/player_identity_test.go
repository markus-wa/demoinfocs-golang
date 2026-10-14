package demoinfocs

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// controllerFieldOrder is the field-path order of the controller props used here - the order in which
// the entity applies them & fires their update handlers within one entity packet.
var controllerFieldOrder = []string{"m_iTeamNum", "m_iConnected", "m_iszPlayerName", "m_steamID"}

// newFakeController returns a new CCSPlayerController entity (not created yet, see stfake.Create).
// Updates (stfake.Update) apply its props in controllerFieldOrder, then fire its update-finished delegates.
// The parser's end-of-packet backstop is deliberately not called, so the tests depend on the per-entity flush.
func newFakeController(p *Parser, id int, steamID uint64, name string) *st.Entity {
	values := map[string]any{
		"m_steamID":       steamID,
		"m_iszPlayerName": name,
		"m_iConnected":    uint32(0),
		"m_iTeamNum":      uint64(common.TeamTerrorists),
	}

	props := make([]stfake.Prop, 0, len(controllerFieldOrder))
	for _, prop := range controllerFieldOrder {
		props = append(props, stfake.P(prop, values[prop]))
	}

	return stfake.New(stfake.Spec{
		Parser: p.stParser,
		Class:  "CCSPlayerController",
		ID:     id,
		Props:  props,
	})
}

type identityEvents struct {
	nameChanges []events.PlayerNameChange
	connects    []events.PlayerConnect
	disconnects []events.PlayerDisconnected
	infos       []events.PlayerInfo
	stApplied   []events.StringTablePlayerUpdateApplied
}

func newIdentityTestParser(t *testing.T) (*Parser, *identityEvents) {
	t.Helper()

	p := newParser()
	p.stParser = stfake.NewParser("CCSPlayerController", "CCSPlayerPawn")
	p.bindPlayers()

	evs := new(identityEvents)

	p.RegisterEventHandler(func(e events.PlayerNameChange) { evs.nameChanges = append(evs.nameChanges, e) })
	p.RegisterEventHandler(func(e events.PlayerConnect) { evs.connects = append(evs.connects, e) })
	p.RegisterEventHandler(func(e events.PlayerDisconnected) { evs.disconnects = append(evs.disconnects, e) })
	p.RegisterEventHandler(func(e events.PlayerInfo) { evs.infos = append(evs.infos, e) })
	p.RegisterEventHandler(func(e events.StringTablePlayerUpdateApplied) { evs.stApplied = append(evs.stApplied, e) })

	return p, evs
}

const (
	testSteamIDA = uint64(76561198000000001)
	testSteamIDB = uint64(76561198000000002)
)

func TestPlayerIdentity_NameAndSteamIDFromControllerOnBind(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	// userinfo carries the Steam name, the controller the server-enforced one (e.g. FACEIT)
	p.setRawPlayer(3, common.PlayerInfo{XUID: testSteamIDA, Name: "steam name", UserID: 3})

	ctrl := newFakeController(p, 4, testSteamIDA, "FACEIT name")
	stfake.Create(ctrl)

	pl := p.gameState.playersByEntityID[4]
	require.NotNil(t, pl)
	assert.Equal(t, "FACEIT name", pl.Name)
	assert.Equal(t, testSteamIDA, pl.SteamID64)
	assert.Equal(t, 3, pl.UserID)
	assert.False(t, pl.IsBot)
	assert.False(t, pl.IsUnknown)
	assert.Same(t, pl, p.gameState.playersByUserID[3])
	assert.Same(t, pl, p.gameState.playersBySteamID32[common.ConvertSteamID64To32(testSteamIDA)])
	assert.Empty(t, evs.nameChanges)
}

func TestPlayerIdentity_NameFromControllerWithoutUserInfo(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	// no userinfo yet (e.g. at the start of some GOTV demos)
	ctrl := newFakeController(p, 1, testSteamIDA, "name")
	stfake.Create(ctrl)

	pl := p.gameState.playersByEntityID[1]
	require.NotNil(t, pl)
	assert.Equal(t, "name", pl.Name)
	assert.Equal(t, testSteamIDA, pl.SteamID64)
	assert.False(t, pl.IsUnknown)

	// userinfo arriving later maps the player to its user-ID without renaming it
	p.setRawPlayer(0, common.PlayerInfo{XUID: testSteamIDA, Name: "steam name", UserID: 0})

	assert.Same(t, pl, p.gameState.playersByEntityID[1])
	assert.Same(t, pl, p.gameState.playersByUserID[0])
	assert.Equal(t, 0, pl.UserID)
	assert.Equal(t, "name", pl.Name)
	assert.Empty(t, evs.nameChanges)
	require.Len(t, evs.stApplied, 1)
	assert.Same(t, pl, evs.stApplied[0].Player)
}

func TestPlayerIdentity_RenameOnControllerNameUpdate(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	p.setRawPlayer(0, common.PlayerInfo{XUID: testSteamIDA, Name: "old", UserID: 0})

	ctrl := newFakeController(p, 1, testSteamIDA, "old")
	stfake.Create(ctrl)
	pl := p.gameState.playersByEntityID[1]

	stfake.Update(ctrl, stfake.P("m_iszPlayerName", "new"))

	assert.Same(t, pl, p.gameState.playersByEntityID[1])
	assert.Equal(t, "new", pl.Name)
	assert.Equal(t, testSteamIDA, pl.SteamID64)
	assert.Equal(t, []events.PlayerNameChange{{Player: pl, OldName: "old", NewName: "new"}}, evs.nameChanges)

	// stale userinfo with the old name must not revert it
	p.setRawPlayer(0, common.PlayerInfo{XUID: testSteamIDA, Name: "stale", UserID: 0})

	assert.Equal(t, "new", pl.Name)
	assert.Len(t, evs.nameChanges, 1)
}

func TestPlayerIdentity_NoNameChangeEventWhenUnchanged(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	p.setRawPlayer(0, common.PlayerInfo{XUID: testSteamIDA, Name: "name", UserID: 0})

	ctrl := newFakeController(p, 1, testSteamIDA, "name")
	stfake.Create(ctrl)

	stfake.Update(ctrl, stfake.P("m_iszPlayerName", "name"))
	stfake.Update(ctrl, stfake.P("m_steamID", testSteamIDA))
	stfake.Update(ctrl, stfake.P("m_iConnected", uint32(0)))

	assert.Equal(t, "name", p.gameState.playersByEntityID[1].Name)
	assert.Empty(t, evs.nameChanges)
}

func TestPlayerIdentity_BotRenameDoesNotDispatchNameChange(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	p.setRawPlayer(0, common.PlayerInfo{Name: "Bot A", UserID: 0, IsFakePlayer: true})

	ctrl := newFakeController(p, 1, 0, "Bot A")
	stfake.Create(ctrl)
	bot := p.gameState.playersByEntityID[1]
	require.True(t, bot.IsBot)

	stfake.Update(ctrl, stfake.P("m_iszPlayerName", "Bot B"))

	assert.Same(t, bot, p.gameState.playersByEntityID[1])
	assert.Equal(t, "Bot B", bot.Name)
	assert.Empty(t, evs.nameChanges)
}

func TestPlayerIdentity_NewPlayerWhenDifferentPersonTakesOverSlot(t *testing.T) {
	t.Run("controller re-created", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})
		stfake.Create(newFakeController(p, 5, testSteamIDA, "A"))
		a := p.gameState.playersByEntityID[5]

		// the GOTV proxy's state dump may contain the slot's next occupant before they connect
		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDB, Name: "B", UserID: 4})

		assert.Same(t, a, p.gameState.playersByEntityID[5])
		assert.Equal(t, "A", a.Name)
		assert.Equal(t, testSteamIDA, a.SteamID64)

		// B connects - the controller entity is re-created for the slot
		stfake.Create(newFakeController(p, 5, testSteamIDB, "B"))
		b := p.gameState.playersByEntityID[5]

		assert.NotSame(t, a, b)
		assert.Equal(t, "B", b.Name)
		assert.Equal(t, testSteamIDB, b.SteamID64)
		assert.Same(t, b, p.gameState.playersByUserID[4])

		assert.Equal(t, "A", a.Name, "the previous occupant must not be renamed")
		assert.Equal(t, testSteamIDA, a.SteamID64, "the previous occupant's SteamID64 must not change")
		assert.Empty(t, evs.nameChanges)
	})

	t.Run("controller updated in place", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})

		ctrl := newFakeController(p, 5, testSteamIDA, "A")
		stfake.Create(ctrl)
		a := p.gameState.playersByEntityID[5]

		stfake.Update(ctrl, stfake.P("m_steamID", testSteamIDB), stfake.P("m_iszPlayerName", "B"))
		b := p.gameState.playersByEntityID[5]

		assert.NotSame(t, a, b)
		assert.Equal(t, "B", b.Name)
		assert.Equal(t, testSteamIDB, b.SteamID64)
		assert.True(t, b.IsConnected)
		assert.Same(t, ctrl, b.Entity)
		assert.Equal(t, "A", a.Name)
		assert.Equal(t, testSteamIDA, a.SteamID64)
		assert.Empty(t, evs.nameChanges)

		// the previous occupant is disconnected & no longer reads the controller's (now B's) data
		assert.False(t, a.IsConnected)
		assert.Nil(t, a.Entity)
		assert.Equal(t, 5, a.EntityID)
		assert.Equal(t, []events.PlayerDisconnected{{Player: a}}, evs.disconnects)
		assert.Equal(t, []events.PlayerConnect{{Player: a}, {Player: b}}, evs.connects)

		// controller prop updates now apply to the new player
		stfake.Update(ctrl, stfake.P("m_iTeamNum", uint64(common.TeamCounterTerrorists)))
		assert.Equal(t, common.TeamCounterTerrorists, b.Team)
		assert.Equal(t, common.TeamTerrorists, a.Team)
	})

	t.Run("same person reconnecting keeps the player", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})
		stfake.Create(newFakeController(p, 5, testSteamIDA, "A"))
		a := p.gameState.playersByEntityID[5]

		stfake.Create(newFakeController(p, 5, testSteamIDA, "A2"))

		assert.Same(t, a, p.gameState.playersByEntityID[5])
		assert.Equal(t, "A2", a.Name)
		assert.Equal(t, []events.PlayerNameChange{{Player: a, OldName: "A", NewName: "A2"}}, evs.nameChanges)
	})
}

func TestPlayerIdentity_PlayerInfoOnlyOnChange(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	stfake.Create(newFakeController(p, 1, testSteamIDA, "name"))

	info := common.PlayerInfo{XUID: testSteamIDA, Name: "name", UserID: 0}

	p.setRawPlayer(0, info)
	p.setRawPlayer(0, info) // identical re-send (e.g. periodic CDemoStringTables dump)

	assert.Len(t, evs.infos, 1)
	assert.Len(t, evs.stApplied, 1)

	info.Name = "steam rename"
	p.setRawPlayer(0, info)

	assert.Len(t, evs.infos, 2)
	assert.Equal(t, info, evs.infos[1].Info)
	assert.Len(t, evs.stApplied, 2)
	assert.Equal(t, "name", p.gameState.playersByEntityID[1].Name, "userinfo must not rename the player")
	assert.Empty(t, evs.nameChanges)

	// user-ID 0xFFFF keeps the previous user-ID - still identical
	info.UserID = 0xFFFF
	p.setRawPlayer(0, info)

	assert.Len(t, evs.infos, 2)
}

// The real entity applies a packet's prop changes field by field (m_iConnected before m_iszPlayerName
// before m_steamID) and fires each prop's handlers right away. A take-over must only be resolved once
// the whole packet is applied, otherwise the previous occupant gets renamed / re-connected.
func TestPlayerIdentity_InPlaceTakeoverInFieldOrder(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})

	ctrl := newFakeController(p, 5, testSteamIDA, "A")
	stfake.Set(ctrl, stfake.P("m_iConnected", uint32(2))) // connecting
	stfake.Create(ctrl)
	a := p.gameState.playersByEntityID[5]
	require.False(t, a.IsConnected)

	stfake.Update(ctrl, stfake.P("m_iConnected", uint32(0)))
	require.True(t, a.IsConnected)
	require.Equal(t, []events.PlayerConnect{{Player: a}}, evs.connects)

	// the pawn's weapons, bound to A
	knife := common.NewEquipment(common.EqKnife, common.UniqueID(1))
	knife.Owner = a
	a.Inventory[100] = knife
	flash := common.NewEquipment(common.EqFlash, common.UniqueID(2))
	flash.Owner = a
	a.Inventory[101] = flash

	// B takes over the slot (on the other team) while A's connection state is re-sent in the same packet
	p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDB, Name: "B", UserID: 4})
	stfake.Update(ctrl, stfake.P("m_iTeamNum", uint64(common.TeamCounterTerrorists)), stfake.P("m_iConnected", uint32(0)),
		stfake.P("m_iszPlayerName", "B"), stfake.P("m_steamID", testSteamIDB))

	b := p.gameState.playersByEntityID[5]
	require.NotSame(t, a, b)
	assert.Empty(t, p.dirtyControllers)

	assert.Equal(t, common.TeamTerrorists, a.Team, "the previous occupant's team must not change")
	assert.Equal(t, common.TeamCounterTerrorists, b.Team)

	assert.Empty(t, a.Inventory)
	assert.Equal(t, map[int]*common.Equipment{100: knife, 101: flash}, b.Inventory, "the pawn's weapons belong to the new occupant")
	assert.Same(t, b, knife.Owner)
	assert.Same(t, flash, p.gameState.lastKnownGrenadeWeapon(b, common.EqFlash), "grenade fallback (#580) for the new occupant")

	assert.Equal(t, "A", a.Name, "the previous occupant must not be renamed")
	assert.Equal(t, testSteamIDA, a.SteamID64)
	assert.False(t, a.IsConnected)
	assert.Nil(t, a.Entity)

	assert.Equal(t, "B", b.Name)
	assert.Equal(t, testSteamIDB, b.SteamID64)
	assert.True(t, b.IsConnected)
	assert.Same(t, b, p.gameState.playersByUserID[4])

	assert.Empty(t, evs.nameChanges)
	assert.Equal(t, []events.PlayerConnect{{Player: a}, {Player: b}}, evs.connects)
	assert.Equal(t, []events.PlayerDisconnected{{Player: a}}, evs.disconnects)
}

func TestPlayerIdentity_UnknownUntilNamed(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	ctrl := newFakeController(p, 1, testSteamIDA, "")
	stfake.Create(ctrl)
	pl := p.gameState.playersByEntityID[1]

	assert.Equal(t, "unknown", pl.Name)
	assert.True(t, pl.IsUnknown)

	// the connection state alone doesn't name the player
	stfake.Update(ctrl, stfake.P("m_iConnected", uint32(0)))

	assert.Equal(t, "unknown", pl.Name)
	assert.True(t, pl.IsUnknown, "must stay unknown until the player has a name")

	stfake.Update(ctrl, stfake.P("m_iszPlayerName", "real"))

	assert.Same(t, pl, p.gameState.playersByEntityID[1])
	assert.Equal(t, "real", pl.Name)
	assert.False(t, pl.IsUnknown)
	assert.Empty(t, evs.nameChanges, "an unknown player getting its name isn't a name change")
}

// Some servers create the controller with the SteamID64 as a placeholder name (e.g. arsham_infernocs2_1stSept23.dem).
func TestPlayerIdentity_SteamIDPlaceholderName(t *testing.T) {
	placeholder := strconv.FormatUint(testSteamIDA, 10)

	t.Run("userinfo name until the controller is named", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(0, common.PlayerInfo{XUID: testSteamIDA, Name: "steam name", UserID: 0})

		ctrl := newFakeController(p, 1, testSteamIDA, placeholder)
		stfake.Create(ctrl)
		pl := p.gameState.playersByEntityID[1]

		assert.Equal(t, "steam name", pl.Name)
		assert.False(t, pl.IsUnknown)

		stfake.Update(ctrl, stfake.P("m_iszPlayerName", "FACEIT name"))

		assert.Same(t, pl, p.gameState.playersByEntityID[1])
		assert.Equal(t, "FACEIT name", pl.Name)
		assert.Empty(t, evs.nameChanges, "replacing a provisional name isn't a name change")

		// later renames are
		stfake.Update(ctrl, stfake.P("m_iszPlayerName", "renamed"))

		assert.Equal(t, []events.PlayerNameChange{{Player: pl, OldName: "FACEIT name", NewName: "renamed"}}, evs.nameChanges)
	})

	t.Run("placeholder without userinfo", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		ctrl := newFakeController(p, 1, testSteamIDA, placeholder)
		stfake.Create(ctrl)
		pl := p.gameState.playersByEntityID[1]

		assert.Equal(t, placeholder, pl.Name)

		stfake.Update(ctrl, stfake.P("m_iszPlayerName", "real"))

		assert.Equal(t, "real", pl.Name)
		assert.Empty(t, evs.nameChanges)
	})

	t.Run("placeholder never replaces a real name", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		ctrl := newFakeController(p, 1, testSteamIDA, "real")
		stfake.Create(ctrl)
		pl := p.gameState.playersByEntityID[1]

		stfake.Update(ctrl, stfake.P("m_iszPlayerName", placeholder))

		assert.Equal(t, "real", pl.Name)
		assert.Empty(t, evs.nameChanges)

		stfake.Update(ctrl, stfake.P("m_iszPlayerName", "real 2"))

		assert.Equal(t, []events.PlayerNameChange{{Player: pl, OldName: "real", NewName: "real 2"}}, evs.nameChanges)
	})
}

func TestPlayerIdentity_ControllerDestroyedInSamePacket(t *testing.T) {
	p, _ := newIdentityTestParser(t)

	ctrl := newFakeController(p, 1, testSteamIDA, "A")
	stfake.Create(ctrl)
	pl := p.gameState.playersByEntityID[1]

	// a prop update and the deletion in the same packet
	stfake.UpdateUnfinished(ctrl, stfake.P("m_iszPlayerName", "A2"))

	stfake.Destroy(ctrl)
	p.flushPlayerControllerUpdates()

	assert.Nil(t, p.gameState.playersByEntityID[1], "a destroyed controller must not be re-bound")
	assert.False(t, pl.IsConnected)
	assert.Equal(t, "A", pl.Name)
}

// The per-entity flush is the norm; the end-of-packet flush is a backstop for controllers whose
// update-finished delegates didn't run.
func TestPlayerIdentity_EndOfPacketBackstop(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	ctrl := newFakeController(p, 1, testSteamIDA, "A")
	stfake.Create(ctrl)
	pl := p.gameState.playersByEntityID[1]

	// prop handlers fire, but the update-finished delegates don't
	stfake.UpdateUnfinished(ctrl, stfake.P("m_iszPlayerName", "A2"))

	assert.Equal(t, "A", pl.Name, "applied only once the controller's updates are finished")

	p.flushPlayerControllerUpdates()

	assert.Equal(t, "A2", pl.Name)
	assert.Empty(t, p.dirtyControllers)
	assert.Len(t, evs.nameChanges, 1)
}
