package demoinfocs

import (
	"fmt"
	"strconv"

	"github.com/golang/geo/r3"

	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

func (p *Parser) handleGameEventList(gel *msg.CSVCMsg_GameEventList) {
	p.gameEventDescs = make(map[int32]*msg.CSVCMsg_GameEventListDescriptorT)
	for _, d := range gel.GetDescriptors() {
		p.gameEventDescs[d.GetEventid()] = d
	}
}

func (p *Parser) handleGameEvent(ge *msg.CSVCMsg_GameEvent) {
	if p.gameEventDescs == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{Message: "Received GameEvent but event descriptors are missing"})
		return
	}

	desc := p.gameEventDescs[ge.Eventid]

	debugGameEvent(desc, ge)

	p.gameEventHandler.handler(desc.Name)(desc, ge)
}

type gameEventHandler struct {
	parser                 *Parser
	gameEventNameToHandler map[string]gameEventHandlerFunc
}

func (geh gameEventHandler) handler(eventName string) gameEventHandlerFunc {
	if handler, eventKnown := geh.gameEventNameToHandler[eventName]; eventKnown {
		return handler
	}
	return geh.unknownEvent
}

func (geh gameEventHandler) dispatch(event interface{}) {
	geh.parser.eventDispatcher.Dispatch(event)
}

func (geh gameEventHandler) gameState() *GameState {
	return geh.parser.gameState
}

func (geh gameEventHandler) playerByUserID(userID int) *common.Player {
	return geh.gameState().playersByUserID[userID]
}

func (geh gameEventHandler) playerByUserID32(userID int32) *common.Player {
	return geh.playerByUserID(int(userID))
}

type gameEventHandlerFunc func(*msg.CSVCMsg_GameEventListDescriptorT, *msg.CSVCMsg_GameEvent)

