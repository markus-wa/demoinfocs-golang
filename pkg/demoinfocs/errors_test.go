package demoinfocs_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// Handlers must not use require / t.FailNow(): runtime.Goexit() on the message-handling goroutine deadlocks the parser.

var errUserBoom = errors.New("user boom")

func skipWithoutDemos(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}
}

func readS2Dem(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(s2DemPath)
	require.NoError(t, err)

	return data
}

// recoverPanicError calls f and returns the *PanicError it panicked with.
func recoverPanicError(t *testing.T, f func()) (pe *demoinfocs.PanicError) {
	t.Helper()

	defer func() {
		r := recover()

		var ok bool

		pe, ok = r.(*demoinfocs.PanicError)
		require.True(t, ok, "expected a *PanicError panic, got %T: %v", r, r)
	}()

	f()

	return nil
}

// assertHandlerPanic parses with parse (which must panic), checks the *PanicError
// and that the parser can still be closed and doesn't parse anymore.
func assertHandlerPanic(t *testing.T, p *demoinfocs.Parser, parse func(), value any, handlerName string) {
	t.Helper()

	pe := recoverPanicError(t, parse)

	assert.Equal(t, value, pe.Value)
	assert.Contains(t, string(pe.Stack), handlerName, "the stack trace must contain the handler")

	if err, ok := value.(error); ok {
		assert.ErrorIs(t, pe, err)
	}

	assert.Contains(t, pe.Error(), handlerName)

	assert.NoError(t, p.Close())

	// only ErrParserStopped, not the panic value
	err := p.ParseToEnd(t.Context())
	assert.Same(t, demoinfocs.ErrParserStopped, err, "ParseToEnd() after a panic must return ErrParserStopped")

	more, err := p.ParseNextFrame()
	assert.False(t, more)
	assert.Same(t, demoinfocs.ErrParserStopped, err, "ParseNextFrame() after a panic must return ErrParserStopped")
}

func newS2Parser(t *testing.T, opts ...option.Option) *demoinfocs.Parser {
	t.Helper()

	return demoinfocs.NewParser(bytes.NewReader(readS2Dem(t)), opts...)
}

//go:noinline
func panickingEventHandler(events.FrameDone) {
	panic(errUserBoom)
}

func TestHandlerPanic_EventHandler(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	p := newS2Parser(t)
	p.RegisterEventHandler(panickingEventHandler)

	assertHandlerPanic(t, p, func() { _ = p.ParseToEnd(t.Context()) }, errUserBoom, "panickingEventHandler")
}

func TestHandlerPanic_EventHandler_ParseNextFrame(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	p := newS2Parser(t)
	p.RegisterEventHandler(panickingEventHandler)

	assertHandlerPanic(t, p, func() {
		for {
			more, err := p.ParseNextFrame()
			if !more || err != nil {
				t.Errorf("ParseNextFrame() returned (%v, %v) instead of panicking", more, err)

				return
			}
		}
	}, errUserBoom, "panickingEventHandler")
}

//go:noinline
func panickingNetMessageHandler(*msg.CSVCMsg_PacketEntities) {
	panic("net-message boom")
}

func TestHandlerPanic_NetMessageHandler(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	p := newS2Parser(t)
	p.RegisterNetMessageHandler(panickingNetMessageHandler)

	assertHandlerPanic(t, p, func() { _ = p.ParseToEnd(t.Context()) }, "net-message boom", "panickingNetMessageHandler")
}

//go:noinline
func panickingPropertyHandler(st.PropertyValue) {
	panic(errUserBoom)
}

func TestHandlerPanic_PropertyOnUpdate(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	p := newS2Parser(t)
	p.RegisterEventHandler(func(events.DataTablesParsed) {
		p.ServerClasses().FindByName("CCSPlayerPawn").OnEntityCreated(func(e *st.Entity) {
			e.Property("m_iHealth").OnUpdate(panickingPropertyHandler)
		})
	})

	assertHandlerPanic(t, p, func() { _ = p.ParseToEnd(t.Context()) }, errUserBoom, "panickingPropertyHandler")
}

//go:noinline
func panickingEntityCreatedHandler(*st.Entity) {
	panic(42)
}

