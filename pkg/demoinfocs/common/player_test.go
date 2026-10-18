package common

import (
	"slices"
	"testing"
	"time"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables/fake"
)

func TestPlayerActiveWeapon(t *testing.T) {
	knife := NewEquipment(EqKnife, 1)
	glock := NewEquipment(EqGlock, 2)
	ak47 := NewEquipment(EqAK47, 3)

	pl := newPlayer(0)
	pl.demoInfoProvider = demoInfoProviderMock{equipment: ak47}
	pl.Inventory[1] = knife
	pl.Inventory[3] = glock
	pl.Entity = entityWithProperty("m_hActiveWeapon", st.PropertyValue{IntVal: 3})

	assert.Equal(t, ak47, pl.ActiveWeapon(), "Should have AK-47 equipped")
}

func TestPlayerWeapons(t *testing.T) {
	knife := NewEquipment(EqKnife, 1)
	glock := NewEquipment(EqGlock, 2)
	ak47 := NewEquipment(EqAK47, 3)

	pl := newPlayer(0)
	pl.Inventory[1] = knife
	pl.Inventory[2] = glock
	pl.Inventory[3] = ak47

	expected := []*Equipment{knife, glock, ak47}
	assert.ElementsMatch(t, expected, pl.Weapons(), "Should have expected weapons")
}

func TestPlayerAlive(t *testing.T) {
	pl := newPlayer(0)

	pl.Entity = entityWithProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{IntVal: 100}},
		{propName: "m_lifeState", value: st.PropertyValue{IntVal: 0}},
	})
	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl.Entity = entityWithProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{IntVal: 1}},
		{propName: "m_lifeState", value: st.PropertyValue{IntVal: 0}},
	})
	assert.Equal(t, true, pl.IsAlive(), "Should be alive")

	pl.Entity = entityWithProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{IntVal: 0}},
		{propName: "m_lifeState", value: st.PropertyValue{IntVal: 2}},
	})
	assert.Equal(t, false, pl.IsAlive(), "Should be dead")

	pl.Entity = entityWithProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{IntVal: -10}},
		{propName: "m_lifeState", value: st.PropertyValue{IntVal: 2}},
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
	pl := playerWithProperty("m_bSpottedByMask.000", st.PropertyValue{IntVal: 2})
	pl.EntityID = 1
	pl.demoInfoProvider = s1DemoInfoProvider

	other := newPlayer(0)
	other.EntityID = 2

	assert.True(t, pl.IsSpottedBy(other))
	assert.True(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_HasSpotted_False(t *testing.T) {
	pl := playerWithProperty("m_bSpottedByMask.000", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider
	pl.EntityID = 1

	other := newPlayer(0)
	other.EntityID = 2

	assert.False(t, pl.IsSpottedBy(other))
	assert.False(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_HasSpotted_BitOver32(t *testing.T) {
	pl := playerWithProperty("m_bSpottedByMask.001", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider
	pl.EntityID = 1

	other := newPlayer(0)
	other.EntityID = 33

	assert.True(t, pl.IsSpottedBy(other))
	assert.True(t, other.HasSpotted(pl))
}

func TestPlayer_IsSpottedBy_EntityNull(t *testing.T) {
	pl := new(Player)
	pl.EntityID = 1
	pl.demoInfoProvider = s1DemoInfoProvider

	other := new(Player)
	other.EntityID = 2
	other.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsSpottedBy(other))
	assert.False(t, other.HasSpotted(pl))
}

func TestPlayer_IsInBombZone(t *testing.T) {
	pl := playerWithProperty("m_bInBombZone", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.IsInBombZone())
}

func TestPlayer_IsInBuyZone(t *testing.T) {
	pl := playerWithProperty("m_bInBuyZone", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.IsInBuyZone())
}

func TestPlayer_IsWalking(t *testing.T) {
	pl := playerWithProperty("m_bIsWalking", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.IsWalking())
}

func TestPlayer_IsScoped(t *testing.T) {
	pl := playerWithProperty("m_bIsScoped", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.IsScoped())
}

func TestPlayer_IsAirborne_NilEntity(t *testing.T) {
	pl := new(Player)
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsAirborne())
}

func TestPlayer_IsAirborne(t *testing.T) {
	pl := playerWithProperty("m_hGroundEntity", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsAirborne())

	pl = playerWithProperty("m_hGroundEntity", st.PropertyValue{IntVal: 2097151})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.IsAirborne())
}

func TestPlayer_IsDucking(t *testing.T) {
	pl := playerWithProperty("m_fFlags", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsDucking())
	assert.True(t, pl.IsStanding())
	assert.False(t, pl.IsDuckingInProgress())
	assert.False(t, pl.IsUnDuckingInProgress())

	pl = playerWithProperty("m_fFlags", st.PropertyValue{IntVal: 1 << 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsDucking())
	assert.False(t, pl.IsStanding())
	assert.False(t, pl.IsDuckingInProgress())
	assert.True(t, pl.IsUnDuckingInProgress())

	pl = playerWithProperty("m_fFlags", st.PropertyValue{IntVal: 1 << 2})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsDucking())
	assert.False(t, pl.IsStanding())
	assert.True(t, pl.IsDuckingInProgress())
	assert.False(t, pl.IsUnDuckingInProgress())

	pl = playerWithProperty("m_fFlags", st.PropertyValue{IntVal: 1<<1 | 1<<2})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.IsDucking())
	assert.False(t, pl.IsStanding())
	assert.False(t, pl.IsDuckingInProgress())
	assert.False(t, pl.IsUnDuckingInProgress())
}

func TestPlayerFlags_OnGround(t *testing.T) {
	pl := playerWithProperty("m_fFlags", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.Flags().OnGround())

	pl = playerWithProperty("m_fFlags", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.Flags().OnGround())
}

func TestPlayer_HasDefuseKit(t *testing.T) {
	pl := playerWithProperty("m_bHasDefuser", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.HasDefuseKit())

	pl = playerWithProperty("m_bHasDefuser", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.HasDefuseKit())
}

func TestPlayer_HasHelmet(t *testing.T) {
	pl := playerWithProperty("m_bHasHelmet", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.HasHelmet())

	pl = playerWithProperty("m_bHasHelmet", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.HasHelmet())
}

func TestPlayer_IsControllingBot_NilEntity(t *testing.T) {
	pl := new(Player)
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsControllingBot())
}

