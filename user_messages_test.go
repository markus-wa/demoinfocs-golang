package demoinfocs

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	assert "github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

func Test_UserMessages_ServerRankUpdate(t *testing.T) {
	rankUpdate := &msg.CCSUsrMsg_ServerRankUpdate{
		RankUpdate: []*msg.CCSUsrMsg_ServerRankUpdate_RankUpdate{{
			AccountId:  123,
			RankOld:    1,
			RankNew:    2,
			NumWins:    5,
			RankChange: 1,
		}, {
			AccountId:  456,
			RankOld:    2,
			RankNew:    3,
			NumWins:    6,
			RankChange: 2,
		}},
	}
	userMessageData, err := proto.Marshal(rankUpdate)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: int32(msg.ECstrike15UserMessages_CS_UM_ServerRankUpdate),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader))
	var evs []events.RankUpdate
	p.RegisterEventHandler(func(update events.RankUpdate) {
		evs = append(evs, update)
	})

	p.handleUserMessage(um)

	expected := []events.RankUpdate{{
		SteamID:    123,
		SteamID32:  123,
		RankOld:    1,
		RankNew:    2,
		WinCount:   5,
		RankChange: 1,
	}, {
		SteamID:    456,
		SteamID32:  456,
		RankOld:    2,
		RankNew:    3,
		WinCount:   6,
		RankChange: 2,
	}}
	assert.Equal(t, expected, evs)
}

func Test_UserMessages_SayText(t *testing.T) {
	sayText := &msg.CCSUsrMsg_SayText{
		EntIdx:      1,
		Text:        "glhf",
		Chat:        true,
		Textallchat: true,
	}
	userMessageData, err := proto.Marshal(sayText)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: int32(msg.ECstrike15UserMessages_CS_UM_SayText),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader))
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
		EntIdx:      1,
		MsgName:     "#CSGO_Coach_Join_T",
		Chat:        true,
		Textallchat: true,
		Params:      []string{"hi there", "hello"},
	}
	userMessageData, err := proto.Marshal(sayText2)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: int32(msg.ECstrike15UserMessages_CS_UM_SayText2),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader))

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
		EntIdx:      1,
		MsgName:     "Cstrike_Chat_All",
		Textallchat: true,
		Params:      []string{"The Suspect", "glhf"},
	}
	userMessageData, err := proto.Marshal(sayText2)
	assert.Nil(t, err)
	um := &msg.CSVCMsg_UserMessage{
		MsgType: int32(msg.ECstrike15UserMessages_CS_UM_SayText2),
		MsgData: userMessageData,
	}

	p := NewParser(new(DevNullReader))

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