func newGameEventHandler(parser *Parser) gameEventHandler {
	geh := gameEventHandler{parser: parser}

	geh.gameEventNameToHandler = map[string]gameEventHandlerFunc{
		"round_start":                     geh.roundStart,                 // Round started
		"cs_win_panel_match":              geh.csWinPanelMatch,            // Not sure, maybe match end event???
		"round_announce_final":            geh.roundAnnounceFinal,         // 30th round for normal de_, not necessarily matchpoint
		"round_announce_last_round_half":  geh.roundAnnounceLastRoundHalf, // Last round of the half
		"round_end":                       geh.roundEnd,                   // Round ended and the winner was announced
		"round_officially_ended":          geh.roundOfficiallyEnded,       // The event after which you get teleported to the spawn (=> You can still walk around between round_end and this event)
		"round_mvp":                       geh.roundMVP,                   // Round MVP was announced
		"bot_takeover":                    geh.botTakeover,                // Bot got taken over
		"begin_new_match":                 geh.beginNewMatch,              // Match started
		"round_freeze_end":                geh.roundFreezeEnd,             // Round start freeze ended
		"player_footstep":                 geh.playerFootstep,             // Footstep sound
		"player_jump":                     geh.playerJump,                 // Player jumped
		"weapon_fire":                     geh.weaponFire,                 // Weapon was fired
		"player_death":                    geh.playerDeath,                // Player died
		"player_hurt":                     geh.playerHurt,                 // Player got hurt
		"player_blind":                    geh.playerBlind,                // Player got blinded by a flash
		"flashbang_detonate":              geh.flashbangDetonate,          // Flash exploded
		"hegrenade_detonate":              geh.hegranadeDetonate,          // HE exploded
		"decoy_started":                   geh.decoyStarted,               // Decoy started
		"decoy_detonate":                  geh.decoyDetonate,              // Decoy exploded/expired
		"smokegrenade_detonate":           geh.smokegrenadeDetonate,       // Smoke popped
		"smokegrenade_expired":            geh.smokegrenadeExpired,        // Smoke expired
		"inferno_startburn":               geh.infernoStartburn,           // Incendiary exploded/started
		"inferno_expire":                  geh.infernoExpire,              // Incendiary expired
		"player_connect":                  geh.playerConnect,              // Bot connected or player reconnected, players normally come in via string tables & data tables
		"player_disconnect":               geh.playerDisconnect,           // Player disconnected (kicked, quit, timed out etc.)
		"player_team":                     geh.playerTeam,                 // Player changed team
		"bomb_beginplant":                 geh.bombBeginplant,             // Plant started
		"bomb_planted":                    geh.bombPlanted,                // Plant finished
		"bomb_defused":                    geh.bombDefused,                // Defuse finished
		"bomb_exploded":                   geh.bombExploded,               // Bomb exploded
		"bomb_begindefuse":                geh.bombBegindefuse,            // Defuse started
		"item_equip":                      geh.itemEquip,                  // Equipped, I think
		"item_pickup":                     geh.itemPickup,                 // Picked up or bought?
		"item_remove":                     geh.itemRemove,                 // Dropped?
		"bomb_dropped":                    geh.bombDropped,                // Bomb dropped
		"bomb_pickup":                     geh.bombPickup,                 // Bomb picked up
		"player_connect_full":             geh.genericEvent,               // Connecting finished
		"player_falldamage":               geh.genericEvent,               // Falldamage
		"weapon_zoom":                     geh.genericEvent,               // Zooming in
		"weapon_reload":                   geh.genericEvent,               // Weapon reloaded
		"round_time_warning":              geh.genericEvent,               // Round time warning
		"round_announce_match_point":      geh.genericEvent,               // Match point announcement
		"player_changename":               geh.genericEvent,               // Name change
		"buytime_ended":                   geh.genericEvent,               // Not actually end of buy time, seems to only be sent once per game at the start
		"round_announce_match_start":      geh.genericEvent,               // Special match start announcement
		"bomb_beep":                       geh.genericEvent,               // Bomb beep
		"player_spawn":                    geh.genericEvent,               // Player spawn
		"hltv_status":                     geh.genericEvent,               // Don't know
		"hltv_chase":                      geh.genericEvent,               // Don't care
		"cs_round_start_beep":             geh.genericEvent,               // Round start beeps
		"cs_round_final_beep":             geh.genericEvent,               // Final beep
		"cs_pre_restart":                  geh.genericEvent,               // Not sure, doesn't seem to be important
		"round_prestart":                  geh.genericEvent,               // Ditto
		"round_poststart":                 geh.genericEvent,               // Ditto
		"cs_win_panel_round":              geh.genericEvent,               // Win panel, (==end of match?)
		"endmatch_cmm_start_reveal_items": geh.genericEvent,               // Drops
		"announce_phase_end":              geh.genericEvent,               // Dunno
		"tournament_reward":               geh.genericEvent,               // Dunno
		"other_death":                     geh.genericEvent,               // Dunno
		"round_announce_warmup":           geh.genericEvent,               // Dunno
		"server_cvar":                     geh.genericEvent,               // Dunno
		"weapon_fire_on_empty":            geh.genericEvent,               // Sounds boring
		"hltv_fixed":                      geh.genericEvent,               // Dunno
		"cs_match_end_restart":            geh.genericEvent,               // Yawn
	}

	return geh
}

func (geh gameEventHandler) roundStart(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)
	geh.dispatch(events.RoundStart{
		TimeLimit: int(data["timelimit"].GetValLong()),
		FragLimit: int(data["fraglimit"].GetValLong()),
		Objective: data["objective"].GetValString(),
	})
}

func (geh gameEventHandler) csWinPanelMatch(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.AnnouncementWinPanelMatch{})
}

func (geh gameEventHandler) roundAnnounceFinal(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.AnnouncementFinalRound{})
}

func (geh gameEventHandler) roundAnnounceLastRoundHalf(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.AnnouncementLastRoundHalf{})
}

func (geh gameEventHandler) roundEnd(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	winner := common.Team(data["winner"].ValByte)
	winnerState := geh.gameState().Team(winner)
	var loserState *common.TeamState
	if winnerState != nil {
		loserState = winnerState.Opponent
	}

	geh.dispatch(events.RoundEnd{
		Message:     data["message"].GetValString(),
		Reason:      events.RoundEndReason(data["reason"].GetValByte()),
		Winner:      winner,
		WinnerState: winnerState,
		LoserState:  loserState,
	})
}

