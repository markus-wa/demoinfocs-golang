package main

import (
	"fmt"
	"log"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
)

// Run like this: go run scores.go -demo /path/to/demo.dem > out.png
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	defer f.Close()
	checkError(err)

	p := dem.NewParser(f)

	// Parse header
	h, err := p.ParseHeader()
	checkError(err)
	fmt.Println("Map:", h.MapName)

	// Register handler on round end to figure out who won
	p.RegisterEventHandler(func(e events.RoundEndedEvent) {
		gs := p.GameState()
		switch e.Winner {
		case common.TeamTerrorists:
			// Winner's score + 1 because it hasn't actually been updated yet
			fmt.Printf("Round finished: winnerSide=T  ; score=%d:%d\n", gs.TState().Score()+1, gs.CTState().Score())
		case common.TeamCounterTerrorists:
			fmt.Printf("Round finished: winnerSide=CT ; score=%d:%d\n", gs.CTState().Score()+1, gs.TState().Score())
		default:
			// Probably match medic or something similar
			fmt.Println("Round finished: No winner (tie)")
		}
	})

	// Parse to end
	err = p.ParseToEnd()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
