package examples

import (
	"image"
	"os"

	"github.com/chai2010/webp"
)

// Map represents a CS:GO map. It contains information required to translate
// in-game world coordinates to coordinates relative to (0, 0) on the provided map-overviews (radar images).
type Map struct {
	PosX  float64 `json:"pos_x,string"`
	PosY  float64 `json:"pos_y,string"`
	Scale float64 `json:"scale,string"`
}

// Translate translates in-game world-relative coordinates to (0, 0) relative coordinates.
func (m Map) Translate(x, y float64) (float64, float64) {
	return x - m.PosX, m.PosY - y
}

// TranslateScale translates and scales in-game world-relative coordinates to (0, 0) relative coordinates.
// The outputs are pixel coordinates for the radar images found in the maps folder.
func (m Map) TranslateScale(x, y float64) (float64, float64) {
	x, y = m.Translate(x, y)
	return x / m.Scale, y / m.Scale
}

// GetMapMetadata fetches metadata for a specific map version from
// `https://radar-overviews.csgo.saiko.tech/<map>/<crc>/info.json`.
// Panics if any error occurs.
func GetMapMetadata(name string, crc uint32) Map {
	return map[string]Map{
		"ar_baggage": {
			PosX:  -1316,
			PosY:  1288,
			Scale: 2.539062,
		},
		"de_nuke": {
			PosX:  -3453,
			PosY:  2887,
			Scale: 7,
		},
		"de_vertigo": {
			PosX:  -3168,
			PosY:  1762,
			Scale: 4,
		},
	}[name]
}

// GetMapRadar fetches the radar image for a specific map version from
// `https://radar-overviews.csgo.saiko.tech/<map>/<crc>/radar.png`.
// Panics if any error occurs.
func GetMapRadar(name string, crc uint32) image.Image {
	f, err := os.Open("assets/" + name + ".webp")
	checkError(err)

	defer f.Close()

	img, err := webp.Decode(f)
	checkError(err)

	return img
}
