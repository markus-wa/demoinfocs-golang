package common

import (
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

type Smoke struct {
	Entity         st.Entity
	IsActive       bool
	ActivationTick int

	demoInfoProvider demoInfoProvider
	thrower          *Player
}

// Thrower returns the player who threw the smoke grenade.
// Could be nil if the player disconnected after throwing it.
func (smk *Smoke) Thrower() *Player {
	if smk.thrower != nil {
		return smk.thrower
	}

	handleProp := smk.Entity.Property("m_hOwnerEntity").Value()
	if smk.demoInfoProvider.IsSource2() {
		return smk.demoInfoProvider.FindPlayerByPawnHandle(handleProp.Handle())
	}

	return smk.demoInfoProvider.FindPlayerByHandle(handleProp.Int())
}

func (smk *Smoke) ExpirationTick() int {
	if !smk.IsActive {
		return -1
	}
	return smk.ActivationTick + 1412
}

func NewSmoke(demoInfoProvider demoInfoProvider, entity st.Entity, thrower *Player) *Smoke {
	return &Smoke{
		Entity:           entity,
		IsActive:         false,
		ActivationTick:   -1,
		demoInfoProvider: demoInfoProvider,
		thrower:          thrower,
	}
}
