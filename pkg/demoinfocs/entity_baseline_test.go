package demoinfocs_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// In POV demos the data of entities entering the PVS is a delta from entity baselines (see
// sendtables.SendTableParser.BeginPacketEntities). With the wrong baseline, properties that aren't sent are wrong,
// e.g. a grenade projectile's model and so its type.
// The weapon_fire game events don't depend on entities and tell which grenade was thrown.
func TestEntityBaselines_GrenadeProjectileTypes(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	cases := []struct {
		demPath       string
		maxMismatches int
	}{
		{demPath: demSetPath + "/POV-dust2-faceit-game_6-30.dem"},
		// the remaining ones are smoke grenades (according to their model) thrown shortly after another grenade
		// whose own weapon_fire event isn't in the demo
		{demPath: demSetPath + "/POV-orbit-skytten-vs-cloud9-gfinity15sm1-nuke.dem", maxMismatches: 3},
	}

	for _, c := range cases {
		t.Run(filepath.Base(c.demPath), func(t *testing.T) {
			t.Parallel()

			matches, mismatches := grenadeProjectileTypesMatchingWeaponFire(t, c.demPath)

			assert.Greater(t, matches, 100)
			assert.LessOrEqual(t, mismatches, c.maxMismatches)
		})
	}
}

// grenadeProjectileTypesMatchingWeaponFire returns how many grenade projectiles have the type of the grenade their
// thrower fired last (weapon_fire event) and how many don't.
func grenadeProjectileTypesMatchingWeaponFire(t *testing.T, demPath string) (matches, mismatches int) {
	t.Helper()

	f := openFile(t, demPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	type grenadeFire struct {
		tick int
		typ  common.EquipmentType
	}

	lastGrenadeFire := map[*common.Player]grenadeFire{}

	p.RegisterEventHandler(func(e events.WeaponFire) {
		if e.Shooter != nil && e.Weapon != nil && e.Weapon.Class() == common.EqClassGrenade {
			lastGrenadeFire[e.Shooter] = grenadeFire{tick: p.GameState().IngameTick(), typ: e.Weapon.Type}
		}
	})

	p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		const maxTicksSinceFire = 128

		fire, ok := lastGrenadeFire[e.Projectile.Thrower]
		if !ok || p.GameState().IngameTick()-fire.tick > maxTicksSinceFire {
			return
		}

		if fire.typ == e.Projectile.WeaponInstance.Type {
			matches++
		} else {
			mismatches++

			t.Logf("tick %d: %s's projectile (entity %d) is a %s but the last grenade fired was a %s",
				p.GameState().IngameTick(), e.Projectile.Thrower, e.Projectile.Entity.ID(), e.Projectile.WeaponInstance.Type, fire.typ)
		}
	})

	assert.NoError(t, p.ParseToEnd())

	return matches, mismatches
}
