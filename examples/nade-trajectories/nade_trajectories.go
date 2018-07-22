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
	s2 "github.com/golang/geo/s2"
	draw2dimg "github.com/llgcode/draw2d/draw2dimg"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
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

type inferno []r3.Vector

func (inf inferno) convexHull() *s2.Loop {
	q := s2.NewConvexHullQuery()
	for i := range inf {
		q.AddPoint(s2.Point{Vector: inf[i]})
	}
	return q.ConvexHull()
}

// Run like this: go run nade_trajectories.go -demo /path/to/demo.dem > nade_trajectories.jpg
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := dem.NewParser(f)

	_, err = p.ParseHeader()
	checkError(err)

	nadeTrajectories := make(map[int64]*nadePath) // Trajectories of all destroyed nades

	p.RegisterEventHandler(func(e events.NadeProjectileDestroyedEvent) {
		id := e.Projectile.UniqueID()
		if nadeTrajectories[id] == nil {
			nadeTrajectories[id] = &nadePath{
				wep:  e.Projectile.Weapon,
				team: e.Projectile.Thrower.Team,
			}
		}

		nadeTrajectories[id].path = e.Projectile.Trajectory
	})

	var infernos []*s2.Loop

	p.RegisterEventHandler(func(events.DataTablesParsedEvent) {
		p.ServerClasses().FindByName("CInferno").OnEntityCreated(func(ent *st.Entity) {
			origin := ent.Position()
			var fires inferno
			var nFires int
			ent.FindProperty("m_fireCount").OnUpdate(func(val st.PropertyValue) {
				for i := nFires; i < val.IntVal; i++ {
					iStr := fmt.Sprintf("%03d", i)
					offset := r3.Vector{
						X: float64(ent.FindProperty("m_fireXDelta." + iStr).Value().IntVal),
						Y: float64(ent.FindProperty("m_fireYDelta." + iStr).Value().IntVal),
						Z: float64(ent.FindProperty("m_fireZDelta." + iStr).Value().IntVal),
					}
					fires = append(fires, origin.Add(offset))
				}
				nFires = val.IntVal
			})

			ent.OnDestroy(func() {
				infernos = append(infernos, fires.convexHull())
			})
		})
	})

	var nadeTrajectoriesFirst5Rounds []*nadePath
	var infernosFirst5Rounds []*s2.Loop
	round := 0
	p.RegisterEventHandler(func(events.RoundEndedEvent) {
		round++
		// We only want the data from the first 5 rounds so the image is not too cluttered
		// This is a very cheap way to do it. Won't work with demos that have match-restarts etc.
		if round == 5 {
			// Copy all nade paths
			for _, np := range nadeTrajectories {
				nadeTrajectoriesFirst5Rounds = append(nadeTrajectoriesFirst5Rounds, np)
			}
			nadeTrajectories = make(map[int64]*nadePath)

			// And infernos
			for _, hull := range infernos {
				infernosFirst5Rounds = append(infernosFirst5Rounds, hull)
			}
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

	// Draw infernos first so they're in the background
	drawInfernos(gc, infernosFirst5Rounds)

	// Then trajectories on top of everything
	drawTrajectories(gc, nadeTrajectoriesFirst5Rounds)

	// Write to standard output
	jpeg.Encode(os.Stdout, dest, &jpeg.Options{
		Quality: 90,
	})
}

func drawInfernos(gc *draw2dimg.GraphicContext, hulls []*s2.Loop) {
	// Draw areas first
	gc.SetFillColor(colorInferno)

	for _, hull := range hulls {
		buildInfernoPath(gc, hull)
		gc.Fill()
	}

	// Then the outline
	gc.SetStrokeColor(colorInfernoHull)
	gc.SetLineWidth(2) // 2 px wide

	for _, hull := range hulls {
		buildInfernoPath(gc, hull)
		gc.FillStroke()
	}
}

func buildInfernoPath(gc *draw2dimg.GraphicContext, hull *s2.Loop) {
	vertices := hull.Vertices()
	gc.MoveTo(translateX(vertices[0].X), translateY(vertices[0].Y))
	for _, fire := range vertices[1:] {
		gc.LineTo(translateX(fire.X), translateY(fire.Y))
	}
	gc.LineTo(translateX(vertices[0].X), translateY(vertices[0].Y))
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
		gc.MoveTo(translateX(np.path[0].X), translateY(np.path[0].Y)) // Move to a position to start the new path

		for _, pos := range np.path[1:] {
			gc.LineTo(translateX(pos.X), translateY(pos.Y))
		}

		gc.FillStroke()
	}
}

// Rough translations for x & y coordinates from de_cache to 1024x1024 px.
// This could be done nicer by only having to provide the mapping between two source & target coordinates and the max size.
// Then we could calculate the correct stretch & offset automatically.

func translateX(x float64) float64 {
	const (
		stretch = 0.18
		offset  = 414
	)

	return x*stretch + offset
}

func translateY(y float64) float64 {
	const (
		stretch = -0.18
		offset  = 630
	)

	return y*stretch + offset
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
