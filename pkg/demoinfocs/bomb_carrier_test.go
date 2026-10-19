package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables/fake"
)

const (
	testBombPawnID       = 10
	testBombControllerID = 20
	// testBombPawnHandle is the handle of the pawn of the player created by addFakeBombCarrier.
	testBombPawnHandle = uint64(testBombPawnID)
	// testBombC4ID is the entity-ID of the bomb created by createFakeC4.
	testBombC4ID = 99
)

// createFakeC4 creates a (not planted) bomb entity with the given owner and returns a function to update the owner,
// and a function that destroys the entity.
func createFakeC4(p *parser, serverClasses fakeServerClasses, owner uint64) (setOwner func(owner uint64), destroy func()) {
	var (
		ownerHandlers []st.PropertyUpdateHandler
		onDestroy     []func()
	)

	ownerProp := new(stfake.Property)
	ownerProp.On("OnUpdate", mock.Anything).Run(func(args mock.Arguments) {
		ownerHandlers = append(ownerHandlers, args.Get(0).(st.PropertyUpdateHandler))
	})

	otherProp := new(stfake.Property)
	otherProp.On("OnUpdate", mock.Anything)

	entity := new(stfake.Entity)
	entity.On("ID").Return(testBombC4ID)
	entity.On("OnPositionUpdate", mock.Anything)
	entity.On("OnDestroy", mock.Anything).Run(func(args mock.Arguments) {
		onDestroy = append(onDestroy, args.Get(0).(func()))
	})
	entity.On("Property", "m_hOwnerEntity").Return(ownerProp)
	entity.On("Property", mock.Anything).Return(otherProp)
	ownerValue := entity.On("PropertyValueMust", "m_hOwnerEntity").Return(st.PropertyValue{Any: owner, S2: true})

	setOwner = func(owner uint64) {
		val := st.PropertyValue{Any: owner, S2: true}
		ownerValue.ReturnArguments = mock.Arguments{val}

		for _, h := range ownerHandlers {
			h(val)
		}
	}

	// like the sendtables parser: a new entity at the same index replaces the old one in the entity map
	p.gameState.entities[testBombC4ID] = entity

	for _, h := range serverClasses["CC4"].onCreated {
		h(entity)
	}

	// the update handlers get the initial values after the entity was created
	setOwner(owner)

	destroy = func() {
		delete(p.gameState.entities, testBombC4ID)

		for _, f := range onDestroy {
			f()
		}
	}

	return setOwner, destroy
}

// fakeBombCarrierController returns a controller entity whose pawn has the given handle.
func fakeBombCarrierController(pawnHandle uint64) *stfake.Entity {
	controller := new(stfake.Entity)
	controller.On("ID").Return(testBombControllerID)
	controller.On("PropertyValue", "m_hPawn").Return(st.PropertyValue{Any: pawnHandle, S2: true}, true)
	controller.On("PropertyValue", "m_hPlayerPawn").Return(st.PropertyValue{Any: pawnHandle, S2: true}, true)

	return controller
}

// fakePawn adds a pawn entity with the given ID to the entity map.
func fakePawn(p *parser, id int) {
	pawn := new(stfake.Entity)
	pawn.On("ID").Return(id)

	p.gameState.entities[id] = pawn
}

// addFakeBombCarrier registers a player whose pawn has the handle testBombPawnHandle.
func addFakeBombCarrier(p *parser) *common.Player {
	fakePawn(p, testBombPawnID)

	pl := common.NewPlayer(p.demoInfoProvider)
	pl.Entity = fakeBombCarrierController(testBombPawnHandle)
	pl.EntityID = testBombControllerID
	pl.UserID = 1
	p.gameState.playersByUserID[pl.UserID] = pl
	p.gameState.playersByEntityID[pl.EntityID] = pl

	return pl
}

