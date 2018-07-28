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

// Run like this: go run print_events.go -demo /path/to/demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	defer f.Close()
	checkError(err)

	p := dem.NewParser(f)

	// Parse header
	h, err := p.ParseHeader()
	checkError(err)
	fmt.Println("Map:", h.MapName)

	// Register handler on kill events
	p.RegisterEventHandler(func(e events.PlayerKilledEvent) {
		var hs string
		if e.IsHeadshot {
			hs = " (HS)"
		}
		var wallBang string
		if e.PenetratedObjects > 0 {
			wallBang = " (WB)"
		}
		fmt.Printf("%s <%v%s%s> %s\n", formatPlayer(e.Killer), e.Weapon.Weapon, hs, wallBang, formatPlayer(e.Victim))
	})

	// Register handler on round end to figure out who won
	p.RegisterEventHandler(func(e events.RoundEndedEvent) {
		gs := p.GameState()
		switch e.Winner {
		case common.TeamTerrorists:
			// Winner's score + 1 because it hasn't actually been updated yet
			fmt.Printf("Round finished: winnerSide=T  ; score=%d:%d\n", gs.TeamTerrorists().Score()+1, gs.TeamCounterTerrorists().Score())
		case common.TeamCounterTerrorists:
			fmt.Printf("Round finished: winnerSide=CT ; score=%d:%d\n", gs.TeamCounterTerrorists().Score()+1, gs.TeamTerrorists().Score())
		default:
			// Probably match medic or something similar
			fmt.Println("Round finished: No winner (tie)")
		}
	})

	// Register handler for chat messages to print them
	p.RegisterEventHandler(func(e events.ChatMessageEvent) {
		fmt.Printf("Chat - %s says: %s\n", formatPlayer(e.Sender), e.Text)
	})

	// Parse to end
	err = p.ParseToEnd()
	checkError(err)
}

func formatPlayer(p *common.Player) string {
	switch p.Team {
	case common.TeamTerrorists:
		return "[T]" + p.Name
	case common.TeamCounterTerrorists:
		return "[CT]" + p.Name
	}
	return p.Name
}
func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
