package sendtables2

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

// entityBaselinesTestParser returns a parser without entities and with two classes (IDs 1 and 2) of fields m_a and m_b,
// whose class baselines are m_a = 1, m_b = 2 and m_a = 3, m_b = 4.
func entityBaselinesTestParser(t *testing.T) *Parser {
	t.Helper()

	p, e := alternateBaselineTestParser(t)
	delete(p.entities, 5)

	for id, baseline := range [][]byte{1: unsignedFields(t, u32(1), u32(2)), 2: unsignedFields(t, u32(3), u32(4))} {
		if baseline == nil {
			continue
		}

		cls := &class{
			classID:     int32(id),
			name:        fmt.Sprintf("CTest%d", id),
			serializer:  e.class.serializer,
			fpNameCache: &fpNameTreeCache{},
			fpFlatCache: make(map[uint64]string),
		}
		p.classesById[cls.classID] = cls
		p.classBaselines[cls.classID] = baseline
	}

	p.classIdSize = 2

	return p
}

// entityPacket returns a delta packet using the given entity baseline slot with the given updates of entity 5.
func entityPacket(baseline int32, updateBaseline bool, updates ...func(w *bitWriter)) *msgs2.CSVCMsg_PacketEntities {
	var w bitWriter

	for _, writeUpdate := range updates {
		writeUpdate(&w)
	}

	w.writeUBitVar(0) // no explicit deletions

	return &msgs2.CSVCMsg_PacketEntities{
		LegacyIsDelta:  proto.Bool(true),
		Baseline:       proto.Int32(baseline),
		UpdateBaseline: proto.Bool(updateBaseline),
		UpdatedEntries: proto.Int32(int32(len(updates))),
		EntityData:     w.buf,
	}
}

func createEntity5(t *testing.T, classID, serial uint32, a, b *uint32) func(w *bitWriter) {
	t.Helper()

	return func(w *bitWriter) {
		w.writeUBitVar(5) // index 5
		w.writeBits(2, 2) // create
		w.writeBits(classID, 2)
		w.writeBits(serial, 17)
		w.writeVarUint32(0)
		w.writeUnsignedFields(t, a, b)
	}
}

func updateEntity5(t *testing.T, a, b *uint32) func(w *bitWriter) {
	t.Helper()

	return func(w *bitWriter) {
		w.writeUBitVar(5) // index 5
		w.writeBits(0, 2) // update
		w.writeUnsignedFields(t, a, b)
	}
}

func leaveEntity5(w *bitWriter) {
	w.writeUBitVar(5) // index 5
	w.writeBits(1, 2) // leave
}

func deleteEntity5(w *bitWriter) {
	w.writeUBitVar(5) // index 5
	w.writeBits(3, 2) // leave and delete
}

// A new entity created at the index of an entity created by an update_baseline packet
// is a delta from that entity's state right after its creation.
func TestEntityBaselines_Replacement(t *testing.T) {
	t.Parallel()

	p := entityBaselinesTestParser(t)

	require.NoError(t, p.OnPacketEntities(entityPacket(0, true, createEntity5(t, 1, 1, u32(10), u32(20)))))
	require.NoError(t, p.OnPacketEntities(entityPacket(1, false, updateEntity5(t, u32(50), u32(60)))))

	old := p.entities[5]

	require.NoError(t, p.OnPacketEntities(entityPacket(1, false, createEntity5(t, 1, 2, u32(100), nil))))

	e := p.entities[5]
	assert.Equal(t, 2, e.SerialNum())
	assert.True(t, old.Replaced())
	assert.EqualValues(t, 60, old.Get("m_b"))
	assert.EqualValues(t, 100, e.Get("m_a"))
	assert.EqualValues(t, 20, e.Get("m_b"))

	// the new entity doesn't share the baseline's state
	require.NoError(t, p.OnPacketEntities(entityPacket(1, false, updateEntity5(t, nil, u32(70)))))
	require.NoError(t, p.OnPacketEntities(entityPacket(1, false, createEntity5(t, 1, 3, u32(100), nil))))

	assert.EqualValues(t, 70, e.Get("m_b"))
	assert.EqualValues(t, 20, p.entities[5].Get("m_b"))
}

