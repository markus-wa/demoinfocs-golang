package demoinfocs

import (
	"cmp"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/golang/geo/r3"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// GameState contains all game-state relevant information, such as players, teams, scores, grenades and entities.
//
// It is obtained via Parser.GameState() and is updated in place while the demo is parsed,
// so it always reflects the state at the parser's current position. The zero value is not usable.
// It must not be modified by users and is not safe for concurrent use with parsing.
type GameState struct {
	ingameTick                   int
	tState                       common.TeamState
	ctState                      common.TeamState
	playersByUserID              map[int]*common.Player    // Maps user-IDs to players
	playersByEntityID            map[int]*common.Player    // Maps entity-IDs to players
	playersBySteamID32           map[uint32]*common.Player // Maps 32-bit-steam-IDs to players
	playerControllerEntities     map[int]*st.Entity
	grenadeProjectiles           map[int]*common.GrenadeProjectile // Maps entity-IDs to active nade-projectiles. That's grenades that have been thrown, but have not yet detonated.
	infernos                     map[int]*common.Inferno           // Maps entity-IDs to active infernos.
	infernoFireStates            map[int]*infernoFireState         // Maps inferno entity-IDs to their burning state, for InfernoFireOut.
	infernoProjectiles           map[int]*common.GrenadeProjectile // Maps inferno entity-IDs to the molotov/incendiary projectile that spawned them, see linkInfernoProjectile.
	pendingInfernoLinks          []pendingInfernoLink              // Fire-grenade projectiles destroyed / infernos created that are still waiting for their counterpart. Entries from earlier frames are pruned on the next link attempt.
	weapons                      map[int]*common.Equipment         // Maps entity IDs to weapons. Used to remember what a weapon is (p250 / cz etc.)
	hostages                     map[int]*common.Hostage           // Maps entity-IDs to hostages.
	entities                     map[int]*st.Entity                // Maps entity IDs to entities
	bomb                         common.Bomb
	totalRoundsPlayed            int
	gamePhase                    common.GamePhase
	isWarmupPeriod               bool
	isFreezetime                 bool
	isMatchStarted               bool
	overtimeCount                int
	lastFlash                    lastFlash                                                       // Information about the last flash that exploded, used to find the attacker and projectile for player_blind events
	currentDefuser               *common.Player                                                  // Player currently defusing the bomb, if any
	currentPlanter               *common.Player                                                  // Player currently planting the bomb, if any
	thrownGrenades               map[*common.Player]map[common.EquipmentType][]*common.Equipment // Information about every player's thrown grenades (from the moment they are thrown to the moment their effect is ended)
	pendingGrenadeThrows         map[pendingThrowKey]pendingThrow                                // Grenade weapons about to be thrown per pawn & type, see primeGrenadeThrow
	lastKnownGrenadeWeapons      map[*common.Player]map[common.EquipmentType]*common.Equipment   // Last-known grenade weapon per player & type, see updateLastKnownGrenadeWeapons
	rules                        GameRules
	demoInfo                     demoInfoProvider
	lastRoundStartEvent          *events.RoundStart             // Used to dispatch this event after a possible MatchStartedChanged event
	lastFreezeTimeChangedEvent   *events.RoundFreezetimeChanged // Used to dispatch this event after a possible RoundStart event
	lastRoundEndEvent            *events.RoundEnd               // Used to dispatch this event before a possible RoundFreezetimeChanged event
	lastMatchStartedChangedEvent *events.MatchStartedChanged    // Used to dispatch this event before a possible RoundStart event and after a possible RoundEnd event
	// Used to mimic missing player_blind events for CS2 demos.
	//
	// When a player throws a flashbang the following happens:
	// 1. A player throws a flashbang
	// 2. A projectile entity is created
	// 3. The projectile entity is destroyed a few seconds later which means the flashbang exploded
	// 4. The prop m_flFlashDuration is updated for all players that are flashed
	//
	// The problem is that the order of the steps 3 and 4 is not guaranteed.
	// So it's not reliable to dispatch player-flashed events either when the projectile is destroyed or when the
	// m_flFlashDuration prop is updated.
	//
	// As a solution, we keep track of flashbang projectiles created and all m_flFlashDuration prop updates related
	// to this projectile. As all m_flFlashDuration prop updates occur during the same frame, we batch dispatch
	// player-flashed events at the end of the frame if there are any.
	// Flashbangs currently in flight (thrown but not yet detonated). Entries are keyed by the
	// projectile identity, not by position - a flashbang is removed when its own projectile entity
	// is destroyed, so a zero-victim detonation no longer desyncs attribution for the round.
	flyingFlashbangs []*FlyingFlashbang
	// Pawn entity-IDs whose m_flFlashDuration was set during the current frame. They are paired with
	// the flashbang(s) that detonated in the same frame at the end of the frame.
	flashedEntitiesThisFrame []int
}

type FlyingFlashbang struct {
	projectile    *common.GrenadeProjectile
	position      r3.Vector // Detonation position, set when the projectile is destroyed. Used to pair victims when multiple flashbangs detonate on the same frame.
	explodedFrame int
}

// infernoFireState tracks an inferno's burning state so InfernoFireOut can be dispatched once, when
// its active fires first transition from >0 to 0 (the true flame-out, distinct from InfernoExpired).
type infernoFireState struct {
	wasBurning bool
	firedOut   bool
}

// pendingInfernoLink is one side of an inferno <-> fire-grenade projectile link that was seen during
// a frame before its counterpart (see parser.linkInfernoProjectile).
// Exactly one of projectile / infernoEntityID is set.
type pendingInfernoLink struct {
	frame           int
	ownerHandle     uint64
	projectile      *common.GrenadeProjectile
	infernoEntityID int
}

type lastFlash struct {
	player             *common.Player
	projectileByPlayer map[*common.Player]*common.GrenadeProjectile
}

type ingameTickNumber int

func (gs *GameState) handleIngameTickNumber(n ingameTickNumber) {
	gs.ingameTick = int(n)
	debugIngameTick(gs.ingameTick)
}

func (gs *GameState) indexPlayerBySteamID(pl *common.Player) {
	if !pl.IsBot && pl.SteamID64 > 0 {
		gs.playersBySteamID32[common.ConvertSteamID64To32(pl.SteamID64)] = pl
	}
}

// IngameTick returns the latest actual tick number of the server during the game.
//
// Watch out, I've seen this return wonky negative numbers at the start of demos.
func (gs *GameState) IngameTick() int {
	return gs.ingameTick
}

// Team returns the TeamState corresponding to team.
// Returns nil if team != TeamTerrorists && team != TeamCounterTerrorists.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *GameState) Team(team common.Team) *common.TeamState {
	switch team { //nolint:exhaustive
	case common.TeamTerrorists:
		return &gs.tState
	case common.TeamCounterTerrorists:
		return &gs.ctState
	}

	return nil
}

