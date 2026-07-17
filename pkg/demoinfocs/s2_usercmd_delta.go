package demoinfocs

import (
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
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
//
// The repeated fields (input_history, subtick_moves) use their own delta
// grammar, decoded by mergeRepeatedDelta.
func applyUserCmdDelta(baseline *msgs2.CSGOUserCmdPB, deltaData []byte) error {
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
			case fd.IsList():
				// TODO Remove it? it slows parsing
				if err := mergeRepeatedDelta(msg, fd, b); err != nil {
					return err
				}
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

// mergeRepeatedDelta applies the delta for a repeated field (input_history,
// subtick_moves), whose elements are sub-messages.
//
// The blob is a sequence of varint tags. Each tag packs a number and a wire
// type as (number << 3) | wireType, like a normal protobuf tag, but the meaning
// is overloaded depending on the wire type:
//
//   - Count header (wire type 7, the reset marker): the number is the list's
//     new element count. It appears at most once, at the start, and only when
//     the count changed since the baseline. The list is resized to that count,
//     keeping existing elements for indices that still exist. When the count is
//     unchanged the header is omitted and entries follow directly.
//
//   - Entry (wire type 2, length-delimited): the number is the index of a
//     changed element — entries are sparse, only changed indices appear. The
//     payload is itself a delta-encoded element (so it may contain nested reset
//     markers) and is merged onto the element currently at that index.
func mergeRepeatedDelta(msg protoreflect.Message, fd protoreflect.FieldDescriptor, data []byte) error {
	if fd.Kind() != protoreflect.MessageKind && fd.Kind() != protoreflect.GroupKind {
		// The repeated user-command fields are all message lists; anything else
		// is unexpected here.
		return errors.Errorf("unsupported repeated delta for field %s (kind %v)", fd.FullName(), fd.Kind())
	}

	list := msg.Mutable(fd).List()
	headerSeen := false

	for len(data) > 0 {
		tag, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return errors.Wrapf(protowire.ParseError(n), "failed to read repeated delta tag for %s", fd.FullName())
		}
		data = data[n:]

		// number is the count for a header tag and the element index for an
		// entry tag; which one is determined by the wire type below.
		number := int(tag >> 3)
		typ := protowire.Type(tag & 7)

		// Count header: resize the list to the new length, keeping existing
		// elements for inherited indices.
		if typ == wireTypeReset {
			if headerSeen {
				return errors.Errorf("unexpected reset marker in repeated delta for %s", fd.FullName())
			}
			headerSeen = true
			resizeList(list, number)

			continue
		}

		if typ != protowire.BytesType {
			return errors.Errorf("unexpected wire type %d in repeated delta for %s", typ, fd.FullName())
		}

		// Entry: number is the index to merge onto. An index beyond the current
		// length can only happen when no count header was sent; grow to fit it.
		index := number
		if index >= list.Len() {
			resizeList(list, index+1)
		}

		entry, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return errors.Wrapf(protowire.ParseError(n), "failed to read repeated delta entry for %s", fd.FullName())
		}
		data = data[n:]

		if err := mergeUserCmdDelta(list.Get(index).Message(), entry); err != nil {
			return err
		}
	}

	return nil
}

// resizeList grows or shrinks a protoreflect list to n elements, keeping the
// existing elements (so inherited indices retain their baseline value).
func resizeList(list protoreflect.List, n int) {
	if n < list.Len() {
		list.Truncate(n)

		return
	}

	for list.Len() < n {
		list.Append(list.NewElement())
	}
}
