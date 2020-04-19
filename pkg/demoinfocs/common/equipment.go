package common

import (
	"math/rand"
	"strings"

	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
)

// EquipmentClass is the type for the various EqClassXYZ constants.
type EquipmentClass int

// EquipmentClass constants give information about the type of an equipment (SMG, Rifle, Grenade etc.).
//
// Note: (EquipmentType+99) / 100 = EquipmentClass
const (
	EqClassUnknown   EquipmentClass = 0
	EqClassPistols   EquipmentClass = 1
	EqClassSMG       EquipmentClass = 2
	EqClassHeavy     EquipmentClass = 3
	EqClassRifle     EquipmentClass = 4
	EqClassEquipment EquipmentClass = 5
	EqClassGrenade   EquipmentClass = 6
)

// EquipmentType is the type for the various EqXYZ constants.
type EquipmentType int

// Class returns the class of the equipment.
// E.g. pistol, smg, heavy etc.
func (e EquipmentType) Class() EquipmentClass {
	const classDenominator = 100

	return EquipmentClass((int(e) + classDenominator - 1) / classDenominator)
}

// String returns a human readable name for the equipment.
// E.g. 'AK-47', 'UMP-45', 'Smoke Grenade' etc.
func (e EquipmentType) String() string {
	return eqElementToName[e]
}

// EquipmentType constants give information about what weapon a player has equipped.
const (
	EqUnknown EquipmentType = 0

	// Pistols

	EqP2000        EquipmentType = 1
	EqGlock        EquipmentType = 2
	EqP250         EquipmentType = 3
	EqDeagle       EquipmentType = 4
	EqFiveSeven    EquipmentType = 5
	EqDualBerettas EquipmentType = 6
	EqTec9         EquipmentType = 7
	EqCZ           EquipmentType = 8
	EqUSP          EquipmentType = 9
	EqRevolver     EquipmentType = 10

	// SMGs

	EqMP7   EquipmentType = 101
	EqMP9   EquipmentType = 102
	EqBizon EquipmentType = 103
	EqMac10 EquipmentType = 104
	EqUMP   EquipmentType = 105
	EqP90   EquipmentType = 106
	EqMP5   EquipmentType = 107

	// Heavy

	EqSawedOff EquipmentType = 201
	EqNova     EquipmentType = 202
	EqMag7     EquipmentType = 203 // You should consider using EqSwag7 instead
	EqSwag7    EquipmentType = 203
	EqXM1014   EquipmentType = 204
	EqM249     EquipmentType = 205
	EqNegev    EquipmentType = 206

	// Rifles

	EqGalil  EquipmentType = 301
	EqFamas  EquipmentType = 302
	EqAK47   EquipmentType = 303
	EqM4A4   EquipmentType = 304
	EqM4A1   EquipmentType = 305
	EqScout  EquipmentType = 306
	EqSSG08  EquipmentType = 306
	EqSG556  EquipmentType = 307
	EqSG553  EquipmentType = 307
	EqAUG    EquipmentType = 308
	EqAWP    EquipmentType = 309
	EqScar20 EquipmentType = 310
	EqG3SG1  EquipmentType = 311

	// Equipment

	EqZeus      EquipmentType = 401
	EqKevlar    EquipmentType = 402
	EqHelmet    EquipmentType = 403
	EqBomb      EquipmentType = 404
	EqKnife     EquipmentType = 405
	EqDefuseKit EquipmentType = 406
	EqWorld     EquipmentType = 407

	// Grenades

	EqDecoy      EquipmentType = 501
	EqMolotov    EquipmentType = 502
	EqIncendiary EquipmentType = 503
	EqFlash      EquipmentType = 504
	EqSmoke      EquipmentType = 505
	EqHE         EquipmentType = 506
)

var eqNameToWeapon map[string]EquipmentType

var eqElementToName map[EquipmentType]string

func init() {
	initEqNameToWeapon()
	initEqElementToName()
}

