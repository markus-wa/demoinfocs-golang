package sendtables

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"google.golang.org/protobuf/proto"
)

/*
from demoinfo2.txt:

// referenced components require pointer indirection
DemoTypeAlias_t { m_TypeAlias = "CBodyComponentDCGBaseAnimating" 		m_UnderlyingType = "CBodyComponentDCGBaseAnimating*" },
DemoTypeAlias_t { m_TypeAlias = "CBodyComponentBaseAnimating"			m_UnderlyingType = "CBodyComponentBaseAnimating*" },
DemoTypeAlias_t { m_TypeAlias = "CBodyComponentBaseAnimatingOverlay"	m_UnderlyingType = "CBodyComponentBaseAnimatingOverlay*" },
DemoTypeAlias_t { m_TypeAlias = "CBodyComponentBaseModelEntity"			m_UnderlyingType = "CBodyComponentBaseModelEntity*" },
DemoTypeAlias_t { m_TypeAlias = "CBodyComponent"						m_UnderlyingType = "CBodyComponent*" },
DemoTypeAlias_t { m_TypeAlias = "CBodyComponentSkeletonInstance"		m_UnderlyingType = "CBodyComponentSkeletonInstance*" },
DemoTypeAlias_t { m_TypeAlias = "CBodyComponentPoint"					m_UnderlyingType = "CBodyComponentPoint*" },
DemoTypeAlias_t { m_TypeAlias = "CLightComponent"						m_UnderlyingType = "CLightComponent*" },
DemoTypeAlias_t { m_TypeAlias = "CRenderComponent"						m_UnderlyingType = "CRenderComponent*" },

// this is legacy, would be good candidate to use demo file upconversion to remove?
DemoTypeAlias_t { m_TypeAlias = "CPhysicsComponent"						m_UnderlyingType = "CPhysicsComponent*" },
*/
var pointerTypes = map[string]bool{
	// "PhysicsRagdollPose_t":   true,
	// "CEntityIdentity":        true,
	// "CPlayerLocalData":       true,
	// "CPlayer_CameraServices": true,
	"CBodyComponentDCGBaseAnimating":     true,
	"CBodyComponentBaseAnimating":        true,
	"CBodyComponentBaseAnimatingOverlay": true,
	"CBodyComponentBaseModelEntity":      true,
	"CBodyComponent":                     true,
	"CBodyComponentSkeletonInstance":     true,
	"CBodyComponentPoint":                true,
	"CLightComponent":                    true,
	"CRenderComponent":                   true,
	"CPhysicsComponent":                  true,
}

var itemCounts = map[string]int{
	"MAX_ITEM_STOCKS":             8,
	"MAX_ABILITY_DRAFT_ABILITIES": 48,
}

type tuple struct {
	ent *Entity
	op  EntityOp
}

// Parser parses CS2 send tables (FlattenedSerializer + PacketEntities).
//
// Parser and its methods are used internally by package demoinfocs and are not covered by API stability guarantees,
// except for ServerClasses(). Use demoinfocs.Parser.ServerClasses() and the GameState's entities instead.
//
// A Parser must be created with NewParser; the zero value is not usable.
type Parser struct {
	serializers       map[string]*serializer
	classIdSize       uint32 //nolint:revive
	classBaselines    map[int32][]byte
	classesById       map[int32]*ServerClass //nolint:revive
	classesByName     map[string]*ServerClass
	entityFullPackets int
	entities          map[int32]*Entity
	entityHandlers    []EntityHandler
	pathCache         []*fieldPath
	tuplesCache       []tuple
	// nextPolyID is a global counter for assigning unique polySerializerIDs to
	// polymorphic fixed-table fields as they are encountered during ParsePacket.
	nextPolyID int
	// polyMaxCache memoizes, per serializer, the highest polySerializerID
	// reachable from it. Used to size per-class polySerializers slices.
	polyMaxCache map[*serializer]int
	// entityBudget bounds the work of decoding entities, see entityBudget.
	entityBudget entityBudget
}

