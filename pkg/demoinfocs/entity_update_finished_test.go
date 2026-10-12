package demoinfocs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// TestEntityOnUpdateFinished asserts that every update of a player controller's props is followed by
// the entity's OnUpdateFinished delegates within the same frame - the parser applies player identity
// & connection changes there.
func TestEntityOnUpdateFinished(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	t.Parallel()

	f := openFile(t, s2DemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	var (
		created, finishedAfterUpdate int
		pending                      = make(map[st.Entity]bool)
		finished                     = make(map[st.Entity]int)
	)

	p.RegisterEventHandler(func(events.DataTablesParsed) {
		p.ServerClasses().FindByName("CCSPlayerController").OnEntityCreated(func(e st.Entity) {
			created++

			for _, prop := range []string{"m_iTeamNum", "m_iConnected", "m_iszPlayerName", "m_steamID"} {
				e.Property(prop).OnUpdate(func(st.PropertyValue) {
					pending[e] = true
				})
			}

			e.OnUpdateFinished(func() {
				if pending[e] && finished[e] > 0 {
					finishedAfterUpdate++
				}

				finished[e]++
				delete(pending, e)
			})

			e.OnDestroy(func() {
				delete(pending, e)
			})
		})
	})

	unfinished := 0

	p.RegisterEventHandler(func(events.FrameDone) {
		unfinished += len(pending)
	})

	require.NoError(t, p.ParseToEnd(t.Context()))

	require.NotZero(t, created)
	assert.Zero(t, unfinished, "frames that ended with controller updates not followed by OnUpdateFinished")
	assert.Len(t, finished, created, "OnUpdateFinished must be called after each controller's creation")
	assert.NotZero(t, finishedAfterUpdate, "OnUpdateFinished must be called after updates of existing controllers")
}
