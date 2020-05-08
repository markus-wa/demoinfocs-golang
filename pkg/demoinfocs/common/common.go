// Package common contains common types, constants and functions used over different demoinfocs packages.
package common

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/golang/geo/r3"

	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
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
//
// Returns 0 if PlaybackTime or PlaybackFrames are 0 (corrupt demo headers).
func (h *DemoHeader) FrameRate() float64 {
	if h.PlaybackTime == 0 {
		return 0
	}

	return float64(h.PlaybackFrames) / h.PlaybackTime.Seconds()
}

// FrameTime returns the time a frame / demo-tick takes in seconds.
//
// Returns 0 if PlaybackTime or PlaybackFrames are 0 (corrupt demo headers).
func (h *DemoHeader) FrameTime() time.Duration {
	if h.PlaybackFrames == 0 {
		return 0
	}

	return time.Duration(h.PlaybackTime.Nanoseconds() / int64(h.PlaybackFrames))
}

// GrenadeProjectile is a grenade thrown intentionally by a player. It is used to track grenade projectile
// positions between the time at which they are thrown and until they detonate.
type GrenadeProjectile struct {
	Entity         st.Entity
	WeaponInstance *Equipment
	Thrower        *Player     // Always seems to be the same as Owner, even if the grenade was picked up
	Owner          *Player     // Always seems to be the same as Thrower, even if the grenade was picked up
	Trajectory     []r3.Vector // List of all known locations of the grenade up to the current point

	// uniqueID is used to distinguish different grenades (which potentially have the same, reused entityID) from each other.
	uniqueID int64
}

// Position returns the current position of the grenade projectile in world coordinates.
func (g *GrenadeProjectile) Position() r3.Vector {
	return g.Entity.Position()
}

// UniqueID returns the unique id of the grenade.
// The unique id is a random int generated internally by this library and can be used to differentiate
// grenades from each other. This is needed because demo-files reuse entity ids.
func (g *GrenadeProjectile) UniqueID() int64 {
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
func (b *Bomb) Position() r3.Vector {
	if b.Carrier != nil {
		return b.Carrier.Position()
	}

	return b.LastOnGroundPosition
}

// TeamState contains a team's ID, score, clan name & country flag.
type TeamState struct {
	team            Team
	membersCallback func(Team) []*Player

	Entity st.Entity

	// Terrorist TeamState for CTs, CT TeamState for Terrorists
	Opponent *TeamState
}

// Team returns the team for which the TeamState contains data.
func (ts *TeamState) Team() Team {
	return ts.team
}

// ID returns the team ID, this stays the same even after switching sides.
func (ts *TeamState) ID() int {
	return getInt(ts.Entity, "m_iTeamNum")
}

// Score returns the current score of the team (usually 0-16 without overtime).
func (ts *TeamState) Score() int {
	return getInt(ts.Entity, "m_scoreTotal")
}

// ClanName returns the team name (e.g. Fnatic).
func (ts *TeamState) ClanName() string {
	return getString(ts.Entity, "m_szClanTeamname")
}

// Flag returns the flag code (e.g. DE, FR, etc.).
//
// Watch out, in some demos this is upper-case and in some lower-case.
func (ts *TeamState) Flag() string {
	return getString(ts.Entity, "m_szTeamFlagImage")
}

// Members returns the players that are members of the team.
func (ts *TeamState) Members() []*Player {
	return ts.membersCallback(ts.team)
}

// CurrentEquipmentValue returns the cumulative value of all equipment currently owned by the members of the team.
func (ts *TeamState) CurrentEquipmentValue() (value int) {
	for _, pl := range ts.Members() {
		value += pl.EquipmentValueCurrent()
	}

	return
}

// RoundStartEquipmentValue returns the cumulative value of all equipment owned by the members of the team at the start of the current round.
func (ts *TeamState) RoundStartEquipmentValue() (value int) {
	for _, pl := range ts.Members() {
		value += pl.EquipmentValueRoundStart()
	}

	return
}

// FreezeTimeEndEquipmentValue returns the cumulative value of all equipment owned by the members of the team at the end of the freeze-time of the current round.
func (ts *TeamState) FreezeTimeEndEquipmentValue() (value int) {
	for _, pl := range ts.Members() {
		value += pl.EquipmentValueFreezeTimeEnd()
	}

	return
}

// MoneySpentThisRound returns the total amount of cash spent by the whole team in the current round.
func (ts *TeamState) MoneySpentThisRound() (value int) {
	for _, pl := range ts.Members() {
		value += pl.MoneySpentThisRound()
	}

	return
}

// MoneySpentThisRound returns the total amount of cash spent by the whole team during the whole game up to the current point.
func (ts *TeamState) MoneySpentTotal() (value int) {
	for _, pl := range ts.Members() {
		value += pl.MoneySpentTotal()
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

// ConvertSteamIDTxtTo32 converts a Steam-ID in text format to a 32-bit variant.
// See https://developer.valvesoftware.com/wiki/SteamID
func ConvertSteamIDTxtTo32(steamID string) (uint32, error) {
	arr := strings.Split(steamID, ":")

	Y, err := strconv.ParseUint(arr[1], 10, 32)
	if err != nil {
		return 0, err
	}

	Z, err := strconv.ParseUint(arr[2], 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32((Z << 1) + Y), nil
}

const steamID64IndividualIdentifier = 0x0110000100000000

// ConvertSteamID32To64 converts a Steam-ID in 32-bit format to a 64-bit variant.
// See https://developer.valvesoftware.com/wiki/SteamID
func ConvertSteamID32To64(steamID32 uint32) uint64 {
	return steamID64IndividualIdentifier + uint64(steamID32)
}

// ConvertSteamID64To32 converts a Steam-ID in 64-bit format to a 32-bit variant.
// See https://developer.valvesoftware.com/wiki/SteamID
func ConvertSteamID64To32(steamID64 uint64) uint32 {
	return uint32(steamID64 - steamID64IndividualIdentifier)
}
