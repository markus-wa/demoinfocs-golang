package common

import (
	"github.com/golang/geo/r3"

	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// HostageState is the type for the various HostageStateXYZ constants.
type HostageState byte

// HostageState constants give information about hostages state.
// e.g. being untied, picked up, rescued etc.
const (
	HostageStateIdle            HostageState = 0
	HostageStateBeingUntied     HostageState = 1
	HostageStateGettingPickedUp HostageState = 2
	HostageStateBeingCarried    HostageState = 3
	HostageStateFollowingPlayer HostageState = 4
	HostageStateGettingDropped  HostageState = 5
	HostageStateRescued         HostageState = 6
	HostageStateDead            HostageState = 7
)

// Hostage represents a hostage.
type Hostage struct {
	Entity           st.Entity
	demoInfoProvider demoInfoProvider
}

// Position returns the current position of the hostage.
func (hostage *Hostage) Position() r3.Vector {
	if hostage.Entity == nil {
		return r3.Vector{}
	}

	return hostage.Entity.Position()
}

// State returns the current hostage's state.
// e.g. being untied, picked up, rescued etc.
// See HostageState for all possible values.
func (hostage *Hostage) State() HostageState {
	return HostageState(getInt(hostage.Entity, "m_nHostageState"))
}

// Health returns the hostage's health points.
// ! On Valve MM matches hostages are invulnerable, it will always return 100 unless "mp_hostages_takedamage" is set to 1
func (hostage *Hostage) Health() int {
	return getInt(hostage.Entity, "m_iHealth")
}

// Leader returns the possible player leading the hostage.
// Returns nil if the hostage is not following a player.
func (hostage *Hostage) Leader() *Player {
	if hostage.demoInfoProvider.IsSource2() {
		return hostage.demoInfoProvider.FindPlayerByPawnHandle(getUInt64(hostage.Entity, "m_leader"))
	}

	return hostage.demoInfoProvider.FindPlayerByHandle(uint64(getInt(hostage.Entity, "m_leader")))
}

// NewHostage creates a hostage.
func NewHostage(demoInfoProvider demoInfoProvider, entity st.Entity) *Hostage {
	return &Hostage{
		demoInfoProvider: demoInfoProvider,
		Entity:           entity,
	}
}
