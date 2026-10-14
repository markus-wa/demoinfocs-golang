package demoinfocs

import (
	"github.com/golang/geo/r3"
	dp "github.com/markus-wa/godispatch"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stbridge"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// This file contains the only places where the parser registers its own handlers.
// Everything registered through the public methods (RegisterNetMessageHandler, RegisterEventHandler,
// the send-table callbacks) is user code, and its panics are tagged as such (see internal/panics).
// Parser-internal code must use the helpers below instead (enforced by forbidigo, see .golangci.yml).

// registerInternal registers one of the parser's own net-message handlers.
//
// Its panics propagate unwrapped (not as dispatch.ConsumerCodePanic), so they aren't mistaken for user-code panics.
func registerInternal[T any](p *Parser, h func(T)) {
	p.msgDispatcher.RegisterHandlerWithOptions(h, dp.HandlerOptions{UnwrappedPanics: true})
}

// The st* helpers register the parser's own entity callbacks.
// They work like the methods of the same name, but don't tag the callbacks' panics as user-code panics.

// stOnUpdate is prop.OnUpdate(h) for the parser's own handlers.
func stOnUpdate(prop *st.Property, h st.PropertyUpdateHandler) {
	stbridge.OnUpdate(prop, h)
}

// stOnEntityCreated is class.OnEntityCreated(h) for the parser's own handlers.
func stOnEntityCreated(class *st.ServerClass, h st.EntityCreatedHandler) {
	stbridge.OnEntityCreated(class, h)
}

// stBindProperty is entity.BindProperty(prop, variable, t) for the parser's own bindings.
func stBindProperty(entity *st.Entity, prop string, variable any, t st.PropertyValueType) {
	stbridge.BindProperty(entity, prop, variable, t)
}

// stOnPositionUpdate is entity.OnPositionUpdate(h) for the parser's own handlers.
func stOnPositionUpdate(entity *st.Entity, h func(pos r3.Vector)) {
	stbridge.OnPositionUpdate(entity, h)
}

// stOnDestroy is entity.OnDestroy(delegate) for the parser's own delegates.
func stOnDestroy(entity *st.Entity, delegate func()) {
	stbridge.OnDestroy(entity, delegate)
}

// stOnCreateFinished is entity.OnCreateFinished(delegate) for the parser's own delegates.
func stOnCreateFinished(entity *st.Entity, delegate func()) {
	stbridge.OnCreateFinished(entity, delegate)
}

// stOnUpdateFinished is entity.OnUpdateFinished(delegate) for the parser's own delegates.
func stOnUpdateFinished(entity *st.Entity, delegate func()) {
	stbridge.OnUpdateFinished(entity, delegate)
}
