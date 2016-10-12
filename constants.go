package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/common"
)

type DemoCommand byte

const (
	maxEntities = (1 << common.MaxEditctBits)
	MaxPlayers  = 64
	MaxWeapons  = 64
)

const (
	DC_Signon DemoCommand = iota + 1
	DC_Packet
	DC_Synctick
	DC_ConsoleCommand
	DC_UserCommand
	DC_DataTables
	DC_Stop
	DC_CustomData
	DC_StringTables
	DC_LastCommand  = DC_StringTables
	DC_FirstCommand = DC_Signon
)

const (
	FDEMO_NORMAL = iota + 1
	FDEMO_USE_ORIGIN2
	FDEMO_USE_ANGLES2
	FDEMO_NOINTERP
)