// TeamCounterTerrorists returns the TeamState of the CT team.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *GameState) TeamCounterTerrorists() *common.TeamState {
	return &gs.ctState
}

// TeamTerrorists returns the TeamState of the T team.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *GameState) TeamTerrorists() *common.TeamState {
	return &gs.tState
}

// Participants returns a struct with all currently connected players & spectators and utility functions.
// The struct contains references to the original maps so it's always up-to-date.
func (gs *GameState) Participants() Participants {
	return Participants{
		playersByEntityID: gs.playersByEntityID,
		playersByUserID:   gs.playersByUserID,
	}
}

// participantsRaw returns the internal user-ID-keyed player map for direct
// iteration by parser internals that neither need snapshot semantics nor a
// deterministic order. Not part of the public API.
func (gs *GameState) participantsRaw() map[int]*common.Player {
	return gs.playersByUserID
}

// Rules returns the GameRules for the current match.
// Contains information like freeze time duration etc.
//
// The result is live: it reflects later changes (e.g. ConVars and the CCSGameRulesProxy entity
// returned by Entity(), which only exists once its entity was created), so it may be kept around.
func (gs *GameState) Rules() *GameRules {
	return &gs.rules
}

// Hostages returns all current hostages, sorted by entity-ID.
func (gs *GameState) Hostages() []*common.Hostage {
	return valuesSortedByKey(gs.hostages)
}

