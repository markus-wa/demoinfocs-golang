package demoinfocs

import (
	"testing"

	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// Field numbers used to hand-build delta blobs (see usercmd.proto).
const (
	fieldCSGOUserCmdBase = 1

	fieldBaseUserCmdClientTick = 2
	fieldBaseUserCmdButtonsPb  = 3
	fieldBaseUserCmdSubtick    = 18
	fieldCSGOUserCmdHistory    = 2

	fieldButtonsState1  = 1
	fieldSubtickButton  = 1
	fieldSubtickPressed = 2
	fieldHistoryFrame   = 64
)

func baseline() *msg.CSGOUserCmdPB {
	return &msg.CSGOUserCmdPB{
		Base: &msg.CBaseUserCmdPB{
			ClientTick:  proto32(100),
			Forwardmove: protoFloat(250),
			ButtonsPb: &msg.CInButtonStatePB{
				Buttonstate1: proto64(512),
				Buttonstate2: proto64(3),
			},
		},
	}
}

// TestApplyUserCmdDelta_ReplaceScalar: a delta that sets a single scalar
// replaces it while all other baseline fields are inherited.
func TestApplyUserCmdDelta_ReplaceScalar(t *testing.T) {
	m := baseline()

	// base { client_tick = 101 }
	inner := varintField(fieldBaseUserCmdClientTick, 101)
	delta := bytesField(fieldCSGOUserCmdBase, inner)

	require.NoError(t, applyUserCmdDelta(m, delta))

	assert.EqualValues(t, 101, m.GetBase().GetClientTick(), "client_tick replaced")
	assert.EqualValues(t, 512, m.GetBase().GetButtonsPb().GetButtonstate1(), "buttons inherited from baseline")
	assert.EqualValues(t, 250, m.GetBase().GetForwardmove(), "forwardmove inherited from baseline")
}

// TestApplyUserCmdDelta_MergeNested: a delta into a nested message replaces only
// the touched nested field, keeping the rest of the nested message.
func TestApplyUserCmdDelta_MergeNested(t *testing.T) {
	m := baseline()

	// base { buttons_pb { buttonstate1 = 0 } }
	buttons := varintField(fieldButtonsState1, 0)
	base := bytesField(fieldBaseUserCmdButtonsPb, buttons)
	delta := bytesField(fieldCSGOUserCmdBase, base)

	require.NoError(t, applyUserCmdDelta(m, delta))

	assert.EqualValues(t, 0, m.GetBase().GetButtonsPb().GetButtonstate1(), "buttonstate1 replaced with explicit 0")
	assert.EqualValues(t, 3, m.GetBase().GetButtonsPb().GetButtonstate2(), "buttonstate2 inherited")
	assert.EqualValues(t, 100, m.GetBase().GetClientTick(), "client_tick inherited")
}

// TestApplyUserCmdDelta_ResetScalar: a wire-type-7 marker resets a scalar to its
// protobuf default.
func TestApplyUserCmdDelta_ResetScalar(t *testing.T) {
	m := baseline()

	// base { <reset buttons_pb.buttonstate1> }
	buttons := resetMarker(fieldButtonsState1)
	base := bytesField(fieldBaseUserCmdButtonsPb, buttons)
	delta := bytesField(fieldCSGOUserCmdBase, base)

	require.NoError(t, applyUserCmdDelta(m, delta))

	assert.EqualValues(t, 0, m.GetBase().GetButtonsPb().GetButtonstate1(), "buttonstate1 reset to default")
	assert.EqualValues(t, 3, m.GetBase().GetButtonsPb().GetButtonstate2(), "buttonstate2 untouched")
}

// TestApplyUserCmdDelta_ResetNestedMessage: a wire-type-7 marker on a message
// field clears the whole sub-message (zeroing all its fields).
func TestApplyUserCmdDelta_ResetNestedMessage(t *testing.T) {
	m := baseline()

	// base { <reset buttons_pb> }
	base := resetMarker(fieldBaseUserCmdButtonsPb)
	delta := bytesField(fieldCSGOUserCmdBase, base)

	require.NoError(t, applyUserCmdDelta(m, delta))

	assert.EqualValues(t, 0, m.GetBase().GetButtonsPb().GetButtonstate1(), "buttonstate1 zeroed")
	assert.EqualValues(t, 0, m.GetBase().GetButtonsPb().GetButtonstate2(), "buttonstate2 zeroed")
	assert.EqualValues(t, 100, m.GetBase().GetClientTick(), "client_tick untouched")
}

// TestApplyUserCmdDelta_ChainedDeltas: consecutive deltas accumulate onto the
// same baseline like the real per-slot stream.
func TestApplyUserCmdDelta_ChainedDeltas(t *testing.T) {
	m := baseline()

	// delta 1: buttonstate1 -> 0
	d1 := bytesField(fieldCSGOUserCmdBase,
		bytesField(fieldBaseUserCmdButtonsPb,
			varintField(fieldButtonsState1, 0)))
	require.NoError(t, applyUserCmdDelta(m, d1))
	assert.EqualValues(t, 0, m.GetBase().GetButtonsPb().GetButtonstate1())

	// delta 2: buttonstate1 -> 512 again
	d2 := bytesField(fieldCSGOUserCmdBase,
		bytesField(fieldBaseUserCmdButtonsPb,
			varintField(fieldButtonsState1, 512)))
	require.NoError(t, applyUserCmdDelta(m, d2))
	assert.EqualValues(t, 512, m.GetBase().GetButtonsPb().GetButtonstate1())
	assert.EqualValues(t, 3, m.GetBase().GetButtonsPb().GetButtonstate2(), "buttonstate2 survived both deltas")
}

func subtick(button uint64, pressed bool) *msg.CSubtickMoveStep {
	return &msg.CSubtickMoveStep{Button: proto64(button), Pressed: &pressed}
}

func repeatedPatch(index protowire.Number, delta []byte) []byte {
	return bytesField(index, delta)
}

func TestApplyUserCmdDelta_RepeatedIndexPatch(t *testing.T) {
	m := &msg.CSGOUserCmdPB{Base: &msg.CBaseUserCmdPB{
		SubtickMoves: []*msg.CSubtickMoveStep{subtick(1, true), subtick(2, true)},
	}}

	// subtick_moves { index 1 { pressed = false } }
	var elementDelta []byte
	elementDelta = varintField(fieldSubtickPressed, 0)
	operation := repeatedPatch(1, elementDelta)
	delta := bytesField(fieldCSGOUserCmdBase, bytesField(fieldBaseUserCmdSubtick, operation))

	require.NoError(t, applyUserCmdDelta(m, delta))
	assert.Len(t, m.GetBase().GetSubtickMoves(), 2)
	assert.True(t, m.GetBase().GetSubtickMoves()[0].GetPressed())
	assert.EqualValues(t, 1, m.GetBase().GetSubtickMoves()[0].GetButton())
	assert.False(t, m.GetBase().GetSubtickMoves()[1].GetPressed())
	assert.EqualValues(t, 2, m.GetBase().GetSubtickMoves()[1].GetButton())
}

func TestApplyUserCmdDelta_RepeatedIndexZeroAndTruncate(t *testing.T) {
	m := &msg.CSGOUserCmdPB{Base: &msg.CBaseUserCmdPB{
		SubtickMoves: []*msg.CSubtickMoveStep{subtick(1, true), subtick(2, true), subtick(3, true)},
	}}

	// Index zero is encoded as field number zero in the repeated operation.
	indexZeroPatch := repeatedPatch(0, varintField(fieldSubtickButton, 9))
	require.NoError(t, applyUserCmdDelta(m, bytesField(fieldCSGOUserCmdBase, bytesField(fieldBaseUserCmdSubtick, indexZeroPatch))))
	assert.EqualValues(t, 9, m.GetBase().GetSubtickMoves()[0].GetButton())

	// A wire-7 repeated operation sets the resulting list length.
	truncate := protowire.AppendTag(nil, 2, wireTypeReset)
	require.NoError(t, applyUserCmdDelta(m, bytesField(fieldCSGOUserCmdBase, bytesField(fieldBaseUserCmdSubtick, truncate))))
	assert.Len(t, m.GetBase().GetSubtickMoves(), 2)
}

func TestApplyUserCmdDelta_RepeatedInputHistory(t *testing.T) {
	m := &msg.CSGOUserCmdPB{InputHistory: []*msg.CSGOInputHistoryEntryPB{
		{FrameNumber: proto32(10)},
	}}

	operation := repeatedPatch(0, varintField(fieldHistoryFrame, 11))
	delta := bytesField(fieldCSGOUserCmdHistory, operation)
	require.NoError(t, applyUserCmdDelta(m, delta))
	assert.EqualValues(t, 11, m.GetInputHistory()[0].GetFrameNumber())
}

func TestUserCmdRingRequiresExactCommandNumber(t *testing.T) {
	var ring userCmdRing
	first := baseline()
	ring.insert(1, first)
	assert.NotNil(t, ring.entries[userCmdRingIndex(1)].command)
	if _, ok := ring.get(151); ok {
		t.Fatal("command-number ring collision was treated as an exact baseline")
	}
	if got, ok := ring.slotCommandNumber(151); !ok || got != 1 {
		t.Fatalf("ring slot mismatch = (%d, %v), want (1, true)", got, ok)
	}

	ring.insert(151, baseline())
	if _, ok := ring.get(1); ok {
		t.Fatal("overwritten command-number ring entry remained addressable")
	}
	if _, ok := ring.get(151); !ok {
		t.Fatal("new command-number ring entry was not addressable")
	}
}

func TestServerUserCmdDeltaDataRoundTrip(t *testing.T) {
	want := &msg.CMsgServerUserCmd{DeltaData: []byte{0x12, 0x01, 0x00}}
	encoded, err := proto.Marshal(want)
	require.NoError(t, err)

	got := &msg.CMsgServerUserCmd{}
	require.NoError(t, proto.Unmarshal(encoded, got))
	assert.Equal(t, want.GetDeltaData(), got.GetDeltaData())
}

func proto32(v int32) *int32        { return &v }
func proto64(v uint64) *uint64      { return &v }
func protoFloat(v float32) *float32 { return &v }

func varintField(field protowire.Number, v uint64) []byte {
	b := protowire.AppendTag(nil, field, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

func bytesField(field protowire.Number, v []byte) []byte {
	b := protowire.AppendTag(nil, field, protowire.BytesType)
	return protowire.AppendBytes(b, v)
}

func resetMarker(field protowire.Number) []byte {
	return protowire.AppendTag(nil, field, wireTypeReset)
}
