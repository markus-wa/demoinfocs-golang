// Package fake provides basic mocks for IParser, IGameState and IParticipants.
// See examples/mocking (https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/mocking).
package fake

import (
	"time"

	dp "github.com/markus-wa/godispatch"
	mock "github.com/stretchr/testify/mock"

	dem "github.com/markus-wa/demoinfocs-golang"
	common "github.com/markus-wa/demoinfocs-golang/common"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

var _ dem.IParser = new(Parser)

// Parser is a mock for of demoinfocs.IParser.
type Parser struct {
	mock.Mock

	// List of events to be dispatched by frame.
	// ParseToEnd() / ParseNextFrame() will dispatch them accordingly.
	// See also: MockEvents() / MockEventsFrame()
	Events map[int][]interface{}

	// List of net-messages to be dispatched by frame.
	// ParseToEnd() / ParseNextFrame() will dispatch them accordingly.
	// See also: MockNetMessages() / MockNetMessagesFrame()
	NetMessages map[int][]interface{}

	eventDispatcher dp.Dispatcher
	msgDispatcher   dp.Dispatcher
	currentFrame    int
	mockFrame       int
}

// MockEvents adds entries to Parser.Events.
// It increments an internal frame-index so each set of events and net-messages added
// in subsequent calls to this or MockNetMessages() is triggered on a separate frame.
//
// See also: MockEventsFrame()
func (p *Parser) MockEvents(events ...interface{}) {
	p.MockEventsFrame(p.mockFrame, events...)
	p.mockFrame++
}

// MockEventsFrame adds entries to Events that will be dispatched at the frame indicated by the first parameter.
//
// See also: MockEvents()
func (p *Parser) MockEventsFrame(frame int, events ...interface{}) {
	p.Events[frame] = append(p.Events[frame], events...)
}

// MockNetMessages adds entries to Parser.NetMessages.
// It increments an internal frame-index so each set of net-messages and events added
// in subsequent calls to this or MockEvents() is triggered on a separate frame.
//
// See also: MockNetMessagesFrame()
func (p *Parser) MockNetMessages(messages ...interface{}) {
	p.MockNetMessagesFrame(p.mockFrame, messages...)
	p.mockFrame++
}

// MockNetMessagesFrame adds entries to NetMessages that will be dispatched at the frame indicated by the first parameter.
//
// See also: MockNetMessages()
func (p *Parser) MockNetMessagesFrame(frame int, messages ...interface{}) {
	p.NetMessages[frame] = append(p.NetMessages[frame], messages...)
}

// NewParser returns a new parser mock with pre-initialized Events and NetMessages.
// Pre-mocks RegisterEventHandler() and RegisterNetMessageHandler().
func NewParser() *Parser {
	p := &Parser{
		Events:      make(map[int][]interface{}),
		NetMessages: make(map[int][]interface{}),
	}

	p.On("RegisterEventHandler").Return()
	p.On("RegisterNetMessageHandler").Return()

	return p
}

// ServerClasses is a mock-implementation of IParser.ServerClasses().
//
// Unfortunately sendtables.ServerClasses currently isn't mockable.
func (p *Parser) ServerClasses() st.ServerClasses {
	return p.Called().Get(0).(st.ServerClasses)
}

// Header is a mock-implementation of IParser.Header().
func (p *Parser) Header() common.DemoHeader {
	return p.Called().Get(0).(common.DemoHeader)
}

// GameState is a mock-implementation of IParser.GameState().
func (p *Parser) GameState() dem.IGameState {
	return p.Called().Get(0).(dem.IGameState)
}

// CurrentFrame is a mock-implementation of IParser.CurrentFrame().
func (p *Parser) CurrentFrame() int {
	return p.Called().Int(0)
}

// CurrentTime is a mock-implementation of IParser.CurrentTime().
func (p *Parser) CurrentTime() time.Duration {
	return p.Called().Get(0).(time.Duration)
}

// Progress is a mock-implementation of IParser.Progress().
func (p *Parser) Progress() float32 {
	return p.Called().Get(0).(float32)
}

// RegisterEventHandler is a mock-implementation of IParser.RegisterEventHandler().
// Return HandlerIdentifier cannot be mocked (for now).
func (p *Parser) RegisterEventHandler(handler interface{}) dp.HandlerIdentifier {
	p.Called()
	return p.eventDispatcher.RegisterHandler(handler)
}

// UnregisterEventHandler is a mock-implementation of IParser.UnregisterEventHandler().
func (p *Parser) UnregisterEventHandler(identifier dp.HandlerIdentifier) {
	p.Called()
	p.eventDispatcher.UnregisterHandler(identifier)
}

// RegisterNetMessageHandler is a mock-implementation of IParser.RegisterNetMessageHandler().
// Return HandlerIdentifier cannot be mocked (for now).
func (p *Parser) RegisterNetMessageHandler(handler interface{}) dp.HandlerIdentifier {
	p.Called()
	return p.msgDispatcher.RegisterHandler(handler)
}

// UnregisterNetMessageHandler is a mock-implementation of IParser.UnregisterNetMessageHandler().
func (p *Parser) UnregisterNetMessageHandler(identifier dp.HandlerIdentifier) {
	p.Called()
	p.msgDispatcher.UnregisterHandler(identifier)
}

// ParseHeader is a mock-implementation of IParser.ParseHeader().
func (p *Parser) ParseHeader() (common.DemoHeader, error) {
	args := p.Called()
	return args.Get(0).(common.DemoHeader), args.Error(1)
}

// ParseToEnd is a mock-implementation of IParser.ParseToEnd().
//
// Dispatches Parser.Events and Parser.NetMessages in the specified order.
//
// Returns the mocked error value.
func (p *Parser) ParseToEnd() (err error) {
	args := p.Called()

	maxFrame := max(p.Events)
	maxNetMessageFrame := max(p.NetMessages)
	if maxFrame < maxNetMessageFrame {
		maxFrame = maxNetMessageFrame
	}

	for p.currentFrame <= maxFrame {
		p.parseNextFrame()
	}

	return args.Error(0)
}

func (p *Parser) parseNextFrame() {
	events, ok := p.Events[p.currentFrame]
	if ok {
		for _, e := range events {
			p.eventDispatcher.Dispatch(e)
		}
	}

	messages, ok := p.NetMessages[p.currentFrame]
	if ok {
		for _, msg := range messages {
			p.msgDispatcher.Dispatch(msg)
		}
	}

	p.currentFrame++
}

// ParseNextFrame is a mock-implementation of IParser.ParseNextFrame().
//
// Dispatches Parser.Events and Parser.NetMessages in the specified order.
//
// Returns the mocked bool and error values.
func (p *Parser) ParseNextFrame() (b bool, err error) {
	args := p.Called()

	p.parseNextFrame()

	return args.Bool(0), args.Error(1)
}

func max(numbers map[int][]interface{}) (maxNumber int) {
	for maxNumber = range numbers {
		break
	}
	for n := range numbers {
		if n > maxNumber {
			maxNumber = n
		}
	}
	return
}

// Cancel is a mock-implementation of IParser.Cancel().
// Does not cancel the mock's ParseToEnd() function,
// mock the return value of ParseToEnd() to be ErrCancelled instead.
func (p *Parser) Cancel() {
	p.Called()
}