// GrenadeProjectiles returns a map from entity-IDs to all live grenade projectiles.
//
// Only constains projectiles currently in-flight or still active (smokes etc.),
// i.e. have been thrown but have yet to detonate.
//
// Note: this is a map, so ranging it yields a non-deterministic order. For reproducible output sort
// the values by entity-ID or GrenadeProjectile.UniqueID() (which is itself deterministic, see common.UniqueID) first.
func (gs *GameState) GrenadeProjectiles() map[int]*common.GrenadeProjectile {
	return gs.grenadeProjectiles
}

// Infernos returns a map from entity-IDs to all currently burning infernos (fires from incendiaries and Molotovs).
//
// Note: this is a map, so ranging it yields a non-deterministic order. For reproducible output sort
// the values by entity-ID or Inferno.UniqueID() (which is itself deterministic, see common.UniqueID) first.
func (gs *GameState) Infernos() map[int]*common.Inferno {
	return gs.infernos
}

// Weapons returns a map from entity-IDs to all weapons currently in the game.
//
// A weapon whose entity left the snapshot (PVS) without being deleted stays listed, with its last Owner and
// its stale Entity, until the entity is destroyed - e.g. on GOTV demos a player's old knife after a respawn.
// If it re-enters the PVS, it's the same Entity again (see Entity.OnEnterPVS).
// Check Equipment.Entity != nil && !Equipment.Entity.InPVS() (or use Entity.OnLeavePVS) for what's stale.
// Player.Inventory / Player.Weapons() follow the player's m_hMyWeapons instead.
func (gs *GameState) Weapons() map[int]*common.Equipment {
	return gs.weapons
}

// Entities returns all currently existing entities.
// (Almost?) everything in the game is an entity, such as weapons, players, fire etc.
//
// Entities that left the snapshot (PVS) without being deleted stay listed (with InPVS() == false) until they're
// destroyed. If they re-enter the PVS, they're the same Entity again, see Entity.OnLeavePVS and Entity.OnEnterPVS.
func (gs *GameState) Entities() map[int]*st.Entity {
	return gs.entities
}

// Bomb returns the current bomb state.
func (gs *GameState) Bomb() *common.Bomb {
	return &gs.bomb
}

// TotalRoundsPlayed returns the amount of total rounds played according to CCSGameRulesProxy.
func (gs *GameState) TotalRoundsPlayed() int {
	return gs.totalRoundsPlayed
}

// GamePhase returns the game phase of the current game state. See common/gamerules.go for more.
func (gs *GameState) GamePhase() common.GamePhase {
	return gs.gamePhase
}

// IsWarmupPeriod returns whether the game is currently in warmup period according to CCSGameRulesProxy.
func (gs *GameState) IsWarmupPeriod() bool {
	return gs.isWarmupPeriod
}

// IsFreezetimePeriod returns whether the game is currently in freezetime period according to CCSGameRulesProxy.
func (gs *GameState) IsFreezetimePeriod() bool {
	return gs.isFreezetime
}

// IsMatchStarted returns whether the match has started according to CCSGameRulesProxy.
func (gs *GameState) IsMatchStarted() bool {
	return gs.isMatchStarted
}

// OvertimeCount returns the number of overtime according to CCSGameRulesProxy.
func (gs *GameState) OvertimeCount() int {
	return gs.overtimeCount
}

// isEntityReplaced returns true if the entity was destroyed because a new entity was created at its index,
// rather than being deleted (see sendtables.Entity.Replaced()).
func isEntityReplaced(entity *st.Entity) bool {
	return entity != nil && entity.Replaced()
}

func entityIDFromHandle(handle uint64) int {
	if handle == constants.InvalidEntityHandleSource2 {
		return -1
	}

	return int(handle & constants.EntityHandleIndexMaskSource2)
}

