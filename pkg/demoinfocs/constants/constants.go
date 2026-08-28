// Package constants contains constants that are used internally across the demoinfocs library.
package constants

// Various constants that are used internally.
const (
	EntityHandleSerialNumberBits = 10

	MaxEdictBits          = 11
	EntityHandleIndexMask = (1 << MaxEdictBits) - 1
	EntityHandleBits      = MaxEdictBits + EntityHandleSerialNumberBits
	InvalidEntityHandle   = (1 << EntityHandleBits) - 1

	MaxEdictBitsSource2          = 14
	EntityHandleIndexMaskSource2 = (1 << MaxEdictBitsSource2) - 1
	EntityHandleBitsSource2      = MaxEdictBitsSource2 + EntityHandleSerialNumberBits
	InvalidEntityHandleSource2   = (1 << EntityHandleBitsSource2) - 1
)

// NetworkProtocolAnimGraph2 is the network protocol version of the CS2
// "AnimGraph 2" update (28/07/2025), the first version that networks the
// pawn velocity (the m_vecVelocity.m_vec* properties).
const NetworkProtocolAnimGraph2 = 14089
