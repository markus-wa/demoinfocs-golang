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

	"github.com/golang/snappy"
	dispatch "github.com/markus-wa/godispatch"
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
	if bp != nil {
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

// Parsing errors
var (
	// ErrUnexpectedEndOfDemo signals that the demo is incomplete / corrupt -
	// these demos may still be useful, check how far the parser got.
	ErrUnexpectedEndOfDemo = errors.New("demo stream ended unexpectedly (ErrUnexpectedEndOfDemo)")

	// ErrInvalidFileType signals that the input isn't a valid CS2 demo.
	ErrInvalidFileType = errors.New("invalid File-Type; expecting PBDEMS2 in the first 8 bytes (ErrInvalidFileType)")
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
//
// See also: ParseNextFrame() for other possible errors.
func (p *Parser) ParseToEnd(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	stopWatchingContext := p.watchContext(ctx)

	defer func() {
		// Make sure all the messages of the demo are handled
		p.msgDispatcher.SyncAllQueues()

		// Only stop watching ctx after the queues are drained: handlers skip messages once ctx is done
		// (see aborted()), so a cancellation during the drain must still be returned.
		stopWatchingContext()

		p.msgDispatcher.RemoveAllQueues()

		// Close msgQueue
		if p.msgQueue != nil {
			close(p.msgQueue)
		}

		if err == nil {
			err = recoverFromUnexpectedEOF(recover())
		}

		// any errors that happened during SyncAllQueues()
		if err == nil {
			err = p.error()
		}

		if err == nil {
			p.ensurePlaybackValuesAreSet()
		}
	}()

	if p.header == nil {
		err = p.parseHeader()
		if err != nil {
			return err
		}
	}

	for {
		if !p.parseFrame() {
			return p.error()
		}

		if err = p.error(); err != nil {
			return err
		}

		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
	}
}

// watchContext aborts parsing via setError() once ctx is done, which also stops
// all handlers and drains the queued backlog without dispatching it.
// Until setError() has run (context.AfterFunc() runs it on a separate goroutine),
// aborted() checks ctx directly so cancelling from within a handler takes effect immediately.
//
// The returned function must be called before the message queues are synced/removed.
// It waits for a running abort to finish so no handler is called after ParseToEnd() returns.
func (p *Parser) watchContext(ctx context.Context) (stop func()) {
	if ctx.Done() == nil {
		// ctx can never be cancelled (e.g. context.Background())
		return func() {}
	}

	p.ctx = ctx

	aborted := make(chan struct{})

	stopAfterFunc := context.AfterFunc(ctx, func() {
		p.setError(context.Cause(ctx))
		close(aborted)
	})

	return func() {
		if !stopAfterFunc() {
			<-aborted
		} else if ctx.Err() != nil {
			// ctx was cancelled but the AfterFunc hasn't been started yet
			p.setError(context.Cause(ctx))
		}
	}
}

func recoverFromUnexpectedEOF(r any) error {
	if r == nil {
		return nil
	}

	if r == io.ErrUnexpectedEOF || r == io.EOF {
		return fmt.Errorf("unexpected EOF: %w", ErrUnexpectedEndOfDemo)
	}

	switch err := r.(type) {
	case dispatch.ConsumerCodePanic:
		panic(err.Value())

	case *panics.Captured:
		panic(err.Value)

	default:
		panic(err)
	}
}

/*
ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).

Returns true unless the demo command 'stop' or an error was encountered.

May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
May panic if the demo is corrupt in some way.

ParseNextFrame doesn't take a context.Context: callers that need cancellation
should check their context between frames and stop calling ParseNextFrame once it's done.

See also: ParseToEnd() for parsing the complete demo in one go (faster).
*/
func (p *Parser) ParseNextFrame() (moreFrames bool, err error) {
	defer func() {
		// Make sure all the messages of the frame are handled
		p.msgDispatcher.SyncAllQueues()

		// Close msgQueue (only if we are done)
		if p.msgQueue != nil && !moreFrames {
			p.msgDispatcher.RemoveAllQueues()
			close(p.msgQueue)
		}

		if err == nil {
			err = recoverFromUnexpectedEOF(recover())
		}
	}()

	if p.header == nil {
		err = p.parseHeader()
		if err != nil {
			return
		}
	}

	moreFrames = p.parseFrame()

	return moreFrames, p.error()
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

// readFramePayload reads (and, if needed, decompresses) the current frame's payload.
//
// Regular demos read into a pooled buffer and reuse the per-parser snappy
// scratch as the decompression destination; the pool entry is returned so the
// caller can put it back after proto.Unmarshal (which copies everything it
// needs). bp is nil whenever buf is not pool-backed.
//
// CSTV broadcasts retain buf in messages (m.Data = buf), so they get a plain,
// unpooled read with a fresh snappy destination.
func (p *Parser) readFramePayload(size int, msgCompressed, isCSTVBroadcast bool) (buf []byte, bp *[]byte) {
	if isCSTVBroadcast {
		buf = p.bitReader.ReadBytes(size)

		if msgCompressed {
			var err error

			buf, err = snappy.Decode(nil, buf)
			if err != nil {
				p.handleSnappyError(err)
			}
		}

		return buf, nil
	}

	bp = getMsgBuf(size)
	p.bitReader.ReadBytesInto(bp, size)
	buf = *bp

	if !msgCompressed {
		return buf, bp
	}

	// Reuse the per-parser scratch as the snappy destination: the decoded
	// payload is dead after the caller's proto.Unmarshal, and frame parsing
	// is single-threaded.
	// snappy.Decode only keeps dst when len(dst) >= decoded length, so grow
	// the scratch's length (capacity permitting) instead of passing a
	// zero-length slice — otherwise every call allocates a fresh buffer.
	var err error

	if dLen, _ := snappy.DecodedLen(buf); dLen <= cap(p.snappyScratch) {
		p.snappyScratch = p.snappyScratch[:dLen]
	}

	buf, err = snappy.Decode(p.snappyScratch, buf)

	// The pooled input is consumed either way; only the decoded scratch
	// carries data from here on.
	putMsgBuf(bp)

	if err != nil {
		buf = nil

		p.handleSnappyError(err)

		return buf, nil
	}

	p.snappyScratch = buf[:0]

	return buf, nil
}

func (p *Parser) handleSnappyError(err error) {
	if errors.Is(err, snappy.ErrCorrupt) {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: "compressed message is corrupt",
		})

		return
	}

	panic(err)
}

//nolint:funlen
func (p *Parser) parseFrame() bool {
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

			p.msgQueue <- frameParsedToken

			return false
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

			p.msgQueue <- frameParsedToken

			return false
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
		p.bitReader.Skip(int(size) << 3)

		return true
	}

	buf, bp := p.readFramePayload(int(size), msgCompressed, isCSTVBroadcast)

	m := msgCreator()

	if m == nil {
		panic(fmt.Sprintf("Unknown demo command: %d", msgType))
	}

	if isCSTVBroadcast {
		switch m := m.(type) {
		case *msg.CDemoPacket:
			m.Data = buf

		case *msg.CDemoSpawnGroups:
			m.Msgs = [][]byte{buf[1:]} // TODO: index might be a varint, also we should collect all entries into one msg

		default:
			err := proto.Unmarshal(buf, m)
			if err != nil {
				panic(err) // FIXME: avoid panic
			}
		}
	} else {
		err := proto.Unmarshal(buf, m)
		if err != nil {
			panic(err) // FIXME: avoid panic
		}

		putMsgBuf(bp)
	}

	p.msgQueue <- m

	switch m := m.(type) {
	case *msg.CDemoPacket:
		p.handleDemoPacket(m, false)

	case *msg.CDemoFullPacket:
		p.msgQueue <- m.StringTable

		if m.Packet.GetData() != nil {
			p.handleDemoPacket(m.Packet, true)
		}
	}

	// Queue up some post processing
	p.msgQueue <- frameParsedToken

	return msgType != msg.EDemoCommands_DEM_Stop
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

	_ = p.stParser.OnPacketEntities(m)

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
