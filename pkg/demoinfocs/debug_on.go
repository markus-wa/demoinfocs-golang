//go:build debugdemoinfocs
// +build debugdemoinfocs

// Functions to print out debug information if the build tag is enabled

package demoinfocs

import (
	"fmt"

	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

const (
	yes = "YES"
	no  = "NO"
)

// Can be overridden via -ldflags="-X 'github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs.debugServerClasses=YES'"
// e.g. `go run -tags debugdemoinfocs -ldflags="-X 'github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs.debugDemoCommands=YES'" examples/print-events/print_events.go -demo example.dem`
// Oh and btw we cant use bools for this, Go says 'cannot use -X with non-string symbol'
var (
	debugGameEvents    = yes
	debugServerClasses = no
	debugIngameTicks = no
)

func debugGameEvent(d *msg.CMsgSource1LegacyGameEventListDescriptorT, ge *msg.CMsgSource1LegacyGameEvent) {
	const (
		typeStr    = 1
		typeFloat  = 2
		typeLong   = 3
		typeShort  = 4
		typeByte   = 5
		typeBool   = 6
		typeUint64 = 7
	)

	if debugGameEvents == yes {
		// Map only the relevant data for each type
		data := make(map[string]any)

		for k, v := range mapGameEventData(d, ge) {
			switch v.GetType() {
			case typeStr:
				data[k] = v.GetValString()
			case typeFloat:
				data[k] = v.GetValFloat()
			case typeLong:
				data[k] = v.GetValLong()
			case typeShort:
				data[k] = v.GetValShort()
			case typeByte:
				data[k] = v.GetValByte()
			case typeBool:
				data[k] = v.GetValBool()
			case typeUint64:
				data[k] = v.GetValUint64()
			}
		}

		fmt.Println("GameEvent:", d.GetName(), "Data:", data)
	}
}

func debugIngameTick(tickNr int) {
	if debugIngameTicks == yes {
		fmt.Printf("IngameTick=%d\n", tickNr)
	}
}

func debugAllServerClasses(classes st.ServerClasses) {
	if debugServerClasses == yes {
		for _, sc := range classes.All() {
			fmt.Println(sc)
			fmt.Println()
		}
	}
}
