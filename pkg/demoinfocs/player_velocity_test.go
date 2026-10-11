package demoinfocs_test

import (
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// s2.dem is a GOTV demo recorded before the AnimGraph 2 update, m_vecVelocity isn't networked,
// so with ParserConfig.VelocityFromPositionDelta Player.Velocity() falls back to the position delta
// between frames (3 ticks apart).
func TestPlayerVelocity_PositionDeltaFallback(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, s2DemPath)
	defer mustClose(t, f)

	cfg := demoinfocs.DefaultParserConfig
	cfg.VelocityFromPositionDelta = true
	p := demoinfocs.NewParserWithConfig(f, cfg)

	var speeds []float64

	p.RegisterEventHandler(func(events.FrameDone) {
		for _, pl := range p.GameState().Participants().Playing() {
			if !pl.IsAlive() {
				continue
			}

			v := pl.Velocity()
			speeds = append(speeds, math.Hypot(v.X, v.Y))
		}
	})

	err := p.ParseToEnd()
	assert.NoError(t, err)

	slices.Sort(speeds)

	var moving int

	for _, s := range speeds {
		if s > 100 {
			moving++
		}
	}

	maxSpeed := speeds[len(speeds)-1]
	p99 := speeds[len(speeds)*99/100]

	t.Logf("samples=%d moving=%d p99=%.1f max=%.1f", len(speeds), moving, p99, maxSpeed)

	// players run at most at 250 u/s (knife) - no teleport spikes
	assert.Less(t, maxSpeed, 350.0)
	assert.Less(t, p99, 260.0)
	// a good share of alive player-frames is moving
	assert.Greater(t, float64(moving)/float64(len(speeds)), 0.3)
}

// Without the config flag, positions are recorded lazily once Velocity() is called for a player.
// The parser warns once, and from the second frame on the velocity is derived as with the flag.
func TestPlayerVelocity_LazyPositionTracking(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, s2DemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	var (
		lazyWarnings int
		maxSpeed     float64
		moving       int
	)

	p.RegisterEventHandler(func(w events.ParserWarn) {
		if w.Type == events.WarnTypeVelocityTrackingStartedLazily {
			lazyWarnings++
		}
	})

	p.RegisterEventHandler(func(events.FrameDone) {
		for _, pl := range p.GameState().Participants().Playing() {
			if !pl.IsAlive() {
				continue
			}

			v := pl.Velocity()
			s := math.Hypot(v.X, v.Y)
			maxSpeed = max(maxSpeed, s)

			if s > 100 {
				moving++
			}
		}
	})

	assert.NoError(t, p.ParseToEnd())
	assert.Equal(t, 1, lazyWarnings)
	assert.Less(t, maxSpeed, 350.0)
	assert.Positive(t, moving)
}
