// Package common contains common types, constants and functions used over different demoinfocs packages.
package common

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/golang/geo/r3"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
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

// GrenadeProjectile is a grenade thrown intentionally by a player. It is used to track grenade projectile
// positions between the time at which they are thrown and until they detonate.
type GrenadeProjectile struct {
	Entity         st.Entity
	WeaponInstance *Equipment
	Thrower        *Player // Always seems to be the same as Owner, even if the grenade was picked up
	Owner          *Player // Always seems to be the same as Thrower, even if the grenade was picked up

	Trajectory []TrajectoryEntry // List of all known locations and the point in time of the grenade up to the current point

	// uniqueID is used to distinguish different grenades (which potentially have the same, reused entityID) from each other.
	uniqueID int64
}

// Position returns the current position of the grenade projectile in world coordinates.
func (g *GrenadeProjectile) Position() r3.Vector {
	return g.Entity.Position()
}

// Velocity returns the projectile's velocity.
func (g *GrenadeProjectile) Velocity() r3.Vector {
	return g.Entity.PropertyValueMust("m_vecVelocity").R3Vec()
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
	return &GrenadeProjectile{uniqueID: rand.Int63()} //nolint:gosec
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
	team             Team
	membersCallback  func(Team) []*Player
	demoInfoProvider demoInfoProvider

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
	return int(getUInt64(ts.Entity, "m_iTeamNum"))
}

// Score returns the current score of the team (usually 0-16 without overtime).
func (ts *TeamState) Score() int {
	return getInt(ts.Entity, "m_iScore")
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

// MoneySpentTotal returns the total amount of cash spent by the whole team during the whole game up to the current point.
func (ts *TeamState) MoneySpentTotal() (value int) {
	for _, pl := range ts.Members() {
		value += pl.MoneySpentTotal()
	}

	return
}

// NewTeamState creates a new TeamState with the given Team and members callback function.
func NewTeamState(team Team, membersCallback func(Team) []*Player, demoInfoProvider demoInfoProvider) TeamState {
	return TeamState{
		team:             team,
		membersCallback:  membersCallback,
		demoInfoProvider: demoInfoProvider,
	}
}

// TrajectoryEntry represents the location of a grenade's trajectory at a specific point in time.
type TrajectoryEntry struct {
	Tick     int
	Position r3.Vector
	FrameID  int
	Time     time.Duration
}

// ConvertSteamIDTxtTo32 converts a Steam-ID in text format to a 32-bit variant.
// See https://developer.valvesoftware.com/wiki/SteamID
func ConvertSteamIDTxtTo32(steamID string) (uint32, error) {
	steamID = strings.TrimSuffix(steamID, "]") // Source 2 has [U:1:397560266] instead of STEAM_0:1:198780133

	arr := strings.Split(steamID, ":")

	if len(arr) != 3 {
		return 0, fmt.Errorf("SteamID '%s' not well formed", steamID)
	}

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

// Color is the type for the various colors constants.
type Color int

// Color constants give information about which color a player has.
const (
	Grey Color = iota - 1
	Yellow
	Purple
	Green
	Blue
	Orange
)

var strColors = map[Color]string{
	Grey:   "Grey",
	Yellow: "Yellow",
	Purple: "Purple",
	Green:  "Green",
	Blue:   "Blue",
	Orange: "Orange",
}

func (c Color) String() string {
	if _, exists := strColors[c]; !exists {
		return "Unknown-Color"
	}

	return strColors[c]
}

// ButtonBitMask represents the bitmask for player button states.
type ButtonBitMask uint64

// https://github.com/SteamDatabase/GameTracking-CS2/blob/master/DumpSource2/schemas/client/InputBitMask_t.h
const (
	ButtonNone          ButtonBitMask = 0x0
	ButtonAttack        ButtonBitMask = 0x1
	ButtonJump          ButtonBitMask = 0x2
	ButtonDuck          ButtonBitMask = 0x4
	ButtonForward       ButtonBitMask = 0x8
	ButtonBack          ButtonBitMask = 0x10
	ButtonUse           ButtonBitMask = 0x20
	ButtonTurnLeft      ButtonBitMask = 0x80
	ButtonTurnRight     ButtonBitMask = 0x100
	ButtonMoveLeft      ButtonBitMask = 0x200
	ButtonMoveRight     ButtonBitMask = 0x400
	ButtonAttack2       ButtonBitMask = 0x800
	ButtonReload        ButtonBitMask = 0x2000
	ButtonSpeed         ButtonBitMask = 0x10000
	ButtonJoyAutoSprint ButtonBitMask = 0x20000
	ButtonUseOrReload   ButtonBitMask = 0x100000000
	ButtonScore         ButtonBitMask = 0x200000000
	ButtonZoom          ButtonBitMask = 0x400000000
	ButtonLookAtWeapon  ButtonBitMask = 0x800000000
)
