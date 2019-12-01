package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

func TestPlayerActiveWeapon(t *testing.T) {
	knife := NewEquipment(EqKnife)
	glock := NewEquipment(EqGlock)
	ak47 := NewEquipment(EqAK47)

	pl := newPlayer(0)
	pl.RawWeapons[1] = &knife
	pl.RawWeapons[2] = &glock
	pl.RawWeapons[3] = &ak47
	pl.ActiveWeaponID = 3

	assert.Equal(t, &ak47, pl.ActiveWeapon(), "Should have AK-47 equipped")
}

func TestPlayerWeapons(t *testing.T) {
	knife := NewEquipment(EqKnife)
	glock := NewEquipment(EqGlock)
	ak47 := NewEquipment(EqAK47)

	pl := newPlayer(0)
	pl.RawWeapons[1] = &knife
	pl.RawWeapons[2] = &glock
	pl.RawWeapons[3] = &ak47

	expected := []*Equipment{&knife, &glock, &ak47}
	assert.ElementsMatch(t, expected, pl.Weapons(), "Should have expected weapons")
}

func TestPlayerAlive(t *testing.T) {
	pl := newPlayer(0)

	pl.Hp = 100
	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl.Hp = 1
	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl.Hp = 0
	assert.Equal(t, false, pl.IsAlive(), "Should be dead")

	pl.Hp = -10
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
	pl := newPlayer(128 * 2)

	pl.FlashDuration = 3
	pl.FlashTick = 128
	assert.Equal(t, 2*time.Second, pl.FlashDurationTimeRemaining())
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

	pl.FlashDuration = 2
	pl.FlashTick = 128 * 2
	assert.Equal(t, 2*time.Second, pl.FlashDurationTimeRemaining())
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

func newPlayer(tick int) *Player {
	return NewPlayer(mockDemoInfoProvider(128, tick))
}

func playerWithProperty(propName string, value st.PropertyValue) *Player {
	return &Player{Entity: entityWithProperty(propName, value)}
}
