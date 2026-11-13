package demoinfocs

import (
	dp "github.com/markus-wa/godispatch"

	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// stateBinder binds the parser's game-state to the entities of the demo.
//
// The binding rules used to be methods of Parser, so testing a single rule meant constructing a
// whole Parser (net-message dispatcher, send-tables parser, header, ...). With this seam a family
// needs only the deps below, so it can be exercised on a bare GameState and synthetic entities.
//
// Families (rules, players, weapons/grenades, inferno, bomb, ...) are method groups on this one
// type, not separate adapters: they share the deps and reach into each other while binding. The
// Parser keeps the bindEntities() fan-out until the last family has moved.
type stateBinder struct {
	deps stateBinderDeps
}

// stateBinderDeps is the concrete set of collaborators the binder's families use.
//
// A struct, not an interface: there is exactly one adapter (the Parser) and no second one has
// appeared. Grow it per family as the families move over.
type stateBinderDeps struct {
	gameState       *GameState
	eventDispatcher *dp.Dispatcher

	// serverClasses is read when a family binds rather than snapshotted up front: the send-tables
	// parser fills its class maps while frames parse, and tests replace the parser on the Parser.
	serverClasses func() st.ServerClasses

	disableMimicSource1GameEvents bool

	// clearGrenadeProjectiles drops the tracked grenade projectiles at a round boundary.
	clearGrenadeProjectiles func()
}

func newStateBinder(deps stateBinderDeps) stateBinder {
	return stateBinder{deps: deps}
}
