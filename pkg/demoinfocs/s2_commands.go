package demoinfocs

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

func (p *parser) handleSendTables(msg *msgs2.CDemoSendTables) {
	err := p.stParser.ParsePacket(msg.Data)
	if err != nil {
		panic(errors.Wrap(err, "failed to unmarshal flattened serializer"))
	}
}

func (p *parser) handleClassInfo(msg *msgs2.CDemoClassInfo) {
	err := p.stParser.OnDemoClassInfo(msg)
	if err != nil {
		panic(err)
	}

	debugAllServerClasses(p.ServerClasses())

	p.mapEquipment()
	p.bindEntities()

	p.eventDispatcher.Dispatch(events.DataTablesParsed{})
}

var netMsgCreators = map[msgs2.NET_Messages]NetMessageCreator{
	msgs2.NET_Messages_net_NOP:                        func() proto.Message { return &msgs2.CNETMsg_NOP{} },
	msgs2.NET_Messages_net_SplitScreenUser:            func() proto.Message { return &msgs2.CNETMsg_SplitScreenUser{} },
	msgs2.NET_Messages_net_Tick:                       func() proto.Message { return &msgs2.CNETMsg_Tick{} },
	msgs2.NET_Messages_net_StringCmd:                  func() proto.Message { return &msgs2.CNETMsg_StringCmd{} },
	msgs2.NET_Messages_net_SetConVar:                  func() proto.Message { return &msgs2.CNETMsg_SetConVar{} },
	msgs2.NET_Messages_net_SignonState:                func() proto.Message { return &msgs2.CNETMsg_SignonState{} },
	msgs2.NET_Messages_net_SpawnGroup_Load:            func() proto.Message { return &msgs2.CNETMsg_SpawnGroup_Load{} },
	msgs2.NET_Messages_net_SpawnGroup_ManifestUpdate:  func() proto.Message { return &msgs2.CNETMsg_SpawnGroup_ManifestUpdate{} },
	msgs2.NET_Messages_net_SpawnGroup_SetCreationTick: func() proto.Message { return &msgs2.CNETMsg_SpawnGroup_SetCreationTick{} },
	msgs2.NET_Messages_net_SpawnGroup_Unload:          func() proto.Message { return &msgs2.CNETMsg_SpawnGroup_Unload{} },
	msgs2.NET_Messages_net_SpawnGroup_LoadCompleted:   func() proto.Message { return &msgs2.CNETMsg_SpawnGroup_LoadCompleted{} },
	msgs2.NET_Messages_net_DebugOverlay:               func() proto.Message { return &msgs2.CNETMsg_DebugOverlay{} },
}

