package demoinfocs

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// fakeController is a minimal CCSPlayerController entity whose props can be changed,
// firing the registered update handlers like the real entity does.
type fakeController struct {
	st.Entity // unimplemented methods panic

	id        int
	props     map[string]any
	handlers  map[string][]st.PropertyUpdateHandler
	onDestroy []func()
	onUpdated []func()
}

func newFakeController(id int, steamID uint64, name string) *fakeController {
	return &fakeController{
		id: id,
		props: map[string]any{
			"m_steamID":       steamID,
			"m_iszPlayerName": name,
			"m_iConnected":    uint32(0),
			"m_iTeamNum":      uint64(common.TeamTerrorists),
		},
		handlers: make(map[string][]st.PropertyUpdateHandler),
	}
}

func (c *fakeController) ID() int {
	return c.id
}

func (c *fakeController) PropertyValueMust(name string) st.PropertyValue {
	return st.PropertyValue{Any: c.props[name]}
}

func (c *fakeController) PropertyValue(name string) (st.PropertyValue, bool) {
	v, ok := c.props[name]

	return st.PropertyValue{Any: v}, ok
}

func (c *fakeController) Property(name string) st.Property {
	return fakeControllerProperty{c: c, name: name}
}

func (c *fakeController) OnDestroy(delegate func()) {
	c.onDestroy = append(c.onDestroy, delegate)
}

func (c *fakeController) OnUpdateFinished(delegate func()) {
	c.onUpdated = append(c.onUpdated, delegate)
}

func (c *fakeController) fireUpdateFinished() {
	for _, f := range c.onUpdated {
		f()
	}
}

// controllerFieldOrder is the field-path order of the controller props used here - the order in which
// the real entity applies them & fires their update handlers within one entity packet.
var controllerFieldOrder = []string{"m_iTeamNum", "m_iConnected", "m_iszPlayerName", "m_steamID"}

// update applies one entity packet's worth of prop changes like the real entity does: each prop is set
// and its handlers fired before the next prop is read (in field-path order, regardless of the order
// given), then the entity's update-finished delegates fire and the parser finishes the packet.
func (c *fakeController) update(p *parser, kv ...any) {
	changed := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		changed[kv[i].(string)] = kv[i+1]
	}

	for _, name := range controllerFieldOrder {
		v, ok := changed[name]
		if !ok {
			continue
		}

		c.props[name] = v

		for _, h := range c.handlers[name] {
			h(st.PropertyValue{Any: v})
		}
	}

	c.fireUpdateFinished()
	p.flushPlayerControllerUpdates()
}

func (c *fakeController) destroy() {
	for _, f := range c.onDestroy {
		f()
	}
}

// createController binds a new controller like an entity packet creating it does: created-handlers,
// then the initial firing of all update handlers and the update-finished delegates, then the end of the packet.
func createController(p *parser, c *fakeController) {
	p.bindNewPlayerController(c)

	for _, name := range controllerFieldOrder {
		for _, h := range c.handlers[name] {
			h(st.PropertyValue{Any: c.props[name]})
		}
	}

	c.fireUpdateFinished()
	p.flushPlayerControllerUpdates()
}

type fakeControllerProperty struct {
	st.Property // unimplemented methods panic

	c    *fakeController
	name string
}

func (p fakeControllerProperty) Value() st.PropertyValue {
	return st.PropertyValue{Any: p.c.props[p.name]}
}

func (p fakeControllerProperty) OnUpdate(handler st.PropertyUpdateHandler) {
	p.c.handlers[p.name] = append(p.c.handlers[p.name], handler)
}

type identityEvents struct {
	nameChanges []events.PlayerNameChange
	connects    []events.PlayerConnect
	disconnects []events.PlayerDisconnected
	infos       []events.PlayerInfo
	stApplied   []events.StringTablePlayerUpdateApplied
}

func newIdentityTestParser(t *testing.T) (*parser, *identityEvents) {
	t.Helper()

	p := newParser()
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

	ctrl := newFakeController(4, testSteamIDA, "FACEIT name")
	createController(p, ctrl)

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
	ctrl := newFakeController(1, testSteamIDA, "name")
	createController(p, ctrl)

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

	ctrl := newFakeController(1, testSteamIDA, "old")
	createController(p, ctrl)
	pl := p.gameState.playersByEntityID[1]

	ctrl.update(p, "m_iszPlayerName", "new")

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

	ctrl := newFakeController(1, testSteamIDA, "name")
	createController(p, ctrl)

	ctrl.update(p, "m_iszPlayerName", "name")
	ctrl.update(p, "m_steamID", testSteamIDA)
	ctrl.update(p, "m_iConnected", uint32(0))

	assert.Equal(t, "name", p.gameState.playersByEntityID[1].Name)
	assert.Empty(t, evs.nameChanges)
}

func TestPlayerIdentity_BotRenameDoesNotDispatchNameChange(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	p.setRawPlayer(0, common.PlayerInfo{Name: "Bot A", UserID: 0, IsFakePlayer: true})

	ctrl := newFakeController(1, 0, "Bot A")
	createController(p, ctrl)
	bot := p.gameState.playersByEntityID[1]
	require.True(t, bot.IsBot)

	ctrl.update(p, "m_iszPlayerName", "Bot B")

	assert.Same(t, bot, p.gameState.playersByEntityID[1])
	assert.Equal(t, "Bot B", bot.Name)
	assert.Empty(t, evs.nameChanges)
}

