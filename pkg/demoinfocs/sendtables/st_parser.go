package sendtables

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"google.golang.org/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/v3/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/constants"
	msg "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2"
)

// SendTableParser provides functions for parsing send-tables.
//
// Intended for internal use only.
type SendTableParser struct {
	sendTables         []sendTable
	serverClasses      serverClasses
	currentExcludes    []*excludeEntry
	currentBaseclasses []*serverClass

	instanceBaselines map[int][]byte // Maps server-class IDs to raw instance baselines, needed for when we don't have the server-class when setting the baseline
}

// the following funcs are S2 only

func (p *SendTableParser) OnDemoClassInfo(*msgs2.CDemoClassInfo) error {
	panic("not implemented")
}

func (p *SendTableParser) OnServerInfo(*msgs2.CSVCMsg_ServerInfo) error {
	panic("not implemented")
}

func (p *SendTableParser) OnPacketEntities(*msgs2.CSVCMsg_PacketEntities) error {
	panic("not implemented")
}

// EntityOp is a bitmask representing the type of operation performed on an Entity
type EntityOp int

const (
	EntityOpNone           EntityOp = 0x00
	EntityOpCreated        EntityOp = 0x01
	EntityOpUpdated        EntityOp = 0x02
	EntityOpDeleted        EntityOp = 0x04
	EntityOpEntered        EntityOp = 0x08
	EntityOpLeft           EntityOp = 0x10
	EntityOpCreatedEntered EntityOp = EntityOpCreated | EntityOpEntered
	EntityOpUpdatedEntered EntityOp = EntityOpUpdated | EntityOpEntered
	EntityOpDeletedLeft    EntityOp = EntityOpDeleted | EntityOpLeft
)

var entityOpNames = map[EntityOp]string{
	EntityOpNone:           "None",
	EntityOpCreated:        "Created",
	EntityOpUpdated:        "Updated",
	EntityOpDeleted:        "Deleted",
	EntityOpEntered:        "Entered",
	EntityOpLeft:           "Left",
	EntityOpCreatedEntered: "Created+Entered",
	EntityOpUpdatedEntered: "Updated+Entered",
	EntityOpDeletedLeft:    "Deleted+Left",
}

// Flag determines whether an EntityOp includes another. This is primarily
// offered to prevent bit flag errors in downstream clients.
func (o EntityOp) Flag(p EntityOp) bool {
	return o&p != 0
}

// String returns a human identifiable string for the EntityOp
func (o EntityOp) String() string {
	return entityOpNames[o]
}

// EntityHandler is a function that receives Entity updates
type EntityHandler func(Entity, EntityOp) error

func (p *SendTableParser) OnEntity(h EntityHandler) {
	panic("not implemented")
}

// ServerClasses is a searchable list of ServerClasses.
type ServerClasses interface {
	All() []ServerClass
	FindByName(name string) ServerClass
}

type serverClasses []*serverClass

func (sc serverClasses) findByDataTableName(name string) *serverClass {
	for _, v := range sc {
		if v.DataTableName() == name {
			return v
		}
	}

	return nil
}

// FindByName finds and returns a server-class by it's name.
//
// Returns nil if the server-class wasn't found.
func (sc serverClasses) FindByName(name string) ServerClass {
	for _, v := range sc {
		if v.Name() == name {
			return v
		}
	}

	return nil
}

// All returns all server-classes.
func (sc serverClasses) All() (res []ServerClass) {
	for _, v := range sc {
		res = append(res, v)
	}

	return
}

type excludeEntry struct {
	varName       string
	dataTableName string
	excludingDt   string
}

// ServerClasses returns the parsed server-classes.
//
// Intended for internal use only.
func (p *SendTableParser) ServerClasses() ServerClasses {
	return p.serverClasses
}