var svcMsgCreators = map[msgs2.SVC_Messages]NetMessageCreator{
	msgs2.SVC_Messages_svc_ServerInfo:              func() proto.Message { return &msgs2.CSVCMsg_ServerInfo{} },
	msgs2.SVC_Messages_svc_FlattenedSerializer:     func() proto.Message { return &msgs2.CSVCMsg_FlattenedSerializer{} },
	msgs2.SVC_Messages_svc_ClassInfo:               func() proto.Message { return &msgs2.CSVCMsg_ClassInfo{} },
	msgs2.SVC_Messages_svc_SetPause:                func() proto.Message { return &msgs2.CSVCMsg_SetPause{} },
	msgs2.SVC_Messages_svc_CreateStringTable:       func() proto.Message { return &msgs2.CSVCMsg_CreateStringTable{} },
	msgs2.SVC_Messages_svc_UpdateStringTable:       func() proto.Message { return &msgs2.CSVCMsg_UpdateStringTable{} },
	msgs2.SVC_Messages_svc_VoiceInit:               func() proto.Message { return &msgs2.CSVCMsg_VoiceInit{} },
	msgs2.SVC_Messages_svc_VoiceData:               func() proto.Message { return &msgs2.CSVCMsg_VoiceData{} },
	msgs2.SVC_Messages_svc_Print:                   func() proto.Message { return &msgs2.CSVCMsg_Print{} },
	msgs2.SVC_Messages_svc_Sounds:                  func() proto.Message { return &msgs2.CSVCMsg_Sounds{} },
	msgs2.SVC_Messages_svc_SetView:                 func() proto.Message { return &msgs2.CSVCMsg_SetView{} },
	msgs2.SVC_Messages_svc_ClearAllStringTables:    func() proto.Message { return &msgs2.CSVCMsg_ClearAllStringTables{} },
	msgs2.SVC_Messages_svc_CmdKeyValues:            func() proto.Message { return &msgs2.CSVCMsg_CmdKeyValues{} },
	msgs2.SVC_Messages_svc_BSPDecal:                func() proto.Message { return &msgs2.CSVCMsg_BSPDecal{} },
	msgs2.SVC_Messages_svc_SplitScreen:             func() proto.Message { return &msgs2.CSVCMsg_SplitScreen{} },
	msgs2.SVC_Messages_svc_PacketEntities:          func() proto.Message { return &msgs2.CSVCMsg_PacketEntities{} },
	msgs2.SVC_Messages_svc_Prefetch:                func() proto.Message { return &msgs2.CSVCMsg_Prefetch{} },
	msgs2.SVC_Messages_svc_Menu:                    func() proto.Message { return &msgs2.CSVCMsg_Menu{} },
	msgs2.SVC_Messages_svc_GetCvarValue:            func() proto.Message { return &msgs2.CSVCMsg_GetCvarValue{} },
	msgs2.SVC_Messages_svc_StopSound:               func() proto.Message { return &msgs2.CSVCMsg_StopSound{} },
	msgs2.SVC_Messages_svc_PeerList:                func() proto.Message { return &msgs2.CSVCMsg_PeerList{} },
	msgs2.SVC_Messages_svc_PacketReliable:          func() proto.Message { return &msgs2.CSVCMsg_PacketReliable{} },
	msgs2.SVC_Messages_svc_HLTVStatus:              func() proto.Message { return &msgs2.CSVCMsg_HLTVStatus{} },
	msgs2.SVC_Messages_svc_ServerSteamID:           func() proto.Message { return &msgs2.CSVCMsg_ServerSteamID{} },
	msgs2.SVC_Messages_svc_FullFrameSplit:          func() proto.Message { return &msgs2.CSVCMsg_FullFrameSplit{} },
	msgs2.SVC_Messages_svc_RconServerDetails:       func() proto.Message { return &msgs2.CSVCMsg_RconServerDetails{} },
	msgs2.SVC_Messages_svc_UserMessage:             func() proto.Message { return &msgs2.CSVCMsg_UserMessage{} },
	msgs2.SVC_Messages_svc_Broadcast_Command:       func() proto.Message { return &msgs2.CSVCMsg_Broadcast_Command{} },
	msgs2.SVC_Messages_svc_HltvFixupOperatorStatus: func() proto.Message { return &msgs2.CSVCMsg_HltvFixupOperatorStatus{} },
	msgs2.SVC_Messages_svc_UserCmds:                func() proto.Message { return &msgs2.CSVCMsg_UserCommands{} },
}

