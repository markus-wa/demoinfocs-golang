package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Just make sure the example runs
func TestScores(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test")
	}

	os.Args = []string{"cmd", "-demo", "../../cs-demos/default.dem"}

	main()
}

// formatPlayer should be null-safe - there are some demos with killer / victim = nil
func TestFormatPlayerNil(t *testing.T) {
	assert.Equal(t, "?", formatPlayer(nil))
}
