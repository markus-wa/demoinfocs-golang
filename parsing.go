package demoinfocs

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/st"
	"os"
	"strconv"
	"strings"
)

const maxOsPath = 260

const (
	playerWeaponPrefix    = "m_hMyWeapons."
	playerWeaponPrePrefix = "bcc_nonlocaldata."
)

const (
	teamName_Unassigned = "Unassigned"
	teamName_Spectator  = "Spectator"
	teamName_Terrorist  = "TERRORIST"
	teamName_Ct         = "CT"
)

// ParseHeader attempts to parse the header of the demo.
// Returns error if the filestamp (first 8 bytes) doesn't match HL2DEMO.
func (p *Parser) ParseHeader() error {
	var h common.DemoHeader
	h.Filestamp = p.bitReader.ReadCString(8)
	h.Protocol = p.bitReader.ReadSignedInt(32)
	h.NetworkProtocol = p.bitReader.ReadSignedInt(32)
	h.ServerName = p.bitReader.ReadCString(maxOsPath)
	h.ClientName = p.bitReader.ReadCString(maxOsPath)
	h.MapName = p.bitReader.ReadCString(maxOsPath)
	h.GameDirectory = p.bitReader.ReadCString(maxOsPath)
	h.PlaybackTime = p.bitReader.ReadFloat()
	h.PlaybackTicks = p.bitReader.ReadSignedInt(32)
	h.PlaybackFrames = p.bitReader.ReadSignedInt(32)
	h.SignonLength = p.bitReader.ReadSignedInt(32)

	if h.Filestamp != "HL2DEMO" {
		return errors.New("Invalid File-Type; expecting HL2DEMO in the first 8 bytes")
	}

	p.header = &h
	p.eventDispatcher.Dispatch(events.HeaderParsedEvent{Header: h})
	return nil
}

// ParseToEnd attempts to parse the demo until the end.
// Aborts and returns an error if Cancel() is called before the end.
// May panic if the demo is corrupt in some way.
func (p *Parser) ParseToEnd() error {
	for {
		select {
		case <-p.cancelChan:
			return errors.New("Parsing was cancelled before it finished")

		default:
			if !p.ParseNextTick() {
				return nil
			}
		}
	}
}

// Cancel aborts ParseToEnd() on the upcoming tick.
func (p *Parser) Cancel() {
	p.cancelChan <- struct{}{}
}

// ParseNextTick attempts to parse the next tick.
// Returns true unless the demo command 'stop' was encountered.
// Panics if header hasn't been parsed yet - see Parser.ParseHeader().
func (p *Parser) ParseNextTick() bool {
	if p.header == nil {
		panic("Tried to parse tick before parsing header")
	}
	b := p.parseTick()

	for k, rp := range p.rawPlayers {
		if rp == nil {
			continue
		}

		if pl := p.players[k]; pl != nil {
			newPlayer := false
			if p.connectedPlayers[rp.UserID] == nil {
				p.connectedPlayers[rp.UserID] = pl
				newPlayer = true
			}

			pl.Name = rp.Name
			pl.SteamID = rp.XUID
			pl.IsBot = rp.IsFakePlayer
			pl.AdditionalPlayerInformation = &p.additionalPlayerInfo[pl.EntityID]

			if pl.IsAlive() {
				pl.LastAlivePosition = pl.Position
			}

			if newPlayer && pl.SteamID != 0 {
				p.eventDispatcher.Dispatch(events.PlayerBindEvent{Player: pl})
			}
		}
	}

	p.eventDispatcher.Dispatch(events.TickDoneEvent{})

	if !b {
		close(p.msgQueue)
	}

	return b
}

