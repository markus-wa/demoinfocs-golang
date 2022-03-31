package demoinfocs

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
	proto "google.golang.org/protobuf/proto"

	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg"
)

func Test_UserMessages_ServerRankUpdate(t *testing.T) {
	rankUpdate := &msg.CCSUsrMsg_ServerRankUpdate{
		RankUpdate: []*msg.CCSUsrMsg_ServerRankUpdate_RankUpdate{{
			AccountId:  proto.Int32(123),
			RankOld:    proto.Int32(1),
			RankNew:    proto.Int32(2),
			NumWins:    proto.Int32(5),
			RankChange: proto.Float32(1),
		}, {
			AccountId:  proto.Int32(456),
			RankOld:    proto.Int32(2),
			RankNew:    proto.Int32(3),
			NumWins:    proto.Int32(6),
			RankChange: proto.Float32(2),
		}},
	}
	userMessageData, err := proto.Marshal(rankUpdate)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: proto.Int32(int32(msg.ECstrike15UserMessages_CS_UM_ServerRankUpdate)),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader)).(*parser)

	plA := newPlayer()
	plB := newPlayer()
	p.gameState.playersBySteamID32[123] = plA
	p.gameState.playersBySteamID32[456] = plB

	var evs []events.RankUpdate
	p.RegisterEventHandler(func(update events.RankUpdate) {
		evs = append(evs, update)
	})

	p.handleUserMessage(um)

	expected := []events.RankUpdate{{
		SteamID32:  123,
		RankOld:    1,
		RankNew:    2,
		WinCount:   5,
		RankChange: 1,
		Player:     plA,
	}, {
		SteamID32:  456,
		RankOld:    2,
		RankNew:    3,
		WinCount:   6,
		RankChange: 2,
		Player:     plB,
	}}
	assert.Equal(t, expected, evs)
}

func Test_UserMessages_SayText(t *testing.T) {
	sayText := &msg.CCSUsrMsg_SayText{
		EntIdx:      proto.Int32(1),
		Text:        proto.String("glhf"),
		Chat:        proto.Bool(true),
		Textallchat: proto.Bool(true),
	}
	userMessageData, err := proto.Marshal(sayText)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: proto.Int32(int32(msg.ECstrike15UserMessages_CS_UM_SayText)),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader)).(*parser)

	var actual events.SayText
	p.RegisterEventHandler(func(chat events.SayText) {
		actual = chat
	})

	p.handleUserMessage(um)

	expected := events.SayText{
		EntIdx:    1,
		IsChat:    true,
		Text:      "glhf",
		IsChatAll: true,
	}
	assert.Equal(t, expected, actual)
}

func Test_UserMessages_SayText2_Generic(t *testing.T) {
	sayText2 := &msg.CCSUsrMsg_SayText2{
		EntIdx:      proto.Int32(1),
		MsgName:     proto.String("#CSGO_Coach_Join_T"),
		Chat:        proto.Bool(true),
		Textallchat: proto.Bool(true),
		Params:      []string{"hi there", "hello"},
	}
	userMessageData, err := proto.Marshal(sayText2)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: proto.Int32(int32(msg.ECstrike15UserMessages_CS_UM_SayText2)),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader)).(*parser)

	chatter := &common.Player{
		Name: "The Suspect",
	}
	p.gameState.playersByEntityID[1] = chatter

	var actual events.SayText2
	p.RegisterEventHandler(func(event events.SayText2) {
		actual = event
	})

	p.handleUserMessage(um)

	expected := events.SayText2{
		EntIdx:    1,
		MsgName:   "#CSGO_Coach_Join_T",
		Params:    sayText2.Params,
		IsChat:    true,
		IsChatAll: true,
	}
	assert.Equal(t, expected, actual)
}

func Test_UserMessages_SayText2_ChatMessage(t *testing.T) {
	sayText2 := &msg.CCSUsrMsg_SayText2{
		EntIdx:      proto.Int32(1),
		MsgName:     proto.String("Cstrike_Chat_All"),
		Textallchat: proto.Bool(true),
		Params:      []string{"The Suspect", "glhf"},
	}
	userMessageData, err := proto.Marshal(sayText2)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: proto.Int32(int32(msg.ECstrike15UserMessages_CS_UM_SayText2)),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader)).(*parser)

	chatter := &common.Player{
		Name: "The Suspect",
	}
	p.gameState.playersByEntityID[1] = chatter

	var actual events.ChatMessage
	p.RegisterEventHandler(func(chat events.ChatMessage) {
		actual = chat
	})

	p.handleUserMessage(um)

	expected := events.ChatMessage{
		Sender:    chatter,
		Text:      "glhf",
		IsChatAll: true,
	}
	assert.Equal(t, expected, actual)
}
