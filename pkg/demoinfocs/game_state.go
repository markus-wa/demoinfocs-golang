package demoinfocs

import (
	"errors"
	"strconv"
	"time"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

//go:generate ifacemaker -f game_state.go -s gameState -i GameState -p demoinfocs -D -y "GameState is an auto-generated interface for gameState." -c "DO NOT EDIT: Auto generated" -o game_state_interface.go
//go:generate ifacemaker -f game_state.go -s participants -i Participants -p demoinfocs -D -y "Participants is an auto-generated interface for participants." -c "DO NOT EDIT: Auto generated" -o participants_interface.go
//go:generate ifacemaker -f game_state.go -s gameRules -i GameRules -p demoinfocs -D -y "GameRules is an auto-generated interface for gameRules." -c "DO NOT EDIT: Auto generated" -o game_rules_interface.go

// gameState contains all game-state relevant information.
type gameState struct {
	ingameTick                   int
	tState                       common.TeamState
	ctState                      common.TeamState
	playersByUserID              map[int]*common.Player    // Maps user-IDs to players
	playersByEntityID            map[int]*common.Player    // Maps entity-IDs to players
	playersBySteamID32           map[uint32]*common.Player // Maps 32-bit-steam-IDs to players
	playerResourceEntity         st.Entity                 // CCSPlayerResource entity instance, contains scoreboard info and more
	playerControllerEntities     map[int]st.Entity
	grenadeProjectiles           map[int]*common.GrenadeProjectile // Maps entity-IDs to active nade-projectiles. That's grenades that have been thrown, but have not yet detonated.
	infernos                     map[int]*common.Inferno           // Maps entity-IDs to active infernos.
	weapons                      map[int]*common.Equipment         // Maps entity IDs to weapons. Used to remember what a weapon is (p250 / cz etc.)
	hostages                     map[int]*common.Hostage           // Maps entity-IDs to hostages.
	entities                     map[int]st.Entity                 // Maps entity IDs to entities
	bomb                         common.Bomb
	totalRoundsPlayed            int
	gamePhase                    common.GamePhase
	isWarmupPeriod               bool
	isFreezetime                 bool
	isMatchStarted               bool
	overtimeCount                int
	lastFlash                    lastFlash                              // Information about the last flash that exploded, used to find the attacker and projectile for player_blind events
	currentDefuser               *common.Player                         // Player currently defusing the bomb, if any
	currentPlanter               *common.Player                         // Player currently planting the bomb, if any
	thrownGrenades               map[*common.Player][]*common.Equipment // Information about every player's thrown grenades (from the moment they are thrown to the moment their effect is ended)
	rules                        gameRules
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
	// This slice acts like a FIFO queue, the first projectile inserted is the first one to be removed when it exploded.
	flyingFlashbangs []*FlyingFlashbang
}

type FlyingFlashbang struct {
	projectile       *common.GrenadeProjectile
	flashedEntityIDs []int
	explodedFrame    int
}

type lastFlash struct {
	player             *common.Player
	projectileByPlayer map[*common.Player]*common.GrenadeProjectile
}

type ingameTickNumber int

func (gs *gameState) handleIngameTickNumber(n ingameTickNumber) {
	gs.ingameTick = int(n)
	debugIngameTick(gs.ingameTick)
}

func (gs *gameState) indexPlayerBySteamID(pl *common.Player) {
	if !pl.IsBot && pl.SteamID64 > 0 {
		gs.playersBySteamID32[common.ConvertSteamID64To32(pl.SteamID64)] = pl
	}
}

// IngameTick returns the latest actual tick number of the server during the game.
//
// Watch out, I've seen this return wonky negative numbers at the start of demos.
func (gs gameState) IngameTick() int {
	return gs.ingameTick
}

// Team returns the TeamState corresponding to team.
// Returns nil if team != TeamTerrorists && team != TeamCounterTerrorists.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *gameState) Team(team common.Team) *common.TeamState {
	if team == common.TeamTerrorists {
		return &gs.tState
	} else if team == common.TeamCounterTerrorists {
		return &gs.ctState
	}

	return nil
}

// TeamCounterTerrorists returns the TeamState of the CT team.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *gameState) TeamCounterTerrorists() *common.TeamState {
	return &gs.ctState
}

// TeamTerrorists returns the TeamState of the T team.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *gameState) TeamTerrorists() *common.TeamState {
	return &gs.tState
}

// Participants returns a struct with all currently connected players & spectators and utility functions.
// The struct contains references to the original maps so it's always up-to-date.
func (gs gameState) Participants() Participants {
	return participants{
		playersByEntityID: gs.playersByEntityID,
		playersByUserID:   gs.playersByUserID,
		getIsSource2:      gs.demoInfo.parser.isSource2,
	}
}

