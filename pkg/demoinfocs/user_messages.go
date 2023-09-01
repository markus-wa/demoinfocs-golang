package demoinfocs

import (
	"fmt"

	unassert "github.com/markus-wa/go-unassert"
	"google.golang.org/protobuf/proto"

	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg"
)

func (p *parser) handleUserMessage(um *msg.CSVCMsg_UserMessage) {
	handler := p.userMessageHandler.handler(msg.ECstrike15UserMessages(um.GetMsgType()))
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

func (umh userMessageHandler) dispatch(event any) {
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
	err := proto.Unmarshal(um.MsgData, st)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode SayText message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	umh.dispatch(events.SayText{
		EntIdx:    int(st.GetEntIdx()),
		IsChat:    st.GetChat(),
		IsChatAll: st.GetTextallchat(),
		Text:      st.GetText(),
	})
}

func (umh userMessageHandler) sayText2(um *msg.CSVCMsg_UserMessage) {
	st := new(msg.CCSUsrMsg_SayText2)
	err := proto.Unmarshal(um.MsgData, st)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode SayText2 message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	umh.dispatch(events.SayText2{
		EntIdx:    int(st.GetEntIdx()),
		IsChat:    st.GetChat(),
		IsChatAll: st.GetTextallchat(),
		MsgName:   st.GetMsgName(),
		Params:    st.Params,
	})

	switch st.GetMsgName() {
	case "Cstrike_Chat_All":
		fallthrough
	case "Cstrike_Chat_AllDead":
		sender := umh.gameState().playersByEntityID[int(st.GetEntIdx())]

		umh.dispatch(events.ChatMessage{
			Sender:    sender,
			Text:      st.Params[1],
			IsChatAll: st.GetTextallchat(),
		})

	case "#CSGO_Coach_Join_T": // Ignore these
	case "#CSGO_Coach_Join_CT":
	case "#Cstrike_Name_Change":
	case "Cstrike_Chat_T_Loc":
	case "Cstrike_Chat_CT_Loc":
	case "Cstrike_Chat_T_Dead":
	case "Cstrike_Chat_CT_Dead":

	default:
		errMsg := fmt.Sprintf("skipped sending ChatMessageEvent for SayText2 with unknown MsgName %q", st.GetMsgName())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}
}

func (umh userMessageHandler) rankUpdate(um *msg.CSVCMsg_UserMessage) {
	st := new(msg.CCSUsrMsg_ServerRankUpdate)
	err := proto.Unmarshal(um.MsgData, st)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode ServerRankUpdate message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	for _, v := range st.RankUpdate {
		// find player (or old instance if he has disconnected already)
		steamID32 := uint32(v.GetAccountId())

		player, ok := umh.parser.gameState.playersBySteamID32[steamID32]
		if !ok {
			errMsg := fmt.Sprintf("rank update for unknown player with SteamID32=%d", steamID32)

			umh.dispatch(events.ParserWarn{Message: errMsg})
			unassert.Error(errMsg)
		}

		umh.dispatch(events.RankUpdate{
			SteamID32:  v.GetAccountId(),
			RankOld:    int(v.GetRankOld()),
			RankNew:    int(v.GetRankNew()),
			WinCount:   int(v.GetNumWins()),
			RankChange: v.GetRankChange(),
			Player:     player,
		})
	}
}

func (umh userMessageHandler) roundImpactScoreData(um *msg.CSVCMsg_UserMessage) {
	impactData := new(msg.CCSUsrMsg_RoundImpactScoreData)
	err := proto.Unmarshal(um.MsgData, impactData)

	if err != nil {
		errMsg := fmt.Sprintf("failed to decode RoundImpactScoreData message: %s", err.Error())

		umh.dispatch(events.ParserWarn{Message: errMsg})
		unassert.Error(errMsg)
	}

	umh.dispatch(events.RoundImpactScoreData{
		RawMessage: impactData,
	})
}
