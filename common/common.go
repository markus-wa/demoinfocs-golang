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

// MapEquipment creates an EquipmentElement from the name of the weapon / equipment.
func MapEquipment(eqName string) EquipmentElement {
	eqName = strings.TrimPrefix(eqName, weaponPrefix)

	var wep EquipmentElement

	if strings.Contains(eqName, "knife") || strings.Contains(eqName, "bayonet") {
		wep = EqKnife
	} else {
		switch eqName {
		case "ak47":
			wep = EqAK47

		case "aug":
			wep = EqAUG

		case "awp":
			wep = EqAWP

		case "bizon":
			wep = EqBizon

		case "c4":
			wep = EqBomb

		case "deagle":
			wep = EqDeagle

		case "decoy":
			fallthrough
		case "decoygrenade":
			fallthrough
		case "decoy_projectile":
			wep = EqDecoy

		case "elite":
			wep = EqDualBarettas

		case "famas":
			wep = EqFamas

		case "fiveseven":
			wep = EqFiveSeven

		case "flashbang":
			wep = EqFlash

		case "g3sg1":
			wep = EqG3SG1

		case "galil":
			fallthrough
		case "galilar":
			wep = EqGallil

		case "glock":
			wep = EqGlock

		case "hegrenade":
			wep = EqHE

		case "hkp2000":
			wep = EqP2000

		case "incgrenade":
			fallthrough
		case "incendiarygrenade":
			wep = EqIncendiary

		case "m249":
			wep = EqM249

		case "m4a1":
			wep = EqM4A4

		case "mac10":
			wep = EqMac10

		case "mag7":
			wep = EqSwag7

		case "molotov":
			fallthrough
		case "molotovgrenade":
			fallthrough
		case "molotov_projectile":
			wep = EqMolotov

		case "mp7":
			wep = EqMP7

		case "mp9":
			wep = EqMP9

		case "negev":
			wep = EqNegev

		case "nova":
			wep = EqNova

		case "p250":
			wep = EqP250

		case "p90":
			wep = EqP90

		case "sawedoff":
			wep = EqSawedOff

		case "scar20":
			wep = EqScar20

		case "sg556":
			wep = EqSG556

		case "smokegrenade":
			wep = EqSmoke

		case "ssg08":
			wep = EqScout

		case "taser":
			wep = EqZeus

		case "tec9":
			wep = EqTec9

		case "ump45":
			wep = EqUMP

		case "xm1014":
			wep = EqXM1014

		case "m4a1_silencer":
			fallthrough
		case "m4a1_silencer_off":
			wep = EqM4A1

		case "cz75a":
			wep = EqCZ

		case "usp":
			fallthrough
		case "usp_silencer":
			fallthrough
		case "usp_silencer_off":
			wep = EqUSP

		case "world":
			wep = EqWorld

		case "inferno":
			wep = EqIncendiary

		case "revolver":
			wep = EqRevolver

		case "vest":
			wep = EqKevlar

		case "vesthelm":
			wep = EqHelmet

		case "defuser":
			wep = EqDefuseKit

		case "sensorgrenade": // Only used in 'Co-op Strike' mode

		case "scar17": //These crash the game when given via give wep_[mp5navy|...], and cannot be purchased ingame.
		case "sg550": //yet the server-classes are networked, so we need to resolve them.
		case "mp5navy":
		case "p228":
		case "scout":
		case "sg552":
		case "tmp":
		case "worldspawn":

		default:
			// TODO: This is really bad imo but we can't access Parser.warn here :/
			wep = EqUnknown
		}
	}
	return wep
}
