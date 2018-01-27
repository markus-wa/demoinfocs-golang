package demoinfocs

import (
	"fmt"
	"sync"

	proto "github.com/gogo/protobuf/proto"
	r3 "github.com/golang/geo/r3"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

var packetEntitiesPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(msg.CSVCMsg_PacketEntities)
	},
}

var gameEventPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return new(msg.CSVCMsg_GameEvent)
	},
}

var byteSlicePool sync.Pool = sync.Pool{
	New: func() interface{} {
		s := make([]byte, 0, 256)
		return &s
	},
}

func (p *Parser) parsePacket() {
	// Booooring
	parseCommandInfo(p.bitReader)
	p.bitReader.ReadInt(32) // SeqNrIn
	p.bitReader.ReadInt(32) // SeqNrOut

	// Here we go
	p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
	for !p.bitReader.ChunkFinished() {
		cmd := int(p.bitReader.ReadVarInt32())
		size := int(p.bitReader.ReadVarInt32())

		p.bitReader.BeginChunk(size << 3)
		var m proto.Message
		switch cmd {
		case int(msg.SVC_Messages_svc_PacketEntities):
			m = packetEntitiesPool.Get().(*msg.CSVCMsg_PacketEntities)
			defer packetEntitiesPool.Put(m)

		case int(msg.SVC_Messages_svc_GameEventList):
			m = new(msg.CSVCMsg_GameEventList)

		case int(msg.SVC_Messages_svc_GameEvent):
			m = gameEventPool.Get().(*msg.CSVCMsg_GameEvent)
			defer gameEventPool.Put(m)

		case int(msg.SVC_Messages_svc_CreateStringTable):
			m = new(msg.CSVCMsg_CreateStringTable)

		case int(msg.SVC_Messages_svc_UpdateStringTable):
			m = new(msg.CSVCMsg_UpdateStringTable)

		default:
			// We don't care about anything else for now
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

		// Reset to 0 length and pool
		*b = (*b)[:0]
		byteSlicePool.Put(b)

		p.bitReader.EndChunk()
	}
	p.bitReader.EndChunk()

	// Make sure the created events are consumed so they can be pooled
	p.msgDispatcher.SyncQueues(p.msgQueue)
}

// TODO: Find out what all this is good for and why we didn't use the removed functions on seVector, split & commandInfo
type commandInfo struct {
	splits [2]split
}

type split struct {
	flags int

	viewOrigin      seVector
	viewAngles      r3.Vector
	localViewAngles r3.Vector

	viewOrigin2      seVector
	viewAngles2      r3.Vector
	localViewAngles2 r3.Vector
}
type seVector struct {
	r3.Vector
}

type boundingBoxInformation struct {
	index int
	min   r3.Vector
	max   r3.Vector
}

func (bbi boundingBoxInformation) contains(point r3.Vector) bool {
	return point.X >= bbi.min.X && point.X <= bbi.max.X &&
		point.Y >= bbi.min.Y && point.Y <= bbi.max.Y &&
		point.Z >= bbi.min.Z && point.Z <= bbi.max.Z
}

type bombsiteInfo struct {
	index  int
	center r3.Vector
}

func parseCommandInfo(r *bit.BitReader) commandInfo {
	return commandInfo{splits: [2]split{parseSplit(r), parseSplit(r)}}
}

func parseSplit(r *bit.BitReader) split {
	return split{
		flags: r.ReadSignedInt(32),

		viewOrigin:      parseSEVector(r),
		viewAngles:      parseVector(r),
		localViewAngles: parseVector(r),

		viewOrigin2:      parseSEVector(r),
		viewAngles2:      parseVector(r),
		localViewAngles2: parseVector(r),
	}
}

func parseSEVector(r *bit.BitReader) seVector {
	return seVector{parseVector(r)}
}

func parseVector(r *bit.BitReader) r3.Vector {
	return r3.Vector{
		X: float64(r.ReadFloat()),
		Y: float64(r.ReadFloat()),
		Z: float64(r.ReadFloat()),
	}
}
