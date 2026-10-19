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
	testFlashA              = 100 // serial 0, so this is also the handle
	testFlashB              = 101
	testMolotov             = 102
	testIncendiary          = 103

	// item definition indexes, also used as the weapons' & projectiles' model
	testItemFlash      = uint64(43)
	testItemMolotov    = uint64(46)
	testItemIncendiary = uint64(48)
)

type thrownGrenadeTest struct {
	p    *Parser
	pl   *common.Player
	pawn *st.Entity
	// grenade weapon entities by entity ID
	weapons map[int]*st.Entity
}

// newThrownGrenadeTest returns a parser with one player whose pawn has the flashbangs testFlashA & testFlashB,
// the molotov testMolotov and the incendiary testIncendiary (not in the inventory yet).
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

	gt := &thrownGrenadeTest{p: p, pl: pl, pawn: pawn, weapons: make(map[int]*st.Entity)}

	for id, item := range map[int]uint64{
		testFlashA:     testItemFlash,
		testFlashB:     testItemFlash,
		testMolotov:    testItemMolotov,
		testIncendiary: testItemIncendiary,
	} {
		wep := stfake.NewEntity(id,
			stfake.P("m_iItemDefinitionIndex", item),
			stfake.P("CBodyComponent.m_hModel", item),
			stfake.P("m_hOwnerEntity", uint64(testThrowerPawnID)),
			stfake.P("m_fLastShotTime", nil),
			stfake.P("m_iClip1", nil),
		)
		p.bindWeapon(wep)
		gt.weapons[id] = wep
	}

	return gt
}

func (gt *thrownGrenadeTest) setSlot(slot int, handle uint64) {
	stfake.Update(gt.pawn, stfake.P(fmt.Sprintf("%s.%04d", playerWeaponPrefixS2, slot), handle))
}

// setInventorySize updates the m_hMyWeapons array (not its elements) to the given size.
func (gt *thrownGrenadeTest) setInventorySize(size int) {
	stfake.Update(gt.pawn, stfake.P(playerWeaponPrefixS2, make([]any, size)))
}

// throw creates a flashbang projectile thrown by the player and returns its weapon instance's entity.
func (gt *thrownGrenadeTest) throw(t *testing.T) *st.Entity {
	t.Helper()

	return gt.throwItem(t, testItemFlash)
}

// throwItem is like throw, for the grenade with the given item definition index.
func (gt *thrownGrenadeTest) throwItem(t *testing.T, item uint64) *st.Entity {
	t.Helper()

	proj := stfake.NewEntity(200,
		stfake.P("m_hOwnerEntity", uint64(testThrowerPawnID)),
		stfake.P("CBodyComponent.m_hModel", item),
		stfake.P("m_hThrower", uint64(testThrowerPawnID)),
	)
	gt.p.bindGrenadeProjectiles(proj)
	stfake.Create(proj)

	wep := gt.p.gameState.grenadeProjectiles[200].WeaponInstance
	require.NotNil(t, wep)
	assert.True(t, isSameGrenadeType(common.EquipmentIndexMapping[item], wep.Type), "type %v", wep.Type)
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
	stfake.Destroy(gt.weapons[testFlashA])

	assert.Same(t, gt.weapons[testFlashA], gt.throw(t), "the projectile is the removed flashbang's")
	assert.Same(t, gt.weapons[testFlashB], gt.pl.Inventory[testFlashB].Entity)
}

// Same, but the projectile is created on the next tick.
func TestThrownGrenadeWeapon_PickedUpTickBeforeProjectile(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, testFlashB)

	gt.p.gameState.ingameTick = 11
	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))
}

// A grenade that left the inventory long before isn't the thrown one.
func TestThrownGrenadeWeapon_OldRemovalIgnored(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, testFlashB)

	gt.p.gameState.ingameTick = 100
	assert.Same(t, gt.weapons[testFlashB], gt.throw(t))
}

// The projectile is created before the inventory update in the same packet.
func TestThrownGrenadeWeapon_ProjectileBeforeInventoryUpdate(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))

	gt.setSlot(3, testFlashB)
	assert.Same(t, gt.weapons[testFlashB], gt.pl.Inventory[testFlashB].Entity)
}

// A throw that doesn't change the inventory (more of the type left).
func TestThrownGrenadeWeapon_InInventory(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)
	gt.pl.Inventory[testFlashA].OriginalString = "weapon_flashbang"

	gt.p.gameState.ingameTick = 10
	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))
	assert.Equal(t, "weapon_flashbang", gt.pl.Inventory[testFlashA].OriginalString, "must not be overwritten")
}

// The last grenade of the type is thrown and nothing is picked up (#580).
func TestThrownGrenadeWeapon_LastOfType(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, constants.InvalidEntityHandleSource2)
	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))
}

