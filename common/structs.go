package common

import (
	"encoding/binary"
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/st"
	"io"
)

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
	EntityId                    int
	SteamId                     int64
	Name                        string
	IsBot                       bool
	Position                    r3.Vector
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
	Entity                      *st.Entity
	IsDisconnected              bool
	ActiveWeaponId              int
	RawWeapons                  map[int]*Equipment
	Weapons                     []*Equipment
	Team                        Team
	HasDefuseKit                bool
	HasHelmet                   bool
	TeamId                      int
	AmmoLeft                    [32]int
	AdditionalPlayerInformation *AdditionalPlayerInformation
}

func (p *Player) IsAlive() bool {
	return p.Hp > 0
}

func (p *Player) ActiveWeapon() *Equipment {
	if p.ActiveWeaponId == IndexMask {
		return nil
	}
	return p.RawWeapons[p.ActiveWeaponId]
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
	Owner          *Player
	ReserveAmmo    int
}

func (e Equipment) Class() EquipmentClass {
	return EquipmentClass(int(e.Weapon) / 100)
}

func NewEquipment(originalString string) Equipment {
	var wep EquipmentElement
	if len(originalString) > 0 {
		wep = MapEquipment(originalString)
	} else {
		wep = EE_Unknown
	}
	return Equipment{Weapon: wep}
}

func NewSkinEquipment(originalString string, skin string) Equipment {
	var wep EquipmentElement
	if len(originalString) > 0 {
		wep = MapEquipment(originalString)
	} else {
		wep = EE_Unknown
	}
	return Equipment{Weapon: wep, SkinId: skin}
}

func ParsePlayerInfo(reader io.Reader) *PlayerInfo {
	br := bs.NewSmallBitReader(reader)
	res := &PlayerInfo{
		Version:     int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		XUID:        int64(binary.BigEndian.Uint64(br.ReadBytes(8))),
		Name:        br.ReadCString(128),
		UserId:      int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
		GUID:        br.ReadCString(33),
		FriendsId:   int(int32(binary.BigEndian.Uint32(br.ReadBytes(4)))),
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
