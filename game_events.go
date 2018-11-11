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
		p.eventDispatcher.Dispatch(events.ParserWarn{Message: "Received GameEvent but event descriptors are missing"})
		return
	}

	d := p.gameEventDescs[ge.Eventid]

	debugGameEvent(d, ge)

	var data map[string]*msg.CSVCMsg_GameEventKeyT

	switch d.Name {
	case "round_start": // Round started
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.RoundStart{
			TimeLimit: int(data["timelimit"].GetValLong()),
			FragLimit: int(data["fraglimit"].GetValLong()),
			Objective: data["objective"].GetValString(),
		})

	case "cs_win_panel_match": // Not sure, maybe match end event???
		p.eventDispatcher.Dispatch(events.AnnouncementWinPanelMatch{})

	case "round_announce_final": // 30th round for normal de_, not necessarily matchpoint
		p.eventDispatcher.Dispatch(events.AnnouncementFinalRound{})

	case "round_announce_last_round_half": // Last round of the half
		p.eventDispatcher.Dispatch(events.AnnouncementLastRoundHalf{})

	case "round_end": // Round ended and the winner was announced
		data = mapGameEventData(d, ge)

		t := common.TeamSpectators

		switch data["winner"].GetValByte() {
		case int32(p.gameState.tState.ID):
			t = common.TeamTerrorists
		case int32(p.gameState.ctState.ID):
			t = common.TeamCounterTerrorists
		}

		p.eventDispatcher.Dispatch(events.RoundEnd{
			Message: data["message"].GetValString(),
			Reason:  events.RoundEndReason(data["reason"].GetValByte()),
			Winner:  t,
		})

	case "round_officially_ended": // The event after which you get teleported to the spawn (=> You can still walk around between round_end and this event)
		// Issue #42
		// Sometimes grenades & infernos aren't deleted / destroyed via entity-updates at the end of the round,
		// so we need to do it here for those that weren't.
		//
		// We're not deleting them from entitites though as that's supposed to be as close to the actual demo data as possible.
		// We're also not using Entity.Destroy() because it would - in some cases - be called twice on the same entity
		// and it's supposed to be called when the demo actually says so (same case as with GameState.entities).
		for _, proj := range p.gameState.grenadeProjectiles {
			p.nadeProjectileDestroyed(proj)
		}

		for _, inf := range p.gameState.infernos {
			p.infernoExpired(inf)
		}

		p.eventDispatcher.Dispatch(events.RoundEndOfficial{})

	case "round_mvp": // Round MVP was announced
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.RoundMVPAnnouncement{
			Player: p.gameState.playersByUserID[int(data["userid"].GetValShort())],
			Reason: events.RoundMVPReason(data["reason"].GetValShort()),
		})

	case "bot_takeover": // Bot got taken over
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.BotTakenOver{Taker: p.gameState.playersByUserID[int(data["userid"].GetValShort())]})

	case "begin_new_match": // Match started
		p.eventDispatcher.Dispatch(events.MatchStart{})

	case "round_freeze_end": // Round start freeze ended
		p.eventDispatcher.Dispatch(events.RoundFreezetimeEnd{})

	case "player_footstep": // Footstep sound
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.Footstep{
			Player: p.gameState.playersByUserID[int(data["userid"].GetValShort())],
		})

	case "player_jump": // Player jumped
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.PlayerJump{Player: p.gameState.playersByUserID[int(data["userid"].GetValShort())]})

	case "weapon_fire": // Weapon was fired
		data = mapGameEventData(d, ge)

		shooter := p.gameState.playersByUserID[int(data["userid"].GetValShort())]
		wepType := common.MapEquipment(data["weapon"].GetValString())

		p.eventDispatcher.Dispatch(events.WeaponFire{
			Shooter: shooter,
			Weapon:  getPlayerWeapon(shooter, wepType),
		})

	case "player_death": // Player died
		data = mapGameEventData(d, ge)

		killer := p.gameState.playersByUserID[int(data["attacker"].GetValShort())]
		wepType := common.MapEquipment(data["weapon"].GetValString())

		p.eventDispatcher.Dispatch(events.Kill{
			Victim:            p.gameState.playersByUserID[int(data["userid"].GetValShort())],
			Killer:            killer,
			Assister:          p.gameState.playersByUserID[int(data["assister"].GetValShort())],
			IsHeadshot:        data["headshot"].GetValBool(),
			PenetratedObjects: int(data["penetrated"].GetValShort()),
			Weapon:            getPlayerWeapon(killer, wepType),
		})

	case "player_hurt": // Player got hurt
		data = mapGameEventData(d, ge)

		attacker := p.gameState.playersByUserID[int(data["attacker"].GetValShort())]
		wepType := common.MapEquipment(data["weapon"].GetValString())

		p.eventDispatcher.Dispatch(events.PlayerHurt{
			Player:       p.gameState.playersByUserID[int(data["userid"].GetValShort())],
			Attacker:     attacker,
			Health:       int(data["health"].GetValByte()),
			Armor:        int(data["armor"].GetValByte()),
			HealthDamage: int(data["dmg_health"].GetValShort()),
			ArmorDamage:  int(data["dmg_armor"].GetValByte()),
			HitGroup:     events.HitGroup(data["hitgroup"].GetValByte()),
			Weapon:       getPlayerWeapon(attacker, wepType),
		})

	case "player_blind": // Player got blinded by a flash
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.PlayerFlashed{Player: p.gameState.playersByUserID[int(data["userid"].GetValShort())]})

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
		thrower := p.gameState.playersByUserID[int(data["userid"].GetValShort())]
		position := r3.Vector{
			X: float64(data["x"].ValFloat),
			Y: float64(data["y"].ValFloat),
			Z: float64(data["z"].ValFloat),
		}
		nadeEntityID := int(data["entityid"].GetValShort())

		switch d.Name {
		case "flashbang_detonate": // Flash exploded
			p.eventDispatcher.Dispatch(events.FlashExplode{GrenadeEvent: buildNadeEvent(common.EqFlash, thrower, position, nadeEntityID)})

		case "hegrenade_detonate": // HE exploded
			p.eventDispatcher.Dispatch(events.HeExplode{GrenadeEvent: buildNadeEvent(common.EqHE, thrower, position, nadeEntityID)})

		case "decoy_started": // Decoy started
			p.eventDispatcher.Dispatch(events.DecoyStart{GrenadeEvent: buildNadeEvent(common.EqDecoy, thrower, position, nadeEntityID)})

		case "decoy_detonate": // Decoy exploded/expired
			p.eventDispatcher.Dispatch(events.DecoyExpired{GrenadeEvent: buildNadeEvent(common.EqDecoy, thrower, position, nadeEntityID)})

		case "smokegrenade_detonate": // Smoke popped
			p.eventDispatcher.Dispatch(events.SmokeStart{GrenadeEvent: buildNadeEvent(common.EqSmoke, thrower, position, nadeEntityID)})

		case "smokegrenade_expired": // Smoke expired
			p.eventDispatcher.Dispatch(events.SmokeExpired{GrenadeEvent: buildNadeEvent(common.EqSmoke, thrower, position, nadeEntityID)})

		case "inferno_startburn": // Incendiary exploded/started
			p.eventDispatcher.Dispatch(events.FireGrenadeStart{GrenadeEvent: buildNadeEvent(common.EqIncendiary, thrower, position, nadeEntityID)})

		case "inferno_expire": // Incendiary expired
			p.eventDispatcher.Dispatch(events.FireGrenadeExpired{GrenadeEvent: buildNadeEvent(common.EqIncendiary, thrower, position, nadeEntityID)})
		}

	case "player_connect": // Bot connected or player reconnected, players normally come in via string tables & data tables
		data = mapGameEventData(d, ge)

		pl := &playerInfo{
			userID: int(data["userid"].GetValShort()),
			name:   data["name"].GetValString(),
			guid:   data["networkid"].GetValString(),
		}

		pl.xuid = getCommunityID(pl.guid)

		p.rawPlayers[int(data["index"].GetValByte())] = pl

	case "player_disconnect": // Player disconnected (kicked, quit, timed out etc.)
		data = mapGameEventData(d, ge)

		uid := int(data["userid"].GetValShort())

		for k, v := range p.rawPlayers {
			if v.userID == uid {
				delete(p.rawPlayers, k)
			}
		}

		pl := p.gameState.playersByUserID[uid]
		if pl != nil {
			e := events.PlayerDisconnected{
				Player: pl,
			}
			p.eventDispatcher.Dispatch(e)
		}

		delete(p.gameState.playersByUserID, uid)

	case "player_team": // Player changed team
		data = mapGameEventData(d, ge)

		player := p.gameState.playersByUserID[int(data["userid"].GetValShort())]
		newTeam := common.Team(data["team"].GetValByte())

		if player != nil {
			if player.Team != newTeam {
				player.Team = newTeam

				p.eventDispatcher.Dispatch(events.PlayerTeamChange{
					Player:  player,
					IsBot:   data["isbot"].GetValBool(),
					Silent:  data["silent"].GetValBool(),
					NewTeam: newTeam,
					OldTeam: common.Team(data["oldteam"].GetValByte()),
				})
			} else {
				p.eventDispatcher.Dispatch(events.ParserWarn{
					Message: "Player team swap game-event occurred but player.Team == newTeam",
				})
			}
		} else {
			p.eventDispatcher.Dispatch(events.ParserWarn{
				Message: "Player team swap game-event occurred but player is nil",
			})
		}

	case "bomb_beginplant": // Plant started
		fallthrough
	case "bomb_planted": // Plant finished
		fallthrough
	case "bomb_defused": // Defuse finished
		fallthrough
	case "bomb_exploded": // Bomb exploded
		data = mapGameEventData(d, ge)

		e := events.BombEvent{Player: p.gameState.playersByUserID[int(data["userid"].GetValShort())]}

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
			p.eventDispatcher.Dispatch(events.BombPlantBegin{BombEvent: e})
		case "bomb_planted":
			p.eventDispatcher.Dispatch(events.BombPlanted{BombEvent: e})
		case "bomb_defused":
			p.eventDispatcher.Dispatch(events.BombDefused{BombEvent: e})
		case "bomb_exploded":
			p.eventDispatcher.Dispatch(events.BombExplode{BombEvent: e})
		}

	case "bomb_begindefuse": // Defuse started
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.BombDefuseStart{
			Player: p.gameState.playersByUserID[int(data["userid"].GetValShort())],
			HasKit: data["haskit"].GetValBool(),
		})

	case "item_equip": // Equipped, I think
		fallthrough
	case "item_pickup": // Picked up or bought?
		fallthrough
	case "item_remove": // Dropped?
		data = mapGameEventData(d, ge)
		player := p.gameState.playersByUserID[int(data["userid"].GetValShort())]

		wepType := common.MapEquipment(data["item"].GetValString())
		weapon := common.NewEquipment(wepType)

		switch d.Name {
		case "item_equip":
			p.eventDispatcher.Dispatch(events.ItemEquip{
				Player: player,
				Weapon: weapon,
			})
		case "item_pickup":
			p.eventDispatcher.Dispatch(events.ItemPickup{
				Player: player,
				Weapon: weapon,
			})
		case "item_remove":
			p.eventDispatcher.Dispatch(events.ItemDrop{
				Player: player,
				Weapon: weapon,
			})
		}

	case "bomb_dropped": // Bomb dropped
		player := p.gameState.playersByUserID[int(data["userid"].GetValShort())]
		entityID := int(data["entityid"].GetValShort())

		p.eventDispatcher.Dispatch(events.BombDropped{
			Player:   player,
			EntityID: entityID,
		})

	case "bomb_pickup": // Bomb picked up
		p.eventDispatcher.Dispatch(events.BombPickup{
			Player: p.gameState.playersByUserID[int(data["userid"].GetValShort())],
		})

	// TODO: Might be interesting:
	case "player_connect_full": // Connecting finished
		fallthrough
	case "player_falldamage": // Falldamage
		fallthrough
	case "weapon_zoom": // Zooming in
		fallthrough
	case "weapon_reload": // Weapon reloaded
		fallthrough
	case "round_time_warning": // Round time warning
		fallthrough
	case "round_announce_match_point": // Match point announcement
		fallthrough
	case "player_changename": // Name change
		fallthrough

	// Probably not that interesting but we'll still emit the GenericGameEvent:
	case "buytime_ended": // Not actually end of buy time, seems to only be sent once per game at the start
		fallthrough
	case "round_announce_match_start": // Special match start announcement
		fallthrough
	case "bomb_beep": // Bomb beep
		fallthrough
	case "player_spawn": // Player spawn
		fallthrough
	case "hltv_status": // Don't know
		fallthrough
	case "hltv_chase": // Don't care
		fallthrough
	case "cs_round_start_beep": // Round start beeps
		fallthrough
	case "cs_round_final_beep": // Final beep
		fallthrough
	case "cs_pre_restart": // Not sure, doesn't seem to be important
		fallthrough
	case "round_prestart": // Ditto
		fallthrough
	case "round_poststart": // Ditto
		fallthrough
	case "cs_win_panel_round": // Win panel, (==end of match?)
		fallthrough
	case "endmatch_cmm_start_reveal_items": // Drops
		fallthrough
	case "announce_phase_end": // Dunno
		fallthrough
	case "tournament_reward": // Dunno
		fallthrough
	case "other_death": // Dunno
		fallthrough
	case "round_announce_warmup": // Dunno
		fallthrough
	case "server_cvar": // Dunno
		fallthrough
	case "weapon_fire_on_empty": // Sounds boring
		fallthrough
	case "hltv_fixed": // Dunno
		fallthrough
	case "cs_match_end_restart": // Yawn
		p.eventDispatcher.Dispatch(events.GenericGameEvent{Name: d.Name, Data: mapGameEventData(d, ge)})

	default:
		p.eventDispatcher.Dispatch(events.ParserWarn{Message: fmt.Sprintf("Unknown event %q", d.Name)})
		p.eventDispatcher.Dispatch(events.GenericGameEvent{Name: d.Name, Data: mapGameEventData(d, ge)})
	}
}

