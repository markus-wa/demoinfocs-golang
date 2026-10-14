package demoinfocs_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

func TestServerClasses_ObtainedBeforeParsingStayLive(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test due to -short flag")
	}

	f, err := os.Open(s2DemPath)
	require.NoError(t, err)

	defer mustClose(t, f)

	p := demoinfocs.NewParser(f)
	sc := p.ServerClasses()

	assert.Empty(t, sc.All())

	var pawnClass *st.ServerClass

	p.RegisterEventHandler(func(events.DataTablesParsed) {
		pawnClass = sc.FindByName("CCSPlayerPawn")
	})

	require.NoError(t, p.ParseToEnd(t.Context()))

	assert.NotNil(t, pawnClass)
	assert.Equal(t, p.ServerClasses().All(), sc.All())
}
