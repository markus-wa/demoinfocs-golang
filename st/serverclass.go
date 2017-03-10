package st

import ()

type ServerClass struct {
	ClassID               int
	DataTableID           int
	Name                  string
	DTName                string
	FlattenedProps        []FlattenedPropEntry
	BaseClasses           []*ServerClass
	entityCreatedHandlers []EntityCreatedHandler
}

func (sc *ServerClass) String() string {
	return sc.Name + " | " + sc.DTName
}

func (sc *ServerClass) FireEntityCreatedEvent(entity *Entity) {
	for _, h := range sc.entityCreatedHandlers {
		if h != nil {
			h(EntityCreatedEvent{Entity: entity, ServerClass: sc})
		}
	}
}

func (sc *ServerClass) RegisterEntityCreatedHandler(handler EntityCreatedHandler) {
	sc.entityCreatedHandlers = append(sc.entityCreatedHandlers, handler)
}

type FlattenedPropEntry struct {
	prop             *SendTableProperty
	arrayElementProp *SendTableProperty
	name             string
}

func (fpe FlattenedPropEntry) Name() string {
	return fpe.name
}

type excludeEntry struct {
	varName     string
	dtName      string
	excludingDt string
}

type EntityCreatedEvent struct {
	ServerClass *ServerClass
	Entity      *Entity
}

type EntityCreatedHandler func(EntityCreatedEvent)
