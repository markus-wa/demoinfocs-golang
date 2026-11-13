package demoinfocs

import (
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"

	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

//go:generate go run ./internal/usercmdgen

// wireTypeReset is the "reset field to its protobuf default" marker used by
// Valve's codegen_delta_encoder. It reuses wire type 7, which is invalid in
// standard protobuf, which is why proto.Unmarshal fails on delta_data.
const wireTypeReset = protowire.Type(7)

const (
	userCmdRingSize      = 150
	userCmdRepeatedLimit = 0x100
)

// userCmdRing stores the last command snapshots by the command-number slot used
// by client.dll. The command number is stored alongside the payload because a
// modulo slot collision is a baseline mismatch, not a valid baseline.
type userCmdRing struct {
	entries [userCmdRingSize]userCmdRingEntry
}

type userCmdRingEntry struct {
	commandNumber int32
	command       *msg.CSGOUserCmdPB
	valid         bool
}

func userCmdRingIndex(commandNumber int32) int {
	index := commandNumber % userCmdRingSize
	if index < 0 {
		index += userCmdRingSize
	}

	return int(index)
}

// insert stores command without copying it: a stored command is never modified,
// a delta is applied to a copy of it (see handleUserCommands()).
// It's also dispatched as events.UserCmd.Command, which users must not modify.
func (r *userCmdRing) insert(commandNumber int32, command *msg.CSGOUserCmdPB) {
	r.entries[userCmdRingIndex(commandNumber)] = userCmdRingEntry{
		commandNumber: commandNumber,
		command:       command,
		valid:         true,
	}
}

func (r *userCmdRing) get(commandNumber int32) (*msg.CSGOUserCmdPB, bool) {
	entry := r.entries[userCmdRingIndex(commandNumber)]
	if !entry.valid || entry.commandNumber != commandNumber {
		return nil, false
	}

	return entry.command, true
}

func (r *userCmdRing) slotCommandNumber(commandNumber int32) (int32, bool) {
	entry := r.entries[userCmdRingIndex(commandNumber)]
	if !entry.valid {
		return 0, false
	}

	return entry.commandNumber, true
}

type userCmdPlayerState struct {
	ring                 userCmdRing
	currentCommandNumber int32
	hasCurrentCommand    bool
	seeded               bool // decoding is based on a full packet's command, see handleFullPacketUserCommands()
}

// dropSeed forgets the full packet's command a decoder is based on after a live command couldn't be decoded
// or didn't match its baseline (e.g. the command numbers started over): the live commands that follow aren't relative to it.
// A decoder based on a live full command keeps its baseline.
func (s *userCmdPlayerState) dropSeed() {
	if s.seeded {
		s.hasCurrentCommand = false
		s.seeded = false
	}
}

// copyUserCmd returns a deep copy of command, like proto.Clone, without protobuf reflection.
func copyUserCmd(command *msg.CSGOUserCmdPB) *msg.CSGOUserCmdPB {
	return copyCSGOUserCmdPB(command)
}

// applyUserCmdDelta reconstructs a full user-command state by applying a delta
// blob onto baseline. It mutates baseline as it goes, so a delta that turns out
// to be malformed leaves it partially merged; callers that need rollback should
// pass a copy (copyUserCmd()).
//
// The delta is the output of Valve's codegen_delta_encoder, protobuf wire format
// with two non-standard constructs, which is why proto.Unmarshal can't read it:
//
//   - A tag with wire type 7 (wireTypeReset) resets the field to its default:
//     it's unset, a nested message is dropped, a repeated field is emptied.
//     A reset marker for a field number the message doesn't have is an error.
//   - The value of a repeated message field is a sequence of indexed operations,
//     each a varint whose high bits are a zero-based element index and whose low
//     three bits are a wire type: a length-delimited value is a delta merged into
//     the element at that index (an error when it's out of bounds), wire type 7
//     sets the length (truncating, or appending empty elements up to
//     userCmdRepeatedLimit). Any other wire type is an error.
//
// Everything else is merged like proto.Unmarshal with Merge would: a nested
// message's delta is merged into the existing nested message (created when
// absent), a scalar value replaces the field's value, a field number the message
// doesn't have is skipped with its value (except under the reserved wire type 6
// or a stray end-group marker, which are errors, as they are for proto.Unmarshal),
// and the value of a known field with the wrong wire type is kept as an unknown
// field. The values of a delta are applied after its reset markers: a reset
// marker that follows a value of the same field in one delta leaves that value
// in place.
//
// The merge is typed, generated per message by internal/usercmdgen into
// s2_usercmd_typed.go. The protobuf-reflection merge it replaced is the
// reference it's tested against (mergeUserCmdDeltaReflect in
// s2_usercmd_delta_reflect_test.go).
func applyUserCmdDelta(baseline *msg.CSGOUserCmdPB, deltaData []byte) error {
	return mergeCSGOUserCmdPBDelta(baseline, deltaData)
}

// userCmdAppendUnknown keeps the value of a known field that arrived with another
// wire type than the field's as an unknown field, as proto.Unmarshal does. value
// is the encoded value after its tag, with the length prefix of a
// length-delimited value.
func userCmdAppendUnknown(m protoreflect.ProtoMessage, num protowire.Number, typ protowire.Type, value []byte) {
	r := m.ProtoReflect()
	unknown := protowire.AppendTag(r.GetUnknown(), num, typ)
	r.SetUnknown(append(unknown, value...))
}

// userCmdCopyUnknown copies the unknown fields of src to dst, like proto.Clone.
func userCmdCopyUnknown(dst, src protoreflect.ProtoMessage) {
	if unknown := src.ProtoReflect().GetUnknown(); len(unknown) > 0 {
		dst.ProtoReflect().SetUnknown(append(protoreflect.RawFields(nil), unknown...))
	}
}
