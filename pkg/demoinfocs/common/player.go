package common

import (
	"fmt"
	"time"

	"github.com/golang/geo/r3"
	"github.com/pkg/errors"

	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/sendtables"
)

// Player contains mostly game-relevant player information.
type Player struct {
	demoInfoProvider demoInfoProvider // provider for demo info such as tick-rate or current tick

	SteamID64         uint64             // 64-bit representation of the user's Steam ID. See https://developer.valvesoftware.com/wiki/SteamID
	LastAlivePosition r3.Vector          // The location where the player was last alive. Should be equal to Position if the player is still alive.
	UserID            int                // Mostly used in game-events to address this player
	Name              string             // Steam / in-game user name
	Inventory         map[int]*Equipment // All weapons / equipment the player is currently carrying. See also Weapons().
	AmmoLeft          [32]int            // Ammo left for special weapons (e.g. grenades), index corresponds Equipment.AmmoType
	EntityID          int                // Usually the same as Entity.ID() but may be different between player death and re-spawn.
	Entity            st.Entity          // May be nil between player-death and re-spawn
	FlashDuration     float32            // Blindness duration from the flashbang currently affecting the player (seconds)
	FlashTick         int                // In-game tick at which the player was last flashed
	TeamState         *TeamState         // When keeping the reference make sure you notice when the player changes teams
	Team              Team               // Team identifier for the player (e.g. TeamTerrorists or TeamCounterTerrorists).
	IsBot             bool               // True if this is a bot-entity. See also IsControllingBot and ControlledBot().
	IsConnected       bool
	IsDefusing        bool
	IsPlanting        bool
	IsReloading       bool
	IsUnknown         bool // Used to identify unknown/broken players. see https://github.com/markus-wa/demoinfocs-golang/issues/162
}

// String returns the player's name.
// Implements fmt.Stringer.
func (p *Player) String() string {
	if p == nil {
		return "(nil)"
	}

	return p.Name
}

// SteamID32 converts SteamID64 to the 32-bit SteamID variant and returns the result.
// See https://developer.valvesoftware.com/wiki/SteamID
func (p *Player) SteamID32() uint32 {
	return ConvertSteamID64To32(p.SteamID64)
}

// IsAlive returns true if the Hp of the player are > 0.
func (p *Player) IsAlive() bool {
	return p.Health() > 0
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

	groundEntityHandle := p.Entity.Property("m_hGroundEntity").Value().IntVal

	return groundEntityHandle == constants.InvalidEntityHandle
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
	tickRate := p.demoInfoProvider.TickRate()
	if tickRate == 0 {
		return p.flashDurationTimeFull()
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

// Used internally to set the active weapon, see ActiveWeapon()
func (p *Player) activeWeaponID() int {
	return getInt(p.Entity, "m_hActiveWeapon") & constants.EntityHandleIndexMask
}

// ActiveWeapon returns the currently active / equipped weapon of the player.
func (p *Player) ActiveWeapon() *Equipment {
	return p.demoInfoProvider.FindWeaponByEntityID(p.activeWeaponID())
}

// Weapons returns all weapons in the player's possession.
// Contains all entries from Player.Inventory but as a slice instead of a map.
func (p *Player) Weapons() []*Equipment {
	res := make([]*Equipment, 0, len(p.Inventory))
	for _, w := range p.Inventory {
		res = append(res, w)
	}

	return res
}

// IsSpottedBy returns true if the player has been spotted by the other player.
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (p *Player) IsSpottedBy(other *Player) bool {
	if p.Entity == nil {
		return false
	}

	clientSlot := other.EntityID - 1
	bit := uint(clientSlot)

	var mask st.Property
	if bit < 32 {
		mask = p.Entity.Property("m_bSpottedByMask.000")
	} else {
		bit -= 32
		mask = p.Entity.Property("m_bSpottedByMask.001")
	}

	return (mask.Value().IntVal & (1 << bit)) != 0
}

// HasSpotted returns true if the player has spotted the other player.
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (p *Player) HasSpotted(other *Player) bool {
	return other.IsSpottedBy(p)
}

// IsInBombZone returns whether the player is currently in the bomb zone or not.
func (p *Player) IsInBombZone() bool {
	return getBool(p.Entity, "m_bInBombZone")
}

// IsInBuyZone returns whether the player is currently in the buy zone or not.
func (p *Player) IsInBuyZone() bool {
	return getBool(p.Entity, "m_bInBuyZone")
}

// IsWalking returns whether the player is currently walking (sneaking) in or not.
func (p *Player) IsWalking() bool {
	return getBool(p.Entity, "m_bIsWalking")
}

// IsScoped returns whether the player is currently scoped in or not.
func (p *Player) IsScoped() bool {
	return getBool(p.Entity, "m_bIsScoped")
}

// IsDucking returns true if the player is currently fully crouching.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsDucking() bool {
	return p.Flags().Ducking() && p.Flags().DuckingKeyPressed()
}

// IsDuckingInProgress returns true if the player is currently in the progress of going from standing to crouched.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsDuckingInProgress() bool {
	return !p.Flags().Ducking() && p.Flags().DuckingKeyPressed()
}

