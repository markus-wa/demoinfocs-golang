package main

import (
	"image"
	"image/png"
	"log"
	"os"

	heatmap "github.com/dustin/go-heatmap"
	schemes "github.com/dustin/go-heatmap/schemes"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

const defaultDemPath = "../../test/cs-demos/default.dem"

// Run like this: go run heatmap.go > out.png
func main() {
	f, err := os.Open(defaultDemPath)
	checkErr(err)
	defer f.Close()

	p := dem.NewParser(f)

	// Parse header (contains map-name etc.)
	_, err = p.ParseHeader()
	checkErr(err)

	// Register handler for WeaponFiredEvent, triggered every time a shot is fired
	points := []heatmap.DataPoint{}
	p.RegisterEventHandler(func(e events.WeaponFiredEvent) {
		// Add shooter's position as datapoint
		points = append(points, heatmap.P(e.Shooter.Position.X, e.Shooter.Position.Y))
	})

	// Parse to end
	err = p.ParseToEnd()
	checkErr(err)

	// Generate heatmap and write to standard output
	img := heatmap.Heatmap(image.Rect(0, 0, 1024, 1024), points, 15, 128, schemes.AlphaFire)
	png.Encode(os.Stdout, img)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
