package demoinfocs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

// POV demos have entity updates that are deltas from an alternate instance baseline ("<classID>:<n>" entries),
// see sendtables2.TestAlternateBaselines. A missing baseline makes parsing fail.
func TestAlternateBaselines_POV(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test")
	}

	f := openFile(t, s2POVDemPath)
	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)

	alternateBaselines := 0

	p.RegisterNetMessageHandler(func(m *msgs2.CSVCMsg_PacketEntities) {
		alternateBaselines += len(m.GetAlternateBaselines())
	})

	assert.NoError(t, p.ParseToEnd())
	assert.Positive(t, alternateBaselines, "the test needs a demo with alternate baselines")
}
