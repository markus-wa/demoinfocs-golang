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
	testProjectileID        = 200

	// item definition indexes, also used as the weapons' & projectiles' model
	testItemFlash      = uint64(43)
	testItemMolotov    = uint64(46)
	testItemIncendiary = uint64(48)
)

type thrownGrenadeTest struct {
	p *Parser
	// the player the pawn belongs to - may be swapped (bot takeover)
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

	pawn := newTestPawn(p, testThrowerPawnID)
	pl := newTestPlayer(p, testThrowerControllerID, testThrowerPawnID, 1)

	gt := &thrownGrenadeTest{p: p, pl: pl, pawn: pawn, weapons: make(map[int]*st.Entity)}

	p.bindPlayerWeapons(pawn, func() *common.Player { return gt.pl })

	for id, item := range map[int]uint64{
		testFlashA:     testItemFlash,
		testFlashB:     testItemFlash,
		testMolotov:    testItemMolotov,
		testIncendiary: testItemIncendiary,
	} {
		gt.weapons[id] = gt.newWeapon(id, item, testThrowerPawnID)
	}

	return gt
}

// newTestPawn returns a pawn entity with an empty inventory, registered with the parser's entities.
func newTestPawn(p *Parser, id int) *st.Entity {
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

	pawn := stfake.NewEntity(id, props...)
	p.gameState.entities[id] = pawn

	return pawn
}

// newTestPlayer returns a player with a controller entity that has the given pawn, registered by user ID.
func newTestPlayer(p *Parser, controllerID, pawnID, userID int) *common.Player {
	pl := common.NewPlayer(p.demoInfoProvider)
	pl.Entity = stfake.NewEntity(controllerID,
		stfake.P("m_hPawn", uint64(pawnID)),
		stfake.P("m_hPlayerPawn", uint64(pawnID)),
		stfake.P("m_pInGameMoneyServices.m_iAccount", int32(0)), // read when a weapon's owner is set
	)
	p.gameState.playersByUserID[userID] = pl

	return pl
}

// newWeapon returns a bound grenade weapon entity owned by the given pawn.
func (gt *thrownGrenadeTest) newWeapon(id int, item uint64, pawnID int) *st.Entity {
	wep := stfake.NewEntity(id,
		stfake.P("m_iItemDefinitionIndex", item),
		stfake.P("CBodyComponent.m_hModel", item),
		stfake.P("m_hOwnerEntity", uint64(pawnID)),
		stfake.P("m_fLastShotTime", nil),
		stfake.P("m_iClip1", nil),
		stfake.P("m_bPinPulled", false),
		stfake.P("m_fThrowTime", float32(0)),
	)
	gt.p.bindWeapon(wep)

	return wep
}

func (gt *thrownGrenadeTest) setSlot(slot int, handle uint64) {
	stfake.Update(gt.pawn, stfake.P(fmt.Sprintf("%s.%04d", playerWeaponPrefixS2, slot), handle))
}

// setInventorySize updates the m_hMyWeapons array (not its elements) to the given size.
func (gt *thrownGrenadeTest) setInventorySize(size int) {
	stfake.Update(gt.pawn, stfake.P(playerWeaponPrefixS2, make([]any, size)))
}

// pullPin is what happens to the weapon when the player starts the throw (holds the attack button).
func (gt *thrownGrenadeTest) pullPin(id int) {
	stfake.Update(gt.weapons[id], stfake.P("m_bPinPulled", true))
}

// release is what happens to the weapon when the player releases the throw, 6-15 ticks before the projectile.
func (gt *thrownGrenadeTest) release(id int) {
	stfake.Update(gt.weapons[id], stfake.P("m_bPinPulled", false), stfake.P("m_fThrowTime", float32(12.5)))
}

// resetThrowTime is what happens to the weapon when the projectile is created and grenades of the type are left.
func (gt *thrownGrenadeTest) resetThrowTime(id int) {
	stfake.Update(gt.weapons[id], stfake.P("m_fThrowTime", float32(0)))
}

// throw creates a flashbang projectile thrown by the player and returns its weapon instance's entity.
func (gt *thrownGrenadeTest) throw(t *testing.T) *st.Entity {
	t.Helper()

	return gt.throwItem(t, testItemFlash)
}

