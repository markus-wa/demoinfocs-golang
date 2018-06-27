package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/common"
)

// GameState contains all game-state relevant information.
type GameState struct {
	ingameTick         int
	tState             TeamState
	ctState            TeamState
	players            map[int]*common.Player
	grenadeProjectiles map[int]*common.GrenadeProjectile // Used to keep track of grenades that have been thrown, but have not yet detonated.
}

type ingameTickNumber int

func (gs *GameState) handleIngameTickNumber(n ingameTickNumber) {
	gs.ingameTick = int(n)
	debugIngameTick(gs.ingameTick)
}

// IngameTick returns the latest actual tick number of the server during the game.
// Watch out, I've seen this return wonky negative numbers at the start of demos.
func (gs GameState) IngameTick() int {
	return gs.ingameTick
}

// CTState returns the TeamState of the CT team.
// Make sure you handle swapping sides properly if you keep the reference.
func (gs *GameState) CTState() *TeamState {
	return &gs.ctState
}

// TState returns the TeamState of the T team.
// Make sure you handle swapping sides properly if you keep the reference.
func (gs *GameState) TState() *TeamState {
	return &gs.tState
}

// Participants returns all connected players.
// This includes spectators.
func (gs GameState) Participants() []*common.Player {
	r := make([]*common.Player, 0, len(gs.players))
	for _, ptcp := range gs.players {
		r = append(r, ptcp)
	}
	return r
}

// PlayingParticipants returns all players that aren't spectating or unassigned.
func (gs GameState) PlayingParticipants() []*common.Player {
	r := make([]*common.Player, 0, len(gs.players))
	for _, ptcp := range gs.players {
		if ptcp.Team != common.TeamSpectators && ptcp.Team != common.TeamUnassigned {
			r = append(r, ptcp)
		}
	}
	return r
}

// TeamMembers returns all players belonging to the requested team.
func (gs GameState) TeamMembers(team common.Team) []*common.Player {
	r := make([]*common.Player, 0, len(gs.players))
	for _, ptcp := range gs.players {
		if ptcp.Team == team {
			r = append(r, ptcp)
		}
	}
	return r
}

// GrenadeProjectiles returns a map with all grenade projectiles. The map contains only projectiles
// that are currently in-flight, i.e. have been thrown but have yet to detonate.
func (gs GameState) GrenadeProjectiles() map[int]*common.GrenadeProjectile {
	return gs.grenadeProjectiles
}

func newGameState() GameState {
	return GameState{
		players:            make(map[int]*common.Player),
		grenadeProjectiles: make(map[int]*common.GrenadeProjectile),
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
// Watch out, in some demos this is upper-case and in some lower-case.
func (ts TeamState) Flag() string {
	return ts.flag
}
