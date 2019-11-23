package demoinfocs

import (
	"fmt"
	"strconv"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/go-unassert"

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
		p.eventDispatcher.Dispatch(events.ParserWarn{Message: "received GameEvent but event descriptors are missing"})
		unassert.Error("received GameEvent but event descriptors are missing")
		return
	}

	desc := p.gameEventDescs[ge.Eventid]

	debugGameEvent(desc, ge)

	data := mapGameEventData(desc, ge)

	if handler, eventKnown := p.gameEventHandler.gameEventNameToHandler[desc.Name]; eventKnown {
		// some events are known but have no handler
		// these will just be dispatched as GenericGameEvent
		if handler != nil {
			handler(data)
		}
	} else {
		p.eventDispatcher.Dispatch(events.ParserWarn{Message: fmt.Sprintf("unknown event %q", desc.Name)})
		unassert.Error("unknown event %q", desc.Name)
	}

	p.eventDispatcher.Dispatch(events.GenericGameEvent{
		Name: desc.Name,
		Data: data,
	})
}

type gameEventHandler struct {
	parser                 *Parser
	gameEventNameToHandler map[string]gameEventHandlerFunc
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

type gameEventHandlerFunc func(map[string]*msg.CSVCMsg_GameEventKeyT)

func newGameEventHandler(parser *Parser) gameEventHandler {
	geh := gameEventHandler{parser: parser}

	// some events need to be delayed until their data is available
	// some events can't be delayed because the required state is lost by the end of the tick
	// TODO: maybe we're supposed to delay all of them and store the data we need until the end of the tick
	delay := func(f gameEventHandlerFunc) gameEventHandlerFunc {
		return func(data map[string]*msg.CSVCMsg_GameEventKeyT) {
			parser.delayedEventHandlers = append(parser.delayedEventHandlers, func() {
				f(data)
			})
		}
	}

	// some events only need to be delayed at the start of the demo until players are connected
	delayIfNoPlayers := func(f gameEventHandlerFunc) gameEventHandlerFunc {
		return func(data map[string]*msg.CSVCMsg_GameEventKeyT) {
			if len(parser.gameState.playersByUserID) == 0 {
				delay(f)
			} else {
				f(data)
			}
		}
	}

	geh.gameEventNameToHandler = map[string]gameEventHandlerFunc{
		// sorted alphabetically
		"announce_phase_end":              nil,                                  // Dunno
		"begin_new_match":                 geh.beginNewMatch,                    // Match started
		"bomb_beep":                       nil,                                  // Bomb beep
		"bomb_begindefuse":                geh.bombBeginDefuse,                  // Defuse started
		"bomb_beginplant":                 geh.bombBeginPlant,                   // Plant started
		"bomb_defused":                    geh.bombDefused,                      // Defuse finished
		"bomb_dropped":                    geh.bombDropped,                      // Bomb dropped
		"bomb_exploded":                   geh.bombExploded,                     // Bomb exploded
		"bomb_pickup":                     geh.bombPickup,                       // Bomb picked up
		"bomb_planted":                    geh.bombPlanted,                      // Plant finished
		"bot_takeover":                    geh.botTakeover,                      // Bot got taken over
		"buytime_ended":                   nil,                                  // Not actually end of buy time, seems to only be sent once per game at the start
		"cs_match_end_restart":            nil,                                  // Yawn
		"cs_pre_restart":                  nil,                                  // Not sure, doesn't seem to be important
		"cs_round_final_beep":             nil,                                  // Final beep
		"cs_round_start_beep":             nil,                                  // Round start beeps
		"cs_win_panel_match":              geh.csWinPanelMatch,                  // Not sure, maybe match end event???
		"cs_win_panel_round":              nil,                                  // Win panel, (==end of match?)
		"decoy_detonate":                  geh.decoyDetonate,                    // Decoy exploded/expired
		"decoy_started":                   delay(geh.decoyStarted),              // Decoy started. Delayed because projectile entity is not yet created
		"endmatch_cmm_start_reveal_items": nil,                                  // Drops
		"entity_visible":                  nil,                                  // Dunno, only in locally recorded demo
		"enter_bombzone":                  nil,                                  // Dunno, only in locally recorded demo
		"exit_bombzone":                   nil,                                  // Dunno, only in locally recorded demo
		"enter_buyzone":                   nil,                                  // Dunno, only in locally recorded demo
		"exit_buyzone":                    nil,                                  // Dunno, only in locally recorded demo
		"flashbang_detonate":              geh.flashBangDetonate,                // Flash exploded
		"hegrenade_detonate":              geh.heGrenadeDetonate,                // HE exploded
		"hltv_chase":                      nil,                                  // Don't care
		"hltv_fixed":                      nil,                                  // Dunno
		"hltv_message":                    nil,                                  // No clue
		"hltv_status":                     nil,                                  // Don't know
		"inferno_expire":                  geh.infernoExpire,                    // Incendiary expired
		"inferno_startburn":               delay(geh.infernoStartBurn),          // Incendiary exploded/started. Delayed because inferno entity is not yet created
		"inspect_weapon":                  nil,                                  // Dunno, only in locally recorded demo
		"item_equip":                      delay(geh.itemEquip),                 // Equipped / weapon swap, I think. Delayed because of #142 - Bot entity possibly not yet created
		"item_pickup":                     delay(geh.itemPickup),                // Picked up or bought? Delayed because of #119 - Equipment.UniqueID()
		"item_remove":                     geh.itemRemove,                       // Dropped?
		"jointeam_failed":                 nil,                                  // Dunno, only in locally recorded demo
		"other_death":                     nil,                                  // Dunno
		"player_blind":                    delay(geh.playerBlind),               // Player got blinded by a flash. Delayed because Player.FlashDuration hasn't been updated yet
		"player_changename":               nil,                                  // Name change
		"player_connect":                  geh.playerConnect,                    // Bot connected or player reconnected, players normally come in via string tables & data tables
		"player_connect_full":             nil,                                  // Connecting finished
		"player_death":                    delayIfNoPlayers(geh.playerDeath),    // Player died
		"player_disconnect":               geh.playerDisconnect,                 // Player disconnected (kicked, quit, timed out etc.)
		"player_falldamage":               nil,                                  // Falldamage
		"player_footstep":                 delayIfNoPlayers(geh.playerFootstep), // Footstep sound.- Delayed because otherwise Player might be nil
		"player_hurt":                     geh.playerHurt,                       // Player got hurt
		"player_jump":                     geh.playerJump,                       // Player jumped
		"player_spawn":                    nil,                                  // Player spawn
		"player_given_c4":                 nil,                                  // Dunno, only present in POV demos

		// Player changed team. Delayed for two reasons
		// - team IDs of other players changing teams in the same tick might not have changed yet
		// - player entities might not have been re-created yet after a reconnect
		"player_team":                    delay(geh.playerTeam),
		"round_announce_final":           geh.roundAnnounceFinal,           // 30th round for normal de_, not necessarily matchpoint
		"round_announce_last_round_half": geh.roundAnnounceLastRoundHalf,   // Last round of the half
		"round_announce_match_point":     nil,                              // Match point announcement
		"round_announce_match_start":     nil,                              // Special match start announcement
		"round_announce_warmup":          nil,                              // Dunno
		"round_end":                      geh.roundEnd,                     // Round ended and the winner was announced
		"round_end_upload_stats":         nil,                              // Dunno, only present in POV demos
		"round_freeze_end":               geh.roundFreezeEnd,               // Round start freeze ended
		"round_mvp":                      geh.roundMVP,                     // Round MVP was announced
		"round_officially_ended":         geh.roundOfficiallyEnded,         // The event after which you get teleported to the spawn (=> You can still walk around between round_end and this event)
		"round_poststart":                nil,                              // Ditto
		"round_prestart":                 nil,                              // Ditto
		"round_start":                    geh.roundStart,                   // Round started
		"round_time_warning":             nil,                              // Round time warning
		"server_cvar":                    nil,                              // Dunno
		"smokegrenade_detonate":          geh.smokeGrenadeDetonate,         // Smoke popped
		"smokegrenade_expired":           geh.smokeGrenadeExpired,          // Smoke expired
		"switch_team":                    nil,                              // Dunno, only present in POV demos
		"tournament_reward":              nil,                              // Dunno
		"weapon_fire":                    delayIfNoPlayers(geh.weaponFire), // Weapon was fired
		"weapon_fire_on_empty":           nil,                              // Sounds boring
		"weapon_reload":                  geh.weaponReload,                 // Weapon reloaded
		"weapon_zoom":                    nil,                              // Zooming in
		"weapon_zoom_rifle":              nil,                              // Dunno, only in locally recorded demo
	}

	return geh
}

func (geh gameEventHandler) roundStart(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.RoundStart{
		TimeLimit: int(data["timelimit"].GetValLong()),
		FragLimit: int(data["fraglimit"].GetValLong()),
		Objective: data["objective"].GetValString(),
	})
}

