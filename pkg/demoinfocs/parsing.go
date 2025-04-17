package demoinfocs

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/golang/snappy"
	dispatch "github.com/markus-wa/godispatch"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables/sendtablescs2"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

const (
	playerWeaponPrefix    = "m_hMyWeapons."
	playerWeaponPrefixS2  = "m_pWeaponServices.m_hMyWeapons."
	playerWeaponPrePrefix = "bcc_nonlocaldata."
	gameRulesPrefix       = "cs_gamerules_data"
	gameRulesPrefixS2     = "m_pGameRules"
)

// Parsing errors
var (
	// ErrCancelled signals that parsing was cancelled via Parser.Cancel()
	ErrCancelled = errors.New("parsing was cancelled before it finished (ErrCancelled)")

	// ErrUnexpectedEndOfDemo signals that the demo is incomplete / corrupt -
	// these demos may still be useful, check how far the parser got.
	ErrUnexpectedEndOfDemo = errors.New("demo stream ended unexpectedly (ErrUnexpectedEndOfDemo)")

	// ErrInvalidFileType signals that the input isn't a valid CS:GO demo.
	ErrInvalidFileType = errors.New("invalid File-Type; expecting HL2DEMO in the first 8 bytes (ErrInvalidFileType)")
)

// parseHeader attempts to parse the header of the demo and returns it.
// If not done manually this will be called by Parser.ParseNextFrame() or Parser.ParseToEnd().
//
// Returns ErrInvalidFileType if the filestamp (first 8 bytes) doesn't match HL2DEMO.
func (p *parser) parseHeader() (header, error) {
	var h header

	isCSTVBroadcast := p.config.Format == DemoFormatCSTVBroadcast

	if isCSTVBroadcast {
		h.Filestamp = "PBDEMS2"
	} else {
		h.Filestamp = p.bitReader.ReadCString(8)
	}

	switch h.Filestamp {
	case "HL2DEMO":
		return h, fmt.Errorf("%w: CS:GO demos are no longer supported, downgrade to v3", ErrInvalidFileType)

	case "PBDEMS2":
		if !isCSTVBroadcast {
			p.bitReader.Skip(8 << 3) // skip 8 bytes
		}

		var warnFunc func(error)

		if p.ignorePacketEntitiesPanic {
			warnFunc = func(err error) {
				p.eventDispatcher.Dispatch(events.ParserWarn{
					Type:    events.WarnTypePacketEntitiesPanic,
					Message: fmt.Sprintf("encountered PacketEntities panic: %v", err),
				})
			}
		}

		p.stParser = sendtablescs2.NewParser(warnFunc)

		p.stParser.OnEntity(p.onEntity)

		p.RegisterNetMessageHandler(p.stParser.OnServerInfo)
		p.RegisterNetMessageHandler(p.stParser.OnPacketEntities)

	default:
		return h, ErrInvalidFileType
	}

	// Initialize queue if the buffer size wasn't specified, the amount of ticks
	// seems to be a good indicator of how many events we'll get
	if p.msgQueue == nil {
		p.initMsgQueue(msgQueueSize(h.PlaybackTicks))
	}

	p.header = &h

	return h, nil
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
// Aborts and returns ErrCancelled if Cancel() is called before the end.
//
// See also: ParseNextFrame() for other possible errors.
func (p *parser) ParseToEnd() (err error) {
	defer func() {
		// Make sure all the messages of the demo are handled
		p.msgDispatcher.SyncAllQueues()
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
		_, err = p.parseHeader()
		if err != nil {
			return
		}
	}

	for {
		if !p.parseFrame() {
			return p.error()
		}

		if err = p.error(); err != nil {
			return
		}
	}
}

func recoverFromUnexpectedEOF(r any) error {
	if r == nil {
		return nil
	}

	if r == io.ErrUnexpectedEOF || r == io.EOF {
		return errors.Wrap(ErrUnexpectedEndOfDemo, "unexpected EOF")
	}

	switch err := r.(type) {
	case dispatch.ConsumerCodePanic:
		panic(err.Value())

	default:
		panic(err)
	}
}

// Cancel aborts ParseToEnd() and drains the internal event queues.
// No further events will be sent to event or message handlers after this.
func (p *parser) Cancel() {
	p.setError(ErrCancelled)
	p.eventDispatcher.UnregisterAllHandlers()
	p.msgDispatcher.UnregisterAllHandlers()
}

/*
ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).

Returns true unless the demo command 'stop' or an error was encountered.

May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
May panic if the demo is corrupt in some way.

See also: ParseToEnd() for parsing the complete demo in one go (faster).
*/
func (p *parser) ParseNextFrame() (moreFrames bool, err error) {
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
		_, err = p.parseHeader()
		if err != nil {
			return
		}
	}

	moreFrames = p.parseFrame()

	return moreFrames, p.error()
}

var demoCommandMsgsCreators = map[msg.EDemoCommands]NetMessageCreator{
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

func (p *parser) parseFrame() bool {
	cmd := msg.EDemoCommands(p.bitReader.ReadVarInt32())

	msgType := cmd & ^msg.EDemoCommands_DEM_IsCompressed
	msgCompressed := (cmd & msg.EDemoCommands_DEM_IsCompressed) != 0

	var (
		tick uint32
		size uint32
	)

	isCSTVBroadcast := p.config.Format == DemoFormatCSTVBroadcast

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

		size = p.bitReader.ReadVarInt32()
	}

	p.msgQueue <- ingameTickNumber(int32(tick))

	msgCreator := demoCommandMsgsCreators[msgType]
	if msgCreator == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("skipping unknown demo commands message type with value %d", msgType),
			Type:    events.WarnUnknownDemoCommandMessageType,
		})
		p.bitReader.Skip(int(size) << 3)

		return true
	}

	buf := p.bitReader.ReadBytes(int(size))

	if msgCompressed {
		var err error

		buf, err = snappy.Decode(nil, buf)
		if err != nil {
			if errors.Is(err, snappy.ErrCorrupt) {
				p.eventDispatcher.Dispatch(events.ParserWarn{
					Message: "compressed message is corrupt",
				})
			} else {
				panic(err)
			}
		}
	}

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
	}

	p.msgQueue <- m

	switch m := m.(type) {
	case *msg.CDemoPacket:
		p.handleDemoPacket(m)

	case *msg.CDemoFullPacket:
		p.msgQueue <- m.StringTable

		if m.Packet.GetData() != nil {
			p.handleDemoPacket(m.Packet)
		}
	}

	// Queue up some post processing
	p.msgQueue <- frameParsedToken

	return msgType != msg.EDemoCommands_DEM_Stop
}

type frameParsedTokenType struct{}

var frameParsedToken = new(frameParsedTokenType)

func (p *parser) handleFrameParsed(*frameParsedTokenType) {
	p.processFrameGameEvents()

	p.currentFrame++
	p.eventDispatcher.Dispatch(events.FrameDone{})
}

// CS2 demos playback info are available in the CDemoFileInfo message that should be parsed at the end of the demo.
// Demos may not contain it, as a workaround we update values with the last parser information at the end of parsing.
func (p *parser) ensurePlaybackValuesAreSet() {
	if p.header.PlaybackTicks == 0 {
		p.header.PlaybackTicks = p.gameState.ingameTick
		p.header.PlaybackFrames = p.currentFrame
		p.header.PlaybackTime = time.Duration(float32(p.header.PlaybackTicks)*float32(p.tickInterval)) * time.Second
	}
}
