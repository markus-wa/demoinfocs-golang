package sendtables

import (
	"fmt"
	"math"
	"sort"

	proto "github.com/gogo/protobuf/proto"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
)

// Parser provides functions for parsing send-tables.
type Parser struct {
	sendTables         []sendTable
	serverClasses      []*ServerClass
	currentExcludes    []*excludeEntry
	currentBaseclasses []*ServerClass
}

// ClassBits seems to calculate how many bits must be read for the server-class ID.
// Not 100% sure how tho tbh.
func (p *Parser) ClassBits() int {
	return int(math.Ceil(math.Log2(float64(len(p.serverClasses)))))
}

// ServerClasses returns the parsed server-classes
func (p *Parser) ServerClasses() []*ServerClass {
	return p.serverClasses
}

// ParsePacket parses a send-table packet.
func (p *Parser) ParsePacket(r *bit.BitReader) {
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
		entry := new(ServerClass)
		entry.ClassID = int(r.ReadInt(16))
		if entry.ClassID > serverClassCount {
			panic("Invalid class index")
		}

		entry.Name = r.ReadString()
		entry.DTName = r.ReadString()
		for j, v := range p.sendTables {
			if v.name == entry.DTName {
				entry.DataTableID = j
			}
		}

		p.serverClasses = append(p.serverClasses, entry)
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

func (p *Parser) flattenDataTable(serverClassIndex int) {
	tab := &p.sendTables[p.serverClasses[serverClassIndex].DataTableID]

	p.currentExcludes = nil
	p.currentBaseclasses = nil

	p.gatherExcludesAndBaseClasses(tab, true)
	p.serverClasses[serverClassIndex].BaseClasses = p.currentBaseclasses

	p.gatherProps(tab, serverClassIndex, "")

	fProps := p.serverClasses[serverClassIndex].FlattenedProps

	// Sort priorities
	prioMap := make(map[int]struct{})
	prioMap[64] = struct{}{}
	for _, v := range fProps {
		prioMap[v.prop.priority] = struct{}{}
	}
	var prios []int
	for k := range prioMap {
		prios = append(prios, k)
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

func (p *Parser) gatherExcludesAndBaseClasses(st *sendTable, collectBaseClasses bool) {
	for _, v := range st.properties {
		if v.flags.hasFlagSet(propFlagExclude) {
			p.currentExcludes = append(p.currentExcludes, &excludeEntry{varName: v.name, dtName: v.dataTableName, excludingDt: st.name})
		}
	}

	for _, v := range st.properties {
		if v.rawType == propTypeDataTable {
			if collectBaseClasses && v.name == "baseclass" {
				p.gatherExcludesAndBaseClasses(p.getTableByName(v.dataTableName), true)
				p.currentBaseclasses = append(p.currentBaseclasses, p.findServerClassByDtName(v.dataTableName))
			} else {
				p.gatherExcludesAndBaseClasses(p.getTableByName(v.dataTableName), false)
			}
		}
	}
}

func (p *Parser) gatherProps(st *sendTable, serverClassIndex int, prefix string) {
	var tmpFlattenedProps []FlattenedPropEntry
	p.gatherPropsIterate(st, serverClassIndex, prefix, &tmpFlattenedProps)
	p.serverClasses[serverClassIndex].FlattenedProps = append(p.serverClasses[serverClassIndex].FlattenedProps, tmpFlattenedProps...)
}

func (p *Parser) gatherPropsIterate(tab *sendTable, serverClassIndex int, prefix string, flattenedProps *[]FlattenedPropEntry) {
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
				*flattenedProps = append(*flattenedProps, FlattenedPropEntry{name: prefix + prop.name, prop: prop, arrayElementProp: &tab.properties[i-1]})
			} else {
				*flattenedProps = append(*flattenedProps, FlattenedPropEntry{name: prefix + prop.name, prop: prop})
			}
		}
	}
}

func (p *Parser) isPropertyExcluded(tab *sendTable, prop *sendTableProperty) bool {
	for _, v := range p.currentExcludes {
		if v.dtName == tab.name && v.varName == prop.name {
			return true
		}
	}
	return false
}

func (p *Parser) getTableByName(name string) *sendTable {
	for i := range p.sendTables {
		if p.sendTables[i].name == name {
			return &p.sendTables[i]
		}
	}
	if len(p.sendTables) > 0 {
		return &p.sendTables[0]
	}
	return nil
}

func (p *Parser) findServerClassByDtName(name string) *ServerClass {
	var sc *ServerClass
	for _, v := range p.serverClasses {
		if v.DTName == name {
			if sc != nil {
				panic(fmt.Sprintf("More than one server class with DT name %q found", name))
			}
			sc = v
		}
	}
	if sc == nil {
		panic(fmt.Sprintf("No server class with DT name %q found", name))
	}
	return sc
}

// FindServerClassByName finds and returns a server-class by it's name.
func (p *Parser) FindServerClassByName(name string) *ServerClass {
	var sc *ServerClass
	for _, v := range p.serverClasses {
		if v.Name == name {
			if sc != nil {
				panic(fmt.Sprintf("More than one server class with name %q found", name))
			}
			sc = v
		}
	}
	if sc == nil {
		panic(fmt.Sprintf("No server class with name %q found", name))
	}
	return sc
}
