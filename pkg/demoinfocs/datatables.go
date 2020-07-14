package demoinfocs

import (
	"fmt"
	"strings"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/go-unassert"

	constants "github.com/markus-wa/demoinfocs-golang/v2/internal/constants"
	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
)

func (p *parser) mapEquipment() {
	for _, sc := range p.stParser.ServerClasses() {
		switch sc.Name() {
		case "CC4":
			p.equipmentMapping[sc] = common.EqBomb
			continue

		case "CWeaponNOVA":
			fallthrough
		case "CWeaponSawedoff":
			fallthrough
		case "CWeaponXM1014":
			p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.Name()[7:]))
			continue

		case "CKnife":
			p.equipmentMapping[sc] = common.EqKnife
			continue
		}

		baseClasses := sc.BaseClasses()
		for _, bc := range baseClasses {
			if bc.Name() == "CBaseGrenade" { // Grenades projectiles, i.e. thrown by player
				p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DataTableName()[3:]))
				break
			}

			if bc.Name() == "CKnife" {
				p.equipmentMapping[sc] = common.EqKnife
				break
			}

			if bc.Name() == "CWeaponCSBaseGun" { // most guns
				p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DataTableName()[9:]))
				break
			}

			if bc.Name() == "CBaseCSGrenade" { // nades
				p.equipmentMapping[sc] = common.MapEquipment(strings.ToLower(sc.DataTableName()[3:]))
				break
			}
		}
	}
}

// Bind the attributes of the various entities to our structs on the parser
func (p *parser) bindEntities() {
	p.bindTeamStates()
	p.bindBombSites()
	p.bindPlayers()
	p.bindWeapons()
	p.bindBomb()
	p.bindGameRules()
}

func (p *parser) bindBomb() {
	bomb := &p.gameState.bomb

	// Track bomb when it is dropped on the ground or being held by a player
	scC4 := p.stParser.ServerClasses().FindByName("CC4")
	scC4.OnEntityCreated(func(bombEntity st.Entity) {
		bombEntity.OnPositionUpdate(func(pos r3.Vector) {
			// Bomb only has a position when not held by a player
			bomb.Carrier = nil

			bomb.LastOnGroundPosition = pos
		})

		bombEntity.Property("m_hOwner").OnUpdate(func(val st.PropertyValue) {
			bomb.Carrier = p.gameState.Participants().FindByHandle(val.IntVal)
		})

		bombEntity.Property("m_bStartedArming").OnUpdate(func(val st.PropertyValue) {
			if val.IntVal != 0 {
				p.gameState.currentPlanter = bomb.Carrier
			} else if p.gameState.currentPlanter != nil {
				p.gameState.currentPlanter.IsPlanting = false
				p.eventDispatcher.Dispatch(events.BombPlantAborted{Player: p.gameState.currentPlanter})
			}
		})
	})

	// Track bomb when it has been planted
	scPlantedC4 := p.stParser.ServerClasses().FindByName("CPlantedC4")
	scPlantedC4.OnEntityCreated(func(bombEntity st.Entity) {
		// Player can't hold the bomb when it has been planted
		p.gameState.bomb.Carrier = nil

		bomb.LastOnGroundPosition = bombEntity.Position()
	})
}

func (p *parser) bindTeamStates() {
	p.stParser.ServerClasses().FindByName("CCSTeam").OnEntityCreated(func(entity st.Entity) {
		team := entity.PropertyValueMust("m_szTeamname").StringVal

		var s *common.TeamState

		switch team {
		case "CT":
			s = &p.gameState.ctState

		case "TERRORIST":
			s = &p.gameState.tState

		case "Unassigned": // Ignore
		case "Spectator": // Ignore

		default:
			p.setError(fmt.Errorf("unexpected team %q", team))
		}

		if s != nil {
			s.Entity = entity

			// Register updates
			var score int
			entity.Property("m_scoreTotal").OnUpdate(func(val st.PropertyValue) {
				oldScore := score
				score = val.IntVal

				p.eventDispatcher.Dispatch(events.ScoreUpdated{
					OldScore:  oldScore,
					NewScore:  val.IntVal,
					TeamState: s,
				})
			})
		}
	})
}

