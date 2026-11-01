package sendtables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A held *Property follows the entity's updates, and Entity.Property() hands out the same handle again.
func TestPropertyValueHeld(t *testing.T) {
	t.Parallel()

	e := newSyntheticEntity(nil, "CTest", 1, 1,
		[]string{"m_iHealth", "m_pSvc.m_flX", "m_pSvc.m_flY"},
		[]any{int32(100), float32(1.5), nil})

	hp := e.Property("m_iHealth")
	x := e.Property("m_pSvc.m_flX")
	y := e.Property("m_pSvc.m_flY")

	assert.Equal(t, int32(100), hp.Value().Any)
	assert.Equal(t, float32(1.5), x.Value().Any)
	assert.Nil(t, y.Value().Any, "a property without a value")

	syntheticSet(e, nil, []string{"m_iHealth", "m_pSvc.m_flY"}, []any{int32(37), float32(2)}, false, false)

	assert.Equal(t, int32(37), hp.Value().Any)
	assert.Equal(t, float32(1.5), x.Value().Any)
	assert.Equal(t, float32(2), y.Value().Any)

	assert.Same(t, hp, e.Property("m_iHealth"))
	assert.Equal(t, hp.Value(), e.PropertyValueMust("m_iHealth"))
}

// assertHeldReadsLikeLookup asserts that the held property p reads want, which is what a lookup of its
// name on e reads right now: the same value, or none while the name doesn't resolve.
func assertHeldReadsLikeLookup(t *testing.T, e *Entity, p *Property, want any) {
	t.Helper()

	assert.Equal(t, want, p.Value().Any)

	if fresh, ok := e.PropertyValue(p.Name()); ok {
		assert.Equal(t, fresh.Any, p.Value().Any)
	} else {
		assert.Nil(t, p.Value().Any, "the name doesn't resolve, the held property has no value")
	}
}

// A *Property held across changes of the entity's active polymorphic type (which clear the entity's
// name caches) reads what a fresh lookup reads: the value under the type active now, none while its
// name doesn't resolve, and the value again once its type is active again.
func TestPropertyValueAcrossTypeSwitch(t *testing.T) {
	t.Parallel()

	cls, modeA, modeB := polyTestClass(t)

	e := newEntity(1, 1, cls)
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})

	health := newFieldPath()
	health.path[0] = 0 // m_iHealth
	e.state.set(health, int32(100))
	e.state.set(polySubPath(), int32(5))

	hp := e.Property("m_iHealth")
	a := e.Property("m_pMode.m_aField")
	require.NotNil(t, a)

	assertHeldReadsLikeLookup(t, e, hp, int32(100))
	assertHeldReadsLikeLookup(t, e, a, int32(5))

	// Mode B: m_aField doesn't resolve any more, the sub-field slot is m_bField now
	// (the stale state under mode B's layout, see TestPolyFieldPaths).
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeB})

	assert.Nil(t, e.Property("m_pMode.m_aField"))
	assertHeldReadsLikeLookup(t, e, a, nil)
	assertHeldReadsLikeLookup(t, e, hp, int32(100))

	b := e.Property("m_pMode.m_bField")
	require.NotNil(t, b)
	assertHeldReadsLikeLookup(t, e, b, int32(5))

	e.state.set(polySubPath(), int32(7))
	assertHeldReadsLikeLookup(t, e, b, int32(7))

	// Back to mode A: the held property of mode A resolves again, the one of mode B doesn't.
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})

	assertHeldReadsLikeLookup(t, e, a, int32(7))
	assertHeldReadsLikeLookup(t, e, b, nil)
	assertHeldReadsLikeLookup(t, e, hp, int32(100))

	// The same type again is no change.
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})

	assertHeldReadsLikeLookup(t, e, a, int32(7))
}

// A re-entry resets the active polymorphic types (see TestPVS_ReentryResetsPolymorphicTypes):
// a held *Property has no value until its type is active again, then reads the new state.
func TestPropertyValueAcrossReentry(t *testing.T) {
	t.Parallel()

	cls, modeA, _ := polyTestClass(t)

	e := newEntity(1, 1, cls)
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})
	e.state.set(polySubPath(), int32(5))

	a := e.Property("m_pMode.m_aField")
	require.NotNil(t, a)
	assertHeldReadsLikeLookup(t, e, a, int32(5))

	w := &bitWriter{}
	w.fieldPathOp(t, "FieldPathEncodeFinish")

	r := newReader(w.buf)
	defer r.release()

	budget := newEntityBudget()
	paths := []*fieldPath(nil)

	e.reenter(r, nil, &paths, &budget)

	assert.Nil(t, e.polySerializers[0])
	assertHeldReadsLikeLookup(t, e, a, nil)

	// the type is sent anew, with a new value
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})
	e.state.set(polySubPath(), int32(9))

	assertHeldReadsLikeLookup(t, e, a, int32(9))
}

// BenchmarkPropertyValue reads one of an entity's 50 properties: through a held *Property (Value),
// and by name (Entity.PropertyValue) for comparison.
func BenchmarkPropertyValue(b *testing.B) {
	const n = 50

	names := make([]string, n)
	values := make([]any, n)

	for i := range n {
		names[i] = fmt.Sprintf("m_pServices.m_prop%02d", i)
		values[i] = int32(i)
	}

	e := newSyntheticEntity(nil, "CBench", 1, 1, names, values)
	name := names[n/2]

	var sink PropertyValue

	b.Run("held", func(b *testing.B) {
		p := e.Property(name)

		for b.Loop() {
			sink = p.Value()
		}
	})

	b.Run("by-name", func(b *testing.B) {
		for b.Loop() {
			sink, _ = e.PropertyValue(name)
		}
	})

	if sink.Any != int32(n/2) {
		b.Fatalf("read %v, want %d", sink.Any, n/2)
	}
}
