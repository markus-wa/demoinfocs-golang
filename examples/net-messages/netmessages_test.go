package main

import (
	"os"
	"testing"
)

// Just make sure the example runs
func TestNetMessages(t *testing.T) {
	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/default.dem"}

	main()
}