func TestPlayer_IsControllingBot(t *testing.T) {
	pl := playerWithProperty("m_bIsControllingBot", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsControllingBot())

	pl = playerWithProperty("m_bIsControllingBot", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

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

	pl := playerWithProperty("m_iControlledBotEntIndex", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = demoInfoProvider

	assert.Nil(t, pl.ControlledBot())

	pl = playerWithProperty("m_iControlledBotEntIndex", st.PropertyValue{IntVal: 12})
	pl.demoInfoProvider = demoInfoProvider

	assert.Same(t, dave, pl.ControlledBot())
}

func TestPlayer_ControlledBotS2(t *testing.T) {
	dave := &Player{
		Name:  "Dave",
		IsBot: true,
	}
	demoInfoProvider := &demoInfoProviderMock{
		playersByHandle: map[uint64]*Player{
			12: dave,
		},
		isSource2: true,
	}

	pl := playerWithProperty("m_hOriginalControllerOfCurrentPawn", st.PropertyValue{Any: uint64(0)})
	pl.demoInfoProvider = demoInfoProvider

	assert.Nil(t, pl.ControlledBot())

	pl = playerWithProperty("m_hOriginalControllerOfCurrentPawn", st.PropertyValue{Any: uint64(12)})
	pl.demoInfoProvider = demoInfoProvider

	assert.Same(t, dave, pl.ControlledBot())
}

/*
TODO: fix for CS2
func TestPlayer_Armor(t *testing.T) {
	pl := playerWithProperty("m_ArmorValue", st.PropertyValue{IntVal: 95})

	assert.Equal(t, 95, pl.Armor())
}
*/

func TestPlayer_Money(t *testing.T) {
	pl := playerWithProperty("m_iAccount", st.PropertyValue{IntVal: 800})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Equal(t, 800, pl.Money())
}

func TestPlayer_ViewDirectionX(t *testing.T) {
	pl := playerWithProperty("m_angEyeAngles[1]", st.PropertyValue{FloatVal: 180})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Equal(t, float32(180), pl.ViewDirectionX())
}

func TestPlayer_ViewDirectionY(t *testing.T) {
	pl := playerWithProperty("m_angEyeAngles[0]", st.PropertyValue{FloatVal: 15})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Equal(t, float32(15), pl.ViewDirectionY())
}

func TestPlayer_Position(t *testing.T) {
	entity := new(stfake.Entity)
	pos := r3.Vector{X: 1, Y: 2, Z: 3}

	entity.On("Position").Return(pos)

	pl := &Player{Entity: entity}
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Equal(t, pos, pl.Position())
}

func TestPlayer_Position_EntityNil(t *testing.T) {
	pl := new(Player)
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Empty(t, pl.Position())
}

func TestPlayer_PositionEyes(t *testing.T) {
	entity := entityWithProperty("localdata.m_vecViewOffset[2]", st.PropertyValue{FloatVal: 2})
	pos := r3.Vector{X: 1, Y: 2, Z: 3}

	entity.On("Position").Return(pos)

	pl := &Player{Entity: entity}
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Equal(t, r3.Vector{X: 1, Y: 2, Z: 5}, pl.PositionEyes())
}

func TestPlayer_PositionEyes_EntityNil(t *testing.T) {
	pl := &Player{}
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Empty(t, pl.PositionEyes())
}

func TestPlayer_Velocity(t *testing.T) {
	entity := new(stfake.Entity)
	entity.On("PropertyValueMust", "localdata.m_vecVelocity[0]").Return(st.PropertyValue{FloatVal: 1})
	entity.On("PropertyValueMust", "localdata.m_vecVelocity[1]").Return(st.PropertyValue{FloatVal: 2})
	entity.On("PropertyValueMust", "localdata.m_vecVelocity[2]").Return(st.PropertyValue{FloatVal: 3})

	pl := &Player{Entity: entity}
	pl.demoInfoProvider = s1DemoInfoProvider

	expected := r3.Vector{X: 1, Y: 2, Z: 3}
	assert.Equal(t, expected, pl.Velocity())
}

func playerWithS2PawnPositionAndProperties(pos r3.Vector, props []fakeProp) *Player {
	controllerEntity := entityWithProperties([]fakeProp{
		{propName: "m_hPlayerPawn", value: st.PropertyValue{Any: uint64(1), S2: true}},
		{propName: "m_hPawn", value: st.PropertyValue{Any: uint64(1), S2: true}},
	})

	pawnEntity := entityWithProperties(props)
	pawnEntity.On("Position").Return(pos)

	pl := &Player{
		Entity: controllerEntity,
	}
	pl.demoInfoProvider = demoInfoProviderMock{
		isSource2:       true,
		networkProtocol: constants.NetworkProtocolAnimGraph2,
		entitiesByHandle: map[uint64]st.Entity{
			1: pawnEntity,
		},
	}

	return pl
}

func TestPlayer_VelocityS2_Networked(t *testing.T) {
	pl := playerWithS2PawnPositionAndProperties(r3.Vector{X: 20, Y: 300, Z: 100}, []fakeProp{
		{propName: "m_vecVelocity.m_vecX", value: st.PropertyValue{Any: float32(250), S2: true}},
		{propName: "m_vecVelocity.m_vecY", value: st.PropertyValue{Any: float32(-130.5), S2: true}},
		{propName: "m_vecVelocity.m_vecZ", value: st.PropertyValue{Any: float32(0), S2: true}},
	})
	pl.PreviousFramePosition = r3.Vector{X: 999, Y: 999, Z: 999} // must be ignored

	assert.Equal(t, r3.Vector{X: 250, Y: -130.5, Z: 0}, pl.Velocity())
}

// On demos recorded before the AnimGraph 2 update the m_vecVelocity.m_vec*
// properties resolve and read as 0 (baseline value) without ever being
// updated; the position delta must be used regardless.
func TestPlayer_PositionEyesS2(t *testing.T) {
	pl := playerWithS2PawnPositionAndProperties(r3.Vector{X: 120.5, Y: -33.25, Z: 14}, []fakeProp{
		{propName: "m_vecViewOffset.m_vecX", value: st.PropertyValue{Any: float32(1.5), S2: true}},
		{propName: "m_vecViewOffset.m_vecY", value: st.PropertyValue{Any: float32(-2.25), S2: true}},
		{propName: "m_vecViewOffset.m_vecZ", value: st.PropertyValue{Any: float32(64), S2: true}},
	})

	assert.Equal(t, r3.Vector{X: 122, Y: -35.5, Z: 78}, pl.PositionEyes())
}

func TestPlayer_PositionEyesS2_OffsetUnavailable(t *testing.T) {
	pos := r3.Vector{X: -10, Y: 5, Z: 42}
	pl := playerWithS2PawnPositionAndProperties(pos, nil)

	assert.Equal(t, pos, pl.PositionEyes())
}

func TestPlayer_Velocity_EntityNil(t *testing.T) {
	pl := new(Player)
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Empty(t, pl.Velocity())
}

func TestPlayer_ClanTag(t *testing.T) {
	pl := playerWithResourceProperty("m_szClan", st.PropertyValue{Any: "SuperClan"})
	pl.demoInfoProvider = demoInfoProviderMock{
		playerResourceEntity: (pl.demoInfoProvider.(demoInfoProviderMock)).playerResourceEntity,
		isSource2:            false,
	}

	assert.Equal(t, "SuperClan", pl.ClanTag())
}

func TestPlayer_CrosshairCode(t *testing.T) {
	pl := playerWithResourceProperty("m_szCrosshairCodes", st.PropertyValue{StringVal: "CSGO-jvnbx-S3xFK-iEJXD-Y27Nd-AO6FP"})

	assert.Equal(t, "CSGO-jvnbx-S3xFK-iEJXD-Y27Nd-AO6FP", pl.CrosshairCode())
}

func TestPlayer_WithoutCrosshairCode(t *testing.T) {
	pl := newPlayer(0)

	assert.Equal(t, pl.CrosshairCode(), "")
}

func TestPlayer_Ping(t *testing.T) {
	pl := playerWithResourceProperty("m_iPing", st.PropertyValue{IntVal: 45})

	assert.Equal(t, 45, pl.Ping())
}

func TestPlayer_Score(t *testing.T) {
	pl := playerWithResourceProperty("m_iScore", st.PropertyValue{IntVal: 10})

	assert.Equal(t, 10, pl.Score())
}

func TestPlayer_Color(t *testing.T) {
	pl := playerWithResourceProperty("m_iCompTeammateColor", st.PropertyValue{IntVal: int(Yellow)})

	assert.Equal(t, Yellow, pl.Color())

	pl = &Player{demoInfoProvider: demoInfoProviderMock{}}

	assert.Equal(t, Grey, pl.Color())

	pl = &Player{
		EntityID: 1,
		demoInfoProvider: demoInfoProviderMock{
			playerResourceEntity: entityWithoutProperty("m_iCompTeammateColor.001"),
		},
	}

	assert.Equal(t, Grey, pl.Color())
}

func TestPlayer_ColorOrErr(t *testing.T) {
	pl := playerWithResourceProperty("m_iCompTeammateColor", st.PropertyValue{IntVal: int(Yellow)})

	color, err := pl.ColorOrErr()
	assert.NoError(t, err)
	assert.Equal(t, Yellow, color)

	pl = &Player{demoInfoProvider: demoInfoProviderMock{}}

	color, err = pl.ColorOrErr()
	assert.ErrorIs(t, err, ErrDataNotAvailable)
	assert.EqualValues(t, Grey, color)

	pl = &Player{
		EntityID: 1,
		demoInfoProvider: demoInfoProviderMock{
			playerResourceEntity: entityWithoutProperty("m_iCompTeammateColor.001"),
		},
	}

	color, err = pl.ColorOrErr()
	assert.ErrorIs(t, err, ErrNotSupportedByDemo)
	assert.Equal(t, Grey, color)
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

func TestPlayer_TotalDamage(t *testing.T) {
	pl := playerWithResourceProperty("m_iMatchStats_Damage_Total", st.PropertyValue{IntVal: 2900})

	assert.Equal(t, 2900, pl.TotalDamage())
}

func TestPlayer_UtilityDamage(t *testing.T) {
	pl := playerWithResourceProperty("m_iMatchStats_UtilityDamage_Total", st.PropertyValue{IntVal: 420})

	assert.Equal(t, 420, pl.UtilityDamage())
}

func TestPlayer_SteamID32(t *testing.T) {
	pl := &Player{SteamID64: 76561198012952267}

	assert.Equal(t, uint32(52686539), pl.SteamID32())
}

func TestPlayer_LastPlaceName(t *testing.T) {
	pl := playerWithProperty("m_szLastPlaceName", st.PropertyValue{Any: "TopofMid"})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.Equal(t, "TopofMid", pl.LastPlaceName())
}

func TestPlayer_RankType(t *testing.T) {
	pl := playerWithResourceProperty("m_iCompetitiveRankType", st.PropertyValue{IntVal: 6})

	assert.Equal(t, 6, pl.RankType())
}

func TestPlayer_Rank(t *testing.T) {
	pl := playerWithResourceProperty("m_iCompetitiveRanking", st.PropertyValue{IntVal: 10})

	assert.Equal(t, 10, pl.Rank())
}

func TestPlayer_CompetitiveWins(t *testing.T) {
	pl := playerWithResourceProperty("m_iCompetitiveWins", st.PropertyValue{IntVal: 190})

	assert.Equal(t, 190, pl.CompetitiveWins())
}

func TestPlayer_IsGrabbingHostage(t *testing.T) {
	pl := playerWithProperty("m_bIsGrabbingHostage", st.PropertyValue{IntVal: 0})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.False(t, pl.IsGrabbingHostage())

	pl = playerWithProperty("m_bIsGrabbingHostage", st.PropertyValue{IntVal: 1})
	pl.demoInfoProvider = s1DemoInfoProvider

	assert.True(t, pl.IsGrabbingHostage())
}

func TestPlayer_ViewmodelOffsetS1(t *testing.T) {
	// Test CS:GO demo
	pl := newPlayer(0)
	assert.Equal(t, r3.Vector{}, pl.ViewmodelOffset())
}

func TestPlayer_ViewmodelOffsetS2(t *testing.T) {
	// Set up controller entity with pawn references
	controllerEntity := entityWithProperties([]fakeProp{
		{propName: "m_hPlayerPawn", value: st.PropertyValue{Any: uint64(1), S2: true}},
		{propName: "m_hPawn", value: st.PropertyValue{Any: uint64(1), S2: true}},
	})

	// Set up pawn entity with viewmodel offset properties
	pawnEntity := entityWithProperties([]fakeProp{
		{propName: "m_flViewmodelOffsetX", value: st.PropertyValue{FloatVal: -1.5}},
		{propName: "m_flViewmodelOffsetY", value: st.PropertyValue{FloatVal: 2.0}},
		{propName: "m_flViewmodelOffsetZ", value: st.PropertyValue{FloatVal: -0.5}},
	})

	pl := &Player{Entity: controllerEntity}
	pl.demoInfoProvider = demoInfoProviderMock{
		isSource2: true,
		entitiesByHandle: map[uint64]st.Entity{
			1: pawnEntity,
		},
	}

	assert.Equal(t, r3.Vector{X: -1.5, Y: 2.0, Z: -0.5}, pl.ViewmodelOffset())
}

func TestPlayer_ViewmodelFOVS1(t *testing.T) {
	// Test CS:GO demo (should return 0 even with property)
	pl := playerWithProperty("m_flViewmodelFOV", st.PropertyValue{FloatVal: 60})
	pl.demoInfoProvider = s1DemoInfoProvider
	assert.Equal(t, float32(0), pl.ViewmodelFOV())
}

func TestPlayer_ViewmodelFOVS2(t *testing.T) {
	// Set up controller entity with pawn references
	controllerEntity := entityWithProperties([]fakeProp{
		{propName: "m_hPlayerPawn", value: st.PropertyValue{Any: uint64(1), S2: true}},
		{propName: "m_hPawn", value: st.PropertyValue{Any: uint64(1), S2: true}},
	})

	// Set up pawn entity with viewmodel FOV property
	pawnEntity := entityWithProperty("m_flViewmodelFOV", st.PropertyValue{FloatVal: 60})

	pl := &Player{Entity: controllerEntity}
	pl.demoInfoProvider = demoInfoProviderMock{
		isSource2: true,
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

func playerWithResourceProperty(propName string, value st.PropertyValue) *Player {
	return &Player{
		EntityID: 1,
		demoInfoProvider: demoInfoProviderMock{
			playerResourceEntity: entityWithProperty(propName+".001", value),
		},
	}
}

// velocityFixture is a player whose pawn position / tick can be changed between frames,
// to test the position delta fallback of Player.Velocity().
type velocityFixture struct {
	pl       *Player
	provider *demoInfoProviderMock
	pawnID   int
	pawn     *stfake.Entity // re-used across frames while pawnID doesn't change, set to nil to re-create the pawn
	pawnOfID int            // pawnID the pawn entity was created for
	pos      r3.Vector
	interp   uint64
	health   int32
	velocity *r3.Vector // networked m_vecVelocity, nil = declared but not populated (pre AnimGraph 2)
}

const velocityFixtureControllerID = 2 // pawn handle is 1

func newVelocityFixture(networkProtocol int) *velocityFixture {
	h := uint64(1)
	f := &velocityFixture{
		provider: &demoInfoProviderMock{
			isSource2:        true,
			tickRate:         64,
			networkProtocol:  networkProtocol,
			tracked:          &[]trackedRecorder{},
			entitiesByHandle: map[uint64]st.Entity{},
		},
		pawnID: 1,
		health: 100,
	}

	controller := entityWithProperties([]fakeProp{
		{propName: "m_hPawn", value: st.PropertyValue{Any: h, S2: true}},
		{propName: "m_hPlayerPawn", value: st.PropertyValue{Any: h, S2: true}},
	})
	controller.ExpectedCalls = slices.DeleteFunc(controller.ExpectedCalls, func(c *mock.Call) bool {
		return c.Method == "ID"
	})
	controller.On("ID").Return(velocityFixtureControllerID)

	// the controller must be found by its entity ID, otherwise the recorder stops (disconnect)
	f.provider.entitiesByHandle[velocityFixtureControllerID] = controller

	f.pl = &Player{
		Entity:           controller,
		EntityID:         velocityFixtureControllerID,
		demoInfoProvider: f.provider,
	}

	f.update()

	return f
}

// update rebuilds the fake pawn from the fixture's current state.
func (f *velocityFixture) update() {
	props := []fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{Any: f.health, S2: true}},
		{propName: "m_lifeState", value: st.PropertyValue{Any: uint64(map[bool]int{true: 0, false: 2}[f.health > 0]), S2: true}},
		{propName: "m_ubInterpolationFrame", value: st.PropertyValue{Any: f.interp, S2: true}},
	}

	if f.velocity != nil {
		props = append(props,
			fakeProp{propName: "m_vecVelocity.m_vecX", value: st.PropertyValue{Any: float32(f.velocity.X), S2: true}},
			fakeProp{propName: "m_vecVelocity.m_vecY", value: st.PropertyValue{Any: float32(f.velocity.Y), S2: true}},
			fakeProp{propName: "m_vecVelocity.m_vecZ", value: st.PropertyValue{Any: float32(f.velocity.Z), S2: true}},
		)
	} else {
		// pre AnimGraph 2 demos declare the property, but it's never populated
		props = append(props,
			fakeProp{propName: "m_vecVelocity.m_vecX", value: st.PropertyValue{}},
			fakeProp{propName: "m_vecVelocity.m_vecY", value: st.PropertyValue{}},
			fakeProp{propName: "m_vecVelocity.m_vecZ", value: st.PropertyValue{}},
		)
	}

	if f.pawn == nil || f.pawnOfID != f.pawnID {
		f.pawn = new(stfake.Entity)
		f.pawnOfID = f.pawnID
	}

	pawn := f.pawn
	pawn.ExpectedCalls = nil

	pawn.On("ID").Return(f.pawnID)
	pawn.On("Position").Return(f.pos)

	for _, prop := range props {
		pawn.On("PropertyValue", prop.propName).Return(prop.value, true)
		pawn.On("PropertyValueMust", prop.propName).Return(prop.value)
	}

	pawn.On("PropertyValue", mock.Anything).Return(st.PropertyValue{}, false)

	f.provider.entitiesByHandle[1] = pawn
}

// frame simulates a parsed frame with an entity snapshot at the given (server) tick with the given pawn position
// and returns the velocity as seen during that frame (e.g. by a FrameDone handler).
func (f *velocityFixture) frame(tick int, pos r3.Vector) r3.Vector {
	f.provider.snapshotTick = tick
	f.pos = pos

	return f.frameNoSnapshot(tick)
}

// frameNoSnapshot simulates a parsed frame without an entity snapshot (e.g. a user command frame of a POV demo),
// which may still advance the in-game tick.
func (f *velocityFixture) frameNoSnapshot(ingameTick int) r3.Vector {
	f.provider.ingameTick = ingameTick
	f.update()

	v := f.pl.Velocity()

	// end of frame: the parser runs the registered position recorders, dropping those that return false
	*f.provider.tracked = slices.DeleteFunc(*f.provider.tracked, func(r trackedRecorder) bool {
		return !r.record()
	})

	return v
}

func assertVectorInDelta(t *testing.T, expected, actual r3.Vector) {
	t.Helper()

	assert.InDelta(t, expected.X, actual.X, 1e-6)
	assert.InDelta(t, expected.Y, actual.Y, 1e-6)
	assert.InDelta(t, expected.Z, actual.Z, 1e-6)
}

func TestPlayer_Velocity_PositionDelta(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	// first frame of a player - no history, no spike
	assert.Equal(t, r3.Vector{}, f.frame(100, r3.Vector{X: 1000, Y: -2000, Z: 50}))

	// 3 ticks per frame (like older GOTV demos) at 64 tick: 12 units / (3/64 s) = 256 u/s
	assertVectorInDelta(t, r3.Vector{X: 256, Y: -128, Z: 0}, f.frame(103, r3.Vector{X: 1012, Y: -2006, Z: 50}))

	// 1 tick per frame
	assertVectorInDelta(t, r3.Vector{X: 192, Y: 0, Z: -64}, f.frame(104, r3.Vector{X: 1015, Y: -2006, Z: 49}))

	// standing still
	assert.Equal(t, r3.Vector{}, f.frame(105, r3.Vector{X: 1015, Y: -2006, Z: 49}))
}

// Positions are recorded lazily: the first Velocity() call registers the player (once) and returns zero.
func TestPlayer_Velocity_PositionDelta_LazyTracking(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	assert.Empty(t, *f.provider.tracked)

	f.provider.snapshotTick = 100
	assert.Equal(t, r3.Vector{}, f.pl.Velocity())
	assert.Equal(t, r3.Vector{}, f.pl.Velocity())

	require.Len(t, *f.provider.tracked, 1, "registered exactly once")
	assert.True(t, (*f.provider.tracked)[0].lazy)
}

// Demos with networked velocity never record positions.
func TestPlayer_Velocity_Networked_NoTracking(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2)
	v := r3.Vector{X: 1}
	f.velocity = &v
	f.update()

	assert.Equal(t, v, f.pl.Velocity())
	assert.Empty(t, *f.provider.tracked)
}

// ParserConfig.EagerVelocityTracking registers players at creation (not lazily), only on older demos.
func TestNewPlayer_EagerPositionTracking(t *testing.T) {
	cases := []struct {
		name     string
		protocol int
		flag     bool
		tracked  bool
	}{
		{"flag on, old demo", constants.NetworkProtocolAnimGraph2 - 1, true, true},
		{"flag on, new demo", constants.NetworkProtocolAnimGraph2, true, false},
		{"flag off, old demo", constants.NetworkProtocolAnimGraph2 - 1, false, false},
		{"flag on, unknown protocol", 0, true, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tracked := []trackedRecorder{}
			NewPlayer(demoInfoProviderMock{isSource2: true, networkProtocol: c.protocol, eagerVelocityTracking: c.flag, tracked: &tracked})

			if c.tracked {
				require.Len(t, tracked, 1)
				assert.False(t, tracked[0].lazy)
			} else {
				assert.Empty(t, tracked)
			}
		})
	}
}

func TestPlayer_Velocity_PositionDelta_FrameWithoutTickAdvance(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	assertVectorInDelta(t, r3.Vector{X: 128}, f.frame(101, r3.Vector{X: 2}))

	// same tick again, delta must be computed against the last frame of an earlier tick, not divided by zero
	assertVectorInDelta(t, r3.Vector{X: 128}, f.frame(101, r3.Vector{X: 2}))
	assertVectorInDelta(t, r3.Vector{X: 64}, f.frame(102, r3.Vector{X: 3}))
}

// POV demos have user command frames that advance the in-game tick without an entity snapshot,
// and the in-game tick jitters against the snapshots' server ticks. Time is measured in server ticks.
func TestPlayer_Velocity_PositionDelta_FrameWithoutSnapshot(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	assertVectorInDelta(t, r3.Vector{X: 128}, f.frame(101, r3.Vector{X: 2}))

	// in-game tick advanced, but no snapshot - keep the last velocity instead of reporting a standstill
	assertVectorInDelta(t, r3.Vector{X: 128}, f.frameNoSnapshot(102))
	assertVectorInDelta(t, r3.Vector{X: 128}, f.frameNoSnapshot(103))

	// the next snapshot, one server tick later - regardless of the in-game tick
	f.provider.snapshotTick = 102
	f.pos = r3.Vector{X: 5}
	assertVectorInDelta(t, r3.Vector{X: 192}, f.frameNoSnapshot(103))
	assertVectorInDelta(t, r3.Vector{X: 192}, f.frameNoSnapshot(103))

	// the next snapshot doesn't update the pawn - standing still
	assert.Equal(t, r3.Vector{}, f.frame(103, r3.Vector{X: 5}))
	assert.Equal(t, r3.Vector{}, f.frameNoSnapshot(104))
}

// Events like WeaponFire are dispatched while a snapshot is applied, when the pawn's position may or may not be
// updated yet - the velocity between the last two samples is used then.
func TestPlayer_Velocity_PositionDelta_DuringSnapshot(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})

	// the next snapshot is being applied, the pawn isn't updated yet - not a standstill
	f.provider.snapshotTick = 102
	f.provider.processingSnapshot = true
	assertVectorInDelta(t, r3.Vector{X: 64}, f.pl.Velocity())

	// updated within the same snapshot - still the last value, consistently for all players
	f.pos = r3.Vector{X: 3}
	f.update()
	assertVectorInDelta(t, r3.Vector{X: 64}, f.pl.Velocity())

	// after the snapshot
	f.provider.processingSnapshot = false
	assertVectorInDelta(t, r3.Vector{X: 128}, f.frameNoSnapshot(102))
}

