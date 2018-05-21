package demoinfocs_test

import (
	"fmt"
	"os"
	"testing"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

// Make sure the example from the README.md compiles and runs.
func TestExample(t *testing.T) {
	f, err := os.Open(defaultDemPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := dem.NewParser(f, dem.WarnToStdErr)

	// Parse header
	h, err := p.ParseHeader()
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
}
