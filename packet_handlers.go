package demoinfocs

import (
	"bytes"
	"fmt"
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"github.com/markus-wa/demoinfocs-golang/st"
	"strconv"
)

func (p *Parser) handlePackageEntities(pe *msg.CSVCMsg_PacketEntities) {
	r := bs.NewBitReader(bytes.NewReader(pe.EntityData), bs.SmallBuffer)

	currentEntity := -1
	for i := 0; i < int(pe.UpdatedEntries); i++ {
		currentEntity += 1 + int(r.ReadUBitInt())
		if !r.ReadBit() {
			if r.ReadBit() {
				e := p.readEnterPVS(r, currentEntity)
				p.entities[currentEntity] = e
				e.ApplyUpdate(r)
			} else {
				p.entities[currentEntity].ApplyUpdate(r)
			}
		} else {
			// FIXME: Might have to destroy the entities contents first, not sure yet
			// Could do weird stuff with event handlers otherwise
			p.entities[currentEntity] = nil
			r.ReadBit()
		}
	}
	r.Close()
}

func (p *Parser) readEnterPVS(reader bs.BitReader, entityId int) *st.Entity {
	scId := int(reader.ReadInt(uint(p.stParser.ClassBits())))
	reader.ReadInt(10)
	newEntity := st.NewEntity(entityId, p.stParser.ServerClasses()[scId])
	newEntity.ServerClass.FireEntityCreatedEvent(newEntity)

	if p.preprocessedBaselines[scId] != nil {
		for _, bl := range p.preprocessedBaselines[scId] {
			newEntity.Props()[bl.PropIndex()].FirePropertyUpdateEvent(bl.Value(), newEntity)
		}
	} else {
		ppBase := make([]*st.RecordedPropertyUpdate, 0)
		if p.instanceBaselines[scId] != nil {
			newEntity.CollectProperties(&ppBase)
			r := bs.NewBitReader(bytes.NewReader(p.instanceBaselines[scId]), bs.SmallBuffer)
			newEntity.ApplyUpdate(r)
			r.Close()
		}
		p.preprocessedBaselines[scId] = ppBase
	}

	return newEntity
}

func (p *Parser) handleGameEventList(gel *msg.CSVCMsg_GameEventList) {
	p.gehDescriptors = make(map[int32]*msg.CSVCMsg_GameEventListDescriptorT)
	for _, d := range gel.GetDescriptors() {
		p.gehDescriptors[d.GetEventid()] = d
	}
}

