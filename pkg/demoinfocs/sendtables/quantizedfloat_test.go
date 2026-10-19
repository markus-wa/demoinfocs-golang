package sendtables

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newQuantizedFloatDecoderOrPanic returns a decoder with 8 bits, or the value newQuantizedFloatDecoder() panicked with.
func newQuantizedFloatDecoderOrPanic(flags int32, low, high float32) (qfd *quantizedFloatDecoder, r any) {
	defer func() {
		r = recover()
	}()

	bitCount := int32(8)

	return newQuantizedFloatDecoder(&bitCount, &flags, &low, &high), nil
}

// A range of (2^61, 2^62] with qff_encode_integers made newQuantizedFloatDecoder() loop forever.
func TestNewQuantizedFloatDecoder_HugeIntegerRange(t *testing.T) {
	t.Parallel()

	for _, high := range []float32{1 << 32, 1 << 62, math.MaxFloat32} {
		done := make(chan any)

		go func() {
			_, r := newQuantizedFloatDecoderOrPanic(int32(qff_encode_integers), 0, high)
			done <- r
		}()

		select {
		case r := <-done:
			assert.Contains(t, r, "quantized float has an invalid integer range")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "newQuantizedFloatDecoder() hangs", "high: %v", high)
		}
	}
}

func TestNewQuantizedFloatDecoder_InvalidRange(t *testing.T) {
	t.Parallel()

	inf := float32(math.Inf(1))
	nan := float32(math.NaN())

	for _, tc := range [][2]float32{{0, inf}, {-inf, 0}, {nan, 1}, {0, nan}} {
		_, r := newQuantizedFloatDecoderOrPanic(0, tc[0], tc[1])
		assert.Contains(t, r, "quantized float has an invalid range", tc)
	}
}

func TestNewQuantizedFloatDecoder_IntegerRange(t *testing.T) {
	t.Parallel()

	// 2^31 is the largest range, it needs 32 bits
	qfd, r := newQuantizedFloatDecoderOrPanic(int32(qff_encode_integers), 0, 1<<31)
	require.Nil(t, r)
	assert.EqualValues(t, 32, qfd.Bitcount)

	qfd, r = newQuantizedFloatDecoderOrPanic(int32(qff_encode_integers), 0, 1000)
	require.Nil(t, r)
	assert.EqualValues(t, 11, qfd.Bitcount)
}
