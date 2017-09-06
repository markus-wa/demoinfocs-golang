package common

import (
	"encoding/binary"
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitread"
	"github.com/markus-wa/demoinfocs-golang/st"
	"io"
)

// DemoHeader contains information about the demo's header.
type DemoHeader struct {
	Filestamp       string
	Protocol        int
	NetworkProtocol int
	ServerName      string
	ClientName      string
	MapName         string
	GameDirectory   string
	PlaybackTime    float32
	PlaybackTicks   int
	PlaybackFrames  int
	SignonLength    int
}

// PlayerInfo contains general player information
type PlayerInfo struct {
	Version     int64
	XUID        int64
	Name        string
	UserID      int
	GUID        string
	FriendsID   int
	FriendsName string
	// Custom files stuff (CRC)
	CustomFiles0 int
	CustomFiles1 int
	CustomFiles2 int
	CustomFiles3 int
	// Amount of downloaded files from the server
	FilesDownloaded byte
	// Bots
	IsFakePlayer bool
	// HLTV Proxy
	IsHltv bool
}

// Player contains mostly game-relevant player information.
type Player struct {
	SteamID                     int64
	Position                    r3.Vector
	LastAlivePosition           r3.Vector
	Velocity                    r3.Vector
	EntityID                    int
	TeamID                      int
	Name                        string
	Hp                          int
	Armor                       int
	Money                       int
	CurrentEquipmentValue       int
	FreezetimeEndEquipmentValue int
	RoundStartEquipmentValue    int
	ActiveWeaponID              int
	RawWeapons                  map[int]*Equipment
	Weapons                     []*Equipment
	AmmoLeft                    [32]int
	Entity                      *st.Entity
	AdditionalPlayerInformation *AdditionalPlayerInformation
	ViewDirectionX              float32
	ViewDirectionY              float32
	FlashDuration               float32
	Team                        Team
	IsBot                       bool
	IsDucking                   bool
	IsDisconnected              bool
	HasDefuseKit                bool
	HasHelmet                   bool
}

func (p *Player) IsAlive() bool {
	return p.Hp > 0
}

func (p *Player) ActiveWeapon() *Equipment {
	return p.RawWeapons[p.ActiveWeaponID]
}

// AdditionalPlayerInformation contains mostly scoreboard information.
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

// Equipment is a weapon / piece of equipment belonging to a player.
// This also includes the skin and some additional data.
type Equipment struct {
	EntityID       int
	Weapon         EquipmentElement
	OriginalString string
	SkinID         string
	AmmoInMagazine int
	AmmoType       int
	Owner          *Player
	ReserveAmmo    int
}

// Class returns the class of the equipment.
// E.g. pistol, smg, heavy etc.
func (e Equipment) Class() EquipmentClass {
	return EquipmentClass(int(e.Weapon) / 100)
}

func NewEquipment(originalString string) Equipment {
	return NewSkinEquipment(originalString, "")
}

func NewSkinEquipment(originalString string, skin string) Equipment {
	var wep EquipmentElement
	if len(originalString) > 0 {
		wep = MapEquipment(originalString)
	} else {
		wep = EE_Unknown
	}
	return Equipment{Weapon: wep, SkinID: skin}
}

func ParsePlayerInfo(reader io.Reader) *PlayerInfo {
	br := bs.NewSmallBitReader(reader)
	res := &PlayerInfo{
		Version:     int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		XUID:        int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		Name:        br.ReadCString(128),
		UserID:      int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		GUID:        br.ReadCString(33),
		FriendsID:   int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		FriendsName: br.ReadCString(128),

		IsFakePlayer: br.ReadSingleByte()&0xff != 0,
		IsHltv:       br.ReadSingleByte()&0xff != 0,

		CustomFiles0: int(br.ReadInt(32)),
		CustomFiles1: int(br.ReadInt(32)),
		CustomFiles2: int(br.ReadInt(32)),
		CustomFiles3: int(br.ReadInt(32)),

		FilesDownloaded: br.ReadSingleByte(),
	}
	br.Pool()
	return res
}

func NewPlayer() *Player {
	return &Player{RawWeapons: make(map[int]*Equipment)}
}
