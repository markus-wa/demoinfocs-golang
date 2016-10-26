package demoinfocs

import (
	"reflect"
	"sync"
)

// The contents of the event are undefined after the method returns
type EventHandler func(interface{})

type EventDispatcher interface {
	Register(reflect.Type, EventHandler)
}

type eventDispatcher struct {
	sync.RWMutex
	syncLock       sync.Mutex
	syncWg         sync.WaitGroup
	handlers       map[reflect.Type][]EventHandler
	cachedHandlers map[reflect.Type][]EventHandler
}

func (d *eventDispatcher) dispatch(event interface{}) {
	d.RLock()
	defer d.RUnlock()

	t := reflect.TypeOf(event)

	if d.cachedHandlers[t] == nil {
		// We'll need a write lock inside
		d.RUnlock()
		d.initCache(t)
		d.RLock()
	}

	for _, h := range d.cachedHandlers[t] {
		if h != nil {
			h(event)
		}
	}
}

// We cache the handlers so we don't have to check the type of each handler group for every event of the same type
// Performance gained >15%, depending on the amount of handlers
func (d *eventDispatcher) initCache(handlerType reflect.Type) {
	d.Lock()
	defer d.Unlock()

	// Read from nil map is allowed, so we initialize it only now if it's nil
	if d.cachedHandlers == nil {
		d.cachedHandlers = make(map[reflect.Type][]EventHandler)
	}

	// Load handlers into cache
	d.cachedHandlers[handlerType] = make([]EventHandler, 0)
	for k := range d.handlers {
		if handlerType.AssignableTo(k) {
			d.cachedHandlers[handlerType] = append(d.cachedHandlers[handlerType], d.handlers[k]...)
		}
	}
}

func (d *eventDispatcher) dispatchQueue(q chan interface{}) {
	for e := range q {
		if e == syncToken {
			d.syncWg.Done()
			continue
		}
		d.dispatch(e)
	}
}

func (d *eventDispatcher) addQueue(q chan interface{}) {
	go d.dispatchQueue(q)
}

var syncToken *struct{} = &struct{}{}

// Syncs the channels dispatch routine to the current go routine
// This ensures all events received up to this point will be handled before continuing
func (d *eventDispatcher) syncQueue(q chan interface{}) {
	// We can not check the channel length as that does not tell us whether the last event has been fully dispatched
	d.syncLock.Lock()
	defer d.syncLock.Unlock()
	// Using sync.Cond would be a race against dispatchQueue
	d.syncWg.Add(1)
	q <- syncToken
	d.syncWg.Wait()
}

func (d *eventDispatcher) Register(eventType reflect.Type, handler EventHandler) {
	d.Lock()
	defer d.Unlock()

	if d.handlers == nil {
		d.handlers = make(map[reflect.Type][]EventHandler)
	}
	d.handlers[eventType] = append(d.handlers[eventType], handler)

	// Add handler to cache if already initialized
	for k := range d.cachedHandlers {
		if eventType.AssignableTo(k) {
			d.cachedHandlers[k] = append(d.cachedHandlers[k], handler)
		}
	}
}
