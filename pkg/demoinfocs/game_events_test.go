package demoinfocs

import (
	"crypto/rand"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/constants"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

// See #90
func TestRoundEnd_LoserState_Score(t *testing.T) {
	p := NewParser(rand.Reader)
	p.disableMimicSource1GameEvents = true

	p.gameState.tState.Entity = stfake.NewEntity(1, stfake.P("m_scoreTotal", int32(1)))
	p.gameState.ctState.Entity = stfake.NewEntity(2, stfake.P("m_scoreTotal", int32(2)))
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
	wep := getPlayerWeapon(nil, common.EqAK47, "", common.UniqueID(1))

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

	wep := getPlayerWeapon(pl, common.EqAK47, "ak47", common.UniqueID(1))

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

	wep := getPlayerWeapon(pl, common.EqM4A1, "", common.UniqueID(1))

	assert.Equal(t, common.EqM4A1, wep.Type)
}

func TestGetPlayerWeapon_OriginalString(t *testing.T) {
	wep := getPlayerWeapon(nil, common.EqKnife, "knife_karambit", common.UniqueID(1))

	assert.Equal(t, common.EqKnife, wep.Type)
	assert.Equal(t, "knife_karambit", wep.OriginalString)
}

func TestGetPlayerWeapon_OriginalString_NonKnife(t *testing.T) {
	wep := getPlayerWeapon(nil, common.EqAK47, "ak47", common.UniqueID(1))

	assert.Equal(t, "ak47", wep.OriginalString)
}

func TestAddThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader)
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(nil, he)

	assert.Empty(t, p.gameState.thrownGrenades)
}

func TestAddThrownGrenade(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades)
	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
	assert.Equal(t, p.gameState.thrownGrenades[pl][common.EqHE][0], he)
}

func TestGetThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader)
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	wep := p.gameEventHandler.getThrownGrenade(nil, he.Type)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}

	he := common.NewEquipment(common.EqSmoke, common.UniqueID(1))

	wep := p.gameEventHandler.getThrownGrenade(pl, he.Type)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getThrownGrenade(pl, he.Type)

	assert.Equal(t, wep.Type, he.Type)
	assert.Equal(t, wep, he)
}

// TestGetThrownGrenade_CircularControlledBot verifies that getThrownGrenade doesn't
// infinitely recurse when ControlledBot() creates a circular reference between two players (see #620).
func TestGetThrownGrenade_CircularControlledBot(t *testing.T) {
	p := NewParser(rand.Reader)
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
	playerA.Entity = stfake.NewEntity(1, stfake.P("m_hOriginalControllerOfCurrentPawn", uint64(2)))
	playerB.Entity = stfake.NewEntity(2, stfake.P("m_hOriginalControllerOfCurrentPawn", uint64(1)))

	smoke := common.NewEquipment(common.EqSmoke, common.UniqueID(1))
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

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
	p := NewParser(rand.Reader)
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	// Do nothing, we just keep sure it doesn't crash
	p.gameEventHandler.deleteThrownGrenade(nil, he.Type)
}

func TestDeleteThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, common.EqSmoke)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
}

func TestDeleteThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, he.Type)

	assert.Empty(t, p.gameState.thrownGrenades[pl])
}

// Without a known grenade the most recently thrown one of the type is resolved and deleted - grenades that
// already detonated may still be in the list (e.g. an HE projectile lingers after the explosion).
func TestThrownGrenade_MostRecentFirst(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he1 := common.NewEquipment(common.EqHE, common.UniqueID(1))
	he2 := common.NewEquipment(common.EqHE, common.UniqueID(2))

	p.gameEventHandler.addThrownGrenade(pl, he1)
	p.gameEventHandler.addThrownGrenade(pl, he2)

	assert.Same(t, he2, p.gameEventHandler.getThrownGrenade(pl, common.EqHE))

	p.gameEventHandler.deleteThrownGrenade(pl, common.EqHE)

	assert.Same(t, he1, p.gameEventHandler.getThrownGrenade(pl, common.EqHE))
}

