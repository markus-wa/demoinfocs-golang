package demoinfocs

import (
	"fmt"
	"sync"

	proto "github.com/gogo/protobuf/proto"

	events "github.com/markus-wa/demoinfocs-golang/events"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

var byteSlicePool = sync.Pool{
	New: func() interface{} {
		s := make([]byte, 0, 256)
		return &s
	},
}

func (p *Parser) parsePacket() {
	// Booooring
	// 152 bytes CommandInfo, 4 bytes SeqNrIn, 4 bytes SeqNrOut
	// See at the bottom what the CommandInfo would contain if you are interested.
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

			if isDebug {
				debugUnhandledMessage(cmd, name)
			}

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
				p.eventDispatcher.Dispatch(events.ParserWarnEvent{Message: fmt.Sprintf("Unknown message command %q", cmd)})
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

// NetMessageCreator creates additional net-messages to be dispatched to net-message handlers.
// See also: ParserConfig.AdditionalNetMessageCreators & Parser.RegisterNetMessageHandler()
type NetMessageCreator func() proto.Message

/*
Format of 'CommandInfos' - I honestly have no clue what they are good for.
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