func (geh gameEventHandler) roundOfficiallyEnded(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	// Issue #42
	// Sometimes grenades & infernos aren't deleted / destroyed via entity-updates at the end of the round,
	// so we need to do it here for those that weren't.
	//
	// We're not deleting them from entitites though as that's supposed to be as close to the actual demo data as possible.
	// We're also not using Entity.Destroy() because it would - in some cases - be called twice on the same entity
	// and it's supposed to be called when the demo actually says so (same case as with GameState.entities).
	for _, proj := range geh.gameState().grenadeProjectiles {
		geh.parser.nadeProjectileDestroyed(proj)
	}

	for _, inf := range geh.gameState().infernos {
		geh.parser.infernoExpired(inf)
	}

	geh.dispatch(events.RoundEndOfficial{})
}

func (geh gameEventHandler) roundMVP(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	geh.dispatch(events.RoundMVPAnnouncement{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
		Reason: events.RoundMVPReason(data["reason"].GetValShort()),
	})
}

func (geh gameEventHandler) botTakeover(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	geh.dispatch(events.BotTakenOver{
		Taker: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

func (geh gameEventHandler) beginNewMatch(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.MatchStart{})
}

func (geh gameEventHandler) roundFreezeEnd(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.RoundFreezetimeEnd{})
}

func (geh gameEventHandler) playerFootstep(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	geh.dispatch(events.Footstep{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

func (geh gameEventHandler) playerJump(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	geh.dispatch(events.PlayerJump{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

func (geh gameEventHandler) weaponFire(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	shooter := geh.playerByUserID32(data["userid"].GetValShort())
	wepType := common.MapEquipment(data["weapon"].GetValString())

	geh.dispatch(events.WeaponFire{
		Shooter: shooter,
		Weapon:  getPlayerWeapon(shooter, wepType),
	})
}

func (geh gameEventHandler) playerDeath(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	killer := geh.playerByUserID32(data["attacker"].GetValShort())
	wepType := common.MapEquipment(data["weapon"].GetValString())

	geh.dispatch(events.Kill{
		Victim:            geh.playerByUserID32(data["userid"].GetValShort()),
		Killer:            killer,
		Assister:          geh.playerByUserID32(data["assister"].GetValShort()),
		IsHeadshot:        data["headshot"].GetValBool(),
		PenetratedObjects: int(data["penetrated"].GetValShort()),
		Weapon:            getPlayerWeapon(killer, wepType),
	})
}

func (geh gameEventHandler) playerHurt(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	attacker := geh.playerByUserID32(data["attacker"].GetValShort())
	wepType := common.MapEquipment(data["weapon"].GetValString())

	geh.dispatch(events.PlayerHurt{
		Player:       geh.playerByUserID32(data["userid"].GetValShort()),
		Attacker:     attacker,
		Health:       int(data["health"].GetValByte()),
		Armor:        int(data["armor"].GetValByte()),
		HealthDamage: int(data["dmg_health"].GetValShort()),
		ArmorDamage:  int(data["dmg_armor"].GetValByte()),
		HitGroup:     events.HitGroup(data["hitgroup"].GetValByte()),
		Weapon:       getPlayerWeapon(attacker, wepType),
	})
}

func (geh gameEventHandler) playerBlind(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	// Player.FlashDuration hasn't been updated yet,
	// so we need to wait until the end of the tick before dispatching
	geh.parser.delayedEvents = append(geh.parser.delayedEvents, events.PlayerFlashed{
		Player:   geh.playerByUserID32(data["userid"].GetValShort()),
		Attacker: geh.gameState().lastFlasher,
	})
}

func (geh gameEventHandler) flashbangDetonate(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	nadeEvent := geh.nadeEvent(desc, ge, common.EqFlash)

	geh.gameState().lastFlasher = nadeEvent.Thrower
	geh.dispatch(events.FlashExplode{
		GrenadeEvent: nadeEvent,
	})
}

func (geh gameEventHandler) hegranadeDetonate(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.HeExplode{
		GrenadeEvent: geh.nadeEvent(desc, ge, common.EqHE),
	})
}

func (geh gameEventHandler) decoyStarted(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.DecoyStart{
		GrenadeEvent: geh.nadeEvent(desc, ge, common.EqDecoy),
	})
}

func (geh gameEventHandler) decoyDetonate(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.DecoyExpired{
		GrenadeEvent: geh.nadeEvent(desc, ge, common.EqDecoy),
	})
}

func (geh gameEventHandler) smokegrenadeDetonate(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.SmokeStart{
		GrenadeEvent: geh.nadeEvent(desc, ge, common.EqSmoke),
	})
}

func (geh gameEventHandler) smokegrenadeExpired(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.SmokeExpired{
		GrenadeEvent: geh.nadeEvent(desc, ge, common.EqSmoke),
	})
}

func (geh gameEventHandler) infernoStartburn(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.FireGrenadeStart{
		GrenadeEvent: geh.nadeEvent(desc, ge, common.EqIncendiary),
	})
}

func (geh gameEventHandler) infernoExpire(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.FireGrenadeExpired{
		GrenadeEvent: geh.nadeEvent(desc, ge, common.EqIncendiary),
	})
}

