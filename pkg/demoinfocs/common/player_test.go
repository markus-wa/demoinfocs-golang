package common

import (
	"testing"
	"time"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables/fake"
)

func TestPlayerActiveWeapon(t *testing.T) {
	knife := NewEquipment(EqKnife)
	glock := NewEquipment(EqGlock)
	ak47 := NewEquipment(EqAK47)

	pl := newPlayer(0)
	pl.Inventory[1] = knife
	pl.Inventory[2] = glock
	pl.Inventory[3] = ak47
	pl.Entity = entityWithProperty("m_hActiveWeapon", st.PropertyValue{IntVal: 3})

	assert.Equal(t, ak47, pl.ActiveWeapon(), "Should have AK-47 equipped")
}

func TestPlayerWeapons(t *testing.T) {
	knife := NewEquipment(EqKnife)
	glock := NewEquipment(EqGlock)
	ak47 := NewEquipment(EqAK47)

	pl := newPlayer(0)
	pl.Inventory[1] = knife
	pl.Inventory[2] = glock
	pl.Inventory[3] = ak47

	expected := []*Equipment{knife, glock, ak47}
	assert.ElementsMatch(t, expected, pl.Weapons(), "Should have expected weapons")
}

func TestPlayerAlive(t *testing.T) {
	pl := newPlayer(0)

	pl.Entity = entityWithProperty("m_iHealth", st.PropertyValue{IntVal: 100})
	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl.Entity = entityWithProperty("m_iHealth", st.PropertyValue{IntVal: 1})
	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl.Entity = entityWithProperty("m_iHealth", st.PropertyValue{IntVal: 0})
	assert.Equal(t, false, pl.IsAlive(), "Should be dead")

	pl.Entity = entityWithProperty("m_iHealth", st.PropertyValue{IntVal: -10})
	assert.Equal(t, false, pl.IsAlive(), "Should be dead")
}

func TestPlayerFlashed(t *testing.T) {
	pl := newPlayer(128)

	assert.False(t, pl.IsBlinded(), "Should not be flashed")

	pl.FlashDuration = 2.3
	pl.FlashTick = 50
	assert.True(t, pl.IsBlinded(), "Should be flashed")
}

func TestPlayerFlashed_FlashDuration_Over(t *testing.T) {
	pl := newPlayer(128 * 3)

	pl.FlashDuration = 1.9
	pl.FlashTick = 128
	assert.False(t, pl.IsBlinded(), "Should not be flashed")
}

func TestPlayer_FlashDurationTime_Default(t *testing.T) {
	pl := newPlayer(0)

	assert.Equal(t, time.Duration(0), pl.FlashDurationTime())
}

func TestPlayer_FlashDurationTime(t *testing.T) {
	pl := newPlayer(0)

	pl.FlashDuration = 2.3

	assert.Equal(t, 2300*time.Millisecond, pl.FlashDurationTime())
}

func TestPlayer_FlashDurationTimeRemaining_Default(t *testing.T) {
	pl := NewPlayer(mockDemoInfoProvider(0, 128))

	assert.Equal(t, time.Duration(0), pl.FlashDurationTimeRemaining())
}

func TestPlayer_FlashDurationTimeRemaining(t *testing.T) {
	pl := newPlayer(32 * 5)

	pl.FlashDuration = 3
	pl.FlashTick = 128
	assert.Equal(t, 2750*time.Millisecond, pl.FlashDurationTimeRemaining())
}

func TestPlayer_FlashDurationTimeRemaining_Zero(t *testing.T) {
	pl := newPlayer(128 * 4)

	pl.FlashDuration = 3
	pl.FlashTick = 128
	assert.Equal(t, time.Duration(0), pl.FlashDurationTimeRemaining())
}

func TestPlayer_FlashDurationTimeRemaining_FlashDuration_Over(t *testing.T) {
	pl := newPlayer(128 * 4)

	pl.FlashDuration = 1
	pl.FlashTick = 128
	assert.Equal(t, time.Duration(0), pl.FlashDurationTimeRemaining())
}

func TestPlayer_FlashDurationTimeRemaining_Fallback(t *testing.T) {
	pl := NewPlayer(mockDemoInfoProvider(0, 128))

	pl.FlashDuration = 2.7
	pl.FlashTick = 128 * 3
	assert.Equal(t, 2700*time.Millisecond, pl.FlashDurationTimeRemaining())
}

