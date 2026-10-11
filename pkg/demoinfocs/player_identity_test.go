package demoinfocs

import (
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

// set updates the given props and then fires their update handlers (in the given order).
func (c *fakeController) set(kv ...any) {
	for i := 0; i < len(kv); i += 2 {
		c.props[kv[i].(string)] = kv[i+1]
	}

	for i := 0; i < len(kv); i += 2 {
		name := kv[i].(string)
		for _, h := range c.handlers[name] {
			h(st.PropertyValue{Any: c.props[name]})
		}
	}
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
	infos       []events.PlayerInfo
	stApplied   []events.StringTablePlayerUpdateApplied
}

func newIdentityTestParser(t *testing.T) (*parser, *identityEvents) {
	t.Helper()

	p := newParser()
	evs := new(identityEvents)

	p.RegisterEventHandler(func(e events.PlayerNameChange) { evs.nameChanges = append(evs.nameChanges, e) })
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
	p.bindNewPlayerController(ctrl)

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
	p.bindNewPlayerController(ctrl)

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
	p.bindNewPlayerController(ctrl)
	pl := p.gameState.playersByEntityID[1]

	ctrl.set("m_iszPlayerName", "new")

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
	p.bindNewPlayerController(ctrl)

	ctrl.set("m_iszPlayerName", "name")
	ctrl.set("m_steamID", testSteamIDA)
	ctrl.set("m_iConnected", uint32(0))

	assert.Equal(t, "name", p.gameState.playersByEntityID[1].Name)
	assert.Empty(t, evs.nameChanges)
}

func TestPlayerIdentity_BotRenameDoesNotDispatchNameChange(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	p.setRawPlayer(0, common.PlayerInfo{Name: "Bot A", UserID: 0, IsFakePlayer: true})

	ctrl := newFakeController(1, 0, "Bot A")
	p.bindNewPlayerController(ctrl)
	bot := p.gameState.playersByEntityID[1]
	require.True(t, bot.IsBot)

	ctrl.set("m_iszPlayerName", "Bot B")

	assert.Same(t, bot, p.gameState.playersByEntityID[1])
	assert.Equal(t, "Bot B", bot.Name)
	assert.Empty(t, evs.nameChanges)
}

func TestPlayerIdentity_NewPlayerWhenDifferentPersonTakesOverSlot(t *testing.T) {
	t.Run("controller re-created", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})
		p.bindNewPlayerController(newFakeController(5, testSteamIDA, "A"))
		a := p.gameState.playersByEntityID[5]

		// the GOTV proxy's state dump may contain the slot's next occupant before they connect
		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDB, Name: "B", UserID: 4})

		assert.Same(t, a, p.gameState.playersByEntityID[5])
		assert.Equal(t, "A", a.Name)
		assert.Equal(t, testSteamIDA, a.SteamID64)

		// B connects - the controller entity is re-created for the slot
		p.bindNewPlayerController(newFakeController(5, testSteamIDB, "B"))
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
		p.bindNewPlayerController(ctrl)
		a := p.gameState.playersByEntityID[5]

		ctrl.set("m_steamID", testSteamIDB, "m_iszPlayerName", "B")
		b := p.gameState.playersByEntityID[5]

		assert.NotSame(t, a, b)
		assert.Equal(t, "B", b.Name)
		assert.Equal(t, testSteamIDB, b.SteamID64)
		assert.Equal(t, "A", a.Name)
		assert.Equal(t, testSteamIDA, a.SteamID64)
		assert.Empty(t, evs.nameChanges)

		// controller prop updates now apply to the new player
		ctrl.set("m_iTeamNum", uint64(common.TeamCounterTerrorists))
		assert.Equal(t, common.TeamCounterTerrorists, b.Team)
		assert.Equal(t, common.TeamTerrorists, a.Team)
	})

	t.Run("same person reconnecting keeps the player", func(t *testing.T) {
		p, evs := newIdentityTestParser(t)

		p.setRawPlayer(4, common.PlayerInfo{XUID: testSteamIDA, Name: "A", UserID: 4})
		p.bindNewPlayerController(newFakeController(5, testSteamIDA, "A"))
		a := p.gameState.playersByEntityID[5]

		p.bindNewPlayerController(newFakeController(5, testSteamIDA, "A2"))

		assert.Same(t, a, p.gameState.playersByEntityID[5])
		assert.Equal(t, "A2", a.Name)
		assert.Equal(t, []events.PlayerNameChange{{Player: a, OldName: "A", NewName: "A2"}}, evs.nameChanges)
	})
}

func TestPlayerIdentity_PlayerInfoOnlyOnChange(t *testing.T) {
	p, evs := newIdentityTestParser(t)

	p.bindNewPlayerController(newFakeController(1, testSteamIDA, "name"))

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
