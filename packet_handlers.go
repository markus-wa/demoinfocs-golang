package demoinfocs

import (
	"bytes"
	"github.com/golang/geo/r3"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"github.com/markus-wa/demoinfocs-golang/st"
	"reflect"
)

func (p *Parser) handlePackageEntities(packageEntities interface{}) {
	pe := packageEntities.(*msg.CSVCMsg_PacketEntities)
	r := bs.NewBitReader(bytes.NewReader(pe.EntityData))

	currentEntity := -1
	for i := 0; i < int(*pe.UpdatedEntries); i++ {
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
			collectProperties(newEntity, ppBase)
			newEntity.ApplyUpdate(bs.NewBitReader(bytes.NewReader(p.instanceBaselines[scId])))
			p.preprocessedBaselines[scId] = append(p.preprocessedBaselines[scId], ppBase...)
		}
	}

	return newEntity
}

func collectProperties(entity *st.Entity, ppBase []*st.RecordedPropertyUpdate) {
	adder := func(event *st.PropertyUpdateEvent) {
		ppBase = append(ppBase, event.Record())
	}

	for _, p := range entity.Props() {
		switch p.Entry().Prop().Type() {
		case st.SPT_Array:
			p.RegisterPropertyUpdateHandler(reflect.TypeOf(make([]interface{}, 0)), adder)

		case st.SPT_Float:
			p.RegisterPropertyUpdateHandler(reflect.TypeOf(float32(0)), adder)

		case st.SPT_Int:
			p.RegisterPropertyUpdateHandler(reflect.TypeOf(int(0)), adder)

		case st.SPT_String:
			p.RegisterPropertyUpdateHandler(reflect.TypeOf(""), adder)

		case st.SPT_Vector:
			fallthrough
		case st.SPT_VectorXY:
			p.RegisterPropertyUpdateHandler(reflect.TypeOf(r3.Vector{}), adder)

		default:
			panic("Unknown type")
		}
	}
}
