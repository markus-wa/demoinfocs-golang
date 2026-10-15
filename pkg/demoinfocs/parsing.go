package demoinfocs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/config"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// msgBufPool recycles the byte slices that hold frame and embedded-message
// payloads. proto.Unmarshal always copies bytes fields into the message, so
// the input buffers are dead once the message is unmarshaled and can be
// returned to this pool. See parseFrame() and handleDemoPacket().
var msgBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func getMsgBuf(n int) *[]byte {
	bp := msgBufPool.Get().(*[]byte)
	if cap(*bp) < n {
		*bp = make([]byte, 0, n)
	}

	*bp = (*bp)[:0]

	return bp
}

func putMsgBuf(bp *[]byte) {
	// Large buffers (from rare large payloads) aren't kept, so they don't stay allocated.
	if bp != nil && cap(*bp) <= maxPooledBuffer {
		msgBufPool.Put(bp)
	}
}

const (
	playerWeaponPrefixS2 = "m_pWeaponServices.m_hMyWeapons"
	gameRulesPrefixS2    = "m_pGameRules"
)

const (
	// Filestamps found in the first 8 bytes of demo files, identifying the demo format.
	filestampS1 = "HL2DEMO"
	filestampS2 = "PBDEMS2"

	// GUID assigned to bot players.
	botGUID = "BOT"
)

// parseHeader attempts to parse the header of the demo and stores it in p.header.
// If not done manually this will be called by Parser.ParseNextFrame() or Parser.ParseToEnd().
//
// Returns ErrInvalidFileType if the filestamp (first 8 bytes) doesn't match PBDEMS2.
func (p *Parser) parseHeader() error {
	var h header

	isCSTVBroadcast := p.config.Format == config.FormatCSTVBroadcast

	if isCSTVBroadcast {
		h.Filestamp = filestampS2
	} else {
		h.Filestamp = p.bitReader.ReadCString(8)
	}

	switch h.Filestamp {
	case filestampS1:
		return fmt.Errorf("%w: CS:GO demos are no longer supported, downgrade to v3", ErrInvalidFileType)

	case filestampS2:
		if !isCSTVBroadcast {
			p.bitReader.Skip(8 << 3) // skip 8 bytes
		}

		// Error returns were already discarded by the dispatcher before these
		// wrappers existed; the wrappers only add the aborted() short-circuit.
		registerInternal(p, func(m *msg.CSVCMsg_ServerInfo) {
			if p.aborted() {
				return
			}

			_ = p.stParser.OnServerInfo(m)
		})
		registerInternal(p, p.handlePacketEntities)

	default:
		return ErrInvalidFileType
	}

	// Initialize queue if the buffer size wasn't specified, the amount of ticks
	// seems to be a good indicator of how many events we'll get
	if p.msgQueue == nil {
		p.initMsgQueue(msgQueueSize(h.PlaybackTicks))
	}

	p.header = &h

	return nil
}

func msgQueueSize(ticks int) int {
	const (
		msgQueueMinSize = 50000
		msgQueueMaxSize = 500000
	)

	size := math.Max(msgQueueMinSize, float64(ticks))
	size = math.Min(msgQueueMaxSize, size)

	return int(size)
}

