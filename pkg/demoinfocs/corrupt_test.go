package demoinfocs

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"runtime"
	"runtime/metrics"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/golang/snappy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// Builders for handmade demos, see also handmadeDemo().

// frame returns a frame of the given command with the payload.
func frame(cmd msg.EDemoCommands, tick uint32, payload []byte) []byte {
	return append(frameHeader(cmd, tick, uint64(len(payload))), payload...)
}

// frameHeader returns the header of a frame of the given command, whose payload has the given size.
func frameHeader(cmd msg.EDemoCommands, tick uint32, size uint64) []byte {
	b := binary.AppendUvarint(nil, uint64(cmd))
	b = binary.AppendUvarint(b, uint64(tick))

	return binary.AppendUvarint(b, size)
}

// protoFrame returns a frame of the given command with m as payload.
func protoFrame(t testing.TB, cmd msg.EDemoCommands, tick uint32, m proto.Message) []byte {
	t.Helper()

	payload, err := proto.Marshal(m)
	require.NoError(t, err)

	return frame(cmd, tick, payload)
}

// compressedFrame returns a frame of the given command with the snappy compressed payload.
func compressedFrame(cmd msg.EDemoCommands, tick uint32, payload []byte) []byte {
	return frame(cmd|msg.EDemoCommands_DEM_IsCompressed, tick, snappy.Encode(nil, payload))
}

// stopFrame returns a DEM_Stop frame (it has no size and payload).
func stopFrame() []byte {
	b := binary.AppendUvarint(nil, uint64(msg.EDemoCommands_DEM_Stop))

	return binary.AppendUvarint(b, 0)
}

// fileHeaderFrame returns a DEM_FileHeader frame.
func fileHeaderFrame(t testing.TB) []byte {
	t.Helper()

	return protoFrame(t, msg.EDemoCommands_DEM_FileHeader, 0, &msg.CDemoFileHeader{
		DemoFileStamp: proto.String(filestampS2),
		MapName:       proto.String("de_test"),
		ClientName:    proto.String("SourceTV Demo"),
		PatchVersion:  proto.Int32(14100),
	})
}

// nestedMessage is a net-message embedded in a (full) packet.
type nestedMessage struct {
	t    int32
	data []byte
	size *uint32 // the size written to the header, len(data) if nil
}

// nested returns m as nestedMessage of type t.
func nested(t testing.TB, typ int32, m proto.Message) nestedMessage {
	t.Helper()

	data, err := proto.Marshal(m)
	require.NoError(t, err)

	return nestedMessage{t: typ, data: data}
}

// uBitInt writes v the way BitReader.ReadUBitInt() reads it.
func (w *stringTableWriter) uBitInt(v uint64) {
	switch {
	case v < 1<<4:
		w.bits(v, 6)
	case v < 1<<8:
		w.bits(v&15|16, 6)
		w.bits(v>>4, 4)
	case v < 1<<12:
		w.bits(v&15|32, 6)
		w.bits(v>>4, 8)
	default:
		w.bits(v&15|48, 6)
		w.bits(v>>4, 28)
	}
}

// packetData encodes net-messages the way decodeDemoPacket() reads them.
func packetData(msgs ...nestedMessage) []byte {
	w := new(stringTableWriter)

	for _, m := range msgs {
		w.uBitInt(uint64(m.t))

		size := uint32(len(m.data))
		if m.size != nil {
			size = *m.size
		}

		w.varUint32(size)
		w.bytes(m.data)
	}

	return w.buf
}

// packetFrame returns a DEM_Packet frame with the given net-messages.
func packetFrame(t testing.TB, tick uint32, msgs ...nestedMessage) []byte {
	t.Helper()

	return protoFrame(t, msg.EDemoCommands_DEM_Packet, tick, &msg.CDemoPacket{Data: packetData(msgs...)})
}

