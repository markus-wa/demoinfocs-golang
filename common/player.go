package common

import (
	"time"

	r3 "github.com/golang/geo/r3"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// Player contains mostly game-relevant player information.
type Player struct {
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
	AmmoLeft                    [32]int            // Ammo left in the various weapons, index corresponds to key of RawWeapons
	Entity                      st.IEntity
	AdditionalPlayerInformation *AdditionalPlayerInformation // Mostly scoreboard information such as kills, deaths, etc.
	ViewDirectionX              float32
	ViewDirectionY              float32
	FlashDuration               float32 // Blindness duration from the flashbang currently affecting the player (seconds)
	FlashTick                   int     // In-game tick at which the player was last flashed
	Team                        Team
	TeamState                   *TeamState // When keeping the reference make sure you notice when the player changes teams
	IsBot                       bool
	IsConnected                 bool
	IsDucking                   bool
	IsDefusing                  bool
	HasDefuseKit                bool
	HasHelmet                   bool

	tickRate           float64            // the in-game tick rate, used for IsBlinded()
	ingameTickProvider ingameTickProvider // provider for the current in-game tick, used for IsBlinded()
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
	if p.tickRate == 0 {
		return time.Duration(p.FlashDuration) * time.Second
	}

	timeSinceFlash := time.Duration(float64(p.ingameTickProvider()-p.FlashTick) / p.tickRate * float64(time.Second))
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

// ActiveWeaponID returns the currently active / equipped weapon ID of the player.
func (p *Player) ActiveWeaponID() int {
	return int(p.RawWeapons[p.ActiveWeaponID].Weapon)
}

// Weapons returns all weapons in the player's possession.
func (p *Player) Weapons() []*Equipment {
	res := make([]*Equipment, 0, len(p.RawWeapons))
	for _, w := range p.RawWeapons {
		res = append(res, w)
	}
	return res
}

// WeaponIDs returns all weapon IDs in the player's possession.
func (p *Player) WeaponIDs() []int {
	res := make([]int, 0, len(p.RawWeapons))
	for _, w := range p.RawWeapons {
		res = append(res, int(w.Weapon))
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

// ingameTickProvider is a function that returns the current ingame tick of the demo related to a player.
type ingameTickProvider func() int

// NewPlayer creates a *Player with an initialized equipment map.
//
// Intended for internal use only.
func NewPlayer(tickRate float64, ingameTickProvider ingameTickProvider) *Player {
	return &Player{
		RawWeapons:         make(map[int]*Equipment),
		tickRate:           tickRate,
		ingameTickProvider: ingameTickProvider,
	}
}
