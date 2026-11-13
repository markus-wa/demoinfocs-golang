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
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// rulesOptions configures the synthetic CCSGameRulesProxy the harness binds.
type rulesOptions struct {
	hasBombTarget bool
	hasRescueZone bool
	disableMimic  bool
}

// rulesHarness exercises the GameRules family without a Parser: a bare GameState, a real dispatcher
// and a synthetic CCSGameRulesProxy entity. It captures everything the family dispatches so tests
// can assert on the events without a Parser's other wiring.
type rulesHarness struct {
	gameState  *GameState
	dispatcher *dp.Dispatcher
	entity     *st.Entity

	clearedGrenadeProjectiles int
	roundEndOfficials         []events.RoundEndOfficial
	freezetimeEvents          []events.RoundFreezetimeChanged
	gamePhaseEvents           []events.GamePhaseChanged
	teamSideSwitches          []events.TeamSideSwitch
	gameHalvesEnded           []events.GameHalfEnded
	warmupEvents              []events.IsWarmupPeriodChanged
	overtimeEvents            []events.OvertimeNumberChanged
	matchStartedEvents        []events.MatchStartedChanged
}

func newRulesHarness(t *testing.T, opts rulesOptions) *rulesHarness {
	t.Helper()

	stParser := stfake.NewParser("CCSGameRulesProxy")

	h := &rulesHarness{
		gameState:  newGameState(demoInfoProvider{}),
		dispatcher: new(dp.Dispatcher),
	}

	h.dispatcher.RegisterHandler(func(e events.RoundEndOfficial) { h.roundEndOfficials = append(h.roundEndOfficials, e) })
	h.dispatcher.RegisterHandler(func(e events.RoundFreezetimeChanged) { h.freezetimeEvents = append(h.freezetimeEvents, e) })
	h.dispatcher.RegisterHandler(func(e events.GamePhaseChanged) { h.gamePhaseEvents = append(h.gamePhaseEvents, e) })
	h.dispatcher.RegisterHandler(func(e events.TeamSideSwitch) { h.teamSideSwitches = append(h.teamSideSwitches, e) })
	h.dispatcher.RegisterHandler(func(e events.GameHalfEnded) { h.gameHalvesEnded = append(h.gameHalvesEnded, e) })
	h.dispatcher.RegisterHandler(func(e events.IsWarmupPeriodChanged) { h.warmupEvents = append(h.warmupEvents, e) })
	h.dispatcher.RegisterHandler(func(e events.OvertimeNumberChanged) { h.overtimeEvents = append(h.overtimeEvents, e) })
	h.dispatcher.RegisterHandler(func(e events.MatchStartedChanged) { h.matchStartedEvents = append(h.matchStartedEvents, e) })

	sb := newStateBinder(stateBinderDeps{
		gameState:                     h.gameState,
		eventDispatcher:               h.dispatcher,
		serverClasses:                 stParser.ServerClasses,
		disableMimicSource1GameEvents: opts.disableMimic,
		clearGrenadeProjectiles:       func() { h.clearedGrenadeProjectiles++ },
	})
	sb.bindGameRules()

	h.entity = stfake.New(stfake.Spec{
		Parser: stParser,
		Class:  "CCSGameRulesProxy",
		ID:     10,
		Props: []stfake.Prop{
			h.prop("m_iRoundTime", int32(115)),
			h.prop("m_bMapHasRescueZone", opts.hasRescueZone),
			h.prop("m_bMapHasBombTarget", opts.hasBombTarget),
			h.prop("m_bFreezePeriod", false),
			h.prop("m_gamePhase", int32(0)),
			h.prop("m_totalRoundsPlayed", int32(0)),
			h.prop("m_bWarmupPeriod", false),
			h.prop("m_bHasMatchStarted", false),
			h.prop("m_nOvertimePlaying", int32(0)),
			h.prop("m_eRoundWinReason", int32(events.RoundEndReasonStillInProgress)),
		},
	})
	stfake.Create(h.entity)

	// the create packet fires the initial property updates - clear them so tests see only their own input
	h.resetCaptured()

	return h
}

func (h *rulesHarness) prop(name string, val any) stfake.Prop {
	return stfake.P(gameRulesPrefixS2+"."+name, val)
}

func (h *rulesHarness) updateProp(name string, val any) {
	stfake.Update(h.entity, h.prop(name, val))
}

func (h *rulesHarness) resetCaptured() {
	h.clearedGrenadeProjectiles = 0
	h.roundEndOfficials = nil
	h.freezetimeEvents = nil
	h.gamePhaseEvents = nil
	h.teamSideSwitches = nil
	h.gameHalvesEnded = nil
	h.warmupEvents = nil
	h.overtimeEvents = nil
	h.matchStartedEvents = nil
}

