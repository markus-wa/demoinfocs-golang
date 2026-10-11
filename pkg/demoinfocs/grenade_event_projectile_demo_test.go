package demoinfocs_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// TestGrenadeEvent_Projectile checks that grenade events carry the projectile they're about,
// which allows tracking individual grenades from throw to effect (see #601 & #361).
func TestGrenadeEvent_Projectile(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	f := openFile(t, s2POVDemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	counts := make(map[string]int)
	projectilesByUniqueID := make(map[int64]*common.GrenadeProjectile)
	uniqueIDsPerEvent := make(map[string]map[int64]bool)

	p.RegisterEventHandler(func(e events.GrenadeEventIf) {
		base := e.Base()
		name := fmt.Sprintf("%T", e)
		counts[name]++

		switch e.(type) {
		case events.FireGrenadeStart:
			// in POV demos inferno_startburn is sent before the inferno entity is created,
			// so the projectile isn't linked yet
			return

		case events.FireGrenadeExpired:
			if !assert.NotNil(t, base.Projectile, name) {
				return
			}

			assert.Contains(t, []common.EquipmentType{common.EqMolotov, common.EqIncendiary}, base.Projectile.WeaponInstance.Type)

		default:
			if !assert.NotNil(t, base.Projectile, name) {
				return
			}

			assert.Same(t, p.GameState().GrenadeProjectiles()[base.GrenadeEntityID], base.Projectile, "%s: Projectile should be the live projectile with GrenadeEntityID", name)
			assert.Equal(t, base.GrenadeType, base.Projectile.WeaponInstance.Type, name)
		}

		assert.Same(t, base.Projectile.WeaponInstance, base.Grenade, "%s: Grenade should be Projectile.WeaponInstance", name)
		assert.Same(t, base.Projectile.Thrower, base.Thrower, name)

		// UniqueID must identify the projectile
		if other, ok := projectilesByUniqueID[base.Projectile.UniqueID()]; ok {
			assert.Same(t, other, base.Projectile, "%s: two projectiles share the same UniqueID", name)
		}

		projectilesByUniqueID[base.Projectile.UniqueID()] = base.Projectile

		// each grenade explodes / pops once
		switch e.(type) {
		case events.HeExplode, events.SmokeStart, events.SmokeExpired:
			if uniqueIDsPerEvent[name] == nil {
				uniqueIDsPerEvent[name] = make(map[int64]bool)
			}

			assert.False(t, uniqueIDsPerEvent[name][base.Projectile.UniqueID()], "%s dispatched twice for the same projectile", name)
			uniqueIDsPerEvent[name][base.Projectile.UniqueID()] = true
		}
	})

	err := p.ParseToEnd()
	assert.NoError(t, err)

	for _, name := range []string{"events.HeExplode", "events.FlashExplode", "events.SmokeStart", "events.SmokeExpired", "events.FireGrenadeExpired"} {
		assert.Positive(t, counts[name], "expected %s events in the demo", name)
	}
}