//nolint:funlen
func initEqNameToWeapon() {
	eqNameToWeapon = make(map[string]EquipmentType)

	eqNameToWeapon["ak47"] = EqAK47
	eqNameToWeapon["aug"] = EqAUG
	eqNameToWeapon["awp"] = EqAWP
	eqNameToWeapon["bizon"] = EqBizon
	eqNameToWeapon["c4"] = EqBomb
	eqNameToWeapon["deagle"] = EqDeagle
	eqNameToWeapon["decoy"] = EqDecoy
	eqNameToWeapon["decoygrenade"] = EqDecoy
	eqNameToWeapon["decoyprojectile"] = EqDecoy
	eqNameToWeapon["decoy_projectile"] = EqDecoy
	eqNameToWeapon["elite"] = EqDualBerettas
	eqNameToWeapon["famas"] = EqFamas
	eqNameToWeapon["fiveseven"] = EqFiveSeven
	eqNameToWeapon["flashbang"] = EqFlash
	eqNameToWeapon["g3sg1"] = EqG3SG1
	eqNameToWeapon["galil"] = EqGalil
	eqNameToWeapon["galilar"] = EqGalil
	eqNameToWeapon["glock"] = EqGlock
	eqNameToWeapon["hegrenade"] = EqHE
	eqNameToWeapon["hkp2000"] = EqP2000
	eqNameToWeapon["incgrenade"] = EqIncendiary
	eqNameToWeapon["incendiarygrenade"] = EqIncendiary
	eqNameToWeapon["m249"] = EqM249
	eqNameToWeapon["m4a1"] = EqM4A4
	eqNameToWeapon["mac10"] = EqMac10
	eqNameToWeapon["mag7"] = EqSwag7
	eqNameToWeapon["molotov"] = EqMolotov
	eqNameToWeapon["molotovgrenade"] = EqMolotov
	eqNameToWeapon["molotovprojectile"] = EqMolotov
	eqNameToWeapon["molotov_projectile"] = EqMolotov
	eqNameToWeapon["mp7"] = EqMP7
	eqNameToWeapon["mp5sd"] = EqMP5
	eqNameToWeapon["mp9"] = EqMP9
	eqNameToWeapon["negev"] = EqNegev
	eqNameToWeapon["nova"] = EqNova
	eqNameToWeapon["p250"] = EqP250
	eqNameToWeapon["p90"] = EqP90
	eqNameToWeapon["sawedoff"] = EqSawedOff
	eqNameToWeapon["scar20"] = EqScar20
	eqNameToWeapon["sg556"] = EqSG556
	eqNameToWeapon["smokegrenade"] = EqSmoke
	eqNameToWeapon["smokegrenadeprojectile"] = EqSmoke
	eqNameToWeapon["smokegrenade_projectile"] = EqSmoke
	eqNameToWeapon["ssg08"] = EqScout
	eqNameToWeapon["taser"] = EqZeus
	eqNameToWeapon["tec9"] = EqTec9
	eqNameToWeapon["ump45"] = EqUMP
	eqNameToWeapon["xm1014"] = EqXM1014
	eqNameToWeapon["m4a1_silencer"] = EqM4A1
	eqNameToWeapon["m4a1_silencer_off"] = EqM4A1
	eqNameToWeapon["cz75a"] = EqCZ
	eqNameToWeapon["usp"] = EqUSP
	eqNameToWeapon["usp_silencer"] = EqUSP
	eqNameToWeapon["usp_silencer_off"] = EqUSP
	eqNameToWeapon["world"] = EqWorld
	eqNameToWeapon["inferno"] = EqIncendiary
	eqNameToWeapon["revolver"] = EqRevolver
	eqNameToWeapon["vest"] = EqKevlar
	eqNameToWeapon["vesthelm"] = EqHelmet
	eqNameToWeapon["defuser"] = EqDefuseKit

	// These don't exist and / or used to crash the game with the give command
	eqNameToWeapon["scar17"] = EqUnknown
	eqNameToWeapon["sensorgrenade"] = EqUnknown
	eqNameToWeapon["mp5navy"] = EqUnknown
	eqNameToWeapon["p228"] = EqUnknown
	eqNameToWeapon["scout"] = EqUnknown
	eqNameToWeapon["sg550"] = EqUnknown
	eqNameToWeapon["sg552"] = EqUnknown // This one still crashes the game :)
	eqNameToWeapon["tmp"] = EqUnknown
	eqNameToWeapon["worldspawn"] = EqWorld
}

