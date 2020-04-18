package constants

const (
	MaxEdictBits                 = 11
	EntityHandleIndexMask        = (1 << MaxEdictBits) - 1
	EntityHandleSerialNumberBits = 10
	EntityHandleBits             = MaxEdictBits + EntityHandleSerialNumberBits
	InvalidEntityHandle          = (1 << EntityHandleBits) - 1
)
