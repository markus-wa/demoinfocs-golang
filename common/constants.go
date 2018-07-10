package common

const weaponPrefix = "weapon_"

type (
	// Team is the type for the various TeamXYZ constants.
	Team byte

	// EquipmentElement is the type for the various EqXYZ constants.
	EquipmentElement int

	// EquipmentClass is the type for the various EqClassXYZ constants.
	EquipmentClass int
)

// Team constants give information about which team a player is on.
const (
	TeamUnassigned Team = iota
	TeamSpectators
	TeamTerrorists
	TeamCounterTerrorists
)

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
