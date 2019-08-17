// DO NOT EDIT: Auto generated

package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/common"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// IGameState is an auto-generated interface for GameState.
// GameState contains all game-state relevant information.
type IGameState interface {
	// IngameTick returns the latest actual tick number of the server during the game.
	//
	// Watch out, I've seen this return wonky negative numbers at the start of demos.
	IngameTick() int
	// Team returns the TeamState corresponding to team.
	// Returns nil if team != TeamTerrorists && team != TeamCounterTerrorists.
	//
	// Make sure to handle swapping sides properly if you keep the reference.
	Team(team common.Team) *common.TeamState
	// TeamCounterTerrorists returns the TeamState of the CT team.
	//
	// Make sure to handle swapping sides properly if you keep the reference.
	TeamCounterTerrorists() *common.TeamState
	// TeamTerrorists returns the TeamState of the T team.
	//
	// Make sure to handle swapping sides properly if you keep the reference.
	TeamTerrorists() *common.TeamState
	// Participants returns a struct with all currently connected players & spectators and utility functions.
	// The struct contains references to the original maps so it's always up-to-date.
	Participants() IParticipants
	// GrenadeProjectiles returns a map from entity-IDs to all live grenade projectiles.
	//
	// Only constains projectiles currently in-flight or still active (smokes etc.),
	// i.e. have been thrown but have yet to detonate.
	GrenadeProjectiles() map[int]*common.GrenadeProjectile
	// Infernos returns a map from entity-IDs to all currently burning infernos (fires from incendiaries and Molotovs).
	Infernos() map[int]*common.Inferno
	// Entities returns all currently existing entities.
	// (Almost?) everything in the game is an entity, such as weapons, players, fire etc.
	Entities() map[int]*st.Entity
	// Bomb returns the current bomb state.
	Bomb() *common.Bomb
	// TotalRoundsPlayed returns the amount of total rounds played according to CCSGameRulesProxy.
	TotalRoundsPlayed() int
	// GamePhase returns the game phase of the current game state. See common/gamerules.go for more.
	GamePhase() common.GamePhase
	// IsWarmupPeriod returns whether the game is currently in warmup period according to CCSGameRulesProxy.
	IsWarmupPeriod() bool
	// IsMatchStarted returns whether the match has started according to CCSGameRulesProxy.
	IsMatchStarted() bool
	// ConVars returns a map of CVar keys and values.
	// Not all values might be set.
	// See also: https://developer.valvesoftware.com/wiki/List_of_CS:GO_Cvars.
	ConVars() map[string]string
}
