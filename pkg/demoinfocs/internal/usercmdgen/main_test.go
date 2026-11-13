package main

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
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

// Every protobuf field of every message the generator walks must be either rejected or generated.
// describeMessage fails on a field kind it can't handle, but a wire field without a protobuf struct tag
// would be skipped silently - comparing against the descriptor turns that into a failure. A field the
// generator drops is missing from s2_usercmd_typed.go, so its deltas would stop being merged.
func TestGeneratedCodeCoversEveryField(t *testing.T) {
	messages, err := collectMessages(reflect.TypeOf(msg.CSGOUserCmdPB{}))
	require.NoError(t, err)
	require.NotEmpty(t, messages)

	for _, m := range messages {
		pm := reflect.New(m.goType).Interface().(protoreflect.ProtoMessage)
		fields := pm.ProtoReflect().Descriptor().Fields()

		generated := make(map[protoreflect.FieldNumber]bool, len(m.fields))
		for _, f := range m.fields {
			generated[protoreflect.FieldNumber(f.num)] = true
		}

		for i := 0; i < fields.Len(); i++ {
			fd := fields.Get(i)
			require.True(t, generated[fd.Number()],
				"%s: field %d (%s) is not generated", m.goType, fd.Number(), fd.Name())
		}

		require.Equal(t, fields.Len(), len(m.fields), "%s: generated field count", m.goType)
	}
}
