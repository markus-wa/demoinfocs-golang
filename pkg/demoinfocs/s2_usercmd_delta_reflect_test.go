package demoinfocs

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

// applyUserCmdDeltaReflect is the protobuf-reflection delta merge that applyUserCmdDelta used before the typed merge
// (s2_usercmd_typed.go) replaced it. It's kept as the reference the typed merge is tested against, see
// TestUserCmdTypedMergeMatchesReflect: the two must agree on every delta, both in the result and in the error.
func applyUserCmdDeltaReflect(baseline *msg.CSGOUserCmdPB, deltaData []byte) error {
	return mergeUserCmdDeltaReflect(baseline.ProtoReflect(), deltaData)
}

// mergeUserCmdDeltaReflect handles the two non-standard constructs in
// codegen_delta_encoder output: wire-type-7 reset markers and indexed repeated
// operations. Ordinary protobuf fields are passed through to the protobuf
// decoder in one merge pass, while nested messages are merged recursively.
//
//nolint:gocognit
func mergeUserCmdDeltaReflect(target protoreflect.Message, data []byte) error {
	fields := target.Descriptor().Fields()

	var scalars []byte

	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return fmt.Errorf("failed to read delta field tag: %w", protowire.ParseError(tagLen))
		}

		if num == 0 {
			return errors.New("delta field number must be non-zero")
		}

		fd := fields.ByNumber(num)

		if typ == wireTypeReset {
			if fd == nil {
				return fmt.Errorf("wire type 7 references unknown field %d", num)
			}

			target.Clear(fd)

			data = data[tagLen:]

			continue
		}

		if typ == protowire.BytesType {
			value, valueLen := protowire.ConsumeBytes(data[tagLen:])
			if valueLen < 0 {
				return fmt.Errorf("failed to read length-delimited delta field: %w", protowire.ParseError(valueLen))
			}

			switch {
			case fd == nil:
				// Unknown fields are intentionally ignored, matching the game decoder.
			case fd.IsList() && (fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind):
				if err := mergeUserCmdRepeatedReflect(target, fd, value); err != nil {
					return err
				}
			case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
				if err := mergeUserCmdDeltaReflect(target.Mutable(fd).Message(), value); err != nil {
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
			return fmt.Errorf("failed to read delta field value: %w", protowire.ParseError(valueLen))
		}

		if fd != nil && !fd.IsList() {
			scalars = append(scalars, data[:tagLen+valueLen]...)
		}

		data = data[tagLen+valueLen:]
	}

	if len(scalars) > 0 {
		if err := (proto.UnmarshalOptions{Merge: true}).Unmarshal(scalars, target.Interface()); err != nil {
			return fmt.Errorf("failed to merge scalar delta fields: %w", err)
		}
	}

	return nil
}

func mergeUserCmdRepeatedReflect(parent protoreflect.Message, fd protoreflect.FieldDescriptor, data []byte) error {
	if fd.Kind() != protoreflect.MessageKind && fd.Kind() != protoreflect.GroupKind {
		return fmt.Errorf("repeated field %s is not a message list", fd.FullName())
	}

	list := parent.Mutable(fd).List()

	for len(data) > 0 {
		// Repeated operations are not protobuf tags: the high bits are a
		// zero-based index, so index zero legitimately has protobuf field
		// number zero. Read the raw varint instead of ConsumeTag.
		key, tagLen := protowire.ConsumeVarint(data)
		if tagLen < 0 {
			return fmt.Errorf("failed to read repeated delta index: %w", protowire.ParseError(tagLen))
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
				return fmt.Errorf("repeated delta index %d exceeds limit %d", index, userCmdRepeatedLimit)
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
				return fmt.Errorf("failed to read repeated delta message: %w", protowire.ParseError(valueLen))
			}

			if index >= list.Len() {
				return fmt.Errorf("repeated delta index %d is out of bounds for length %d", index, list.Len())
			}

			if err := mergeUserCmdDeltaReflect(list.Get(index).Message(), value); err != nil {
				return fmt.Errorf("failed to merge repeated field %s[%d]: %w", fd.FullName(), index, err)
			}

			data = data[tagLen+valueLen:]
		default:
			return fmt.Errorf("unsupported repeated delta wire type %d", typ)
		}
	}

	return nil
}