// fullPacketFrame returns a DEM_FullPacket frame with the given net-messages.
func fullPacketFrame(t testing.TB, tick uint32, msgs ...nestedMessage) []byte {
	t.Helper()

	return protoFrame(t, msg.EDemoCommands_DEM_FullPacket, tick, &msg.CDemoFullPacket{
		StringTable: &msg.CDemoStringTables{},
		Packet:      &msg.CDemoPacket{Data: packetData(msgs...)},
	})
}

// netTick returns a net_Tick net-message.
func netTick(t testing.TB, tick uint32) nestedMessage {
	t.Helper()

	return nested(t, int32(msg.NET_Messages_net_Tick), &msg.CNETMsg_Tick{Tick: proto.Uint32(tick)})
}

// allocatedBytes returns the number of bytes f allocated on the heap (of all goroutines, so tests using it
// mustn't run in parallel). testing.AllocsPerRun() counts allocations, not bytes.
func allocatedBytes(f func()) uint64 {
	sample := []metrics.Sample{{Name: "/gc/heap/allocs:bytes"}}

	metrics.Read(sample)
	before := sample[0].Value.Uint64()

	f()

	metrics.Read(sample)

	return sample[0].Value.Uint64() - before
}

// parse parses the demo with ParseToEnd(), it returns the number of FrameDone events and the error.
func parse(t testing.TB, demo []byte) (frames int, err error) {
	t.Helper()

	p := NewParser(bytes.NewReader(demo))
	defer p.Close()

	p.RegisterEventHandler(func(events.FrameDone) { frames++ })

	err = p.ParseToEnd(t.Context())

	return frames, err
}

func TestMinimalDemo(t *testing.T) {
	t.Parallel()

	p := NewParser(bytes.NewReader(handmadeDemo(fileHeaderFrame(t), stopFrame())))
	defer p.Close()

	require.NoError(t, p.ParseToEnd(t.Context()))
	assert.Equal(t, "de_test", p.header.MapName)
	assert.Equal(t, 2, p.CurrentFrame())
}

func TestReadError_Handmade(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	p := NewParser(io.MultiReader(bytes.NewReader(handmadeDemo(fileHeaderFrame(t))), iotest.ErrReader(errBoom)))
	defer p.Close()

	err := p.ParseToEnd(t.Context())
	require.ErrorIs(t, err, errBoom)
	assert.NotErrorIs(t, err, ErrUnexpectedEndOfDemo)
	assert.NotErrorIs(t, err, ErrDecodeFailed)
}

// assertDecodeError asserts that err is a DecodeError of frame 1 (the frame after the file header) containing the given text.
func assertDecodeError(t *testing.T, err error, contains string) {
	t.Helper()

	require.ErrorIs(t, err, ErrDecodeFailed)
	assert.NotErrorIs(t, err, ErrUnexpectedEndOfDemo)
	assert.Contains(t, err.Error(), contains)

	var de *DecodeError
	if assert.ErrorAs(t, err, &de) {
		assert.Equal(t, 1, de.Frame)
	}
}

func assertTruncated(t *testing.T, err error) {
	t.Helper()

	assert.ErrorIs(t, err, ErrUnexpectedEndOfDemo)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.NotErrorIs(t, err, ErrDecodeFailed)
}

// The limits are checked before anything is allocated.
// Not parallel, see allocatedBytes().
//
//nolint:paralleltest
func TestOversizeFrame(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  msg.EDemoCommands
		size uint64
	}{
		{"packet", msg.EDemoCommands_DEM_Packet, maxFramePayload + 1},
		{"compressed packet", msg.EDemoCommands_DEM_Packet | msg.EDemoCommands_DEM_IsCompressed, maxFramePayload + 1},
		{"animation data", msg.EDemoCommands_DEM_AnimationData, maxAnimationDataPayload + 1},
		{"max uint32", msg.EDemoCommands_DEM_Packet, 1<<32 - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			demo := handmadeDemo(fileHeaderFrame(t), frameHeader(tc.cmd, 0, tc.size), make([]byte, 64))

			var err error

			allocated := allocatedBytes(func() { _, err = parse(t, demo) })

			assertDecodeError(t, err, fmt.Sprintf("payload of %d bytes exceeds the limit", tc.size))
			assert.Less(t, allocated, uint64(4<<20))
		})
	}
}

