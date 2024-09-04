// Package sendtables contains code related to decoding sendtables.
// Mostly used internally but can be interesting for direct access to server-classes and entities.
package sendtables

import (
	"bytes"
	"fmt"
	"strings"

	bit "github.com/markus-wa/demoinfocs-golang/v5/internal/bitread"
)

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

// Stores meta information about a property of an Entity.
type flattenedPropEntry struct {
	prop             *sendTableProperty
	arrayElementProp *sendTableProperty
	name             string
}

//go:generate ifacemaker -f sendtables.go -s serverClass -i ServerClass -p sendtables -D -y "ServerClass is an auto-generated interface for property, intended to be used when mockability is needed." -c "DO NOT EDIT: Auto generated" -o serverclass_interface.go

// serverClass stores meta information about Entity types (e.g. palyers, teams etc.).
type serverClass struct {
	id              int
	name            string
	dataTableID     int
	dataTableName   string
	baseClasses     []*serverClass
	flattenedProps  []flattenedPropEntry
	propNameToIndex map[string]int

	createdHandlers      []EntityCreatedHandler
	instanceBaseline     []byte                // Raw baseline
	preprocessedBaseline map[int]PropertyValue // Preprocessed baseline
}

// ID returns the server-class's ID.
func (sc *serverClass) ID() int {
	return sc.id
}

// Name returns the server-class's name.
func (sc *serverClass) Name() string {
	return sc.name
}

// DataTableID returns the data-table ID.
func (sc *serverClass) DataTableID() int {
	return sc.dataTableID
}

// DataTableName returns the data-table name.
func (sc *serverClass) DataTableName() string {
	return sc.dataTableName
}

// BaseClasses returns the base-classes of this server-class.
func (sc *serverClass) BaseClasses() (res []ServerClass) {
	for _, v := range sc.baseClasses {
		res = append(res, v)
	}

	return
}

// PropertyEntries returns the names of all property-entries on this server-class.
func (sc *serverClass) PropertyEntries() []string {
	propEntryCount := len(sc.flattenedProps)
	names := make([]string, propEntryCount)

	for i := 0; i < propEntryCount; i++ {
		names[i] = sc.flattenedProps[i].name
	}

	return names
}

type PropertyEntry struct {
	Name    string
	IsArray bool
	Type    PropertyType
}

// PropertyEntryDefinitions returns all property-entries on this server-class.
func (sc *serverClass) PropertyEntryDefinitions() []PropertyEntry {
	propEntryCount := len(sc.flattenedProps)
	res := make([]PropertyEntry, propEntryCount)

	for i := 0; i < propEntryCount; i++ {
		res[i].Name = sc.flattenedProps[i].name
		res[i].IsArray = sc.flattenedProps[i].prop.rawType == propTypeArray

		if res[i].IsArray {
			res[i].Type = PropertyType(sc.flattenedProps[i].arrayElementProp.rawType)
		} else {
			res[i].Type = PropertyType(sc.flattenedProps[i].prop.rawType)
		}
	}

	return res
}

func (sc *serverClass) newEntity(entityDataReader *bit.BitReader, entityID int, serialNum int, recordingPlayerSlot int) *entity {
	props := make([]property, len(sc.flattenedProps))

	for i := range sc.flattenedProps {
		props[i] = property{entry: &sc.flattenedProps[i]}
	}

	entity := &entity{serverClass: sc, id: entityID, serialNum: serialNum, props: props}

	entity.initialize(recordingPlayerSlot)

	if sc.preprocessedBaseline != nil {
		entity.applyBaseline(sc.preprocessedBaseline)
	} else if sc.instanceBaseline != nil {
		r := bit.NewSmallBitReader(bytes.NewReader(sc.instanceBaseline))
		sc.preprocessedBaseline = entity.initializeBaseline(r)
		r.Pool()
	} else {
		sc.preprocessedBaseline = make(map[int]PropertyValue)
	}

	entity.ApplyUpdate(entityDataReader)

	// Fire created-handlers so update-handlers can be registered
	for _, h := range sc.createdHandlers {
		h(entity)
	}

	// Fire all post-creation actions
	for _, f := range entity.onCreateFinished {
		f()
	}

	return entity
}

// OnEntityCreated registers a function to be called when a new entity is created from this serverClass.
func (sc *serverClass) OnEntityCreated(handler EntityCreatedHandler) {
	sc.createdHandlers = append(sc.createdHandlers, handler)
}

// EntityCreatedHandler is the interface for handlers that are interested in EntityCreatedEvents.
type EntityCreatedHandler func(Entity)

var serverClassStringFormat = `serverClass: id=%d name=%s
	dataTableId=%d
	dataTableName=%s
	baseClasses:
		%s
	properties:
		%s`

func (sc *serverClass) String() string {
	baseClasses := make([]string, len(sc.baseClasses))
	for i, bc := range sc.baseClasses {
		baseClasses[i] = bc.name
	}

	props := make([]string, len(sc.flattenedProps))
	for i, fProp := range sc.flattenedProps {
		props[i] = fProp.name
	}

	baseClassesStr := "-"
	if len(baseClasses) > 0 {
		baseClassesStr = strings.Join(baseClasses, "\n\t\t")
	}

	propsStr := "-"
	if len(props) > 0 {
		propsStr = strings.Join(props, "\n\t\t")
	}

	return fmt.Sprintf(serverClassStringFormat, sc.id, sc.name, sc.dataTableID, sc.dataTableName, baseClassesStr, propsStr)
}
