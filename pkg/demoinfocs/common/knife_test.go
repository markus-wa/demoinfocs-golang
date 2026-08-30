package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapKnifeType(t *testing.T) {
	assert.Equal(t, KnifeTypeKarambit, MapKnifeType("knife_karambit"), "'knife_karambit' should map to KnifeTypeKarambit")
	assert.Equal(t, KnifeTypeBayonet, MapKnifeType("bayonet"), "'bayonet' should map to KnifeTypeBayonet")
	assert.Equal(t, KnifeTypeBayonet, MapKnifeType("weapon_bayonet"), "'weapon_bayonet' should map to KnifeTypeBayonet")
	assert.Equal(t, KnifeTypeTalon, MapKnifeType("knife_widowmaker"), "'knife_widowmaker' should map to KnifeTypeTalon")
	assert.Equal(t, KnifeTypeDefault, MapKnifeType("knife"), "'knife' should map to KnifeTypeDefault")
	assert.Equal(t, KnifeTypeUnknown, MapKnifeType("knife_unknown_future_knife"), "unknown knife names should map to KnifeTypeUnknown")
	assert.Equal(t, KnifeTypeUnknown, MapKnifeType("ak47"), "non-knife names should map to KnifeTypeUnknown")
}

func TestKnifeTypeString(t *testing.T) {
	assert.Equal(t, "Shadow Daggers", KnifeTypeShadowDaggers.String(), "KnifeTypeShadowDaggers should be named correctly")
	assert.Equal(t, "Classic Knife", KnifeTypeClassic.String(), "KnifeTypeClassic should be named correctly")
	assert.Equal(t, "Kukri Knife", KnifeTypeKukri.String(), "KnifeTypeKukri should be named correctly")
	assert.Equal(t, "KnifeType(0)", KnifeTypeUnknown.String(), "KnifeTypeUnknown should have a fallback name")
	assert.Equal(t, "KnifeType(999)", KnifeType(999).String(), "unknown knife types should have a fallback name")
}

func TestKnifeTypeNamesExtension(t *testing.T) {
	custom := KnifeType(500)
	KnifeTypeNames[custom] = "Custom Knife"

	assert.Equal(t, "Custom Knife", custom.String(), "custom knife types should be nameable via KnifeTypeNames")
}

func TestKnifeTypeIndexMappingCompleteness(t *testing.T) {
	// Every knife in EquipmentIndexMapping (index >= 500, excluding the novelty
	// knifegg and knife_ghost items) should have a KnifeTypeIndexMapping entry
	// so knife details can always be resolved from the item definition index.
	for itemIndex, eqType := range EquipmentIndexMapping {
		if eqType != EqKnife {
			continue
		}

		if itemIndex < 500 {
			continue
		}

		knifeType, ok := KnifeTypeIndexMapping[itemIndex]
		assert.True(t, ok, "index %d should have a KnifeTypeIndexMapping entry", itemIndex)
		assert.NotEqual(t, KnifeTypeUnknown, knifeType, "index %d should map a known knife type", itemIndex)
	}
}

func TestKnifeTypesValues(t *testing.T) {
	for name, knifeType := range KnifeTypes {
		assert.NotEqual(t, KnifeTypeUnknown, knifeType, "name %q should map to a known knife type", name)
	}
}