// EntityByHandle returns the entity corresponding to the given handle.
// Returns nil if the handle is invalid.
func (gs *GameState) EntityByHandle(handle uint64) *st.Entity {
	return gs.entities[entityIDFromHandle(handle)]
}

// grenadeThrowEndedGraceTicks is how many ticks after a primed grenade weapon stopped being the pawn's
// (endGrenadeThrow) its record still counts for a projectile: the last grenade of a type is destroyed in the
// packet that creates the projectile entity, possibly before the projectile is created.
// (It was set to 2 when POV demos still re-created the same projectile entity on the 1-2 ticks after the throw,
// re-entries keep the entity since #742.)
const grenadeThrowEndedGraceTicks = 2

// pendingThrowKey identifies a thrower's pawn and the thrown grenade type.
type pendingThrowKey struct {
	pawnID  int
	wepType common.EquipmentType
}

// pendingThrow is a grenade weapon whose throw was primed, see primeGrenadeThrow.
type pendingThrow struct {
	wep *common.Equipment
	// ended is whether the weapon stopped being the pawn's (destroyed or owner changed) and endedTick the
	// ingame tick it did, see endGrenadeThrow
	ended     bool
	endedTick int
}

// primeGrenadeThrow remembers that the pawn's grenade weapon is about to be thrown: its pin was pulled
// (m_bPinPulled) or its throw was released (m_fThrowTime). The projectile entity is created 6-15 ticks after
// the release and is attributed to the weapon by grenadeThrowWeapon.
//
// Keyed by the pawn rather than the player: the weapon entity knows its owner pawn, so no player lookup is
// needed, and a bot taken over in place keeps its pawn. A newer priming of the same type replaces an older one.
//
// The record isn't aged out: the pin may be held for any length of time, and the release isn't always
// networked (then the pin pull is the only signal). While the weapon is still the pawn's, it is the pawn's
// grenade of that type whether or not the throw was seen. Once it isn't (endGrenadeThrow), the record only
// counts for the rest of that tick.
func (gs *GameState) primeGrenadeThrow(pawnID int, wep *common.Equipment) {
	gs.pendingGrenadeThrows[pendingThrowKey{pawnID: pawnID, wepType: wep.Type}] = pendingThrow{wep: wep}
}

// isPrimedGrenadeThrow returns whether the pawn's current record for the weapon's type is the weapon and hasn't ended.
func (gs *GameState) isPrimedGrenadeThrow(pawnID int, wep *common.Equipment) bool {
	pending, ok := gs.pendingGrenadeThrows[pendingThrowKey{pawnID: pawnID, wepType: wep.Type}]

	return ok && pending.wep == wep && !pending.ended
}

// endGrenadeThrow records that the primed grenade weapon stopped being the pawn's: it was destroyed or its owner
// changed. Its throw may still be attributed for grenadeThrowEndedGraceTicks, but not later.
func (gs *GameState) endGrenadeThrow(pawnID int, wep *common.Equipment) {
	key := pendingThrowKey{pawnID: pawnID, wepType: wep.Type}

	pending, ok := gs.pendingGrenadeThrows[key]
	if !ok || pending.wep != wep || pending.ended {
		return
	}

	pending.ended, pending.endedTick = true, gs.ingameTick
	gs.pendingGrenadeThrows[key] = pending
}

// grenadeThrowWeapon returns the grenade weapon of the given type that the pawn primed (primeGrenadeThrow) and
// that is still the pawn's, or stopped being it within the last grenadeThrowEndedGraceTicks ticks. nil if there
// is none. Molotovs and incendiaries count as the same type, but the exact type is preferred.
//
// The record isn't consumed. (Before #742 POV demos re-created the same projectile entity on the ticks after the throw,
// re-entries keep the entity now.)
func (gs *GameState) grenadeThrowWeapon(pawnID int, wepType common.EquipmentType) *common.Equipment {
	for _, t := range [2]common.EquipmentType{wepType, fireGrenadeAlternative(wepType)} {
		if t == common.EqUnknown {
			continue
		}

		pending, ok := gs.pendingGrenadeThrows[pendingThrowKey{pawnID: pawnID, wepType: t}]
		if !ok {
			continue
		}

		if !pending.ended {
			return pending.wep
		}

		// a negative age means the ticks jumped back (e.g. a server restart on a POV demo): not the same throw
		if age := gs.ingameTick - pending.endedTick; age >= 0 && age <= grenadeThrowEndedGraceTicks {
			return pending.wep
		}
	}

	return nil
}

