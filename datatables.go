package demoinfocs

import (
	"fmt"
	"strings"

	r3 "github.com/golang/geo/r3"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

const (
	maxEdictBits                 = 11
	entityHandleIndexMask        = ((1 << maxEdictBits) - 1)
	entityHandleSerialNumberBits = 10
	entityHandleBits             = maxEdictBits + entityHandleSerialNumberBits
	invalidEntityHandle          = (1 << entityHandleBits) - 1
	maxEntities                  = (1 << maxEdictBits)
	maxPlayers                   = 64
	maxWeapons                   = 64
)

func (p *Parser) mapEquipment() {
	for _, sc := range p.stParser.ServerClasses() {
		baseClasses := sc.BaseClasses()
		for _, bc := range baseClasses {
			if bc.Name() == "CBaseGrenade" { // Grenades projectiles, i.e. thrown by player
				p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DataTableName()[3:]))
			}
		}

		if len(baseClasses) > 6 && baseClasses[6].Name() == "CWeaponCSBase" {
			if len(baseClasses) > 7 {
				switch baseClasses[7].Name() {
				case "CWeaponCSBaseGun":
					// Most guns
					p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DataTableName()[9:]))
				case "CBaseCSGrenade":
					// Nades
					p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DataTableName()[3:]))
				}
			} else if sc.Name() == "CKnife" || (len(baseClasses) > 6 && baseClasses[6].Name() == "CKnife") {
				p.equipmentMapping[sc] = common.EqKnife
			} else {
				switch sc.Name() {
				case "CC4":
					p.equipmentMapping[sc] = common.EqBomb
				case "CWeaponNOVA":
					fallthrough
				case "CWeaponSawedoff":
					fallthrough
				case "CWeaponXM1014":
					p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.Name()[7:]))
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
	p.bindBomb()
}

func (p *Parser) bindBomb() {
	bomb := &p.gameState.bomb

	// Track bomb when it is not held by a player
	scDroppedC4 := p.stParser.ServerClasses().FindByName("CC4")
	scDroppedC4.OnEntityCreated(func(bomb *st.Entity) {
		bomb.OnPositionUpdate(func(pos r3.Vector) {
			// Bomb only has a position when not held by a player
			p.gameState.bomb.Carrier = nil

			p.gameState.bomb.LastOnGroundPosition = pos
		})
	})

	// Track bomb when it has been planted
	scPlantedC4 := p.stParser.ServerClasses().FindByName("CPlantedC4")
	scPlantedC4.OnEntityCreated(func(bombEntity *st.Entity) {
		// Player can't hold the bomb when it has been planted
		p.gameState.bomb.Carrier = nil

		bomb.LastOnGroundPosition = bombEntity.Position()
	})

	// Track bomb when it is being held by a player
	scPlayerC4 := p.stParser.ServerClasses().FindByName("CC4")
	scPlayerC4.OnEntityCreated(func(bombEntity *st.Entity) {
		bombEntity.FindProperty("m_hOwner").OnUpdate(func(val st.PropertyValue) {
			bomb.Carrier = p.gameState.Participants().FindByHandle(val.IntVal)
		})
	})
}

func (p *Parser) bindTeamScores() {
	p.stParser.ServerClasses().FindByName("CCSTeam").OnEntityCreated(func(entity *st.Entity) {
		teamID := -1
		var clanName string
		var flagImage string
		score := 0

		entity.BindProperty("m_iTeamNum", &teamID, st.ValTypeInt)
		entity.BindProperty("m_szClanTeamname", &clanName, st.ValTypeString)
		entity.BindProperty("m_szTeamFlagImage", &flagImage, st.ValTypeString)
		entity.BindProperty("m_scoreTotal", &score, st.ValTypeInt)

		entity.FindProperty("m_szTeamname").OnUpdate(func(val st.PropertyValue) {
			team := val.StringVal

			var s *common.TeamState

			switch team {
			case "CT":
				s = &p.gameState.ctState

			case "TERRORIST":
				s = &p.gameState.tState

			case "Unassigned": // Ignore
			case "Spectator": // Ignore

			default:
				panic(fmt.Sprintf("Unexpected team %q", team))
			}

			if s != nil {
				// Set values that were already updated
				s.ID = teamID
				s.ClanName = clanName
				s.Flag = flagImage
				s.Score = score

				// Register direct updates for the future
				// Except for teamId, it doesn't change; players swap teams instead
				entity.BindProperty("m_szClanTeamname", &s.ClanName, st.ValTypeString)
				entity.BindProperty("m_szTeamFlagImage", &s.Flag, st.ValTypeString)

				entity.FindProperty("m_scoreTotal").OnUpdate(func(val st.PropertyValue) {
					s.Score = val.IntVal

					p.eventDispatcher.Dispatch(events.ScoreUpdated{
						TeamState: s,
					})
				})
			}
		})
	})
}

