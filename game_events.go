package demoinfocs

import (
	"fmt"
	"strconv"

	r3 "github.com/golang/geo/r3"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

func (p *Parser) handleGameEventList(gel *msg.CSVCMsg_GameEventList) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	p.gameEventDescs = make(map[int32]*msg.CSVCMsg_GameEventListDescriptorT)
	for _, d := range gel.GetDescriptors() {
		p.gameEventDescs[d.GetEventid()] = d
	}
}

func (p *Parser) handleGameEvent(ge *msg.CSVCMsg_GameEvent) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	if p.gameEventDescs == nil {
		p.warn("Received GameEvent but event descriptors are missing")
		return
	}

	d := p.gameEventDescs[ge.Eventid]

	debugGameEvent(d, ge)

	// Ignore events before players are connected to speed things up
	if len(p.gameState.players) == 0 && d.Name != "player_connect" {
		return
	}

	var data map[string]*msg.CSVCMsg_GameEventKeyT

	switch d.Name {
	case "round_start": // Round started
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.RoundStartedEvent{
			TimeLimit: int(data["timelimit"].GetValLong()),
			FragLimit: int(data["fraglimit"].GetValLong()),
			Objective: data["objective"].GetValString(),
		})

	case "cs_win_panel_match": // Not sure, maybe match end event???
		p.eventDispatcher.Dispatch(events.WinPanelMatchEvent{})

	case "round_announce_final": // 30th round for normal de_, not necessarily matchpoint
		p.eventDispatcher.Dispatch(events.FinalRoundEvent{})

	case "round_announce_last_round_half": // Last round of the half
		p.eventDispatcher.Dispatch(events.LastRoundHalfEvent{})

	case "round_end": // Round ended and the winner was announced
		data = mapGameEventData(d, ge)

		t := common.TeamSpectators

		switch data["winner"].GetValByte() {
		case int32(p.gameState.tState.id):
			t = common.TeamTerrorists
		case int32(p.gameState.ctState.id):
			t = common.TeamCounterTerrorists
		}

		p.eventDispatcher.Dispatch(events.RoundEndedEvent{
			Message: data["message"].GetValString(),
			Reason:  common.RoundEndReason(data["reason"].GetValByte()),
			Winner:  t,
		})

	case "round_officially_ended": // Round ended. . . probably the event where you get teleported to the spawn (=> You can still walk around between round_end and this?)
		p.eventDispatcher.Dispatch(events.RoundOfficialyEndedEvent{})

	case "round_mvp": // Round MVP was announced
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.RoundMVPEvent{
			Player: p.gameState.players[int(data["userid"].GetValShort())],
			Reason: common.RoundMVPReason(data["reason"].GetValShort()),
		})

	case "bot_takeover": // Bot got taken over
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.BotTakenOverEvent{Taker: p.gameState.players[int(data["userid"].GetValShort())]})

	case "round_announce_match_start": // Special match start announcement
		fallthrough
	case "begin_new_match": // Match started
		p.eventDispatcher.Dispatch(events.MatchStartedEvent{})

	case "round_freeze_end": // Round start freeze ended
		p.eventDispatcher.Dispatch(events.FreezetimeEndedEvent{})

	case "player_footstep": // Footstep sound
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.PlayerFootstepEvent{
			Player: p.gameState.players[int(data["userid"].GetValShort())],
		})

	case "player_jump": // Player jumped
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.PlayerJumpEvent{Player: p.gameState.players[int(data["userid"].GetValShort())]})

	case "weapon_fire": // Weapon was fired
		data = mapGameEventData(d, ge)

		e := events.WeaponFiredEvent{Shooter: p.gameState.players[int(data["userid"].GetValShort())]}
		wep := common.NewEquipment(data["weapon"].GetValString())

		if e.Shooter != nil && wep.Class() != common.EqClassGrenade {
			e.Weapon = e.Shooter.ActiveWeapon()
		} else {
			e.Weapon = &wep
		}

		p.eventDispatcher.Dispatch(e)

	case "player_death": // Player died
		data = mapGameEventData(d, ge)

		e := events.PlayerKilledEvent{
			Victim:            p.gameState.players[int(data["userid"].GetValShort())],
			Killer:            p.gameState.players[int(data["attacker"].GetValShort())],
			Assister:          p.gameState.players[int(data["assister"].GetValShort())],
			IsHeadshot:        data["headshot"].GetValBool(),
			PenetratedObjects: int(data["penetrated"].GetValShort()),
		}

		wep := common.NewSkinEquipment(data["weapon"].GetValString(), data["weapon_itemid"].GetValString())

		// FIXME: Should we do that last weapons > 0 check above as well?????
		if e.Killer != nil && wep.Class() != common.EqClassGrenade && len(e.Killer.Weapons) > 0 {
			e.Weapon = e.Killer.ActiveWeapon()
		} else {
			e.Weapon = &wep
		}

		p.eventDispatcher.Dispatch(e)

	case "player_hurt": // Player got hurt
		data = mapGameEventData(d, ge)

		e := events.PlayerHurtEvent{
			Player:       p.gameState.players[int(data["userid"].GetValShort())],
			Attacker:     p.gameState.players[int(data["attacker"].GetValShort())],
			Health:       int(data["health"].GetValByte()),
			Armor:        int(data["armor"].GetValByte()),
			HealthDamage: int(data["dmg_health"].GetValShort()),
			ArmorDamage:  int(data["dmg_armor"].GetValByte()),
			HitGroup:     common.HitGroup(data["hitgroup"].GetValByte()),
		}

		wep := common.NewEquipment(data["weapon"].GetValString())

		if e.Attacker != nil && wep.Class() != common.EqClassGrenade && len(e.Attacker.Weapons) > 0 {
			e.Weapon = e.Attacker.ActiveWeapon()
		} else {
			e.Weapon = &wep
		}

		p.eventDispatcher.Dispatch(e)

	case "player_blind": // Player got blinded by a flash
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.PlayerFlashedEvent{Player: p.gameState.players[int(data["userid"].GetValShort())]})

	case "flashbang_detonate": // Flash exploded
		fallthrough
	case "hegrenade_detonate": // HE exploded
		fallthrough
	case "decoy_started": // Decoy started
		fallthrough
	case "decoy_detonate": // Decoy exploded/expired
		fallthrough
	case "smokegrenade_detonate": // Smoke popped
		fallthrough
	case "smokegrenade_expired": // Smoke expired
		fallthrough
	case "inferno_startburn": // Incendiary exploded/started
		fallthrough
	case "inferno_expire": // Incendiary expired
		data = mapGameEventData(d, ge)
		thrower := p.gameState.players[int(data["userid"].GetValShort())]
		position := r3.Vector{
			X: float64(data["x"].ValFloat),
			Y: float64(data["y"].ValFloat),
			Z: float64(data["z"].ValFloat),
		}

		switch d.Name {
		case "flashbang_detonate": // Flash exploded
			p.eventDispatcher.Dispatch(events.FlashExplodedEvent{NadeEvent: buildNadeEvent(common.EqFlash, thrower, position)})

		case "hegrenade_detonate": // HE exploded
			p.eventDispatcher.Dispatch(events.HeExplodedEvent{NadeEvent: buildNadeEvent(common.EqHE, thrower, position)})

		case "decoy_started": // Decoy started
			p.eventDispatcher.Dispatch(events.DecoyStartEvent{NadeEvent: buildNadeEvent(common.EqDecoy, thrower, position)})

		case "decoy_detonate": // Decoy exploded/expired
			p.eventDispatcher.Dispatch(events.DecoyEndEvent{NadeEvent: buildNadeEvent(common.EqDecoy, thrower, position)})

		case "smokegrenade_detonate": // Smoke popped
			p.eventDispatcher.Dispatch(events.SmokeStartEvent{NadeEvent: buildNadeEvent(common.EqSmoke, thrower, position)})

		case "smokegrenade_expired": // Smoke expired
			p.eventDispatcher.Dispatch(events.SmokeEndEvent{NadeEvent: buildNadeEvent(common.EqSmoke, thrower, position)})

		case "inferno_startburn": // Incendiary exploded/started
			p.eventDispatcher.Dispatch(events.FireNadeStartEvent{NadeEvent: buildNadeEvent(common.EqIncendiary, thrower, position)})

		case "inferno_expire": // Incendiary expired
			p.eventDispatcher.Dispatch(events.FireNadeEndEvent{NadeEvent: buildNadeEvent(common.EqIncendiary, thrower, position)})
		}

	case "player_connect": // Bot connected, players come in via string tables & data tables
		data = mapGameEventData(d, ge)

		pl := &playerInfo{
			userID: int(data["userid"].GetValShort()),
			name:   data["name"].GetValString(),
			guid:   data["networkid"].GetValString(),
		}

		pl.xuid = getCommunityID(pl.guid)

		p.rawPlayers[int(data["index"].GetValShort())] = pl

	case "player_disconnect": // Bot disconnected, players come in via string tables & data tables
		data = mapGameEventData(d, ge)

		uid := int(data["userid"].GetValShort())

		for i := range p.rawPlayers {
			if p.rawPlayers[i].userID == uid {
				delete(p.rawPlayers, i)
			}
		}

		pl := p.gameState.players[uid]
		if pl != nil {
			e := events.PlayerDisconnectEvent{
				Player: pl,
			}
			p.eventDispatcher.Dispatch(e)
		}

		delete(p.gameState.players, uid)

	case "player_team": // Player changed team
		data = mapGameEventData(d, ge)

		e := events.PlayerTeamChangeEvent{
			Player: p.gameState.players[int(data["userid"].GetValShort())],
			IsBot:  data["isbot"].GetValBool(),
			Silent: data["silent"].GetValBool(),
		}

		// FIXME: We could probably just cast team & oldteam to common.Team, should always be correct. . . Needs testing
		switch data["team"].GetValByte() {
		case int32(p.gameState.tState.id):
			e.NewTeam = common.TeamTerrorists
		case int32(p.gameState.ctState.id):
			e.NewTeam = common.TeamCounterTerrorists
		default:
			e.NewTeam = common.TeamSpectators
		}

		switch data["oldteam"].GetValByte() {
		case int32(p.gameState.tState.id):
			e.OldTeam = common.TeamTerrorists
		case int32(p.gameState.ctState.id):
			e.OldTeam = common.TeamCounterTerrorists
		default:
			e.OldTeam = common.TeamSpectators
		}

		p.eventDispatcher.Dispatch(e)

	case "bomb_beginplant": // Plant started
		fallthrough
	case "bomb_planted": // Plant finished
		fallthrough
	case "bomb_defused": // Defuse finished
		fallthrough
	case "bomb_exploded": // Bomb exploded
		data = mapGameEventData(d, ge)

		e := events.BombEvent{Player: p.gameState.players[int(data["userid"].GetValShort())]}

		site := int(data["site"].GetValShort())

		switch site {
		case p.bombsiteA.index:
			e.Site = events.BombsiteA
		case p.bombsiteB.index:
			e.Site = events.BombsiteB
		default:
			t := p.triggers[site]

			if t == nil {
				panic(fmt.Sprintf("Bombsite with index %d not found", site))
			}

			if t.contains(p.bombsiteA.center) {
				e.Site = events.BombsiteA
				p.bombsiteA.index = site
			} else if t.contains(p.bombsiteB.center) {
				e.Site = events.BombsiteB
				p.bombsiteB.index = site
			} else {
				panic("Bomb not planted on bombsite A or B")
			}
		}

		switch d.Name {
		case "bomb_beginplant":
			p.eventDispatcher.Dispatch(events.BombBeginPlant{BombEvent: e})
		case "bomb_planted":
			p.eventDispatcher.Dispatch(events.BombPlantedEvent{BombEvent: e})
		case "bomb_defused":
			p.eventDispatcher.Dispatch(events.BombDefusedEvent{BombEvent: e})
		case "bomb_exploded":
			p.eventDispatcher.Dispatch(events.BombExplodedEvent{BombEvent: e})
		}

	case "bomb_begindefuse": // Defuse started
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.BombBeginDefuseEvent{
			Player: p.gameState.players[int(data["userid"].GetValShort())],
			HasKit: data["haskit"].GetValBool(),
		})

	case "item_equip": // Equipped, I think
		fallthrough
	case "item_pickup": // Picked up or bought?
		fallthrough
	case "item_remove": // Dropped?
		data = mapGameEventData(d, ge)
		player := p.gameState.players[int(data["userid"].GetValShort())]
		weapon := common.NewSkinEquipment(data["item"].GetValString(), "")

		switch d.Name {
		case "item_equip":
			p.eventDispatcher.Dispatch(events.ItemEquipEvent{
				Player: player,
				Weapon: weapon,
			})
		case "item_pickup":
			p.eventDispatcher.Dispatch(events.ItemPickupEvent{
				Player: player,
				Weapon: weapon,
			})
		case "item_remove":
			p.eventDispatcher.Dispatch(events.ItemDropEvent{
				Player: player,
				Weapon: weapon,
			})
		}

	// TODO: Might be interesting:
	case "player_connect_full": // Connecting finished
	case "player_falldamage": // Falldamage
	case "weapon_zoom": // Zooming in
	case "weapon_reload": // Weapon reloaded
	case "bomb_dropped": // Bomb dropped
	case "bomb_pickup": // Bomb picked up
	case "round_time_warning": // Round time warning
	case "round_announce_match_point": // Match point announcement
	case "player_changename": // Name change

	// Probably not that interesting:
	case "buytime_ended": // Not actually end of buy time, seems to only be sent once per game at the start
	case "bomb_beep": // Bomb beep
	case "player_spawn": // Player spawn
	case "hltv_status": // Don't know
	case "hltv_chase": // Don't care
	case "cs_round_start_beep": // Round start beeps
	case "cs_round_final_beep": // Final beep
	case "cs_pre_restart": // Not sure, doesn't seem to be important
	case "round_prestart": // Ditto
	case "round_poststart": // Ditto
	case "cs_win_panel_round": // Win panel, (==end of match?)
	case "endmatch_cmm_start_reveal_items": // Drops
	case "announce_phase_end": // Dunno
	case "tournament_reward": // Dunno
	case "other_death": // Dunno
	case "round_announce_warmup": // Dunno
	case "server_cvar": // Dunno
	case "weapon_fire_on_empty": // Sounds boring
	case "hltv_fixed": // Dunno
	case "cs_match_end_restart": // Yawn

	default:
		if p.warn != nil {
			p.warn(fmt.Sprintf("Unknown event %q", d.Name))
		}
	}
}

