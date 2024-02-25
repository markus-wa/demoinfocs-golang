package common

import (
	"time"

	"github.com/golang/geo/r3"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// Player contains mostly game-relevant player information.
type Player struct {
	demoInfoProvider demoInfoProvider // provider for demo info such as tick-rate or current tick

	SteamID64             uint64             // 64-bit representation of the user's Steam ID. See https://developer.valvesoftware.com/wiki/SteamID
	LastAlivePosition     r3.Vector          // The location where the player was last alive. Should be equal to Position if the player is still alive.
	UserID                int                // Mostly used in game-events to address this player
	Name                  string             // Steam / in-game user name
	Inventory             map[int]*Equipment // All weapons / equipment the player is currently carrying. See also Weapons().
	AmmoLeft              [32]int            // Ammo left for special weapons (e.g. grenades), index corresponds Equipment.AmmoType
	EntityID              int                // Usually the same as Entity.ID() but may be different between player death and re-spawn.
	Entity                st.Entity          // May be nil between player-death and re-spawn
	FlashDuration         float32            // Blindness duration from the flashbang currently affecting the player (seconds)
	FlashTick             int                // In-game tick at which the player was last flashed
	TeamState             *TeamState         // When keeping the reference make sure you notice when the player changes teams
	Team                  Team               // Team identifier for the player (e.g. TeamTerrorists or TeamCounterTerrorists).
	IsBot                 bool               // True if this is a bot-entity. See also IsControllingBot and ControlledBot().
	IsConnected           bool
	IsDefusing            bool
	IsPlanting            bool
	IsReloading           bool
	IsUnknown             bool      // Used to identify unknown/broken players. see https://github.com/markus-wa/demoinfocs-golang/issues/162
	PreviousFramePosition r3.Vector // CS2 only, used to compute velocity as it's not networked in CS2 demos
	Distance              Distance  // Distance since last freezetime end
	Alive                 bool      // True if player is alive
}

type Distance struct {
	Running float64
	Walking float64
	Ducking float64
}

func (p *Player) PlayerPawnEntity() st.Entity {
	if p.Entity == nil {
		return nil
	}
	pawn, exists := p.Entity.PropertyValue("m_hPawn")
	if !exists {
		return nil
	}

	if pawn.Handle() == constants.InvalidEntityHandleSource2 {
		return nil
	}

	playerPawn, exists := p.Entity.PropertyValue("m_hPlayerPawn")
	if !exists {
		return nil
	}

	return p.demoInfoProvider.FindEntityByHandle(playerPawn.Handle())
}

func (p *Player) GetTeam() Team {
	return Team(p.PlayerPawnEntity().PropertyValueMust("m_iTeamNum").S2UInt64())
}

func (p *Player) GetFlashDuration() float32 {
	return p.PlayerPawnEntity().PropertyValueMust("m_flFlashDuration").Float()
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

// IsAlive returns true if the player is alive.
func (p *Player) IsAlive() bool {
	if p.Team < 2 {
		return false
	}

	if pawnEntity := p.PlayerPawnEntity(); pawnEntity != nil {
		if lifeStateVal, ok := pawnEntity.PropertyValue("m_lifeState"); ok {
			if lifeStateVal.S2UInt64() == 0 {
				return p.Health() > 0
			}
			return false
		}
		return p.Health() > 0
	}
	return false
}

// IsBlinded returns true if the player is currently flashed.
// This is more accurate than 'FlashDuration != 0' as it also takes into account FlashTick, DemoHeader.TickRate() and GameState.IngameTick().
func (p *Player) IsBlinded() bool {
	return p.FlashDurationTimeRemaining() > 0
}

// IsAirborne returns true if the player is jumping or falling.
func (p *Player) IsAirborne() bool {
	groundEntityHandle := getUInt64(p.PlayerPawnEntity(), "m_hGroundEntity")
	return groundEntityHandle == constants.InvalidEntityHandleSource2
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
	if pawnEntity := p.PlayerPawnEntity(); pawnEntity != nil {
		if pawnEntity.Property("m_pWeaponServices.m_hActiveWeapon") == nil {
			return 0
		}

		return int(pawnEntity.PropertyValueMust("m_pWeaponServices.m_hActiveWeapon").S2UInt64() & constants.EntityHandleIndexMaskSource2)
	}

	return 0
}

// ActiveWeapon returns the currently active / equipped weapon of the player.
// ! Can be nil
func (p *Player) ActiveWeapon() *Equipment {
	if p.activeWeaponID() == 0 {
		return nil
	}
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
		mask = p.PlayerPawnEntity().Property("m_bSpottedByMask.0000")
	} else {
		bit -= 32
		mask = p.PlayerPawnEntity().Property("m_bSpottedByMask.0001")
	}

	return (mask.Value().S2UInt64() & (1 << bit)) != 0
}

// HasSpotted returns true if the player has spotted the other player.
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (p *Player) HasSpotted(other *Player) bool {
	return other.IsSpottedBy(p)
}

// IsInBombZone returns whether the player is currently in the bomb zone or not.
func (p *Player) IsInBombZone() bool {
	return getBool(p.PlayerPawnEntity(), "m_bInBombZone")
}

// IsInBuyZone returns whether the player is currently in the buy zone or not.
func (p *Player) IsInBuyZone() bool {
	return getBool(p.PlayerPawnEntity(), "m_bInBuyZone")
}

// IsWalking returns whether the player is currently walking (sneaking) in or not.
func (p *Player) IsWalking() bool {
	return getBool(p.PlayerPawnEntity(), "m_bIsWalking")
}

// IsScoped returns whether the player is currently scoped in or not.
func (p *Player) IsScoped() bool {
	return getBool(p.PlayerPawnEntity(), "m_bIsScoped")
}

// IsDucking returns true if the player is currently fully crouching.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsDucking() bool {
	return p.Flags().Ducking()
}

