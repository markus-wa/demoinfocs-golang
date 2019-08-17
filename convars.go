package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

func (p *Parser) handleSetConVar(setConVar *msg.CNETMsg_SetConVar) {
	updated := make(map[string]string)
	for _, cvar := range setConVar.Convars.Cvars {
		updated[cvar.Name] = cvar.Value
		p.gameState.conVars[cvar.Name] = cvar.Value
	}

	p.eventDispatcher.Dispatch(events.ConVarsUpdated{
		UpdatedConVars: updated,
	})
}
