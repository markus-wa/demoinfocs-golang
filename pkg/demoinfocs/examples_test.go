package demoinfocs_test

import (
	"log"
	"testing"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

/*
This will print all kills of a demo in the format '[[killer]] <[[weapon]] [(HS)] [(WB)]> [[victim]]'
*/
//noinspection GoUnhandledErrorResult
func ExampleParser() {
	onKill := func(kill events.Kill) {
		var hs string
		if kill.IsHeadshot {
			hs = " (HS)"
		}

		var wallBang string
		if kill.PenetratedObjects > 0 {
			wallBang = " (WB)"
		}

		log.Printf("%s <%v%s%s> %s\n", kill.Killer, kill.Weapon, hs, wallBang, kill.Victim)
	}

	err := demoinfocs.ParseFile("../../test/cs-demos/s2/s2.dem", func(p demoinfocs.Parser) error {
		p.RegisterEventHandler(onKill)

		return nil
	})
	if err != nil {
		log.Panic("failed to parse demo: ", err)
	}
}

func TestExamplesWithoutOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long running test")
	}

	ExampleParser()
}
