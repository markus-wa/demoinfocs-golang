package demoinfocs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

const (
	testThrowerPawnID       = 1
	testThrowerControllerID = 2
	testFlashModel          = uint64(7)
	testFlashA              = 100 // serial 0, so this is also the handle
	testFlashB              = 101
)

type thrownGrenadeTest struct {
	p    *Parser
	pl   *common.Player
	pawn *st.Entity
	// flashbang weapon entities by entity ID
	flashes map[int]*st.Entity
}

// newThrownGrenadeTest returns a parser with one player whose pawn has the flashbangs testFlashA & testFlashB
// (not in the inventory yet).
func newThrownGrenadeTest(t *testing.T) *thrownGrenadeTest {
	t.Helper()

	p := newParser()

	const slots = 64

	props := make([]stfake.Prop, 0, 3+slots)
	props = append(props,
		stfake.P(playerWeaponPrefixS2, nil),
		stfake.P("m_pWeaponServices.m_hActiveWeapon", uint64(constants.InvalidEntityHandleSource2)),
		stfake.P("m_bInBuyZone", false), // read when a weapon is destroyed
	)

	for i := range slots {
		props = append(props, stfake.P(fmt.Sprintf("%s.%04d", playerWeaponPrefixS2, i), nil))
	}

	pawn := stfake.NewEntity(testThrowerPawnID, props...)
	p.gameState.entities[testThrowerPawnID] = pawn

	pl := common.NewPlayer(p.demoInfoProvider)
	pl.Entity = stfake.NewEntity(testThrowerControllerID,
		stfake.P("m_hPawn", uint64(testThrowerPawnID)),
		stfake.P("m_hPlayerPawn", uint64(testThrowerPawnID)),
	)
	p.gameState.playersByUserID[1] = pl
	p.bindPlayerWeapons(pawn, func() *common.Player { return pl })

	gt := &thrownGrenadeTest{p: p, pl: pl, pawn: pawn, flashes: make(map[int]*st.Entity)}

	for _, id := range []int{testFlashA, testFlashB} {
		wep := stfake.NewEntity(id,
			stfake.P("m_iItemDefinitionIndex", uint64(43)), // flashbang
			stfake.P("CBodyComponent.m_hModel", testFlashModel),
			stfake.P("m_hOwnerEntity", uint64(testThrowerPawnID)),
			stfake.P("m_fLastShotTime", nil),
			stfake.P("m_iClip1", nil),
		)
		p.bindWeapon(wep)
		gt.flashes[id] = wep
	}

	return gt
}

func (gt *thrownGrenadeTest) setSlot(slot int, handle uint64) {
	stfake.Update(gt.pawn, stfake.P(fmt.Sprintf("%s.%04d", playerWeaponPrefixS2, slot), handle))
}

// throw creates a flashbang projectile thrown by the player and returns its weapon instance's entity.
func (gt *thrownGrenadeTest) throw(t *testing.T) *st.Entity {
	t.Helper()

	proj := stfake.NewEntity(200,
		stfake.P("m_hOwnerEntity", uint64(testThrowerPawnID)),
		stfake.P("CBodyComponent.m_hModel", testFlashModel),
		stfake.P("m_hThrower", uint64(testThrowerPawnID)),
	)
	gt.p.bindGrenadeProjectiles(proj)
	stfake.Create(proj)

	wep := gt.p.gameState.grenadeProjectiles[200].WeaponInstance
	require.NotNil(t, wep)
	assert.Equal(t, common.EqFlash, wep.Type)
	assert.Same(t, gt.pl, wep.Owner)

	return wep.Entity
}

// A player throws their last flashbang and picks one up from the floor on the same tick:
// the thrown one leaves the inventory and the picked up one is the only flashbang left.
func TestThrownGrenadeWeapon_PickedUpOnThrowTick(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, testFlashB)
	stfake.Destroy(gt.flashes[testFlashA])

	assert.Same(t, gt.flashes[testFlashA], gt.throw(t), "the projectile is the removed flashbang's")
	assert.Same(t, gt.flashes[testFlashB], gt.pl.Inventory[testFlashB].Entity)
}

// Same, but the projectile is created on the next tick.
func TestThrownGrenadeWeapon_PickedUpTickBeforeProjectile(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, testFlashB)

	gt.p.gameState.ingameTick = 11
	assert.Same(t, gt.flashes[testFlashA], gt.throw(t))
}

// A grenade that left the inventory long before isn't the thrown one.
func TestThrownGrenadeWeapon_OldRemovalIgnored(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, testFlashB)

	gt.p.gameState.ingameTick = 100
	assert.Same(t, gt.flashes[testFlashB], gt.throw(t))
}

// The projectile is created before the inventory update in the same packet.
func TestThrownGrenadeWeapon_ProjectileBeforeInventoryUpdate(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	assert.Same(t, gt.flashes[testFlashA], gt.throw(t))

	gt.setSlot(3, testFlashB)
	assert.Same(t, gt.flashes[testFlashB], gt.pl.Inventory[testFlashB].Entity)
}

// A throw that doesn't change the inventory (more of the type left).
func TestThrownGrenadeWeapon_InInventory(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)
	gt.pl.Inventory[testFlashA].OriginalString = "weapon_flashbang"

	gt.p.gameState.ingameTick = 10
	assert.Same(t, gt.flashes[testFlashA], gt.throw(t))
	assert.Equal(t, "weapon_flashbang", gt.pl.Inventory[testFlashA].OriginalString, "must not be overwritten")
}

// The last grenade of the type is thrown and nothing is picked up (#580).
func TestThrownGrenadeWeapon_LastOfType(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, constants.InvalidEntityHandleSource2)
	assert.Same(t, gt.flashes[testFlashA], gt.throw(t))
}

// Also with the removal more than a tick before the projectile is created.
func TestThrownGrenadeWeapon_LastOfTypeRemovedEarlier(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, constants.InvalidEntityHandleSource2)

	gt.p.gameState.ingameTick = 20
	assert.Same(t, gt.flashes[testFlashA], gt.throw(t))
}

// With several of a type in the inventory, the active one is chosen, otherwise the one with the lowest entity ID.
func TestGetPlayerWeapon_Deterministic(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashB)
	gt.setSlot(4, testFlashA)

	for range 20 {
		assert.Same(t, gt.flashes[testFlashA], getPlayerWeapon(gt.pl, common.EqFlash, "", 0).Entity)
		assert.Same(t, gt.flashes[testFlashA], gt.p.gameState.lastKnownGrenadeWeapon(gt.pl, common.EqFlash).Entity)
	}

	stfake.Set(gt.pawn, stfake.P("m_pWeaponServices.m_hActiveWeapon", uint64(testFlashB)))

	for range 20 {
		assert.Same(t, gt.flashes[testFlashB], getPlayerWeapon(gt.pl, common.EqFlash, "", 0).Entity)
	}
}
