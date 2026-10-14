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

// errorOf returns the error of a.result().
func errorOf(a *abortState) error {
	_, err := a.result()

	return err
}

// panicOf returns the handler panic of a.result().
func panicOf(a *abortState) *PanicError {
	pe, _ := a.result()

	return pe
}

func TestAbortState_Precedence(t *testing.T) {
	errQueue := errors.New("queue")
	errQueue2 := errors.New("queue 2")
	errCancel := errors.New("cancel")
	errReader := errors.New("reader")

	tests := []struct {
		name   string
		record func(a *abortState)
		want   error
	}{
		{"error before cancellation", func(a *abortState) {
			a.setError(errQueue)
			a.setCancelled(errCancel)
		}, errQueue},
		{"cancellation before error", func(a *abortState) {
			a.setCancelled(errCancel)
			a.setError(errQueue)
		}, errCancel},
		{"first error wins", func(a *abortState) {
			a.setError(errQueue)
			a.setError(errQueue2)
		}, errQueue},
		{"reader error before cancellation", func(a *abortState) {
			a.setReaderError(errReader)
			a.setCancelled(errCancel)
		}, errReader},
		{"cancellation before reader error", func(a *abortState) {
			a.setCancelled(errCancel)
			a.setReaderError(errReader)
		}, errCancel},
		// the error of the message-handling goroutine comes from an earlier frame
		{"reader error before error", func(a *abortState) {
			a.setReaderError(errReader)
			a.setError(errQueue)
		}, errQueue},
		{"error before reader error", func(a *abortState) {
			a.setError(errQueue)
			a.setReaderError(errReader)
		}, errQueue},
		{"reader error, error, cancellation", func(a *abortState) {
			a.setReaderError(errReader)
			a.setError(errQueue)
			a.setCancelled(errCancel)
		}, errQueue},
		{"reader error, cancellation, error", func(a *abortState) {
			a.setReaderError(errReader)
			a.setCancelled(errCancel)
			a.setError(errQueue)
		}, errReader},
		{"first reader error wins", func(a *abortState) {
			a.setReaderError(errReader)
			a.setReaderError(errQueue)
		}, errReader},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := newTestAbortState()

			tc.record(a)

			assert.Same(t, tc.want, errorOf(a))
		})
	}

	t.Run("nil is ignored", func(t *testing.T) {
		a, stops := newTestAbortState()

		a.setError(nil)
		a.setReaderError(nil)

		assert.NoError(t, errorOf(a))
		assert.False(t, a.aborted())
		assert.Zero(t, *stops)
	})

	t.Run("errors stop once", func(t *testing.T) {
		a, stops := newTestAbortState()

		a.setError(errQueue)
		a.setCancelled(errCancel)

		assert.True(t, a.aborted())
		assert.Equal(t, 1, *stops)
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

		assert.Same(t, errStop, errorOf(a))
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

		assertCorruptDemoPacket(t, p.ParseToEnd(t.Context()))
	})

	t.Run("ParseNextFrame", func(t *testing.T) {
		p := NewParser(bytes.NewReader(data))
		defer p.Close()

		more, err := p.ParseNextFrame()
		assert.True(t, more)
		assert.NoError(t, err)

		more, err = p.ParseNextFrame()
		assert.False(t, more)
		assertCorruptDemoPacket(t, err)
	})
}

func assertCorruptDemoPacket(t *testing.T, err error) {
	t.Helper()

	assert.ErrorIs(t, err, ErrDecodeFailed)
	assert.NotErrorIs(t, err, ErrUnexpectedEndOfDemo)
	assert.NotErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.NotErrorIs(t, err, io.EOF)
	assert.Contains(t, err.Error(), "unexpected end of data")

	var de *DecodeError
	if assert.ErrorAs(t, err, &de) {
		assert.Equal(t, 1, de.Frame)
	}
}

// An error of a frame that's still queued when the demo turns out to be truncated takes precedence.
func TestPrecedence_QueuedDecodeErrorOverTruncation(t *testing.T) {
	p := NewParser(bytes.NewReader(handmadeDemo(demoPacketFrame(t, nil))))
	defer p.Close()

	// Called on the message-handling goroutine for the first frame,
	// it blocks until the parsing goroutine recorded the end of the demo.
	registerInternal(p, func(*msg.CDemoPacket) {
		deadline := time.Now().Add(10 * time.Second)

		for errorOf(&p.abort) == nil && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		panic("corrupt")
	})

	err := p.ParseToEnd(t.Context())
	assert.ErrorIs(t, err, ErrDecodeFailed)
	assert.NotErrorIs(t, err, ErrUnexpectedEndOfDemo)
	assert.NotErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.EqualError(t, errors.Unwrap(err), "corrupt")

	var de *DecodeError
	if assert.ErrorAs(t, err, &de) {
		assert.Zero(t, de.Frame)
	}
}

// A cancellation of ctx while the queue is drained after a decoding failure doesn't replace the error.
// The error has the number of the frame that failed, also though the frames before it weren't all handled.
func TestPrecedence_DecodeErrorOverLaterCancel(t *testing.T) {
	empty := demoPacketFrame(t, nil)

	p := NewParser(bytes.NewReader(handmadeDemo(empty, empty, empty, demoPacketFrame(t, corruptPacketData))))
	defer p.Close()

	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	errStop := errors.New("stop")

	// Called on the message-handling goroutine for the first frame,
	// it blocks until the parsing goroutine recorded the error of the last frame and is draining the queue.
	p.RegisterEventHandler(func(events.FrameDone) {
		deadline := time.Now().Add(10 * time.Second)

		for errorOf(&p.abort) == nil && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		cancel(errStop)
	})

	err := p.ParseToEnd(ctx)
	assert.ErrorIs(t, err, ErrDecodeFailed)
	assert.NotErrorIs(t, err, errStop)

	var de *DecodeError
	if assert.ErrorAs(t, err, &de) {
		assert.Equal(t, 3, de.Frame)
	}

	assert.Less(t, p.CurrentFrame(), 3, "the drain must have been cut short")
}
