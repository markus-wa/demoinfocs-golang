package demoinfocs_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// TestGrenadeEvent_Projectile checks that grenade events carry the projectile they're about,
// which allows tracking individual grenades from throw to effect (see #601 & #361).
func TestGrenadeEvent_Projectile(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	t.Run("POV", func(t *testing.T) {
		t.Parallel()

		testGrenadeEventProjectile(t, s2POVDemPath, true)
	})

	t.Run("GOTV", func(t *testing.T) {
		t.Parallel()

		testGrenadeEventProjectile(t, demSetPathS2+"/arsham_infernocs2_1stSept23.dem", false)
	})
}

func testGrenadeEventProjectile(t *testing.T, demPath string, isPOV bool) {
	t.Helper()

	f := openFile(t, demPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	counts := make(map[string]int)
	projectilesByUniqueID := make(map[common.UniqueID]*common.GrenadeProjectile)
	uniqueIDsPerEvent := make(map[string]map[common.UniqueID]bool)

	p.RegisterEventHandler(func(e events.GrenadeEventIf) {
		base := e.Base()
		name := fmt.Sprintf("%T", e)
		counts[name]++

		switch e.(type) {
		case events.FireGrenadeStart, events.FireGrenadeExpired:
			if _, isStart := e.(events.FireGrenadeStart); isStart && isPOV {
				// in POV demos inferno_startburn is sent before the inferno entity is created,
				// so the projectile isn't linked yet
				return
			}

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
				uniqueIDsPerEvent[name] = make(map[common.UniqueID]bool)
			}

			assert.False(t, uniqueIDsPerEvent[name][base.Projectile.UniqueID()], "%s dispatched twice for the same projectile", name)
			uniqueIDsPerEvent[name][base.Projectile.UniqueID()] = true
		}
	})

	var infernoEvents, infernoEventsWithProjectile int

	checkInfernoProjectile := func(inf *common.Inferno, proj *common.GrenadeProjectile) {
		infernoEvents++

		if proj == nil {
			return
		}

		infernoEventsWithProjectile++

		assert.Contains(t, []common.EquipmentType{common.EqMolotov, common.EqIncendiary}, proj.WeaponInstance.Type)
		assert.Same(t, inf.Thrower(), proj.Thrower)
	}

	p.RegisterEventHandler(func(e events.InfernoStart) { checkInfernoProjectile(e.Inferno, e.Projectile) })
	p.RegisterEventHandler(func(e events.InfernoExpired) { checkInfernoProjectile(e.Inferno, e.Projectile) })

	err := p.ParseToEnd(t.Context())
	assert.NoError(t, err)

	names := []string{"events.HeExplode", "events.FlashExplode", "events.SmokeStart", "events.SmokeExpired", "events.FireGrenadeExpired"}
	if !isPOV {
		names = append(names, "events.FireGrenadeStart")

		// POV demos re-create infernos re-entering the PVS, those can't be linked
		assert.Equal(t, infernoEvents, infernoEventsWithProjectile, "InfernoStart/Expired without Projectile")
	}

	for _, name := range names {
		assert.Positive(t, counts[name], "expected %s events in the demo", name)
	}

	t.Logf("inferno events: %d, with projectile: %d", infernoEvents, infernoEventsWithProjectile)
}
