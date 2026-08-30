package common

import (
	"fmt"
	"strings"
)

// KnifeType identifies the concrete knife model of a piece of equipment whose
// Type is EqKnife. It is KnifeTypeUnknown for all other equipment.
//
// As Valve can add new knives over time, the known names and item definition
// indexes are exposed through the mutable maps KnifeTypes and
// KnifeTypeIndexMapping so new knives can be registered without upgrading the
// library.
type KnifeType int

// All currently known knife types.
const (
	KnifeTypeUnknown       KnifeType = 0
	KnifeTypeDefault       KnifeType = 1  // weapon_knife / weapon_knife_t
	KnifeTypeBayonet       KnifeType = 2  // weapon_bayonet
	KnifeTypeClassic       KnifeType = 3  // weapon_knife_css
	KnifeTypeFlip          KnifeType = 4  // weapon_knife_flip
	KnifeTypeGut           KnifeType = 5  // weapon_knife_gut
	KnifeTypeKarambit      KnifeType = 6  // weapon_knife_karambit
	KnifeTypeM9Bayonet     KnifeType = 7  // weapon_knife_m9_bayonet
	KnifeTypeHuntsman      KnifeType = 8  // weapon_knife_tactical
	KnifeTypeFalchion      KnifeType = 9  // weapon_knife_falchion
	KnifeTypeBowie         KnifeType = 10 // weapon_knife_survival_bowie
	KnifeTypeButterfly     KnifeType = 11 // weapon_knife_butterfly
	KnifeTypeShadowDaggers KnifeType = 12 // weapon_knife_push
	KnifeTypeParacord      KnifeType = 13 // weapon_knife_cord
	KnifeTypeSurvival      KnifeType = 14 // weapon_knife_canis
	KnifeTypeUrsus         KnifeType = 15 // weapon_knife_ursus
	KnifeTypeNavaja        KnifeType = 16 // weapon_knife_gypsy_jackknife
	KnifeTypeNomad         KnifeType = 17 // weapon_knife_outdoor
	KnifeTypeStiletto      KnifeType = 18 // weapon_knife_stiletto
	KnifeTypeTalon         KnifeType = 19 // weapon_knife_widowmaker
	KnifeTypeSkeleton      KnifeType = 20 // weapon_knife_skeleton
	KnifeTypeKukri         KnifeType = 21 // weapon_knife_kukri
)

// KnifeTypes maps the names of knives as they appear in game events and
// MapEquipment (e.g. "knife_karambit") to their KnifeType.
//
// This map is mutable so new knives can be registered without upgrading the
// library. The "weapon_" prefix, if present, is stripped before lookup.
var KnifeTypes = map[string]KnifeType{
	"knife":                 KnifeTypeDefault,
	"knife_t":               KnifeTypeDefault,
	"bayonet":               KnifeTypeBayonet,
	"knife_css":             KnifeTypeClassic,
	"knife_flip":            KnifeTypeFlip,
	"knife_gut":             KnifeTypeGut,
	"knife_karambit":        KnifeTypeKarambit,
	"knife_m9_bayonet":      KnifeTypeM9Bayonet,
	"knife_tactical":        KnifeTypeHuntsman,
	"knife_falchion":        KnifeTypeFalchion,
	"knife_survival_bowie":  KnifeTypeBowie,
	"knife_butterfly":       KnifeTypeButterfly,
	"knife_push":            KnifeTypeShadowDaggers,
	"knife_cord":            KnifeTypeParacord,
	"knife_canis":           KnifeTypeSurvival,
	"knife_ursus":           KnifeTypeUrsus,
	"knife_gypsy_jackknife": KnifeTypeNavaja,
	"knife_outdoor":         KnifeTypeNomad,
	"knife_stiletto":        KnifeTypeStiletto,
	"knife_widowmaker":      KnifeTypeTalon,
	"knife_skeleton":        KnifeTypeSkeleton,
	"knife_kukri":           KnifeTypeKukri,
}

// KnifeTypeIndexMapping maps m_iItemDefinitionIndex values of knives to their
// KnifeType.
//
// This map is mutable so new knives can be registered without upgrading the
// library. The novelty knifegg (41) and ghost (80) knives are intentionally
// not mapped and resolve to KnifeTypeUnknown.
var KnifeTypeIndexMapping = map[uint64]KnifeType{
	42:  KnifeTypeDefault,
	59:  KnifeTypeDefault,
	500: KnifeTypeBayonet,
	503: KnifeTypeClassic,
	505: KnifeTypeFlip,
	506: KnifeTypeGut,
	507: KnifeTypeKarambit,
	508: KnifeTypeM9Bayonet,
	509: KnifeTypeHuntsman,
	512: KnifeTypeFalchion,
	514: KnifeTypeBowie,
	515: KnifeTypeButterfly,
	516: KnifeTypeShadowDaggers,
	517: KnifeTypeParacord,
	518: KnifeTypeSurvival,
	519: KnifeTypeUrsus,
	520: KnifeTypeNavaja,
	521: KnifeTypeNomad,
	522: KnifeTypeStiletto,
	523: KnifeTypeTalon,
	525: KnifeTypeSkeleton,
	526: KnifeTypeKukri,
}

// KnifeTypeNames contains the display names of all known knife types.
//
// This map is mutable so display names can be registered for custom knife
// types added to KnifeTypes / KnifeTypeIndexMapping.
var KnifeTypeNames = map[KnifeType]string{
	KnifeTypeDefault:       "Default Knife",
	KnifeTypeBayonet:       "Bayonet",
	KnifeTypeClassic:       "Classic Knife",
	KnifeTypeFlip:          "Flip Knife",
	KnifeTypeGut:           "Gut Knife",
	KnifeTypeKarambit:      "Karambit",
	KnifeTypeM9Bayonet:     "M9 Bayonet",
	KnifeTypeHuntsman:      "Huntsman Knife",
	KnifeTypeFalchion:      "Falchion Knife",
	KnifeTypeBowie:         "Bowie Knife",
	KnifeTypeButterfly:     "Butterfly Knife",
	KnifeTypeShadowDaggers: "Shadow Daggers",
	KnifeTypeParacord:      "Paracord Knife",
	KnifeTypeSurvival:      "Survival Knife",
	KnifeTypeUrsus:         "Ursus Knife",
	KnifeTypeNavaja:        "Navaja Knife",
	KnifeTypeNomad:         "Nomad Knife",
	KnifeTypeStiletto:      "Stiletto Knife",
	KnifeTypeTalon:         "Talon Knife",
	KnifeTypeSkeleton:      "Skeleton Knife",
	KnifeTypeKukri:         "Kukri Knife",
}

// String returns the human readable name of the knife type.
// E.g. 'Karambit', 'Shadow Daggers', 'Kukri Knife' etc.
func (k KnifeType) String() string {
	if name, ok := KnifeTypeNames[k]; ok {
		return name
	}

	return fmt.Sprintf("KnifeType(%d)", k)
}

// MapKnifeType creates a KnifeType from the name of the knife.
// The "weapon_" prefix, if present, is ignored, so both "knife_karambit" and
// "weapon_knife_karambit" map to KnifeTypeKarambit.
// Returns KnifeTypeUnknown for names that don't map to a known knife type.
func MapKnifeType(eqName string) KnifeType {
	return KnifeTypes[strings.TrimPrefix(eqName, weaponPrefix)]
}