func TestPlayer_IsSpottedBy_HasSpotted_True(t *testing.T) {
	pl := playerWithProperty("m_bSpottedByMask.000", st.PropertyValue{IntVal: 2})
	pl.EntityID = 1

	other := newPlayer(0)
	other.EntityID = 2

	assert.True(t, pl.IsSpottedBy(other))
	assert.True(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_HasSpotted_False(t *testing.T) {
	pl := playerWithProperty("m_bSpottedByMask.000", st.PropertyValue{IntVal: 0})
	pl.EntityID = 1

	other := newPlayer(0)
	other.EntityID = 2

	assert.False(t, pl.IsSpottedBy(other))
	assert.False(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_HasSpotted_BitOver32(t *testing.T) {
	pl := playerWithProperty("m_bSpottedByMask.001", st.PropertyValue{IntVal: 1})
	pl.EntityID = 1

	other := newPlayer(0)
	other.EntityID = 33

	assert.True(t, pl.IsSpottedBy(other))
	assert.True(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_EntityNull(t *testing.T) {
	pl := new(Player)
	pl.EntityID = 1
	other := new(Player)
	other.EntityID = 2

	assert.False(t, pl.IsSpottedBy(other))
	assert.False(t, other.HasSpotted(pl))
}

func TestPlayer_IsInBombZone(t *testing.T) {
	pl := playerWithProperty("m_bInBombZone", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.IsInBombZone())
}

func TestPlayer_IsInBuyZone(t *testing.T) {
	pl := playerWithProperty("m_bInBuyZone", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.IsInBuyZone())
}

func TestPlayer_IsWalking(t *testing.T) {
	pl := playerWithProperty("m_bIsWalking", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.IsWalking())
}

func TestPlayer_IsScoped(t *testing.T) {
	pl := playerWithProperty("m_bIsScoped", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.IsScoped())
}

func TestPlayer_IsAirborne_NilEntity(t *testing.T) {
	pl := new(Player)

	assert.False(t, pl.IsAirborne())
}

func TestPlayer_IsAirborne(t *testing.T) {
	pl := playerWithProperty("m_hGroundEntity", st.PropertyValue{IntVal: 0})

	assert.False(t, pl.IsAirborne())

	pl = playerWithProperty("m_hGroundEntity", st.PropertyValue{IntVal: 2097151})

	assert.True(t, pl.IsAirborne())
}

func TestPlayer_IsDucking(t *testing.T) {
	pl := playerWithProperty("localdata.m_Local.m_bDucking", st.PropertyValue{IntVal: 0})

	assert.False(t, pl.IsDucking())

	pl = playerWithProperty("localdata.m_Local.m_bDucking", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.IsDucking())
}

func TestPlayer_HasDefuseKit(t *testing.T) {
	pl := playerWithProperty("m_bHasDefuser", st.PropertyValue{IntVal: 0})

	assert.False(t, pl.HasDefuseKit())

	pl = playerWithProperty("m_bHasDefuser", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.HasDefuseKit())
}

func TestPlayer_HasHelmet(t *testing.T) {
	pl := playerWithProperty("m_bHasHelmet", st.PropertyValue{IntVal: 0})

	assert.False(t, pl.HasHelmet())

	pl = playerWithProperty("m_bHasHelmet", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.HasHelmet())
}

func TestPlayer_IsControllingBot_NilEntity(t *testing.T) {
	pl := new(Player)

	assert.False(t, pl.IsControllingBot())
}

func TestPlayer_IsControllingBot(t *testing.T) {
	pl := playerWithProperty("m_bIsControllingBot", st.PropertyValue{IntVal: 0})

	assert.False(t, pl.IsControllingBot())

	pl = playerWithProperty("m_bIsControllingBot", st.PropertyValue{IntVal: 1})

	assert.True(t, pl.IsControllingBot())
}

func TestPlayer_ControlledBot_NilEntity(t *testing.T) {
	pl := new(Player)

	assert.Nil(t, pl.ControlledBot())
}

func TestPlayer_ControlledBot(t *testing.T) {
	dave := &Player{
		Name:  "Dave",
		IsBot: true,
	}
	demoInfoProvider := &demoInfoProviderMock{
		playersByHandle: map[int]*Player{
			12: dave,
		},
	}

	pl := playerWithProperty("m_iControlledBotEntIndex", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = demoInfoProvider

	assert.Nil(t, pl.ControlledBot())

	pl = playerWithProperty("m_iControlledBotEntIndex", st.PropertyValue{IntVal: 12})
	pl.demoInfoProvider = demoInfoProvider

	assert.Same(t, dave, pl.ControlledBot())
}

func TestPlayer_Armor(t *testing.T) {
	pl := playerWithProperty("m_ArmorValue", st.PropertyValue{IntVal: 95})

	assert.Equal(t, 95, pl.Armor())
}

func TestPlayer_Money(t *testing.T) {
	pl := playerWithProperty("m_iAccount", st.PropertyValue{IntVal: 800})

	assert.Equal(t, 800, pl.Money())
}

func TestPlayer_ViewDirectionX(t *testing.T) {
	pl := playerWithProperty("m_angEyeAngles[1]", st.PropertyValue{FloatVal: 180})

	assert.Equal(t, float32(180), pl.ViewDirectionX())
}

func TestPlayer_ViewDirectionY(t *testing.T) {
	pl := playerWithProperty("m_angEyeAngles[0]", st.PropertyValue{FloatVal: 15})

	assert.Equal(t, float32(15), pl.ViewDirectionY())
}

func TestPlayer_Position(t *testing.T) {
	entity := new(stfake.Entity)
	pos := r3.Vector{X: 1, Y: 2, Z: 3}

	entity.On("Position").Return(pos)

	pl := &Player{Entity: entity}

	assert.Equal(t, pos, pl.Position())
}

func TestPlayer_Position_EntityNil(t *testing.T) {
	pl := new(Player)

	assert.Empty(t, pl.Position())
}

func TestPlayer_Velocity(t *testing.T) {
	entity := new(stfake.Entity)
	entity.On("PropertyValueMust", "localdata.m_vecVelocity[0]").Return(st.PropertyValue{FloatVal: 1})
	entity.On("PropertyValueMust", "localdata.m_vecVelocity[1]").Return(st.PropertyValue{FloatVal: 2})
	entity.On("PropertyValueMust", "localdata.m_vecVelocity[2]").Return(st.PropertyValue{FloatVal: 3})

	pl := &Player{Entity: entity}

	expected := r3.Vector{X: 1, Y: 2, Z: 3}
	assert.Equal(t, expected, pl.Velocity())
}

func TestPlayer_Velocity_EntityNil(t *testing.T) {
	pl := new(Player)

	assert.Empty(t, pl.Velocity())
}

func TestPlayer_ClanTag(t *testing.T) {
	pl := playerWithResourceProperty("m_szClan", st.PropertyValue{StringVal: "SuperClan"})

	assert.Equal(t, "SuperClan", pl.ClanTag())
}

func TestPlayer_Ping(t *testing.T) {
	pl := playerWithResourceProperty("m_iPing", st.PropertyValue{IntVal: 45})

	assert.Equal(t, 45, pl.Ping())
}

func TestPlayer_Score(t *testing.T) {
	pl := playerWithResourceProperty("m_iScore", st.PropertyValue{IntVal: 10})

	assert.Equal(t, 10, pl.Score())
}

func TestPlayer_Kills(t *testing.T) {
	pl := playerWithResourceProperty("m_iKills", st.PropertyValue{IntVal: 5})

	assert.Equal(t, 5, pl.Kills())
}

func TestPlayer_Deaths(t *testing.T) {
	pl := playerWithResourceProperty("m_iDeaths", st.PropertyValue{IntVal: 2})

	assert.Equal(t, 2, pl.Deaths())
}

func TestPlayer_Assists(t *testing.T) {
	pl := playerWithResourceProperty("m_iAssists", st.PropertyValue{IntVal: 3})

	assert.Equal(t, 3, pl.Assists())
}

func TestPlayer_MVPs(t *testing.T) {
	pl := playerWithResourceProperty("m_iMVPs", st.PropertyValue{IntVal: 4})

	assert.Equal(t, 4, pl.MVPs())
}

func TestPlayer_ResourceEntity(t *testing.T) {
	resourceEntity := entityWithID(1)
	pl := &Player{
		demoInfoProvider: demoInfoProviderMock{
			playerResourceEntity: resourceEntity,
		},
	}

	assert.Same(t, resourceEntity, pl.ResourceEntity())
}

func TestPlayer_SteamID32(t *testing.T) {
	pl := &Player{SteamID64: 76561198012952267}

	assert.Equal(t, uint32(52686539), pl.SteamID32())
}

func newPlayer(tick int) *Player {
	return NewPlayer(mockDemoInfoProvider(128, tick))
}

func playerWithProperty(propName string, value st.PropertyValue) *Player {
	return &Player{Entity: entityWithProperty(propName, value)}
}

func playerWithResourceProperty(propName string, value st.PropertyValue) *Player {
	return &Player{
		EntityID: 1,
		demoInfoProvider: demoInfoProviderMock{
			playerResourceEntity: entityWithProperty(propName+".001", value),
		},
	}
}