// ParsePacket parses a send-table packet.
//
// Intended for internal use only.
func (p *SendTableParser) ParsePacket(b []byte) error {
	r := bit.NewSmallBitReader(bytes.NewReader(b))

	for {
		t := msg.SVC_Messages(r.ReadVarInt32())
		if t != msg.SVC_Messages_svc_SendTable {
			panic(fmt.Sprintf("Expected SendTable (%s), got %q", msg.SVC_Messages_svc_SendTable, t))
		}

		st := parseSendTable(r)
		if st.isEnd {
			break
		}

		p.sendTables = append(p.sendTables, st)
	}

	serverClassCount := int(r.ReadInt(16))

	for i := 0; i < serverClassCount; i++ {
		class := new(serverClass)
		class.id = int(r.ReadInt(16))

		if class.id > serverClassCount {
			panic("Invalid class index")
		}

		class.name = r.ReadString()
		class.dataTableName = r.ReadString()

		for j, v := range p.sendTables {
			if v.name == class.dataTableName {
				class.dataTableID = j
			}
		}

		class.instanceBaseline = p.instanceBaselines[i]

		p.serverClasses = append(p.serverClasses, class)
	}

	for i := 0; i < serverClassCount; i++ {
		p.flattenDataTable(i)
	}

	return nil
}

func parseSendTable(r *bit.BitReader) sendTable {
	var st msg.CSVCMsg_SendTable

	size := int(r.ReadVarInt32())

	err := proto.Unmarshal(r.ReadBytes(size), &st)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal SendTable: %s", err.Error()))
	}

	var res sendTable

	for _, v := range st.GetProps() {
		var prop sendTableProperty
		prop.dataTableName = v.GetDtName()
		prop.highValue = v.GetHighValue()
		prop.lowValue = v.GetLowValue()
		prop.name = v.GetVarName()
		prop.numberOfBits = int(v.GetNumBits())
		prop.numberOfElements = int(v.GetNumElements())
		prop.priority = int(v.GetPriority())
		prop.flags = sendPropertyFlags(v.GetFlags())
		prop.rawType = int(v.GetType())

		res.properties = append(res.properties, prop)
	}

	res.name = st.GetNetTableName()
	res.isEnd = st.GetIsEnd()

	return res
}

func (p *SendTableParser) flattenDataTable(serverClassIndex int) {
	tab := &p.sendTables[p.serverClasses[serverClassIndex].dataTableID]

	p.currentExcludes = nil
	p.currentBaseclasses = nil

	p.gatherExcludesAndBaseClasses(tab, true)
	p.serverClasses[serverClassIndex].baseClasses = p.currentBaseclasses

	p.gatherProps(tab, serverClassIndex, "")

	fProps := p.serverClasses[serverClassIndex].flattenedProps
	prios := sortProperyPrios(fProps)

	// I honestly have no idea what the following bit of code does but the statshelix guys do it too (please don't sue me)
	start := 0

	for _, prio := range prios {
		for {
			cp := start
			for ; cp < len(fProps); cp++ {
				prop := fProps[cp].prop
				if prop.priority == prio || (prio == 64 && prop.flags.hasFlagSet(propFlagChangesOften)) {
					if start != cp {
						fProps[start], fProps[cp] = fProps[cp], fProps[start]
					}

					start++

					break
				}
			}

			if cp == len(fProps) {
				break
			}
		}
	}

	p.serverClasses[serverClassIndex].propNameToIndex = buildPropertyLookupTable(p.serverClasses[serverClassIndex].flattenedProps)
}

func buildPropertyLookupTable(props []flattenedPropEntry) map[string]int {
	lookupTable := make(map[string]int)

	for i := range props {
		propName := props[i].name
		lookupTable[propName] = i
	}

	return lookupTable
}

func sortProperyPrios(fProps []flattenedPropEntry) []int {
	prioSet := make(map[int]struct{})
	prioSet[64] = struct{}{}

	for _, v := range fProps {
		prioSet[v.prop.priority] = struct{}{}
	}

	prios := make([]int, len(prioSet))

	i := 0
	for k := range prioSet { //nolint:wsl
		prios[i] = k
		i++
	}

	sort.Ints(prios)

	return prios
}

func (p *SendTableParser) gatherExcludesAndBaseClasses(st *sendTable, collectBaseClasses bool) {
	for _, v := range st.properties {
		if v.flags.hasFlagSet(propFlagExclude) {
			p.currentExcludes = append(p.currentExcludes, &excludeEntry{varName: v.name, dataTableName: v.dataTableName, excludingDt: st.name})
		}
	}

	for _, v := range st.properties {
		if v.rawType == propTypeDataTable {
			if collectBaseClasses && v.name == "baseclass" {
				p.gatherExcludesAndBaseClasses(p.getTableByName(v.dataTableName), true)
				p.currentBaseclasses = append(p.currentBaseclasses, p.serverClasses.findByDataTableName(v.dataTableName))
			} else {
				p.gatherExcludesAndBaseClasses(p.getTableByName(v.dataTableName), false)
			}
		}
	}
}

