// DO NOT EDIT: Auto generated

package demoinfocs

import (
	"time"

	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// GameRules is an auto-generated interface for gameRules.
type GameRules interface {
	// RoundTime returns how long rounds in the current match last for (excluding freeze time).
	// May return error if cs_gamerules_data.m_iRoundTime is not set.
	RoundTime() (time.Duration, error)
	// FreezeTime returns how long freeze time lasts for in the current match (mp_freezetime).
	// May return error if mp_freezetime cannot be converted to a time duration.
	FreezeTime() (time.Duration, error)
	// BombTime returns how long freeze time lasts for in the current match (mp_freezetime).
	// May return error if mp_c4timer cannot be converted to a time duration.
	BombTime() (time.Duration, error)
	// ConVars returns a map of CVar keys and values.
	// Not all values might be set.
	// See also: https://developer.valvesoftware.com/wiki/List_of_CS:GO_Cvars.
	ConVars() map[string]string
	// Entity returns the game's CCSGameRulesProxy entity.
	Entity() st.Entity
}
