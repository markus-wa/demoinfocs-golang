package common

import (
	"testing"
	"time"

	r3 "github.com/golang/geo/r3"
	assert "github.com/stretchr/testify/assert"
)

func TestBombPosition(t *testing.T) {
	groundPos := r3.Vector{X: 1, Y: 2, Z: 3}
	bomb := Bomb{
		LastOnGroundPosition: groundPos,
	}

	assert.Equal(t, groundPos, bomb.Position(), "Bomb position should be LastOnGroundPosition")

	playerPos := r3.Vector{X: 4, Y: 5, Z: 6}
	bomb.Carrier = &Player{Position: playerPos}
	assert.Equal(t, playerPos, bomb.Position(), "Bomb position should be Player.Position")
}

func TestGrenadeProjectileUniqueID(t *testing.T) {
	assert.NotEqual(t, NewGrenadeProjectile().UniqueID(), NewGrenadeProjectile().UniqueID(), "UniqueIDs of different grenade projectiles should be different")
}

func TestDemoHeader(t *testing.T) {
	header := DemoHeader{
		PlaybackFrames: 256,
		PlaybackTicks:  512,
		PlaybackTime:   4 * time.Second,
	}

	assert.Equal(t, float64(64), header.FrameRate(), "FrameRate should be 64")
	assert.Equal(t, time.Second/64, header.FrameTime(), "FrameTime should be 1/64 sec")

	assert.Equal(t, float64(128.0), header.TickRate(), "TickRate should be 128")
	assert.Equal(t, time.Second/128, header.TickTime(), "TickTime should be 1/128")
}

func TestTeamState(t *testing.T) {
	assert.Equal(t, TeamTerrorists, NewTeamState(TeamTerrorists).Team())
	assert.Equal(t, TeamCounterTerrorists, NewTeamState(TeamCounterTerrorists).Team())
}