// A payload that's larger than the rest of the demo is a truncated demo, it doesn't allocate its size up front.
// Not parallel, see allocatedBytes().
//
//nolint:paralleltest
func TestTruncatedPayload(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		t.Run(fmt.Sprint("compressed=", compressed), func(t *testing.T) {
			cmd := msg.EDemoCommands_DEM_Packet
			if compressed {
				cmd |= msg.EDemoCommands_DEM_IsCompressed
			}

			demo := handmadeDemo(fileHeaderFrame(t), frameHeader(cmd, 0, maxFramePayload), make([]byte, 100))

			var (
				err    error
				frames int
			)

			allocated := allocatedBytes(func() { frames, err = parse(t, demo) })

			assertTruncated(t, err)
			assert.Equal(t, 1, frames, "the frames before must be handled")
			assert.Less(t, allocated, uint64(4<<20))
		})
	}
}

// Not parallel, see allocatedBytes().
//
//nolint:paralleltest
func TestSnappyBomb(t *testing.T) {
	for _, tc := range []struct {
		name       string
		decodedLen uint64
	}{
		{"above the limit", maxFramePayload + 1},
		{"above the ratio", maxFramePayload},
		{"max uint32", 1<<32 - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := binary.AppendUvarint(nil, tc.decodedLen)
			payload = append(payload, make([]byte, 100)...)

			demo := handmadeDemo(fileHeaderFrame(t), frame(msg.EDemoCommands_DEM_Packet|msg.EDemoCommands_DEM_IsCompressed, 0, payload), stopFrame())

			var err error

			allocated := allocatedBytes(func() { _, err = parse(t, demo) })

			assertDecodeError(t, err, "")
			assert.True(t, errors.Is(err, errSnappyLimit) || errors.Is(err, snappy.ErrTooLarge), err)
			assert.Less(t, allocated, uint64(4<<20))
		})
	}
}

// Not parallel, see allocatedBytes().
//
//nolint:paralleltest
func TestOversizeNestedMessage(t *testing.T) {
	size := uint32(1 << 30)

	demo := handmadeDemo(fileHeaderFrame(t), packetFrame(t, 0, nestedMessage{t: int32(msg.NET_Messages_net_Tick), data: []byte{1, 2, 3}, size: &size}), stopFrame())

	var err error

	allocated := allocatedBytes(func() { _, err = parse(t, demo) })

	assertDecodeError(t, err, "size 1073741824 of net-message of type 4 exceeds the remaining 3 bytes of the packet")
	assert.Less(t, allocated, uint64(4<<20))
}

func TestCorruptCompressedPacket(t *testing.T) {
	t.Parallel()

	demo := handmadeDemo(fileHeaderFrame(t), frame(msg.EDemoCommands_DEM_Packet|msg.EDemoCommands_DEM_IsCompressed, 0, []byte{10, 0xff, 0xff, 0xff}), stopFrame())

	_, err := parse(t, demo)
	assertDecodeError(t, err, "snappy")
	assert.ErrorIs(t, err, snappy.ErrCorrupt)
}

func TestCorruptProtobufPacket(t *testing.T) {
	t.Parallel()

	demo := handmadeDemo(fileHeaderFrame(t), frame(msg.EDemoCommands_DEM_Packet, 0, []byte{0x1a, 0xff}), stopFrame())

	_, err := parse(t, demo)
	assertDecodeError(t, err, "proto")
}

func TestCorruptNestedProtobuf(t *testing.T) {
	t.Parallel()

	demo := handmadeDemo(fileHeaderFrame(t), packetFrame(t, 0, nestedMessage{t: int32(msg.NET_Messages_net_Tick), data: []byte{0x0a, 0xff}}), stopFrame())

	_, err := parse(t, demo)
	assertDecodeError(t, err, "failed to unmarshal net-message of type 4")
}