func (p *SendTableParser) gatherProps(st *sendTable, serverClassIndex int, prefix string) {
	var tmpFlattenedProps []flattenedPropEntry
	p.gatherPropsIterate(st, serverClassIndex, prefix, &tmpFlattenedProps) //nolint:wsl
	p.serverClasses[serverClassIndex].flattenedProps = append(p.serverClasses[serverClassIndex].flattenedProps, tmpFlattenedProps...)
}

func (p *SendTableParser) gatherPropsIterate(tab *sendTable, serverClassIndex int, prefix string, flattenedProps *[]flattenedPropEntry) {
	for i := range tab.properties {
		prop := &tab.properties[i]
		if prop.flags.hasFlagSet(propFlagInsideArray) || prop.flags.hasFlagSet(propFlagExclude) || p.isPropertyExcluded(tab, prop) {
			continue
		}

		if prop.rawType == propTypeDataTable {
			subTab := p.getTableByName(prop.dataTableName)

			if prop.flags.hasFlagSet(propFlagCollapsible) {
				p.gatherPropsIterate(subTab, serverClassIndex, prefix, flattenedProps)
			} else {
				nfix := prefix
				if len(prop.name) > 0 {
					nfix += prop.name + "."
				}
				p.gatherProps(subTab, serverClassIndex, nfix)
			}
		} else {
			if prop.rawType == propTypeArray {
				*flattenedProps = append(*flattenedProps, flattenedPropEntry{name: prefix + prop.name, prop: prop, arrayElementProp: &tab.properties[i-1]})
			} else {
				*flattenedProps = append(*flattenedProps, flattenedPropEntry{name: prefix + prop.name, prop: prop})
			}
		}
	}
}

func (p *SendTableParser) isPropertyExcluded(tab *sendTable, prop *sendTableProperty) bool {
	for _, v := range p.currentExcludes {
		if v.dataTableName == tab.name && v.varName == prop.name {
			return true
		}
	}

	return false
}

func (p *SendTableParser) getTableByName(name string) *sendTable {
	for i := range p.sendTables {
		if p.sendTables[i].name == name {
			return &p.sendTables[i]
		}
	}

	return nil
}

// SetInstanceBaseline sets the raw instance-baseline data for a serverclass by ID.
//
// Intended for internal use only.
func (p *SendTableParser) SetInstanceBaseline(scID int, data []byte) {
	if len(p.serverClasses) > scID {
		p.serverClasses[scID].instanceBaseline = data
	} else {
		p.instanceBaselines[scID] = data
	}
}

// ReadEnterPVS reads an entity entering the PVS (potentially visible system).
//
// Intended for internal use only.
func (p *SendTableParser) ReadEnterPVS(r *bit.BitReader, entityID int, existingEntities map[int]Entity, recordingPlayerSlot int) Entity {
	classID := int(r.ReadInt(p.classBits()))
	serialNum := int(r.ReadInt(constants.EntityHandleSerialNumberBits))
	existingEntity := existingEntities[entityID]

	if existingEntity != nil && existingEntity.SerialNum() == serialNum {
		existingEntity.ApplyUpdate(r)
		return existingEntity
	}

	// Serial numbers are different, delete the entity
	if existingEntity != nil {
		existingEntity.Destroy()
		delete(existingEntities, entityID)
	}

	return p.serverClasses[classID].newEntity(r, entityID, serialNum, recordingPlayerSlot)
}

// classBits seems to calculate how many bits must be read for the server-class ID.
// Not 100% sure how tho tbh.
func (p *SendTableParser) classBits() int {
	return int(math.Ceil(math.Log2(float64(len(p.serverClasses)))))
}

// NewSendTableParser returns a new SendTableParser.
//
// Intended for internal use only.
func NewSendTableParser() *SendTableParser {
	return &SendTableParser{
		instanceBaselines: make(map[int][]byte),
	}
}
