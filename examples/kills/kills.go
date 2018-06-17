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

// Run like this: go run kills.go -demo /path/to/demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	defer f.Close()
	checkError(err)

	p := dem.NewParser(f)

	// Parse header
	_, err = p.ParseHeader()
	checkError(err)

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