// IsDuckingInProgress returns true if the player is currently in the progress of going from standing to crouched.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsDuckingInProgress() bool {
	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity == nil {
		return false
	}
	duckAmountVal, ok := pawnEntity.PropertyValue("m_pMovementServices.m_flDuckAmount")
	if !ok {
		return false
	}
	duckAmount := duckAmountVal.Float()
	wantToDuckVal, ok := pawnEntity.PropertyValue("m_pMovementServices.m_bDesiresDuck")
	if !ok {
		return false
	}
	wantToDuck := wantToDuckVal.BoolVal()

	return !p.Flags().Ducking() && wantToDuck && duckAmount > 0
}

// IsUnDuckingInProgress returns true if the player is currently in the progress of going from crouched to standing.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsUnDuckingInProgress() bool {
	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity == nil {
		return false
	}
	duckAmountVal, ok := pawnEntity.PropertyValue("m_pMovementServices.m_flDuckAmount")
	if !ok {
		return false
	}
	duckAmount := duckAmountVal.Float()
	wantToDuckVal, ok := pawnEntity.PropertyValue("m_pMovementServices.m_bDesiresDuck")
	if !ok {
		return false
	}
	wantToDuck := wantToDuckVal.BoolVal()

	return !p.Flags().Ducking() && !wantToDuck && duckAmount > 0
}

// IsStanding returns true if the player is currently fully standing upright.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsStanding() bool {
	return !p.Flags().Ducking() && !p.Flags().DuckingKeyPressed()
}

// HasDefuseKit returns true if the player currently has a defuse kit in his inventory.
func (p *Player) HasDefuseKit() bool {
	return getBool(p.PlayerPawnEntity(), "m_pItemServices.m_bHasDefuser")
}

// HasHelmet returns true if the player is currently wearing head armor.
func (p *Player) HasHelmet() bool {
	return getBool(p.PlayerPawnEntity(), "m_pItemServices.m_bHasHelmet")
}

// IsControllingBot returns true if the player is currently controlling a bot.
// See also ControlledBot().
func (p *Player) IsControllingBot() bool {
	return getBool(p.Entity, "m_bControllingBot")
}

// ControlledPawn returns the player instance of the pawn that the player is controlling, if any.
func (p *Player) ControlledPawn() *Player {
	if p.Entity == nil {
		return p
	}

	playerPawn, exists := p.Entity.PropertyValue("m_hOriginalControllerOfCurrentPawn")
	if !exists || !p.IsControllingBot() {
		return p
	}

	return p.demoInfoProvider.FindPlayerByHandle(playerPawn.S2UInt64())
}

// Controller returns the player instance of the controller that the is controlling player, if any.
func (p *Player) Controller() *Player {
	if p.Entity == nil {
		return p
	}

	playerPawn, exists := p.Entity.PropertyValue("m_hOriginalControllerOfCurrentPawn")
	if !exists || !p.IsBot {
		return p
	}

	return p.demoInfoProvider.FindPlayerByHandle(playerPawn.S2UInt64())
}

// Health returns the player's health points, normally 0-100.
func (p *Player) Health() int {
	return getInt(p.PlayerPawnEntity(), "m_iHealth")
}

func (p *Player) LifeState() int {
	return int(getUInt64(p.PlayerPawnEntity(), "m_lifeState"))
}

// Armor returns the player's armor points, normally 0-100.
func (p *Player) Armor() int {
	return getInt(p.PlayerPawnEntity(), "m_ArmorValue")
}