var usrMsgCreators = map[msgs2.EBaseUserMessages]NetMessageCreator{
	msgs2.EBaseUserMessages_UM_AchievementEvent:        func() proto.Message { return &msgs2.CUserMessageAchievementEvent{} },
	msgs2.EBaseUserMessages_UM_CloseCaption:            func() proto.Message { return &msgs2.CUserMessageCloseCaption{} },
	msgs2.EBaseUserMessages_UM_CloseCaptionDirect:      func() proto.Message { return &msgs2.CUserMessageCloseCaptionDirect{} },
	msgs2.EBaseUserMessages_UM_CurrentTimescale:        func() proto.Message { return &msgs2.CUserMessageCurrentTimescale{} },
	msgs2.EBaseUserMessages_UM_DesiredTimescale:        func() proto.Message { return &msgs2.CUserMessageDesiredTimescale{} },
	msgs2.EBaseUserMessages_UM_Fade:                    func() proto.Message { return &msgs2.CUserMessageFade{} },
	msgs2.EBaseUserMessages_UM_GameTitle:               func() proto.Message { return &msgs2.CUserMessageGameTitle{} },
	msgs2.EBaseUserMessages_UM_HudMsg:                  func() proto.Message { return &msgs2.CUserMessageHudMsg{} },
	msgs2.EBaseUserMessages_UM_HudText:                 func() proto.Message { return &msgs2.CUserMessageHudText{} },
	msgs2.EBaseUserMessages_UM_ColoredText:             func() proto.Message { return &msgs2.CUserMessageColoredText{} },
	msgs2.EBaseUserMessages_UM_RequestState:            func() proto.Message { return &msgs2.CUserMessageRequestState{} },
	msgs2.EBaseUserMessages_UM_ResetHUD:                func() proto.Message { return &msgs2.CUserMessageResetHUD{} },
	msgs2.EBaseUserMessages_UM_Rumble:                  func() proto.Message { return &msgs2.CUserMessageRumble{} },
	msgs2.EBaseUserMessages_UM_SayText:                 func() proto.Message { return &msgs2.CUserMessageSayText{} },
	msgs2.EBaseUserMessages_UM_SayText2:                func() proto.Message { return &msgs2.CUserMessageSayText2{} },
	msgs2.EBaseUserMessages_UM_SayTextChannel:          func() proto.Message { return &msgs2.CUserMessageSayTextChannel{} },
	msgs2.EBaseUserMessages_UM_Shake:                   func() proto.Message { return &msgs2.CUserMessageShake{} },
	msgs2.EBaseUserMessages_UM_ShakeDir:                func() proto.Message { return &msgs2.CUserMessageShakeDir{} },
	msgs2.EBaseUserMessages_UM_TextMsg:                 func() proto.Message { return &msgs2.CUserMessageTextMsg{} },
	msgs2.EBaseUserMessages_UM_ScreenTilt:              func() proto.Message { return &msgs2.CUserMessageScreenTilt{} },
	msgs2.EBaseUserMessages_UM_VoiceMask:               func() proto.Message { return &msgs2.CUserMessageVoiceMask{} },
	msgs2.EBaseUserMessages_UM_SendAudio:               func() proto.Message { return &msgs2.CUserMessageSendAudio{} },
	msgs2.EBaseUserMessages_UM_ItemPickup:              func() proto.Message { return &msgs2.CUserMessageItemPickup{} },
	msgs2.EBaseUserMessages_UM_AmmoDenied:              func() proto.Message { return &msgs2.CUserMessageAmmoDenied{} },
	msgs2.EBaseUserMessages_UM_ShowMenu:                func() proto.Message { return &msgs2.CUserMessageShowMenu{} },
	msgs2.EBaseUserMessages_UM_CreditsMsg:              func() proto.Message { return &msgs2.CUserMessageCreditsMsg{} },
	msgs2.EBaseUserMessages_UM_CloseCaptionPlaceholder: func() proto.Message { return &msgs2.CUserMessageCloseCaptionPlaceholder{} },
	msgs2.EBaseUserMessages_UM_CameraTransition:        func() proto.Message { return &msgs2.CUserMessageCameraTransition{} },
	msgs2.EBaseUserMessages_UM_AudioParameter:          func() proto.Message { return &msgs2.CUserMessageAudioParameter{} },
	msgs2.EBaseUserMessages_UM_ParticleManager:         func() proto.Message { return &msgs2.CUserMsg_ParticleManager{} },
	msgs2.EBaseUserMessages_UM_HudError:                func() proto.Message { return &msgs2.CUserMsg_HudError{} },
	msgs2.EBaseUserMessages_UM_CustomGameEvent:         func() proto.Message { return &msgs2.CUserMsg_CustomGameEvent{} },
	msgs2.EBaseUserMessages_UM_AnimGraphUpdate:         func() proto.Message { return &msgs2.CUserMessageAnimStateGraphState{} },
	msgs2.EBaseUserMessages_UM_HapticsManagerPulse:     func() proto.Message { return &msgs2.CUserMessageHapticsManagerPulse{} },
	msgs2.EBaseUserMessages_UM_HapticsManagerEffect:    func() proto.Message { return &msgs2.CUserMessageHapticsManagerEffect{} },
	msgs2.EBaseUserMessages_UM_UpdateCssClasses:        func() proto.Message { return &msgs2.CUserMessageUpdateCssClasses{} },
	msgs2.EBaseUserMessages_UM_ServerFrameTime:         func() proto.Message { return &msgs2.CUserMessageServerFrameTime{} },
	msgs2.EBaseUserMessages_UM_LagCompensationError:    func() proto.Message { return &msgs2.CUserMessageLagCompensationError{} },
	msgs2.EBaseUserMessages_UM_RequestDllStatus:        func() proto.Message { return &msgs2.CUserMessageRequestDllStatus{} },
	msgs2.EBaseUserMessages_UM_RequestUtilAction:       func() proto.Message { return &msgs2.CUserMessageRequestUtilAction{} },
	msgs2.EBaseUserMessages_UM_RequestInventory:        func() proto.Message { return &msgs2.CUserMessageRequestInventory{} },
	msgs2.EBaseUserMessages_UM_InventoryResponse:       func() proto.Message { return &msgs2.CUserMessage_Inventory_Response{} },
	msgs2.EBaseUserMessages_UM_UtilActionResponse:      func() proto.Message { return &msgs2.CUserMessage_UtilMsg_Response{} },
	msgs2.EBaseUserMessages_UM_DllStatusResponse:       func() proto.Message { return &msgs2.CUserMessage_DllStatus{} },
	msgs2.EBaseUserMessages_UM_RequestDiagnostic:       func() proto.Message { return &msgs2.CUserMessageRequestDiagnostic{} },
	msgs2.EBaseUserMessages_UM_DiagnosticResponse:      func() proto.Message { return &msgs2.CUserMessage_Diagnostic_Response{} },
	msgs2.EBaseUserMessages_UM_ExtraUserData:           func() proto.Message { return &msgs2.CUserMessage_ExtraUserData{} },
	msgs2.EBaseUserMessages_UM_NotifyResponseFound:     func() proto.Message { return &msgs2.CUserMessage_NotifyResponseFound{} },
	msgs2.EBaseUserMessages_UM_PlayResponseConditional: func() proto.Message { return &msgs2.CUserMessage_PlayResponseConditional{} },
}

