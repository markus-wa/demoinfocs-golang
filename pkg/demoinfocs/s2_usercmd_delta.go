package demoinfocs

import (
	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// wireTypeReset is the "reset field to its protobuf default" marker used by
// Valve's codegen_delta_encoder. It reuses wire type 7, which is invalid in
// standard protobuf, which is why proto.Unmarshal fails on delta_data.
const wireTypeReset = protowire.Type(7)

// applyUserCmdDelta reconstructs the full user-command state for a player by
// applying a delta blob (CMsgServerUserCmd.delta_data) onto the previously
// accumulated full snapshot (baseline), which is mutated in place.
//
// Since the 2026-07-09 CS2 update, the prop m_nButtonDownMaskPrev has been removed
// and user commands are delta-encoded: only some commands carry a full protobuf
// snapshot in CMsgServerUserCmd.data (the first command for a player slot, server-generated
// substitute commands, and 60s DEM_FullPacket checkpoints).
// Every other command only carries the fields that changed since the previous command
// for that slot, in delta_data.
//
// delta_data is almost standard protobuf wire format, with one extension: a
// field encoded with wire type 7 is a "reset to default" marker rather than a
// value. Present fields replace the baseline field, nested messages are merged
// recursively, reset markers clear the field (recursively for messages), and
// omitted fields keep their baseline value.
func applyUserCmdDelta(baseline *msg.CSGOUserCmdPB, deltaData []byte) error {
	return mergeUserCmdDelta(baseline.ProtoReflect(), deltaData)
}

// mergeUserCmdDelta walks the delta blob handling only the two non-standard
// constructs — wire-type-7 reset markers and the repeated-field grammar — and
// recursing into nested messages (which may themselves contain resets). Every
// ordinary scalar field is standard protobuf wire format, so it is collected
// verbatim and handed to the protobuf decoder in one merge pass, rather than
// re-implementing per-type value decoding here.
func mergeUserCmdDelta(msg protoreflect.Message, data []byte) error {
	fields := msg.Descriptor().Fields()

	var scalars []byte // standard-protobuf scalar fields, decoded by proto.Unmarshal below

	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return errors.Wrap(protowire.ParseError(tagLen), "failed to read delta field tag")
		}

		fd := fields.ByNumber(num)

		// Reset marker: restore the field to its protobuf default (recursively
		// zeroing a sub-message). It carries no value payload.
		if typ == wireTypeReset {
			if fd != nil {
				msg.Clear(fd)
			}
			data = data[tagLen:]

			continue
		}

		// Length-delimited fields need dispatching: lists use their own grammar,
		// sub-messages are merged recursively, string/bytes fall through to the
		// scalar passthrough.
		if typ == protowire.BytesType {
			b, valLen := protowire.ConsumeBytes(data[tagLen:])
			if valLen < 0 {
				return errors.Wrap(protowire.ParseError(valLen), "failed to read length-delimited delta field")
			}

			switch {
			case fd == nil: // unknown field, skip
			case fd.IsList(): // Repeated fields such as subtick_moves/input_history, it slows down parsing and are not needed by the lib, skip it.
			case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
				if err := mergeUserCmdDelta(msg.Mutable(fd).Message(), b); err != nil {
					return err
				}
			default: // string / bytes
				scalars = append(scalars, data[:tagLen+valLen]...)
			}

			data = data[tagLen+valLen:]

			continue
		}

		// Fixed/varint scalar: pass the raw field bytes to the protobuf decoder.
		valLen := protowire.ConsumeFieldValue(num, typ, data[tagLen:])
		if valLen < 0 {
			return errors.Wrap(protowire.ParseError(valLen), "failed to read delta field value")
		}
		if fd != nil && !fd.IsList() {
			scalars = append(scalars, data[:tagLen+valLen]...)
		}

		data = data[tagLen+valLen:]
	}

	if len(scalars) > 0 {
		// Merge (don't reset) so untouched baseline fields survive.
		return proto.UnmarshalOptions{Merge: true}.Unmarshal(scalars, msg.Interface())
	}

	return nil
}
