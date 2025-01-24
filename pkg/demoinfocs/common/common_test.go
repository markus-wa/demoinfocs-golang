package common

import (
	"testing"
	"time"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables/fake"
)

func TestGrenadeProjectileUniqueID(t *testing.T) {
	assert.NotEqual(t, NewGrenadeProjectile().UniqueID(), NewGrenadeProjectile().UniqueID(), "UniqueIDs of different grenade projectiles should be different")
}

func TestGrenadeProjectile_Velocity(t *testing.T) {
	expected := r3.Vector{
		X: 1,
		Y: 2,
		Z: 3,
	}

	p := GrenadeProjectile{
		Entity: entityWithProperty("m_vecVelocity", st.PropertyValue{
			Any: []float32{1, 2, 3},
		}),
	}

	assert.Equal(t, expected, p.Velocity())
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
	tState := NewTeamState(TeamTerrorists, nil, demoInfoProviderMock{})
	ctState := NewTeamState(TeamCounterTerrorists, nil, demoInfoProviderMock{})

	assert.Equal(t, TeamTerrorists, tState.Team())
	assert.Equal(t, TeamCounterTerrorists, ctState.Team())
}

func TestTeamState_Members(t *testing.T) {
	members := []*Player{new(Player), new(Player)}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members }, demoInfoProviderMock{})

	assert.Equal(t, members, state.Members())
}

func TestTeamState_EquipmentValueCurrent(t *testing.T) {
	members := []*Player{
		playerWithPawnProperty("m_unCurrentEquipmentValue", st.PropertyValue{Any: uint64(100)}),
		playerWithPawnProperty("m_unCurrentEquipmentValue", st.PropertyValue{Any: uint64(200)}),
	}

	dip := demoInfoProviderMock{}

	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members }, dip)

	assert.Equal(t, 300, state.CurrentEquipmentValue())
}

func TestTeamState_EquipmentValueRoundStart(t *testing.T) {
	members := []*Player{
		playerWithPawnProperty("m_unRoundStartEquipmentValue", st.PropertyValue{Any: uint64(100)}),
		playerWithPawnProperty("m_unRoundStartEquipmentValue", st.PropertyValue{Any: uint64(200)}),
	}

	dip := demoInfoProviderMock{}

	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members }, dip)

	assert.Equal(t, 300, state.RoundStartEquipmentValue())
}

func TestTeamState_EquipmentValueFreezeTimeEnd(t *testing.T) {
	members := []*Player{
		playerWithPawnProperty("m_unFreezetimeEndEquipmentValue", st.PropertyValue{Any: uint64(100)}),
		playerWithPawnProperty("m_unFreezetimeEndEquipmentValue", st.PropertyValue{Any: uint64(200)}),
	}

	dip := demoInfoProviderMock{}

	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members }, dip)

	assert.Equal(t, 300, state.FreezeTimeEndEquipmentValue())
}

func TestTeamState_MoneySpentThisRound(t *testing.T) {
	members := []*Player{
		playerWithProperty("m_pInGameMoneyServices.m_iCashSpentThisRound", st.PropertyValue{Any: int32(100)}),
		playerWithProperty("m_pInGameMoneyServices.m_iCashSpentThisRound", st.PropertyValue{Any: int32(200)}),
	}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members }, demoInfoProviderMock{})

	assert.Equal(t, 300, state.MoneySpentThisRound())
}

func TestTeamState_MoneySpentTotal(t *testing.T) {
	members := []*Player{
		playerWithProperty("m_pInGameMoneyServices.m_iTotalCashSpent", st.PropertyValue{Any: int32(100)}),
		playerWithProperty("m_pInGameMoneyServices.m_iTotalCashSpent", st.PropertyValue{Any: int32(200)}),
	}
	state := NewTeamState(TeamTerrorists, func(Team) []*Player { return members }, demoInfoProviderMock{})

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

	id, err = ConvertSteamIDTxtTo32("STEAM_0:b")

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

type fakeProp struct {
	propName string
	value    st.PropertyValue
	isNil    bool
}

type demoInfoProviderMock struct {
	tickRate         float64
	ingameTick       int
	playersByHandle  map[uint64]*Player
	entitiesByHandle map[uint64]st.Entity
	equipment        *Equipment
}

func (p demoInfoProviderMock) FindEntityByHandle(handle uint64) st.Entity {
	return p.entitiesByHandle[handle]
}

func (p demoInfoProviderMock) TickRate() float64 {
	return p.tickRate
}

func (p demoInfoProviderMock) IngameTick() int {
	return p.ingameTick
}

func (p demoInfoProviderMock) FindPlayerByHandle(handle uint64) *Player {
	return p.playersByHandle[handle]
}

func (p demoInfoProviderMock) FindPlayerByPawnHandle(handle uint64) *Player {
	return p.playersByHandle[handle]
}

func (p demoInfoProviderMock) FindWeaponByEntityID(id int) *Equipment {
	return p.equipment
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

func entityWithProperties(properties []fakeProp) *stfake.Entity {
	entity := entityWithID(1)

	for _, prop := range properties {
		if prop.isNil {
			entity.On("Property", prop.propName).Return(nil)

			continue
		}

		property := new(stfake.Property)
		property.On("Value").Return(prop.value)

		entity.On("Property", prop.propName).Return(property)
		entity.On("PropertyValue", prop.propName).Return(prop.value, true)
		entity.On("PropertyValueMust", prop.propName).Return(prop.value)
	}

	entity.On("Property", mock.Anything).Return(nil)

	return entity
}

func entityWithoutProperty(propName string) *stfake.Entity {
	entity := entityWithID(1)

	entity.On("Property", propName).Return(nil)
	entity.On("PropertyValue", propName).Return(st.PropertyValue{}, false)
	entity.On("PropertyValueMust", propName).Panic("property not found")

	return entity
}
