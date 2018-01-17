package sendtables

import (
	"fmt"
	"math"

	r3 "github.com/golang/geo/r3"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
)

const (
	coordFractionalBitsMp             = 5
	coordFractionalBitsMpLowPrecision = 3
	coordDenominator                  = 1 << coordFractionalBitsMp
	coordResolution                   = 1.0 / coordDenominator
	coordDenominatorLowPrecision      = 1 << coordFractionalBitsMpLowPrecision
	coordResolutionLowPrecision       = 1.0 / coordDenominatorLowPrecision
	coordIntegerBitsMp                = 11
	coordIntegerBits                  = 14
)

const (
	normalFractBits   = 11
	normalDenominator = 1 << (normalFractBits - 1)
	normalResolution  = 1.0 / normalDenominator
)

const specialFloatFlags = propFlagNoScale | propFlagCoord | propFlagCellCoord | propFlagNormal | propFlagCoordMp | propFlagCoordMpLowPrecision | propFlagCoordMpIntegral | propFlagCellCoordLowPrecision | propFlagCellCoordIntegral

var propDecoder propertyDecoder

// PropValue stores parsed & decoded send-table values.
// For instance player health, location etc.
type PropValue struct {
	VectorVal r3.Vector
	IntVal    int
	ArrayVal  []PropValue
	StringVal string
	FloatVal  float32
}

type propertyDecoder struct{}

func (propertyDecoder) decodeProp(fProp *FlattenedPropEntry, reader *bit.BitReader) PropValue {
	switch fProp.prop.rawType {
	case propTypeFloat:
		return PropValue{FloatVal: propDecoder.decodeFloat(fProp.prop, reader)}

	case propTypeInt:
		return PropValue{IntVal: propDecoder.decodeInt(fProp.prop, reader)}

	case propTypeVectorXY:
		return PropValue{VectorVal: propDecoder.decodeVectorXY(fProp.prop, reader)}

	case propTypeVector:
		return PropValue{VectorVal: propDecoder.decodeVector(fProp.prop, reader)}

	case propTypeArray:
		return PropValue{ArrayVal: propDecoder.decodeArray(fProp, reader)}

	case propTypeString:
		return PropValue{StringVal: propDecoder.decodeString(fProp.prop, reader)}

	default:
		panic(fmt.Sprintf("Unknown prop type %d", fProp.prop.rawType))
	}
}

func (propertyDecoder) decodeInt(prop *sendTableProperty, reader *bit.BitReader) int {
	if prop.flags.hasFlagSet(propFlagVarInt) {
		if prop.flags.hasFlagSet(propFlagUnsigned) {
			return int(reader.ReadVarInt32())
		}
		return int(reader.ReadSignedVarInt32())
	}
	if prop.flags.hasFlagSet(propFlagUnsigned) {
		return int(reader.ReadInt(uint(prop.numberOfBits)))
	}
	return reader.ReadSignedInt(uint(prop.numberOfBits))
}

func (propertyDecoder) decodeFloat(prop *sendTableProperty, reader *bit.BitReader) float32 {
	if prop.flags&specialFloatFlags != 0 {
		return propDecoder.decodeSpecialFloat(prop, reader)
	}

	dwInterp := reader.ReadInt(uint(prop.numberOfBits))
	return prop.lowValue + ((prop.highValue - prop.lowValue) * (float32(dwInterp) / float32((int(1)<<uint(prop.numberOfBits))-1)))
}

func (propertyDecoder) decodeSpecialFloat(prop *sendTableProperty, reader *bit.BitReader) float32 {
	// Because multiple flags can be set this order is fixed for now (priorities).
	// TODO: Would be more efficient to first check the most common ones tho.
	if prop.flags.hasFlagSet(propFlagCoord) {
		return propDecoder.readBitCoord(reader)
	} else if prop.flags.hasFlagSet(propFlagCoordMp) {
		return propDecoder.readBitCoordMp(reader, false, false)
	} else if prop.flags.hasFlagSet(propFlagCoordMpLowPrecision) {
		return propDecoder.readBitCoordMp(reader, false, true)
	} else if prop.flags.hasFlagSet(propFlagCoordMpIntegral) {
		return propDecoder.readBitCoordMp(reader, true, false)
	} else if prop.flags.hasFlagSet(propFlagNoScale) {
		return reader.ReadFloat()
	} else if prop.flags.hasFlagSet(propFlagNormal) {
		return propDecoder.readBitNormal(reader)
	} else if prop.flags.hasFlagSet(propFlagCellCoord) {
		return propDecoder.readBitCellCoord(reader, uint(prop.numberOfBits), false, false)
	} else if prop.flags.hasFlagSet(propFlagCellCoordLowPrecision) {
		return propDecoder.readBitCellCoord(reader, uint(prop.numberOfBits), true, false)
	} else if prop.flags.hasFlagSet(propFlagCellCoordIntegral) {
		return propDecoder.readBitCellCoord(reader, uint(prop.numberOfBits), false, true)
	}
	panic(fmt.Sprintf("Unexpected special float flag (Flags %v)", prop.flags))
}