// maxPolymorphicFields bounds the number of polymorphic pointer fields of a schema (see Parser.nextPolyID).
// Real demos have one (m_pGameRules.m_pGameModeRules).
const maxPolymorphicFields = 64

// ServerClasses is a searchable list of ServerClasses.
//
// It's obtained from Parser.ServerClasses() and reflects classes that are added later on.
// The zero value is an empty list.
type ServerClasses struct {
	byID   map[int32]*ServerClass
	byName map[string]*ServerClass
}

// All returns all server-classes, sorted by ID.
// It returns nil if there are none.
func (sc ServerClasses) All() []*ServerClass {
	if len(sc.byID) == 0 {
		return nil
	}

	res := make([]*ServerClass, 0, len(sc.byID))

	for _, c := range sc.byID {
		res = append(res, c)
	}

	slices.SortFunc(res, func(a, b *ServerClass) int {
		return cmp.Compare(a.classId, b.classId)
	})

	return res
}

// FindByName returns the server-class with the given name, or nil if there is none.
func (sc ServerClasses) FindByName(name string) *ServerClass {
	return sc.byName[name]
}

// String returns the names of all server-classes, one per line, sorted by ID.
func (sc ServerClasses) String() string {
	all := sc.All()
	names := make([]string, 0, len(all))

	for _, c := range all {
		names = append(names, c.name)
	}

	return strings.Join(names, "\n")
}

// ServerClasses returns a ServerClasses view of all known classes.
// The view shares the parser's class maps, so classes added later on are included.
// This relies on classesById and classesByName never being reassigned after NewParser.
func (p *Parser) ServerClasses() ServerClasses {
	return ServerClasses{byID: p.classesById, byName: p.classesByName}
}

// NewParser creates a new CS2 send-tables parser.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
func NewParser() *Parser {
	return &Parser{
		serializers:    make(map[string]*serializer),
		classBaselines: make(map[int32][]byte),
		classesById:    make(map[int32]*ServerClass),
		classesByName:  make(map[string]*ServerClass),
		entities:       make(map[int32]*Entity),
		polyMaxCache:   make(map[*serializer]int),
		entityBudget:   newEntityBudget(),
	}
}

// OnServerInfo processes a CSVCMsg_ServerInfo message.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
func (p *Parser) OnServerInfo(m *msg.CSVCMsg_ServerInfo) error {
	// This may be needed to parse PacketEntities.
	p.classIdSize = uint32(math.Log(float64(m.GetMaxClasses()))/math.Log(2)) + 1

	return nil
}

// OnDemoClassInfo processes a CDemoClassInfo message, registering the server-classes.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
func (p *Parser) OnDemoClassInfo(m *msg.CDemoClassInfo) error {
	for _, c := range m.GetClasses() {
		classId := c.GetClassId() //nolint:revive
		networkName := c.GetNetworkName()

		// Size the per-entity polySerializers slice to the highest polymorphic
		// serializer ID reachable from this class's serializer, so entities of
		// classes without polymorphic fields stay on the shared fast paths.
		polyCount := 0

		if ser := p.serializers[networkName]; ser != nil {
			if maxID := ser.maxPolyID(p.polyMaxCache); maxID >= 0 {
				polyCount = maxID + 1
			}
		}

		class := &ServerClass{
			classId:     classId,
			name:        networkName,
			serializer:  p.serializers[networkName],
			polyCount:   polyCount,
			fpNameCache: &fpNameTreeCache{},
			fpFlatCache: make(map[uint64]string),
		}
		p.classesById[class.classId] = class
		p.classesByName[class.name] = class
	}

	return nil
}

// SetInstanceBaseline sets the raw instance-baseline data for a serverclass by ID.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
func (p *Parser) SetInstanceBaseline(scID int, data []byte) {
	if scID < 0 || scID > math.MaxInt32 {
		panic(fmt.Sprintf("scID %d is out of bounds", scID))
	}

	p.classBaselines[int32(scID)] = data
}