// IsUnDuckingInProgress returns true if the player is currently in the progress of going from crouched to standing.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsUnDuckingInProgress() bool {
	return p.Flags().Ducking() && !p.Flags().DuckingKeyPressed()
}

// IsStanding returns true if the player is currently fully standing upright.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsStanding() bool {
	return !p.Flags().Ducking() && !p.Flags().DuckingKeyPressed()
}

// HasDefuseKit returns true if the player currently has a defuse kit in his inventory.
func (p *Player) HasDefuseKit() bool {
	return getBool(p.Entity, "m_bHasDefuser")
}

// HasHelmet returns true if the player is currently wearing head armor.
func (p *Player) HasHelmet() bool {
	return getBool(p.Entity, "m_bHasHelmet")
}

// IsControllingBot returns true if the player is currently controlling a bot.
// See also ControlledBot().
func (p *Player) IsControllingBot() bool {
	return getBool(p.Entity, "m_bIsControllingBot")
}

// ControlledBot returns the player instance of the bot that the player is controlling, if any.
// Returns nil if the player is not controlling a bot.
func (p *Player) ControlledBot() *Player {
	if p.Entity == nil {
		return nil
	}

	botHandle := p.Entity.Property("m_iControlledBotEntIndex").Value().IntVal

	return p.demoInfoProvider.FindPlayerByHandle(botHandle)
}

// Health returns the player's health points, normally 0-100.
func (p *Player) Health() int {
	return getInt(p.Entity, "m_iHealth")
}

// Armor returns the player's armor points, normally 0-100.
func (p *Player) Armor() int {
	return getInt(p.Entity, "m_ArmorValue")
}

// Money returns the amount of money in the player's bank.
func (p *Player) Money() int {
	return getInt(p.Entity, "m_iAccount")
}

// EquipmentValueCurrent returns the current value of equipment in the player's inventory.
func (p *Player) EquipmentValueCurrent() int {
	return getInt(p.Entity, "m_unCurrentEquipmentValue")
}

// EquipmentValueRoundStart returns the value of equipment in the player's inventory at the time of the round start.
// This is before the player has bought any new items in the freeze time.
// See also Player.EquipmentValueFreezetimeEnd().
func (p *Player) EquipmentValueRoundStart() int {
	return getInt(p.Entity, "m_unRoundStartEquipmentValue")
}

// EquipmentValueFreezeTimeEnd returns the value of equipment in the player's inventory at the end of the freeze time.
func (p *Player) EquipmentValueFreezeTimeEnd() int {
	return getInt(p.Entity, "m_unFreezetimeEndEquipmentValue")
}

// ViewDirectionX returns the Yaw value in degrees, 0 to 360.
func (p *Player) ViewDirectionX() float32 {
	return getFloat(p.Entity, "m_angEyeAngles[1]")
}

// ViewDirectionY returns the Pitch value in degrees, 270 to 90 (270=-90).
func (p *Player) ViewDirectionY() float32 {
	return getFloat(p.Entity, "m_angEyeAngles[0]")
}

// Position returns the in-game coordinates.
// Note: the Z value is not on the player's eye height but instead at his feet.
// See also PositionEyes().
func (p *Player) Position() r3.Vector {
	if p.Entity == nil {
		return r3.Vector{}
	}

	return p.Entity.Position()
}

// PositionEyes returns the player's position with the Z value at eye height.
// This is what you get from cl_showpos 1.
// See lso Position().
func (p *Player) PositionEyes() r3.Vector {
	if p.Entity == nil {
		return r3.Vector{}
	}

	pos := p.Position()
	pos.Z += float64(p.Entity.PropertyValueMust("localdata.m_vecViewOffset[2]").FloatVal)

	return pos
}