// Rules returns the GameRules for the current match.
// Contains information like freeze time duration etc.
func (gs gameState) Rules() GameRules {
	return gs.rules
}

// Hostages returns all current hostages.
func (gs gameState) Hostages() []*common.Hostage {
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
func (gs gameState) GrenadeProjectiles() map[int]*common.GrenadeProjectile {
	return gs.grenadeProjectiles
}

// Infernos returns a map from entity-IDs to all currently burning infernos (fires from incendiaries and Molotovs).
func (gs gameState) Infernos() map[int]*common.Inferno {
	return gs.infernos
}

// Weapons returns a map from entity-IDs to all weapons currently in the game.
func (gs gameState) Weapons() map[int]*common.Equipment {
	return gs.weapons
}

// Entities returns all currently existing entities.
// (Almost?) everything in the game is an entity, such as weapons, players, fire etc.
func (gs gameState) Entities() map[int]st.Entity {
	return gs.entities
}

// Bomb returns the current bomb state.
func (gs gameState) Bomb() *common.Bomb {
	return &gs.bomb
}

// TotalRoundsPlayed returns the amount of total rounds played according to CCSGameRulesProxy.
func (gs gameState) TotalRoundsPlayed() int {
	return gs.totalRoundsPlayed
}

// GamePhase returns the game phase of the current game state. See common/gamerules.go for more.
func (gs gameState) GamePhase() common.GamePhase {
	return gs.gamePhase
}

// IsWarmupPeriod returns whether the game is currently in warmup period according to CCSGameRulesProxy.
func (gs gameState) IsWarmupPeriod() bool {
	return gs.isWarmupPeriod
}

// IsFreezetimePeriod returns whether the game is currently in freezetime period according to CCSGameRulesProxy.
func (gs gameState) IsFreezetimePeriod() bool {
	return gs.isFreezetime
}

// IsMatchStarted returns whether the match has started according to CCSGameRulesProxy.
func (gs gameState) IsMatchStarted() bool {
	return gs.isMatchStarted
}

// OvertimeCount returns the number of overtime according to CCSGameRulesProxy.
func (gs gameState) OvertimeCount() int {
	return gs.overtimeCount
}

// PlayerResourceEntity returns the game's CCSPlayerResource entity.
// Contains scoreboard information and more.
func (gs gameState) PlayerResourceEntity() st.Entity {
	return gs.playerResourceEntity
}

func entityIDFromHandle(handle uint64, isS2 bool) int {
	if isS2 {
		if handle == constants.InvalidEntityHandleSource2 {
			return -1
		}

		return int(handle & constants.EntityHandleIndexMaskSource2)
	}

	if handle == constants.InvalidEntityHandle {
		return -1
	}

	return int(handle & constants.EntityHandleIndexMask)
}

// EntityByHandle returns the entity corresponding to the given handle.
// Returns nil if the handle is invalid.
func (gs gameState) EntityByHandle(handle uint64) st.Entity {
	return gs.entities[entityIDFromHandle(handle, gs.demoInfo.parser.isSource2())]
}

func newGameState(demoInfo demoInfoProvider) *gameState {
	gs := &gameState{
		playerControllerEntities: make(map[int]st.Entity),
		playersByEntityID:        make(map[int]*common.Player),
		playersByUserID:          make(map[int]*common.Player),
		playersBySteamID32:       make(map[uint32]*common.Player),
		grenadeProjectiles:       make(map[int]*common.GrenadeProjectile),
		infernos:                 make(map[int]*common.Inferno),
		weapons:                  make(map[int]*common.Equipment),
		hostages:                 make(map[int]*common.Hostage),
		entities:                 make(map[int]st.Entity),
		thrownGrenades:           make(map[*common.Player][]*common.Equipment),
		flyingFlashbangs:         make([]*FlyingFlashbang, 0),
		lastFlash: lastFlash{
			projectileByPlayer: make(map[*common.Player]*common.GrenadeProjectile),
		},
		rules: gameRules{
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

type gameRules struct {
	conVars map[string]string
	entity  st.Entity
}

var ErrFailedToRetrieveGameRule = errors.New("failed to retrieve GameRule value, it's recommended to have a fallback to a default value for this scenario")

// RoundTime returns how long rounds in the current match last for (excluding freeze time).
// May return error if cs_gamerules_data.m_iRoundTime is not set.
func (gr gameRules) RoundTime() (time.Duration, error) {
	if gr.entity == nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	prop := gr.entity.Property("cs_gamerules_data.m_iRoundTime")
	if prop == nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	return time.Duration(prop.Value().IntVal) * time.Second, nil
}

// FreezeTime returns how long freeze time lasts for in the current match (mp_freezetime).
// May return error if mp_freezetime cannot be converted to a time duration.
func (gr gameRules) FreezeTime() (time.Duration, error) {
	t, err := strconv.Atoi(gr.conVars["mp_freezetime"])
	if err != nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	return time.Duration(t) * time.Second, nil
}

// BombTime returns how long freeze time lasts for in the current match (mp_freezetime).
// May return error if mp_c4timer cannot be converted to a time duration.
func (gr gameRules) BombTime() (time.Duration, error) {
	t, err := strconv.Atoi(gr.conVars["mp_c4timer"])
	if err != nil {
		return 0, ErrFailedToRetrieveGameRule
	}

	return time.Duration(t) * time.Second, nil
}

// ConVars returns a map of CVar keys and values.
// Not all values might be set.
// See also: https://developer.valvesoftware.com/wiki/List_of_CS:GO_Cvars.
func (gr gameRules) ConVars() map[string]string {
	return gr.conVars
}

// Entity returns the game's CCSGameRulesProxy entity.
func (gr gameRules) Entity() st.Entity {
	return gr.entity
}

// participants provides helper functions on top of the currently connected players.
// E.g. ByUserID(), ByEntityID(), TeamMembers(), etc.
//
// See GameState.Participants()
type participants struct {
	playersByUserID   map[int]*common.Player // Maps user-IDs to players
	playersByEntityID map[int]*common.Player // Maps entity-IDs to players
	getIsSource2      func() bool
}

// ByUserID returns all currently connected players in a map where the key is the user-ID.
// The returned map is a snapshot and is not updated on changes (not a reference to the actual, underlying map).
// Includes spectators.
func (ptcp participants) ByUserID() map[int]*common.Player {
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
func (ptcp participants) ByEntityID() map[int]*common.Player {
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
func (ptcp participants) AllByUserID() map[int]*common.Player {
	res := make(map[int]*common.Player)
	for k, v := range ptcp.playersByUserID {
		res[k] = v
	}

	return res
}

// All returns all currently known players & spectators, including disconnected ones, of the demo.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp participants) All() []*common.Player {
	res := make([]*common.Player, 0, len(ptcp.playersByUserID))
	for _, p := range ptcp.playersByUserID {
		res = append(res, p)
	}

	return res
}

// Connected returns all currently connected players & spectators.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp participants) Connected() []*common.Player {
	res, original := ptcp.initializeSliceFromByUserID()
	for _, p := range original {
		res = append(res, p)
	}

	return res
}

// Playing returns all players that aren't spectating or unassigned.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp participants) Playing() []*common.Player {
	res, original := ptcp.initializeSliceFromByUserID()
	for _, p := range original {
		if p.Team != common.TeamSpectators && p.Team != common.TeamUnassigned {
			res = append(res, p)
		}
	}

	return res
}

// TeamMembers returns all players belonging to the requested team at this time.
// The returned slice is a snapshot and is not updated on changes.
func (ptcp participants) TeamMembers(team common.Team) []*common.Player {
	res, original := ptcp.initializeSliceFromByUserID()
	for _, p := range original {
		if p.Team == team {
			res = append(res, p)
		}
	}

	return res
}

// FindByPawnHandle attempts to find a player by his pawn entity-handle.
// This works only for Source 2 demos.
//
// Returns nil if not found.
func (ptcp participants) FindByPawnHandle(handle uint64) *common.Player {
	entityID := entityIDFromHandle(handle, ptcp.getIsSource2())

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
func (ptcp participants) FindByHandle64(handle uint64) *common.Player {
	return ptcp.playersByEntityID[entityIDFromHandle(handle, ptcp.getIsSource2())]
}

// FindByHandle attempts to find a player by his entity-handle.
// The entity-handle is often used in entity-properties when referencing other entities such as a weapon's owner.
//
// Returns nil if not found or if handle == invalidEntityHandle (used when referencing no entity).
//
// Deprecated: Use FindByHandle64 instead.
func (ptcp participants) FindByHandle(handle int) *common.Player {
	return ptcp.FindByHandle64(uint64(handle))
}

func (ptcp participants) initializeSliceFromByUserID() ([]*common.Player, map[int]*common.Player) {
	byUserID := ptcp.ByUserID()
	return make([]*common.Player, 0, len(byUserID)), byUserID
}

// SpottersOf returns a list of all players who have spotted the passed player.
// This is NOT "Line of Sight" / FOV - look up "CSGO TraceRay" for that.
// May not behave as expected with multiple spotters.
func (ptcp participants) SpottersOf(spotted *common.Player) (spotters []*common.Player) {
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
func (ptcp participants) SpottedBy(spotter *common.Player) (spotted []*common.Player) {
	for _, other := range ptcp.playersByUserID {
		if other.Entity != nil && other.IsSpottedBy(spotter) {
			spotted = append(spotted, other)
		}
	}

	return
}
