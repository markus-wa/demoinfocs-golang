package sendtables

// sendPropertyFlags stores multiple send property flags.
type sendPropertyFlags int

// hasFlagSet returns true if the given flag is set
func (spf sendPropertyFlags) hasFlagSet(flag sendPropertyFlags) bool {
	return int(spf)&int(flag) == int(flag)
}

const (
	propTypeInt int = iota
	propTypeFloat
	propTypeVector
	propTypeVectorXY
	propTypeString
	propTypeArray
	propTypeDataTable
	propTypeInt64
)

const (
	propFlagUnsigned sendPropertyFlags = (1 << iota)
	propFlagCoord
	propFlagNoScale
	propFlagRoundDown
	propFlagRoundUp
	propFlagNormal
	propFlagExclude
	propFlagXYZE
	propFlagInsideArray
	propFlagProxyAlwaysYes
	propFlagIsVectorElement
	propFlagCollapsible
	propFlagCoordMp
	propFlagCoordMpLowPrecision
	propFlagCoordMpIntegral
	propFlagCellCoord
	propFlagCellCoordLowPrecision
	propFlagCellCoordIntegral
	propFlagChangesOften
	propFlagVarInt
)

const (
	dataTableMaxStringBits   = 9
	dataTableMaxStringLength = 1 << dataTableMaxStringBits
)
