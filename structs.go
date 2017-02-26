package demoinfocs

import (
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitread"
)

type TeamState struct {
	id       int
	score    int
	clanName string
	flag     string
}

func (ts TeamState) ID() int {
	return ts.id
}

func (ts TeamState) Score() int {
	return ts.score
}

func (ts TeamState) ClanName() string {
	return ts.clanName
}

func (ts TeamState) Flag() string {
	return ts.flag
}

type seVector struct {
	r3.Vector
}

type split struct {
	flags int

	viewOrigin      seVector
	viewAngles      r3.Vector
	localViewAngles r3.Vector

	viewOrigin2      seVector
	viewAngles2      r3.Vector
	localViewAngles2 r3.Vector
}

// TODO: Find out what this is good for and why we didn't use the removed functions on seVector, split & commandInfo
type commandInfo struct {
	splits [2]split
}

type boundingBoxInformation struct {
	index int
	min   r3.Vector
	max   r3.Vector
}

func (bbi boundingBoxInformation) contains(point r3.Vector) bool {
	return point.X >= bbi.min.X && point.X <= bbi.max.X &&
		point.Y >= bbi.min.Y && point.Y <= bbi.max.Y &&
		point.Z >= bbi.min.Z && point.Z <= bbi.max.Z
}

type bombsiteInfo struct {
	index  int
	center r3.Vector
}

func parseCommandInfo(r *bs.BitReader) commandInfo {
	return commandInfo{splits: [2]split{parseSplit(r), parseSplit(r)}}
}

func parseSplit(r *bs.BitReader) split {
	return split{
		flags: r.ReadSignedInt(32),

		viewOrigin:      parseSEVector(r),
		viewAngles:      parseVector(r),
		localViewAngles: parseVector(r),

		viewOrigin2:      parseSEVector(r),
		viewAngles2:      parseVector(r),
		localViewAngles2: parseVector(r),
	}
}

func parseSEVector(r *bs.BitReader) seVector {
	return seVector{parseVector(r)}
}

func parseVector(r *bs.BitReader) r3.Vector {
	return r3.Vector{
		X: float64(r.ReadFloat()),
		Y: float64(r.ReadFloat()),
		Z: float64(r.ReadFloat()),
	}
}
