package sendtables

import (
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
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

	set := func(path ...int) [2]int {
		allocated, freed := s.set(&fieldPath{path: path, last: len(path) - 1}, 1)

		return [2]int{allocated, freed}
	}

	assert.Equal(t, [2]int{101, 0}, set(99)) // z+2
	assert.Equal(t, [2]int{0, 0}, set(99))
	assert.Equal(t, [2]int{202, 101}, set(100)) // 2x
	assert.Equal(t, [2]int{0, 0}, set(150))
	assert.Equal(t, [2]int{newFieldStateSlots, 0}, set(1, 2))                            // replaces the scalar at 1
	assert.Equal(t, [2]int{newFieldStateSlots + 1002, newFieldStateSlots}, set(2, 1000)) // nested state grows
	assert.Equal(t, 202+2*newFieldStateSlots+1002-newFieldStateSlots, s.slots())
}

// Slices that grow one slot at a time are reallocated O(log n) times, up to maxStateCap.
func TestFieldStateSet_AmortisedGrowth(t *testing.T) {
	t.Parallel()

	s := &fieldState{}
	reallocs := 0

	for i := range maxFieldIndex + 1 {
		if allocated, _ := s.set(&fieldPath{path: []int{i}}, 1); allocated > 0 {
			reallocs++
		}
	}

	assert.Less(t, reallocs, 20)
	assert.Equal(t, maxStateCap, cap(s.state))

	assert.Equal(t, maxStateCap, grownCap(maxStateCap-1, maxStateCap))
	assert.Equal(t, maxStateCap+1, grownCap(maxStateCap, maxStateCap+1))
	assert.Equal(t, 16, grownCap(8, 9))
}

func TestEntityBudget(t *testing.T) {
	t.Parallel()

	fp := &fieldPath{path: []int{1}}
	e := &Entity{class: &ServerClass{classId: 7}}

	b := newEntityBudget()
	b.spendState(e, entityBudgetBaseLiveSlots, 0, fp)
	b.spendBaseline(entityBudgetBaseBaselineBytes, 1)

	assert.PanicsWithValue(t, "entity state exceeds the budget of live slots per byte of entity data (path [1]): corrupt or desynced demo bitstream", func() {
		b.spendState(e, 1, 0, fp)
	})
	assert.PanicsWithValue(t, "decoded instance baselines exceed the budget of 64 bytes per byte of entity data (class 1): corrupt demo", func() {
		b.spendBaseline(1, 1)
	})

	// refunds release the live slots of the entity, but not the allocated slots
	e = &Entity{class: &ServerClass{classId: 7}}
	b = newEntityBudget()
	b.credit(10)
	b.spendState(e, 1000, 0, fp)
	b.spendState(e, 2000, 1000, fp) // reallocation
	assert.EqualValues(t, 2000, b.liveSlots)
	b.release(e, 500)
	assert.EqualValues(t, 1500, e.budgetSlots)
	b.refund(e)
	assert.Zero(t, b.liveSlots)
	assert.Zero(t, e.budgetSlots)
	assert.EqualValues(t, 3000, b.allocatedSlots)

	b.spendState(e, int(b.maxAllocatedSlots-b.allocatedSlots), 0, fp)
	b.refund(e)

	assert.PanicsWithValue(t, "entities exceed the budget of allocated slots per byte of entity data (class 7): corrupt demo", func() {
		b.spendEntity(e, 0)
	})

	// no overflow on 32-bit platforms
	b = newEntityBudget()

	for range 1000 {
		b.credit(1 << 30)
	}

	assert.Positive(t, b.maxLiveSlots)
	assert.Positive(t, b.maxAllocatedSlots)
	assert.Positive(t, b.maxBaselineBytes)
}

// bitWriter writes bits in the order reader reads them.
type bitWriter struct {
	buf []byte
	n   int
}

func (w *bitWriter) bits(v uint64, n int) {
	for i := range n {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}

		w.buf[w.n/8] |= byte(v>>i&1) << (w.n % 8)
		w.n++
	}
}

