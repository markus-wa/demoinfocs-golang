package sendtables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pvsTestParser returns a budgetTestParser that logs the entity handlers' ops and the
// OnLeavePVS and OnDestroy delegates of all its entities.
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

// An update after a leave brings the entity back (Entered). Not yet seen during testing, but may occur:
// in real demos entities re-enter the PVS as same-serial re-creates.
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

	// rare: an update without a re-create brings the same object back into the PVS
	assert.Equal(t, []string{"5/1 Updated+Entered"}, pvsPacket(t, p, log, entityOp{index: 5, update: true, arrSize: 2}))
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

	// an update re-activates the destroyed entity (Entered)
	assert.Equal(t, []string{"5/1 Updated+Entered"}, pvsPacket(t, p, log, entityOp{index: 5, update: true}))
	assert.True(t, p.entities[5].InPVS())
}

// An entity that left the PVS is destroyed (Replaced) when its index is reused for a new entity (#712).
// The new object is in the PVS.
func TestPVS_LeaveThenRecreatedNewSerial(t *testing.T) {
	t.Parallel()

	p, log := pvsTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1})
	assert.Equal(t, []string{"5/1 leave", "5/1 Left"}, pvsPacket(t, p, log, entityOp{index: 5, leave: true}))

	old := p.entities[5]

	assert.Equal(t, []string{"5/1 destroy", "5/1 Deleted+Left", "5/2 Created+Entered"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 2}))
	assert.True(t, old.Replaced())
	assert.False(t, old.active)
	assert.False(t, old.InPVS())

	e := p.entities[5]
	require.NotSame(t, old, e)
	assert.True(t, e.InPVS())
	assert.False(t, e.Replaced())
	assert.Equal(t, int32(2), e.serial)
}

// reentryTestParser returns a pvsTestParser that also logs the OnEnterPVS and OnUpdateFinished delegates,
// the m_arr update handler (with the array's length) and the created-handlers of class 0.
func reentryTestParser(t *testing.T) (*Parser, *[]string) {
	t.Helper()

	p, log := pvsTestParser(t)

	p.classesById[0].OnEntityCreated(func(e *Entity) {
		*log = append(*log, fmt.Sprintf("%d/%d created", e.index, e.serial))

		e.OnEnterPVS(func() { *log = append(*log, fmt.Sprintf("%d/%d enter", e.index, e.serial)) })
		e.OnUpdateFinished(func() { *log = append(*log, fmt.Sprintf("%d/%d finished", e.index, e.serial)) })
		e.Property("m_arr").OnUpdate(func(v PropertyValue) {
			*log = append(*log, fmt.Sprintf("%d/%d m_arr=%d", e.index, e.serial, len(v.Array())))
		})
	})

	return p, log
}

// An entity that left the PVS and is re-created with the same serial (how entities re-enter the PVS on POV demos)
// is the same object again (#742): no OnDestroy, no created-handlers, the update handlers fire for the changed values.
func TestPVS_LeaveThenRecreatedSameSerial(t *testing.T) {
	t.Parallel()

	p, log := reentryTestParser(t)

	assert.Equal(t, []string{"5/1 created", "5/1 Created+Entered", "5/1 m_arr=2", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 2}))
	assert.Equal(t, []string{"5/1 leave", "5/1 Left"}, pvsPacket(t, p, log, entityOp{index: 5, leave: true}))

	e := p.entities[5]

	assert.Equal(t, []string{"5/1 Updated+Entered", "5/1 m_arr=3", "5/1 enter", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 3}))
	assert.Same(t, e, p.entities[5])
	assert.True(t, e.InPVS())
	assert.True(t, e.active)
	assert.False(t, e.Replaced())
	assert.Len(t, e.PropertyValueMust("m_arr").Array(), 3)

	// it can leave and re-enter again, an unchanged value doesn't fire its update handlers
	assert.Equal(t, []string{"5/1 leave", "5/1 Left"}, pvsPacket(t, p, log, entityOp{index: 5, leave: true}))
	assert.Equal(t, []string{"5/1 Updated+Entered", "5/1 enter", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 3}))

	// regular updates keep working
	assert.Equal(t, []string{"5/1 m_arr=4", "5/1 finished", "5/1 Updated"},
		pvsPacket(t, p, log, entityOp{index: 5, update: true, arrSize: 4}))
}

// OnEnterPVS also fires if an entity that left re-enters through an update, after the update's handlers.
func TestPVS_LeaveThenUpdateFiresEnter(t *testing.T) {
	t.Parallel()

	p, log := reentryTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1})
	pvsPacket(t, p, log, entityOp{index: 5, leave: true})

	assert.Equal(t, []string{"5/1 m_arr=2", "5/1 enter", "5/1 finished", "5/1 Updated+Entered"},
		pvsPacket(t, p, log, entityOp{index: 5, update: true, arrSize: 2}))
}

// A same-serial re-create of an entity that didn't leave the PVS (seen on POV demos, #741) re-enters it too:
// the old object isn't left behind active and InPVS.
func TestPVS_RecreatedSameSerialWithoutLeave(t *testing.T) {
	t.Parallel()

	p, log := reentryTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 2})

	e := p.entities[5]

	assert.Equal(t, []string{"5/1 Updated+Entered", "5/1 enter", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 2}))
	assert.Same(t, e, p.entities[5])
	assert.True(t, e.InPVS())
	assert.False(t, e.Replaced())
}