// goroutineID returns the ID of the calling goroutine.
func goroutineID() uint64 {
	var buf [64]byte

	s := strings.TrimPrefix(string(buf[:runtime.Stack(buf[:], false)]), "goroutine ")
	id, _ := strconv.ParseUint(s[:strings.IndexByte(s, ' ')], 10, 64)

	return id
}

// brokenFullPacketDemos returns a demo with a broken DEM_FullPacket for each variant,
// and the same demo with a valid one.
func brokenFullPacketDemos(t testing.TB) (valid []byte, broken map[string][]byte) {
	t.Helper()

	demo := func(fullPacket []byte) []byte {
		return handmadeDemo(
			fileHeaderFrame(t),
			packetFrame(t, 1, netTick(t, 1)),
			fullPacket,
			packetFrame(t, 3, netTick(t, 3)),
			stopFrame(),
		)
	}

	fp := fullPacketFrame(t, 2, netTick(t, 2))
	fpPayload := fp[len(frameHeader(msg.EDemoCommands_DEM_FullPacket, 2, 0)):] // the size has one byte

	return demo(fp), map[string][]byte{
		"snappy":           demo(frame(msg.EDemoCommands_DEM_FullPacket|msg.EDemoCommands_DEM_IsCompressed, 2, []byte{10, 0xff, 0xff, 0xff})),
		"snappy limit":     demo(frame(msg.EDemoCommands_DEM_FullPacket|msg.EDemoCommands_DEM_IsCompressed, 2, binary.AppendUvarint(nil, maxFramePayload+1))),
		"proto":            demo(frame(msg.EDemoCommands_DEM_FullPacket, 2, fpPayload[:len(fpPayload)-1])),
		"nested size":      demo(protoFrame(t, msg.EDemoCommands_DEM_FullPacket, 2, &msg.CDemoFullPacket{Packet: &msg.CDemoPacket{Data: corruptPacketData}})),
		"nested proto":     demo(fullPacketFrame(t, 2, netTick(t, 2), nestedMessage{t: int32(msg.NET_Messages_net_Tick), data: []byte{0x0a, 0xff}})),
		"compressed valid": demo(compressedFrame(msg.EDemoCommands_DEM_FullPacket, 2, fpPayload)), // not broken, see below
	}
}

type parseRecord struct {
	err             error
	frames          int
	currentFrame    int
	ticks           []int // ingame tick at each FrameDone
	warns           []events.ParserWarn
	warnFrames      []int // CurrentFrame() when the warnings were dispatched
	sameGoroutine   bool  // whether the warnings were dispatched on the goroutine of the FrameDone handlers
	fullPacketsSeen int
}

func parseRecorded(t *testing.T, demo []byte) parseRecord {
	t.Helper()

	p := NewParser(bytes.NewReader(demo))
	defer p.Close()

	var (
		rec       = parseRecord{sameGoroutine: true}
		handlerID uint64
	)

	p.RegisterEventHandler(func(events.FrameDone) {
		handlerID = goroutineID()
		rec.frames++
		rec.ticks = append(rec.ticks, p.GameState().IngameTick())
	})

	p.RegisterEventHandler(func(w events.ParserWarn) {
		if handlerID != 0 && goroutineID() != handlerID {
			rec.sameGoroutine = false
		}

		rec.warns = append(rec.warns, w)
		rec.warnFrames = append(rec.warnFrames, p.CurrentFrame())
	})

	p.RegisterNetMessageHandler(func(*msg.CDemoFullPacket) { rec.fullPacketsSeen++ })

	rec.err = p.ParseToEnd(t.Context())
	rec.currentFrame = p.CurrentFrame()

	return rec
}

