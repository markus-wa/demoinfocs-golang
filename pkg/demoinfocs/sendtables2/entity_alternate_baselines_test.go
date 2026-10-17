package sendtables2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// writeFieldPathOp writes the huffman code of a field path op.
func (w *bitWriter) writeFieldPathOp(t *testing.T, name string) {
	t.Helper()

	op := -1

	for i, o := range fieldPathTable {
		if o.name == name {
			op = i
		}
	}

	require.NotEqual(t, -1, op, "unknown field path op %s", name)

	var find func(node int16, code []bool) []bool

	find = func(node int16, code []bool) []bool {
		n := fpHuffNodes[node]
		if n.left < 0 {
			if int(n.value) == op {
				return code
			}

			return nil
		}

		if c := find(n.left, append(code[:len(code):len(code)], false)); c != nil {
			return c
		}

		return find(n.right, append(code[:len(code):len(code)], true))
	}

	for _, bit := range find(0, nil) {
		if bit {
			w.writeBits(1, 1)
		} else {
			w.writeBits(0, 1)
		}
	}
}

func (w *bitWriter) writeVarUint32(v uint32) {
	for v >= 0x80 {
		w.writeBits(v&0x7f|0x80, 8)
		v >>= 7
	}

	w.writeBits(v, 8)
}

// writeUnsignedFields writes field data that sets the fields m_a (index 0) and/or m_b (index 1).
func (w *bitWriter) writeUnsignedFields(t *testing.T, a, b *uint32) {
	t.Helper()

	switch {
	case a != nil:
		w.writeFieldPathOp(t, "PlusOne")

		if b != nil {
			w.writeFieldPathOp(t, "PlusOne")
		}
	case b != nil:
		w.writeFieldPathOp(t, "PlusTwo")
	}

	w.writeFieldPathOp(t, "FieldPathEncodeFinish")

	for _, v := range []*uint32{a, b} {
		if v != nil {
			w.writeVarUint32(*v)
		}
	}
}

func unsignedFields(t *testing.T, a, b *uint32) []byte {
	t.Helper()

	var w bitWriter

	w.writeUnsignedFields(t, a, b)

	return w.buf
}

func u32(v uint32) *uint32 {
	return &v
}

// alternateBaselineTestParser returns a parser with entity 5 (m_a = 10, m_b = 20)
// and an alternate baseline (m_a = 1, m_b = 2) at instancebaseline index 7.
func alternateBaselineTestParser(t *testing.T) (*Parser, *Entity) {
	t.Helper()

	ser := newSerializer("CTest", 0)

	for _, name := range []string{"m_a", "m_b"} {
		f := simpleField(name, "")
		f.decoder = unsignedDecoder
		ser.addField(f)
	}

	cls := &class{
		name:        "CTest",
		serializer:  ser,
		fpNameCache: &fpNameTreeCache{},
		fpFlatCache: make(map[uint64]string),
	}

	p := NewParser(nil)

	e := newEntity(5, 1, cls)
	e.readFields(newReader(unsignedFields(t, u32(10), u32(20))), &p.pathCache)
	p.entities[5] = e

	p.SetInstanceBaselineByIndex(7, unsignedFields(t, u32(1), u32(2)))

	return p, e
}

// alternateBaselinePacket returns a delta packet updating m_a of entity 5 to 100,
// with entity 5 listed in alternate_baselines (baseline index 7).
func alternateBaselinePacket(t *testing.T, withSerializedEntities bool) *msgs2.CSVCMsg_PacketEntities {
	t.Helper()

	var w bitWriter

	w.writeUBitVar(5) // index 5
	w.writeBits(0, 2) // update
	w.writeUnsignedFields(t, u32(100), nil)
	w.writeUBitVar(0) // no explicit deletions

	m := &msgs2.CSVCMsg_PacketEntities{
		LegacyIsDelta:  proto.Bool(true),
		UpdatedEntries: proto.Int32(1),
		EntityData:     w.buf,
		AlternateBaselines: []*msgs2.CSVCMsg_PacketEntitiesAlternateBaselineT{{
			EntityIndex:   proto.Int32(5),
			BaselineIndex: proto.Int32(7),
		}},
	}

	if withSerializedEntities {
		m.SerializedEntities = []byte{}
	}

	return m
}

func TestAlternateBaselines(t *testing.T) {
	t.Parallel()

	p, e := alternateBaselineTestParser(t)

	var bUpdates []any

	e.Property("m_b").OnUpdate(func(v st.PropertyValue) {
		bUpdates = append(bUpdates, v.Any)
	})

	require.NoError(t, p.OnPacketEntities(alternateBaselinePacket(t, true)))

	// m_b isn't in the update, it's reset to the baseline the update is a delta from
	assert.EqualValues(t, 100, e.Get("m_a"))
	assert.EqualValues(t, 2, e.Get("m_b"))
	assert.Equal(t, []any{e.Get("m_b")}, bUpdates)
}

// The engine only reads alternate_baselines along serialized_entities.
func TestAlternateBaselines_IgnoredWithoutSerializedEntities(t *testing.T) {
	t.Parallel()

	p, e := alternateBaselineTestParser(t)

	require.NoError(t, p.OnPacketEntities(alternateBaselinePacket(t, false)))

	assert.EqualValues(t, 100, e.Get("m_a"))
	assert.EqualValues(t, 20, e.Get("m_b"))
}

func TestAlternateBaselines_MissingBaseline(t *testing.T) {
	t.Parallel()

	p, _ := alternateBaselineTestParser(t)
	delete(p.baselinesByIndex, 7)

	assert.Panics(t, func() {
		_ = p.OnPacketEntities(alternateBaselinePacket(t, true))
	})
}
