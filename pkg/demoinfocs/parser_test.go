package demoinfocs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"testing"
	"time"

	dispatch "github.com/markus-wa/godispatch"
	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
)

func TestParser_CurrentFrame(t *testing.T) {
	assert.Equal(t, 1, (&Parser{currentFrame: 1}).CurrentFrame())
}

func TestParser_ServerClasses_BeforeParsing(t *testing.T) {
	p := NewParser(bytes.NewReader(make([]byte, 1024)))
	defer p.Close()

	assert.NotPanics(t, func() {
		assert.Nil(t, p.ServerClasses().All())
		assert.Nil(t, p.ServerClasses().FindByName("x"))
	})
}

func TestParser_GameState(t *testing.T) {
	gs := new(GameState)
	assert.Equal(t, gs, (&Parser{gameState: gs}).GameState())
}

func TestParser_CurrentTime(t *testing.T) {
	p := &Parser{
		tickInterval: 2,
		gameState:    &GameState{ingameTick: 3},
	}

	assert.Equal(t, 6*time.Second, p.CurrentTime())
}

func TestParser_TickRate(t *testing.T) {
	assert.Equal(t, float64(5), math.Round((&Parser{tickInterval: 0.2}).TickRate()))
}

func TestParser_TickRate_FallbackToHeader(t *testing.T) {
	p := &Parser{
		header: &header{
			PlaybackTime:  time.Second,
			PlaybackTicks: 5,
		},
	}

	assert.Equal(t, float64(5), p.TickRate())
}

func TestParser_TickTime(t *testing.T) {
	assert.Equal(t, time.Duration(200)*time.Millisecond, (&Parser{tickInterval: 0.2}).TickTime())
}

func TestParser_TickTime_FallbackToHeader(t *testing.T) {
	p := &Parser{
		header: &header{
			PlaybackTime:  time.Second,
			PlaybackTicks: 5,
		},
	}

	assert.Equal(t, time.Duration(200)*time.Millisecond, p.TickTime())
}

func TestParser_Progress_NoHeader(t *testing.T) {
	assert.Zero(t, new(Parser).Progress())
	assert.Zero(t, (&Parser{header: &header{}}).Progress())
}

func finishParsingWith(t *testing.T, r any) (err error) {
	t.Helper()

	p := NewParser(bytes.NewReader(nil))
	t.Cleanup(func() { _ = p.Close() })

	return p.finishParsing(r, nil, true, func() {})
}

func TestFinishParsing_EndOfDemo(t *testing.T) {
	assert.NoError(t, finishParsingWith(t, nil))

	for _, r := range []any{io.EOF, io.ErrUnexpectedEOF} {
		err := finishParsingWith(t, r)
		assert.ErrorIs(t, err, ErrUnexpectedEndOfDemo)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.EqualError(t, err, "unexpected EOF: "+ErrUnexpectedEndOfDemo.Error())
	}
}

func TestFinishParsing_DecodeError(t *testing.T) {
	cause := errors.New("test")

	err := finishParsingWith(t, cause)
	assert.ErrorIs(t, err, ErrDecodeFailed)
	assert.ErrorIs(t, err, cause)
	assert.NotErrorIs(t, err, ErrUnexpectedEndOfDemo)

	var de *DecodeError
	assert.ErrorAs(t, err, &de)
	assert.Zero(t, de.Frame)
	assert.Nil(t, de.Stack)

	// a wrapped EOF (e.g. of a nested reader) isn't the end of the demo
	assert.ErrorIs(t, finishParsingWith(t, fmt.Errorf("nested: %w", io.EOF)), ErrDecodeFailed)
}

func TestFinishParsing_ReadError(t *testing.T) {
	err := finishParsingWith(t, &panics.ReadError{Err: io.ErrClosedPipe})
	assert.ErrorIs(t, err, io.ErrClosedPipe)
	assert.NotErrorIs(t, err, ErrDecodeFailed)
	assert.EqualError(t, err, "reading demo: "+io.ErrClosedPipe.Error())
}

type consumerCodePanicMock struct {
	value any
}

func (ucp consumerCodePanicMock) String() string {
	return fmt.Sprint(ucp.value)
}

func (ucp consumerCodePanicMock) Value() any {
	return ucp.value
}

func TestFinishParsing_ConsumerCodePanic(t *testing.T) {
	defer func() {
		pe, ok := recover().(*PanicError)
		if assert.True(t, ok, "expected *PanicError") {
			assert.Equal(t, 1, pe.Value)
		}
	}()

	_ = finishParsingWith(t, consumerCodePanicMock{value: 1})

	assert.Fail(t, "expected panic")
}

func TestParser_SetError(t *testing.T) {
	err := errors.New("test")

	p := new(Parser)
	p.setError(err)

	assert.Same(t, err, p.abort.error())
}

func TestParser_SetError_Multiple(t *testing.T) {
	err := errors.New("test")

	p := new(Parser)
	p.setError(err)
	p.setError(errors.New("second error"))

	assert.Same(t, err, p.abort.error())
}

func TestParser_Close(t *testing.T) {
	p := new(Parser)
	q := make(chan any, 1)

	p.msgDispatcher = new(dispatch.Dispatcher)
	p.msgDispatcher.AddQueues(q)

	called := false
	p.msgDispatcher.RegisterHandler(func(any) {
		called = true
	})

	err := p.Close()
	assert.NoError(t, err)

	q <- "this should not trigger the handler"

	p.msgDispatcher.SyncAllQueues()

	assert.False(t, called)
}

// TestParser_SetError_UnregistersHandlers ensures that recording a fatal error
// stops further handler work, so queued backlog messages are not delivered to
// user-registered handlers anymore.
func TestParser_SetError_UnregistersHandlers(t *testing.T) {
	p := new(Parser)

	p.msgDispatcher = new(dispatch.Dispatcher)
	p.eventDispatcher = new(dispatch.Dispatcher)
	p.abort.onStop = p.unregisterAllHandlers

	var msgCalls int32
	p.msgDispatcher.RegisterHandler(func(any) {
		atomic.AddInt32(&msgCalls, 1)
	})

	var eventCalls int32
	p.eventDispatcher.RegisterHandler(func(any) {
		atomic.AddInt32(&eventCalls, 1)
	})

	p.setError(errors.New("fatal"))

	q := make(chan any, 1)
	p.msgDispatcher.AddQueues(q)
	p.msgDispatcher.Dispatch("msg backlog")
	p.msgDispatcher.SyncAllQueues()

	p.eventDispatcher.Dispatch("event backlog")

	assert.Zero(t, atomic.LoadInt32(&msgCalls))
	assert.Zero(t, atomic.LoadInt32(&eventCalls))
}