// Broken DEM_FullPacket frames are skipped with a warning (#409), the frame is still counted.
func TestBrokenFullPacket(t *testing.T) {
	t.Parallel()

	valid, broken := brokenFullPacketDemos(t)

	want := parseRecorded(t, valid)
	require.NoError(t, want.err)
	require.Empty(t, want.warns)
	require.Equal(t, 1, want.fullPacketsSeen)
	require.Equal(t, 5, want.frames)

	for name, demo := range broken {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := parseRecorded(t, demo)
			require.NoError(t, got.err)
			assert.Equal(t, want.frames, got.frames, "FrameDone count")
			assert.Equal(t, want.currentFrame, got.currentFrame, "CurrentFrame()")
			assert.Equal(t, want.ticks, got.ticks, "ticks")

			if name == "compressed valid" {
				assert.Empty(t, got.warns)
				assert.Equal(t, 1, got.fullPacketsSeen)

				return
			}

			assert.Zero(t, got.fullPacketsSeen, "the broken full packet must not be dispatched")

			require.Len(t, got.warns, 1)
			assert.EqualValues(t, events.WarnTypeBrokenFullPacket, got.warns[0].Type)
			assert.True(t, got.sameGoroutine, "the warning must be dispatched on the message-handling goroutine")
			assert.Equal(t, []int{2}, got.warnFrames, "the warning must be dispatched in order")

			var de *DecodeError
			if assert.ErrorAs(t, got.warns[0].Err, &de) {
				assert.Equal(t, 2, de.Frame)
			}
		})
	}
}

// The handlers of the warning and of other events can share state without synchronization (run with -race).
func TestBrokenFullPacket_Race(t *testing.T) {
	t.Parallel()

	_, broken := brokenFullPacketDemos(t)

	frames := make([]byte, 0, 50*len(broken["snappy"]))

	for range 50 {
		frames = append(frames, broken["snappy"][16:len(broken["snappy"])-len(stopFrame())]...)
	}

	p := NewParser(bytes.NewReader(handmadeDemo(frames, stopFrame())), option.MsgQueueBufferSize(1000))
	defer p.Close()

	var shared []string

	p.RegisterEventHandler(func(events.ParserWarn) { shared = append(shared, "warn") })
	p.RegisterEventHandler(func(events.FrameDone) { shared = append(shared, "frame") })

	require.NoError(t, p.ParseToEnd(t.Context()))
	assert.Len(t, shared, 50*5+1)
}

// A string table with huge sizes fails without allocating them.
// Not parallel, see allocatedBytes().
//
//nolint:paralleltest
func TestStringTableHugeSizes(t *testing.T) {
	t.Run("compressed", func(t *testing.T) {
		data := binary.AppendUvarint(nil, 1<<30)
		data = append(data, make([]byte, 100)...)

		demo := handmadeDemo(fileHeaderFrame(t), packetFrame(t, 0, nested(t, int32(msg.SVC_Messages_svc_CreateStringTable), &msg.CSVCMsg_CreateStringTable{
			Name:             proto.String(stNameUserInfo),
			NumEntries:       proto.Int32(1 << 30),
			DataCompressed:   proto.Bool(true),
			UncompressedSize: proto.Int32(1 << 30),
			StringData:       data,
		})), stopFrame())

		var err error

		allocated := allocatedBytes(func() { _, err = parse(t, demo) })

		require.ErrorIs(t, err, ErrDecodeFailed)
		assert.ErrorIs(t, err, errSnappyLimit)
		assert.Less(t, allocated, uint64(4<<20))
	})

	for _, tc := range []struct {
		name            string
		userDataFixed   bool
		userDataSize    int32
		variantBitCount bool
		entry           func(w *stringTableWriter)
	}{
		{"item size", false, 0, false, func(w *stringTableWriter) {
			w.bits(1<<17-1, 17) // 128 KB
		}},
		{"variant item size", false, 0, true, func(w *stringTableWriter) {
			w.uBitInt(1 << 27) // 128 MB
		}},
		{"negative fixed size", true, -1, false, func(*stringTableWriter) {}},
		{"huge fixed size", true, 1 << 30, false, func(*stringTableWriter) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newParser()

			var warns []events.ParserWarn
			p.RegisterEventHandler(func(e events.ParserWarn) { warns = append(warns, e) })

			w := new(stringTableWriter)
			w.bit(true)  // increment
			w.bit(false) // no key
			w.bit(true)  // value
			tc.entry(w)
			w.bytes([]byte{1, 2, 3})

			var items []*stringTableItem

			allocated := allocatedBytes(func() {
				items = p.parseStringTable(w.buf, 1<<30, stNameUserInfo, tc.userDataFixed, tc.userDataSize, 0, tc.variantBitCount)
			})

			assert.Empty(t, items)
			// not exact, the runtime accounts some allocations in batches
			assert.Less(t, allocated, uint64(1<<20))

			if assert.Len(t, warns, 1) {
				assert.EqualValues(t, events.WarnTypeStringTableParsingFailure, warns[0].Type)
				assert.ErrorIs(t, warns[0].Err, ErrDecodeFailed)
			}
		})
	}

	// many items that aren't used don't take up memory
	t.Run("item count", func(t *testing.T) {
		p := newParser()

		buf := bytes.Repeat([]byte{0x49, 0x92, 0x24}, 1<<16) // increment, no key, no value (3 bits each, 8 per 3 bytes)

		var items []*stringTableItem

		allocated := allocatedBytes(func() {
			items = p.parseStringTable(buf, 1<<30, stNameUserInfo, false, 0, 0, false)
		})

		assert.Empty(t, items)
		assert.Less(t, allocated, uint64(1<<20), "~500k items would take up ~30 MB")
	})
}