var emCreators = map[msgs2.EBaseEntityMessages]NetMessageCreator{
	msgs2.EBaseEntityMessages_EM_PlayJingle:      func() proto.Message { return &msgs2.CEntityMessagePlayJingle{} },
	msgs2.EBaseEntityMessages_EM_ScreenOverlay:   func() proto.Message { return &msgs2.CEntityMessageScreenOverlay{} },
	msgs2.EBaseEntityMessages_EM_RemoveAllDecals: func() proto.Message { return &msgs2.CEntityMessageRemoveAllDecals{} },
	msgs2.EBaseEntityMessages_EM_PropagateForce:  func() proto.Message { return &msgs2.CEntityMessagePropagateForce{} },
	msgs2.EBaseEntityMessages_EM_DoSpark:         func() proto.Message { return &msgs2.CEntityMessageDoSpark{} },
	msgs2.EBaseEntityMessages_EM_FixAngle:        func() proto.Message { return &msgs2.CEntityMessageFixAngle{} },
}

var gameEventCreators = map[msgs2.EBaseGameEvents]NetMessageCreator{
	msgs2.EBaseGameEvents_GE_VDebugGameSessionIDEvent:   func() proto.Message { return &msgs2.CMsgVDebugGameSessionIDEvent{} },
	msgs2.EBaseGameEvents_GE_PlaceDecalEvent:            func() proto.Message { return &msgs2.CMsgPlaceDecalEvent{} },
	msgs2.EBaseGameEvents_GE_ClearWorldDecalsEvent:      func() proto.Message { return &msgs2.CMsgClearWorldDecalsEvent{} },
	msgs2.EBaseGameEvents_GE_ClearEntityDecalsEvent:     func() proto.Message { return &msgs2.CMsgClearEntityDecalsEvent{} },
	msgs2.EBaseGameEvents_GE_ClearDecalsForEntityEvent:  func() proto.Message { return &msgs2.CMsgClearDecalsForEntityEvent{} },
	msgs2.EBaseGameEvents_GE_Source1LegacyGameEventList: func() proto.Message { return &msgs2.CMsgSource1LegacyGameEventList{} },
	msgs2.EBaseGameEvents_GE_Source1LegacyListenEvents:  func() proto.Message { return &msgs2.CMsgSource1LegacyListenEvents{} },
	msgs2.EBaseGameEvents_GE_Source1LegacyGameEvent:     func() proto.Message { return &msgs2.CMsgSource1LegacyGameEvent{} },
	msgs2.EBaseGameEvents_GE_SosStartSoundEvent:         func() proto.Message { return &msgs2.CMsgSosStartSoundEvent{} },
	msgs2.EBaseGameEvents_GE_SosStopSoundEvent:          func() proto.Message { return &msgs2.CMsgSosStopSoundEvent{} },
	msgs2.EBaseGameEvents_GE_SosSetSoundEventParams:     func() proto.Message { return &msgs2.CMsgSosSetSoundEventParams{} },
	msgs2.EBaseGameEvents_GE_SosSetLibraryStackFields:   func() proto.Message { return &msgs2.CMsgSosSetLibraryStackFields{} },
	msgs2.EBaseGameEvents_GE_SosStopSoundEventHash:      func() proto.Message { return &msgs2.CMsgSosStopSoundEventHash{} },
}

var csgoGameEventCreators = map[msgs2.ECsgoGameEvents]NetMessageCreator{
	msgs2.ECsgoGameEvents_GE_PlayerAnimEventId: func() proto.Message { return &msgs2.CMsgTEPlayerAnimEvent{} },
	msgs2.ECsgoGameEvents_GE_RadioIconEventId:  func() proto.Message { return &msgs2.CMsgTERadioIcon{} },
	msgs2.ECsgoGameEvents_GE_FireBulletsId:     func() proto.Message { return &msgs2.CMsgTEFireBullets{} },
}

