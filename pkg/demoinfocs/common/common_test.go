package common

import (
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables/sendtablescs2"
)

func TestGrenadeProjectileUniqueID(t *testing.T) {
	assert.NotEqual(t, NewGrenadeProjectile(UniqueID(1)).UniqueID(), NewGrenadeProjectile(UniqueID(2)).UniqueID(), "UniqueIDs of different grenade projectiles should be different")
}

func TestGrenadeProjectile_Velocity(t *testing.T) {
	expected := r3.Vector{
		X: 1,
		Y: 2,
		Z: 3,
	}

	p := GrenadeProjectile{
		Entity: entityWithProperties([]fakeProp{
			{propName: "m_vecVelocity.m_vecX", value: st.PropertyValue{Any: float32(1)}},
			{propName: "m_vecVelocity.m_vecY", value: st.PropertyValue{Any: float32(2)}},
			{propName: "m_vecVelocity.m_vecZ", value: st.PropertyValue{Any: float32(3)}},
		}),
	}

	assert.Equal(t, expected, p.Velocity())
}

// Demos where the projectile's class does not also declare view-offset
// components keep the un-prefixed component names.
func TestGrenadeProjectile_Velocity_CS2_BareComponentNames(t *testing.T) {
	expected := r3.Vector{
		X: 1,
		Y: 2,
		Z: 3,
	}

	p := GrenadeProjectile{
		Entity: entityWithProperties([]fakeProp{
			{propName: "m_vecX", value: st.PropertyValue{Any: float32(1)}},
			{propName: "m_vecY", value: st.PropertyValue{Any: float32(2)}},
			{propName: "m_vecZ", value: st.PropertyValue{Any: float32(3)}},
		}),
	}

	assert.Equal(t, expected, p.Velocity())
}

func TestGrenadeProjectile_Velocity_Unavailable(t *testing.T) {
	p := GrenadeProjectile{
		Entity: entityWithProperties(nil),
	}

	assert.Equal(t, r3.Vector{}, p.Velocity())
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
	tickRate              float64
	ingameTick            int
	playersByHandle       map[uint64]*Player
	entitiesByHandle      map[uint64]st.Entity
	equipment             *Equipment
	networkProtocol       int
	eagerVelocityTracking bool
	snapshotTick          int
	processingSnapshot    bool
	tracked               *[]trackedRecorder
	pov                   bool // recordingSlot is only used if set, so the zero value is a GOTV demo
	recordingSlot         int
}

type trackedRecorder struct {
	record func() bool
	lazy   bool
}

func (p demoInfoProviderMock) EagerVelocityTracking() bool {
	return p.eagerVelocityTracking
}

func (p demoInfoProviderMock) SnapshotTick() int {
	return p.snapshotTick
}

func (p demoInfoProviderMock) ProcessingSnapshot() bool {
	return p.processingSnapshot
}

func (p demoInfoProviderMock) RecordingPlayerSlot() int {
	if !p.pov {
		return -1
	}

	return p.recordingSlot
}

func (p demoInfoProviderMock) TrackFramePosition(record func() bool, lazy bool) {
	if p.tracked != nil {
		*p.tracked = append(*p.tracked, trackedRecorder{record: record, lazy: lazy})
	}
}

func (p demoInfoProviderMock) NetworkProtocol() int {
	return p.networkProtocol
}

func (p demoInfoProviderMock) FindEntityByHandle(handle uint64) st.Entity {
	return p.entitiesByHandle[handle]
}

// FindEntityByID uses entitiesByHandle as well, handles of the fake entities have serial 0.
func (p demoInfoProviderMock) FindEntityByID(id int) st.Entity {
	return p.entitiesByHandle[uint64(id)]
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

func entityWithID(id int) *sendtablescs2.Entity {
	return stfake.NewEntity(id)
}

func entityWithProperty(propName string, value st.PropertyValue) *sendtablescs2.Entity {
	return stfake.NewEntity(1, stfake.P(propName, value))
}

// entityWithProperties returns an entity that has the given properties, except for the ones marked isNil.
func entityWithProperties(properties []fakeProp) *sendtablescs2.Entity {
	props := make([]stfake.Prop, 0, len(properties))

	for _, prop := range properties {
		if prop.isNil {
			continue
		}

		props = append(props, stfake.P(prop.propName, prop.value))
	}

	return stfake.NewEntity(1, props...)
}

// entityWithoutProperty returns an entity that doesn't have the given property (nor any other).
func entityWithoutProperty(string) *sendtablescs2.Entity {
	return stfake.NewEntity(1)
}
