package demoinfocs

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

// See #90
func TestRoundEnd_LoserState_Score(t *testing.T) {
	p := NewParser(rand.Reader)

	p.gameState.tState.Score = 1
	p.gameState.ctState.Score = 2

	eventOccurred := 0
	p.RegisterEventHandler(func(e events.RoundEnd) {
		eventOccurred++
		assert.Equal(t, e, events.RoundEnd{
			Winner:      common.TeamTerrorists,
			WinnerState: p.GameState().TeamTerrorists(),
			LoserState:  p.GameState().TeamCounterTerrorists(),
			Message:     "test",
			Reason:      events.RoundEndReasonTerroristsWin,
		})
	})

	p.gameEventDescs = map[int32]*msg.CSVCMsg_GameEventListDescriptorT{
		1: {
			Name: "round_end",
			Keys: []*msg.CSVCMsg_GameEventListKeyT{
				{Name: "winner"},
				{Name: "message"},
				{Name: "reason"},
			},
		},
	}

	ge := new(msg.CSVCMsg_GameEvent)
	ge.Eventid = 1
	ge.EventName = "round_end"
	ge.Keys = []*msg.CSVCMsg_GameEventKeyT{
		{ValByte: 2},
		{ValString: "test"},
		{ValByte: 9},
	}
	p.handleGameEvent(ge)

	assert.Equal(t, 1, eventOccurred)
}

func TestGetPlayerWeapon_NilPlayer(t *testing.T) {
	wep := getPlayerWeapon(nil, common.EqAK47)

	assert.NotNil(t, wep)
	assert.Equal(t, common.EqAK47, wep.Weapon)
}

func TestGetPlayerWeapon_Found(t *testing.T) {
	ak := &common.Equipment{Weapon: common.EqAK47}
	pl := &common.Player{
		RawWeapons: map[int]*common.Equipment{
			1: ak,
		},
	}

	wep := getPlayerWeapon(pl, common.EqAK47)

	assert.True(t, wep == ak)
}

func TestGetPlayerWeapon_NotFound(t *testing.T) {
	ak := &common.Equipment{Weapon: common.EqAK47}
	pl := &common.Player{
		RawWeapons: map[int]*common.Equipment{
			1: ak,
		},
	}

	wep := getPlayerWeapon(pl, common.EqM4A1)

	assert.Equal(t, common.EqM4A1, wep.Weapon)
}