func mapGameEventData(d *msg.CSVCMsg_GameEventListDescriptorT, e *msg.CSVCMsg_GameEvent) map[string]*msg.CSVCMsg_GameEventKeyT {
	data := make(map[string]*msg.CSVCMsg_GameEventKeyT)
	for i, k := range d.Keys {
		data[k.Name] = e.Keys[i]
	}
	return data
}

// Just so we can nicely create NadeEvents in one line
func buildNadeEvent(nadeType common.EquipmentElement, thrower *common.Player, position r3.Vector) events.NadeEvent {
	return events.NadeEvent{
		NadeType: nadeType,
		Thrower:  thrower,
		Position: position,
	}
}

// We're all better off not asking questions
const valveMagicNumber = 76561197960265728

func getCommunityID(guid string) int64 {
	if guid == "BOT" {
		return 0
	}

	authSrv, errSrv := strconv.ParseInt(guid[8:9], 10, 64)
	authID, errID := strconv.ParseInt(guid[10:], 10, 64)

	if errSrv != nil {
		panic(errSrv.Error())
	}
	if errID != nil {
		panic(errID.Error())
	}

	// WTF are we doing here?
	return valveMagicNumber + authID*2 + authSrv
}

func (p *Parser) handleUserMessage(um *msg.CSVCMsg_UserMessage) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	switch msg.ECstrike15UserMessages(um.MsgType) {
	case msg.ECstrike15UserMessages_CS_UM_SayText:
		st := new(msg.CCSUsrMsg_SayText)
		err := st.Unmarshal(um.MsgData)
		if err != nil && p.warn != nil {
			// Just send a warning, chat messages aren't that important
			p.warn(fmt.Sprintf("Failed to decode SayText message: %s", err.Error()))
		}

		p.eventDispatcher.Dispatch(events.SayTextEvent{
			EntityIndex: int(st.EntIdx),
			IsChat:      st.Chat,
			IsChatAll:   st.Textallchat,
			Text:        st.Text,
		})

	case msg.ECstrike15UserMessages_CS_UM_SayText2:
		st := new(msg.CCSUsrMsg_SayText2)
		err := st.Unmarshal(um.MsgData)
		if err != nil && p.warn != nil {
			p.warn(fmt.Sprintf("Failed to decode SayText2 message: %s", err.Error()))
		}

		sender := p.gameState.players[int(st.EntIdx)]
		p.eventDispatcher.Dispatch(events.SayText2Event{
			Sender:    sender,
			IsChat:    st.Chat,
			IsChatAll: st.Textallchat,
			MsgName:   st.MsgName,
			Params:    st.Params,
		})

		switch st.MsgName {
		case "Cstrike_Chat_All":
			fallthrough
		case "Cstrike_Chat_AllDead":
			p.eventDispatcher.Dispatch(events.ChatMessageEvent{
				Sender:    sender,
				Text:      st.Params[1],
				IsChatAll: st.Textallchat,
			})

		case "#CSGO_Coach_Join_T": // Ignore these
		case "#CSGO_Coach_Join_CT":

		default:
			if p.warn != nil {
				p.warn(fmt.Sprintf("Skipped sending ChatMessageEvent for SayText2 with unknown MsgName %q", st.MsgName))
			}
		}

	default:
		// TODO: handle more user messages (if they are interesting)
		// Maybe msg.ECstrike15UserMessages_CS_UM_RadioText
	}
}
