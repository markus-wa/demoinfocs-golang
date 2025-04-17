// Package constants contains constants that are used internally across the demoinfocs library.
package constants

// Various constants that are used internally.
const (
	EntityHandleSerialNumberBits = 10
	MaxEdictBitsSource2          = 14
	EntityHandleIndexMaskSource2 = (1 << MaxEdictBitsSource2) - 1
	EntityHandleBitsSource2      = MaxEdictBitsSource2 + EntityHandleSerialNumberBits
	InvalidEntityHandleSource2   = (1 << EntityHandleBitsSource2) - 1
)
