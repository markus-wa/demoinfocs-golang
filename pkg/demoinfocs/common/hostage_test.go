package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	st "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/sendtables"
)

func TestHostage_Leader(t *testing.T) {
	player := new(Player)
	player.EntityID = 10
	provider := demoInfoProviderMock{
		playersByHandle: map[int]*Player{10: player},
	}
	hostage := hostageWithProperty("m_leader", st.PropertyValue{IntVal: 10}, provider)

	assert.Equal(t, player, hostage.Leader())
}

func TestHostage_State(t *testing.T) {
	hostage := hostageWithProperty("m_nHostageState", st.PropertyValue{IntVal: int(HostageStateFollowingPlayer)}, demoInfoProviderMock{})

	assert.Equal(t, HostageStateFollowingPlayer, hostage.State())
}

func TestHostage_Health(t *testing.T) {
	hostage := hostageWithProperty("m_iHealth", st.PropertyValue{IntVal: 40}, demoInfoProviderMock{})

	assert.Equal(t, 40, hostage.Health())
}

func hostageWithProperty(propName string, value st.PropertyValue, provider demoInfoProviderMock) *Hostage {
	return &Hostage{Entity: entityWithProperty(propName, value), demoInfoProvider: provider}
}
