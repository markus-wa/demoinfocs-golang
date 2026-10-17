package sendtables2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// bitWriter writes bits in the order reader reads them (LSB first).
type bitWriter struct {
	buf []byte
	n   uint
}

func (w *bitWriter) writeBits(v uint32, n uint) {
	for i := range n {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}

		if v>>i&1 == 1 {
			w.buf[w.n/8] |= 1 << (w.n % 8)
		}

		w.n++
	}
}

func (w *bitWriter) writeUBitVar(v uint32) {
	switch {
	case v < 1<<4:
		w.writeBits(v, 6)
	case v < 1<<8:
		w.writeBits(v&15|16, 6)
		w.writeBits(v>>4, 4)
	case v < 1<<12:
		w.writeBits(v&15|32, 6)
		w.writeBits(v>>4, 8)
	default:
		w.writeBits(v&15|48, 6)
		w.writeBits(v>>4, 28)
	}
}

type entityOp struct {
	index int
	op    st.EntityOp
}

// deletionsTestParser returns a parser with active entities at index 2 and 9 and a destroyed one at index 7.
func deletionsTestParser(t *testing.T) (*Parser, *[]entityOp) {
	t.Helper()

	cls, _, _ := polyTestClass(t)

	p := NewParser(nil)

	for _, index := range []int32{2, 7, 9} {
		p.entities[index] = newEntity(index, 1, cls)
	}

	p.entities[7].Destroy()

	var ops []entityOp

	p.OnEntity(func(e st.Entity, op st.EntityOp) error {
		ops = append(ops, entityOp{e.ID(), op})

		return nil
	})

	return p, &ops
}

func deltaPacket(entityData []byte) *msgs2.CSVCMsg_PacketEntities {
	return &msgs2.CSVCMsg_PacketEntities{
		LegacyIsDelta:  proto.Bool(true),
		UpdatedEntries: proto.Int32(0),
		EntityData:     entityData,
	}
}

func TestExplicitDeletions(t *testing.T) {
	t.Parallel()

	p, ops := deletionsTestParser(t)

	destroyed := 0
	p.entities[2].OnDestroy(func() { destroyed++ })

	var w bitWriter

	w.writeUBitVar(3)   // count
	w.writeUBitVar(3)   // 2
	w.writeUBitVar(5)   // 7, already destroyed
	w.writeUBitVar(300) // 307, unknown

	require.NoError(t, p.OnPacketEntities(deltaPacket(w.buf)))

	assert.Equal(t, 1, destroyed)
	assert.False(t, p.entities[2].active)
	assert.False(t, p.entities[2].Replaced())
	assert.True(t, p.entities[9].active)
	assert.Equal(t, []entityOp{{2, st.EntityOpLeft | st.EntityOpDeleted}}, *ops)
}

func TestExplicitDeletions_NotReadWithNonTransmittedEntities(t *testing.T) {
	t.Parallel()

	p, ops := deletionsTestParser(t)

	var w bitWriter

	w.writeUBitVar(1)
	w.writeUBitVar(3)

	msg := deltaPacket(w.buf)
	msg.NonTransmittedEntities = &msgs2.CSVCMsg_PacketEntitiesNonTransmittedEntitiesT{}

	require.NoError(t, p.OnPacketEntities(msg))

	assert.True(t, p.entities[2].active)
	assert.Empty(t, *ops)
}

func TestExplicitDeletions_PaddingOnly(t *testing.T) {
	t.Parallel()

	p, ops := deletionsTestParser(t)

	var w bitWriter

	w.writeBits(0, 5)

	require.NoError(t, p.OnPacketEntities(deltaPacket(w.buf)))

	assert.True(t, p.entities[2].active)
	assert.Empty(t, *ops)
}

func TestExplicitDeletions_InvalidDelta(t *testing.T) {
	t.Parallel()

	p, _ := deletionsTestParser(t)

	var w bitWriter

	w.writeUBitVar(1)
	w.writeUBitVar(0) // indices must increase

	assert.Panics(t, func() {
		_ = p.OnPacketEntities(deltaPacket(w.buf))
	})
}
