package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

// Field numbers used to hand-build delta blobs (see usercmd.proto).
const (
	fieldCSGOUserCmdBase = 1

	fieldBaseUserCmdClientTick = 2
	fieldBaseUserCmdButtonsPb  = 3

	fieldButtonsState1 = 1
)

func baseline() *msgs2.CSGOUserCmdPB {
	return &msgs2.CSGOUserCmdPB{
		Base: &msgs2.CBaseUserCmdPB{
			ClientTick:  proto32(100),
			Forwardmove: protoFloat(250),
			ButtonsPb: &msgs2.CInButtonStatePB{
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
	inner := appendVarintField(nil, fieldBaseUserCmdClientTick, 101)
	delta := appendBytesField(nil, fieldCSGOUserCmdBase, inner)

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
	buttons := appendVarintField(nil, fieldButtonsState1, 0)
	base := appendBytesField(nil, fieldBaseUserCmdButtonsPb, buttons)
	delta := appendBytesField(nil, fieldCSGOUserCmdBase, base)

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
	buttons := appendResetMarker(nil, fieldButtonsState1)
	base := appendBytesField(nil, fieldBaseUserCmdButtonsPb, buttons)
	delta := appendBytesField(nil, fieldCSGOUserCmdBase, base)

	require.NoError(t, applyUserCmdDelta(m, delta))

	assert.EqualValues(t, 0, m.GetBase().GetButtonsPb().GetButtonstate1(), "buttonstate1 reset to default")
	assert.EqualValues(t, 3, m.GetBase().GetButtonsPb().GetButtonstate2(), "buttonstate2 untouched")
}

// TestApplyUserCmdDelta_ResetNestedMessage: a wire-type-7 marker on a message
// field clears the whole sub-message (zeroing all its fields).
func TestApplyUserCmdDelta_ResetNestedMessage(t *testing.T) {
	m := baseline()

	// base { <reset buttons_pb> }
	base := appendResetMarker(nil, fieldBaseUserCmdButtonsPb)
	delta := appendBytesField(nil, fieldCSGOUserCmdBase, base)

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
	d1 := appendBytesField(nil, fieldCSGOUserCmdBase,
		appendBytesField(nil, fieldBaseUserCmdButtonsPb,
			appendVarintField(nil, fieldButtonsState1, 0)))
	require.NoError(t, applyUserCmdDelta(m, d1))
	assert.EqualValues(t, 0, m.GetBase().GetButtonsPb().GetButtonstate1())

	// delta 2: buttonstate1 -> 512 again
	d2 := appendBytesField(nil, fieldCSGOUserCmdBase,
		appendBytesField(nil, fieldBaseUserCmdButtonsPb,
			appendVarintField(nil, fieldButtonsState1, 512)))
	require.NoError(t, applyUserCmdDelta(m, d2))
	assert.EqualValues(t, 512, m.GetBase().GetButtonsPb().GetButtonstate1())
	assert.EqualValues(t, 3, m.GetBase().GetButtonsPb().GetButtonstate2(), "buttonstate2 survived both deltas")
}

// Field numbers for the repeated grammar (see cs_usercmd.proto / usercmd.proto).
const (
	fieldCSGOUserCmdInputHistory = 2
	fieldBaseUserCmdSubtickMoves = 18

	fieldInputHistoryRenderTickCount = 4
)

// TestApplyUserCmdDelta_RepeatedCountHeader: a repeated delta with a count
// header (wire 7, field number = new count) resizes the list and merges the
// sparse per-index entries; unlisted indices inherit the baseline element.
func TestApplyUserCmdDelta_RepeatedCountHeader(t *testing.T) {
	m := baseline()
	m.InputHistory = []*msgs2.CSGOInputHistoryEntryPB{
		{RenderTickCount: proto32(10)},
		{RenderTickCount: proto32(20)},
	}

	// input_history: <count=3 header> idx1{render_tick_count=99}
	// -> length 3, idx0 inherited (10), idx1 replaced (99), idx2 new (default).
	repeated := appendCountHeader(nil, 3)
	repeated = appendIndexEntry(repeated, 1, appendVarintField(nil, fieldInputHistoryRenderTickCount, 99))
	delta := appendBytesField(nil, fieldCSGOUserCmdInputHistory, repeated)

	require.NoError(t, applyUserCmdDelta(m, delta))

	require.Len(t, m.GetInputHistory(), 3)
	assert.EqualValues(t, 10, m.GetInputHistory()[0].GetRenderTickCount(), "idx0 inherited")
	assert.EqualValues(t, 99, m.GetInputHistory()[1].GetRenderTickCount(), "idx1 replaced")
	assert.EqualValues(t, 0, m.GetInputHistory()[2].GetRenderTickCount(), "idx2 new/default")
}

// TestApplyUserCmdDelta_RepeatedNoHeader: without a count header the list keeps
// its baseline length and only the listed indices are merged.
func TestApplyUserCmdDelta_RepeatedNoHeader(t *testing.T) {
	m := baseline()
	m.InputHistory = []*msgs2.CSGOInputHistoryEntryPB{
		{RenderTickCount: proto32(10)},
		{RenderTickCount: proto32(20)},
	}

	// input_history: idx0{render_tick_count=11} (no header, count unchanged = 2)
	repeated := appendIndexEntry(nil, 0, appendVarintField(nil, fieldInputHistoryRenderTickCount, 11))
	delta := appendBytesField(nil, fieldCSGOUserCmdInputHistory, repeated)

	require.NoError(t, applyUserCmdDelta(m, delta))

	require.Len(t, m.GetInputHistory(), 2, "length unchanged")
	assert.EqualValues(t, 11, m.GetInputHistory()[0].GetRenderTickCount(), "idx0 merged")
	assert.EqualValues(t, 20, m.GetInputHistory()[1].GetRenderTickCount(), "idx1 inherited")
}

// TestApplyUserCmdDelta_RepeatedShrink: a smaller count header truncates the
// list.
func TestApplyUserCmdDelta_RepeatedShrink(t *testing.T) {
	m := baseline()
	m.Base.SubtickMoves = []*msgs2.CSubtickMoveStep{
		{Button: proto64(1), Pressed: protoBool(true)},
		{Button: proto64(2), Pressed: protoBool(false)},
		{Button: proto64(4), Pressed: protoBool(true)},
	}

	// base { subtick_moves: <count=1 header> } -> keep only idx0.
	subtick := appendCountHeader(nil, 1)
	base := appendBytesField(nil, fieldBaseUserCmdSubtickMoves, subtick)
	delta := appendBytesField(nil, fieldCSGOUserCmdBase, base)

	require.NoError(t, applyUserCmdDelta(m, delta))

	require.Len(t, m.GetBase().GetSubtickMoves(), 1)
	assert.EqualValues(t, 1, m.GetBase().GetSubtickMoves()[0].GetButton(), "idx0 kept")
	assert.True(t, m.GetBase().GetSubtickMoves()[0].GetPressed())
}

func proto32(v int32) *int32        { return &v }
func proto64(v uint64) *uint64      { return &v }
func protoFloat(v float32) *float32 { return &v }
func protoBool(v bool) *bool        { return &v }

func appendCountHeader(b []byte, count protowire.Number) []byte {
	return protowire.AppendTag(b, count, wireTypeReset)
}

func appendIndexEntry(b []byte, index protowire.Number, payload []byte) []byte {
	b = protowire.AppendTag(b, index, protowire.BytesType)
	return protowire.AppendBytes(b, payload)
}

func appendVarintField(b []byte, field protowire.Number, v uint64) []byte {
	b = protowire.AppendTag(b, field, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

func appendBytesField(b []byte, field protowire.Number, v []byte) []byte {
	b = protowire.AppendTag(b, field, protowire.BytesType)
	return protowire.AppendBytes(b, v)
}

func appendResetMarker(b []byte, field protowire.Number) []byte {
	return protowire.AppendTag(b, field, wireTypeReset)
}
