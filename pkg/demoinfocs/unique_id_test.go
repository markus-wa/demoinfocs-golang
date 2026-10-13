package demoinfocs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// Unique ids are reproducible: parsing the same demo twice yields the same ids for the same things.
func TestUniqueID_Deterministic(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	parse := func() []common.UniqueID {
		f := openFile(t, s2DemPath)
		defer mustClose(t, f)

		p := demoinfocs.NewParser(f)

		var ids []common.UniqueID

		p.RegisterEventHandler(func(e events.WeaponFire) { ids = append(ids, e.Weapon.UniqueID()) })
		p.RegisterEventHandler(func(e events.ItemPickup) { ids = append(ids, e.Weapon.UniqueID()) })
		p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) { ids = append(ids, e.Projectile.UniqueID()) })
		p.RegisterEventHandler(func(e events.InfernoStart) { ids = append(ids, e.Inferno.UniqueID()) })

		assert.NoError(t, p.ParseToEnd(t.Context()))

		return ids
	}

	first := parse()

	assert.NotEmpty(t, first)
	assert.NotContains(t, first, common.UniqueID(0), "zero ids must never be assigned")
	assert.Equal(t, first, parse())
}
