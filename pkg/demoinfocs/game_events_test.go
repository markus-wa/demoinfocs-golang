package demoinfocs

import (
	"crypto/rand"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables/fake"
)

// See #90
func TestRoundEnd_LoserState_Score(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.disableMimicSource1GameEvents = true

	p.gameState.tState.Entity = stfake.NewEntityWithProperty("m_scoreTotal", st.PropertyValue{Any: 1})
	p.gameState.ctState.Entity = stfake.NewEntityWithProperty("m_scoreTotal", st.PropertyValue{Any: 2})
	eventOccurred := 0

	p.RegisterEventHandler(func(e events.RoundEnd) {
		eventOccurred++
		assert.Equal(t, e, events.RoundEnd{
			Winner:      common.TeamTerrorists,
			WinnerState: p.GameState().TeamTerrorists(),
			LoserState:  p.GameState().TeamCounterTerrorists(),
			Message:     "test",
			Reason:      events.RoundEndReasonTerroristsWin,
		})
	})

	p.gameEventDescs = map[int32]*msg.CMsgSource1LegacyGameEventListDescriptorT{
		1: {
			Name: proto.String("round_end"),
			Keys: []*msg.CMsgSource1LegacyGameEventListKeyT{
				{Name: proto.String("winner")},
				{Name: proto.String("message")},
				{Name: proto.String("reason")},
			},
		},
	}

	ge := new(msg.CMsgSource1LegacyGameEvent)
	ge.Eventid = proto.Int32(1)
	ge.EventName = proto.String("round_end")
	ge.Keys = []*msg.CMsgSource1LegacyGameEventKeyT{
		{ValByte: proto.Int32(2)},
		{ValString: proto.String("test")},
		{ValByte: proto.Int32(9)},
	}
	p.handleGameEvent(ge)

	assert.Equal(t, 1, eventOccurred)
}

func TestGetPlayerWeapon_NilPlayer(t *testing.T) {
	wep := getPlayerWeapon(nil, common.EqAK47, "", 1)

	assert.NotNil(t, wep)
	assert.Equal(t, common.EqAK47, wep.Type)
}

func TestGetPlayerWeapon_Found(t *testing.T) {
	ak := &common.Equipment{Type: common.EqAK47}
	pl := &common.Player{
		Inventory: map[int]*common.Equipment{
			1: ak,
		},
	}

	wep := getPlayerWeapon(pl, common.EqAK47, "ak47", 1)

	assert.True(t, wep == ak)
	assert.Equal(t, "ak47", wep.OriginalString)
}

func TestGetPlayerWeapon_NotFound(t *testing.T) {
	ak := &common.Equipment{Type: common.EqAK47}
	pl := &common.Player{
		Inventory: map[int]*common.Equipment{
			1: ak,
		},
	}

	wep := getPlayerWeapon(pl, common.EqM4A1, "", 1)

	assert.Equal(t, common.EqM4A1, wep.Type)
}

func TestGetPlayerWeapon_OriginalString(t *testing.T) {
	wep := getPlayerWeapon(nil, common.EqKnife, "knife_karambit", 1)

	assert.Equal(t, common.EqKnife, wep.Type)
	assert.Equal(t, "knife_karambit", wep.OriginalString)
}

func TestGetPlayerWeapon_OriginalString_NonKnife(t *testing.T) {
	wep := getPlayerWeapon(nil, common.EqAK47, "ak47", 1)

	assert.Equal(t, "ak47", wep.OriginalString)
}

func TestAddThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	he := common.NewEquipment(common.EqHE, 1)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(nil, he)

	assert.Empty(t, p.gameState.thrownGrenades)
}

func TestAddThrownGrenade(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, 1)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades)
	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
	assert.Equal(t, p.gameState.thrownGrenades[pl][common.EqHE][0], he)
}

func TestGetThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	he := common.NewEquipment(common.EqHE, 1)

	wep := p.gameEventHandler.getThrownGrenade(nil, he.Type)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}

	he := common.NewEquipment(common.EqSmoke, 1)

	wep := p.gameEventHandler.getThrownGrenade(pl, he.Type)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, 1)

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getThrownGrenade(pl, he.Type)

	assert.Equal(t, wep.Type, he.Type)
	assert.Equal(t, wep, he)
}

