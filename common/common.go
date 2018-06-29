// Package common contains common types, constants and functions used over different demoinfocs packages.
// Some constants prefixes:
// MVPReason - the reason why someone got the MVP award.
// HG - HitGroup - where a bullet hit the player.
// EE - EquipmentElement - basically the weapon identifiers.
// RER - RoundEndReason - why the round ended (bomb exploded, defused, time ran out. . .).
// EC - EquipmentClass - type of equipment (pistol, smg, heavy. . .).
package common

import (
	"strings"
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
	eqElementToName[EqM4A4] = "M4A1"
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
}

// MapEquipment creates an EquipmentElement from the name of the weapon / equipment.
func MapEquipment(eqName string) EquipmentElement {
	eqName = strings.TrimPrefix(eqName, weaponPrefix)

	var wep EquipmentElement
	if strings.Contains(eqName, "knife") || strings.Contains(eqName, "bayonet") {
		wep = EqKnife
	} else {
		// If the eqName isn't known it will be EqUnknown as that is the default value for EquipmentElement
		var ok bool
		if wep, ok = eqNameToWeapon[eqName]; !ok {
			// TODO: Return error / warning for unmapped weapons
		}
	}

	return wep
}
