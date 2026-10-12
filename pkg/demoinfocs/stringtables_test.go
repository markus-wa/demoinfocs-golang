package demoinfocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// stringTableWriter encodes string-table entries the way parseStringTable reads them (bits LSB-first).
type stringTableWriter struct {
	buf   []byte
	nBits int
}

func (w *stringTableWriter) bits(v uint64, n int) {
	for i := range n {
		if w.nBits%8 == 0 {
			w.buf = append(w.buf, 0)
		}

		if v&(1<<i) != 0 {
			w.buf[w.nBits/8] |= 1 << (w.nBits % 8)
		}

		w.nBits++
	}
}

func (w *stringTableWriter) bit(b bool) {
	if b {
		w.bits(1, 1)
	} else {
		w.bits(0, 1)
	}
}

func (w *stringTableWriter) bytes(b []byte) {
	for _, c := range b {
		w.bits(uint64(c), 8)
	}
}

func (w *stringTableWriter) varUint32(v uint32) {
	for v >= 0x80 {
		w.bits(uint64(v&0x7f|0x80), 8)
		v >>= 7
	}

	w.bits(uint64(v), 8)
}

type stringTableTestEntry struct {
	skip  *uint32 // nil = increment, otherwise the encoded skip value
	key   string
	value []byte
}

// entry writes one entry: index op, key (without history) and value (17-bit byte count, not compressed).
func (w *stringTableWriter) entry(e stringTableTestEntry) {
	w.bit(e.skip == nil)

	if e.skip != nil {
		w.varUint32(*e.skip)
	}

	w.bit(e.key != "")

	if e.key != "" {
		w.bit(false) // no key history
		w.bytes(append([]byte(e.key), 0))
	}

	w.bit(e.value != nil)

	if e.value != nil {
		w.bits(uint64(len(e.value)), 17)
		w.bytes(e.value)
	}
}

func skipBy(v uint32) *uint32 {
	return &v
}

// Non-incremental entry indices are relative to the previous index: index += varint + 2
// (same as Clarity, demoparser2 & DemoFile.Net).
func TestParseStringTable_EntryIndices(t *testing.T) {
	tests := []struct {
		name    string
		entries []stringTableTestEntry
		want    []int32
	}{
		{
			name:    "increments",
			entries: []stringTableTestEntry{{key: "a"}, {key: "b"}, {key: "c"}},
			want:    []int32{0, 1, 2},
		},
		{
			name:    "skip zero after an increment",
			entries: []stringTableTestEntry{{key: "a"}, {key: "b"}, {skip: skipBy(0), key: "d"}, {key: "e"}},
			want:    []int32{0, 1, 3, 4},
		},
		{
			// the old absolute decoding (index = varint + 1) gives 4 here
			name:    "skip after the first entry",
			entries: []stringTableTestEntry{{key: "a"}, {skip: skipBy(3), key: "f"}, {key: "g"}},
			want:    []int32{0, 5, 6},
		},
		{
			name: "values and multi-byte skips",
			entries: []stringTableTestEntry{
				{key: "a", value: []byte{1, 2, 3}},
				{skip: skipBy(200), value: []byte("player")},
				{value: []byte{}},
			},
			want: []int32{0, 202, 203},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newParser()

			var warns []events.ParserWarn
			p.RegisterEventHandler(func(e events.ParserWarn) { warns = append(warns, e) })

			w := new(stringTableWriter)
			for _, e := range tc.entries {
				w.entry(e)
			}

			w.buf = append(w.buf, make([]byte, 16)...) // the bit reader reads ahead

			items := p.parseStringTable(w.buf, int32(len(tc.entries)), "test", false, 0, 0, false)

			require.Empty(t, warns)
			require.Len(t, items, len(tc.entries))

			for i, item := range items {
				assert.Equal(t, tc.want[i], item.Index, "entry %d", i)
				assert.Equal(t, tc.entries[i].key, item.Key, "entry %d", i)

				if tc.entries[i].value != nil {
					assert.Equal(t, tc.entries[i].value, append([]byte{}, item.Value...), "entry %d", i)
				}
			}
		})
	}
}
