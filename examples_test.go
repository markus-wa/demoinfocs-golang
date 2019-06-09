package demoinfocs_test

import (
	"fmt"
	"os"
	"testing"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/events"
)

/*
This will print all kills of a demo in the format '[[killer]] <[[weapon]] [(HS)] [(WB)]> [[victim]]'
*/
func ExampleParser() {
	f, err := os.Open("cs-demos/default.dem")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	p := dem.NewParser(f)

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
		fmt.Printf("%s <%v%s%s> %s\n", e.Killer.Name, e.Weapon.Weapon, hs, wallBang, e.Victim.Name)
	})

	// Parse to end
	err = p.ParseToEnd()
	if err != nil {
		panic(err)
	}
}

func TestExamplesWithoutOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long running test")
	}
	ExampleParser()
}