// A known grenade is removed exactly, regardless of its position in the list.
func TestRemoveThrownGrenade(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	smoke1 := common.NewEquipment(common.EqSmoke, common.UniqueID(1))
	smoke2 := common.NewEquipment(common.EqSmoke, common.UniqueID(2))
	he := common.NewEquipment(common.EqHE, common.UniqueID(3))

	p.gameEventHandler.addThrownGrenade(pl, smoke1)
	p.gameEventHandler.addThrownGrenade(pl, smoke2)
	p.gameEventHandler.addThrownGrenade(pl, he)

	p.gameEventHandler.removeThrownGrenade(smoke2)

	assert.Equal(t, []*common.Equipment{smoke1}, p.gameState.thrownGrenades[pl][common.EqSmoke])
	assert.Equal(t, []*common.Equipment{he}, p.gameState.thrownGrenades[pl][common.EqHE])

	// not thrown (anymore) - no-op
	p.gameEventHandler.removeThrownGrenade(smoke2)
	p.gameEventHandler.removeThrownGrenade(nil)

	p.gameEventHandler.removeThrownGrenade(smoke1)
	p.gameEventHandler.removeThrownGrenade(he)

	assert.Empty(t, p.gameState.thrownGrenades[pl])
}

// The grenade is removed even if it was stored for a different player than the one an event reports
// (e.g. the projectile's thrower changed or the event has no thrower).
func TestRemoveThrownGrenade_StoredForOtherPlayer(t *testing.T) {
	p := NewParser(rand.Reader)
	thrower := &common.Player{Name: "thrower"}
	other := &common.Player{Name: "other"}
	smokeOther := common.NewEquipment(common.EqSmoke, common.UniqueID(1))
	smoke := common.NewEquipment(common.EqSmoke, common.UniqueID(2))

	p.gameEventHandler.addThrownGrenade(other, smokeOther)
	p.gameEventHandler.addThrownGrenade(thrower, smoke)

	p.gameEventHandler.removeThrownGrenade(smoke)

	assert.Empty(t, p.gameState.thrownGrenades[thrower])
	assert.Equal(t, []*common.Equipment{smokeOther}, p.gameState.thrownGrenades[other][common.EqSmoke])
}

// An expired inferno removes the grenade of the projectile it's linked to, even without a known thrower.
func TestInfernoExpired_RemovesLinkedProjectilesGrenade(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	molotov1 := common.NewEquipment(common.EqMolotov, common.UniqueID(1))
	molotov2 := common.NewEquipment(common.EqMolotov, common.UniqueID(2))

	p.gameEventHandler.addThrownGrenade(pl, molotov1)
	p.gameEventHandler.addThrownGrenade(pl, molotov2)

	const infernoID = 77

	infEntity := stfake.NewEntity(infernoID)

	inf := common.NewInferno(p.demoInfoProvider, infEntity, nil, common.UniqueID(3))
	p.gameState.infernos[infernoID] = inf
	p.gameState.infernoProjectiles[infernoID] = &common.GrenadeProjectile{WeaponInstance: molotov2}

	var got []events.InfernoExpired
	p.RegisterEventHandler(func(e events.InfernoExpired) {
		got = append(got, e)
	})

	p.infernoExpired(inf)

	assert.Len(t, got, 1)
	assert.Same(t, molotov2, got[0].Projectile.WeaponInstance)
	assert.Equal(t, []*common.Equipment{molotov1}, p.gameState.thrownGrenades[pl][common.EqMolotov])
	assert.Empty(t, p.gameState.infernoProjectiles)
}

// Leftover projectiles and infernos are destroyed at round end in ascending entity-ID order,
// not in Go's randomised map order.
func TestClearGrenadeProjectiles_DeterministicOrder(t *testing.T) {
	ids := []int{30, 5, 117, 17, 64}

	for range 20 {
		p := NewParser(rand.Reader)
		pl := &common.Player{}

		for i, id := range ids {
			he := common.NewEquipment(common.EqHE, common.UniqueID(i))
			proj := newTestProjectile(uint64(i), pl, he)
			proj.Entity = stfake.NewEntity(id)
			p.gameState.grenadeProjectiles[id] = proj

			infID := id + 1000
			molotov := common.NewEquipment(common.EqMolotov, common.UniqueID(100+i))
			p.gameState.infernos[infID] = common.NewInferno(p.demoInfoProvider, stfake.NewEntity(infID), nil, common.UniqueID(200+i))
			p.gameState.infernoProjectiles[infID] = &common.GrenadeProjectile{WeaponInstance: molotov}
		}

		var (
			destroyed []int
			expired   []int
		)

		p.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
			assert.Empty(t, expired, "projectiles must be destroyed before infernos expire")
			destroyed = append(destroyed, e.Projectile.Entity.ID())
		})
		p.RegisterEventHandler(func(e events.InfernoExpired) {
			expired = append(expired, e.Inferno.Entity.ID())
		})

		p.gameEventHandler.clearGrenadeProjectiles()

		assert.Equal(t, []int{5, 17, 30, 64, 117}, destroyed)
		assert.Equal(t, []int{1005, 1017, 1030, 1064, 1117}, expired)
		assert.Empty(t, p.gameState.grenadeProjectiles)
		assert.Empty(t, p.gameState.infernos)
	}
}