var csUsrMsgCreators = map[msgs2.ECstrike15UserMessages]NetMessageCreator{
	msgs2.ECstrike15UserMessages_CS_UM_VGUIMenu:                     func() proto.Message { return &msgs2.CCSUsrMsg_VGUIMenu{} },
	msgs2.ECstrike15UserMessages_CS_UM_Geiger:                       func() proto.Message { return &msgs2.CCSUsrMsg_Geiger{} },
	msgs2.ECstrike15UserMessages_CS_UM_Train:                        func() proto.Message { return &msgs2.CCSUsrMsg_Train{} },
	msgs2.ECstrike15UserMessages_CS_UM_HudText:                      func() proto.Message { return &msgs2.CCSUsrMsg_HudText{} },
	msgs2.ECstrike15UserMessages_CS_UM_HudMsg:                       func() proto.Message { return &msgs2.CCSUsrMsg_HudMsg{} },
	msgs2.ECstrike15UserMessages_CS_UM_ResetHud:                     func() proto.Message { return &msgs2.CCSUsrMsg_ResetHud{} },
	msgs2.ECstrike15UserMessages_CS_UM_GameTitle:                    func() proto.Message { return &msgs2.CCSUsrMsg_GameTitle{} },
	msgs2.ECstrike15UserMessages_CS_UM_Shake:                        func() proto.Message { return &msgs2.CCSUsrMsg_Shake{} },
	msgs2.ECstrike15UserMessages_CS_UM_Fade:                         func() proto.Message { return &msgs2.CCSUsrMsg_Fade{} },
	msgs2.ECstrike15UserMessages_CS_UM_Rumble:                       func() proto.Message { return &msgs2.CCSUsrMsg_Rumble{} },
	msgs2.ECstrike15UserMessages_CS_UM_CloseCaption:                 func() proto.Message { return &msgs2.CCSUsrMsg_CloseCaption{} },
	msgs2.ECstrike15UserMessages_CS_UM_CloseCaptionDirect:           func() proto.Message { return &msgs2.CCSUsrMsg_CloseCaptionDirect{} },
	msgs2.ECstrike15UserMessages_CS_UM_SendAudio:                    func() proto.Message { return &msgs2.CCSUsrMsg_SendAudio{} },
	msgs2.ECstrike15UserMessages_CS_UM_RawAudio:                     func() proto.Message { return &msgs2.CCSUsrMsg_RawAudio{} },
	msgs2.ECstrike15UserMessages_CS_UM_VoiceMask:                    func() proto.Message { return &msgs2.CCSUsrMsg_VoiceMask{} },
	msgs2.ECstrike15UserMessages_CS_UM_RequestState:                 func() proto.Message { return &msgs2.CCSUsrMsg_RequestState{} },
	msgs2.ECstrike15UserMessages_CS_UM_Damage:                       func() proto.Message { return &msgs2.CCSUsrMsg_Damage{} },
	msgs2.ECstrike15UserMessages_CS_UM_RadioText:                    func() proto.Message { return &msgs2.CCSUsrMsg_RadioText{} },
	msgs2.ECstrike15UserMessages_CS_UM_HintText:                     func() proto.Message { return &msgs2.CCSUsrMsg_HintText{} },
	msgs2.ECstrike15UserMessages_CS_UM_KeyHintText:                  func() proto.Message { return &msgs2.CCSUsrMsg_KeyHintText{} },
	msgs2.ECstrike15UserMessages_CS_UM_ProcessSpottedEntityUpdate:   func() proto.Message { return &msgs2.CCSUsrMsg_ProcessSpottedEntityUpdate{} },
	msgs2.ECstrike15UserMessages_CS_UM_ReloadEffect:                 func() proto.Message { return &msgs2.CCSUsrMsg_ReloadEffect{} },
	msgs2.ECstrike15UserMessages_CS_UM_AdjustMoney:                  func() proto.Message { return &msgs2.CCSUsrMsg_AdjustMoney{} },
	msgs2.ECstrike15UserMessages_CS_UM_StopSpectatorMode:            func() proto.Message { return &msgs2.CCSUsrMsg_StopSpectatorMode{} },
	msgs2.ECstrike15UserMessages_CS_UM_KillCam:                      func() proto.Message { return &msgs2.CCSUsrMsg_KillCam{} },
	msgs2.ECstrike15UserMessages_CS_UM_DesiredTimescale:             func() proto.Message { return &msgs2.CCSUsrMsg_DesiredTimescale{} },
	msgs2.ECstrike15UserMessages_CS_UM_CurrentTimescale:             func() proto.Message { return &msgs2.CCSUsrMsg_CurrentTimescale{} },
	msgs2.ECstrike15UserMessages_CS_UM_AchievementEvent:             func() proto.Message { return &msgs2.CCSUsrMsg_AchievementEvent{} },
	msgs2.ECstrike15UserMessages_CS_UM_MatchEndConditions:           func() proto.Message { return &msgs2.CCSUsrMsg_MatchEndConditions{} },
	msgs2.ECstrike15UserMessages_CS_UM_DisconnectToLobby:            func() proto.Message { return &msgs2.CCSUsrMsg_DisconnectToLobby{} },
	msgs2.ECstrike15UserMessages_CS_UM_PlayerStatsUpdate:            func() proto.Message { return &msgs2.CCSUsrMsg_PlayerStatsUpdate{} },
	msgs2.ECstrike15UserMessages_CS_UM_ClientInfo:                   func() proto.Message { return &msgs2.CCSUsrMsg_ClientInfo{} },
	msgs2.ECstrike15UserMessages_CS_UM_XRankGet:                     func() proto.Message { return &msgs2.CCSUsrMsg_XRankGet{} },
	msgs2.ECstrike15UserMessages_CS_UM_XRankUpd:                     func() proto.Message { return &msgs2.CCSUsrMsg_XRankUpd{} },
	msgs2.ECstrike15UserMessages_CS_UM_CallVoteFailed:               func() proto.Message { return &msgs2.CCSUsrMsg_CallVoteFailed{} },
	msgs2.ECstrike15UserMessages_CS_UM_VoteStart:                    func() proto.Message { return &msgs2.CCSUsrMsg_VoteStart{} },
	msgs2.ECstrike15UserMessages_CS_UM_VotePass:                     func() proto.Message { return &msgs2.CCSUsrMsg_VotePass{} },
	msgs2.ECstrike15UserMessages_CS_UM_VoteFailed:                   func() proto.Message { return &msgs2.CCSUsrMsg_VoteFailed{} },
	msgs2.ECstrike15UserMessages_CS_UM_VoteSetup:                    func() proto.Message { return &msgs2.CCSUsrMsg_VoteSetup{} },
	msgs2.ECstrike15UserMessages_CS_UM_ServerRankRevealAll:          func() proto.Message { return &msgs2.CCSUsrMsg_ServerRankRevealAll{} },
	msgs2.ECstrike15UserMessages_CS_UM_SendLastKillerDamageToClient: func() proto.Message { return &msgs2.CCSUsrMsg_SendLastKillerDamageToClient{} },
	msgs2.ECstrike15UserMessages_CS_UM_ServerRankUpdate:             func() proto.Message { return &msgs2.CCSUsrMsg_ServerRankUpdate{} },
	msgs2.ECstrike15UserMessages_CS_UM_ItemPickup:                   func() proto.Message { return &msgs2.CCSUsrMsg_ItemPickup{} },
	msgs2.ECstrike15UserMessages_CS_UM_ShowMenu:                     func() proto.Message { return &msgs2.CCSUsrMsg_ShowMenu{} },
	msgs2.ECstrike15UserMessages_CS_UM_BarTime:                      func() proto.Message { return &msgs2.CCSUsrMsg_BarTime{} },
	msgs2.ECstrike15UserMessages_CS_UM_AmmoDenied:                   func() proto.Message { return &msgs2.CCSUsrMsg_AmmoDenied{} },
	msgs2.ECstrike15UserMessages_CS_UM_MarkAchievement:              func() proto.Message { return &msgs2.CCSUsrMsg_MarkAchievement{} },
	msgs2.ECstrike15UserMessages_CS_UM_MatchStatsUpdate:             func() proto.Message { return &msgs2.CCSUsrMsg_MatchStatsUpdate{} },
	msgs2.ECstrike15UserMessages_CS_UM_ItemDrop:                     func() proto.Message { return &msgs2.CCSUsrMsg_ItemDrop{} },
	msgs2.ECstrike15UserMessages_CS_UM_SendPlayerItemDrops:          func() proto.Message { return &msgs2.CCSUsrMsg_SendPlayerItemDrops{} },
	msgs2.ECstrike15UserMessages_CS_UM_RoundBackupFilenames:         func() proto.Message { return &msgs2.CCSUsrMsg_RoundBackupFilenames{} },
	msgs2.ECstrike15UserMessages_CS_UM_SendPlayerItemFound:          func() proto.Message { return &msgs2.CCSUsrMsg_SendPlayerItemFound{} },
	msgs2.ECstrike15UserMessages_CS_UM_ReportHit:                    func() proto.Message { return &msgs2.CCSUsrMsg_ReportHit{} },
	msgs2.ECstrike15UserMessages_CS_UM_XpUpdate:                     func() proto.Message { return &msgs2.CCSUsrMsg_XpUpdate{} },
	msgs2.ECstrike15UserMessages_CS_UM_QuestProgress:                func() proto.Message { return &msgs2.CCSUsrMsg_QuestProgress{} },
	msgs2.ECstrike15UserMessages_CS_UM_ScoreLeaderboardData:         func() proto.Message { return &msgs2.CCSUsrMsg_ScoreLeaderboardData{} },
	msgs2.ECstrike15UserMessages_CS_UM_PlayerDecalDigitalSignature:  func() proto.Message { return &msgs2.CCSUsrMsg_PlayerDecalDigitalSignature{} },
	msgs2.ECstrike15UserMessages_CS_UM_WeaponSound:                  func() proto.Message { return &msgs2.CCSUsrMsg_WeaponSound{} },
	msgs2.ECstrike15UserMessages_CS_UM_UpdateScreenHealthBar:        func() proto.Message { return &msgs2.CCSUsrMsg_UpdateScreenHealthBar{} },
	msgs2.ECstrike15UserMessages_CS_UM_EntityOutlineHighlight:       func() proto.Message { return &msgs2.CCSUsrMsg_EntityOutlineHighlight{} },
	msgs2.ECstrike15UserMessages_CS_UM_SSUI:                         func() proto.Message { return &msgs2.CCSUsrMsg_SSUI{} },
	msgs2.ECstrike15UserMessages_CS_UM_SurvivalStats:                func() proto.Message { return &msgs2.CCSUsrMsg_SurvivalStats{} },
	msgs2.ECstrike15UserMessages_CS_UM_DisconnectToLobby2:           func() proto.Message { return &msgs2.CCSUsrMsg_DisconnectToLobby{} },
	msgs2.ECstrike15UserMessages_CS_UM_EndOfMatchAllPlayersData:     func() proto.Message { return &msgs2.CCSUsrMsg_EndOfMatchAllPlayersData{} },
	msgs2.ECstrike15UserMessages_CS_UM_PostRoundDamageReport:        func() proto.Message { return &msgs2.CCSUsrMsg_PostRoundDamageReport{} },
	msgs2.ECstrike15UserMessages_CS_UM_RoundEndReportData:           func() proto.Message { return &msgs2.CCSUsrMsg_RoundEndReportData{} },
	msgs2.ECstrike15UserMessages_CS_UM_CurrentRoundOdds:             func() proto.Message { return &msgs2.CCSUsrMsg_CurrentRoundOdds{} },
	msgs2.ECstrike15UserMessages_CS_UM_DeepStats:                    func() proto.Message { return &msgs2.CCSUsrMsg_DeepStats{} },
	msgs2.ECstrike15UserMessages_CS_UM_ShootInfo:                    func() proto.Message { return &msgs2.CCSUsrMsg_ShootInfo{} },
	msgs2.ECstrike15UserMessages_CS_UM_CounterStrafe:                func() proto.Message { return &msgs2.CCSUsrMsg_CounterStrafe{} },
	msgs2.ECstrike15UserMessages_CS_UM_DamagePrediction:             func() proto.Message { return &msgs2.CCSUsrMsg_DamagePrediction{} },
	msgs2.ECstrike15UserMessages_CS_UM_RecurringMissionSchema:       func() proto.Message { return &msgs2.CCSUsrMsg_RecurringMissionSchema{} },
}