func TestNegativeStringTableID(t *testing.T) {
	t.Parallel()

	demo := handmadeDemo(fileHeaderFrame(t), packetFrame(t, 0, nested(t, int32(msg.SVC_Messages_svc_UpdateStringTable), &msg.CSVCMsg_UpdateStringTable{
		TableId: proto.Int32(-1),
	})), stopFrame())

	_, err := parse(t, demo)
	assertDecodeError(t, err, "invalid string table ID -1")
}

func TestFileInfoWithoutFields(t *testing.T) {
	t.Parallel()

	demo := handmadeDemo(fileHeaderFrame(t), protoFrame(t, msg.EDemoCommands_DEM_FileInfo, 0, &msg.CDemoFileInfo{}), stopFrame())

	_, err := parse(t, demo)
	require.NoError(t, err)
}

func TestUnknownDemoCommand_Skipped(t *testing.T) {
	t.Parallel()

	demo := handmadeDemo(fileHeaderFrame(t), frame(50, 0, make([]byte, 100)), stopFrame())

	frames, err := parse(t, demo)
	require.NoError(t, err)
	assert.Equal(t, 2, frames) // the unknown frame isn't counted
}

// fuzzSeeds returns handmade demos for FuzzParse.
func fuzzSeeds(t testing.TB) [][]byte {
	t.Helper()

	valid, broken := brokenFullPacketDemos(t)

	seeds := make([][]byte, 0, 6+len(broken))
	seeds = append(seeds,
		valid,
		handmadeDemo(fileHeaderFrame(t), stopFrame()),
		handmadeDemo(fileHeaderFrame(t), compressedFrame(msg.EDemoCommands_DEM_Packet, 1, mustMarshal(t, &msg.CDemoPacket{Data: packetData(netTick(t, 1))})), stopFrame()),
		handmadeDemo(fileHeaderFrame(t), packetFrame(t, 0, nested(t, int32(msg.SVC_Messages_svc_CreateStringTable), &msg.CSVCMsg_CreateStringTable{
			Name:       proto.String(stNameUserInfo),
			NumEntries: proto.Int32(1),
			StringData: stringTableData(stringTableTestEntry{key: "0", value: mustMarshal(t, &msg.CMsgPlayerInfo{Name: proto.String("x"), Xuid: proto.Uint64(1)})}),
		})), stopFrame()),
		handmadeDemo(fileHeaderFrame(t), demoPacketFrame(t, corruptPacketData), stopFrame()),
		handmadeDemo(fileHeaderFrame(t), frame(50, 0, []byte{1, 2, 3}), stopFrame()),
	)

	for _, demo := range broken {
		seeds = append(seeds, demo)
	}

	return seeds
}

func mustMarshal(t testing.TB, m proto.Message) []byte {
	t.Helper()

	b, err := proto.Marshal(m)
	require.NoError(t, err)

	return b
}