// Several leftover molotovs of the same owner and one inferno of that owner waiting for its projectile
// in the current frame: the molotov with the lowest entity-ID is linked to the inferno.
func TestClearGrenadeProjectiles_LinksLowestLeftoverMolotov(t *testing.T) {
	const (
		ownerHandle = uint64(42)
		infernoID   = 500
	)

	for range 20 {
		p := NewParser(rand.Reader)
		pl := &common.Player{}

		for i, id := range []int{40, 12, 77} {
			molotov := common.NewEquipment(common.EqMolotov, common.UniqueID(i))
			proj := newTestProjectile(uint64(i), pl, molotov)
			proj.Entity = stfake.NewEntity(id, stfake.P("m_hOwnerEntity", ownerHandle))
			p.gameState.grenadeProjectiles[id] = proj
		}

		p.gameState.infernos[infernoID] = common.NewInferno(p.demoInfoProvider, stfake.NewEntity(infernoID), nil, common.UniqueID(100))
		p.gameState.pendingInfernoLinks = []pendingInfernoLink{{
			frame:           p.currentFrame,
			ownerHandle:     ownerHandle,
			infernoEntityID: infernoID,
		}}

		var expired []events.InfernoExpired
		p.RegisterEventHandler(func(e events.InfernoExpired) {
			expired = append(expired, e)
		})

		p.gameEventHandler.clearGrenadeProjectiles()

		if assert.Len(t, expired, 1) && assert.NotNil(t, expired[0].Projectile) {
			assert.Equal(t, 12, expired[0].Projectile.Entity.ID())
		}
	}
}

// Grenades thrown while controlling a bot are stored for the bot and deleted from there.
func TestDeleteThrownGrenade_ControlledBot(t *testing.T) {
	p := NewParser(rand.Reader)
	provider := demoInfoProvider{parser: p}

	human := common.NewPlayer(provider)
	human.SteamID64 = 1
	bot := common.NewPlayer(provider)
	bot.SteamID64 = 0

	p.gameState.playersByEntityID[2] = bot
	human.Entity = stfake.NewEntity(1, stfake.P("m_hOriginalControllerOfCurrentPawn", uint64(2)))
	bot.Entity = stfake.NewEntity(2, stfake.P("m_hOriginalControllerOfCurrentPawn", uint64(constants.InvalidEntityHandleSource2)))

	smoke := common.NewEquipment(common.EqSmoke, common.UniqueID(1))
	p.gameEventHandler.addThrownGrenade(bot, smoke)

	assert.Same(t, smoke, p.gameEventHandler.getThrownGrenade(human, common.EqSmoke))

	p.gameEventHandler.deleteThrownGrenade(human, common.EqSmoke)

	assert.Empty(t, p.gameState.thrownGrenades[bot])
}

// Molotovs may be reported as incendiaries and vice versa (e.g. by InfernoExpired), deleting must handle that too.
func TestDeleteThrownGrenade_MolotovReportedAsIncendiary(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	molotov := common.NewEquipment(common.EqMolotov, common.UniqueID(1))

	p.gameEventHandler.addThrownGrenade(pl, molotov)
	p.gameEventHandler.deleteThrownGrenade(pl, common.EqIncendiary)

	assert.Empty(t, p.gameState.thrownGrenades[pl])
}

func TestGetEquipmentInstance_NotGrenade(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqAK47, "")

	assert.Equal(t, common.EqAK47, wep.Type)
}

func TestGetEquipmentInstance_Grenade_NotThrown(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqSmoke, "")

	assert.Nil(t, wep)
}

func TestGetEquipmentInstance_Grenade_Thrown(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, common.UniqueID(1))

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getEquipmentInstance(pl, he.Type, "")

	assert.Equal(t, he, wep)
}

func TestAttackerWeaponType_UnknownStaysUnknownWithoutContext(t *testing.T) {
	p := NewParser(rand.Reader)
	p.currentFrame = 24

	wepType := p.gameEventHandler.attackerWeaponType(common.EqUnknown, 123)

	assert.Equal(t, common.EqUnknown, wepType)
}