// RankType returns the current rank type that the player is playing for.
// -1 -> Not available, demo probably not coming from a Valve server
// 0 -> None?
// 7 -> Wingman 2v2
// 11 -> Premier mode
// 12 -> Classic Competitive
func (p *Player) RankType() int {
	return getInt(p.Entity, "m_iCompetitiveRankType")
}

// Ranking returns the current rank of the player for the current RankType.
// CS:GO demos -> from 0 to 18 (0 = unranked/unknown, 18 = Global Elite)
// CS2 demos -> Number representation of the player's rank.
func (p *Player) Ranking() int {
	return getInt(p.Entity, "m_iCompetitiveRanking")
}

// CompetitiveWins returns the amount of competitive wins the player has for the current RankType.
func (p *Player) CompetitiveWins() int {
	return getInt(p.Entity, "m_iCompetitiveWins")
}

func (p *Player) RankingPredictedWin() int {
	return getInt(p.Entity, "m_iCompetitiveRankingPredicted_Win")
}

func (p *Player) RankingPredictedLoss() int {
	return getInt(p.Entity, "m_iCompetitiveRankingPredicted_Loss")
}

func (p *Player) RankingPredictedTie() int {
	return getInt(p.Entity, "m_iCompetitiveRankingPredicted_Tie")
}

// Money returns the amount of money in the player's bank.
func (p *Player) Money() int {
	return getInt(p.Entity, "m_pInGameMoneyServices.m_iAccount")
}

// EquipmentValueCurrent returns the current value of equipment in the player's inventory.
func (p *Player) EquipmentValueCurrent() int {
	return int(getUInt64(p.PlayerPawnEntity(), "m_unCurrentEquipmentValue"))
}

// EquipmentValueRoundStart returns the value of equipment in the player's inventory at the time of the round start.
// This is before the player has bought any new items in the freeze time.
// See also Player.EquipmentValueFreezetimeEnd().
func (p *Player) EquipmentValueRoundStart() int {
	return int(getUInt64(p.PlayerPawnEntity(), "m_unRoundStartEquipmentValue"))
}

// EquipmentValueFreezeTimeEnd returns the value of equipment in the player's inventory at the end of the freeze time.
func (p *Player) EquipmentValueFreezeTimeEnd() int {
	return int(getUInt64(p.PlayerPawnEntity(), "m_unFreezetimeEndEquipmentValue"))
}

// ViewDirectionX returns the Yaw value in degrees, 0 to 360.
func (p *Player) ViewDirectionX() float32 {
	if pawnEntity := p.PlayerPawnEntity(); pawnEntity != nil {
		return float32(pawnEntity.PropertyValueMust("m_angEyeAngles").R3Vec().Y)
	}

	return 0
}

// ViewDirectionY returns the Pitch value in degrees, 270 to 90 (270=-90).
func (p *Player) ViewDirectionY() float32 {
	if pawnEntity := p.PlayerPawnEntity(); pawnEntity != nil {
		return float32(pawnEntity.PropertyValueMust("m_angEyeAngles").R3Vec().X)
	}

	return 0
}

// Position returns the in-game coordinates.
// Note: the Z value is not on the player's eye height but instead at his feet.
// See also PositionEyes().
func (p *Player) Position() r3.Vector {
	if pawnEntity := p.PlayerPawnEntity(); pawnEntity != nil {
		return pawnEntity.Position()
	}

	return r3.Vector{}
}