func (p *Parser) bindBombSites() {
	p.stParser.ServerClasses().FindByName("CCSPlayerResource").OnEntityCreated(func(playerResource *st.Entity) {
		playerResource.BindProperty("m_bombsiteCenterA", &p.bombsiteA.center, st.ValTypeVector)
		playerResource.BindProperty("m_bombsiteCenterB", &p.bombsiteB.center, st.ValTypeVector)
	})

	p.stParser.ServerClasses().FindByName("CBaseTrigger").OnEntityCreated(func(baseTrigger *st.Entity) {
		t := new(boundingBoxInformation)
		p.triggers[baseTrigger.ID()] = t

		baseTrigger.BindProperty("m_Collision.m_vecMins", &t.min, st.ValTypeVector)
		baseTrigger.BindProperty("m_Collision.m_vecMaxs", &t.max, st.ValTypeVector)
	})
}

func (p *Parser) bindPlayers() {
	p.stParser.ServerClasses().FindByName("CCSPlayer").OnEntityCreated(func(player *st.Entity) {
		p.bindNewPlayer(player)
	})

	p.stParser.ServerClasses().FindByName("CCSPlayerResource").OnEntityCreated(func(plInfo *st.Entity) {
		for i := 0; i < maxPlayers; i++ {
			i2 := i // Copy so it stays the same (for passing to handlers)
			iStr := fmt.Sprintf("%03d", i)

			plInfo.BindProperty("m_szClan."+iStr, &p.additionalPlayerInfo[i2].ClanTag, st.ValTypeString)
			plInfo.BindProperty("m_iPing."+iStr, &p.additionalPlayerInfo[i2].Ping, st.ValTypeInt)
			plInfo.BindProperty("m_iScore."+iStr, &p.additionalPlayerInfo[i2].Score, st.ValTypeInt)
			plInfo.BindProperty("m_iKills."+iStr, &p.additionalPlayerInfo[i2].Kills, st.ValTypeInt)
			plInfo.BindProperty("m_iDeaths."+iStr, &p.additionalPlayerInfo[i2].Deaths, st.ValTypeInt)
			plInfo.BindProperty("m_iAssists."+iStr, &p.additionalPlayerInfo[i2].Assists, st.ValTypeInt)
			plInfo.BindProperty("m_iMVPs."+iStr, &p.additionalPlayerInfo[i2].MVPs, st.ValTypeInt)
			plInfo.BindProperty("m_iTotalCashSpent."+iStr, &p.additionalPlayerInfo[i2].TotalCashSpent, st.ValTypeInt)
		}
	})
}