func (p *parser) bindBombSites() {
	p.stParser.ServerClasses().FindByName("CCSPlayerResource").OnEntityCreated(func(playerResource st.Entity) {
		playerResource.BindProperty("m_bombsiteCenterA", &p.bombsiteA.center, st.ValTypeVector)
		playerResource.BindProperty("m_bombsiteCenterB", &p.bombsiteB.center, st.ValTypeVector)
	})

	p.stParser.ServerClasses().FindByName("CBaseTrigger").OnEntityCreated(func(baseTrigger st.Entity) {
		t := new(boundingBoxInformation)
		p.triggers[baseTrigger.ID()] = t

		baseTrigger.BindProperty("m_Collision.m_vecMins", &t.min, st.ValTypeVector)
		baseTrigger.BindProperty("m_Collision.m_vecMaxs", &t.max, st.ValTypeVector)
	})
}

func (p *parser) bindPlayers() {
	p.stParser.ServerClasses().FindByName("CCSPlayer").OnEntityCreated(func(player st.Entity) {
		p.bindNewPlayer(player)
	})

	p.stParser.ServerClasses().FindByName("CCSPlayerResource").OnEntityCreated(func(entity st.Entity) {
		p.playerResourceEntity = entity
	})
}

func (p *parser) getOrCreatePlayer(entityID int, rp *playerInfo) (isNew bool, player *common.Player) {
	player = p.gameState.playersByEntityID[entityID]

	if player == nil {
		if rp != nil {
			player = p.gameState.playersByUserID[rp.userID]

			if player == nil {
				isNew = true

				player = common.NewPlayer(p.demoInfoProvider)
				player.Name = rp.name
				player.SteamID64 = rp.xuid
				player.IsBot = rp.isFakePlayer || rp.guid == "BOT"
				player.UserID = rp.userID
			}
		} else {
			// see #162.
			// GOTV doesn't crash here either so we just initialize this player with default values.
			// this happens in some demos since November 2019 for players that were are actually connected.
			// in GOTV these players are just called "unknown".
			player = common.NewPlayer(p.demoInfoProvider)
			player.Name = "unknown"
			player.IsUnknown = true
		}
	}

	p.gameState.playersByEntityID[entityID] = player

	if rp != nil {
		p.gameState.playersByUserID[rp.userID] = player
	}

	return isNew, player
}

//nolint:funlen
func (p *parser) bindNewPlayer(playerEntity st.Entity) {
	entityID := playerEntity.ID()
	rp := p.rawPlayers[entityID-1]

	isNew, pl := p.getOrCreatePlayer(entityID, rp)

	pl.EntityID = entityID
	pl.Entity = playerEntity
	pl.IsConnected = true

	playerEntity.OnDestroy(func() {
		delete(p.gameState.playersByEntityID, entityID)
		pl.Entity = nil
	})

	// Position
	playerEntity.OnPositionUpdate(func(pos r3.Vector) {
		if pl.IsAlive() {
			pl.LastAlivePosition = pos
		}
	})

	// General info
	playerEntity.Property("m_iTeamNum").OnUpdate(func(val st.PropertyValue) {
		pl.Team = common.Team(val.IntVal)
		pl.TeamState = p.gameState.Team(pl.Team)
	})

	playerEntity.Property("m_flFlashDuration").OnUpdate(func(val st.PropertyValue) {
		if val.FloatVal == 0 {
			pl.FlashTick = 0
		} else {
			pl.FlashTick = p.gameState.ingameTick
		}

		pl.FlashDuration = val.FloatVal
	})

	p.bindPlayerWeapons(playerEntity, pl)

	// Active weapon
	playerEntity.Property("m_hActiveWeapon").OnUpdate(func(val st.PropertyValue) {
		pl.IsReloading = false
	})

	for i := 0; i < 32; i++ {
		i2 := i // Copy so it stays the same
		playerEntity.BindProperty("m_iAmmo."+fmt.Sprintf("%03d", i2), &pl.AmmoLeft[i2], st.ValTypeInt)
	}

	playerEntity.Property("m_bIsDefusing").OnUpdate(func(val st.PropertyValue) {
		if p.gameState.currentDefuser == pl && pl.IsDefusing && val.IntVal == 0 {
			p.eventDispatcher.Dispatch(events.BombDefuseAborted{Player: pl})
			p.gameState.currentDefuser = nil
		}

		pl.IsDefusing = val.IntVal != 0
	})

	spottedByMaskProp := playerEntity.Property("m_bSpottedByMask.000")
	if spottedByMaskProp != nil {
		spottersChanged := func(val st.PropertyValue) {
			p.eventDispatcher.Dispatch(events.PlayerSpottersChanged{Spotted: pl})
		}

		spottedByMaskProp.OnUpdate(spottersChanged)
		playerEntity.Property("m_bSpottedByMask.001").OnUpdate(spottersChanged)
	}

	if isNew && pl.SteamID64 != 0 {
		p.eventDispatcher.Dispatch(events.PlayerConnect{Player: pl})
	}
}

