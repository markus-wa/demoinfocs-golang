package examples

import (
	"encoding/json"
	"fmt"
	"image"
	"net/http"
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
	url := fmt.Sprintf("https://radar-overviews.csgo.saiko.tech/%s/%d/info.json", name, crc)

	resp, err := http.Get(url)
	checkError(err)

	defer resp.Body.Close()

	var data map[string]Map

	err = json.NewDecoder(resp.Body).Decode(&data)
	checkError(err)

	mapInfo, ok := data[name]
	if !ok {
		panic(fmt.Sprintf("failed to get map info.json entry for %q", name))
	}

	return mapInfo
}

// GetMapRadar fetches the radar image for a specific map version from
// `https://radar-overviews.csgo.saiko.tech/<map>/<crc>/radar.png`.
// Panics if any error occurs.
func GetMapRadar(name string, crc uint32) image.Image {
	url := fmt.Sprintf("https://radar-overviews.csgo.saiko.tech/%s/%d/radar.png", name, crc)

	resp, err := http.Get(url)
	checkError(err)

	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	checkError(err)

	return img
}
