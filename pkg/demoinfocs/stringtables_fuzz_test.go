package demoinfocs

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/snappy"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/fuzzcheck"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

// stringTableSeed is a string table of a real demo, see testdata/stringtables.
type stringTableSeed struct {
	data                 []byte
	numEntries, flags    int32
	userDataFixed        bool
	userDataSize         int32
	usingVarintBitcounts bool
	userInfo             bool
}

// readStringTableSeeds reads the string tables in testdata/stringtables, which were extracted from s2.dem.
// Format: varints numEntries, flags, userDataFixedSize, userDataSize, usingVarintBitcounts, followed by the string data.
func readStringTableSeeds(tb testing.TB) []stringTableSeed {
	tb.Helper()

	paths, err := filepath.Glob("testdata/stringtables/*.bin")
	require.NoError(tb, err)
	require.NotEmpty(tb, paths)

	seeds := make([]stringTableSeed, 0, len(paths))

	for _, path := range paths {
		b, err := os.ReadFile(path)
		require.NoError(tb, err)

		r := bytes.NewReader(b)

		var v [5]uint64

		for i := range v {
			v[i], err = binary.ReadUvarint(r)
			require.NoError(tb, err)
		}

		seeds = append(seeds, stringTableSeed{
			data:                 b[len(b)-r.Len():],
			numEntries:           int32(v[0]),
			flags:                int32(v[1]),
			userDataFixed:        v[2] != 0,
			userDataSize:         int32(v[3]),
			usingVarintBitcounts: v[4] != 0,
			userInfo:             bytes.Contains([]byte(path), []byte("userinfo")),
		})
	}

	return seeds
}

// FuzzStringTable processes a fuzzed userinfo or instancebaseline string table.
//
//	go test -tags unassert_panic -run '^$' -fuzz '^FuzzStringTable$' -fuzztime 15m ./pkg/demoinfocs
//
// Not parallel, see fuzzcheck.AllocatedBytes().
func FuzzStringTable(f *testing.F) {
	for _, s := range readStringTableSeeds(f) {
		f.Add(s.data, s.numEntries, s.flags, s.userDataFixed, s.userDataSize, s.usingVarintBitcounts, s.userInfo, false)
		f.Add(s.data, s.numEntries, s.flags, s.userDataFixed, s.userDataSize, s.usingVarintBitcounts, !s.userInfo, true)
	}

	f.Fuzz(func(t *testing.T, data []byte, numEntries, flags int32, userDataFixed bool, userDataSize int32, usingVarintBitcounts, userInfo, compressed bool) {
		name := stNameInstanceBaseline
		if userInfo {
			name = stNameUserInfo
		}

		if compressed {
			data = snappy.Encode(nil, data)
		}

		tab := &msg.CSVCMsg_CreateStringTable{
			Name:                 proto.String(name),
			NumEntries:           proto.Int32(numEntries),
			Flags:                proto.Int32(flags),
			UserDataFixedSize:    proto.Bool(userDataFixed),
			UserDataSize:         proto.Int32(userDataSize),
			UsingVarintBitcounts: proto.Bool(usingVarintBitcounts),
			DataCompressed:       proto.Bool(compressed),
			StringData:           data,
		}

		fuzzcheck.Run(t, len(data), func() {
			p := newParser()

			// failures are panics, which the message handler's boundary turns into a DecodeError
			defer func() { _ = recover() }()

			p.processStringTable(tab)
		})
	})
}