func TestPlayerIdentity_NewPlayerWhenDifferentPersonTakesOverSlot(t *testing.T) {
	t.Run("controller re-created", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})
		createController(p, newFakeController(5, testSteamIDA, "A"))
		a := p.gameState.playersByEntityID[5]

		// the GOTV proxy's state dump may contain the slot's next occupant before they connect
		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDB, Name: "B", UserID: 4})

		assert.Same(t, a, p.gameState.playersByEntityID[5])
		assert.Equal(t, "A", a.Name)
		assert.Equal(t, testSteamIDA, a.SteamID64)

		// B connects - the controller entity is re-created for the slot
		createController(p, newFakeController(5, testSteamIDB, "B"))
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

		ctrl := newFakeController(5, testSteamIDA, "A")
		createController(p, ctrl)
		a := p.gameState.playersByEntityID[5]

		ctrl.update(p, "m_steamID", testSteamIDB, "m_iszPlayerName", "B")
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
		ctrl.update(p, "m_iTeamNum", uint64(common.TeamCounterTerrorists))
		assert.Equal(t, common.TeamCounterTerrorists, b.Team)
		assert.Equal(t, common.TeamTerrorists, a.Team)
	})

	t.Run("same person reconnecting keeps the player", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})
		createController(p, newFakeController(5, testSteamIDA, "A"))
		a := p.gameState.playersByEntityID[5]

		createController(p, newFakeController(5, testSteamIDA, "A2"))

		assert.Same(t, a, p.gameState.playersByEntityID[5])
		assert.Equal(t, "A2", a.Name)
		assert.Equal(t, []events.PlayerNameChange{{Player: a, OldName: "A", NewName: "A2"}}, evs.nameChanges)
	})
}

func TestPlayerIdentity_PlayerInfoOnlyOnChange(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	createController(p, newFakeController(1, testSteamIDA, "name"))

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

	ctrl := newFakeController(5, testSteamIDA, "A")
	ctrl.props["m_iConnected"] = uint32(2) // connecting
	createController(p, ctrl)
	a := p.gameState.playersByEntityID[5]
	require.False(t, a.IsConnected)

	ctrl.update(p, "m_iConnected", uint32(0))
	require.True(t, a.IsConnected)
	require.Equal(t, []events.PlayerConnect{{Player: a}}, evs.connects)

	// B takes over the slot while A's connection state is re-sent in the same packet
	p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDB, Name: "B", UserID: 4})
	ctrl.update(p, "m_iConnected", uint32(0), "m_iszPlayerName", "B", "m_steamID", testSteamIDB)

	b := p.gameState.playersByEntityID[5]
	require.NotSame(t, a, b)

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

	ctrl := newFakeController(1, testSteamIDA, "")
	createController(p, ctrl)
	pl := p.gameState.playersByEntityID[1]

	assert.Equal(t, "unknown", pl.Name)
	assert.True(t, pl.IsUnknown)

	// the connection state alone doesn't name the player
	ctrl.update(p, "m_iConnected", uint32(0))

	assert.Equal(t, "unknown", pl.Name)
	assert.True(t, pl.IsUnknown, "must stay unknown until the player has a name")

	ctrl.update(p, "m_iszPlayerName", "real")

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

		ctrl := newFakeController(1, testSteamIDA, placeholder)
		createController(p, ctrl)
		pl := p.gameState.playersByEntityID[1]

		assert.Equal(t, "steam name", pl.Name)
		assert.False(t, pl.IsUnknown)

		ctrl.update(p, "m_iszPlayerName", "FACEIT name")

		assert.Same(t, pl, p.gameState.playersByEntityID[1])
		assert.Equal(t, "FACEIT name", pl.Name)
		assert.Empty(t, evs.nameChanges, "replacing a provisional name isn't a name change")

		// later renames are
		ctrl.update(p, "m_iszPlayerName", "renamed")

		assert.Equal(t, []events.PlayerNameChange{{Player: pl, OldName: "FACEIT name", NewName: "renamed"}}, evs.nameChanges)
	})

	t.Run("placeholder without userinfo", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		ctrl := newFakeController(1, testSteamIDA, placeholder)
		createController(p, ctrl)
		pl := p.gameState.playersByEntityID[1]

		assert.Equal(t, placeholder, pl.Name)

		ctrl.update(p, "m_iszPlayerName", "real")

		assert.Equal(t, "real", pl.Name)
		assert.Empty(t, evs.nameChanges)
	})

	t.Run("placeholder never replaces a real name", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		ctrl := newFakeController(1, testSteamIDA, "real")
		createController(p, ctrl)
		pl := p.gameState.playersByEntityID[1]

		ctrl.update(p, "m_iszPlayerName", placeholder)

		assert.Equal(t, "real", pl.Name)
		assert.Empty(t, evs.nameChanges)

		ctrl.update(p, "m_iszPlayerName", "real 2")

		assert.Equal(t, []events.PlayerNameChange{{Player: pl, OldName: "real", NewName: "real 2"}}, evs.nameChanges)
	})
}

func TestPlayerIdentity_ControllerDestroyedInSamePacket(t *testing.T) {
	p, _ := newIdentityTestParser(t)

	ctrl := newFakeController(1, testSteamIDA, "A")
	createController(p, ctrl)
	pl := p.gameState.playersByEntityID[1]

	// a prop update and the deletion in the same packet
	ctrl.props["m_iszPlayerName"] = "A2"
	for _, h := range ctrl.handlers["m_iszPlayerName"] {
		h(st.PropertyValue{Any: "A2"})
	}

	ctrl.destroy()
	p.flushPlayerControllerUpdates()

	assert.Nil(t, p.gameState.playersByEntityID[1], "a destroyed controller must not be re-bound")
	assert.False(t, pl.IsConnected)
	assert.Equal(t, "A", pl.Name)
}
