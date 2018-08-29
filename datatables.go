package demoinfocs

import (
	"fmt"
	"strings"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

const (
	maxEdictBits = 11
	indexMask    = ((1 << maxEdictBits) - 1)
	maxEntities  = (1 << maxEdictBits)
	maxPlayers   = 64
	maxWeapons   = 64
)

// Everything here feels kinda fucked

func (p *Parser) mapEquipment() {
	for _, sc := range p.stParser.ServerClasses() {
		for _, bc := range sc.BaseClasses {
			if bc.Name == "CBaseGrenade" { // Grenades projectiles, i.e. thrown by player
				p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DTName[3:]))
			}
		}

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

		event.Entity.BindProperty("m_iTeamNum", &teamID, st.ValTypeInt)
		event.Entity.BindProperty("m_szClanTeamname", &clanName, st.ValTypeString)
		event.Entity.BindProperty("m_szTeamFlagImage", &flagImage, st.ValTypeString)
		event.Entity.BindProperty("m_scoreTotal", &score, st.ValTypeInt)

		event.Entity.FindProperty("m_szTeamname").OnUpdate(func(val st.PropValue) {
			team := val.StringVal

			var s *TeamState

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
				s.id = teamID
				s.clanName = clanName
				s.flag = flagImage
				s.score = score

				// Register direct updates for the future
				// Except for teamId, it doesn't change; players swap teams instead
				event.Entity.BindProperty("m_szClanTeamname", &s.clanName, st.ValTypeString)
				event.Entity.BindProperty("m_szTeamFlagImage", &s.flag, st.ValTypeString)
				event.Entity.BindProperty("m_scoreTotal", &s.score, st.ValTypeInt)
			}
		})
	})
}

func (p *Parser) bindBombSites() {
	p.stParser.FindServerClassByName("CCSPlayerResource").RegisterEntityCreatedHandler(func(playerResource st.EntityCreatedEvent) {
		playerResource.Entity.BindProperty("m_bombsiteCenterA", &p.bombsiteA.center, st.ValTypeVector)
		playerResource.Entity.BindProperty("m_bombsiteCenterB", &p.bombsiteB.center, st.ValTypeVector)
	})

	p.stParser.FindServerClassByName("CBaseTrigger").RegisterEntityCreatedHandler(func(baseTrigger st.EntityCreatedEvent) {
		t := new(boundingBoxInformation)
		p.triggers[baseTrigger.Entity.ID] = t

		baseTrigger.Entity.BindProperty("m_Collision.m_vecMins", &t.min, st.ValTypeVector)
		baseTrigger.Entity.BindProperty("m_Collision.m_vecMaxs", &t.max, st.ValTypeVector)
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

			pr.Entity.BindProperty("m_szClan."+iStr, &p.additionalPlayerInfo[i2].ClanTag, st.ValTypeString)
			pr.Entity.BindProperty("m_iPing."+iStr, &p.additionalPlayerInfo[i2].Ping, st.ValTypeInt)
			pr.Entity.BindProperty("m_iScore."+iStr, &p.additionalPlayerInfo[i2].Score, st.ValTypeInt)
			pr.Entity.BindProperty("m_iKills."+iStr, &p.additionalPlayerInfo[i2].Kills, st.ValTypeInt)
			pr.Entity.BindProperty("m_iDeaths."+iStr, &p.additionalPlayerInfo[i2].Deaths, st.ValTypeInt)
			pr.Entity.BindProperty("m_iAssists."+iStr, &p.additionalPlayerInfo[i2].Assists, st.ValTypeInt)
			pr.Entity.BindProperty("m_iMVPs."+iStr, &p.additionalPlayerInfo[i2].MVPs, st.ValTypeInt)
			pr.Entity.BindProperty("m_iTotalCashSpent."+iStr, &p.additionalPlayerInfo[i2].TotalCashSpent, st.ValTypeInt)
		}
	})
}