const maxWeapons = 64

func (p *parser) bindPlayerWeapons(playerEntity st.Entity, pl *common.Player) {
	// Some demos have an additional prefix for player weapons weapon
	var wepPrefix string
	if playerEntity.Property(playerWeaponPrefix+"000") != nil {
		wepPrefix = playerWeaponPrefix
	} else {
		wepPrefix = playerWeaponPrePrefix + playerWeaponPrefix
	}

	// Weapons
	var cache [maxWeapons]int
	for i := range cache {
		i2 := i // Copy for passing to handler
		playerEntity.Property(wepPrefix + fmt.Sprintf("%03d", i)).OnUpdate(func(val st.PropertyValue) {
			entityID := val.IntVal & constants.EntityHandleIndexMask
			if entityID != constants.EntityHandleIndexMask {
				if cache[i2] != 0 {
					// Player already has a weapon in this slot.
					delete(pl.Inventory, cache[i2])
				}
				cache[i2] = entityID

				wep := p.gameState.weapons[entityID]

				if wep == nil {
					// sometimes a weapon is assigned to a player before the weapon entity is created
					wep = common.NewEquipment(common.EqUnknown)
					p.gameState.weapons[entityID] = wep
				}

				// Clear previous owner
				if wep.Owner != nil && wep.Entity != nil {
					delete(wep.Owner.Inventory, wep.Entity.ID())
				}

				// Attribute weapon to player
				wep.Owner = pl
				pl.Inventory[entityID] = wep
			} else {
				if cache[i2] != 0 && pl.Inventory[cache[i2]] != nil {
					pl.Inventory[cache[i2]].Owner = nil
				}
				delete(pl.Inventory, cache[i2])

				cache[i2] = 0
			}
		})
	}
}

func (p *parser) bindWeapons() {
	for _, sc := range p.stParser.ServerClasses() {
		for _, bc := range sc.BaseClasses() {
			switch bc.Name() {
			case "CWeaponCSBase":
				sc2 := sc // Local copy for loop
				sc.OnEntityCreated(func(e st.Entity) { p.bindWeapon(e, p.equipmentMapping[sc2]) })
			case "CBaseGrenade": // Grenade that has been thrown by player.
				sc.OnEntityCreated(p.bindGrenadeProjectiles)
			case "CBaseCSGrenade":
				// @micvbang TODO: handle grenades dropped by dead player.
				// Grenades that were dropped by a dead player (and can be picked up by other players).
			} //nolint:wsl
		}
	}

	p.stParser.ServerClasses().FindByName("CInferno").OnEntityCreated(p.bindNewInferno)
}