// ParsePacket parses a FlattenedSerializer packet, building the serializer/field model.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
//
//nolint:gocognit,funlen
func (p *Parser) ParsePacket(b []byte) error {
	r := newReader(b)
	buf := r.readBytes(r.readVarUint32())
	r.release()

	msg := &msg.CSVCMsg_FlattenedSerializer{}
	if err := proto.Unmarshal(buf, msg); err != nil {
		return err
	}

	fields := map[int32]*field{}
	fieldTypes := map[string]*fieldType{}

	// all serializer instances of this message, including ones later shadowed
	// in p.serializers by another version of the same name; they stay reachable
	// through field.serializer/polyTypes and must be re-keyed on renames too
	serializers := make([]*serializer, 0, len(msg.GetSerializers()))

	for _, s := range msg.GetSerializers() {
		serializer := newSerializer(
			msg.GetSymbols()[s.GetSerializerNameSym()],
			s.GetSerializerVersion(),
		)
		serializers = append(serializers, serializer)

		for _, i := range s.GetFieldsIndex() {
			if _, ok := fields[i]; !ok { //nolint:nestif
				// create a new field
				field := newField(p.serializers, msg, msg.GetFields()[i])

				// dotabuff/manta patches parent name in builds <= 990
				// if p.gameBuild <= 990 {
				//	field.parentName = serializer.name
				//}

				// find or create a field type
				if _, ok := fieldTypes[field.varType]; !ok {
					fieldTypes[field.varType] = newFieldType(field.varType)
				}

				field.fieldType = fieldTypes[field.varType]

				// find associated serializer
				if field.serializerName != "" {
					field.serializer = p.serializers[field.serializerName]
				}

				// apply any build-specific patches to the field
				for _, h := range fieldPatches {
					h.patch(field)
				}

				// determine field model
				if field.serializer != nil || len(field.polyTypes) > 0 { //nolint:gocritic
					if field.fieldType.pointer || pointerTypes[field.fieldType.baseType] || len(field.polyTypes) > 0 {
						if len(field.polyTypes) > 0 {
							// Every entity of a class with polymorphic fields allocates a slot for each ID,
							// so a corrupt schema with lots of them would allocate a lot per entity.
							if p.nextPolyID >= maxPolymorphicFields {
								_panicf("more than %d polymorphic fields: corrupt schema", maxPolymorphicFields)
							}

							// Assign a unique per-entity slot BEFORE calling setModel,
							// so the closure in setModel captures the correct ID.
							field.polySerializerID = p.nextPolyID
							p.nextPolyID++
						}

						field.setModel(fieldModelFixedTable)
					} else {
						field.setModel(fieldModelVariableTable)
					}
				} else if field.fieldType.count > 0 && field.fieldType.baseType != "char" {
					field.setModel(fieldModelFixedArray)
				} else if field.fieldType.baseType == "CUtlVector" || field.fieldType.baseType == "CNetworkUtlVectorBase" {
					field.setModel(fieldModelVariableArray)
				} else {
					field.setModel(fieldModelSimple)
				}

				// store the field
				fields[i] = field
			}

			// add the field to the serializer
			serializer.addField(fields[i])
		}

		// store the serializer for field reference
		p.serializers[serializer.name] = serializer

		if c, ok := p.classesByName[serializer.name]; ok {
			// The full reachable graph of this serializer is resolved by now
			// (fields can only reference previously parsed serializers), so it's
			// safe to back-patch both the serializer and the poly slot count for
			// classes whose CDemoClassInfo arrived before this FSV packet.
			c.serializer = serializer

			if maxID := serializer.maxPolyID(p.polyMaxCache); maxID >= 0 {
				c.polyCount = maxID + 1
			}
		}
	}

	resolveFieldNameCollisions(serializers)

	return nil
}