// The recorder skips the (relatively expensive) position lookup in frames without an entity snapshot.
func TestPlayer_RecordFramePosition_SkipsFramesWithoutSnapshot(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})

	history := f.pl.positionHistory

	f.pos = r3.Vector{X: 1000} // must not be looked up
	f.frameNoSnapshot(102)
	f.frameNoSnapshot(103)

	assert.Equal(t, history, f.pl.positionHistory)
}

func TestIsPawnAlive_LifeStateTypes(t *testing.T) {
	assert.True(t, isPawnAlive(entityWithProperty("m_lifeState", st.PropertyValue{Any: uint64(0), S2: true})))
	assert.False(t, isPawnAlive(entityWithProperty("m_lifeState", st.PropertyValue{Any: uint64(2), S2: true})))
	assert.False(t, isPawnAlive(entityWithProperty("m_lifeState", st.PropertyValue{Any: int32(2), S2: true})))
	assert.False(t, isPawnAlive(entityWithProperty("m_lifeState", st.PropertyValue{Any: uint32(2), S2: true})))
	assert.True(t, isPawnAlive(entityWithProperty("m_lifeState", st.PropertyValue{})))
	assert.True(t, isPawnAlive(entityWithProperty("m_lifeState", st.PropertyValue{Any: "unexpected", S2: true})))
}

