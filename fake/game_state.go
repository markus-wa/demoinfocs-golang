package fake

import (
	"github.com/stretchr/testify/mock"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

var _ dem.IGameState = new(GameState)

// GameState is a mock for of demoinfocs.IGameState.
type GameState struct {
	mock.Mock
}

// IngameTick is a mock-implementation of IGameState.IngameTick().
func (gs *GameState) IngameTick() int {
	return gs.Called().Int(0)
}

// TeamCounterTerrorists is a mock-implementation of IGameState.TeamCounterTerrorists().
func (gs *GameState) TeamCounterTerrorists() *common.TeamState {
	return gs.Called().Get(0).(*common.TeamState)
}

// TeamTerrorists is a mock-implementation of IGameState.TeamTerrorists().
func (gs *GameState) TeamTerrorists() *common.TeamState {
	return gs.Called().Get(0).(*common.TeamState)
}

// Team is a mock-implementation of IGameState.Team().
func (gs *GameState) Team(team common.Team) *common.TeamState {
	return gs.Called().Get(0).(*common.TeamState)
}

// Participants is a mock-implementation of IGameState.Participants().
func (gs *GameState) Participants() dem.IParticipants {
	return gs.Called().Get(0).(dem.IParticipants)
}

// GrenadeProjectiles is a mock-implementation of IGameState.GrenadeProjectiles().
func (gs *GameState) GrenadeProjectiles() map[int]*common.GrenadeProjectile {
	return gs.Called().Get(0).(map[int]*common.GrenadeProjectile)
}

// Infernos is a mock-implementation of IGameState.Infernos().
func (gs *GameState) Infernos() map[int]*common.Inferno {
	return gs.Called().Get(0).(map[int]*common.Inferno)
}

// Entities is a mock-implementation of IGameState.Entities().
func (gs *GameState) Entities() map[int]*st.Entity {
	return gs.Called().Get(0).(map[int]*st.Entity)
}

// Bomb is a mock-implementation of IGameState.Bomb().
func (gs *GameState) Bomb() *common.Bomb {
	return gs.Called().Get(0).(*common.Bomb)
}

// TotalRoundsPlayed is a mock-implementation of IGameState.TotalRoundsPlayed().
func (gs *GameState) TotalRoundsPlayed() int {
	return gs.Called().Int(0)
}

// GamePhase is a mock-implementation of IGameState.GamePhase().
func (gs *GameState) GamePhase() common.GamePhase {
	return gs.Called().Get(0).(common.GamePhase)
}

// IsWarmupPeriod is a mock-implementation of IGameState.IsWarmupPeriod().
func (gs *GameState) IsWarmupPeriod() bool {
	return gs.Called().Bool(0)
}

// IsMatchStarted is a mock-implementation of IGameState.IsMatchStarted().
func (gs *GameState) IsMatchStarted() bool {
	return gs.Called().Bool(0)
}

// ConVars is a mock-implementation of IGameState.ConVars().
func (gs *GameState) ConVars() map[string]string {
	return gs.Called().Get(0).(map[string]string)
}