func (p *Parser) bindNewPlayer(playerEntity *st.Entity) {
	var pl *common.Player
	playerIndex := playerEntity.ID()
	if p.gameState.playersByEntityID[playerIndex] != nil {
		pl = p.gameState.playersByEntityID[playerIndex]
	} else {
		pl = common.NewPlayer()
		p.gameState.playersByEntityID[playerIndex] = pl
		pl.SteamID = -1
		pl.Name = "unconnected"
	}

	pl.EntityID = playerEntity.ID()
	pl.Entity = playerEntity

	playerEntity.OnDestroy(func() {
		delete(p.gameState.playersByEntityID, pl.EntityID)
	})

	// Position
	playerEntity.BindPosition(&pl.Position)

	// General info
	playerEntity.FindProperty("m_iTeamNum").OnUpdate(func(val st.PropertyValue) {
		pl.Team = common.Team(val.IntVal) // Need to cast to team so we can't use BindProperty
	})
	playerEntity.BindProperty("m_iHealth", &pl.Hp, st.ValTypeInt)
	playerEntity.BindProperty("m_ArmorValue", &pl.Armor, st.ValTypeInt)
	playerEntity.BindProperty("m_bHasDefuser", &pl.HasDefuseKit, st.ValTypeBoolInt)
	playerEntity.BindProperty("m_bHasHelmet", &pl.HasHelmet, st.ValTypeBoolInt)
	playerEntity.BindProperty("localdata.m_Local.m_bDucking", &pl.IsDucking, st.ValTypeBoolInt)
	playerEntity.BindProperty("m_iAccount", &pl.Money, st.ValTypeInt)

	playerEntity.BindProperty("m_angEyeAngles[1]", &pl.ViewDirectionX, st.ValTypeFloat32)
	playerEntity.BindProperty("m_angEyeAngles[0]", &pl.ViewDirectionY, st.ValTypeFloat32)
	playerEntity.BindProperty("m_flFlashDuration", &pl.FlashDuration, st.ValTypeFloat32)

	// Velocity
	playerEntity.BindProperty("localdata.m_vecVelocity[0]", &pl.Velocity.X, st.ValTypeFloat64)
	playerEntity.BindProperty("localdata.m_vecVelocity[1]", &pl.Velocity.Y, st.ValTypeFloat64)
	playerEntity.BindProperty("localdata.m_vecVelocity[2]", &pl.Velocity.Z, st.ValTypeFloat64)

	// Eq value
	playerEntity.BindProperty("m_unCurrentEquipmentValue", &pl.CurrentEquipmentValue, st.ValTypeInt)
	playerEntity.BindProperty("m_unRoundStartEquipmentValue", &pl.RoundStartEquipmentValue, st.ValTypeInt)
	playerEntity.BindProperty("m_unFreezetimeEndEquipmentValue", &pl.FreezetimeEndEquipmentValue, st.ValTypeInt)

	// Some demos have an additional prefix for player weapons weapon
	var wepPrefix string
	if playerEntity.FindProperty(playerWeaponPrefix+"000") != nil {
		wepPrefix = playerWeaponPrefix
	} else {
		wepPrefix = playerWeaponPrePrefix + playerWeaponPrefix
	}

	// Weapons
	var cache [maxWeapons]int
	for i := range cache {
		i2 := i // Copy for passing to handler
		playerEntity.FindProperty(wepPrefix + fmt.Sprintf("%03d", i)).OnUpdate(func(val st.PropertyValue) {
			entityID := val.IntVal & entityHandleIndexMask
			if entityID != entityHandleIndexMask {
				if cache[i2] != 0 {
					// Player already has a weapon in this slot.
					delete(pl.RawWeapons, cache[i2])
				}
				cache[i2] = entityID

				// Attribute weapon to player
				wep := &p.weapons[entityID]
				wep.Owner = pl
				pl.RawWeapons[entityID] = wep
			} else {
				if cache[i2] != 0 && pl.RawWeapons[cache[i2]] != nil {
					pl.RawWeapons[cache[i2]].Owner = nil
				}
				delete(pl.RawWeapons, cache[i2])

				cache[i2] = 0
			}
		})
	}

	// Active weapon
	playerEntity.FindProperty("m_hActiveWeapon").OnUpdate(func(val st.PropertyValue) {
		pl.ActiveWeaponID = val.IntVal & entityHandleIndexMask
	})

	for i := 0; i < 32; i++ {
		i2 := i // Copy so it stays the same
		playerEntity.BindProperty("m_iAmmo."+fmt.Sprintf("%03d", i2), &pl.AmmoLeft[i2], st.ValTypeInt)
	}
}

func (p *Parser) bindWeapons() {
	for _, sc := range p.stParser.ServerClasses() {
		for _, bc := range sc.BaseClasses() {
			switch bc.Name() {
			case "CWeaponCSBase":
				sc2 := sc // Local copy for loop
				sc.OnEntityCreated(func(e *st.Entity) { p.bindWeapon(e, p.equipmentMapping[sc2]) })
			case "CBaseGrenade": // Grenade that has been thrown by player.
				sc.OnEntityCreated(p.bindGrenadeProjectiles)
			case "CBaseCSGrenade":
				// @micvbang TODO: handle grenades dropped by dead player.
				// Grenades that were dropped by a dead player (and can be picked up by other players).
			}
		}
	}

	p.stParser.ServerClasses().FindByName("CInferno").OnEntityCreated(p.bindNewInferno)
}

// bindGrenadeProjectiles keeps track of the location of live grenades (Parser.gameState.grenadeProjectiles), actively thrown by players.
// It does NOT track the location of grenades lying on the ground, i.e. that were dropped by dead players.
func (p *Parser) bindGrenadeProjectiles(entity *st.Entity) {
	entityID := entity.ID()

	proj := common.NewGrenadeProjectile()
	proj.EntityID = entityID
	p.gameState.grenadeProjectiles[entityID] = proj

	entity.OnCreateFinished(func() {
		p.eventDispatcher.Dispatch(events.GrenadeProjectileThrow{
			Projectile: proj,
		})
	})

	entity.OnDestroy(func() {
		p.nadeProjectileDestroyed(proj)
	})

	entity.FindProperty("m_nModelIndex").OnUpdate(func(val st.PropertyValue) {
		proj.Weapon = p.grenadeModelIndices[val.IntVal]
	})

	// @micvbang: not quite sure what the difference between Thrower and Owner is.
	entity.FindProperty("m_hThrower").OnUpdate(func(val st.PropertyValue) {
		proj.Thrower = p.gameState.Participants().FindByHandle(val.IntVal)
	})

	entity.FindProperty("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
		proj.Owner = p.gameState.Participants().FindByHandle(val.IntVal)
	})

	entity.OnPositionUpdate(func(newPos r3.Vector) {
		proj.Position = newPos

		proj.Trajectory = append(proj.Trajectory, newPos)
	})

	// Some demos don't have this property as it seems
	// So we need to check for nil and can't send out bounce events if it's missing
	bounceProp := entity.FindProperty("m_nBounces")
	if bounceProp != nil {
		bounceProp.OnUpdate(func(val st.PropertyValue) {
			if val.IntVal != 0 {
				p.eventDispatcher.Dispatch(events.GrenadeProjectileBounce{
					Projectile: proj,
					BounceNr:   val.IntVal,
				})
			}
		})
	}
}

