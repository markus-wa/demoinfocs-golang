// decoding the quantized float is a bit to messy to do in one function, so
// all the decoder steps are located in this file

package sendtablescs2

import (
	"math"
)

// Quantized float flags
const qff_rounddown uint32 = (1 << 0)
const qff_roundup uint32 = (1 << 1)
const qff_encode_zero uint32 = (1 << 2)
const qff_encode_integers uint32 = (1 << 3)

// Quantized-decoder struct containing the computed properties
type quantizedFloatDecoder struct {
	Low        float32 // Gets recomputed for round up / down
	High       float32
	HighLowMul float32
	DecMul     float32
	Offset     float32
	Bitcount   uint32 // Gets recomputed for qff_encode_int
	Flags      uint32
	NoScale    bool // Whether to decodes this as a noscale
}

// Validates / recomputes decoder flags
func (qfd *quantizedFloatDecoder) validateFlags() {
	// Check that we have some flags set
	if qfd.Flags == 0 {
		return
	}

	// Discard zero flag when encoding min / max set to 0
	if (qfd.Low == 0.0 && (qfd.Flags&qff_rounddown) != 0) || (qfd.High == 0.0 && (qfd.Flags&qff_roundup) != 0) {
		qfd.Flags &= ^qff_encode_zero
	}

	// If min / max is zero when encoding zero, switch to round up / round down instead
	if qfd.Low == 0.0 && (qfd.Flags&qff_encode_zero) != 0 {
		qfd.Flags |= qff_rounddown
		qfd.Flags &= ^qff_encode_zero
	}

	if qfd.High == 0.0 && (qfd.Flags&qff_encode_zero) != 0 {
		qfd.Flags |= qff_roundup
		qfd.Flags &= ^qff_encode_zero
	}

	// Check if the range spans zero
	if qfd.Low > 0.0 || qfd.High < 0.0 {
		qfd.Flags &= ^qff_encode_zero
	}

	// If we are left with encode zero, only leave integer flag
	if (qfd.Flags & qff_encode_integers) != 0 {
		qfd.Flags &= ^(qff_roundup | qff_rounddown | qff_encode_zero)
	}

	// Verify that we don;t have roundup / rounddown set
	if qfd.Flags&(qff_rounddown|qff_roundup) == (qff_rounddown | qff_roundup) {
		_panicf("Roundup / Rounddown are mutually exclusive")
	}
}

var qFloatMultipliers = []float32{0.9999, 0.99, 0.9, 0.8, 0.7}

// Assign multipliers
func (qfd *quantizedFloatDecoder) assignMultipliers(steps uint32) {
	qfd.HighLowMul = 0.0
	Range := qfd.High - qfd.Low

	High := uint32(0)
	if qfd.Bitcount == 32 {
		High = 0xFFFFFFFE
	} else {
		High = (1 << qfd.Bitcount) - 1
	}

	HighMul := float32(0.0)
	if math.Abs(float64(Range)) <= 0.0 {
		HighMul = float32(High)
	} else {
		HighMul = float32(High) / Range
	}

	// Adjust precision
	if (HighMul*Range > float32(High)) || (float64(HighMul*Range) > float64(High)) {

		for _, mult := range qFloatMultipliers {
			HighMul = float32(High) / Range * mult

			if (HighMul*Range > float32(High)) || (float64(HighMul*Range) > float64(High)) {
				continue
			}

			break
		}
	}

	qfd.HighLowMul = HighMul
	qfd.DecMul = 1.0 / float32(steps-1)

	if qfd.HighLowMul == 0.0 {
		_panicf("Error computing high / low multiplier")
	}
}

