package demoinfocs

import (
	"errors"
	"slices"
	"sort"
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
	lastKnownGrenadeWeapons      map[*common.Player]map[common.EquipmentType]*common.Equipment   // Last-known grenade weapon per player & type, see updateLastKnownGrenadeWeapons
	removedGrenadeWeapons        map[*common.Player]map[common.EquipmentType]removedGrenade      // Grenade weapon that last left the inventory per player & type, see recordRemovedGrenadeWeapons
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

// Hostages returns all current hostages.
func (gs *GameState) Hostages() []*common.Hostage {
	hostages := make([]*common.Hostage, 0, len(gs.hostages))
	for _, hostage := range gs.hostages {
		hostages = append(hostages, hostage)
	}

	return hostages
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
func (gs *GameState) Weapons() map[int]*common.Equipment {
	return gs.weapons
}

// Entities returns all currently existing entities.
// (Almost?) everything in the game is an entity, such as weapons, players, fire etc.
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

// removedGrenade is a grenade weapon that left a player's inventory at the given ingame tick.
type removedGrenade struct {
	wep  *common.Equipment
	tick int
}

// inventoryEntry is a weapon in a player's inventory with its entity ID.
type inventoryEntry struct {
	entityID int
	wep      *common.Equipment
}

// recordRemovedGrenadeWeapons remembers the grenade weapons that just left the player's inventory,
// i.e. that are in the previous but not in the current inventory (by entity ID).
//
// CS2 removes the weapon of a thrown grenade from the inventory when the last grenade of the type
// is thrown. If the player picks up a grenade of the same type on the same tick, the picked up one
// is the only one of the type left in the inventory - but the projectile is still the removed one's.
// Sometimes CS2 keeps the thrown grenade's weapon entity instead (the pickup tops it up), then nothing
// is removed and the weapon is found in the inventory.
//
// previous must be sorted by descending entity ID: if several of a type are removed at once, the one with
// the lowest entity ID is recorded (last), like findPlayerWeapon() would choose.
// The class is checked here, not when previous is built: a placeholder weapon may only get its type later (bindWeapon).
func (gs *GameState) recordRemovedGrenadeWeapons(pl *common.Player, previous []inventoryEntry, current map[int]*common.Equipment) {
	for _, e := range previous {
		if e.wep.Class() == common.EqClassGrenade && current[e.entityID] != e.wep {
			gs.recordRemovedGrenadeWeapon(pl, e.wep)
		}
	}
}

func (gs *GameState) recordRemovedGrenadeWeapon(pl *common.Player, wep *common.Equipment) {
	perType, ok := gs.removedGrenadeWeapons[pl]
	if !ok {
		perType = make(map[common.EquipmentType]removedGrenade)
		gs.removedGrenadeWeapons[pl] = perType
	}

	perType[wep.Type] = removedGrenade{wep: wep, tick: gs.ingameTick}
}

// takeRemovedGrenadeWeapon returns and forgets the grenade weapon of the given type that left the player's
// inventory on the current or the previous tick. Returns nil if there is none.
// Molotovs and incendiaries are treated as the same, but one of the exact type is preferred.
func (gs *GameState) takeRemovedGrenadeWeapon(pl *common.Player, wepType common.EquipmentType) *common.Equipment {
	perType := gs.removedGrenadeWeapons[pl]

	for _, removedType := range [2]common.EquipmentType{wepType, fireGrenadeAlternative(wepType)} {
		removed, ok := perType[removedType]
		if !ok {
			continue
		}

		delete(perType, removedType)

		// older ones were dropped or used up otherwise, not thrown now
		if age := gs.ingameTick - removed.tick; age >= 0 && age <= 1 {
			return removed.wep
		}
	}

	return nil
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
		lastKnownGrenadeWeapons:  make(map[*common.Player]map[common.EquipmentType]*common.Equipment),
		removedGrenadeWeapons:    make(map[*common.Player]map[common.EquipmentType]removedGrenade),
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
	sort.Slice(players, func(i, j int) bool {
		a, b := players[i], players[j]
		if a.SteamID64 != b.SteamID64 {
			return a.SteamID64 < b.SteamID64
		}

		return a.UserID < b.UserID
	})
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

// SpottersOf returns a list of all players who have spotted the passed player.
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (ptcp Participants) SpottersOf(spotted *common.Player) (spotters []*common.Player) {
	for _, other := range ptcp.playersByUserID {
		if spotted.IsSpottedBy(other) {
			spotters = append(spotters, other)
		}
	}

	return
}

// SpottedBy returns a list of all players that the passed player has spotted.
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (ptcp Participants) SpottedBy(spotter *common.Player) (spotted []*common.Player) {
	for _, other := range ptcp.playersByUserID {
		if other.Entity != nil && other.IsSpottedBy(spotter) {
			spotted = append(spotted, other)
		}
	}

	return
}
