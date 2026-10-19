package sendtables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pvsTestParser returns a budgetTestParser that logs the entity handlers' ops and the
// OnLeavePVS, OnEnterPVS and OnDestroy delegates of all its entities.
func pvsTestParser(t *testing.T) (*Parser, *[]string) {
	t.Helper()

	p := budgetTestParser(t)

	var log []string

	p.OnEntity(func(e *Entity, op EntityOp) error {
		log = append(log, fmt.Sprintf("%d/%d %v", e.index, e.serial, op))

		return nil
	})

	p.classesById[0].OnEntityCreated(func(e *Entity) {
		e.OnLeavePVS(func() { log = append(log, fmt.Sprintf("%d/%d leave", e.index, e.serial)) })
		e.OnEnterPVS(func() { log = append(log, fmt.Sprintf("%d/%d enter in-pvs=%v", e.index, e.serial, e.InPVS())) })
		e.OnDestroy(func() { log = append(log, fmt.Sprintf("%d/%d destroy", e.index, e.serial)) })
	})

	return p, &log
}

func pvsPacket(t *testing.T, p *Parser, log *[]string, ops ...entityOp) []string {
	t.Helper()

	*log = nil

	require.NoError(t, p.OnPacketEntities(entitiesMsg(t, 16, ops...)))

	return *log
}

func TestPVS_LeaveAndReenter(t *testing.T) {
	t.Parallel()

	p, log := pvsTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1})

	e := p.entities[5]
	require.True(t, e.InPVS())

	assert.Equal(t, []string{"5/1 leave", "5/1 Left"}, pvsPacket(t, p, log, entityOp{index: 5, leave: true}))
	assert.False(t, e.InPVS())
	assert.True(t, e.active, "leaving the PVS doesn't destroy the entity")

	// leaving again does nothing
	assert.Empty(t, pvsPacket(t, p, log, entityOp{index: 5, leave: true}))

	assert.Equal(t, []string{"5/1 enter in-pvs=true", "5/1 Updated+Entered"}, pvsPacket(t, p, log, entityOp{index: 5, update: true, arrSize: 2}))
	assert.True(t, e.InPVS())
	assert.Same(t, e, p.entities[5])
	assert.Len(t, e.PropertyValueMust("m_arr").Any, 2)

	assert.Equal(t, []string{"5/1 Updated"}, pvsPacket(t, p, log, entityOp{index: 5, update: true, arrSize: 3}))

	// it can leave again after re-entering
	assert.Equal(t, []string{"5/1 leave", "5/1 Left"}, pvsPacket(t, p, log, entityOp{index: 5, leave: true}))
}

// A delete for an entity that already left the PVS still destroys it, without another OnLeavePVS.
func TestPVS_LeaveThenDelete(t *testing.T) {
	t.Parallel()

	p, log := pvsTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1})
	pvsPacket(t, p, log, entityOp{index: 5, leave: true})

	e := p.entities[5]

	assert.Equal(t, []string{"5/1 destroy", "5/1 Deleted+Left"}, pvsPacket(t, p, log, entityOp{index: 5, delete: true}))
	assert.False(t, e.active)
	assert.False(t, e.InPVS())
	assert.False(t, e.Replaced())

	assert.Empty(t, pvsPacket(t, p, log, entityOp{index: 5, delete: true}))
}

// A delete of an entity in the PVS fires OnDestroy only, not OnLeavePVS.
func TestPVS_DeleteDoesntFireLeave(t *testing.T) {
	t.Parallel()

	p, log := pvsTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1})

	assert.Equal(t, []string{"5/1 destroy", "5/1 Deleted+Left"}, pvsPacket(t, p, log, entityOp{index: 5, delete: true}))
	assert.False(t, p.entities[5].InPVS())

	// an update re-activates the destroyed entity (Entered), but it didn't leave the PVS, so OnEnterPVS doesn't fire
	assert.Equal(t, []string{"5/1 Updated+Entered"}, pvsPacket(t, p, log, entityOp{index: 5, update: true}))
	assert.True(t, p.entities[5].InPVS())
}

// An entity that left the PVS is destroyed (Replaced) when it's re-created at its index (#712), with a new serial
// or with the same serial (how entities usually re-enter the PVS on POV demos). The new object is in the PVS.
func TestPVS_LeaveThenRecreated(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		serial uint64
	}{
		{"same serial", 1},
		{"new serial", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, log := pvsTestParser(t)

			pvsPacket(t, p, log, entityOp{index: 5, serial: 1})
			assert.Equal(t, []string{"5/1 leave", "5/1 Left"}, pvsPacket(t, p, log, entityOp{index: 5, leave: true}))

			old := p.entities[5]

			assert.Equal(t, []string{"5/1 destroy", "5/1 Deleted+Left", fmt.Sprintf("5/%d Created+Entered", tc.serial)},
				pvsPacket(t, p, log, entityOp{index: 5, serial: tc.serial}))
			assert.True(t, old.Replaced())
			assert.False(t, old.active)
			assert.False(t, old.InPVS())

			e := p.entities[5]
			require.NotSame(t, old, e)
			assert.True(t, e.InPVS())
			assert.False(t, e.Replaced())
			assert.Equal(t, int32(tc.serial), e.serial)
		})
	}
}

// A same-serial re-create of an entity that didn't leave the PVS doesn't destroy the old object.
func TestPVS_RecreatedSameSerialWithoutLeave(t *testing.T) {
	t.Parallel()

	p, log := pvsTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1})

	old := p.entities[5]

	assert.Equal(t, []string{"5/1 Created+Entered"}, pvsPacket(t, p, log, entityOp{index: 5, serial: 1}))
	assert.False(t, old.Replaced())
	assert.True(t, old.active)
	assert.True(t, old.InPVS())
	assert.NotSame(t, old, p.entities[5])
}

func TestGuards_OnLeavePVS(t *testing.T) {
	p, e := newGuardTestEntity()
	e.OnLeavePVS(userCallbackThatPanics)

	requireTagged(t, recoverPanic(func() { syntheticLeave(e, p) }))
}

func TestGuards_OnEnterPVS(t *testing.T) {
	reenter := func(e *Entity, p *Parser) {
		syntheticLeave(e, p)
		syntheticSet(e, p, []string{"m_x"}, []any{2}, true, true)
	}

	p, e := newGuardTestEntity()
	e.OnEnterPVS(userCallbackThatPanics)

	requireTagged(t, recoverPanic(func() { reenter(e, p) }))
}

// User panics in OnLeavePVS are tagged as user-code panics by the real parser too.
func TestGuards_OnLeavePVS_Parser(t *testing.T) {
	p := budgetTestParser(t)
	p.classesById[0].OnEntityCreated(func(e *Entity) { e.OnLeavePVS(userCallbackThatPanics) })

	require.NoError(t, p.OnPacketEntities(entitiesMsg(t, 16, entityOp{index: 5, serial: 1})))

	requireTagged(t, recoverPanic(func() { _ = p.OnPacketEntities(entitiesMsg(t, 16, entityOp{index: 5, leave: true})) }))
}
