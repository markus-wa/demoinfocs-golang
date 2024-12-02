package examples

import (
	"embed"
	"encoding/json"
	"fmt"
	"image"

	"github.com/andygrunwald/vdf"
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

//go:embed _assets/*
var fs embed.FS

// GetMapMetadata fetches metadata for a specific map version from
// `https://radar-overviews.csgo.saiko.tech/<map>/<crc>/info.json`.
// Panics if any error occurs.
func GetMapMetadata(name string) Map {
	f, err := fs.Open(fmt.Sprintf("_assets/metadata/%s.txt", name))
	checkError(err)

	defer f.Close()

	m, err := vdf.NewParser(f).Parse()
	checkError(err)

	b, err := json.Marshal(m)
	checkError(err)

	var data map[string]Map

	err = json.Unmarshal(b, &data)
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
func GetMapRadar(name string) image.Image {
	f, err := fs.Open(fmt.Sprintf("_assets/radar/%s_radar_psd.png", name))
	checkError(err)

	defer f.Close()

	img, _, err := image.Decode(f)
	checkError(err)

	return img
}
