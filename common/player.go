package common

import (
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
	Entity                      *st.Entity
	AdditionalPlayerInformation *AdditionalPlayerInformation // Mostly scoreboard information such as kills, deaths, etc.
	ViewDirectionX              float32
	ViewDirectionY              float32
	FlashDuration               float32 // How long this player is flashed for from now on
	Team                        Team
	IsBot                       bool
	IsDucking                   bool
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

// NewPlayer creates a *Player with an initialized equipment map.
//
// Intended for internal use only.
func NewPlayer() *Player {
	return &Player{RawWeapons: make(map[int]*Equipment)}
}