func TestStateBinder_BindGameRules(t *testing.T) {
	h := newRulesHarness(t, rulesOptions{hasBombTarget: true})

	assert.Same(t, h.entity, h.gameState.Rules().Entity())

	// RoundTime() reads the entity property live, so an update is visible without rebinding.
	roundTime, err := h.gameState.Rules().RoundTime()
	require.NoError(t, err)
	assert.Equal(t, 115*time.Second, roundTime)

	h.updateProp("m_iRoundTime", int32(130))

	roundTime, err = h.gameState.Rules().RoundTime()
	require.NoError(t, err)
	assert.Equal(t, 130*time.Second, roundTime)

	// The create packet already consumed the first m_eRoundWinReason update, so this one is a round start.
	h.updateProp("m_totalRoundsPlayed", int32(1))
	h.updateProp("m_eRoundWinReason", int32(events.RoundEndReasonStillInProgress))

	assert.Equal(t, 1, h.clearedGrenadeProjectiles)
	require.Len(t, h.roundEndOfficials, 1)
	require.NotNil(t, h.gameState.lastRoundStartEvent)
	assert.Equal(t, 130, h.gameState.lastRoundStartEvent.TimeLimit)
	assert.Equal(t, "BOMB TARGET", h.gameState.lastRoundStartEvent.Objective)
}

func TestStateBinder_BindGameRules_Objective(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts rulesOptions
		want string
	}{
		{"bomb target", rulesOptions{hasBombTarget: true}, "BOMB TARGET"},
		{"hostage rescue", rulesOptions{hasRescueZone: true}, "HOSTAGE RESCUE"},
		{"deathmatch", rulesOptions{}, "DEATHMATCH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newRulesHarness(t, tc.opts)

			h.updateProp("m_eRoundWinReason", int32(events.RoundEndReasonStillInProgress))

			require.NotNil(t, h.gameState.lastRoundStartEvent)
			assert.Equal(t, tc.want, h.gameState.lastRoundStartEvent.Objective)
		})
	}
}

func TestStateBinder_BindGameRules_RoundEndReasons(t *testing.T) {
	cases := []struct {
		reason  events.RoundEndReason
		winner  common.Team
		message string
	}{
		{events.RoundEndReasonTargetBombed, common.TeamTerrorists, "#SFUI_Notice_Target_Bombed"},
		{events.RoundEndReasonTerroristsEscaped, common.TeamTerrorists, "#SFUI_Notice_Terrorists_Escaped"},
		{events.RoundEndReasonTerroristsWin, common.TeamTerrorists, "#SFUI_Notice_Terrorists_Win"},
		{events.RoundEndReasonHostagesNotRescued, common.TeamTerrorists, "#SFUI_Notice_Hostages_Not_Rescued"},
		{events.RoundEndReasonTerroristsPlanted, common.TeamTerrorists, "#SFUI_Notice_Terrorists_Planted"},
		{events.RoundEndReasonCTSurrender, common.TeamTerrorists, "#SFUI_Notice_CTs_Surrender"},
		{events.RoundEndReasonCTsReachedHostage, common.TeamCounterTerrorists, "#SFUI_Notice_CTs_ReachedHostage"},
		{events.RoundEndReasonCTStoppedEscape, common.TeamCounterTerrorists, "#SFUI_Notice_CTs_PreventEscape"},
		{events.RoundEndReasonTerroristsStopped, common.TeamCounterTerrorists, "#SFUI_Notice_Escaping_Terrorists_Neutralized"},
		{events.RoundEndReasonBombDefused, common.TeamCounterTerrorists, "#SFUI_Notice_Bomb_Defused"},
		{events.RoundEndReasonCTWin, common.TeamCounterTerrorists, "#SFUI_Notice_CTs_Win"},
		{events.RoundEndReasonHostagesRescued, common.TeamCounterTerrorists, "#SFUI_Notice_All_Hostages_Rescued"},
		{events.RoundEndReasonTargetSaved, common.TeamCounterTerrorists, "#SFUI_Notice_Target_Saved"},
		{events.RoundEndReasonTerroristsNotEscaped, common.TeamCounterTerrorists, "#SFUI_Notice_Terrorists_Not_Escaped"},
		{events.RoundEndReasonTerroristsSurrender, common.TeamCounterTerrorists, "#SFUI_Notice_Terrorists_Surrender"},
		{events.RoundEndReasonGameStart, common.TeamSpectators, "#SFUI_Notice_Game_Commencing"},
		{events.RoundEndReasonDraw, common.TeamSpectators, "#SFUI_Notice_Round_Draw"},
		// a reason outside the switch keeps the defaults
		{events.RoundEndReasonVIPKilled, common.TeamUnassigned, "UNKNOWN"},
	}

	h := newRulesHarness(t, rulesOptions{hasBombTarget: true})

	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			h.gameState.currentPlanter = &common.Player{}

			h.updateProp("m_eRoundWinReason", int32(tc.reason))

			end := h.gameState.lastRoundEndEvent
			require.NotNil(t, end)
			assert.Equal(t, tc.reason, end.Reason)
			assert.Equal(t, tc.winner, end.Winner)
			assert.Equal(t, tc.message, end.Message)

			if tc.winner == common.TeamUnassigned || tc.winner == common.TeamSpectators {
				assert.Nil(t, end.WinnerState)
				assert.Nil(t, end.LoserState)
			} else {
				require.NotNil(t, end.WinnerState)
				require.NotNil(t, end.LoserState)
				assert.Same(t, end.WinnerState.Opponent, end.LoserState)
			}

			assert.Nil(t, h.gameState.currentPlanter)
		})
	}
}

