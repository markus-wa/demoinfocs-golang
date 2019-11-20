// Package common contains common types, constants and functions used over different demoinfocs packages.
package common

import (
	"math/rand"
	"time"

	"github.com/golang/geo/r3"
)

// Team is the type for the various TeamXYZ constants.
type Team byte

// Team constants give information about which team a player is on.
const (
	TeamUnassigned        Team = 0
	TeamSpectators        Team = 1
	TeamTerrorists        Team = 2
	TeamCounterTerrorists Team = 3
)

// DemoHeader contains information from a demo's header.
type DemoHeader struct {
	Filestamp       string        // aka. File-type, must be HL2DEMO
	Protocol        int           // Should be 4
	NetworkProtocol int           // Not sure what this is for
	ServerName      string        // Server's 'hostname' config value
	ClientName      string        // Usually 'GOTV Demo'
	MapName         string        // E.g. de_cache, de_nuke, cs_office, etc.
	GameDirectory   string        // Usually 'csgo'
	PlaybackTime    time.Duration // Demo duration in seconds (= PlaybackTicks / Server's tickrate)
	PlaybackTicks   int           // Game duration in ticks (= PlaybackTime * Server's tickrate)
	PlaybackFrames  int           // Amount of 'frames' aka demo-ticks recorded (= PlaybackTime * Demo's recording rate)
	SignonLength    int           // Length of the Signon package in bytes
}

// FrameRate returns the frame rate of the demo (frames / demo-ticks per second).
// Not necessarily the tick-rate the server ran on during the game.
func (h DemoHeader) FrameRate() float64 {
	return float64(h.PlaybackFrames) / h.PlaybackTime.Seconds()
}

// FrameTime returns the time a frame / demo-tick takes in seconds.
//
// Returns 0 if PlaybackTime or PlaybackFrames are 0 (corrupt demo headers).
func (h DemoHeader) FrameTime() time.Duration {
	if h.PlaybackFrames == 0 {
		return 0
	}
	return time.Duration(h.PlaybackTime.Nanoseconds() / int64(h.PlaybackFrames))
}

// TickRate returns the tick-rate the server ran on during the game.
// VolvoPlx128TixKTnxBye
func (h DemoHeader) TickRate() float64 {
	return float64(h.PlaybackTicks) / h.PlaybackTime.Seconds()
}

// TickTime returns the time a single tick takes in seconds.
func (h DemoHeader) TickTime() time.Duration {
	return time.Duration(h.PlaybackTime.Nanoseconds() / int64(h.PlaybackTicks))
}

// GrenadeProjectile is a grenade thrown intentionally by a player. It is used to track grenade projectile
// positions between the time at which they are thrown and until they detonate.
type GrenadeProjectile struct {
	EntityID   int
	// IMPORTANT: I Switched Weapon from EquipmentElement to Equipment type, this means it could break user's code (feel free to do otherwise)...
	Weapon     Equipment
	Thrower    *Player // Always seems to be the same as Owner, even if the grenade was picked up
	Owner      *Player // Always seems to be the same as Thrower, even if the grenade was picked up
	Position   r3.Vector
	Trajectory []r3.Vector // List of all known locations of the grenade up to the current point

	// uniqueID is used to distinguish different grenades (which potentially have the same, reused entityID) from each other.
	uniqueID int64
}

// UniqueID returns the unique id of the grenade.
// The unique id is a random int generated internally by this library and can be used to differentiate
// grenades from each other. This is needed because demo-files reuse entity ids.
func (g GrenadeProjectile) UniqueID() int64 {
	return g.uniqueID
}

// NewGrenadeProjectile creates a grenade projectile and sets the Unique-ID.
//
// Intended for internal use only.
func NewGrenadeProjectile() *GrenadeProjectile {
	return &GrenadeProjectile{uniqueID: rand.Int63()}
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

// TeamState contains a team's ID, score, clan name & country flag.
type TeamState struct {
	team            Team
	membersCallback func(Team) []*Player

	// ID stays the same even after switching sides.
	ID int

	Score    int
	ClanName string

	// Flag, e.g. DE, FR, etc.
	//
	// Watch out, in some demos this is upper-case and in some lower-case.
	Flag string

	// Terrorist TeamState for CTs, CT TeamState for Terrorists
	Opponent *TeamState
}

// Team returns the team for which the TeamState contains data.
func (ts TeamState) Team() Team {
	return ts.team
}

// Members returns the members of the team.
func (ts TeamState) Members() []*Player {
	return ts.membersCallback(ts.team)
}

// CurrentEquipmentValue returns the cumulative value of all equipment currently owned by the members of the team.
func (ts TeamState) CurrentEquipmentValue() (value int) {
	for _, pl := range ts.Members() {
		value += pl.CurrentEquipmentValue
	}
	return
}

// RoundStartEquipmentValue returns the cumulative value of all equipment owned by the members of the team at the start of the current round.
func (ts TeamState) RoundStartEquipmentValue() (value int) {
	for _, pl := range ts.Members() {
		value += pl.RoundStartEquipmentValue
	}
	return
}

// FreezeTimeEndEquipmentValue returns the cumulative value of all equipment owned by the members of the team at the end of the freeze-time of the current round.
func (ts TeamState) FreezeTimeEndEquipmentValue() (value int) {
	for _, pl := range ts.Members() {
		value += pl.FreezetimeEndEquipmentValue
	}
	return
}

// CashSpentThisRound returns the total amount of cash spent by the whole team in the current round.
func (ts TeamState) CashSpentThisRound() (value int) {
	for _, pl := range ts.Members() {
		value += pl.AdditionalPlayerInformation.CashSpentThisRound
	}
	return
}

// CashSpentThisRound returns the total amount of cash spent by the whole team during the whole game up to the current point.
func (ts TeamState) CashSpentTotal() (value int) {
	for _, pl := range ts.Members() {
		value += pl.AdditionalPlayerInformation.TotalCashSpent
	}
	return
}

// NewTeamState creates a new TeamState with the given Team and members callback function.
func NewTeamState(team Team, membersCallback func(Team) []*Player) TeamState {
	return TeamState{
		team:            team,
		membersCallback: membersCallback,
	}
}
