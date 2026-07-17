package demoinfocs

import (
	"testing"

	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

// Field numbers used to hand-build delta blobs (see usercmd.proto).
const (
	fieldCSGOUserCmdBase = 1

	fieldBaseUserCmdClientTick = 2
	fieldBaseUserCmdButtonsPb  = 3

	fieldButtonsState1 = 1
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
