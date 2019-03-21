// Package metadata provides metadata and utility functions,
// like translations from ingame coordinates to radar image pixels.
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

func MakeMap(name string, x, y, scale float64) Map {
    return Map{
		Name: name,
		PZero: r2.Point{
			X: x,
			Y: y,
		},
		Scale: scale,
	}
}

// Pre-defined map translations.
// MapNameToMap translates a map name to a Map.
var MapNameToMap = map[string]Map{
	"de_cache":    MakeMap("de_cache",    -2000, 1250, 5.5),
	"de_canals":   MakeMap("de_canals",   -2496, 1792, 4.0),
	"de_cbble":    MakeMap("de_cbble",    -3840, 3072, 6.0),
	"de_dust2":    MakeMap("de_dust2",    -2476, 3239, 4.4),
	"de_inferno":  MakeMap("de_inferno",  -2087, 3870, 4.9),
	"de_mirage":   MakeMap("de_mirage",   -3230, 1713, 5.0),
	"de_nuke":     MakeMap("de_nuke",     -3453, 2887, 7.0),
	"de_overpass": MakeMap("de_overpass", -4831, 1781, 5.2),
	"de_train":    MakeMap("de_train",    -2477, 2392, 4.7),
}
