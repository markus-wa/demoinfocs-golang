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
			h(EntityCreatedEvent{entity: entity, serverClass: sc})
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

func (fpe FlattenedPropEntry) Prop() *SendTableProperty {
	return fpe.prop
}

func (fpe FlattenedPropEntry) ArrayElementProp() *SendTableProperty {
	return fpe.arrayElementProp
}

func (fpe FlattenedPropEntry) Name() string {
	return fpe.name
}

type ExcludeEntry struct {
	varName     string
	dtName      string
	excludingDt string
}

func (ee ExcludeEntry) VarName() string {
	return ee.varName
}

func (ee ExcludeEntry) DtName() string {
	return ee.dtName
}

func (ee ExcludeEntry) ExcludingDt() string {
	return ee.excludingDt
}

type EntityCreatedEvent struct {
	serverClass *ServerClass
	entity      *Entity
}

func (e EntityCreatedEvent) ServerClass() *ServerClass {
	return e.serverClass
}
func (e EntityCreatedEvent) Entity() *Entity {
	return e.entity
}

type EntityCreatedHandler func(EntityCreatedEvent)
