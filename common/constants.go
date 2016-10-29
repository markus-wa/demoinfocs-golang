package common

import ()

const (
	MaxEditctBits = 11
	IndexMask     = ((1 << MaxEditctBits) - 1)
)

const weaponPrefix = "weapon_"

type (
	RoundMVPReason   byte
	Hitgroup         byte
	RoundEndReason   byte
	Team             byte
	EquipmentElement int
	EquipmentClass   int
)

const (
	MVPReason_MostEliminations RoundMVPReason = iota + 1
	MVPReason_BombDefused
	MVPReason_BombPlanted
)

const (
	HG_Generic  Hitgroup = 0
	HG_Head     Hitgroup = 1
	HG_Chest    Hitgroup = 2
	HG_Stomach  Hitgroup = 3
	HG_LeftArm  Hitgroup = 4
	HG_RightArm Hitgroup = 5
	HG_LeftLeg  Hitgroup = 6
	HG_RightLeg Hitgroup = 7
	HG_Gear     Hitgroup = 10
)

const (
	RER_TargetBombed RoundEndReason = iota + 1
	RER_VIPEscaped
	RER_VIPKilled
	RER_TerroristsEscaped
	RER_CTStoppedEscape
	RER_TerroristsStopped
	RER_BombDefused
	RER_CTWin
	RER_TerrprostWin
	RER_Draw
	RER_HostagesRescued
	RER_TargetSaved
	RER_HostagesNotRescued
	RER_TerroristsNotEscaped
	RER_VIPNotEscaped
	RER_GameStart
	RER_TerroristsSurrender
	RER_CTSurrender
)

const (
	Team_Spectators Team = iota + 1
	Team_Terrorists
	Team_CounterTerrorists
)

const (
	EE_Unknown EquipmentElement = 0

	// Pistols
	EE_P2000        EquipmentElement = 1
	EE_Glock        EquipmentElement = 2
	EE_P250         EquipmentElement = 3
	EE_Deagle       EquipmentElement = 4
	EE_FiveSeven    EquipmentElement = 5
	EE_DualBarettas EquipmentElement = 6
	EE_Tec9         EquipmentElement = 7
	EE_CZ           EquipmentElement = 8
	EE_USP          EquipmentElement = 9
	EE_Revolver     EquipmentElement = 10

	// SMGs
	EE_MP7   EquipmentElement = 101
	EE_MP9   EquipmentElement = 102
	EE_Bizon EquipmentElement = 103
	EE_Mac10 EquipmentElement = 104
	EE_UMP   EquipmentElement = 105
	EE_P90   EquipmentElement = 106

	// Heavy
	EE_SawedOff EquipmentElement = 201
	EE_Nova     EquipmentElement = 202
	EE_Swag7    EquipmentElement = 203
	EE_XM1014   EquipmentElement = 204
	EE_M249     EquipmentElement = 205
	EE_Negev    EquipmentElement = 206

	// Rifle
	EE_Gallil EquipmentElement = 301
	EE_Famas  EquipmentElement = 302
	EE_AK47   EquipmentElement = 303
	EE_M4A4   EquipmentElement = 304
	EE_M4A1   EquipmentElement = 305
	EE_Scout  EquipmentElement = 306
	EE_SG556  EquipmentElement = 307
	EE_AUG    EquipmentElement = 308
	EE_AWP    EquipmentElement = 309
	EE_Scar20 EquipmentElement = 310
	EE_G3SG1  EquipmentElement = 311

	// Equipment
	EE_Zeus      EquipmentElement = 401
	EE_Kevlar    EquipmentElement = 402
	EE_Helmet    EquipmentElement = 403
	EE_Bomb      EquipmentElement = 404
	EE_Knife     EquipmentElement = 405
	EE_DefuseKit EquipmentElement = 406
	EE_World     EquipmentElement = 407

	// Grenades
	EE_Decoy      EquipmentElement = 501
	EE_Molotov    EquipmentElement = 502
	EE_Incendiary EquipmentElement = 503
	EE_Flash      EquipmentElement = 504
	EE_Smoke      EquipmentElement = 505
	EE_HE         EquipmentElement = 506
)

const (
	EC_Unknown EquipmentClass = iota
	EC_Pistols
	EC_SMG
	EC_Heavy
	EC_Rifle
	EC_Equipment
	EC_Grenade
)
