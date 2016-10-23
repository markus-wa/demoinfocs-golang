package demoinfocs

import (
	"bytes"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"github.com/markus-wa/demoinfocs-golang/st"
)

func (p *Parser) handlePackageEntities(packageEntities interface{}) {
	pe := packageEntities.(*msg.CSVCMsg_PacketEntities)
	r := bs.NewBitReader(bytes.NewReader(pe.EntityData), bs.SmallBuffer)

	currentEntity := -1
	for i := 0; i < int(pe.UpdatedEntries); i++ {
		currentEntity += 1 + int(r.ReadUBitInt())
		if !r.ReadBit() {
			if r.ReadBit() {
				e := p.readEnterPVS(r, currentEntity)
				p.entities[currentEntity] = e
				e.ApplyUpdate(r)
			} else {
				p.entities[currentEntity].ApplyUpdate(r)
			}
		} else {
			// FIXME: Might have to destroy the entities contents first, not sure yet
			// Could do weird stuff with event handlers otherwise
			p.entities[currentEntity] = nil
			r.ReadBit()
		}
	}
	r.Close()
}

func (p *Parser) readEnterPVS(reader bs.BitReader, entityId int) *st.Entity {
	scId := int(reader.ReadInt(uint(p.stParser.ClassBits())))
	reader.ReadInt(10)
	newEntity := st.NewEntity(entityId, p.stParser.ServerClasses()[scId])
	newEntity.ServerClass.FireEntityCreatedEvent(newEntity)

	if p.preprocessedBaselines[scId] != nil {
		for _, bl := range p.preprocessedBaselines[scId] {
			newEntity.Props()[bl.PropIndex()].FirePropertyUpdateEvent(bl.Value(), newEntity)
		}
	} else {
		ppBase := make([]*st.RecordedPropertyUpdate, 0)
		if p.instanceBaselines[scId] != nil {
			newEntity.CollectProperties(&ppBase)
			r := bs.NewBitReader(bytes.NewReader(p.instanceBaselines[scId]), bs.SmallBuffer)
			newEntity.ApplyUpdate(r)
			r.Close()
		}
		p.preprocessedBaselines[scId] = ppBase
	}

	return newEntity
}
