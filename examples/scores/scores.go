package main

import (
	"fmt"
	"log"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

const defaultDemPath = "../../test/cs-demos/default.dem"

func main() {
	f, err := os.Open(defaultDemPath)
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}

	p := dem.NewParser(f)

	// Parse header
	h, err := p.ParseHeader()
	if err != nil {
		log.Fatal(err)
	}
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
	if err != nil {
		log.Fatal(err)
	}
}