func (w *bitWriter) varUint(v uint64) {
	for {
		b := v & 0x7f
		v >>= 7

		if v != 0 {
			b |= 0x80
		}

		w.bits(b, 8)

		if v == 0 {
			return
		}
	}
}

// uBitVar writes v as read by reader.readUBitVar().
func (w *bitWriter) uBitVar(v uint32) {
	switch {
	case v < 1<<4:
		w.bits(uint64(v), 6)
	case v < 1<<8:
		w.bits(uint64(v&15|16), 6)
		w.bits(uint64(v>>4), 4)
	case v < 1<<12:
		w.bits(uint64(v&15|32), 6)
		w.bits(uint64(v>>4), 8)
	default:
		w.bits(uint64(v&15|48), 6)
		w.bits(uint64(v>>4), 28)
	}
}

// fieldPathOp writes the huffman code of the field path op with the given name, see readFieldPaths().
func (w *bitWriter) fieldPathOp(t *testing.T, name string) {
	t.Helper()

	op := slices.IndexFunc(fieldPathTable, func(o fieldPathOp) bool { return o.name == name })
	require.GreaterOrEqual(t, op, 0, name)

	var find func(node int16, code []bool) []bool

	find = func(node int16, code []bool) []bool {
		n := fpHuffNodes[node]
		if n.left < 0 {
			if int(n.value) == op {
				return code
			}

			return nil
		}

		if c := find(n.left, append(slices.Clone(code), false)); c != nil {
			return c
		}

		return find(n.right, append(slices.Clone(code), true))
	}

	code := find(0, nil)
	require.NotNil(t, code, name)

	for _, bit := range code {
		if bit {
			w.bits(1, 1)
		} else {
			w.bits(0, 1)
		}
	}
}

// arrayFieldUpdate writes the field paths and value of an update of m_arr (see budgetTestParser()) to length n.
func (w *bitWriter) arrayFieldUpdate(t *testing.T, n uint64) {
	t.Helper()

	w.fieldPathOp(t, "PlusOne")
	w.fieldPathOp(t, "FieldPathEncodeFinish")
	w.varUint(n)
}

// budgetTestParser returns a parser with one class (ID 0) with one field, the variable array m_arr.
func budgetTestParser(t *testing.T) *Parser {
	t.Helper()

	arr := &field{
		varName:          "m_arr",
		name:             "m_arr",
		varType:          "CUtlVector< int32 >",
		fieldType:        newFieldType("CUtlVector< int32 >"),
		polySerializerID: -1,
	}
	arr.setModel(fieldModelVariableArray)

	ser := newSerializer("CArr", 0)
	ser.addField(arr)

	p := NewParser()
	p.classIdSize = 1
	p.classesById[0] = &ServerClass{
		classId:     0,
		name:        "CArr",
		serializer:  ser,
		fpNameCache: &fpNameTreeCache{},
		fpFlatCache: make(map[uint64]string),
	}

	return p
}

type entityOp struct {
	index   int32
	delete  bool
	serial  uint64
	arrSize uint64
}

// entitiesMsg returns a delta PacketEntities message with the ops (in ascending index order),
// its entity data is padded to size bytes.
func entitiesMsg(t *testing.T, size int, ops ...entityOp) *msg.CSVCMsg_PacketEntities {
	t.Helper()

	w := &bitWriter{}
	prev := int32(-1)

	for _, op := range ops {
		w.uBitVar(uint32(op.index - prev - 1))
		prev = op.index

		if op.delete {
			w.bits(3, 2)

			continue
		}

		w.bits(2, 2) // create
		w.bits(0, 1) // class ID
		w.bits(op.serial, 17)
		w.varUint(0)
		w.arrayFieldUpdate(t, op.arrSize)
	}

	require.LessOrEqual(t, len(w.buf), size)

	return &msg.CSVCMsg_PacketEntities{
		UpdatedEntries: proto.Int32(int32(len(ops))),
		LegacyIsDelta:  proto.Bool(true),
		EntityData:     append(w.buf, make([]byte, size-len(w.buf))...),
	}
}

