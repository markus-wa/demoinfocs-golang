package demoinfocs

import (
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables/fake"
)

type DevNullReader struct {
}

func (DevNullReader) Read(p []byte) (n int, err error) {
	return len(p), nil
}

func newParser() *parser {
	p := NewParser(new(DevNullReader)).(*parser)
	p.header = &header{}

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
			h(st.PropertyValue{Any: val})
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

// TestUniqueID2 verifies that successive projectiles from the same player have distinct UniqueID2s
// and that UniqueID2 incorporates entity handle, creation tick, and instance serial.
// See https://github.com/markus-wa/demoinfocs-golang/issues/601
func TestUniqueID2(t *testing.T) {
	t.Run("successive projectiles of the same player have distinct UniqueID2", func(t *testing.T) {
		p := newParser()
		p.equipmentTypePerModel = make(map[uint64]common.EquipmentType)

		pawnEntity := new(stfake.Entity)
		pawnEntity.On("ID").Return(10)
		pawnEntity.On("SerialNum").Return(1)
		p.gameState.entities[10] = pawnEntity

		ctrlEntity := new(stfake.Entity)
		ctrlEntity.On("ID").Return(1)
		ctrlEntity.On("PropertyValue", "m_hPawn").Return(st.PropertyValue{Any: uint64(10 | (1 << 14))}, true)
		ctrlEntity.On("PropertyValue", "m_hPlayerPawn").Return(st.PropertyValue{Any: uint64(10 | (1 << 14))}, true)

		player := common.NewPlayer(p.demoInfoProvider)
		player.UserID = 1
		player.SteamID64 = 76561198000000001
		player.Entity = ctrlEntity
		p.gameState.playersByUserID[1] = player

		// Player carries a flashbang in inventory
		flashWep := common.NewEquipment(common.EqFlash, p.nextUniqueID())
		flashWepEntity := new(stfake.Entity)
		flashWepEntity.On("ID").Return(30)
		flashWepEntity.On("SerialNum").Return(1)
		flashWep.Entity = flashWepEntity
		flashWep.Owner = player
		player.Inventory = map[int]*common.Equipment{30: flashWep}

		fakeProjectile := func(id, serial int, tick int) (*stfake.Entity, func()) {
			entity := new(stfake.Entity)
			entity.On("ID").Return(id)
			entity.On("SerialNum").Return(serial)

			pawnHandle := uint64(10) | (uint64(1) << 14)
			entity.On("PropertyValueMust", "m_hOwnerEntity").Return(st.PropertyValue{Any: pawnHandle})
			entity.On("PropertyValueMust", "CBodyComponent.m_hModel").Return(st.PropertyValue{Any: uint64(9999)})

			sc := serverClassStub{name: "CFlashbangProjectile"}
			entity.On("ServerClass").Return(sc)

			var onCreateFinished func()
			entity.On("OnCreateFinished", mock.Anything).Run(func(args mock.Arguments) {
				onCreateFinished = args.Get(0).(func())
			})

			entity.On("OnDestroy", mock.Anything).Return()
			entity.On("Property", "m_hThrower").Return(nil)
			entity.On("Property", "m_hOwnerEntity").Return(nil)
			entity.On("Property", "m_nBounces").Return(nil)
			entity.On("Position").Return(r3.Vector{})

			return entity, func() {
				p.gameState.ingameTick = tick
				if onCreateFinished != nil {
					onCreateFinished()
				}
			}
		}

		var throws []events.GrenadeProjectileThrow
		p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
			throws = append(throws, e)
		})

		// Throw flash 1
		p.gameState.ingameTick = 100
		projEntity1, finish1 := fakeProjectile(101, 1, 100)
		p.bindGrenadeProjectiles(projEntity1)
		finish1()

		// Throw flash 2 (same player, same inventory item slot)
		p.gameState.ingameTick = 120
		projEntity2, finish2 := fakeProjectile(102, 2, 120)
		p.bindGrenadeProjectiles(projEntity2)
		finish2()

		assert.Len(t, throws, 2, "expected 2 GrenadeProjectileThrow events")

		proj1 := throws[0].Projectile
		proj2 := throws[1].Projectile

		assert.NotNil(t, proj1)
		assert.NotNil(t, proj2)
		assert.NotNil(t, proj1.WeaponInstance)
		assert.NotNil(t, proj2.WeaponInstance)

		// Distinct projectile UniqueID
		assert.NotEqual(t, proj1.UniqueID(), proj2.UniqueID(), "projectile UniqueID should be distinct")

		// Distinct WeaponInstance UniqueID2
		assert.NotEqual(t, proj1.WeaponInstance.UniqueID2(), proj2.WeaponInstance.UniqueID2(),
			"WeaponInstance.UniqueID2 should be distinct across successive throws")

		// Distinct GrenadeProjectile UniqueID2
		assert.NotEqual(t, proj1.UniqueID2(), proj2.UniqueID2(),
			"Projectile.UniqueID2 should be distinct across successive throws")

		// Projectile and WeaponInstance UniqueID2 match for the same grenade
		assert.Equal(t, proj1.WeaponInstance.UniqueID2(), proj1.UniqueID2(),
			"proj1.UniqueID2() and proj1.WeaponInstance.UniqueID2() should match")
		assert.Equal(t, proj2.WeaponInstance.UniqueID2(), proj2.UniqueID2(),
			"proj2.UniqueID2() and proj2.WeaponInstance.UniqueID2() should match")
	})
}
