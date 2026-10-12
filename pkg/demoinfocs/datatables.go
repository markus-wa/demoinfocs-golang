package demoinfocs

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/go-unassert"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/constants"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
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

//nolint:gocognit,gocyclo,funlen
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
			if val.BoolVal() { //nolint:nestif
				planterHandle := bombEntity.PropertyValueMust("m_hOwnerEntity").Handle()

				pawnEntity := p.gameState.entities[entityIDFromHandle(planterHandle)]
				if pawnEntity == nil {
					return
				}

				ctlHandle := pawnEntity.PropertyValueMust("m_hController").Handle()

				ctlEntity := p.gameState.entities[entityIDFromHandle(ctlHandle)]
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
				site := events.BombsiteUnknown

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

		var planter *common.Player

		if ownerProp.Any != nil {
			planter = p.gameState.Participants().FindByPawnHandle(ownerProp.Handle())

			if planter != nil {
				planter.IsPlanting = false
			}
		}

		siteNumberVal := bombEntity.PropertyValueMust("m_nBombSite")

		site := events.BombsiteUnknown

		if siteNumberVal.Any != nil {
			siteNumber := siteNumberVal.Int()
			switch siteNumber {
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

		target.BindProperty("m_vecMins", &t.min, st.ValTypeVector)
		target.BindProperty("m_vecMaxs", &t.max, st.ValTypeVector)
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
		p.bindNewPlayerController(player)
	})
	p.stParser.ServerClasses().FindByName("CCSPlayerPawn").OnEntityCreated(func(player st.Entity) {
		p.bindNewPlayerPawn(player)
	})
}

// playerUserID returns the user-ID under which a player with the given raw info is registered
// (-1 if there is no raw info). Game-event user-IDs only carry the slot in the low byte.
func playerUserID(rp *common.PlayerInfo) int {
	userID := -1

	if rp != nil {
		userID = rp.UserID
	}

	if userID <= math.MaxUint16 {
		userID &= 0xff
	}

	return userID
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
// (see bindNewPlayerController), so it always sees the controller's complete state - never a half-applied
// take-over.
func (p *parser) getOrCreatePlayerFromControllerEntity(controllerEntity st.Entity) *common.Player {
	controllerEntityID := controllerEntity.ID()
	p.gameState.playerControllerEntities[controllerEntityID] = controllerEntity

	steamID := controllerEntity.PropertyValueMust("m_steamID").UInt64()
	name := controllerEntity.PropertyValueMust("m_iszPlayerName").String()
	isBot := steamID == 0
	rp := p.rawPlayerForController(controllerEntity, isBot)
	userID := playerUserID(rp)

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
		player.Team = common.Team(team.UInt64())
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
		pl.Team = common.Team(d.entity.PropertyValueMust("m_iTeamNum").UInt64())
		pl.TeamState = p.gameState.Team(pl.Team)
	}

	if d.connectionChanged {
		p.applyControllerConnectionState(pl, d.entity)
	}
}

func (p *parser) applyControllerConnectionState(pl *common.Player, controllerEntity st.Entity) {
	state := controllerEntity.PropertyValueMust("m_iConnected").UInt32()

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
	if !ok || player.IsConnected || state.UInt32() != 0 {
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

	if !isBot && !r.IsFakePlayer && r.XUID != 0 && r.XUID == controllerEntity.Property("m_steamID").Value().UInt64() {
		return r
	}

	if isBot && r.IsFakePlayer && r.XUID == 0 && r.Name == controllerEntity.PropertyValueMust("m_iszPlayerName").String() {
		return r
	}

	return nil
}

func (p *parser) bindNewPlayerController(controllerEntity st.Entity) {
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

//nolint:funlen
func (p *parser) bindNewPlayerPawn(pawnEntity st.Entity) {
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
		p.bindPlayerWeapons(pawnEntity, func() *common.Player {
			if current := p.gameState.playersByEntityID[controllerEntityID]; current != nil {
				return current
			}

			return pl
		})
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

			state := val.UInt64()
			pl.ButtonsPressedState = state

			p.eventDispatcher.Dispatch(events.PlayerButtonsStateUpdate{
				Player:       pl,
				ButtonsState: state,
			})
		})
	}
}

//nolint:funlen
func (p *parser) bindPlayerWeapons(pawnEntity st.Entity, currentPlayer func() *common.Player) {
	const inventoryCapacity = 64

	inventorySize := 64

	type eq struct {
		*common.Equipment
		entityID int
	}

	playerInventory := make(map[int]eq)

	getWep := func(wepSlotPropertyValue st.PropertyValue) (uint64, *common.Equipment) {
		entityID := wepSlotPropertyValue.UInt64() & constants.EntityHandleIndexMaskSource2
		wep := p.gameState.weapons[int(entityID)]

		if wep == nil {
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
		inventorySize = len(pv.Array())

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
			sc.OnEntityCreated(p.bindWeapon)
		}

		if hasThrower {
			sc.OnEntityCreated(p.bindGrenadeProjectiles)
		}
	}

	p.stParser.ServerClasses().FindByName("CInferno").OnEntityCreated(p.bindNewInferno)
}

// bindGrenadeProjectiles keeps track of the location of live grenades (parser.gameState.grenadeProjectiles), actively thrown by players.
// It does NOT track the location of grenades lying on the ground, i.e. that were dropped by dead players.
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