// throwItem is like throw, for the grenade with the given item definition index.
func (gt *thrownGrenadeTest) throwItem(t *testing.T, item uint64) *st.Entity {
	t.Helper()

	wep := gt.throwFrom(t, testProjectileID, testThrowerPawnID, item)
	assert.Same(t, gt.pl, wep.Owner)

	return wep.Entity
}

// throwFrom creates the projectile entity of the grenade with the given item definition index,
// thrown by the given pawn, and returns its weapon instance.
func (gt *thrownGrenadeTest) throwFrom(t *testing.T, projectileID, pawnID int, item uint64) *common.Equipment {
	t.Helper()

	proj := stfake.NewEntity(projectileID,
		stfake.P("m_hOwnerEntity", uint64(pawnID)),
		stfake.P("CBodyComponent.m_hModel", item),
		stfake.P("m_hThrower", uint64(pawnID)),
	)
	gt.p.bindGrenadeProjectiles(proj)
	stfake.Create(proj)

	wep := gt.p.gameState.grenadeProjectiles[projectileID].WeaponInstance
	require.NotNil(t, wep)
	assert.True(t, isSameGrenadeType(common.EquipmentIndexMapping[item], wep.Type), "type %v", wep.Type)

	return wep
}

// Throws of flashbang A (in slot 3), with the inventory update on the tick the projectile is created.
func TestThrownGrenadeWeapon_Flashbang(t *testing.T) {
	cases := []struct {
		name string
		// prime primes flashbang A's throw, on tick 4 (nil: no priming is seen)
		prime func(gt *thrownGrenadeTest)
		// update is the inventory update on the tick the projectile is created
		update func(gt *thrownGrenadeTest)
		tick   int // the tick the projectile is created on
		want   int // entity ID of the projectile's weapon
		// pickedUp is whether flashbang B must be in the inventory after the throw
		pickedUp bool
	}{
		{
			// the thrown one leaves the inventory and is destroyed, the picked up one is the only flashbang left
			name:  "last one thrown & one picked up on the same tick",
			prime: (*thrownGrenadeTest).releaseA,
			update: func(gt *thrownGrenadeTest) {
				gt.setSlot(3, testFlashB)
				stfake.Destroy(gt.weapons[testFlashA])
			},
			tick:     10,
			want:     testFlashA, // the projectile is the destroyed flashbang's
			pickedUp: true,
		},
		{
			name:   "last one thrown & one picked up the tick before",
			prime:  (*thrownGrenadeTest).releaseA,
			update: func(gt *thrownGrenadeTest) { gt.setSlot(3, testFlashB) },
			tick:   11,
			want:   testFlashA,
		},
		{
			// the last grenade of the type is thrown and nothing is picked up (#580)
			name:   "last of type",
			prime:  (*thrownGrenadeTest).releaseA,
			update: func(gt *thrownGrenadeTest) { gt.setSlot(3, constants.InvalidEntityHandleSource2) },
			tick:   10,
			want:   testFlashA,
		},
		{
			// the projectile is created more than a frame after the release, e.g. on demos with 3 ticks per frame
			name:   "last of type, removed & projectile created several ticks after the release",
			prime:  (*thrownGrenadeTest).releaseA,
			update: func(gt *thrownGrenadeTest) { gt.setSlot(3, constants.InvalidEntityHandleSource2) },
			tick:   4 + 15,
			want:   testFlashA,
		},
		{
			// the thrown one stays in the inventory and its throw time is reset before the projectile is created
			name:   "more of the type left",
			prime:  (*thrownGrenadeTest).releaseA,
			update: func(gt *thrownGrenadeTest) { gt.resetThrowTime(testFlashA) },
			tick:   10,
			want:   testFlashA,
		},
		{
			// the flashbang is removed by the inventory array shrinking rather than its slot being updated
			name:   "last of type, removed by the array shrinking",
			prime:  (*thrownGrenadeTest).releaseA,
			update: func(gt *thrownGrenadeTest) { gt.setInventorySize(3) },
			tick:   10,
			want:   testFlashA,
		},
		{
			// POV demos may not have the release but only the pin pull, and destroy the weapon with the projectile
			name:  "pin pulled, no release seen, destroyed on the throw tick",
			prime: (*thrownGrenadeTest).pullPinA,
			update: func(gt *thrownGrenadeTest) {
				gt.setSlot(3, constants.InvalidEntityHandleSource2)
				stfake.Destroy(gt.weapons[testFlashA])
			},
			tick: 4 + 21,
			want: testFlashA,
		},
		{
			// the pin may be held for any length of time, and the release may not be networked
			name:  "pin pulled long before, no release seen, destroyed on the throw tick",
			prime: (*thrownGrenadeTest).pullPinA,
			update: func(gt *thrownGrenadeTest) {
				gt.setSlot(3, constants.InvalidEntityHandleSource2)
				stfake.Destroy(gt.weapons[testFlashA])
			},
			tick: 4 + 2000,
			want: testFlashA,
		},
		{
			name:   "no priming seen",
			update: func(gt *thrownGrenadeTest) {},
			tick:   10,
			want:   testFlashA, // the inventory's
		},
		{
			// the weapon's updates weren't networked and it left the inventory with the throw (#580)
			name:   "no priming seen, last of type",
			update: func(gt *thrownGrenadeTest) { gt.setSlot(3, constants.InvalidEntityHandleSource2) },
			tick:   10,
			want:   testFlashA, // the last-known one
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gt := newThrownGrenadeTest(t)
			gt.setSlot(3, testFlashA)

			if tc.prime != nil {
				gt.p.gameState.ingameTick = 4
				tc.prime(gt)
			}

			gt.p.gameState.ingameTick = tc.tick
			tc.update(gt)

			assert.Same(t, gt.weapons[tc.want], gt.throw(t))

			if tc.pickedUp {
				assert.Same(t, gt.weapons[testFlashB], gt.pl.Inventory[testFlashB].Entity)
			}
		})
	}
}

