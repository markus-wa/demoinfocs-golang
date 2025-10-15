package main

import (
	"flag"
	"fmt"
	"os"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// Run like this: go run broadcasts.go -url "http://localhost:8080/<token>"
func main() {
	fl := new(flag.FlagSet)

	urlPtr := fl.String("url", "", "CSTV Broadcast URL")

	err := fl.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}

	url := *urlPtr

	if url == "" {
		fmt.Println("Please provide a CSTV Broadcast URL via -url")
		return
	}

	p, err := demoinfocs.NewCSTVBroadcastParser(url)
	checkError(err)

	p.RegisterNetMessageHandler(func(m *msg.CDemoFileHeader) {
		fmt.Println("Map:", m.GetMapName())
	})

	// Register handler on kill events
	p.RegisterEventHandler(func(e events.Kill) {
		var hs string
		if e.IsHeadshot {
			hs = " (HS)"
		}
		var wallBang string
		if e.PenetratedObjects > 0 {
			wallBang = " (WB)"
		}

		fmt.Printf("%s <%v%s%s> %s\n", formatPlayer(e.Killer), e.Weapon, hs, wallBang, formatPlayer(e.Victim))
	})

	// Register handler on round end to figure out who won
	p.RegisterEventHandler(func(e events.RoundEnd) {
		gs := p.GameState()
		switch e.Winner {
		case common.TeamTerrorists:
			// Winner's score + 1 because it hasn't actually been updated yet
			fmt.Printf("Round finished: winnerSide=T  ; score=%d:%d\n", gs.TeamTerrorists().Score(), gs.TeamCounterTerrorists().Score())
		case common.TeamCounterTerrorists:
			fmt.Printf("Round finished: winnerSide=CT ; score=%d:%d\n", gs.TeamCounterTerrorists().Score(), gs.TeamTerrorists().Score())
		default:
			// Probably match medic or something similar
			fmt.Println("Round finished: No winner (tie)")
		}
	})

	// Register handler for chat messages to print them
	p.RegisterEventHandler(func(e events.ChatMessage) {
		fmt.Printf("Chat - %s says: %s\n", formatPlayer(e.Sender), e.Text)
	})

	p.RegisterEventHandler(func(e events.RankUpdate) {
		fmt.Printf("Rank Update: %d went from rank %d to rank %d, change: %f\n", e.SteamID32, e.RankOld, e.RankNew, e.RankChange)
	})

	// Parse to end
	err = p.ParseToEnd()
	checkError(err)
}

func formatPlayer(p *common.Player) string {
	if p == nil {
		return "?"
	}

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
		panic(err)
	}
}