// ParseToEnd attempts to parse the demo until the end.
//
// Parsing is aborted when ctx is done (it may be cancelled from a handler or another goroutine):
// no further events are dispatched, all handlers are unregistered, net-messages that are already
// queued are dropped and context.Cause(ctx) is returned, so errors.Is(err, context.Canceled)
// and errors.Is(err, context.DeadlineExceeded) can be used to detect it.
// If ctx is already done, ParseToEnd returns right away without parsing anything.
// To stop parsing from a handler, cancel ctx (e.g. via context.WithCancelCause()) - don't panic.
//
// Returned errors:
//   - ErrUnexpectedEndOfDemo (also matches io.ErrUnexpectedEOF) if the demo is truncated.
//     All net-messages read up to that point are still handled.
//   - a *DecodeError (matches ErrDecodeFailed) if some data couldn't be decoded,
//     because it's corrupt or uses a schema / format this version doesn't support.
//     Sizes read from the demo are limited (e.g. frames to 16 MiB, far above what real demos use),
//     larger ones are a DecodeError too.
//     Broken DEM_FullPacket frames are skipped with a ParserWarn instead, see events.WarnTypeBrokenFullPacket.
//   - ErrInvalidFileType if the stream doesn't contain a CS2 demo.
//   - errors returned by the demo stream (wrapped).
//   - context.Cause(ctx), see above.
//
// If several of these happen, the one of the earliest frame of the demo is returned: e.g. a decoding failure
// of a frame that was read before the demo turned out to be truncated takes precedence over ErrUnexpectedEndOfDemo.
// Between a cancellation and an error, the one that happened first is returned (e.g. a cancellation after the
// demo turned out to be truncated doesn't replace ErrUnexpectedEndOfDemo, and vice versa).
// The events dispatched before an error are valid.
//
// If a handler panics, parsing stops and ParseToEnd panics with a *PanicError on the calling goroutine,
// which contains the original panic value and stack trace. This takes precedence over any error.
// Close() can still be called afterwards.
//
// ParseToEnd can't be resumed after it returned (or panicked):
// further calls (to ParseToEnd() or ParseNextFrame()) return the same error, nil after success,
// or ErrParserStopped after a handler panic.
func (p *Parser) ParseToEnd(ctx context.Context) (err error) {
	if p.done {
		return p.doneErr
	}

	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	stopWatchingContext := p.watchContext(ctx)

	defer func() {
		err = p.finishParsing(recover(), err, true, stopWatchingContext)
		if err == nil {
			p.ensurePlaybackValuesAreSet()
		}
	}()

	p.ensureOpen()

	if p.header == nil {
		err = p.parseHeader()
		if err != nil {
			return err
		}
	}

	// Errors of the message-handling goroutine and the cancellation of ctx are collected by finishParsing().
	for {
		more, err := p.parseFrame()
		if err != nil {
			return err
		}

		if !more || p.aborted() {
			return nil
		}
	}
}

// watchContext aborts parsing via abortState.setCancelled() once ctx is done, which also stops
// all handlers and drains the queued backlog without dispatching it.
// Until setCancelled() has run (context.AfterFunc() runs it on a separate goroutine),
// aborted() checks ctx directly so cancelling from within a handler takes effect immediately.
//
// The returned function must be called before the message queues are removed.
// It waits for a running abort to finish so no handler is called after ParseToEnd() returns.
func (p *Parser) watchContext(ctx context.Context) (stop func()) {
	if ctx.Done() == nil {
		// ctx can never be cancelled (e.g. context.Background())
		return func() {}
	}

	p.abort.ctx = ctx

	aborted := make(chan struct{})

	stopAfterFunc := context.AfterFunc(ctx, func() {
		p.abort.setCancelled(context.Cause(ctx))
		close(aborted)
	})

	return func() {
		if !stopAfterFunc() {
			<-aborted
		} else if ctx.Err() != nil {
			// ctx was cancelled but the AfterFunc hasn't been started yet
			p.abort.setCancelled(context.Cause(ctx))
		}
	}
}

// finishParsing ends a ParseToEnd() / ParseNextFrame() call, it must be deferred by them.
//
// r is the value recovered from the parsing goroutine (nil if it didn't panic), readerErr the error it returned.
// If last is set (or there's an error), the parser is done: the message queue is closed and further
// ParseToEnd() / ParseNextFrame() calls return the resulting error.
//
// A handler panic is re-panicked as *PanicError, otherwise the error that takes precedence is returned,
// see abortState.
//
// stopWatchingContext is the function returned by watchContext().
func (p *Parser) finishParsing(r any, readerErr error, last bool, stopWatchingContext func()) error {
	if r != nil {
		readerErr = p.readerFailure(r)
	}

	// Recorded before the queue is drained (so a cancellation of ctx during the drain doesn't replace it),
	// but without stopping the handlers: all the messages that were read are still handled,
	// also if the demo is truncated or corrupt. Their errors take precedence, see abortState.
	p.abort.setReaderError(readerErr)

	// After a handler panic the handlers are unregistered already (see abortState.setPanic()),
	// so this just empties the queue.
	p.msgDispatcher.SyncAllQueues()

	// Only stop watching ctx after the queue is drained: handlers skip messages once ctx is done
	// (see aborted()), so a cancellation during the drain must still be recorded.
	stopWatchingContext()

	pe, err := p.abort.result()
	if pe != nil {
		p.markDone(ErrParserStopped)

		panic(pe)
	}

	if err == nil {
		if last {
			p.markDone(nil)
		}

		return nil
	}

	// stops the handlers, if the error didn't already (errors of the parsing goroutine don't)
	p.abort.stop()
	p.markDone(err)

	return err
}

