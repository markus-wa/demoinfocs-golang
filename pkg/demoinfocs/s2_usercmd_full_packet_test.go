package demoinfocs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

const userCmdTestSlot = 2

// userCmds returns a svc_UserCmds net-message with the given commands.
func userCmds(t testing.TB, cmds ...*msg.CMsgServerUserCmd) nestedMessage {
	t.Helper()

	return nested(t, int32(msg.SVC_Messages_svc_UserCmds), &msg.CSVCMsg_UserCommands{Commands: cmds})
}

// fullUserCmd returns command number n of userCmdTestSlot as full payload.
func fullUserCmd(t testing.TB, n int32, cmd *msg.CSGOUserCmdPB) *msg.CMsgServerUserCmd {
	t.Helper()

	data, err := proto.Marshal(cmd)
	require.NoError(t, err)

	return &msg.CMsgServerUserCmd{CmdNumber: proto32(n), PlayerSlot: proto32(userCmdTestSlot), Data: data}
}

// deltaUserCmd returns command number n of userCmdTestSlot as delta with the given delta data.
func deltaUserCmd(n int32, deltaData []byte) *msg.CMsgServerUserCmd {
	return &msg.CMsgServerUserCmd{CmdNumber: proto32(n), PlayerSlot: proto32(userCmdTestSlot), DeltaData: deltaData}
}

// clientTickDelta returns delta data that sets the client tick.
func clientTickDelta(tick uint64) []byte {
	return bytesField(fieldCSGOUserCmdBase, varintField(fieldBaseUserCmdClientTick, tick))
}

// buttonsDelta returns delta data that sets buttonstate1.
func buttonsDelta(buttons uint64) []byte {
	return bytesField(fieldCSGOUserCmdBase, bytesField(fieldBaseUserCmdButtonsPb, varintField(fieldButtonsState1, buttons)))
}

type userCmdParseResult struct {
	commands      []events.UserCmd
	buttonUpdates []uint64
	warns         []events.ParserWarn
}

// parseUserCmdDemo parses a demo of the given frames with a player in userCmdTestSlot.
// It collects UserCmd and PlayerButtonsStateUpdate events, and the warnings of both dispatchers.
func parseUserCmdDemo(t *testing.T, mode option.UserCmdParsingMode, frames ...[]byte) userCmdParseResult {
	t.Helper()

	demo := handmadeDemo(append(append([][]byte{fileHeaderFrame(t)}, frames...), stopFrame())...)
	p := NewParser(bytes.NewReader(demo), option.UserCmdParsing(mode))
	t.Cleanup(func() { assert.NoError(t, p.Close()) })

	pl := common.NewPlayer(p.demoInfoProvider)
	pl.EntityID = userCmdTestSlot + 1
	p.gameState.playersByEntityID[pl.EntityID] = pl

	var res userCmdParseResult

	p.RegisterEventHandler(func(e events.UserCmd) { res.commands = append(res.commands, e) })
	p.RegisterEventHandler(func(e events.PlayerButtonsStateUpdate) { res.buttonUpdates = append(res.buttonUpdates, e.ButtonsState) })
	p.RegisterEventHandler(func(w events.ParserWarn) { res.warns = append(res.warns, w) })
	p.RegisterNetMessageHandler(func(w events.ParserWarn) { res.warns = append(res.warns, w) })

	require.NoError(t, p.ParseToEnd(t.Context()))

	return res
}

var userCmdDecodingModes = map[string]option.UserCmdParsingMode{
	"full":         option.UserCmdParsingFull,
	"buttons only": option.UserCmdParsingButtonsOnly,
}

// A full packet's command lets the live deltas that follow it be decoded,
// it isn't dispatched itself.
func TestFullPacketUserCmd_SeedsEmptyDecoder(t *testing.T) {
	res := parseUserCmdDemo(t, option.UserCmdParsingFull,
		fullPacketFrame(t, 1, netTick(t, 1), userCmds(t, fullUserCmd(t, 100, baseline()))),
		packetFrame(t, 2, netTick(t, 2), userCmds(t, deltaUserCmd(101, clientTickDelta(101)), deltaUserCmd(102, clientTickDelta(102)))),
	)

	assert.Empty(t, res.warns)
	require.Len(t, res.commands, 2)

	for i, cmd := range res.commands {
		n := int32(101 + i)
		assert.Equal(t, n, cmd.CommandNumber)
		assert.Equal(t, n, cmd.Command.GetBase().GetClientTick())
		assert.InDelta(t, 250, cmd.Command.GetBase().GetForwardmove(), 0, "inherited from the full packet's command")
	}
}

// A full packet's command is commonly ahead of the live stream, it must not replace the live baseline.
func TestFullPacketUserCmd_KeepsLiveBaseline(t *testing.T) {
	ahead := baseline()
	ahead.Base.ClientTick = proto32(104)
	ahead.Base.Forwardmove = protoFloat(-250)

	res := parseUserCmdDemo(t, option.UserCmdParsingFull,
		packetFrame(t, 1, netTick(t, 1), userCmds(t, fullUserCmd(t, 100, baseline()))),
		fullPacketFrame(t, 2, netTick(t, 2), userCmds(t, fullUserCmd(t, 104, ahead))),
		packetFrame(t, 3, netTick(t, 3), userCmds(t, deltaUserCmd(101, clientTickDelta(101)))),
	)

	assert.Empty(t, res.warns)
	require.Len(t, res.commands, 2)
	assert.EqualValues(t, 100, res.commands[0].CommandNumber)
	assert.EqualValues(t, 101, res.commands[1].CommandNumber)
	assert.InDelta(t, 250, res.commands[1].Command.GetBase().GetForwardmove(), 0, "decoded against the live command 100, not 104")
}

