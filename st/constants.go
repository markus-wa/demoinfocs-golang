package st

import ()

type SendPropertyFlags int

func (spf SendPropertyFlags) HasFlagSet(flag SendPropertyFlags) bool {
	return int(spf)&int(flag) == int(flag)
}

const (
	SPT_Int int = iota
	SPT_Float
	SPT_Vector
	SPT_VectorXY
	SPT_String
	SPT_Array
	SPT_DataTable
	SPT_Int64
)

const (
	SPF_Unsigned SendPropertyFlags = (1 << iota)
	SPF_Coord
	SPF_NoScale
	SPF_RoundDown
	SPF_RoundUp
	SPF_Normal
	SPF_Exclude
	SPF_XYZE
	SPF_InsideArray
	SPF_ProxyAlwaysYes
	SPF_IsVectorElement
	SPF_Collapsible
	SPF_CoordMp
	SPF_CoordMpLowPrecision
	SPF_CoordMpIntegral
	SPF_CellCoord
	SPF_CellCoordLowPrecision
	SPF_CellCoordIntegral
	SPF_ChangesOften
	SPF_VarInt
)
