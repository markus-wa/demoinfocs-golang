//+build !debugdemoinfocs

// This file is just a bunch of NOPs for the release build, see debug_on.go for debugging stuff

package demoinfocs

import (
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

const isDebug = false

func debugGameEvent(descriptor *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	// NOP
}

func debugUnhandledMessage(cmd int, name string) {
	// NOP
}

func debugIngameTick(tickNr int) {
	// NOP
}
