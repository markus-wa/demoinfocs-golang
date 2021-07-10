package demoinfocs

import (
	"errors"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	dispatch "github.com/markus-wa/godispatch"
	"github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
)

func TestParser_CurrentFrame(t *testing.T) {
	assert.Equal(t, 1, (&parser{currentFrame: 1}).CurrentFrame())
}

func TestParser_GameState(t *testing.T) {
	gs := new(gameState)
	assert.Equal(t, gs, (&parser{gameState: gs}).GameState())
}

func TestParser_CurrentTime(t *testing.T) {
	p := &parser{
		tickInterval: 2,
		gameState:    &gameState{ingameTick: 3},
	}

	assert.Equal(t, 6*time.Second, p.CurrentTime())
}

func TestParser_TickRate(t *testing.T) {
	assert.Equal(t, float64(5), math.Round((&parser{tickInterval: 0.2}).TickRate()))
}

func TestParser_TickRate_FallbackToHeader(t *testing.T) {
	p := &parser{
		header: &common.DemoHeader{
			PlaybackTime:  time.Second,
			PlaybackTicks: 5,
		},
	}

	assert.Equal(t, float64(5), p.TickRate())
}

func TestParser_TickTime(t *testing.T) {
	assert.Equal(t, time.Duration(200)*time.Millisecond, (&parser{tickInterval: 0.2}).TickTime())
}

func TestParser_TickTime_FallbackToHeader(t *testing.T) {
	p := &parser{
		header: &common.DemoHeader{
			PlaybackTime:  time.Second,
			PlaybackTicks: 5,
		},
	}

	assert.Equal(t, time.Duration(200)*time.Millisecond, p.TickTime())
}

func TestParser_Progress_NoHeader(t *testing.T) {
	assert.Zero(t, new(parser).Progress())
	assert.Zero(t, (&parser{header: &common.DemoHeader{}}).Progress())
}

func TestRecoverFromUnexpectedEOF(t *testing.T) {
	assert.Nil(t, recoverFromUnexpectedEOF(nil))
	assert.Equal(t, ErrUnexpectedEndOfDemo, recoverFromUnexpectedEOF(io.ErrUnexpectedEOF))
	assert.Equal(t, ErrUnexpectedEndOfDemo, recoverFromUnexpectedEOF(io.EOF))

	assert.Panics(t, func() {
		r := recoverFromUnexpectedEOF(errors.New("test"))
		assert.Failf(t, "expected panic, got recovery", "recovered value = '%v'", r)
	})
}

type consumerCodePanicMock struct {
	value interface{}
}

func (ucp consumerCodePanicMock) String() string {
	return fmt.Sprint(ucp.value)
}

func (ucp consumerCodePanicMock) Value() interface{} {
	return ucp.value
}

func TestRecoverFromPanic_ConsumerCodePanic(t *testing.T) {
	assert.PanicsWithValue(t, 1, func() {
		err := recoverFromUnexpectedEOF(consumerCodePanicMock{value: 1})
		assert.Nil(t, err)
	})
}

func TestParser_SetError(t *testing.T) {
	err := errors.New("test")

	p := new(parser)
	p.setError(err)

	assert.Same(t, err, p.error())
}

func TestParser_SetError_Multiple(t *testing.T) {
	err := errors.New("test")

	p := new(parser)
	p.setError(err)
	p.setError(errors.New("second error"))

	assert.Same(t, err, p.error())
}

func TestParser_Close(t *testing.T) {
	p := new(parser)
	q := make(chan interface{}, 1)

	p.msgDispatcher = new(dispatch.Dispatcher)
	p.msgDispatcher.AddQueues(q)

	called := false
	p.msgDispatcher.RegisterHandler(func(interface{}) {
		called = true
	})

	err := p.Close()
	assert.NoError(t, err)

	q <- "this should not trigger the handler"

	p.msgDispatcher.SyncAllQueues()

	assert.False(t, called)
}