func (gt *thrownGrenadeTest) releaseA() { gt.release(testFlashA) }
func (gt *thrownGrenadeTest) pullPinA() { gt.pullPin(testFlashA) }

// The projectile is created before the inventory update in the same packet.
func TestThrownGrenadeWeapon_ProjectileBeforeInventoryUpdate(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 4
	gt.release(testFlashA)

	gt.p.gameState.ingameTick = 10
	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))

	gt.setSlot(3, testFlashB)
	assert.Same(t, gt.weapons[testFlashB], gt.pl.Inventory[testFlashB].Entity)
}

// A throw that doesn't change the inventory (more of the type left) keeps the inventory weapon's OriginalString.
func TestThrownGrenadeWeapon_InInventory(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)
	gt.pl.Inventory[testFlashA].OriginalString = "weapon_flashbang"

	gt.p.gameState.ingameTick = 4
	gt.release(testFlashA)

	gt.p.gameState.ingameTick = 10
	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))
	assert.Equal(t, "weapon_flashbang", gt.pl.Inventory[testFlashA].OriginalString, "must not be overwritten")
	assert.Empty(t, gt.p.gameState.grenadeProjectiles[testProjectileID].WeaponInstance.OriginalString)
}

// A grenade primed long before that was destroyed or dropped since isn't the thrown one.
func TestThrownGrenadeWeapon_PrimedWeaponGone(t *testing.T) {
	cases := []struct {
		name string
		gone func(gt *thrownGrenadeTest) // on tick 10
	}{
		{
			name: "destroyed",
			gone: func(gt *thrownGrenadeTest) { stfake.Destroy(gt.weapons[testFlashA]) },
		},
		{
			name: "dropped",
			gone: func(gt *thrownGrenadeTest) {
				stfake.Update(gt.weapons[testFlashA], stfake.P("m_hOwnerEntity", uint64(constants.InvalidEntityHandleSource2)))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gt := newThrownGrenadeTest(t)
			gt.setSlot(3, testFlashA)

			gt.p.gameState.ingameTick = 4
			gt.release(testFlashA)

			gt.p.gameState.ingameTick = 10
			gt.setSlot(3, testFlashB) // the pickup replaces A in its slot
			tc.gone(gt)

			// still the thrown one on the tick it went (last of type: destroyed with the projectile)
			assert.Same(t, gt.weapons[testFlashA], gt.throw(t))

			gt.p.gameState.ingameTick = 11
			assert.Same(t, gt.weapons[testFlashB], gt.throwFrom(t, 201, testThrowerPawnID, testItemFlash).Entity)
		})
	}
}