func (p *Parser) parseTick() bool {
	cmd := demoCommand(p.bitReader.ReadSingleByte())

	// Tick number
	p.ingameTick = p.bitReader.ReadSignedInt(32)
	// Skip 'player slot'
	p.bitReader.ReadSingleByte()

	p.currentTick++

	switch cmd {
	case dc_Synctick:
		// Ignore

	case dc_Stop:
		return false

	case dc_ConsoleCommand:
		// Skip
		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.bitReader.EndChunk()

	case dc_DataTables:
		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.stParser.ParsePacket(p.bitReader)
		p.bitReader.EndChunk()

		p.mapEquipment()
		p.bindEntities()

	case dc_StringTables:
		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.parseStringTables()
		p.bitReader.EndChunk()

	case dc_UserCommand:
		// Skip
		p.bitReader.ReadInt(32)
		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.bitReader.EndChunk()

	case dc_Signon:
		fallthrough
	case dc_Packet:
		// Booooring
		parseCommandInfo(p.bitReader)
		p.bitReader.ReadInt(32) // SeqNrIn
		p.bitReader.ReadInt(32) // SeqNrOut

		p.bitReader.BeginChunk(p.bitReader.ReadSignedInt(32) << 3)
		p.parsePacket()
		p.bitReader.EndChunk()

	case dc_CustomData:
		fmt.Fprintf(os.Stderr, "WARNING: Found CustomData but not handled\n")

	default:
		panic(fmt.Sprintf("Canny handle it anymoe (command %v unknown)", cmd))
	}
	return true
}

func (p *Parser) parseStringTables() {
	tables := int(p.bitReader.ReadSingleByte())
	for i := 0; i < tables; i++ {
		tableName := p.bitReader.ReadString()
		p.parseSingleStringTable(tableName)
	}
}

func (p *Parser) parseSingleStringTable(name string) {
	strings := p.bitReader.ReadSignedInt(16)
	for i := 0; i < strings; i++ {
		stringName := p.bitReader.ReadString()
		if len(stringName) >= 100 {
			panic("Someone said that Roy said I should panic")
		}
		if p.bitReader.ReadBit() {
			userDataSize := p.bitReader.ReadSignedInt(16)
			data := p.bitReader.ReadBytes(userDataSize)
			switch name {
			case stName_UserInfo:
				player := common.ParsePlayerInfo(bytes.NewReader(data))
				pid, err := strconv.ParseInt(stringName, 10, 64)
				if err != nil {
					panic("Couldn't parse id from string")
				}
				p.rawPlayers[int(pid)] = player

			case stName_InstanceBaseline:
				pid, err := strconv.ParseInt(stringName, 10, 64)
				if err != nil {
					panic("Couldn't parse id from string")
				}
				p.instanceBaselines[int(pid)] = data

			case stName_ModelPreCache:
				p.modelPreCache = append(p.modelPreCache, stringName)

			default:
				// Irrelevant table
			}
		}
	}
	// Client side stuff, dgaf
	if p.bitReader.ReadBit() {
		strings2 := p.bitReader.ReadSignedInt(16)
		for i := 0; i < strings2; i++ {
			p.bitReader.ReadString()
			if p.bitReader.ReadBit() {
				p.bitReader.ReadBytes(p.bitReader.ReadSignedInt(16))
			}
		}
	}
}

func (p *Parser) mapEquipment() {
	for _, sc := range p.stParser.ServerClasses() {
		if len(sc.BaseClasses) > 6 && sc.BaseClasses[6].Name == "CWeaponCSBase" {
			if len(sc.BaseClasses) > 7 {
				switch sc.BaseClasses[7].Name {
				case "CWeaponCSBaseGun":
					// Most guns
					p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DTName[9:]))
				case "CBaseCSGrenade":
					// Nades
					p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DTName[3:]))
				}
			} else if sc.Name == "CKnife" || (len(sc.BaseClasses) > 6 && sc.BaseClasses[6].Name == "CKnife") {
				p.equipmentMapping[sc] = common.EE_Knife
			} else {
				switch sc.Name {
				case "CC4":
					p.equipmentMapping[sc] = common.EE_Bomb

				case "CWeaponNOVA":
					fallthrough
				case "CWeaponSawedoff":
					fallthrough
				case "CWeaponXM1014":
					p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.Name[7:]))
				}
			}
		}
	}
}

// Everything down here feels fucked

func (p *Parser) bindEntities() {
	p.handleTeamScores()
	p.handleBombSites()
	p.handlePlayers()
	p.handleWeapons()
}

