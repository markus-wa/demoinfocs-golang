// Package fake provides basic mocks for Parser, GameState and Participants.
// See examples/mocking (https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/mocking).
package fake

import (
	"time"

	dp "github.com/markus-wa/godispatch"
	mock "github.com/stretchr/testify/mock"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/sendtables"
)

var _ demoinfocs.Parser = new(Parser)

// Parser is a mock for of demoinfocs.Parser.
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

// ServerClasses is a mock-implementation of Parser.ServerClasses().
//
// Unfortunately sendtables.ServerClasses currently isn't mockable.
func (p *Parser) ServerClasses() st.ServerClasses {
	return p.Called().Get(0).(st.ServerClasses)
}

// Header is a mock-implementation of Parser.Header().
func (p *Parser) Header() common.DemoHeader {
	return p.Called().Get(0).(common.DemoHeader)
}

// GameState is a mock-implementation of Parser.GameState().
func (p *Parser) GameState() demoinfocs.GameState {
	return p.Called().Get(0).(demoinfocs.GameState)
}

// CurrentFrame is a mock-implementation of Parser.CurrentFrame().
func (p *Parser) CurrentFrame() int {
	return p.Called().Int(0)
}

// CurrentTime is a mock-implementation of Parser.CurrentTime().
func (p *Parser) CurrentTime() time.Duration {
	return p.Called().Get(0).(time.Duration)
}

// TickRate is a mock-implementation of Parser.TickRate().
func (p *Parser) TickRate() float64 {
	return p.Called().Get(0).(float64)
}

// TickTime is a mock-implementation of Parser.TickTime().
func (p *Parser) TickTime() time.Duration {
	return p.Called().Get(0).(time.Duration)
}

// Progress is a mock-implementation of Parser.Progress().
func (p *Parser) Progress() float32 {
	return p.Called().Get(0).(float32)
}

// RegisterEventHandler is a mock-implementation of Parser.RegisterEventHandler().
// Return HandlerIdentifier cannot be mocked (for now).
func (p *Parser) RegisterEventHandler(handler interface{}) dp.HandlerIdentifier {
	p.Called()
	return p.eventDispatcher.RegisterHandler(handler)
}

// UnregisterEventHandler is a mock-implementation of Parser.UnregisterEventHandler().
func (p *Parser) UnregisterEventHandler(identifier dp.HandlerIdentifier) {
	p.Called()
	p.eventDispatcher.UnregisterHandler(identifier)
}

// RegisterNetMessageHandler is a mock-implementation of Parser.RegisterNetMessageHandler().
// Return HandlerIdentifier cannot be mocked (for now).
func (p *Parser) RegisterNetMessageHandler(handler interface{}) dp.HandlerIdentifier {
	p.Called()
	return p.msgDispatcher.RegisterHandler(handler)
}

// UnregisterNetMessageHandler is a mock-implementation of Parser.UnregisterNetMessageHandler().
func (p *Parser) UnregisterNetMessageHandler(identifier dp.HandlerIdentifier) {
	p.Called()
	p.msgDispatcher.UnregisterHandler(identifier)
}

// ParseHeader is a mock-implementation of Parser.ParseHeader().
func (p *Parser) ParseHeader() (common.DemoHeader, error) {
	args := p.Called()
	return args.Get(0).(common.DemoHeader), args.Error(1)
}

// ParseToEnd is a mock-implementation of Parser.ParseToEnd().
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

// ParseNextFrame is a mock-implementation of Parser.ParseNextFrame().
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
	for n := range numbers {
		if n > maxNumber {
			maxNumber = n
		}
	}

	return
}

// Cancel is a mock-implementation of Parser.Cancel().
// Does not cancel the mock's ParseToEnd() function,
// mock the return value of ParseToEnd() to be ErrCancelled instead.
func (p *Parser) Cancel() {
	p.Called()
}

// Close is a mock-implementation of Parser.Close().
// NOP implementation.
func (p *Parser) Close() error {
	return p.Called().Error(0)
}