// bindGrenadeProjectiles keeps track of the location of live grenades (parser.gameState.grenadeProjectiles), actively thrown by players.
// It does NOT track the location of grenades lying on the ground, i.e. that were dropped by dead players.
func (p *parser) bindGrenadeProjectiles(entity st.Entity) {
	entityID := entity.ID()

	proj := common.NewGrenadeProjectile()
	proj.Entity = entity
	p.gameState.grenadeProjectiles[entityID] = proj

	var wep common.EquipmentType
	entity.OnCreateFinished(func() { //nolint:wsl
		// copy the weapon so it doesn't get overwritten by a new entity in parser.weapons
		wepCopy := *(getPlayerWeapon(proj.Thrower, wep))
		proj.WeaponInstance = &wepCopy

		unassert.NotNilf(proj.WeaponInstance, "couldn't find grenade instance for player")
		if proj.WeaponInstance != nil {
			unassert.NotNilf(proj.WeaponInstance.Owner, "getPlayerWeapon() returned weapon instance with Owner=nil")
		}

		p.gameEventHandler.addThrownGrenade(proj.Thrower, proj.WeaponInstance)

		p.eventDispatcher.Dispatch(events.GrenadeProjectileThrow{
			Projectile: proj,
		})
	})

	entity.OnDestroy(func() {
		p.nadeProjectileDestroyed(proj)
	})

	entity.Property("m_nModelIndex").OnUpdate(func(val st.PropertyValue) {
		wep = p.grenadeModelIndices[val.IntVal]
	})

	// @micvbang: not quite sure what the difference between Thrower and Owner is.
	entity.Property("m_hThrower").OnUpdate(func(val st.PropertyValue) {
		proj.Thrower = p.gameState.Participants().FindByHandle(val.IntVal)
	})

	entity.Property("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
		proj.Owner = p.gameState.Participants().FindByHandle(val.IntVal)
	})

	entity.OnPositionUpdate(func(newPos r3.Vector) {
		proj.Trajectory = append(proj.Trajectory, newPos)
	})

	// Some demos don't have this property as it seems
	// So we need to check for nil and can't send out bounce events if it's missing
	if bounceProp := entity.Property("m_nBounces"); bounceProp != nil {
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

// Separate function because we also use it in round_officially_ended (issue #42)
func (p *parser) nadeProjectileDestroyed(proj *common.GrenadeProjectile) {
	// If the grenade projectile entity is destroyed AFTER round_officially_ended
	// we already executed this code when we received that event.
	if _, exists := p.gameState.grenadeProjectiles[proj.Entity.ID()]; !exists {
		return
	}

	p.eventDispatcher.Dispatch(events.GrenadeProjectileDestroy{
		Projectile: proj,
	})

	delete(p.gameState.grenadeProjectiles, proj.Entity.ID())

	if proj.WeaponInstance.Type == common.EqFlash {
		p.gameState.lastFlash.projectileByPlayer[proj.Owner] = proj
	}

	// We delete from the Owner.ThrownGrenades (only if not inferno or smoke, because they will be deleted when they expire)
	isInferno := proj.WeaponInstance.Type == common.EqMolotov || proj.WeaponInstance.Type == common.EqIncendiary
	isSmoke := proj.WeaponInstance.Type == common.EqSmoke
	isDecoy := proj.WeaponInstance.Type == common.EqDecoy

	if !isInferno && !isSmoke && !isDecoy {
		p.gameEventHandler.deleteThrownGrenade(proj.Thrower, proj.WeaponInstance.Type)
	}
}

func (p *parser) bindWeapon(entity st.Entity, wepType common.EquipmentType) {
	entityID := entity.ID()

	eq, eqExists := p.gameState.weapons[entityID]
	if !eqExists {
		eq = common.NewEquipment(wepType)
		p.gameState.weapons[entityID] = eq
	} else {
		// If we are here, we already have a player that holds this weapon
		// so the zero-valued Equipment instance was already created in bindPlayer().
		// In this case we should create update the weapon type
		// but keep the same memory address so player's rawWeapons would still have a pointer to it
		eq.Type = wepType
	}

	eq.Entity = entity

	entity.OnDestroy(func() {
		delete(p.gameState.weapons, entityID)
	})

	entity.Property("m_iClip1").OnUpdate(func(val st.PropertyValue) {
		if eq.Owner != nil {
			eq.Owner.IsReloading = false
		}
	})

	// Detect alternative weapons (P2k -> USP, M4A4 -> M4A1-S etc.)
	modelIndex := entity.Property("m_nModelIndex").Value().IntVal
	eq.OriginalString = p.modelPreCache[modelIndex]

	wepFix := func(altName string, alt common.EquipmentType) {
		// Check 'altName' first because otherwise the m4a1_s is recognized as m4a4
		if strings.Contains(eq.OriginalString, altName) {
			eq.Type = alt
		}
	}

	switch eq.Type {
	case common.EqP2000:
		wepFix("_pist_223", common.EqUSP)
	case common.EqM4A4:
		wepFix("_rif_m4a1_s", common.EqM4A1)
	case common.EqP250:
		wepFix("_pist_cz_75", common.EqCZ)
	case common.EqDeagle:
		wepFix("_pist_revolver", common.EqRevolver)
	case common.EqMP7:
		wepFix("_smg_mp5sd", common.EqMP5)
	}
}

func (p *parser) bindNewInferno(entity st.Entity) {
	inf := common.NewInferno(p.demoInfoProvider, entity)
	p.gameState.infernos[entity.ID()] = inf

	entity.OnCreateFinished(func() {
		p.eventDispatcher.Dispatch(events.InfernoStart{
			Inferno: inf,
		})
	})

	entity.OnDestroy(func() {
		p.infernoExpired(inf)
	})
}

// Separate function because we also use it in round_officially_ended (issue #42)
func (p *parser) infernoExpired(inf *common.Inferno) {
	// If the inferno entity is destroyed AFTER round_officially_ended
	// we already executed this code when we received that event.
	if _, exists := p.gameState.infernos[inf.Entity.ID()]; !exists {
		return
	}

	p.eventDispatcher.Dispatch(events.InfernoExpired{
		Inferno: inf,
	})

	delete(p.gameState.infernos, inf.Entity.ID())

	p.gameEventHandler.deleteThrownGrenade(inf.Thrower(), common.EqIncendiary)
}

//nolint:funlen
func (p *parser) bindGameRules() {
	grPrefix := func(s string) string {
		return fmt.Sprintf("%s.%s", gameRulesPrefix, s)
	}

	gameRules := p.ServerClasses().FindByName("CCSGameRulesProxy")
	gameRules.OnEntityCreated(func(entity st.Entity) {
		entity.Property(grPrefix("m_gamePhase")).OnUpdate(func(val st.PropertyValue) {
			oldGamePhase := p.gameState.gamePhase
			p.gameState.gamePhase = common.GamePhase(val.IntVal)

			p.eventDispatcher.Dispatch(events.GamePhaseChanged{
				OldGamePhase: oldGamePhase,
				NewGamePhase: p.gameState.gamePhase,
			})

			switch p.gameState.gamePhase {
			case common.GamePhaseTeamSideSwitch:
				p.eventDispatcher.Dispatch(events.TeamSideSwitch{})
			case common.GamePhaseGameHalfEnded:
				p.eventDispatcher.Dispatch(events.GameHalfEnded{})
			}
		})

		entity.BindProperty(grPrefix("m_totalRoundsPlayed"), &p.gameState.totalRoundsPlayed, st.ValTypeInt)
		entity.Property(grPrefix("m_bWarmupPeriod")).OnUpdate(func(val st.PropertyValue) {
			oldIsWarmupPeriod := p.gameState.isWarmupPeriod
			p.gameState.isWarmupPeriod = val.IntVal == 1

			p.eventDispatcher.Dispatch(events.IsWarmupPeriodChanged{
				OldIsWarmupPeriod: oldIsWarmupPeriod,
				NewIsWarmupPeriod: p.gameState.isWarmupPeriod,
			})
		})

		entity.Property(grPrefix("m_bHasMatchStarted")).OnUpdate(func(val st.PropertyValue) {
			oldMatchStarted := p.gameState.isMatchStarted
			p.gameState.isMatchStarted = val.IntVal == 1

			p.eventDispatcher.Dispatch(events.MatchStartedChanged{
				OldIsStarted: oldMatchStarted,
				NewIsStarted: p.gameState.isMatchStarted,
			})
		})

		// TODO: seems like this is more reliable than RoundEnd events
		// "m_eRoundWinReason"

		// TODO: future fields to use
		// "m_iRoundWinStatus"
		// "m_nOvertimePlaying"
		// "m_bGameRestart"
		// "m_MatchDevice"
		// "m_bHasMatchStarted"
		// "m_numBestOfMaps"
		// "m_fWarmupPeriodEnd"
		// "m_timeUntilNextPhaseStarts"

		// TODO: timeout data
		// "m_bTerroristTimeOutActive"
		// "m_bCTTimeOutActive"
		// "m_flTerroristTimeOutRemaining"
		// "m_flCTTimeOutRemaining"
		// "m_nTerroristTimeOuts"
		// "m_nCTTimeOuts"
	})
}
