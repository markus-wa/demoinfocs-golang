package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

// Field numbers used to hand-build snapshot/delta blobs (see usercmd.proto).
const (
	fieldCSGOUserCmdBase = 1

	fieldBaseUserCmdClientTick = 2
	fieldBaseUserCmdButtonsPb  = 3

	fieldButtonsState1 = 1
	fieldButtonsState2 = 2
)

// baselineButtons builds a full CSGOUserCmdPB snapshot on the wire equivalent to:
//
//	base { client_tick: 100  buttons_pb { buttonstate1: 512  buttonstate2: 3 } }
func baselineButtons() []byte {
	buttons := append(
		varintField(fieldButtonsState1, 512),
		varintField(fieldButtonsState2, 3)...)

	base := append(
		varintField(fieldBaseUserCmdClientTick, 100),
		bytesField(fieldBaseUserCmdButtonsPb, buttons)...)

	return bytesField(fieldCSGOUserCmdBase, base)
}

func mergeButtons(t *testing.T, buttons *uint64, data []byte) {
	t.Helper()
	require.NoError(t, mergeUserCmdButtons(data, 0, buttons))
}

// TestMergeUserCmdButtons_FullSnapshot: a full snapshot yields buttonstate1,
// with the surrounding fields skipped over rather than decoded.
func TestMergeUserCmdButtons_FullSnapshot(t *testing.T) {
	var buttons uint64

	mergeButtons(t, &buttons, baselineButtons())

	assert.EqualValues(t, 512, buttons)
}

// TestMergeUserCmdButtons_UnrelatedFieldsInherit: a delta that only touches
// other fields leaves the accumulated button state untouched.
func TestMergeUserCmdButtons_UnrelatedFieldsInherit(t *testing.T) {
	var buttons uint64

	mergeButtons(t, &buttons, baselineButtons())

	// base { client_tick = 101 }
	mergeButtons(t, &buttons, bytesField(fieldCSGOUserCmdBase, varintField(fieldBaseUserCmdClientTick, 101)))

	assert.EqualValues(t, 512, buttons, "buttons inherited from baseline")
}

// TestMergeUserCmdButtons_Replace: a delta setting buttonstate1 replaces it,
// including with an explicit 0.
func TestMergeUserCmdButtons_Replace(t *testing.T) {
	var buttons uint64

	mergeButtons(t, &buttons, baselineButtons())

	// base { buttons_pb { buttonstate1 = 0 } }
	mergeButtons(t, &buttons, bytesField(fieldCSGOUserCmdBase,
		bytesField(fieldBaseUserCmdButtonsPb,
			varintField(fieldButtonsState1, 0))))

	assert.EqualValues(t, 0, buttons, "buttonstate1 replaced with explicit 0")
}

// TestMergeUserCmdButtons_SiblingFieldIgnored: a delta touching only
// buttonstate2 does not disturb buttonstate1.
func TestMergeUserCmdButtons_SiblingFieldIgnored(t *testing.T) {
	var buttons uint64

	mergeButtons(t, &buttons, baselineButtons())

	// base { buttons_pb { buttonstate2 = 7 } }
	mergeButtons(t, &buttons, bytesField(fieldCSGOUserCmdBase,
		bytesField(fieldBaseUserCmdButtonsPb,
			varintField(fieldButtonsState2, 7))))

	assert.EqualValues(t, 512, buttons)
}

// TestMergeUserCmdButtons_ResetScalar: a wire-type-7 marker on buttonstate1
// resets it to its protobuf default.
func TestMergeUserCmdButtons_ResetScalar(t *testing.T) {
	var buttons uint64

	mergeButtons(t, &buttons, baselineButtons())

	// base { buttons_pb { <reset buttonstate1> } }
	mergeButtons(t, &buttons, bytesField(fieldCSGOUserCmdBase,
		bytesField(fieldBaseUserCmdButtonsPb,
			resetMarker(fieldButtonsState1))))

	assert.EqualValues(t, 0, buttons, "buttonstate1 reset to default")
}

// TestMergeUserCmdButtons_ResetNestedMessage: a wire-type-7 marker on a message
// along the path clears the button state, at either level.
func TestMergeUserCmdButtons_ResetNestedMessage(t *testing.T) {
	for name, delta := range map[string][]byte{
		"reset buttons_pb": bytesField(fieldCSGOUserCmdBase, resetMarker(fieldBaseUserCmdButtonsPb)),
		"reset base":       resetMarker(fieldCSGOUserCmdBase),
	} {
		t.Run(name, func(t *testing.T) {
			var buttons uint64

			mergeButtons(t, &buttons, baselineButtons())
			mergeButtons(t, &buttons, delta)

			assert.EqualValues(t, 0, buttons, "buttonstate1 zeroed")
		})
	}
}

// TestMergeUserCmdButtons_ChainedDeltas: consecutive deltas accumulate onto the
// same state like the real per-slot stream.
func TestMergeUserCmdButtons_ChainedDeltas(t *testing.T) {
	var buttons uint64

	mergeButtons(t, &buttons, baselineButtons())

	// delta 1: buttonstate1 -> 0
	mergeButtons(t, &buttons, bytesField(fieldCSGOUserCmdBase,
		bytesField(fieldBaseUserCmdButtonsPb,
			varintField(fieldButtonsState1, 0))))
	assert.EqualValues(t, 0, buttons)

	// delta 2: client_tick only, buttons must survive
	mergeButtons(t, &buttons, bytesField(fieldCSGOUserCmdBase,
		varintField(fieldBaseUserCmdClientTick, 102)))
	assert.EqualValues(t, 0, buttons)

	// delta 3: buttonstate1 -> 512 again
	mergeButtons(t, &buttons, bytesField(fieldCSGOUserCmdBase,
		bytesField(fieldBaseUserCmdButtonsPb,
			varintField(fieldButtonsState1, 512))))
	assert.EqualValues(t, 512, buttons)
}

// TestMergeUserCmdButtons_Malformed: a truncated blob is reported as an error
// rather than silently yielding a bogus state.
func TestMergeUserCmdButtons_Malformed(t *testing.T) {
	var buttons uint64

	// A bytes field claiming a longer payload than what follows.
	delta := append(protowire.AppendTag(nil, fieldCSGOUserCmdBase, protowire.BytesType), 0x10)

	assert.Error(t, mergeUserCmdButtons(delta, 0, &buttons))
}

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
