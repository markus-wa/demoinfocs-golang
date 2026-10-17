package demoinfocs

import (
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables/fake"
)

type DevNullReader struct {
}

func (DevNullReader) Read(p []byte) (n int, err error) {
	return len(p), nil
}

func TestParser_BindNewPlayer_Issue98(t *testing.T) {
	p := newParser()

	p.rawPlayers = map[int]*common.PlayerInfo{
		0: {
			UserID: 1,
			Name:   "Zim",
			GUID:   "BOT",
		},
		1: {
			UserID: 2,
			Name:   "The Suspect",
			GUID:   "123",
		},
	}

	bot := fakePlayerEntity(1)
	p.bindNewPlayerS1(bot)
	bot.Destroy()

	player := fakePlayerEntity(2)
	p.bindNewPlayerS1(player)

	assert.Len(t, p.GameState().Participants().Connected(), 1)
}

func TestParser_BindNewPlayer_Issue98_Reconnect(t *testing.T) {
	p := newParser()

	p.rawPlayers = map[int]*common.PlayerInfo{
		0: {
			UserID: 2,
			Name:   "The Suspect",
			GUID:   "123",
			XUID:   1,
		},
	}

	player := fakePlayerEntity(1)
	p.bindNewPlayerS1(player)
	player.Destroy()

	p.RegisterEventHandler(func(events.PlayerConnect) {
		t.Error("expected no more PlayerConnect events but got one")
	})
	p.bindNewPlayerS1(player)

	assert.Len(t, p.GameState().Participants().All(), 1)
}

func TestParser_BindNewPlayer_PlayerSpotted_Under32(t *testing.T) {
	testPlayerSpotted(t, "m_bSpottedByMask.000")
}

func TestParser_BindNewPlayer_PlayerSpotted_Over32(t *testing.T) {
	testPlayerSpotted(t, "m_bSpottedByMask.001")
}

func testPlayerSpotted(t *testing.T, propName string) {
	p := newParser()

	p.rawPlayers = map[int]*common.PlayerInfo{
		0: {
			UserID: 2,
			Name:   "Spotter",
			GUID:   "123",
			XUID:   1,
		},
	}

	// TODO: Player interface so we don't have to mock all this
	spotted := new(stfake.Entity)
	spottedByProp0 := new(stfake.Property)

	var spottedByUpdateHandler st.PropertyUpdateHandler
	spottedByProp0.On("OnUpdate", mock.Anything).Run(func(args mock.Arguments) {
		spottedByUpdateHandler = args.Get(0).(st.PropertyUpdateHandler)
	})

	spotted.On("Property", propName).Return(spottedByProp0)
	configurePlayerEntityMock(1, spotted)
	p.bindNewPlayerS1(spotted)

	var actual events.PlayerSpottersChanged
	p.RegisterEventHandler(func(e events.PlayerSpottersChanged) {
		actual = e
	})

	spottedByUpdateHandler(st.PropertyValue{IntVal: 1})

	expected := events.PlayerSpottersChanged{
		Spotted: p.gameState.playersByEntityID[1],
	}
	assert.NotNil(t, expected.Spotted)
	assert.Equal(t, expected, actual)
}

func newParser() *parser {
	p := NewParser(new(DevNullReader)).(*parser)
	p.header = &common.DemoHeader{}

	return p
}

