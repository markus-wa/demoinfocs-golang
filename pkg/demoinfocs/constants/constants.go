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

// NetworkProtocolAnimGraph2 is the network protocol version (demo header patch version) of the CS2
// "AnimGraph 2" update (28/07/2025). Demos recorded since then network the player pawn velocity
// (m_vecVelocity.m_vec* properties); older demos declare these properties, but don't populate them on GOTV
// demos and only populate them correctly for the recording player on POV demos.
const NetworkProtocolAnimGraph2 = 14089

// DefaultMaxVelocity is the default value of the sv_maxvelocity convar, the maximum velocity (per axis, in
// game units per second) the engine allows for any entity.
const DefaultMaxVelocity = 3500