func stringTableData(entries ...stringTableTestEntry) []byte {
	w := new(stringTableWriter)

	for _, e := range entries {
		w.entry(e)
	}

	return append(w.buf, make([]byte, 16)...) // the bit reader reads ahead
}

// fuzzAllocBudget is the number of bytes parsing input may allocate (see allocatedBytes()):
// the per-message limits, plus a bounded amount per input byte.
func fuzzAllocBudget(input []byte) uint64 {
	return 64<<20 + 1024*uint64(len(input))
}

// fuzzTimeout is how long parsing an input may take before the watchdog of checkParse() fails.
const fuzzTimeout = 5 * time.Second

// startWatchdog panics (crashing the fuzz worker, so the fuzzer records the input) with the stacks
// of all goroutines if stop isn't called within timeout: the fuzzer doesn't detect hangs itself.
func startWatchdog(name string, timeout time.Duration) (stop func()) {
	done := make(chan struct{})

	go func() {
		select {
		case <-done:
		case <-time.After(timeout):
			buf := make([]byte, 1<<22)
			buf = buf[:runtime.Stack(buf, true)]

			panic(fmt.Sprintf("%s didn't finish within %v (hang?), goroutines:\n%s", name, timeout, buf))
		}
	}()

	return func() { close(done) }
}

// checkParse parses data, it fails if parsing fails with an unexpected error, a panic, takes too long
// or allocates too much memory.
// It must not run in parallel with other tests, see allocatedBytes().
func checkParse(t *testing.T, data []byte) {
	t.Helper()

	stop := startWatchdog("parsing", fuzzTimeout)
	defer stop()

	var err error

	allocated := allocatedBytes(func() {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// A buffer size of 0 runs the message-handling goroutine in lockstep with the parsing goroutine:
		// still two goroutines, but frames don't pile up in the queue.
		p := NewParser(bytes.NewReader(data), option.MsgQueueBufferSize(0))
		defer p.Close()

		registerFuzzHandlers(p)

		var pe *PanicError

		func() {
			defer func() {
				if r := recover(); r != nil {
					if !errors.As(asError(r), &pe) {
						panic(r)
					}
				}
			}()

			err = p.ParseToEnd(ctx)
		}()

		if pe != nil {
			t.Fatalf("unexpected panic: %v", pe)
		}
	})

	if err != nil && !errors.Is(err, ErrUnexpectedEndOfDemo) && !errors.Is(err, ErrDecodeFailed) && !errors.Is(err, ErrInvalidFileType) {
		t.Fatalf("unexpected error: %v", err)
	}

	if budget := fuzzAllocBudget(data); allocated > budget {
		t.Fatalf("allocated %d bytes for %d input bytes, more than the budget of %d bytes (err: %v)", allocated, len(data), budget, err)
	}
}

func asError(r any) error {
	if err, ok := r.(error); ok {
		return err
	}

	return nil
}

// registerFuzzHandlers registers handlers for everything, so all events and net-messages are created and dispatched.
// They mustn't fail (e.g. with require), that would be a handler panic.
func registerFuzzHandlers(p *Parser) {
	p.RegisterEventHandler(func(any) {})
	p.RegisterNetMessageHandler(func(any) {})

	// Entity update handlers, which only run if a property is bound.
	p.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, sc := range p.ServerClasses().All() {
			sc.OnEntityCreated(func(e *st.Entity) {
				for _, prop := range e.Properties() {
					prop.OnUpdate(func(st.PropertyValue) {})

					break
				}
			})
		}
	})
}

// Seeds run as normal tests (also with -short), fuzzing runs locally:
//
//	go test -tags unassert_panic -run '^$' -fuzz '^FuzzParse$' -fuzztime 15m ./pkg/demoinfocs
//
// Not parallel, see allocatedBytes().
func FuzzParse(f *testing.F) {
	for _, seed := range fuzzSeeds(f) {
		f.Add(seed)
	}

	f.Fuzz(checkParse)
}
