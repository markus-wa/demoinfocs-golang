// Package metadata provides metadata and utility functions,
// like translations from ingame coordinates to radar image pixels (see also /assets/maps directory).
package metadata

import (
	"github.com/golang/geo/r2"
)

// Map represents a CS:GO map. It contains information required to translate
// in-game world coordinates to coordinates relative to (0, 0) on the provided map-overviews (radar images).
type Map struct {
	Name  string
	PZero r2.Point
	Scale float64
}

// Translate translates in-game world-relative coordinates to (0, 0) relative coordinates.
func (m Map) Translate(x, y float64) (float64, float64) {
	return x - m.PZero.X, m.PZero.Y - y
}

// TranslateScale translates and scales in-game world-relative coordinates to (0, 0) relative coordinates.
// The outputs are pixel coordinates for the radar images found in the maps folder.
func (m Map) TranslateScale(x, y float64) (float64, float64) {
	x, y = m.Translate(x, y)
	return x / m.Scale, y / m.Scale
}

// MapNameToMap translates a map name to a Map.
var MapNameToMap = make(map[string]Map)

// makeMap creates a map stuct initialized with the given parameters.
func makeMap(name string, x, y, scale float64) Map {
	m := Map{Name: name, PZero: r2.Point{X: x, Y: y}, Scale: scale}

	MapNameToMap[name] = m

	return m
}

// Pre-defined map translations.
// see "steamapps/common/Counter-Strike Global Offensive/csgo/resource/overviews/*.txt"
var (
	MapDeAncient  = makeMap("de_ancient", -2953, 2164, 5)
	MapDeCache    = makeMap("de_cache", -2000, 3250, 5.5)
	MapDeCanals   = makeMap("de_canals", -2496, 1792, 4)
	MapDeCbble    = makeMap("de_cbble", -3840, 3072, 6)
	MapDeDust2    = makeMap("de_dust2", -2476, 3239, 4.4)
	MapDeInferno  = makeMap("de_inferno", -2087, 3870, 4.9)
	MapDeMirage   = makeMap("de_mirage", -3230, 1713, 5)
	MapDeNuke     = makeMap("de_nuke", -3453, 2887, 7)
	MapDeOverpass = makeMap("de_overpass", -4831, 1781, 5.2)
	MapDeTrain    = makeMap("de_train", -2477, 2392, 4.7)
	MapDeVertigo  = makeMap("de_vertigo", -3168, 1762, 4)
	MapCsAgency   = makeMap("cs_agency", -2947, 2492, 5)
	MapCsOffice   = makeMap("cs_office", -1838, 1858, 4.1)
)
