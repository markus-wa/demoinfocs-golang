package demoinfocs

import (
	"errors"
	"fmt"

	"github.com/golang/snappy"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

// Limits for sizes read from the demo, so corrupt or crafted demos can't make the parser allocate huge amounts of memory.
// They are far above what real demos use: the largest frame payload measured across 50 CS2 demos (2023-2026, incl. POV)
// is ~5 MB (DEM_AnimationData of a POV demo), all other frames are below 256 KB, and nested net-messages below 160 KB.
const (
	// maxFramePayload bounds the raw and decompressed payload of a demo frame (except DEM_AnimationData).
	// Protobuf messages can expand in memory, the limit also bounds that.
	maxFramePayload = 16 << 20

	// maxAnimationDataPayload bounds the payload of DEM_AnimationData frames, which are large in POV demos.
	// They're a single bytes field, so they don't expand in memory like other protobuf messages.
	maxAnimationDataPayload = 64 << 20

	// maxSnappyRatio bounds the ratio of a snappy payload's decoded and compressed length.
	// Snappy's maximum expansion is 64/3 (a 3 byte copy of 64 bytes), the decoded length header of a payload
	// that claims more is corrupt.
	maxSnappyRatio = 24

	// maxPreallocatedPayload bounds the memory allocated for a payload before it was read:
	// larger payloads grow while they're read, so a size that's larger than the rest of the demo doesn't allocate much.
	// Once a larger payload was read completely, payloads up to its size are preallocated, see preallocSize().
	maxPreallocatedPayload = 1 << 20

	// maxPooledBuffer is the largest buffer that's kept in msgBufPool, and in Parser.snappyScratch
	// until a larger payload was read completely (see retainLimit()). Larger payloads use Parser.largeFrameBuf.
	maxPooledBuffer = 4 << 20

	// maxRetainedBuffer is the hard cap of preallocSize(), retainLimit() and Parser.largeFrameBuf,
	// however large the payloads read so far were.
	maxRetainedBuffer = maxFramePayload
)

// preallocSize returns how much memory to allocate for a payload of n bytes before it's read,
// given the largest payload read completely so far.
// It stays at maxPreallocatedPayload unless a larger payload was read, so a corrupt size can't make it allocate much
// more than real payloads of the demo did (at most maxRetainedBuffer).
func preallocSize(n, largestPayload int) int {
	return min(n, max(maxPreallocatedPayload, min(largestPayload, maxRetainedBuffer)))
}

// retainLimit returns the largest buffer that's kept for reuse, given the largest payload read completely so far.
// POV demos have payloads of ~5 MB (DEM_AnimationData) in many frames, their buffers are reused too.
func retainLimit(largestPayload int) int {
	return min(max(maxPooledBuffer, largestPayload), maxRetainedBuffer)
}

// errFrameTooLarge is the cause of a frame that's larger than allowed, see framePayloadLimit().
var errFrameTooLarge = errors.New("frame too large")

// errSnappyLimit is the cause of a snappy payload that's larger than allowed, see decodeSnappy().
var errSnappyLimit = errors.New("decoded length of compressed data exceeds the limit")

// framePayloadLimit returns the maximum payload size of a frame of the given type.
func framePayloadLimit(t msg.EDemoCommands) uint64 {
	if t == msg.EDemoCommands_DEM_AnimationData {
		return maxAnimationDataPayload
	}

	return maxFramePayload
}

// decodeSnappy decompresses src. It checks the decoded length before decompressing,
// so a corrupt length can't make it allocate more than limit bytes (or more than maxSnappyRatio times len(src)).
// dst is used as destination if it has enough capacity, see snappy.Decode().
func decodeSnappy(dst, src []byte, limit uint64) ([]byte, error) {
	n, err := snappy.DecodedLen(src)
	if err != nil {
		return nil, fmt.Errorf("failed to decode snappy compressed data: %w", err)
	}

	if uint64(n) > limit || uint64(n) > maxSnappyRatio*uint64(len(src)) {
		return nil, fmt.Errorf("%w: %d bytes (limit %d) from %d compressed bytes", errSnappyLimit, n, limit, len(src))
	}

	// snappy.Decode() only uses dst if it's long enough.
	if n <= cap(dst) {
		dst = dst[:n]
	} else {
		dst = nil
	}

	b, err := snappy.Decode(dst, src)
	if err != nil {
		return nil, fmt.Errorf("failed to decode snappy compressed data: %w", err)
	}

	return b, nil
}
