package dp

import (
	"github.com/golang/protobuf/proto"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

func ParsePacket(reader bs.BitReader, msgQueue chan interface{}) {
	for !reader.ChunkFinished() {
		cmd := int(reader.ReadVarInt32())
		size := int(reader.ReadVarInt32())

		reader.BeginChunk(size * 8)
		var m proto.Message
		switch cmd {
		case int(msg.SVC_Messages_svc_PacketEntities):
			m = &msg.CSVCMsg_PacketEntities{}

		case int(msg.SVC_Messages_svc_GameEventList):
			m = &msg.CSVCMsg_GameEventList{}

		case int(msg.SVC_Messages_svc_GameEvent):
			m = &msg.CSVCMsg_GameEvent{}

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