func TestHandlerPanic_OnEntityCreated(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	// also with IgnorePacketEntitiesPanic: the option only applies to decoding failures
	for _, ignore := range []bool{false, true} {
		p := newS2Parser(t, option.IgnorePacketEntitiesPanic(ignore))
		p.RegisterEventHandler(func(events.DataTablesParsed) {
			p.ServerClasses().FindByName("CCSPlayerPawn").OnEntityCreated(panickingEntityCreatedHandler)
		})

		assertHandlerPanic(t, p, func() { _ = p.ParseToEnd(t.Context()) }, 42, "panickingEntityCreatedHandler")
	}
}

// unknownDemoCommandDemo returns a minimal demo with a single frame of an unknown demo command,
// for which a ParserWarn is dispatched on the parsing goroutine.
func unknownDemoCommandDemo() []byte {
	b := []byte("PBDEMS2\x00")
	b = append(b, make([]byte, 8)...) // skipped
	b = append(b, 50, 0, 0)           // cmd = 50 (unknown), tick = 0, size = 0

	return b
}

//go:noinline
func panickingParserWarnHandler(events.ParserWarn) {
	panic(errUserBoom)
}

// ParserWarn is dispatched synchronously on the parsing goroutine for unknown demo commands.
func TestHandlerPanic_ParserWarnOnParsingGoroutine(t *testing.T) {
	t.Parallel()

	t.Run("ParseToEnd", func(t *testing.T) {
		t.Parallel()

		p := demoinfocs.NewParser(bytes.NewReader(unknownDemoCommandDemo()))
		p.RegisterEventHandler(panickingParserWarnHandler)

		assertHandlerPanic(t, p, func() { _ = p.ParseToEnd(t.Context()) }, errUserBoom, "panickingParserWarnHandler")
	})

	t.Run("ParseNextFrame", func(t *testing.T) {
		t.Parallel()

		p := demoinfocs.NewParser(bytes.NewReader(unknownDemoCommandDemo()))
		p.RegisterEventHandler(panickingParserWarnHandler)

		assertHandlerPanic(t, p, func() { _, _ = p.ParseNextFrame() }, errUserBoom, "panickingParserWarnHandler")
	})

	t.Run("without panic", func(t *testing.T) {
		t.Parallel()

		p := demoinfocs.NewParser(bytes.NewReader(unknownDemoCommandDemo()))
		defer p.Close()

		var warns []events.ParserWarn

		p.RegisterEventHandler(func(w events.ParserWarn) { warns = append(warns, w) })

		err := p.ParseToEnd(t.Context())
		assert.ErrorIs(t, err, demoinfocs.ErrUnexpectedEndOfDemo)

		if assert.Len(t, warns, 1) {
			assert.EqualValues(t, events.WarnTypeUnknownDemoCommandMessageType, warns[0].Type)
		}
	})
}

func TestEmptyStream(t *testing.T) {
	t.Parallel()

	p := demoinfocs.NewParser(bytes.NewReader(nil))
	defer p.Close()

	err := p.ParseToEnd(t.Context())
	assert.ErrorIs(t, err, demoinfocs.ErrUnexpectedEndOfDemo)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Contains(t, err.Error(), "(ErrUnexpectedEndOfDemo)")

	// further calls return the same error
	assert.Equal(t, err, p.ParseToEnd(t.Context()))

	more, err2 := p.ParseNextFrame()
	assert.False(t, more)
	assert.Equal(t, err, err2)
}

// signalingReader closes done once r returned an error (e.g. io.EOF).
type signalingReader struct {
	r    io.Reader
	done chan struct{}
	once sync.Once
}

func newSignalingReader(r io.Reader) *signalingReader {
	return &signalingReader{r: r, done: make(chan struct{})}
}

func (sr *signalingReader) Read(b []byte) (int, error) {
	n, err := sr.r.Read(b)
	if err != nil {
		sr.once.Do(func() { close(sr.done) })
	}

	return n, err
}

// blockUntil blocks until done is closed and gives the parsing goroutine some time to return,
// so the queued messages are only handled afterwards.
func blockUntil(done <-chan struct{}) {
	select {
	case <-done:
		time.Sleep(200 * time.Millisecond)
	case <-time.After(10 * time.Second):
	}
}

