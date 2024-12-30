package main

import (
	"fmt"
	_ "image/jpeg"
	"os"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

// Run like this: go run entities.go -demo /path/to/demo.dem
func main() {
	f, err := os.Open(ex.DemoPathFromArgs())
	checkError(err)
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	p.RegisterEventHandler(func(events.DataTablesParsed) {
		p.ServerClasses().FindByName("CWeaponAWP").OnEntityCreated(func(ent st.Entity) {
			ent.Property("m_hOwnerEntity").OnUpdate(func(val st.PropertyValue) {
				x := p.GameState().Participants().FindByHandle64(val.S2UInt64())
				if x != nil {
					var prev string
					prevHandle := ent.Property("m_hPrevOwner").Value().S2UInt64()
					prevPlayer := p.GameState().Participants().FindByHandle64(prevHandle)
					if prevPlayer != nil {
						if prevHandle != val.S2UInt64() {
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
		panic(err)
	}
}