func TestPlayer_Velocity_PositionDelta_Teleport(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})

	// teleport flagged by m_ubInterpolationFrame, small enough to look like a plausible movement
	f.interp++
	assert.Equal(t, r3.Vector{}, f.frame(102, r3.Vector{X: 30}))

	// back to normal on the next frame
	assertVectorInDelta(t, r3.Vector{X: 64}, f.frame(103, r3.Vector{X: 31}))

	// not flagged, but physically impossible (> sv_maxvelocity)
	assert.Equal(t, r3.Vector{}, f.frame(104, r3.Vector{X: 31 + 100}))
}

func TestPlayer_Velocity_PositionDelta_Respawn(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})

	// died this frame - the history is cleared at the end of it
	f.health = 0
	f.frame(102, r3.Vector{X: 2})
	assert.Equal(t, r3.Vector{}, f.frame(103, r3.Vector{X: 2}))

	// respawned elsewhere - first frame after respawn is zero, not a spike
	f.health = 100
	assert.Equal(t, r3.Vector{}, f.frame(104, r3.Vector{X: 50}))
	assertVectorInDelta(t, r3.Vector{X: 64}, f.frame(105, r3.Vector{X: 51}))
}

func TestPlayer_Velocity_PositionDelta_PawnChanged(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})

	f.pawnID = 2
	assert.Equal(t, r3.Vector{}, f.frame(102, r3.Vector{X: 2}))
	assertVectorInDelta(t, r3.Vector{X: 64}, f.frame(103, r3.Vector{X: 3}))
}