// readerFailure converts the panic r of the parsing goroutine into an error.
// A handler panic (e.g. of a ParserWarn handler, called on the parsing goroutine) is recorded instead, nil is returned.
func (p *Parser) readerFailure(r any) error {
	switch classifyPanic(r) {
	case panics.KindUser:
		p.abort.setPanic(panics.Capture(r))

		return nil

	case panics.KindEndOfDemo:
		// Only the main demo stream panics with a bare io.EOF / io.ErrUnexpectedEOF here,
		// nested readers have their own boundaries (see handleDemoPacket()).
		return errUnexpectedEndOfDemo

	case panics.KindRead:
		return fmt.Errorf("reading demo: %w", r.(*panics.ReadError).Err) //nolint:forcetypeassert // see Classify()

	case panics.KindDecode:
	}

	return newDecodeError(panics.NewFailure(r), p.readerFrame)
}

// markDone marks the parser as done, further ParseToEnd() / ParseNextFrame() calls return err.
func (p *Parser) markDone(err error) {
	p.closeMsgQueue()

	p.done = true
	p.doneErr = err
}

func (p *Parser) closeMsgQueue() {
	p.msgDispatcher.RemoveAllQueues()

	if p.msgQueue != nil {
		close(p.msgQueue)
	}
}

/*
ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).

Returns true unless the demo command 'stop' or an error was encountered.
The returned errors are the same as for ParseToEnd() (except for the cancellation):
e.g. ErrUnexpectedEndOfDemo for truncated demos and a *DecodeError if some data couldn't be decoded.

If a handler panics, ParseNextFrame panics with a *PanicError, see ParseToEnd().

Once ParseNextFrame returned false (or panicked), further calls return the same error, nil after the end of the demo,
or ErrParserStopped after a handler panic.

ParseNextFrame doesn't take a context.Context: callers that need cancellation
should check their context between frames and stop calling ParseNextFrame once it's done.

See also: ParseToEnd() for parsing the complete demo in one go (faster).
*/
func (p *Parser) ParseNextFrame() (moreFrames bool, err error) {
	if p.done {
		return false, p.doneErr
	}

	defer func() {
		r := recover()
		if r != nil {
			moreFrames = false
		}

		err = p.finishParsing(r, err, !moreFrames, func() {})
		if err != nil {
			moreFrames = false
		}
	}()

	p.ensureOpen()

	if p.header == nil {
		err = p.parseHeader()
		if err != nil {
			return false, err
		}
	}

	return p.parseFrame()
}

var demoCommandMsgsCreators = map[msg.EDemoCommands]config.NetMessageCreator{
	msg.EDemoCommands_DEM_Stop:            func() proto.Message { return &msg.CDemoStop{} },
	msg.EDemoCommands_DEM_FileHeader:      func() proto.Message { return &msg.CDemoFileHeader{} },
	msg.EDemoCommands_DEM_FileInfo:        func() proto.Message { return &msg.CDemoFileInfo{} },
	msg.EDemoCommands_DEM_SyncTick:        func() proto.Message { return &msg.CDemoSyncTick{} },
	msg.EDemoCommands_DEM_SendTables:      func() proto.Message { return &msg.CDemoSendTables{} },
	msg.EDemoCommands_DEM_ClassInfo:       func() proto.Message { return &msg.CDemoClassInfo{} },
	msg.EDemoCommands_DEM_StringTables:    func() proto.Message { return &msg.CDemoStringTables{} },
	msg.EDemoCommands_DEM_Packet:          func() proto.Message { return &msg.CDemoPacket{} },
	msg.EDemoCommands_DEM_SignonPacket:    func() proto.Message { return &msg.CDemoPacket{} },
	msg.EDemoCommands_DEM_ConsoleCmd:      func() proto.Message { return &msg.CDemoConsoleCmd{} },
	msg.EDemoCommands_DEM_CustomData:      func() proto.Message { return &msg.CDemoCustomData{} },
	msg.EDemoCommands_DEM_UserCmd:         func() proto.Message { return &msg.CDemoUserCmd{} },
	msg.EDemoCommands_DEM_FullPacket:      func() proto.Message { return &msg.CDemoFullPacket{} },
	msg.EDemoCommands_DEM_SaveGame:        func() proto.Message { return &msg.CDemoSaveGame{} },
	msg.EDemoCommands_DEM_SpawnGroups:     func() proto.Message { return &msg.CDemoSpawnGroups{} },
	msg.EDemoCommands_DEM_AnimationData:   func() proto.Message { return &msg.CDemoAnimationData{} },
	msg.EDemoCommands_DEM_AnimationHeader: func() proto.Message { return &msg.CDemoAnimationHeader{} },
	msg.EDemoCommands_DEM_Recovery:        func() proto.Message { return &msg.CDemoRecovery{} },
}

