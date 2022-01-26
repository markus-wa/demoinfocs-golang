package demoinfocs

import (
	"fmt"

	unassert "github.com/markus-wa/go-unassert"

	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg"
)

func (p *parser) handleUserMessage(um *msg.CSVCMsg_UserMessage) {
	handler := p.userMessageHandler.handler(msg.ECstrike15UserMessages(um.MsgType))
	if handler != nil {
		handler(um)
	}
}

type userMessageHandler struct {
	parser           *parser
	msgTypeToHandler map[msg.ECstrike15UserMessages]userMessageHandlerFunc
}

func (umh userMessageHandler) handler(msgType msg.ECstrike15UserMessages) userMessageHandlerFunc {
	if handler, eventKnown := umh.msgTypeToHandler[msgType]; eventKnown {
		return handler
	}

	return nil
}

func (umh userMessageHandler) dispatch(event interface{}) {
	umh.parser.eventDispatcher.Dispatch(event)
}

func (umh userMessageHandler) gameState() *gameState {
	return umh.parser.gameState
}

type userMessageHandlerFunc func(*msg.CSVCMsg_UserMessage)

func newUserMessageHandler(parser *parser) userMessageHandler {
	umh := userMessageHandler{parser: parser}

	umh.msgTypeToHandler = map[msg.ECstrike15UserMessages]userMessageHandlerFunc{
		msg.ECstrike15UserMessages_CS_UM_SayText:              umh.sayText,
		msg.ECstrike15UserMessages_CS_UM_SayText2:             umh.sayText2,
		msg.ECstrike15UserMessages_CS_UM_ServerRankUpdate:     umh.rankUpdate,
		msg.ECstrike15UserMessages_CS_UM_RoundImpactScoreData: umh.roundImpactScoreData,
		// TODO: handle more user messages (if they are interesting)
		// Maybe msg.ECstrike15UserMessages_CS_UM_RadioText
	}

	return umh
}

func (umh userMessageHandler) sayText(um *msg.CSVCMsg_UserMessage) {
	st := new(msg.CCSUsrMsg_SayText)
	err := st.Unmarshal(um.MsgData)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode SayText message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	umh.dispatch(events.SayText{
		EntIdx:    int(st.EntIdx),
		IsChat:    st.Chat,
		IsChatAll: st.Textallchat,
		Text:      st.Text,
	})
}

func (umh userMessageHandler) sayText2(um *msg.CSVCMsg_UserMessage) {
	st := new(msg.CCSUsrMsg_SayText2)
	err := st.Unmarshal(um.MsgData)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode SayText2 message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	umh.dispatch(events.SayText2{
		EntIdx:    int(st.EntIdx),
		IsChat:    st.Chat,
		IsChatAll: st.Textallchat,
		MsgName:   st.MsgName,
		Params:    st.Params,
	})

	switch st.MsgName {
	case "Cstrike_Chat_All":
		fallthrough
	case "Cstrike_Chat_AllDead":
		sender := umh.gameState().playersByEntityID[int(st.EntIdx)]

		umh.dispatch(events.ChatMessage{
			Sender:    sender,
			Text:      st.Params[1],
			IsChatAll: st.Textallchat,
		})

	case "#CSGO_Coach_Join_T": // Ignore these
	case "#CSGO_Coach_Join_CT":
	case "#Cstrike_Name_Change":
	case "Cstrike_Chat_T_Loc":
	case "Cstrike_Chat_CT_Loc":
	case "Cstrike_Chat_T_Dead":
	case "Cstrike_Chat_CT_Dead":

	default:
		errMsg := fmt.Sprintf("skipped sending ChatMessageEvent for SayText2 with unknown MsgName %q", st.MsgName)

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}
}

func (umh userMessageHandler) rankUpdate(um *msg.CSVCMsg_UserMessage) {
	st := new(msg.CCSUsrMsg_ServerRankUpdate)
	err := st.Unmarshal(um.MsgData)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode ServerRankUpdate message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	for _, v := range st.RankUpdate {
		// find player (or old instance if he has disconnected already)
		steamID32 := uint32(v.AccountId)
		player, ok := umh.parser.gameState.playersBySteamID32[steamID32]
		if !ok {
			errMsg := fmt.Sprintf("rank update for unknown player with SteamID32=%d", steamID32)

			umh.dispatch(events.ParserWarn{Message: errMsg})
			unassert.Error(errMsg)
		}

		umh.dispatch(events.RankUpdate{
			SteamID32:  v.AccountId,
			RankOld:    int(v.RankOld),
			RankNew:    int(v.RankNew),
			WinCount:   int(v.NumWins),
			RankChange: v.RankChange,
			Player:     player,
		})
	}
}

func (umh userMessageHandler) roundImpactScoreData(um *msg.CSVCMsg_UserMessage) {
	impactData := new(msg.CCSUsrMsg_RoundImpactScoreData)
	err := impactData.Unmarshal(um.MsgData)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode RoundImpactScoreData message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	umh.dispatch(events.RoundImpactScoreData{
		RawMessage: impactData,
	})
}
