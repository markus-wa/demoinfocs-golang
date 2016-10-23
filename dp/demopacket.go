package dp

import (
	"github.com/gogo/protobuf/proto"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"sync"
)

var packetEntitiesPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return &msg.CSVCMsg_PacketEntities{}
	},
}

var gameEventPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return &msg.CSVCMsg_GameEvent{}
	},
}

func ParsePacket(reader bs.BitReader, msgQueue chan interface{}) {
	for !reader.ChunkFinished() {
		cmd := int(reader.ReadVarInt32())
		size := int(reader.ReadVarInt32())

		reader.BeginChunk(size * 8)
		var m proto.Message
		switch cmd {
		case int(msg.SVC_Messages_svc_PacketEntities):
			m = packetEntitiesPool.Get().(*msg.CSVCMsg_PacketEntities)
			defer packetEntitiesPool.Put(m)

		case int(msg.SVC_Messages_svc_GameEventList):
			m = &msg.CSVCMsg_GameEventList{}

		case int(msg.SVC_Messages_svc_GameEvent):
			m = gameEventPool.Get().(*msg.CSVCMsg_GameEvent)
			defer gameEventPool.Put(m)

		case int(msg.SVC_Messages_svc_CreateStringTable):
			m = &msg.CSVCMsg_CreateStringTable{}

		case int(msg.SVC_Messages_svc_UpdateStringTable):
			m = &msg.CSVCMsg_UpdateStringTable{}

		case int(msg.NET_Messages_net_Tick):
			m = &msg.CNETMsg_Tick{}

		case int(msg.SVC_Messages_svc_UserMessage):
			m = &msg.CSVCMsg_UserMessage{}

		default:
			// We don't care about anything else for now
		}
		if m != nil {
			proto.Unmarshal(reader.ReadBytes(size), m)
			msgQueue <- m
		}
		reader.EndChunk()
	}
}
