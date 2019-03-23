package main

import (
	"os"
	"testing"
)

// Just make sure the example runs
func TestNetMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../cs-demos/default.dem"}

	main()
}
