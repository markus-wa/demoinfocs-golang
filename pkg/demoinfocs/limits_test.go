package demoinfocs

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/fuzzcheck"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func TestPreallocSize(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		n, largest, want int
	}{
		{100, 0, 100},
		{maxFramePayload, 0, maxPreallocatedPayload},
		{maxFramePayload, 5 << 20, 5 << 20},
		{3 << 20, 5 << 20, 3 << 20},
		{maxAnimationDataPayload, maxAnimationDataPayload, maxRetainedBuffer},
	} {
		assert.Equal(t, tc.want, preallocSize(tc.n, tc.largest), "n=%d largest=%d", tc.n, tc.largest)
	}
}

func TestRetainLimit(t *testing.T) {
	t.Parallel()

	assert.Equal(t, maxPooledBuffer, retainLimit(0))
	assert.Equal(t, maxPooledBuffer, retainLimit(1<<20))
	assert.Equal(t, 5<<20, retainLimit(5<<20))
	assert.Equal(t, maxRetainedBuffer, retainLimit(maxAnimationDataPayload))
}

// Payloads larger than maxPooledBuffer (DEM_AnimationData of POV demos, ~5 MB in many frames)
// reuse the read and snappy buffers of the frames before (#743):
// each further frame only allocates about its payload (proto.Unmarshal copies bytes fields).
// Packets with nested messages between them must not take the buffers (msgBufPool).
// Not parallel, see fuzzcheck.AllocatedBytes().
//
//nolint:paralleltest
func TestLargePayloadBuffersReused(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector makes sync.Pool drop buffers at random")
	}

	const payloadSize = 5 << 20

	data := make([]byte, payloadSize)
	rng := rand.New(rand.NewPCG(1, 2))

	for i := range data {
		data[i] = byte(rng.Uint32())
	}

	payload, err := proto.Marshal(&msg.CDemoAnimationData{Data: data})
	require.NoError(t, err)

	for _, compressed := range []bool{false, true} {
		t.Run(fmt.Sprint("compressed=", compressed), func(t *testing.T) {
			demo := func(frames int) []byte {
				fs := [][]byte{fileHeaderFrame(t)}

				for i := range frames {
					fs = append(fs, packetFrame(t, uint32(i), netTick(t, uint32(i))))

					if compressed {
						fs = append(fs, compressedFrame(msg.EDemoCommands_DEM_AnimationData, 0, payload))
					} else {
						fs = append(fs, frame(msg.EDemoCommands_DEM_AnimationData, 0, payload))
					}
				}

				return handmadeDemo(append(fs, stopFrame())...)
			}

			one, five := demo(1), demo(5)

			var errOne, errFive error

			allocatedOne := fuzzcheck.AllocatedBytes(func() { _, errOne = parse(t, one) })
			allocatedFive := fuzzcheck.AllocatedBytes(func() { _, errFive = parse(t, five) })

			assert.NoError(t, errOne)
			assert.NoError(t, errFive)

			perFrame := (allocatedFive - allocatedOne) / 4
			// main (before #743) allocated 6-8x the payload per frame
			assert.Less(t, perFrame, uint64(payloadSize*2), "allocated bytes per further frame")
		})
	}
}
