package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

func TestHostage_Leader(t *testing.T) {
	player := new(Player)
	player.EntityID = 10
	provider := demoInfoProviderMock{
		playersByHandle: map[uint64]*Player{10: player},
	}
	hostage := hostageWithProperty("m_leader", st.PropertyValue{Any: uint64(10)}, provider)

	assert.Equal(t, player, hostage.Leader())
}

func TestHostage_LeaderWithInvalidHandleS2(t *testing.T) {
	player := new(Player)
	player.EntityID = 10
	provider := demoInfoProviderMock{
		playersByHandle: map[uint64]*Player{10: player},
	}
	hostage := hostageWithProperties([]fakeProp{
		{
			propName: "m_leader",
			value:    st.PropertyValue{Any: uint64(constants.InvalidEntityHandleSource2)},
			isNil:    false,
		},
		{
			propName: "m_hHostageGrabber",
			value:    st.PropertyValue{Any: uint64(10)},
			isNil:    false,
		},
	}, provider)

	assert.Equal(t, player, hostage.Leader())
}

func TestHostage_State(t *testing.T) {
	hostage := hostageWithProperty("m_nHostageState", st.PropertyValue{Any: int32(HostageStateFollowingPlayer)}, demoInfoProviderMock{})

	assert.Equal(t, HostageStateFollowingPlayer, hostage.State())
}

func TestHostage_Health(t *testing.T) {
	hostage := hostageWithProperty("m_iHealth", st.PropertyValue{Any: int32(40)}, demoInfoProviderMock{})

	assert.Equal(t, 40, hostage.Health())
}

func hostageWithProperty(propName string, value st.PropertyValue, provider demoInfoProviderMock) *Hostage {
	return &Hostage{Entity: entityWithProperty(propName, value), demoInfoProvider: provider}
}

func hostageWithProperties(properties []fakeProp, provider demoInfoProviderMock) *Hostage {
	return &Hostage{Entity: entityWithProperties(properties), demoInfoProvider: provider}
}
