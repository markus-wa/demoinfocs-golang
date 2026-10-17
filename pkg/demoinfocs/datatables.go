package demoinfocs

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/go-unassert"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

func (p *parser) mapEquipment() {
	if p.isSource2() {
		return
	}

	for _, sc := range p.stParser.ServerClasses().All() {
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
	p.bindHostages()
}

//nolint:gocognit,gocyclo,funlen
func (p *parser) bindBomb() {
	bomb := &p.gameState.bomb

	// Track bomb when it is dropped on the ground or being held by a player
	scC4 := p.stParser.ServerClasses().FindByName("CC4")
	scC4.OnEntityCreated(func(bombEntity st.Entity) {
		bombEntity.OnPositionUpdate(func(pos r3.Vector) {
			bomb.LastOnGroundPosition = pos
		})

		//nolint:nestif
		if p.isSource2() {
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
		} else {
			bombEntity.Property("m_hOwner").OnUpdate(func(val st.PropertyValue) {
				bomb.Carrier = p.gameState.Participants().FindByHandle(val.Int())
			})
		}

		// Updated when a player starts/stops planting the bomb
		bombEntity.Property("m_bStartedArming").OnUpdate(func(val st.PropertyValue) {
			//nolint:nestif
			if val.BoolVal() {
				if p.isSource2() {
					planterHandle := bombEntity.PropertyValueMust("m_hOwnerEntity").Handle()

					pawnEntity := p.gameState.entities[entityIDFromHandle(planterHandle, true)]
					if pawnEntity == nil {
						return
					}

					ctlHandle := pawnEntity.PropertyValueMust("m_hController").Handle()

					ctlEntity := p.gameState.entities[entityIDFromHandle(ctlHandle, true)]
					if ctlEntity == nil {
						return
					}

					ctlID := ctlEntity.ID()

					planter := p.gameState.playersByEntityID[ctlID]
					if planter == nil {
						return
					}

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
				} else {
					p.gameState.currentPlanter = bomb.Carrier
				}
			} else if p.gameState.currentPlanter != nil {
				p.gameState.currentPlanter.IsPlanting = false
				p.eventDispatcher.Dispatch(events.BombPlantAborted{Player: p.gameState.currentPlanter})
			}
		})

		bombEntity.OnDestroy(func() {
			// the bomb was re-created (e.g. round restart) while it was being planted
			if isEntityReplaced(bombEntity) && p.gameState.currentPlanter != nil && p.gameState.currentPlanter.IsPlanting {
				p.gameState.currentPlanter.IsPlanting = false
				p.eventDispatcher.Dispatch(events.BombPlantAborted{Player: p.gameState.currentPlanter})
			}

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

		//nolint:nestif
		if p.isSource2() {
			ownerProp := bombEntity.PropertyValueMust("m_hOwnerEntity")

			var planter *common.Player

			if ownerProp.Any != nil {
				planter = p.gameState.Participants().FindByPawnHandle(ownerProp.Handle())

				if planter != nil {
					planter.IsPlanting = false
				}
			}

			siteNumberVal := bombEntity.PropertyValueMust("m_nBombSite")

			site := events.BomsiteUnknown

			if siteNumberVal.Any != nil {
				switch siteNumberVal.Int() {
				case 0:
					site = events.BombsiteA
				case 1:
					site = events.BombsiteB
				}
			}

			if !p.disableMimicSource1GameEvents {
				p.eventDispatcher.Dispatch(events.BombPlanted{
					BombEvent: events.BombEvent{
						Player: planter,
						Site:   site,
					},
				})
			}

			dispatchBombExplode := func() {
				if p.disableMimicSource1GameEvents {
					return
				}

				p.gameEventHandler.frameToBombExploded[p.currentFrame] = true
				p.eventDispatcher.Dispatch(events.BombExplode{
					BombEvent: events.BombEvent{
						Player: planter,
						Site:   site,
					},
				})
			}

			if hasExplodedProp := bombEntity.Property("m_bHasExploded"); hasExplodedProp != nil {
				// Set to true when the bomb explodes.
				// m_bBombTicking is not suitable for this: after the round has already been decided,
				// the server may stop the bomb without detonating it (e.g. at half-time or map end).
				hasExploded := false

				hasExplodedProp.OnUpdate(func(val st.PropertyValue) {
					if val.Any == nil || !val.BoolVal() || hasExploded {
						return
					}

					hasExploded = true

					dispatchBombExplode()
				})
			} else {
				// Fallback for demos without m_bHasExploded.
				// Set to true when the bomb has been planted and to false when it has been defused or has exploded.
				bombEntity.Property("m_bBombTicking").OnUpdate(func(val st.PropertyValue) {
					if val.Any == nil || val.BoolVal() {
						return
					}

					// At this point the bomb stopped ticking either because it has been defused or has exploded.
					// We detect only explosions here, defuse events are detected with m_bBombDefused updates which seems more suitable.
					// When the bomb is defused, m_bBombTicking is set to false and then m_hBombDefuser is set to nil.
					// It means that if a player is currently defusing the bomb, it's a defuse event.
					if p.gameState.currentDefuser != nil {
						return
					}

					dispatchBombExplode()
				})
			}

			// Updated when a player starts/stops defusing the bomb
			bombEntity.Property("m_hBombDefuser").OnUpdate(func(val st.PropertyValue) {
				if val.Any == nil {
					return
				}

				isValidPlayer := val.Handle() != constants.InvalidEntityHandleSource2
				if isValidPlayer {
					defuser := p.gameState.Participants().FindByPawnHandle(val.Handle())
					p.gameState.currentDefuser = defuser
					hasKit := false

					// defuser may be nil for POV demos
					if defuser != nil {
						hasKit = defuser.HasDefuseKit()
					}

					if !p.disableMimicSource1GameEvents {
						p.eventDispatcher.Dispatch(events.BombDefuseStart{
							Player: defuser,
							HasKit: hasKit,
							Site:   site,
						})
					}

					return
				}

				isDefusedVal := bombEntity.PropertyValueMust("m_bBombDefused")

				if isDefusedVal.Any != nil {
					isDefused := isDefusedVal.BoolVal()
					if !isDefused && p.gameState.currentDefuser != nil {
						p.eventDispatcher.Dispatch(events.BombDefuseAborted{
							Player: p.gameState.currentDefuser,
						})
					}
				}

				p.gameState.currentDefuser = nil
			})

			// Updated when the bomb has been planted and defused.
			bombEntity.Property("m_bBombDefused").OnUpdate(func(val st.PropertyValue) {
				if val.Any == nil {
					return
				}

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
				p.gameState.currentDefuser = nil
			})
		}
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

			if p.isSource2() {
				scoreProp = entity.Property("m_iScore")
			} else {
				scoreProp = entity.Property("m_scoreTotal")
			}

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

		target.BindProperty("m_Collision.m_vecMins", &t.min, st.ValTypeVector)
		target.BindProperty("m_Collision.m_vecMaxs", &t.max, st.ValTypeVector)
	}

	if p.isSource2() {
		// CBombTarget is not available with CS2 demos created in the early days of the limited test.
		bombTargetClass := p.stParser.ServerClasses().FindByName("CBombTarget")
		if bombTargetClass != nil {
			bombTargetClass.OnEntityCreated(onBombTargetEntityCreated)
			return
		}
	}

	p.stParser.ServerClasses().FindByName("CBaseTrigger").OnEntityCreated(onBombTargetEntityCreated)
}

func (p *parser) bindPlayers() {
	if p.isSource2() {
		p.stParser.ServerClasses().FindByName("CCSPlayerController").OnEntityCreated(func(player st.Entity) {
			p.bindNewPlayerControllerS2(player)
		})
		p.stParser.ServerClasses().FindByName("CCSPlayerPawn").OnEntityCreated(func(player st.Entity) {
			p.bindNewPlayerPawnS2(player)
		})
	} else {
		p.stParser.ServerClasses().FindByName("CCSPlayerResource").OnEntityCreated(func(entity st.Entity) {
			p.gameState.playerResourceEntity = entity
		})

		p.stParser.ServerClasses().FindByName("CCSPlayer").OnEntityCreated(func(player st.Entity) {
			p.bindNewPlayerS1(player)
		})
	}
}

// playerUserID returns the user-ID under which a player with the given raw info is registered
// (-1 if there is no raw info). CS2 game-event user-IDs only carry the slot in the low byte.
func (p *parser) playerUserID(rp *common.PlayerInfo) int {
	userID := -1

	if rp != nil {
		userID = rp.UserID
	}

	if p.isSource2() && userID <= math.MaxUint16 {
		userID &= 0xff
	}

	return userID
}

func (p *parser) getOrCreatePlayer(entityID int, rp *common.PlayerInfo) (isNew bool, player *common.Player) {
	player = p.gameState.playersByEntityID[entityID]
	userID := p.playerUserID(rp)

	if player == nil {
		if rp != nil {
			player = p.gameState.playersByUserID[userID]

			if player == nil {
				isNew = true

				player = common.NewPlayer(p.demoInfoProvider)
				player.Name = rp.Name
				player.SteamID64 = rp.XUID
				player.IsBot = rp.IsFakePlayer || rp.GUID == botGUID

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

	player.UserID = userID

	p.gameState.playersByEntityID[entityID] = player

	if rp != nil {
		p.gameState.playersByUserID[userID] = player
	}

	return isNew, player
}

//nolint:funlen
func (p *parser) bindNewPlayerS1(playerEntity st.Entity) {
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

	for i := 0; i < 32; i++ {
		i2 := i // Copy so it stays the same
		playerEntity.BindProperty("m_iAmmo."+fmt.Sprintf("%03d", i2), &pl.AmmoLeft[i2], st.ValTypeInt)
	}

	activeWep := playerEntity.Property("m_hActiveWeapon")

	activeWep.OnUpdate(func(val st.PropertyValue) {
		pl.IsReloading = false
	})

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

	inspectingWeaponProp := playerEntity.Property("m_bIsLookingAtWeapon")
	if inspectingWeaponProp != nil {
		inspectingWeaponProp.OnUpdate(func(val st.PropertyValue) {
			if val.BoolVal() {
				p.eventDispatcher.Dispatch(events.PlayerInspectingWeapon{Player: pl})
			} else {
				p.eventDispatcher.Dispatch(events.PlayerStopInspectingWeapon{Player: pl})
			}
		})
	}

	if isNew {
		if pl.SteamID64 != 0 {
			p.eventDispatcher.Dispatch(events.PlayerConnect{Player: pl})
		} else {
			p.eventDispatcher.Dispatch(events.BotConnect{Player: pl})
		}
	}
}

// getOrCreatePlayerFromControllerEntity returns the player bound to the given CCSPlayerController,
// creating it if necessary.
//
// The controller is the source of truth for a player's identity: Name and SteamID64 come from its
// m_iszPlayerName and m_steamID props (the name the server enforces & the scoreboard shows, which
// may differ from the Steam name in the userinfo string table & player_connect event, e.g. on
// FACEIT - see #692). The userinfo string table is only used to map the controller to a user-ID.
//
// If the controller's slot is taken over by a different person (different m_steamID), a new
// *Player is created - an existing player's SteamID64 is never changed. The *Player is kept per
// controller slot (or live user-ID): a person reconnecting into a different slot gets a new *Player.
//
// Controller prop updates call this once all of the controller's updates in an entity packet are applied
// (see bindNewPlayerControllerS2), so it always sees the controller's complete state - never a half-applied
// take-over.
func (p *parser) getOrCreatePlayerFromControllerEntity(controllerEntity st.Entity) *common.Player {
	controllerEntityID := controllerEntity.ID()
	p.gameState.playerControllerEntities[controllerEntityID] = controllerEntity

	steamID := controllerEntity.PropertyValueMust("m_steamID").S2UInt64()
	name := controllerEntity.PropertyValueMust("m_iszPlayerName").String()
	isBot := steamID == 0
	rp := p.rawPlayerForController(controllerEntity, isBot)
	userID := p.playerUserID(rp)

	player := p.gameState.playersByEntityID[controllerEntityID]
	if player == nil && rp != nil {
		player = p.gameState.playersByUserID[userID]
	}

	// A different person (or a bot replacing a human & vice versa) took over the slot.
	// The controller entity is usually re-created when that happens, but may also be updated in place.
	var replacedInPlace *common.Player

	if player != nil && player.SteamID64 != steamID {
		delete(p.provisionalNames, player)

		if player.Entity == controllerEntity {
			replacedInPlace = player
			p.unbindReplacedPlayer(player)
		}

		player = nil
	}

	name, provisional := playerNameFromController(name, steamID, isBot, rp)

	if player == nil {
		player = p.newPlayerFromController(controllerEntity, steamID, name, provisional)
	} else {
		p.setPlayerName(player, name, isBot, provisional)
	}

	if name != "" {
		player.IsUnknown = false
	} else if player.Name == "" {
		// see #162: some demos since November 2019 have connected players without a name,
		// GOTV calls them "unknown" too.
		player.Name = "unknown"
		player.IsUnknown = true
	}

	player.UserID = userID
	player.Entity = controllerEntity
	player.EntityID = controllerEntityID
	player.IsBot = isBot
	player.RefreshPawnHandles()

	p.gameState.playersByEntityID[controllerEntityID] = player

	if rp != nil {
		p.gameState.playersByUserID[userID] = player
	}

	if replacedInPlace != nil {
		p.takeOverInPlace(replacedInPlace, player, controllerEntity)
	}

	return player
}

// rawPlayerForController returns the raw player info (userinfo string table) of a controller, if any.
func (p *parser) rawPlayerForController(controllerEntity st.Entity, isBot bool) *common.PlayerInfo {
	controllerEntityID := controllerEntity.ID()

	// In regular demos player slots map to controller entities 1:1 (slot N is entity N+1),
	// so the raw info for this controller lives at slot controllerEntityID-1. Match against
	// the controller's props to make sure the slot really belongs to this entity - after a
	// bot slot's name is re-used the slot-first lookup would otherwise bind a stale entry.
	rp := p.matchRawPlayerToController(controllerEntityID-1, isBot, controllerEntity)

	if rp == nil {
		// Fallback (e.g. broadcast drift, see #655): scan raw info by XUID (humans) or name
		// (bots). Iterates slots in ascending order so the result is deterministic; a naive
		// map range randomly bound a re-used bot name to a stale entry, registering the same
		// *Player under two user-IDs (see #692).
		for _, slot := range slices.Sorted(maps.Keys(p.rawPlayers)) {
			if r := p.matchRawPlayerToController(slot, isBot, controllerEntity); r != nil {
				rp = r

				break
			}
		}
	}

	if rp == nil {
		rp = p.rawPlayers[controllerEntityID-1]
	}

	return rp
}

func (p *parser) newPlayerFromController(controllerEntity st.Entity, steamID uint64, name string, provisional bool) *common.Player {
	player := common.NewPlayer(p.demoInfoProvider)
	player.SteamID64 = steamID
	player.Name = name

	if provisional && name != "" {
		p.provisionalNames[player] = struct{}{}
	}

	if team, ok := controllerEntity.PropertyValue("m_iTeamNum"); ok {
		player.Team = common.Team(team.S2UInt64())
		player.TeamState = p.gameState.Team(player.Team)
	}

	p.gameState.indexPlayerBySteamID(player)

	return player
}

// takeOverInPlace hands a controller that was taken over in place from the previous to the new occupant.
func (p *parser) takeOverInPlace(previous, player *common.Player, controllerEntity st.Entity) {
	// the pawn's weapons now belong to the new occupant (see bindPlayerWeapons)
	player.Inventory, previous.Inventory = previous.Inventory, make(map[int]*common.Equipment)

	for _, wep := range player.Inventory {
		wep.Owner = player
	}

	p.gameState.updateLastKnownGrenadeWeapons(player)

	// update handlers don't fire the controller's current state for in-place take-overs
	p.connectPlayerIfConnected(player, controllerEntity)
}

// playerNameFromController returns the name to use for a controller's player. Some servers create the
// controller before naming it - with an empty name or the SteamID as a placeholder. The userinfo name is
// used until the controller has a real one; provisional reports that the name isn't the controller's yet.
func playerNameFromController(name string, steamID uint64, isBot bool, rp *common.PlayerInfo) (string, bool) {
	provisional := name == "" || isSteamIDPlaceholderName(name, steamID)
	if provisional && rp != nil && !isBot && rp.XUID == steamID && rp.Name != "" {
		return rp.Name, true
	}

	return name, provisional
}

// isSteamIDPlaceholderName reports whether a controller name is just the player's SteamID64,
// which some servers use until the player's real name is set.
func isSteamIDPlaceholderName(name string, steamID uint64) bool {
	return steamID != 0 && name == strconv.FormatUint(steamID, 10)
}

// unbindReplacedPlayer detaches a player from a controller that was taken over in place by a different
// person, so the stale *Player neither stays connected nor reads the new occupant's entity data.
func (p *parser) unbindReplacedPlayer(player *common.Player) {
	player.Entity = nil // EntityID is kept for reference

	if !player.IsConnected {
		return
	}

	player.IsConnected = false

	p.gameEventHandler.dispatch(events.PlayerDisconnected{
		Player: player,
	})
}

// dirtyPlayerController is a player controller whose identity, team or connection state changed
// in the entity packet currently being applied.
type dirtyPlayerController struct {
	entity            st.Entity
	teamChanged       bool
	connectionChanged bool
}

func (p *parser) markPlayerControllerDirty(controllerEntity st.Entity) *dirtyPlayerController {
	d := p.dirtyControllers[controllerEntity.ID()]
	if d == nil || d.entity != controllerEntity {
		d = &dirtyPlayerController{entity: controllerEntity}
		p.dirtyControllers[controllerEntity.ID()] = d
	}

	return d
}

// flushPlayerControllerUpdates applies all pending controller identity & connection changes.
// Controllers are normally flushed one by one once their own updates are applied (see
// bindNewPlayerController); this is the backstop at the end of each entity packet.
func (p *parser) flushPlayerControllerUpdates() {
	if len(p.dirtyControllers) == 0 {
		return
	}

	for _, id := range slices.Sorted(maps.Keys(p.dirtyControllers)) {
		p.flushPlayerController(id)
	}
}

// flushPlayerController applies the pending identity & connection changes of one controller.
func (p *parser) flushPlayerController(controllerEntityID int) {
	d := p.dirtyControllers[controllerEntityID]
	if d == nil {
		return
	}

	delete(p.dirtyControllers, controllerEntityID)

	pl := p.getOrCreatePlayerFromControllerEntity(d.entity)

	if d.teamChanged {
		pl.Team = common.Team(d.entity.PropertyValueMust("m_iTeamNum").S2UInt64())
		pl.TeamState = p.gameState.Team(pl.Team)
	}

	if d.connectionChanged {
		p.applyControllerConnectionState(pl, d.entity)
	}
}

func (p *parser) applyControllerConnectionState(pl *common.Player, controllerEntity st.Entity) {
	state := controllerEntity.PropertyValueMust("m_iConnected").S2UInt32()

	isDisconnection := state == 8
	if isDisconnection {
		pl.IsConnected = false

		p.gameEventHandler.dispatch(events.PlayerDisconnected{
			Player: pl,
		})

		return
	}

	if state != 0 {
		pl.IsConnected = false

		return
	}

	p.connectPlayerIfConnected(pl, controllerEntity)
}

// connectPlayerIfConnected marks a player as connected and dispatches PlayerConnect / BotConnect
// if the controller's m_iConnected state says it's connected and the player wasn't before.
func (p *parser) connectPlayerIfConnected(player *common.Player, controllerEntity st.Entity) {
	state, ok := controllerEntity.PropertyValue("m_iConnected")
	if !ok || player.IsConnected || state.S2UInt32() != 0 {
		return
	}

	player.IsConnected = true

	if player.SteamID64 != 0 {
		p.eventDispatcher.Dispatch(events.PlayerConnect{Player: player})
	} else {
		p.eventDispatcher.Dispatch(events.BotConnect{Player: player})
	}
}

// setPlayerName applies a (controller) name to an existing player and dispatches PlayerNameChange
// for humans if it changed. The old name is always taken from the player, never from a game-event.
//
// provisional means the name isn't the controller's real name yet (userinfo fallback or SteamID
// placeholder): it never replaces a real name, and replacing it isn't a name change.
func (p *parser) setPlayerName(player *common.Player, name string, isBot, provisional bool) {
	_, wasProvisional := p.provisionalNames[player]
	if !provisional {
		delete(p.provisionalNames, player)
	}

	if name == "" || player.Name == name {
		return
	}

	if provisional && !wasProvisional && !player.IsUnknown {
		return
	}

	oldName := player.Name
	wasUnknown := player.IsUnknown
	player.Name = name

	if provisional {
		p.provisionalNames[player] = struct{}{}
	}

	// A renamed bot controller is treated as the same bot without dispatching an event, and a player
	// getting its first real name (from unknown or a provisional name) isn't a name change either.
	if isBot || player.IsBot || wasUnknown || wasProvisional {
		return
	}

	p.eventDispatcher.Dispatch(events.PlayerNameChange{
		Player:  player,
		OldName: oldName,
		NewName: name,
	})
}

// matchRawPlayerToController returns the raw player info at the given slot if it belongs to
// the given controller entity: humans by XUID, bots by name (both from controller props).
func (p *parser) matchRawPlayerToController(slot int, isBot bool, controllerEntity st.Entity) *common.PlayerInfo {
	r := p.rawPlayers[slot]
	if r == nil {
		return nil
	}

	if !isBot && !r.IsFakePlayer && r.XUID != 0 && r.XUID == controllerEntity.Property("m_steamID").Value().S2UInt64() {
		return r
	}

	if isBot && r.IsFakePlayer && r.XUID == 0 && r.Name == controllerEntity.PropertyValueMust("m_iszPlayerName").String() {
		return r
	}

	return nil
}

func (p *parser) bindNewPlayerControllerS2(controllerEntity st.Entity) {
	pl := p.getOrCreatePlayerFromControllerEntity(controllerEntity)

	// the player bound to this controller may be replaced if a different person takes over the slot
	currentPlayer := func() *common.Player {
		if current := p.gameState.playersByEntityID[controllerEntity.ID()]; current != nil {
			return current
		}

		return pl
	}

	// The controller's name & SteamID are the source of truth for the player's identity.
	// Prop update handlers fire per field in field-path order (m_iTeamNum, m_iConnected, m_iszPlayerName,
	// m_steamID), so reacting to each of them directly would see a half-applied update when a different
	// person takes over a slot - e.g. renaming the previous occupant to the new name or changing its team.
	// The handlers only mark the controller dirty; the changes are applied once all of the controller's
	// updates in the packet are.
	controllerEntity.OnUpdateFinished(func() {
		p.flushPlayerController(controllerEntity.ID())
	})

	syncIdentity := func(st.PropertyValue) {
		p.markPlayerControllerDirty(controllerEntity)
	}

	controllerEntity.Property("m_iszPlayerName").OnUpdate(syncIdentity)
	controllerEntity.Property("m_steamID").OnUpdate(syncIdentity)

	controllerEntity.Property("m_iConnected").OnUpdate(func(st.PropertyValue) {
		p.markPlayerControllerDirty(controllerEntity).connectionChanged = true
	})

	controllerEntity.Property("m_iTeamNum").OnUpdate(func(st.PropertyValue) {
		p.markPlayerControllerDirty(controllerEntity).teamChanged = true
	})

	// Keep the cached pawn handles of whoever currently owns the controller up to date, see Player.PlayerPawnEntity.
	syncPawnHandles := func(st.PropertyValue) {
		currentPlayer().RefreshPawnHandles()
	}

	controllerEntity.Property("m_hPawn").OnUpdate(syncPawnHandles)
	controllerEntity.Property("m_hPlayerPawn").OnUpdate(syncPawnHandles)

	controllerEntity.OnDestroy(func() {
		pl := currentPlayer()
		pl.IsConnected = false

		if d := p.dirtyControllers[controllerEntity.ID()]; d != nil && d.entity == controllerEntity {
			delete(p.dirtyControllers, controllerEntity.ID())
		}

		delete(p.provisionalNames, pl)
		delete(p.gameState.playersByEntityID, controllerEntity.ID())
		delete(p.gameState.playerControllerEntities, controllerEntity.ID())
		delete(p.gameState.playersByUserID, pl.UserID)
	})
}

// controllerPlayerFunc returns a function that returns the player currently bound to the given controller,
// or fallback if there is none.
func (p *parser) controllerPlayerFunc(controllerEntityID int, fallback *common.Player) func() *common.Player {
	return func() *common.Player {
		if current := p.gameState.playersByEntityID[controllerEntityID]; current != nil {
			return current
		}

		return fallback
	}
}

//nolint:funlen
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
		if controllerHandle == constants.InvalidEntityHandleSource2 || controllerHandle == prevControllerHandle {
			return
		}

		prevControllerHandle = controllerHandle

		controllerEntityID := int(controllerHandle & constants.EntityHandleIndexMaskSource2)
		controllerEntity := p.gameState.playerControllerEntities[controllerEntityID]

		pl := p.getOrCreatePlayerFromControllerEntity(controllerEntity)

		// the controller's player may be replaced by an in-place take-over while the pawn lives on
		p.bindPlayerWeaponsS2(pawnEntity, p.controllerPlayerFunc(controllerEntityID, pl))
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

		if pl.FlashDuration > 0 && len(p.gameState.flyingFlashbangs) > 0 {
			// The detonation (projectile OnDestroy) and these m_flFlashDuration updates arrive in the
			// same frame but in an unspecified order, so we can't yet know which projectile caused
			// this. Buffer the flashed pawn and pair it with the flashbang that detonated this frame
			// at the end of the frame (see processFlyingFlashbangs).
			p.gameState.flashedEntitiesThisFrame = append(p.gameState.flashedEntitiesThisFrame, pl.EntityID)
		}
	})

	pawnEntity.Property("m_pWeaponServices.m_hActiveWeapon").OnUpdate(func(val st.PropertyValue) {
		pl := getPlayerFromPawnEntity(pawnEntity)
		if pl == nil {
			return
		}

		pl.IsReloading = false
	})

	pawnEntity.Property("m_bIsDefusing").OnUpdate(func(val st.PropertyValue) {
		pl := getPlayerFromPawnEntity(pawnEntity)
		if pl == nil {
			return
		}

		pl.IsDefusing = val.BoolVal()
	})

	spottedByMaskProp := pawnEntity.Property("m_entitySpottedState.m_bSpottedByMask.0000")
	if spottedByMaskProp != nil {
		spottersChanged := func(val st.PropertyValue) {
			pl := getPlayerFromPawnEntity(pawnEntity)
			if pl == nil {
				return
			}

			p.eventDispatcher.Dispatch(events.PlayerSpottersChanged{Spotted: pl})
		}

		spottedByMaskProp.OnUpdate(spottersChanged)
		pawnEntity.Property("m_entitySpottedState.m_bSpottedByMask.0001").OnUpdate(spottersChanged)
	}

	// Legacy fallback for demos that don't contain CSVCMsg_UserCommands messages.
	// The m_nButtonDownMaskPrev prop was removed in the 2026-07-09 CS2 update; on
	// newer demos this prop is absent and button state comes from CSVCMsg_UserCommands messages
	// instead. When both are present, the user command messages take precedence.
	buttonDownMaskProp := pawnEntity.Property("m_pMovementServices.m_nButtonDownMaskPrev")
	if !p.hasUserCmdMessages && buttonDownMaskProp != nil {
		buttonDownMaskProp.OnUpdate(func(val st.PropertyValue) {
			pl := getPlayerFromPawnEntity(pawnEntity)
			if pl == nil {
				return
			}

			state := val.S2UInt64()
			pl.ButtonsPressedState = state

			p.eventDispatcher.Dispatch(events.PlayerButtonsStateUpdate{
				Player:       pl,
				ButtonsState: state,
			})
		})
	}
}

const maxWeapons = 64

func (p *parser) bindPlayerWeapons(playerEntity st.Entity, pl *common.Player) {
	// Some demos have an additional prefix for player weapons weapon
	var wepPrefix string
	if playerEntity.Property(playerWeaponPrefix+".000") != nil {
		wepPrefix = playerWeaponPrefix
	} else {
		wepPrefix = "bcc_nonlocaldata." + playerWeaponPrefix
	}

	// The player may be re-bound to a new entity (e.g. after a reconnect or on round restarts) while keeping its
	// inventory, the cache below only knows the slots of this entity so weapons of the previous one would be kept forever.
	// The new entity's slots re-populate the inventory.
	for _, wep := range pl.Inventory {
		if wep.Owner == pl {
			wep.Owner = nil
		}
	}

	pl.Inventory = make(map[int]*common.Equipment)

	// Weapons
	var cache [maxWeapons]int
	for i := range cache {
		i2 := i // Copy for passing to handler
		playerEntity.Property(wepPrefix + fmt.Sprintf(".%03d", i)).OnUpdate(func(val st.PropertyValue) {
			entityID := val.IntVal & constants.EntityHandleIndexMask
			//nolint:nestif
			if entityID != constants.EntityHandleIndexMask {
				if cache[i2] != 0 {
					// Player already has a weapon in this slot.
					delete(pl.Inventory, cache[i2])
				}

				cache[i2] = entityID

				wep := p.gameState.weapons[entityID]

				// Entities are updated in index order, so a player whose index is lower than the weapon's gets a
				// weapon whose index is re-used for a new entity (see SendTableParser.ReadEnterPVS) before the
				// previous weapon is destroyed. Only a newer serial makes the entry stale, see bindPlayerWeaponsS2.
				handleSerial := val.IntVal >> constants.MaxEdictBits
				isStale := wep != nil && wep.Entity != nil && isNewerSerial(handleSerial, wep.Entity.SerialNum())

				if wep == nil || isStale {
					// sometimes a weapon is assigned to a player before the weapon entity is created
					wep = common.NewEquipment(common.EqUnknown, p.nextUniqueID())
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

//nolint:funlen
func (p *parser) bindPlayerWeaponsS2(pawnEntity st.Entity, currentPlayer func() *common.Player) {
	const inventoryCapacity = 64

	inventorySize := 64

	type eq struct {
		*common.Equipment
		entityID int
	}

	playerInventory := make(map[int]eq)

	getWep := func(wepSlotPropertyValue st.PropertyValue) (uint64, *common.Equipment) {
		handle := wepSlotPropertyValue.S2UInt64()
		entityID := handle & constants.EntityHandleIndexMaskSource2
		wep := p.gameState.weapons[int(entityID)]

		// If the index is re-used for a new weapon (see sendtables2.Entity.Replaced()), the map may still hold the
		// previous weapon when the pawn gets the new one before the new weapon entity is created in the same packet.
		// The other way around (new weapon created first) the pawn may still reference the previous serial in
		// slots that haven't been updated yet, so only a newer serial makes the entry stale.
		handleSerial := int(handle >> constants.MaxEdictBitsSource2)
		isStale := wep != nil && wep.Entity != nil && isNewerSerial(handleSerial, wep.Entity.SerialNum())

		if wep == nil || isStale {
			// sometimes a weapon is assigned to a player before the weapon entity is created
			wep = common.NewEquipment(common.EqUnknown, p.nextUniqueID())
			p.gameState.weapons[int(entityID)] = wep
		}

		return entityID, wep
	}

	setPlayerInventory := func() {
		inventory := make(map[int]*common.Equipment, inventorySize)

		for i := 0; i < inventorySize; i++ {
			val := pawnEntity.Property(playerWeaponPrefixS2 + fmt.Sprintf(".%04d", i)).Value()
			if val.Any == nil {
				continue
			}

			entityID, wep := getWep(val)
			inventory[int(entityID)] = wep
		}

		pl := currentPlayer()
		pl.Inventory = inventory

		// Track grenades for the fallback in bindGrenadeProjectiles (#580).
		// Not deleted when a grenade disappears from the inventory - see
		// updateLastKnownGrenadeWeapons.
		p.gameState.updateLastKnownGrenadeWeapons(pl)
	}

	pawnEntity.Property(playerWeaponPrefixS2).OnUpdate(func(pv st.PropertyValue) {
		inventorySize = len(pv.S2Array())

		setPlayerInventory()
	})

	for i := 0; i < inventoryCapacity; i++ {
		updateWeapon := func(val st.PropertyValue) {
			if val.Any == nil {
				return
			}

			entityID, wep := getWep(val)
			pl := currentPlayer()
			wep.Owner = pl

			entityWasCreated := entityID != constants.EntityHandleIndexMaskSource2

			if i < inventorySize {
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

		property := pawnEntity.Property(playerWeaponPrefixS2 + fmt.Sprintf(".%04d", i))
		updateWeapon(property.Value())
		property.OnUpdate(updateWeapon)
	}
}

//nolint:gocognit
func (p *parser) bindWeapons() {
	s2 := p.isSource2()

	for _, sc := range p.stParser.ServerClasses().All() {
		if s2 { //nolint:nestif
			hasIndexProp := false
			hasClipProp := false
			hasThrower := false

			for _, prop := range sc.PropertyEntries() {
				if prop == "m_AttributeManager.m_Item.m_iItemDefinitionIndex" {
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
		} else {
			for _, bc := range sc.BaseClasses() {
				switch bc.Name() {
				case "CWeaponCSBase":
					if s2 {
						continue
					}

					sc2 := sc // Local copy for loop
					sc.OnEntityCreated(func(e st.Entity) { p.bindWeapon(e, p.equipmentMapping[sc2]) })

				case "CBaseGrenade": // Grenade that has been thrown by player.
					sc.OnEntityCreated(p.bindGrenadeProjectiles)

				case "CBaseCSGrenade":
					// @micvbang TODO: handle grenades dropped by dead player.
					// Grenades that were dropped by a dead player (and can be picked up by other players).
				}
			}
		}
	}

	p.stParser.ServerClasses().FindByName("CInferno").OnEntityCreated(p.bindNewInferno)
}

// equipmentTypeFromProjectileClass resolves a grenade type from the projectile entity's server
// class. Used as a fallback when the model hash isn't yet in equipmentTypePerModel.
func equipmentTypeFromProjectileClass(className string) common.EquipmentType {
	switch className {
	case "CFlashbangProjectile":
		return common.EqFlash
	case "CHEGrenadeProjectile":
		return common.EqHE
	case "CSmokeGrenadeProjectile":
		return common.EqSmoke
	case "CDecoyProjectile":
		return common.EqDecoy
	case "CMolotovProjectile":
		// Molotov and incendiary share this projectile class and are only distinguishable by model,
		// so this fallback can't tell them apart - default to molotov.
		return common.EqMolotov
	}

	return common.EqUnknown
}

// bindGrenadeProjectiles keeps track of the location of live grenades (parser.gameState.grenadeProjectiles), actively thrown by players.
// It does NOT track the location of grenades lying on the ground, i.e. that were dropped by dead players.
//
//nolint:gocognit,gocyclo,funlen
func (p *parser) bindGrenadeProjectiles(entity st.Entity) {
	entityID := entity.ID()

	proj := common.NewGrenadeProjectile(p.nextUniqueID())
	proj.Entity = entity
	p.gameState.grenadeProjectiles[entityID] = proj

	if p.demoInfoProvider.IsSource2() {
		ownerEntVal := entity.PropertyValueMust("m_hOwnerEntity")
		if ownerEntVal.Any != nil {
			player := p.demoInfoProvider.FindPlayerByPawnHandle(ownerEntVal.Handle())
			proj.Thrower = player
			proj.Owner = player
		}
	}

	var wep common.EquipmentType

	entity.OnCreateFinished(func() {
		if p.demoInfoProvider.IsSource2() {
			modelVal := entity.PropertyValueMust("CBodyComponent.m_skeletonInstance.m_modelState.m_hModel")

			if modelVal.Any != nil {
				model := modelVal.S2UInt64()

				if weaponType, exists := p.equipmentTypePerModel[model]; exists {
					wep = weaponType
				} else {
					// The model hash isn't in equipmentTypePerModel yet (it's built lazily from weapon
					// entities), e.g. on a mid-round GOTV start or a post-patch model variant. Fall back
					// to the projectile's server class.
					wep = equipmentTypeFromProjectileClass(entity.ServerClass().Name())

					switch wep { //nolint:exhaustive // only the two warning cases are of interest
					case common.EqUnknown:
						p.eventDispatcher.Dispatch(events.ParserWarn{
							Message: fmt.Sprintf("unknown grenade model %d", model),
							Type:    events.WarnTypeUnknownGrenadeModel,
						})
					case common.EqMolotov:
						// CMolotovProjectile is shared by molotov and incendiary; the class alone can't
						// tell them apart, so EqMolotov here is a best-effort guess - warn so consumers
						// know WeaponInstance.Type may be wrong for this projectile.
						p.eventDispatcher.Dispatch(events.ParserWarn{
							Message: fmt.Sprintf("grenade model %d unknown; guessed molotov from class (may be incendiary)", model),
							Type:    events.WarnTypeUnknownGrenadeModel,
						})
					}
				}
			}
		}

		// CS2 removes the grenade from the thrower's inventory before the projectile entity is
		// fully created - usually when the last grenade of the type is thrown (#580). In that
		// case getPlayerWeapon() can't find the weapon in the inventory and returns a new one
		// without an entity, so fall back to the last-known weapon of that type.
		wepInstance := getPlayerWeapon(proj.Thrower, wep, "", p.nextUniqueID())
		if wepInstance.Entity == nil && proj.Thrower != nil {
			if lastKnownWep := p.gameState.lastKnownGrenadeWeapon(proj.Thrower, wepInstance.Type); lastKnownWep != nil {
				wepInstance = lastKnownWep
			}
		}

		// copy the weapon so it doesn't get overwritten by a new entity in parser.weapons
		wepCopy := *wepInstance
		wepCopy.OriginalString = ""

		// the inventory weapon's owner may be unknown if it was never updated since the
		// parser started (e.g. mid-round GOTV start) - the thrower is always known here though
		if proj.Thrower != nil {
			wepCopy.Owner = proj.Thrower
		}

		proj.WeaponInstance = &wepCopy

		unassert.NotNilf(proj.WeaponInstance, "couldn't find grenade instance for player")

		if proj.WeaponInstance != nil {
			unassert.NotNilf(proj.WeaponInstance.Owner, "getPlayerWeapon() returned weapon instance with Owner=nil")
		}

		p.gameEventHandler.addThrownGrenade(proj.Thrower, proj.WeaponInstance)

		if p.demoInfoProvider.IsSource2() && wep == common.EqFlash {
			p.gameState.flyingFlashbangs = append(p.gameState.flyingFlashbangs, &FlyingFlashbang{
				projectile: proj,
			})
		}

		newPos := proj.Position()

		proj.Trajectory = append(proj.Trajectory, newPos) //nolint:staticcheck

		proj.Trajectory2 = append(proj.Trajectory2, common.TrajectoryEntry{
			Position: newPos,
			FrameID:  p.CurrentFrame(),
			Time:     p.CurrentTime(),
		})

		if p.isSource2() && !p.disableMimicSource1GameEvents {
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
		newPos := proj.Position()

		proj.Trajectory = append(proj.Trajectory, newPos) //nolint:staticcheck

		proj.Trajectory2 = append(proj.Trajectory2, common.TrajectoryEntry{
			Position: newPos,
			FrameID:  p.CurrentFrame(),
			Time:     p.CurrentTime(),
		})

		// a replaced projectile (e.g. round restart) didn't detonate
		if p.demoInfoProvider.IsSource2() && wep == common.EqFlash && !p.disableMimicSource1GameEvents && !isEntityReplaced(entity) {
			p.gameEventHandler.dispatch(events.FlashExplode{
				GrenadeEvent: events.GrenadeEvent{
					GrenadeType:     common.EqFlash,
					Grenade:         proj.WeaponInstance,
					Projectile:      proj,
					Position:        proj.Position(),
					Thrower:         proj.Thrower,
					GrenadeEntityID: proj.Entity.ID(),
				},
			})

			// Mark the entry for the projectile that actually detonated (entity-keyed), not the
			// head of the queue - otherwise a zero-victim detonation while another flashbang is
			// airborne shifts victim attribution by one for the rest of the round.
			for _, flashbang := range p.gameState.flyingFlashbangs {
				if flashbang.projectile == proj {
					flashbang.explodedFrame = p.currentFrame
					flashbang.position = proj.Position()

					break
				}
			}
		}

		p.nadeProjectileDestroyed(proj)
	})

	if !p.demoInfoProvider.IsSource2() {
		entity.Property("m_nModelIndex").OnUpdate(func(val st.PropertyValue) {
			wep = p.grenadeModelIndices[val.Int()]
		})
	}

	// @micvbang: not quite sure what the difference between Thrower and Owner is.
	entity.Property("m_hThrower").OnUpdate(func(val st.PropertyValue) {
		if val.Any == nil {
			return
		}

		if p.demoInfoProvider.IsSource2() {
			proj.Thrower = p.demoInfoProvider.FindPlayerByPawnHandle(val.Handle())
		} else {
			proj.Thrower = p.gameState.Participants().FindByHandle(val.Int())
		}
	})

	entity.Property("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
		if val.Any == nil {
			return
		}

		if p.demoInfoProvider.IsSource2() {
			proj.Owner = p.gameState.Participants().FindByPawnHandle(val.Handle())
		} else {
			proj.Owner = p.gameState.Participants().FindByHandle(val.Int())
		}
	})

	// Some demos don't have this property as it seems
	// So we need to check for nil and can't send out bounce events if it's missing
	if bounceProp := entity.Property("m_nBounces"); bounceProp != nil {
		bounceProp.OnUpdate(func(val st.PropertyValue) {
			if val.Any == nil {
				return
			}

			bounceNumber := val.Int()
			if bounceNumber != 0 {
				p.eventDispatcher.Dispatch(events.GrenadeProjectileBounce{
					Projectile: proj,
					BounceNr:   bounceNumber,
				})
			}

			newPos := proj.Position()

			proj.Trajectory = append(proj.Trajectory, newPos) //nolint:staticcheck

			proj.Trajectory2 = append(proj.Trajectory2, common.TrajectoryEntry{
				Position: newPos,
				FrameID:  p.CurrentFrame(),
				Time:     p.CurrentTime(),
			})
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

	// A replaced projectile (its index was re-used, e.g. on a round restart) didn't detonate - it won't flash
	// anybody, spawn an inferno or expire.
	replaced := isEntityReplaced(proj.Entity)

	if proj.WeaponInstance.Type == common.EqFlash && !replaced {
		p.gameState.lastFlash.projectileByPlayer[proj.Owner] = proj
	}

	// We delete from the Owner.ThrownGrenades (only if not inferno or smoke, because they will be deleted when they expire)
	isInferno := proj.WeaponInstance.Type == common.EqMolotov || proj.WeaponInstance.Type == common.EqIncendiary
	isSmoke := proj.WeaponInstance.Type == common.EqSmoke
	isDecoy := proj.WeaponInstance.Type == common.EqDecoy

	if replaced || (!isInferno && !isSmoke && !isDecoy) {
		p.gameEventHandler.removeThrownGrenade(proj.WeaponInstance)
	}

	if isInferno && !replaced {
		if ownerVal := proj.Entity.PropertyValueMust("m_hOwnerEntity"); ownerVal.Any != nil {
			p.linkInfernoProjectile(pendingInfernoLink{
				frame:       p.currentFrame,
				ownerHandle: ownerVal.Handle(),
				projectile:  proj,
			})
		}
	}
}

// linkInfernoProjectile links an inferno (CInferno entity) to the molotov / incendiary projectile it
// was spawned from, so FireGrenadeStart / FireGrenadeExpired can carry the projectile.
//
// The CInferno entity has no property referencing the projectile. However the server destroys the
// projectile and creates the inferno in the same tick, so both show up in the same frame (in either
// order) with the same m_hOwnerEntity (the thrower's pawn). It is called once for each side; the
// side that comes first is kept as pending until the other one arrives in the same frame.
//
// A link is only made if exactly one counterpart from the same owner is pending in the current frame,
// otherwise (e.g. the projectile was extinguished by a smoke and never spawned an inferno, or the
// inferno entity was re-created after leaving the PVS in a POV demo) the inferno stays unlinked.
func (p *parser) linkInfernoProjectile(link pendingInfernoLink) {
	gs := p.gameState
	kept := gs.pendingInfernoLinks[:0]
	match := -1
	nMatches := 0

	for _, pending := range gs.pendingInfernoLinks {
		if pending.frame != link.frame {
			// stale - its counterpart never showed up
			continue
		}

		isCounterpart := (pending.projectile == nil) != (link.projectile == nil)
		if isCounterpart && pending.ownerHandle == link.ownerHandle {
			match = len(kept)
			nMatches++
		}

		kept = append(kept, pending)
	}

	gs.pendingInfernoLinks = kept

	switch nMatches {
	case 0:
		gs.pendingInfernoLinks = append(gs.pendingInfernoLinks, link)
	case 1:
		counterpart := gs.pendingInfernoLinks[match]
		gs.pendingInfernoLinks = append(gs.pendingInfernoLinks[:match], gs.pendingInfernoLinks[match+1:]...)

		if link.projectile != nil {
			gs.infernoProjectiles[counterpart.infernoEntityID] = link.projectile
		} else {
			gs.infernoProjectiles[link.infernoEntityID] = counterpart.projectile
		}
	default:
		// ambiguous - don't guess
	}
}

//nolint:funlen,gocognit
func (p *parser) bindWeaponS2(entity st.Entity) {
	entityID := entity.ID()
	itemIndexVal := entity.PropertyValueMust("m_AttributeManager.m_Item.m_iItemDefinitionIndex")

	if itemIndexVal.Any == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Type:    events.WarnTypeMissingItemDefinitionIndex,
			Message: "missing m_iItemDefinitionIndex property in weapon entity",
		})

		return
	}

	itemIndex := itemIndexVal.S2UInt64()
	wepType := common.EquipmentIndexMapping[itemIndex]

	if wepType == common.EqUnknown {
		// Dispatch on the event dispatcher (the one RegisterEventHandler binds to) so consumers can
		// actually receive and filter this warning, rather than the msg dispatcher.
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("unknown equipment with index %d", itemIndex),
			Type:    events.WarnTypeUnknownEquipmentIndex,
		})
	} else {
		model := entity.PropertyValueMust("CBodyComponent.m_skeletonInstance.m_modelState.m_hModel").S2UInt64()
		p.equipmentTypePerModel[model] = wepType
	}

	equipment, exists := p.gameState.weapons[entityID]
	if !exists {
		equipment = common.NewEquipment(wepType, p.nextUniqueID())
		p.gameState.weapons[entityID] = equipment
	} else {
		equipment.Type = wepType
	}

	equipment.Entity = entity

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
		if val.Any == nil {
			return
		}

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

	entity.OnDestroy(func() {
		owner := p.GameState().Participants().FindByPawnHandle(entity.PropertyValueMust("m_hOwnerEntity").Handle())
		// replaced weapons (e.g. round restart / backup restore) weren't refunded, even though the money may have changed
		if owner != nil && !isEntityReplaced(entity) && owner.IsInBuyZone() &&
			p.GameState().IngameTick() == lastMoneyUpdateTick && lastMoneyIncreased {
			p.eventDispatcher.Dispatch(events.ItemRefund{
				Player: owner,
				Weapon: equipment,
			})
		}

		lastMoneyIncreased = false
		equipment.Owner = nil

		// the entry may already belong to the weapon entity replacing this one (see bindPlayerWeapons)
		if p.gameState.weapons[entityID] == equipment {
			delete(p.gameState.weapons, entityID)
		}
	})

	// Detect weapon firing, we don't use m_iClip1 because it would not work with weapons such as the knife (no ammo).
	// WeaponFire events for grenades are dispatched when the grenade's projectile is created.
	if p.isSource2() && equipment.Class() != common.EqClassGrenade && !p.disableMimicSource1GameEvents {
		entity.Property("m_fLastShotTime").OnUpdate(func(val st.PropertyValue) {
			if val.Any == nil {
				return
			}

			ownerHandleVal := entity.PropertyValueMust("m_hOwnerEntity")

			var shooter *common.Player

			if ownerHandleVal.Any != nil {
				shooter = p.GameState().Participants().FindByPawnHandle(ownerHandleVal.Handle())
			}

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

	entity.Property("m_iClip1").OnUpdate(func(val st.PropertyValue) {
		if equipment.Owner != nil {
			equipment.Owner.IsReloading = false
		}
	})
}

func (p *parser) bindWeapon(entity st.Entity, wepType common.EquipmentType) {
	entityID := entity.ID()

	eq, eqExists := p.gameState.weapons[entityID]
	if !eqExists {
		eq = common.NewEquipment(wepType, p.nextUniqueID())
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
		// the entry may already belong to the weapon entity replacing this one (see bindPlayerWeapons)
		if p.gameState.weapons[entityID] == eq {
			delete(p.gameState.weapons, entityID)
		}
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

	switch eq.Type { //nolint:exhaustive
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
	ownerEntVal := entity.PropertyValueMust("m_hOwnerEntity")

	if ownerEntVal.Any == nil {
		return
	}

	throwerHandle := ownerEntVal.Handle()

	var thrower *common.Player

	if p.isSource2() {
		thrower = p.gameState.Participants().FindByPawnHandle(throwerHandle)
	} else {
		thrower = p.gameState.Participants().FindByHandle64(throwerHandle)
	}

	inf := common.NewInferno(p.demoInfoProvider, entity, thrower, p.nextUniqueID())
	p.gameState.infernos[entity.ID()] = inf

	p.linkInfernoProjectile(pendingInfernoLink{
		frame:           p.currentFrame,
		ownerHandle:     throwerHandle,
		infernoEntityID: entity.ID(),
	})

	entity.OnCreateFinished(func() {
		// at the end of the frame - the projectile may be destroyed later in the same frame (see linkInfernoProjectile)
		p.delayedEventHandlers = append(p.delayedEventHandlers, func() {
			p.eventDispatcher.Dispatch(events.InfernoStart{
				Inferno:    inf,
				Projectile: p.gameState.infernoProjectiles[entity.ID()],
			})
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

	proj := p.gameState.infernoProjectiles[inf.Entity.ID()]

	p.eventDispatcher.Dispatch(events.InfernoExpired{
		Inferno:    inf,
		Projectile: proj,
	})

	delete(p.gameState.infernos, inf.Entity.ID())
	delete(p.gameState.infernoFireStates, inf.Entity.ID())
	delete(p.gameState.infernoProjectiles, inf.Entity.ID())

	if proj != nil && proj.WeaponInstance != nil {
		p.gameEventHandler.removeThrownGrenade(proj.WeaponInstance)
	} else {
		p.gameEventHandler.deleteThrownGrenade(inf.Thrower(), common.EqIncendiary)
	}
}

//nolint:funlen,gocognit,gocyclo
func (p *parser) bindGameRules() {
	gameRules := p.ServerClasses().FindByName("CCSGameRulesProxy")
	gameRules.OnEntityCreated(func(entity st.Entity) {
		grPrefix := func(s string) string {
			if p.isSource2() {
				return fmt.Sprintf("%s.%s", gameRulesPrefixS2, s)
			}

			return fmt.Sprintf("%s.%s", gameRulesPrefix, s)
		}

		p.gameState.rules.isSource2 = p.isSource2()
		p.gameState.rules.entity = entity

		roundTime := entity.PropertyValueMust(grPrefix("m_iRoundTime")).Int()
		hasRescueZone := entity.PropertyValueMust(grPrefix("m_bMapHasRescueZone")).BoolVal()
		hasBombTarget := entity.PropertyValueMust(grPrefix("m_bMapHasBombTarget")).BoolVal()

		dispatchRoundStart := func() {
			p.gameEventHandler.clearGrenadeProjectiles()

			if p.disableMimicSource1GameEvents {
				return
			}

			if p.gameState.TotalRoundsPlayed() > 0 {
				p.gameEventHandler.dispatch(events.RoundEndOfficial{})
			}

			var objective string

			switch {
			case hasBombTarget:
				objective = "BOMB TARGET"
			case hasRescueZone:
				objective = "HOSTAGE RESCUE"
			default:
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

			if p.isSource2() {
				if p.disableMimicSource1GameEvents {
					p.eventDispatcher.Dispatch(freezetimeEvent)
				} else {
					p.gameState.lastFreezeTimeChangedEvent = &freezetimeEvent
				}
			} else {
				p.eventDispatcher.Dispatch(freezetimeEvent)
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

			switch p.gameState.gamePhase { //nolint:exhaustive
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
			if p.isSource2() && !p.disableMimicSource1GameEvents {
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

		if p.isSource2() {
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

				winner := common.TeamUnassigned

				// StillInProgress is handled above and VIP reasons don't exist in CS:GO/CS2,
				// they intentionally fall through to the UNKNOWN/TeamUnassigned defaults.
				switch reason { //nolint:exhaustive
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

				var (
					winnerState *common.TeamState
					loserState  *common.TeamState
				)

				if winner != common.TeamUnassigned && winner != common.TeamSpectators {
					winnerState = p.gameState.Team(winner)
					loserState = winnerState.Opponent
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

				p.gameState.currentPlanter = nil
			})
		}
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
	return vectorA.Sub(vectorB).Norm()
}

func (p *parser) getClosestBombsiteFromPosition(position r3.Vector) events.Bombsite {
	distanceFromBombsiteA := getDistanceBetweenVectors(position, p.bombsiteA.center)
	distanceFromBombsiteB := getDistanceBetweenVectors(position, p.bombsiteB.center)

	if distanceFromBombsiteA < distanceFromBombsiteB {
		return events.BombsiteA
	}

	return events.BombsiteB
}