//nolint:funlen
func initEqElementToName() {
	eqElementToName = make(map[EquipmentType]string)
	eqElementToName[EqAK47] = "AK-47"
	eqElementToName[EqAUG] = "AUG"
	eqElementToName[EqAWP] = "AWP"
	eqElementToName[EqBizon] = "PP-Bizon"
	eqElementToName[EqBomb] = "C4"
	eqElementToName[EqDeagle] = "Desert Eagle"
	eqElementToName[EqDecoy] = "Decoy Grenade"
	eqElementToName[EqDualBerettas] = "Dual Berettas"
	eqElementToName[EqFamas] = "FAMAS"
	eqElementToName[EqFiveSeven] = "Five-SeveN"
	eqElementToName[EqFlash] = "Flashbang"
	eqElementToName[EqG3SG1] = "G3SG1"
	eqElementToName[EqGalil] = "Galil AR"
	eqElementToName[EqGlock] = "Glock-18"
	eqElementToName[EqHE] = "HE Grenade"
	eqElementToName[EqP2000] = "P2000"
	eqElementToName[EqIncendiary] = "Incendiary Grenade"
	eqElementToName[EqM249] = "M249"
	eqElementToName[EqM4A4] = "M4A4"
	eqElementToName[EqMac10] = "MAC-10"
	eqElementToName[EqSwag7] = "MAG-7"
	eqElementToName[EqMolotov] = "Molotov"
	eqElementToName[EqMP7] = "MP7"
	eqElementToName[EqMP5] = "MP5-SD"
	eqElementToName[EqMP9] = "MP9"
	eqElementToName[EqNegev] = "Negev"
	eqElementToName[EqNova] = "Nova"
	eqElementToName[EqP250] = "p250"
	eqElementToName[EqP90] = "P90"
	eqElementToName[EqSawedOff] = "Sawed-Off"
	eqElementToName[EqScar20] = "SCAR-20"
	eqElementToName[EqSG553] = "SG 553"
	eqElementToName[EqSmoke] = "Smoke Grenade"
	eqElementToName[EqScout] = "SSG 08"
	eqElementToName[EqZeus] = "Zeus x27"
	eqElementToName[EqTec9] = "Tec-9"
	eqElementToName[EqUMP] = "UMP-45"
	eqElementToName[EqXM1014] = "XM1014"
	eqElementToName[EqM4A1] = "M4A1"
	eqElementToName[EqCZ] = "CZ75 Auto"
	eqElementToName[EqUSP] = "USP-S"
	eqElementToName[EqWorld] = "World"
	eqElementToName[EqRevolver] = "R8 Revolver"
	eqElementToName[EqKevlar] = "Kevlar Vest"
	eqElementToName[EqHelmet] = "Kevlar + Helmet"
	eqElementToName[EqDefuseKit] = "Defuse Kit"
	eqElementToName[EqKnife] = "Knife"
	eqElementToName[EqUnknown] = "UNKNOWN"
}

const weaponPrefix = "weapon_"

// MapEquipment creates an EquipmentType from the name of the weapon / equipment.
// Returns EqUnknown if no mapping can be found.
func MapEquipment(eqName string) EquipmentType {
	eqName = strings.TrimPrefix(eqName, weaponPrefix)

	var wep EquipmentType
	if strings.Contains(eqName, "knife") || strings.Contains(eqName, "bayonet") {
		wep = EqKnife
	} else {
		// If the eqName isn't known it will be EqUnknown as that is the default value for EquipmentType
		wep = eqNameToWeapon[eqName]
	}

	return wep
}

type ZoomLevel int

const (
	ZoomNone ZoomLevel = 0
	ZoomHalf ZoomLevel = 1
	ZoomFull ZoomLevel = 2
)