// readFramePayload reads (and, if needed, decompresses) the current frame's payload of the given size.
//
// Regular demos read into a pooled buffer and reuse the per-parser snappy
// scratch as the decompression destination; the pool entry is returned so the
// caller can put it back after proto.Unmarshal (which copies everything it
// needs). bp is nil whenever buf is not pool-backed.
//
// CSTV broadcasts retain buf in messages (m.Data = buf), so they get a plain,
// unpooled read with a fresh snappy destination.
//
// A payload that's larger than the limit for msgType (see framePayloadLimit()) isn't read, errFrameTooLarge is returned.
// A compressed payload that can't be decompressed is consumed from the demo stream, an error is returned
// (the caller decides what to do, see parseFrame()).
// If the demo ends within the payload, it panics with io.ErrUnexpectedEOF (truncated demo).
func (p *Parser) readFramePayload(msgType msg.EDemoCommands, size uint32, msgCompressed, isCSTVBroadcast bool) (buf []byte, bp *[]byte, err error) {
	limit := framePayloadLimit(msgType)
	if uint64(size) > limit {
		return nil, nil, fmt.Errorf("%w: %v payload of %d bytes exceeds the limit of %d bytes", errFrameTooLarge, msgType, size, limit)
	}

	n := int(size)

	if isCSTVBroadcast {
		buf = make([]byte, 0, min(n, maxPreallocatedPayload))
		p.readPayloadInto(&buf, n)

		if msgCompressed {
			buf, err = decodeSnappy(nil, buf, limit)
			if err != nil {
				return nil, nil, err
			}
		}

		return buf, nil, nil
	}

	bp = getMsgBuf(min(n, maxPreallocatedPayload))
	p.readPayloadInto(bp, n)
	buf = *bp

	if !msgCompressed {
		return buf, bp, nil
	}

	// Reuse the per-parser scratch as the snappy destination: the decoded
	// payload is dead after the caller's proto.Unmarshal, and frame parsing
	// is single-threaded.
	buf, err = decodeSnappy(p.snappyScratch, buf, limit)

	// The pooled input is consumed either way; only the decoded scratch
	// carries data from here on.
	putMsgBuf(bp)

	if err != nil {
		return nil, nil, err
	}

	// Large scratches (from rare large payloads) aren't kept, so they don't stay allocated.
	if cap(buf) <= maxPooledBuffer {
		p.snappyScratch = buf[:0]
	} else {
		p.snappyScratch = nil
	}

	return buf, nil, nil
}

// readPayloadInto appends n bytes of the demo stream to buf.
// It panics with io.ErrUnexpectedEOF if the stream ends before (the demo is truncated):
// the bit reader returns the bytes up to the end of the stream without an error.
func (p *Parser) readPayloadInto(buf *[]byte, n int) {
	start := len(*buf)

	p.bitReader.ReadBytesInto(buf, n)

	if len(*buf)-start < n {
		panic(io.ErrUnexpectedEOF)
	}
}

// skipPayload skips the current frame's payload of the given size.
func (p *Parser) skipPayload(size uint32) {
	// in steps, so the number of bits doesn't overflow int on 32-bit platforms
	const maxStep = 1 << 28 // bytes

	for remaining := uint64(size); remaining > 0; {
		step := min(remaining, maxStep)

		p.bitReader.Skip(int(step) << 3)

		remaining -= step
	}
}

