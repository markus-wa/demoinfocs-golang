package demoinfocs

import (
	"bytes"
	"slices"
	"testing"

	"github.com/golang/snappy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

// rawNetMessage is a net-message embedded in a packet, with a raw (possibly corrupt) payload and size.
type rawNetMessage struct {
	t       int
	size    int
	payload []byte
}

func packetData(msgs ...rawNetMessage) []byte {
	w := &bitWriter{}

	for _, m := range msgs {
		w.writeUBitInt(uint64(m.t))
		w.writeVarInt(uint64(m.size))

		for _, x := range m.payload {
			w.writeBits(uint64(x), 8)
		}
	}

	return w.buf
}

func marshalledNetMessage(t *testing.T, msgType int, m proto.Message) rawNetMessage {
	t.Helper()

	b, err := proto.Marshal(m)
	require.NoError(t, err)

	return rawNetMessage{t: msgType, size: len(b), payload: b}
}

// demoFrame encodes a frame of a CS2 demo: command, tick, size and payload.
func demoFrame(cmd msg.EDemoCommands, tick int, payload []byte) []byte {
	b := protowire.AppendVarint(nil, uint64(cmd))
	b = protowire.AppendVarint(b, uint64(tick))
	b = protowire.AppendVarint(b, uint64(len(payload)))

	return append(b, payload...)
}

func fullPacketFrame(t *testing.T, data []byte) []byte {
	t.Helper()

	b, err := proto.Marshal(&msg.CDemoFullPacket{Packet: &msg.CDemoPacket{Data: data}})
	require.NoError(t, err)

	return demoFrame(msg.EDemoCommands_DEM_FullPacket, 7, b)
}

// parseFrames parses the frames of stream with parseFrameS2() and returns the queued messages.
func parseFrames(t *testing.T, p *parser, stream []byte, nFrames int) []any {
	t.Helper()

	p.bitReader = bit.NewLargeBitReader(bytes.NewReader(stream))
	p.msgQueue = make(chan any, 1000)

	for range nFrames {
		assert.True(t, p.parseFrameS2())
	}

	close(p.msgQueue)

	var queued []any

	for m := range p.msgQueue {
		queued = append(queued, m)
	}

	return queued
}

func assertBrokenFullPacketSkipped(t *testing.T, queued []any, cause string) {
	t.Helper()

	require.Len(t, queued, 3)
	assert.Equal(t, ingameTickNumber(7), queued[0])
	require.IsType(t, &brokenFullPacket{}, queued[1])
	assert.ErrorContains(t, queued[1].(*brokenFullPacket).err, cause)
	assert.Equal(t, frameParsedToken, queued[2])
}

func TestParseFrameS2_FullPacket(t *testing.T) {
	data := packetData(marshalledNetMessage(t, int(msg.NET_Messages_net_Tick), &msg.CNETMsg_Tick{Tick: proto.Uint32(7)}))

	queued := parseFrames(t, newParser(), fullPacketFrame(t, data), 1)

	require.Len(t, queued, 5)
	assert.Equal(t, ingameTickNumber(7), queued[0])
	assert.IsType(t, &msg.CDemoFullPacket{}, queued[1])
	assert.IsType(t, (*msg.CDemoStringTables)(nil), queued[2])
	require.IsType(t, &msg.CNETMsg_Tick{}, queued[3])
	assert.Equal(t, uint32(7), queued[3].(*msg.CNETMsg_Tick).GetTick())
	assert.Equal(t, frameParsedToken, queued[4])
}

func TestParseFrameS2_BrokenFullPacket_CorruptSnappy(t *testing.T) {
	frame := demoFrame(msg.EDemoCommands_DEM_FullPacket|msg.EDemoCommands_DEM_IsCompressed, 7, []byte{0x05, 0xff, 0xff, 0xff})

	assertBrokenFullPacketSkipped(t, parseFrames(t, newParser(), frame, 1), snappy.ErrCorrupt.Error())
}

func TestParseFrameS2_BrokenFullPacket_CorruptProtobuf(t *testing.T) {
	frame := demoFrame(msg.EDemoCommands_DEM_FullPacket, 7, []byte{0xff, 0xff, 0xff})

	assertBrokenFullPacketSkipped(t, parseFrames(t, newParser(), frame, 1), "proto")
}

func TestParseFrameS2_BrokenFullPacket_CorruptNetMessage(t *testing.T) {
	// the valid net-message before the corrupt one mustn't be queued either
	data := packetData(
		marshalledNetMessage(t, int(msg.NET_Messages_net_Tick), &msg.CNETMsg_Tick{Tick: proto.Uint32(7)}),
		rawNetMessage{t: int(msg.NET_Messages_net_Tick), size: 2, payload: []byte{0xff, 0xff}},
	)

	assertBrokenFullPacketSkipped(t, parseFrames(t, newParser(), fullPacketFrame(t, data), 1), "failed to unmarshal net-message of type 4")
}

func TestParseFrameS2_BrokenFullPacket_NetMessageSizeTooLarge(t *testing.T) {
	data := packetData(rawNetMessage{t: int(msg.NET_Messages_net_Tick), size: 1 << 30, payload: []byte{1, 2}})

	assertBrokenFullPacketSkipped(t, parseFrames(t, newParser(), fullPacketFrame(t, data), 1), "exceeds the remaining 2 bytes of the packet")
}

func TestParseFrameS2_BrokenFullPacket_ParsingContinues(t *testing.T) {
	broken := demoFrame(msg.EDemoCommands_DEM_FullPacket, 7, []byte{0xff, 0xff, 0xff})

	packet, err := proto.Marshal(&msg.CDemoPacket{Data: packetData(marshalledNetMessage(t, int(msg.NET_Messages_net_Tick), &msg.CNETMsg_Tick{Tick: proto.Uint32(8)}))})
	require.NoError(t, err)

	stream := slices.Concat(broken, demoFrame(msg.EDemoCommands_DEM_Packet, 8, packet))

	queued := parseFrames(t, newParser(), stream, 2)

	assertBrokenFullPacketSkipped(t, queued[:3], "proto")
	require.Len(t, queued, 7)
	assert.Equal(t, ingameTickNumber(8), queued[3])
	assert.IsType(t, &msg.CDemoPacket{}, queued[4])
	assert.IsType(t, &msg.CNETMsg_Tick{}, queued[5])
	assert.Equal(t, frameParsedToken, queued[6])
}

func TestParseFrameS2_BrokenPacket_Panics(t *testing.T) {
	data := packetData(rawNetMessage{t: int(msg.NET_Messages_net_Tick), size: 2, payload: []byte{0xff, 0xff}})

	packet, err := proto.Marshal(&msg.CDemoPacket{Data: data})
	require.NoError(t, err)

	p := newParser()
	p.bitReader = bit.NewLargeBitReader(bytes.NewReader(demoFrame(msg.EDemoCommands_DEM_Packet, 7, packet)))
	p.msgQueue = make(chan any, 1000)

	assert.Panics(t, func() { p.parseFrameS2() })
}

func TestParser_HandleBrokenFullPacket(t *testing.T) {
	p := newParser()

	var warns []events.ParserWarn

	p.RegisterEventHandler(func(w events.ParserWarn) {
		warns = append(warns, w)
	})

	p.handleBrokenFullPacket(&brokenFullPacket{err: snappy.ErrCorrupt})

	require.Len(t, warns, 1)
	assert.Equal(t, events.WarnType(events.WarnTypeBrokenFullPacket), warns[0].Type)
	assert.Equal(t, "skipped broken DEM_FullPacket: snappy: corrupt input", warns[0].Message)
}