func TestAttackerWeaponType_FallDamageWins(t *testing.T) {
	p := NewParser(rand.Reader)
	p.currentFrame = 36
	p.gameEventHandler.userIDToFallDamageFrame[123] = p.currentFrame

	wepType := p.gameEventHandler.attackerWeaponType(common.EqUnknown, 123)

	assert.Equal(t, common.EqWorld, wepType)
}

func TestAttackerWeaponType_RoundEndReasonTargetBombedWins(t *testing.T) {
	p := NewParser(rand.Reader)
	p.currentFrame = 48
	p.gameEventHandler.frameToRoundEndReason[p.currentFrame] = events.RoundEndReasonTargetBombed

	wepType := p.gameEventHandler.attackerWeaponType(common.EqUnknown, 123)

	assert.Equal(t, common.EqBomb, wepType)
}

func TestPlayerHurt_UnknownWeaponDefaultsToWorld(t *testing.T) {
	p := NewParser(rand.Reader)
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
	p := NewParser(rand.Reader)
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
	p := NewParser(rand.Reader)
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

// newPlayerWithPawnHealth registers a victim whose pawn reports the given (start-of-tick) health.
func newPlayerWithPawnHealth(p *Parser, userID int, health int) *common.Player {
	const pawnHandle = 7

	controller := stfake.NewEntity(1,
		stfake.P("m_hPawn", uint64(pawnHandle)),
		stfake.P("m_hPlayerPawn", uint64(pawnHandle)),
	)

	p.gameState.entities[pawnHandle] = stfake.NewEntity(pawnHandle, stfake.P("m_iHealth", int32(health)))

	victim := common.NewPlayer(demoInfoProvider{parser: p})
	victim.Entity = controller
	victim.UserID = userID
	p.gameState.playersByUserID[userID] = victim

	return victim
}

func TestPlayerHurt_HealthDamageTakenIsHealthDrop(t *testing.T) {
	p := NewParser(rand.Reader)
	p.gameState.ingameTick = 5000

	victim := newPlayerWithPawnHealth(p, 42, 100)

	var got []events.PlayerHurt
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		got = append(got, e)
	})

	// On many server builds a hit can remove dmg_health+1 HP (fractional damage).
	data := playerHurtEventData(int32(victim.UserID), 1, "ak47")
	data["health"].ValByte = proto.Int32(73)
	data["armor"].ValByte = proto.Int32(50)
	data["dmg_health"].ValShort = proto.Int32(26)

	p.gameEventHandler.playerHurt(data)

	assert.Len(t, got, 1)
	assert.Equal(t, 26, got[0].HealthDamage)
	assert.Equal(t, 27, got[0].HealthDamageTaken)
}

func TestPlayerHurt_SameTickHitsMeasuredFromPriorHitHealth(t *testing.T) {
	p := NewParser(rand.Reader)
	p.currentFrame = 100
	p.gameState.ingameTick = 5000

	victim := newPlayerWithPawnHealth(p, 42, 50)

	var got []events.PlayerHurt
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		got = append(got, e)
	})

	// First hit: non-fatal, dmg_health 14 but victim drops from 50 to 35.
	data := playerHurtEventData(int32(victim.UserID), 1, "ak47")
	data["health"].ValByte = proto.Int32(35)
	data["armor"].ValByte = proto.Int32(50)
	data["dmg_health"].ValShort = proto.Int32(14)

	p.gameEventHandler.playerHurt(data)

	// Second hit, same tick: fatal (health 0). Must be measured from the victim's HP right
	// before it (35), not the start-of-tick value (50) which would over-report.
	data2 := playerHurtEventData(int32(victim.UserID), 1, "ak47")
	data2["health"].ValByte = proto.Int32(0)
	data2["armor"].ValByte = proto.Int32(50)
	data2["dmg_health"].ValShort = proto.Int32(100)

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
	p := NewParser(rand.Reader)

	killed := stfake.New(stfake.Spec{
		Class: "CDynamicProp",
		ID:    5,
		Props: []stfake.Prop{stfake.Pos(r3.Vector{X: 1, Y: 2, Z: 3})},
	})

	p.gameState.entities[5] = killed

	inflictor := stfake.New(stfake.Spec{Class: "CInferno", ID: 12})

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
	p := NewParser(rand.Reader)

	for _, className := range []string{"CCSPlayerPawn", "CCSPlayerPawnBase"} {
		killed := stfake.New(stfake.Spec{Class: className, ID: 5})

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
	p := NewParser(rand.Reader)

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

// CS2's bomb_dropped game-event has the C4's entity-ID in entindex.
func TestBombDropped_EntityIDFromEntindex(t *testing.T) {
	p := NewParser(rand.Reader)
	p.disableMimicSource1GameEvents = true

	var got []events.BombDropped
	p.RegisterEventHandler(func(e events.BombDropped) {
		got = append(got, e)
	})

	p.gameEventHandler.bombDropped(map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"userid":   {ValShort: proto.Int32(3)},
		"entindex": {ValLong: proto.Int32(586)},
	})

	assert.Len(t, got, 1)
	assert.Equal(t, 586, got[0].EntityID)
}

