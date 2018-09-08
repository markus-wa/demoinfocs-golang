package main

import (
	"fmt"
	_ "image/jpeg"
	"log"
	"os"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
	ex "github.com/markus-wa/demoinfocs-golang/examples"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

// Run like this: go run entities.go -demo /path/to/demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := dem.NewParser(f)

	_, err = p.ParseHeader()
	checkError(err)

	p.RegisterEventHandler(func(events.DataTablesParsed) {
		p.ServerClasses().FindByName("CWeaponAWP").OnEntityCreated(func(ent *st.Entity) {
			ent.FindProperty("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
				x := p.GameState().Participants().FindByHandle(val.IntVal)
				if x != nil {
					var prev string
					prevHandle := ent.FindProperty("m_hPrevOwner").Value().IntVal
					prevPlayer := p.GameState().Participants().FindByHandle(prevHandle)
					if prevPlayer != nil {
						if prevHandle != val.IntVal {
							prev = prevPlayer.Name + "'s"
						} else {
							prev = "his dropped"
						}
					} else {
						prev = "a brand new"
					}
					fmt.Printf("%s picked up %s AWP (#%d)\n", x.Name, prev, ent.ID())
				}
			})
		})
	})

	err = p.ParseToEnd()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