// The messages that were read before the demo ended are still handled.
func TestTruncatedDemo(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	data := readS2Dem(t)
	sr := newSignalingReader(bytes.NewReader(data[:len(data)/2]))

	p := demoinfocs.NewParser(sr, option.MsgQueueBufferSize(1<<21))
	defer p.Close()

	var (
		once   sync.Once
		frames int
	)

	p.RegisterEventHandler(func(events.FrameDone) {
		once.Do(func() { blockUntil(sr.done) })

		frames++
	})

	err := p.ParseToEnd(t.Context())
	assert.ErrorIs(t, err, demoinfocs.ErrUnexpectedEndOfDemo)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Greater(t, frames, 1000, "the queued messages must be handled")
	assert.Equal(t, p.CurrentFrame(), frames)
}

func TestReadError(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	errBoom := errors.New("boom")

	newParser := func() *demoinfocs.Parser {
		data := readS2Dem(t)

		return demoinfocs.NewParser(io.MultiReader(bytes.NewReader(data[:len(data)/4]), iotest.ErrReader(errBoom)))
	}

	t.Run("returned", func(t *testing.T) {
		t.Parallel()

		p := newParser()
		defer p.Close()

		err := p.ParseToEnd(t.Context())
		assert.ErrorIs(t, err, errBoom)
		assert.NotErrorIs(t, err, demoinfocs.ErrUnexpectedEndOfDemo)
		assert.NotErrorIs(t, err, demoinfocs.ErrDecodeFailed)
	})

	// A handler panicking with the same error still panics.
	t.Run("handler panic", func(t *testing.T) {
		t.Parallel()

		p := newParser()
		p.RegisterEventHandler(func(events.RoundStart) {
			panic(errBoom)
		})

		pe := recoverPanicError(t, func() { _ = p.ParseToEnd(t.Context()) })
		assert.ErrorIs(t, pe, errBoom)
		assert.NoError(t, p.Close())
	})
}

// A handler panic while the queue is drained after the demo stream failed takes precedence over the stream's error.
func TestPrecedence_PanicDuringDrainAfterReadError(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	data := readS2Dem(t)
	sr := newSignalingReader(io.MultiReader(bytes.NewReader(data[:len(data)/4]), iotest.ErrReader(errors.New("read error"))))

	p := demoinfocs.NewParser(sr, option.MsgQueueBufferSize(1<<21))

	p.RegisterEventHandler(func(events.FrameDone) {
		blockUntil(sr.done)

		panic(errUserBoom)
	})

	pe := recoverPanicError(t, func() { _ = p.ParseToEnd(t.Context()) })
	assert.ErrorIs(t, pe, errUserBoom)
	assert.NoError(t, p.Close())
}

// The cancellation of ctx takes precedence over a decoding failure.
func TestPrecedence_CancelCauseOverDecodeError(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	p := newS2Parser(t)
	defer p.Close()

	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)

	errStop := errors.New("stop")

	p.RegisterEventHandler(func(events.RoundStart) {
		cancel(errStop)
	})
	p.RegisterEventHandler(func(events.RoundStart) {
		lookUpMissingProperty(p)
	})

	err := p.ParseToEnd(ctx)
	assert.ErrorIs(t, err, errStop)
	assert.NotErrorIs(t, err, demoinfocs.ErrDecodeFailed)
}

// lookUpMissingProperty calls PropertyValueMust() for a property that doesn't exist.
func lookUpMissingProperty(p *demoinfocs.Parser) {
	for _, e := range p.GameState().Entities() {
		e.PropertyValueMust("m_doesNotExist")
	}
}

