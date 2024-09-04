package fake

import (
	"github.com/stretchr/testify/mock"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var _ demoinfocs.GameState = new(GameState)

// GameState is a mock for of demoinfocs.GameState.
type GameState struct {
	mock.Mock
}

// OvertimeCount is a mock-implementation of GameState.OvertimeCount().
func (gs *GameState) OvertimeCount() int {
	return gs.Called().Int(0)
}

// IngameTick is a mock-implementation of GameState.IngameTick().
func (gs *GameState) IngameTick() int {
	return gs.Called().Int(0)
}

// TeamCounterTerrorists is a mock-implementation of GameState.TeamCounterTerrorists().
func (gs *GameState) TeamCounterTerrorists() *common.TeamState {
	return gs.Called().Get(0).(*common.TeamState)
}

// TeamTerrorists is a mock-implementation of GameState.TeamTerrorists().
func (gs *GameState) TeamTerrorists() *common.TeamState {
	return gs.Called().Get(0).(*common.TeamState)
}

// Team is a mock-implementation of GameState.Team().
func (gs *GameState) Team(team common.Team) *common.TeamState {
	return gs.Called().Get(0).(*common.TeamState)
}

// Participants is a mock-implementation of GameState.Participants().
func (gs *GameState) Participants() demoinfocs.Participants {
	return gs.Called().Get(0).(demoinfocs.Participants)
}

// GrenadeProjectiles is a mock-implementation of GameState.GrenadeProjectiles().
func (gs *GameState) GrenadeProjectiles() map[int]*common.GrenadeProjectile {
	return gs.Called().Get(0).(map[int]*common.GrenadeProjectile)
}

// Infernos is a mock-implementation of GameState.Infernos().
func (gs *GameState) Infernos() map[int]*common.Inferno {
	return gs.Called().Get(0).(map[int]*common.Inferno)
}

// Weapons is a mock-implementation of GameState.Weapons().
func (gs *GameState) Weapons() map[int]*common.Equipment {
	return gs.Called().Get(0).(map[int]*common.Equipment)
}

// Entities is a mock-implementation of GameState.Entities().
func (gs *GameState) Entities() map[int]st.Entity {
	return gs.Called().Get(0).(map[int]st.Entity)
}

// Bomb is a mock-implementation of GameState.Bomb().
func (gs *GameState) Bomb() *common.Bomb {
	return gs.Called().Get(0).(*common.Bomb)
}

// TotalRoundsPlayed is a mock-implementation of GameState.TotalRoundsPlayed().
func (gs *GameState) TotalRoundsPlayed() int {
	return gs.Called().Int(0)
}

// GamePhase is a mock-implementation of GameState.GamePhase().
func (gs *GameState) GamePhase() common.GamePhase {
	return gs.Called().Get(0).(common.GamePhase)
}

// IsWarmupPeriod is a mock-implementation of GameState.IsWarmupPeriod().
func (gs *GameState) IsWarmupPeriod() bool {
	return gs.Called().Bool(0)
}

// IsFreezetimePeriod is a mock-implementation of GameState.IsFreezetimePeriod().
func (gs *GameState) IsFreezetimePeriod() bool {
	return gs.Called().Bool(0)
}

// IsMatchStarted is a mock-implementation of GameState.IsMatchStarted().
func (gs *GameState) IsMatchStarted() bool {
	return gs.Called().Bool(0)
}

// Rules is a mock-implementation of GameState.Rules().
func (gs *GameState) Rules() demoinfocs.GameRules {
	return gs.Called().Get(0).(demoinfocs.GameRules)
}

// PlayerResourceEntity is a mock-implementation of GameState.PlayerResourceEntity().
func (gs *GameState) PlayerResourceEntity() st.Entity {
	return gs.Called().Get(0).(st.Entity)
}

// Hostages is a mock-implementation of GameState.Hostages().
func (gs *GameState) Hostages() []*common.Hostage {
	return gs.Called().Get(0).([]*common.Hostage)
}

// EntityByHandle is a mock-implementation of GameState.EntityByHandle().
func (gs *GameState) EntityByHandle(handle uint64) st.Entity {
	return gs.Called(handle).Get(0).(st.Entity)
}
