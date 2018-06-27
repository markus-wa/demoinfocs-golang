package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"

	"github.com/golang/geo/r3"

	draw2dimg "github.com/llgcode/draw2d/draw2dimg"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// Run like this: go run bouncynades.go -demo /path/to/demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := dem.NewParser(f)

	_, err = p.ParseHeader()
	checkError(err)

	stp := p.SendTableParser()

	// Initialize the graphic context on an RGBA image
	dest := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	gc := draw2dimg.NewGraphicContext(dest)

	// Set some properties
	gc.SetFillColor(color.RGBA{0x00, 0x00, 0x00, 0x00})
	gc.SetStrokeColor(color.RGBA{0xff, 0x00, 0x00, 0xff})
	gc.SetLineWidth(2)

	lastBouncePositionsID := make(map[int]r3.Vector)
	var lastEntityID int

	first := true
	stopped := false

	drawNadeLine := func(entityID int, pos r3.Vector) {
		if stopped {
			return
		}

		lastPos, ok := lastBouncePositionsID[entityID]

		x := (2000 + pos.X) / 5
		y := (2000 - pos.Y) / 5
		fmt.Println(entityID, pos.X, pos.Y, x, y)

		if !ok {
			if first {
				first = false
			} else {
				gc.FillStroke()
			}

			gc.BeginPath()  // Initialize a new path
			gc.MoveTo(x, y) // Move to a position to start the new path
		} else {
			if lastEntityID != entityID {
				gc.FillStroke()

				gc.BeginPath()                  // Initialize a new path
				gc.MoveTo(lastPos.X, lastPos.Y) // Move to a position to start the new path
			}
			gc.LineTo(x, y)
		}

		lastEntityID = entityID
		lastBouncePositionsID[entityID] = r3.Vector{X: x, Y: y}
	}

	p.RegisterEventHandler(func(events.RoundEndedEvent) {
		stopped = true
		p.Cancel()
	})

	p.RegisterEventHandler(func(e events.NadeEventIf) {
		ne := e.Base()
		drawNadeLine(ne.NadeEntityID, ne.Position)
		delete(lastBouncePositionsID, ne.NadeEntityID)
		if lastEntityID == ne.NadeEntityID {
			lastEntityID = -1
		}
	})

	p.RegisterEventHandler(func(events.DataTablesParsedEvent) {
		projectileTypes := []string{"CBaseCSGrenadeProjectile", "CDecoyProjectile", "CMolotovProjectile", "CSmokeGrenadeProjectile"}

		for _, pt := range projectileTypes {
			stp.FindServerClassByName(pt).RegisterEntityCreatedHandler(func(e st.EntityCreatedEvent) {
				ent := e.Entity
				prop := ent.FindProperty("m_nBounces")
				drawNadeLine(ent.ID, ent.Position())
				prop.RegisterPropertyUpdateHandler(func(st.PropValue) {
					drawNadeLine(ent.ID, ent.Position())
				})
			})
		}
	})

	err = p.ParseToEnd()

	gc.FillStroke()

	// Save to file
	draw2dimg.SaveToPngFile("bouncynades.png", dest)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
