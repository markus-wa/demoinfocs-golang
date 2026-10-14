package mocking

import (
	"context"
	"errors"
	"reflect"
	"testing"

	dp "github.com/markus-wa/godispatch"
	assert "github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// The real parser must satisfy the interface, otherwise collectKills couldn't be used with it.
var _ parser = (*demoinfocs.Parser)(nil)

// fakeParser is a hand-written fake for the parser interface.
// ParseToEnd dispatches events to the registered handlers and then returns err.
type fakeParser struct {
	events   []any
	err      error
	handlers []any
}

func (p *fakeParser) RegisterEventHandler(handler any) dp.HandlerIdentifier {
	p.handlers = append(p.handlers, handler)
	id := len(p.handlers)

	return &id
}

func (p *fakeParser) ParseToEnd(context.Context) error {
	for _, e := range p.events {
		for _, h := range p.handlers {
			hv := reflect.ValueOf(h)
			if ev := reflect.ValueOf(e); ev.Type().AssignableTo(hv.Type().In(0)) {
				hv.Call([]reflect.Value{ev})
			}
		}
	}

	return p.err
}

func TestCollectKills(t *testing.T) {
	kill1 := kill(common.EqAK47)
	kill2 := kill(common.EqScout)
	kill3 := kill(common.EqAUG)
	p := &fakeParser{
		events: []any{kill1, events.RoundEnd{}, kill2, kill3},
	}

	actual, err := collectKills(t.Context(), p)

	assert.NoError(t, err)
	expected := []events.Kill{kill1, kill2, kill3}
	assert.Equal(t, expected, actual)
}

func kill(wep common.EquipmentType) events.Kill {
	eq := common.NewEquipment(wep, common.UniqueID(1))

	return events.Kill{
		Killer: new(common.Player),
		Weapon: eq,
		Victim: new(common.Player),
	}
}

func TestCollectKillsError(t *testing.T) {
	expectedErr := errors.New("test error")
	p := &fakeParser{err: expectedErr}

	kills, actualErr := collectKills(t.Context(), p)

	assert.Nil(t, kills)
	assert.Equal(t, expectedErr, actualErr)
}
