package common

import (
	"testing"
	"time"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

func TestPlayerActiveWeapon(t *testing.T) {

	pl := playerWithPawnProperty("m_pWeaponServices.m_hActiveWeapon", st.PropertyValue{Any: uint64(3)})
	knife := NewEquipment(EqKnife)
	glock := NewEquipment(EqGlock)
	ak47 := NewEquipment(EqAK47)
	pl.demoInfoProvider = demoInfoProviderMock{equipment: ak47}
	pl.Inventory = make(map[int]*Equipment)
	pl.Inventory[1] = knife
	pl.Inventory[3] = glock

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
	pl := playerWithPawnProperty("m_iHealth", st.PropertyValue{Any: int32(100)})

	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl = playerWithPawnProperty("m_iHealth", st.PropertyValue{Any: int32(1)})
	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl = playerWithPawnProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{Any: int32(0)}},
		{propName: "m_lifeState", value: st.PropertyValue{Any: uint64(2)}},
	})
	assert.Equal(t, false, pl.IsAlive(), "Should be dead")

	pl = playerWithPawnProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{Any: int32(-10)}},
		{propName: "m_lifeState", value: st.PropertyValue{Any: uint64(2)}},
	})
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
	pl := playerWithPawnProperties([]fakeProp{
		{propName: "m_bSpottedByMask.0000", value: st.PropertyValue{Any: uint64(2)}},
	})
	pl.EntityID = 1

	other := newPlayer(0)
	other.EntityID = 2

	assert.True(t, pl.IsSpottedBy(other))
	assert.True(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_HasSpotted_False(t *testing.T) {
	pl := playerWithPawnProperties([]fakeProp{
		{propName: "m_bSpottedByMask.0000", value: st.PropertyValue{Any: uint64(0)}},
	})
	pl.EntityID = 1

	other := newPlayer(0)
	other.EntityID = 2

	assert.False(t, pl.IsSpottedBy(other))
	assert.False(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_HasSpotted_BitOver32(t *testing.T) {
	pl := playerWithPawnProperties([]fakeProp{
		{propName: "m_bSpottedByMask.0001", value: st.PropertyValue{Any: uint64(1)}},
	})
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
	pl := playerWithPawnProperty("m_bInBombZone", st.PropertyValue{Any: true})

	assert.True(t, pl.IsInBombZone())
}

func TestPlayer_IsInBuyZone(t *testing.T) {
	pl := playerWithPawnProperty("m_bInBuyZone", st.PropertyValue{Any: true})

	assert.True(t, pl.IsInBuyZone())
}

func TestPlayer_IsWalking(t *testing.T) {
	pl := playerWithPawnProperty("m_bIsWalking", st.PropertyValue{Any: true})

	assert.True(t, pl.IsWalking())
}

func TestPlayer_IsScoped(t *testing.T) {
	pl := playerWithPawnProperty("m_bIsScoped", st.PropertyValue{Any: true})

	assert.True(t, pl.IsScoped())
}

func TestPlayer_IsAirborne_NilEntity(t *testing.T) {
	pl := playerWithProperty("m_hPawn", st.PropertyValue{Any: uint64(constants.InvalidEntityHandleSource2)})

	assert.False(t, pl.IsAirborne())
}

func TestPlayer_IsAirborne(t *testing.T) {
	pl := playerWithPawnProperty("m_hGroundEntity", st.PropertyValue{Any: uint64(0)})

	assert.False(t, pl.IsAirborne())

	pl = playerWithPawnProperty("m_hGroundEntity", st.PropertyValue{Any: uint64(constants.InvalidEntityHandleSource2)})

	assert.True(t, pl.IsAirborne())
}

func TestPlayer_IsDucking(t *testing.T) {
	pl := playerWithPawnProperties([]fakeProp{
		{propName: "m_fFlags", value: st.PropertyValue{Any: uint64(0)}},
		{propName: "m_pMovementServices.m_flDuckAmount", value: st.PropertyValue{Any: float32(0)}},
		{propName: "m_pMovementServices.m_bDesiresDuck", value: st.PropertyValue{Any: false}},
	})

	assert.False(t, pl.IsDucking())
	assert.True(t, pl.IsStanding())
	assert.False(t, pl.IsDuckingInProgress())
	assert.False(t, pl.IsUnDuckingInProgress())

	pl = playerWithPawnProperties([]fakeProp{
		{propName: "m_fFlags", value: st.PropertyValue{Any: uint64(4)}},
		{propName: "m_pMovementServices.m_flDuckAmount", value: st.PropertyValue{Any: float32(0.5)}},
		{propName: "m_pMovementServices.m_bDesiresDuck", value: st.PropertyValue{Any: false}},
	})

	assert.False(t, pl.IsDucking())
	assert.False(t, pl.IsStanding())
	assert.False(t, pl.IsDuckingInProgress())
	assert.True(t, pl.IsUnDuckingInProgress())

	pl = playerWithPawnProperties([]fakeProp{
		{propName: "m_fFlags", value: st.PropertyValue{Any: uint64(4)}},
		{propName: "m_pMovementServices.m_flDuckAmount", value: st.PropertyValue{Any: float32(0.5)}},
		{propName: "m_pMovementServices.m_bDesiresDuck", value: st.PropertyValue{Any: true}},
	})

	assert.False(t, pl.IsDucking())
	assert.False(t, pl.IsStanding())
	assert.True(t, pl.IsDuckingInProgress())
	assert.False(t, pl.IsUnDuckingInProgress())

	pl = playerWithPawnProperties([]fakeProp{
		{propName: "m_fFlags", value: st.PropertyValue{Any: uint64(2)}},
		{propName: "m_pMovementServices.m_flDuckAmount", value: st.PropertyValue{Any: float32(1)}},
		{propName: "m_pMovementServices.m_bDesiresDuck", value: st.PropertyValue{Any: true}},
	})

	assert.True(t, pl.IsDucking())
	assert.False(t, pl.IsStanding())
	assert.False(t, pl.IsDuckingInProgress())
	assert.False(t, pl.IsUnDuckingInProgress())
}

func TestPlayerFlags_OnGround(t *testing.T) {
	pl := playerWithPawnProperty("m_fFlags", st.PropertyValue{Any: uint64(0)})

	assert.False(t, pl.Flags().OnGround())

	pl = playerWithPawnProperty("m_fFlags", st.PropertyValue{Any: uint64(1)})

	assert.True(t, pl.Flags().OnGround())
}

func TestPlayer_HasDefuseKit(t *testing.T) {
	pl := playerWithPawnProperty("m_pItemServices.m_bHasDefuser", st.PropertyValue{Any: false})

	assert.False(t, pl.HasDefuseKit())

	pl = playerWithPawnProperty("m_pItemServices.m_bHasDefuser", st.PropertyValue{Any: true})

	assert.True(t, pl.HasDefuseKit())
}

func TestPlayer_HasHelmet(t *testing.T) {
	pl := playerWithPawnProperty("m_pItemServices.m_bHasHelmet", st.PropertyValue{Any: false})

	assert.False(t, pl.HasHelmet())

	pl = playerWithPawnProperty("m_pItemServices.m_bHasHelmet", st.PropertyValue{Any: true})

	assert.True(t, pl.HasHelmet())
}

func TestPlayer_IsControllingBot_NilEntity(t *testing.T) {
	pl := new(Player)

	assert.False(t, pl.IsControllingBot())
}

func TestPlayer_IsControllingBot(t *testing.T) {
	pl := playerWithProperty("m_bControllingBot", st.PropertyValue{Any: false})

	assert.False(t, pl.IsControllingBot())

	pl = playerWithProperty("m_bControllingBot", st.PropertyValue{Any: true})

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
		playersByHandle: map[uint64]*Player{
			12: dave,
		},
	}

	pl := playerWithProperty("m_hOriginalControllerOfCurrentPawn", st.PropertyValue{Any: uint64(0)})
	pl.demoInfoProvider = demoInfoProvider

	assert.Nil(t, pl.ControlledBot())

	pl = playerWithProperty("m_hOriginalControllerOfCurrentPawn", st.PropertyValue{Any: uint64(12)})
	pl.demoInfoProvider = demoInfoProvider

	assert.Same(t, dave, pl.ControlledBot())
}

func TestPlayer_ClanTag(t *testing.T) {
	pl := playerWithProperty("m_szClan", st.PropertyValue{Any: "SuperClan"})

	assert.Equal(t, "SuperClan", pl.ClanTag())
}

func TestPlayer_CrosshairCode(t *testing.T) {
	pl := playerWithProperty("m_szCrosshairCodes", st.PropertyValue{Any: "CSGO-jvnbx-S3xFK-iEJXD-Y27Nd-AO6FP"})

	assert.Equal(t, "CSGO-jvnbx-S3xFK-iEJXD-Y27Nd-AO6FP", pl.CrosshairCode())
}

func TestPlayer_WithoutCrosshairCode(t *testing.T) {
	pl := newPlayer(0)

	assert.Equal(t, pl.CrosshairCode(), "")
}

func TestPlayer_Ping(t *testing.T) {
	pl := playerWithProperty("m_iPing", st.PropertyValue{Any: uint64(45)})

	assert.Equal(t, 45, pl.Ping())
}

func TestPlayer_Score(t *testing.T) {
	pl := playerWithProperty("m_iScore", st.PropertyValue{Any: int32(10)})

	assert.Equal(t, 10, pl.Score())
}

func TestPlayer_Color(t *testing.T) {
	pl := playerWithProperty("m_iCompTeammateColor", st.PropertyValue{Any: int32(Yellow)})

	color, err := pl.ColorOrErr()
	assert.NoError(t, err)
	assert.Equal(t, Yellow, color)

	pl = &Player{
		EntityID:         1,
		demoInfoProvider: demoInfoProviderMock{},
	}

	color, err = pl.ColorOrErr()
	assert.ErrorIs(t, ErrDataNotAvailable, err)
}

func TestPlayer_Kills(t *testing.T) {
	pl := playerWithProperty("m_pActionTrackingServices.m_iKills", st.PropertyValue{Any: int32(5)})

	assert.Equal(t, 5, pl.Kills())
}

func TestPlayer_Deaths(t *testing.T) {
	pl := playerWithProperty("m_pActionTrackingServices.m_iDeaths", st.PropertyValue{Any: int32(2)})

	assert.Equal(t, 2, pl.Deaths())
}

func TestPlayer_Assists(t *testing.T) {
	pl := playerWithProperty("m_pActionTrackingServices.m_iAssists", st.PropertyValue{Any: int32(3)})

	assert.Equal(t, 3, pl.Assists())
}

func TestPlayer_MVPs(t *testing.T) {
	pl := playerWithProperty("m_iMVPs", st.PropertyValue{Any: int32(4)})

	assert.Equal(t, 4, pl.MVPs())
}

func TestPlayer_TotalDamage(t *testing.T) {
	pl := playerWithProperty("m_pActionTrackingServices.m_iDamage", st.PropertyValue{Any: int32(2900)})

	assert.Equal(t, 2900, pl.TotalDamage())
}

func TestPlayer_UtilityDamage(t *testing.T) {
	pl := playerWithProperty("m_pActionTrackingServices.m_iUtilityDamage", st.PropertyValue{Any: int32(420)})

	assert.Equal(t, 420, pl.UtilityDamage())
}

func TestPlayer_SteamID32(t *testing.T) {
	pl := &Player{SteamID64: 76561198012952267}

	assert.Equal(t, uint32(52686539), pl.SteamID32())
}

func TestPlayer_RankType(t *testing.T) {
	pl := playerWithProperty("m_iCompetitiveRankType", st.PropertyValue{Any: int32(6)})

	assert.Equal(t, 6, pl.RankType())
}

func TestPlayer_Rank(t *testing.T) {
	pl := playerWithProperty("m_iCompetitiveRanking", st.PropertyValue{Any: int32(10)})

	assert.Equal(t, 10, pl.Rank())
}

func TestPlayer_CompetitiveWins(t *testing.T) {
	pl := playerWithProperty("m_iCompetitiveWins", st.PropertyValue{Any: int32(190)})

	assert.Equal(t, 190, pl.CompetitiveWins())
}

func TestPlayer_ViewmodelOffset(t *testing.T) {
	// Set up controller entity with pawn references
	controllerEntity := entityWithProperties([]fakeProp{
		{propName: "m_hPlayerPawn", value: st.PropertyValue{Any: uint64(1)}},
		{propName: "m_hPawn", value: st.PropertyValue{Any: uint64(1)}},
	})

	// Set up pawn entity with viewmodel offset properties
	pawnEntity := entityWithProperties([]fakeProp{
		{propName: "m_flViewmodelOffsetX", value: st.PropertyValue{Any: float32(-1.5)}},
		{propName: "m_flViewmodelOffsetY", value: st.PropertyValue{Any: float32(2.0)}},
		{propName: "m_flViewmodelOffsetZ", value: st.PropertyValue{Any: float32(-0.5)}},
	})

	pl := &Player{Entity: controllerEntity}
	pl.demoInfoProvider = demoInfoProviderMock{
		entitiesByHandle: map[uint64]st.Entity{
			1: pawnEntity,
		},
	}

	assert.Equal(t, r3.Vector{X: -1.5, Y: 2.0, Z: -0.5}, pl.ViewmodelOffset())
}

func TestPlayer_ViewmodelFOV(t *testing.T) {
	// Set up controller entity with pawn references
	controllerEntity := entityWithProperties([]fakeProp{
		{propName: "m_hPlayerPawn", value: st.PropertyValue{Any: uint64(1)}},
		{propName: "m_hPawn", value: st.PropertyValue{Any: uint64(1)}},
	})

	// Set up pawn entity with viewmodel FOV property
	pawnEntity := entityWithProperty("m_flViewmodelFOV", st.PropertyValue{Any: float32(60)})

	pl := &Player{Entity: controllerEntity}
	pl.demoInfoProvider = demoInfoProviderMock{
		entitiesByHandle: map[uint64]st.Entity{
			1: pawnEntity,
		},
	}

	assert.Equal(t, float32(60), pl.ViewmodelFOV())
}

func newPlayer(tick int) *Player {
	return NewPlayer(mockDemoInfoProvider(128, tick))
}

func playerWithProperty(propName string, value st.PropertyValue) *Player {
	return &Player{Entity: entityWithProperty(propName, value)}
}

func playerWithPawnProperties(props []fakeProp) *Player {
	h := uint64(1)

	return &Player{
		Entity: entityWithProperties([]fakeProp{
			{propName: "m_hPawn", value: st.PropertyValue{Any: h}},
			{propName: "m_hPlayerPawn", value: st.PropertyValue{Any: h}},
		}),
		demoInfoProvider: demoInfoProviderMock{
			entitiesByHandle: map[uint64]st.Entity{
				h: entityWithProperties(props),
			},
		},
	}
}

func playerWithPawnProperty(propName string, value st.PropertyValue) *Player {
	return playerWithPawnProperties([]fakeProp{{
		propName: propName,
		value:    value,
	}})
}
