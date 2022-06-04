package examples

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/golang/geo/r2"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
)

// DemoPathFromArgs returns the value of the -demo command line flag.
// Panics if an error occurs.
func DemoPathFromArgs() string {
	fl := new(flag.FlagSet)

	demPathPtr := fl.String("demo", "", "Demo file `path`")

	err := fl.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}

	demPath := *demPathPtr

	return demPath
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// RedirectStdout redirects standard output to dev null.
// Panics if an error occurs.
func RedirectStdout(f func()) {
	// Redirect stdout, the resulting image is written to this
	old := os.Stdout

	r, w, err := os.Pipe()
	checkError(err)

	os.Stdout = w

	// Discard the output in a separate goroutine so writing to stdout can't block indefinitely
	go func() {
		for err := error(nil); err == nil; _, err = io.Copy(ioutil.Discard, r) {
		}
	}()

	f()

	os.Stdout = old
}

type mapMetadata struct {
	PosX  float64 `json:"pos_x,string"`
	PosY  float64 `json:"pos_y,string"`
	Scale float64 `json:"scale,string"`
}

// GetMapMetadata fetches metadata for a specific map version from
// `https://radar-overviews.csgo.saiko.tech/<map>/<crc>/info.json`.
// Panics if any error occurs.
func GetMapMetadata(name string, crc uint32) metadata.Map {
	url := fmt.Sprintf("https://radar-overviews.csgo.saiko.tech/%s/%d/info.json", name, crc)

	resp, err := http.Get(url)
	checkError(err)

	defer resp.Body.Close()

	var data map[string]mapMetadata

	err = json.NewDecoder(resp.Body).Decode(&data)
	checkError(err)

	mapInfo, ok := data[name]
	if !ok {
		panic(fmt.Sprintf("failed to get map info.json entry for %q", name))
	}

	return metadata.Map{
		Name: name,
		PZero: r2.Point{
			X: mapInfo.PosX,
			Y: mapInfo.PosY,
		},
		Scale: mapInfo.Scale,
	}
}

// GetMapMetadata fetches metadata for a specific map version from
// `https://radar-overviews.csgo.saiko.tech/<map>/<crc>/info.json`.
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
