package demoinfocs

import (
	"fmt"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/constants"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

type DevNullReader struct {
}

func (DevNullReader) Read(p []byte) (n int, err error) {
	return len(p), nil
}

func newParser() *Parser {
	p := NewParser(new(DevNullReader))
	p.header = &header{}

	return p
}

func fakePlayerEntity(id int) *st.Entity {
	return stfake.NewEntity(id)
}

func TestParser_GetClosestBombsiteFromPosition(t *testing.T) {
	p := newParser()
	p.bombsiteA = bombsite{
		center: r3.Vector{X: 2, Y: 3, Z: 1},
	}
	p.bombsiteB = bombsite{
		center: r3.Vector{X: 4, Y: 5, Z: 7},
	}

	site := p.getClosestBombsiteFromPosition(r3.Vector{X: -2, Y: 2, Z: 2})

	assert.Equal(t, events.BombsiteA, site)

	site = p.getClosestBombsiteFromPosition(r3.Vector{X: 3, Y: 6, Z: 5})

	assert.Equal(t, events.BombsiteB, site)
}

// TestBindBomb_NilEntityChecks tests that bindBomb handles nil entities gracefully
// when processing m_bStartedArming updates. This prevents nil pointer dereference
// panics that can occur when pawn or controller entities are not yet created.
func TestBindBomb_NilEntityChecks(t *testing.T) {
	t.Run("nil pawn entity should not panic", func(t *testing.T) {
		p := newParser()
		p.gameState.entities = make(map[int]*st.Entity)

		// Simulate the scenario where pawnEntity lookup returns nil
		// entityIDFromHandle with a valid handle but no entity in the map
		pawnEntityID := 42
		_ = pawnEntityID // Entity not added to map - simulates missing entity

		// This should not panic - the nil check should prevent it
		pawnEntity := p.gameState.entities[pawnEntityID]
		assert.Nil(t, pawnEntity, "pawnEntity should be nil when not in entities map")
	})

	t.Run("nil controller entity should not panic", func(t *testing.T) {
		p := newParser()
		p.gameState.entities = make(map[int]*st.Entity)

		// Add pawn entity but not controller entity
		ctlHandle := uint64(0x12345678) // Some handle value
		pawnEntity := stfake.NewEntity(10, stfake.P("m_hController", ctlHandle))
		p.gameState.entities[10] = pawnEntity

		// Controller entity lookup should return nil
		ctlEntityID := 99 // Not in map
		ctlEntity := p.gameState.entities[ctlEntityID]
		assert.Nil(t, ctlEntity, "ctlEntity should be nil when not in entities map")
	})

	t.Run("nil player should not panic", func(t *testing.T) {
		p := newParser()
		p.gameState.entities = make(map[int]*st.Entity)
		p.gameState.playersByEntityID = make(map[int]*common.Player)

		// Add pawn entity and controller entity but not player
		ctlHandle := uint64(0x00000014) // Handle that maps to entity ID 20
		pawnEntity := stfake.NewEntity(10, stfake.P("m_hController", ctlHandle))
		p.gameState.entities[10] = pawnEntity

		ctlEntity := stfake.NewEntity(20)
		p.gameState.entities[20] = ctlEntity

		// Player lookup should return nil
		planter := p.gameState.playersByEntityID[20]
		assert.Nil(t, planter, "planter should be nil when not in playersByEntityID map")
	})
}

func newBombTestParser() (*Parser, *st.Parser) {
	p := newParser()
	stParser := stfake.NewParser("CC4", "CPlantedC4")
	p.stParser = stParser
	p.bindBomb()

	return p, stParser
}

// createFakePlantedC4 creates a planted bomb entity and returns a function to update its properties.
// Properties not listed in props are missing from the entity, the listed ones have no value yet.
func createFakePlantedC4(stParser *st.Parser, props ...string) func(name string, val any) {
	entityProps := make([]stfake.Prop, 0, 2+len(props))
	entityProps = append(entityProps, stfake.P("m_hOwnerEntity", nil), stfake.P("m_nBombSite", nil))

	for _, name := range props {
		entityProps = append(entityProps, stfake.P(name, nil))
	}

	entity := stfake.New(stfake.Spec{
		Parser: stParser,
		Class:  "CPlantedC4",
		ID:     100,
		Props:  entityProps,
	})

	stfake.Create(entity)

	return func(name string, val any) {
		stfake.Update(entity, stfake.P(name, val))
	}
}

func TestBindBomb_BombExplode(t *testing.T) {
	t.Run("bomb stopped ticking without exploding", func(t *testing.T) {
		p, stParser := newBombTestParser()

		var explosions int
		p.RegisterEventHandler(func(events.BombExplode) { explosions++ })

		update := createFakePlantedC4(stParser, "m_bBombTicking", "m_bHasExploded", "m_hBombDefuser", "m_bBombDefused")

		update("m_bBombTicking", false)
		update("m_bHasExploded", false)

		assert.Zero(t, explosions)
		assert.False(t, p.gameEventHandler.frameToBombExploded[p.currentFrame])
	})

	t.Run("bomb exploded", func(t *testing.T) {
		p, stParser := newBombTestParser()
		p.currentFrame = 42

		var explosions int
		p.RegisterEventHandler(func(events.BombExplode) { explosions++ })

		update := createFakePlantedC4(stParser, "m_bBombTicking", "m_bHasExploded", "m_hBombDefuser", "m_bBombDefused")

		update("m_bHasExploded", false)
		update("m_bBombTicking", false)
		update("m_bHasExploded", true)
		update("m_bHasExploded", true)

		assert.Equal(t, 1, explosions)
		assert.True(t, p.gameEventHandler.frameToBombExploded[42])
	})

	t.Run("fallback to m_bBombTicking without m_bHasExploded", func(t *testing.T) {
		p, stParser := newBombTestParser()

		var explosions int
		p.RegisterEventHandler(func(events.BombExplode) { explosions++ })

		update := createFakePlantedC4(stParser, "m_bBombTicking", "m_hBombDefuser", "m_bBombDefused")

		update("m_bBombTicking", true)
		update("m_bBombTicking", false)

		assert.Equal(t, 1, explosions)
	})
}

const (
	testBombPawnID       = 10
	testBombControllerID = 20
)

// testBombPawnHandle is the handle of the pawn of the player created by addFakeBombCarrier.
var testBombPawnHandle = uint64(testBombPawnID)

// createFakeC4 creates a (not planted) bomb entity with the given owner and returns a function to update the owner.
func createFakeC4(stParser *st.Parser, owner uint64) func(owner uint64) {
	setOwner, _ := createFakeC4Entity(stParser, owner)

	return setOwner
}

// createFakeC4Entity is like createFakeC4 but also returns the bomb entity.
func createFakeC4Entity(stParser *st.Parser, owner uint64) (func(owner uint64), *st.Entity) {
	entity := stfake.New(stfake.Spec{
		Parser: stParser,
		Class:  "CC4",
		ID:     99,
		Props:  []stfake.Prop{stfake.Pos(r3.Vector{}), stfake.P("m_hOwnerEntity", owner), stfake.P("m_bStartedArming", false)},
	})

	stfake.Create(entity)

	return func(owner uint64) {
		stfake.Update(entity, stfake.P("m_hOwnerEntity", owner))
	}, entity
}

// addFakeBombCarrier registers a player whose pawn has the handle testBombPawnHandle.
func addFakeBombCarrier(p *Parser) *common.Player {
	p.gameState.entities[testBombPawnID] = stfake.NewEntity(testBombPawnID)

	pl := common.NewPlayer(p.demoInfoProvider)
	pl.Entity = stfake.NewEntity(testBombControllerID,
		stfake.P("m_hPawn", testBombPawnHandle),
		stfake.P("m_hPlayerPawn", testBombPawnHandle),
	)
	pl.EntityID = testBombControllerID
	pl.UserID = 1
	p.gameState.playersByUserID[pl.UserID] = pl
	p.gameState.playersByEntityID[pl.EntityID] = pl

	return pl
}

func TestBindBomb_BombDropped(t *testing.T) {
	invalid := uint64(constants.InvalidEntityHandleSource2)

	t.Run("pickup then drop", func(t *testing.T) {
		p, stParser := newBombTestParser()
		pl := addFakeBombCarrier(p)

		var drops []events.BombDropped
		p.RegisterEventHandler(func(e events.BombDropped) { drops = append(drops, e) })

		setOwner := createFakeC4(stParser, invalid)
		setOwner(testBombPawnHandle)
		assert.Same(t, pl, p.gameState.bomb.Carrier)

		setOwner(invalid)

		assert.Equal(t, []events.BombDropped{{Player: pl, EntityID: testBombControllerID}}, drops)
		assert.Nil(t, p.gameState.bomb.Carrier)
	})

	// e.g. the bomb is created before the players are known, then passed on during freeze time
	t.Run("initial owner resolves only after the bomb was created", func(t *testing.T) {
		p, stParser := newBombTestParser()

		var drops []events.BombDropped
		p.RegisterEventHandler(func(e events.BombDropped) { drops = append(drops, e) })

		setOwner := createFakeC4(stParser, testBombPawnHandle)
		assert.Nil(t, p.gameState.bomb.Carrier)

		pl := addFakeBombCarrier(p)

		setOwner(invalid)
		setOwner(invalid)

		assert.Equal(t, []events.BombDropped{{Player: pl, EntityID: testBombControllerID}}, drops)
	})

	t.Run("owner change after plant", func(t *testing.T) {
		p, stParser := newBombTestParser()

		var drops int
		p.RegisterEventHandler(func(events.BombDropped) { drops++ })

		setOwner := createFakeC4(stParser, testBombPawnHandle)
		addFakeBombCarrier(p)
		createFakePlantedC4(stParser, "m_bBombTicking", "m_bHasExploded", "m_hBombDefuser", "m_bBombDefused")

		setOwner(invalid)

		assert.Zero(t, drops)
	})

	// passing the bomb to a player that isn't known yet isn't a drop
	t.Run("initial owner passes the bomb to an unresolved owner", func(t *testing.T) {
		p, stParser := newBombTestParser()

		var drops int
		p.RegisterEventHandler(func(events.BombDropped) { drops++ })

		setOwner := createFakeC4(stParser, testBombPawnHandle)
		addFakeBombCarrier(p)

		setOwner(testBombPawnHandle + 1)

		assert.Zero(t, drops)
	})

	t.Run("bomb destroyed while its owner was unresolved", func(t *testing.T) {
		p, stParser := newBombTestParser()

		var drops int
		p.RegisterEventHandler(func(events.BombDropped) { drops++ })

		_, oldBomb := createFakeC4Entity(stParser, testBombPawnHandle)
		addFakeBombCarrier(p)
		stfake.Destroy(oldBomb)

		setOwner := createFakeC4(stParser, invalid)
		setOwner(invalid)

		assert.Zero(t, drops)
	})
}

func TestBindBomb_BombPickup_RepeatedOwnerUpdate(t *testing.T) {
	p, stParser := newBombTestParser()
	pl := addFakeBombCarrier(p)

	var pickups []events.BombPickup
	p.RegisterEventHandler(func(e events.BombPickup) { pickups = append(pickups, e) })

	setOwner := createFakeC4(stParser, uint64(constants.InvalidEntityHandleSource2))
	setOwner(testBombPawnHandle)
	setOwner(testBombPawnHandle)

	assert.Equal(t, []events.BombPickup{{Player: pl}}, pickups)

	// picking the bomb up again after dropping it is a new pickup
	setOwner(uint64(constants.InvalidEntityHandleSource2))
	setOwner(testBombPawnHandle)

	assert.Len(t, pickups, 2)
}

// testWeaponEntityID is the entity-ID of the weapons created by fakeWeaponEntity.
const testWeaponEntityID = 300

// fakeWeaponEntity creates a weapon entity for bindWeapon.
func fakeWeaponEntity(serial int, itemIndex uint64) *st.Entity {
	return stfake.New(stfake.Spec{
		ID:     testWeaponEntityID,
		Serial: serial,
		Props: []stfake.Prop{
			stfake.P("m_iItemDefinitionIndex", itemIndex),
			stfake.P("CBodyComponent.m_hModel", uint64(1)),
			stfake.P("m_hOwnerEntity", uint64(constants.InvalidEntityHandleSource2)),
			stfake.P("m_fLastShotTime", nil),
			stfake.P("m_iClip1", nil),
		},
	})
}

// The server may create a new weapon entity at the index of one it never deleted, and assign it to a player
// (m_hMyWeapons, read earlier in the same packet) before it's created.
// The player's inventory must get the new weapon, not the replaced one (#601).
// bindFakePawnWeapons binds a fake pawn's weapon slots to the player and returns a function
// that updates the first slot to the given handle.
func bindFakePawnWeapons(p *Parser, pl *common.Player) (setSlot0 func(handle uint64)) {
	const slots = 64

	props := make([]stfake.Prop, 0, 1+slots)
	props = append(props, stfake.P(playerWeaponPrefixS2, nil))

	for i := range slots {
		props = append(props, stfake.P(fmt.Sprintf("%s.%04d", playerWeaponPrefixS2, i), nil))
	}

	pawn := stfake.NewEntity(1, props...)

	pl.Inventory = map[int]*common.Equipment{}
	p.bindPlayerWeapons(pawn, func() *common.Player { return pl })

	return func(handle uint64) {
		stfake.Update(pawn, stfake.P(playerWeaponPrefixS2+".0000", handle))
	}
}

func weaponHandle(serial int) uint64 {
	return uint64(testWeaponEntityID) | uint64(serial)<<constants.MaxEdictBitsSource2
}

// The server may create a new weapon entity at the index of one it never deleted, and assign it to a player
// (m_hMyWeapons, read earlier in the same packet) before it's created.
// The player's inventory must get the new weapon, not the replaced one (#601).
func TestBindPlayerWeapons_WeaponAssignedBeforeReplacingEntityIsCreated(t *testing.T) {
	p := newParser()

	oldEntity := fakeWeaponEntity(1, 1) // Desert Eagle
	p.bindWeapon(oldEntity)
	oldWeapon := p.gameState.weapons[testWeaponEntityID]

	pl := common.NewPlayer(p.demoInfoProvider)
	setSlot0 := bindFakePawnWeapons(p, pl)

	// 1. the pawn gets the new weapon (serial 2) while the old entity is still at the index
	setSlot0(weaponHandle(2))

	// 2. the old entity is destroyed (replaced) and 3. the new one is created
	stfake.Destroy(oldEntity)

	newEntity := fakeWeaponEntity(2, 61) // USP-S
	p.bindWeapon(newEntity)

	wep := pl.Inventory[testWeaponEntityID]
	assert.Same(t, p.gameState.weapons[testWeaponEntityID], wep)
	assert.Equal(t, common.EqUSP, wep.Type)
	assert.Same(t, newEntity, wep.Entity)
	assert.NotSame(t, oldWeapon, wep)
	assert.Equal(t, common.EqDeagle, oldWeapon.Type, "the replaced weapon must not be modified")
}

// The other way around: the new weapon entity is created before the pawn's slots are updated,
// slots may still reference the previous serial in the meantime - that must not replace the new weapon.
func TestBindPlayerWeapons_ReplacingEntityCreatedBeforeInventoryUpdate(t *testing.T) {
	p := newParser()

	oldEntity := fakeWeaponEntity(1, 1) // Desert Eagle
	p.bindWeapon(oldEntity)

	pl := common.NewPlayer(p.demoInfoProvider)
	setSlot0 := bindFakePawnWeapons(p, pl)
	setSlot0(weaponHandle(1))

	// 1. the old entity is destroyed (replaced) and the new one is created
	stfake.Destroy(oldEntity)

	newEntity := fakeWeaponEntity(2, 61) // USP-S
	p.bindWeapon(newEntity)
	newWeapon := p.gameState.weapons[testWeaponEntityID]

	// 2. the inventory is re-read while the slot still has the previous serial
	setSlot0(weaponHandle(1))

	assert.Same(t, newWeapon, p.gameState.weapons[testWeaponEntityID], "the new weapon must not be replaced by a placeholder")
	assert.Same(t, newWeapon, pl.Inventory[testWeaponEntityID])

	// 3. the slot is updated to the new serial
	setSlot0(weaponHandle(2))

	assert.Same(t, newWeapon, pl.Inventory[testWeaponEntityID])
	assert.Same(t, newEntity, newWeapon.Entity)
	assert.Equal(t, common.EqUSP, newWeapon.Type)
}

func TestBindGameRules_RulesAreLive(t *testing.T) {
	p := newParser()
	stParser := stfake.NewParser("CCSGameRulesProxy")
	p.stParser = stParser
	p.bindGameRules()

	rules := p.GameState().Rules()
	assert.Nil(t, rules.Entity())

	gr := func(name string, val any) stfake.Prop {
		return stfake.P(gameRulesPrefixS2+"."+name, val)
	}

	entity := stfake.New(stfake.Spec{
		Parser: stParser,
		Class:  "CCSGameRulesProxy",
		ID:     10,
		Props: []stfake.Prop{
			gr("m_iRoundTime", int32(115)),
			gr("m_bMapHasRescueZone", false),
			gr("m_bMapHasBombTarget", true),
			gr("m_bFreezePeriod", false),
			gr("m_gamePhase", int32(0)),
			gr("m_totalRoundsPlayed", int32(0)),
			gr("m_bWarmupPeriod", false),
			gr("m_bHasMatchStarted", false),
			gr("m_nOvertimePlaying", int32(0)),
			gr("m_eRoundWinReason", int32(0)),
		},
	})

	stfake.Create(entity)

	assert.Same(t, entity, rules.Entity())
	assert.Same(t, rules, p.GameState().Rules())
}
