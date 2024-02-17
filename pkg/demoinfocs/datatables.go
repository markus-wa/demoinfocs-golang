package demoinfocs

import (
	"fmt"
	"math"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/go-unassert"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// Bind the attributes of the various entities to our structs on the parser
func (p *parser) bindEntities() {
	p.bindTeamStates()
	p.bindBombSites()
	p.bindPlayers()
	p.bindWeapons()
	p.bindBomb()
	p.bindGameRules()
	p.bindHostages()
}

func (p *parser) bindBomb() {
	bomb := &p.gameState.bomb

	// Track bomb when it is dropped on the ground or being held by a player
	scC4 := p.stParser.ServerClasses().FindByName("CC4")
	scC4.OnEntityCreated(func(bombEntity st.Entity) {
		bombEntity.OnPositionUpdate(func(pos r3.Vector) {
			bomb.LastOnGroundPosition = pos
		})

		bombEntity.Property("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
			carrier := p.gameState.Participants().FindByPawnHandle(val.Handle())
			if !p.disableMimicSource1GameEvents {
				if carrier != nil {
					p.eventDispatcher.Dispatch(events.BombPickup{
						Player: carrier,
					})
				} else if bomb.Carrier != nil {
					p.eventDispatcher.Dispatch(events.BombDropped{
						Player:   bomb.Carrier,
						EntityID: bomb.Carrier.EntityID,
					})
				}
			}

			bomb.Carrier = carrier
		})

		// Updated when a player starts/stops planting the bomb
		bombEntity.Property("m_bStartedArming").OnUpdate(func(val st.PropertyValue) {
			if val.BoolVal() {
				planter := p.gameState.Participants().FindByPawnHandle(bombEntity.PropertyValueMust("m_hOwnerEntity").Handle())
				planter.IsPlanting = true
				p.gameState.currentPlanter = planter

				siteNumber := p.gameState.currentPlanter.PlayerPawnEntity().PropertyValueMust("m_nWhichBombZone").Int()
				site := events.BomsiteUnknown
				switch siteNumber {
				case 1:
					site = events.BombsiteA
				case 2:
					site = events.BombsiteB
				case 0:
					site = p.getClosestBombsiteFromPosition(planter.Position())
				}

				if !p.disableMimicSource1GameEvents {
					p.eventDispatcher.Dispatch(events.BombPlantBegin{
						BombEvent: events.BombEvent{
							Player: p.gameState.currentPlanter,
							Site:   site,
						},
					})
				}
			} else if p.gameState.currentPlanter != nil {
				p.gameState.currentPlanter.IsPlanting = false
				p.eventDispatcher.Dispatch(events.BombPlantAborted{Player: p.gameState.currentPlanter})
			}
		})

		bombEntity.OnDestroy(func() {
			p.gameState.currentPlanter = nil
		})
	})

	// Track bomb when it has been planted
	scPlantedC4 := p.stParser.ServerClasses().FindByName("CPlantedC4")
	scPlantedC4.OnEntityCreated(func(bombEntity st.Entity) {
		// Player can't hold the bomb when it has been planted
		p.gameState.bomb.Carrier = nil
		p.gameState.currentPlanter = nil

		bomb.LastOnGroundPosition = bombEntity.Position()

		ownerProp := bombEntity.PropertyValueMust("m_hOwnerEntity")
		planter := p.gameState.Participants().FindByPawnHandle(ownerProp.Handle())
		if planter == nil {
			return
		}
		isTicking := true
		planter.IsPlanting = false

		siteNumber := bombEntity.PropertyValueMust("m_nBombSite").Int()
		site := events.BomsiteUnknown
		if siteNumber == 0 {
			site = events.BombsiteA
		} else if siteNumber == 1 {
			site = events.BombsiteB
		}

		if !p.disableMimicSource1GameEvents {
			p.eventDispatcher.Dispatch(events.BombPlanted{
				BombEvent: events.BombEvent{
					Player: planter,
					Site:   site,
				},
			})
		}

		// Set to true when the bomb has been planted and to false when it has been defused or has exploded.
		bombEntity.Property("m_bBombTicking").OnUpdate(func(val st.PropertyValue) {
			isTicking = val.BoolVal()
			if isTicking {
				return
			}

			// At this point the bomb stopped ticking either because it has been defused or has exploded.
			// We detect only explosions here, defuse events are detected with m_bBombDefused updates which seems more suitable.
			// When the bomb is defused, m_bBombTicking is set to false and then m_hBombDefuser is set to nil.
			// It means that if a player is currently defusing the bomb, it's a defuse event.
			isDefuseEvent := p.gameState.currentDefuser != nil
			if isDefuseEvent || p.disableMimicSource1GameEvents {
				return
			}

			p.eventDispatcher.Dispatch(events.BombExplode{
				BombEvent: events.BombEvent{
					Player: planter,
					Site:   site,
				},
			})
		})

		// Updated when a player starts/stops defusing the bomb
		bombEntity.Property("m_hBombDefuser").OnUpdate(func(val st.PropertyValue) {
			isValidPlayer := val.Handle() != constants.InvalidEntityHandleSource2
			if isValidPlayer {
				defuser := p.gameState.Participants().FindByPawnHandle(val.Handle())
				p.gameState.currentDefuser = defuser
				if !p.disableMimicSource1GameEvents {
					p.eventDispatcher.Dispatch(events.BombDefuseStart{
						Player: defuser,
						HasKit: defuser.HasDefuseKit(),
					})
				}
				return
			}

			isDefused := bombEntity.PropertyValueMust("m_bBombDefused").BoolVal()
			if !isDefused && p.gameState.currentDefuser != nil {
				p.eventDispatcher.Dispatch(events.BombDefuseAborted{
					Player: p.gameState.currentDefuser,
				})
			}

			p.gameState.currentDefuser = nil
		})

		// Updated when the bomb has been planted and defused.
		bombEntity.Property("m_bBombDefused").OnUpdate(func(val st.PropertyValue) {
			isDefused := val.BoolVal()
			if isDefused && !p.disableMimicSource1GameEvents {
				defuser := p.gameState.Participants().FindByPawnHandle(bombEntity.PropertyValueMust("m_hBombDefuser").Handle())
				p.eventDispatcher.Dispatch(events.BombDefused{
					BombEvent: events.BombEvent{
						Player: defuser,
						Site:   site,
					},
				})
			}
		})

		bombEntity.OnDestroy(func() {
			isTicking = true
			p.gameState.currentDefuser = nil
		})
	})
}

