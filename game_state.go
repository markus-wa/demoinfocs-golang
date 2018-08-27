package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/common"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// GameState contains all game-state relevant information.
type GameState struct {
	ingameTick         int
	tState             TeamState
	ctState            TeamState
	playersByUserID    map[int]*common.Player            // Maps user-IDs to players
	playersByEntityID  map[int]*common.Player            // Maps entity-IDs to players
	grenadeProjectiles map[int]*common.GrenadeProjectile // Maps entity-IDs to active nade-projectiles. That's grenades that have been thrown, but have not yet detonated.
	infernos           map[int]*common.Inferno           // Maps entity-IDs to active infernos.
	entities           map[int]*st.Entity                // Maps entity IDs to entities
	bomb               common.Bomb
}

type ingameTickNumber int

func (gs *GameState) handleIngameTickNumber(n ingameTickNumber) {
	gs.ingameTick = int(n)
	debugIngameTick(gs.ingameTick)
}

// IngameTick returns the latest actual tick number of the server during the game.
//
// Watch out, I've seen this return wonky negative numbers at the start of demos.
func (gs GameState) IngameTick() int {
	return gs.ingameTick
}

// TeamCounterTerrorists returns the TeamState of the CT team.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *GameState) TeamCounterTerrorists() *TeamState {
	return &gs.ctState
}

// TeamTerrorists returns the TeamState of the T team.
//
// Make sure to handle swapping sides properly if you keep the reference.
func (gs *GameState) TeamTerrorists() *TeamState {
	return &gs.tState
}

// Participants returns a struct with all currently connected players & spectators and utility functions.
// The struct contains references to the original maps so it's always up-to-date.
func (gs GameState) Participants() Participants {
	return Participants{
		playersByEntityID: gs.playersByEntityID,
		playersByUserID:   gs.playersByUserID,
	}
}

// GrenadeProjectiles returns a map from entity-IDs to all live grenade projectiles.
//
// Only constains projectiles currently in-flight or still active (smokes etc.),
// i.e. have been thrown but have yet to detonate.
func (gs GameState) GrenadeProjectiles() map[int]*common.GrenadeProjectile {
	return gs.grenadeProjectiles
}

// Infernos returns a map from entity-IDs to all currently burning infernos (fires from incendiaries and Molotovs).
func (gs GameState) Infernos() map[int]*common.Inferno {
	return gs.infernos
}

// Entities returns all currently existing entities.
func (gs GameState) Entities() map[int]*st.Entity {
	return gs.entities
}

// Bomb returns the current bomb state.
func (gs GameState) Bomb() *common.Bomb {
	return &gs.bomb
}

func newGameState() GameState {
	return GameState{
		playersByEntityID:  make(map[int]*common.Player),
		playersByUserID:    make(map[int]*common.Player),
		grenadeProjectiles: make(map[int]*common.GrenadeProjectile),
		infernos:           make(map[int]*common.Inferno),
		entities:           make(map[int]*st.Entity),
	}
}

// TeamState contains a team's ID, score, clan name & country flag.
type TeamState struct {
	id       int
	score    int
	clanName string
	flag     string
}

// ID returns the team-ID.
//
// This stays the same even after switching sides.
func (ts TeamState) ID() int {
	return ts.id
}

// Score returns the team's number of rounds won.
func (ts TeamState) Score() int {
	return ts.score
}

// ClanName returns the team's clan name.
func (ts TeamState) ClanName() string {
	return ts.clanName
}

// Flag returns the team's country flag.
//
// Watch out, in some demos this is upper-case and in some lower-case.
func (ts TeamState) Flag() string {
	return ts.flag
}

// Participants provides helper functions on top of the currently connected players.
// E.g. ByUserID(), ByEntityID(), TeamMembers() etc.
//
// See GameState.Participants()
type Participants struct {
	playersByUserID   map[int]*common.Player // Maps user-IDs to players
	playersByEntityID map[int]*common.Player // Maps entity-IDs to players
}

// ByUserID returns all currently connected players in a map where the key is the user-ID.
// The map is a snapshot and is not updated (not a reference to the actual, underlying map).
// Includes spectators.
func (ptcp Participants) ByUserID() map[int]*common.Player {
	res := make(map[int]*common.Player)
	for k, v := range ptcp.playersByUserID {
		res[k] = v
	}
	return res
}

// ByEntityID returns all currently connected players in a map where the key is the entity-ID.
// The map is a snapshot and is not updated (not a reference to the actual, underlying map).
// Includes spectators.
func (ptcp Participants) ByEntityID() map[int]*common.Player {
	res := make(map[int]*common.Player)
	for k, v := range ptcp.playersByEntityID {
		res[k] = v
	}
	return res
}

// All returns all currently connected players & spectators.
func (ptcp Participants) All() []*common.Player {
	res := make([]*common.Player, 0, len(ptcp.playersByUserID))
	for _, p := range ptcp.playersByUserID {
		res = append(res, p)
	}
	return res
}

// Playing returns all players that aren't spectating or unassigned.
func (ptcp Participants) Playing() []*common.Player {
	res := make([]*common.Player, 0, len(ptcp.playersByUserID))
	for _, p := range ptcp.playersByUserID {
		if p.Team != common.TeamSpectators && p.Team != common.TeamUnassigned {
			res = append(res, p)
		}
	}
	return res
}

// TeamMembers returns all players belonging to the requested team at this time.
func (ptcp Participants) TeamMembers(team common.Team) []*common.Player {
	res := make([]*common.Player, 0, len(ptcp.playersByUserID))
	for _, ptcp := range ptcp.playersByUserID {
		if ptcp.Team == team {
			res = append(res, ptcp)
		}
	}
	return res
}

// FindByHandle attempts to find a player by his entity-handle.
// The entity-handle is often used in entity-properties when referencing other entities such as a weapon's owner.
//
// Returns nil if not found or if handle == invalidEntityHandle (used when referencing no entity).
func (ptcp Participants) FindByHandle(handle int) *common.Player {
	if handle == invalidEntityHandle {
		return nil
	}

	return ptcp.playersByEntityID[handle&entityHandleIndexMask]
}