// The player primed one flashbang, cancelled, and then threw the other one.
func TestThrownGrenadeWeapon_NewerPrimingReplacesOlder(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)
	gt.setSlot(4, testFlashB)

	gt.p.gameState.ingameTick = 4
	gt.pullPin(testFlashA)
	stfake.Update(gt.weapons[testFlashA], stfake.P("m_bPinPulled", false)) // cancelled, no throw time

	gt.p.gameState.ingameTick = 8
	gt.release(testFlashB)

	gt.p.gameState.ingameTick = 14
	assert.Same(t, gt.weapons[testFlashB], gt.throw(t))
}

// POV demos re-create the same projectile entity on the ticks after the throw: every creation is the thrown weapon's.
func TestThrownGrenadeWeapon_ProjectileRecreated(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 4
	gt.release(testFlashA)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, constants.InvalidEntityHandleSource2)
	stfake.Destroy(gt.weapons[testFlashA])

	for tick := 10; tick <= 12; tick++ {
		gt.p.gameState.ingameTick = tick
		assert.Same(t, gt.weapons[testFlashA], gt.throw(t), "tick %d", tick)
	}
}

// Molotovs and incendiaries may be reported as each other, the primed one is the thrown one.
func TestThrownGrenadeWeapon_FireGrenades(t *testing.T) {
	cases := []struct {
		name   string
		slots  map[int]uint64 // entity IDs by slot
		primed []int          // released in this order, on ticks 4, 5, ...
		thrown uint64         // item index of the projectile
		want   int            // entity ID of the projectile's weapon
	}{
		{
			name:   "molotov projectile, incendiary released",
			slots:  map[int]uint64{3: testIncendiary},
			primed: []int{testIncendiary},
			thrown: testItemMolotov,
			want:   testIncendiary,
		},
		{
			name:   "incendiary projectile, molotov released",
			slots:  map[int]uint64{3: testMolotov},
			primed: []int{testMolotov},
			thrown: testItemIncendiary,
			want:   testMolotov,
		},
		{
			// the exact type is preferred over the newer alternative
			name:   "both released, molotov projectile",
			slots:  map[int]uint64{3: testMolotov, 4: testIncendiary},
			primed: []int{testMolotov, testIncendiary},
			thrown: testItemMolotov,
			want:   testMolotov,
		},
		{
			name:   "both released, incendiary projectile",
			slots:  map[int]uint64{3: testIncendiary, 4: testMolotov},
			primed: []int{testIncendiary, testMolotov},
			thrown: testItemIncendiary,
			want:   testIncendiary,
		},
		{
			// no priming: the inventory's
			name:   "incendiary projectile, incendiary in the inventory",
			slots:  map[int]uint64{3: testIncendiary},
			thrown: testItemIncendiary,
			want:   testIncendiary,
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

				for i, id := range tc.primed {
					gt.p.gameState.ingameTick = 4 + i
					gt.release(id)
				}

				gt.p.gameState.ingameTick = 10

				if len(tc.primed) > 0 {
					gt.setInventorySize(3) // the thrown one leaves the inventory
				}

				assert.Same(t, gt.weapons[tc.want], gt.throwItem(t, tc.thrown))
			}
		})
	}
}

// Demos without the priming properties on grenade weapons fall back to the inventory (and don't abort parsing).
func TestThrownGrenadeWeapon_WithoutPrimingProperties(t *testing.T) {
	gt := newThrownGrenadeTest(t)

	const id = 104

	wep := stfake.NewEntity(id,
		stfake.P("m_iItemDefinitionIndex", testItemFlash),
		stfake.P("CBodyComponent.m_hModel", testItemFlash),
		stfake.P("m_hOwnerEntity", uint64(testThrowerPawnID)),
		stfake.P("m_fLastShotTime", nil),
		stfake.P("m_iClip1", nil),
	)
	gt.p.bindWeapon(wep)
	gt.setSlot(3, id)

	gt.p.gameState.ingameTick = 10
	assert.Same(t, wep, gt.throw(t))
}

