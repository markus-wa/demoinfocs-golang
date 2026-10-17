package demoinfocs_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// Delta packets without non_transmitted_entities end with a list of deleted entities, e.g. ones that left the PVS
// before being deleted. If the list isn't read, these entities stay alive until a new entity is created at their index.
// These demos have no entity the server doesn't delete before re-using its index.
func TestExplicitEntityDeletions(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	demPaths := []string{
		s2DemPath,
		demSetPathS2 + "/1_2v2_6thAug23_64cf951f9b4ce6b86c73b089.dem",
	}

	for _, demPath := range demPaths {
		t.Run(filepath.Base(demPath), func(t *testing.T) {
			t.Parallel()

			testNoEntityReplaced(t, demPath)
		})
	}
}

func testNoEntityReplaced(t *testing.T, demPath string) {
	t.Helper()

	f := openFile(t, demPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	type replacedEntity struct {
		class  string
		id     int
		serial int
	}

	var replaced []replacedEntity

	p.RegisterEventHandler(func(events.DataTablesParsed) {
		for _, sc := range p.ServerClasses().All() {
			sc.OnEntityCreated(func(e st.Entity) {
				e.OnDestroy(func() {
					if e.(interface{ Replaced() bool }).Replaced() {
						replaced = append(replaced, replacedEntity{sc.Name(), e.ID(), e.SerialNum()})
					}
				})
			})
		}
	})

	assert.NoError(t, p.ParseToEnd())
	assert.Empty(t, replaced)
}
