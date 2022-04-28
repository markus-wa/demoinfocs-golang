package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables/fake"
)

type DevNullReader struct {
}

func (DevNullReader) Read(p []byte) (n int, err error) {
	return len(p), nil
}

func TestParser_BindNewPlayer_Issue98(t *testing.T) {
	p := newParser()

	p.rawPlayers = map[int]*common.PlayerInfo{
		0: {
			UserID: 1,
			Name:   "Zim",
			GUID:   "BOT",
		},
		1: {
			UserID: 2,
			Name:   "The Suspect",
			GUID:   "123",
		},
	}

	bot := fakePlayerEntity(1)
	p.bindNewPlayer(bot)
	bot.Destroy()

	player := fakePlayerEntity(2)
	p.bindNewPlayer(player)

	assert.Len(t, p.GameState().Participants().Connected(), 1)
}

func TestParser_BindNewPlayer_Issue98_Reconnect(t *testing.T) {
	p := newParser()

	p.rawPlayers = map[int]*common.PlayerInfo{
		0: {
			UserID: 2,
			Name:   "The Suspect",
			GUID:   "123",
			XUID:   1,
		},
	}

	player := fakePlayerEntity(1)
	p.bindNewPlayer(player)
	player.Destroy()

	p.RegisterEventHandler(func(events.PlayerConnect) {
		t.Error("expected no more PlayerConnect events but got one")
	})
	p.bindNewPlayer(player)

	assert.Len(t, p.GameState().Participants().All(), 1)
}

func TestParser_BindNewPlayer_PlayerSpotted_Under32(t *testing.T) {
	testPlayerSpotted(t, "m_bSpottedByMask.000")
}

func TestParser_BindNewPlayer_PlayerSpotted_Over32(t *testing.T) {
	testPlayerSpotted(t, "m_bSpottedByMask.001")
}

func testPlayerSpotted(t *testing.T, propName string) {
	p := newParser()

	p.rawPlayers = map[int]*common.PlayerInfo{
		0: {
			UserID: 2,
			Name:   "Spotter",
			GUID:   "123",
			XUID:   1,
		},
	}

	// TODO: Player interface so we don't have to mock all this
	spotted := new(stfake.Entity)
	spottedByProp0 := new(stfake.Property)

	var spottedByUpdateHandler st.PropertyUpdateHandler
	spottedByProp0.On("OnUpdate", mock.Anything).Run(func(args mock.Arguments) {
		spottedByUpdateHandler = args.Get(0).(st.PropertyUpdateHandler)
	})

	spotted.On("Property", propName).Return(spottedByProp0)
	configurePlayerEntityMock(1, spotted)
	p.bindNewPlayer(spotted)

	var actual events.PlayerSpottersChanged
	p.RegisterEventHandler(func(e events.PlayerSpottersChanged) {
		actual = e
	})

	spottedByUpdateHandler(st.PropertyValue{IntVal: 1})

	expected := events.PlayerSpottersChanged{
		Spotted: p.gameState.playersByEntityID[1],
	}
	assert.NotNil(t, expected.Spotted)
	assert.Equal(t, expected, actual)
}

func newParser() *parser {
	p := NewParser(new(DevNullReader)).(*parser)
	p.header = &common.DemoHeader{}

	return p
}

func fakePlayerEntity(id int) *stfake.Entity {
	entity := new(stfake.Entity)
	configurePlayerEntityMock(id, entity)

	return entity
}

func configurePlayerEntityMock(id int, entity *stfake.Entity) {
	entity.On("ID").Return(id)

	var destroyCallback func()
	entity.On("OnDestroy", mock.Anything).Run(func(args mock.Arguments) {
		destroyCallback = args.Get(0).(func())
	})

	entity.On("OnPositionUpdate", mock.Anything).Return()
	prop := new(stfake.Property)
	prop.On("OnUpdate", mock.Anything).Return()
	entity.On("Property", mock.Anything).Return(prop)
	entity.On("BindProperty", mock.Anything, mock.Anything, mock.Anything)
	entity.On("Destroy").Run(func(mock.Arguments) {
		destroyCallback()
	})
}
