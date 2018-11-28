package mocking

import (
	"errors"
	"testing"

	assert "github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
	fake "github.com/markus-wa/demoinfocs-golang/fake"
)

func TestCollectKills(t *testing.T) {
	parser := fake.NewParser()
	kill1 := kill(common.EqAK47)
	kill2 := kill(common.EqScout)
	kill3 := kill(common.EqAUG)
	parser.MockEvents(kill1)        // First frame
	parser.MockEvents(kill2, kill3) // Second frame

	parser.On("ParseToEnd").Return(nil) // Return no error

	actual, err := collectKills(parser)

	assert.Nil(t, err)
	expected := []events.Kill{kill1, kill2, kill3}
	assert.Equal(t, expected, actual)
}

func kill(wep common.EquipmentElement) events.Kill {
	eq := common.NewEquipment(wep)
	return events.Kill{
		Killer: new(common.Player),
		Weapon: &eq,
		Victim: new(common.Player),
	}
}

func TestCollectKillsError(t *testing.T) {
	parser := fake.NewParser()
	expectedErr := errors.New("Test error")
	parser.On("ParseToEnd").Return(expectedErr)

	kills, actualErr := collectKills(parser)

	assert.Nil(t, kills)
	assert.Equal(t, expectedErr, actualErr)
}