func (p *Parser) handleGameEvent(ge *msg.CSVCMsg_GameEvent) {
	if p.gehDescriptors == nil {
		return
	}

	data := make(map[string]*msg.CSVCMsg_GameEventKeyT)
	d := p.gehDescriptors[ge.Eventid]

	// Ignore events before players are connected to speed things up
	if len(p.connectedPlayers) == 0 && d.Name != "player_connect" {
		return
	}

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

		t := common.Team_Spectators

		switch data["winner"].GetValByte() {
		case int32(p.tState.id):
			t = common.Team_Terrorists
		case int32(p.ctState.id):
			t = common.Team_CounterTerrorists
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
			Player: p.connectedPlayers[int(data["userid"].GetValShort())],
			Reason: common.RoundMVPReason(data["reason"].GetValShort()),
		})

	case "bot_takeover": // Bot got taken over
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.BotTakenOverEvent{Taker: p.connectedPlayers[int(data["userid"].GetValShort())]})

	case "begin_new_match": // Match started
		p.eventDispatcher.Dispatch(events.MatchStartedEvent{})

	case "round_freeze_end": // Round start freeze ended
		p.eventDispatcher.Dispatch(events.FreezetimeEndedEvent{})

	case "player_jump": // Player jumped
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.PlayerJumpEvent{Player: p.connectedPlayers[int(data["userid"].GetValShort())]})

	case "weapon_fire": // Weapon was fired
		data = mapGameEventData(d, ge)

		e := events.WeaponFiredEvent{Shooter: p.connectedPlayers[int(data["userid"].GetValShort())]}
		wep := common.NewEquipment(data["weapon"].GetValString())

		if e.Shooter != nil && wep.Class() != common.EC_Grenade {
			e.Weapon = e.Shooter.ActiveWeapon()
		} else {
			e.Weapon = &wep
		}

		p.eventDispatcher.Dispatch(e)

	case "player_death": // Player died
		data = mapGameEventData(d, ge)

		e := events.PlayerKilledEvent{
			Victim:            p.connectedPlayers[int(data["userid"].GetValShort())],
			Killer:            p.connectedPlayers[int(data["attacker"].GetValShort())],
			Assister:          p.connectedPlayers[int(data["assister"].GetValShort())],
			IsHeadshot:        data["assister"].GetValBool(),
			PenetratedObjects: int(data["penetrated"].GetValShort()),
		}

		wep := common.NewSkinEquipment(data["weapon"].GetValString(), data["weapon_itemid"].GetValString())

		// FIXME: Should we do that last weapons > 0 check above as well?????
		if e.Killer != nil && wep.Class() != common.EC_Grenade && len(e.Killer.Weapons) > 0 {
			e.Weapon = e.Killer.ActiveWeapon()
		} else {
			e.Weapon = &wep
		}

		p.eventDispatcher.Dispatch(e)

	case "player_hurt": // Player got hurt
		data = mapGameEventData(d, ge)

		e := events.PlayerHurtEvent{
			Player:       p.connectedPlayers[int(data["userid"].GetValShort())],
			Attacker:     p.connectedPlayers[int(data["attacker"].GetValShort())],
			Health:       int(data["health"].GetValShort()),
			Armor:        int(data["armor"].GetValShort()),
			HealthDamage: int(data["dmg_health"].GetValShort()),
			ArmorDamage:  int(data["dmg_armor"].GetValShort()),
			Hitgroup:     common.Hitgroup(data["penetrated"].GetValByte()),
		}

		wep := common.NewEquipment(data["weapon"].GetValString())

		if e.Attacker != nil && wep.Class() != common.EC_Grenade && len(e.Attacker.Weapons) > 0 {
			e.Weapon = e.Attacker.ActiveWeapon()
		} else {
			e.Weapon = &wep
		}

		p.eventDispatcher.Dispatch(e)

	case "player_blind": // Player got blinded by a flash
		data = mapGameEventData(d, ge)
		p.eventDispatcher.Dispatch(events.PlayerFlashedEvent{Player: p.connectedPlayers[int(data["userid"].GetValShort())]})

	case "flashbang_detonate": // Flash exploded
		p.eventDispatcher.Dispatch(events.FlashExplodedEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_Flash)})

	case "hegrenade_detonate": // HE exploded
		p.eventDispatcher.Dispatch(events.HeExplodedEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_HE)})

	case "decoy_started": // Decoy started
		p.eventDispatcher.Dispatch(events.DecoyStartEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_Decoy)})

	case "decoy_detonate": // Decoy exploded/expired
		p.eventDispatcher.Dispatch(events.DecoyEndEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_Decoy)})

	case "smokegrenade_detonate": // Smoke popped
		p.eventDispatcher.Dispatch(events.SmokeStartEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_Smoke)})

	case "smokegrenade_expired": // Smoke expired
		p.eventDispatcher.Dispatch(events.SmokeEndEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_Smoke)})

	case "inferno_startburn": // Incendiary exploded/started
		p.eventDispatcher.Dispatch(events.FireNadeStartEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_Incendiary)})

	case "inferno_expire": // Incendiary expired
		p.eventDispatcher.Dispatch(events.FireNadeEndEvent{p.buildNadeEvent(mapGameEventData(d, ge), common.EE_Incendiary)})

	case "player_connect": // Player connected. . .?
		// FIXME: This doesn't seem to happen, ever???
		data = mapGameEventData(d, ge)

		pl := &common.PlayerInfo{
			UserId: int(data["userid"].GetValShort()),
			Name:   data["name"].GetValString(),
			GUID:   data["networkid"].GetValString(),
		}

		pl.XUID = getCommunityId(pl.GUID)

		p.rawPlayers[data["index"].GetValShort()] = pl

	case "player_disconnect": // Player disconnected
		data = mapGameEventData(d, ge)

		uid := int(data["userid"].GetValShort())
		e := events.PlayerDisconnectEvent{
			Player: p.connectedPlayers[uid],
		}
		p.eventDispatcher.Dispatch(e)

		for i := range p.rawPlayers {
			if p.rawPlayers[i] != nil && p.rawPlayers[i].UserId == uid {
				p.rawPlayers[i] = nil
			}
		}

		p.connectedPlayers[uid] = nil

	case "player_team": // Player changed team
		data = mapGameEventData(d, ge)

		e := events.PlayerTeamChangeEvent{
			Player: p.connectedPlayers[int(data["userid"].GetValShort())],
			IsBot:  data["isbot"].GetValBool(),
			Silent: data["silent"].GetValBool(),
		}

		switch data["team"].GetValByte() {
		case int32(p.tState.id):
			e.NewTeam = common.Team_Terrorists
		case int32(p.ctState.id):
			e.NewTeam = common.Team_CounterTerrorists
		default:
			e.NewTeam = common.Team_Spectators
		}

		switch data["oldteam"].GetValByte() {
		case int32(p.tState.id):
			e.OldTeam = common.Team_Terrorists
		case int32(p.ctState.id):
			e.OldTeam = common.Team_CounterTerrorists
		default:
			e.OldTeam = common.Team_Spectators
		}

		p.eventDispatcher.Dispatch(e)

	case "bomb_beginplant": // Plant started
		fallthrough
	case "bomb_abortplant": // Plant stopped
		fallthrough
	case "bomb_planted": // Plant finished
		fallthrough
	case "bomb_defused": // Defuse finished
		fallthrough
	case "bomb_exploded": // Bomb exploded
		data = mapGameEventData(d, ge)

		e := events.BombEvent{Player: p.connectedPlayers[int(data["userid"].GetValShort())]}

		site := int(data["site"].GetValShort())

		switch site {
		case p.bombsiteAIndex:
			e.Site = 'A'
		case p.bombsiteBIndex:
			e.Site = 'B'
		default:
			var t *BoundingBoxInformation
			for _, tr := range p.triggers {
				if tr.index == site {
					t = tr
				}
			}

			if t.contains(p.bombsiteACenter) {
				e.Site = 'A'
			} else if t.contains(p.bombsiteBCenter) {
				e.Site = 'B'
			} else {
				panic("Bomb not planted on bombsite A or B")
			}
		}

		switch d.Name {
		case "bomb_beginplant":
			p.eventDispatcher.Dispatch(events.BombBeginPlant{BombEvent: e})
		case "bomb_abortplant":
			p.eventDispatcher.Dispatch(events.BombAbortPlant{BombEvent: e})
		case "bomb_planted":
			p.eventDispatcher.Dispatch(events.BombPlantedEvent{BombEvent: e})
		case "bomb_defused":
			p.eventDispatcher.Dispatch(events.BombDefusedEvent{BombEvent: e})
		case "bomb_exploded":
			p.eventDispatcher.Dispatch(events.BombExplodedEvent{BombEvent: e})
		}

	case "bomb_begindefuse": // Defuse started
		data = mapGameEventData(d, ge)

		p.eventDispatcher.Dispatch(events.BombBeginDefuse{
			Defuser: p.connectedPlayers[int(data["userid"].GetValShort())],
			HasKit:  data["haskit"].GetValBool(),
		})

	case "bomb_abortdefuse": // Defuse aborted
		data = mapGameEventData(d, ge)

		pl := p.connectedPlayers[int(data["userid"].GetValShort())]

		p.eventDispatcher.Dispatch(events.BombAbortDefuse{
			Defuser: pl,
			HasKit:  pl.HasDefuseKit,
		})

	// TODO: Might be interesting:
	case "player_connect_full": // Connecting finished
	case "player_falldamage": // Falldamage
	case "weapon_zoom": // Zooming in
	case "weapon_reload": // Weapon reloaded
	case "bomb_dropped": // Bomb dropped
	case "bomb_pickup": // Bomb picked up
	case "round_time_warning": // Round time warning
	case "round_announce_match_point": // Match point announcement

	// Probably not that interesting:
	case "buytime_ended": // Not actually end of buy time, seems to only be sent once per game at the start
	case "round_announce_match_start": // Special match start announcement
	case "player_footstep": // Footstep sound
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
	default:
		fmt.Println("Unknown event", d.Name)
	}
}

