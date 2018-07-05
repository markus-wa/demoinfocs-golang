package common

import (
	"math/rand"

	r3 "github.com/golang/geo/r3"

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

// FrameRate returns the frame rate of the demo (frames / demo-ticks per second).
// Not necessarily the tick-rate the server ran on during the game.
// VolvoPlx128TixKTnxBye
func (h DemoHeader) FrameRate() float32 {
	return float32(h.PlaybackFrames) / h.PlaybackTime
}

// FrameTime returns the time a frame / demo-tick takes in seconds.
func (h DemoHeader) FrameTime() float32 {
	return h.PlaybackTime / float32(h.PlaybackFrames)
}

// Player contains mostly game-relevant player information.
type Player struct {
	SteamID                     int64
	Position                    r3.Vector
	LastAlivePosition           r3.Vector
	Velocity                    r3.Vector
	EntityID                    int
	Name                        string
	Hp                          int
	Armor                       int
	Money                       int
	CurrentEquipmentValue       int
	FreezetimeEndEquipmentValue int
	RoundStartEquipmentValue    int
	ActiveWeaponID              int
	RawWeapons                  map[int]*Equipment
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
	Connected                   bool
}

// IsAlive returns true if the Hp of the player are > 0.
func (p *Player) IsAlive() bool {
	return p.Hp > 0
}

// ActiveWeapon returns the currently active / equipped weapon of the player.
func (p *Player) ActiveWeapon() *Equipment {
	return p.RawWeapons[p.ActiveWeaponID]
}

// Weapons returns all weapons in the player's possession
func (p *Player) Weapons() []*Equipment {
	res := make([]*Equipment, 0, len(p.RawWeapons))
	for _, w := range p.RawWeapons {
		res = append(res, w)
	}
	return res
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

// GrenadeProjectile is a grenade thrown intentionally by a player. It is used to track grenade projectile
// positions between the time at which they are thrown and until they detonate.
type GrenadeProjectile struct {
	EntityID int
	Weapon   EquipmentElement
	Thrower  *Player
	Owner    *Player

	// uniqueID is used to distinguish different grenades (which potentially have the same, reused entityID) from each other.
	uniqueID int64

	Position r3.Vector
}

// Class returns the class of the equipment.
// E.g. pistol, smg, heavy etc.
func (e Equipment) Class() EquipmentClass {
	return e.Weapon.Class()
}

// NewGrenadeProjectile creates a grenade projectile and sets.
func NewGrenadeProjectile() *GrenadeProjectile {
	return &GrenadeProjectile{uniqueID: rand.Int63()}
}

// UniqueID returns the internal id of the grenade, which is used to distinguish grenades that reuse another entity's entity id.
func (g GrenadeProjectile) UniqueID() int64 {
	return g.uniqueID
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

// NewPlayer creates a *Player with an initialized equipment map.
func NewPlayer() *Player {
	return &Player{RawWeapons: make(map[int]*Equipment)}
}