// brokenFullPacket is queued instead of a DEM_FullPacket frame that couldn't be decoded, see parseFrame().
// The ParserWarn is dispatched when it's handled (on the message-handling goroutine), in order with the other events.
type brokenFullPacket struct {
	err *DecodeError
}

func (p *Parser) handleBrokenFullPacket(b *brokenFullPacket) {
	if p.aborted() {
		return
	}

	p.eventDispatcher.Dispatch(events.ParserWarn{
		Message: fmt.Sprintf("skipped broken DEM_FullPacket: %v", b.err.Cause),
		Type:    events.WarnTypeBrokenFullPacket,
		Err:     b.err,
	})
}

// queueFrameParsed queues the end of the current frame. readerFrame counts the frames like currentFrame
// (see handleFrameParsed()), but on the parsing goroutine.
func (p *Parser) queueFrameParsed() {
	p.msgQueue <- frameParsedToken

	p.readerFrame++
}

// parseFrame reads the next frame and queues its messages.
// It returns false after the last frame, failures to decode the frame are returned as *DecodeError.
//
//nolint:funlen
func (p *Parser) parseFrame() (bool, error) {
	cmd := msg.EDemoCommands(p.bitReader.ReadVarInt32())

	msgType := cmd & ^msg.EDemoCommands_DEM_IsCompressed
	msgCompressed := (cmd & msg.EDemoCommands_DEM_IsCompressed) != 0

	var (
		tick uint32
		size uint32
	)

	isCSTVBroadcast := p.config.Format == config.FormatCSTVBroadcast

	if isCSTVBroadcast {
		tick = uint32(p.bitReader.ReadInt(32))

		p.bitReader.Skip(8)

		if cmd == msg.EDemoCommands_DEM_Stop {
			p.msgQueue <- ingameTickNumber(int32(tick))

			p.queueFrameParsed()

			return false, nil
		}

		size = uint32(p.bitReader.ReadInt(32))
	} else {
		tick = p.bitReader.ReadVarInt32()

		// This appears to actually be an int32, where a -1 means pre-game.
		if tick == 4294967295 {
			tick = 0
		}

		if msgType == msg.EDemoCommands_DEM_Stop {
			p.msgQueue <- ingameTickNumber(int32(tick))

			p.queueFrameParsed()

			return false, nil
		}

		size = p.bitReader.ReadVarInt32()
	}

	p.msgQueue <- ingameTickNumber(int32(tick))

	msgCreator := demoCommandMsgsCreators[msgType]
	if msgCreator == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("skipping unknown demo commands message type with value %d", msgType),
			Type:    events.WarnTypeUnknownDemoCommandMessageType,
		})
		p.skipPayload(size)

		return true, nil
	}

	m, nested, err := p.decodeFrame(msgCreator(), size, msgType, msgCompressed, isCSTVBroadcast)
	if err != nil {
		// Some Valve matchmaking demos (~10% in 2023) contain a broken DEM_FullPacket, which the game skips as well (#409).
		// Full packets are snapshots of the state that's also sent with the regular packets, so they can be skipped.
		// The frame is still counted (see queueFrameParsed()), so CurrentFrame() stays the same for all later frames.
		// A frame that's too large wasn't read, parsing can't continue after it.
		if msgType != msg.EDemoCommands_DEM_FullPacket || errors.Is(err, errFrameTooLarge) {
			return false, newDecodeError(err, p.readerFrame)
		}

		p.msgQueue <- &brokenFullPacket{err: newDecodeError(err, p.readerFrame)}

		p.queueFrameParsed()

		return true, nil
	}

	p.msgQueue <- m

	if m, ok := m.(*msg.CDemoFullPacket); ok {
		p.msgQueue <- m.StringTable
	}

	for _, nm := range nested {
		p.msgQueue <- nm
	}

	// Drop the references, the slice is reused for the next packet.
	clear(nested)

	// Queue up some post processing
	p.queueFrameParsed()

	return msgType != msg.EDemoCommands_DEM_Stop, nil
}

