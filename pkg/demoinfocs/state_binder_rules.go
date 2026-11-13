package demoinfocs

import (
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// bindGameRules binds the CCSGameRulesProxy entity: round time, freeze time, game phase, warmup,
// overtime, match-start and the round-start / round-end detection.
//
// It is the only writer of the round-boundary events on the game state.
//
//nolint:funlen,gocognit,gocyclo
func (b stateBinder) bindGameRules() {
	gameRules := b.deps.serverClasses().FindByName("CCSGameRulesProxy")
	stOnEntityCreated(gameRules, func(entity *st.Entity) {
		grPrefix := func(s string) string {
			return fmt.Sprintf("%s.%s", gameRulesPrefixS2, s)
		}

		b.deps.gameState.rules.entity = entity

		roundTime := entity.PropertyValueMust(grPrefix("m_iRoundTime")).Int()
		hasRescueZone := entity.PropertyValueMust(grPrefix("m_bMapHasRescueZone")).BoolVal()
		hasBombTarget := entity.PropertyValueMust(grPrefix("m_bMapHasBombTarget")).BoolVal()

		dispatchRoundStart := func() {
			b.deps.clearGrenadeProjectiles()

			if b.deps.disableMimicSource1GameEvents {
				return
			}

			if b.deps.gameState.TotalRoundsPlayed() > 0 {
				b.deps.eventDispatcher.Dispatch(events.RoundEndOfficial{})
			}

			var objective string

			switch {
			case hasBombTarget:
				objective = "BOMB TARGET"
			case hasRescueZone:
				objective = "HOSTAGE RESCUE"
			default:
				objective = "DEATHMATCH"
			}

			b.deps.gameState.lastRoundStartEvent = &events.RoundStart{
				TimeLimit: roundTime,
				FragLimit: 0, // Always 0, seems hardcoded in the game
				Objective: objective,
			}
		}

		stOnUpdate(entity.Property(grPrefix("m_iRoundTime")), func(val st.PropertyValue) {
			roundTime = val.Int()
		})

		stOnUpdate(entity.Property(grPrefix("m_bFreezePeriod")), func(val st.PropertyValue) {
			newIsFreezetime := val.BoolVal()
			freezetimeEvent := events.RoundFreezetimeChanged{
				OldIsFreezetime: b.deps.gameState.isFreezetime,
				NewIsFreezetime: newIsFreezetime,
			}

			if b.deps.disableMimicSource1GameEvents {
				b.deps.eventDispatcher.Dispatch(freezetimeEvent)
			} else {
				b.deps.gameState.lastFreezeTimeChangedEvent = &freezetimeEvent
			}

			b.deps.gameState.isFreezetime = newIsFreezetime
		})

		stOnUpdate(entity.Property(grPrefix("m_gamePhase")), func(val st.PropertyValue) {
			oldGamePhase := b.deps.gameState.gamePhase
			b.deps.gameState.gamePhase = common.GamePhase(val.Int())

			b.deps.eventDispatcher.Dispatch(events.GamePhaseChanged{
				OldGamePhase: oldGamePhase,
				NewGamePhase: b.deps.gameState.gamePhase,
			})

			switch b.deps.gameState.gamePhase { //nolint:exhaustive
			case common.GamePhaseTeamSideSwitch:
				b.deps.eventDispatcher.Dispatch(events.TeamSideSwitch{})
			case common.GamePhaseGameHalfEnded:
				b.deps.eventDispatcher.Dispatch(events.GameHalfEnded{})
			}
		})

		stBindProperty(entity, grPrefix("m_totalRoundsPlayed"), &b.deps.gameState.totalRoundsPlayed, st.ValTypeInt)
		stOnUpdate(entity.Property(grPrefix("m_bWarmupPeriod")), func(val st.PropertyValue) {
			oldIsWarmupPeriod := b.deps.gameState.isWarmupPeriod
			b.deps.gameState.isWarmupPeriod = val.BoolVal()

			b.deps.eventDispatcher.Dispatch(events.IsWarmupPeriodChanged{
				OldIsWarmupPeriod: oldIsWarmupPeriod,
				NewIsWarmupPeriod: b.deps.gameState.isWarmupPeriod,
			})
		})

		stOnUpdate(entity.Property(grPrefix("m_bHasMatchStarted")), func(val st.PropertyValue) {
			oldMatchStarted := b.deps.gameState.isMatchStarted
			newMatchStarted := val.BoolVal()

			event := events.MatchStartedChanged{
				OldIsStarted: oldMatchStarted,
				NewIsStarted: newMatchStarted,
			}
			if !b.deps.disableMimicSource1GameEvents {
				b.deps.gameState.lastMatchStartedChangedEvent = &event
				// First round start event detection, we can't detect it by listening for a m_eRoundWinReason prop update
				// because there is no update triggered when the first round starts as the prop value is already 0.
				if newMatchStarted {
					winRoundReason := events.RoundEndReason(entity.PropertyValueMust(grPrefix("m_eRoundWinReason")).Int())
					if winRoundReason == events.RoundEndReasonStillInProgress {
						dispatchRoundStart()
					}
				}
			} else {
				b.deps.gameState.isMatchStarted = newMatchStarted
				b.deps.eventDispatcher.Dispatch(event)
			}
		})

		// Incremented at the beginning of a new overtime.
		stOnUpdate(entity.Property(grPrefix("m_nOvertimePlaying")), func(val st.PropertyValue) {
			overtimeCount := val.Int()
			b.deps.eventDispatcher.Dispatch(events.OvertimeNumberChanged{
				OldCount: b.deps.gameState.overtimeCount,
				NewCount: overtimeCount,
			})
			b.deps.gameState.overtimeCount = overtimeCount
		})

		firstUpdateOccurred := false
		// Updated when a round ends or starts.
		// The value 0 means there is no result yet and the round is in progress.
		stOnUpdate(entity.Property(grPrefix("m_eRoundWinReason")), func(val st.PropertyValue) {
			// Ignore the first update that contains initial CCSGameRulesProxy class values.
			if !firstUpdateOccurred {
				firstUpdateOccurred = true
				return
			}

			reason := events.RoundEndReason(val.Int())
			if reason == events.RoundEndReasonStillInProgress {
				dispatchRoundStart()
				return
			}

			message := "UNKNOWN"

			var winner = common.TeamUnassigned

			switch reason { //nolint:exhaustive
			case events.RoundEndReasonTargetBombed:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Target_Bombed"
			case events.RoundEndReasonTerroristsEscaped:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Terrorists_Escaped"
			case events.RoundEndReasonTerroristsWin:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Terrorists_Win"
			case events.RoundEndReasonHostagesNotRescued:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Hostages_Not_Rescued"
			case events.RoundEndReasonTerroristsPlanted:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Terrorists_Planted"
			case events.RoundEndReasonCTSurrender:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_CTs_Surrender"
			case events.RoundEndReasonCTsReachedHostage:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_CTs_ReachedHostage"
			case events.RoundEndReasonCTStoppedEscape:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_CTs_PreventEscape"
			case events.RoundEndReasonTerroristsStopped:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Escaping_Terrorists_Neutralized"
			case events.RoundEndReasonBombDefused:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Bomb_Defused"
			case events.RoundEndReasonCTWin:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_CTs_Win"
			case events.RoundEndReasonHostagesRescued:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_All_Hostages_Rescued"
			case events.RoundEndReasonTargetSaved:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Target_Saved"
			case events.RoundEndReasonTerroristsNotEscaped:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Terrorists_Not_Escaped"
			case events.RoundEndReasonTerroristsSurrender:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Terrorists_Surrender"
			case events.RoundEndReasonGameStart:
				winner = common.TeamSpectators
				message = "#SFUI_Notice_Game_Commencing"
			case events.RoundEndReasonDraw:
				winner = common.TeamSpectators
				message = "#SFUI_Notice_Round_Draw"
			}

			var (
				winnerState *common.TeamState
				loserState  *common.TeamState
			)

			if winner != common.TeamUnassigned && winner != common.TeamSpectators {
				winnerState = b.deps.gameState.Team(winner)
				loserState = winnerState.Opponent
			}

			if !b.deps.disableMimicSource1GameEvents {
				b.deps.gameState.lastRoundEndEvent = &events.RoundEnd{
					Reason:      reason,
					Message:     message,
					Winner:      winner,
					WinnerState: winnerState,
					LoserState:  loserState,
				}
			}

			b.deps.gameState.currentPlanter = nil
		})

		// TODO: future fields to use
		// "m_bGameRestart"
		// "m_MatchDevice"
		// "m_bHasMatchStarted"
		// "m_numBestOfMaps"
		// "m_fWarmupPeriodEnd"
		// "m_timeUntilNextPhaseStarts"

		// TODO: timeout data
		// "m_bTerroristTimeOutActive"
		// "m_bCTTimeOutActive"
		// "m_flTerroristTimeOutRemaining"
		// "m_flCTTimeOutRemaining"
		// "m_nTerroristTimeOuts"
		// "m_nCTTimeOuts"
	})
}
