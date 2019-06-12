//+build debugdemoinfocs

// Functions to print out debug information if the build tag is enabled

package demoinfocs

import (
	"fmt"

	msg "github.com/markus-wa/demoinfocs-golang/msg"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

const (
	yes = "YES"
	no  = "NO"
)

// Can be overridden via -ldflags '-X github.com/markus-wa/demoinfocs-golang.debugServerClasses=YES'
// Oh and btw we cant use bools for this, Go says 'cannot use -X with non-string symbol'
var debugGameEvents = yes
var debugUnhandledMessages = no
var debugIngameTicks = yes
var debugDemoCommands = no
var debugServerClasses = no

func debugGameEvent(d *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	if debugGameEvents == yes {
		// Map only the relevant data for each type
		data := make(map[string]interface{})
		for k, v := range mapGameEventData(d, ge) {
			switch v.Type {
			case 1:
				data[k] = v.ValString
			case 2:
				data[k] = v.ValFloat
			case 3:
				data[k] = v.ValLong
			case 4:
				data[k] = v.ValShort
			case 5:
				data[k] = v.ValByte
			case 6:
				data[k] = v.ValBool
			case 7:
				data[k] = v.ValUint64
			}
		}
		fmt.Println("GameEvent:", d.Name, "Data:", data)
	}
}

func debugUnhandledMessage(cmd int, name string) {
	if debugUnhandledMessages == yes {
		fmt.Printf("UnhandledMessage: id=%d name=%s\n", cmd, name)
	}
}

func debugIngameTick(tickNr int) {
	if debugIngameTicks == yes {
		fmt.Printf("IngameTick=%d\n", tickNr)
	}
}

func (dc demoCommand) String() string {
	switch dc {
	case dcConsoleCommand:
		return "ConsoleCommand"
	case dcCustomData:
		return "CustomData"
	case dcDataTables:
		return "DataTables"
	case dcPacket:
		return "Packet"
	case dcSignon:
		return "Signon"
	case dcStop:
		return "Stop"
	case dcStringTables:
		return "StringTables"
	case dcSynctick:
		return "Synctick"
	case dcUserCommand:
		return "UserCommand"
	default:
		return "UnknownCommand"
	}
}

func debugDemoCommand(cmd demoCommand) {
	if debugDemoCommands == yes {
		fmt.Println("Demo-Command:", cmd)
	}
}

func debugAllServerClasses(classes st.ServerClasses) {
	if debugServerClasses == yes {
		for _, sc := range classes {
			fmt.Println(sc)
			fmt.Println()
		}
	}
}