// Velocity returns the player's velocity.
func (p *Player) Velocity() r3.Vector {
	if p.Entity == nil {
		return r3.Vector{}
	}

	return r3.Vector{
		X: float64(p.Entity.PropertyValueMust("localdata.m_vecVelocity[0]").FloatVal),
		Y: float64(p.Entity.PropertyValueMust("localdata.m_vecVelocity[1]").FloatVal),
		Z: float64(p.Entity.PropertyValueMust("localdata.m_vecVelocity[2]").FloatVal),
	}
}

// see https://github.com/ValveSoftware/source-sdk-2013/blob/master/mp/src/public/const.h#L146-L188
const (
	flOnGround = 1 << iota
	flDucking
	flAnimDucking
)

// PlayerFlags wraps m_fFlags and provides accessors for the various known flags a player may have set.
type PlayerFlags uint32

func (pf PlayerFlags) Get(f PlayerFlags) bool {
	return pf&f != 0
}

// OnGround returns true if the player is touching the ground.
// See m_fFlags FL_ONGROUND https://github.com/ValveSoftware/source-sdk-2013/blob/master/mp/src/public/const.h#L146-L188
func (pf PlayerFlags) OnGround() bool {
	return pf.Get(flOnGround)
}

// Ducking returns true if the player is/was fully crouched.
//
//	Fully ducked: Ducking() && DuckingKeyPressed()
//	Previously fully ducked, unducking in progress: Ducking() && !DuckingKeyPressed()
//	Fully unducked: !Ducking() && !DuckingKeyPressed()
//	Previously fully unducked, ducking in progress: !Ducking() && DuckingKeyPressed()
//
// See m_fFlags FL_DUCKING https://github.com/ValveSoftware/source-sdk-2013/blob/master/mp/src/public/const.h#L146-L188
func (pf PlayerFlags) Ducking() bool {
	return pf.Get(flDucking)
}

// DuckingKeyPressed returns true if the player is holding the crouch key pressed.
//
//	Fully ducked: Ducking() && DuckingKeyPressed()
//	Previously fully ducked, unducking in progress: Ducking() && !DuckingKeyPressed()
//	Fully unducked: !Ducking() && !DuckingKeyPressed()
//	Previously fully unducked, ducking in progress: !Ducking() && DuckingKeyPressed()
//
// See m_fFlags FL_ANIMDUCKING https://github.com/ValveSoftware/source-sdk-2013/blob/master/mp/src/public/const.h#L146-L188
func (pf PlayerFlags) DuckingKeyPressed() bool {
	return pf.Get(flAnimDucking)
}

// Flags returns flags currently set on m_fFlags.
func (p *Player) Flags() PlayerFlags {
	return PlayerFlags(getInt(p.Entity, "m_fFlags"))
}

// //////////////////////////
// CCSPlayerResource stuff //
// //////////////////////////

func (p *Player) entityIDStr() string {
	return fmt.Sprintf("%03d", p.EntityID)
}

func (p *Player) resourceEntity() st.Entity {
	return p.demoInfoProvider.PlayerResourceEntity()
}

// ClanTag returns the player's individual clan tag (Steam Groups etc.).
func (p *Player) ClanTag() string {
	return getString(p.resourceEntity(), "m_szClan."+p.entityIDStr())
}

// CrosshairCode returns the player's crosshair code or an empty string if there isn't one.
func (p *Player) CrosshairCode() string {
	if p.resourceEntity() == nil {
		return ""
	}

	// if the property doesn't exist we return empty string by default
	val, _ := p.resourceEntity().PropertyValue("m_szCrosshairCodes." + p.entityIDStr())

	return val.StringVal
}

// Ping returns the players latency to the game server.
func (p *Player) Ping() int {
	return getInt(p.resourceEntity(), "m_iPing."+p.entityIDStr())
}

// Score returns the players score as shown on the scoreboard.
func (p *Player) Score() int {
	return getInt(p.resourceEntity(), "m_iScore."+p.entityIDStr())
}

// Color returns the players color as shown on the minimap.
// It will return Grey (-1) if the resource entity does not exist when the function is called or when the demo does not support player colors.
// Deprecated: Use ColorOrErr() instead.
func (p *Player) Color() Color {
	resourceEnt := p.resourceEntity()
	if resourceEnt == nil {
		return Grey
	}

	n, ok := resourceEnt.PropertyValue("m_iCompTeammateColor." + p.entityIDStr())
	if !ok {
		return Grey
	}

	return Color(n.IntVal)
}

