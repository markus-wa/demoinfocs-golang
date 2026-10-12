package demoinfocs

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func grenadeEventData(entityID int32) map[string]*msg.CMsgSource1LegacyGameEventKeyT {
	return map[string]*msg.CMsgSource1LegacyGameEventKeyT{
		"entityid": {ValShort: proto.Int32(entityID)},
		"x":        {ValFloat: proto.Float32(1)},
		"y":        {ValFloat: proto.Float32(2)},
		"z":        {ValFloat: proto.Float32(3)},
	}
}

func newTestProjectile(id int64, thrower *common.Player, wep *common.Equipment) *common.GrenadeProjectile {
	proj := common.NewGrenadeProjectile(id)
	proj.Thrower = thrower
	proj.Owner = thrower
	proj.WeaponInstance = wep

	return proj
}

// Two smokes thrown by the same player are in the air - the event must use the weapon of the
// projectile it refers to, not the player's last thrown smoke.
func TestNadeEvent_ResolvesProjectileByEntityID(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{Name: "thrower"}

	smoke1 := common.NewEquipment(common.EqSmoke, 1)
	smoke2 := common.NewEquipment(common.EqSmoke, 2)
	p.gameEventHandler.addThrownGrenade(pl, smoke1)
	p.gameEventHandler.addThrownGrenade(pl, smoke2)

	proj1 := newTestProjectile(10, pl, smoke1)
	p.gameState.grenadeProjectiles[42] = proj1
	p.gameState.grenadeProjectiles[43] = newTestProjectile(11, pl, smoke2)

	var got []events.SmokeStart
	p.RegisterEventHandler(func(e events.SmokeStart) {
		got = append(got, e)
	})

	p.gameEventHandler.smokeGrenadeDetonate(grenadeEventData(42))

	assert.Len(t, got, 1)
	assert.Same(t, proj1, got[0].Projectile)
	assert.Same(t, smoke1, got[0].Grenade)
	assert.Same(t, pl, got[0].Thrower, "thrower should be taken from the projectile if missing in the event")
	assert.Equal(t, 42, got[0].GrenadeEntityID)
	assert.Equal(t, common.EqSmoke, got[0].GrenadeType)
}

func TestNadeEvent_UnknownProjectile_FallsBackToLastThrownGrenade(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)

	var got []events.HeExplode
	p.RegisterEventHandler(func(e events.HeExplode) {
		got = append(got, e)
	})

	p.gameEventHandler.heGrenadeDetonate(grenadeEventData(42))

	assert.Len(t, got, 1)
	assert.Nil(t, got[0].Projectile)
	assert.Nil(t, got[0].Grenade)
	assert.Nil(t, got[0].Thrower)
}

func TestNadeEvent_ProjectileWithoutWeaponInstance_FallsBack(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE, 1)
	p.gameEventHandler.addThrownGrenade(pl, he)

	proj := newTestProjectile(10, pl, nil)
	p.gameState.grenadeProjectiles[42] = proj

	var got []events.HeExplode
	p.RegisterEventHandler(func(e events.HeExplode) {
		got = append(got, e)
	})

	p.gameEventHandler.heGrenadeDetonate(grenadeEventData(42))

	assert.Len(t, got, 1)
	assert.Same(t, proj, got[0].Projectile)
	assert.Same(t, he, got[0].Grenade)
}