func mapGameEventData(d *msg.CSVCMsg_GameEventListDescriptorT, e *msg.CSVCMsg_GameEvent) map[string]*msg.CSVCMsg_GameEventKeyT {
	data := make(map[string]*msg.CSVCMsg_GameEventKeyT)
	for i, k := range d.Keys {
		data[k.Name] = e.Keys[i]
	}
	return data
}

func getCommunityId(guid string) int64 {
	if guid == "BOT" {
		return 0
	}

	authSrv, errSrv := strconv.ParseInt(guid[8:9], 10, 64)
	authId, errId := strconv.ParseInt(guid[10:], 10, 64)

	if errSrv != nil {
		panic(errSrv.Error())
	}

	if errId != nil {
		panic(errId.Error())
	}

	// FIXME: WTF are we doing here???
	return 76561197960265728 + authId*2 + authSrv
}

func (p *Parser) buildNadeEvent(data map[string]*msg.CSVCMsg_GameEventKeyT, nadeType common.EquipmentElement) events.NadeEvent {
	return events.NadeEvent{
		NadeType: nadeType,
		Thrower:  p.connectedPlayers[int(data["userid"].GetValShort())],
		Position: r3.Vector{
			X: float64(data["x"].ValFloat),
			Z: float64(data["y"].ValFloat),
			Y: float64(data["z"].ValFloat),
		},
	}
}
func (p *Parser) handleUpdateStringTable(tab *msg.CSVCMsg_UpdateStringTable) {
	cTab := p.stringTables[tab.TableId]
	switch cTab.Name {
	case "userinfo":
		fallthrough
	case "modelprecache":
		fallthrough
	case "instancebaseline":
		// Only handle updates for the above types
		p.handleCreateStringTable(cTab)
	}

}

