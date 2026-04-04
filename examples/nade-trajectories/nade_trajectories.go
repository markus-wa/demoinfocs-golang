package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"

	"github.com/golang/geo/r2"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
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

	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		// Get metadata for the map that the game was played on for coordinate translations
		curMap = ex.GetMapMetadata(msg.GetMapName())

		// Load map overview image
		mapRadarImg = ex.GetMapRadar(msg.GetMapName())
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

		nadeTrajectories[id].path = e.Projectile.Trajectory
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

	p.RegisterEventHandler(func(start events.RoundEnd) {
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

	// canvas uses mm with y-up; we use DPMM(1) so 1mm = 1px.
	// y coordinates must be flipped: canvas_y = height - image_y
	h := float64(dest.Bounds().Dy())

	ras := rasterizer.FromImage(dest, canvas.DPMM(1), canvas.SRGBColorSpace{})
	ctx := canvas.NewContext(ras)

	// Draw infernos first so they're in the background
	drawInfernos(ctx, infernosFirst5Rounds, h)

	// Then trajectories on top of everything
	drawTrajectories(ctx, nadeTrajectoriesFirst5Rounds, h)

	ras.Close()

	// Write to standard output
	err = jpeg.Encode(os.Stdout, dest, &jpeg.Options{
		Quality: 90,
	})
	checkError(err)
}

func drawInfernos(ctx *canvas.Context, infernos []*common.Inferno, h float64) {
	// Draw areas first
	ctx.SetFillColor(colorInferno)

	// Calculate hulls
	hulls := make([][]r2.Point, len(infernos))
	for i := range infernos {
		hulls[i] = infernos[i].Fires().ConvexHull2D()
	}

	for _, hull := range hulls {
		buildInfernoPath(ctx, hull, h)
		ctx.Fill()
	}

	// Then the outline
	ctx.SetFillColor(colorInferno)
	ctx.SetStrokeColor(colorInfernoHull)
	ctx.SetStrokeWidth(1) // 1 px wide (1mm at DPMM(1))

	for _, hull := range hulls {
		buildInfernoPath(ctx, hull, h)
		ctx.FillStroke()
	}
}

func buildInfernoPath(ctx *canvas.Context, vertices []r2.Point, h float64) {
	xOrigin, yOrigin := curMap.TranslateScale(vertices[0].X, vertices[0].Y)
	ctx.MoveTo(xOrigin, h-yOrigin)

	for _, fire := range vertices[1:] {
		x, y := curMap.TranslateScale(fire.X, fire.Y)
		ctx.LineTo(x, h-y)
	}

	ctx.Close()
}

func drawTrajectories(ctx *canvas.Context, trajectories []*nadePath, h float64) {
	ctx.SetStrokeWidth(1) // 1 px lines (1mm at DPMM(1))

	for _, np := range trajectories {
		// Set colors
		switch np.wep {
		case common.EqMolotov:
			fallthrough
		case common.EqIncendiary:
			ctx.SetStrokeColor(colorFireNade)

		case common.EqHE:
			ctx.SetStrokeColor(colorHE)

		case common.EqFlash:
			ctx.SetStrokeColor(colorFlash)

		case common.EqSmoke:
			ctx.SetStrokeColor(colorSmoke)

		case common.EqDecoy:
			ctx.SetStrokeColor(colorDecoy)

		default:
			// Set alpha to 0 so we don't draw unknown stuff
			ctx.SetStrokeColor(color.RGBA{0x00, 0x00, 0x00, 0x00})
			fmt.Println("Unknown grenade type", np.wep)
		}

		// Draw path
		x, y := curMap.TranslateScale(np.path[0].Position.X, np.path[0].Position.Y)
		ctx.MoveTo(x, h-y) // Move to a position to start the new path

		for _, pos := range np.path[1:] {
			x, y := curMap.TranslateScale(pos.Position.X, pos.Position.Y)
			ctx.LineTo(x, h-y)
		}

		ctx.Stroke()
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
