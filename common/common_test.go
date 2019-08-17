package common

import (
	"testing"
	"time"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/sendtables/fake"
)

func TestBombPosition(t *testing.T) {
	groundPos := r3.Vector{X: 1, Y: 2, Z: 3}
	bomb := Bomb{
		LastOnGroundPosition: groundPos,
	}

	assert.Equal(t, groundPos, bomb.Position(), "Bomb position should be LastOnGroundPosition")

	playerPos := r3.Vector{X: 4, Y: 5, Z: 6}
	bomb.Carrier = &Player{Position: playerPos}
	assert.Equal(t, playerPos, bomb.Position(), "Bomb position should be Player.Position")
}

func TestGrenadeProjectileUniqueID(t *testing.T) {
	assert.NotEqual(t, NewGrenadeProjectile().UniqueID(), NewGrenadeProjectile().UniqueID(), "UniqueIDs of different grenade projectiles should be different")
}

func TestDemoHeader(t *testing.T) {
	header := DemoHeader{
		PlaybackFrames: 256,
		PlaybackTicks:  512,
		PlaybackTime:   4 * time.Second,
	}

	assert.Equal(t, float64(64), header.FrameRate(), "FrameRate should be 64")
	assert.Equal(t, time.Second/64, header.FrameTime(), "FrameTime should be 1/64 sec")

	assert.Equal(t, float64(128.0), header.TickRate(), "TickRate should be 128")
	assert.Equal(t, time.Second/128, header.TickTime(), "TickTime should be 1/128")
}

func TestDemoHeader_FrameTime_PlaybackFrames_Zero(t *testing.T) {
	assert.Zero(t, DemoHeader{}.FrameTime())
}

func TestTeamState_Team(t *testing.T) {
	assert.Equal(t, TeamTerrorists, NewTeamState(TeamTerrorists, nil).Team())
	assert.Equal(t, TeamCounterTerrorists, NewTeamState(TeamCounterTerrorists, nil).Team())
}

func TestTeamState_Members(t *testing.T) {
	members := []*Player{new(Player), new(Player)}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, members, state.Members())
}

func TestTeamState_CurrentEquipmentValue(t *testing.T) {
	members := []*Player{{CurrentEquipmentValue: 100}, {CurrentEquipmentValue: 200}}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.CurrentEquipmentValue())
}

func TestTeamState_RoundStartEquipmentValue(t *testing.T) {
	members := []*Player{{RoundStartEquipmentValue: 100}, {RoundStartEquipmentValue: 200}}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.RoundStartEquipmentValue())
}

func TestTeamState_FreezeTimeEndEquipmentValue(t *testing.T) {
	members := []*Player{{FreezetimeEndEquipmentValue: 100}, {FreezetimeEndEquipmentValue: 200}}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.FreezeTimeEndEquipmentValue())
}

type demoInfoProviderMock struct {
	tickRate        float64
	ingameTick      int
	playersByHandle map[int]*Player
}

func (p demoInfoProviderMock) TickRate() float64 {
	return p.tickRate
}

func (p demoInfoProviderMock) IngameTick() int {
	return p.ingameTick
}

func (p demoInfoProviderMock) FindPlayerByHandle(handle int) *Player {
	return p.playersByHandle[handle]
}

func mockDemoInfoProvider(tickRate float64, tick int) demoInfoProvider {
	return demoInfoProviderMock{
		tickRate:   tickRate,
		ingameTick: tick,
	}
}

func entityWithProperty(propName string, value st.PropertyValue) st.IEntity {
	entity := new(stfake.Entity)
	entity.On("ID").Return(1)
	prop := new(stfake.Property)
	prop.On("Value").Return(value)
	entity.On("FindPropertyI", propName).Return(prop)
	return entity
}