// Quantize a float
func (qfd *quantizedFloatDecoder) quantize(val float32) float32 {
	if val < qfd.Low {
		if (qfd.Flags & qff_roundup) == 0 {
			_panicf("Field tried to quantize an out of range value")
		}

		return qfd.Low
	} else if val > qfd.High {
		if (qfd.Flags & qff_rounddown) == 0 {
			_panicf("Field tried to quantize an out of range value")
		}

		return qfd.High
	}

	i := uint32((val - qfd.Low) * qfd.HighLowMul)
	//nolint:unconvert
	return qfd.Low + float32((qfd.High-qfd.Low)*float32(float32(i)*qfd.DecMul))
}

// Actual float decoding
func (qfd *quantizedFloatDecoder) decode(r *reader) float32 {
	if (qfd.Flags&qff_rounddown) != 0 && r.readBoolean() {
		return qfd.Low
	}

	if (qfd.Flags&qff_roundup) != 0 && r.readBoolean() {
		return qfd.High
	}

	if (qfd.Flags&qff_encode_zero) != 0 && r.readBoolean() {
		return 0.0
	}

	return qfd.Low + (qfd.High-qfd.Low)*float32(r.readBits(qfd.Bitcount))*qfd.DecMul
}

// Creates a new quantized float decoder based on given field
func newQuantizedFloatDecoder(bitCount, flags *int32, lowValue, highValue *float32) *quantizedFloatDecoder {
	qfd := &quantizedFloatDecoder{}

	// Set common properties
	if *bitCount == 0 || *bitCount >= 32 {
		qfd.NoScale = true
		qfd.Bitcount = 32
		return qfd
	} else {
		qfd.NoScale = false
		qfd.Bitcount = uint32(*bitCount)
		qfd.Offset = 0.0

		if lowValue != nil {
			qfd.Low = *lowValue
		} else {
			qfd.Low = 0.0
		}

		if highValue != nil {
			qfd.High = *highValue
		} else {
			qfd.High = 1.0
		}
	}
	if flags != nil {
		qfd.Flags = uint32(*flags)
	} else {
		qfd.Flags = 0
	}

	// Validate flags
	qfd.validateFlags()

	// Handle Round Up, Round Down
	steps := (1 << uint(qfd.Bitcount))

	Range := float32(0)
	if (qfd.Flags & qff_rounddown) != 0 {
		Range = qfd.High - qfd.Low
		qfd.Offset = (Range / float32(steps))
		qfd.High -= qfd.Offset
	} else if (qfd.Flags & qff_roundup) != 0 {
		Range = qfd.High - qfd.Low
		qfd.Offset = (Range / float32(steps))
		qfd.Low += qfd.Offset
	}

	// Handle integer encoding flag
	if (qfd.Flags & qff_encode_integers) != 0 {
		delta := qfd.High - qfd.Low

		if delta < 1 {
			delta = 1
		}

		deltaLog2 := math.Ceil(math.Log2(float64(delta)))
		Range2 := (1 << uint(deltaLog2))
		bc := qfd.Bitcount

		for {
			if (1 << uint(bc)) > Range2 {
				break
			} else {
				bc++
			}
		}

		if bc > qfd.Bitcount {
			qfd.Bitcount = bc
			steps = (1 << uint(qfd.Bitcount))
		}

		qfd.Offset = float32(Range2) / float32(steps)
		qfd.High = qfd.Low + float32(Range2) - qfd.Offset
	}

	// Assign multipliers
	qfd.assignMultipliers(uint32(steps))

	// Remove unessecary flags
	if (qfd.Flags & qff_rounddown) != 0 {
		if qfd.quantize(qfd.Low) == qfd.Low {
			qfd.Flags &= ^qff_rounddown
		}
	}

	if (qfd.Flags & qff_roundup) != 0 {
		if qfd.quantize(qfd.High) == qfd.High {
			qfd.Flags &= ^qff_roundup
		}
	}

	if (qfd.Flags & qff_encode_zero) != 0 {
		if qfd.quantize(0.0) == 0.0 {
			qfd.Flags &= ^qff_encode_zero
		}
	}

	return qfd
}
