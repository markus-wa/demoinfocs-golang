package mocking

import (
	"context"

	dp "github.com/markus-wa/godispatch"

	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// parser is the subset of *demoinfocs.Parser that collectKills needs.
// Declaring it here (on the consumer side) lets tests pass a fake instead of a real parser.
type parser interface {
	RegisterEventHandler(handler any) dp.HandlerIdentifier
	ParseToEnd(ctx context.Context) error
}

func collectKills(ctx context.Context, p parser) (kills []events.Kill, err error) {
	p.RegisterEventHandler(func(kill events.Kill) {
		kills = append(kills, kill)
	})
	err = p.ParseToEnd(ctx)

	return
}
