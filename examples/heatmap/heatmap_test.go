package main

import (
	"os"
	"testing"
)

// Just make sure the example runs
func TestHeatmap(t *testing.T) {
	os.Args = []string{"cmd", "-demo", "../../cs-demos/default.dem"}

	// Redirect stdout, the resulting image is written to this
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	main()

	os.Stdout = old
}
