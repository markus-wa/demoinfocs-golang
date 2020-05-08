package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
)

func TestEquipmentElement_Class(t *testing.T) {
	assert.Equal(t, EqClassUnknown, EqUnknown.Class(), "EqUnknown should have the class EqClassUnknown")
	assert.Equal(t, EqClassPistols, EqP2000.Class(), "EqP2000 should have the class EqClassPistols")
	assert.Equal(t, EqClassPistols, EqRevolver.Class(), "EqRevolver should have the class EqClassPistols")
	assert.Equal(t, EqClassRifle, EqG3SG1.Class(), "EqG3SG1 should have the class EqClassRifle")
}

func TestEquipmentElement_Name(t *testing.T) {
	assert.Equal(t, "Dual Berettas", EqDualBerettas.String(), "EqDualBerettas should be named correctly")
}

func TestMapEquipment(t *testing.T) {
	assert.Equal(t, EqKnife, MapEquipment("weapon_bayonet"), "'weapon_bayonet' should be mapped to EqKnife")
	assert.Equal(t, EqKnife, MapEquipment("weapon_knife_butterfly"), "'weapon_knife_butterfly' should be mapped to EqKnife")
	assert.Equal(t, EqM4A4, MapEquipment("weapon_m4a1"), "'weapon_m4a1' should be mapped to EqM4A4") // This is correct, weapon_m4a1 == M4A4
	assert.Equal(t, EqM4A1, MapEquipment("weapon_m4a1_silencer"), "'weapon_m4a1_silencer' should be mapped to EqM4A1")
	assert.Equal(t, EqUnknown, MapEquipment("asdf"), "'asdf' should be mapped to EqUnknown")
}

func TestEquipment_Class(t *testing.T) {
	assert.Equal(t, EqClassUnknown, NewEquipment(EqUnknown).Class(), "EqUnknown should have the class EqClassUnknown")
	assert.Equal(t, EqClassPistols, NewEquipment(EqP2000).Class(), "EqP2000 should have the class EqClassPistols")
	assert.Equal(t, EqClassPistols, NewEquipment(EqRevolver).Class(), "EqRevolver should have the class EqClassPistols")
	assert.Equal(t, EqClassRifle, NewEquipment(EqG3SG1).Class(), "EqG3SG1 should have the class EqClassRifle")
}

func TestEquipment_UniqueID(t *testing.T) {
	assert.NotEqual(t, NewEquipment(EqAK47).UniqueID(), NewEquipment(EqAK47).UniqueID(), "UniqueIDs of different equipment instances should be different")
}

func TestEquipment_AmmoInMagazine(t *testing.T) {
	wep := &Equipment{
		Type:   EqAK47,
		Entity: entityWithProperty("m_iClip1", st.PropertyValue{IntVal: 31}),
	}

	// returned value should be minus 1, m_iClip1 is always 1 more than the actual number of bullets
	assert.Equal(t, 30, wep.AmmoInMagazine())
}

func TestEquipment_AmmoInMagazine_NotFound(t *testing.T) {
	entity := entityWithID(1)
	entity.On("PropertyValue", "m_iClip1").Return(st.PropertyValue{}, false)

	wep := &Equipment{
		Type:   EqAK47,
		Entity: entity,
	}

	// returned value should be minus 1, m_iClip1 is always 1 more than the actual number of bullets
	assert.Equal(t, -1, wep.AmmoInMagazine())
}

func TestEquipment_AmmoInMagazine_Grenade(t *testing.T) {
	wep := &Equipment{
		Type: EqFlash,
	}

	assert.Equal(t, 1, wep.AmmoInMagazine())
}

func TestEquipment_AmmoInMagazine_EntityNil(t *testing.T) {
	wep := &Equipment{
		Type: EqAK47,
	}

	assert.Equal(t, 0, wep.AmmoInMagazine())
}

func TestEquipment_AmmoReserve(t *testing.T) {
	entity := entityWithProperty("m_iPrimaryReserveAmmoCount", st.PropertyValue{IntVal: 60})
	wep := &Equipment{
		Type:   EqAK47,
		Entity: entity,
	}

	assert.Equal(t, 60, wep.AmmoReserve())
}

func TestEquipment_AmmoReserve_Grenade(t *testing.T) {
	owner := new(Player)
	owner.AmmoLeft[1] = 2

	entity := entityWithProperty("LocalWeaponData.m_iPrimaryAmmoType", st.PropertyValue{IntVal: 1})
	wep := &Equipment{
		Type:   EqFlash,
		Owner:  owner,
		Entity: entity,
	}

	assert.Equal(t, 1, wep.AmmoReserve())
}

func TestEquipment_AmmoReserve_Grenade_OwnerNil(t *testing.T) {
	wep := &Equipment{
		Type: EqFlash,
	}

	assert.Equal(t, 0, wep.AmmoReserve())
}

func TestEquipment_AmmoReserve_EntityNil(t *testing.T) {
	wep := &Equipment{
		Type: EqAK47,
	}

	assert.Equal(t, 0, wep.AmmoReserve())
}

func TestEquipment_ZoomLevel(t *testing.T) {
	wep := &Equipment{
		Type:   EqAK47,
		Entity: entityWithProperty("m_zoomLevel", st.PropertyValue{IntVal: 2}),
	}

	assert.Equal(t, ZoomFull, wep.ZoomLevel())
}

func TestEquipment_ZoomLevel_EntityNil(t *testing.T) {
	wep := &Equipment{
		Type: EqAK47,
	}

	assert.Equal(t, ZoomLevel(0), wep.ZoomLevel())
}

func TestEquipmentAlternative(t *testing.T) {
	assert.Equal(t, EqUSP, EquipmentAlternative(EqP2000))
	assert.Equal(t, EqCZ, EquipmentAlternative(EqP250))
	assert.Equal(t, EqCZ, EquipmentAlternative(EqFiveSeven))
	assert.Equal(t, EqCZ, EquipmentAlternative(EqTec9))
	assert.Equal(t, EqRevolver, EquipmentAlternative(EqDeagle))
	assert.Equal(t, EqMP5, EquipmentAlternative(EqMP7))
	assert.Equal(t, EqM4A1, EquipmentAlternative(EqM4A4))
}
