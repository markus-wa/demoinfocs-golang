package demoinfocs

import (
	"bytes"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	"github.com/markus-wa/demoinfocs-golang/events"
	"github.com/markus-wa/demoinfocs-golang/msg"
)

const entitySentinel = 9999

func (p *Parser) handlePacketEntities(pe *msg.CSVCMsg_PacketEntities) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	r := bit.NewSmallBitReader(bytes.NewReader(pe.EntityData))

	currentEntity := -1
	for i := 0; i < int(pe.UpdatedEntries); i++ {
		currentEntity += 1 + int(r.ReadUBitInt())

		if currentEntity > entitySentinel {
			break
		}

		if r.ReadBit() {
			// Leave PVS
			if entity := p.gameState.entities[currentEntity]; entity != nil {
				entity.Destroy()
				delete(p.gameState.entities, currentEntity)
			}

			// 'Force Delete' flag, not exactly sure what it's supposed to do
			r.ReadBit()
		} else if r.ReadBit() {
			// Enter PVS
			if existing := p.gameState.entities[currentEntity]; existing != nil {
				// Sometimes entities don't get destroyed when they should be
				// For instance when a player is replaced by a BOT
				existing.Destroy()
			}
			p.gameState.entities[currentEntity] = p.stParser.ReadEnterPVS(r, currentEntity)
		} else {
			// Delta Update
			if entity := p.gameState.entities[currentEntity]; entity != nil {
				entity.ApplyUpdate(r)
			}
		}
	}
	r.Pool()
}

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

func (p *Parser) handleServerInfo(srvInfo *msg.CSVCMsg_ServerInfo) {
	// srvInfo.MapCrc might be interesting as well
	p.tickInterval = srvInfo.TickInterval
}