// The live budget is refunded when an entity is replaced, so long demos with entity churn don't exceed it.
func TestEntityBudget_Churn(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		arrSize uint64
	}{
		// ~8 slots per byte (incl. the fixed charge per entity), like the peak of real demos
		{"realistic", 16},
		// ~100 slots per byte: more than the live budget per byte, so it only passes because of the refunds
		{"large state", 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := budgetTestParser(t)

			for i := range 50_000 {
				// create and destroy an entity at the same index, every other create replaces the entity directly
				m := entitiesMsg(t, 10, entityOp{index: 5, serial: uint64(i % 7), arrSize: tc.arrSize})
				require.NoError(t, p.OnPacketEntities(m), "message %d", i)

				if i%2 == 0 {
					require.NoError(t, p.OnPacketEntities(entitiesMsg(t, 10, entityOp{index: 5, delete: true})), "message %d", i)
				}
			}

			e := p.entities[5]
			require.NotNil(t, e)
			assert.Equal(t, e.budgetSlots, p.entityBudget.liveSlots, "only the last entity is live")
		})
	}
}

// assertBudgetExceeded asserts that err is a failure of the library (a DecodeError / ErrDecodeFailed in package demoinfocs).
func assertBudgetExceeded(t *testing.T, err error, contains string) {
	t.Helper()

	var f *panics.Failure

	require.ErrorAs(t, err, &f)
	assert.Contains(t, err.Error(), contains)
}

// Destroyed entities stay in Parser.entities until their index is reused, they aren't refunded.
func TestEntityBudget_GrowthWithoutRefund(t *testing.T) {
	t.Parallel()

	p := budgetTestParser(t)

	var err error

	for i := int32(0); i < 100_000 && err == nil; i++ {
		ops := []entityOp{{index: i, arrSize: 1000}}
		if i > 0 {
			ops = []entityOp{{index: i - 1, delete: true}, ops[0]}
		}

		err = p.OnPacketEntities(entitiesMsg(t, 10, ops...))
	}

	assertBudgetExceeded(t, err, "entity state exceeds the budget of live slots per byte of entity data")
	assert.LessOrEqual(t, p.entityBudget.liveSlots, p.entityBudget.maxLiveSlots+1000)
}

// Re-creating an entity at the same index costs a few bytes, the work budget bounds what's decoded and allocated.
func TestEntityBudget_SameIndexRecreation(t *testing.T) {
	t.Parallel()

	t.Run("large baseline", func(t *testing.T) {
		t.Parallel()

		p := budgetTestParser(t)

		w := &bitWriter{}
		w.arrayFieldUpdate(t, 0)
		p.SetInstanceBaseline(0, append(w.buf, make([]byte, 64<<10)...))

		var err error

		for i := 0; i < 100_000 && err == nil; i++ {
			err = p.OnPacketEntities(entitiesMsg(t, 6, entityOp{index: 5, serial: uint64(i % 2)}))
		}

		assertBudgetExceeded(t, err, "decoded instance baselines exceed the budget")
	})

	t.Run("large state", func(t *testing.T) {
		t.Parallel()

		p := budgetTestParser(t)

		var err error

		for i := 0; i < 100_000 && err == nil; i++ {
			err = p.OnPacketEntities(entitiesMsg(t, 8, entityOp{index: 5, serial: uint64(i % 2), arrSize: maxFieldIndex}))
		}

		assertBudgetExceeded(t, err, "entity state exceeds the budget of allocated slots per byte of entity data")
		assert.Less(t, p.entityBudget.liveSlots, int64(2*maxFieldIndex), "the live slots are refunded")
	})
}

// Full (non-delta) PacketEntities after the first are ignored, their data doesn't add to the budget.
func TestEntityBudget_IgnoredFullPacketsDontCredit(t *testing.T) {
	t.Parallel()

	p := budgetTestParser(t)

	full := entitiesMsg(t, 1<<20)
	full.LegacyIsDelta = proto.Bool(false)

	require.NoError(t, p.OnPacketEntities(full))

	want := p.entityBudget
	assert.Equal(t, int64(entityBudgetBaseLiveSlots+(1<<20)*entityBudgetLiveSlotsPerByte), want.maxLiveSlots)

	require.NoError(t, p.OnPacketEntities(full))
	assert.Equal(t, want, p.entityBudget)
}

