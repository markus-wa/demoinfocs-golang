package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

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
	command       *msgs2.CSGOUserCmdPB
	valid         bool
}

func userCmdRingIndex(commandNumber int32) int {
	index := commandNumber % userCmdRingSize
	if index < 0 {
		index += userCmdRingSize
	}

	return int(index)
}

func (r *userCmdRing) insert(commandNumber int32, command *msgs2.CSGOUserCmdPB) {
	r.entries[userCmdRingIndex(commandNumber)] = userCmdRingEntry{
		commandNumber: commandNumber,
		command:       proto.Clone(command).(*msgs2.CSGOUserCmdPB),
		valid:         true,
	}
}

func (r *userCmdRing) get(commandNumber int32) (*msgs2.CSGOUserCmdPB, bool) {
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
}

// applyUserCmdDelta reconstructs a full user-command state by applying a delta
// blob onto baseline. It mutates baseline only after each operation is validated;
// callers that need rollback should pass a clone.
func applyUserCmdDelta(baseline *msgs2.CSGOUserCmdPB, deltaData []byte) error {
	return mergeUserCmdDelta(baseline.ProtoReflect(), deltaData)
}

// mergeUserCmdDelta handles the two non-standard constructs in
// codegen_delta_encoder output: wire-type-7 reset markers and indexed repeated
// operations. Ordinary protobuf fields are passed through to the protobuf
// decoder in one merge pass, while nested messages are merged recursively.
//
//nolint:gocognit,funlen
func mergeUserCmdDelta(target protoreflect.Message, data []byte) error {
	fields := target.Descriptor().Fields()

	var scalars []byte

	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return errors.Wrap(protowire.ParseError(tagLen), "failed to read delta field tag")
		}

		if num == 0 {
			return errors.New("delta field number must be non-zero")
		}

		fd := fields.ByNumber(num)

		if typ == wireTypeReset {
			if fd == nil {
				return errors.Errorf("wire type 7 references unknown field %d", num)
			}

			target.Clear(fd)

			data = data[tagLen:]

			continue
		}

		if typ == protowire.BytesType {
			value, valueLen := protowire.ConsumeBytes(data[tagLen:])
			if valueLen < 0 {
				return errors.Wrap(protowire.ParseError(valueLen), "failed to read length-delimited delta field")
			}

			switch {
			case fd == nil:
				// Unknown fields are intentionally ignored, matching the game decoder.
			case fd.IsList() && (fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind):
				if err := mergeUserCmdRepeated(target, fd, value); err != nil {
					return err
				}
			case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
				if err := mergeUserCmdDelta(target.Mutable(fd).Message(), value); err != nil {
					return err
				}
			default:
				// String/bytes and any scalar encoded as length-delimited are
				// standard protobuf fields.
				scalars = append(scalars, data[:tagLen+valueLen]...)
			}

			data = data[tagLen+valueLen:]

			continue
		}

		valueLen := protowire.ConsumeFieldValue(num, typ, data[tagLen:])
		if valueLen < 0 {
			return errors.Wrap(protowire.ParseError(valueLen), "failed to read delta field value")
		}

		if fd != nil && !fd.IsList() {
			scalars = append(scalars, data[:tagLen+valueLen]...)
		}

		data = data[tagLen+valueLen:]
	}

	if len(scalars) > 0 {
		if err := (proto.UnmarshalOptions{Merge: true}).Unmarshal(scalars, target.Interface()); err != nil {
			return errors.Wrap(err, "failed to merge scalar delta fields")
		}
	}

	return nil
}

func mergeUserCmdRepeated(parent protoreflect.Message, fd protoreflect.FieldDescriptor, data []byte) error {
	if fd.Kind() != protoreflect.MessageKind && fd.Kind() != protoreflect.GroupKind {
		return errors.Errorf("repeated field %s is not a message list", fd.FullName())
	}

	list := parent.Mutable(fd).List()

	for len(data) > 0 {
		// Repeated operations are not protobuf tags: the high bits are a
		// zero-based index, so index zero legitimately has protobuf field
		// number zero. Read the raw varint instead of ConsumeTag.
		key, tagLen := protowire.ConsumeVarint(data)
		if tagLen < 0 {
			return errors.Wrap(protowire.ParseError(tagLen), "failed to read repeated delta index")
		}

		if key>>3 > uint64(^uint(0)>>1) {
			return errors.New("repeated delta index does not fit in int")
		}

		index := int(key >> 3)
		typ := protowire.Type(key & 0x07)

		//nolint:exhaustive // The repeated-delta grammar only uses a subset of
		// protowire types (reset marker and length-delimited); all others are invalid.
		switch typ {
		case wireTypeReset:
			if index > userCmdRepeatedLimit {
				return errors.Errorf("repeated delta index %d exceeds limit %d", index, userCmdRepeatedLimit)
			}

			if index > list.Len() {
				for list.Len() < index {
					list.Append(list.NewElement())
				}
			} else {
				list.Truncate(index)
			}

			data = data[tagLen:]
		case protowire.BytesType:
			value, valueLen := protowire.ConsumeBytes(data[tagLen:])
			if valueLen < 0 {
				return errors.Wrap(protowire.ParseError(valueLen), "failed to read repeated delta message")
			}

			if index >= list.Len() {
				return errors.Errorf("repeated delta index %d is out of bounds for length %d", index, list.Len())
			}

			if err := mergeUserCmdDelta(list.Get(index).Message(), value); err != nil {
				return errors.Wrapf(err, "failed to merge repeated field %s[%d]", fd.FullName(), index)
			}

			data = data[tagLen+valueLen:]
		default:
			return errors.Errorf("unsupported repeated delta wire type %d", typ)
		}
	}

	return nil
}
