package common

import (
	"encoding/binary"
	"io"

	r3 "github.com/golang/geo/r3"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
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

// IsAlive returns true if the Hp of the player are > 0.
func (p *Player) IsAlive() bool {
	return p.Hp > 0
}

// ActiveWeapon returns the currently active / equipped weapon of the player.
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

// NewEquipment is a wrapper for NewSkinEquipment to create weapons without skins.
func NewEquipment(eqName string) Equipment {
	return NewSkinEquipment(eqName, "")
}

// NewSkinEquipment creates an equipment with a skin from a skinID and equipment name.
func NewSkinEquipment(eqName string, skinID string) Equipment {
	var wep EquipmentElement
	if len(eqName) > 0 {
		wep = MapEquipment(eqName)
	} else {
		wep = EqUnknown
	}
	return Equipment{Weapon: wep, SkinID: skinID}
}

// ParsePlayerInfo parses player information from a byte stream.
func ParsePlayerInfo(reader io.Reader) *PlayerInfo {
	br := bit.NewSmallBitReader(reader)
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

// NewPlayer creates a *Player with an initialized equipment map.
func NewPlayer() *Player {
	return &Player{RawWeapons: make(map[int]*Equipment)}
}
