package common

import (
	"math/rand"
	"time"

	r3 "github.com/golang/geo/r3"
	s2 "github.com/golang/geo/s2"

	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// DemoHeader contains information from a demo's header.
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
func (h DemoHeader) FrameRate() float32 {
	return float32(h.PlaybackFrames) / h.PlaybackTime
}

// FrameTime returns the time a frame / demo-tick takes in seconds.
func (h DemoHeader) FrameTime() time.Duration {
	return time.Duration(h.PlaybackTime / float32(h.PlaybackFrames) * float32(time.Second))
}

// TickRate returns the tick-rate the server ran on during the game.
// VolvoPlx128TixKTnxBye
func (h DemoHeader) TickRate() float32 {
	return float32(h.PlaybackTicks) / h.PlaybackTime
}

// TickTime returns the time a single tick takes in seconds.
func (h DemoHeader) TickTime() time.Duration {
	return time.Duration(h.PlaybackTime / float32(h.PlaybackTicks) * float32(time.Second))
}

// Player contains mostly game-relevant player information.
type Player struct {
	SteamID                     int64
	Position                    r3.Vector
	LastAlivePosition           r3.Vector
	Velocity                    r3.Vector
	EntityID                    int
	UserID                      int
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

// Weapons returns all weapons in the player's possession.
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

	uniqueID int64
}

// GrenadeProjectile is a grenade thrown intentionally by a player. It is used to track grenade projectile
// positions between the time at which they are thrown and until they detonate.
type GrenadeProjectile struct {
	EntityID   int
	Weapon     EquipmentElement
	Thrower    *Player
	Owner      *Player
	Position   r3.Vector
	Trajectory []r3.Vector

	// uniqueID is used to distinguish different grenades (which potentially have the same, reused entityID) from each other.
	uniqueID int64
}

// Class returns the class of the equipment.
// E.g. pistol, smg, heavy etc.
func (e Equipment) Class() EquipmentClass {
	return e.Weapon.Class()
}

// UniqueID returns the unique id of the equipment element.
// The unique id is a random int generated internally by this library and can be used to differentiate
// equipment from each other. This is needed because demo-files reuse entity ids.
func (e Equipment) UniqueID() int64 {
	return e.uniqueID
}

// NewGrenadeProjectile creates a grenade projectile and sets the Unique-ID.
//
// Intended for internal use only.
func NewGrenadeProjectile() *GrenadeProjectile {
	return &GrenadeProjectile{uniqueID: rand.Int63()}
}

// UniqueID returns the unique id of the grenade.
// The unique id is a random int generated internally by this library and can be used to differentiate
// grenades from each other. This is needed because demo-files reuse entity ids.
func (g GrenadeProjectile) UniqueID() int64 {
	return g.uniqueID
}

// NewEquipment is a wrapper for NewSkinEquipment to create weapons without skins.
//
// Intended for internal use only.
func NewEquipment(eqName string) Equipment {
	return NewSkinEquipment(eqName, "")
}

// NewSkinEquipment creates an equipment with a skin from a skinID and equipment name.
//
// Intended for internal use only.
func NewSkinEquipment(eqName string, skinID string) Equipment {
	var wep EquipmentElement
	if len(eqName) > 0 {
		wep = MapEquipment(eqName)
	} else {
		wep = EqUnknown
	}
	return Equipment{Weapon: wep, SkinID: skinID, uniqueID: rand.Int63()}
}

// NewPlayer creates a *Player with an initialized equipment map.
//
// Intended for internal use only.
func NewPlayer() *Player {
	return &Player{RawWeapons: make(map[int]*Equipment)}
}

// Inferno is a list of Fires with helper functions.
// Also contains already extinguished fires.
//
// See also: Inferno.Active() and Fire.IsBurning
type Inferno struct {
	EntityID int
	Fires    []*Fire

	// uniqueID is used to distinguish different infernos (which potentially have the same, reused entityID) from each other.
	uniqueID int64
}

// Fire is a component of an Inferno.
type Fire struct {
	r3.Vector

	IsBurning bool
}

// Active returns an Inferno containing only the active fires of the original.
// The returned Inferno will have the same Unique-ID as the original.
func (inf Inferno) Active() Inferno {
	res := Inferno{
		uniqueID: inf.uniqueID,
	}
	res.Fires = make([]*Fire, 0, len(inf.Fires))
	for i := range inf.Fires {
		res.Fires = append(res.Fires, inf.Fires[i])
	}
	return res
}

// ConvexHull2D returns the 2D convex hull of all the fires in the inferno.
// Useful for drawing on 2D maps.
func (inf Inferno) ConvexHull2D() *s2.Loop {
	q := s2.NewConvexHullQuery()
	for i := range inf.Fires {
		q.AddPoint(s2.Point{
			Vector: r3.Vector{
				X: inf.Fires[i].Vector.X,
				Y: inf.Fires[i].Vector.Y,
				Z: 1,
			},
		})
	}
	return q.ConvexHull()
}

// ConvexHull3D returns the 3D convex hull of all the fires in the inferno.
func (inf Inferno) ConvexHull3D() *s2.Loop {
	q := s2.NewConvexHullQuery()
	for i := range inf.Fires {
		q.AddPoint(s2.Point{Vector: inf.Fires[i].Vector})
	}
	return q.ConvexHull()
}

// NewInferno creates a inferno and sets the Unique-ID.
//
// Intended for internal use only.
func NewInferno() *Inferno {
	return &Inferno{uniqueID: rand.Int63()}
}

// Bomb tracks the bomb's position, and the player carrying it, if any.
type Bomb struct {
	// Intended for internal use only. Use Position() instead.
	// Contains the last location of the dropped or planted bomb.
	LastOnGroundPosition r3.Vector
	Carrier              *Player
}

// Position returns the current position of the bomb.
// This is either the position of the player holding it
// or LastOnGroundPosition if it's dropped or planted.
func (b Bomb) Position() r3.Vector {
	if b.Carrier != nil {
		return b.Carrier.Position
	}

	return b.LastOnGroundPosition
}
