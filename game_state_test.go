package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/common"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
	"github.com/markus-wa/demoinfocs-golang/sendtables/fake"
)

func TestNewGameState(t *testing.T) {
	gs := newGameState()

	assert.NotNil(t, gs.playersByEntityID)
	assert.NotNil(t, gs.playersByUserID)
	assert.NotNil(t, gs.grenadeProjectiles)
	assert.NotNil(t, gs.infernos)
	assert.NotNil(t, gs.entities)
	assert.Equal(t, common.TeamTerrorists, gs.tState.Team())
	assert.Equal(t, common.TeamCounterTerrorists, gs.ctState.Team())
}

func TestNewGameState_TeamState_Pointers(t *testing.T) {
	gs := newGameState()

	assert.True(t, gs.TeamCounterTerrorists() == &gs.ctState)
	assert.True(t, gs.TeamTerrorists() == &gs.tState)
}

func TestNewGameState_TeamState_Opponent(t *testing.T) {
	gs := newGameState()

	assert.True(t, &gs.ctState == gs.tState.Opponent)
	assert.True(t, &gs.tState == gs.ctState.Opponent)
}

func TestGameState_Participants(t *testing.T) {
	gs := newGameState()
	ptcp := gs.Participants()
	byEntity := ptcp.ByEntityID()
	byUserID := ptcp.ByUserID()

	// Should update ptcp as well since it uses the same map
	gs.playersByEntityID[0] = newPlayer()
	gs.playersByUserID[0] = newPlayer()

	assert.Equal(t, gs.playersByEntityID, ptcp.ByEntityID())
	assert.Equal(t, gs.playersByUserID, ptcp.ByUserID())

	// But should not update byEntity or byUserID since they're copies
	assert.NotEqual(t, byEntity, ptcp.ByEntityID())
	assert.NotEqual(t, byUserID, ptcp.ByUserID())
}

func TestGameState_ConVars(t *testing.T) {
	cvars := make(map[string]string)
	gs := GameState{conVars: cvars}

	assert.Equal(t, cvars, gs.ConVars())
}

func TestParticipants_All(t *testing.T) {
	pl := newPlayer()
	ptcps := Participants{
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

	ptcps := Participants{
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

	ptcps := Participants{
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

	ptcps := Participants{
		playersByEntityID: map[int]*common.Player{
			3000 & entityHandleIndexMask: pl,
		},
	}

	found := ptcps.FindByHandle(3000)

	assert.Equal(t, pl, found)
}

func TestParticipants_FindByHandle_InvalidEntityHandle(t *testing.T) {
	pl := newPlayer()
	pl.Team = common.TeamTerrorists
	ptcps := Participants{
		playersByEntityID: map[int]*common.Player{
			invalidEntityHandle & entityHandleIndexMask: pl,
		},
	}

	found := ptcps.FindByHandle(invalidEntityHandle)

	assert.Nil(t, found)
}

func TestParticipants_Connected_SuppressNoEntity(t *testing.T) {
	pl := newPlayer()
	pl2 := common.NewPlayer(nil)
	pl2.IsConnected = true

	ptcps := Participants{
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

	ptcps := Participants{
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
	entity := new(fake.Entity)
	prop0 := new(fake.Property)
	prop0.On("Value").Return(st.PropertyValue{IntVal: 1})
	entity.On("FindPropertyI", "m_bSpottedByMask.000").Return(prop0)
	prop1 := new(fake.Property)
	prop1.On("Value").Return(st.PropertyValue{IntVal: 1 << 2})
	entity.On("FindPropertyI", "m_bSpottedByMask.001").Return(prop1)
	spotted.Entity = entity
	spotted.EntityID = 3

	ptcps := Participants{
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
	spotted1.EntityID = 1
	spotted2 := newPlayer()
	spotted2.EntityID = 35

	entity := new(fake.Entity)
	prop0 := new(fake.Property)
	prop0.On("Value").Return(st.PropertyValue{IntVal: 1})
	entity.On("FindPropertyI", "m_bSpottedByMask.000").Return(prop0)
	spotted1.Entity = entity
	spotted2.Entity = entity

	unSpotted := newPlayer()
	unSpotted.EntityID = 5
	spotter := newPlayer()
	spotter.EntityID = 1

	unSpottedEntity := new(fake.Entity)
	unSpottedProp := new(fake.Property)
	unSpottedProp.On("Value").Return(st.PropertyValue{IntVal: 0})
	unSpottedEntity.On("FindPropertyI", "m_bSpottedByMask.000").Return(unSpottedProp)
	unSpotted.Entity = unSpottedEntity
	spotter.Entity = unSpottedEntity

	ptcps := Participants{
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

func newPlayer() *common.Player {
	pl := common.NewPlayer(nil)
	pl.Entity = new(st.Entity)
	pl.IsConnected = true
	return pl
}
