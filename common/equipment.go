package common

import (
	"math/rand"
	"strings"
)

// EquipmentClass is the type for the various EqClassXYZ constants.
type EquipmentClass int

// EquipmentClass constants give information about the type of an equipment (SMG, Rifle, Grenade etc.).
//
// Note: (EquipmentElement+99) / 100 = EquipmentClass
const (
	EqClassUnknown   EquipmentClass = 0
	EqClassPistols   EquipmentClass = 1
	EqClassSMG       EquipmentClass = 2
	EqClassHeavy     EquipmentClass = 3
	EqClassRifle     EquipmentClass = 4
	EqClassEquipment EquipmentClass = 5
	EqClassGrenade   EquipmentClass = 6
)

// EquipmentElement is the type for the various EqXYZ constants.
type EquipmentElement int

// Class returns the class of the equipment.
// E.g. pistol, smg, heavy etc.
func (e EquipmentElement) Class() EquipmentClass {
	return EquipmentClass((int(e) + 99) / 100)
}

// String returns a human readable name for the equipment.
// E.g. 'AK-47', 'UMP-45', 'Smoke Grenade' etc.
func (e EquipmentElement) String() string {
	return eqElementToName[e]
}

// EquipmentElement constants give information about what weapon a player has equipped.
const (
	EqUnknown EquipmentElement = 0

	// Pistols

	EqP2000        EquipmentElement = 1
	EqGlock        EquipmentElement = 2
	EqP250         EquipmentElement = 3
	EqDeagle       EquipmentElement = 4
	EqFiveSeven    EquipmentElement = 5
	EqDualBarettas EquipmentElement = 6
	EqTec9         EquipmentElement = 7
	EqCZ           EquipmentElement = 8
	EqUSP          EquipmentElement = 9
	EqRevolver     EquipmentElement = 10

	// SMGs

	EqMP7   EquipmentElement = 101
	EqMP9   EquipmentElement = 102
	EqBizon EquipmentElement = 103
	EqMac10 EquipmentElement = 104
	EqUMP   EquipmentElement = 105
	EqP90   EquipmentElement = 106

	// Heavy

	EqSawedOff EquipmentElement = 201
	EqNova     EquipmentElement = 202
	EqMag7     EquipmentElement = 203 // You should consider using EqSwag7 instead
	EqSwag7    EquipmentElement = 203
	EqXM1014   EquipmentElement = 204
	EqM249     EquipmentElement = 205
	EqNegev    EquipmentElement = 206

	// Rifles

	EqGalil  EquipmentElement = 301
	EqFamas  EquipmentElement = 302
	EqAK47   EquipmentElement = 303
	EqM4A4   EquipmentElement = 304
	EqM4A1   EquipmentElement = 305
	EqScout  EquipmentElement = 306
	EqSSG08  EquipmentElement = 306
	EqSG556  EquipmentElement = 307
	EqSG553  EquipmentElement = 307
	EqAUG    EquipmentElement = 308
	EqAWP    EquipmentElement = 309
	EqScar20 EquipmentElement = 310
	EqG3SG1  EquipmentElement = 311

	// Equipment

	EqZeus      EquipmentElement = 401
	EqKevlar    EquipmentElement = 402
	EqHelmet    EquipmentElement = 403
	EqBomb      EquipmentElement = 404
	EqKnife     EquipmentElement = 405
	EqDefuseKit EquipmentElement = 406
	EqWorld     EquipmentElement = 407

	// Grenades

	EqDecoy      EquipmentElement = 501
	EqMolotov    EquipmentElement = 502
	EqIncendiary EquipmentElement = 503
	EqFlash      EquipmentElement = 504
	EqSmoke      EquipmentElement = 505
	EqHE         EquipmentElement = 506
)

var eqNameToWeapon map[string]EquipmentElement

var eqElementToName map[EquipmentElement]string

func init() {
	initEqNameToWeapon()
	initEqElementToName()
}

func initEqNameToWeapon() {
	eqNameToWeapon = make(map[string]EquipmentElement)

	eqNameToWeapon["ak47"] = EqAK47
	eqNameToWeapon["aug"] = EqAUG
	eqNameToWeapon["awp"] = EqAWP
	eqNameToWeapon["bizon"] = EqBizon
	eqNameToWeapon["c4"] = EqBomb
	eqNameToWeapon["deagle"] = EqDeagle
	eqNameToWeapon["decoy"] = EqDecoy
	eqNameToWeapon["decoygrenade"] = EqDecoy
	eqNameToWeapon["decoyprojectile"] = EqDecoy
	eqNameToWeapon["elite"] = EqDualBarettas
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
	eqNameToWeapon["mp7"] = EqMP7
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
	eqNameToWeapon["worldspawn"] = EqUnknown
}

func initEqElementToName() {
	eqElementToName = make(map[EquipmentElement]string)
	eqElementToName[EqAK47] = "AK-47"
	eqElementToName[EqAUG] = "AUG"
	eqElementToName[EqAWP] = "AWP"
	eqElementToName[EqBizon] = "PP-Bizon"
	eqElementToName[EqBomb] = "C4"
	eqElementToName[EqDeagle] = "Desert Eagle"
	eqElementToName[EqDecoy] = "Decoy Grenade"
	eqElementToName[EqDualBarettas] = "Dual Barettas"
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

// MapEquipment creates an EquipmentElement from the name of the weapon / equipment.
// Returns EqUnknown if no mapping can be found.
func MapEquipment(eqName string) EquipmentElement {
	eqName = strings.TrimPrefix(eqName, weaponPrefix)

	var wep EquipmentElement
	if strings.Contains(eqName, "knife") || strings.Contains(eqName, "bayonet") {
		wep = EqKnife
	} else {
		// If the eqName isn't known it will be EqUnknown as that is the default value for EquipmentElement
		wep = eqNameToWeapon[eqName]
	}

	return wep
}

// Equipment is a weapon / piece of equipment belonging to a player.
// This also includes the skin and some additional data.
type Equipment struct {
	EntityID       int              // ID of the game entity
	Weapon         EquipmentElement // The type of weapon which the equipment instantiates.
	Owner          *Player          // The player carrying the equipment, not necessarily the buyer.
	AmmoType       int              // TODO: Remove this? doesn't seem applicable to CS:GO
	AmmoInMagazine int              // Amount of bullets in the weapon's magazine
	AmmoReserve    int              // Amount of reserve bullets
	OriginalString string           // E.g. 'models/weapons/w_rif_m4a1_s.mdl'. Used internally to differentiate alternative weapons (M4A4 / M4A1-S etc.).
	ZoomLevel      int              // How far the player has zoomed in on the weapon. 0=no zoom, 1=first level, 2=maximum zoom

	uniqueID int64
}

// Class returns the class of the equipment.
// E.g. pistol, smg, heavy etc.
func (e Equipment) Class() EquipmentClass {
	return e.Weapon.Class()
}

// UniqueID returns the unique id of the equipment element.
// The unique id is a random int generated internally by this library and can be used to differentiate
// equipment from each other. This is needed because demo-files reuse entity ids.
func (e Equipment) UniqueID() int64 {
	return e.uniqueID
}

// NewEquipment creates a new Equipment and sets the UniqueID.
//
// Intended for internal use only.
func NewEquipment(wep EquipmentElement) Equipment {
	return Equipment{Weapon: wep, uniqueID: rand.Int63()}
}