// A missing property is a decoding failure (schema drift), also when the user's handler looks it up.
func TestPropertyNotFoundInHandler(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	assertPropertyNotFound := func(t *testing.T, err error) {
		t.Helper()

		assert.ErrorIs(t, err, demoinfocs.ErrDecodeFailed)

		var pnf *st.PropertyNotFoundError
		if assert.ErrorAs(t, err, &pnf) {
			assert.Equal(t, "m_doesNotExist", pnf.Name)
		}

		// the stack of the panic in the handler is kept
		var de *demoinfocs.DecodeError
		if assert.ErrorAs(t, err, &de) {
			assert.Positive(t, de.Frame)
			assert.Contains(t, string(de.Stack), "TestPropertyNotFoundInHandler.func")
			assert.Contains(t, string(de.Stack), "PropertyValueMust")
		}
	}

	t.Run("event handler", func(t *testing.T) {
		t.Parallel()

		p := newS2Parser(t)
		defer p.Close()

		p.RegisterEventHandler(func(events.RoundStart) {
			lookUpMissingProperty(p)
		})

		assertPropertyNotFound(t, p.ParseToEnd(t.Context()))
	})

	// Also with IgnorePacketEntitiesPanic: the option only skips decoding failures of the parser itself,
	// a missing property in user code stops parsing.
	for _, ignore := range []bool{false, true} {
		t.Run(fmt.Sprintf("entity handler, IgnorePacketEntitiesPanic=%v", ignore), func(t *testing.T) {
			t.Parallel()

			p := newS2Parser(t, option.IgnorePacketEntitiesPanic(ignore))
			defer p.Close()

			var packetEntitiesWarns atomic.Int64

			p.RegisterEventHandler(func(w events.ParserWarn) {
				if w.Type == events.WarnTypePacketEntitiesPanic {
					packetEntitiesWarns.Add(1)
				}
			})

			p.RegisterEventHandler(func(events.DataTablesParsed) {
				p.ServerClasses().FindByName("CCSPlayerPawn").OnEntityCreated(func(e *st.Entity) {
					e.PropertyValueMust("m_doesNotExist")
				})
			})

			assertPropertyNotFound(t, p.ParseToEnd(t.Context()))
			assert.Zero(t, packetEntitiesWarns.Load())
		})
	}
}

// corruptPacketEntities replaces the entity data of the nth delta CSVCMsg_PacketEntities with garbage.
// Net-message handlers are called in registration order, so this runs before the parser's own handler.
func corruptPacketEntities(p *demoinfocs.Parser, nth int) {
	var n int

	p.RegisterNetMessageHandler(func(m *msg.CSVCMsg_PacketEntities) {
		if !m.GetLegacyIsDelta() {
			return
		}

		n++

		if n == nth {
			m.EntityData = bytes.Repeat([]byte{0xff}, 64)
		}
	})
}

func TestPacketEntitiesDecodeError(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	t.Run("IgnorePacketEntitiesPanic", func(t *testing.T) {
		t.Parallel()

		p := newS2Parser(t, option.IgnorePacketEntitiesPanic(true))
		defer p.Close()

		corruptPacketEntities(p, 1000)

		var (
			warns       []events.ParserWarn
			framesAfter atomic.Int64
		)

		p.RegisterEventHandler(func(w events.ParserWarn) {
			if w.Type == events.WarnTypePacketEntitiesPanic {
				warns = append(warns, w)
			}
		})
		p.RegisterEventHandler(func(events.FrameDone) {
			if len(warns) > 0 {
				framesAfter.Add(1)
			}
		})

		err := p.ParseToEnd(t.Context())
		assert.NoError(t, err)

		if assert.Len(t, warns, 1) {
			assert.ErrorIs(t, warns[0].Err, demoinfocs.ErrDecodeFailed)
			assert.Contains(t, warns[0].Message, "unable to find existing entity")
		}

		assert.Greater(t, framesAfter.Load(), int64(1000), "parsing must continue")
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		p := newS2Parser(t)
		defer p.Close()

		corruptPacketEntities(p, 1000)

		err := p.ParseToEnd(t.Context())
		assert.ErrorIs(t, err, demoinfocs.ErrDecodeFailed)
		assert.Contains(t, err.Error(), "unable to find existing entity")
	})
}

func TestParseToEnd_AfterSuccess(t *testing.T) {
	t.Parallel()
	skipWithoutDemos(t)

	p := newS2Parser(t)
	defer p.Close()

	require.NoError(t, p.ParseToEnd(t.Context()))
	assert.NoError(t, p.ParseToEnd(t.Context()))

	more, err := p.ParseNextFrame()
	assert.False(t, more)
	assert.NoError(t, err)
}
