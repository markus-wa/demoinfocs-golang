package fuzzy

import (
	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	dp "github.com/markus-wa/godispatch"
)

// TeamSwitchEvent signals that the teams have switched.
//
// See also: ValveMatchmakingTeamSwitchEmitter
type TeamSwitchEvent struct{}

// ValveMatchmakingTeamSwitchEmitter emits a TeamSwitchEvent for Valve MM demos.
// Sadly this WON'T work for Major games as it currently doesn't account for overtime.
//
// This is a beta feature and may be changed or replaced without notice.
//
// See also: github.com/markus-wa/demoinfocs-golang/ParserConfig.AdditionalEventEmitters
type ValveMatchmakingTeamSwitchEmitter struct {
	parser              dem.Parser
	dispatch            func(interface{})
	currentHandlerID    dp.HandlerIdentifier
	tScoreBeforeSwitch  int
	ctScoreBeforeSwitch int
}

// Register registers the emitter on the parser. It should only be used by the parser.
func (em *ValveMatchmakingTeamSwitchEmitter) Register(parser dem.Parser, dispatch func(interface{})) {
	em.parser = parser
	em.dispatch = dispatch

	em.currentHandlerID = parser.RegisterEventHandler(em.handleLastRoundHalf)
}

// Get to the last round of the first half
func (em *ValveMatchmakingTeamSwitchEmitter) handleLastRoundHalf(events.AnnouncementLastRoundHalf) {
	em.parser.UnregisterEventHandler(em.currentHandlerID)
	em.currentHandlerID = em.parser.RegisterEventHandler(em.handleRoundEnded)
}

// Get scores before switch
func (em *ValveMatchmakingTeamSwitchEmitter) handleRoundEnded(ev events.RoundEnd) {
	em.tScoreBeforeSwitch = em.parser.GameState().TeamTerrorists().Score
	em.ctScoreBeforeSwitch = em.parser.GameState().TeamCounterTerrorists().Score

	// Score hasn't been updated yet because CS:GO demos are weird
	switch ev.Winner {
	case common.TeamTerrorists:
		em.tScoreBeforeSwitch++
	case common.TeamCounterTerrorists:
		em.ctScoreBeforeSwitch++
	}

	em.parser.UnregisterEventHandler(em.currentHandlerID)
	em.currentHandlerID = em.parser.RegisterEventHandler(em.handleRoundStarted)
}

// Find first round of second half
func (em *ValveMatchmakingTeamSwitchEmitter) handleRoundStarted(events.RoundStart) {
	em.parser.UnregisterEventHandler(em.currentHandlerID)
	em.currentHandlerID = em.parser.RegisterEventHandler(em.handleTickDone)
}

// Wait for score to update - this isn't (necessarily?) the case after RoundStart
func (em *ValveMatchmakingTeamSwitchEmitter) handleTickDone(events.TickDone) {
	tScoreOk := em.parser.GameState().TeamTerrorists().Score == em.ctScoreBeforeSwitch
	ctScoreOk := em.parser.GameState().TeamCounterTerrorists().Score == em.tScoreBeforeSwitch
	if tScoreOk && ctScoreOk {
		em.dispatch(TeamSwitchEvent{})

		em.parser.UnregisterEventHandler(em.currentHandlerID)
	}
}