// Also with the removal more than a tick before the projectile is created.
func TestThrownGrenadeWeapon_LastOfTypeRemovedEarlier(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, constants.InvalidEntityHandleSource2)

	gt.p.gameState.ingameTick = 20
	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))
}

// With several of a type in the inventory, the active one is chosen, otherwise the one with the lowest entity ID.
func TestGetPlayerWeapon_Deterministic(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashB)
	gt.setSlot(4, testFlashA)

	for range 20 {
		assert.Same(t, gt.weapons[testFlashA], getPlayerWeapon(gt.pl, common.EqFlash, "", 0).Entity)
		assert.Same(t, gt.weapons[testFlashA], gt.p.gameState.lastKnownGrenadeWeapon(gt.pl, common.EqFlash).Entity)
	}

	stfake.Set(gt.pawn, stfake.P("m_pWeaponServices.m_hActiveWeapon", uint64(testFlashB)))

	for range 20 {
		assert.Same(t, gt.weapons[testFlashB], getPlayerWeapon(gt.pl, common.EqFlash, "", 0).Entity)
	}
}

// Throws while grenades just left the inventory (all on the same tick).
func TestThrownGrenadeWeapon_RemovedWeapons(t *testing.T) {
	cases := []struct {
		name  string
		slots map[int]uint64 // entity IDs by slot, the inventory before the update
		// update removes (some of) the slots' weapons
		update func(gt *thrownGrenadeTest)
		thrown uint64 // item index of the projectile
		want   int    // entity ID of the projectile's weapon
	}{
		{
			name:   "molotov projectile, incendiary removed",
			slots:  map[int]uint64{3: testIncendiary},
			update: func(gt *thrownGrenadeTest) { gt.setSlot(3, testFlashB) },
			thrown: testItemMolotov,
			want:   testIncendiary,
		},
		{
			name:   "incendiary projectile, molotov removed",
			slots:  map[int]uint64{3: testMolotov},
			update: func(gt *thrownGrenadeTest) { gt.setSlot(3, testFlashB) },
			thrown: testItemIncendiary,
			want:   testMolotov,
		},
		{
			name:   "molotov & incendiary removed, molotov projectile",
			slots:  map[int]uint64{3: testMolotov, 4: testIncendiary},
			update: func(gt *thrownGrenadeTest) { gt.setInventorySize(3) },
			thrown: testItemMolotov,
			want:   testMolotov,
		},
		{
			name:   "molotov & incendiary removed, incendiary projectile",
			slots:  map[int]uint64{3: testMolotov, 4: testIncendiary},
			update: func(gt *thrownGrenadeTest) { gt.setInventorySize(3) },
			thrown: testItemIncendiary,
			want:   testIncendiary,
		},
		{
			name:   "two flashbangs removed at once (array shrinks)",
			slots:  map[int]uint64{3: testFlashB, 4: testFlashA},
			update: func(gt *thrownGrenadeTest) { gt.setInventorySize(3) },
			thrown: testItemFlash,
			want:   testFlashA, // the lowest entity ID, like getPlayerWeapon() would choose
		},
		{
			name:   "flashbang removed by the array shrinking, other flashbang picked up",
			slots:  map[int]uint64{4: testFlashA},
			update: func(gt *thrownGrenadeTest) { gt.setSlot(3, testFlashB); gt.setInventorySize(4) },
			thrown: testItemFlash,
			want:   testFlashA,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// several times, so map iteration order dependencies show
			for range 10 {
				gt := newThrownGrenadeTest(t)
				for slot, entityID := range tc.slots {
					gt.setSlot(slot, entityID)
				}

				gt.p.gameState.ingameTick = 10
				tc.update(gt)

				assert.Same(t, gt.weapons[tc.want], gt.throwItem(t, tc.thrown))
			}
		})
	}
}

// Without the active weapon property, the lowest entity ID is chosen (and parsing isn't aborted).
func TestGetPlayerWeapon_SeveralWithoutActiveWeaponProperty(t *testing.T) {
	p := newParser()

	pawn := stfake.NewEntity(testThrowerPawnID)
	p.gameState.entities[testThrowerPawnID] = pawn

	pl := common.NewPlayer(p.demoInfoProvider)
	pl.Entity = stfake.NewEntity(testThrowerControllerID,
		stfake.P("m_hPawn", uint64(testThrowerPawnID)),
		stfake.P("m_hPlayerPawn", uint64(testThrowerPawnID)),
	)

	a := common.NewEquipment(common.EqFlash, 1)
	b := common.NewEquipment(common.EqFlash, 2)
	pl.Inventory = map[int]*common.Equipment{testFlashA: a, testFlashB: b}

	assert.Same(t, a, getPlayerWeapon(pl, common.EqFlash, "", 0))

	p.gameState.updateLastKnownGrenadeWeapons(pl)
	assert.Same(t, a, p.gameState.lastKnownGrenadeWeapon(pl, common.EqFlash))
}