// A pawn destroyed and re-created in the same entity slot between two samples is a different pawn.
func TestPlayer_Velocity_PositionDelta_PawnRecreatedInSameSlot(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})

	f.pawn = nil // same ID, new entity
	assert.Equal(t, r3.Vector{}, f.frame(102, r3.Vector{X: 2}))
	assertVectorInDelta(t, r3.Vector{X: 64}, f.frame(103, r3.Vector{X: 3}))
}

func TestPlayer_Velocity_PositionDelta_TickWentBackwards(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})

	assert.Equal(t, r3.Vector{}, f.frame(50, r3.Vector{X: 2}))
	assertVectorInDelta(t, r3.Vector{X: 64}, f.frame(51, r3.Vector{X: 3}))
}

func TestPlayer_Velocity_PositionDelta_NoTickRate(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)
	f.provider.tickRate = 0

	f.frame(100, r3.Vector{})
	assert.Equal(t, r3.Vector{}, f.frame(101, r3.Vector{X: 1}))
}

// Since AnimGraph 2 the networked value is used, even when it's zero (standing still) -
// the source must not depend on the value.
func TestPlayer_Velocity_AnimGraph2_UsesNetworkedValue(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2)
	f.velocity = &r3.Vector{X: 250, Y: -10, Z: 0}

	assertVectorInDelta(t, r3.Vector{X: 250, Y: -10}, f.frame(100, r3.Vector{}))
	assertVectorInDelta(t, r3.Vector{X: 250, Y: -10}, f.frame(101, r3.Vector{X: 3.9}))

	f.velocity = &r3.Vector{}
	assert.Equal(t, r3.Vector{}, f.frame(102, r3.Vector{X: 10}))
}

