package demoinfocs

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

const maxOsPath = 260

const (
	playerWeaponPrefix    = "m_hMyWeapons."
	playerWeaponPrePrefix = "bcc_nonlocaldata."
	gameRulesPrefix       = "cs_gamerules_data"
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
func (p *Parser) ParseHeader() (common.DemoHeader, error) {
	var h common.DemoHeader
	h.Filestamp = p.bitReader.ReadCString(8)
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

	if h.Filestamp != "HL2DEMO" {
		return h, ErrInvalidFileType
	}

	// Initialize queue if the buffer size wasn't specified, the amount of ticks
	// seems to be a good indicator of how many events we'll get
	if p.msgQueue == nil {
		p.initMsgQueue(h.PlaybackTicks)
	}

	p.header = &h

	return h, nil
}

// ParseToEnd attempts to parse the demo until the end.
// Aborts and returns ErrCancelled if Cancel() is called before the end.
//
// See also: ParseNextFrame() for other possible errors.
func (p *Parser) ParseToEnd() (err error) {
	defer func() {
		// Make sure all the messages of the demo are handled
		p.msgDispatcher.SyncAllQueues()

		// Close msgQueue
		if p.msgQueue != nil {
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

	for {
		select {
		case <-p.cancelChan:
			return ErrCancelled

		default:
			if !p.parseFrame() {
				return p.error()
			}
		}

		if err = p.error(); err != nil {
			return
		}
	}
}

func recoverFromUnexpectedEOF(r interface{}) error {
	if r != nil {
		if r == io.ErrUnexpectedEOF || r == io.EOF {
			return ErrUnexpectedEndOfDemo
		}
		panic(r)
	}
	return nil
}

// Cancel aborts ParseToEnd().
// All information that was already read up to this point may still be used (and new events may still be sent out).
func (p *Parser) Cancel() {
	p.cancelChan <- struct{}{}
}

/*
ParseNextFrame attempts to parse the next frame / demo-tick (not ingame tick).

Returns true unless the demo command 'stop' or an error was encountered.

May return ErrUnexpectedEndOfDemo for incomplete / corrupt demos.
May panic if the demo is corrupt in some way.

See also: ParseToEnd() for parsing the complete demo in one go (faster).
*/
func (p *Parser) ParseNextFrame() (moreFrames bool, err error) {
	defer func() {
		// Make sure all the messages of the frame are handled
		p.msgDispatcher.SyncAllQueues()

		// Close msgQueue (only if we are done)
		if p.msgQueue != nil && !moreFrames {
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

	moreFrames = p.parseFrame()

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

func (p *Parser) parseFrame() bool {
	cmd := demoCommand(p.bitReader.ReadSingleByte())

	// Send ingame tick number update
	p.msgQueue <- ingameTickNumber(p.bitReader.ReadSignedInt(32))

	// Skip 'player slot'
	p.bitReader.Skip(8)

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

		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.stParser.ParsePacket(p.bitReader)
		p.bitReader.EndChunk()

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

var byteSlicePool = sync.Pool{
	New: func() interface{} {
		s := make([]byte, 0, 256)
		return &s
	},
}

func (p *Parser) parsePacket() {
	// Booooring
	// 152 bytes CommandInfo, 4 bytes SeqNrIn, 4 bytes SeqNrOut
	// See at the bottom of the file what the CommandInfo would contain if you are interested.
	p.bitReader.Skip((152 + 4 + 4) << 3)

	// Here we go
	p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
	for !p.bitReader.ChunkFinished() {
		cmd := int(p.bitReader.ReadVarInt32())
		size := int(p.bitReader.ReadVarInt32())

		p.bitReader.BeginChunk(size << 3)
		var m proto.Message
		switch cmd {
		case int(msg.SVC_Messages_svc_PacketEntities):
			// We could pool CSVCMsg_PacketEntities as they take up A LOT of the allocations
			// but unless we're on a system that's doing a lot of concurrent parsing there isn't really a point
			// as handling packets is a lot slower than creating them and we can't pool until they are handled.
			m = new(msg.CSVCMsg_PacketEntities)

		case int(msg.SVC_Messages_svc_GameEventList):
			m = new(msg.CSVCMsg_GameEventList)

		case int(msg.SVC_Messages_svc_GameEvent):
			m = new(msg.CSVCMsg_GameEvent)

		case int(msg.SVC_Messages_svc_CreateStringTable):
			m = new(msg.CSVCMsg_CreateStringTable)

		case int(msg.SVC_Messages_svc_UpdateStringTable):
			m = new(msg.CSVCMsg_UpdateStringTable)

		case int(msg.SVC_Messages_svc_UserMessage):
			m = new(msg.CSVCMsg_UserMessage)

		default:
			var name string
			if cmd < 8 || cmd >= 100 {
				name = msg.NET_Messages_name[int32(cmd)]
			} else {
				name = msg.SVC_Messages_name[int32(cmd)]
			}

			debugUnhandledMessage(cmd, name)

			if name != "" {
				// Handle additional net-messages as defined by the user
				creator := p.additionalNetMessageCreators[cmd]
				if creator != nil {
					m = creator()
					break
				}
			} else {
				// Send a warning if the command is unknown
				// This might mean our proto files are out of date
				p.eventDispatcher.Dispatch(events.ParserWarn{Message: fmt.Sprintf("Unknown message command %q", cmd)})
			}

			// On to the next one
			p.bitReader.EndChunk()
			continue
		}

		b := byteSlicePool.Get().(*[]byte)
		p.bitReader.ReadBytesInto(b, size)

		if proto.Unmarshal(*b, m) != nil {
			// TODO: Don't crash here, happens with demos that work in gotv
			panic(fmt.Sprintf("Failed to unmarshal cmd %d", cmd))
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

func (p *Parser) handleFrameParsed(*frameParsedTokenType) {
	// PlayerFlashed events need to be dispatched at the end of the tick
	// because Player.FlashDuration is updated after the game-events are parsed.
	for _, e := range p.delayedEvents {
		p.eventDispatcher.Dispatch(e)
	}
	p.delayedEvents = p.delayedEvents[:0]

	p.currentFrame++
	p.eventDispatcher.Dispatch(events.TickDone{})
	p.eventDispatcher.Dispatch(events.FrameDone{})
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
