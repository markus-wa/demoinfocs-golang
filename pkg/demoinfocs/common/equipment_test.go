package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
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
	assert.Equal(t, EqKevlar, MapEquipment("weapon_vest"), "'weapon_vest' should be mapped to EqKevlar")
	assert.Equal(t, EqHelmet, MapEquipment("weapon_vesthelm"), "'weapon_vesthelm' should be mapped to EqHelmet")
	assert.Equal(t, EqUnknown, MapEquipment("asdf"), "'asdf' should be mapped to EqUnknown")
}

func TestWeirdCS2Values(t *testing.T) {
	assert.Equal(t, EqDeagle, MapEquipment("deagle_vip"), "'deagle_vip' should be mapped to EqDeagle")
	assert.Equal(t, EqUSP, MapEquipment("usp_silencer_vip"), "'usp_silencer_vip' should be mapped to EqUSP")
	assert.Equal(t, EqUSP, MapEquipment("5e_vip_usp_silencer"), "'5e_vip_usp_silencer' should be mapped to EqUSP")
	assert.Equal(t, EqAK47, MapEquipment("5e_vip_ak47"), "'5e_vip_ak47' should be mapped to EqAK47")
	assert.Equal(t, EqDeagle, MapEquipment("5e_default_deagle"), "'5e_default_deagle' should be mapped to EqDeagle")
}

func TestEquipment_Class(t *testing.T) {
	assert.Equal(t, EqClassUnknown, NewEquipment(EqUnknown).Class(), "EqUnknown should have the class EqClassUnknown")
	assert.Equal(t, EqClassPistols, NewEquipment(EqP2000).Class(), "EqP2000 should have the class EqClassPistols")
	assert.Equal(t, EqClassPistols, NewEquipment(EqRevolver).Class(), "EqRevolver should have the class EqClassPistols")
	assert.Equal(t, EqClassRifle, NewEquipment(EqG3SG1).Class(), "EqG3SG1 should have the class EqClassRifle")
}

func TestEquipment_UniqueID(t *testing.T) {
	assert.NotEqual(t, NewEquipment(EqAK47).UniqueID2(), NewEquipment(EqAK47).UniqueID2(), "UniqueIDs of different equipment instances should be different")
}

func TestEquipment_AmmoInMagazine(t *testing.T) {
	wep := &Equipment{
		Type:   EqAK47,
		Entity: entityWithProperty("m_iClip1", st.PropertyValue{Any: uint32(31)}),
	}

	// returned value should be minus 1, m_iClip1 is always 1 more than the actual number of bullets
	assert.Equal(t, 30, wep.AmmoInMagazine())
}

func TestEquipment_AmmoInMagazine_NotFound(t *testing.T) {
	entity := entityWithID(1)
	entity.On("PropertyValue", "m_iClip1").Return(st.PropertyValue{
		Any: uint32(0),
	}, false)

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
	entity := entityWithProperties([]fakeProp{
		{
			propName: "m_pReserveAmmo.0000",
			value:    st.PropertyValue{},
			isNil:    true,
		},
		{
			propName: "m_iPrimaryReserveAmmoCount",
			value:    st.PropertyValue{Any: int32(60)},
			isNil:    false,
		},
	})
	wep := &Equipment{
		Type:   EqAK47,
		Entity: entity,
	}

	assert.Equal(t, 60, wep.AmmoReserve())
}

func TestEquipment_AmmoReserve_Grenade(t *testing.T) {
	owner := new(Player)

	entity := entityWithProperties([]fakeProp{
		{
			propName: "m_pReserveAmmo.0001",
			value:    st.PropertyValue{},
			isNil:    true,
		},
		{
			propName: "m_iPrimaryReserveAmmoCount",
			value:    st.PropertyValue{Any: int32(1)},
			isNil:    false,
		},
	})
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
		Entity: entityWithProperty("m_zoomLevel", st.PropertyValue{Any: int32(2)}),
	}

	assert.Equal(t, ZoomFull, wep.ZoomLevel())
}

func TestEquipment_ZoomLevel_EntityNil(t *testing.T) {
	wep := &Equipment{
		Type: EqAK47,
	}

	assert.Equal(t, ZoomLevel(0), wep.ZoomLevel())
}

func TestEquipment_Not_Silenced(t *testing.T) {
	wep := &Equipment{
		Type:   EqAK47,
		Entity: entityWithProperty("m_bSilencerOn", st.PropertyValue{Any: false}),
	}

	assert.Equal(t, false, wep.Silenced())
}

func TestEquipment_Silenced_On_Off(t *testing.T) {
	wep := &Equipment{
		Type:   EqUSP,
		Entity: entityWithProperty("m_bSilencerOn", st.PropertyValue{Any: true}),
	}
	assert.Equal(t, true, wep.Silenced(), "Weapon should be silenced after the property value has been set to 1.")

	wep.Entity = entityWithProperty("m_bSilencerOn", st.PropertyValue{Any: false})
	assert.Equal(t, false, wep.Silenced(), "Weapon should not be silenced after the property value has been set to 0.")
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
