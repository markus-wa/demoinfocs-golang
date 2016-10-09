package common

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/dt"
)

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
