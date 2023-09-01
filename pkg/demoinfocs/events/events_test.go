package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

func TestPlayerFlashed_FlashDuration(t *testing.T) {
	p := common.NewPlayer(demoInfoProviderMock{})
	e := PlayerFlashed{Player: p}

	assert.Equal(t, time.Duration(0), e.FlashDuration())

	p.FlashDuration = 2.3

	assert.Equal(t, 2300*time.Millisecond, e.FlashDuration())
}

func TestGrenadeEvent_Base(t *testing.T) {
	base := GrenadeEvent{GrenadeEntityID: 1}
	flashEvent := FlashExplode{base}

	assert.Equal(t, base, flashEvent.Base())
}

func TestBombEvents(t *testing.T) {
	events := []BombEventIf{
		BombDefuseStart{},
		BombDefuseAborted{},
		BombDefused{},
		BombExplode{},
		BombPlantBegin{},
		BombPlanted{},
	}

	for _, e := range events {
		e.implementsBombEventIf()
	}
}

func TestRankUpdate_SteamID64(t *testing.T) {
	event := RankUpdate{SteamID32: 52686539}

	assert.Equal(t, uint64(76561198012952267), event.SteamID64())
}

func TestKill_IsWallBang(t *testing.T) {
	event := Kill{PenetratedObjects: 1}

	assert.True(t, event.IsWallBang())
}

type demoInfoProviderMock struct {
	isSource2 bool
}

func (p demoInfoProviderMock) FindEntityByHandle(handle uint64) st.Entity {
	panic("implement me")
}

func (p demoInfoProviderMock) IngameTick() int {
	return 0
}

func (p demoInfoProviderMock) IsSource2() bool {
	return p.isSource2
}

func (p demoInfoProviderMock) TickRate() float64 {
	return 128
}

func (p demoInfoProviderMock) FindPlayerByHandle(int) *common.Player {
	return nil
}

func (p demoInfoProviderMock) FindPlayerByPawnHandle(uint64) *common.Player {
	return nil
}

func (p demoInfoProviderMock) PlayerResourceEntity() st.Entity {
	return nil
}

func (p demoInfoProviderMock) FindWeaponByEntityID(int) *common.Equipment {
	return nil
}
