package demoinfocs

import (
	"bytes"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
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
			p.gameState.entities[currentEntity].Destroy()
			delete(p.gameState.entities, currentEntity)

			// 'Force Delete' flag, not exactly sure what it's supposed to do
			r.ReadBit()
		} else if r.ReadBit() {
			// Enter PVS
			p.gameState.entities[currentEntity] = p.stParser.ReadEnterPVS(r, currentEntity)
		} else {
			// Delta Update
			p.gameState.entities[currentEntity].ApplyUpdate(r)
		}
	}
	r.Pool()
}
