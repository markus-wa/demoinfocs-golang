package main

import (
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	dt "github.com/markus-wa/demoinfocs-golang/dt"
	"math"
)

// FIXME: Use OS value
const MaxOsPath = 260

type DemoHeader struct {
	filestamp       string
	protocol        int
	networkProtocol int
	serverName      string
	clientName      string
	mapName         string
	gameDirectory   string
	playbackTime    float32
	playbackTicks   int
	playbackFrames  int
	signonLength    int
}

func (dh DemoHeader) Filestamp() string {
	return dh.filestamp
}

func (dh DemoHeader) Protocol() int {
	return dh.protocol
}

func (dh DemoHeader) NetworkProtocol() int {
	return dh.networkProtocol
}

func (dh DemoHeader) ServerName() string {
	return dh.serverName
}

func (dh DemoHeader) ClientName() string {
	return dh.clientName
}

func (dh DemoHeader) MapName() string {
	return dh.mapName
}

func (dh DemoHeader) GameDirectory() string {
	return dh.gameDirectory
}

func (dh DemoHeader) PlaybackTime() float32 {
	return dh.playbackTime
}

func (dh DemoHeader) PlaybackTicks() int {
	return dh.playbackTicks
}

func (dh DemoHeader) PlaybackFrames() int {
	return dh.playbackFrames
}

func (dh DemoHeader) SignonLenght() int {
	return dh.signonLength
}

type PlayerInfo struct {
	Version     int64
	XUID        int64
	Name        string
	UserId      int
	GUID        string
	FriendsId   int
	FriendsName string
	// Bots
	IsFakePlayer bool
	// HLTV Proxy
	IsHltv bool
	// Custom files stuff (CRC)
	CustomFiles0 int
	CustomFiles1 int
	CustomFiles2 int
	CustomFiles3 int
	// Amount of downloaded files from the server
	FilesDownloaded byte
}

type Player struct {
	Name                        string
	SteamId                     int64
	Position                    r3.Vector
	EntityId                    int
	Hp                          int
	Armor                       int
	LastAlivePosition           r3.Vector
	Velocity                    r3.Vector
	ViewDirectionX              float32
	ViewDirectionY              float32
	FlashDuration               float32
	Money                       int
	CurrentEquipmentValue       int
	FreezetimeEndEquipmentValue int
	RoundStartEquipmentValue    int
	IsDucking                   bool
	Entity                      dt.Entity
	IsDisconnected              bool
	ActiveWeaponId              int
	rawWeapons                  map[int]Equipment
	Weapons                     []Equipment
	Team                        Team
	HasDefuseKit                bool
	TeamId                      int
	AmmoLeft                    []int
	AdditionalPlayerInformation AdditionalPlayerInformation
}

func (p Player) IsAlive() bool {
	return p.Hp > 0
}

type AdditionalPlayerInformation struct {
	Kills          int
	Deaths         int
	Assists        int
	Score          int
	MVPs           int
	Ping           int
	ClanTag        string
	TotalCashSpent int
}

type Equipment struct {
	EntityId       int
	Weapon         EquipmentElement
	OriginalString string
	SkinId         string
	AmmoInMagazine int
	AmmoType       int
	Owner          Player
	ReserveAmmo    int
}

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

func (e Equipment) Class() EquipmentClass {
	return EquipmentClass(int(e.Weapon) / 100)
}

func NewEquipment() *Equipment {
	return &Equipment{Weapon: EE_Unknown}
}

type SEVector struct {
	*r3.Vector
}

func (v *SEVector) Angle2D() float64 {
	return math.Atan2(v.Y, v.X)
}

func (v *SEVector) Absolute() float64 {
	return math.Sqrt(v.AbsoluteSquared())
}

func (v *SEVector) AbsoluteSquared() float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

type Split struct {
	flags int

	viewOrigin      *SEVector
	viewAngles      *r3.Vector
	localViewAngles *r3.Vector

	viewOrigin2      *SEVector
	viewAngles2      *r3.Vector
	localViewAngles2 *r3.Vector
}

func (s *Split) ViewOrigin() *SEVector {
	if s.flags&FDEMO_USE_ORIGIN2 != 0 {
		return s.viewOrigin2
	}
	return s.viewOrigin
}

func (s *Split) ViewAngles() *r3.Vector {
	if s.flags&FDEMO_USE_ANGLES2 != 0 {
		return s.viewAngles2
	}
	return s.viewAngles
}

func (s *Split) LocalViewAngles() *r3.Vector {
	if s.flags&FDEMO_USE_ANGLES2 != 0 {
		return s.localViewAngles2
	}
	return s.localViewAngles
}

type CommandInfo struct {
	splits [2]*Split
}

func (ci *CommandInfo) Splits() [2]*Split {
	return ci.splits
}

func parseCommandInfo(r bs.BitReader) *CommandInfo {
	return &CommandInfo{splits: [2]*Split{parseSplit(r), parseSplit(r)}}
}

func parseSplit(r bs.BitReader) *Split {
	s := &Split{}
	s.flags = r.ReadSignedInt(32)

	s.viewOrigin = parseSEVector(r)
	s.viewAngles = parseVector(r)
	s.localViewAngles = parseVector(r)

	s.viewOrigin2 = parseSEVector(r)
	s.viewAngles2 = parseVector(r)
	s.localViewAngles2 = parseVector(r)
	return s
}

func parseSEVector(r bs.BitReader) *SEVector {
	return &SEVector{parseVector(r)}
}

func parseVector(r bs.BitReader) *r3.Vector {
	v := &r3.Vector{}
	v.X = float64(r.ReadFloat())
	v.Y = float64(r.ReadFloat())
	v.Z = float64(r.ReadFloat())
	return v
}
