package demoinfocs_test

import (
	"math"
	"slices"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// s2.dem is a GOTV demo recorded before the AnimGraph 2 update, m_vecVelocity isn't networked,
// so with ParserConfig.EagerVelocityTracking Player.Velocity() falls back to the position delta
// between frames (3 ticks apart).
func TestPlayerVelocity_PositionDeltaFallback(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, s2DemPath)
	defer mustClose(t, f)

	cfg := demoinfocs.DefaultParserConfig
	cfg.EagerVelocityTracking = true
	p := demoinfocs.NewParserWithConfig(f, cfg)

	type sample struct {
		pos  r3.Vector
		tick int
	}

	var (
		speeds []float64
		ratios []float64 // derived speed / speed from the raw position delta between frames
		last   = map[int]sample{}
	)

	p.RegisterEventHandler(func(events.FrameDone) {
		tick := p.GameState().IngameTick()

		for _, pl := range p.GameState().Participants().Playing() {
			if !pl.IsAlive() {
				delete(last, pl.EntityID)

				continue
			}

			v := pl.Velocity()
			speed := math.Hypot(v.X, v.Y)
			speeds = append(speeds, speed)

			pos := pl.Position()
			if prev, ok := last[pl.EntityID]; ok && tick > prev.tick {
				delta := pos.Sub(prev.pos)
				rawSpeed := math.Hypot(delta.X, delta.Y) / (float64(tick-prev.tick) / p.TickRate())

				if speed > 100 && rawSpeed > 100 {
					ratios = append(ratios, speed/rawSpeed)
				}
			}

			last[pl.EntityID] = sample{pos: pos, tick: tick}
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

	slices.Sort(ratios)
	ratioP50 := ratios[len(ratios)/2]

	t.Logf("samples=%d moving=%d p99=%.1f max=%.1f ratio to raw delta p50=%.3f", len(speeds), moving, p99, maxSpeed, ratioP50)

	// players run at most at 250 u/s (knife) - no teleport spikes
	assert.Less(t, maxSpeed, 350.0)
	assert.Less(t, p99, 260.0)
	// a good share of alive player-frames is moving
	assert.Greater(t, float64(moving)/float64(len(speeds)), 0.3)
	// GOTV demos have a snapshot every frame, so the derived velocity matches the frame-to-frame position delta
	assert.InDelta(t, 1.0, ratioP50, 0.02)
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
