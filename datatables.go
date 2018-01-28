package demoinfocs

import (
	"fmt"
	"strings"

	common "github.com/markus-wa/demoinfocs-golang/common"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// Everything here feels kinda fucked

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
				p.equipmentMapping[sc] = common.EqKnife
			} else {
				switch sc.Name {
				case "CC4":
					p.equipmentMapping[sc] = common.EqBomb

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

// Bind the attributes of the various entities to our structs on the parser
func (p *Parser) bindEntities() {
	p.bindTeamScores()
	p.bindBombSites()
	p.bindPlayers()
	p.bindWeapons()
}

func (p *Parser) bindTeamScores() {
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
			case "CT":
				s = &p.ctState
				t = common.TeamCounterTerrorists

			case "TERRORIST":
				s = &p.tState
				t = common.TeamTerrorists

			case "Unassigned": // Ignore
			case "Spectator": // Ignore

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

func (p *Parser) bindBombSites() {
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

func (p *Parser) bindPlayers() {
	p.stParser.FindServerClassByName("CCSPlayer").RegisterEntityCreatedHandler(func(e st.EntityCreatedEvent) {
		p.bindNewPlayer(e.Entity)
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

func (p *Parser) bindNewPlayer(playerEntity *st.Entity) {
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
			pl.Team = common.TeamCounterTerrorists
		case p.tState.id:
			pl.Team = common.TeamTerrorists
		default:
			pl.Team = common.TeamSpectators
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

	for i := range cache {
		i2 := i // Copy for passing to handler
		playerEntity.FindProperty(wepPrefix + fmt.Sprintf("%03d", i)).RegisterPropertyUpdateHandler(func(val st.PropValue) {
			idx := val.IntVal & indexMask
			if idx != indexMask {
				if cache[i2] != 0 {
					// Player already has a weapon in this slot.
					pl.RawWeapons[cache[i2]] = nil
				}
				cache[i2] = idx

				// Attribute weapon to player
				wep := &p.weapons[idx]
				wep.Owner = pl
				pl.RawWeapons[idx] = wep
			} else {
				if cache[i2] != 0 && pl.RawWeapons[cache[i2]] != nil {
					pl.RawWeapons[cache[i2]].Owner = nil
				}
				pl.RawWeapons[cache[i2]] = nil
			}
			cache[i2] = 0
		})
	}

	setIntLazy("m_hActiveWeapon", func(val int) { pl.ActiveWeaponID = val & indexMask })

	for i := 0; i < 32; i++ {
		i2 := i // Copy so it stays the same
		setIntLazy("m_iAmmo."+fmt.Sprintf("%03d", i2), func(val int) { pl.AmmoLeft[i2] = val })
	}
}

func (p *Parser) bindWeapons() {
	for i := 0; i < maxEntities; i++ {
		p.weapons[i] = common.NewEquipment("")
	}

	for _, sc := range p.stParser.ServerClasses() {
		for _, bc := range sc.BaseClasses {
			if bc.Name == "CWeaponCSBase" {
				sc.RegisterEntityCreatedHandler(p.bindWeapon)
			}
		}
	}
}

func (p *Parser) bindWeapon(event st.EntityCreatedEvent) {
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
	case common.EqP2000:
		wepFix("_pist_hkp2000", "_pist_223", func() { eq.Weapon = common.EqUSP })
	case common.EqM4A4:
		wepFix("_rif_m4a1", "_rif_m4a1_s", func() { eq.Weapon = common.EqM4A1 })
	case common.EqP250:
		wepFix("_pist_p250", "_pist_cz_75", func() { eq.Weapon = common.EqCZ })
	}
}
