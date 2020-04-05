package mocking

import (
	"github.com/markus-wa/demoinfocs-golang/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/pkg/demoinfocs/events"
)

func collectKills(parser demoinfocs.IParser) (kills []events.Kill, err error) {
	parser.RegisterEventHandler(func(kill events.Kill) {
		kills = append(kills, kill)
	})
	err = parser.ParseToEnd()
	return
}
