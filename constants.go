package demoinfocs

const (
	maxEditctBits = 11
	indexMask     = ((1 << maxEditctBits) - 1)
	maxEntities   = (1 << maxEditctBits)
	maxPlayers    = 64
	maxWeapons    = 64
)

// Demo commands as documented at https://developer.valvesoftware.com/wiki/DEM_Format
type demoCommand byte

const (
	dcSignon         demoCommand = 1
	dcPacket         demoCommand = 2
	dcSynctick       demoCommand = 3
	dcConsoleCommand demoCommand = 4
	dcUserCommand    demoCommand = 5
	dcDataTables     demoCommand = 6
	dcStop           demoCommand = 7
	dcCustomData     demoCommand = 8
	dcStringTables   demoCommand = 9
)

const (
	stNameInstanceBaseline = "instancebaseline"
	stNameUserInfo         = "userinfo"
	stNameModelPreCache    = "modelprecache"
)

// We're all better off not asking questions
const valveMagicNumber = 76561197960265728
