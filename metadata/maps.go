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

// Pre-defined map translations.
var (
	MapDeCache = Map{
		Name: "de_cache",
		PZero: r2.Point{
			X: -2000,
			Y: 3250,
		},
		Scale: 5.5,
	}

	MapDeCanals = Map{
		Name: "de_canals",
		PZero: r2.Point{
			X: -2496,
			Y: 1792,
		},
		Scale: 4,
	}

	MapDeCbble = Map{
		Name: "de_cbble",
		PZero: r2.Point{
			X: -3840,
			Y: 3072,
		},
		Scale: 6,
	}

	MapDeDust2 = Map{
		Name: "de_dust2",
		PZero: r2.Point{
			X: -2476,
			Y: 3239,
		},
		Scale: 4.4,
	}

	MapDeInferno = Map{
		Name: "de_inferno",
		PZero: r2.Point{
			X: -2087,
			Y: 3870,
		},
		Scale: 4.9,
	}

	MapDeMirage = Map{
		Name: "de_mirage",
		PZero: r2.Point{
			X: -3230,
			Y: 1713,
		},
		Scale: 5,
	}

	MapDeNuke = Map{
		Name: "de_nuke",
		PZero: r2.Point{
			X: -3453,
			Y: 2887,
		},
		Scale: 7,
	}

	MapDeOverpass = Map{
		Name: "de_overpass",
		PZero: r2.Point{
			X: -4831,
			Y: 1781,
		},
		Scale: 5.2,
	}

	MapDeTrain = Map{
		Name: "de_train",
		PZero: r2.Point{
			X: -2477,
			Y: 2392,
		},
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
