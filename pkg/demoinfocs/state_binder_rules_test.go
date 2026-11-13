package demoinfocs

import (
	"testing"
	"time"

	dp "github.com/markus-wa/godispatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
)

// The GameRules family is exercised without a Parser: a bare GameState, a real dispatcher and a
// synthetic CCSGameRulesProxy entity are enough to reach the binding rules.
func TestStateBinder_BindGameRules(t *testing.T) {
	stParser := stfake.NewParser("CCSGameRulesProxy")
	gameState := newGameState(demoInfoProvider{})
	dispatcher := new(dp.Dispatcher)

	var roundEndOfficials []events.RoundEndOfficial

	dispatcher.RegisterHandler(func(e events.RoundEndOfficial) {
		roundEndOfficials = append(roundEndOfficials, e)
	})

	var clearedGrenadeProjectiles int

	sb := newStateBinder(stateBinderDeps{
		gameState:       gameState,
		eventDispatcher: dispatcher,
		serverClasses:   stParser.ServerClasses,
		clearGrenadeProjectiles: func() {
			clearedGrenadeProjectiles++
		},
	})

	sb.bindGameRules()

	gr := func(name string, val any) stfake.Prop {
		return stfake.P(gameRulesPrefixS2+"."+name, val)
	}

	entity := stfake.New(stfake.Spec{
		Parser: stParser,
		Class:  "CCSGameRulesProxy",
		ID:     10,
		Props: []stfake.Prop{
			gr("m_iRoundTime", int32(115)),
			gr("m_bMapHasRescueZone", false),
			gr("m_bMapHasBombTarget", true),
			gr("m_bFreezePeriod", false),
			gr("m_gamePhase", int32(0)),
			gr("m_totalRoundsPlayed", int32(0)),
			gr("m_bWarmupPeriod", false),
			gr("m_bHasMatchStarted", false),
			gr("m_nOvertimePlaying", int32(0)),
			gr("m_eRoundWinReason", int32(0)),
		},
	})

	stfake.Create(entity)

	assert.Same(t, entity, gameState.Rules().Entity())

	// RoundTime() reads the entity property live, so an update is visible without rebinding.
	roundTime, err := gameState.Rules().RoundTime()
	require.NoError(t, err)
	assert.Equal(t, 115*time.Second, roundTime)

	stfake.Update(entity, gr("m_iRoundTime", int32(130)))

	roundTime, err = gameState.Rules().RoundTime()
	require.NoError(t, err)
	assert.Equal(t, 130*time.Second, roundTime)

	// The create packet already consumed the first m_eRoundWinReason update, so this one is a round start.
	stfake.Update(entity, gr("m_totalRoundsPlayed", int32(1)))
	stfake.Update(entity, gr("m_eRoundWinReason", int32(events.RoundEndReasonStillInProgress)))

	assert.Equal(t, 1, clearedGrenadeProjectiles)
	require.Len(t, roundEndOfficials, 1)
	require.NotNil(t, gameState.lastRoundStartEvent)
	assert.Equal(t, 130, gameState.lastRoundStartEvent.TimeLimit)
	assert.Equal(t, "BOMB TARGET", gameState.lastRoundStartEvent.Objective)

	// Any other reason ends the round.
	stfake.Update(entity, gr("m_eRoundWinReason", int32(events.RoundEndReasonTargetBombed)))

	require.NotNil(t, gameState.lastRoundEndEvent)
	assert.Equal(t, events.RoundEndReasonTargetBombed, gameState.lastRoundEndEvent.Reason)
	assert.Equal(t, common.TeamTerrorists, gameState.lastRoundEndEvent.Winner)
}
