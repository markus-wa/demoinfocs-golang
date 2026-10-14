package demoinfocs

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func newTestAbortState() (*abortState, *int) {
	var stops int

	return &abortState{onStop: func() { stops++ }}, &stops
}

func TestAbortState_FirstErrorWins(t *testing.T) {
	errFirst := errors.New("first")
	errSecond := errors.New("second")

	t.Run("error before cancellation", func(t *testing.T) {
		a, stops := newTestAbortState()

		a.setError(errFirst)
		a.setCancelled(errSecond)

		assert.Same(t, errFirst, a.error())
		assert.True(t, a.aborted())
		assert.Equal(t, 1, *stops)
	})

	t.Run("cancellation before error", func(t *testing.T) {
		a, _ := newTestAbortState()

		a.setCancelled(errFirst)
		a.setError(errSecond)

		assert.Same(t, errFirst, a.error())
	})

	t.Run("reader error before cancellation", func(t *testing.T) {
		a, _ := newTestAbortState()

		a.setReaderError(errFirst)
		a.setCancelled(errSecond)

		assert.Same(t, errFirst, a.error())
	})

	t.Run("nil is ignored", func(t *testing.T) {
		a, stops := newTestAbortState()

		a.setError(nil)
		a.setReaderError(nil)

		assert.NoError(t, a.error())
		assert.False(t, a.aborted())
		assert.Zero(t, *stops)
	})
}

// context.AfterFunc() records the cancellation asynchronously, an error recorded after ctx is done is too late.
func TestAbortState_CancelledContextIsFirst(t *testing.T) {
	errStop := errors.New("stop")

	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errStop)

	for _, record := range []func(*abortState, error){(*abortState).setError, (*abortState).setReaderError} {
		a, _ := newTestAbortState()
		a.ctx = ctx

		assert.True(t, a.aborted())

		record(a, errors.New("decode"))

		assert.Same(t, errStop, a.error())
	}
}

// Errors of the parsing goroutine don't stop the handlers: the queued messages must still be handled.
func TestAbortState_ReaderErrorDoesntStop(t *testing.T) {
	a, stops := newTestAbortState()

	a.setReaderError(io.ErrUnexpectedEOF)

	assert.False(t, a.aborted())
	assert.Zero(t, *stops)

	a.stop()
	a.stop()

	assert.True(t, a.aborted())
	assert.Equal(t, 1, *stops)
}

func TestAbortState_PanicWins(t *testing.T) {
	a, stops := newTestAbortState()

	a.setError(errors.New("first"))
	a.setPanic(panics.Capture("boom"))
	a.setPanic(panics.Capture("second"))

	pe, err := a.result()
	require.NotNil(t, pe)
	assert.Equal(t, "boom", pe.Value)
	assert.EqualError(t, err, "first")
	assert.Equal(t, 1, *stops)
}

// handmadeDemo returns a demo consisting of the given frames.
func handmadeDemo(frames ...[]byte) []byte {
	b := []byte("PBDEMS2\x00")
	b = append(b, make([]byte, 8)...) // skipped

	for _, f := range frames {
		b = append(b, f...)
	}

	return b
}

// demoPacketFrame returns a DEM_Packet frame with the given packet data.
func demoPacketFrame(t *testing.T, data []byte) []byte {
	t.Helper()

	payload, err := proto.Marshal(&msg.CDemoPacket{Data: data})
	require.NoError(t, err)

	b := binary.AppendUvarint(nil, uint64(msg.EDemoCommands_DEM_Packet))
	b = binary.AppendUvarint(b, 0) // tick
	b = binary.AppendUvarint(b, uint64(len(payload)))

	return append(b, payload...)
}

// corruptPacketData is packet data with a message (type 0) whose size (16 bytes) runs past the end of the data.
// The nested bit reader runs out of data while reading it.
var corruptPacketData = []byte{0x00, 0x04, 0x00}

// A nested reader running out of data is a decoding failure, not the end of the demo.
func TestCorruptDemoPacket(t *testing.T) {
	data := handmadeDemo(demoPacketFrame(t, nil), demoPacketFrame(t, corruptPacketData))

	t.Run("ParseToEnd", func(t *testing.T) {
		p := NewParser(bytes.NewReader(data))
		defer p.Close()

		err := p.ParseToEnd(t.Context())
		assert.ErrorIs(t, err, ErrDecodeFailed)
		assert.NotErrorIs(t, err, ErrUnexpectedEndOfDemo)

		var de *DecodeError
		if assert.ErrorAs(t, err, &de) {
			assert.Equal(t, 1, de.Frame)
		}
	})

	t.Run("ParseNextFrame", func(t *testing.T) {
		p := NewParser(bytes.NewReader(data))
		defer p.Close()

		more, err := p.ParseNextFrame()
		assert.True(t, more)
		assert.NoError(t, err)

		more, err = p.ParseNextFrame()
		assert.False(t, more)
		assert.ErrorIs(t, err, ErrDecodeFailed)
		assert.NotErrorIs(t, err, ErrUnexpectedEndOfDemo)
	})
}

// A cancellation of ctx while the queue is drained after a decoding failure doesn't replace the error.
func TestPrecedence_DecodeErrorOverLaterCancel(t *testing.T) {
	p := NewParser(bytes.NewReader(handmadeDemo(demoPacketFrame(t, nil), demoPacketFrame(t, corruptPacketData))))
	defer p.Close()

	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	errStop := errors.New("stop")

	// Called on the message-handling goroutine for the first frame,
	// it blocks until the parsing goroutine recorded the error of the second frame and is draining the queue.
	p.RegisterEventHandler(func(events.FrameDone) {
		deadline := time.Now().Add(10 * time.Second)

		for p.abort.error() == nil && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		cancel(errStop)
	})

	err := p.ParseToEnd(ctx)
	assert.ErrorIs(t, err, ErrDecodeFailed)
	assert.NotErrorIs(t, err, errStop)
}