// The polySerializers of existing entities grow with the polyCount of their class (see Parser.ParsePacket()).
func TestEntityGrowsPolySerializers(t *testing.T) {
	t.Parallel()

	class, _, _ := polyTestClass(t)
	e := newEntity(1, 1, class)
	require.Len(t, e.polySerializers, 1)

	class.polyCount = 3

	w := &bitWriter{}
	w.fieldPathOp(t, "FieldPathEncodeFinish")

	budget := newEntityBudget()
	r := newReader(w.buf)
	e.readFields(r, &[]*fieldPath{}, &budget)
	r.release()

	assert.Len(t, e.polySerializers, 3)
	assert.EqualValues(t, polySerializersSlots(3)-polySerializersSlots(1), budget.liveSlots, "only the added slots are live")
	assert.EqualValues(t, polySerializersSlots(3), budget.allocatedSlots)
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

	// the limit is per schema, the IDs of all schemas are bounded by maxPolySerializerIDs
	p = NewParser()
	b := manyPolyFieldsFSV(t, maxPolymorphicFields)

	for range maxPolySerializerIDs / maxPolymorphicFields {
		require.NoError(t, p.ParsePacket(b))
	}

	assert.Equal(t, maxPolySerializerIDs, p.nextPolyID)
	assert.PanicsWithValue(t, "more than 4096 polymorphic fields in all schemas: corrupt demo", func() {
		_ = p.ParsePacket(b)
	})
}

// polyFieldIDs returns the polymorphic fields reachable from the serializers by their ID.
// It fails if two different fields have the same ID.
func polyFieldIDs(t *testing.T, sers map[string]*serializer) map[int]*field {
	t.Helper()

	ids := map[int]*field{}
	seen := map[*serializer]bool{}

	var walk func(s *serializer)

	walk = func(s *serializer) {
		if s == nil || seen[s] {
			return
		}

		seen[s] = true

		for _, f := range s.fields {
			if f.polySerializerID >= 0 {
				if other, ok := ids[f.polySerializerID]; ok && other != f {
					t.Errorf("polymorphic fields %s.%s and %s share ID %d", s.name, f.name, other.name, f.polySerializerID)
				}

				ids[f.polySerializerID] = f
			}

			walk(f.serializer)

			for _, pt := range f.polyTypes {
				walk(pt)
			}
		}
	}

	for _, s := range sers {
		walk(s)
	}

	return ids
}

// A schema that's sent again (e.g. by a stream that repeats DEM_SendTables) gets new polymorphic IDs (they're globally unique,
// see Parser.nextPolyID), the entities created with the previous ones keep working.
func TestParsePacket_Resent(t *testing.T) {
	t.Parallel()

	sendTables := s2File(t, "sendtables.bin")

	p, err := newS2Parser(t, sendTables)
	require.NoError(t, err)

	msgs := readPacketEntities(s2File(t, "packetentities.bin"))
	require.NoError(t, applyPacketEntities(p, msgs[:10]))

	polyIDs := p.nextPolyID
	require.Positive(t, polyIDs)

	rest := msgs[10:]

	for i := range 100 {
		require.NoError(t, p.ParsePacket(sendTables))
		assert.Equal(t, polyIDs*(i+2), p.nextPolyID)

		// entity updates between the schemas
		if i%10 == 9 {
			require.NoError(t, applyPacketEntities(p, rest[:1]), "re-send %d", i)
			rest = rest[1:]
		}
	}

	assert.LessOrEqual(t, len(p.polyMaxCache), len(p.serializers)*2, "the cache doesn't keep the serializers of previous schemas")

	ids := polyFieldIDs(t, p.serializers)
	assert.Len(t, ids, polyIDs, "the current schema references only its own polymorphic fields")

	for name, c := range p.classesByName {
		assert.Equal(t, c.serializer.maxPolyID(map[*serializer]int{})+1, c.polyCount, name)
	}

	require.NoError(t, applyPacketEntities(p, rest))
	require.Len(t, p.entities, 161)
}

