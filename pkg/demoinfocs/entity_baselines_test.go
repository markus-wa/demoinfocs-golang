package demoinfocs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// In POV demos, an entity created at the index of an entity created by an update_baseline packet
// is a delta from that entity's state right after its creation, not from its class baseline.
// Physics props re-created this way all keep the collision hierarchy ID of the previous one,
// only ~49% do if the class baseline is used instead (see also sendtables2.TestEntityBaselines_Replacement).
func TestEntityBaselines_POV(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, s2POVDemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	const hierarchyID = "m_Collision.m_collisionAttribute.m_nHierarchyId"

	previous := map[int]st.Entity{}
	replaced, sameHierarchyID := 0, 0

	p.RegisterEventHandler(func(events.DataTablesParsed) {
		p.ServerClasses().FindByName("CPhysicsPropMultiplayer").OnEntityCreated(func(e st.Entity) {
			prev := previous[e.ID()]
			previous[e.ID()] = e

			if prev == nil || prev.SerialNum() == e.SerialNum() {
				return
			}

			replaced++

			if prev.PropertyValueMust(hierarchyID).Any == e.PropertyValueMust(hierarchyID).Any {
				sameHierarchyID++
			}
		})
	})

	assert.NoError(t, p.ParseToEnd())
	assert.Positive(t, replaced, "the test needs a POV demo with re-created props")
	assert.Equal(t, replaced, sameHierarchyID, "re-created props that kept their hierarchy ID")
}
