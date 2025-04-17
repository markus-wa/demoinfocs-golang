package main

import (
	"os"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/examples"
)

// Just make sure the example runs
func TestHeatmap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/s2/s2.dem"}

	examples.RedirectStdout(func() {
		main()
	})
}
