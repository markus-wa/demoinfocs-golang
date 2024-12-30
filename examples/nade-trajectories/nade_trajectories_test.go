package main

import (
	"os"
	"testing"

	ex "github.com/markus-wa/demoinfocs-golang/v5/examples"
)

// Just make sure the example runs
func TestBouncyNades(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../test/cs-demos/s2/s2.dem"}

	ex.RedirectStdout(func() {
		main()
	})
}
