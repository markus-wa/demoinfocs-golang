package common

import (
	"errors"
	"math"
	"time"

	"github.com/golang/geo/r3"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// Player contains mostly game-relevant player information.
type Player struct {
	demoInfoProvider demoInfoProvider // provider for demo info such as tick-rate or current tick

	SteamID64           uint64             // 64-bit representation of the user's Steam ID. See https://developer.valvesoftware.com/wiki/SteamID
	UserID              int                // Mostly used in game-events to address this player
	Name                string             // In-game name as enforced by the server & shown on the scoreboard (the player controller's m_iszPlayerName). May differ from the Steam name in PlayerInfo / PlayerConnect, e.g. on FACEIT.
	Inventory           map[int]*Equipment // All weapons / equipment the player is currently carrying. See also Weapons().
	EntityID            int                // Usually the same as Entity.ID() but may be different between player death and re-spawn.
	Entity              *st.Entity         // May be nil between player-death and re-spawn
	FlashDuration       float32            // Effective blindness duration from the flashbang currently affecting the player (seconds). The engine's flash-affected window (e.g. for assist attribution) runs ~1.4x longer; there is no continuous remaining-intensity netprop.
	FlashTick           int                // In-game tick at which the player was last flashed
	TeamState           *TeamState         // When keeping the reference make sure you notice when the player changes teams
	Team                Team               // Team identifier for the player (e.g. TeamTerrorists or TeamCounterTerrorists).
	IsBot               bool               // True if this is a bot-entity. See also IsControllingBot and ControlledBot().
	IsConnected         bool
	IsDefusing          bool
	IsPlanting          bool
	IsReloading         bool
	IsUnknown           bool   // Used to identify unknown/broken players. see https://github.com/markus-wa/demoinfocs-golang/issues/162
	ButtonsPressedState uint64 // Pressed buttons state represented as an uint64. You can use IsPressingButton(buttonMask) to check for specific buttons.

	// positionHistory holds the pawn position after the last two entity snapshots ([0] = most recent).
	// Used by Velocity() to derive the velocity when m_vecVelocity isn't networked.
	positionHistory [2]positionSample
	// trackingPosition is set once the parser records this player's position at the end of every frame,
	// see startPositionTracking().
	trackingPosition bool
	// recordedTick is the SnapshotTick() recordFramePosition() last ran in full for, reset to -1 when tracking
	// starts so the next snapshot is recorded.
	recordedTick int
}

// positionSample is a pawn position recorded at the end of a frame.
type positionSample struct {
	pos         r3.Vector
	tick        int        // server tick of the entity snapshot, see demoInfoProvider.SnapshotTick()
	pawn        *st.Entity // compared by identity: a pawn re-created in the same slot is a new entity
	interpFrame any        // m_ubInterpolationFrame, changes when the pawn is teleported
	valid       bool
}

// isPawnAlive returns whether the pawn's m_lifeState is LIFE_ALIVE.
func isPawnAlive(pawnEntity *st.Entity) bool {
	v, ok := pawnEntity.PropertyValue("m_lifeState")
	if !ok {
		return true
	}

	switch lifeState := v.Any.(type) {
	case uint64:
		return lifeState == 0
	case uint32:
		return lifeState == 0
	case int32:
		return lifeState == 0
	default:
		return true
	}
}

// interpolationFrame returns the pawn's m_ubInterpolationFrame value (nil if unavailable).
// The engine changes it whenever an entity is teleported, so that clients don't interpolate.
func interpolationFrame(pawnEntity *st.Entity) any {
	v, ok := pawnEntity.PropertyValue("m_ubInterpolationFrame")
	if !ok {
		return nil
	}

	return v.Any
}

func (p *Player) PlayerPawnEntity() *st.Entity {
	if p == nil || p.Entity == nil {
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
	return Team(p.PlayerPawnEntity().PropertyValueMust("m_iTeamNum").UInt64())
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
	if p.Health() > 0 {
		return true
	}

	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity != nil {
		return pawnEntity.PropertyValueMust("m_lifeState").UInt64() == 0
	}

	return getBool(p.Entity, "m_bPawnIsAlive")
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

// ActiveWeaponID is used internally to set the active weapon, see ActiveWeapon()
func (p *Player) ActiveWeaponID() int {
	if pawnEntity := p.PlayerPawnEntity(); pawnEntity != nil {
		return int(pawnEntity.PropertyValueMust("m_pWeaponServices.m_hActiveWeapon").UInt64() & constants.EntityHandleIndexMaskSource2)
	}

	return 0
}

// ActiveWeapon returns the currently active / equipped weapon of the player.
// ! Can be nil
func (p *Player) ActiveWeapon() *Equipment {
	return p.demoInfoProvider.FindWeaponByEntityID(p.ActiveWeaponID())
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

// FlashbangCount returns the amount of flashbangs the player currently has in his inventory.
func (p *Player) FlashbangCount() uint64 {
	pawn := p.PlayerPawnEntity()
	if pawn == nil {
		return 0
	}

	flashCountProp, ok := pawn.PropertyValue("m_pWeaponServices.m_iAmmo.0014")
	if !ok {
		return 0
	}

	return flashCountProp.UInt64()
}

// IsSpottedBy returns true if the player has been spotted by the other player.
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (p *Player) IsSpottedBy(other *Player) bool {
	if p.Entity == nil {
		return false
	}

	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity == nil {
		return false
	}

	clientSlot := other.EntityID - 1
	bit := uint(clientSlot)

	var mask *st.Property

	if bit < 32 {
		mask = pawnEntity.Property("m_bSpottedByMask.0000")
	} else {
		bit -= 32
		mask = pawnEntity.Property("m_bSpottedByMask.0001")
	}

	if mask == nil {
		return false
	}

	return (mask.Value().UInt64() & (1 << bit)) != 0
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

// IsWalking returns whether the player is holding the walk (sneak) key,
// i.e. is in sneak mode.
//
// Note: this does not indicate that the player is actually moving - it is true
// even while standing still, as long as the walk key is held.
// For actual movement detection use Velocity() instead.
//
// See https://github.com/markus-wa/demoinfocs-golang/issues/318
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

	duckAmount := pawnEntity.PropertyValueMust("m_pMovementServices.m_flDuckAmount").Float()
	wantToDuck := pawnEntity.PropertyValueMust("m_pMovementServices.m_bDesiresDuck").BoolVal()

	return !p.Flags().Ducking() && wantToDuck && duckAmount > 0
}

// IsUnDuckingInProgress returns true if the player is currently in the progress of going from crouched to standing.
// See also: Flags().Ducking() & Flags().DuckingKeyPressed()
func (p *Player) IsUnDuckingInProgress() bool {
	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity == nil {
		return false
	}

	duckAmount := pawnEntity.PropertyValueMust("m_pMovementServices.m_flDuckAmount").Float()
	wantToDuck := pawnEntity.PropertyValueMust("m_pMovementServices.m_bDesiresDuck").BoolVal()

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

// ControlledBot returns the player instance of the bot that the player is controlling, if any.
// Returns nil if the player is not controlling a bot.
func (p *Player) ControlledBot() *Player {
	if p.Entity == nil {
		return nil
	}

	controllerHandler := p.Entity.Property("m_hOriginalControllerOfCurrentPawn").Value().UInt64()

	return p.demoInfoProvider.FindPlayerByHandle(controllerHandler)
}

// Health returns the player's health points, normally 0-100.
func (p *Player) Health() int {
	return getInt(p.PlayerPawnEntity(), "m_iHealth")
}

// Armor returns the player's armor points, normally 0-100.
func (p *Player) Armor() int {
	return getInt(p.PlayerPawnEntity(), "m_ArmorValue")
}

// RankType returns the current rank type that the player is playing for.
// CS:GO values:
// -1 -> Information not present, the demo is too old
// 0 -> None/not available
// 6 -> Classic Competitive
// 7 -> Wingman 2v2
// 10 -> Danger zone
//
// CS2 values:
// -1 -> Not available, demo probably not coming from a Valve server
// 0 -> None?
// 7 -> Wingman 2v2
// 11 -> Premier mode
// 12 -> Classic Competitive
func (p *Player) RankType() int {
	return getInt(p.Entity, "m_iCompetitiveRankType")
}

// Rank returns the current rank of the player for the current RankType.
// CS:GO demos -> from 0 to 18 (0 = unranked/unknown, 18 = Global Elite)
// CS2 demos -> Number representation of the player's rank.
func (p *Player) Rank() int {
	return getInt(p.Entity, "m_iCompetitiveRanking")
}

// CompetitiveWins returns the amount of competitive wins the player has for the current RankType.
func (p *Player) CompetitiveWins() int {
	return getInt(p.Entity, "m_iCompetitiveWins")
}

// Money returns the amount of money in the player's bank.
func (p *Player) Money() int {
	return getInt(p.Entity, "m_pInGameMoneyServices.m_iAccount")
}

// EquipmentValueCurrent returns the current value of equipment in the player's inventory.
func (p *Player) EquipmentValueCurrent() int {
	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity == nil {
		return 0
	}

	equipmentValue, exists := pawnEntity.PropertyValue("m_unCurrentEquipmentValue")
	if !exists {
		return 0
	}

	return int(equipmentValue.UInt64()) //#nosec G115
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

// Velocity returns the player's velocity in game units per second.
//
// The velocity is read from the pawn's m_vecVelocity property when it is networked, which is the case
// for GOTV demos recorded since the CS2 "AnimGraph 2" update (28/07/2025, network protocol 14089),
// and as far as known for POV demos too. On older demos the property isn't populated on GOTV demos, and
// holds stale values for everybody but the recording player on POV demos - only the recording player's
// networked value is used there. For everybody else on those demos, the velocity is derived from the
// position delta between two entity snapshots, divided by the server time elapsed between them (server
// ticks, see TickRate()).
//
// Positions are only recorded for players Velocity() has been called for, so the fallback costs nothing
// unless it's used. Calls for a player therefore return a zero vector until the entity snapshot after the
// first call has been applied, and the parser dispatches a ParserWarn (WarnTypeVelocityTrackingStartedLazily)
// the first time this happens. Use option.EagerVelocityTracking(true) to record all players from the start
// instead. On those demos Velocity() thus registers the player with the parser - like the parser itself,
// it must not be called concurrently with parsing (e.g. from another goroutine).
//
// The fallback:
//   - returns a zero vector until the next entity snapshot has been applied after a player (re-)spawned,
//     while dead, and when a pawn was teleported (m_ubInterpolationFrame changed, or a delta above
//     sv_maxvelocity);
//   - keeps the last value during frames without an entity snapshot (e.g. the user command frames of POV
//     demos), and returns the value of the previous snapshot while one is applied (e.g. in WeaponFire
//     handlers) - for every player, since a pawn's position may or may not be updated yet at that point.
//     Values there lag by one snapshot (e.g. a median of ~5 u/s on POV and ~20 u/s on older GOTV demos
//     at WeaponFire) instead of reading as a standstill;
//   - lags the networked value by up to one snapshot and is noisier, since it is a finite difference over
//     the snapshot interval (e.g. 3 ticks on older GOTV demos) - e.g. Z briefly spikes when a pawn steps up
//     stairs or lands.
func (p *Player) Velocity() r3.Vector {
	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity == nil {
		return r3.Vector{}
	}

	if !p.isVelocityNetworked(pawnEntity) {
		// not lazy if the config flag is set but tracking stopped (the controller is gone, e.g. Velocity()
		// called on a Player kept from before a disconnect - a reconnect creates a new Player)
		p.startPositionTracking(!p.demoInfoProvider.EagerVelocityTracking())

		return p.positionDeltaVelocity(pawnEntity)
	}

	x, _ := getFloatIfExists(pawnEntity, "m_vecVelocity.m_vecX")
	y, _ := getFloatIfExists(pawnEntity, "m_vecVelocity.m_vecY")
	z, _ := getFloatIfExists(pawnEntity, "m_vecVelocity.m_vecZ")

	return r3.Vector{
		X: float64(x),
		Y: float64(y),
		Z: float64(z),
	}
}

// isVelocityNetworked returns whether the pawn's m_vecVelocity property holds the actual velocity.
//
// The decision is made per demo (network protocol) and per pawn (property presence) - never on the
// value itself - so the source doesn't flip between frames for a player legitimately standing still.
// Before AnimGraph 2 the property is either never populated (GOTV) or holds stale values for everybody but
// the recording player (POV), so it's ignored for everybody but the recording player.
// An unknown network protocol (0, i.e. no file header with a patch version parsed yet) is decided by
// property presence alone.
func (p *Player) isVelocityNetworked(pawnEntity *st.Entity) bool {
	protocol := p.demoInfoProvider.NetworkProtocol()
	if protocol > 0 && protocol < constants.NetworkProtocolAnimGraph2 && !p.isRecordingPlayer() {
		return false
	}

	_, ok := getFloatIfExists(pawnEntity, "m_vecVelocity.m_vecX")

	return ok
}

// isRecordingPlayer returns whether the player recorded the (POV) demo.
func (p *Player) isRecordingPlayer() bool {
	slot := p.demoInfoProvider.RecordingPlayerSlot()

	return slot >= 0 && p.EntityID-1 == slot
}

// maxVelocity is the default value of the sv_maxvelocity convar, the maximum velocity (per axis, in
// game units per second) the engine allows for any entity.
const maxVelocity = 3500

// positionDeltaVelocity derives the velocity from the position history recorded by recordFramePosition().
//
// Time is measured in server ticks of entity snapshots (see demoInfoProvider.SnapshotTick()), not in-game
// ticks: on POV demos the latter advance in frames without a snapshot and jitter against the snapshots,
// which made the velocity alternate between the real value and zero and produced spikes.
func (p *Player) positionDeltaVelocity(pawnEntity *st.Entity) r3.Vector {
	last, prev := p.positionHistory[0], p.positionHistory[1]
	tick := p.demoInfoProvider.SnapshotTick()

	if !last.valid || last.tick > tick {
		// no history, or the tick went backwards
		return r3.Vector{}
	}

	// While a snapshot is being processed (e.g. in a WeaponFire handler), the pawn's position may or may not
	// be updated yet - use the velocity between the last two samples instead, which lags by one snapshot.
	// Without a new snapshot since the last sample (e.g. a user command frame of a POV demo), the position
	// hasn't changed since, so that's the current velocity as well.
	if p.demoInfoProvider.ProcessingSnapshot() || last.tick == tick {
		return p.velocitySince(prev, last.pos, last.tick, pawnEntity)
	}

	return p.velocitySince(last, pawnEntity.Position(), tick, pawnEntity)
}

// velocitySince returns the average velocity from the reference sample to the given position and tick
// of the same pawn, or a zero vector if they can't be compared.
func (p *Player) velocitySince(ref positionSample, pos r3.Vector, tick int, pawnEntity *st.Entity) r3.Vector {
	tickRate := p.demoInfoProvider.TickRate()

	if tickRate <= 0 || !ref.valid || ref.tick >= tick || ref.pawn != pawnEntity {
		return r3.Vector{}
	}

	if ref.interpFrame != interpolationFrame(pawnEntity) {
		// teleported (e.g. round restart) since the reference sample - not a real movement
		return r3.Vector{}
	}

	dt := float64(tick-ref.tick) / tickRate
	v := pos.Sub(ref.pos).Mul(1 / dt)

	if math.Abs(v.X) > maxVelocity || math.Abs(v.Y) > maxVelocity || math.Abs(v.Z) > maxVelocity {
		// physically impossible, must be a teleport that wasn't flagged as such
		return r3.Vector{}
	}

	return v
}

// startPositionTracking makes the parser record this player's position at the end of every frame,
// used by Velocity() when the velocity isn't networked. Registered at most once per player.
func (p *Player) startPositionTracking(lazy bool) {
	if p.trackingPosition {
		return
	}

	p.trackingPosition = true
	p.recordedTick = -1
	p.demoInfoProvider.TrackFramePosition(p.recordFramePosition, lazy)
}

// recordFramePosition records the pawn position at the end of a frame.
// Returns false once the player's controller entity is gone (disconnect) and recording should stop -
// a reconnect creates a new Player.
func (p *Player) recordFramePosition() (keep bool) {
	// Entities (including the controller and pawn) only change with an entity snapshot,
	// skip frames without one (e.g. the user command frames of POV demos).
	tick := p.demoInfoProvider.SnapshotTick()
	if tick == p.recordedTick {
		return true
	}

	p.recordedTick = tick

	if p.Entity == nil || p.demoInfoProvider.FindEntityByID(p.Entity.ID()) != p.Entity {
		p.positionHistory = [2]positionSample{}
		p.trackingPosition = false

		return false
	}

	pawnEntity := p.PlayerPawnEntity()

	// The position lookup is relatively expensive, skip it when the networked velocity is available anyway.
	if pawnEntity == nil || !isPawnAlive(pawnEntity) || p.isVelocityNetworked(pawnEntity) {
		p.positionHistory = [2]positionSample{}

		return true
	}

	sample := positionSample{
		pos:         pawnEntity.Position(),
		tick:        tick,
		pawn:        pawnEntity,
		interpFrame: interpolationFrame(pawnEntity),
		valid:       true,
	}

	last := p.positionHistory[0]

	switch {
	case last.valid && last.pawn == sample.pawn && sample.tick > last.tick:
		p.positionHistory = [2]positionSample{sample, last}
	default:
		// first sample, new pawn or tick went backwards
		p.positionHistory = [2]positionSample{sample}
	}

	return true
}

// Position returns the in-game coordinates.
// Note: the Z value is not on the player's eye height but instead at his feet.
// See also PositionEyes().
func (p *Player) Position() r3.Vector {
	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity != nil {
		return pawnEntity.Position()
	}

	return r3.Vector{}
}

// PositionEyes returns the best available in-game coordinates for the player's eye position
// and whether the CS2 eye-offset fields were available.
//
// The bool result is true only when the returned vector includes the pawn eye offset. If the pawn
// exists but the eye-offset fields are unavailable, the returned vector is the pawn base position
// and the bool result is false. If there is no pawn, it returns the zero vector and false.
func (p *Player) PositionEyes() (r3.Vector, bool) {
	pawnEntity := p.PlayerPawnEntity()
	if pawnEntity == nil {
		return r3.Vector{}, false
	}

	pos := pawnEntity.Position()

	offset, ok := p.eyePositionOffset(pawnEntity)
	if !ok {
		return pos, false
	}

	return pos.Add(offset), true
}

func getViewOffsetComponent(pawnEntity *st.Entity, component string) (float32, bool) {
	if v, ok := getFloatIfExists(pawnEntity, "m_vecViewOffset."+component); ok {
		return v, true
	}

	return getFloatIfExists(pawnEntity, component)
}

func (p *Player) eyePositionOffset(pawnEntity *st.Entity) (r3.Vector, bool) {
	x, ok := getViewOffsetComponent(pawnEntity, "m_vecX")
	if !ok {
		return r3.Vector{}, false
	}

	y, ok := getViewOffsetComponent(pawnEntity, "m_vecY")
	if !ok {
		return r3.Vector{}, false
	}

	z, ok := getViewOffsetComponent(pawnEntity, "m_vecZ")
	if !ok {
		return r3.Vector{}, false
	}

	return r3.Vector{
		X: float64(x),
		Y: float64(y),
		Z: float64(z),
	}, true
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

// ///////////////////
// Scoreboard stuff //
// ///////////////////

// ClanTag returns the player's individual clan tag (Steam Groups etc.).
func (p *Player) ClanTag() string {
	return getString(p.Entity, "m_szClan")
}

// CrosshairCode returns the player's crosshair code or an empty string if there isn't one.
func (p *Player) CrosshairCode() string {
	return getString(p.Entity, "m_szCrosshairCodes")
}

// ViewmodelOffset returns the player's viewmodel offset as a 3D vector (X, Y, Z).
// Returns zero vector if not available (CS:GO demos or player not alive).
func (p *Player) ViewmodelOffset() r3.Vector {
	pawn := p.PlayerPawnEntity()
	if pawn == nil {
		return r3.Vector{}
	}

	return r3.Vector{
		X: float64(getFloat(pawn, "m_flViewmodelOffsetX")),
		Y: float64(getFloat(pawn, "m_flViewmodelOffsetY")),
		Z: float64(getFloat(pawn, "m_flViewmodelOffsetZ")),
	}
}

// ViewmodelFOV returns the player's viewmodel field of view.
// Returns 0 if not available (CS:GO demos or player not alive).
func (p *Player) ViewmodelFOV() float32 {
	pawn := p.PlayerPawnEntity()
	if pawn == nil {
		return 0
	}

	return getFloat(pawn, "m_flViewmodelFOV")
}

// Ping returns the players latency to the game server.
func (p *Player) Ping() int {
	return int(getUInt64(p.Entity, "m_iPing"))
}

// Score returns the players score as shown on the scoreboard.
func (p *Player) Score() int {
	return getInt(p.Entity, "m_iScore")
}

var (
	// ErrDataNotAvailable is returned if some data is not (yet) available.
	// Reading the same data later during parsing may work.
	ErrDataNotAvailable = errors.New("some data is not (yet) available")

	// ErrNotSupportedByDemo is returned if some data is not supported by the demo, e.g. because the demo is too old.
	ErrNotSupportedByDemo = errors.New("this data is not supported by the demo (this may be because the demo is too old)")
)

// ColorOrErr returns the players color as shown on the minimap.
// Returns ErrDataNotAvailable if the resource entity does not exist (it may exist later during parsing).
// Returns ErrNotSupportedByDemo if the demo does not support player colors (e.g. very old demos).
func (p *Player) ColorOrErr() (Color, error) {
	if p.Entity == nil {
		return 0, ErrDataNotAvailable
	}

	return Color(p.Entity.PropertyValueMust("m_iCompTeammateColor").Int()), nil
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

// IsPressingButton returns true if the player is currently pressing the given button.
// ! "Button" doesn't refer to the key on the keyboard/mouse but rather to an in-game action such as "move forward", "attack", "jump", etc.
func (p *Player) IsPressingButton(buttonBitMask ButtonBitMask) bool {
	return (p.ButtonsPressedState & uint64(buttonBitMask)) != 0
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

type demoInfoProvider interface {
	IngameTick() int   // current in-game tick, used for IsBlinded()
	TickRate() float64 // in-game tick rate, used for Player.IsBlinded()
	FindPlayerByHandle(handle uint64) *Player
	FindPlayerByPawnHandle(handle uint64) *Player
	FindWeaponByEntityID(id int) *Equipment
	FindEntityByHandle(handle uint64) *st.Entity
	FindEntityByID(id int) *st.Entity
	NetworkProtocol() int     // demo network protocol (patch version), 0 if unknown, used for Player.Velocity()
	RecordingPlayerSlot() int // player slot of the POV demo's recording player, -1 for GOTV demos
	// EagerVelocityTracking returns whether option.EagerVelocityTracking(true) is set.
	EagerVelocityTracking() bool
	// SnapshotTick returns the server tick of the last entity snapshot (svc_PacketEntities), -1 if none yet.
	// Unlike IngameTick() it only advances with entity data, used for Player.Velocity().
	SnapshotTick() int
	// ProcessingSnapshot returns whether an entity snapshot is currently being applied
	// (e.g. while dispatching events triggered by entity updates).
	ProcessingSnapshot() bool
	// TrackFramePosition registers a function the parser calls at the end of every frame (after FrameDone),
	// until it returns false.
	// lazy is true if tracking was started by a Velocity() call rather than by the config.
	TrackFramePosition(record func() (keep bool), lazy bool)
}

// NewPlayer creates a *Player with an initialized equipment map.
//
// Intended for internal use only.
func NewPlayer(demoInfoProvider demoInfoProvider) *Player {
	pl := &Player{
		Inventory:        make(map[int]*Equipment),
		demoInfoProvider: demoInfoProvider,
	}

	// option.EagerVelocityTracking(true): record positions from the start on demos that don't network
	// the velocity, instead of lazily on the first Velocity() call.
	// Not with an unknown protocol (0, i.e. no file header with a patch version parsed yet) - Velocity() still
	// falls back lazily then if the velocity isn't networked.
	if demoInfoProvider != nil && demoInfoProvider.EagerVelocityTracking() &&
		demoInfoProvider.NetworkProtocol() > 0 && demoInfoProvider.NetworkProtocol() < constants.NetworkProtocolAnimGraph2 {
		pl.startPositionTracking(false)
	}

	return pl
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
