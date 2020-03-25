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

func TestAddThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader)
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(nil, he)

	assert.Empty(t, p.gameState.thrownGrenades)
}

func TestAddThrownGrenade(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades)
	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
	assert.Equal(t, p.gameState.thrownGrenades[pl][0], he)
}

func TestGetThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader)
	he := common.NewEquipment(common.EqHE)

	wep := p.gameEventHandler.getThrownGrenade(nil, he.Weapon)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}

	he := common.NewEquipment(common.EqSmoke)

	wep := p.gameEventHandler.getThrownGrenade(pl, he.Weapon)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getThrownGrenade(pl, he.Weapon)

	assert.Equal(t, wep.Weapon, he.Weapon)
	assert.Equal(t, wep, he)
}

func TestDeleteThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader)
	he := common.NewEquipment(common.EqHE)

	// Do nothing, we just keep sure it doesn't crash
	p.gameEventHandler.deleteThrownGrenade(nil, he.Weapon)
}

func TestDeleteThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, common.EqSmoke)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
}

func TestDeleteThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, he.Weapon)

	assert.Empty(t, p.gameState.thrownGrenades[pl])
}

func TestGetEquipmentInstance_NotGrenade(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqAK47)

	assert.Equal(t, common.EqAK47, wep.Weapon)
}

func TestGetEquipmentInstance_Grenade_NotThrown(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqSmoke)

	assert.Nil(t, wep)
}

func TestGetEquipmentInstance_Grenade_Thrown(t *testing.T) {
	p := NewParser(rand.Reader)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getEquipmentInstance(pl, he.Weapon)

	assert.Equal(t, he, wep)
}

func TestGetCommunityId(t *testing.T) {
	xuid, err := getCommunityID("abcdefgh1:3")
	assert.Nil(t, err)
	expected := int64(76561197960265728 + 6 + 1)
	assert.Equal(t, expected, xuid)
}

func TestGetCommunityId_BOT(t *testing.T) {
	xuid, err := getCommunityID("BOT")
	assert.Zero(t, xuid)
	assert.Nil(t, err)
}

func TestGetCommunityId_Errors(t *testing.T) {
	_, err := getCommunityID("12345678a90123")
	assert.NotNil(t, err)
	assert.Equal(t, "strconv.ParseInt: parsing \"a\": invalid syntax", err.Error())

	_, err = getCommunityID("1234567890abc")
	assert.NotNil(t, err)
	assert.Equal(t, "strconv.ParseInt: parsing \"abc\": invalid syntax", err.Error())
}
