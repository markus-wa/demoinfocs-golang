package demoinfocs_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// brokenDemPath is a CS2 GOTV demo showing both player-identity regressions of #692
// (reconnecting player's name reverted by stale userinfo updates; a re-used bot slot's
// name binding the same *Player to two user-IDs, non-deterministically). It lives in the
// cs-demos submodule and is skipped if the submodule isn't initialized.
const brokenDemPath = csDemosPath + "/s2/broken_1.dem"

func openBrokenDem(t *testing.T) demoinfocs.Parser {
	t.Helper()

	if _, err := os.Stat(brokenDemPath); err != nil {
		t.Skipf("test data %q not available, is the cs-demos submodule initialized?", brokenDemPath)
	}

	f, err := os.Open(brokenDemPath)
	require.NoError(t, err, "error opening demo %q", brokenDemPath)
	t.Cleanup(func() { mustClose(t, f) })

	return demoinfocs.NewParser(f)
}

// Test692NameChangeNotReverted asserts that a reconnecting player's name change
// is applied exactly once and never reverted by stale GOTV userinfo updates.
// See https://github.com/markus-wa/demoinfocs-golang/issues/692
func Test692NameChangeNotReverted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	t.Parallel()

	p := openBrokenDem(t)

	type change struct {
		oldName, newName string
	}

	changes := make(map[int][]change)

	p.RegisterEventHandler(func(e events.PlayerNameChange) {
		changes[e.Player.UserID] = append(changes[e.Player.UserID],
			change{oldName: e.OldName, newName: e.NewName})
	})

	require.NoError(t, p.ParseToEnd())

	// uid 4 disconnects and reconnects under a new name - the change must stick (applied once).
	assert.Equal(t, []change{{oldName: "ZoRoBeast", newName: "- [Z]oRo"}},
		changes[4], "uid 4's reconnect name change must be applied exactly once and not reverted")

	// uid 9 reconnects under a new name mid-round (same tick as the string-table update) -
	// the connect event's name must win and never revert.
	assert.Equal(t, []change{{oldName: "iBaaaaaaad", newName: "BuddhaMonsteRRR"}},
		changes[9], "uid 9's reconnect name change must be applied exactly once and not reverted")

	for uid, c := range changes {
		for i, ch := range c {
			assert.NotEqual(t, ch.oldName, ch.newName, "uid %d change %d: no-op change dispatched", uid, i)

			for _, prev := range c[:i] {
				assert.NotEqual(t, prev.newName, ch.oldName,
					"uid %d change %d: name %q was reverted by a stale update", uid, i, prev.newName)
			}
		}
	}
}

// Test692NoDuplicatePlayerInPlaying asserts that GameState.Participants().Playing()
// never contains the same *common.Player twice after a bot slot's name is re-used.
// See https://github.com/markus-wa/demoinfocs-golang/issues/692
func Test692NoDuplicatePlayerInPlaying(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	t.Parallel()

	p := openBrokenDem(t)

	seen := make(map[int]*common.Player)
	last := -1

	p.RegisterEventHandler(func(events.FrameDone) {
		tick := p.GameState().IngameTick()
		if tick == last {
			return
		}

		last = tick
		clear(seen)

		for _, pl := range p.GameState().Participants().Playing() {
			other, dup := seen[pl.UserID]
			if dup {
				assert.NotSame(t, other, pl,
					"tick %d: Playing() contains the same *Player twice under user-id %d", tick, pl.UserID)

				continue
			}

			seen[pl.UserID] = pl
		}
	})

	require.NoError(t, p.ParseToEnd())
}