// fsvBytes returns the FlattenedSerializer message, length-prefixed as ParsePacket expects.
func fsvBytes(t *testing.T, m *msg.CSVCMsg_FlattenedSerializer) []byte {
	t.Helper()

	b, err := proto.Marshal(m)
	require.NoError(t, err)

	return append(binary.AppendUvarint(nil, uint64(len(b))), b...)
}

// A later schema can reference serializers of an earlier one (which keep their polymorphic IDs),
// its new polymorphic fields get different IDs.
func TestParsePacket_PolyIDsUniqueAcrossSchemas(t *testing.T) {
	t.Parallel()

	sym := func(i int) *int32 { return proto.Int32(int32(i)) }
	symbols := []string{"CModeA", "CModeB", "CHolder", "CMode*", "int32", "m_a", "m_b", "m_pMode", "CEnt2", "CHolder*", "m_holder", "m_pMode2"}
	polyTypes := []*msg.ProtoFlattenedSerializerFieldTPolymorphicFieldT{
		{PolymorphicFieldSerializerNameSym: sym(0)},
		{PolymorphicFieldSerializerNameSym: sym(1)},
	}

	p := NewParser()

	require.NoError(t, p.ParsePacket(fsvBytes(t, &msg.CSVCMsg_FlattenedSerializer{
		Symbols: symbols,
		Fields: []*msg.ProtoFlattenedSerializerFieldT{
			{VarNameSym: sym(5), VarTypeSym: sym(4)},
			{VarNameSym: sym(6), VarTypeSym: sym(4)},
			{VarNameSym: sym(7), VarTypeSym: sym(3), FieldSerializerNameSym: sym(0), PolymorphicTypes: polyTypes},
		},
		Serializers: []*msg.ProtoFlattenedSerializerT{
			{SerializerNameSym: sym(0), FieldsIndex: []int32{0}},
			{SerializerNameSym: sym(1), FieldsIndex: []int32{1}},
			{SerializerNameSym: sym(2), FieldsIndex: []int32{2}},
		},
	})))

	holder := p.serializers["CHolder"]

	// CHolder isn't sent again, CEnt2 references the one of the first schema
	require.NoError(t, p.ParsePacket(fsvBytes(t, &msg.CSVCMsg_FlattenedSerializer{
		Symbols: symbols,
		Fields: []*msg.ProtoFlattenedSerializerFieldT{
			{VarNameSym: sym(10), VarTypeSym: sym(9), FieldSerializerNameSym: sym(2)},
			{VarNameSym: sym(11), VarTypeSym: sym(3), FieldSerializerNameSym: sym(0), PolymorphicTypes: polyTypes},
		},
		Serializers: []*msg.ProtoFlattenedSerializerT{
			{SerializerNameSym: sym(8), FieldsIndex: []int32{0, 1}},
		},
	})))

	ent := p.serializers["CEnt2"]
	require.Same(t, holder, ent.fields[0].serializer)

	ids := polyFieldIDs(t, map[string]*serializer{"CEnt2": ent})
	require.Len(t, ids, 2)
	assert.NotEqual(t, holder.fields[0].polySerializerID, ent.fields[1].polySerializerID)
	assert.Equal(t, 1, ent.maxPolyID(p.polyMaxCache))
}

// vecElemField is the index of the field of the elements of m_vec that vecUpdate() sets, see vecTestClass().
const vecElemField = 19

// vecTestClass returns a class with one field, m_vec, a vector of structs with vecElemField+1 int32 fields.
func vecTestClass(t *testing.T) *ServerClass {
	t.Helper()

	elem := newSerializer("CElem", 0)

	for i := range vecElemField + 1 {
		f := &field{
			varName:          fmt.Sprintf("m_x%d", i),
			name:             fmt.Sprintf("m_x%d", i),
			varType:          "int32",
			fieldType:        newFieldType("int32"),
			polySerializerID: -1,
		}
		f.setModel(fieldModelSimple)
		elem.addField(f)
	}

	vec := &field{
		varName:          "m_vec",
		name:             "m_vec",
		varType:          "CUtlVectorEmbeddedNetworkVar< CElem >",
		fieldType:        newFieldType("CUtlVectorEmbeddedNetworkVar< CElem >"),
		serializer:       elem,
		polySerializerID: -1,
	}
	vec.setModel(fieldModelVariableTable)

	ser := newSerializer("CVec", 0)
	ser.addField(vec)

	return &ServerClass{
		classId:     0,
		name:        "CVec",
		serializer:  ser,
		fpNameCache: &fpNameTreeCache{},
		fpFlatCache: make(map[uint64]string),
	}
}

