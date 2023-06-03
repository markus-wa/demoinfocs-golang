package sendtables2

import (
	"math"
	"strings"

	"google.golang.org/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/v3/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/sendtables"
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

type Parser struct {
	serializers       map[string]*serializer
	classIdSize       uint32
	classBaselines    map[int32][]byte
	classesById       map[int32]*class
	classesByName     map[string]*class
	entityFullPackets int
	entities          map[int32]*Entity
	entityHandlers    []EntityHandler
}

func (p *Parser) ReadEnterPVS(r *bit.BitReader, index int, entities map[int]st.Entity, slot int) st.Entity {
	panic("implement me")
}

type serverClasses Parser

func (sc *serverClasses) All() (res []st.ServerClass) {
	for _, c := range sc.classesById {
		res = append(res, c)
	}

	return
}

func (sc *serverClasses) FindByName(name string) st.ServerClass {
	class := sc.classesByName[name]
	if class == nil {
		return nil
	}

	return sc.classesByName[name]
}

func (sc *serverClasses) String() string {
	names := make([]string, 0, len(sc.classesById))

	for _, c := range sc.classesById {
		names = append(names, c.name)
	}

	return strings.Join(names, "\n")
}

func (p *Parser) ServerClasses() st.ServerClasses {
	return (*serverClasses)(p)
}

func NewParser() *Parser {
	return &Parser{
		serializers:    make(map[string]*serializer),
		classBaselines: make(map[int32][]byte),
		classesById:    make(map[int32]*class),
		classesByName:  make(map[string]*class),
		entities:       make(map[int32]*Entity),
	}
}

// Internal callback for OnCSVCMsg_ServerInfo.
func (p *Parser) OnServerInfo(m *msgs2.CSVCMsg_ServerInfo) error {
	// This may be needed to parse PacketEntities.
	p.classIdSize = uint32(math.Log(float64(m.GetMaxClasses()))/math.Log(2)) + 1

	return nil
}

func (p *Parser) OnDemoClassInfo(m *msgs2.CDemoClassInfo) error {
	for _, c := range m.GetClasses() {
		classId := c.GetClassId()
		networkName := c.GetNetworkName()

		class := &class{
			classId:    classId,
			name:       networkName,
			serializer: p.serializers[networkName],
		}
		p.classesById[class.classId] = class
		p.classesByName[class.name] = class
	}

	return nil
}

// SetInstanceBaseline sets the raw instance-baseline data for a serverclass by ID.
//
// Intended for internal use only.
func (p *Parser) SetInstanceBaseline(scID int, data []byte) {
	p.classBaselines[int32(scID)] = data
}

func (p *Parser) ParsePacket(b []byte) error {
	r := newReader(b)
	buf := r.readBytes(r.readVarUint32())

	msg := &msgs2.CSVCMsg_FlattenedSerializer{}
	if err := proto.Unmarshal(buf, msg); err != nil {
		return err
	}

	fields := map[int32]*field{}
	fieldTypes := map[string]*fieldType{}

	for _, s := range msg.GetSerializers() {
		serializer := &serializer{
			name:    msg.GetSymbols()[s.GetSerializerNameSym()],
			version: s.GetSerializerVersion(),
			fields:  []*field{},
		}

		for _, i := range s.GetFieldsIndex() {
			if _, ok := fields[i]; !ok {
				// create a new field
				field := newField(msg, msg.GetFields()[i])

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
				if field.serializer != nil {
					if field.fieldType.pointer || pointerTypes[field.fieldType.baseType] {
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
			serializer.fields = append(serializer.fields, fields[i])
		}

		// store the serializer for field reference
		p.serializers[serializer.name] = serializer

		if _, ok := p.classesByName[serializer.name]; ok {
			p.classesByName[serializer.name].serializer = serializer
		}
	}

	return nil
}