func (geh gameEventHandler) csWinPanelMatch(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.AnnouncementWinPanelMatch{})
}

func (geh gameEventHandler) roundAnnounceFinal(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.AnnouncementFinalRound{})
}

func (geh gameEventHandler) roundAnnounceLastRoundHalf(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.AnnouncementLastRoundHalf{})
}

func (geh gameEventHandler) roundEnd(data map[string]*msg.CSVCMsg_GameEventKeyT) {
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

func (geh gameEventHandler) roundOfficiallyEnded(data map[string]*msg.CSVCMsg_GameEventKeyT) {
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

func (geh gameEventHandler) roundMVP(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.RoundMVPAnnouncement{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
		Reason: events.RoundMVPReason(data["reason"].GetValShort()),
	})
}

func (geh gameEventHandler) botTakeover(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.BotTakenOver{
		Taker: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

func (geh gameEventHandler) beginNewMatch(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.MatchStart{})
}

func (geh gameEventHandler) roundFreezeEnd(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.RoundFreezetimeEnd{})
}

func (geh gameEventHandler) playerFootstep(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.Footstep{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

func (geh gameEventHandler) playerJump(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.PlayerJump{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

func (geh gameEventHandler) weaponFire(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	shooter := geh.playerByUserID32(data["userid"].GetValShort())
	wepType := common.MapEquipment(data["weapon"].GetValString())

	geh.dispatch(events.WeaponFire{
		Shooter: shooter,
		Weapon:  getPlayerWeapon(shooter, wepType),
	})
}

func (geh gameEventHandler) weaponReload(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	pl := geh.playerByUserID32(data["userid"].GetValShort())
	if pl == nil {
		// see #162, "unknown" players since November 2019 update
		return
	}

	pl.IsReloading = true
	geh.dispatch(events.WeaponReload{
		Player: pl,
	})
}

func (geh gameEventHandler) playerDeath(data map[string]*msg.CSVCMsg_GameEventKeyT) {
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

func (geh gameEventHandler) playerHurt(data map[string]*msg.CSVCMsg_GameEventKeyT) {
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

func (geh gameEventHandler) playerBlind(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	attacker := geh.gameState().lastFlash.player
	projectile := geh.gameState().lastFlash.projectileByPlayer[attacker]
	unassert.NotNilf(projectile, "PlayerFlashed.Projectile should never be nil")

	if projectile != nil {
		unassert.Samef(attacker, projectile.Thrower, "PlayerFlashed.Attacker != PlayerFlashed.Projectile.Thrower")
		unassert.Samef(projectile.Weapon, common.EqFlash, "PlayerFlashed.Projectile.Weapon != EqFlash")
	}

	geh.dispatch(events.PlayerFlashed{
		Player:     geh.playerByUserID32(data["userid"].GetValShort()),
		Attacker:   attacker,
		Projectile: projectile,
	})
}

func (geh gameEventHandler) flashBangDetonate(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	nadeEvent := geh.nadeEvent(data, common.EqFlash)

	geh.gameState().lastFlash.player = nadeEvent.Thrower
	geh.dispatch(events.FlashExplode{
		GrenadeEvent: nadeEvent,
	})
}

func (geh gameEventHandler) heGrenadeDetonate(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.HeExplode{
		GrenadeEvent: geh.nadeEvent(data, common.EqHE),
	})
}

func (geh gameEventHandler) decoyStarted(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.DecoyStart{
		GrenadeEvent: geh.nadeEvent(data, common.EqDecoy),
	})
}

func (geh gameEventHandler) decoyDetonate(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.DecoyExpired{
		GrenadeEvent: geh.nadeEvent(data, common.EqDecoy),
	})
}

func (geh gameEventHandler) smokeGrenadeDetonate(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.SmokeStart{
		GrenadeEvent: geh.nadeEvent(data, common.EqSmoke),
	})
}

func (geh gameEventHandler) smokeGrenadeExpired(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.SmokeExpired{
		GrenadeEvent: geh.nadeEvent(data, common.EqSmoke),
	})
}

func (geh gameEventHandler) infernoStartBurn(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.FireGrenadeStart{
		GrenadeEvent: geh.nadeEvent(data, common.EqIncendiary),
	})
}

func (geh gameEventHandler) infernoExpire(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.FireGrenadeExpired{
		GrenadeEvent: geh.nadeEvent(data, common.EqIncendiary),
	})
}

