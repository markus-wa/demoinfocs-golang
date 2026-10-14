package demoinfocs_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// BenchmarkUserPropertyHandlers measures the overhead of user send-table callbacks (e.g. tagging their panics)
// by registering an update handler on every property of every player pawn.
func BenchmarkUserPropertyHandlers(b *testing.B) {
	f := openFile(b, s2DemPath)
	defer mustClose(b, f)

	d, err := io.ReadAll(f)
	require.NoError(b, err, "failed to read file %q", s2DemPath)

	var updates int

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := demoinfocs.NewParser(bytes.NewReader(d))

		p.RegisterEventHandler(func(events.DataTablesParsed) {
			p.ServerClasses().FindByName("CCSPlayerPawn").OnEntityCreated(func(e *st.Entity) {
				for _, prop := range e.Properties() {
					prop.OnUpdate(func(st.PropertyValue) {
						updates++
					})
				}
			})
		})

		err = p.ParseToEnd(b.Context())
		assert.NoError(b, err, "ParseToEnd() returned an error")
	}

	b.ReportMetric(float64(updates)/float64(b.N), "updates/op")
}
