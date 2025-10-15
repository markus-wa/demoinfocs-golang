package demoinfocs

import (
	"bytes"
	"embed"
	"fmt"
	"slices"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v5/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

func (p *parser) handleSendTables(msg *msg.CDemoSendTables) {
	err := p.stParser.ParsePacket(msg.Data)
	if err != nil {
		panic(errors.Wrap(err, "failed to unmarshal flattened serializer"))
	}
}

func (p *parser) handleClassInfo(msg *msg.CDemoClassInfo) {
	err := p.stParser.OnDemoClassInfo(msg)
	if err != nil {
		panic(err)
	}

	debugAllServerClasses(p.ServerClasses())

	p.bindEntities()

	p.eventDispatcher.Dispatch(events.DataTablesParsed{})
}

var netMsgCreators = map[msg.NET_Messages]NetMessageCreator{
	msg.NET_Messages_net_NOP:                        func() proto.Message { return &msg.CNETMsg_NOP{} },
	msg.NET_Messages_net_SplitScreenUser:            func() proto.Message { return &msg.CNETMsg_SplitScreenUser{} },
	msg.NET_Messages_net_Tick:                       func() proto.Message { return &msg.CNETMsg_Tick{} },
	msg.NET_Messages_net_StringCmd:                  func() proto.Message { return &msg.CNETMsg_StringCmd{} },
	msg.NET_Messages_net_SetConVar:                  func() proto.Message { return &msg.CNETMsg_SetConVar{} },
	msg.NET_Messages_net_SignonState:                func() proto.Message { return &msg.CNETMsg_SignonState{} },
	msg.NET_Messages_net_SpawnGroup_Load:            func() proto.Message { return &msg.CNETMsg_SpawnGroup_Load{} },
	msg.NET_Messages_net_SpawnGroup_ManifestUpdate:  func() proto.Message { return &msg.CNETMsg_SpawnGroup_ManifestUpdate{} },
	msg.NET_Messages_net_SpawnGroup_SetCreationTick: func() proto.Message { return &msg.CNETMsg_SpawnGroup_SetCreationTick{} },
	msg.NET_Messages_net_SpawnGroup_Unload:          func() proto.Message { return &msg.CNETMsg_SpawnGroup_Unload{} },
	msg.NET_Messages_net_SpawnGroup_LoadCompleted:   func() proto.Message { return &msg.CNETMsg_SpawnGroup_LoadCompleted{} },
	msg.NET_Messages_net_DebugOverlay:               func() proto.Message { return &msg.CNETMsg_DebugOverlay{} },
}

var svcMsgCreators = map[msg.SVC_Messages]NetMessageCreator{
	msg.SVC_Messages_svc_ServerInfo:              func() proto.Message { return &msg.CSVCMsg_ServerInfo{} },
	msg.SVC_Messages_svc_FlattenedSerializer:     func() proto.Message { return &msg.CSVCMsg_FlattenedSerializer{} },
	msg.SVC_Messages_svc_ClassInfo:               func() proto.Message { return &msg.CSVCMsg_ClassInfo{} },
	msg.SVC_Messages_svc_SetPause:                func() proto.Message { return &msg.CSVCMsg_SetPause{} },
	msg.SVC_Messages_svc_CreateStringTable:       func() proto.Message { return &msg.CSVCMsg_CreateStringTable{} },
	msg.SVC_Messages_svc_UpdateStringTable:       func() proto.Message { return &msg.CSVCMsg_UpdateStringTable{} },
	msg.SVC_Messages_svc_VoiceInit:               func() proto.Message { return &msg.CSVCMsg_VoiceInit{} },
	msg.SVC_Messages_svc_VoiceData:               func() proto.Message { return &msg.CSVCMsg_VoiceData{} },
	msg.SVC_Messages_svc_Print:                   func() proto.Message { return &msg.CSVCMsg_Print{} },
	msg.SVC_Messages_svc_Sounds:                  func() proto.Message { return &msg.CSVCMsg_Sounds{} },
	msg.SVC_Messages_svc_SetView:                 func() proto.Message { return &msg.CSVCMsg_SetView{} },
	msg.SVC_Messages_svc_ClearAllStringTables:    func() proto.Message { return &msg.CSVCMsg_ClearAllStringTables{} },
	msg.SVC_Messages_svc_CmdKeyValues:            func() proto.Message { return &msg.CSVCMsg_CmdKeyValues{} },
	msg.SVC_Messages_svc_BSPDecal:                func() proto.Message { return &msg.CSVCMsg_BSPDecal{} },
	msg.SVC_Messages_svc_SplitScreen:             func() proto.Message { return &msg.CSVCMsg_SplitScreen{} },
	msg.SVC_Messages_svc_PacketEntities:          func() proto.Message { return &msg.CSVCMsg_PacketEntities{} },
	msg.SVC_Messages_svc_Prefetch:                func() proto.Message { return &msg.CSVCMsg_Prefetch{} },
	msg.SVC_Messages_svc_Menu:                    func() proto.Message { return &msg.CSVCMsg_Menu{} },
	msg.SVC_Messages_svc_GetCvarValue:            func() proto.Message { return &msg.CSVCMsg_GetCvarValue{} },
	msg.SVC_Messages_svc_StopSound:               func() proto.Message { return &msg.CSVCMsg_StopSound{} },
	msg.SVC_Messages_svc_PeerList:                func() proto.Message { return &msg.CSVCMsg_PeerList{} },
	msg.SVC_Messages_svc_PacketReliable:          func() proto.Message { return &msg.CSVCMsg_PacketReliable{} },
	msg.SVC_Messages_svc_HLTVStatus:              func() proto.Message { return &msg.CSVCMsg_HLTVStatus{} },
	msg.SVC_Messages_svc_ServerSteamID:           func() proto.Message { return &msg.CSVCMsg_ServerSteamID{} },
	msg.SVC_Messages_svc_FullFrameSplit:          func() proto.Message { return &msg.CSVCMsg_FullFrameSplit{} },
	msg.SVC_Messages_svc_RconServerDetails:       func() proto.Message { return &msg.CSVCMsg_RconServerDetails{} },
	msg.SVC_Messages_svc_UserMessage:             func() proto.Message { return &msg.CSVCMsg_UserMessage{} },
	msg.SVC_Messages_svc_Broadcast_Command:       func() proto.Message { return &msg.CSVCMsg_Broadcast_Command{} },
	msg.SVC_Messages_svc_HltvFixupOperatorStatus: func() proto.Message { return &msg.CSVCMsg_HltvFixupOperatorStatus{} },
	msg.SVC_Messages_svc_UserCmds:                func() proto.Message { return &msg.CSVCMsg_UserCommands{} },
}

var usrMsgCreators = map[msg.EBaseUserMessages]NetMessageCreator{
	msg.EBaseUserMessages_UM_AchievementEvent:        func() proto.Message { return &msg.CUserMessageAchievementEvent{} },
	msg.EBaseUserMessages_UM_CloseCaption:            func() proto.Message { return &msg.CUserMessageCloseCaption{} },
	msg.EBaseUserMessages_UM_CloseCaptionDirect:      func() proto.Message { return &msg.CUserMessageCloseCaptionDirect{} },
	msg.EBaseUserMessages_UM_CurrentTimescale:        func() proto.Message { return &msg.CUserMessageCurrentTimescale{} },
	msg.EBaseUserMessages_UM_DesiredTimescale:        func() proto.Message { return &msg.CUserMessageDesiredTimescale{} },
	msg.EBaseUserMessages_UM_Fade:                    func() proto.Message { return &msg.CUserMessageFade{} },
	msg.EBaseUserMessages_UM_GameTitle:               func() proto.Message { return &msg.CUserMessageGameTitle{} },
	msg.EBaseUserMessages_UM_HudMsg:                  func() proto.Message { return &msg.CUserMessageHudMsg{} },
	msg.EBaseUserMessages_UM_HudText:                 func() proto.Message { return &msg.CUserMessageHudText{} },
	msg.EBaseUserMessages_UM_ColoredText:             func() proto.Message { return &msg.CUserMessageColoredText{} },
	msg.EBaseUserMessages_UM_RequestState:            func() proto.Message { return &msg.CUserMessageRequestState{} },
	msg.EBaseUserMessages_UM_ResetHUD:                func() proto.Message { return &msg.CUserMessageResetHUD{} },
	msg.EBaseUserMessages_UM_Rumble:                  func() proto.Message { return &msg.CUserMessageRumble{} },
	msg.EBaseUserMessages_UM_SayText:                 func() proto.Message { return &msg.CUserMessageSayText{} },
	msg.EBaseUserMessages_UM_SayText2:                func() proto.Message { return &msg.CUserMessageSayText2{} },
	msg.EBaseUserMessages_UM_SayTextChannel:          func() proto.Message { return &msg.CUserMessageSayTextChannel{} },
	msg.EBaseUserMessages_UM_Shake:                   func() proto.Message { return &msg.CUserMessageShake{} },
	msg.EBaseUserMessages_UM_ShakeDir:                func() proto.Message { return &msg.CUserMessageShakeDir{} },
	msg.EBaseUserMessages_UM_TextMsg:                 func() proto.Message { return &msg.CUserMessageTextMsg{} },
	msg.EBaseUserMessages_UM_ScreenTilt:              func() proto.Message { return &msg.CUserMessageScreenTilt{} },
	msg.EBaseUserMessages_UM_VoiceMask:               func() proto.Message { return &msg.CUserMessageVoiceMask{} },
	msg.EBaseUserMessages_UM_SendAudio:               func() proto.Message { return &msg.CUserMessageSendAudio{} },
	msg.EBaseUserMessages_UM_ItemPickup:              func() proto.Message { return &msg.CUserMessageItemPickup{} },
	msg.EBaseUserMessages_UM_AmmoDenied:              func() proto.Message { return &msg.CUserMessageAmmoDenied{} },
	msg.EBaseUserMessages_UM_ShowMenu:                func() proto.Message { return &msg.CUserMessageShowMenu{} },
	msg.EBaseUserMessages_UM_CreditsMsg:              func() proto.Message { return &msg.CUserMessageCreditsMsg{} },
	msg.EBaseUserMessages_UM_CloseCaptionPlaceholder: func() proto.Message { return &msg.CUserMessageCloseCaptionPlaceholder{} },
	msg.EBaseUserMessages_UM_CameraTransition:        func() proto.Message { return &msg.CUserMessageCameraTransition{} },
	msg.EBaseUserMessages_UM_AudioParameter:          func() proto.Message { return &msg.CUserMessageAudioParameter{} },
	msg.EBaseUserMessages_UM_ParticleManager:         func() proto.Message { return &msg.CUserMsg_ParticleManager{} },
	msg.EBaseUserMessages_UM_HudError:                func() proto.Message { return &msg.CUserMsg_HudError{} },
	msg.EBaseUserMessages_UM_CustomGameEvent:         func() proto.Message { return &msg.CUserMsg_CustomGameEvent{} },
	msg.EBaseUserMessages_UM_AnimGraphUpdate:         func() proto.Message { return &msg.CUserMessageAnimStateGraphState{} },
	msg.EBaseUserMessages_UM_HapticsManagerPulse:     func() proto.Message { return &msg.CUserMessageHapticsManagerPulse{} },
	msg.EBaseUserMessages_UM_HapticsManagerEffect:    func() proto.Message { return &msg.CUserMessageHapticsManagerEffect{} },
	msg.EBaseUserMessages_UM_UpdateCssClasses:        func() proto.Message { return &msg.CUserMessageUpdateCssClasses{} },
	msg.EBaseUserMessages_UM_ServerFrameTime:         func() proto.Message { return &msg.CUserMessageServerFrameTime{} },
	msg.EBaseUserMessages_UM_LagCompensationError:    func() proto.Message { return &msg.CUserMessageLagCompensationError{} },
	msg.EBaseUserMessages_UM_RequestDllStatus:        func() proto.Message { return &msg.CUserMessageRequestDllStatus{} },
	msg.EBaseUserMessages_UM_RequestUtilAction:       func() proto.Message { return &msg.CUserMessageRequestUtilAction{} },
	msg.EBaseUserMessages_UM_RequestInventory:        func() proto.Message { return &msg.CUserMessageRequestInventory{} },
	msg.EBaseUserMessages_UM_InventoryResponse:       func() proto.Message { return &msg.CUserMessage_Inventory_Response{} },
	msg.EBaseUserMessages_UM_UtilActionResponse:      func() proto.Message { return &msg.CUserMessage_UtilMsg_Response{} },
	msg.EBaseUserMessages_UM_DllStatusResponse:       func() proto.Message { return &msg.CUserMessage_DllStatus{} },
	msg.EBaseUserMessages_UM_RequestDiagnostic:       func() proto.Message { return &msg.CUserMessageRequestDiagnostic{} },
	msg.EBaseUserMessages_UM_DiagnosticResponse:      func() proto.Message { return &msg.CUserMessage_Diagnostic_Response{} },
	msg.EBaseUserMessages_UM_ExtraUserData:           func() proto.Message { return &msg.CUserMessage_ExtraUserData{} },
	msg.EBaseUserMessages_UM_NotifyResponseFound:     func() proto.Message { return &msg.CUserMessage_NotifyResponseFound{} },
	msg.EBaseUserMessages_UM_PlayResponseConditional: func() proto.Message { return &msg.CUserMessage_PlayResponseConditional{} },
}

var emCreators = map[msg.EBaseEntityMessages]NetMessageCreator{
	msg.EBaseEntityMessages_EM_PlayJingle:      func() proto.Message { return &msg.CEntityMessagePlayJingle{} },
	msg.EBaseEntityMessages_EM_ScreenOverlay:   func() proto.Message { return &msg.CEntityMessageScreenOverlay{} },
	msg.EBaseEntityMessages_EM_RemoveAllDecals: func() proto.Message { return &msg.CEntityMessageRemoveAllDecals{} },
	msg.EBaseEntityMessages_EM_PropagateForce:  func() proto.Message { return &msg.CEntityMessagePropagateForce{} },
	msg.EBaseEntityMessages_EM_DoSpark:         func() proto.Message { return &msg.CEntityMessageDoSpark{} },
	msg.EBaseEntityMessages_EM_FixAngle:        func() proto.Message { return &msg.CEntityMessageFixAngle{} },
}

var gameEventCreators = map[msg.EBaseGameEvents]NetMessageCreator{
	msg.EBaseGameEvents_GE_VDebugGameSessionIDEvent:   func() proto.Message { return &msg.CMsgVDebugGameSessionIDEvent{} },
	msg.EBaseGameEvents_GE_PlaceDecalEvent:            func() proto.Message { return &msg.CMsgPlaceDecalEvent{} },
	msg.EBaseGameEvents_GE_ClearWorldDecalsEvent:      func() proto.Message { return &msg.CMsgClearWorldDecalsEvent{} },
	msg.EBaseGameEvents_GE_ClearEntityDecalsEvent:     func() proto.Message { return &msg.CMsgClearEntityDecalsEvent{} },
	msg.EBaseGameEvents_GE_ClearDecalsForEntityEvent:  func() proto.Message { return &msg.CMsgClearDecalsForEntityEvent{} },
	msg.EBaseGameEvents_GE_Source1LegacyGameEventList: func() proto.Message { return &msg.CMsgSource1LegacyGameEventList{} },
	msg.EBaseGameEvents_GE_Source1LegacyListenEvents:  func() proto.Message { return &msg.CMsgSource1LegacyListenEvents{} },
	msg.EBaseGameEvents_GE_Source1LegacyGameEvent:     func() proto.Message { return &msg.CMsgSource1LegacyGameEvent{} },
	msg.EBaseGameEvents_GE_SosStartSoundEvent:         func() proto.Message { return &msg.CMsgSosStartSoundEvent{} },
	msg.EBaseGameEvents_GE_SosStopSoundEvent:          func() proto.Message { return &msg.CMsgSosStopSoundEvent{} },
	msg.EBaseGameEvents_GE_SosSetSoundEventParams:     func() proto.Message { return &msg.CMsgSosSetSoundEventParams{} },
	msg.EBaseGameEvents_GE_SosSetLibraryStackFields:   func() proto.Message { return &msg.CMsgSosSetLibraryStackFields{} },
	msg.EBaseGameEvents_GE_SosStopSoundEventHash:      func() proto.Message { return &msg.CMsgSosStopSoundEventHash{} },
}

var csgoGameEventCreators = map[msg.ECsgoGameEvents]NetMessageCreator{
	msg.ECsgoGameEvents_GE_PlayerAnimEventId: func() proto.Message { return &msg.CMsgTEPlayerAnimEvent{} },
	msg.ECsgoGameEvents_GE_RadioIconEventId:  func() proto.Message { return &msg.CMsgTERadioIcon{} },
	msg.ECsgoGameEvents_GE_FireBulletsId:     func() proto.Message { return &msg.CMsgTEFireBullets{} },
}

var csUsrMsgCreators = map[msg.ECstrike15UserMessages]NetMessageCreator{
	msg.ECstrike15UserMessages_CS_UM_VGUIMenu:                     func() proto.Message { return &msg.CCSUsrMsg_VGUIMenu{} },
	msg.ECstrike15UserMessages_CS_UM_Geiger:                       func() proto.Message { return &msg.CCSUsrMsg_Geiger{} },
	msg.ECstrike15UserMessages_CS_UM_Train:                        func() proto.Message { return &msg.CCSUsrMsg_Train{} },
	msg.ECstrike15UserMessages_CS_UM_HudText:                      func() proto.Message { return &msg.CCSUsrMsg_HudText{} },
	msg.ECstrike15UserMessages_CS_UM_HudMsg:                       func() proto.Message { return &msg.CCSUsrMsg_HudMsg{} },
	msg.ECstrike15UserMessages_CS_UM_ResetHud:                     func() proto.Message { return &msg.CCSUsrMsg_ResetHud{} },
	msg.ECstrike15UserMessages_CS_UM_GameTitle:                    func() proto.Message { return &msg.CCSUsrMsg_GameTitle{} },
	msg.ECstrike15UserMessages_CS_UM_Shake:                        func() proto.Message { return &msg.CCSUsrMsg_Shake{} },
	msg.ECstrike15UserMessages_CS_UM_Fade:                         func() proto.Message { return &msg.CCSUsrMsg_Fade{} },
	msg.ECstrike15UserMessages_CS_UM_Rumble:                       func() proto.Message { return &msg.CCSUsrMsg_Rumble{} },
	msg.ECstrike15UserMessages_CS_UM_CloseCaption:                 func() proto.Message { return &msg.CCSUsrMsg_CloseCaption{} },
	msg.ECstrike15UserMessages_CS_UM_CloseCaptionDirect:           func() proto.Message { return &msg.CCSUsrMsg_CloseCaptionDirect{} },
	msg.ECstrike15UserMessages_CS_UM_SendAudio:                    func() proto.Message { return &msg.CCSUsrMsg_SendAudio{} },
	msg.ECstrike15UserMessages_CS_UM_RawAudio:                     func() proto.Message { return &msg.CCSUsrMsg_RawAudio{} },
	msg.ECstrike15UserMessages_CS_UM_VoiceMask:                    func() proto.Message { return &msg.CCSUsrMsg_VoiceMask{} },
	msg.ECstrike15UserMessages_CS_UM_RequestState:                 func() proto.Message { return &msg.CCSUsrMsg_RequestState{} },
	msg.ECstrike15UserMessages_CS_UM_Damage:                       func() proto.Message { return &msg.CCSUsrMsg_Damage{} },
	msg.ECstrike15UserMessages_CS_UM_RadioText:                    func() proto.Message { return &msg.CCSUsrMsg_RadioText{} },
	msg.ECstrike15UserMessages_CS_UM_HintText:                     func() proto.Message { return &msg.CCSUsrMsg_HintText{} },
	msg.ECstrike15UserMessages_CS_UM_KeyHintText:                  func() proto.Message { return &msg.CCSUsrMsg_KeyHintText{} },
	msg.ECstrike15UserMessages_CS_UM_ProcessSpottedEntityUpdate:   func() proto.Message { return &msg.CCSUsrMsg_ProcessSpottedEntityUpdate{} },
	msg.ECstrike15UserMessages_CS_UM_ReloadEffect:                 func() proto.Message { return &msg.CCSUsrMsg_ReloadEffect{} },
	msg.ECstrike15UserMessages_CS_UM_AdjustMoney:                  func() proto.Message { return &msg.CCSUsrMsg_AdjustMoney{} },
	msg.ECstrike15UserMessages_CS_UM_StopSpectatorMode:            func() proto.Message { return &msg.CCSUsrMsg_StopSpectatorMode{} },
	msg.ECstrike15UserMessages_CS_UM_KillCam:                      func() proto.Message { return &msg.CCSUsrMsg_KillCam{} },
	msg.ECstrike15UserMessages_CS_UM_DesiredTimescale:             func() proto.Message { return &msg.CCSUsrMsg_DesiredTimescale{} },
	msg.ECstrike15UserMessages_CS_UM_CurrentTimescale:             func() proto.Message { return &msg.CCSUsrMsg_CurrentTimescale{} },
	msg.ECstrike15UserMessages_CS_UM_AchievementEvent:             func() proto.Message { return &msg.CCSUsrMsg_AchievementEvent{} },
	msg.ECstrike15UserMessages_CS_UM_MatchEndConditions:           func() proto.Message { return &msg.CCSUsrMsg_MatchEndConditions{} },
	msg.ECstrike15UserMessages_CS_UM_DisconnectToLobby:            func() proto.Message { return &msg.CCSUsrMsg_DisconnectToLobby{} },
	msg.ECstrike15UserMessages_CS_UM_PlayerStatsUpdate:            func() proto.Message { return &msg.CCSUsrMsg_PlayerStatsUpdate{} },
	msg.ECstrike15UserMessages_CS_UM_ClientInfo:                   func() proto.Message { return &msg.CCSUsrMsg_ClientInfo{} },
	msg.ECstrike15UserMessages_CS_UM_XRankGet:                     func() proto.Message { return &msg.CCSUsrMsg_XRankGet{} },
	msg.ECstrike15UserMessages_CS_UM_XRankUpd:                     func() proto.Message { return &msg.CCSUsrMsg_XRankUpd{} },
	msg.ECstrike15UserMessages_CS_UM_CallVoteFailed:               func() proto.Message { return &msg.CCSUsrMsg_CallVoteFailed{} },
	msg.ECstrike15UserMessages_CS_UM_VoteStart:                    func() proto.Message { return &msg.CCSUsrMsg_VoteStart{} },
	msg.ECstrike15UserMessages_CS_UM_VotePass:                     func() proto.Message { return &msg.CCSUsrMsg_VotePass{} },
	msg.ECstrike15UserMessages_CS_UM_VoteFailed:                   func() proto.Message { return &msg.CCSUsrMsg_VoteFailed{} },
	msg.ECstrike15UserMessages_CS_UM_VoteSetup:                    func() proto.Message { return &msg.CCSUsrMsg_VoteSetup{} },
	msg.ECstrike15UserMessages_CS_UM_ServerRankRevealAll:          func() proto.Message { return &msg.CCSUsrMsg_ServerRankRevealAll{} },
	msg.ECstrike15UserMessages_CS_UM_SendLastKillerDamageToClient: func() proto.Message { return &msg.CCSUsrMsg_SendLastKillerDamageToClient{} },
	msg.ECstrike15UserMessages_CS_UM_ServerRankUpdate:             func() proto.Message { return &msg.CCSUsrMsg_ServerRankUpdate{} },
	msg.ECstrike15UserMessages_CS_UM_ItemPickup:                   func() proto.Message { return &msg.CCSUsrMsg_ItemPickup{} },
	msg.ECstrike15UserMessages_CS_UM_ShowMenu:                     func() proto.Message { return &msg.CCSUsrMsg_ShowMenu{} },
	msg.ECstrike15UserMessages_CS_UM_BarTime:                      func() proto.Message { return &msg.CCSUsrMsg_BarTime{} },
	msg.ECstrike15UserMessages_CS_UM_AmmoDenied:                   func() proto.Message { return &msg.CCSUsrMsg_AmmoDenied{} },
	msg.ECstrike15UserMessages_CS_UM_MarkAchievement:              func() proto.Message { return &msg.CCSUsrMsg_MarkAchievement{} },
	msg.ECstrike15UserMessages_CS_UM_MatchStatsUpdate:             func() proto.Message { return &msg.CCSUsrMsg_MatchStatsUpdate{} },
	msg.ECstrike15UserMessages_CS_UM_ItemDrop:                     func() proto.Message { return &msg.CCSUsrMsg_ItemDrop{} },
	msg.ECstrike15UserMessages_CS_UM_SendPlayerItemDrops:          func() proto.Message { return &msg.CCSUsrMsg_SendPlayerItemDrops{} },
	msg.ECstrike15UserMessages_CS_UM_RoundBackupFilenames:         func() proto.Message { return &msg.CCSUsrMsg_RoundBackupFilenames{} },
	msg.ECstrike15UserMessages_CS_UM_SendPlayerItemFound:          func() proto.Message { return &msg.CCSUsrMsg_SendPlayerItemFound{} },
	msg.ECstrike15UserMessages_CS_UM_ReportHit:                    func() proto.Message { return &msg.CCSUsrMsg_ReportHit{} },
	msg.ECstrike15UserMessages_CS_UM_XpUpdate:                     func() proto.Message { return &msg.CCSUsrMsg_XpUpdate{} },
	msg.ECstrike15UserMessages_CS_UM_QuestProgress:                func() proto.Message { return &msg.CCSUsrMsg_QuestProgress{} },
	msg.ECstrike15UserMessages_CS_UM_ScoreLeaderboardData:         func() proto.Message { return &msg.CCSUsrMsg_ScoreLeaderboardData{} },
	msg.ECstrike15UserMessages_CS_UM_PlayerDecalDigitalSignature:  func() proto.Message { return &msg.CCSUsrMsg_PlayerDecalDigitalSignature{} },
	msg.ECstrike15UserMessages_CS_UM_WeaponSound:                  func() proto.Message { return &msg.CCSUsrMsg_WeaponSound{} },
	msg.ECstrike15UserMessages_CS_UM_UpdateScreenHealthBar:        func() proto.Message { return &msg.CCSUsrMsg_UpdateScreenHealthBar{} },
	msg.ECstrike15UserMessages_CS_UM_EntityOutlineHighlight:       func() proto.Message { return &msg.CCSUsrMsg_EntityOutlineHighlight{} },
	msg.ECstrike15UserMessages_CS_UM_SSUI:                         func() proto.Message { return &msg.CCSUsrMsg_SSUI{} },
	msg.ECstrike15UserMessages_CS_UM_SurvivalStats:                func() proto.Message { return &msg.CCSUsrMsg_SurvivalStats{} },
	msg.ECstrike15UserMessages_CS_UM_DisconnectToLobby2:           func() proto.Message { return &msg.CCSUsrMsg_DisconnectToLobby{} },
	msg.ECstrike15UserMessages_CS_UM_EndOfMatchAllPlayersData:     func() proto.Message { return &msg.CCSUsrMsg_EndOfMatchAllPlayersData{} },
	msg.ECstrike15UserMessages_CS_UM_PostRoundDamageReport:        func() proto.Message { return &msg.CCSUsrMsg_PostRoundDamageReport{} },
	msg.ECstrike15UserMessages_CS_UM_RoundEndReportData:           func() proto.Message { return &msg.CCSUsrMsg_RoundEndReportData{} },
	msg.ECstrike15UserMessages_CS_UM_CurrentRoundOdds:             func() proto.Message { return &msg.CCSUsrMsg_CurrentRoundOdds{} },
	msg.ECstrike15UserMessages_CS_UM_DeepStats:                    func() proto.Message { return &msg.CCSUsrMsg_DeepStats{} },
	msg.ECstrike15UserMessages_CS_UM_ShootInfo:                    func() proto.Message { return &msg.CCSUsrMsg_ShootInfo{} },
}

var teCreators = map[msg.ETEProtobufIds]NetMessageCreator{
	msg.ETEProtobufIds_TE_EffectDispatchId: func() proto.Message { return &msg.CMsgTEEffectDispatch{} },
	msg.ETEProtobufIds_TE_ArmorRicochetId:  func() proto.Message { return &msg.CMsgTEArmorRicochet{} },
	msg.ETEProtobufIds_TE_BeamEntPointId:   func() proto.Message { return &msg.CMsgTEBeamEntPoint{} },
	msg.ETEProtobufIds_TE_BeamEntsId:       func() proto.Message { return &msg.CMsgTEBeamEnts{} },
	msg.ETEProtobufIds_TE_BeamPointsId:     func() proto.Message { return &msg.CMsgTEBeamPoints{} },
	msg.ETEProtobufIds_TE_BeamRingId:       func() proto.Message { return &msg.CMsgTEBeamRing{} },
	msg.ETEProtobufIds_TE_BubblesId:        func() proto.Message { return &msg.CMsgTEBubbles{} },
	msg.ETEProtobufIds_TE_BubbleTrailId:    func() proto.Message { return &msg.CMsgTEBubbleTrail{} },
	msg.ETEProtobufIds_TE_DecalId:          func() proto.Message { return &msg.CMsgTEDecal{} },
	msg.ETEProtobufIds_TE_WorldDecalId:     func() proto.Message { return &msg.CMsgTEWorldDecal{} },
	msg.ETEProtobufIds_TE_EnergySplashId:   func() proto.Message { return &msg.CMsgTEEnergySplash{} },
	msg.ETEProtobufIds_TE_FizzId:           func() proto.Message { return &msg.CMsgTEFizz{} },
	msg.ETEProtobufIds_TE_ShatterSurfaceId: func() proto.Message { return &msg.CMsgTEShatterSurface{} },
	msg.ETEProtobufIds_TE_GlowSpriteId:     func() proto.Message { return &msg.CMsgTEGlowSprite{} },
	msg.ETEProtobufIds_TE_ImpactId:         func() proto.Message { return &msg.CMsgTEImpact{} },
	msg.ETEProtobufIds_TE_MuzzleFlashId:    func() proto.Message { return &msg.CMsgTEMuzzleFlash{} },
	msg.ETEProtobufIds_TE_BloodStreamId:    func() proto.Message { return &msg.CMsgTEBloodStream{} },
	msg.ETEProtobufIds_TE_ExplosionId:      func() proto.Message { return &msg.CMsgTEExplosion{} },
	msg.ETEProtobufIds_TE_DustId:           func() proto.Message { return &msg.CMsgTEDust{} },
	msg.ETEProtobufIds_TE_LargeFunnelId:    func() proto.Message { return &msg.CMsgTELargeFunnel{} },
	msg.ETEProtobufIds_TE_SparksId:         func() proto.Message { return &msg.CMsgTESparks{} },
	msg.ETEProtobufIds_TE_PhysicsPropId:    func() proto.Message { return &msg.CMsgTEPhysicsProp{} },
	msg.ETEProtobufIds_TE_SmokeId:          func() proto.Message { return &msg.CMsgTESmoke{} },
}

var bidirectionalMessageCreators = map[msg.Bidirectional_Messages]NetMessageCreator{
	msg.Bidirectional_Messages_bi_RebroadcastGameEvent: func() proto.Message { return &msg.CBidirMsg_RebroadcastGameEvent{} },
	msg.Bidirectional_Messages_bi_RebroadcastSource:    func() proto.Message { return &msg.CBidirMsg_RebroadcastSource{} },
	msg.Bidirectional_Messages_bi_GameEvent:            func() proto.Message { return &msg.CBidirMsg_RebroadcastGameEvent{} },
	msg.Bidirectional_Messages_bi_PredictionEvent:      func() proto.Message { return &msg.CBidirMsg_PredictionEvent{} },
}

type pendingMessage struct {
	t   int32
	buf []byte
}

// Calculates the priority of the message. Lower is more important.
func (m *pendingMessage) priority() int {
	switch m.t {
	case
		// These messages provide context needed for the rest of the tick
		// and should have the highest priority.
		int32(msg.NET_Messages_net_Tick),
		int32(msg.SVC_Messages_svc_CreateStringTable),
		int32(msg.SVC_Messages_svc_UpdateStringTable),
		int32(msg.NET_Messages_net_SpawnGroup_Load):
		return -10

	case
		// These messages benefit from having context but may also need to
		// provide context in terms of delta updates.
		int32(msg.SVC_Messages_svc_PacketEntities):
		return 5
	}

	return 0
}

func (p *parser) handleDemoPacket(pack *msg.CDemoPacket) {
	b := pack.GetData()

	if len(b) == 0 {
		return
	}

	r := bitread.NewSmallBitReader(bytes.NewReader(b))

	p.pendingMessagesCache = p.pendingMessagesCache[:0]

	for len(b)*8-r.ActualPosition() > 7 {
		t := int32(r.ReadUBitInt())
		size := r.ReadVarInt32()
		buf := r.ReadBytes(int(size))

		p.pendingMessagesCache = append(p.pendingMessagesCache, pendingMessage{t, buf})
	}

	slices.SortStableFunc(p.pendingMessagesCache, func(a, b pendingMessage) int {
		return a.priority() - b.priority()
	})

	for _, m := range p.pendingMessagesCache {
		var msgCreator NetMessageCreator

		if m.t < int32(msg.SVC_Messages_svc_ServerInfo) {
			msgCreator = netMsgCreators[msg.NET_Messages(m.t)]

			if msgCreator == nil {
				msgCreator = bidirectionalMessageCreators[msg.Bidirectional_Messages(m.t)]
			}
		} else if m.t < int32(msg.EBaseUserMessages_UM_AchievementEvent) {
			msgCreator = svcMsgCreators[msg.SVC_Messages(m.t)]
		} else if m.t < int32(msg.EBaseGameEvents_GE_VDebugGameSessionIDEvent) {
			msgCreator = usrMsgCreators[msg.EBaseUserMessages(m.t)]

			if msgCreator == nil {
				msgCreator = emCreators[msg.EBaseEntityMessages(m.t)]
			}
		} else if m.t < int32(msg.ECstrike15UserMessages_CS_UM_VGUIMenu) {
			msgCreator = gameEventCreators[msg.EBaseGameEvents(m.t)]
		} else if m.t < int32(msg.ETEProtobufIds_TE_EffectDispatchId) {
			msgCreator = csUsrMsgCreators[msg.ECstrike15UserMessages(m.t)]
		} else if m.t < int32(msg.ECsgoGameEvents_GE_PlayerAnimEventId) {
			msgCreator = teCreators[msg.ETEProtobufIds(m.t)]
		} else {
			msgCreator = csgoGameEventCreators[msg.ECsgoGameEvents(m.t)]
		}

		if msgCreator == nil {
			p.msgDispatcher.Dispatch(events.ParserWarn{
				Message: fmt.Sprintf("unknown message type: %d", m.t),
				Type:    events.WarnTypeUnknownProtobufMessage,
			})

			continue
		}

		msg := msgCreator()

		err := proto.Unmarshal(m.buf, msg)
		if err != nil {
			panic(err) // FIXME: avoid panic
		}

		p.msgQueue <- msg
	}
}

func (p *parser) handleFullPacket(msg *msg.CDemoFullPacket) {
	p.handleStringTables(msg.StringTable)

	if msg.Packet.GetData() != nil {
		p.handleDemoPacket(msg.Packet)
	}
}

func (p *parser) handleFileInfo(msg *msg.CDemoFileInfo) {
	p.header.PlaybackTicks = int(*msg.PlaybackTicks)
	p.header.PlaybackFrames = int(*msg.PlaybackFrames)
	p.header.PlaybackTime = time.Duration(*msg.PlaybackTime) * time.Second
}

//go:embed event-list-dump/*.bin
var eventListFolder embed.FS

func getGameEventListBinForProtocol(networkProtocol int) ([]byte, error) {
	switch {
	case networkProtocol < 13992:
		return eventListFolder.ReadFile("event-list-dump/13990.bin")

	case networkProtocol < 14023:
		return eventListFolder.ReadFile("event-list-dump/13992.bin")

	case networkProtocol < 14069:
		return eventListFolder.ReadFile("event-list-dump/14023.bin")

	case networkProtocol < 14089:
		return eventListFolder.ReadFile("event-list-dump/14070.bin")

	default:
		return eventListFolder.ReadFile("event-list-dump/14089.bin")
	}
}

func (p *parser) handleDemoFileHeader(msg *msg.CDemoFileHeader) {
	p.header.ClientName = msg.GetClientName()
	p.header.ServerName = msg.GetServerName()
	p.header.GameDirectory = msg.GetGameDirectory()
	p.header.MapName = msg.GetMapName()
	networkProtocol := int(msg.GetNetworkProtocol())
	p.header.NetworkProtocol = networkProtocol

	if p.source2FallbackGameEventListBin == nil {
		gameEventListBin, err := getGameEventListBinForProtocol(networkProtocol)
		if err != nil {
			panic(fmt.Sprintf("failed to load game event list for protocol %d: %v", networkProtocol, err))
		}

		p.source2FallbackGameEventListBin = gameEventListBin
	}
}
