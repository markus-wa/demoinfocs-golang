package demoinfocs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables/fake"
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
	gs.playersByEntityID[0] = newPlayerS1()
	gs.playersByUserID[0] = newPlayerS1()

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
	pl := newPlayerS1()
	ptcps := participants{
		playersByUserID: map[int]*common.Player{0: pl},
	}

	allPlayers := ptcps.All()

	assert.ElementsMatch(t, []*common.Player{pl}, allPlayers)
}

func TestParticipants_Playing(t *testing.T) {
	terrorist := newPlayerS1()
	terrorist.Team = common.TeamTerrorists
	ct := newPlayerS1()
	ct.Team = common.TeamCounterTerrorists
	unassigned := newPlayerS1()
	unassigned.Team = common.TeamUnassigned
	spectator := newPlayerS1()
	spectator.Team = common.TeamSpectators
	def := newPlayerS1()

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
	terrorist := newPlayerS1()
	terrorist.Team = common.TeamTerrorists
	ct := newPlayerS1()
	ct.Team = common.TeamCounterTerrorists
	unassigned := newPlayerS1()
	unassigned.Team = common.TeamUnassigned
	spectator := newPlayerS1()
	spectator.Team = common.TeamSpectators
	def := newPlayerS1()

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
	pl := newPlayerS1()
	pl.Team = common.TeamTerrorists

	ptcps := participants{
		playersByEntityID: map[int]*common.Player{
			3000 & constants.EntityHandleIndexMaskSource2: pl,
		},
	}

	found := ptcps.FindByHandle64(3000)

	assert.Equal(t, pl, found)
}

func TestParticipants_FindByHandle_InvalidEntityHandle(t *testing.T) {
	pl := newPlayerS1()
	pl.Team = common.TeamTerrorists
	ptcps := participants{
		playersByEntityID: map[int]*common.Player{
			constants.InvalidEntityHandleSource2 & constants.EntityHandleIndexMaskSource2: pl,
		},
	}

	found := ptcps.FindByHandle64(constants.InvalidEntityHandleSource2)

	assert.Nil(t, found)
}

func TestParticipants_Connected_SuppressNoEntity(t *testing.T) {
	pl := newPlayerS1()
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
	pl := newPlayerS1()
	pl2 := newPlayerS1()
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

func TestGameRules_ConVars(t *testing.T) {
	cvars := make(map[string]string)
	gr := gameRules{conVars: cvars}

	assert.Equal(t, cvars, gr.ConVars())
}

func TestGameRules_Entity(t *testing.T) {
	ent := stfake.NewEntityWithProperty("m_iGameMode", st.PropertyValue{Any: 1})
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
	prop.On("Value").Return(st.PropertyValue{Any: int32(115)})
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

func TestGameRules_IsFreezetimePeriod(t *testing.T) {
	gs := gameState{isFreezetime: true}

	assert.Equal(t, true, gs.IsFreezetimePeriod())
}

func newPlayerS1() *common.Player {
	pl := newPlayerWithEntityIDS1(1)
	return pl
}

func newPlayerWithEntityIDS1(id int) *common.Player {
	pl := common.NewPlayer(demoInfoProvider{
		parser: &parser{header: &common.DemoHeader{Filestamp: "HL2DEMO"}},
	})
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