// updateLastKnownGrenadeWeapons caches the grenade-type weapons currently in the player's inventory.
//
// CS2 removes a grenade from the thrower's inventory before the projectile entity is fully
// created - usually when the last grenade of the type is thrown. The cache allows resolving
// the weapon instance for a thrown grenade even after that removal (#580).
//
// Entries are only ever overwritten, never deleted: a weapon disappearing from the inventory
// is exactly the case this cache exists for.
//
// If there are several of a type (e.g. one was just picked up), the choice is the one
// findPlayerWeapon() makes. Called every frame for every player, so it's a single pass
// over the inventory that doesn't allocate (besides creating the player's cache).
func (gs *GameState) updateLastKnownGrenadeWeapons(pl *common.Player) {
	type candidate struct {
		wep      *common.Equipment
		entityID int
		several  bool
	}

	// the lowest entity ID per grenade type
	candidates := make([]candidate, 0, 8)

	for entityID, wep := range pl.Inventory {
		if wep.Class() != common.EqClassGrenade {
			continue
		}

		i := slices.IndexFunc(candidates, func(c candidate) bool { return c.wep.Type == wep.Type })
		if i < 0 {
			candidates = append(candidates, candidate{wep: wep, entityID: entityID})

			continue
		}

		candidates[i].several = true

		if entityID < candidates[i].entityID {
			candidates[i].wep, candidates[i].entityID = wep, entityID
		}
	}

	if len(candidates) == 0 {
		return
	}

	perType, ok := gs.lastKnownGrenadeWeapons[pl]
	if !ok {
		perType = make(map[common.EquipmentType]*common.Equipment)
		gs.lastKnownGrenadeWeapons[pl] = perType
	}

	for _, c := range candidates {
		wep := c.wep

		if c.several {
			if active := activeWeapon(pl); active != nil && active.Type == wep.Type {
				wep = active
			}
		}

		perType[wep.Type] = wep
	}
}

// lastKnownGrenadeWeapon returns the last weapon of the given grenade type that was seen
// in the player's inventory, or nil if none was seen.
func (gs *GameState) lastKnownGrenadeWeapon(pl *common.Player, wepType common.EquipmentType) *common.Equipment {
	return gs.lastKnownGrenadeWeapons[pl][wepType]
}

func newGameState(demoInfo demoInfoProvider) *GameState {
	gs := &GameState{
		playerControllerEntities: make(map[int]*st.Entity),
		playersByEntityID:        make(map[int]*common.Player),
		playersByUserID:          make(map[int]*common.Player),
		playersBySteamID32:       make(map[uint32]*common.Player),
		grenadeProjectiles:       make(map[int]*common.GrenadeProjectile),
		infernos:                 make(map[int]*common.Inferno),
		infernoFireStates:        make(map[int]*infernoFireState),
		infernoProjectiles:       make(map[int]*common.GrenadeProjectile),
		weapons:                  make(map[int]*common.Equipment),
		hostages:                 make(map[int]*common.Hostage),
		entities:                 make(map[int]*st.Entity),
		thrownGrenades:           make(map[*common.Player]map[common.EquipmentType][]*common.Equipment),
		pendingGrenadeThrows:     make(map[pendingThrowKey]pendingThrow),
		lastKnownGrenadeWeapons:  make(map[*common.Player]map[common.EquipmentType]*common.Equipment),
		flyingFlashbangs:         make([]*FlyingFlashbang, 0),
		lastFlash: lastFlash{
			projectileByPlayer: make(map[*common.Player]*common.GrenadeProjectile),
		},
		rules: GameRules{
			conVars: make(map[string]string),
		},
		demoInfo: demoInfo,
	}

	gs.tState = common.NewTeamState(common.TeamTerrorists, gs.Participants().TeamMembers, gs.demoInfo)
	gs.ctState = common.NewTeamState(common.TeamCounterTerrorists, gs.Participants().TeamMembers, gs.demoInfo)
	gs.tState.Opponent = &gs.ctState
	gs.ctState.Opponent = &gs.tState

	return gs
}