func (geh gameEventHandler) playerConnect(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	pl := &playerInfo{
		userID: int(data["userid"].GetValShort()),
		name:   data["name"].GetValString(),
		guid:   data["networkid"].GetValString(),
	}

	pl.xuid = getCommunityID(pl.guid)

	geh.parser.rawPlayers[int(data["index"].GetValByte())] = pl
}

func (geh gameEventHandler) playerDisconnect(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	uid := int(data["userid"].GetValShort())

	for k, v := range geh.parser.rawPlayers {
		if v.userID == uid {
			delete(geh.parser.rawPlayers, k)
		}
	}

	pl := geh.playerByUserID(uid)
	if pl != nil {
		// Dispatch this event early since we delete the player on the next line
		geh.dispatch(events.PlayerDisconnected{
			Player: pl,
		})

		geh.playerByUserID(uid).IsConnected = false
	}
}

func (geh gameEventHandler) playerTeam(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	player := geh.playerByUserID32(data["userid"].GetValShort())
	newTeam := common.Team(data["team"].GetValByte())

	if player != nil {
		if player.Team != newTeam {
			player.Team = newTeam

			oldTeam := common.Team(data["oldteam"].GetValByte())
			// Delayed for two reasons
			// - team IDs of other players changing teams in the same tick might not have changed yet
			// - player entities might not have been re-created yet after a reconnect
			geh.parser.delayedEvents = append(geh.parser.delayedEvents, events.PlayerTeamChange{
				Player:       player,
				IsBot:        data["isbot"].GetValBool(),
				Silent:       data["silent"].GetValBool(),
				NewTeam:      newTeam,
				NewTeamState: geh.gameState().Team(newTeam),
				OldTeam:      oldTeam,
				OldTeamState: geh.gameState().Team(oldTeam),
			})
		} else {
			geh.dispatch(events.ParserWarn{
				Message: "Player team swap game-event occurred but player.Team == newTeam",
			})
		}
	} else {
		geh.dispatch(events.ParserWarn{
			Message: "Player team swap game-event occurred but player is nil",
		})
	}
}

func (geh gameEventHandler) bombBeginplant(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.BombPlantBegin{BombEvent: geh.bombEvent(desc, ge)})
}

func (geh gameEventHandler) bombPlanted(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.BombPlanted{BombEvent: geh.bombEvent(desc, ge)})
}

func (geh gameEventHandler) bombDefused(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	bombEvent := geh.bombEvent(desc, ge)
	geh.gameState().currentDefuser = nil
	geh.dispatch(events.BombDefused{BombEvent: bombEvent})
}

func (geh gameEventHandler) bombExploded(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	bombEvent := geh.bombEvent(desc, ge)
	geh.gameState().currentDefuser = nil
	geh.dispatch(events.BombExplode{BombEvent: bombEvent})
}

