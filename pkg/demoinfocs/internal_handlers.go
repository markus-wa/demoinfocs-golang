package demoinfocs

import (
	dp "github.com/markus-wa/godispatch"
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
