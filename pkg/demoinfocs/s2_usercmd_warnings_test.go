package demoinfocs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/option"
)

// User command decoding warnings are dispatched to event handlers, like the parser's other warnings.
func TestUserCmdWarningsReachEventHandlers(t *testing.T) {
	// A delta for a player without a full command yet (WarnTypeUserCommandBaselineMissing).
	commands := nested(t, int32(msg.SVC_Messages_svc_UserCmds), &msg.CSVCMsg_UserCommands{
		Commands: []*msg.CMsgServerUserCmd{{
			CmdNumber:  proto32(100),
			PlayerSlot: proto32(2),
			DeltaData:  bytesField(fieldCSGOUserCmdBase, varintField(fieldBaseUserCmdClientTick, 100)),
		}},
	})
	demo := handmadeDemo(fileHeaderFrame(t), packetFrame(t, 1, netTick(t, 1), commands), stopFrame())

	for name, mode := range map[string]option.UserCmdParsingMode{
		"full":         option.UserCmdParsingFull,
		"buttons only": option.UserCmdParsingButtonsOnly,
	} {
		t.Run(name, func(t *testing.T) {
			p := NewParser(bytes.NewReader(demo), option.UserCmdParsing(mode))
			defer p.Close()

			var eventWarns, netMessageWarns []events.ParserWarn

			p.RegisterEventHandler(func(w events.ParserWarn) { eventWarns = append(eventWarns, w) })
			p.RegisterNetMessageHandler(func(w events.ParserWarn) { netMessageWarns = append(netMessageWarns, w) })

			require.NoError(t, p.ParseToEnd(t.Context()))

			require.Len(t, eventWarns, 1)
			assert.EqualValues(t, events.WarnTypeUserCommandBaselineMissing, eventWarns[0].Type)
			assert.Empty(t, netMessageWarns)
		})
	}
}