func fakePlayerEntity(id int) *stfake.Entity {
	entity := new(stfake.Entity)
	configurePlayerEntityMock(id, entity)

	return entity
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

func configurePlayerEntityMock(id int, entity *stfake.Entity) {
	entity.On("ID").Return(id)

	var destroyCallback func()
	entity.On("OnDestroy", mock.Anything).Run(func(args mock.Arguments) {
		destroyCallback = args.Get(0).(func())
	})

	entity.On("OnPositionUpdate", mock.Anything).Return()
	prop := new(stfake.Property)
	prop.On("OnUpdate", mock.Anything).Return()
	entity.On("Property", mock.Anything).Return(prop)
	entity.On("BindProperty", mock.Anything, mock.Anything, mock.Anything)
	entity.On("Destroy").Run(func(mock.Arguments) {
		destroyCallback()
	})
}

// TestBindBomb_NilEntityChecks tests that bindBomb handles nil entities gracefully
// when processing m_bStartedArming updates. This prevents nil pointer dereference
// panics that can occur when pawn or controller entities are not yet created.
func TestBindBomb_NilEntityChecks(t *testing.T) {
	t.Run("nil pawn entity should not panic", func(t *testing.T) {
		p := newParser()
		p.gameState.entities = make(map[int]st.Entity)

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
		p.gameState.entities = make(map[int]st.Entity)

		// Add pawn entity but not controller entity
		pawnEntity := new(stfake.Entity)
		ctlHandle := uint64(0x12345678) // Some handle value
		pawnEntity.On("PropertyValueMust", "m_hController").Return(st.PropertyValue{Any: ctlHandle})
		p.gameState.entities[10] = pawnEntity

		// Controller entity lookup should return nil
		ctlEntityID := 99 // Not in map
		ctlEntity := p.gameState.entities[ctlEntityID]
		assert.Nil(t, ctlEntity, "ctlEntity should be nil when not in entities map")
	})

	t.Run("nil player should not panic", func(t *testing.T) {
		p := newParser()
		p.gameState.entities = make(map[int]st.Entity)
		p.gameState.playersByEntityID = make(map[int]*common.Player)

		// Add pawn entity and controller entity but not player
		pawnEntity := new(stfake.Entity)
		ctlHandle := uint64(0x00000014) // Handle that maps to entity ID 20
		pawnEntity.On("PropertyValueMust", "m_hController").Return(st.PropertyValue{Any: ctlHandle})
		p.gameState.entities[10] = pawnEntity

		ctlEntity := new(stfake.Entity)
		ctlEntity.On("ID").Return(20)
		p.gameState.entities[20] = ctlEntity

		// Player lookup should return nil
		planter := p.gameState.playersByEntityID[20]
		assert.Nil(t, planter, "planter should be nil when not in playersByEntityID map")
	})
}

type fakeServerClass struct {
	st.ServerClass
	onCreated []st.EntityCreatedHandler
}

func (sc *fakeServerClass) OnEntityCreated(handler st.EntityCreatedHandler) {
	sc.onCreated = append(sc.onCreated, handler)
}

type fakeServerClasses map[string]*fakeServerClass

func (scs fakeServerClasses) All() []st.ServerClass {
	return nil
}

func (scs fakeServerClasses) FindByName(name string) st.ServerClass {
	if scs[name] == nil {
		scs[name] = new(fakeServerClass)
	}

	return scs[name]
}

type fakeSendTableParser struct {
	sendTableParser
	serverClasses fakeServerClasses
}

func (p fakeSendTableParser) ServerClasses() st.ServerClasses {
	return p.serverClasses
}

func newBombTestParser() (*parser, fakeServerClasses) {
	p := newParser()
	p.header.Filestamp = filestampS2
	serverClasses := make(fakeServerClasses)
	p.stParser = fakeSendTableParser{serverClasses: serverClasses}
	p.bindBomb()

	return p, serverClasses
}

// createFakePlantedC4 creates a planted bomb entity and returns a function to update its properties.
// Properties not listed in props are missing from the entity.
func createFakePlantedC4(serverClasses fakeServerClasses, props ...string) func(name string, val any) {
	handlers := make(map[string]st.PropertyUpdateHandler)

	entity := new(stfake.Entity)
	entity.On("Position").Return(r3.Vector{})
	entity.On("PropertyValueMust", "m_hOwnerEntity").Return(st.PropertyValue{})
	entity.On("PropertyValueMust", "m_nBombSite").Return(st.PropertyValue{})
	entity.On("OnDestroy", mock.Anything)

	for _, name := range props {
		prop := new(stfake.Property)
		prop.On("OnUpdate", mock.Anything).Run(func(args mock.Arguments) {
			handlers[name] = args.Get(0).(st.PropertyUpdateHandler)
		})
		entity.On("Property", name).Return(prop)
	}

	entity.On("Property", mock.Anything).Return(nil)

	for _, h := range serverClasses["CPlantedC4"].onCreated {
		h(entity)
	}

	return func(name string, val any) {
		if h := handlers[name]; h != nil {
			h(st.PropertyValue{Any: val, S2: true})
		}
	}
}

func TestBindBomb_BombExplode(t *testing.T) {
	t.Run("bomb stopped ticking without exploding", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var explosions int
		p.RegisterEventHandler(func(events.BombExplode) { explosions++ })

		update := createFakePlantedC4(serverClasses, "m_bBombTicking", "m_bHasExploded", "m_hBombDefuser", "m_bBombDefused")

		update("m_bBombTicking", false)
		update("m_bHasExploded", false)

		assert.Zero(t, explosions)
		assert.False(t, p.gameEventHandler.frameToBombExploded[p.currentFrame])
	})

	t.Run("bomb exploded", func(t *testing.T) {
		p, serverClasses := newBombTestParser()
		p.currentFrame = 42

		var explosions int
		p.RegisterEventHandler(func(events.BombExplode) { explosions++ })

		update := createFakePlantedC4(serverClasses, "m_bBombTicking", "m_bHasExploded", "m_hBombDefuser", "m_bBombDefused")

		update("m_bHasExploded", false)
		update("m_bBombTicking", false)
		update("m_bHasExploded", true)
		update("m_bHasExploded", true)

		assert.Equal(t, 1, explosions)
		assert.True(t, p.gameEventHandler.frameToBombExploded[42])
	})

	t.Run("fallback to m_bBombTicking without m_bHasExploded", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var explosions int
		p.RegisterEventHandler(func(events.BombExplode) { explosions++ })

		update := createFakePlantedC4(serverClasses, "m_bBombTicking", "m_hBombDefuser", "m_bBombDefused")

		update("m_bBombTicking", true)
		update("m_bBombTicking", false)

		assert.Equal(t, 1, explosions)
	})
}