// UserCmdParsingFull and UserCmdParsingButtonsOnly dispatch the same button updates.
func TestFullPacketUserCmd_ButtonUpdates(t *testing.T) {
	pressed := baseline() // buttonstate1 512
	released := baseline()
	released.Base.ButtonsPb.Buttonstate1 = proto64(0)

	for name, mode := range userCmdDecodingModes {
		t.Run(name, func(t *testing.T) {
			t.Run("seeds empty decoder", func(t *testing.T) {
				res := parseUserCmdDemo(t, mode,
					fullPacketFrame(t, 1, netTick(t, 1), userCmds(t, fullUserCmd(t, 100, pressed))),
					packetFrame(t, 2, netTick(t, 2), userCmds(t, deltaUserCmd(101, buttonsDelta(8)))),
				)

				assert.Empty(t, res.warns)
				assert.Equal(t, []uint64{8}, res.buttonUpdates, "only the live command is dispatched")
			})

			t.Run("keeps live baseline", func(t *testing.T) {
				res := parseUserCmdDemo(t, mode,
					packetFrame(t, 1, netTick(t, 1), userCmds(t, fullUserCmd(t, 100, pressed))),
					fullPacketFrame(t, 2, netTick(t, 2), userCmds(t, fullUserCmd(t, 104, released))),
					packetFrame(t, 3, netTick(t, 3), userCmds(t, deltaUserCmd(101, clientTickDelta(101)))),
				)

				assert.Empty(t, res.warns)
				assert.Equal(t, []uint64{512}, res.buttonUpdates, "101 is decoded against the live command 100, not 104")
			})
		})
	}
}

// While decoding is based on a full packet's command, a live command that can't be decoded against it drops it:
// the deltas that follow aren't decoded against the seed. A live full command restores decoding.
func TestFullPacketUserCmd_DropsSeedWhenLiveCommandFails(t *testing.T) {
	for name, mode := range userCmdDecodingModes {
		t.Run(name, func(t *testing.T) {
			t.Run("decode failure", func(t *testing.T) {
				res := parseUserCmdDemo(t, mode,
					fullPacketFrame(t, 1, netTick(t, 1), userCmds(t, fullUserCmd(t, 100, baseline()))),
					packetFrame(t, 2, netTick(t, 2), userCmds(t, deltaUserCmd(101, []byte{0xff}))),
					packetFrame(t, 3, netTick(t, 3), userCmds(t, deltaUserCmd(102, buttonsDelta(8)))),
					packetFrame(t, 4, netTick(t, 4), userCmds(t, fullUserCmd(t, 103, baseline()), deltaUserCmd(104, buttonsDelta(16)))),
				)

				require.Len(t, res.warns, 2)
				assert.EqualValues(t, events.WarnTypeUserCommandDeltaDecodeFailed, res.warns[0].Type)
				assert.EqualValues(t, events.WarnTypeUserCommandBaselineMissing, res.warns[1].Type)
				assert.Equal(t, []uint64{512, 16}, res.buttonUpdates, "102 isn't decoded, 103 and 104 are")

				if mode == option.UserCmdParsingFull {
					require.Len(t, res.commands, 2)
					assert.EqualValues(t, 103, res.commands[0].CommandNumber)
					assert.EqualValues(t, 104, res.commands[1].CommandNumber)
				}
			})

			// The command numbers start over (e.g. a reconnect): the later deltas aren't relative to the seed.
			t.Run("baseline mismatch", func(t *testing.T) {
				res := parseUserCmdDemo(t, mode,
					fullPacketFrame(t, 1, netTick(t, 1), userCmds(t, fullUserCmd(t, 100, baseline()))),
					packetFrame(t, 2, netTick(t, 2), userCmds(t, deltaUserCmd(101, clientTickDelta(101)))),
					packetFrame(t, 3, netTick(t, 3), userCmds(t, deltaUserCmd(1, buttonsDelta(8)))),
					packetFrame(t, 4, netTick(t, 4), userCmds(t, deltaUserCmd(102, clientTickDelta(102)))),
				)

				require.Len(t, res.warns, 2)
				assert.EqualValues(t, events.WarnTypeUserCommandBaselineMismatch, res.warns[0].Type)
				assert.EqualValues(t, events.WarnTypeUserCommandBaselineMissing, res.warns[1].Type)
				assert.Equal(t, []uint64{512}, res.buttonUpdates, "only 101 is decoded")

				if mode == option.UserCmdParsingFull {
					require.Len(t, res.commands, 1)
					assert.EqualValues(t, 101, res.commands[0].CommandNumber)
				}
			})
		})
	}
}

func TestFullPacketUserCmd_Disabled(t *testing.T) {
	res := parseUserCmdDemo(t, option.UserCmdParsingDisabled,
		fullPacketFrame(t, 1, netTick(t, 1), userCmds(t, fullUserCmd(t, 100, baseline()))),
		packetFrame(t, 2, netTick(t, 2), userCmds(t, deltaUserCmd(101, buttonsDelta(8)))),
	)

	assert.Empty(t, res.warns)
	assert.Empty(t, res.commands)
	assert.Empty(t, res.buttonUpdates)
}

// A full packet whose user commands can't be decoded is skipped like any other broken full packet.
func TestFullPacketUserCmd_Undecodable(t *testing.T) {
	corrupt := nestedMessage{t: int32(msg.SVC_Messages_svc_UserCmds), data: []byte{0xff}}

	res := parseUserCmdDemo(t, option.UserCmdParsingFull, fullPacketFrame(t, 1, netTick(t, 1), corrupt))

	require.Len(t, res.warns, 1)
	assert.EqualValues(t, events.WarnTypeBrokenFullPacket, res.warns[0].Type)
}
