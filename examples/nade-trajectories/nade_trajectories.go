package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"

	"github.com/golang/geo/r2"
	"github.com/llgcode/draw2d/draw2dimg"

	ex "github.com/markus-wa/demoinfocs-golang/v4/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

type nadePath struct {
	wep  common.EquipmentType
	path []common.TrajectoryEntry
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
var curMap ex.Map

// Run like this: go run nade_trajectories.go -demo /path/to/demo.dem > nade_trajectories.jpg
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)

	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	var (
		mapRadarImg image.Image
	)

	p.RegisterNetMessageHandler(func(msg *msgs2.CSVCMsg_ServerInfo) {
		// Get metadata for the map that the game was played on for coordinate translations
		curMap = ex.GetMapMetadata(msg.GetMapName(), 0)

		// Load map overview image
		mapRadarImg = ex.GetMapRadar(msg.GetMapName(), 0)
	})

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
				wep:  e.Projectile.WeaponInstance.Type,
				team: team,
			}
		}

		nadeTrajectories[id].path = e.Projectile.Trajectory2
	})

	var infernos []*common.Inferno

	p.RegisterEventHandler(func(e events.InfernoExpired) {
		infernos = append(infernos, e.Inferno)
	})

	var (
		nadeTrajectoriesFirst5Rounds []*nadePath
		infernosFirst5Rounds         []*common.Inferno
		round                        = 0
	)

	p.RegisterEventHandler(func(start events.RoundStart) {
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

		round++
	})

	err = p.ParseToEnd()
	checkError(err)

	// Create output canvas
	dest := image.NewRGBA(mapRadarImg.Bounds())

	// Draw image
	draw.Draw(dest, dest.Bounds(), mapRadarImg, image.Point{}, draw.Src)

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
		hulls[i] = infernos[i].Fires().ConvexHull2D()
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
		if len(np.path) == 0 {
			fmt.Fprintf(os.Stderr, "No path for nade trajectory of type %v\n", np.wep)

			continue
		}

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
		x, y := curMap.TranslateScale(np.path[0].Position.X, np.path[0].Position.Y)
		gc.MoveTo(x, y) // Move to a position to start the new path

		for _, pos := range np.path[1:] {
			x, y := curMap.TranslateScale(pos.Position.X, pos.Position.Y)
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