// Equipment is a weapon / piece of equipment belonging to a player.
// This also includes the skin and some additional data.
type Equipment struct {
	Type           EquipmentType // The type of weapon which the equipment instantiates.
	Entity         st.Entity     // The game entity instance
	Owner          *Player       // The player carrying the equipment, not necessarily the buyer.
	OriginalString string        // E.g. 'models/weapons/w_rif_m4a1_s.mdl'. Used internally to differentiate alternative weapons (M4A4 / M4A1-S etc.).

	uniqueID int64
}

// String returns a human readable name for the equipment.
// E.g. 'AK-47', 'UMP-45', 'Smoke Grenade' etc.
func (e *Equipment) String() string {
	return e.Type.String()
}

// Class returns the class of the equipment.
// E.g. pistol, smg, heavy etc.
func (e *Equipment) Class() EquipmentClass {
	return e.Type.Class()
}

// UniqueID returns the unique id of the equipment element.
// The unique id is a random int generated internally by this library and can be used to differentiate
// equipment from each other. This is needed because demo-files reuse entity ids.
func (e *Equipment) UniqueID() int64 {
	return e.uniqueID
}

// AmmoInMagazine returns the ammo left in the magazine.
// Returns CWeaponCSBase.m_iClip1 for most weapons and 1 for grenades.
func (e *Equipment) AmmoInMagazine() int {
	if e.Class() == EqClassGrenade {
		return 1
	}

	if e.Entity == nil {
		return 0
	}

	val, ok := e.Entity.PropertyValue("m_iClip1")
	if !ok {
		return -1
	}

	// need to subtract 1 as m_iClip1 is nrOfBullets + 1
	return val.IntVal - 1
}

// AmmoType returns the weapon's ammo type, mostly (only?) relevant for grenades.
func (e *Equipment) AmmoType() int {
	return getInt(e.Entity, "LocalWeaponData.m_iPrimaryAmmoType")
}

// ZoomLevel returns how far the player has zoomed in on the weapon.
// Only weapons with scopes have a valid zoom level.
func (e *Equipment) ZoomLevel() ZoomLevel {
	if e.Entity == nil {
		return 0
	}

	// if the property doesn't exist we return 0 by default
	val, _ := e.Entity.PropertyValue("m_zoomLevel")

	return ZoomLevel(val.IntVal)
}

// AmmoReserve returns the ammo left available for reloading.
// Returns CWeaponCSBase.m_iPrimaryReserveAmmoCount for most weapons and 'Owner.AmmoLeft[AmmoType] - 1' for grenades.
// Use AmmoInMagazine() + AmmoReserve() to quickly get the amount of grenades a player owns.
func (e *Equipment) AmmoReserve() int {
	if e.Class() == EqClassGrenade {
		if e.Owner != nil {
			// minus one for 'InMagazine'
			return e.Owner.AmmoLeft[e.AmmoType()] - 1
		}

		return 0
	}

	if e.Entity == nil {
		return 0
	}

	// if the property doesn't exist we return 0 by default
	val, _ := e.Entity.PropertyValue("m_iPrimaryReserveAmmoCount")

	return val.IntVal
}

// NewEquipment creates a new Equipment and sets the UniqueID.
//
// Intended for internal use only.
func NewEquipment(wep EquipmentType) *Equipment {
	return &Equipment{Type: wep, uniqueID: rand.Int63()}
}

var equipmentToAlternative = map[EquipmentType]EquipmentType{
	EqP2000:     EqUSP,
	EqP250:      EqCZ, // for old demos where the CZ was the alternative for the P250
	EqFiveSeven: EqCZ,
	EqTec9:      EqCZ,
	EqDeagle:    EqRevolver,
	EqMP7:       EqMP5,
	EqM4A4:      EqM4A1,
}

// EquipmentAlternative returns the EquipmentType of the alternatively equippable weapon.
// E.g. returns EquipmentAlternative(EqP2000) returns EqUSP.
// Only works one way (default-to-alternative) as the Five-Seven and Tec-9 both map to the CZ-75.
func EquipmentAlternative(eq EquipmentType) EquipmentType {
	return equipmentToAlternative[eq]
}