func (p *Parser) bindNewPlayer(playerEntity *st.Entity) {
	var pl *common.Player
	playerIndex := playerEntity.ID - 1
	if p.entityIDToPlayers[playerIndex] != nil {
		pl = p.entityIDToPlayers[playerIndex]
	} else {
		pl = common.NewPlayer()
		p.entityIDToPlayers[playerIndex] = pl
		pl.SteamID = -1
		pl.Name = "unconnected"
	}

	pl.EntityID = playerEntity.ID
	pl.Entity = playerEntity

	// Position
	playerEntity.FindProperty("cslocaldata.m_vecOrigin").OnUpdate(func(val st.PropValue) {
		pl.Position.X = val.VectorVal.X
		pl.Position.Y = val.VectorVal.Y
	})
	playerEntity.BindProperty("cslocaldata.m_vecOrigin[2]", &pl.Position.Z, st.ValTypeFloat64)

	// General info
	playerEntity.FindProperty("m_iTeamNum").OnUpdate(func(val st.PropValue) {
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

	// Weapons
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
		playerEntity.FindProperty(wepPrefix + fmt.Sprintf("%03d", i)).OnUpdate(func(val st.PropValue) {
			entityID := val.IntVal & indexMask
			if entityID != indexMask {
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
	playerEntity.FindProperty("m_hActiveWeapon").OnUpdate(func(val st.PropValue) {
		pl.ActiveWeaponID = val.IntVal & indexMask
	})

	for i := 0; i < 32; i++ {
		i2 := i // Copy so it stays the same
		playerEntity.BindProperty("m_iAmmo."+fmt.Sprintf("%03d", i2), &pl.AmmoLeft[i2], st.ValTypeInt)
	}
}

func (p *Parser) bindWeapons() {
	for _, sc := range p.stParser.ServerClasses() {
		for _, bc := range sc.BaseClasses {
			switch bc.Name {
			case "CWeaponCSBase":
				sc.RegisterEntityCreatedHandler(p.bindWeapon)
			case "CBaseGrenade": // Grenade that has been thrown by player.
				sc.RegisterEntityCreatedHandler(p.bindGrenadeProjectiles)
			case "CBaseCSGrenade":
				// @micvbang TODO: handle grenades dropped by dead player.
				// Grenades that were dropped by a dead player (and can be picked up by other players).
			}
		}
	}

}

// bindGrenadeProjectiles keeps track of the location of live grenades (Parser.gameState.grenadeProjectiles), actively thrown by players.
// It does NOT track the location of grenades lying on the ground, i.e. that were dropped by dead players.
func (p *Parser) bindGrenadeProjectiles(event st.EntityCreatedEvent) {
	p.gameState.grenadeProjectiles[event.Entity.ID] = common.NewGrenadeProjectile()

	event.Entity.OnDestroy(func() {
		delete(p.gameState.grenadeProjectiles, event.Entity.ID)
	})

	proj := p.gameState.grenadeProjectiles[event.Entity.ID]
	proj.EntityID = event.Entity.ID

	event.Entity.FindProperty("m_nModelIndex").OnUpdate(func(val st.PropValue) {
		proj.Weapon = p.grenadeModelIndices[val.IntVal]
	})

	// @micvbang: not quite sure what the difference between Thrower and Owner is.
	event.Entity.FindProperty("m_hThrower").OnUpdate(func(val st.PropValue) {
		throwerID := val.IntVal & indexMask
		throwerIndex := throwerID - 1

		thrower := p.entityIDToPlayers[throwerIndex]
		proj.Thrower = thrower
	})

	event.Entity.FindProperty("m_hOwnerEntity").OnUpdate(func(val st.PropValue) {
		ownerID := val.IntVal & indexMask
		ownerIndex := ownerID - 1
		player := p.entityIDToPlayers[ownerIndex]
		proj.Owner = player
	})

	event.Entity.FindProperty("m_vecOrigin").OnUpdate(func(st.PropValue) {
		proj.Position = event.Entity.Position()
	})

	// Some demos don't have this property as it seems
	// So we need to check for nil and can't send out bounce events if it's missing
	bounceProp := event.Entity.FindProperty("m_nBounces")
	if bounceProp != nil {
		bounceProp.OnUpdate(func(val st.PropValue) {
			if val.IntVal != 0 {
				p.eventDispatcher.Dispatch(events.NadeProjectileBouncedEvent{
					Projectile: proj,
					BounceNr:   val.IntVal,
				})
			}
		})
	}

	event.Entity.OnCreateFinished(func() {
		p.eventDispatcher.Dispatch(events.NadeProjectileThrownEvent{
			Projectile: proj,
		})
	})
}

// Seperate function because we also use it in round_officially_ended (issue #42)
func (p *Parser) nadeProjectileDestroyed(proj *common.GrenadeProjectile) {
	// If the grenade projectile entity is destroyed AFTER round_officially_ended
	// we already executed this code when we received that event.
	if _, exists := p.gameState.grenadeProjectiles[proj.EntityID]; !exists {
		return
	}

	delete(p.gameState.grenadeProjectiles, proj.EntityID)
}

func (p *Parser) bindWeapon(event st.EntityCreatedEvent) {
	p.weapons[event.Entity.ID] = common.NewEquipment("")
	eq := &p.weapons[event.Entity.ID]
	eq.EntityID = event.Entity.ID
	eq.Weapon = p.equipmentMapping[event.ServerClass]
	eq.AmmoInMagazine = -1

	event.Entity.FindProperty("m_iClip1").OnUpdate(func(val st.PropValue) {
		eq.AmmoInMagazine = val.IntVal - 1
	})

	event.Entity.BindProperty("LocalWeaponData.m_iPrimaryAmmoType", &eq.AmmoType, st.ValTypeInt)

	// Detect alternative weapons (P2k -> USP, M4A4 -> M4A1-S etc.)
	wepFix := func(defaultName, altName string, alt common.EquipmentElement) {
		event.Entity.FindProperty("m_nModelIndex").OnUpdate(func(val st.PropValue) {
			eq.OriginalString = p.modelPreCache[val.IntVal]
			// Check 'altName' first because otherwise the m4a1_s is recognized as m4a4
			if strings.Contains(eq.OriginalString, altName) {
				eq.Weapon = alt
			} else if !strings.Contains(eq.OriginalString, defaultName) {
				panic(fmt.Sprintf("Unknown weapon model %q", eq.OriginalString))
			}
		})
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
