package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/common"
)

func TestNewGameState(t *testing.T) {
	gs := newGameState()

	assert.Equal(t, common.TeamTerrorists, gs.tState.Team())
	assert.Equal(t, common.TeamCounterTerrorists, gs.ctState.Team())
}