func (propertyDecoder) readBitCoord(reader *bit.BitReader) float32 {
	var intVal, fractVal int
	var res float32
	isNegative := false

	intVal = int(reader.ReadInt(1))
	fractVal = int(reader.ReadInt(1))

	if intVal|fractVal != 0 {
		isNegative = reader.ReadBit()
		if intVal == 1 {
			intVal = int(reader.ReadInt(coordIntegerBits) + 1)
		}

		if fractVal == 1 {
			fractVal = int(reader.ReadInt(coordFractionalBitsMp))
		}

		res = float32(intVal) + (float32(fractVal) * coordResolution)
	}

	if isNegative {
		res *= -1
	}

	return res
}

func (propertyDecoder) readBitCoordMp(reader *bit.BitReader, isIntegral bool, isLowPrecision bool) float32 {
	var res float32
	isNegative := false

	inBounds := reader.ReadBit()
	if isIntegral {
		if reader.ReadBit() {
			isNegative = reader.ReadBit()
			if inBounds {
				res = float32(reader.ReadInt(coordIntegerBitsMp) + 1)
			} else {
				res = float32(reader.ReadInt(coordIntegerBits) + 1)
			}
		}
	} else {
		readIntVal := reader.ReadBit()
		isNegative = reader.ReadBit()

		var intVal int
		if readIntVal {
			if inBounds {
				intVal = int(reader.ReadInt(coordIntegerBitsMp)) + 1
			} else {
				intVal = int(reader.ReadInt(coordIntegerBits)) + 1
			}
		}
		if isLowPrecision {
			res = float32(intVal) + (float32(reader.ReadInt(coordFractionalBitsMpLowPrecision)) * coordResolutionLowPrecision)
		} else {
			res = float32(intVal) + (float32(reader.ReadInt(coordFractionalBitsMp)) * coordResolution)
		}
	}

	if isNegative {
		res *= -1
	}

	return res
}

func (propertyDecoder) readBitNormal(reader *bit.BitReader) float32 {
	isNegative := reader.ReadBit()

	fractVal := reader.ReadInt(normalFractBits)

	res := float32(fractVal) * normalResolution

	if isNegative {
		res *= -1
	}

	return res
}

func (propertyDecoder) readBitCellCoord(reader *bit.BitReader, bits uint, isIntegral bool, isLowPrecision bool) float32 {
	var intVal, fractVal int
	var res float32

	if isIntegral {
		res = float32(reader.ReadInt(bits))
	} else {
		intVal = int(reader.ReadInt(bits))
		if isLowPrecision {
			fractVal = int(reader.ReadInt(coordFractionalBitsMpLowPrecision))

			res = float32(intVal) + (float32(fractVal) * (coordResolutionLowPrecision))
		} else {
			fractVal = int(reader.ReadInt(coordFractionalBitsMp))

			res = float32(intVal) + (float32(fractVal) * (coordResolution))
		}
	}

	return res
}

func (propertyDecoder) decodeVector(prop *sendTableProperty, reader *bit.BitReader) r3.Vector {
	res := r3.Vector{
		X: float64(propDecoder.decodeFloat(prop, reader)),
		Y: float64(propDecoder.decodeFloat(prop, reader)),
	}

	if !prop.flags.hasFlagSet(propFlagNormal) {
		res.Z = float64(propDecoder.decodeFloat(prop, reader))
	} else {
		absolute := res.X*res.X + res.Y*res.Y
		if absolute < 1.0 {
			res.Z = math.Sqrt(1 - absolute)
		} else {
			res.Z = 0
		}

		if reader.ReadBit() {
			res.Z *= -1
		}
	}

	return res
}

func (propertyDecoder) decodeArray(fProp *FlattenedPropEntry, reader *bit.BitReader) []PropValue {
	numElement := fProp.prop.numberOfElements

	var numBits uint = 1

	for maxElements := (numElement >> 1); maxElements != 0; maxElements = maxElements >> 1 {
		numBits++
	}

	nElements := int(reader.ReadInt(numBits))

	res := make([]PropValue, 0, nElements)

	tmp := &FlattenedPropEntry{prop: fProp.arrayElementProp}

	for i := 0; i < nElements; i++ {
		res = append(res, propDecoder.decodeProp(tmp, reader))
	}

	return res
}

func (propertyDecoder) decodeString(fProp *sendTableProperty, reader *bit.BitReader) string {
	length := int(reader.ReadInt(dataTableMaxStringBits))
	if length > dataTableMaxStringLength {
		length = dataTableMaxStringLength
	}
	return reader.ReadCString(length)
}

func (propertyDecoder) decodeVectorXY(prop *sendTableProperty, reader *bit.BitReader) r3.Vector {
	return r3.Vector{
		X: float64(propDecoder.decodeFloat(prop, reader)),
		Y: float64(propDecoder.decodeFloat(prop, reader)),
	}
}
