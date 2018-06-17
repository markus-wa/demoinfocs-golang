package main

import (
	"os"
	"testing"
)

// Just make sure the example runs
func TestPrintKills(t *testing.T) {
	os.Args = []string{"cmd", "-demo", "../../cs-demos/default.dem"}

	main()
}