// If a demo after AnimGraph 2 doesn't populate m_vecVelocity for a pawn, fall back to the position delta.
func TestPlayer_Velocity_AnimGraph2_NotPopulated(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2)

	f.frame(100, r3.Vector{})
	assertVectorInDelta(t, r3.Vector{Y: 128}, f.frame(101, r3.Vector{Y: 2}))
}

// Before AnimGraph 2 the property may hold stale values (e.g. on POV demos for non-recording players),
// they must be ignored.
func TestPlayer_Velocity_BeforeAnimGraph2_IgnoresNetworkedValue(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)
	f.velocity = &r3.Vector{X: 999}

	f.frame(100, r3.Vector{})
	assertVectorInDelta(t, r3.Vector{Y: 64}, f.frame(101, r3.Vector{Y: 1}))
}

// On POV demos before AnimGraph 2 the recording player's networked velocity is exact and used as-is,
// without recording positions.
func TestPlayer_Velocity_BeforeAnimGraph2_RecordingPlayer(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)
	f.provider.pov = true
	f.provider.recordingSlot = velocityFixtureControllerID - 1
	f.velocity = &r3.Vector{X: 250, Y: -10}

	assertVectorInDelta(t, r3.Vector{X: 250, Y: -10}, f.frame(100, r3.Vector{}))
	assertVectorInDelta(t, r3.Vector{X: 250, Y: -10}, f.frame(101, r3.Vector{Y: 1}))
	assert.Empty(t, *f.provider.tracked)
}

