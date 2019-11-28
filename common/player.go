package common

import (
	"time"

	"github.com/golang/geo/r3"

	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

const (
	maxEdictBits                 = 11
	entityHandleSerialNumberBits = 10
	entityHandleBits             = maxEdictBits + entityHandleSerialNumberBits
	invalidEntityHandle          = (1 << entityHandleBits) - 1
)

// Player contains mostly game-relevant player information.
type Player struct {
	demoInfoProvider demoInfoProvider // provider for demo info such as tick-rate or current tick

	SteamID                     int64     // int64 representation of the User's Steam ID
	Position                    r3.Vector // In-game coordinates. Like the one you get from cl_showpos 1
	LastAlivePosition           r3.Vector // The location where the player was last alive. Should be equal to Position if the player is still alive.
	Velocity                    r3.Vector // Movement velocity
	EntityID                    int       // The ID of the player-entity, see Entity field
	UserID                      int       // Mostly used in game-events to address this player
	Name                        string    // Steam / in-game user name
	Hp                          int
	Armor                       int
	Money                       int
	CurrentEquipmentValue       int
	FreezetimeEndEquipmentValue int
	RoundStartEquipmentValue    int
	ActiveWeaponID              int                // Used internally to set the active weapon, see ActiveWeapon()
	RawWeapons                  map[int]*Equipment // All weapons the player is currently carrying
	AmmoLeft                    [32]int            // Ammo left for special weapons (e.g. grenades), index corresponds Equipment.AmmoType
	Entity                      st.IEntity
	AdditionalPlayerInformation *AdditionalPlayerInformation // Mostly scoreboard information such as kills, deaths, etc.
	ViewDirectionX              float32
	ViewDirectionY              float32
	FlashDuration               float32    // Blindness duration from the flashbang currently affecting the player (seconds)
	FlashTick                   int        // In-game tick at which the player was last flashed
	TeamState                   *TeamState // When keeping the reference make sure you notice when the player changes teams
	Team                        Team
	IsBot                       bool
	IsConnected                 bool
	IsDucking                   bool
	IsDefusing                  bool
	IsPlanting                  bool
	IsReloading                 bool
	IsUnknown                   bool // Used to identify unknown/broken players. see https://github.com/markus-wa/demoinfocs-golang/issues/162
	HasDefuseKit                bool
	HasHelmet                   bool
}

// String returns the player's name.
// Implements fmt.Stringer.
func (p *Player) String() string {
	if p == nil {
		return "(nil)"
	}

	return p.Name
}

// IsAlive returns true if the Hp of the player are > 0.
func (p *Player) IsAlive() bool {
	return p.Hp > 0
}

// IsBlinded returns true if the player is currently flashed.
// This is more accurate than 'FlashDuration != 0' as it also takes into account FlashTick, DemoHeader.TickRate() and GameState.IngameTick().
func (p *Player) IsBlinded() bool {
	return p.FlashDurationTimeRemaining() > 0
}

// IsAirborne returns true if the player is jumping or falling.
func (p *Player) IsAirborne() bool {
	if p.Entity == nil {
		return false
	}

	groundEntityHandle := p.Entity.FindPropertyI("m_hGroundEntity").Value().IntVal

	return groundEntityHandle == invalidEntityHandle
}

// FlashDurationTime returns the duration of the blinding effect as time.Duration instead of float32 in seconds.
// Will return 0 if IsBlinded() returns false.
func (p *Player) FlashDurationTime() time.Duration {
	if !p.IsBlinded() {
		return time.Duration(0)
	}
	return p.flashDurationTimeFull()
}

func (p *Player) flashDurationTimeFull() time.Duration {
	return time.Duration(float32(time.Second) * p.FlashDuration)
}

// FlashDurationTimeRemaining returns the remaining duration of the blinding effect (or 0 if the player is not currently blinded).
// It takes into consideration FlashDuration, FlashTick, DemoHeader.TickRate() and GameState.IngameTick().
func (p *Player) FlashDurationTimeRemaining() time.Duration {
	// In case the demo header is broken
	// TODO: read tickRate from CVARs as fallback
	tickRate := p.demoInfoProvider.TickRate()
	if tickRate == 0 {
		return time.Duration(p.FlashDuration) * time.Second
	}

	timeSinceFlash := time.Duration(float64(p.demoInfoProvider.IngameTick()-p.FlashTick) / tickRate * float64(time.Second))
	remaining := p.flashDurationTimeFull() - timeSinceFlash
	if remaining < 0 {
		return 0
	}
	return remaining
}

/*
Some interesting data regarding flashes.

player time flash-duration
10 49m0.613347564s 0
10 49m50.54364714s 3.4198754
10 49m53.122207212s 3.8876143
10 49m54.84124726s 2.1688643
10 49m58.552811s 0

Going by the last two lines, the player should not have been blinded at ~49m57.0, but he was only cleared at ~49m58.5

This isn't very conclusive but it looks like IsFlashed isn't super reliable currently.
*/

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

// IsSpottedBy returns true if the player has been spotted by the other player.
func (p *Player) IsSpottedBy(other *Player) bool {
	if p.Entity == nil {
		return false
	}

	// TODO extract ClientSlot() function
	clientSlot := other.EntityID - 1
	bit := uint(clientSlot)
	var mask st.IProperty
	if bit < 32 {
		mask = p.Entity.FindPropertyI("m_bSpottedByMask.000")
	} else {
		bit -= 32
		mask = p.Entity.FindPropertyI("m_bSpottedByMask.001")
	}
	return (mask.Value().IntVal & (1 << bit)) != 0
}

// HasSpotted returns true if the player has spotted the other player.
func (p *Player) HasSpotted(other *Player) bool {
	return other.IsSpottedBy(p)
}

// IsInBombZone returns whether the player is currently in the bomb zone or not.
func (p *Player) IsInBombZone() bool {
	return p.Entity.FindPropertyI("m_bInBombZone").Value().IntVal == 1
}

// IsInBuyZone returns whether the player is currently in the buy zone or not.
func (p *Player) IsInBuyZone() bool {
	return p.Entity.FindPropertyI("m_bInBuyZone").Value().IntVal == 1
}

// IsWalking returns whether the player is currently walking (sneaking) in or not.
func (p *Player) IsWalking() bool {
	return p.Entity.FindPropertyI("m_bIsWalking").Value().IntVal == 1
}

// IsScoped returns whether the player is currently scoped in or not.
func (p *Player) IsScoped() bool {
	return p.Entity.FindPropertyI("m_bIsScoped").Value().IntVal == 1
}

// CashSpentThisRound returns the amount of cash the player spent in the current round.
//
// Deprecated, use Player.AdditionalPlayerInformation.CashSpentThisRound instead.
func (p *Player) CashSpentThisRound() int {
	return p.AdditionalPlayerInformation.CashSpentThisRound
}

// CashSpentTotal returns the amount of cash the player spent during the whole game up to the current point.
//
// Deprecated, use Player.AdditionalPlayerInformation.TotalCashSpent instead.
func (p *Player) CashSpentTotal() int {
	return p.AdditionalPlayerInformation.TotalCashSpent
}

// AdditionalPlayerInformation contains mostly scoreboard information.
type AdditionalPlayerInformation struct {
	Kills              int
	Deaths             int
	Assists            int
	Score              int
	MVPs               int
	Ping               int
	ClanTag            string
	TotalCashSpent     int
	CashSpentThisRound int
}

type demoInfoProvider interface {
	IngameTick() int   // current in-game tick, used for IsBlinded()
	TickRate() float64 // in-game tick rate, used for Player.IsBlinded()
	FindPlayerByHandle(handle int) *Player
}

// NewPlayer creates a *Player with an initialized equipment map.
//
// Intended for internal use only.
func NewPlayer(demoInfoProvider demoInfoProvider) *Player {
	return &Player{
		RawWeapons:       make(map[int]*Equipment),
		demoInfoProvider: demoInfoProvider,
	}
}
