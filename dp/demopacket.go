package dp

import (
	"github.com/golang/protobuf/proto"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

func ParsePacket(r bs.BitReader) {
	for !r.ChunkFinished() {
		cmd := msg.SVC_Messages(r.ReadVarInt32())
		size := int(r.ReadVarInt32())

		r.BeginChunk(size * 8)
		switch cmd {
		case msg.SVC_Messages_svc_GameEvent:
			ge := &msg.CSVCMsg_GameEvent{}
			b := r.ReadBytes(size)
			proto.Unmarshal(b, ge)
		}
		r.EndChunk()
	}
}
