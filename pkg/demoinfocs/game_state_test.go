package demoinfocs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/sendtables/fake"
)

func TestNewGameState(t *testing.T) {
	gs := newGameState(demoInfoProvider{})

	assert.NotNil(t, gs.playersByEntityID)
	assert.NotNil(t, gs.playersByUserID)
	assert.NotNil(t, gs.grenadeProjectiles)
	assert.NotNil(t, gs.infernos)
	assert.NotNil(t, gs.weapons)
	assert.NotNil(t, gs.hostages)
	assert.NotNil(t, gs.entities)
	assert.NotNil(t, gs.rules.conVars)
	assert.Equal(t, common.TeamTerrorists, gs.tState.Team())
	assert.Equal(t, common.TeamCounterTerrorists, gs.ctState.Team())
}

func TestNewGameState_TeamState_Pointers(t *testing.T) {
	gs := newGameState(demoInfoProvider{})

	assert.True(t, gs.TeamCounterTerrorists() == &gs.ctState)
	assert.True(t, gs.TeamTerrorists() == &gs.tState)
}

func TestNewGameState_TeamState_Opponent(t *testing.T) {
	gs := newGameState(demoInfoProvider{})

	assert.True(t, &gs.ctState == gs.tState.Opponent)
	assert.True(t, &gs.tState == gs.ctState.Opponent)
}

func TestGameState_Participants(t *testing.T) {
	gs := newGameState(demoInfoProvider{})
	ptcp := gs.Participants()
	byEntity := ptcp.ByEntityID()
	byUserID := ptcp.ByUserID()
	allByUserID := ptcp.AllByUserID()

	// Should update ptcp as well since it uses the same map
	gs.playersByEntityID[0] = newPlayer()
	gs.playersByUserID[0] = newPlayer()

	assert.Equal(t, gs.playersByEntityID, ptcp.ByEntityID())
	assert.Equal(t, gs.playersByUserID, ptcp.ByUserID())
	assert.Equal(t, gs.playersByUserID, ptcp.AllByUserID())

	// But should not update byEntity or byUserID since they're copies
	assert.NotEqual(t, byEntity, ptcp.ByEntityID())
	byUserID2 := ptcp.ByUserID()
	assert.NotEqual(t, byUserID, byUserID2)
	assert.Equal(t, gs.playersByUserID, ptcp.AllByUserID())

	gs.playersByEntityID[1] = newDisconnectedPlayer()
	gs.playersByUserID[1] = newDisconnectedPlayer()

	assert.Equal(t, gs.playersByUserID, ptcp.AllByUserID())

	assert.NotEqual(t, byEntity, ptcp.ByEntityID())
	// Should be equal since ByUserID() do not return disconnected users
	assert.Equal(t, byUserID2, ptcp.ByUserID())
	assert.NotEqual(t, allByUserID, ptcp.ByUserID())
}

func TestParticipants_All(t *testing.T) {
	pl := newPlayer()
	ptcps := participants{
		playersByUserID: map[int]*common.Player{0: pl},
	}

	allPlayers := ptcps.All()

	assert.ElementsMatch(t, []*common.Player{pl}, allPlayers)
}

func TestParticipants_Playing(t *testing.T) {
	terrorist := newPlayer()
	terrorist.Team = common.TeamTerrorists
	ct := newPlayer()
	ct.Team = common.TeamCounterTerrorists
	unassigned := newPlayer()
	unassigned.Team = common.TeamUnassigned
	spectator := newPlayer()
	spectator.Team = common.TeamSpectators
	def := newPlayer()

	ptcps := participants{
		playersByUserID: map[int]*common.Player{
			0: terrorist,
			1: ct,
			2: unassigned,
			3: spectator,
			4: def,
		},
	}

	playing := ptcps.Playing()

	assert.Len(t, playing, 2)
	assert.ElementsMatch(t, []*common.Player{terrorist, ct}, playing)
}