// vecUpdate returns the entity data of an update of m_vec (see vecTestClass()) to n elements (at most 32),
// which sets field vecElemField of each element.
func vecUpdate(t *testing.T, n int) []byte {
	t.Helper()

	w := &bitWriter{}
	w.fieldPathOp(t, "PlusOne") // [0]: the length

	for i := range n {
		if i == 0 {
			w.fieldPathOp(t, "PushTwoPack5LeftDeltaZero") // [0 0 vecElemField]
			w.bits(0, 5)
			w.bits(vecElemField, 5)

			continue
		}

		w.fieldPathOp(t, "NonTopoPenultimatePlusOne") // [0 i vecElemField]
	}

	w.fieldPathOp(t, "FieldPathEncodeFinish")
	w.varUint(uint64(n))

	for range n {
		w.varUint(0)
	}

	// padded, the reader reads ahead
	return append(w.buf, make([]byte, 8)...)
}

// heldSlots returns the slots e holds, see entityBudget.
func heldSlots(e *Entity) int64 {
	return int64(entityFixedSlots - entityInitialStateSlots + e.state.slots() + polySerializersSlots(len(e.polySerializers)))
}

// The live charge of an entity is what it holds: replaced slices and dropped nested states are credited back,
// so a long-lived entity's charge doesn't drift upwards. The allocated slots aren't credited.
func TestEntityBudget_LiveChargeMatchesHeld(t *testing.T) {
	t.Parallel()

	e := newEntity(1, 1, vecTestClass(t))
	budget := newEntityBudget()
	budget.credit(1 << 20)
	budget.spendEntity(e, len(e.polySerializers))

	require.Equal(t, heldSlots(e), e.budgetSlots)

	var (
		paths   []*fieldPath
		maxHeld int64
	)

	// grows (reallocating m_vec and the root state), shrinks (dropping the nested states of the elements) and regrows
	sizes := []int{3, 20, 1, 32, 0, 17, 32, 2, 9}

	for round := range 1000 {
		n := sizes[round%len(sizes)]

		r := newReader(vecUpdate(t, n))
		e.readFields(r, &paths, &budget)
		r.release()

		vec, ok := e.state.get(&fieldPath{path: []int{0}}).(*fieldState)
		require.True(t, ok)
		require.Len(t, vec.state, n)

		require.Equal(t, heldSlots(e), e.budgetSlots, "round %d", round)
		require.Equal(t, e.budgetSlots, budget.liveSlots)

		if e.budgetSlots > maxHeld {
			maxHeld = e.budgetSlots
		}
	}

	assert.Less(t, maxHeld, int64(entityFixedSlots+2*32*(vecElemField+2)), "the charge doesn't drift beyond what the entity holds")
	assert.Greater(t, budget.allocatedSlots, 100*maxHeld, "the allocated slots aren't credited")
}

// Collections that grow one element at a time are reallocated O(log n) times.
func TestReadFields_AmortisedCollectionGrowth(t *testing.T) {
	t.Parallel()

	p := budgetTestParser(t)
	e := newEntity(1, 1, p.classesById[0])
	budget := newEntityBudget()
	budget.credit(1 << 20)
	budget.spendEntity(e, 0)

	var paths []*fieldPath

	const n = 10_000

	for i := 1; i <= n; i++ {
		w := &bitWriter{}
		w.arrayFieldUpdate(t, uint64(i))

		r := newReader(append(w.buf, make([]byte, 8)...))
		e.readFields(r, &paths, &budget)
		r.release()
	}

	assert.Less(t, budget.allocatedSlots, int64(entityFixedSlots+4*n))
	assert.Equal(t, heldSlots(e), e.budgetSlots)
}
