package demoinfocs

import (
	constants "github.com/markus-wa/demoinfocs-golang/v2/internal/constants"
	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
)

//go:generate ifacemaker -f game_state.go -s gameState -i GameState -p demoinfocs -D -y "GameState is an auto-generated interface for gameState." -c "DO NOT EDIT: Auto generated" -o game_state_interface.go
//go:generate ifacemaker -f game_state.go -s participants -i Participants -p demoinfocs -D -y "Participants is an auto-generated interface for participants." -c "DO NOT EDIT: Auto generated" -o participants_interface.go

// gameState contains all game-state relevant information.
type gameState struct {
	ingameTick         int
	tState             common.TeamState
	ctState            common.TeamState
	playersByUserID    map[int]*common.Player            // Maps user-IDs to players
	playersByEntityID  map[int]*common.Player            // Maps entity-IDs to players
	grenadeProjectiles map[int]*common.GrenadeProjectile // Maps entity-IDs to active nade-projectiles. That's grenades that have been thrown, but have not yet detonated.
	infernos           map[int]*common.Inferno           // Maps entity-IDs to active infernos.
	weapons            map[int]*common.Equipment         // Maps entity IDs to weapons. Used to remember what a weapon is (p250 / cz etc.)
	entities           map[int]st.Entity                 // Maps entity IDs to entities
	conVars            map[string]string
	bomb               common.Bomb
	totalRoundsPlayed  int
	gamePhase          common.GamePhase
	isWarmupPeriod     bool
	isMatchStarted     bool
	lastFlash          lastFlash                              // Information about the last flash that exploded, used to find the attacker and projectile for player_blind events
	currentDefuser     *common.Player                         // Player currently defusing the bomb, if any
	currentPlanter     *common.Player                         // Player currently planting the bomb, if any
	thrownGrenades     map[*common.Player][]*common.Equipment // Information about every player's thrown grenades (from the moment they are thrown to the moment their effect is ended)
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
	}
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

// IsMatchStarted returns whether the match has started according to CCSGameRulesProxy.
func (gs gameState) IsMatchStarted() bool {
	return gs.isMatchStarted
}

// ConVars returns a map of CVar keys and values.
// Not all values might be set.
// See also: https://developer.valvesoftware.com/wiki/List_of_CS:GO_Cvars.
func (gs *gameState) ConVars() map[string]string {
	return gs.conVars
}

func newGameState() *gameState {
	gs := &gameState{
		playersByEntityID:  make(map[int]*common.Player),
		playersByUserID:    make(map[int]*common.Player),
		grenadeProjectiles: make(map[int]*common.GrenadeProjectile),
		infernos:           make(map[int]*common.Inferno),
		weapons:            make(map[int]*common.Equipment),
		entities:           make(map[int]st.Entity),
		conVars:            make(map[string]string),
		thrownGrenades:     make(map[*common.Player][]*common.Equipment),
		lastFlash: lastFlash{
			projectileByPlayer: make(map[*common.Player]*common.GrenadeProjectile),
		},
	}

	gs.tState = common.NewTeamState(common.TeamTerrorists, gs.Participants().TeamMembers)
	gs.ctState = common.NewTeamState(common.TeamCounterTerrorists, gs.Participants().TeamMembers)
	gs.tState.Opponent = &gs.ctState
	gs.ctState.Opponent = &gs.tState

	return gs
}

// participants provides helper functions on top of the currently connected players.
// E.g. ByUserID(), ByEntityID(), TeamMembers(), etc.
//
// See GameState.Participants()
type participants struct {
	playersByUserID   map[int]*common.Player // Maps user-IDs to players
	playersByEntityID map[int]*common.Player // Maps entity-IDs to players
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

// FindByHandle attempts to find a player by his entity-handle.
// The entity-handle is often used in entity-properties when referencing other entities such as a weapon's owner.
//
// Returns nil if not found or if handle == invalidEntityHandle (used when referencing no entity).
func (ptcp participants) FindByHandle(handle int) *common.Player {
	if handle == constants.InvalidEntityHandle {
		return nil
	}

	entityID := handle & constants.EntityHandleIndexMask
	player := ptcp.playersByEntityID[entityID]

	if player == nil {
		for _, p := range ptcp.playersByUserID {
			if p.EntityID == entityID {
				player = p
				break
			}
		}
	}

	return player
}

func (ptcp participants) initializeSliceFromByUserID() ([]*common.Player, map[int]*common.Player) {
	byUserID := ptcp.ByUserID()
	return make([]*common.Player, 0, len(byUserID)), byUserID
}

// SpottersOf returns a list of all players who have spotted the passed player.
func (ptcp participants) SpottersOf(spotted *common.Player) (spotters []*common.Player) {
	for _, other := range ptcp.playersByUserID {
		if spotted.IsSpottedBy(other) {
			spotters = append(spotters, other)
		}
	}

	return
}

// SpottedBy returns a list of all players that the passed player has spotted.
func (ptcp participants) SpottedBy(spotter *common.Player) (spotted []*common.Player) {
	for _, other := range ptcp.playersByUserID {
		if other.Entity != nil && other.IsSpottedBy(spotter) {
			spotted = append(spotted, other)
		}
	}

	return
}
