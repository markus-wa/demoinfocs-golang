package demoinfocs

import (
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"math"
)

type TeamState struct {
	id       int
	score    int
	clanName string
	flag     string
}

func (ts TeamState) Id() int {
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

func (v seVector) Angle2D() float64 {
	return math.Atan2(v.Y, v.X)
}

func (v seVector) Absolute() float64 {
	return math.Sqrt(v.AbsoluteSquared())
}

func (v seVector) AbsoluteSquared() float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
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

func (s split) ViewOrigin() seVector {
	if s.flags&fdemo_UseOrigin2 != 0 {
		return s.viewOrigin2
	}
	return s.viewOrigin
}

func (s split) ViewAngles() r3.Vector {
	if s.flags&fdemo_UseAngles2 != 0 {
		return s.viewAngles2
	}
	return s.viewAngles
}

func (s split) LocalViewAngles() r3.Vector {
	if s.flags&fdemo_UseAngles2 != 0 {
		return s.localViewAngles2
	}
	return s.localViewAngles
}

type commandInfo struct {
	splits [2]split
}

func (ci commandInfo) Splits() [2]split {
	return ci.splits
}

type BoundingBoxInformation struct {
	index int
	min   r3.Vector
	max   r3.Vector
}

func (bbi BoundingBoxInformation) contains(point r3.Vector) bool {
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
