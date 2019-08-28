package main

import (
	"os"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/examples"
)

// Just make sure the example runs
func TestHeatmap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/default.dem"}

	examples.RedirectStdout(func() {
		main()
	})
}