// A weapon first seen mid-throw (mid-demo start, PVS re-entry) primes its throw with the initial value.
func TestThrownGrenadeWeapon_InitialThrowTime(t *testing.T) {
	gt := newThrownGrenadeTest(t)

	const id = 104

	wep := stfake.NewEntity(id,
		stfake.P("m_iItemDefinitionIndex", testItemFlash),
		stfake.P("CBodyComponent.m_hModel", testItemFlash),
		stfake.P("m_hOwnerEntity", uint64(testThrowerPawnID)),
		stfake.P("m_fLastShotTime", nil),
		stfake.P("m_iClip1", nil),
		stfake.P("m_bPinPulled", false),
		stfake.P("m_fThrowTime", float32(7.5)),
	)

	gt.p.gameState.ingameTick = 4
	gt.p.bindWeapon(wep)
	stfake.Create(wep) // fires the update handlers with the initial values

	gt.p.gameState.ingameTick = 10
	assert.Same(t, wep, gt.throw(t))
}

// Priming is per pawn.
func TestThrownGrenadeWeapon_TwoPawns(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	const (
		pawn2ID       = 3
		controller2ID = 4
		flash2ID      = 104
	)

	newTestPawn(gt.p, pawn2ID)
	pl2 := newTestPlayer(gt.p, controller2ID, pawn2ID, 2)
	flash2 := gt.newWeapon(flash2ID, testItemFlash, pawn2ID)

	gt.p.gameState.ingameTick = 4
	gt.release(testFlashA)
	stfake.Update(flash2, stfake.P("m_fThrowTime", float32(12.5)))

	gt.p.gameState.ingameTick = 10

	wep2 := gt.throwFrom(t, 201, pawn2ID, testItemFlash)
	assert.Same(t, flash2, wep2.Entity)
	assert.Same(t, pl2, wep2.Owner)

	assert.Same(t, gt.weapons[testFlashA], gt.throw(t))
}

// A bot taken over in place keeps its pawn: the new occupant's throws are attributed without a handover.
func TestThrownGrenadeWeapon_InPlaceTakeover(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashA)

	gt.p.gameState.ingameTick = 4
	gt.release(testFlashA)

	previous := gt.pl
	gt.pl = newTestPlayer(gt.p, testThrowerControllerID, testThrowerPawnID, 1)
	gt.p.takeOverInPlace(previous, gt.pl, gt.pl.Entity)

	gt.p.gameState.ingameTick = 10
	gt.setSlot(3, constants.InvalidEntityHandleSource2)

	assert.Same(t, gt.weapons[testFlashA], gt.throw(t)) // asserts Owner == gt.pl
	assert.Empty(t, previous.Inventory)
}

// With several of a type in the inventory, the active one is chosen, otherwise the one with the lowest entity ID.
func TestGetPlayerWeapon_Deterministic(t *testing.T) {
	gt := newThrownGrenadeTest(t)
	gt.setSlot(3, testFlashB)
	gt.setSlot(4, testFlashA)
	gt.setSlot(5, testMolotov)

	for range 20 {
		assert.Same(t, gt.weapons[testFlashA], getPlayerWeapon(gt.pl, common.EqFlash, "", 0).Entity)
		assert.Same(t, gt.weapons[testFlashA], gt.p.gameState.lastKnownGrenadeWeapon(gt.pl, common.EqFlash).Entity)
	}

	stfake.Set(gt.pawn, stfake.P("m_pWeaponServices.m_hActiveWeapon", uint64(testFlashB)))

	for range 20 {
		assert.Same(t, gt.weapons[testFlashB], getPlayerWeapon(gt.pl, common.EqFlash, "", 0).Entity)
	}

	// a single match is returned regardless of the active weapon
	assert.Same(t, gt.weapons[testMolotov], getPlayerWeapon(gt.pl, common.EqMolotov, "", 0).Entity)
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
