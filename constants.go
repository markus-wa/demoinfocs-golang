package demoinfocs

type demoCommand byte

const (
	maxEditctBits = 11
	indexMask     = ((1 << maxEditctBits) - 1)
	maxEntities   = (1 << maxEditctBits)
	maxPlayers    = 64
	maxWeapons    = 64
)

const (
	dcSignon demoCommand = iota + 1
	dcPacket
	dcSynctick
	dcConsoleCommand
	dcUserCommand
	dcDataTables
	dcStop
	dcCustomData
	dcStringTables
)

const (
	stNameInstanceBaseline = "instancebaseline"
	stNameUserInfo         = "userinfo"
	stNameModelPreCache    = "modelprecache"
)