func TestLinkInfernoProjectile(t *testing.T) {
	const (
		frame      = 100
		owner      = uint64(1234)
		otherOwner = uint64(5678)
		infernoID  = 77
	)

	projectileLink := func(proj *common.GrenadeProjectile, frame int, owner uint64) pendingInfernoLink {
		return pendingInfernoLink{frame: frame, ownerHandle: owner, projectile: proj}
	}
	infernoLink := func(frame int, owner uint64) pendingInfernoLink {
		return pendingInfernoLink{frame: frame, ownerHandle: owner, infernoEntityID: infernoID}
	}

	tests := []struct {
		name   string
		links  func(proj, other *common.GrenadeProjectile) []pendingInfernoLink
		linked bool
	}{
		{
			name: "projectile destroyed before inferno created",
			links: func(proj, _ *common.GrenadeProjectile) []pendingInfernoLink {
				return []pendingInfernoLink{projectileLink(proj, frame, owner), infernoLink(frame, owner)}
			},
			linked: true,
		},
		{
			name: "inferno created before projectile destroyed",
			links: func(proj, _ *common.GrenadeProjectile) []pendingInfernoLink {
				return []pendingInfernoLink{infernoLink(frame, owner), projectileLink(proj, frame, owner)}
			},
			linked: true,
		},
		{
			name: "other owner's projectile isn't linked",
			links: func(_, other *common.GrenadeProjectile) []pendingInfernoLink {
				return []pendingInfernoLink{projectileLink(other, frame, otherOwner), infernoLink(frame, owner)}
			},
		},
		{
			name: "projectile destroyed in an earlier frame isn't linked",
			links: func(proj, _ *common.GrenadeProjectile) []pendingInfernoLink {
				return []pendingInfernoLink{projectileLink(proj, frame-1, owner), infernoLink(frame, owner)}
			},
		},
		{
			name: "ambiguous - two projectiles of the same owner",
			links: func(proj, other *common.GrenadeProjectile) []pendingInfernoLink {
				return []pendingInfernoLink{projectileLink(proj, frame, owner), projectileLink(other, frame, owner), infernoLink(frame, owner)}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParser(rand.Reader).(*parser)
			proj := newTestProjectile(1, nil, common.NewEquipment(common.EqMolotov, 1))
			other := newTestProjectile(2, nil, common.NewEquipment(common.EqMolotov, 2))

			for _, link := range tc.links(proj, other) {
				p.currentFrame = link.frame
				p.linkInfernoProjectile(link)
			}

			if tc.linked {
				assert.Same(t, proj, p.gameState.infernoProjectiles[infernoID])
			} else {
				assert.Nil(t, p.gameState.infernoProjectiles[infernoID])
			}
		})
	}
}

func TestFireGrenadeEvents_UseLinkedProjectile(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	molotov := common.NewEquipment(common.EqMolotov, 1)
	proj := newTestProjectile(1, pl, molotov)

	const infernoID = 77

	p.gameState.infernoProjectiles[infernoID] = proj
	// a projectile entity with the inferno's entity-ID must not be used for fire events
	p.gameState.grenadeProjectiles[infernoID] = newTestProjectile(2, pl, common.NewEquipment(common.EqSmoke, 2))

	var got []events.GrenadeEventIf
	p.RegisterEventHandler(func(e events.GrenadeEventIf) {
		got = append(got, e)
	})

	p.gameEventHandler.infernoStartBurn(grenadeEventData(infernoID))
	p.gameEventHandler.infernoExpire(grenadeEventData(infernoID))

	assert.Len(t, got, 2)
	assert.IsType(t, events.FireGrenadeStart{}, got[0])
	assert.IsType(t, events.FireGrenadeExpired{}, got[1])

	for _, e := range got {
		assert.Same(t, proj, e.Base().Projectile)
		assert.Same(t, molotov, e.Base().Grenade)
		assert.Same(t, pl, e.Base().Thrower)
		assert.Equal(t, common.EqIncendiary, e.Base().GrenadeType)
	}

	p.gameEventHandler.infernoExpire(grenadeEventData(infernoID + 1))
	assert.Nil(t, got[2].Base().Projectile)
}

// The entity-ID of an event may have been re-used by a different grenade - don't attribute the event to it.
func TestNadeEvent_ProjectileOfDifferentType_Ignored(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	p.gameState.grenadeProjectiles[42] = newTestProjectile(1, pl, common.NewEquipment(common.EqSmoke, 1))

	var got []events.HeExplode
	p.RegisterEventHandler(func(e events.HeExplode) {
		got = append(got, e)
	})

	p.gameEventHandler.heGrenadeDetonate(grenadeEventData(42))

	assert.Len(t, got, 1)
	assert.Nil(t, got[0].Projectile)
	assert.Nil(t, got[0].Grenade)
}

// Molotovs and incendiaries may be reported as each other, that's not a different grenade.
func TestIsSameGrenadeType(t *testing.T) {
	assert.True(t, isSameGrenadeType(common.EqSmoke, common.EqSmoke))
	assert.True(t, isSameGrenadeType(common.EqMolotov, common.EqIncendiary))
	assert.True(t, isSameGrenadeType(common.EqIncendiary, common.EqMolotov))
	assert.False(t, isSameGrenadeType(common.EqSmoke, common.EqHE))
	assert.False(t, isSameGrenadeType(common.EqMolotov, common.EqHE))
}