// Returns the players instance of the weapon if applicable or a new instance otherwise.
func getPlayerWeapon(player *common.Player, wepType common.EquipmentElement) *common.Equipment {
	class := wepType.Class()
	isSpecialWeapon := class == common.EqClassGrenade || (class == common.EqClassEquipment && wepType != common.EqKnife)
	if !isSpecialWeapon && player != nil {
		for _, wep := range player.Weapons() {
			if wep.Weapon == wepType {
				return wep
			}
		}
	}

	wep := common.NewEquipment(wepType)
	return &wep
}

func mapGameEventData(d *msg.CSVCMsg_GameEventListDescriptorT, e *msg.CSVCMsg_GameEvent) map[string]*msg.CSVCMsg_GameEventKeyT {
	data := make(map[string]*msg.CSVCMsg_GameEventKeyT)
	for i, k := range d.Keys {
		data[k.Name] = e.Keys[i]
	}
	return data
}

// Just so we can nicely create GrenadeEvents in one line
func buildNadeEvent(nadeType common.EquipmentElement, thrower *common.Player, position r3.Vector, nadeEntityID int) events.GrenadeEvent {
	return events.GrenadeEvent{
		GrenadeType:     nadeType,
		Thrower:         thrower,
		Position:        position,
		GrenadeEntityID: nadeEntityID,
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
		if err != nil {
			p.eventDispatcher.Dispatch(events.ParserWarn{Message: fmt.Sprintf("Failed to decode SayText message: %s", err.Error())})
		}

		p.eventDispatcher.Dispatch(events.SayText{
			EntIdx:    int(st.EntIdx),
			IsChat:    st.Chat,
			IsChatAll: st.Textallchat,
			Text:      st.Text,
		})

	case msg.ECstrike15UserMessages_CS_UM_SayText2:
		st := new(msg.CCSUsrMsg_SayText2)
		err := st.Unmarshal(um.MsgData)
		if err != nil {
			p.eventDispatcher.Dispatch(events.ParserWarn{Message: fmt.Sprintf("Failed to decode SayText2 message: %s", err.Error())})
		}

		p.eventDispatcher.Dispatch(events.SayText2{
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
			var sender *common.Player
			for _, pl := range p.gameState.playersByUserID {
				// This could be a problem if the player changed his name
				// as the name is only set initially and never updated
				if pl.Name == st.Params[0] {
					sender = pl
				}
			}

			p.eventDispatcher.Dispatch(events.ChatMessage{
				Sender:    sender,
				Text:      st.Params[1],
				IsChatAll: st.Textallchat,
			})

		case "#CSGO_Coach_Join_T": // Ignore these
		case "#CSGO_Coach_Join_CT":

		default:
			p.eventDispatcher.Dispatch(events.ParserWarn{Message: fmt.Sprintf("Skipped sending ChatMessageEvent for SayText2 with unknown MsgName %q", st.MsgName)})
		}

	default:
		// TODO: handle more user messages (if they are interesting)
		// Maybe msg.ECstrike15UserMessages_CS_UM_RadioText
	}
}