var (
	ErrDataNotAvailable   = errors.New("some data is not (yet) available (reading the same data later during parsing may work)")
	ErrNotSupportedByDemo = errors.New("this data is not supported by the demo (this may be because the demos is too old)")
)

// ColorOrErr returns the players color as shown on the minimap.
// Returns ErrDataNotAvailable if the resource entity does not exist (it may exist later during parsing).
// Returns ErrNotSupportedByDemo if the demo does not support player colors (e.g. very old demos).
func (p *Player) ColorOrErr() (Color, error) {
	resourceEnt := p.resourceEntity()
	if resourceEnt == nil {
		return Grey, errors.Wrap(ErrDataNotAvailable, "player resource entity is nil")
	}

	colorVal, ok := resourceEnt.PropertyValue("m_iCompTeammateColor." + p.entityIDStr())
	if !ok {
		return Grey, errors.Wrap(ErrNotSupportedByDemo, "failed to get player color from resource entity")
	}

	return Color(colorVal.IntVal), nil
}

// Kills returns the amount of kills the player has as shown on the scoreboard.
func (p *Player) Kills() int {
	return getInt(p.resourceEntity(), "m_iKills."+p.entityIDStr())
}

// Deaths returns the amount of deaths the player has as shown on the scoreboard.
func (p *Player) Deaths() int {
	return getInt(p.resourceEntity(), "m_iDeaths."+p.entityIDStr())
}

// Assists returns the amount of assists the player has as shown on the scoreboard.
func (p *Player) Assists() int {
	return getInt(p.resourceEntity(), "m_iAssists."+p.entityIDStr())
}

// MVPs returns the amount of Most-Valuable-Player awards the player has as shown on the scoreboard.
func (p *Player) MVPs() int {
	return getInt(p.resourceEntity(), "m_iMVPs."+p.entityIDStr())
}

// TotalDamage returns the total health damage done by the player.
func (p *Player) TotalDamage() int {
	return getInt(p.resourceEntity(), "m_iMatchStats_Damage_Total."+p.entityIDStr())
}

// UtilityDamage returns the total damage done by the player with grenades.
func (p *Player) UtilityDamage() int {
	return getInt(p.resourceEntity(), "m_iMatchStats_UtilityDamage_Total."+p.entityIDStr())
}

// MoneySpentTotal returns the total amount of money the player has spent in the current match.
func (p *Player) MoneySpentTotal() int {
	return getInt(p.resourceEntity(), "m_iTotalCashSpent."+p.entityIDStr())
}

// MoneySpentThisRound returns the amount of money the player has spent in the current round.
func (p *Player) MoneySpentThisRound() int {
	return getInt(p.resourceEntity(), "m_iCashSpentThisRound."+p.entityIDStr())
}

// LastPlaceName returns the string value of the player's position.
func (p *Player) LastPlaceName() string {
	return getString(p.Entity, "m_szLastPlaceName")
}

type demoInfoProvider interface {
	IngameTick() int   // current in-game tick, used for IsBlinded()
	TickRate() float64 // in-game tick rate, used for Player.IsBlinded()
	FindPlayerByHandle(handle int) *Player
	PlayerResourceEntity() st.Entity
	FindWeaponByEntityID(id int) *Equipment
}

// NewPlayer creates a *Player with an initialized equipment map.
//
// Intended for internal use only.
func NewPlayer(demoInfoProvider demoInfoProvider) *Player {
	return &Player{
		Inventory:        make(map[int]*Equipment),
		demoInfoProvider: demoInfoProvider,
	}
}

// PlayerInfo contains information about a player such as their name and SteamID.
// Primarily intended for internal use.
type PlayerInfo struct {
	Version     int64
	XUID        uint64 // SteamID64
	Name        string
	UserID      int
	GUID        string
	FriendsID   int
	FriendsName string
	// Custom files stuff (CRC)
	CustomFiles0 int
	CustomFiles1 int
	CustomFiles2 int
	CustomFiles3 int
	// Amount of downloaded files from the server
	FilesDownloaded byte
	// Bots
	IsFakePlayer bool
	// HLTV Proxy
	IsHltv bool
}
