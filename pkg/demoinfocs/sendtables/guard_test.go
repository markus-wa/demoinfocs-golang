package sendtables

import (
	"strings"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
)

func recoverPanic(f func()) (r any) {
	defer func() {
		r = recover()
	}()

	f()

	return nil
}

//go:noinline
func userCallbackThatPanics() {
	panic("boom")
}

func requireTagged(t *testing.T, r any) {
	t.Helper()

	c, ok := r.(*panics.Captured)
	require.True(t, ok, "expected *panics.Captured, got %T (%v)", r, r)
	assert.Equal(t, "boom", c.Value)
	assert.Contains(t, string(c.Stack), "userCallbackThatPanics")
	assert.Equal(t, 1, strings.Count(string(c.Stack), "userCallbackThatPanics("), "the panic must be captured exactly once")
}

func requireUntagged(t *testing.T, r any) {
	t.Helper()

	assert.Equal(t, "boom", r)
}

const guardTestClass = "CGuardTest"

func newGuardTestEntity() (*Parser, *Entity) {
	p := newSyntheticParser([]string{guardTestClass})
	e := newSyntheticEntity(p, guardTestClass, 1, 1, []string{"m_x"}, []any{1})

	return p, e
}

func TestGuards_OnUpdate(t *testing.T) {
	_, e := newGuardTestEntity()
	e.Property("m_x").OnUpdate(func(PropertyValue) { userCallbackThatPanics() })

	requireTagged(t, recoverPanic(func() { syntheticSet(e, nil, []string{"m_x"}, []any{2}, true, false) }))

	_, e = newGuardTestEntity()
	e.Property("m_x").onUpdate(func(PropertyValue) { userCallbackThatPanics() })

	requireUntagged(t, recoverPanic(func() { syntheticSet(e, nil, []string{"m_x"}, []any{2}, true, false) }))
}

// Bind and BindProperty tag the binding's panics (e.g. a wrong pointer type), bind and bindProperty don't.
func TestGuards_Bind(t *testing.T) {
	bindings := []struct {
		name   string
		public func(e *Entity, variable any)
		intern func(e *Entity, variable any)
	}{
		{"Bind", func(e *Entity, v any) { e.Property("m_x").Bind(v, ValTypeInt) }, func(e *Entity, v any) { e.Property("m_x").bind(v, ValTypeInt) }},
		{"BindProperty", func(e *Entity, v any) { e.BindProperty("m_x", v, ValTypeInt) }, func(e *Entity, v any) { e.bindProperty("m_x", v, ValTypeInt) }},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			var wrongType string // bound as int, the binding's type assertion panics

			_, e := newGuardTestEntity()
			b.public(e, &wrongType)

			r := recoverPanic(func() { syntheticSet(e, nil, []string{"m_x"}, []any{2}, true, false) })
			_, ok := r.(*panics.Captured)
			assert.True(t, ok, "expected *panics.Captured, got %T", r)

			_, e = newGuardTestEntity()
			b.intern(e, &wrongType)

			r = recoverPanic(func() { syntheticSet(e, nil, []string{"m_x"}, []any{2}, true, false) })
			assert.NotNil(t, r)
			assert.False(t, panics.IsTagged(r))

			var i int

			_, e = newGuardTestEntity()
			b.public(e, &i)
			syntheticSet(e, nil, []string{"m_x"}, []any{int32(2)}, true, false)

			assert.Equal(t, 2, i)
		})
	}
}

func setPosition(e *Entity) {
	syntheticSet(e, nil, syntheticPositionProps, []any{uint64(1), uint64(1), uint64(1), float32(1), float32(1), float32(1)}, true, false)
}

func TestGuards_OnPositionUpdate(t *testing.T) {
	_, e := newGuardTestEntity()
	e.OnPositionUpdate(func(r3.Vector) { userCallbackThatPanics() })

	requireTagged(t, recoverPanic(func() { setPosition(e) }))

	_, e = newGuardTestEntity()
	e.onPositionUpdate(func(r3.Vector) { userCallbackThatPanics() })

	requireUntagged(t, recoverPanic(func() { setPosition(e) }))
}

func TestGuards_OnDestroy(t *testing.T) {
	p, e := newGuardTestEntity()
	e.OnDestroy(userCallbackThatPanics)

	requireTagged(t, recoverPanic(func() { syntheticDestroy(e, p) }))

	p, e = newGuardTestEntity()
	e.addOnDestroy(userCallbackThatPanics)

	requireUntagged(t, recoverPanic(func() { syntheticDestroy(e, p) }))
}

func TestGuards_OnCreateFinished(t *testing.T) {
	p, e := newGuardTestEntity()
	e.OnCreateFinished(userCallbackThatPanics)

	requireTagged(t, recoverPanic(func() { syntheticCreate(e, p) }))

	p, e = newGuardTestEntity()
	e.addOnCreateFinished(userCallbackThatPanics)

	requireUntagged(t, recoverPanic(func() { syntheticCreate(e, p) }))
}

func TestGuards_OnUpdateFinished(t *testing.T) {
	p, e := newGuardTestEntity()
	e.OnUpdateFinished(userCallbackThatPanics)

	requireTagged(t, recoverPanic(func() { syntheticSet(e, p, []string{"m_x"}, []any{2}, true, true) }))

	p, e = newGuardTestEntity()
	e.addOnUpdateFinished(userCallbackThatPanics)

	requireUntagged(t, recoverPanic(func() { syntheticSet(e, p, []string{"m_x"}, []any{2}, true, true) }))
}

func TestGuards_OnEntityCreated(t *testing.T) {
	p, e := newGuardTestEntity()
	p.classesByName[guardTestClass].OnEntityCreated(func(*Entity) { userCallbackThatPanics() })

	requireTagged(t, recoverPanic(func() { syntheticCreate(e, p) }))

	p, e = newGuardTestEntity()
	p.classesByName[guardTestClass].onEntityCreated(func(*Entity) { userCallbackThatPanics() })

	requireUntagged(t, recoverPanic(func() { syntheticCreate(e, p) }))
}

// A user handler registered from an internal callback is still guarded, an internal one registered from a user callback isn't.
func TestGuards_Nested(t *testing.T) {
	p, e := newGuardTestEntity()
	p.classesByName[guardTestClass].onEntityCreated(func(e *Entity) {
		e.Property("m_x").OnUpdate(func(PropertyValue) { userCallbackThatPanics() })
	})

	requireTagged(t, recoverPanic(func() { syntheticCreate(e, p) }))
}
