package common

// GamePhase represents a phase in CS:GO
type GamePhase int

// The following game rules have been found at https://github.com/pmrowla/hl2sdk-csgo/blob/master/game/shared/teamplayroundbased_gamerules.h#L37.
// It seems that the naming used in the source engine is _not_ what is used in-game.
// The original names of the enum fields are added as comments to each field.
const (
	// GamePhaseInit is the default value of the game phase
	GamePhaseInit GamePhase = 0 // enum name: Init

	// GamePhasePregame
	GamePhasePregame GamePhase = 1 // enum name: Pregame

	// GamePhaseStartGamePhase is set whenever a new game phase is started.
	// A game phase can be the normal match, i.e. first to 16 rounds, or an overtime match,
	// i.e. first to 4 rounds. It is set for _all_ overtimes played, i.e. for a match
	// with 3 overtimes,  GamePhaseStartGamePhase is set 1 time for the normal
	// match and 1 time for each overtime played, for a total of 4 times.
	GamePhaseStartGamePhase GamePhase = 2 // enum name: StartGame

	// GamePhaseTeamSideSwitch is set whenever a team side switch happened,
	// i.e. both during normal game and overtime play.
	GamePhaseTeamSideSwitch GamePhase = 3 // enum name: PreRound

	// GamePhaseGameHalfEnded is set whenever a game phase has ended.
	// A game phase can be the normal match, i.e. first to 16 rounds, or an overtime match,
	// i.e. first to 4 rounds. It is set once for all overtimes played, i.e. for a match
	// with 3 overtimes,  GamePhaseGameHalfEnded is set 1 time for the normal
	// match and 1 time for each overtime played, for a total of 4 times.
	GamePhaseGameHalfEnded GamePhase = 4 // enum name: TeamWin

	// GamePhaseGameEnded is set when the full game has ended.
	// This existence of this event is not reliable: it has been observed that a demo ends
	// before this event is set
	GamePhaseGameEnded GamePhase = 5 // enum name: Restart

	// GamePhaseStaleMate has not been observed so far
	GamePhaseStaleMate GamePhase = 6 // enum name: StaleMate

	// GamePhaseGameOver has not been observed so far
	GamePhaseGameOver GamePhase = 7 // enum name: GameOver
)

// gamePhaseToString maps a GamePhase to a user friendly string
var gamePhaseToString = map[GamePhase]string{
	GamePhaseInit:           "Init",
	GamePhasePregame:        "Pregame",
	GamePhaseStartGamePhase: "Start game phase",
	GamePhaseTeamSideSwitch: "Team side switch",
	GamePhaseGameHalfEnded:  "Game half ended",
	GamePhaseGameEnded:      "Game ended",
	GamePhaseStaleMate:      "StaleMate",
	GamePhaseGameOver:       "GameOver",
}

func (r GamePhase) String() string {
	return gamePhaseToString[r]
}