// decodeFrame reads and decodes the current frame's payload into m.
// For (full) packets, the embedded net-messages are decoded as well and returned as nested,
// so nothing is queued for a frame that can't be decoded completely.
// nested is only valid until the next call.
func (p *Parser) decodeFrame(
	m proto.Message,
	size uint32,
	msgType msg.EDemoCommands,
	msgCompressed, isCSTVBroadcast bool,
) (_ proto.Message, nested []proto.Message, err error) {
	buf, bp, err := p.readFramePayload(msgType, size, msgCompressed, isCSTVBroadcast)
	if err != nil {
		return nil, nil, err
	}

	if isCSTVBroadcast {
		switch m := m.(type) {
		case *msg.CDemoPacket:
			m.Data = buf

		case *msg.CDemoSpawnGroups:
			if len(buf) == 0 {
				return nil, nil, errors.New("empty DEM_SpawnGroups payload") //nolint:err113 // only used as cause of a DecodeError
			}

			m.Msgs = [][]byte{buf[1:]} // TODO: index might be a varint, also we should collect all entries into one msg

		default:
			err = proto.Unmarshal(buf, m)
		}
	} else {
		err = proto.Unmarshal(buf, m)

		putMsgBuf(bp)
	}

	if err != nil {
		return nil, nil, err
	}

	switch m := m.(type) {
	case *msg.CDemoPacket:
		nested, err = p.decodeDemoPacket(m, false)

	case *msg.CDemoFullPacket:
		if m.Packet.GetData() != nil {
			nested, err = p.decodeDemoPacket(m.Packet, true)
		}
	}

	if err != nil {
		return nil, nil, err
	}

	return m, nested, nil
}

func (p *Parser) handlePacketEntities(m *msg.CSVCMsg_PacketEntities) {
	if p.aborted() {
		return
	}

	// Used by Player.Velocity(). Also set for snapshots OnPacketEntities() ignores (e.g. the ones of
	// DEM_FullPacket, which repeat the last server tick) - position recording skips repeated ticks.
	if m.ServerTick != nil {
		p.snapshotTick = int(m.GetServerTick())
	}

	p.processingSnapshot = true

	err := p.stParser.OnPacketEntities(m)
	if err != nil {
		de := newDecodeError(err, p.currentFrame)

		if !p.ignorePacketEntitiesPanic {
			p.processingSnapshot = false

			p.setError(de)

			return
		}

		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("encountered PacketEntities panic: %v", err),
			Type:    events.WarnTypePacketEntitiesPanic,
			Err:     de,
		})
	}

	// backstop for controller identity & connection changes not applied per entity yet
	p.flushPlayerControllerUpdates()

	p.processingSnapshot = false
}

type frameParsedTokenType struct{}

var frameParsedToken = new(frameParsedTokenType)

func (p *Parser) handleFrameParsed(*frameParsedTokenType) {
	if p.aborted() {
		return
	}

	p.processFrameGameEvents()

	// Cache grenade weapons for the case where CS2 removes a grenade from the thrower's
	// inventory before the projectile entity is fully created (#580).
	// Iterates the underlying map directly: no snapshot allocation or sorting
	// is needed because updateLastKnownGrenadeWeapons only reads the player
	// and mutates parser-owned cache state, and map order doesn't matter here.
	for _, pl := range p.gameState.participantsRaw() {
		p.gameState.updateLastKnownGrenadeWeapons(pl)
	}

	p.currentFrame++
	p.eventDispatcher.Dispatch(events.FrameDone{})

	// Record positions after FrameDone so that handlers see the velocity relative to the previous frame.
	// Used by Player.Velocity() when m_vecVelocity isn't networked (demos before the AnimGraph 2 update).
	// Players only get registered once Velocity() is called for them (or with option.EagerVelocityTracking(true)).
	// Recorders of players whose controller is gone return false and are dropped.
	p.framePositionRecorders = slices.DeleteFunc(p.framePositionRecorders, func(record func() bool) bool {
		return !record()
	})
}

// CS2 demos playback info are available in the CDemoFileInfo message that should be parsed at the end of the demo.
// Demos may not contain it, as a workaround we update values with the last parser information at the end of parsing.
func (p *Parser) ensurePlaybackValuesAreSet() {
	if p.header.PlaybackTicks == 0 {
		p.header.PlaybackTicks = p.gameState.ingameTick
		p.header.PlaybackFrames = p.currentFrame
		p.header.PlaybackTime = time.Duration(float32(p.header.PlaybackTicks)*float32(p.tickInterval)) * time.Second
	}
}
