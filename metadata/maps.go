package metadata

// Map represents a CS:GO map. It contains information required to translate
// in-game world coordinates to coordinates relative to (0, 0)
type Map struct {
	Name string

	PosX float64
	PosY float64

	Scale  float64
	Rotate float64
	Zoom   float64
}

// Translate translates in-game world-relative coordinates to (0, 0) relative coordinates
func (m Map) Translate(x, y, z float64) (float64, float64, float64) {
	return x - m.PosX, m.PosY - y, z
}

// TranslateScale translates and scales in-game world-relative coordinates to (0, 0) relative coordinates
func (m Map) TranslateScale(x, y, z float64) (float64, float64, float64) {
	x, y, z = m.Translate(x, y, z)
	return x / m.Scale, y / m.Scale, z
}

var (
	MapDeCache = Map{
		Name:  "de_cache",
		PosX:  -2000,
		PosY:  3250,
		Scale: 5.5,
	}

	MapDeCanals = Map{
		Name:  "de_canals",
		PosX:  -2496,
		PosY:  1792,
		Scale: 4,
	}

	MapDeCbble = Map{
		Name:  "de_cbble",
		PosX:  -3840,
		PosY:  3072,
		Scale: 6,
	}

	MapDeDust2 = Map{
		Name:   "de_dust2",
		PosX:   -2400,
		PosY:   3383,
		Scale:  4.4,
		Rotate: 1,
		Zoom:   1.1,
	}

	MapDeInferno = Map{
		Name:  "de_inferno",
		PosX:  -2087,
		PosY:  3870,
		Scale: 4.9,
	}

	MapDeMirage = Map{
		Name:   "de_mirage",
		PosX:   -3230,
		PosY:   1713,
		Scale:  5,
		Rotate: 0,
		Zoom:   0,
	}

	MapDeNuke = Map{
		Name:  "de_nuke",
		PosX:  -3453,
		PosY:  2887,
		Scale: 7,
	}

	MapDeOverpass = Map{
		Name:   "de_overpass",
		PosX:   -4831,
		PosY:   1781,
		Scale:  5.2,
		Rotate: 0,
		Zoom:   0,
	}

	MapDeTrain = Map{
		Name:  "de_train",
		PosX:  -2477,
		PosY:  2392,
		Scale: 4.7,
	}
)

// MapNameToMap translates a map name to a Map.
var MapNameToMap = map[string]Map{
	"de_cache":    MapDeCache,
	"de_canals":   MapDeCanals,
	"de_cbble":    MapDeCbble,
	"de_dust2":    MapDeDust2,
	"de_inferno":  MapDeInferno,
	"de_mirage":   MapDeMirage,
	"de_nuke":     MapDeNuke,
	"de_overpass": MapDeOverpass,
	"de_train":    MapDeTrain,
}
