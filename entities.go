package demoinfocs

import (
	"bytes"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
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

			// FIXME: Might have to destroy the entities contents first, not sure yet
			// Could do weird stuff with event handlers otherwise
			p.entities[currentEntity] = nil

			if r.ReadBit() {
				// TODO: Force Delete??
			}
		} else {
			if r.ReadBit() {
				// Enter PVS
				e := p.readEnterPVS(r, currentEntity)
				p.entities[currentEntity] = e
				e.ApplyUpdate(r)
			} else {
				// Delta Update
				p.entities[currentEntity].ApplyUpdate(r)
			}
		}
	}
	r.Pool()
}

func (p *Parser) readEnterPVS(reader *bit.BitReader, entityID int) *st.Entity {
	scID := int(reader.ReadInt(p.stParser.ClassBits()))
	reader.Skip(10) // Serial Number

	newEntity := st.NewEntity(entityID, p.stParser.ServerClasses()[scID])
	newEntity.ServerClass.FireEntityCreatedEvent(newEntity)

	if p.preprocessedBaselines[scID] != nil {
		for idx, val := range p.preprocessedBaselines[scID] {
			newEntity.Props()[idx].FirePropertyUpdate(val)
		}
	} else {
		ppBase := make(map[int]st.PropValue, 0)
		if p.instanceBaselines[scID] != nil {
			newEntity.CollectProperties(&ppBase)
			r := bit.NewSmallBitReader(bytes.NewReader(p.instanceBaselines[scID]))
			newEntity.ApplyUpdate(r)
			r.Pool()
			// TODO: Unregister PropertyUpdateHandlers from CollectProperties()
			// PropertyUpdateHandlers would have to be registered as pointers for that to work
		}
		p.preprocessedBaselines[scID] = ppBase
	}

	return newEntity
}
