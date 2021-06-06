package demoinfocs

import (
	"bytes"
	"strconv"
	"time"

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

	r.Pool()
}

func (p *parser) handleSetConVar(setConVar *msg.CNETMsg_SetConVar) {
	updated := make(map[string]string)
	for _, cvar := range setConVar.Convars.Cvars {
		updated[cvar.Name] = cvar.Value
		p.gameState.rules.conVars[cvar.Name] = cvar.Value
	}

	// note we should only update the frame rate if it's not determinable from the header
	// this is because changing the frame rate mid game has no effect on the active recording
	if rate, ok := updated["tv_snapshotrate"]; ok && p.frameRate == 0 {
		tvSnapshotRate, err := strconv.Atoi(rate)
		if err != nil {
			p.setError(err)
		} else {
			p.setFrameRate(float64(tvSnapshotRate), events.FrameRateSourceConVars)
		}
	}

	p.eventDispatcher.Dispatch(events.ConVarsUpdated{
		UpdatedConVars: updated,
	})
}

func frameRateInfoAvailableEvent(rate float64, source events.FrameRateSource) events.FrameRateInfo {
	return events.FrameRateInfo{
		Source:    source,
		FrameRate: rate,
		FrameTime: time.Duration(float64(time.Second) / rate),
	}
}

func (p *parser) handleServerInfo(srvInfo *msg.CSVCMsg_ServerInfo) {
	// srvInfo.MapCrc might be interesting as well
	p.tickInterval = srvInfo.TickInterval

	p.eventDispatcher.Dispatch(events.TickRateInfo{
		Source:   events.TickRateSourceServerInfo,
		TickRate: p.TickRate(),
		TickTime: p.TickTime(),
	})
}
