package fake

import (
	"github.com/stretchr/testify/mock"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
)

var _ demoinfocs.Participants = new(Participants)

// Participants is a mock for of demoinfocs.Participants.
type Participants struct {
	mock.Mock
}

// ByUserID is a mock-implementation of Participants.ByUserID().
func (ptcp *Participants) ByUserID() map[int]*common.Player {
	return ptcp.Called().Get(0).(map[int]*common.Player)
}

// ByEntityID is a mock-implementation of Participants.ByEntityID().
func (ptcp *Participants) ByEntityID() map[int]*common.Player {
	return ptcp.Called().Get(0).(map[int]*common.Player)
}

// AllByUserID is a mock-implementation of Participants.AllByUserID().
func (ptcp *Participants) AllByUserID() map[int]*common.Player {
	return ptcp.Called().Get(0).(map[int]*common.Player)
}

// All is a mock-implementation of Participants.All().
func (ptcp *Participants) All() []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// Connected is a mock-implementation of Participants.Connected().
func (ptcp *Participants) Connected() []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// Playing is a mock-implementation of Participants.Playing().
func (ptcp *Participants) Playing() []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// TeamMembers is a mock-implementation of Participants.TeamMembers().
func (ptcp *Participants) TeamMembers(team common.Team) []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// FindByHandle is a mock-implementation of Participants.FindByHandle().
func (ptcp *Participants) FindByHandle(handle int) *common.Player {
	return ptcp.Called().Get(0).(*common.Player)
}

// SpottersOf is a mock-implementation of Participants.SpottersOf().
func (ptcp *Participants) SpottersOf(spotted *common.Player) []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}

// SpottedBy is a mock-implementation of Participants.SpottedBy().
func (ptcp *Participants) SpottedBy(spotter *common.Player) []*common.Player {
	return ptcp.Called().Get(0).([]*common.Player)
}
