package panics

import (
	"errors"
	"io"
	"testing"

	dispatch "github.com/markus-wa/godispatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recoverFrom(f func()) (r any) {
	defer func() {
		r = recover()
	}()

	f()

	return nil
}

//go:noinline
func userCodeThatPanics(v any) {
	panic(v)
}

func guarded(v any) {
	defer GuardUser()

	userCodeThatPanics(v)
}

func TestGuardUser_DeferredDirectly(t *testing.T) {
	r := recoverFrom(func() { guarded("boom") })

	c, ok := r.(*Captured)
	require.True(t, ok, "expected *Captured, got %T", r)
	assert.Equal(t, "boom", c.Value)
	assert.Equal(t, "boom", c.String())
	assert.True(t, IsTagged(r))
}

func TestGuardUser_NoPanic(t *testing.T) {
	assert.Nil(t, recoverFrom(func() {
		defer GuardUser()
	}))
}

func TestGuardUser_NotDeferredDirectly(t *testing.T) {
	// recover() only works in the deferred function itself, a nested GuardUser() call doesn't stop the panic.
	r := recoverFrom(func() {
		defer func() {
			GuardUser()
		}()

		userCodeThatPanics("boom")
	})

	assert.Equal(t, "boom", r)
	assert.False(t, IsTagged(r))
}

func TestCapture_StackContainsPanickingFunction(t *testing.T) {
	r := recoverFrom(func() { guarded("boom") })

	c, ok := r.(*Captured)
	require.True(t, ok)
	assert.Contains(t, string(c.Stack), "userCodeThatPanics")
}

func TestCapture_UnwrapsNestedConsumerCodePanic(t *testing.T) {
	inner := new(dispatch.Dispatcher)
	inner.RegisterHandler(func(int) {
		userCodeThatPanics("boom")
	})

	outer := new(dispatch.Dispatcher)
	outer.RegisterHandler(func(string) {
		inner.Dispatch(1)
	})

	r := recoverFrom(func() { outer.Dispatch("x") })

	ccp, ok := r.(dispatch.ConsumerCodePanic)
	require.True(t, ok, "expected ConsumerCodePanic, got %T", r)
	_, nested := ccp.Value().(dispatch.ConsumerCodePanic)
	require.True(t, nested, "expected nested ConsumerCodePanic")
	assert.True(t, IsTagged(r))

	c := Capture(r)
	assert.Equal(t, "boom", c.Value)
}

func TestCapture_KeepsExistingCaptured(t *testing.T) {
	d := new(dispatch.Dispatcher)
	d.RegisterHandler(func(int) {
		guarded("boom")
	})

	r := recoverFrom(func() { d.Dispatch(1) })

	ccp, ok := r.(dispatch.ConsumerCodePanic)
	require.True(t, ok)

	orig, ok := ccp.Value().(*Captured)
	require.True(t, ok)

	assert.Same(t, orig, Capture(r))
	assert.Same(t, orig, Capture(orig))
	assert.Contains(t, string(Capture(r).Stack), "userCodeThatPanics")
}

func TestCapture_UnwrappedPanicsStayUntagged(t *testing.T) {
	d := new(dispatch.Dispatcher)
	d.RegisterHandlerWithOptions(func(int) {
		userCodeThatPanics("boom")
	}, dispatch.HandlerOptions{UnwrappedPanics: true})

	r := recoverFrom(func() { d.Dispatch(1) })

	assert.Equal(t, "boom", r)
	assert.False(t, IsTagged(r))
}

func TestIsTagged(t *testing.T) {
	assert.False(t, IsTagged(nil))
	assert.False(t, IsTagged("boom"))
	assert.False(t, IsTagged(errors.New("boom")))
	assert.True(t, IsTagged(&Captured{Value: "boom"}))
}

func TestReadError(t *testing.T) {
	err := error(&ReadError{Err: io.ErrClosedPipe})

	assert.ErrorIs(t, err, io.ErrClosedPipe)
	assert.Equal(t, "failed to read demo stream: io: read/write on closed pipe", err.Error())
}
