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
	"strconv"

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

// RedirectStdout redirects standard output to dev null.
// Panics if an error occurs.
func RedirectStdout(f func()) {
	// Redirect stdout, the resulting image is written to this
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	os.Stdout = w

	// Discard the output in a separate goroutine so writing to stdout can't block indefinitely
	go func() {
		for err := error(nil); err == nil; _, err = io.Copy(ioutil.Discard, r) {
		}
	}()

	f()

	os.Stdout = old
}

func GetMapMetadata(name string, crc uint32) (metadata.Map, error) {
	url := fmt.Sprintf("https://radar-overviews.csgo.saiko.tech/%s/%d/info.json", name, crc)

	resp, err := http.Get(url)
	if err != nil {
		return metadata.Map{}, fmt.Errorf("failed to get map info.json from %q: %v", url, err)
	}

	defer resp.Body.Close()

	var data map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return metadata.Map{}, fmt.Errorf("failed to decode as JSON: %v", err)
	}

	mapInfo, ok := data[name].(map[string]interface{})
	if !ok {
		return metadata.Map{}, fmt.Errorf("failed to get map info.json entry for %q", name)
	}

	x, err := strconv.ParseFloat(mapInfo["pos_x"].(string), 64)
	if err != nil {
		return metadata.Map{}, fmt.Errorf("failed to get origin for X coordinate: %v", err)
	}

	y, err := strconv.ParseFloat(mapInfo["pos_y"].(string), 64)
	if err != nil {
		return metadata.Map{}, fmt.Errorf("failed to get origin for Y coordinate: %v", err)
	}

	scale, err := strconv.ParseFloat(mapInfo["scale"].(string), 64)
	if err != nil {
		return metadata.Map{}, fmt.Errorf("failed to get scale: %v", err)
	}

	return metadata.Map{
		Name: name,
		PZero: r2.Point{
			X: x,
			Y: y,
		},
		Scale: scale,
	}, nil
}

func GetMapRadar(name string, crc uint32) (image.Image, error) {
	url := fmt.Sprintf("https://radar-overviews.csgo.saiko.tech/%s/%d/radar.png", name, crc)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get map radar.png from %q: %v", url, err)
	}

	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode as image: %v", err)
	}

	return img, nil
}
