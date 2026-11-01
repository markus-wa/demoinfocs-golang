package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The checked-in s2_usercmd_typed.go is what the generator produces from the current protobuf types.
func TestGeneratedCodeIsCurrent(t *testing.T) {
	want, err := generate()
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join("..", "..", "s2_usercmd_typed.go"))
	require.NoError(t, err)

	got = bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")) // a Windows checkout

	require.Equal(t, string(want), string(got), "s2_usercmd_typed.go is stale, run go generate ./pkg/demoinfocs")
}