// GameRules provides information about the rules of the current match, such as round, freeze and bomb time.
//
// It's obtained from GameState.Rules() and is always up-to-date; the zero value is not usable.
type GameRules struct {
	conVars map[string]string
	entity  *st.Entity
}

// ErrFailedToRetrieveGameRule is returned by GameRules methods if the value isn't available (yet).
// It's recommended to have a fallback to a default value for this scenario.
var ErrFailedToRetrieveGameRule = errors.New("failed to retrieve GameRule value")

// RoundTime returns how long rounds in the current match last for (excluding freeze time).
// May return error if m_pGameRules.m_iRoundTime is not set.
func (gr *GameRules) RoundTime() (time.Duration, error) {
	if gr.entity == nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	prop := gr.entity.Property(gameRulesPrefixS2 + ".m_iRoundTime")
	if prop == nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	return time.Duration(prop.Value().Int()) * time.Second, nil
}

// FreezeTime returns how long freeze time lasts for in the current match (mp_freezetime).
// May return error if mp_freezetime cannot be converted to a time duration.
func (gr *GameRules) FreezeTime() (time.Duration, error) {
	t, err := strconv.Atoi(gr.conVars["mp_freezetime"])
	if err != nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	return time.Duration(t) * time.Second, nil
}

// BombTime returns how long it takes for a planted bomb to explode in the current match (mp_c4timer).
// May return error if mp_c4timer cannot be converted to a time duration.
func (gr *GameRules) BombTime() (time.Duration, error) {
	t, err := strconv.Atoi(gr.conVars["mp_c4timer"])
	if err != nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	return time.Duration(t) * time.Second, nil
}

// ConVars returns a map of CVar keys and values.
// Not all values might be set.
// See also: https://developer.valvesoftware.com/wiki/List_of_CS:GO_Cvars.
func (gr *GameRules) ConVars() map[string]string {
	return gr.conVars
}

// Entity returns the game's CCSGameRulesProxy entity.
func (gr *GameRules) Entity() *st.Entity {
	return gr.entity
}

// Participants provides helper functions on top of the currently connected players.
// E.g. ByUserID(), ByEntityID(), TeamMembers(), etc.
//
// It's obtained from GameState.Participants() and holds references to the parser's internal player maps,
// so it's always up-to-date. The zero value has no participants.
type Participants struct {
	playersByUserID   map[int]*common.Player // Maps user-IDs to players
	playersByEntityID map[int]*common.Player // Maps entity-IDs to players
}

// ByUserID returns all currently connected players in a map where the key is the user-ID.
// The returned map is a snapshot and is not updated on changes (not a reference to the actual, underlying map).
// Includes spectators.
func (ptcp Participants) ByUserID() map[int]*common.Player {
	res := make(map[int]*common.Player)

	for k, v := range ptcp.playersByUserID {
		// We need to check if the player entity hasn't been destroyed yet
		// See https://github.com/markus-wa/demoinfocs-golang/issues/98
		if v.IsConnected && v.Entity != nil {
			res[k] = v
		}
	}

	return res
}

// ByEntityID returns all currently connected players in a map where the key is the entity-ID.
// The returned map is a snapshot and is not updated on changes (not a reference to the actual, underlying map).
// Includes spectators.
func (ptcp Participants) ByEntityID() map[int]*common.Player {
	res := make(map[int]*common.Player)
	for k, v := range ptcp.playersByEntityID {
		res[k] = v
	}

	return res
}

// AllByUserID returns all currently known players & spectators, including disconnected ones,
// in a map where the key is the user-ID.
// The returned map is a snapshot and is not updated on changes (not a reference to the actual, underlying map).
// Includes spectators.
func (ptcp Participants) AllByUserID() map[int]*common.Player {
	res := make(map[int]*common.Player)
	for k, v := range ptcp.playersByUserID {
		res[k] = v
	}

	return res
}

