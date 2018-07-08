package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	"log"
	"os"

	r3 "github.com/golang/geo/r3"
	draw2dimg "github.com/llgcode/draw2d/draw2dimg"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
)

type nadePath struct {
	wep  common.EquipmentElement
	path []r3.Vector
	team common.Team
}

var (
	colorFire  color.Color = color.RGBA{0xff, 0x00, 0x00, 0xff} // Red
	colorHE    color.Color = color.RGBA{0xff, 0xff, 0x00, 0xff} // Yellow
	colorFlash color.Color = color.RGBA{0x00, 0x00, 0xff, 0xff} // Blue, because of the color on the nade
	colorSmoke color.Color = color.RGBA{0xbe, 0xbe, 0xbe, 0xff} // Light gray
	colorDecoy color.Color = color.RGBA{0x96, 0x4b, 0x00, 0xff} // Brown, because it's shit :)
)

// Run like this: go run nade_trajectories.go -demo /path/to/demo.dem > nade_trajectories.jpg
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := dem.NewParser(f)

	_, err = p.ParseHeader()
	checkError(err)

	nadePaths := make(map[int64]*nadePath) // Currently live projectiles

	storeNadePath := func(id int64, pos r3.Vector, wep common.EquipmentElement, team common.Team) {
		if nadePaths[id] == nil {
			nadePaths[id] = &nadePath{
				wep:  wep,
				team: team,
			}
		}

		nadePaths[id].path = append(nadePaths[id].path, pos)
	}

	p.RegisterEventHandler(func(e events.NadeProjectileThrownEvent) {
		storeNadePath(e.Projectile.UniqueID(), e.Projectile.Position, e.Projectile.Weapon, e.Projectile.Thrower.Team)
	})

	p.RegisterEventHandler(func(e events.NadeProjectileBouncedEvent) {
		storeNadePath(e.Projectile.UniqueID(), e.Projectile.Position, e.Projectile.Weapon, e.Projectile.Thrower.Team)
	})

	var nadePathsFirstHalf []*nadePath
	round := 0
	p.RegisterEventHandler(func(events.RoundEndedEvent) {
		round++
		// Very cheap first half check (we only want the first teams CT-side nades in the example).
		// Won't work with demos that have match-restarts etc.
		if round == 15 {
			// Copy all nade paths from the first 15 rounds
			for _, np := range nadePaths {
				nadePathsFirstHalf = append(nadePathsFirstHalf, np)
			}
			nadePaths = make(map[int64]*nadePath)
		}
	})

	err = p.ParseToEnd()
	checkError(err)

	// Draw image

	// Create output canvas
	dest := image.NewRGBA(image.Rect(0, 0, 1024, 1024))

	// Use cache map overview as base
	fCache, err := os.Open("../de_cache.jpg")
	checkError(err)

	imgCache, _, err := image.Decode(fCache)
	checkError(err)
	draw.Draw(dest, dest.Bounds(), imgCache, image.Point{0, 0}, draw.Src)

	// Initialize the graphic context
	gc := draw2dimg.NewGraphicContext(dest)

	gc.SetLineWidth(1)                      // 1 px lines
	gc.SetFillColor(color.RGBA{0, 0, 0, 0}) // No fill, alpha 0

	for _, np := range nadePathsFirstHalf {
		if np.team != common.TeamCounterTerrorists {
			// Only draw CT nades
			continue
		}

		// Set colors
		switch np.wep {
		case common.EqMolotov:
			fallthrough
		case common.EqIncendiary:
			gc.SetStrokeColor(colorFire)

		case common.EqHE:
			gc.SetStrokeColor(colorHE)

		case common.EqFlash:
			gc.SetStrokeColor(colorFlash)

		case common.EqSmoke:
			gc.SetStrokeColor(colorSmoke)

		case common.EqDecoy:
			gc.SetStrokeColor(colorDecoy)

		default:
			// Set alpha to 0 so we don't draw unknown stuff
			gc.SetStrokeColor(color.RGBA{0x00, 0x00, 0x00, 0x00})
			fmt.Println("Unknown grenade type", np.wep)
		}

		// Draw path
		gc.MoveTo(translateX(np.path[0].X), translateY(np.path[0].Y)) // Move to a position to start the new path

		for _, pos := range np.path[1:] {
			gc.LineTo(translateX(pos.X), translateY(pos.Y))
		}

		gc.FillStroke()
	}

	// Write to standard output
	jpeg.Encode(os.Stdout, dest, &jpeg.Options{
		Quality: 90,
	})
}

// Rough translations for x & y coordinates from de_cache to 1024x1024 px.
// This could be done nicer by only having to provide the mapping between two source & target coordinates and the max size.
// Then we could calculate the correct stretch & offset automatically.

const (
	stretchX = 0.18
	offsetX  = 414

	stretchY = -0.18
	offsetY  = 630
)

func translateX(x float64) float64 {
	return x*stretchX + offsetX
}

func translateY(y float64) float64 {
	return y*stretchY + offsetY
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