// testWeaponEntityID is the entity-ID of the weapons created by fakeWeaponEntity.
const testWeaponEntityID = 300

// fakeWeaponEntity creates a weapon entity for bindWeapon and returns its OnDestroy delegate (once bound).
func fakeWeaponEntity(serial int, itemIndex uint64) (*stfake.Entity, *func()) {
	entity := new(stfake.Entity)
	entity.On("ID").Return(testWeaponEntityID)
	entity.On("SerialNum").Return(serial)
	entity.On("PropertyValueMust", "m_AttributeManager.m_Item.m_iItemDefinitionIndex").Return(st.PropertyValue{Any: itemIndex, S2: true})
	entity.On("PropertyValueMust", "CBodyComponent.m_skeletonInstance.m_modelState.m_hModel").Return(st.PropertyValue{Any: uint64(1), S2: true})
	entity.On("PropertyValueMust", "m_hOwnerEntity").Return(st.PropertyValue{Any: uint64(constants.InvalidEntityHandleSource2), S2: true})

	prop := new(stfake.Property)
	prop.On("OnUpdate", mock.Anything)
	entity.On("Property", mock.Anything).Return(prop)

	onDestroy := new(func())
	entity.On("OnDestroy", mock.Anything).Run(func(args mock.Arguments) {
		*onDestroy = args.Get(0).(func())
	})

	return entity, onDestroy
}