// All returns all currently known players & spectators, including disconnected ones, of the demo.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp Participants) All() []*common.Player {
	res := make([]*common.Player, 0, len(ptcp.playersByUserID))
	for _, p := range ptcp.playersByUserID {
		res = append(res, p)
	}

	sortPlayers(res)

	return res
}

// sortPlayers sorts players by a stable key so that slice accessors are deterministic across runs
// (they are built by ranging maps, whose iteration order Go randomises). SteamID64 is stable across
// demos; UserID breaks ties (e.g. between bots, whose SteamID64 is 0).
func sortPlayers(players []*common.Player) {
	slices.SortFunc(players, func(a, b *common.Player) int {
		return cmp.Or(cmp.Compare(a.SteamID64, b.SteamID64), cmp.Compare(a.UserID, b.UserID))
	})
}

// valuesSortedByKey returns a snapshot of m's values in ascending key order,
// so callers don't depend on Go's randomised map iteration order.
func valuesSortedByKey[K cmp.Ordered, V any](m map[K]V) []V {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	res := make([]V, 0, len(keys))
	for _, k := range keys {
		res = append(res, m[k])
	}

	return res
}

// Connected returns all currently connected players & spectators.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp Participants) Connected() []*common.Player {
	res, original := ptcp.initializeSliceFromByUserID()
	for _, p := range original {
		res = append(res, p)
	}

	sortPlayers(res)

	return res
}

// Playing returns all players that aren't spectating or unassigned.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp Participants) Playing() []*common.Player {
	res, original := ptcp.initializeSliceFromByUserID()
	for _, p := range original {
		if p.Team != common.TeamSpectators && p.Team != common.TeamUnassigned {
			res = append(res, p)
		}
	}

	sortPlayers(res)

	return res
}

// TeamMembers returns all players belonging to the requested team at this time.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp Participants) TeamMembers(team common.Team) []*common.Player {
	res, original := ptcp.initializeSliceFromByUserID()
	for _, p := range original {
		if p.Team == team {
			res = append(res, p)
		}
	}

	sortPlayers(res)

	return res
}

// FindByPawnHandle attempts to find a player by his pawn entity-handle.
// This works only for Source 2 demos.
//
// Returns nil if not found.
func (ptcp Participants) FindByPawnHandle(handle uint64) *common.Player {
	entityID := entityIDFromHandle(handle)

	for _, player := range ptcp.All() {
		pawnEntity := player.PlayerPawnEntity()

		if pawnEntity == nil {
			continue
		}

		if pawnEntity.ID() == entityID {
			return player
		}
	}

	return nil
}

// FindByHandle64 attempts to find a player by his entity-handle.
// The entity-handle is often used in entity-properties when referencing other entities such as a weapon's owner.
//
// Returns nil if not found or if handle == invalidEntityHandle (used when referencing no entity).
func (ptcp Participants) FindByHandle64(handle uint64) *common.Player {
	return ptcp.playersByEntityID[entityIDFromHandle(handle)]
}

func (ptcp Participants) initializeSliceFromByUserID() ([]*common.Player, map[int]*common.Player) {
	byUserID := ptcp.ByUserID()
	return make([]*common.Player, 0, len(byUserID)), byUserID
}

// SpottersOf returns a list of all players who have spotted the passed player, sorted like All().
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (ptcp Participants) SpottersOf(spotted *common.Player) (spotters []*common.Player) {
	for _, other := range ptcp.All() {
		if spotted.IsSpottedBy(other) {
			spotters = append(spotters, other)
		}
	}

	return
}

// SpottedBy returns a list of all players that the passed player has spotted, sorted like All().
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (ptcp Participants) SpottedBy(spotter *common.Player) (spotted []*common.Player) {
	for _, other := range ptcp.All() {
		if other.Entity != nil && other.IsSpottedBy(spotter) {
			spotted = append(spotted, other)
		}
	}

	return
}