// bomb_dropped falls back to userid_pawn if userid doesn't resolve.
func TestBombDropped_PlayerFromUserIDPawn(t *testing.T) {
	p := NewParser(rand.Reader)
	p.disableMimicSource1GameEvents = true
	pl := addFakeBombCarrier(p)

	var got []events.BombDropped
	p.RegisterEventHandler(func(e events.BombDropped) {
		got = append(got, e)
	})

	p.gameEventHandler.bombDropped(map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"userid_pawn": {ValLong: proto.Int32(int32(testBombPawnHandle))},
		"entindex":    {ValLong: proto.Int32(586)},
	})

	assert.Equal(t, []events.BombDropped{{Player: pl, EntityID: 586}}, got)
}

// CS2's bomb_pickup has no userid, only userid_pawn.
func TestBombPickup_PlayerFromUserIDPawn(t *testing.T) {
	p := NewParser(rand.Reader)
	p.disableMimicSource1GameEvents = true
	pl := addFakeBombCarrier(p)

	var got []events.BombPickup
	p.RegisterEventHandler(func(e events.BombPickup) {
		got = append(got, e)
	})

	p.gameEventHandler.bombPickup(map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"userid_pawn": {ValLong: proto.Int32(int32(testBombPawnHandle))},
	})

	assert.Equal(t, []events.BombPickup{{Player: pl}}, got)
}

// Inferno fire properties without a value hold the default (not burning), e.g. the flag of an
// extinguished fire after a re-entry into the PVS on POV demos. They don't panic and don't block InfernoFireOut.
func TestProcessInfernoFireOut_UndecodedFireProperties(t *testing.T) {
	p := NewParser(rand.Reader)

	const infernoID = 77

	infEntity := stfake.NewEntity(infernoID,
		stfake.P("m_fireCount", nil),
		stfake.P("m_bFireIsBurning.0000", nil),
		stfake.P("m_bFireIsBurning.0001", nil),
	)

	inf := common.NewInferno(p.demoInfoProvider, infEntity, nil, common.UniqueID(1))
	p.gameState.infernos[infernoID] = inf

	var got []events.InfernoFireOut
	p.RegisterEventHandler(func(e events.InfernoFireOut) {
		got = append(got, e)
	})

	assert.NotPanics(t, p.processInfernoFireOut, "undecoded m_fireCount")
	assert.False(t, p.gameState.infernoFireStates[infernoID].wasBurning)

	stfake.Set(infEntity, stfake.P("m_fireCount", int32(2)), stfake.P("m_bFireIsBurning.0001", true))
	assert.NotPanics(t, p.processInfernoFireOut, "undecoded m_bFireIsBurning.0000")
	assert.True(t, p.gameState.infernoFireStates[infernoID].wasBurning)
	assert.Empty(t, got)

	stfake.Set(infEntity, stfake.P("m_bFireIsBurning.0001", false))
	p.processInfernoFireOut()
	p.processInfernoFireOut()
	assert.Len(t, got, 1, "an undecoded flag counts as not burning")
	assert.Same(t, inf, got[0].Inferno)
}

func TestProcessInfernoFireOut_MissingFireCount(t *testing.T) {
	p := NewParser(rand.Reader)

	const infernoID = 77

	p.gameState.infernos[infernoID] = common.NewInferno(p.demoInfoProvider, stfake.NewEntity(infernoID), nil, common.UniqueID(1))

	assert.NotPanics(t, p.processInfernoFireOut)
	assert.False(t, p.gameState.infernoFireStates[infernoID].wasBurning)
}