var teCreators = map[msgs2.ETEProtobufIds]NetMessageCreator{
	msgs2.ETEProtobufIds_TE_EffectDispatchId: func() proto.Message { return &msgs2.CMsgTEEffectDispatch{} },
	msgs2.ETEProtobufIds_TE_ArmorRicochetId:  func() proto.Message { return &msgs2.CMsgTEArmorRicochet{} },
	msgs2.ETEProtobufIds_TE_BeamEntPointId:   func() proto.Message { return &msgs2.CMsgTEBeamEntPoint{} },
	msgs2.ETEProtobufIds_TE_BeamEntsId:       func() proto.Message { return &msgs2.CMsgTEBeamEnts{} },
	msgs2.ETEProtobufIds_TE_BeamPointsId:     func() proto.Message { return &msgs2.CMsgTEBeamPoints{} },
	msgs2.ETEProtobufIds_TE_BeamRingId:       func() proto.Message { return &msgs2.CMsgTEBeamRing{} },
	msgs2.ETEProtobufIds_TE_BubblesId:        func() proto.Message { return &msgs2.CMsgTEBubbles{} },
	msgs2.ETEProtobufIds_TE_BubbleTrailId:    func() proto.Message { return &msgs2.CMsgTEBubbleTrail{} },
	msgs2.ETEProtobufIds_TE_DecalId:          func() proto.Message { return &msgs2.CMsgTEDecal{} },
	msgs2.ETEProtobufIds_TE_WorldDecalId:     func() proto.Message { return &msgs2.CMsgTEWorldDecal{} },
	msgs2.ETEProtobufIds_TE_EnergySplashId:   func() proto.Message { return &msgs2.CMsgTEEnergySplash{} },
	msgs2.ETEProtobufIds_TE_FizzId:           func() proto.Message { return &msgs2.CMsgTEFizz{} },
	msgs2.ETEProtobufIds_TE_ShatterSurfaceId: func() proto.Message { return &msgs2.CMsgTEShatterSurface{} },
	msgs2.ETEProtobufIds_TE_GlowSpriteId:     func() proto.Message { return &msgs2.CMsgTEGlowSprite{} },
	msgs2.ETEProtobufIds_TE_ImpactId:         func() proto.Message { return &msgs2.CMsgTEImpact{} },
	msgs2.ETEProtobufIds_TE_MuzzleFlashId:    func() proto.Message { return &msgs2.CMsgTEMuzzleFlash{} },
	msgs2.ETEProtobufIds_TE_BloodStreamId:    func() proto.Message { return &msgs2.CMsgTEBloodStream{} },
	msgs2.ETEProtobufIds_TE_ExplosionId:      func() proto.Message { return &msgs2.CMsgTEExplosion{} },
	msgs2.ETEProtobufIds_TE_DustId:           func() proto.Message { return &msgs2.CMsgTEDust{} },
	msgs2.ETEProtobufIds_TE_LargeFunnelId:    func() proto.Message { return &msgs2.CMsgTELargeFunnel{} },
	msgs2.ETEProtobufIds_TE_SparksId:         func() proto.Message { return &msgs2.CMsgTESparks{} },
	msgs2.ETEProtobufIds_TE_PhysicsPropId:    func() proto.Message { return &msgs2.CMsgTEPhysicsProp{} },
	msgs2.ETEProtobufIds_TE_SmokeId:          func() proto.Message { return &msgs2.CMsgTESmoke{} },
}

