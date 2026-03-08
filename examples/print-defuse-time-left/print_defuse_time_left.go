package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

// Run like this: go run print_defuse_time_left.go -demo /path/to/demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)

	defer f.Close()

	p := demoinfocs.NewParser(f)

	defer p.Close()

	type Plant struct {
		time   time.Duration
		round  int
		player *common.Player
		site   events.Bombsite
	}

	var lastPlant *Plant
	round := 0
	bombTimer := 40 * time.Second

	p.RegisterEventHandler(func(e events.MatchStart) {
		customBombTimer, err := p.GameState().Rules().BombTime()
		if err == nil {
			bombTimer = customBombTimer
		}
	})

	p.RegisterEventHandler(func(e events.RoundStart) {
		round++
	})

	p.RegisterEventHandler(func(e events.BombPlanted) {
		lastPlant = &Plant{
			time:   p.CurrentTime(),
			round:  round,
			player: e.Player,
			site:   e.Site,
		}
	})

	p.RegisterEventHandler(func(e events.BombDefused) {

		if lastPlant == nil {
			fmt.Fprintf(os.Stderr, "Internal error, defuse event before any plant event at round %d\n", round)
			return
		}

		if round != lastPlant.round {
			fmt.Fprintf(os.Stderr, "Internal error, last plant was round %d but defuse is round %d\n",
				lastPlant.round, round)
			return
		}

		if e.Site != lastPlant.site {
			fmt.Fprintf(os.Stderr, "Internal error, last plant was site %s but defuse is site %s\n",
				string(lastPlant.site), string(e.Site))
			return
		}

		currentTime := p.CurrentTime()

		bomb_remaining_time := bombTimer - (currentTime - lastPlant.time)

		fmt.Println("> Round", round, "/ Site", string(lastPlant.site))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "Bomb planted by %s\t [%v]\n", lastPlant.player, lastPlant.time)
		fmt.Fprintf(w, "Bomb defused by %s\t [%v]\n", e.Player, currentTime)
		w.Flush()
		fmt.Printf("Time remaining on the bomb: %.2f seconds\n", bomb_remaining_time.Seconds())
		fmt.Println("")

	})

	// Parse to end
	err = p.ParseToEnd()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
