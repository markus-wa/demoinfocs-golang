package demoinfocs

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
)

type internalHandlerTestMsg struct{}

type internalHandlerTestEvent struct{}

func newInternalHandlerTestParser(t *testing.T) *Parser {
	t.Helper()

	p := NewParser(bytes.NewReader(make([]byte, 1024)))
	t.Cleanup(func() { _ = p.Close() })

	return p
}

// registerWrapped registers h like RegisterNetMessageHandler does (= how internal handlers were registered before registerInternal).
func registerWrapped[T any](p *Parser, h func(T)) {
	p.msgDispatcher.RegisterHandler(h)
}

// queuedPanicError dispatches a message that makes a handler panic via the message queue (= the message-handling goroutine)
// and returns the resulting parser error (set by the dispatcher's PanicHandler).
func queuedPanicError(t *testing.T, register func(p *Parser, h func(*internalHandlerTestMsg)), panicValue any) error {
	t.Helper()

	p := newInternalHandlerTestParser(t)
	p.initMsgQueue(1)

	register(p, func(*internalHandlerTestMsg) {
		panic(panicValue)
	})

	laterHandlerCalled := false

	register(p, func(*internalHandlerTestMsg) {
		laterHandlerCalled = true
	})

	p.msgQueue <- &internalHandlerTestMsg{}

	p.msgDispatcher.SyncAllQueues()

	assert.False(t, laterHandlerCalled, "handlers after the panicking one must not be called")

	return p.error()
}

func errorMessage(t *testing.T, err error) string {
	t.Helper()

	require.Error(t, err)

	msg, _, found := strings.Cut(err.Error(), "\nstacktrace:\n")
	require.True(t, found, "error should contain a stack trace: %q", err.Error())

	return msg
}

// Internal handlers no longer have their panics wrapped in a dispatch.ConsumerCodePanic,
// the resulting parser error must stay the same.
func TestRegisterInternal_PanicSetsSameError(t *testing.T) {
	for _, v := range []any{"boom", errors.New("boom error"), 42} {
		wrapped := errorMessage(t, queuedPanicError(t, registerWrapped[*internalHandlerTestMsg], v))
		internal := errorMessage(t, queuedPanicError(t, registerInternal[*internalHandlerTestMsg], v))

		assert.Equal(t, wrapped, internal)
	}

	assert.Equal(t, "boom", errorMessage(t, queuedPanicError(t, registerInternal[*internalHandlerTestMsg], "boom")))
}

// A user event handler panicking inside an internal net-message handler.
func TestRegisterInternal_NestedUserPanicSetsSameError(t *testing.T) {
	nested := func(register func(p *Parser, h func(*internalHandlerTestMsg))) error {
		p := newInternalHandlerTestParser(t)
		p.initMsgQueue(1)

		p.RegisterEventHandler(func(internalHandlerTestEvent) {
			panic("user boom")
		})

		register(p, func(*internalHandlerTestMsg) {
			p.eventDispatcher.Dispatch(internalHandlerTestEvent{})
		})

		p.msgQueue <- &internalHandlerTestMsg{}

		p.msgDispatcher.SyncAllQueues()

		return p.error()
	}

	wrapped := errorMessage(t, nested(registerWrapped[*internalHandlerTestMsg]))
	internal := errorMessage(t, nested(registerInternal[*internalHandlerTestMsg]))

	assert.Equal(t, "user boom", internal)
	assert.Equal(t, wrapped, internal)
}

// Net-messages dispatched synchronously on the reader goroutine (e.g. from handleDemoPacket):
// recoverFromUnexpectedEOF() must re-panic with the same raw value as before.
func TestRegisterInternal_ReaderSidePanicSameValue(t *testing.T) {
	recovered := func(register func(p *Parser, h func(*internalHandlerTestMsg))) (r any) {
		p := newInternalHandlerTestParser(t)

		register(p, func(*internalHandlerTestMsg) {
			panic("boom")
		})

		defer func() {
			r = recover()
		}()

		p.msgDispatcher.Dispatch(&internalHandlerTestMsg{})

		return nil
	}

	for _, register := range []func(p *Parser, h func(*internalHandlerTestMsg)){
		registerWrapped[*internalHandlerTestMsg],
		registerInternal[*internalHandlerTestMsg],
	} {
		r := recovered(register)
		require.NotNil(t, r)

		assert.PanicsWithValue(t, "boom", func() {
			_ = recoverFromUnexpectedEOF(r)
		})
	}
}

// A user send-table callback (tagged as *panics.Captured) panicking inside an internal net-message handler
// must result in the same error as the untagged panic value.
func TestCapturedPanicSetsSameError(t *testing.T) {
	for _, v := range []any{"boom", errors.New("boom error"), 42} {
		raw := errorMessage(t, queuedPanicError(t, registerInternal[*internalHandlerTestMsg], v))
		captured := errorMessage(t, queuedPanicError(t, registerInternal[*internalHandlerTestMsg], panics.Capture(v)))

		assert.Equal(t, raw, captured)
	}

	assert.PanicsWithValue(t, "boom", func() {
		_ = recoverFromUnexpectedEOF(panics.Capture("boom"))
	})
}
