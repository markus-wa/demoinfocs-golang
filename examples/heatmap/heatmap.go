package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"

	heatmap "github.com/dustin/go-heatmap"
	schemes "github.com/dustin/go-heatmap/schemes"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
	metadata "github.com/markus-wa/demoinfocs-golang/metadata"
)

// Run like this: go run heatmap.go -demo /path/to/demo.dem > out.jpg
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := dem.NewParser(f)

	// Parse header (contains map-name etc.)
	header, err := p.ParseHeader()
	checkError(err)

	// Get metadata for the map that's being played
	m := metadata.MapNameToMap[header.MapName]

	// Register handler for WeaponFire, triggered every time a shot is fired
	points := []heatmap.DataPoint{}
	var bounds image.Rectangle
	p.RegisterEventHandler(func(e events.WeaponFire) {
		// Translate positions from in-game coordinates to radar overview
		x, y := m.TranslateScale(e.Shooter.Position.X, e.Shooter.Position.Y)

		// Track bounds to get around the normalization done by the heatmap library
		bounds = updatedBounds(bounds, int(x), int(y))

		// Add shooter's position as datapoint
		// Invert Y since it expects data to be ordered from bottom to top
		points = append(points, heatmap.P(x, y*-1))
	})

	// Parse to end
	err = p.ParseToEnd()
	checkError(err)

	// Get map overview as base image
	fMap, err := os.Open(fmt.Sprintf("../../metadata/maps/%s.jpg", header.MapName))
	checkError(err)

	imgMap, _, err := image.Decode(fMap)
	checkError(err)

	// Create output canvas
	img := image.NewRGBA(imgMap.Bounds())

	draw.Draw(img, imgMap.Bounds(), imgMap, image.ZP, draw.Over)

	// Generate heatmap
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	imgHeatmap := heatmap.Heatmap(image.Rect(0, 0, width, height), points, 15, 128, schemes.AlphaFire)

	// Draw it on top of the overview
	draw.Draw(img, bounds, imgHeatmap, image.ZP, draw.Over)

	// Write to stdout
	err = jpeg.Encode(os.Stdout, img, &jpeg.Options{
		Quality: 90,
	})
	checkError(err)
}

func updatedBounds(b image.Rectangle, x, y int) image.Rectangle {
	if b.Min.X > x || b.Min.X == 0 {
		b.Min.X = x
	} else if b.Max.X < x {
		b.Max.X = x
	}

	if b.Min.Y > y || b.Min.Y == 0 {
		b.Min.Y = y
	} else if b.Max.Y < y {
		b.Max.Y = y
	}

	return b
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
