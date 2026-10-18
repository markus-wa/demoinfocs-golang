package demoinfocs_test

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// Unique ids are reproducible: parsing the same demo twice yields the same ids for the same things.
func TestUniqueID_Deterministic(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	type uniqueIDs struct {
		equipment  []ulid.ULID
		projectile []int64
		inferno    []int64
	}

	parse := func() uniqueIDs {
		f := openFile(t, s2DemPath)
		defer mustClose(t, f)

		p := demoinfocs.NewParser(f)

		var ids uniqueIDs

		p.RegisterEventHandler(func(e events.WeaponFire) { ids.equipment = append(ids.equipment, e.Weapon.UniqueID2()) })
		p.RegisterEventHandler(func(e events.ItemPickup) { ids.equipment = append(ids.equipment, e.Weapon.UniqueID2()) })
		p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
			ids.projectile = append(ids.projectile, e.Projectile.UniqueID())
		})
		p.RegisterEventHandler(func(e events.InfernoStart) { ids.inferno = append(ids.inferno, e.Inferno.UniqueID()) })

		assert.NoError(t, p.ParseToEnd())

		return ids
	}

	first := parse()

	assert.NotEmpty(t, first.equipment)
	assert.NotEmpty(t, first.projectile)
	assert.NotEmpty(t, first.inferno)
	assert.NotContains(t, first.equipment, ulid.ULID{}, "zero ids must never be assigned")
	assert.NotContains(t, first.projectile, int64(0), "zero ids must never be assigned")
	assert.NotContains(t, first.inferno, int64(0), "zero ids must never be assigned")
	assert.Equal(t, first, parse())
}
