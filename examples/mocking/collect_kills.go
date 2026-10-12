package mocking

import (
	"context"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

func collectKills(ctx context.Context, parser demoinfocs.Parser) (kills []events.Kill, err error) {
	parser.RegisterEventHandler(func(kill events.Kill) {
		kills = append(kills, kill)
	})
	err = parser.ParseToEnd(ctx)

	return
}
