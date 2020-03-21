package demoinfocs

import (
	"errors"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/common"
)

func TestParser_CurrentFrame(t *testing.T) {
	assert.Equal(t, 1, (&Parser{currentFrame: 1}).CurrentFrame())
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
		header: &common.DemoHeader{
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
		header: &common.DemoHeader{
			PlaybackTime:  time.Second,
			PlaybackTicks: 5,
		},
	}

	assert.Equal(t, time.Duration(200)*time.Millisecond, p.TickTime())
}

func TestParser_Progress_NoHeader(t *testing.T) {
	assert.Zero(t, new(Parser).Progress())
	assert.Zero(t, (&Parser{header: &common.DemoHeader{}}).Progress())
}

func TestRecoverFromPanic(t *testing.T) {
	assert.Nil(t, recoverFromPanic(nil))
	assert.Equal(t, ErrUnexpectedEndOfDemo, recoverFromPanic(io.ErrUnexpectedEOF))
	assert.Equal(t, ErrUnexpectedEndOfDemo, recoverFromPanic(io.EOF))

	err := errors.New("test")
	assert.Equal(t, err, recoverFromPanic(err))

	assert.Equal(t, "test", recoverFromPanic("test").Error())
	assert.Equal(t, "unexpected error: 1", recoverFromPanic(1).Error())
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
		err := recoverFromPanic(consumerCodePanicMock{value: 1})
		assert.Nil(t, err)
	})
}

func TestParser_SetError(t *testing.T) {
	err := errors.New("test")

	p := new(Parser)
	p.setError(err)

	assert.Same(t, err, p.error())
}

func TestParser_SetError_Multiple(t *testing.T) {
	err := errors.New("test")

	p := new(Parser)
	p.setError(err)
	p.setError(errors.New("second error"))

	assert.Same(t, err, p.error())
}
