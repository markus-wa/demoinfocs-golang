// DO NOT EDIT: Auto generated

package demoinfocs

import (
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// GameState is an auto-generated interface for gameState.
// gameState contains all game-state relevant information.
type GameState interface {
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
	Participants() Participants
	// Rules returns the GameRules for the current match.
	// Contains information like freeze time duration etc.
	Rules() GameRules
	// Hostages returns all current hostages.
	Hostages() []*common.Hostage
	// GrenadeProjectiles returns a map from entity-IDs to all live grenade projectiles.
	//
	// Only constains projectiles currently in-flight or still active (smokes etc.),
	// i.e. have been thrown but have yet to detonate.
	GrenadeProjectiles() map[int]*common.GrenadeProjectile
	// Infernos returns a map from entity-IDs to all currently burning infernos (fires from incendiaries and Molotovs).
	Infernos() map[int]*common.Inferno
	// Smokes returns a map from entity-IDs to all smokes currently in the game.
	Smokes() map[int]*common.Smoke
	// Weapons returns a map from entity-IDs to all weapons currently in the game.
	Weapons() map[int]*common.Equipment
	// Entities returns all currently existing entities.
	// (Almost?) everything in the game is an entity, such as weapons, players, fire etc.
	Entities() map[int]st.Entity
	// Bomb returns the current bomb state.
	Bomb() *common.Bomb
	// TotalRoundsPlayed returns the amount of total rounds played according to CCSGameRulesProxy.
	TotalRoundsPlayed() int
	// GamePhase returns the game phase of the current game state. See common/gamerules.go for more.
	GamePhase() common.GamePhase
	// IsWarmupPeriod returns whether the game is currently in warmup period according to CCSGameRulesProxy.
	IsWarmupPeriod() bool
	// IsFreezetimePeriod returns whether the game is currently in freezetime period according to CCSGameRulesProxy.
	IsFreezetimePeriod() bool
	// IsMatchStarted returns whether the match has started according to CCSGameRulesProxy.
	IsMatchStarted() bool
	// OvertimeCount returns the number of overtime according to CCSGameRulesProxy.
	OvertimeCount() int
	// PlayerResourceEntity returns the game's CCSPlayerResource entity.
	// Contains scoreboard information and more.
	PlayerResourceEntity() st.Entity
	// EntityByHandle returns the entity corresponding to the given handle.
	// Returns nil if the handle is invalid.
	EntityByHandle(handle uint64) st.Entity
}
