package demoinfocs

import (
	"reflect"
	"sync"
)

type EventHandler func(interface{})

type EventDispatcher interface {
	Register(reflect.Type, EventHandler)
}

type eventDispatcher struct {
	sync.RWMutex
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
			for _, h := range d.handlers[k] {
				d.cachedHandlers[handlerType] = append(d.cachedHandlers[handlerType], h)
			}
		}
	}
}

func (d *eventDispatcher) dispatchQueue(eventQueue chan interface{}) {
	for e := range eventQueue {
		d.dispatch(e)
	}
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
