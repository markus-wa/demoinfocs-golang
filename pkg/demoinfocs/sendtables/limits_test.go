package sendtables

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func TestFieldStateSet_MaxFieldIndex(t *testing.T) {
	t.Parallel()

	for _, path := range [][]int{{maxFieldIndex}, {1, maxFieldIndex}} {
		fp := &fieldPath{path: path, last: len(path) - 1}

		assert.NotPanics(t, func() { newFieldState().set(fp, "x") }, "set(%v)", path)
	}

	for _, path := range [][]int{{maxFieldIndex + 1}, {1, maxFieldIndex + 1}} {
		fp := &fieldPath{path: path, last: len(path) - 1}

		assert.Panics(t, func() { newFieldState().set(fp, "x") }, "set(%v)", path)
	}
}

func TestFieldStateSet_Allocated(t *testing.T) {
	t.Parallel()

	s := &fieldState{}

	assert.Equal(t, 101, s.set(&fieldPath{path: []int{99}}, 1)) // z+2
	assert.Zero(t, s.set(&fieldPath{path: []int{99}}, 1))
	assert.Equal(t, 200, s.set(&fieldPath{path: []int{100}}, 1)) // 2x
	assert.Zero(t, s.set(&fieldPath{path: []int{150}}, 1))
	assert.Equal(t, newFieldStateSlots, s.set(&fieldPath{path: []int{1, 2}, last: 1}, 1))
	assert.Equal(t, 1002+newFieldStateSlots, s.set(&fieldPath{path: []int{2, 1000}, last: 1}, 1))
}

func TestEntityBudget(t *testing.T) {
	t.Parallel()

	fp := &fieldPath{path: []int{1}}

	b := newEntityBudget()
	b.spendState(entityBudgetBaseStateSlots, fp)
	b.spendBaseline(entityBudgetBaseBaselineBytes, 1)

	assert.PanicsWithValue(t, "entity state exceeds the budget of 32 slots per byte of entity data (path [1]): corrupt or desynced demo bitstream", func() {
		b.spendState(1, fp)
	})
	assert.PanicsWithValue(t, "decoded instance baselines exceed the budget of 64 bytes per byte of entity data (class 1): corrupt demo", func() {
		b.spendBaseline(1, 1)
	})

	b = newEntityBudget()
	b.credit(10)
	b.spendState(entityBudgetBaseStateSlots+10*entityBudgetStateSlotsPerByte, fp)
	b.spendBaseline(entityBudgetBaseBaselineBytes+10*entityBudgetBaselineBytesPerByte, 1)

	assert.Panics(t, func() { b.spendState(1, fp) })

	// the remaining budget is bounded
	b = newEntityBudget()

	for range 1000 {
		b.credit(1 << 30)
	}

	assert.Equal(t, entityBudgetMax, b.stateSlots)
	assert.Equal(t, entityBudgetMax, b.baselineBytes)
}

// manyPolyFieldsFSV returns a FlattenedSerializer message with n polymorphic fields, length-prefixed as ParsePacket expects.
func manyPolyFieldsFSV(t *testing.T, n int) []byte {
	t.Helper()

	symbols := make([]string, 0, 7+n)
	symbols = append(symbols, "CEnt", "CModeA", "CModeB", "CMode*", "int32", "m_aField", "m_bField")
	for i := range n {
		symbols = append(symbols, fmt.Sprintf("m_pMode%d", i))
	}

	sym := func(i int) *int32 { return proto.Int32(int32(i)) }

	m := &msg.CSVCMsg_FlattenedSerializer{
		Symbols: symbols,
		Fields: []*msg.ProtoFlattenedSerializerFieldT{
			{VarNameSym: sym(5), VarTypeSym: sym(4)},
			{VarNameSym: sym(6), VarTypeSym: sym(4)},
		},
		Serializers: []*msg.ProtoFlattenedSerializerT{
			{SerializerNameSym: sym(1), FieldsIndex: []int32{0}},
			{SerializerNameSym: sym(2), FieldsIndex: []int32{1}},
		},
	}

	ent := &msg.ProtoFlattenedSerializerT{SerializerNameSym: sym(0)}

	for i := range n {
		m.Fields = append(m.Fields, &msg.ProtoFlattenedSerializerFieldT{
			VarNameSym:             sym(7 + i),
			VarTypeSym:             sym(3),
			FieldSerializerNameSym: sym(1),
			PolymorphicTypes: []*msg.ProtoFlattenedSerializerFieldTPolymorphicFieldT{
				{PolymorphicFieldSerializerNameSym: sym(1)},
				{PolymorphicFieldSerializerNameSym: sym(2)},
			},
		})
		ent.FieldsIndex = append(ent.FieldsIndex, int32(2+i))
	}

	m.Serializers = append(m.Serializers, ent)

	b, err := proto.Marshal(m)
	require.NoError(t, err)

	return append(binary.AppendUvarint(nil, uint64(len(b))), b...)
}

func TestParsePacket_MaxPolymorphicFields(t *testing.T) {
	t.Parallel()

	p := NewParser()
	require.NoError(t, p.ParsePacket(manyPolyFieldsFSV(t, maxPolymorphicFields)))
	assert.Equal(t, maxPolymorphicFields, p.nextPolyID)

	p = NewParser()

	assert.PanicsWithValue(t, "more than 64 polymorphic fields: corrupt schema", func() {
		_ = p.ParsePacket(manyPolyFieldsFSV(t, maxPolymorphicFields+1))
	})
}
