package main

import (
	"os"
	"testing"
)

// Just make sure the example runs
func TestHeatmap(t *testing.T) {
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	main()

	os.Stdout = old
}
