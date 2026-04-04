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
	colorFireNade    color.Color = color.RGBA{0xff, 0x44, 0x44, 0xff} // Red
	colorInferno     color.Color = color.RGBA{0xff, 0x88, 0x00, 0xb4} // Orange, semi-transparent
	colorInfernoHull color.Color = color.RGBA{0xff, 0xdd, 0x00, 0xe0} // Yellow
	colorHE          color.Color = color.RGBA{0x44, 0xff, 0x44, 0xff} // Green
	colorFlash       color.Color = color.RGBA{0x44, 0x88, 0xff, 0xff} // Blue
	colorSmoke       color.Color = color.RGBA{0xcc, 0xcc, 0xcc, 0xff} // Light gray
	colorDecoy       color.Color = color.RGBA{0xaa, 0x66, 0x22, 0xff} // Brown
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

	ctx.SetStrokeCapper(canvas.RoundCap)
	ctx.SetStrokeJoiner(canvas.RoundJoin)

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

// withAlpha returns the color with the alpha channel replaced.
func withAlpha(c color.Color, a uint8) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), a}
}

func drawInfernos(ctx *canvas.Context, infernos []*common.Inferno, h float64) {
	// Calculate hulls
	hulls := make([][]r2.Point, len(infernos))
	for i := range infernos {
		hulls[i] = infernos[i].Fires().ConvexHull2D()
	}

	// Soft glow halo around each inferno
	ctx.SetFillColor(withAlpha(colorInferno, 40))
	ctx.SetStrokeColor(withAlpha(colorInfernoHull, 0))
	ctx.SetStrokeWidth(6)
	for _, hull := range hulls {
		buildInfernoPath(ctx, hull, h)
		ctx.Fill()
	}

	// Semi-transparent fill
	ctx.SetFillColor(colorInferno) // already has alpha 0xb4
	ctx.SetStrokeColor(colorInfernoHull)
	ctx.SetStrokeWidth(1.5)
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
	for _, np := range trajectories {
		var col color.Color

		switch np.wep {
		case common.EqMolotov, common.EqIncendiary:
			col = colorFireNade
		case common.EqHE:
			col = colorHE
		case common.EqFlash:
			col = colorFlash
		case common.EqSmoke:
			col = colorSmoke
		case common.EqDecoy:
			col = colorDecoy
		default:
			fmt.Println("Unknown grenade type", np.wep)
			continue
		}

		buildTrajPath := func() {
			x, y := curMap.TranslateScale(np.path[0].Position.X, np.path[0].Position.Y)
			ctx.MoveTo(x, h-y)
			for _, pos := range np.path[1:] {
				x, y := curMap.TranslateScale(pos.Position.X, pos.Position.Y)
				ctx.LineTo(x, h-y)
			}
		}

		// Outer glow pass
		ctx.SetStrokeColor(withAlpha(col, 55))
		ctx.SetStrokeWidth(5)
		buildTrajPath()
		ctx.Stroke()

		// Inner glow pass
		ctx.SetStrokeColor(withAlpha(col, 130))
		ctx.SetStrokeWidth(2)
		buildTrajPath()
		ctx.Stroke()

		// Crisp core line
		ctx.SetStrokeColor(withAlpha(col, 230))
		ctx.SetStrokeWidth(1)
		buildTrajPath()
		ctx.Stroke()

		// Dot at throw origin
		first := np.path[0]
		x0, y0 := curMap.TranslateScale(first.Position.X, first.Position.Y)
		ctx.SetFillColor(withAlpha(col, 200))
		ctx.SetStrokeColor(color.RGBA{0, 0, 0, 0})
		ctx.DrawPath(x0, h-y0, canvas.Circle(3))

		// Dot at landing point
		last := np.path[len(np.path)-1]
		x1, y1 := curMap.TranslateScale(last.Position.X, last.Position.Y)
		ctx.SetFillColor(withAlpha(col, 255))
		ctx.DrawPath(x1, h-y1, canvas.Circle(4))
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