// TestGetThrownGrenade_CircularControlledBot verifies that getThrownGrenade doesn't
// infinitely recurse when ControlledBot() creates a circular reference between two players (see #620).
func TestGetThrownGrenade_CircularControlledBot(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	provider := demoInfoProvider{parser: p}

	playerA := common.NewPlayer(provider)
	playerA.SteamID64 = 1
	playerB := common.NewPlayer(provider)
	playerB.SteamID64 = 2

	// Add players to the game state indexed by entity ID so FindPlayerByHandle can find them.
	p.gameState.playersByEntityID[1] = playerA
	p.gameState.playersByEntityID[2] = playerB

	// Create circular bot references: playerA.ControlledBot() returns playerB and vice versa.
	// ControlledBot() reads "m_hOriginalControllerOfCurrentPawn" from the entity and calls
	// FindPlayerByHandle. entityIDFromHandle(2) == 2, entityIDFromHandle(1) == 1.
	playerA.Entity = stfake.NewEntityWithProperty("m_hOriginalControllerOfCurrentPawn", st.PropertyValue{Any: uint64(2)})
	playerB.Entity = stfake.NewEntityWithProperty("m_hOriginalControllerOfCurrentPawn", st.PropertyValue{Any: uint64(1)})

	smoke := common.NewEquipment(common.EqSmoke, 1)
	he := common.NewEquipment(common.EqHE, 1)

	// Should not panic or infinite loop - returns nil since no grenade was added
	wep := p.gameEventHandler.getThrownGrenade(playerA, smoke.Type)
	assert.Nil(t, wep)

	// Should find a grenade stored for playerB when searching via playerA
	p.gameEventHandler.addThrownGrenade(playerB, smoke)
	wep = p.gameEventHandler.getThrownGrenade(playerA, smoke.Type)
	assert.Equal(t, smoke, wep)

	// Should also find a grenade stored for playerA when searching via playerB (different grenade type)
	p.gameEventHandler.addThrownGrenade(playerA, he)
	wep = p.gameEventHandler.getThrownGrenade(playerB, he.Type)
	assert.Equal(t, he, wep)
}

func TestDeleteThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	he := common.NewEquipment(common.EqHE, 1)

	// Do nothing, we just keep sure it doesn't crash
	p.gameEventHandler.deleteThrownGrenade(nil, he.Type)
}

func TestDeleteThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, 1)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, common.EqSmoke)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
}

func TestDeleteThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, 1)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, he.Type)

	assert.Empty(t, p.gameState.thrownGrenades[pl])
}

func TestGetEquipmentInstance_NotGrenade(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqAK47, "")

	assert.Equal(t, common.EqAK47, wep.Type)
}

func TestGetEquipmentInstance_Grenade_NotThrown(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqSmoke, "")

	assert.Nil(t, wep)
}

func TestGetEquipmentInstance_Grenade_Thrown(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, 1)

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getEquipmentInstance(pl, he.Type, "")

	assert.Equal(t, he, wep)
}

func TestAttackerWeaponType_UnknownStaysUnknownWithoutContext(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.currentFrame = 24

	wepType := p.gameEventHandler.attackerWeaponType(common.EqUnknown, 123)

	assert.Equal(t, common.EqUnknown, wepType)
}

func TestAttackerWeaponType_FallDamageWins(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.currentFrame = 36
	p.gameEventHandler.userIDToFallDamageFrame[123] = p.currentFrame

	wepType := p.gameEventHandler.attackerWeaponType(common.EqUnknown, 123)

	assert.Equal(t, common.EqWorld, wepType)
}

func TestAttackerWeaponType_RoundEndReasonTargetBombedWins(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.currentFrame = 48
	p.gameEventHandler.frameToRoundEndReason[p.currentFrame] = events.RoundEndReasonTargetBombed

	wepType := p.gameEventHandler.attackerWeaponType(common.EqUnknown, 123)

	assert.Equal(t, common.EqBomb, wepType)
}

func TestPlayerHurt_UnknownWeaponDefaultsToWorld(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.currentFrame = 60

	var got []events.PlayerHurt
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		got = append(got, e)
	})

	p.gameEventHandler.playerHurt(playerHurtEventData(11, 65535, ""))

	assert.Len(t, got, 1)
	assert.NotNil(t, got[0].Weapon)
	assert.Equal(t, common.EqWorld, got[0].Weapon.Type)
}

func TestPlayerHurt_UnknownWeaponUsesBombWhenBombExplodedThisFrame(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.currentFrame = 72
	p.gameEventHandler.frameToBombExploded[p.currentFrame] = true

	var got []events.PlayerHurt
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		got = append(got, e)
	})

	p.gameEventHandler.playerHurt(playerHurtEventData(12, 65535, ""))

	assert.Len(t, got, 1)
	assert.NotNil(t, got[0].Weapon)
	assert.Equal(t, common.EqBomb, got[0].Weapon.Type)
}

func TestPlayerHurt_KnownWeaponDispatchesImmediately(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.currentFrame = 84

	var got []events.PlayerHurt
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		got = append(got, e)
	})

	p.gameEventHandler.playerHurt(playerHurtEventData(13, 7, "ak47"))

	assert.Len(t, got, 1)
	assert.NotNil(t, got[0].Weapon)
	assert.Equal(t, common.EqAK47, got[0].Weapon.Type)
}