func TestParticipants_TeamMembers(t *testing.T) {
	terrorist := newPlayer()
	terrorist.Team = common.TeamTerrorists
	ct := newPlayer()
	ct.Team = common.TeamCounterTerrorists
	unassigned := newPlayer()
	unassigned.Team = common.TeamUnassigned
	spectator := newPlayer()
	spectator.Team = common.TeamSpectators
	def := newPlayer()

	ptcps := participants{
		playersByUserID: map[int]*common.Player{
			0: terrorist,
			1: ct,
			2: unassigned,
			3: spectator,
			4: def,
		},
	}

	cts := ptcps.TeamMembers(common.TeamCounterTerrorists)

	assert.ElementsMatch(t, []*common.Player{ct}, cts)
}

func TestParticipants_FindByHandle(t *testing.T) {
	pl := newPlayer()
	pl.Team = common.TeamTerrorists

	ptcps := participants{
		playersByEntityID: map[int]*common.Player{
			3000 & constants.EntityHandleIndexMask: pl,
		},
	}

	found := ptcps.FindByHandle(3000)

	assert.Equal(t, pl, found)
}

func TestParticipants_FindByHandle_InvalidEntityHandle(t *testing.T) {
	pl := newPlayer()
	pl.Team = common.TeamTerrorists
	ptcps := participants{
		playersByEntityID: map[int]*common.Player{
			constants.InvalidEntityHandle & constants.EntityHandleIndexMask: pl,
		},
	}

	found := ptcps.FindByHandle(constants.InvalidEntityHandle)

	assert.Nil(t, found)
}

func TestParticipants_Connected_SuppressNoEntity(t *testing.T) {
	pl := newPlayer()
	pl2 := common.NewPlayer(nil)
	pl2.IsConnected = true

	ptcps := participants{
		playersByUserID: map[int]*common.Player{
			0: pl,
			1: pl2,
		},
	}

	allPlayers := ptcps.Connected()

	assert.ElementsMatch(t, []*common.Player{pl}, allPlayers)
}

func TestParticipants_Connected_SuppressNotConnected(t *testing.T) {
	pl := newPlayer()
	pl2 := newPlayer()
	pl2.IsConnected = false

	ptcps := participants{
		playersByUserID: map[int]*common.Player{
			0: pl,
			1: pl2,
		},
	}

	allPlayers := ptcps.Connected()

	assert.ElementsMatch(t, []*common.Player{pl}, allPlayers)
}

func TestParticipants_SpottersOf(t *testing.T) {
	spotter1 := newPlayer()
	spotter1.EntityID = 1
	spotter2 := newPlayer()
	spotter2.EntityID = 35
	nonSpotter := newPlayer()
	nonSpotter.EntityID = 5

	spotted := newPlayer()
	entity := new(stfake.Entity)
	entity.On("ID").Return(3)
	prop0 := new(stfake.Property)
	prop0.On("Value").Return(st.PropertyValue{IntVal: 1})
	entity.On("Property", "m_bSpottedByMask.000").Return(prop0)
	prop1 := new(stfake.Property)
	prop1.On("Value").Return(st.PropertyValue{IntVal: 1 << 2})
	entity.On("Property", "m_bSpottedByMask.001").Return(prop1)
	spotted.Entity = entity

	ptcps := participants{
		playersByUserID: map[int]*common.Player{
			0: spotted,
			1: spotter1,
			2: spotter2,
			3: nonSpotter,
		},
	}

	spotters := ptcps.SpottersOf(spotted)

	assert.ElementsMatch(t, []*common.Player{spotter1, spotter2}, spotters)
}

