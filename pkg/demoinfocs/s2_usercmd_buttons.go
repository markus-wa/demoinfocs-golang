package demoinfocs

import (
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"
)

// wireTypeReset is the "reset field to its protobuf default" marker used by
// Valve's codegen_delta_encoder. It reuses wire type 7, which is invalid in
// standard protobuf, which is why proto.Unmarshal fails on delta_data.
const wireTypeReset = protowire.Type(7)

// userCmdButtonsPath is the field-number path from the root of a CSGOUserCmdPB
// down to the only value we care about:
//
//	CSGOUserCmdPB.base (1) -> CBaseUserCmdPB.buttons_pb (3) -> CInButtonStatePB.buttonstate1 (1)
var userCmdButtonsPath = [...]protowire.Number{1, 3, 1}

// mergeUserCmdButtons walks a full CSGOUserCmdPB snapshot or a delta blob and
// merges buttonstate1 into buttons, skipping every other field without decoding
// it. Deltas leave buttons untouched when the path is absent, and zero it on a
// reset marker anywhere along the path.
//
// Since the 2026-07-09 CS2 update, the prop m_nButtonDownMaskPrev has been removed
// and user commands are delta-encoded: only some commands carry a full protobuf
// snapshot in CMsgServerUserCmd.data (the first command for a player slot, server-generated
// substitute commands, and 60s DEM_FullPacket checkpoints). Every other command only
// carries the fields that changed since the previous command for that slot, in delta_data.
//
// delta_data is almost standard protobuf wire format, with one extension: a
// field encoded with wire type 7 is a "reset to default" marker rather than a
// value. Present fields replace the previous value, reset markers clear the
// field (recursively for messages), and omitted fields keep their previous value.
//
// This is the hot path: it runs once per user command, i.e. once per player per
// tick, so it deliberately avoids protoreflect, proto.Unmarshal and any
// allocation.
func mergeUserCmdButtons(data []byte, level int, buttons *uint64) error {
	target := userCmdButtonsPath[level]

	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return errors.Wrap(protowire.ParseError(tagLen), "failed to read delta field tag")
		}

		if typ == wireTypeReset {
			// Resetting any message along the path clears the button state.
			if num == target {
				*buttons = 0
			}

			data = data[tagLen:]

			continue
		}

		if num != target {
			valLen := protowire.ConsumeFieldValue(num, typ, data[tagLen:])
			if valLen < 0 {
				return errors.Wrap(protowire.ParseError(valLen), "failed to read delta field value")
			}

			data = data[tagLen+valLen:]

			continue
		}

		if level == len(userCmdButtonsPath)-1 {
			v, valLen := protowire.ConsumeVarint(data[tagLen:])
			if valLen < 0 {
				return errors.Wrap(protowire.ParseError(valLen), "failed to read buttonstate1")
			}

			*buttons = v
			data = data[tagLen+valLen:]

			continue
		}

		b, valLen := protowire.ConsumeBytes(data[tagLen:])
		if valLen < 0 {
			return errors.Wrap(protowire.ParseError(valLen), "failed to read length-delimited delta field")
		}

		if err := mergeUserCmdButtons(b, level+1, buttons); err != nil {
			return err
		}

		data = data[tagLen+valLen:]
	}

	return nil
}
