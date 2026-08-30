package demoinfocs

import (
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// userCmdButtonRing keeps the same command-number validation as the full
// decoder, but stores only the one value needed by PlayerButtonsStateUpdate.
type userCmdButtonRing struct {
	entries [userCmdRingSize]userCmdButtonRingEntry
}

type userCmdButtonRingEntry struct {
	commandNumber int32
	buttons       uint64
	valid         bool
}

func (r *userCmdButtonRing) insert(commandNumber int32, buttons uint64) {
	r.entries[userCmdRingIndex(commandNumber)] = userCmdButtonRingEntry{
		commandNumber: commandNumber,
		buttons:       buttons,
		valid:         true,
	}
}

func (r *userCmdButtonRing) get(commandNumber int32) (uint64, bool) {
	entry := r.entries[userCmdRingIndex(commandNumber)]
	if !entry.valid || entry.commandNumber != commandNumber {
		return 0, false
	}
	return entry.buttons, true
}

type userCmdButtonPlayerState struct {
	ring                 userCmdButtonRing
	currentCommandNumber int32
	hasCurrentCommand    bool
}

// The buttonstate1 path is CSGOUserCmdPB.base.buttons_pb.buttonstate1.
var userCmdButtonsPath = [...]protowire.Number{1, 3, 1}

// mergeUserCmdButtons extracts only buttonstate1 from a full snapshot or delta.
// It skips all other fields without proto.Unmarshal or allocation. Wire type 7
// resets a target field to its protobuf default, including when the marker is on
// base or buttons_pb.
func mergeUserCmdButtons(data []byte, level int, buttons *uint64) error {
	if level >= len(userCmdButtonsPath) {
		return errors.New("invalid user command button path")
	}
	target := userCmdButtonsPath[level]

	for len(data) > 0 {
		num, typ, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return errors.Wrap(protowire.ParseError(tagLen), "failed to read user command button tag")
		}
		if num == 0 {
			return errors.New("user command button field number must be non-zero")
		}

		if typ == wireTypeReset {
			if num == target {
				*buttons = 0
			}
			data = data[tagLen:]
			continue
		}

		if num != target {
			valueLen := protowire.ConsumeFieldValue(num, typ, data[tagLen:])
			if valueLen < 0 {
				return errors.Wrap(protowire.ParseError(valueLen), "failed to skip user command field")
			}
			data = data[tagLen+valueLen:]
			continue
		}

		if level == len(userCmdButtonsPath)-1 {
			if typ != protowire.VarintType {
				return fmt.Errorf("buttonstate1 uses unexpected wire type %d", typ)
			}
			value, valueLen := protowire.ConsumeVarint(data[tagLen:])
			if valueLen < 0 {
				return errors.Wrap(protowire.ParseError(valueLen), "failed to read buttonstate1")
			}
			*buttons = value
			data = data[tagLen+valueLen:]
			continue
		}

		if typ != protowire.BytesType {
			return fmt.Errorf("user command button message uses unexpected wire type %d", typ)
		}
		value, valueLen := protowire.ConsumeBytes(data[tagLen:])
		if valueLen < 0 {
			return errors.Wrap(protowire.ParseError(valueLen), "failed to read user command button message")
		}
		if err := mergeUserCmdButtons(value, level+1, buttons); err != nil {
			return err
		}
		data = data[tagLen+valueLen:]
	}

	return nil
}

func (p *parser) handleUserCommandButtons(m *msg.CSVCMsg_UserCommands) {
	p.hasUserCmdMessages = true

	for _, cmd := range m.Commands {
		if cmd == nil || cmd.CmdNumber == nil {
			continue
		}

		slot := cmd.GetPlayerSlot()
		if slot < 0 {
			continue
		}
		state := p.userCmdButtonStates[slot]
		if state == nil {
			state = &userCmdButtonPlayerState{}
			p.userCmdButtonStates[slot] = state
		}

		var buttons uint64
		if data := cmd.GetData(); len(data) > 0 {
			if err := mergeUserCmdButtons(data, 0, &buttons); err != nil {
				p.msgDispatcher.Dispatch(events.ParserWarn{
					Message: err.Error(),
					Type:    events.WarnTypeUserCommandDeltaDecodeFailed,
				})
				continue
			}
		} else if len(cmd.GetDeltaData()) > 0 {
			if !state.hasCurrentCommand {
				p.msgDispatcher.Dispatch(events.ParserWarn{
					Message: fmt.Sprintf("user command %d has no baseline for player slot %d", cmd.GetCmdNumber(), slot),
					Type:    events.WarnTypeUserCommandBaselineMissing,
				})
				continue
			}
			requestedBaseline := state.currentCommandNumber
			if cmd.GetCmdNumber() < requestedBaseline {
				p.msgDispatcher.Dispatch(events.ParserWarn{
					Message: fmt.Sprintf("user command %d precedes its current baseline %d for player slot %d", cmd.GetCmdNumber(), requestedBaseline, slot),
					Type:    events.WarnTypeUserCommandBaselineMismatch,
				})
				continue
			}
			var found bool
			buttons, found = state.ring.get(requestedBaseline)
			if !found {
				p.msgDispatcher.Dispatch(events.ParserWarn{
					Message: fmt.Sprintf("user command %d has a baseline ring mismatch for player slot %d (requested %d)", cmd.GetCmdNumber(), slot, requestedBaseline),
					Type:    events.WarnTypeUserCommandBaselineMismatch,
				})
				continue
			}
		} else {
			continue
		}

		if deltaData := cmd.GetDeltaData(); len(deltaData) > 0 {
			if err := mergeUserCmdButtons(deltaData, 0, &buttons); err != nil {
				p.msgDispatcher.Dispatch(events.ParserWarn{
					Message: err.Error(),
					Type:    events.WarnTypeUserCommandDeltaDecodeFailed,
				})
				continue
			}
		}

		commandNumber := cmd.GetCmdNumber()
		state.ring.insert(commandNumber, buttons)
		state.currentCommandNumber = commandNumber
		state.hasCurrentCommand = true

		// Player slots map to controller entities: slot N is entity N+1
		// (getOrCreatePlayerFromControllerEntity relies on the same convention).
		player := p.gameState.playersByEntityID[int(slot)+1]
		if player == nil {
			continue
		}
		if player.ButtonsPressedState != buttons {
			player.ButtonsPressedState = buttons
			p.eventDispatcher.Dispatch(events.PlayerButtonsStateUpdate{
				Player:       player,
				ButtonsState: buttons,
			})
		}
	}
}
