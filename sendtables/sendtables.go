// Package sendtables contains sendtable specific magic and should really be better documented (TODO).
package sendtables

// sendPropertyFlags stores multiple send property flags.
type sendPropertyFlags int

// hasFlagSet returns true if the given flag is set
func (spf sendPropertyFlags) hasFlagSet(flag sendPropertyFlags) bool {
	return int(spf)&int(flag) == int(flag)
}

type sendTable struct {
	properties []sendTableProperty
	name       string
	isEnd      bool
}

type sendTableProperty struct {
	flags            sendPropertyFlags
	name             string
	dataTableName    string
	lowValue         float32
	highValue        float32
	numberOfBits     int
	numberOfElements int
	priority         int
	rawType          int
}

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

// CreateFinishedDummyPropertyName is a dummy property for entities.
// When it's updated it signals that the entity creation has finished (all property updates sent out).
// TODO: Figure out a better way to do this.
const CreateFinishedDummyPropertyName = "CreateFinishedDummyProp"

// FireEntityCreatedEvent triggers all registered EntityCreatedHandlers
// on the ServerClass with a new EntityCreatedEvent.
func (sc *ServerClass) FireEntityCreatedEvent(entity *Entity) {
	entity.props = append(entity.props, PropertyEntry{
		entry: &FlattenedPropEntry{
			name: CreateFinishedDummyPropertyName,
		},
	})

	for _, h := range sc.entityCreatedHandlers {
		if h != nil {
			h(EntityCreatedEvent{Entity: entity, ServerClass: sc})
		}
	}

	for _, v := range entity.props {
		v.FirePropertyUpdate(v.value)
	}
}

// RegisterEntityCreatedHandler registers a EntityCreatedHandler on the ServerClass.
// The handler will be triggered on every FireEntityCreatedEvent call.
func (sc *ServerClass) RegisterEntityCreatedHandler(handler EntityCreatedHandler) {
	sc.entityCreatedHandlers = append(sc.entityCreatedHandlers, handler)
}

// FlattenedPropEntry stores meta information about a property of an Entity.
// Might be renamed in a future major release (Deprecated).
type FlattenedPropEntry struct {
	prop             *sendTableProperty
	arrayElementProp *sendTableProperty
	name             string
}

// Name returs the name of the prop entry.
func (fpe FlattenedPropEntry) Name() string {
	return fpe.name
}

// EntityCreatedEvent contains information about a newly created entity.
type EntityCreatedEvent struct {
	ServerClass *ServerClass
	Entity      *Entity
}

// EntityCreatedHandler is the interface for handlers that are interested in EntityCreatedEvents.
type EntityCreatedHandler func(EntityCreatedEvent)
