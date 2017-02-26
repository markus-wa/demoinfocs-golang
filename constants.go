package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/common"
)

type demoCommand byte

const (
	maxEntities = (1 << common.MaxEditctBits)
	maxPlayers  = 64
	maxWeapons  = 64
)

const (
	dc_Signon demoCommand = iota + 1
	dc_Packet
	dc_Synctick
	dc_ConsoleCommand
	dc_UserCommand
	dc_DataTables
	dc_Stop
	dc_CustomData
	dc_StringTables
)

const (
	stName_InstanceBaseline = "instancebaseline"
	stName_UserInfo         = "userinfo"
	stName_ModelPreCache    = "modelprecache"
)