func (p *Parser) handleTeamScores() {
	p.stParser.FindServerClassByName("CCSTeam").RegisterEntityCreatedHandler(func(event st.EntityCreatedEvent) {
		teamID := -1
		var clanName string
		var flagImage string
		score := 0

		event.Entity.FindProperty("m_iTeamNum").RegisterPropertyUpdateHandler(func(val st.PropValue) {
			teamID = val.IntVal
		})
		event.Entity.FindProperty("m_szClanTeamname").RegisterPropertyUpdateHandler(func(val st.PropValue) {
			clanName = val.StringVal
		})
		event.Entity.FindProperty("m_szTeamFlagImage").RegisterPropertyUpdateHandler(func(val st.PropValue) {
			flagImage = val.StringVal
		})
		event.Entity.FindProperty("m_scoreTotal").RegisterPropertyUpdateHandler(func(val st.PropValue) {
			score = val.IntVal
		})

		event.Entity.FindProperty("m_szTeamname").RegisterPropertyUpdateHandler(func(val st.PropValue) {
			team := val.StringVal

			var s *TeamState
			var t common.Team

			switch team {
			case teamName_Ct:
				s = &p.ctState
				t = common.Team_CounterTerrorists

			case teamName_Terrorist:
				s = &p.tState
				t = common.Team_Terrorists

			case teamName_Unassigned: // Ignore
			case teamName_Spectator: // Ignore

			default:
				panic(fmt.Sprintf("Unexpected team %q", team))
			}

			if s != nil {
				// Set values that were already updated
				s.id = teamID
				s.clanName = clanName
				s.flag = flagImage
				s.score = score

				// Register direct updates for the future
				event.Entity.FindProperty("m_iTeamNum").RegisterPropertyUpdateHandler(func(val st.PropValue) {
					s.id = val.IntVal
				})
				event.Entity.FindProperty("m_szClanTeamname").RegisterPropertyUpdateHandler(func(val st.PropValue) {
					s.clanName = val.StringVal
				})
				event.Entity.FindProperty("m_szTeamFlagImage").RegisterPropertyUpdateHandler(func(val st.PropValue) {
					s.flag = val.StringVal
				})
				event.Entity.FindProperty("m_scoreTotal").RegisterPropertyUpdateHandler(func(val st.PropValue) {
					s.score = val.IntVal
				})

				// FIXME: This only sets the team at the start. . . We also have a player-specific update handler that changes the team so maybe this is unnecessary?
				if teamID != -1 {
					s.id = teamID
					for _, pl := range p.players {
						if pl != nil && pl.TeamID == teamID {
							pl.Team = t
						}
					}
				}
			}
		})
	})
}

func (p *Parser) handleBombSites() {
	p.stParser.FindServerClassByName("CCSPlayerResource").RegisterEntityCreatedHandler(func(playerResource st.EntityCreatedEvent) {
		playerResource.Entity.FindProperty("m_bombsiteCenterA").RegisterPropertyUpdateHandler(func(center st.PropValue) {
			p.bombsiteA.center = center.VectorVal
		})
		playerResource.Entity.FindProperty("m_bombsiteCenterB").RegisterPropertyUpdateHandler(func(center st.PropValue) {
			p.bombsiteB.center = center.VectorVal
		})
	})

	p.stParser.FindServerClassByName("CBaseTrigger").RegisterEntityCreatedHandler(func(baseTrigger st.EntityCreatedEvent) {
		// TODO: Switch triggers to map[int]boundingBoxInformation?
		t := &boundingBoxInformation{index: baseTrigger.Entity.ID}
		p.triggers = append(p.triggers, t)

		baseTrigger.Entity.FindProperty("m_Collision.m_vecMins").RegisterPropertyUpdateHandler(func(vec st.PropValue) {
			t.min = vec.VectorVal
		})
		baseTrigger.Entity.FindProperty("m_Collision.m_vecMaxs").RegisterPropertyUpdateHandler(func(vec st.PropValue) {
			t.max = vec.VectorVal
		})
	})
}

