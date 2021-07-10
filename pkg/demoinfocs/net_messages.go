package demoinfocs

import (
	"bytes"

	bit "github.com/markus-wa/demoinfocs-golang/v2/internal/bitread"
	events "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
	msg "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/msg"
)

func (p *parser) handlePacketEntities(pe *msg.CSVCMsg_PacketEntities) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	r := bit.NewSmallBitReader(bytes.NewReader(pe.EntityData))

	currentEntity := -1
	for i := 0; i < int(pe.UpdatedEntries); i++ {
		currentEntity += 1 + int(r.ReadUBitInt())

		cmd := r.ReadBitsToByte(2)
		if cmd&1 == 0 {
			if cmd&2 != 0 {
				// Enter PVS
				if existing := p.gameState.entities[currentEntity]; existing != nil {
					// Sometimes entities don't get destroyed when they should be
					// For instance when a player is replaced by a BOT
					existing.Destroy()
				}

				p.gameState.entities[currentEntity] = p.stParser.ReadEnterPVS(r, currentEntity)
			} else { //nolint:gocritic
				// Delta Update
				if entity := p.gameState.entities[currentEntity]; entity != nil {
					entity.ApplyUpdate(r)
				}
			}
		} else {
			if cmd&2 != 0 {
				// Leave PVS
				if entity := p.gameState.entities[currentEntity]; entity != nil {
					entity.Destroy()
					delete(p.gameState.entities, currentEntity)
				}
			}
		}
	}

	err := r.Pool()
	if err != nil {
		p.eventDispatcher.Dispatch(events.ParserWarn{
			Message: err.Error(),
		})
	}
}

func (p *parser) handleSetConVar(setConVar *msg.CNETMsg_SetConVar) {
	updated := make(map[string]string)
	for _, cvar := range setConVar.Convars.Cvars {
		updated[cvar.Name] = cvar.Value
		p.gameState.rules.conVars[cvar.Name] = cvar.Value
	}

	p.eventDispatcher.Dispatch(events.ConVarsUpdated{
		UpdatedConVars: updated,
	})
}

func (p *parser) handleServerInfo(srvInfo *msg.CSVCMsg_ServerInfo) {
	// srvInfo.MapCrc might be interesting as well
	p.tickInterval = srvInfo.TickInterval

	p.eventDispatcher.Dispatch(events.TickRateInfoAvailable{
		TickRate: p.TickRate(),
		TickTime: p.TickTime(),
	})
}
