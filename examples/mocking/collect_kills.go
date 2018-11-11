package mocking

import (
	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

func collectKills(parser dem.IParser) (kills []events.Kill, err error) {
	parser.RegisterEventHandler(func(kill events.Kill) {
		kills = append(kills, kill)
	})
	err = parser.ParseToEnd()
	return
}
