package sendtables

import (
	"fmt"
	"math"
	"sort"

	proto "github.com/gogo/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

// SendTableParser provides functions for parsing send-tables.
//
// Intended for internal use only.
type SendTableParser struct {
	sendTables         []sendTable
	serverClasses      ServerClasses
	currentExcludes    []*excludeEntry
	currentBaseclasses []*ServerClass

	instanceBaselines map[int][]byte // Maps server-class IDs to raw instance baselines, needed for when we don't have the server-class when setting the baseline
}

// ServerClasses is a searchable list of ServerClasses.
type ServerClasses []*ServerClass

func (sc ServerClasses) findByDataTableName(name string) *ServerClass {
	var res *ServerClass
	for _, v := range sc {
		if v.dataTableName == name {
			if res != nil {
				panic(fmt.Sprintf("More than one server class with DT name %q found", name))
			}
			res = v
		}
	}
	return res
}

// FindByName finds and returns a server-class by it's name.
//
// Returns nil if the server-class wasn't found.
//
// Panics if more than one server-class with the same name was found.
func (sc ServerClasses) FindByName(name string) *ServerClass {
	var res *ServerClass
	for _, v := range sc {
		if v.name == name {
			if res != nil {
				panic(fmt.Sprintf("More than one server class with name %q found", name))
			}
			res = v
		}
	}
	return res
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
func (p *SendTableParser) ParsePacket(r *bit.BitReader) {
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
		class := new(ServerClass)
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
}

func parseSendTable(r *bit.BitReader) sendTable {
	size := int(r.ReadVarInt32())
	r.BeginChunk(size << 3)
	st := new(msg.CSVCMsg_SendTable)
	if err := proto.Unmarshal(r.ReadBytes(size), st); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal SendTable: %s", err.Error()))
	}
	r.EndChunk()

	var res sendTable
	for _, v := range st.GetProps() {
		var prop sendTableProperty
		prop.dataTableName = v.DtName
		prop.highValue = v.HighValue
		prop.lowValue = v.LowValue
		prop.name = v.VarName
		prop.numberOfBits = int(v.NumBits)
		prop.numberOfElements = int(v.NumElements)
		prop.priority = int(v.Priority)
		prop.flags = sendPropertyFlags(v.Flags)
		prop.rawType = int(v.Type)

		res.properties = append(res.properties, prop)
	}

	res.name = st.NetTableName
	res.isEnd = st.IsEnd

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

	// Sort priorities
	prioSet := make(map[int]struct{})
	prioSet[64] = struct{}{}
	for _, v := range fProps {
		prioSet[v.prop.priority] = struct{}{}
	}
	prios := make([]int, len(prioSet))
	{
		i := 0
		for k := range prioSet {
			prios[i] = k
			i++
		}
	}
	sort.Ints(prios)

	// I honestly have no idea what the following bit of code does but the statshelix guys do it too (please don't sue me)
	start := 0
	for _, prio := range prios {
		for {
			cp := start
			for ; cp < len(fProps); cp++ {
				prop := fProps[cp].prop
				if prop.priority == prio || (prio == 64 && prop.flags.hasFlagSet(propFlagChangesOften)) {
					if start != cp {
						tmp := fProps[start]
						fProps[start] = fProps[cp]
						fProps[cp] = tmp
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
	p.gatherPropsIterate(st, serverClassIndex, prefix, &tmpFlattenedProps)
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
func (p *SendTableParser) ReadEnterPVS(r *bit.BitReader, entityID int) *Entity {
	scID := int(r.ReadInt(p.classBits()))
	r.Skip(10) // Serial Number

	return p.serverClasses[scID].newEntity(r, entityID)
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
	p := new(SendTableParser)
	p.instanceBaselines = make(map[int][]byte)
	return p
}