func (geh gameEventHandler) playerConnect(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	pl := &playerInfo{
		userID: int(data["userid"].GetValShort()),
		name:   data["name"].GetValString(),
		guid:   data["networkid"].GetValString(),
	}

	pl.xuid = getCommunityID(pl.guid)

	geh.parser.rawPlayers[int(data["index"].GetValByte())] = pl
}

func (geh gameEventHandler) playerDisconnect(data map[string]*msg.CSVCMsg_GameEventKeyT) {
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

func (geh gameEventHandler) playerTeam(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	player := geh.playerByUserID32(data["userid"].GetValShort())
	newTeam := common.Team(data["team"].GetValByte())

	if player != nil {
		if player.Team != newTeam {
			player.Team = newTeam
		}

		oldTeam := common.Team(data["oldteam"].GetValByte())
		geh.dispatch(events.PlayerTeamChange{
			Player:       player,
			IsBot:        data["isbot"].GetValBool(),
			Silent:       data["silent"].GetValBool(),
			NewTeam:      newTeam,
			NewTeamState: geh.gameState().Team(newTeam),
			OldTeam:      oldTeam,
			OldTeamState: geh.gameState().Team(oldTeam),
		})
	} else {
		// TODO: figure out why this happens and whether it's a bug or not
		geh.dispatch(events.ParserWarn{
			Message: "Player team swap game-event occurred but player is nil",
		})
	}
}

func (geh gameEventHandler) bombBeginPlant(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	event := events.BombPlantBegin{BombEvent: geh.bombEvent(data)}
	event.Player.IsPlanting = true
	geh.parser.gameState.currentPlanter = event.Player
	geh.dispatch(event)
}

func (geh gameEventHandler) bombPlanted(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	event := events.BombPlanted{BombEvent: geh.bombEvent(data)}
	event.Player.IsPlanting = false
	geh.parser.gameState.currentPlanter = nil
	geh.dispatch(event)
}

func (geh gameEventHandler) bombDefused(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	bombEvent := geh.bombEvent(data)
	geh.gameState().currentDefuser = nil
	geh.dispatch(events.BombDefused{BombEvent: bombEvent})
}

func (geh gameEventHandler) bombExploded(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	bombEvent := geh.bombEvent(data)
	geh.gameState().currentDefuser = nil
	geh.dispatch(events.BombExplode{BombEvent: bombEvent})
}

func (geh gameEventHandler) bombEvent(data map[string]*msg.CSVCMsg_GameEventKeyT) events.BombEvent {
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

func (geh gameEventHandler) bombBeginDefuse(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.gameState().currentDefuser = geh.playerByUserID32(data["userid"].GetValShort())

	geh.dispatch(events.BombDefuseStart{
		Player: geh.gameState().currentDefuser,
		HasKit: data["haskit"].GetValBool(),
	})
}

func (geh gameEventHandler) itemEquip(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	player, weapon := geh.itemEvent(data)
	geh.dispatch(events.ItemEquip{
		Player:    player,
		Weapon:    *weapon,
		WeaponPtr: weapon,
	})
}

func (geh gameEventHandler) itemPickup(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	player, weapon := geh.itemEvent(data)
	geh.dispatch(events.ItemPickup{
		Player: player,
		Weapon: *weapon,
	})
}

func (geh gameEventHandler) itemRemove(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	player, weapon := geh.itemEvent(data)
	geh.dispatch(events.ItemDrop{
		Player:    player,
		Weapon:    *weapon,
		WeaponPtr: weapon,
	})
}

func (geh gameEventHandler) itemEvent(data map[string]*msg.CSVCMsg_GameEventKeyT) (*common.Player, *common.Equipment) {
	player := geh.playerByUserID32(data["userid"].GetValShort())

	wepType := common.MapEquipment(data["item"].GetValString())
	weapon := getPlayerWeapon(player, wepType)

	return player, weapon
}

func (geh gameEventHandler) bombDropped(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	player := geh.playerByUserID32(data["userid"].GetValShort())
	entityID := int(data["entityid"].GetValShort())

	geh.dispatch(events.BombDropped{
		Player:   player,
		EntityID: entityID,
	})
}

func (geh gameEventHandler) bombPickup(data map[string]*msg.CSVCMsg_GameEventKeyT) {
	geh.dispatch(events.BombPickup{
		Player: geh.playerByUserID32(data["userid"].GetValShort()),
	})
}

// Just so we can nicely create GrenadeEvents in one line
func (geh gameEventHandler) nadeEvent(data map[string]*msg.CSVCMsg_GameEventKeyT, nadeType common.EquipmentElement) events.GrenadeEvent {
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
	if player != nil {
		alternateWepType := common.EquipmentAlternative(wepType)
		for _, wep := range player.Weapons() {
			if wep.Weapon == wepType || wep.Weapon == alternateWepType {
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
