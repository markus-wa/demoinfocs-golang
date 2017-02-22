package common

import (
	"fmt"
	"strings"
)

func MapEquipment(originalString string) EquipmentElement {
	originalString = strings.TrimPrefix(originalString, weaponPrefix)

	wep := EE_Unknown

	if strings.Contains(originalString, "knife") || strings.Contains(originalString, "bayonet") {
		wep = EE_Knife
	} else {
		switch originalString {
		case "ak47":
			wep = EE_AK47

		case "aug":
			wep = EE_AUG

		case "awp":
			wep = EE_AWP

		case "bizon":
			wep = EE_Bizon

		case "c4":
			wep = EE_Bomb

		case "deagle":
			wep = EE_Deagle

		case "decoy":
			fallthrough
		case "decoygrenade":
			wep = EE_Decoy

		case "elite":
			wep = EE_DualBarettas

		case "famas":
			wep = EE_Famas

		case "fiveseven":
			wep = EE_FiveSeven

		case "flashbang":
			wep = EE_Flash

		case "g3sg1":
			wep = EE_G3SG1

		case "galil":
			fallthrough
		case "galilar":
			wep = EE_Gallil

		case "glock":
			wep = EE_Glock

		case "hegrenade":
			wep = EE_HE

		case "hkp2000":
			wep = EE_P2000

		case "incgrenade":
			fallthrough
		case "incendiarygrenade":
			wep = EE_Incendiary

		case "m249":
			wep = EE_M249

		case "m4a1":
			wep = EE_M4A4

		case "mac10":
			wep = EE_Mac10

		case "mag7":
			wep = EE_Swag7

		case "molotov":
			fallthrough
		case "molotovgrenade":
			fallthrough
		case "molotov_projectile":
			wep = EE_Molotov

		case "mp7":
			wep = EE_MP7

		case "mp9":
			wep = EE_MP9

		case "negev":
			wep = EE_Negev

		case "nova":
			wep = EE_Nova

		case "p250":
			wep = EE_P250

		case "p90":
			wep = EE_P90

		case "sawedoff":
			wep = EE_SawedOff

		case "scar20":
			wep = EE_Scar20

		case "sg556":
			wep = EE_SG556

		case "smokegrenade":
			wep = EE_Smoke

		case "ssg08":
			wep = EE_Scout

		case "taser":
			wep = EE_Zeus

		case "tec9":
			wep = EE_Tec9

		case "ump45":
			wep = EE_UMP

		case "xm1014":
			wep = EE_XM1014

		case "m4a1_silencer":
			fallthrough
		case "m4a1_silencer_off":
			wep = EE_M4A1

		case "cz75a":
			wep = EE_CZ

		case "usp":
			fallthrough
		case "usp_silencer":
			fallthrough
		case "usp_silencer_off":
			wep = EE_USP

		case "world":
			wep = EE_World

		case "inferno":
			wep = EE_Incendiary

		case "revolver":
			wep = EE_Revolver

		case "sensorgrenade": // Only used in 'Co-op Strike' mode

		case "scar17": //These crash the game when given via give wep_[mp5navy|...], and cannot be purchased ingame.
		case "sg550": //yet the server-classes are networked, so we need to resolve them.
		case "mp5navy":
		case "p228":
		case "scout":
		case "sg552":
		case "tmp":

		default:
			fmt.Println("Unknown weapon " + originalString)
		}
	}
	return wep
}