func (p *parser) bindTeamStates() {
	p.stParser.ServerClasses().FindByName("CCSTeam").OnEntityCreated(func(entity st.Entity) {
		teamVal := entity.PropertyValueMust("m_szTeamname")
		team := teamVal.String()

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
			var (
				scoreProp st.Property
				score     int

				clanName string
			)

			scoreProp = entity.Property("m_iScore")

			scoreProp.OnUpdate(func(val st.PropertyValue) {
				oldScore := score
				score = val.Int()

				p.eventDispatcher.Dispatch(events.ScoreUpdated{
					OldScore:  oldScore,
					NewScore:  val.Int(),
					TeamState: s,
				})
			})

			entity.Property("m_szClanTeamname").OnUpdate(func(val st.PropertyValue) {
				oldClanName := clanName
				clanName = val.Str()

				p.eventDispatcher.Dispatch(events.TeamClanNameUpdated{
					OldName:   oldClanName,
					NewName:   clanName,
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

	onBombTargetEntityCreated := func(target st.Entity) {
		t := new(boundingBoxInformation)
		p.triggers[target.ID()] = t

		minPropName := "m_vecMins"
		maxPropName := "m_vecMaxs"

		target.BindProperty(minPropName, &t.min, st.ValTypeVector)
		target.BindProperty(maxPropName, &t.max, st.ValTypeVector)
	}

	// CBombTarget is not available with CS2 demos created in the early days of the limited test.
	bombTargetClass := p.stParser.ServerClasses().FindByName("CBombTarget")
	if bombTargetClass != nil {
		bombTargetClass.OnEntityCreated(onBombTargetEntityCreated)
		return
	}

	p.stParser.ServerClasses().FindByName("CBaseTrigger").OnEntityCreated(onBombTargetEntityCreated)
}

func (p *parser) bindPlayers() {
	p.stParser.ServerClasses().FindByName("CCSPlayerController").OnEntityCreated(func(player st.Entity) {
		p.bindNewPlayerControllerS2(player)
	})
	p.stParser.ServerClasses().FindByName("CCSPlayerPawn").OnEntityCreated(func(player st.Entity) {
		p.bindNewPlayerPawnS2(player)
	})
}

func (p *parser) getOrCreatePlayer(entityID int, rp *common.PlayerInfo) (isNew bool, player *common.Player) {
	player = p.gameState.playersByEntityID[entityID]

	if player == nil {
		if rp != nil {
			player = p.gameState.playersByUserID[rp.UserID]

			if player == nil {
				isNew = true

				player = common.NewPlayer(p.demoInfoProvider)
				player.Name = rp.Name
				player.SteamID64 = rp.XUID
				player.IsBot = rp.IsFakePlayer || rp.GUID == "BOT"
				player.UserID = rp.UserID

				p.gameState.indexPlayerBySteamID(player)
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
		p.gameState.playersByUserID[rp.UserID] = player
	}

	return isNew, player
}

func (p *parser) getOrCreatePlayerFromControllerEntity(controllerEntity st.Entity) *common.Player {
	controllerEntityID := controllerEntity.ID()
	p.gameState.playerControllerEntities[controllerEntityID] = controllerEntity

	rp := p.rawPlayers[controllerEntityID-1]
	_, player := p.getOrCreatePlayer(controllerEntityID, rp)
	player.Entity = controllerEntity
	player.EntityID = controllerEntityID
	player.IsBot = controllerEntity.PropertyValueMust("m_steamID").String() == "0"

	if player.IsBot {
		player.Name = controllerEntity.PropertyValueMust("m_iszPlayerName").String()
		player.IsUnknown = false
	}

	return player
}

func (p *parser) bindNewPlayerControllerS2(controllerEntity st.Entity) {
	pl := p.getOrCreatePlayerFromControllerEntity(controllerEntity)

	controllerEntity.Property("m_hPawn").OnUpdate(func(val st.PropertyValue) {
		p.gameState.setPlayerLifeState(pl, pl.IsAlive())
	})

	controllerEntity.Property("m_iConnected").OnUpdate(func(val st.PropertyValue) {
		state := val.S2UInt32()
		wasConnected := pl.IsConnected
		pl.IsConnected = state == 0

		isDisconnection := state == 8
		if isDisconnection {
			for k, v := range p.rawPlayers {
				if v.XUID == pl.SteamID64 {
					delete(p.rawPlayers, k)
				}
			}
			p.gameEventHandler.dispatch(events.PlayerDisconnected{
				Player: pl,
			})
			return
		}

		isConnection := !wasConnected && pl.IsConnected
		if isConnection {
			if pl.SteamID64 != 0 {
				p.eventDispatcher.Dispatch(events.PlayerConnect{Player: pl})
			} else {
				p.eventDispatcher.Dispatch(events.BotConnect{Player: pl})
				playerInfo := common.PlayerInfo{
					XUID:         0,
					Name:         pl.Name,
					UserID:       pl.EntityID - 1,
					IsFakePlayer: true,
					IsHltv:       false,
				}
				p.setRawPlayer(pl.EntityID-1, playerInfo)
			}
		}
		p.gameState.setPlayerLifeState(pl, pl.IsAlive())
	})

	controllerEntity.Property("m_iTeamNum").OnUpdate(func(val st.PropertyValue) {
		team := val.S2UInt64()
		pl.Team = common.Team(val.S2UInt64())
		pl.TeamState = p.gameState.Team(pl.Team)

		if team < 2 {
			p.gameState.setPlayerLifeState(pl, false)
		}
	})

	controllerEntity.OnDestroy(func() {
		pl.IsConnected = false
		delete(p.gameState.playersByEntityID, controllerEntity.ID())
		p.gameState.setPlayerLifeState(pl, false)
	})
}

func (p *parser) bindNewPlayerPawnS2(pawnEntity st.Entity) {
	var prevControllerHandle uint64

	getPlayerFromPawnEntity := func(pawnEntity st.Entity) *common.Player {
		controllerProp, hasProp := pawnEntity.PropertyValue("m_hController")
		if !hasProp {
			return nil
		}

		return p.gameState.Participants().FindByHandle64(controllerProp.Handle())
	}

	pawnEntity.Property("m_hController").OnUpdate(func(controllerHandleVal st.PropertyValue) {
		controllerHandle := controllerHandleVal.Handle()
		if controllerHandle == constants.InvalidEntityHandleSource2 {
			return
		}

		controllerEntityID := int(controllerHandle & constants.EntityHandleIndexMaskSource2)
		controllerEntity := p.gameState.playerControllerEntities[controllerEntityID]
		pl := p.getOrCreatePlayerFromControllerEntity(controllerEntity)
		p.gameState.setPlayerLifeState(pl, pl.IsAlive())

		if controllerHandle == prevControllerHandle {
			return
		}

		prevControllerHandle = controllerHandle

		p.bindPlayerWeaponsS2(pawnEntity, pl)
	})

	// Position
	pawnEntity.OnPositionUpdate(func(pos r3.Vector) {
		pl := getPlayerFromPawnEntity(pawnEntity)
		if pl == nil {
			return
		}
		if pl.IsAlive() {
			pl.LastAlivePosition = pos
		}
	})

	pawnEntity.Property("m_flFlashDuration").OnUpdate(func(val st.PropertyValue) {
		pl := getPlayerFromPawnEntity(pawnEntity)
		if pl == nil {
			return
		}
		if val.Float() == 0 {
			pl.FlashTick = 0
		} else {
			pl.FlashTick = p.gameState.ingameTick
		}

		pl.FlashDuration = val.Float()
	})

	pawnEntity.Property("m_pWeaponServices.m_hActiveWeapon").OnUpdate(func(val st.PropertyValue) {
		pl := getPlayerFromPawnEntity(pawnEntity)
		if pl == nil {
			return
		}

		if pl.IsReloading {
			p.eventDispatcher.Dispatch(events.WeaponReloadEnd{
				Player: pl,
			})

			pl.IsReloading = false
		}
	})

	pawnEntity.Property("m_bIsDefusing").OnUpdate(func(val st.PropertyValue) {
		pl := getPlayerFromPawnEntity(pawnEntity)
		if pl == nil {
			return
		}
		pl.IsDefusing = val.BoolVal()
	})

	pawnEntity.Property("m_iHealth").OnUpdate(func(val st.PropertyValue) {
		pl := getPlayerFromPawnEntity(pawnEntity)
		if pl == nil {
			return
		}
		if val.Int() == 0 {
			p.gameState.setPlayerLifeState(pl, false)
			return
		}
		if pl.LifeState() == 0 {
			p.gameState.setPlayerLifeState(pl, true)
		}
	})

	if lifeStateProp := pawnEntity.Property("m_lifeState"); lifeStateProp != nil {
		lifeStateProp.OnUpdate(func(val st.PropertyValue) {
			pl := getPlayerFromPawnEntity(pawnEntity)
			if pl == nil {
				return
			}
			if val.S2UInt64() == 0 {
				p.gameState.setPlayerLifeState(pl, pl.Health() > 0)
				return
			}
			p.gameState.setPlayerLifeState(pl, false)
		})
	}

	if playerPawnProp := pawnEntity.Property("m_hPlayerPawn"); playerPawnProp != nil {
		playerPawnProp.OnUpdate(func(val st.PropertyValue) {
			pl := getPlayerFromPawnEntity(pawnEntity)
			if pl == nil {
				return
			}
			p.gameState.setPlayerLifeState(pl, pl.IsAlive())
		})
	}

	spottedByMaskProp := pawnEntity.Property("m_bSpottedByMask.0000")
	if spottedByMaskProp != nil {
		spottersChanged := func(val st.PropertyValue) {
			pl := getPlayerFromPawnEntity(pawnEntity)
			if pl == nil {
				return
			}

			p.eventDispatcher.Dispatch(events.PlayerSpottersChanged{Spotted: pl})
		}

		spottedByMaskProp.OnUpdate(spottersChanged)
		pawnEntity.Property("m_bSpottedByMask.0001").OnUpdate(spottersChanged)
	}
}

func (p *parser) bindPlayerWeaponsS2(pawnEntity st.Entity, pl *common.Player) {
	if pl.PlayerPawnEntity() == nil || pl.PlayerPawnEntity().ID() != pawnEntity.ID() {
		return
	}

	const inventoryCapacity = 64

	var inventorySize uint64 = 64

	type eq struct {
		*common.Equipment
		entityID int
	}

	playerInventory := make(map[int]eq)

	getWep := func(wepSlotPropertyValue st.PropertyValue) (uint64, *common.Equipment) {
		entityID := wepSlotPropertyValue.S2UInt64() & constants.EntityHandleIndexMaskSource2
		wep := p.gameState.weapons[int(entityID)]

		if wep == nil {
			// sometimes a weapon is assigned to a player before the weapon entity is created
			wep = common.NewEquipment(common.EqUnknown)
			p.gameState.weapons[int(entityID)] = wep
		}

		return entityID, wep
	}

	setPlayerInventory := func() {
		inventory := make(map[int]*common.Equipment, inventorySize)

		for i := uint64(0); i < inventorySize; i++ {
			val := pawnEntity.Property(playerWeaponPrefixS2 + fmt.Sprintf("%04d", i)).Value()
			if val.Any == nil {
				continue
			}

			entityID, wep := getWep(val)
			inventory[int(entityID)] = wep
		}

		pl.Inventory = inventory
	}

	pawnEntity.Property("m_pWeaponServices.m_hMyWeapons").OnUpdate(func(pv st.PropertyValue) {
		if val, ok := pv.Any.(uint64); ok {
			inventorySize = val
		}
		setPlayerInventory()
	})

	for i := 0; i < inventoryCapacity; i++ {
		i := i
		updateWeapon := func(val st.PropertyValue) {
			if val.Any == nil {
				return
			}

			entityID, wep := getWep(val)
			wep.Owner = pl

			entityWasCreated := entityID != constants.EntityHandleIndexMaskSource2

			if uint64(i) < inventorySize {
				if entityWasCreated {
					existingWeapon, exists := playerInventory[i]
					if exists {
						delete(pl.Inventory, existingWeapon.entityID)
					}

					pl.Inventory[int(entityID)] = wep
					playerInventory[i] = eq{
						Equipment: wep,
						entityID:  int(entityID),
					}
				} else {
					delete(pl.Inventory, int(entityID))
				}

				setPlayerInventory()
			}
		}

		property := pawnEntity.Property(playerWeaponPrefixS2 + fmt.Sprintf("%04d", i))
		updateWeapon(property.Value())
		property.OnUpdate(updateWeapon)
	}
}

func (p *parser) bindWeapons() {
	for _, sc := range p.stParser.ServerClasses().All() {
		hasIndexProp := false
		hasClipProp := false
		hasThrower := false

		for _, prop := range sc.PropertyEntries() {
			if prop == "m_iItemDefinitionIndex" {
				hasIndexProp = true
			}

			if prop == "m_iClip1" {
				hasClipProp = true
			}

			if prop == "m_hThrower" {
				hasThrower = true
			}
		}

		isEquipmentClass := hasClipProp && hasIndexProp

		if isEquipmentClass {
			sc.OnEntityCreated(p.bindWeaponS2)
		}

		if hasThrower {
			sc.OnEntityCreated(p.bindGrenadeProjectiles)
		}
	}

	p.stParser.ServerClasses().FindByName("CInferno").OnEntityCreated(p.bindNewInferno)
	p.stParser.ServerClasses().FindByName("CSmokeGrenadeProjectile").OnEntityCreated(p.bindNewSmoke)
}

// bindGrenadeProjectiles keeps track of the location of live grenades (parser.gameState.grenadeProjectiles), actively thrown by players.
// It does NOT track the location of grenades lying on the ground, i.e. that were dropped by dead players.
func (p *parser) bindGrenadeProjectiles(entity st.Entity) {
	entityID := entity.ID()

	proj := common.NewGrenadeProjectile()
	proj.Entity = entity
	p.gameState.grenadeProjectiles[entityID] = proj

	player := p.demoInfoProvider.FindPlayerByPawnHandle(entity.PropertyValueMust("m_hOwnerEntity").Handle())
	proj.Thrower = player
	proj.Owner = player

	var wep common.EquipmentType
	entity.OnCreateFinished(func() { //nolint:wsl
		model := entity.PropertyValueMust("CBodyComponent.m_hModel").S2UInt64()
		weaponType, exists := p.equipmentTypePerModel[model]
		if exists {
			wep = weaponType
		}

		// copy the weapon so it doesn't get overwritten by a new entity in parser.weapons
		wepCopy := *(getPlayerWeapon(proj.Thrower, wep))
		proj.WeaponInstance = &wepCopy

		unassert.NotNilf(proj.WeaponInstance, "couldn't find grenade instance for player")
		if proj.WeaponInstance != nil {
			unassert.NotNilf(proj.WeaponInstance.Owner, "getPlayerWeapon() returned weapon instance with Owner=nil")
		}

		p.gameEventHandler.addThrownGrenade(proj.Thrower, proj.WeaponInstance)

		if !p.disableMimicSource1GameEvents {
			p.eventDispatcher.Dispatch(events.WeaponFire{
				Shooter: proj.Owner,
				Weapon:  proj.WeaponInstance,
			})
		}

		p.eventDispatcher.Dispatch(events.GrenadeProjectileThrow{
			Projectile: proj,
		})
	})

	entity.OnDestroy(func() {
		if wep == common.EqFlash && !p.disableMimicSource1GameEvents {
			p.gameEventHandler.dispatch(events.FlashExplode{
				GrenadeEvent: events.GrenadeEvent{
					GrenadeType:     common.EqFlash,
					Grenade:         proj.WeaponInstance,
					Position:        proj.Position(),
					Thrower:         proj.Thrower,
					GrenadeEntityID: proj.Entity.ID(),
				},
			})
		}

		p.nadeProjectileDestroyed(proj)
	})

	// @micvbang: not quite sure what the difference between Thrower and Owner is.
	entity.Property("m_hThrower").OnUpdate(func(val st.PropertyValue) {
		proj.Thrower = p.demoInfoProvider.FindPlayerByPawnHandle(val.Handle())
	})

	entity.Property("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
		proj.Owner = p.gameState.Participants().FindByPawnHandle(val.Handle())
	})

	entity.OnPositionUpdate(func(newPos r3.Vector) {
		proj.Trajectory = append(proj.Trajectory, newPos)

		proj.Trajectory2 = append(proj.Trajectory2, common.TrajectoryEntry{
			Position: newPos,
			FrameID:  p.CurrentFrame(),
			Time:     p.CurrentTime(),
		})
	})

	// Some demos don't have this property as it seems
	// So we need to check for nil and can't send out bounce events if it's missing
	if bounceProp := entity.Property("m_nBounces"); bounceProp != nil {
		bounceProp.OnUpdate(func(val st.PropertyValue) {
			bounceNumber := val.Int()
			if bounceNumber != 0 {
				p.eventDispatcher.Dispatch(events.GrenadeProjectileBounce{
					Projectile: proj,
					BounceNr:   bounceNumber,
				})
			}
		})
	}

	if voxelProp := entity.Property("m_VoxelFrameData"); voxelProp != nil {
		voxelProp.OnUpdate(func(val st.PropertyValue) {
			smk := p.gameState.smokes[entityID]
			for i := len(smk.VoxelFrameData); i < 10000; i++ {
				val := smk.Entity.Property("m_VoxelFrameData." + fmt.Sprintf("%04d", i)).Value()
				if val.Any == nil {
					break
				}
				smk.VoxelFrameData = append(smk.VoxelFrameData, uint8(val.S2UInt64()))
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

	// We delete from the Owner.ThrownGrenades (only if not inferno or smoke, because they will be deleted when they expire)
	isInferno := proj.WeaponInstance.Type == common.EqMolotov || proj.WeaponInstance.Type == common.EqIncendiary
	isSmoke := proj.WeaponInstance.Type == common.EqSmoke
	isDecoy := proj.WeaponInstance.Type == common.EqDecoy

	if !isInferno && !isSmoke && !isDecoy {
		p.gameEventHandler.deleteThrownGrenade(proj.Thrower, proj.WeaponInstance.Type)
	}
}

func (p *parser) bindWeaponS2(entity st.Entity) {
	entityID := entity.ID()
	itemIndex := entity.PropertyValueMust("m_iItemDefinitionIndex").S2UInt64()
	wepType := common.EquipmentIndexMapping[itemIndex]

	if wepType == common.EqUnknown {
		fmt.Println("unknown equipment with index", itemIndex)
		p.msgDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("unknown equipment with index %d", itemIndex),
			Type:    events.WarnTypeUnknownEquipmentIndex,
		})
	} else {
		model := entity.PropertyValueMust("CBodyComponent.m_hModel").S2UInt64()
		p.equipmentTypePerModel[model] = wepType
	}

	equipment, exists := p.gameState.weapons[entityID]
	if !exists {
		equipment = common.NewEquipment(wepType)
		p.gameState.weapons[entityID] = equipment
	} else {
		equipment.Type = wepType
	}

	equipment.Entity = entity
	equipment.Skin = equipment.GetSkin()

	// Used to detect when a player has been refunded for a weapon
	// This happens when:
	// - The player is inside the buy zone
	// - The player's money has increased AND the weapon entity is destroyed at the same tick (unfortunately the money is updated first)
	var (
		oldOwnerMoney       int
		lastMoneyUpdateTick int
		lastMoneyIncreased  bool
	)

	entity.Property("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
		owner := p.GameState().Participants().FindByPawnHandle(val.Handle())
		if owner == nil {
			equipment.Owner = nil
			return
		}

		oldOwnerMoney = owner.Money()

		owner.Entity.Property("m_pInGameMoneyServices.m_iAccount").OnUpdate(func(val st.PropertyValue) {
			lastMoneyUpdateTick = p.gameState.ingameTick
			currentMoney := owner.Money()
			lastMoneyIncreased = currentMoney > oldOwnerMoney
			oldOwnerMoney = currentMoney
		})
	})

	entity.Property("m_bInReload").OnUpdate(func(val st.PropertyValue) {
		owner := p.GameState().Participants().FindByPawnHandle(entity.PropertyValueMust("m_hOwnerEntity").Handle())
		if owner != nil {
			if val.BoolVal() {
				p.eventDispatcher.Dispatch(events.WeaponReloadBegin{
					Player: owner,
				})

				owner.IsReloading = true
			} else if !val.BoolVal() && owner.IsReloading {
				p.eventDispatcher.Dispatch(events.WeaponReloadEnd{
					Player: owner,
				})

				owner.IsReloading = false
			}
		}
	})

	entity.Property("m_bReloadVisuallyComplete").OnUpdate(func(val st.PropertyValue) {
		owner := p.GameState().Participants().FindByPawnHandle(entity.PropertyValueMust("m_hOwnerEntity").Handle())
		reload := val.BoolVal()
		if !reload || owner == nil || !owner.IsReloading {
			return
		}

		p.eventDispatcher.Dispatch(events.WeaponReloadEnd{
			Player:  owner,
			Success: true,
		})

		owner.IsReloading = false
	})

	entity.OnDestroy(func() {
		owner := p.GameState().Participants().FindByPawnHandle(entity.PropertyValueMust("m_hOwnerEntity").Handle())
		if owner != nil && owner.IsInBuyZone() && p.GameState().IngameTick() == lastMoneyUpdateTick && lastMoneyIncreased {
			p.eventDispatcher.Dispatch(events.ItemRefund{
				Player: owner,
				Weapon: equipment,
			})
		}

		lastMoneyIncreased = false
		p.gameState.wepsToRemove[entityID] = equipment
	})

	// Detect weapon firing, we don't use m_iClip1 because it would not work with weapons such as the knife (no ammo).
	// WeaponFire events for grenades are dispatched when the grenade's projectile is created.
	if equipment.Class() != common.EqClassGrenade && !p.disableMimicSource1GameEvents {
		entity.Property("m_fLastShotTime").OnUpdate(func(val st.PropertyValue) {
			shooter := p.GameState().Participants().FindByPawnHandle(entity.PropertyValueMust("m_hOwnerEntity").Handle())
			if shooter == nil {
				shooter = equipment.Owner
			}
			if shooter != nil && val.Float() > 0 {
				p.eventDispatcher.Dispatch(events.WeaponFire{
					Shooter: shooter,
					Weapon:  equipment,
				})
			}
		})
	}
}

func (p *parser) bindNewInferno(entity st.Entity) {
	throwerHandle := entity.PropertyValueMust("m_hOwnerEntity").Handle()
	thrower := p.gameState.Participants().FindByPawnHandle(throwerHandle)
	inf := common.NewInferno(p.demoInfoProvider, entity, thrower)
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

func (p *parser) bindNewSmoke(entity st.Entity) {
	throwerHandle := entity.PropertyValueMust("m_hOwnerEntity").Handle()
	thrower := p.gameState.Participants().FindByPawnHandle(throwerHandle)
	smk := common.NewSmoke(p.demoInfoProvider, entity, thrower)
	p.gameState.smokes[entity.ID()] = smk

	entity.OnDestroy(func() {
		p.smokeExpired(smk)
	})

	entity.Property("m_bDidSmokeEffect").OnUpdate(func(val st.PropertyValue) {
		if val.BoolVal() {
			smk.ActivationTick = p.demoInfoProvider.IngameTick()
		}
	})
}

// Separate function because we also use it in round_officially_ended (issue #42)
func (p *parser) smokeExpired(inf *common.Smoke) {
	// If the smoke entity is destroyed AFTER round_officially_ended
	// we already executed this code when we received that event.
	if _, exists := p.gameState.smokes[inf.Entity.ID()]; !exists {
		return
	}

	delete(p.gameState.smokes, inf.Entity.ID())
}

//nolint:funlen
func (p *parser) bindGameRules() {
	gameRules := p.ServerClasses().FindByName("CCSGameRulesProxy")
	gameRules.OnEntityCreated(func(entity st.Entity) {
		grPrefix := func(s string) string {
			return fmt.Sprintf("%s.%s", gameRulesPrefixS2, s)
		}

		p.gameState.rules.entity = entity

		roundTime := entity.PropertyValueMust(grPrefix("m_iRoundTime")).Int()
		hasRescueZone := entity.PropertyValueMust(grPrefix("m_bMapHasRescueZone")).BoolVal()
		hasBombTarget := entity.PropertyValueMust(grPrefix("m_bMapHasBombTarget")).BoolVal()

		dispatchRoundStart := func() {
			if p.gameState.TotalRoundsPlayed() > 0 {
				p.gameEventHandler.dispatch(events.RoundEndOfficial{})
			}

			p.gameEventHandler.clearGrenadeProjectiles()

			for _, player := range p.gameState.playersByEntityID {
				player.IsPlanting = false
				player.IsDefusing = false
			}
			p.gameState.currentPlanter = nil
			p.gameState.currentDefuser = nil

			if p.disableMimicSource1GameEvents {
				return
			}

			var objective string
			if hasBombTarget {
				objective = "BOMB TARGET"
			} else if hasRescueZone {
				objective = "HOSTAGE RESCUE"
			} else {
				objective = "DEATHMATCH"
			}

			p.gameState.lastRoundStartEvent = &events.RoundStart{
				TimeLimit: roundTime,
				FragLimit: 0, // Always 0, seems hardcoded in the game
				Objective: objective,
			}
		}

		entity.Property(grPrefix("m_iRoundTime")).OnUpdate(func(val st.PropertyValue) {
			roundTime = val.Int()
		})

		entity.Property(grPrefix("m_bFreezePeriod")).OnUpdate(func(val st.PropertyValue) {
			newIsFreezetime := val.BoolVal()
			freezetimeEvent := events.RoundFreezetimeChanged{
				OldIsFreezetime: p.gameState.isFreezetime,
				NewIsFreezetime: newIsFreezetime,
			}

			if p.disableMimicSource1GameEvents {
				p.eventDispatcher.Dispatch(freezetimeEvent)
			} else {
				p.gameState.lastFreezeTimeChangedEvent = &freezetimeEvent
			}

			p.gameState.isFreezetime = newIsFreezetime
		})

		entity.Property(grPrefix("m_gamePhase")).OnUpdate(func(val st.PropertyValue) {
			oldGamePhase := p.gameState.gamePhase
			p.gameState.gamePhase = common.GamePhase(val.Int())

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
			p.gameState.isWarmupPeriod = val.BoolVal()

			p.eventDispatcher.Dispatch(events.IsWarmupPeriodChanged{
				OldIsWarmupPeriod: oldIsWarmupPeriod,
				NewIsWarmupPeriod: p.gameState.isWarmupPeriod,
			})
		})

		entity.Property(grPrefix("m_bHasMatchStarted")).OnUpdate(func(val st.PropertyValue) {
			oldMatchStarted := p.gameState.isMatchStarted
			newMatchStarted := val.BoolVal()

			event := events.MatchStartedChanged{
				OldIsStarted: oldMatchStarted,
				NewIsStarted: newMatchStarted,
			}
			if !p.disableMimicSource1GameEvents {
				p.gameState.lastMatchStartedChangedEvent = &event
				// First round start event detection, we can't detect it by listening for a m_eRoundWinReason prop update
				// because there is no update triggered when the first round starts as the prop value is already 0.
				if newMatchStarted {
					winRoundReason := events.RoundEndReason(entity.PropertyValueMust(grPrefix("m_eRoundWinReason")).Int())
					if winRoundReason == events.RoundEndReasonStillInProgress {
						dispatchRoundStart()
					}
				}
			} else {
				p.gameState.isMatchStarted = newMatchStarted
				p.eventDispatcher.Dispatch(event)
			}
		})

		// Incremented at the beginning of a new overtime.
		entity.Property(grPrefix("m_nOvertimePlaying")).OnUpdate(func(val st.PropertyValue) {
			overtimeCount := val.Int()
			p.eventDispatcher.Dispatch(events.OvertimeNumberChanged{
				OldCount: p.gameState.overtimeCount,
				NewCount: overtimeCount,
			})
			p.gameState.overtimeCount = overtimeCount
		})

		firstUpdateOccurred := false
		// Updated when a round ends or starts.
		// The value 0 means there is no result yet and the round is in progress.
		entity.Property(grPrefix("m_eRoundWinReason")).OnUpdate(func(val st.PropertyValue) {
			// Ignore the first update that contains initial CCSGameRulesProxy class values.
			if !firstUpdateOccurred {
				firstUpdateOccurred = true
				return
			}

			reason := events.RoundEndReason(val.Int())
			if reason == events.RoundEndReasonStillInProgress {
				dispatchRoundStart()
				return
			}

			message := "UNKNOWN"
			var winner common.Team = common.TeamUnassigned
			switch reason {
			case events.RoundEndReasonTargetBombed:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Target_Bombed"
			case events.RoundEndReasonTerroristsEscaped:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Terrorists_Escaped"
			case events.RoundEndReasonTerroristsWin:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Terrorists_Win"
			case events.RoundEndReasonHostagesNotRescued:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Hostages_Not_Rescued"
			case events.RoundEndReasonTerroristsPlanted:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_Terrorists_Planted"
			case events.RoundEndReasonCTSurrender:
				winner = common.TeamTerrorists
				message = "#SFUI_Notice_CTs_Surrender"
			case events.RoundEndReasonCTsReachedHostage:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_CTs_ReachedHostage"
			case events.RoundEndReasonCTStoppedEscape:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_CTs_PreventEscape"
			case events.RoundEndReasonTerroristsStopped:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Escaping_Terrorists_Neutralized"
			case events.RoundEndReasonBombDefused:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Bomb_Defused"
			case events.RoundEndReasonCTWin:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_CTs_Win"
			case events.RoundEndReasonHostagesRescued:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_All_Hostages_Rescued"
			case events.RoundEndReasonTargetSaved:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Target_Saved"
			case events.RoundEndReasonTerroristsNotEscaped:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Terrorists_Not_Escaped"
			case events.RoundEndReasonTerroristsSurrender:
				winner = common.TeamCounterTerrorists
				message = "#SFUI_Notice_Terrorists_Surrender"
			case events.RoundEndReasonGameStart:
				winner = common.TeamSpectators
				message = "#SFUI_Notice_Game_Commencing"
			case events.RoundEndReasonDraw:
				winner = common.TeamSpectators
				message = "#SFUI_Notice_Round_Draw"
			}

			var winnerState *common.TeamState
			var loserState *common.TeamState
			if winner != common.TeamUnassigned {
				if winner == common.TeamSpectators {
					winnerState = p.gameState.Team(winner)
					loserState = p.gameState.Team(winner)
				} else {
					winnerState = p.gameState.Team(winner)
					loserState = winnerState.Opponent
				}
			}

			if !p.disableMimicSource1GameEvents {
				p.gameState.lastRoundEndEvent = &events.RoundEnd{
					Reason:      reason,
					Message:     message,
					Winner:      winner,
					WinnerState: winnerState,
					LoserState:  loserState,
				}
			}
		})

		entity.Property(grPrefix("m_nTerroristTimeOuts")).OnUpdate(func(val st.PropertyValue) {
			if p.gameState.tState.Timeouts > val.Int() {
				p.gameState.tState.Timeouts = val.Int()
				p.eventDispatcher.Dispatch(events.Timeout{
					TeamState: &p.gameState.tState,
				})
				return
			}
			p.gameState.tState.Timeouts = val.Int()
		})

		entity.Property(grPrefix("m_nCTTimeOuts")).OnUpdate(func(val st.PropertyValue) {
			if p.gameState.ctState.Timeouts > val.Int() {
				p.gameState.ctState.Timeouts = val.Int()
				p.eventDispatcher.Dispatch(events.Timeout{
					TeamState: &p.gameState.ctState,
				})
				return
			}
			p.gameState.ctState.Timeouts = val.Int()
		})

		entity.Property(grPrefix("m_bTechnicalTimeOut")).OnUpdate(func(val st.PropertyValue) {
			if val.BoolVal() {
				p.eventDispatcher.Dispatch(events.Timeout{
					Tech: true,
				})
			}
		})

		// TODO: future fields to use
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

func (p *parser) bindHostages() {
	p.stParser.ServerClasses().FindByName("CHostage").OnEntityCreated(func(entity st.Entity) {
		entityID := entity.ID()
		p.gameState.hostages[entityID] = common.NewHostage(p.demoInfoProvider, entity)

		entity.OnDestroy(func() {
			delete(p.gameState.hostages, entityID)
		})

		var state common.HostageState
		entity.Property("m_nHostageState").OnUpdate(func(val st.PropertyValue) {
			oldState := state
			state = common.HostageState(val.Int())
			if oldState != state {
				p.eventDispatcher.Dispatch(events.HostageStateChanged{OldState: oldState, NewState: state, Hostage: p.gameState.hostages[entityID]})
			}
		})
	})
}

func getDistanceBetweenVectors(vectorA r3.Vector, vectorB r3.Vector) float64 {
	return math.Sqrt(math.Pow(vectorA.X-vectorB.X, 2) + math.Pow(vectorA.Y-vectorB.Y, 2) + math.Pow(vectorA.Z-vectorB.Z, 2))
}

func (p *parser) getClosestBombsiteFromPosition(position r3.Vector) events.Bombsite {
	distanceFromBombsiteA := getDistanceBetweenVectors(position, p.bombsiteA.center)
	distanceFromBombsiteB := getDistanceBetweenVectors(position, p.bombsiteB.center)

	if distanceFromBombsiteA < distanceFromBombsiteB {
		return events.BombsiteA
	}

	return events.BombsiteB
}
