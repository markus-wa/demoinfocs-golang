package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"

	heatmap "github.com/dustin/go-heatmap"
	schemes "github.com/dustin/go-heatmap/schemes"
	r2 "github.com/golang/geo/r2"

	ex "github.com/markus-wa/demoinfocs-golang/v2/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	metadata "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
)

const (
	dotSize     = 15
	opacity     = 128
	jpegQuality = 90
)

// Run like this: go run heatmap.go -demo /path/to/demo.dem > out.jpg
func main() {
	//
	// Parsing
	//

	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	// Parse header (contains map-name etc.)
	header, err := p.ParseHeader()
	checkError(err)

	// Get metadata for the map that the game was played on for coordinate translations
	mapMetadata := metadata.MapNameToMap[header.MapName]

	// Register handler for WeaponFire, triggered every time a shot is fired
	var points []r2.Point
	p.RegisterEventHandler(func(e events.WeaponFire) {
		// Translate positions from in-game coordinates to radar overview image pixels
		x, y := mapMetadata.TranslateScale(e.Shooter.Position().X, e.Shooter.Position().Y)

		points = append(points, r2.Point{X: x, Y: y})
	})

	// Parse the whole demo
	err = p.ParseToEnd()
	checkError(err)

	//
	// Preparation of heatmap data
	//

	// Find bounding rectangle for points to get around the normalization done by the heatmap library
	r2Bounds := r2.RectFromPoints(points...)
	padding := float64(dotSize) / 2.0 // Calculating padding amount to avoid shrinkage by the heatmap library
	bounds := image.Rectangle{
		Min: image.Point{X: int(r2Bounds.X.Lo - padding), Y: int(r2Bounds.Y.Lo - padding)},
		Max: image.Point{X: int(r2Bounds.X.Hi + padding), Y: int(r2Bounds.Y.Hi + padding)},
	}

	// Transform r2.Points into heatmap.DataPoints
	var data []heatmap.DataPoint
	for _, p := range points[1:] {
		// Invert Y since go-heatmap expects data to be ordered from bottom to top
		data = append(data, heatmap.P(p.X, p.Y*-1))
	}

	//
	// Drawing the image
	//

	// Load map overview image
	fMap, err := os.Open(fmt.Sprintf("../../assets/maps/%s.jpg", header.MapName))
	checkError(err)
	imgMap, _, err := image.Decode(fMap)
	checkError(err)

	// Create output canvas and use map overview image as base
	img := image.NewRGBA(imgMap.Bounds())
	draw.Draw(img, imgMap.Bounds(), imgMap, image.Point{}, draw.Over)

	// Generate and draw heatmap overlay on top of the overview
	imgHeatmap := heatmap.Heatmap(image.Rect(0, 0, bounds.Dx(), bounds.Dy()), data, dotSize, opacity, schemes.AlphaFire)
	draw.Draw(img, bounds, imgHeatmap, image.Point{}, draw.Over)

	// Write to stdout
	err = jpeg.Encode(os.Stdout, img, &jpeg.Options{Quality: jpegQuality})
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
