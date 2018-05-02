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
	defer f.Close()
	if err != nil {
		t.Fatal(err)
	}

	p := dem.NewParser(f, dem.WarnToStdErr)

	// Parse header
	h, err := p.ParseHeader()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Map:", h.MapName)

	// Register handler on round end to figure out who won
	p.RegisterEventHandler(func(e events.RoundEndedEvent) {
		switch e.Winner {
		case common.TeamTerrorists:
			fmt.Println("T-side won the round - score:", p.GameState().TState().Score()+1) // Score + 1 because it hasn't actually been updated yet
		case common.TeamCounterTerrorists:
			fmt.Println("CT-side won the round - score:", p.GameState().CTState().Score()+1)
		default:
			fmt.Println("Apparently neither the Ts nor CTs won the round, interesting")
		}
	})

	// Parse to end
	err = p.ParseToEnd()
	if err != nil {
		t.Fatal(err)
	}
}