// Other players on the same POV demo hold stale values and use the position delta.
func TestPlayer_Velocity_BeforeAnimGraph2_NonRecordingPlayer(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)
	f.provider.pov = true
	f.provider.recordingSlot = velocityFixtureControllerID // a different slot
	f.velocity = &r3.Vector{X: 999}

	f.frame(100, r3.Vector{})
	assertVectorInDelta(t, r3.Vector{Y: 64}, f.frame(101, r3.Vector{Y: 1}))
}

// An unknown network protocol (0, e.g. no file header yet) is decided by property presence alone.
func TestPlayer_Velocity_UnknownProtocol(t *testing.T) {
	f := newVelocityFixture(0)
	f.velocity = &r3.Vector{X: 250}

	assertVectorInDelta(t, r3.Vector{X: 250}, f.frame(100, r3.Vector{}))
	assert.Empty(t, *f.provider.tracked)

	f.velocity = nil
	f.frame(101, r3.Vector{})
	assertVectorInDelta(t, r3.Vector{Y: 64}, f.frame(102, r3.Vector{Y: 1}))
}

// Once the controller entity is gone (disconnect), the recorder is dropped and the history cleared.
// A later Velocity() call registers again.
func TestPlayer_Velocity_PositionDelta_ControllerDestroyed(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)

	f.frame(100, r3.Vector{})
	f.frame(101, r3.Vector{X: 1})
	require.Len(t, *f.provider.tracked, 1)

	controller := f.provider.entitiesByHandle[velocityFixtureControllerID]
	delete(f.provider.entitiesByHandle, velocityFixtureControllerID)

	f.frame(102, r3.Vector{X: 2})
	assert.Empty(t, *f.provider.tracked, "recorder dropped")
	assert.False(t, f.pl.trackingPosition)
	assert.Equal(t, [2]positionSample{}, f.pl.positionHistory)

	f.provider.entitiesByHandle[velocityFixtureControllerID] = controller

	assert.Equal(t, r3.Vector{}, f.frame(103, r3.Vector{X: 3}))
	require.Len(t, *f.provider.tracked, 1, "registered again")
	assertVectorInDelta(t, r3.Vector{X: 64}, f.frame(104, r3.Vector{X: 4}))
}

// With ParserConfig.EagerVelocityTracking a re-registration (after the recorder was dropped) isn't lazy (no warning).
func TestPlayer_Velocity_PositionDelta_ReRegistrationWithFlag(t *testing.T) {
	f := newVelocityFixture(constants.NetworkProtocolAnimGraph2 - 1)
	f.provider.eagerVelocityTracking = true

	f.frame(100, r3.Vector{})
	require.Len(t, *f.provider.tracked, 1)

	controller := f.provider.entitiesByHandle[velocityFixtureControllerID]
	delete(f.provider.entitiesByHandle, velocityFixtureControllerID)

	f.frame(101, r3.Vector{})
	require.Empty(t, *f.provider.tracked, "recorder dropped")

	f.provider.entitiesByHandle[velocityFixtureControllerID] = controller

	f.frame(102, r3.Vector{})
	require.Len(t, *f.provider.tracked, 1, "registered again")
	assert.False(t, (*f.provider.tracked)[0].lazy)
}
