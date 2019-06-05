package fake

import (
	"github.com/stretchr/testify/mock"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
)

var _ dem.IParticipants = new(Participants)

// Participants is a mock for of demoinfocs.IParticipants.
type Participants struct {
	mock.Mock
}

// ByUserID is a mock-implementation of IParticipants.ByUserID().
func (ptcp *Participants) ByUserID() map[int]*common.Player {
	return ptcp.Called().Get(0).(map[int]*common.Player)
}

// ByEntityID is a mock-implementation of IParticipants.ByEntityID().
func (ptcp *Participants) ByEntityID() map[int]*common.Player {
	return ptcp.Called().Get(0).(map[int]*common.Player)
}

// All is a mock-implementation of IParticipants.All().
func (ptcp *Participants) All() []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// Connected is a mock-implementation of IParticipants.Connected().
func (ptcp *Participants) Connected() []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// Playing is a mock-implementation of IParticipants.Playing().
func (ptcp *Participants) Playing() []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// TeamMembers is a mock-implementation of IParticipants.TeamMembers().
func (ptcp *Participants) TeamMembers(team common.Team) []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// FindByHandle is a mock-implementation of IParticipants.FindByHandle().
func (ptcp *Participants) FindByHandle(handle int) *common.Player {
	return ptcp.Called().Get(0).(*common.Player)
}

// SpottersOf is a mock-implementation of IParticipants.SpottersOf().
func (ptcp *Participants) SpottersOf(spotted *common.Player) []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// SpottedBy is a mock-implementation of IParticipants.SpottedBy().
func (ptcp *Participants) SpottedBy(spotter *common.Player) []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}
