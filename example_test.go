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

	p := dem.NewParser(f)

	// Parse header
	p.ParseHeader()

	// Get T / CT team state references (contains scores)
	tState := p.TState()
	ctState := p.CTState()

	// We need this to skip restarts and team switches before the match start
	// Might not be necessary for all demos (especially MM)
	// But for pro matches / scrims it might be depending on how the server was set up
	// TODO: This might not always be correct, needs testing
	matchStarted := false
	p.RegisterEventHandler(func(events.MatchStartedEvent) {
		matchStarted = true
	})
	matchReallyStarted := false
	p.RegisterEventHandler(func(events.RoundStartedEvent) {
		if matchStarted {
			matchReallyStarted = true
		}
	})

	// Register handler on round end to figure out who won
	p.RegisterEventHandler(func(e events.RoundEndedEvent) {
		if matchReallyStarted {
			if e.Winner == common.TeamTerrorists {
				fmt.Println("T-side won the round - score:", tState.Score()+1) // Score + 1 because it hasn't actually been updated yet
			} else if e.Winner == common.TeamCounterTerrorists {
				fmt.Println("CT-side won the round - score:", ctState.Score()+1)
			} else {
				fmt.Println("Apparently neither the Ts nor CTs won the round, interesting")
			}
		} else {
			fmt.Println("Skipping warmup event")
		}
	})

	// Parse to end
	err = p.ParseToEnd()
	if err != nil {
		t.Fatal(err)
	}
}