func (p *Parser) handleCreateStringTable(tab *msg.CSVCMsg_CreateStringTable) {
	if tab.Name == "modelprecache" {
		for i := len(p.modelPreCache); i < int(tab.MaxEntries); i++ {
			p.modelPreCache = append(p.modelPreCache, "")
		}
	}

	br := bs.NewBitReader(bytes.NewReader(tab.StringData), bs.SmallBuffer)

	if br.ReadBit() {
		panic("Can't decode")
	}

	nTmp := tab.MaxEntries
	var nEntryBits uint = 0

	for nTmp != 0 {
		nTmp = nTmp >> 1
		nEntryBits++
	}
	if nEntryBits > 0 {
		nEntryBits--
	}

	hist := make([]string, 0)
	lastEntry := -1
	for i := 0; i < int(tab.NumEntries); i++ {
		entryIndex := lastEntry + 1
		if !br.ReadBit() {
			entryIndex = int(br.ReadInt(nEntryBits))
		}

		lastEntry = entryIndex

		var entry string
		if entryIndex < 0 || entryIndex >= int(tab.MaxEntries) {
			panic("Something went to shit")
		}
		if br.ReadBit() {
			if br.ReadBit() {
				idx := br.ReadInt(5)
				bytes2cp := int(br.ReadInt(5))
				entry = hist[idx][:bytes2cp]

				entry += br.ReadString()
			} else {
				entry = br.ReadString()
			}
		}

		if len(hist) > 31 {
			hist = hist[1:]
		}
		hist = append(hist, entry)

		var userdat []byte
		if br.ReadBit() {
			if tab.UserDataFixedSize {
				userdat = br.ReadBits(uint(tab.UserDataSizeBits))
			} else {
				userdat = br.ReadBytes(int(br.ReadInt(14)))
			}
		}

		if len(userdat) == 0 {
			break
		}

		switch tab.Name {
		case "userinfo":
			p.rawPlayers[entryIndex] = common.ParsePlayerInfo(bs.NewBitReader(bytes.NewReader(userdat), bs.SmallBuffer))
		case "instancebaseline":
			classid, err := strconv.ParseInt(entry, 10, 64)
			if err != nil {
				panic("WTF VOLVO PLS")
			}
			p.instanceBaselines[int(classid)] = userdat
		case "modelprecache":
			p.modelPreCache[entryIndex] = entry
		}
	}
	p.stringTables = append(p.stringTables, tab)
}