// A re-entry reads the state anew: properties that aren't sent fall back to the instance baseline,
// the update handlers don't see the baseline's values before the sent ones.
func TestPVS_ReentryReadsBaseline(t *testing.T) {
	t.Parallel()

	p, log := reentryTestParser(t)

	w := &bitWriter{}
	w.arrayFieldUpdate(t, 4)
	p.classBaselines[0] = w.buf

	assert.Equal(t, []string{"5/1 created", "5/1 Created+Entered", "5/1 m_arr=2", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 2}))

	// the sent value equals the current one, the baseline's value in between doesn't fire
	assert.Equal(t, []string{"5/1 Updated+Entered", "5/1 enter", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 2}))

	// not sent: the baseline's value
	assert.Equal(t, []string{"5/1 Updated+Entered", "5/1 m_arr=4", "5/1 enter", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, noArr: true}))
	assert.Len(t, p.entities[5].PropertyValueMust("m_arr").Array(), 4)
}

// A property without a value after the re-entry (no baseline, not sent) doesn't fire its update handlers.
func TestPVS_ReentryWithoutValue(t *testing.T) {
	t.Parallel()

	p, log := reentryTestParser(t)

	pvsPacket(t, p, log, entityOp{index: 5, serial: 1, arrSize: 2})

	assert.Equal(t, []string{"5/1 Updated+Entered", "5/1 enter", "5/1 finished"},
		pvsPacket(t, p, log, entityOp{index: 5, serial: 1, noArr: true}))
	assert.Nil(t, p.entities[5].PropertyValueMust("m_arr").Any)
}

// A same-serial re-create with a different class, or of a destroyed entity, creates a new object.
func TestPVS_RecreatedSameSerialNewObject(t *testing.T) {
	t.Parallel()

	t.Run("different class", func(t *testing.T) {
		t.Parallel()

		p, log := reentryTestParser(t)

		pvsPacket(t, p, log, entityOp{index: 5, serial: 1})

		old := p.entities[5]

		assert.Equal(t, []string{"5/1 Created+Entered"}, pvsPacket(t, p, log, entityOp{index: 5, serial: 1, classID: 1}))
		assert.NotSame(t, old, p.entities[5])
		assert.Equal(t, "CArr2", p.entities[5].ServerClass().Name())
	})

	t.Run("destroyed", func(t *testing.T) {
		t.Parallel()

		p, log := reentryTestParser(t)

		pvsPacket(t, p, log, entityOp{index: 5, serial: 1})
		pvsPacket(t, p, log, entityOp{index: 5, delete: true})

		old := p.entities[5]

		assert.Equal(t, []string{"5/1 created", "5/1 Created+Entered", "5/1 m_arr=0", "5/1 finished"},
			pvsPacket(t, p, log, entityOp{index: 5, serial: 1}))
		assert.NotSame(t, old, p.entities[5])
	})
}

// The live budget is refunded and charged again on every re-entry, so it doesn't add up.
func TestPVS_ReentryBudget(t *testing.T) {
	t.Parallel()

	p := budgetTestParser(t)

	for i := range 10_000 {
		require.NoError(t, p.OnPacketEntities(entitiesMsg(t, 10, entityOp{index: 5, serial: 1, arrSize: uint64(100 + i%3)})), "message %d", i)
	}

	e := p.entities[5]
	assert.Equal(t, e.budgetSlots, p.entityBudget.liveSlots)
	assert.Less(t, e.budgetSlots, int64(entityFixedSlots+200))
}

// A re-entry resets the active types of polymorphic pointers, they're sent anew.
func TestPVS_ReentryResetsPolymorphicTypes(t *testing.T) {
	t.Parallel()

	cls, modeA, _ := polyTestClass(t)

	e := newEntity(1, 1, cls)
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})
	require.NotNil(t, e.Property("m_pMode.m_aField"))

	w := &bitWriter{}
	w.fieldPathOp(t, "FieldPathEncodeFinish")

	r := newReader(w.buf)
	defer r.release()

	budget := newEntityBudget()
	paths := []*fieldPath(nil)

	e.reenter(r, nil, &paths, &budget)

	assert.Nil(t, e.polySerializers[0])
	assert.Nil(t, e.Property("m_pMode.m_aField"))
	assert.True(t, e.InPVS())
}

func TestGuards_OnLeavePVS(t *testing.T) {
	p, e := newGuardTestEntity()
	e.OnLeavePVS(userCallbackThatPanics)

	requireTagged(t, recoverPanic(func() { syntheticLeave(e, p) }))
}

// User panics in OnLeavePVS are tagged as user-code panics by the real parser too.
func TestGuards_OnLeavePVS_Parser(t *testing.T) {
	p := budgetTestParser(t)
	p.classesById[0].OnEntityCreated(func(e *Entity) { e.OnLeavePVS(userCallbackThatPanics) })

	require.NoError(t, p.OnPacketEntities(entitiesMsg(t, 16, entityOp{index: 5, serial: 1})))

	requireTagged(t, recoverPanic(func() { _ = p.OnPacketEntities(entitiesMsg(t, 16, entityOp{index: 5, leave: true})) }))
}

func TestGuards_OnEnterPVS(t *testing.T) {
	_, e := newGuardTestEntity()
	e.OnEnterPVS(userCallbackThatPanics)

	requireTagged(t, recoverPanic(e.fireEnterPVS))
}

// User panics in OnEnterPVS are tagged as user-code panics by the real parser too.
func TestGuards_OnEnterPVS_Parser(t *testing.T) {
	p := budgetTestParser(t)
	p.classesById[0].OnEntityCreated(func(e *Entity) { e.OnEnterPVS(userCallbackThatPanics) })

	require.NoError(t, p.OnPacketEntities(entitiesMsg(t, 16, entityOp{index: 5, serial: 1})))

	requireTagged(t, recoverPanic(func() { _ = p.OnPacketEntities(entitiesMsg(t, 16, entityOp{index: 5, serial: 1})) }))
}