func (p *Parser) handlePlayers() {
	p.stParser.FindServerClassByName("CCSPlayer").RegisterEntityCreatedHandler(func(e st.EntityCreatedEvent) {
		p.handleNewPlayer(e.Entity)
	})

	p.stParser.FindServerClassByName("CCSPlayerResource").RegisterEntityCreatedHandler(func(pr st.EntityCreatedEvent) {
		for i := 0; i < maxPlayers; i++ {
			i2 := i // Copy so it stays the same (for passing to handlers)
			iStr := fmt.Sprintf("%03d", i)

			pr.Entity.FindProperty("m_szClan." + iStr).RegisterPropertyUpdateHandler(func(val st.PropValue) {
				p.additionalPlayerInfo[i2].ClanTag = val.StringVal
			})

			setIntLazy := func(prop string, setter func(int)) {
				pr.Entity.FindProperty(prop).RegisterPropertyUpdateHandler(func(val st.PropValue) {
					setter(val.IntVal)
				})
			}

			setIntLazy("m_iPing."+iStr, func(val int) { p.additionalPlayerInfo[i2].Ping = val })
			setIntLazy("m_iScore."+iStr, func(val int) { p.additionalPlayerInfo[i2].Score = val })
			setIntLazy("m_iKills."+iStr, func(val int) { p.additionalPlayerInfo[i2].Kills = val })
			setIntLazy("m_iDeaths."+iStr, func(val int) { p.additionalPlayerInfo[i2].Deaths = val })
			setIntLazy("m_iAssists."+iStr, func(val int) { p.additionalPlayerInfo[i2].Assists = val })
			setIntLazy("m_iMVPs."+iStr, func(val int) { p.additionalPlayerInfo[i2].MVPs = val })
			setIntLazy("m_iTotalCashSpent."+iStr, func(val int) { p.additionalPlayerInfo[i2].TotalCashSpent = val })
		}
	})
}

func (p *Parser) handleNewPlayer(playerEntity *st.Entity) {
	var pl *common.Player
	if p.players[playerEntity.ID-1] != nil {
		pl = p.players[playerEntity.ID-1]
	} else {
		pl = common.NewPlayer()
		p.players[playerEntity.ID-1] = pl
		pl.SteamID = -1
		pl.Name = "unconnected"
	}

	pl.EntityID = playerEntity.ID
	pl.Entity = playerEntity

	playerEntity.FindProperty("cslocaldata.m_vecOrigin").RegisterPropertyUpdateHandler(func(val st.PropValue) {
		pl.Position.X = val.VectorVal.X
		pl.Position.Y = val.VectorVal.Y
	})

	playerEntity.FindProperty("cslocaldata.m_vecOrigin[2]").RegisterPropertyUpdateHandler(func(val st.PropValue) {
		pl.Position.Z = val.VectorVal.Z
	})

	playerEntity.FindProperty("m_iTeamNum").RegisterPropertyUpdateHandler(func(val st.PropValue) {
		pl.TeamID = val.IntVal

		// FIXME: We could probably just cast TeamID to common.Team or not even set it because the teamIDs should be the same. . . needs testing
		switch pl.TeamID {
		case p.ctState.id:
			pl.Team = common.Team_CounterTerrorists
		case p.tState.id:
			pl.Team = common.Team_Terrorists
		default:
			pl.Team = common.Team_Spectators
		}
	})

	// Some helpers because I cant be arsed
	setIntLazy := func(prop string, setter func(int)) {
		playerEntity.FindProperty(prop).RegisterPropertyUpdateHandler(func(val st.PropValue) {
			setter(val.IntVal)
		})
	}

	setFloatLazy := func(prop string, setter func(float32)) {
		playerEntity.FindProperty(prop).RegisterPropertyUpdateHandler(func(val st.PropValue) {
			setter(val.FloatVal)
		})
	}

	setFloat64Lazy := func(prop string, setter func(float64)) {
		playerEntity.FindProperty(prop).RegisterPropertyUpdateHandler(func(val st.PropValue) {
			setter(float64(val.FloatVal))
		})
	}

	setIntLazy("m_iHealth", func(val int) { pl.Hp = val })
	setIntLazy("m_ArmorValue", func(val int) { pl.Armor = val })
	setIntLazy("m_bHasDefuser", func(val int) { pl.HasDefuseKit = val == 1 })
	setIntLazy("m_bHasHelmet", func(val int) { pl.HasHelmet = val == 1 })
	setIntLazy("localdata.m_Local.m_bDucking", func(val int) { pl.IsDucking = val == 1 })
	setIntLazy("m_iAccount", func(val int) { pl.Money = val })

	setFloatLazy("m_angEyeAngles[1]", func(val float32) { pl.ViewDirectionX = val })
	setFloatLazy("m_angEyeAngles[0]", func(val float32) { pl.ViewDirectionY = val })
	setFloatLazy("m_flFlashDuration", func(val float32) { pl.FlashDuration = val })

	setFloat64Lazy("localdata.m_vecVelocity[0]", func(val float64) { pl.Velocity.X = val })
	setFloat64Lazy("localdata.m_vecVelocity[1]", func(val float64) { pl.Velocity.Y = val })
	setFloat64Lazy("localdata.m_vecVelocity[2]", func(val float64) { pl.Velocity.Z = val })

	setIntLazy("m_unCurrentEquipmentValue", func(val int) { pl.CurrentEquipmentValue = val })
	setIntLazy("m_unRoundStartEquipmentValue", func(val int) { pl.RoundStartEquipmentValue = val })
	setIntLazy("m_unFreezetimeEndEquipmentValue", func(val int) { pl.FreezetimeEndEquipmentValue = val })

	wepPrefix := playerWeaponPrePrefix + playerWeaponPrefix

	for _, prop := range playerEntity.Props() {
		if prop.Entry().Name() == playerWeaponPrefix+"000" {
			wepPrefix = playerWeaponPrefix
			break
		}
	}

	var cache [maxWeapons]int

	for i, v := range cache {
		i2 := i // Copy for passing to handler
		playerEntity.FindProperty(wepPrefix + fmt.Sprintf("%03d", i)).RegisterPropertyUpdateHandler(func(val st.PropValue) {
			idx := val.IntVal & common.IndexMask
			if idx != common.IndexMask {
				if v != 0 {
					// Player already has a weapon in this slot.
					pl.RawWeapons[cache[i2]] = nil
					cache[i2] = 0
				}
				cache[i2] = idx
				p.attributeWeapon(idx, pl)
			} else {
				if cache[i2] != 0 && pl.RawWeapons[cache[i2]] != nil {
					pl.RawWeapons[cache[i2]].Owner = nil
				}
				pl.RawWeapons[cache[i2]] = nil
			}
			cache[i2] = 0
		})
	}

	setIntLazy("m_hActiveWeapon", func(val int) { pl.ActiveWeaponID = val & common.IndexMask })

	for i := 0; i < 32; i++ {
		i2 := i // Copy so it stays the same
		setIntLazy("m_iAmmo."+fmt.Sprintf("%03d", i2), func(val int) { pl.AmmoLeft[i2] = val })
	}
}