func TestStateBinder_BindGameRules_DisableMimic(t *testing.T) {
	h := newRulesHarness(t, rulesOptions{hasBombTarget: true, disableMimic: true})

	// round start clears the grenade projectiles but records no round-start event
	h.updateProp("m_eRoundWinReason", int32(events.RoundEndReasonStillInProgress))
	assert.Equal(t, 1, h.clearedGrenadeProjectiles)
	assert.Nil(t, h.gameState.lastRoundStartEvent)

	// round end records no round-end event
	h.gameState.currentPlanter = &common.Player{}
	h.updateProp("m_eRoundWinReason", int32(events.RoundEndReasonTargetBombed))
	assert.Nil(t, h.gameState.lastRoundEndEvent)
	assert.Nil(t, h.gameState.currentPlanter)

	// freeze time is dispatched instead of recorded
	h.updateProp("m_bFreezePeriod", true)
	require.Len(t, h.freezetimeEvents, 1)
	assert.False(t, h.freezetimeEvents[0].OldIsFreezetime)
	assert.True(t, h.freezetimeEvents[0].NewIsFreezetime)
	assert.True(t, h.gameState.isFreezetime)

	// match start is dispatched and stored
	h.updateProp("m_bHasMatchStarted", true)
	require.Len(t, h.matchStartedEvents, 1)
	assert.True(t, h.matchStartedEvents[0].NewIsStarted)
	assert.True(t, h.gameState.isMatchStarted)
}

func TestStateBinder_BindGameRules_GamePhase(t *testing.T) {
	h := newRulesHarness(t, rulesOptions{hasBombTarget: true})

	h.updateProp("m_gamePhase", int32(common.GamePhaseTeamSideSwitch))
	require.Len(t, h.gamePhaseEvents, 1)
	assert.Equal(t, common.GamePhase(0), h.gamePhaseEvents[0].OldGamePhase)
	assert.Equal(t, common.GamePhaseTeamSideSwitch, h.gamePhaseEvents[0].NewGamePhase)
	assert.Len(t, h.teamSideSwitches, 1)

	h.updateProp("m_gamePhase", int32(common.GamePhaseGameHalfEnded))
	require.Len(t, h.gamePhaseEvents, 2)
	assert.Equal(t, common.GamePhaseTeamSideSwitch, h.gamePhaseEvents[1].OldGamePhase)
	assert.Equal(t, common.GamePhaseGameHalfEnded, h.gamePhaseEvents[1].NewGamePhase)
	assert.Len(t, h.gameHalvesEnded, 1)
}

func TestStateBinder_BindGameRules_FreezeWarmupOvertime(t *testing.T) {
	h := newRulesHarness(t, rulesOptions{hasBombTarget: true})

	h.updateProp("m_bFreezePeriod", true)
	require.NotNil(t, h.gameState.lastFreezeTimeChangedEvent)
	assert.False(t, h.gameState.lastFreezeTimeChangedEvent.OldIsFreezetime)
	assert.True(t, h.gameState.lastFreezeTimeChangedEvent.NewIsFreezetime)
	assert.Empty(t, h.freezetimeEvents, "without mimic the event is recorded, not dispatched")
	assert.True(t, h.gameState.isFreezetime)

	h.updateProp("m_bWarmupPeriod", true)
	require.Len(t, h.warmupEvents, 1)
	assert.False(t, h.warmupEvents[0].OldIsWarmupPeriod)
	assert.True(t, h.warmupEvents[0].NewIsWarmupPeriod)
	assert.True(t, h.gameState.isWarmupPeriod)

	h.updateProp("m_nOvertimePlaying", int32(2))
	require.Len(t, h.overtimeEvents, 1)
	assert.Equal(t, 0, h.overtimeEvents[0].OldCount)
	assert.Equal(t, 2, h.overtimeEvents[0].NewCount)
	assert.Equal(t, 2, h.gameState.overtimeCount)
}

func TestStateBinder_BindGameRules_MatchStarted(t *testing.T) {
	h := newRulesHarness(t, rulesOptions{hasBombTarget: true})

	// m_eRoundWinReason is still "in progress" (as at create), so starting the match starts the first round
	h.updateProp("m_bHasMatchStarted", true)

	require.NotNil(t, h.gameState.lastMatchStartedChangedEvent)
	assert.False(t, h.gameState.lastMatchStartedChangedEvent.OldIsStarted)
	assert.True(t, h.gameState.lastMatchStartedChangedEvent.NewIsStarted)
	assert.Empty(t, h.matchStartedEvents, "without mimic the event is recorded, not dispatched")

	require.NotNil(t, h.gameState.lastRoundStartEvent)
	assert.Equal(t, 1, h.clearedGrenadeProjectiles)
}
