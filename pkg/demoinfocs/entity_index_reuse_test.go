package demoinfocs_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// The server sometimes creates a new entity at the index of an entity it never deleted (e.g. on round restarts).
// The old entity must be destroyed, otherwise the new weapon entity re-uses the old *Equipment (and its unique id).
// Player inventories must not keep the old *Equipment either, if the new weapon is assigned before it's created.
// See https://github.com/markus-wa/demoinfocs-golang/issues/601
func TestEntityIndexReuse_NewEquipment(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	for _, demPath := range []string{s2DemPath, demSetPathS2 + "/1_2v2_6thAug23_64cf951f9b4ce6b86c73b089.dem"} {
		t.Run(filepath.Base(demPath), func(t *testing.T) {
			t.Parallel()

			testEntityIndexReuseNewEquipment(t, demPath)
		})
	}
}

func testEntityIndexReuseNewEquipment(t *testing.T, demPath string) {
	t.Helper()

	f := openFile(t, demPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	type entityKey struct{ id, serial int }

	equipmentByEntity := map[entityKey]*common.Equipment{}
	entitiesByEquipment := map[*common.Equipment]map[entityKey]bool{}

	p.RegisterEventHandler(func(events.FrameDone) {
		// inventories must hold the current weapon entities, not the replaced ones
		for _, pl := range p.GameState().Participants().Playing() {
			for entityID, w := range pl.Inventory {
				// destroyed weapons may linger in the inventory for a bit (e.g. a planted C4), but a live one must match
				if live := p.GameState().Weapons()[entityID]; live != nil {
					assert.Same(t, live, w, "%s's %s isn't the weapon with entity-ID %d (%s)", pl, w, entityID, live)
				}
			}
		}

		for _, w := range p.GameState().Weapons() {
			if w.Entity == nil {
				continue
			}

			k := entityKey{w.Entity.ID(), w.Entity.SerialNum()}
			equipmentByEntity[k] = w

			if entitiesByEquipment[w] == nil {
				entitiesByEquipment[w] = map[entityKey]bool{}
			}

			entitiesByEquipment[w][k] = true
		}
	})

	assert.NoError(t, p.ParseToEnd())
	assert.NotEmpty(t, equipmentByEntity)

	for w, entities := range entitiesByEquipment {
		assert.Len(t, entities, 1, "%s (%s) is shared by multiple weapon entities: %v", w, w.UniqueID(), entities)
	}
}
