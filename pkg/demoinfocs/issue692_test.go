package demoinfocs_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

// brokenDemPath is a CS2 GOTV demo showing both player-identity issues of #692
// (reconnecting players' names flip-flopping between their Steam & FACEIT names; a re-used
// bot slot's name binding the same *Player to two user-IDs, non-deterministically). It lives in the
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

// Test692ControllerNameIsAuthoritative asserts that player names come from the player controller
// (m_iszPlayerName, the server-enforced name shown on the scoreboard) and never flip-flop.
//
// The demo is from a FACEIT server: after reconnecting, two players' userinfo & player_connect carry
// their Steam names ("- [Z]oRo", "BuddhaMonsteRRR"), while the server keeps enforcing their FACEIT
// names ("ZoRoBeast", "iBaaaaaaad") - the controller never changes and the server's own
// player_disconnect at the end of the demo still says "ZoRoBeast". The Steam names are only available
// via PlayerInfo / player_connect.
// See https://github.com/markus-wa/demoinfocs-golang/issues/692
func Test692ControllerNameIsAuthoritative(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	t.Parallel()

	p := openBrokenDem(t)

	const (
		zoroSteamID = uint64(76561198870444951)
		ibadSteamID = uint64(76561199129120571)
	)

	var nameChanges []events.PlayerNameChange

	p.RegisterEventHandler(func(e events.PlayerNameChange) {
		nameChanges = append(nameChanges, e)
	})

	steamNamesSeen := make(map[uint64]map[string]bool)

	p.RegisterEventHandler(func(e events.PlayerInfo) {
		if steamNamesSeen[e.Info.XUID] == nil {
			steamNamesSeen[e.Info.XUID] = make(map[string]bool)
		}

		steamNamesSeen[e.Info.XUID][e.Info.Name] = true
	})

	namesBySteamID := make(map[uint64]map[string]bool)
	mismatches := 0

	p.RegisterEventHandler(func(events.FrameDone) {
		for _, pl := range p.GameState().Participants().Playing() {
			if pl.Entity == nil {
				continue
			}

			if ctrlName := pl.Entity.PropertyValueMust("m_iszPlayerName").String(); pl.Name != ctrlName {
				if mismatches == 0 {
					t.Errorf("tick %d: player name %q != controller name %q", p.GameState().IngameTick(), pl.Name, ctrlName)
				}

				mismatches++
			}

			if namesBySteamID[pl.SteamID64] == nil {
				namesBySteamID[pl.SteamID64] = make(map[string]bool)
			}

			namesBySteamID[pl.SteamID64][pl.Name] = true
		}
	})

	require.NoError(t, p.ParseToEnd())

	assert.Zero(t, mismatches, "frames where a player's name didn't match its controller")

	// the controller names never change in this demo, so neither may the players' names
	assert.Empty(t, nameChanges, "no PlayerNameChange expected (names must not flip-flop)")
	assert.Equal(t, map[string]bool{"ZoRoBeast": true}, namesBySteamID[zoroSteamID])
	assert.Equal(t, map[string]bool{"iBaaaaaaad": true}, namesBySteamID[ibadSteamID])

	// the Steam names are still available via PlayerInfo
	assert.True(t, steamNamesSeen[zoroSteamID]["- [Z]oRo"], "Steam name of uid 4 should be available via PlayerInfo")
	assert.True(t, steamNamesSeen[ibadSteamID]["BuddhaMonsteRRR"], "Steam name of uid 9 should be available via PlayerInfo")
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
