package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	"os"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/llgcode/draw2d/draw2dimg"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
	"github.com/markus-wa/demoinfocs-golang/metadata"
)

type nadePath struct {
	wep  common.EquipmentElement
	path []r3.Vector
	team common.Team
}

var (
	colorFireNade    color.Color = color.RGBA{0xff, 0x00, 0x00, 0xff} // Red
	colorInferno     color.Color = color.RGBA{0xff, 0xa5, 0x00, 0xff} // Orange
	colorInfernoHull color.Color = color.RGBA{0xff, 0xff, 0x00, 0xff} // Yellow
	colorHE          color.Color = color.RGBA{0x00, 0xff, 0x00, 0xff} // Green
	colorFlash       color.Color = color.RGBA{0x00, 0x00, 0xff, 0xff} // Blue, because of the color on the nade
	colorSmoke       color.Color = color.RGBA{0xbe, 0xbe, 0xbe, 0xff} // Light gray
	colorDecoy       color.Color = color.RGBA{0x96, 0x4b, 0x00, 0xff} // Brown, because it's shit :)
)

// Store the curret map so we don't have to pass it to functions
var curMap metadata.Map

// Run like this: go run nade_trajectories.go -demo /path/to/demo.dem > nade_trajectories.jpg
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := dem.NewParser(f)

	header, err := p.ParseHeader()
	checkError(err)

	curMap = metadata.MapNameToMap[header.MapName]

	nadeTrajectories := make(map[int64]*nadePath) // Trajectories of all destroyed nades

	p.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
		id := e.Projectile.UniqueID()

		// Sometimes the thrower is nil, in that case we want the team to be unassigned (which is the default value)
		var team common.Team
		if e.Projectile.Thrower != nil {
			team = e.Projectile.Thrower.Team
		}

		if nadeTrajectories[id] == nil {
			nadeTrajectories[id] = &nadePath{
				wep:  e.Projectile.Weapon,
				team: team,
			}
		}

		nadeTrajectories[id].path = e.Projectile.Trajectory
	})

	var infernos []*common.Inferno

	p.RegisterEventHandler(func(e events.InfernoExpired) {
		infernos = append(infernos, e.Inferno)
	})

	var nadeTrajectoriesFirst5Rounds []*nadePath
	var infernosFirst5Rounds []*common.Inferno
	round := 0
	p.RegisterEventHandler(func(events.RoundEnd) {
		round++
		// We only want the data from the first 5 rounds so the image is not too cluttered
		// This is a very cheap way to do it. Won't work with demos that have match-restarts etc.
		if round == 5 {
			// Copy nade paths
			for _, np := range nadeTrajectories {
				nadeTrajectoriesFirst5Rounds = append(nadeTrajectoriesFirst5Rounds, np)
			}
			nadeTrajectories = make(map[int64]*nadePath)

			// Copy infernos
			infernosFirst5Rounds = make([]*common.Inferno, len(infernos))
			copy(infernosFirst5Rounds, infernos)
		}
	})

	err = p.ParseToEnd()
	checkError(err)

	// Use map overview as base image
	fMap, err := os.Open(fmt.Sprintf("../../metadata/maps/%s.jpg", header.MapName))
	checkError(err)

	imgMap, _, err := image.Decode(fMap)
	checkError(err)

	// Create output canvas
	dest := image.NewRGBA(imgMap.Bounds())

	// Draw image
	draw.Draw(dest, dest.Bounds(), imgMap, image.ZP, draw.Src)

	// Initialize the graphic context
	gc := draw2dimg.NewGraphicContext(dest)

	// Draw infernos first so they're in the background
	drawInfernos(gc, infernosFirst5Rounds)

	// Then trajectories on top of everything
	drawTrajectories(gc, nadeTrajectoriesFirst5Rounds)

	// Write to standard output
	err = jpeg.Encode(os.Stdout, dest, &jpeg.Options{
		Quality: 90,
	})
	checkError(err)
}

func drawInfernos(gc *draw2dimg.GraphicContext, infernos []*common.Inferno) {
	// Draw areas first
	gc.SetFillColor(colorInferno)

	// Calculate hulls
	hulls := make([][]r2.Point, len(infernos))
	for i := range infernos {
		hulls[i] = infernos[i].ConvexHull2D()
	}

	for _, hull := range hulls {
		buildInfernoPath(gc, hull)
		gc.Fill()
	}

	// Then the outline
	gc.SetStrokeColor(colorInfernoHull)
	gc.SetLineWidth(1) // 1 px wide

	for _, hull := range hulls {
		buildInfernoPath(gc, hull)
		gc.FillStroke()
	}
}

func buildInfernoPath(gc *draw2dimg.GraphicContext, vertices []r2.Point) {
	xOrigin, yOrigin := curMap.TranslateScale(vertices[0].X, vertices[0].Y)
	gc.MoveTo(xOrigin, yOrigin)

	for _, fire := range vertices[1:] {
		x, y := curMap.TranslateScale(fire.X, fire.Y)
		gc.LineTo(x, y)
	}

	gc.LineTo(xOrigin, yOrigin)
}

func drawTrajectories(gc *draw2dimg.GraphicContext, trajectories []*nadePath) {
	gc.SetLineWidth(1)                      // 1 px lines
	gc.SetFillColor(color.RGBA{0, 0, 0, 0}) // No fill, alpha 0

	for _, np := range trajectories {
		// Set colors
		switch np.wep {
		case common.EqMolotov:
			fallthrough
		case common.EqIncendiary:
			gc.SetStrokeColor(colorFireNade)

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
		x, y := curMap.TranslateScale(np.path[0].X, np.path[0].Y)
		gc.MoveTo(x, y) // Move to a position to start the new path

		for _, pos := range np.path[1:] {
			x, y := curMap.TranslateScale(pos.X, pos.Y)
			gc.LineTo(x, y)
		}

		gc.FillStroke()
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
