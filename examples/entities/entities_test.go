package main

import (
	"os"
	"testing"
)

// Just make sure the example runs
func TestEntities(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/default.dem"}

	main()
}