func TestPlayerHurt_FatalHitCappedAtPriorHitHealthSameTick(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.currentFrame = 100
	p.gameState.ingameTick = 5000

	victim := newPlayerWithEntityID(3)
	victim.UserID = 42

	p.gameState.playersByUserID[victim.UserID] = victim

	var got []events.PlayerHurt
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		got = append(got, e)
	})

	// First hit: non-fatal, victim reported at 35 HP after the hit.
	data := playerHurtEventData(int32(victim.UserID), 1, "ak47")
	data["health"].ValByte = proto.Int32(35)
	data["armor"].ValByte = proto.Int32(50)
	data["dmg_health"].ValShort = proto.Int32(15)

	p.gameEventHandler.playerHurt(data)

	// Second hit, same tick: fatal (health 0). Must be capped at the victim's remaining
	// HP right before it (35), not the start-of-tick value (100) which would over-report.
	data2 := playerHurtEventData(int32(victim.UserID), 1, "ak47")
	data2["health"].ValByte = proto.Int32(0)
	data2["armor"].ValByte = proto.Int32(50)
	data2["dmg_health"].ValShort = proto.Int32(35)

	p.gameEventHandler.playerHurt(data2)

	assert.Len(t, got, 2)
	assert.Equal(t, 15, got[0].HealthDamageTaken)
	assert.Equal(t, 35, got[1].HealthDamageTaken)
}

func playerHurtEventData(userID int32, attacker int32, weapon string) map[string]*msg.CMsgSource1LegacyGameEventKeyT {
	return map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"userid": {
			ValShort: proto.Int32(userID),
		},
		"attacker": {
			ValShort: proto.Int32(attacker),
		},
		"weapon": {
			ValString: proto.String(weapon),
		},
		"health": {
			ValByte: proto.Int32(92),
		},
		"armor": {
			ValByte: proto.Int32(0),
		},
		"dmg_health": {
			ValShort: proto.Int32(8),
		},
		"dmg_armor": {
			ValByte: proto.Int32(0),
		},
		"hitgroup": {
			ValByte: proto.Int32(int32(events.HitGroupGeneric)),
		},
	}
}

func TestGetCommunityId(t *testing.T) {
	xuid, err := guidToSteamID64("STEAM_0:1:26343269")
	assert.Nil(t, err)
	assert.Equal(t, uint64(76561198012952267), xuid)
}

func TestEntityKilled_DispatchesOtherDeathForNonPlayerEntity(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)

	killed := new(stfake.Entity)
	killed.On("ID").Return(5)
	killed.On("ServerClass").Return(serverClassStub{name: "CDynamicProp"})
	killed.On("Position").Return(r3.Vector{X: 1, Y: 2, Z: 3})

	p.gameState.entities[5] = killed

	inflictor := new(stfake.Entity)
	inflictor.On("ServerClass").Return(serverClassStub{name: "CInferno"})

	p.gameState.entities[12] = inflictor

	var got []events.OtherDeath
	p.RegisterEventHandler(func(e events.OtherDeath) {
		got = append(got, e)
	})

	p.gameEventHandler.entityKilled(map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"entindex_killed":    {ValLong: proto.Int32(5)},
		"entindex_inflictor": {ValLong: proto.Int32(12)},
		"entindex_attacker":  {ValLong: proto.Int32(0)},
	})

	assert.Len(t, got, 1)
	assert.Equal(t, "CDynamicProp", got[0].OtherType)
	assert.Equal(t, int32(5), got[0].OtherID)
	assert.Equal(t, r3.Vector{X: 1, Y: 2, Z: 3}, got[0].OtherPosition)
	assert.Equal(t, "CInferno", got[0].InflictorType)
}

func TestEntityKilled_SkipsPlayerPawns(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)

	for _, className := range []string{"CCSPlayerPawn", "CCSPlayerPawnBase"} {
		killed := new(stfake.Entity)
		killed.On("ServerClass").Return(serverClassStub{name: className})

		p.gameState.entities[5] = killed

		var got []events.OtherDeath
		p.RegisterEventHandler(func(e events.OtherDeath) {
			got = append(got, e)
		})

		p.gameEventHandler.entityKilled(map[string]*msg.CMsgSource1LegacyGameEventKeyT{
			"entindex_killed":    {ValLong: proto.Int32(5)},
			"entindex_inflictor": {ValLong: proto.Int32(0)},
			"entindex_attacker":  {ValLong: proto.Int32(0)},
		})

		assert.Empty(t, got, "player-pawn deaths should be surfaced via player_death, not OtherDeath")
	}
}

func TestEntityKilled_UnknownEntityIsSkipped(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)

	var got []events.OtherDeath
	p.RegisterEventHandler(func(e events.OtherDeath) {
		got = append(got, e)
	})

	p.gameEventHandler.entityKilled(map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"entindex_killed":    {ValLong: proto.Int32(999)},
		"entindex_inflictor": {ValLong: proto.Int32(0)},
		"entindex_attacker":  {ValLong: proto.Int32(0)},
	})

	assert.Empty(t, got)
}

type serverClassStub struct {
	st.ServerClass
	name string
}

func (s serverClassStub) Name() string {
	return s.name
}

func TestGetCommunityId_BOT(t *testing.T) {
	xuid, err := guidToSteamID64("BOT")
	assert.Zero(t, xuid)
	assert.Nil(t, err)
}

func TestGetCommunityId_Error(t *testing.T) {
	_, err := guidToSteamID64("STEAM_0:1:abc")
	assert.NotNil(t, err)
	assert.Equal(t, "strconv.ParseUint: parsing \"abc\": invalid syntax", err.Error())
}
