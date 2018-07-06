package demoinfocs

import (
	"strings"

	common "github.com/markus-wa/demoinfocs-golang/common"
)

var modelPrecacheNameToEq = []struct {
	name string
	eq   common.EquipmentElement
}{
	{"flashbang_dropped", common.EqFlash},
	{"fraggrenade_dropped", common.EqHE},
	{"smokegrenade_thrown", common.EqSmoke},
	{"molotov_dropped", common.EqMolotov},
	{"incendiarygrenade_dropped", common.EqIncendiary},
	{"decoy_dropped", common.EqDecoy},
	// @micvbang TODO: add all other weapons too.
}

func (p *Parser) processModelPreCacheUpdate() {
	for i, name := range p.modelPreCache {
		for _, nade := range modelPrecacheNameToEq {
			if strings.Contains(name, nade.name) {
				p.grenadeModelIndices[i] = nade.eq
			}
		}
	}
}
