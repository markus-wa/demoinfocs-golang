package sendtables

// ServerClass stores meta information about Entity types (e.g. palyers, teams etc.).
type ServerClass struct {
	ClassID               int
	DataTableID           int
	Name                  string
	DTName                string
	FlattenedProps        []FlattenedPropEntry
	BaseClasses           []*ServerClass
	entityCreatedHandlers []EntityCreatedHandler
}

// String returns a human readable identification of the ServerClass.
func (sc *ServerClass) String() string {
	return sc.Name + " | " + sc.DTName
}

// FireEntityCreatedEvent triggers all registered EntityCreatedHandlers
// on the ServerClass with a new EntityCreatedEvent.
func (sc *ServerClass) FireEntityCreatedEvent(entity *Entity) {
	for _, h := range sc.entityCreatedHandlers {
		if h != nil {
			h(EntityCreatedEvent{Entity: entity, ServerClass: sc})
		}
	}
}

// RegisterEntityCreatedHandler registers a EntityCreatedHandler on the ServerClass.
// The handler will be triggered on every FireEntityCreatedEvent call.
func (sc *ServerClass) RegisterEntityCreatedHandler(handler EntityCreatedHandler) {
	sc.entityCreatedHandlers = append(sc.entityCreatedHandlers, handler)
}

// FlattenedPropEntry stores meta information about a property of an Entity.
type FlattenedPropEntry struct {
	prop             *sendTableProperty
	arrayElementProp *sendTableProperty
	name             string
}

// Name returs the name of the prop entry.
func (fpe FlattenedPropEntry) Name() string {
	return fpe.name
}

type excludeEntry struct {
	varName     string
	dtName      string
	excludingDt string
}

// EntityCreatedEvent contains information about a newly created entity.
type EntityCreatedEvent struct {
	ServerClass *ServerClass
	Entity      *Entity
}

// EntityCreatedHandler is the interface for handlers that are interested in EntityCreatedEvents.
type EntityCreatedHandler func(EntityCreatedEvent)
