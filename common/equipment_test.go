package common

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func TestEquipmentElementClass(t *testing.T) {
	assert.Equal(t, EqClassUnknown, EqUnknown.Class(), "EqUnknown should have the class EqClassUnknown")
	assert.Equal(t, EqClassPistols, EqP2000.Class(), "EqP2000 should have the class EqClassPistols")
	assert.Equal(t, EqClassPistols, EqRevolver.Class(), "EqRevolver should have the class EqClassPistols")
	assert.Equal(t, EqClassRifle, EqG3SG1.Class(), "EqG3SG1 should have the class EqClassRifle")
}

func TestEquipmentClass(t *testing.T) {
	assert.Equal(t, EqClassUnknown, NewEquipment(EqUnknown).Class(), "EqUnknown should have the class EqClassUnknown")
	assert.Equal(t, EqClassPistols, NewEquipment(EqP2000).Class(), "EqP2000 should have the class EqClassPistols")
	assert.Equal(t, EqClassPistols, NewEquipment(EqRevolver).Class(), "EqRevolver should have the class EqClassPistols")
	assert.Equal(t, EqClassRifle, NewEquipment(EqG3SG1).Class(), "EqG3SG1 should have the class EqClassRifle")
}

func TestMapEquipment(t *testing.T) {
	assert.Equal(t, EqKnife, MapEquipment("weapon_bayonet"), "'weapon_bayonet' should be mapped to EqKnife")
	assert.Equal(t, EqKnife, MapEquipment("weapon_knife_butterfly"), "'weapon_knife_butterfly' should be mapped to EqKnife")
	assert.Equal(t, EqM4A4, MapEquipment("weapon_m4a1"), "'weapon_m4a1' should be mapped to EqM4A4") // This is correct, weapon_m4a1 == M4A4
	assert.Equal(t, EqM4A1, MapEquipment("weapon_m4a1_silencer"), "'weapon_m4a1_silencer' should be mapped to EqM4A1")
	assert.Equal(t, EqUnknown, MapEquipment("asdf"), "'asdf' should be mapped to EqUnknown")
}

func TestEquipmentUniqueID(t *testing.T) {
	assert.NotEqual(t, NewEquipment(EqAK47).UniqueID(), NewEquipment(EqAK47).UniqueID(), "UniqueIDs of different equipment instances should be different")
}

func TestEquipmentName(t *testing.T) {
	assert.Equal(t, "Dual Barettas", EqDualBarettas.String(), "EqDualBarettas should be named correctly")
}
