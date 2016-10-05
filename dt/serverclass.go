package dt

import ()

// TODO: EntityCreatedEvent

type ServerClass struct {
	ClassId        int
	DataTableId    int
	Name           string
	DTName         string
	FlattenedProps []*FlattenedPropEntry
	BaseClasses    []*ServerClass
}

func (sc *ServerClass) String() string {
	return sc.Name + " | " + sc.DTName
}

type FlattenedPropEntry struct {
	prop             *SendTableProperty
	arrayElementProp *SendTableProperty
	name             string
}

func (fpe *FlattenedPropEntry) Prop() *SendTableProperty {
	return fpe.prop
}

func (fpe *FlattenedPropEntry) ArrayElementProp() *SendTableProperty {
	return fpe.arrayElementProp
}

func (fpe *FlattenedPropEntry) Name() string {
	return fpe.name
}

type ExcludeEntry struct {
	varName     string
	dtName      string
	excludingDt string
}

func (ee *ExcludeEntry) VarName() string {
	return ee.varName
}

func (ee *ExcludeEntry) DtName() string {
	return ee.dtName
}

func (ee *ExcludeEntry) ExcludingDt() string {
	return ee.excludingDt
}

type EntityCreatedEvent struct {
	serverClass *ServerClass
	entity      *Entity
}

func (ece *EntityCreatedEvent) ServerClass() *ServerClass {
	return ece.serverClass
}
func (ece *EntityCreatedEvent) Entity() *Entity {
	return ece.entity
}