func (p *Parser) attributeWeapon(index int, player *common.Player) {
	wep := &p.weapons[index]
	wep.Owner = player
	player.RawWeapons[index] = wep
}

func (p *Parser) handleWeapons() {
	for i := 0; i < maxEntities; i++ {
		p.weapons[i] = common.NewEquipment("")
	}

	for _, sc := range p.stParser.ServerClasses() {
		for _, bc := range sc.BaseClasses {
			if bc.Name == "CWeaponCSBase" {
				sc.RegisterEntityCreatedHandler(p.handleWeapon)
			}
		}
	}
}

func (p *Parser) handleWeapon(event st.EntityCreatedEvent) {
	eq := p.weapons[event.Entity.ID]
	eq.EntityID = event.Entity.ID
	eq.Weapon = p.equipmentMapping[event.ServerClass]
	eq.AmmoInMagazine = -1

	event.Entity.FindProperty("m_iClip1").RegisterPropertyUpdateHandler(func(val st.PropValue) {
		eq.AmmoInMagazine = val.IntVal - 1
	})

	event.Entity.FindProperty("LocalWeaponData.m_iPrimaryAmmoType").RegisterPropertyUpdateHandler(func(val st.PropValue) {
		eq.AmmoType = val.IntVal
	})

	wepFix := func(ok string, change string, changer func()) {
		event.Entity.FindProperty("m_nModelIndex").RegisterPropertyUpdateHandler(func(val st.PropValue) {
			eq.OriginalString = p.modelPreCache[val.IntVal]
			if strings.Contains(eq.OriginalString, ok) {
				// That's already ok!
			} else if strings.Contains(eq.OriginalString, change) {
				changer()
			} else {
				panic(fmt.Sprintf("Unknown weapon model %q", eq.OriginalString))
			}
		})
	}

	// FIXME: Deag/R8???
	switch eq.Weapon {
	case common.EE_P2000:
		wepFix("_pist_hkp2000", "_pist_223", func() { eq.Weapon = common.EE_USP })
	case common.EE_M4A4:
		wepFix("_rif_m4a1", "_rif_m4a1_s", func() { eq.Weapon = common.EE_M4A1 })
	case common.EE_P250:
		wepFix("_pist_p250", "_pist_cz_75", func() { eq.Weapon = common.EE_CZ })
	}
}