func TestParticipants_SpottedBy(t *testing.T) {
	spotted1 := newPlayer()
	spotted1.EntityID = 2
	spotted2 := newPlayer()
	spotted2.EntityID = 35

	prop0 := new(stfake.Property)
	prop0.On("Value").Return(st.PropertyValue{IntVal: 1})
	spotted1Entity := new(stfake.Entity)
	spotted1Entity.On("Property", "m_bSpottedByMask.000").Return(prop0)
	spotted1.Entity = spotted1Entity
	spotted2Entity := new(stfake.Entity)
	spotted2Entity.On("ID").Return(35)
	spotted2Entity.On("Property", "m_bSpottedByMask.000").Return(prop0)
	spotted2.Entity = spotted2Entity

	unSpotted := newPlayer()
	unSpotted.EntityID = 5
	spotter := newPlayer()
	spotter.EntityID = 1

	unSpottedProp := new(stfake.Property)
	unSpottedProp.On("Value").Return(st.PropertyValue{IntVal: 0})
	unSpottedEntity := new(stfake.Entity)
	unSpottedEntity.On("Property", "m_bSpottedByMask.000").Return(unSpottedProp)
	unSpotted.Entity = unSpottedEntity

	spotterEntity := new(stfake.Entity)
	spotterEntity.On("Property", "m_bSpottedByMask.000").Return(unSpottedProp)
	spotter.Entity = spotterEntity

	ptcps := participants{
		playersByUserID: map[int]*common.Player{
			0: spotter,
			1: spotted1,
			2: spotted2,
			3: unSpotted,
		},
	}

	spotted := ptcps.SpottedBy(spotter)

	assert.ElementsMatch(t, []*common.Player{spotted1, spotted2}, spotted)
}

func TestGameRules_ConVars(t *testing.T) {
	cvars := make(map[string]string)
	gr := gameRules{conVars: cvars}

	assert.Equal(t, cvars, gr.ConVars())
}

func TestGameRules_Entity(t *testing.T) {
	ent := stfake.NewEntityWithProperty("m_iGameMode", st.PropertyValue{IntVal: 1})
	gr := gameRules{
		entity: ent,
	}

	assert.Equal(t, ent, gr.Entity())
}

func TestGameRules_BombTime(t *testing.T) {
	gs := gameRules{conVars: map[string]string{"mp_c4timer": "5"}}

	bt, err := gs.BombTime()

	assert.Nil(t, err)
	assert.Equal(t, 5*time.Second, bt)
}

func TestGameRules_FreezeTime(t *testing.T) {
	gs := gameRules{conVars: map[string]string{"mp_freezetime": "5"}}

	bt, err := gs.FreezeTime()

	assert.Nil(t, err)
	assert.Equal(t, 5*time.Second, bt)
}

func TestGameRules_RoundTime(t *testing.T) {
	prop := new(stfake.Property)
	prop.On("Value").Return(st.PropertyValue{IntVal: 115})
	ent := new(stfake.Entity)
	ent.On("Property", "cs_gamerules_data.m_iRoundTime").Return(prop)
	gr := gameRules{entity: ent}

	rt, err := gr.RoundTime()

	assert.Nil(t, err)
	assert.Equal(t, 115*time.Second, rt)
}

func TestGameRules(t *testing.T) {
	gr := gameRules{
		conVars: map[string]string{},
	}

	_, err := gr.RoundTime()
	assert.Equal(t, ErrFailedToRetrieveGameRule, err)

	_, err = gr.BombTime()
	assert.Equal(t, ErrFailedToRetrieveGameRule, err)

	_, err = gr.FreezeTime()
	assert.Equal(t, ErrFailedToRetrieveGameRule, err)

	ent := new(stfake.Entity)
	ent.On("Property", "cs_gamerules_data.m_iRoundTime").Return(nil)
	gr = gameRules{entity: ent}

	_, err = gr.RoundTime()
	assert.Equal(t, ErrFailedToRetrieveGameRule, err)
}

func newPlayer() *common.Player {
	pl := newPlayerWithEntityID(1)
	return pl
}

func newPlayerWithEntityID(id int) *common.Player {
	pl := common.NewPlayer(nil)
	pl.Entity = fakePlayerEntity(id)
	pl.IsConnected = true

	return pl
}

func newDisconnectedPlayer() *common.Player {
	pl := common.NewPlayer(nil)
	pl.Entity = new(stfake.Entity)

	return pl
}

func TestGameState_Hostages(t *testing.T) {
	hostageA := common.NewHostage(nil, new(stfake.Entity))
	hostageB := common.NewHostage(nil, new(stfake.Entity))
	hostages := map[int]*common.Hostage{0: hostageA, 1: hostageB}
	gs := gameState{hostages: hostages}

	expectedHostages := []*common.Hostage{hostageA, hostageB}
	assert.Equal(t, expectedHostages, gs.Hostages())
}
