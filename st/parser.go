package st

import (
	"github.com/gogo/protobuf/proto"
	bs "github.com/markus-wa/demoinfocs-golang/bitstream"
	"github.com/markus-wa/demoinfocs-golang/msg"
	"math"
	"sort"
)

type Parser struct {
	sendTables         []SendTable
	serverClasses      []*ServerClass
	currentExcludes    []*ExcludeEntry
	currentBaseclasses []*ServerClass
}

func (p *Parser) ClassBits() int {
	return int(math.Ceil(math.Log2(float64(len(p.serverClasses)))))
}

func (p *Parser) ServerClasses() []*ServerClass {
	return p.serverClasses
}

func (p *Parser) ParsePacket(r *bs.BitReader) {
	for {
		t := msg.SVC_Messages(r.ReadVarInt32())
		if t != msg.SVC_Messages_svc_SendTable {
			panic("Expected SendTable (" + string(msg.SVC_Messages_svc_SendTable) + "), got" + string(t))
		}

		st := parseSendTable(r)
		if st.IsEnd {
			break
		}
		p.sendTables = append(p.sendTables, st)
	}

	serverClassCount := int(r.ReadInt(16))

	for i := 0; i < serverClassCount; i++ {
		entry := new(ServerClass)
		entry.ClassId = int(r.ReadInt(16))
		if entry.ClassId > serverClassCount {
			panic("Invalid class index")
		}

		entry.Name = r.ReadString()
		entry.DTName = r.ReadString()
		for j, v := range p.sendTables {
			if v.Name == entry.DTName {
				entry.DataTableId = j
			}
		}

		p.serverClasses = append(p.serverClasses, entry)
	}

	for i := 0; i < serverClassCount; i++ {
		p.flattenDataTable(i)
	}
}

func parseSendTable(r *bs.BitReader) SendTable {
	size := int(r.ReadVarInt32())
	r.BeginChunk(size * 8)
	st := new(msg.CSVCMsg_SendTable)
	proto.Unmarshal(r.ReadBytes(size), st)
	r.EndChunk()

	var res SendTable
	for _, v := range st.GetProps() {
		var prop SendTableProperty
		prop.DataTableName = v.DtName
		prop.HighValue = v.HighValue
		prop.LowValue = v.LowValue
		prop.Name = v.VarName
		prop.NumberOfBits = int(v.NumBits)
		prop.NumberOfElements = int(v.NumElements)
		prop.Priority = int(v.Priority)
		prop.Flags = SendPropertyFlags(v.Flags)
		prop.RawType = int(v.Type)

		res.properties = append(res.properties, prop)
	}

	res.Name = st.NetTableName
	res.IsEnd = st.IsEnd

	return res
}

func (p *Parser) flattenDataTable(serverClassIndex int) {
	tab := &p.sendTables[p.serverClasses[serverClassIndex].DataTableId]

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
		prioMap[v.prop.Priority] = struct{}{}
	}
	var prios []int
	for k := range prioMap {
		prios = append(prios, k)
	}
	sort.Ints(prios)

	// I honestly have no idea what the following bit of code does but the statshelix guys do it too (please don't sue me)
	start := 0
	for _, prio := range prios {
		for true {
			cp := start
			for ; cp < len(fProps); cp++ {
				prop := fProps[cp].prop
				if prop.Priority == prio || (prio == 64 && prop.Flags.HasFlagSet(SPF_ChangesOften)) {
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

func (p *Parser) gatherExcludesAndBaseClasses(st *SendTable, collectBaseClasses bool) {
	for _, v := range st.properties {
		if v.Flags.HasFlagSet(SPF_Exclude) {
			p.currentExcludes = append(p.currentExcludes, &ExcludeEntry{varName: v.Name, dtName: v.DataTableName, excludingDt: st.Name})
		}
	}

	for _, v := range st.properties {
		if v.RawType == SPT_DataTable {
			if collectBaseClasses && v.Name == "baseclass" {
				p.gatherExcludesAndBaseClasses(p.getTableByName(v.DataTableName), true)
				p.currentBaseclasses = append(p.currentBaseclasses, p.findServerClassByDtName(v.DataTableName))
			} else {
				p.gatherExcludesAndBaseClasses(p.getTableByName(v.DataTableName), false)
			}
		}
	}
}

func (p *Parser) gatherProps(st *SendTable, serverClassIndex int, prefix string) {
	var tmpFlattenedProps []FlattenedPropEntry
	p.gatherPropsIterate(st, serverClassIndex, prefix, &tmpFlattenedProps)
	p.serverClasses[serverClassIndex].FlattenedProps = append(p.serverClasses[serverClassIndex].FlattenedProps, tmpFlattenedProps...)
}

func (p *Parser) gatherPropsIterate(tab *SendTable, serverClassIndex int, prefix string, flattenedProps *[]FlattenedPropEntry) {
	for i, _ := range tab.properties {
		prop := &tab.properties[i]
		if prop.Flags.HasFlagSet(SPF_InsideArray) || prop.Flags.HasFlagSet(SPF_Exclude) || p.isPropertyExcluded(tab, prop) {
			continue
		}

		if prop.RawType == SPT_DataTable {
			subTab := p.getTableByName(prop.DataTableName)

			if prop.Flags.HasFlagSet(SPF_Collapsible) {
				p.gatherPropsIterate(subTab, serverClassIndex, prefix, flattenedProps)
			} else {
				nfix := prefix
				if len(prop.Name) > 0 {
					nfix += prop.Name + "."
				}
				p.gatherProps(subTab, serverClassIndex, nfix)
			}
		} else {
			if prop.RawType == SPT_Array {
				*flattenedProps = append(*flattenedProps, FlattenedPropEntry{name: prefix + prop.Name, prop: prop, arrayElementProp: &tab.properties[i-1]})
			} else {
				*flattenedProps = append(*flattenedProps, FlattenedPropEntry{name: prefix + prop.Name, prop: prop})
			}
		}
	}
}

func (p *Parser) isPropertyExcluded(tab *SendTable, prop *SendTableProperty) bool {
	for _, v := range p.currentExcludes {
		if v.dtName == tab.Name && v.varName == prop.Name {
			return true
		}
	}
	return false
}

func (p *Parser) getTableByName(name string) *SendTable {
	for i, _ := range p.sendTables {
		if p.sendTables[i].Name == name {
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
				panic("More than one server class with DT name " + name + "found")
			}
			sc = v
		}
	}
	if sc == nil {
		panic("No server class with DT name " + name + " found")
	}
	return sc
}

func (p *Parser) FindServerClassByName(name string) *ServerClass {
	var sc *ServerClass
	for _, v := range p.serverClasses {
		if v.Name == name {
			if sc != nil {
				panic("More than one server class with name " + name + "found")
			}
			sc = v
		}
	}
	if sc == nil {
		panic("No server class with name " + name + " found")
	}
	return sc
}