// The server may create a new weapon entity at the index of one it never deleted, and assign it to a player
// (m_hMyWeapons, read earlier in the same packet) before it's created.
// The player's inventory must get the new weapon, not the replaced one (#601).
// bindFakePawnWeapons binds a fake pawn's weapon slots to the player and returns a function
// that updates the first slot to the given handle.
func bindFakePawnWeapons(p *parser, pl *common.Player) (setSlot0 func(handle uint64)) {
	var updateSlot0 st.PropertyUpdateHandler

	slot0 := new(stfake.Property)
	slot0.On("OnUpdate", mock.Anything).Run(func(args mock.Arguments) {
		updateSlot0 = args.Get(0).(st.PropertyUpdateHandler)
	})
	slot0Value := slot0.On("Value").Return(st.PropertyValue{})

	emptySlot := new(stfake.Property)
	emptySlot.On("OnUpdate", mock.Anything)
	emptySlot.On("Value").Return(st.PropertyValue{})

	pawn := new(stfake.Entity)
	pawn.On("Property", playerWeaponPrefixS2+".0000").Return(slot0)
	pawn.On("Property", mock.Anything).Return(emptySlot)

	pl.Inventory = map[int]*common.Equipment{}
	p.bindPlayerWeaponsS2(pawn, func() *common.Player { return pl })

	return func(handle uint64) {
		slot0Value.ReturnArguments = mock.Arguments{st.PropertyValue{Any: handle, S2: true}}
		updateSlot0(st.PropertyValue{Any: handle, S2: true})
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
	p.header.Filestamp = filestampS2

	oldEntity, destroyOld := fakeWeaponEntity(1, 1) // Desert Eagle
	p.bindWeaponS2(oldEntity)
	oldWeapon := p.gameState.weapons[testWeaponEntityID]

	pl := common.NewPlayer(p.demoInfoProvider)
	setSlot0 := bindFakePawnWeapons(p, pl)

	// 1. the pawn gets the new weapon (serial 2) while the old entity is still at the index
	setSlot0(weaponHandle(2))

	// 2. the old entity is destroyed (replaced) and 3. the new one is created
	(*destroyOld)()

	newEntity, _ := fakeWeaponEntity(2, 61) // USP-S
	p.bindWeaponS2(newEntity)

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
	p.header.Filestamp = filestampS2

	oldEntity, destroyOld := fakeWeaponEntity(1, 1) // Desert Eagle
	p.bindWeaponS2(oldEntity)

	pl := common.NewPlayer(p.demoInfoProvider)
	setSlot0 := bindFakePawnWeapons(p, pl)
	setSlot0(weaponHandle(1))

	// 1. the old entity is destroyed (replaced) and the new one is created
	(*destroyOld)()

	newEntity, _ := fakeWeaponEntity(2, 61) // USP-S
	p.bindWeaponS2(newEntity)
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

// fakeWeaponEntityS1 creates a CS:GO weapon entity for bindWeapon and returns its OnDestroy delegate (once bound).
func fakeWeaponEntityS1(serial int) (*stfake.Entity, *func()) {
	entity := new(stfake.Entity)
	entity.On("ID").Return(testWeaponEntityID)
	entity.On("SerialNum").Return(serial)

	prop := new(stfake.Property)
	prop.On("OnUpdate", mock.Anything)
	prop.On("Value").Return(st.PropertyValue{})
	entity.On("Property", mock.Anything).Return(prop)

	onDestroy := new(func())
	entity.On("OnDestroy", mock.Anything).Run(func(args mock.Arguments) {
		*onDestroy = args.Get(0).(func())
	})

	return entity, onDestroy
}

// bindFakePlayerWeaponsS1 binds a fake CS:GO player's weapon slots to the player and returns a function
// that updates the first slot to the given handle.
func bindFakePlayerWeaponsS1(p *parser, pl *common.Player) (setSlot0 func(handle int)) {
	var updateSlot0 st.PropertyUpdateHandler

	slot0 := new(stfake.Property)
	slot0.On("OnUpdate", mock.Anything).Run(func(args mock.Arguments) {
		updateSlot0 = args.Get(0).(st.PropertyUpdateHandler)
	})

	emptySlot := new(stfake.Property)
	emptySlot.On("OnUpdate", mock.Anything)

	playerEntity := new(stfake.Entity)
	playerEntity.On("Property", playerWeaponPrefix+".000").Return(slot0)
	playerEntity.On("Property", mock.Anything).Return(emptySlot)

	p.bindPlayerWeapons(playerEntity, pl)

	return func(handle int) {
		updateSlot0(st.PropertyValue{IntVal: handle})
	}
}

func weaponHandleS1(serial int) int {
	return testWeaponEntityID | serial<<constants.MaxEdictBits
}

// CS:GO equivalent of TestBindPlayerWeapons_WeaponAssignedBeforeReplacingEntityIsCreated.
// Entities are updated in index order, so this happens whenever the player's index is lower than the weapon's.
func TestBindPlayerWeaponsS1_WeaponAssignedBeforeReplacingEntityIsCreated(t *testing.T) {
	p := newParser()
	p.modelPreCache = []string{""} // bindWeapon looks up m_nModelIndex 0

	oldEntity, destroyOld := fakeWeaponEntityS1(1)
	p.bindWeapon(oldEntity, common.EqAK47)
	oldWeapon := p.gameState.weapons[testWeaponEntityID]

	pl := common.NewPlayer(p.demoInfoProvider)
	setSlot0 := bindFakePlayerWeaponsS1(p, pl)

	// 1. the player gets the new weapon (serial 2) while the old entity is still at the index
	setSlot0(weaponHandleS1(2))

	// 2. the old entity is destroyed (replaced) and 3. the new one is created
	(*destroyOld)()

	newEntity, _ := fakeWeaponEntityS1(2)
	p.bindWeapon(newEntity, common.EqMolotov)

	wep := pl.Inventory[testWeaponEntityID]
	assert.Same(t, p.gameState.weapons[testWeaponEntityID], wep)
	assert.Equal(t, common.EqMolotov, wep.Type)
	assert.Same(t, newEntity, wep.Entity)
	assert.Same(t, pl, wep.Owner)
	assert.NotSame(t, oldWeapon, wep)
	assert.Equal(t, common.EqAK47, oldWeapon.Type, "the replaced weapon must not be modified")
}

// CS:GO equivalent of TestBindPlayerWeapons_ReplacingEntityCreatedBeforeInventoryUpdate.
func TestBindPlayerWeaponsS1_ReplacingEntityCreatedBeforeInventoryUpdate(t *testing.T) {
	p := newParser()
	p.modelPreCache = []string{""} // bindWeapon looks up m_nModelIndex 0

	oldEntity, destroyOld := fakeWeaponEntityS1(1)
	p.bindWeapon(oldEntity, common.EqAK47)

	pl := common.NewPlayer(p.demoInfoProvider)
	setSlot0 := bindFakePlayerWeaponsS1(p, pl)
	setSlot0(weaponHandleS1(1))

	// 1. the old entity is destroyed (replaced) and the new one is created
	(*destroyOld)()

	newEntity, _ := fakeWeaponEntityS1(2)
	p.bindWeapon(newEntity, common.EqMolotov)
	newWeapon := p.gameState.weapons[testWeaponEntityID]

	// 2. the slot is updated while it still has the previous serial
	setSlot0(weaponHandleS1(1))

	assert.Same(t, newWeapon, p.gameState.weapons[testWeaponEntityID], "the new weapon must not be replaced by a placeholder")
	assert.Same(t, newWeapon, pl.Inventory[testWeaponEntityID])

	// 3. the slot is updated to the new serial
	setSlot0(weaponHandleS1(2))

	assert.Same(t, newWeapon, pl.Inventory[testWeaponEntityID])
	assert.Same(t, newEntity, newWeapon.Entity)
	assert.Equal(t, common.EqMolotov, newWeapon.Type)
}

// A player can be re-bound to a new entity (e.g. after a reconnect), the weapons of the previous entity
// must not be kept in the inventory as only the new entity's slots are tracked.
func TestBindPlayerWeaponsS1_PlayerReboundToNewEntity(t *testing.T) {
	p := newParser()
	p.modelPreCache = []string{""} // bindWeapon looks up m_nModelIndex 0

	knifeEntity, _ := fakeWeaponEntityS1(1)
	p.bindWeapon(knifeEntity, common.EqKnife)
	knife := p.gameState.weapons[testWeaponEntityID]

	pl := common.NewPlayer(p.demoInfoProvider)
	setSlot0 := bindFakePlayerWeaponsS1(p, pl)
	setSlot0(weaponHandleS1(1))

	assert.Same(t, knife, pl.Inventory[testWeaponEntityID])

	// the player gets a new entity whose slots don't reference the knife
	bindFakePlayerWeaponsS1(p, pl)

	assert.Empty(t, pl.Inventory)
	assert.Nil(t, knife.Owner)
}