// Seperate function because we also use it in round_officially_ended (issue #42)
func (p *Parser) nadeProjectileDestroyed(proj *common.GrenadeProjectile) {
	// If the grenade projectile entity is destroyed AFTER round_officially_ended
	// we already executed this code when we received that event.
	if _, exists := p.gameState.grenadeProjectiles[proj.EntityID]; !exists {
		return
	}

	p.eventDispatcher.Dispatch(events.GrenadeProjectileDestroy{
		Projectile: proj,
	})

	delete(p.gameState.grenadeProjectiles, proj.EntityID)
}

func (p *Parser) bindWeapon(entity *st.Entity, wepType common.EquipmentElement) {
	entityID := entity.ID()
	p.weapons[entityID] = common.NewEquipment(wepType)
	eq := &p.weapons[entityID]
	eq.EntityID = entityID
	eq.AmmoInMagazine = -1

	entity.FindProperty("m_iClip1").OnUpdate(func(val st.PropertyValue) {
		eq.AmmoInMagazine = val.IntVal - 1
	})

	eq.AmmoType = entity.FindProperty("LocalWeaponData.m_iPrimaryAmmoType").Value().IntVal

	// Detect alternative weapons (P2k -> USP, M4A4 -> M4A1-S etc.)
	modelIndex := entity.FindProperty("m_nModelIndex").Value().IntVal
	eq.OriginalString = p.modelPreCache[modelIndex]

	wepFix := func(defaultName, altName string, alt common.EquipmentElement) {
		// Check 'altName' first because otherwise the m4a1_s is recognized as m4a4
		if strings.Contains(eq.OriginalString, altName) {
			eq.Weapon = alt
		} else if !strings.Contains(eq.OriginalString, defaultName) {
			panic(fmt.Sprintf("Unknown weapon model %q", eq.OriginalString))
		}
	}

	switch eq.Weapon {
	case common.EqP2000:
		wepFix("_pist_hkp2000", "_pist_223", common.EqUSP)
	case common.EqM4A4:
		wepFix("_rif_m4a1", "_rif_m4a1_s", common.EqM4A1)
	case common.EqP250:
		wepFix("_pist_p250", "_pist_cz_75", common.EqCZ)
	case common.EqDeagle:
		wepFix("_pist_deagle", "_pist_revolver", common.EqRevolver)
	}
}

func (p *Parser) bindNewInferno(entity *st.Entity) {
	entityID := entity.ID()
	inf := common.NewInferno()
	inf.EntityID = entityID
	p.gameState.infernos[entityID] = inf

	entity.OnCreateFinished(func() {
		p.eventDispatcher.Dispatch(events.InfernoStart{
			Inferno: inf,
		})
	})

	entity.OnDestroy(func() {
		p.infernoExpired(inf)
	})

	origin := entity.Position()
	nFires := 0
	entity.FindProperty("m_fireCount").OnUpdate(func(val st.PropertyValue) {
		for i := nFires; i < val.IntVal; i++ {
			iStr := fmt.Sprintf("%03d", i)
			offset := r3.Vector{
				X: float64(entity.FindProperty("m_fireXDelta." + iStr).Value().IntVal),
				Y: float64(entity.FindProperty("m_fireYDelta." + iStr).Value().IntVal),
				Z: float64(entity.FindProperty("m_fireZDelta." + iStr).Value().IntVal),
			}

			fire := &common.Fire{Vector: origin.Add(offset), IsBurning: true}
			entity.BindProperty("m_bFireIsBurning."+iStr, &fire.IsBurning, st.ValTypeBoolInt)

			inf.Fires = append(inf.Fires, fire)
		}
		nFires = val.IntVal
	})
}

// Seperate function because we also use it in round_officially_ended (issue #42)
func (p *Parser) infernoExpired(inf *common.Inferno) {
	// If the inferno entity is destroyed AFTER round_officially_ended
	// we already executed this code when we received that event.
	if _, exists := p.gameState.infernos[inf.EntityID]; !exists {
		return
	}

	p.eventDispatcher.Dispatch(events.InfernoExpired{
		Inferno: inf,
	})

	delete(p.gameState.infernos, inf.EntityID)
}