// Velocity returns the player's velocity.
func (p *Player) Velocity() r3.Vector {
	if !p.Alive {
		return r3.Vector{}
	}

	t := 64.0
	diff := p.Position().Sub(p.PreviousFramePosition)

	return r3.Vector{
		X: diff.X * t,
		Y: diff.Y * t,
		Z: diff.Z * t,
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
	return PlayerFlags(getUInt64(p.PlayerPawnEntity(), "m_fFlags"))
}

// //////////////////////////
// CCSPlayerResource stuff //
// //////////////////////////

// ClanTag returns the player's individual clan tag (Steam Groups etc.).
func (p *Player) ClanTag() string {
	return getString(p.Entity, "m_szClan")
}

// CrosshairCode returns the player's crosshair code or an empty string if there isn't one.
func (p *Player) CrosshairCode() string {
	return getString(p.Entity, "m_szCrosshairCodes")

}

// Ping returns the players latency to the game server.
func (p *Player) Ping() int {
	// TODO change this func return type to uint64? (small BC)
	return int(getUInt64(p.Entity, "m_iPing"))
}

// Score returns the players score as shown on the scoreboard.
func (p *Player) Score() int {
	return getInt(p.Entity, "m_iScore")

}

// Color returns the players color as shown on the minimap.
func (p *Player) Color() Color {
	return Color(getInt(p.Entity, "m_iCompTeammateColor"))
}

// Kills returns the amount of kills the player has as shown on the scoreboard.
func (p *Player) Kills() int {
	return getInt(p.Entity, "m_pActionTrackingServices.m_iKills")
}

// Deaths returns the amount of deaths the player has as shown on the scoreboard.
func (p *Player) Deaths() int {
	return getInt(p.Entity, "m_pActionTrackingServices.m_iDeaths")
}

// Assists returns the amount of assists the player has as shown on the scoreboard.
func (p *Player) Assists() int {
	return getInt(p.Entity, "m_pActionTrackingServices.m_iAssists")
}

// MVPs returns the amount of Most-Valuable-Player awards the player has as shown on the scoreboard.
func (p *Player) MVPs() int {
	return getInt(p.Entity, "m_iMVPs")
}

// TotalDamage returns the total health damage done by the player.
func (p *Player) TotalDamage() int {
	value := p.Entity.PropertyValueMust("m_pActionTrackingServices.m_iDamage")
	if value.Any == nil {
		return 0
	}
	return value.Int()
}

// UtilityDamage returns the total damage done by the player with grenades.
func (p *Player) UtilityDamage() int {
	value := p.Entity.PropertyValueMust("m_pActionTrackingServices.m_iUtilityDamage")
	if value.Any == nil {
		return 0
	}
	return value.Int()
}

// MoneySpentTotal returns the total amount of money the player has spent in the current match.
func (p *Player) MoneySpentTotal() int {
	return getInt(p.Entity, "m_pInGameMoneyServices.m_iTotalCashSpent")
}

// MoneySpentThisRound returns the amount of money the player has spent in the current round.
func (p *Player) MoneySpentThisRound() int {
	return getInt(p.Entity, "m_pInGameMoneyServices.m_iCashSpentThisRound")
}

// LastPlaceName returns the string value of the player's position.
func (p *Player) LastPlaceName() string {
	return getString(p.PlayerPawnEntity(), "m_szLastPlaceName")
}

// IsGrabbingHostage returns true if the player is currently grabbing a hostage.
func (p *Player) IsGrabbingHostage() bool {
	return getBool(p.PlayerPawnEntity(), "m_bIsGrabbingHostage")
}

func (p *Player) PublicLevel() int {
	return getInt(p.Entity, "m_pInventoryServices.m_nPersonaDataPublicLevel")
}

func (p *Player) PublicCommendsLeader() int {
	return getInt(p.Entity, "m_pInventoryServices.m_nPersonaDataPublicCommendsLeader")
}

func (p *Player) PublicCommendsTeacher() int {
	return getInt(p.Entity, "m_pInventoryServices.m_nPersonaDataPublicCommendsTeacher")
}

func (p *Player) PublicCommendsFriendly() int {
	return getInt(p.Entity, "m_pInventoryServices.m_nPersonaDataPublicCommendsFriendly")
}

func (p *Player) XpTrailLevel() int {
	return getInt(p.Entity, "m_pInventoryServices.m_nPersonaDataXpTrailLevel")
}

type demoInfoProvider interface {
	IngameTick() int   // current in-game tick, used for IsBlinded()
	TickRate() float64 // in-game tick rate, used for Player.IsBlinded()
	FindPlayerByHandle(handle uint64) *Player
	FindPlayerByPawnHandle(handle uint64) *Player
	PlayerResourceEntity() st.Entity
	FindWeaponByEntityID(id int) *Equipment
	FindEntityByHandle(handle uint64) st.Entity
}

// NewPlayer creates a *Player with an initialized equipment map.
//
// Intended for internal use only.
func NewPlayer(demoInfoProvider demoInfoProvider) *Player {
	return &Player{
		Inventory:             make(map[int]*Equipment),
		demoInfoProvider:      demoInfoProvider,
		PreviousFramePosition: r3.Vector{},
	}
}

// PlayerInfo contains information about a player such as their name and SteamID.
// Primarily intended for internal use.
type PlayerInfo struct {
	Version     int64  // Not available with CS2 demos
	XUID        uint64 // SteamID64
	Name        string
	UserID      int    // Not available with CS2 demos
	GUID        string // Not available with CS2 demos
	FriendsID   int    // Not available with CS2 demos
	FriendsName string // Not available with CS2 demos
	// Custom files stuff (CRC). Not available with CS2 demos
	CustomFiles0 int
	CustomFiles1 int
	CustomFiles2 int
	CustomFiles3 int
	// Amount of downloaded files from the server
	FilesDownloaded byte // Not available with CS2 demos
	// Bots
	IsFakePlayer bool
	// HLTV Proxy
	IsHltv bool
}