func TestBindBomb_BombDropped(t *testing.T) {
	invalid := uint64(constants.InvalidEntityHandleSource2)

	t.Run("pickup then drop", func(t *testing.T) {
		p, serverClasses := newBombTestParser()
		pl := addFakeBombCarrier(p)

		var drops []events.BombDropped
		p.RegisterEventHandler(func(e events.BombDropped) { drops = append(drops, e) })

		setOwner, _ := createFakeC4(p, serverClasses, invalid)
		setOwner(testBombPawnHandle)
		assert.Same(t, pl, p.gameState.bomb.Carrier)

		setOwner(invalid)

		assert.Equal(t, []events.BombDropped{{Player: pl, EntityID: testBombC4ID}}, drops)
		assert.Nil(t, p.gameState.bomb.Carrier)
	})

	// e.g. the bomb comes before its owner's pawn in the first packet of the demo, then is dropped during freeze time
	t.Run("initial owner resolves later in the frame", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var (
			pickups []events.BombPickup
			drops   []events.BombDropped
		)

		p.RegisterEventHandler(func(e events.BombPickup) { pickups = append(pickups, e) })
		p.RegisterEventHandler(func(e events.BombDropped) { drops = append(drops, e) })

		setOwner, _ := createFakeC4(p, serverClasses, testBombPawnHandle)
		assert.Nil(t, p.gameState.bomb.Carrier)

		pl := addFakeBombCarrier(p)
		p.processFrameGameEvents()

		assert.Same(t, pl, p.gameState.bomb.Carrier)
		assert.Equal(t, []events.BombPickup{{Player: pl}}, pickups)

		setOwner(invalid)
		setOwner(invalid)

		assert.Equal(t, []events.BombDropped{{Player: pl, EntityID: testBombC4ID}}, drops)
	})

	t.Run("initial owner never resolves", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var dispatched int
		p.RegisterEventHandler(func(events.BombPickup) { dispatched++ })
		p.RegisterEventHandler(func(events.BombDropped) { dispatched++ })

		setOwner, _ := createFakeC4(p, serverClasses, testBombPawnHandle)
		p.processFrameGameEvents()
		setOwner(invalid)

		assert.Nil(t, p.gameState.bomb.Carrier)
		assert.Zero(t, dispatched)
	})

	t.Run("owner change after plant", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var drops int
		p.RegisterEventHandler(func(events.BombDropped) { drops++ })

		setOwner, _ := createFakeC4(p, serverClasses, testBombPawnHandle)
		addFakeBombCarrier(p)
		createFakePlantedC4(serverClasses, "m_bBombTicking", "m_bHasExploded", "m_hBombDefuser", "m_bBombDefused")

		setOwner(invalid)

		assert.Zero(t, drops)
	})

	// passing the bomb to a player that isn't known yet isn't a drop
	t.Run("initial owner passes the bomb to an unresolved owner", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var drops int
		p.RegisterEventHandler(func(events.BombDropped) { drops++ })

		setOwner, _ := createFakeC4(p, serverClasses, testBombPawnHandle)
		addFakeBombCarrier(p)

		setOwner(testBombPawnHandle + 1)
		p.processFrameGameEvents()

		assert.Zero(t, drops)
	})

	t.Run("carrier passes the bomb to an unresolved owner", func(t *testing.T) {
		p, serverClasses := newBombTestParser()
		pl := addFakeBombCarrier(p)

		var drops []events.BombDropped
		p.RegisterEventHandler(func(e events.BombDropped) { drops = append(drops, e) })

		setOwner, _ := createFakeC4(p, serverClasses, testBombPawnHandle)
		setOwner(invalid)
		setOwner(testBombPawnHandle)
		drops = nil

		setOwner(testBombPawnHandle + 1)

		assert.Empty(t, drops)
		assert.Nil(t, p.gameState.bomb.Carrier)

		// the new owner resolves (pawn index testBombPawnID+1) and drops the bomb
		fakePawn(p, testBombPawnID+1)
		pl.Entity = fakeBombCarrierController(testBombPawnHandle + 1)
		p.processFrameGameEvents()

		assert.Same(t, pl, p.gameState.bomb.Carrier)

		setOwner(invalid)

		assert.Equal(t, []events.BombDropped{{Player: pl, EntityID: testBombC4ID}}, drops)
	})

	t.Run("bomb destroyed while its owner was unresolved", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var dispatched int
		p.RegisterEventHandler(func(events.BombPickup) { dispatched++ })
		p.RegisterEventHandler(func(events.BombDropped) { dispatched++ })

		_, destroyOldBomb := createFakeC4(p, serverClasses, testBombPawnHandle)
		addFakeBombCarrier(p)
		destroyOldBomb()
		p.processFrameGameEvents()

		setOwner, _ := createFakeC4(p, serverClasses, invalid)
		setOwner(invalid)

		assert.Nil(t, p.gameState.bomb.Carrier)
		assert.Zero(t, dispatched)
	})

	// the sendtables parser re-creates an entity at the same index with the same serial without destroying the old one
	// (e.g. when it re-enters the PVS on POV demos), so the old entity's pending resolve must not fire
	t.Run("bomb re-created while its owner was unresolved", func(t *testing.T) {
		p, serverClasses := newBombTestParser()

		var pickups []events.BombPickup
		p.RegisterEventHandler(func(e events.BombPickup) { pickups = append(pickups, e) })

		createFakeC4(p, serverClasses, testBombPawnHandle)
		createFakeC4(p, serverClasses, testBombPawnHandle)
		pl := addFakeBombCarrier(p)
		p.processFrameGameEvents()

		assert.Same(t, pl, p.gameState.bomb.Carrier)
		assert.Equal(t, []events.BombPickup{{Player: pl}}, pickups)
	})
}