//nolint:funlen,gocognit
func (p *parser) bindGrenadeProjectiles(entity st.Entity) {
	entityID := entity.ID()

	proj := common.NewGrenadeProjectile(p.nextUniqueID())
	proj.Entity = entity
	p.gameState.grenadeProjectiles[entityID] = proj

	ownerEntVal := entity.PropertyValueMust("m_hOwnerEntity")
	if ownerEntVal.Any != nil {
		player := p.demoInfoProvider.FindPlayerByPawnHandle(ownerEntVal.Handle())
		proj.Thrower = player
		proj.Owner = player
	}

	var wep common.EquipmentType

	entity.OnCreateFinished(func() { //nolint:wsl
		modelVal := entity.PropertyValueMust("CBodyComponent.m_hModel")

		if modelVal.Any != nil {
			model := modelVal.UInt64()

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

		if proj.WeaponInstance.Type == common.EqFlash {
			p.gameState.flyingFlashbangs = append(p.gameState.flyingFlashbangs, &FlyingFlashbang{
				projectile: proj,
			})
		}

		proj.Trajectory = append(proj.Trajectory, common.TrajectoryEntry{
			Tick:     p.gameState.ingameTick,
			Position: proj.Position(),
			FrameID:  p.CurrentFrame(),
			Time:     p.CurrentTime(),
		})

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
		proj.Trajectory = append(proj.Trajectory, common.TrajectoryEntry{
			Tick:     p.gameState.ingameTick,
			Position: proj.Position(),
			FrameID:  p.CurrentFrame(),
			Time:     p.CurrentTime(),
		})

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

	// @micvbang: not quite sure what the difference between Thrower and Owner is.
	entity.Property("m_hThrower").OnUpdate(func(val st.PropertyValue) {
		if val.Any == nil {
			return
		}

		proj.Thrower = p.demoInfoProvider.FindPlayerByPawnHandle(val.Handle())
	})

	entity.Property("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
		if val.Any == nil {
			return
		}

		proj.Owner = p.gameState.Participants().FindByPawnHandle(val.Handle())
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

			proj.Trajectory = append(proj.Trajectory, common.TrajectoryEntry{
				Tick:     p.gameState.ingameTick,
				Position: proj.Position(),
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

//nolint:funlen
func (p *parser) bindWeapon(entity st.Entity) {
	entityID := entity.ID()
	itemIndexVal := entity.PropertyValueMust("m_iItemDefinitionIndex")

	if itemIndexVal.Any == nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Type:    events.WarnTypeMissingItemDefinitionIndex,
			Message: "missing m_iItemDefinitionIndex property in weapon entity",
		})

		return
	}

	itemIndex := itemIndexVal.UInt64()
	wepType := common.EquipmentIndexMapping[itemIndex]

	if wepType == common.EqUnknown {
		// Dispatch on the event dispatcher (the one RegisterEventHandler binds to) so consumers can
		// actually receive and filter this warning, rather than the msg dispatcher.
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: fmt.Sprintf("unknown equipment with index %d", itemIndex),
			Type:    events.WarnTypeUnknownEquipmentIndex,
		})
	} else {
		model := entity.PropertyValueMust("CBodyComponent.m_hModel").UInt64()
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
		if owner != nil && owner.IsInBuyZone() && p.GameState().IngameTick() == lastMoneyUpdateTick && lastMoneyIncreased {
			p.eventDispatcher.Dispatch(events.ItemRefund{
				Player: owner,
				Weapon: equipment,
			})
		}

		lastMoneyIncreased = false
		equipment.Owner = nil

		delete(p.gameState.weapons, entityID)
	})

	// Detect weapon firing, we don't use m_iClip1 because it would not work with weapons such as the knife (no ammo).
	// WeaponFire events for grenades are dispatched when the grenade's projectile is created.
	if equipment.Class() != common.EqClassGrenade && !p.disableMimicSource1GameEvents {
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

func (p *parser) bindNewInferno(entity st.Entity) {
	ownerEntVal := entity.PropertyValueMust("m_hOwnerEntity")

	if ownerEntVal.Any == nil {
		return
	}

	throwerHandle := ownerEntVal.Handle()

	thrower := p.gameState.Participants().FindByPawnHandle(throwerHandle)

	inf := common.NewInferno(p.demoInfoProvider, entity, thrower, p.nextUniqueID())
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
	delete(p.gameState.infernoFireStates, inf.Entity.ID())

	p.gameEventHandler.deleteThrownGrenade(inf.Thrower(), common.EqIncendiary)
}

//nolint:funlen,gocognit,gocyclo
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

			var winner = common.TeamUnassigned

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
	return math.Sqrt((vectorA.X-vectorB.X)*(vectorA.X-vectorB.X) + (vectorA.Y-vectorB.Y)*(vectorA.Y-vectorB.Y) + (vectorA.Z-vectorB.Z)*(vectorA.Z-vectorB.Z))
}

func (p *parser) getClosestBombsiteFromPosition(position r3.Vector) events.Bombsite {
	distanceFromBombsiteA := getDistanceBetweenVectors(position, p.bombsiteA.center)
	distanceFromBombsiteB := getDistanceBetweenVectors(position, p.bombsiteB.center)

	if distanceFromBombsiteA < distanceFromBombsiteB {
		return events.BombsiteA
	}

	return events.BombsiteB
}
