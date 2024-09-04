package fake

import (
	"time"

	"github.com/stretchr/testify/mock"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

var _ demoinfocs.GameRules = new(GameRules)

// GameRules is a mock for of demoinfocs.GameRules.
type GameRules struct {
	mock.Mock
}

func (gr *GameRules) Entity() st.Entity {
	return gr.Called().Get(0).(st.Entity)
}

// BombTime is a mock-implementation of GameRules.BombTime().
func (gr *GameRules) BombTime() (time.Duration, error) {
	return gr.Called().Get(0).(time.Duration), gr.Called().Get(0).(error)
}

// FreezeTime is a mock-implementation of GameRules.FreezeTime().
func (gr *GameRules) FreezeTime() (time.Duration, error) {
	return gr.Called().Get(0).(time.Duration), gr.Called().Get(0).(error)
}

// RoundTime is a mock-implementation of GameRules.RoundTime().
func (gr *GameRules) RoundTime() (time.Duration, error) {
	return gr.Called().Get(0).(time.Duration), gr.Called().Get(0).(error)
}

// ConVars is a mock-implementation of GameRules.ConVars().
func (gr *GameRules) ConVars() map[string]string {
	return gr.Called().Get(0).(map[string]string)
}
