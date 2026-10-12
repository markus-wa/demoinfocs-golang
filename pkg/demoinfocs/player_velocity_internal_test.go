package demoinfocs

import (
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

const povDemPathInternal = "../../test/cs-demos/s2/pov.dem"

// pov.dem is a POV demo recorded before the AnimGraph 2 update: the recording player's velocity is networked
// and must be used as-is, everybody else's is derived from position deltas.
// Recorders of players whose controller is gone must be dropped, so they don't pile up over reconnects.
func TestPlayerVelocity_POV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	f, err := os.Open(povDemPathInternal)
	require.NoError(t, err)

	defer f.Close()

	p := NewParser(f).(*parser)

	var (
		recordingSlot              = -1
		recordingChecked, recorded int
		maxRecorders, dropped      int
	)

	p.RegisterEventHandler(func(e events.POVRecordingPlayerDetected) {
		recordingSlot = e.PlayerSlot
	})

	p.RegisterEventHandler(func(events.FrameDone) {
		for _, pl := range p.GameState().Participants().All() {
			if !pl.IsAlive() {
				continue
			}

			v := pl.Velocity()

			if pl.EntityID-1 != recordingSlot {
				continue
			}

			pawn := pl.PlayerPawnEntity()
			x, ok := pawn.PropertyValue("m_vecVelocity.m_vecX")
			require.True(t, ok)

			if x.Any == nil {
				continue
			}

			recordingChecked++

			assert.InDelta(t, float64(x.Float()), v.X, 1e-6)
			assert.InDelta(t, float64(pawn.PropertyValueMust("m_vecVelocity.m_vecY").Float()), v.Y, 1e-6)

			if math.Hypot(v.X, v.Y) > 100 {
				recorded++
			}
		}

		maxRecorders = max(maxRecorders, len(p.framePositionRecorders))
		assert.LessOrEqual(t, len(p.framePositionRecorders), len(p.gameState.playerControllerEntities))
	})

	for {
		n := len(p.framePositionRecorders)

		more, err := p.ParseNextFrame()
		require.NoError(t, err)

		dropped += max(0, n-len(p.framePositionRecorders))

		if !more {
			break
		}
	}

	t.Logf("recording player samples=%d moving=%d, max recorders=%d, dropped=%d", recordingChecked, recorded, maxRecorders, dropped)

	assert.NotEqual(t, -1, recordingSlot)
	assert.Positive(t, recordingChecked)
	assert.Positive(t, recorded)
	assert.Positive(t, maxRecorders)
	// a player disconnects mid-demo and their controller is destroyed
	assert.Positive(t, dropped, "recorder of the disconnected player wasn't dropped")
}