func (geh gameEventHandler) bombEvent(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) events.BombEvent {
	data := mapGameEventData(desc, ge)

	bombEvent := events.BombEvent{Player: geh.playerByUserID32(data["userid"].GetValShort())}

	site := int(data["site"].GetValShort())

	switch site {
	case geh.parser.bombsiteA.index:
		bombEvent.Site = events.BombsiteA
	case geh.parser.bombsiteB.index:
		bombEvent.Site = events.BombsiteB
	default:
		t := geh.parser.triggers[site]

		if t == nil {
			panic(fmt.Sprintf("Bombsite with index %d not found", site))
		}

		if t.contains(geh.parser.bombsiteA.center) {
			bombEvent.Site = events.BombsiteA
			geh.parser.bombsiteA.index = site
		} else if t.contains(geh.parser.bombsiteB.center) {
			bombEvent.Site = events.BombsiteB
			geh.parser.bombsiteB.index = site
		} else {
			panic("Bomb not planted on bombsite A or B")
		}
	}

	return bombEvent
}

func (geh gameEventHandler) bombBegindefuse(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	geh.gameState().currentDefuser = geh.playerByUserID32(data["userid"].GetValShort())

	geh.dispatch(events.BombDefuseStart{
		Player: geh.gameState().currentDefuser,
		HasKit: data["haskit"].GetValBool(),
	})
}

func (geh gameEventHandler) itemEquip(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	player, weapon := geh.itemEvent(desc, ge)
	geh.dispatch(events.ItemEquip{
		Player:    player,
		Weapon:    *weapon,
		WeaponPtr: weapon,
	})
}

func (geh gameEventHandler) itemPickup(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	player, weapon := geh.itemEvent(desc, ge)
	// Delayed because of #119 - Equipment.UniqueID()
	geh.parser.delayedEvents = append(geh.parser.delayedEvents, events.ItemPickup{
		Player: player,
		Weapon: *weapon,
	})
}

func (geh gameEventHandler) itemRemove(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	player, weapon := geh.itemEvent(desc, ge)
	geh.dispatch(events.ItemDrop{
		Player:    player,
		Weapon:    *weapon,
		WeaponPtr: weapon,
	})
}

func (geh gameEventHandler) itemEvent(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) (*common.Player, *common.Equipment) {
	data := mapGameEventData(desc, ge)
	player := geh.playerByUserID32(data["userid"].GetValShort())

	wepType := common.MapEquipment(data["item"].GetValString())
	weapon := getPlayerWeapon(player, wepType)

	return player, weapon
}

func (geh gameEventHandler) bombDropped(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	player := geh.playerByUserID32(data["userid"].GetValShort())
	entityID := int(data["entityid"].GetValShort())

	geh.dispatch(events.BombDropped{
		Player:   player,
		EntityID: entityID,
	})
}

func (geh gameEventHandler) bombPickup(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	data := mapGameEventData(desc, ge)

	geh.dispatch(events.BombPickup{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

func (geh gameEventHandler) genericEvent(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.GenericGameEvent{
		Name: desc.Name,
		Data: mapGameEventData(desc, ge),
	})
}

func (geh gameEventHandler) unknownEvent(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent) {
	geh.dispatch(events.ParserWarn{Message: fmt.Sprintf("Unknown event %q", desc.Name)})
	geh.genericEvent(desc, ge)
}

// Just so we can nicely create GrenadeEvents in one line
func (geh gameEventHandler) nadeEvent(desc *msg.CSVCMsg_GameEventListDescriptorT, ge *msg.CSVCMsg_GameEvent, nadeType common.EquipmentElement) events.GrenadeEvent {
	data := mapGameEventData(desc, ge)
	thrower := geh.playerByUserID32(data["userid"].GetValShort())
	position := r3.Vector{
		X: float64(data["x"].ValFloat),
		Y: float64(data["y"].ValFloat),
		Z: float64(data["z"].ValFloat),
	}
	nadeEntityID := int(data["entityid"].GetValShort())

	return events.GrenadeEvent{
		GrenadeType:     nadeType,
		Thrower:         thrower,
		Position:        position,
		GrenadeEntityID: nadeEntityID,
	}
}

func mapGameEventData(d *msg.CSVCMsg_GameEventListDescriptorT, e *msg.CSVCMsg_GameEvent) map[string]*msg.CSVCMsg_GameEventKeyT {
	data := make(map[string]*msg.CSVCMsg_GameEventKeyT)
	for i, k := range d.Keys {
		data[k.Name] = e.Keys[i]
	}
	return data
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
