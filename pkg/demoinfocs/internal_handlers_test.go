package demoinfocs

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

type internalHandlerTestMsg struct{}

type internalHandlerTestEvent struct{}

func newInternalHandlerTestParser(t *testing.T) *Parser {
	t.Helper()

	p := NewParser(bytes.NewReader(make([]byte, 1024)))
	t.Cleanup(func() { _ = p.Close() })

	p.initMsgQueue(1)

	return p
}

// dispatchQueued dispatches a message via the message queue (= the message-handling goroutine, like during parsing).
func dispatchQueued(p *Parser) {
	p.msgQueue <- &internalHandlerTestMsg{}

	p.msgDispatcher.SyncAllQueues()
}

// A panic of an internal handler is a decoding failure, the handlers after it aren't called.
func TestRegisterInternal_PanicIsDecodeError(t *testing.T) {
	for _, v := range []any{"boom", errors.New("boom"), 42} {
		p := newInternalHandlerTestParser(t)
		p.currentFrame = 7

		registerInternal(p, func(*internalHandlerTestMsg) {
			panic(v)
		})

		laterHandlerCalled := false

		registerInternal(p, func(*internalHandlerTestMsg) {
			laterHandlerCalled = true
		})

		dispatchQueued(p)

		assert.False(t, laterHandlerCalled, "handlers after the panicking one must not be called")
		assert.Nil(t, p.abort.handlerPanic())

		var de *DecodeError
		require.ErrorAs(t, p.abort.error(), &de)
		assert.ErrorIs(t, de, ErrDecodeFailed)
		assert.Equal(t, 7, de.Frame)
		assert.NotNil(t, de.Cause)
	}
}

// A user event handler panicking inside an internal net-message handler is a handler panic.
func TestRegisterInternal_NestedUserPanic(t *testing.T) {
	p := newInternalHandlerTestParser(t)

	p.RegisterEventHandler(func(internalHandlerTestEvent) {
		panic("user boom")
	})

	registerInternal(p, func(*internalHandlerTestMsg) {
		p.eventDispatcher.Dispatch(internalHandlerTestEvent{})
	})

	dispatchQueued(p)

	pe := p.abort.handlerPanic()
	require.NotNil(t, pe)
	assert.Equal(t, "user boom", pe.Value)
	assert.Contains(t, string(pe.Stack), "TestRegisterInternal_NestedUserPanic")
	assert.NoError(t, p.abort.error())
	assert.True(t, p.aborted())
}

// A user send-table callback (tagged as *panics.Captured) panicking inside an internal net-message handler
// is a handler panic, the originally captured stack is kept.
func TestRegisterInternal_CapturedPanic(t *testing.T) {
	p := newInternalHandlerTestParser(t)

	c := panics.Capture("boom")

	registerInternal(p, func(*internalHandlerTestMsg) {
		panic(c)
	})

	dispatchQueued(p)

	pe := p.abort.handlerPanic()
	require.NotNil(t, pe)
	assert.Equal(t, "boom", pe.Value)
	assert.Equal(t, c.Stack, pe.Stack)
}

// A missing property is a decoding failure, also when a user handler looked it up.
func TestRegisterInternal_PropertyNotFoundInUserHandler(t *testing.T) {
	p := newInternalHandlerTestParser(t)

	pnf := &st.PropertyNotFoundError{Name: "m_x"}

	p.RegisterEventHandler(func(internalHandlerTestEvent) {
		panic(pnf)
	})

	registerInternal(p, func(*internalHandlerTestMsg) {
		p.eventDispatcher.Dispatch(internalHandlerTestEvent{})
	})

	dispatchQueued(p)

	assert.Nil(t, p.abort.handlerPanic())
	assert.ErrorIs(t, p.abort.error(), ErrDecodeFailed)

	var err *st.PropertyNotFoundError
	require.ErrorAs(t, p.abort.error(), &err)
	assert.Same(t, pnf, err)
}

// Only the first handler panic is kept.
func TestSetPanic_FirstWins(t *testing.T) {
	p := newInternalHandlerTestParser(t)

	p.abort.setPanic(panics.Capture("first"))
	p.abort.setPanic(panics.Capture("second"))

	assert.Equal(t, "first", p.abort.handlerPanic().Value)
}