func TestBindBomb_BombPickup_RepeatedOwnerUpdate(t *testing.T) {
	p, serverClasses := newBombTestParser()
	pl := addFakeBombCarrier(p)

	var pickups []events.BombPickup
	p.RegisterEventHandler(func(e events.BombPickup) { pickups = append(pickups, e) })

	setOwner, _ := createFakeC4(p, serverClasses, uint64(constants.InvalidEntityHandleSource2))
	setOwner(testBombPawnHandle)
	setOwner(testBombPawnHandle)

	assert.Equal(t, []events.BombPickup{{Player: pl}}, pickups)

	// picking the bomb up again after dropping it is a new pickup
	setOwner(uint64(constants.InvalidEntityHandleSource2))
	setOwner(testBombPawnHandle)

	assert.Len(t, pickups, 2)
}

func newBombGameEventTestParser() *parser {
	p := newParser()
	p.header.Filestamp = filestampS2
	p.disableMimicSource1GameEvents = true

	return p
}

// bomb_dropped has the C4's entity-ID in entindex (CS:GO and CS2).
func TestBombDropped_EntityIDFromEntindex(t *testing.T) {
	p := newBombGameEventTestParser()

	var got []events.BombDropped
	p.RegisterEventHandler(func(e events.BombDropped) {
		got = append(got, e)
	})

	p.gameEventHandler.bombDropped(map[string]*msg.CSVCMsg_GameEventKeyT{
		"userid":   {ValShort: proto.Int32(3)},
		"entindex": {ValLong: proto.Int32(586)},
	})

	assert.Len(t, got, 1)
	assert.Equal(t, 586, got[0].EntityID)
}

// bomb_dropped falls back to userid_pawn if userid doesn't resolve.
func TestBombDropped_PlayerFromUserIDPawn(t *testing.T) {
	p := newBombGameEventTestParser()
	pl := addFakeBombCarrier(p)

	var got []events.BombDropped
	p.RegisterEventHandler(func(e events.BombDropped) {
		got = append(got, e)
	})

	p.gameEventHandler.bombDropped(map[string]*msg.CSVCMsg_GameEventKeyT{
		"userid_pawn": {ValLong: proto.Int32(int32(testBombPawnHandle))},
		"entindex":    {ValLong: proto.Int32(586)},
	})

	assert.Equal(t, []events.BombDropped{{Player: pl, EntityID: 586}}, got)
}

// CS2's bomb_pickup has no userid, only userid_pawn.
func TestBombPickup_PlayerFromUserIDPawn(t *testing.T) {
	p := newBombGameEventTestParser()
	pl := addFakeBombCarrier(p)

	var got []events.BombPickup
	p.RegisterEventHandler(func(e events.BombPickup) {
		got = append(got, e)
	})

	p.gameEventHandler.bombPickup(map[string]*msg.CSVCMsg_GameEventKeyT{
		"userid_pawn": {ValLong: proto.Int32(int32(testBombPawnHandle))},
	})

	assert.Equal(t, []events.BombPickup{{Player: pl}}, got)
}
