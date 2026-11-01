package main

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The checked-in s2_usercmd_typed.go is what the generator produces from the current protobuf types.
// Both sides are compared after go/format, so a change in gofmt's output between Go versions doesn't
// fail the test; only a change in what is generated does.
func TestGeneratedCodeIsCurrent(t *testing.T) {
	want, err := generate()
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join("..", "..", "s2_usercmd_typed.go"))
	require.NoError(t, err)

	got = bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")) // a Windows checkout

	want, err = format.Source(want)
	require.NoError(t, err)

	got, err = format.Source(got)
	require.NoError(t, err, "s2_usercmd_typed.go doesn't parse")

	require.Equal(t, string(want), string(got), "s2_usercmd_typed.go is stale, run go generate ./pkg/demoinfocs")
}