var bidirectionalMessageCreators = map[msgs2.Bidirectional_Messages]NetMessageCreator{
	msgs2.Bidirectional_Messages_bi_RebroadcastGameEvent: func() proto.Message { return &msgs2.CBidirMsg_RebroadcastGameEvent{} },
	msgs2.Bidirectional_Messages_bi_RebroadcastSource:    func() proto.Message { return &msgs2.CBidirMsg_RebroadcastSource{} },
	msgs2.Bidirectional_Messages_bi_GameEvent:            func() proto.Message { return &msgs2.CBidirMsg_RebroadcastGameEvent{} },
	msgs2.Bidirectional_Messages_bi_PredictionEvent:      func() proto.Message { return &msgs2.CBidirMsg_PredictionEvent{} },
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
		int32(msgs2.NET_Messages_net_Tick),
		int32(msgs2.SVC_Messages_svc_CreateStringTable),
		int32(msgs2.SVC_Messages_svc_UpdateStringTable),
		int32(msgs2.NET_Messages_net_SpawnGroup_Load):
		return -10

	case
		// These messages benefit from having context but may also need to
		// provide context in terms of delta updates.
		int32(msgs2.SVC_Messages_svc_PacketEntities):
		return 5
	}

	return 0
}

func (p *parser) handleDemoPacket(pack *msgs2.CDemoPacket) {
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

	sort.SliceStable(p.pendingMessagesCache, func(i, j int) bool {
		return p.pendingMessagesCache[i].priority() < p.pendingMessagesCache[j].priority()
	})

	for _, m := range p.pendingMessagesCache {
		var msgCreator NetMessageCreator

		if m.t < int32(msgs2.SVC_Messages_svc_ServerInfo) {
			msgCreator = netMsgCreators[msgs2.NET_Messages(m.t)]
			if msgCreator == nil {
				msgCreator = bidirectionalMessageCreators[msgs2.Bidirectional_Messages(m.t)]
			}
		} else if m.t < int32(msgs2.EBaseUserMessages_UM_AchievementEvent) {
			msgCreator = svcMsgCreators[msgs2.SVC_Messages(m.t)]
		} else if m.t < int32(msgs2.EBaseGameEvents_GE_VDebugGameSessionIDEvent) {
			msgCreator = usrMsgCreators[msgs2.EBaseUserMessages(m.t)]

			if msgCreator == nil {
				msgCreator = emCreators[msgs2.EBaseEntityMessages(m.t)]
			}
		} else if m.t < int32(msgs2.ECstrike15UserMessages_CS_UM_VGUIMenu) {
			msgCreator = gameEventCreators[msgs2.EBaseGameEvents(m.t)]
		} else if m.t < int32(msgs2.ETEProtobufIds_TE_EffectDispatchId) {
			msgCreator = csUsrMsgCreators[msgs2.ECstrike15UserMessages(m.t)]
		} else if m.t < int32(msgs2.ECsgoGameEvents_GE_PlayerAnimEventId) {
			msgCreator = teCreators[msgs2.ETEProtobufIds(m.t)]
		} else {
			msgCreator = csgoGameEventCreators[msgs2.ECsgoGameEvents(m.t)]
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

func (p *parser) handleFullPacket(msg *msgs2.CDemoFullPacket) {
	p.handleStringTables(msg.StringTable)

	if msg.Packet.GetData() != nil {
		p.handleDemoPacket(msg.Packet)
	}
}

func (p *parser) handleFileInfo(msg *msgs2.CDemoFileInfo) {
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

func (p *parser) handleDemoFileHeader(msg *msgs2.CDemoFileHeader) {
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

func (p *parser) updatePlayersPreviousFramePosition() {
	for _, player := range p.GameState().Participants().Playing() {
		player.PreviousFramePosition = player.Position()
	}
}
