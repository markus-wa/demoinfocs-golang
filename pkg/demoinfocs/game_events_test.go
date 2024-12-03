package demoinfocs

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables/fake"
)

// See #90
func TestRoundEnd_LoserState_Score(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	p.disableMimicSource1GameEvents = true

	p.gameState.tState.Entity = stfake.NewEntityWithProperty("m_scoreTotal", st.PropertyValue{Any: 1})
	p.gameState.ctState.Entity = stfake.NewEntityWithProperty("m_scoreTotal", st.PropertyValue{Any: 2})
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

	p.gameEventDescs = map[int32]*msg.CMsgSource1LegacyGameEventListDescriptorT{
		1: {
			Name: proto.String("round_end"),
			Keys: []*msg.CMsgSource1LegacyGameEventListKeyT{
				{Name: proto.String("winner")},
				{Name: proto.String("message")},
				{Name: proto.String("reason")},
			},
		},
	}

	ge := new(msg.CMsgSource1LegacyGameEvent)
	ge.Eventid = proto.Int32(1)
	ge.EventName = proto.String("round_end")
	ge.Keys = []*msg.CMsgSource1LegacyGameEventKeyT{
		{ValByte: proto.Int32(2)},
		{ValString: proto.String("test")},
		{ValByte: proto.Int32(9)},
	}
	p.handleGameEvent(ge)

	assert.Equal(t, 1, eventOccurred)
}

func TestGetPlayerWeapon_NilPlayer(t *testing.T) {
	wep := getPlayerWeapon(nil, common.EqAK47)

	assert.NotNil(t, wep)
	assert.Equal(t, common.EqAK47, wep.Type)
}

func TestGetPlayerWeapon_Found(t *testing.T) {
	ak := &common.Equipment{Type: common.EqAK47}
	pl := &common.Player{
		Inventory: map[int]*common.Equipment{
			1: ak,
		},
	}

	wep := getPlayerWeapon(pl, common.EqAK47)

	assert.True(t, wep == ak)
}

func TestGetPlayerWeapon_NotFound(t *testing.T) {
	ak := &common.Equipment{Type: common.EqAK47}
	pl := &common.Player{
		Inventory: map[int]*common.Equipment{
			1: ak,
		},
	}

	wep := getPlayerWeapon(pl, common.EqM4A1)

	assert.Equal(t, common.EqM4A1, wep.Type)
}

func TestAddThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(nil, he)

	assert.Empty(t, p.gameState.thrownGrenades)
}

func TestAddThrownGrenade(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades)
	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
	assert.Equal(t, p.gameState.thrownGrenades[pl][0], he)
}

func TestGetThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	he := common.NewEquipment(common.EqHE)

	wep := p.gameEventHandler.getThrownGrenade(nil, he.Type)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}

	he := common.NewEquipment(common.EqSmoke)

	wep := p.gameEventHandler.getThrownGrenade(pl, he.Type)

	assert.Nil(t, wep)
}

func TestGetThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getThrownGrenade(pl, he.Type)

	assert.Equal(t, wep.Type, he.Type)
	assert.Equal(t, wep, he)
}

func TestDeleteThrownGrenade_NilPlayer(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	he := common.NewEquipment(common.EqHE)

	// Do nothing, we just keep sure it doesn't crash
	p.gameEventHandler.deleteThrownGrenade(nil, he.Type)
}

func TestDeleteThrownGrenade_NotFound(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, common.EqSmoke)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])
}

func TestDeleteThrownGrenade_Found(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	assert.Empty(t, p.gameState.thrownGrenades)

	p.gameEventHandler.addThrownGrenade(pl, he)

	assert.NotEmpty(t, p.gameState.thrownGrenades[pl])

	p.gameEventHandler.deleteThrownGrenade(pl, he.Type)

	assert.Empty(t, p.gameState.thrownGrenades[pl])
}

func TestGetEquipmentInstance_NotGrenade(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqAK47)

	assert.Equal(t, common.EqAK47, wep.Type)
}

func TestGetEquipmentInstance_Grenade_NotThrown(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}

	wep := p.gameEventHandler.getEquipmentInstance(pl, common.EqSmoke)

	assert.Nil(t, wep)
}

func TestGetEquipmentInstance_Grenade_Thrown(t *testing.T) {
	p := NewParser(rand.Reader).(*parser)
	pl := &common.Player{}
	he := common.NewEquipment(common.EqHE)

	p.gameEventHandler.addThrownGrenade(pl, he)
	wep := p.gameEventHandler.getEquipmentInstance(pl, he.Type)

	assert.Equal(t, he, wep)
}

func TestGetCommunityId(t *testing.T) {
	xuid, err := guidToSteamID64("STEAM_0:1:26343269")
	assert.Nil(t, err)
	assert.Equal(t, uint64(76561198012952267), xuid)
}

func TestGetCommunityId_BOT(t *testing.T) {
	xuid, err := guidToSteamID64("BOT")
	assert.Zero(t, xuid)
	assert.Nil(t, err)
}

func TestGetCommunityId_Error(t *testing.T) {
	_, err := guidToSteamID64("STEAM_0:1:abc")
	assert.NotNil(t, err)
	assert.Equal(t, "strconv.ParseUint: parsing \"abc\": invalid syntax", err.Error())
}
