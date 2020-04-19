package common

import (
	"testing"
	"time"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables/fake"
)

func TestBombPosition(t *testing.T) {
	groundPos := r3.Vector{X: 1, Y: 2, Z: 3}
	bomb := Bomb{
		LastOnGroundPosition: groundPos,
	}

	assert.Equal(t, groundPos, bomb.Position(), "Bomb position should be LastOnGroundPosition")

	playerPos := r3.Vector{X: 4, Y: 5, Z: 6}

	plEntity := entityWithID(1)
	plEntity.On("Position").Return(playerPos)

	bomb.Carrier = &Player{Entity: plEntity}
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
}

func TestDemoHeader_FrameRate_PlaybackTime_Zero(t *testing.T) {
	assert.Zero(t, new(DemoHeader).FrameRate())
}

func TestDemoHeader_FrameTime_PlaybackFrames_Zero(t *testing.T) {
	assert.Zero(t, new(DemoHeader).FrameTime())
}

func TestTeamState_Team(t *testing.T) {
	tState := NewTeamState(TeamTerrorists, nil)
	ctState := NewTeamState(TeamCounterTerrorists, nil)

	assert.Equal(t, TeamTerrorists, tState.Team())
	assert.Equal(t, TeamCounterTerrorists, ctState.Team())
}

func TestTeamState_Members(t *testing.T) {
	members := []*Player{new(Player), new(Player)}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, members, state.Members())
}

func TestTeamState_EquipmentValueCurrent(t *testing.T) {
	members := []*Player{
		playerWithProperty("m_unCurrentEquipmentValue", st.PropertyValue{IntVal: 100}),
		playerWithProperty("m_unCurrentEquipmentValue", st.PropertyValue{IntVal: 200}),
	}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.CurrentEquipmentValue())
}

func TestTeamState_EquipmentValueRoundStart(t *testing.T) {
	members := []*Player{
		playerWithProperty("m_unRoundStartEquipmentValue", st.PropertyValue{IntVal: 100}),
		playerWithProperty("m_unRoundStartEquipmentValue", st.PropertyValue{IntVal: 200}),
	}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.RoundStartEquipmentValue())
}

func TestTeamState_EquipmentValueFreezeTimeEnd(t *testing.T) {
	members := []*Player{
		playerWithProperty("m_unFreezetimeEndEquipmentValue", st.PropertyValue{IntVal: 100}),
		playerWithProperty("m_unFreezetimeEndEquipmentValue", st.PropertyValue{IntVal: 200}),
	}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.FreezeTimeEndEquipmentValue())
}

func TestTeamState_MoneySpentThisRound(t *testing.T) {
	members := []*Player{
		NewPlayer(demoInfoProviderMock{playerResourceEntity: entityWithProperty("m_iCashSpentThisRound.000", st.PropertyValue{IntVal: 100})}),
		NewPlayer(demoInfoProviderMock{playerResourceEntity: entityWithProperty("m_iCashSpentThisRound.000", st.PropertyValue{IntVal: 200})}),
	}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.MoneySpentThisRound())
}

func TestTeamState_MoneySpentTotal(t *testing.T) {
	members := []*Player{
		NewPlayer(demoInfoProviderMock{playerResourceEntity: entityWithProperty("m_iTotalCashSpent.000", st.PropertyValue{IntVal: 100})}),
		NewPlayer(demoInfoProviderMock{playerResourceEntity: entityWithProperty("m_iTotalCashSpent.000", st.PropertyValue{IntVal: 200})}),
	}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members })

	assert.Equal(t, 300, state.MoneySpentTotal())
}

func TestConvertSteamIDTxtTo32(t *testing.T) {
	id, err := ConvertSteamIDTxtTo32("STEAM_0:1:26343269")

	assert.Nil(t, err)
	assert.Equal(t, uint32(52686539), id)
}

func TestConvertSteamIDTxtTo32_Error(t *testing.T) {
	id, err := ConvertSteamIDTxtTo32("STEAM_0:1:a")

	assert.Equal(t, uint32(0), id)
	assert.NotNil(t, err)

	id, err = ConvertSteamIDTxtTo32("STEAM_0:b:21643603")

	assert.Equal(t, uint32(0), id)
	assert.NotNil(t, err)
}

func TestConvertSteamID32To64(t *testing.T) {
	id := ConvertSteamID32To64(52686539)

	assert.Equal(t, uint64(76561198012952267), id)
}

func TestConvertSteamID64To32(t *testing.T) {
	id := ConvertSteamID64To32(76561198012952267)

	assert.Equal(t, uint32(52686539), id)
}

type demoInfoProviderMock struct {
	tickRate             float64
	ingameTick           int
	playersByHandle      map[int]*Player
	playerResourceEntity st.Entity
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

func (p demoInfoProviderMock) PlayerResourceEntity() st.Entity {
	return p.playerResourceEntity
}

func mockDemoInfoProvider(tickRate float64, tick int) demoInfoProvider {
	return demoInfoProviderMock{
		tickRate:   tickRate,
		ingameTick: tick,
	}
}

func entityWithID(id int) *stfake.Entity {
	entity := new(stfake.Entity)
	entity.On("ID").Return(id)

	return entity
}

func entityWithProperty(propName string, value st.PropertyValue) *stfake.Entity {
	entity := entityWithID(1)

	prop := new(stfake.Property)
	prop.On("Value").Return(value)

	entity.On("Property", propName).Return(prop)
	entity.On("PropertyValue", propName).Return(value, true)
	entity.On("PropertyValueMust", propName).Return(value)

	return entity
}
