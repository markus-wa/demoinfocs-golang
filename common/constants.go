package common

const weaponPrefix = "weapon_"

type (
	// RoundMVPReason is the type for the various MVPReasonYXZ constants.
	RoundMVPReason byte

	// HitGroup is the type for the various HitGroupXYZ constants.
	HitGroup byte

	// RoundEndReason is the type for the various RoundEndReasonXYZ constants.
	RoundEndReason byte

	// Team is the type for the various TeamXYZ constants.
	Team byte

	// EquipmentElement is the type for the various EqXYZ constants.
	EquipmentElement int

	// EquipmentClass is the type for the various EqClassXYZ constants.
	EquipmentClass int
)

// RoundMVPReasons constants give information about why a player got the MVP award.
const (
	MVPReasonMostEliminations RoundMVPReason = 1
	MVPReasonBombDefused      RoundMVPReason = 2
	MVPReasonBombPlanted      RoundMVPReason = 3
)

// HitGroup constants give information about where a player got hit.
// e.g. head, chest, legs etc.
const (
	HitGroupGeneric  HitGroup = 0
	HitGroupHead     HitGroup = 1
	HitGroupChest    HitGroup = 2
	HitGroupStomach  HitGroup = 3
	HitGroupLeftArm  HitGroup = 4
	HitGroupRightArm HitGroup = 5
	HitGroupLeftLeg  HitGroup = 6
	HitGroupRightLeg HitGroup = 7
	HitGroupGear     HitGroup = 10
)

// RoundEndReasons constants give information about why a round ended (Bomb defused, exploded etc.).
const (
	RoundEndReasonTargetBombed         RoundEndReason = 1
	RoundEndReasonVIPEscaped           RoundEndReason = 2
	RoundEndReasonVIPKilled            RoundEndReason = 3
	RoundEndReasonTerroristsEscaped    RoundEndReason = 4
	RoundEndReasonCTStoppedEscape      RoundEndReason = 5
	RoundEndReasonTerroristsStopped    RoundEndReason = 6
	RoundEndReasonBombDefused          RoundEndReason = 7
	RoundEndReasonCTWin                RoundEndReason = 8
	RoundEndReasonTerroristsWin        RoundEndReason = 9
	RoundEndReasonDraw                 RoundEndReason = 10
	RoundEndReasonHostagesRescued      RoundEndReason = 11
	RoundEndReasonTargetSaved          RoundEndReason = 12
	RoundEndReasonHostagesNotRescued   RoundEndReason = 13
	RoundEndReasonTerroristsNotEscaped RoundEndReason = 14
	RoundEndReasonVIPNotEscaped        RoundEndReason = 15
	RoundEndReasonGameStart            RoundEndReason = 16
	RoundEndReasonTerroristsSurrender  RoundEndReason = 17
	RoundEndReasonCTSurrender          RoundEndReason = 18
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
	EqSwag7    EquipmentElement = 203
	EqXM1014   EquipmentElement = 204
	EqM249     EquipmentElement = 205
	EqNegev    EquipmentElement = 206

	// Rifles

	EqGallil EquipmentElement = 301
	EqFamas  EquipmentElement = 302
	EqAK47   EquipmentElement = 303
	EqM4A4   EquipmentElement = 304
	EqM4A1   EquipmentElement = 305
	EqScout  EquipmentElement = 306
	EqSG556  EquipmentElement = 307
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

// EquipmentClass constants give information about the type of an equipment (SMG, Rifle, Grenade etc.).
// Note: EquipmentElement / 100 = EquipmentClass
const (
	EqClassUnknown   EquipmentClass = 0
	EqClassPistols   EquipmentClass = 1
	EqClassSMG       EquipmentClass = 2
	EqClassHeavy     EquipmentClass = 3
	EqClassRifle     EquipmentClass = 4
	EqClassEquipment EquipmentClass = 5
	EqClassGrenade   EquipmentClass = 6
)
