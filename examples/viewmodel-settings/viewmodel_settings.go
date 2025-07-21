package main

import (
	"fmt"
	"os"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

// Run like this: go run viewmodel_settings.go -demo /path/to/cs2-demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	if err != nil {
		panic(err)
	}

	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	// Register handler on round start to collect viewmodel settings
	p.RegisterEventHandler(func(e events.RoundStart) {
		fmt.Println("Player viewmodels:")
		gs := p.GameState()

		// Get all connected players
		players := gs.Participants().Playing()

		for _, player := range players {
			if player == nil {
				continue
			}

			// Get viewmodel settings
			offset := player.ViewmodelOffset()
			fov := player.ViewmodelFOV()

			fmt.Printf("%s: Viewmodel Offset=(%.1f, %.1f, %.1f), FOV=%.1f\n",
				player.Name, offset.X, offset.Y, offset.Z, fov)
		}
		fmt.Println() // Empty line for readability
	})

	// Parse to end
	err = p.ParseToEnd()
	if err != nil {
		panic(err)
	}
}