// The engine never removes entity baselines, it only replaces them.
func TestEntityBaselines_Kept(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		packets func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities
	}{
		{
			name: "entity left the PVS",
			packets: func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities {
				t.Helper()

				return []*msgs2.CSVCMsg_PacketEntities{
					entityPacket(0, true, createEntity5(t, 1, 1, u32(10), u32(20))),
					entityPacket(1, false, leaveEntity5),
					entityPacket(1, false, createEntity5(t, 1, 1, u32(100), nil)),
				}
			},
		},
		{
			name: "entity deleted",
			packets: func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities {
				t.Helper()

				return []*msgs2.CSVCMsg_PacketEntities{
					entityPacket(0, true, createEntity5(t, 1, 1, u32(10), u32(20))),
					entityPacket(1, false, deleteEntity5),
					entityPacket(1, false, createEntity5(t, 1, 2, u32(100), nil)),
				}
			},
		},
		{
			name: "copied to the other slot",
			packets: func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities {
				t.Helper()

				return []*msgs2.CSVCMsg_PacketEntities{
					entityPacket(0, true, createEntity5(t, 1, 1, u32(10), u32(20))),
					entityPacket(1, true),
					entityPacket(0, false, createEntity5(t, 1, 2, u32(100), nil)),
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			p := entityBaselinesTestParser(t)

			for _, m := range c.packets(t) {
				require.NoError(t, p.OnPacketEntities(m))
			}

			e := p.entities[5]
			assert.True(t, e.active)
			assert.EqualValues(t, 100, e.Get("m_a"))
			assert.EqualValues(t, 20, e.Get("m_b"))
		})
	}
}

func TestEntityBaselines_ClassBaseline(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		packets func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities
		b       uint32
	}{
		{
			name: "baselines not updated",
			packets: func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities {
				t.Helper()

				return []*msgs2.CSVCMsg_PacketEntities{
					entityPacket(0, false, createEntity5(t, 1, 1, u32(10), u32(20))),
					entityPacket(1, false, createEntity5(t, 1, 2, u32(100), nil)),
				}
			},
			b: 2,
		},
		{
			name: "entity created before the update",
			packets: func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities {
				t.Helper()

				return []*msgs2.CSVCMsg_PacketEntities{
					entityPacket(0, false, createEntity5(t, 1, 1, u32(10), u32(20))),
					entityPacket(0, true),
					entityPacket(1, false, createEntity5(t, 1, 2, u32(100), nil)),
				}
			},
			b: 2,
		},
		{
			name: "other slot",
			packets: func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities {
				t.Helper()

				return []*msgs2.CSVCMsg_PacketEntities{
					entityPacket(0, true, createEntity5(t, 1, 1, u32(10), u32(20))),
					entityPacket(0, false, createEntity5(t, 1, 2, u32(100), nil)),
				}
			},
			b: 2,
		},
		{
			name: "other class",
			packets: func(t *testing.T) []*msgs2.CSVCMsg_PacketEntities {
				t.Helper()

				return []*msgs2.CSVCMsg_PacketEntities{
					entityPacket(0, true, createEntity5(t, 1, 1, u32(10), u32(20))),
					entityPacket(1, false, createEntity5(t, 2, 2, u32(100), nil)),
				}
			},
			b: 4,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			p := entityBaselinesTestParser(t)

			for _, m := range c.packets(t) {
				require.NoError(t, p.OnPacketEntities(m))
			}

			e := p.entities[5]
			assert.Equal(t, 2, e.SerialNum())
			assert.EqualValues(t, 100, e.Get("m_a"))
			assert.EqualValues(t, c.b, e.Get("m_b"))
		})
	}
}
