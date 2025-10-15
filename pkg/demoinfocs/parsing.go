package demoinfocs

import (
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/markus-wa/go-unassert"
	dispatch "github.com/markus-wa/godispatch"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables2"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg"
)

const maxOsPath = 260

const (
	playerWeaponPrefix   = "m_hMyWeapons"
	playerWeaponPrefixS2 = "m_pWeaponServices.m_hMyWeapons"
	gameRulesPrefix      = "cs_gamerules_data"
	gameRulesPrefixS2    = "m_pGameRules"
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

// ParseHeader attempts to parse the header of the demo and returns it.
// If not done manually this will be called by Parser.ParseNextFrame() or Parser.ParseToEnd().
//
// Returns ErrInvalidFileType if the filestamp (first 8 bytes) doesn't match HL2DEMO.
func (p *parser) ParseHeader() (common.DemoHeader, error) {
	var h common.DemoHeader
	h.Filestamp = p.bitReader.ReadCString(8)

	switch h.Filestamp {
	case "HL2DEMO":
		h.Protocol = p.bitReader.ReadSignedInt(32)
		h.NetworkProtocol = p.bitReader.ReadSignedInt(32)
		h.ServerName = p.bitReader.ReadCString(maxOsPath)
		h.ClientName = p.bitReader.ReadCString(maxOsPath)
		h.MapName = p.bitReader.ReadCString(maxOsPath)
		h.GameDirectory = p.bitReader.ReadCString(maxOsPath)
		h.PlaybackTime = time.Duration(p.bitReader.ReadFloat() * float32(time.Second))
		h.PlaybackTicks = p.bitReader.ReadSignedInt(32)
		h.PlaybackFrames = p.bitReader.ReadSignedInt(32)
		h.SignonLength = p.bitReader.ReadSignedInt(32)

		p.stParser = st.NewSendTableParser()

	case "PBDEMS2":
		p.bitReader.Skip(8 << 3) // skip 8 bytes

		var warnFunc func(error)

		if p.ignorePacketEntitiesPanic {
			warnFunc = func(err error) {
				p.eventDispatcher.Dispatch(events.ParserWarn{
					Type:    events.WarnTypePacketEntitiesPanic,
					Message: fmt.Sprintf("encountered PacketEntities panic: %v", err),
				})
			}
		}

		p.stParser = sendtables2.NewParser(warnFunc)

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
		_, err = p.ParseHeader()
		if err != nil {
			return
		}
	}

	parseFrame := p.parseFrameFn()

	for {
		if !parseFrame() {
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
		_, err = p.ParseHeader()
		if err != nil {
			return
		}
	}

	moreFrames = p.parseFrameFn()()

	return moreFrames, p.error()
}

// Demo commands as documented at https://developer.valvesoftware.com/wiki/DEM_Format
type demoCommand byte

const (
	dcSignon         demoCommand = 1
	dcPacket         demoCommand = 2
	dcSynctick       demoCommand = 3
	dcConsoleCommand demoCommand = 4
	dcUserCommand    demoCommand = 5
	dcDataTables     demoCommand = 6
	dcStop           demoCommand = 7
	dcCustomData     demoCommand = 8
	dcStringTables   demoCommand = 9
)

//nolint:funlen,cyclop
func (p *parser) parseFrameS1() bool {
	cmd := demoCommand(p.bitReader.ReadSingleByte())

	// Send ingame tick number update
	p.msgQueue <- ingameTickNumber(p.bitReader.ReadSignedInt(32))

	// Skip 'player slot'
	const nSlotBits = 8
	p.bitReader.Skip(nSlotBits) //nolint:wsl

	debugDemoCommand(cmd)

	switch cmd {
	case dcSynctick:
		// Ignore

	case dcStop:
		return false

	case dcConsoleCommand:
		// Skip
		p.bitReader.Skip(p.bitReader.ReadSignedInt(32) << 3)

	case dcDataTables:
		p.msgDispatcher.SyncAllQueues()

		b := p.bitReader.ReadBytes(p.bitReader.ReadSignedInt(32))

		err := p.stParser.ParsePacket(b)
		if err != nil {
			panic(err)
		}

		debugAllServerClasses(p.ServerClasses())

		p.mapEquipment()
		p.bindEntities()

		p.eventDispatcher.Dispatch(events.DataTablesParsed{})

	case dcStringTables:
		p.msgDispatcher.SyncAllQueues()

		p.parseStringTables()

	case dcUserCommand:
		// Skip
		p.bitReader.Skip(32)
		p.bitReader.Skip(p.bitReader.ReadSignedInt(32) << 3)

	case dcSignon:
		fallthrough
	case dcPacket:
		p.parsePacket()

	case dcCustomData:
		// Might as well panic since we'll be way off if we dont skip the whole thing
		panic("Found CustomData but not handled")

	default:
		panic(fmt.Sprintf("I haven't programmed that pathway yet (command %v unknown)", cmd))
	}

	// Queue up some post processing
	p.msgQueue <- frameParsedToken

	return true
}

var demoCommandMsgsCreators = map[msgs2.EDemoCommands]NetMessageCreator{
	msgs2.EDemoCommands_DEM_Stop:            func() proto.Message { return &msgs2.CDemoStop{} },
	msgs2.EDemoCommands_DEM_FileHeader:      func() proto.Message { return &msgs2.CDemoFileHeader{} },
	msgs2.EDemoCommands_DEM_FileInfo:        func() proto.Message { return &msgs2.CDemoFileInfo{} },
	msgs2.EDemoCommands_DEM_SyncTick:        func() proto.Message { return &msgs2.CDemoSyncTick{} },
	msgs2.EDemoCommands_DEM_SendTables:      func() proto.Message { return &msgs2.CDemoSendTables{} },
	msgs2.EDemoCommands_DEM_ClassInfo:       func() proto.Message { return &msgs2.CDemoClassInfo{} },
	msgs2.EDemoCommands_DEM_StringTables:    func() proto.Message { return &msgs2.CDemoStringTables{} },
	msgs2.EDemoCommands_DEM_Packet:          func() proto.Message { return &msgs2.CDemoPacket{} },
	msgs2.EDemoCommands_DEM_SignonPacket:    func() proto.Message { return &msgs2.CDemoPacket{} },
	msgs2.EDemoCommands_DEM_ConsoleCmd:      func() proto.Message { return &msgs2.CDemoConsoleCmd{} },
	msgs2.EDemoCommands_DEM_CustomData:      func() proto.Message { return &msgs2.CDemoCustomData{} },
	msgs2.EDemoCommands_DEM_UserCmd:         func() proto.Message { return &msgs2.CDemoUserCmd{} },
	msgs2.EDemoCommands_DEM_FullPacket:      func() proto.Message { return &msgs2.CDemoFullPacket{} },
	msgs2.EDemoCommands_DEM_SaveGame:        func() proto.Message { return &msgs2.CDemoSaveGame{} },
	msgs2.EDemoCommands_DEM_SpawnGroups:     func() proto.Message { return &msgs2.CDemoSpawnGroups{} },
	msgs2.EDemoCommands_DEM_AnimationData:   func() proto.Message { return &msgs2.CDemoAnimationData{} },
	msgs2.EDemoCommands_DEM_AnimationHeader: func() proto.Message { return &msgs2.CDemoAnimationHeader{} },
	msgs2.EDemoCommands_DEM_Recovery:        func() proto.Message { return &msgs2.CDemoRecovery{} },
}

func (p *parser) parseFrameS2() bool {
	cmd := msgs2.EDemoCommands(p.bitReader.ReadVarInt32())

	msgType := cmd & ^msgs2.EDemoCommands_DEM_IsCompressed
	msgCompressed := (cmd & msgs2.EDemoCommands_DEM_IsCompressed) != 0

	tick := p.bitReader.ReadVarInt32()

	// This appears to actually be an int32, where a -1 means pre-game.
	if tick == 4294967295 {
		tick = 0
	}

	p.msgQueue <- ingameTickNumber(int32(tick))

	if msgType == msgs2.EDemoCommands_DEM_Stop {
		p.msgQueue <- frameParsedToken
		return false
	}

	size := p.bitReader.ReadVarInt32()

	msgCreator := demoCommandMsgsCreators[msgType]
	if msgCreator == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("skipping unknown demo commands message type with value %d", msgType),
			Type:    events.WarnTypeUnknownDemoCommandMessageType,
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

	msg := msgCreator()

	if msg == nil {
		panic(fmt.Sprintf("Unknown demo command: %d", msgType))
	}

	err := proto.Unmarshal(buf, msg)
	if err != nil {
		panic(err) // FIXME: avoid panic
	}

	p.msgQueue <- msg

	switch m := msg.(type) {
	case *msgs2.CDemoPacket:
		p.handleDemoPacket(m)

	case *msgs2.CDemoFullPacket:
		p.msgQueue <- m.StringTable

		if m.Packet.GetData() != nil {
			p.handleDemoPacket(m.Packet)
		}
	}

	// Queue up some post processing
	p.msgQueue <- frameParsedToken

	return msgType != msgs2.EDemoCommands_DEM_Stop
}

// FIXME: refactor to interface instead of switch
func (p *parser) parseFrameFn() func() bool {
	switch p.header.Filestamp {
	case "HL2DEMO":
		return p.parseFrameS1

	case "PBDEMS2":
		return p.parseFrameS2

	default:
		panic(fmt.Sprintf("Unknown demo version: %s", p.header.Filestamp))
	}
}

var byteSlicePool = sync.Pool{
	New: func() any {
		s := make([]byte, 0, 256)

		return &s
	},
}

var defaultNetMessageCreators = map[int]NetMessageCreator{
	// We could pool CSVCMsg_PacketEntities as they take up A LOT of the allocations
	// but unless we're on a system that's doing a lot of concurrent parsing there isn't really a point
	// as handling packets is a lot slower than creating them and we can't pool until they are handled.
	int(msg.SVC_Messages_svc_PacketEntities):    func() proto.Message { return new(msg.CSVCMsg_PacketEntities) },
	int(msg.SVC_Messages_svc_GameEventList):     func() proto.Message { return new(msg.CSVCMsg_GameEventList) },
	int(msg.SVC_Messages_svc_GameEvent):         func() proto.Message { return new(msg.CSVCMsg_GameEvent) },
	int(msg.SVC_Messages_svc_CreateStringTable): func() proto.Message { return new(msg.CSVCMsg_CreateStringTable) },
	int(msg.SVC_Messages_svc_UpdateStringTable): func() proto.Message { return new(msg.CSVCMsg_UpdateStringTable) },
	int(msg.SVC_Messages_svc_UserMessage):       func() proto.Message { return new(msg.CSVCMsg_UserMessage) },
	int(msg.SVC_Messages_svc_ServerInfo):        func() proto.Message { return new(msg.CSVCMsg_ServerInfo) },
	int(msg.NET_Messages_net_SetConVar):         func() proto.Message { return new(msg.CNETMsg_SetConVar) },
	int(msg.SVC_Messages_svc_EncryptedData):     func() proto.Message { return new(msg.CSVCMsg_EncryptedData) },
}

func (p *parser) netMessageForCmd(cmd int) proto.Message {
	msgCreator := defaultNetMessageCreators[cmd]

	if msgCreator != nil {
		return msgCreator()
	}

	var msgName string
	if cmd < 8 || cmd >= 100 {
		msgName = msg.NET_Messages_name[int32(cmd)]
	} else {
		msgName = msg.SVC_Messages_name[int32(cmd)]
	}

	if msgName == "" {
		// Send a warning if the command is unknown
		// This might mean our proto files are out of date
		p.eventDispatcher.Dispatch(events.ParserWarn{Message: fmt.Sprintf("unknown message command %q", cmd)})
		unassert.Error("unknown message command %q", cmd)
	}

	// Handle additional net-messages as defined by the user
	msgCreator = p.additionalNetMessageCreators[cmd]
	if msgCreator != nil {
		return msgCreator()
	}

	debugUnhandledMessage(cmd, msgName)

	return nil
}

func (p *parser) parsePacket() {
	// Booooring
	// 152 bytes CommandInfo, 4 bytes SeqNrIn, 4 bytes SeqNrOut
	// See at the bottom of the file what the CommandInfo would contain if you are interested.
	const nCommandInfoBits = (152 + 4 + 4) << 3
	p.bitReader.Skip(nCommandInfoBits) //nolint:wsl

	// Here we go
	p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)

	for !p.bitReader.ChunkFinished() {
		cmd := int(p.bitReader.ReadVarInt32())
		size := int(p.bitReader.ReadVarInt32())

		p.bitReader.BeginChunk(size << 3)

		m := p.netMessageForCmd(cmd)

		if m == nil {
			// On to the next one
			p.bitReader.EndChunk()

			continue
		}

		b := byteSlicePool.Get().(*[]byte)

		p.bitReader.ReadBytesInto(b, size)

		err := proto.Unmarshal(*b, m)
		if err != nil {
			// TODO: Don't crash here, happens with demos that work in gotv
			p.setError(errors.Wrapf(err, "failed to unmarshal cmd %d", cmd))

			return
		}

		p.msgQueue <- m

		// Reset length to 0 and pool
		*b = (*b)[:0]
		byteSlicePool.Put(b)

		p.bitReader.EndChunk()
	}

	p.bitReader.EndChunk()
}

type frameParsedTokenType struct{}

var frameParsedToken = new(frameParsedTokenType)

func (p *parser) handleFrameParsed(*frameParsedTokenType) {
	p.processFrameGameEvents()

	p.currentFrame++
	p.eventDispatcher.Dispatch(events.FrameDone{})

	if p.isSource2() {
		p.updatePlayersPreviousFramePosition()
	}
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

/*
Format of 'CommandInfos' - I honestly have no clue what they are good for.
This data is skipped in Parser.parsePacket().
If you find a use for this please let me know!

Here is all i know:

CommandInfo [152 bytes]
- [2]Split

Split [76 bytes]
- flags [4 bytes]
- viewOrigin [12 bytes]
- viewAngles [12 bytes]
- localViewAngles [12 bytes]
- viewOrigin2 [12 bytes]
- viewAngles2 [12 bytes]
- localViewAngles2 [12 bytes]

Origin [12 bytes]
- X [4 bytes]
- Y [4 bytes]
- Z [4 bytes]

Angle [12 bytes]
- X [4 bytes]
- Y [4 bytes]
- Z [4 bytes]

They are parsed in the following order:
split1.flags
split1.viewOrigin.x
split1.viewOrigin.y
split1.viewOrigin.z
split1.viewAngles.x
split1.viewAngles.y
split1.viewAngles.z
split1.localViewAngles.x
split1.localViewAngles.y
split1.localViewAngles.z
split1.viewOrigin2...
split1.viewAngles2...
split1.localViewAngles2...
split2.flags
...

Or just check this file's history for an example on how to parse them
*/
