package sendtablescs2

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

// polyTestSchema builds a small polymorphic schema:
//
//	CEnt: { m_iHealth (simple), m_pMode (poly fixed-table, 2 alternatives) }
//	CModeA: { m_aField }
//	CModeB: { m_bField }
//
// The poly field's polySerializerID is 0 (the first one allocated).
func polyTestSchema(t *testing.T) (*serializer, *serializer, *serializer) {
	t.Helper()

	modeA := newSerializer("CModeA", 0)
	modeA.addField(simpleField("m_aField", ""))

	modeB := newSerializer("CModeB", 0)
	modeB.addField(simpleField("m_bField", ""))

	polyField := &field{
		varName:          "m_pMode",
		name:             "m_pMode",
		varType:          "CMode*",
		model:            fieldModelFixedTable,
		serializer:       modeA,
		polyTypes:        []*serializer{modeA, modeB},
		polySerializerID: 0,
	}
	polyField.setModel(fieldModelFixedTable)

	ent := newSerializer("CEnt", 0)
	ent.addField(simpleField("m_iHealth", ""))
	ent.addField(polyField)

	return ent, modeA, modeB
}

// polyTestClass wraps the mini schema in a class so entities can be created.
func polyTestClass(t *testing.T) (*class, *serializer, *serializer) {
	t.Helper()

	ent, modeA, modeB := polyTestSchema(t)

	return &class{
		name:        "CEnt",
		serializer:  ent,
		polyCount:   1,
		fpNameCache: &fpNameTreeCache{},
		fpFlatCache: make(map[uint64]string),
	}, modeA, modeB
}

// polySubPath returns the field path of a mode serializer's sub-field under
// m_pMode: component 0 is m_pMode, component 1 is the first field of the
// active mode serializer.
func polySubPath() *fieldPath {
	fp := newFieldPath()
	fp.path[0] = 1 // m_pMode
	fp.path[1] = 0 // first field of the active mode serializer
	fp.last = 1

	return fp
}

// TestPolyBaseDecoderWire verifies the polymorphic base decoder reads exactly a
// boolean followed (only when active) by a ubitvar type index, and returns a
// polyUpdate carrying the selected serializer for per-entity state tracking.
//
// Bit layout (LSB-first, see reader.readBits/readBoolean):
//
//	inactive            : 0b0
//	active, type index 0: 0b1 000000  → 0x01
//	active, type index 1: 0b1 000001  → 0x03  (6-bit ubitvar, no extension)
func TestPolyBaseDecoderWire(t *testing.T) {
	t.Parallel()

	ent, modeA, modeB := polyTestSchema(t)
	polyField := ent.fields[1]

	cases := []struct {
		name string
		buf  []byte
		want *serializer
	}{
		{name: "inactive", buf: []byte{0x00}, want: nil},
		{name: "active index 0", buf: []byte{0x01}, want: modeA},
		{name: "active index 1", buf: []byte{0x03}, want: modeB},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newReader(c.buf)

			val := polyField.baseDecoder(r)

			pu, ok := val.(*polyUpdate)
			require.True(t, ok, "expected *polyUpdate, got %T", val)
			assert.Equal(t, 0, pu.id)
			assert.Same(t, c.want, pu.ser)
		})
	}
}

// TestPolyPerEntityIndependence is the core regression test guarding the
// flyweight-hazard spheenik warned about: the active polymorphic type must be
// tracked per entity, not on the shared field object, so two entities of the
// same class can hold different active types simultaneously.
func TestPolyPerEntityIndependence(t *testing.T) {
	t.Parallel()

	cls, modeA, modeB := polyTestClass(t)

	e1 := newEntity(1, 1, cls)
	e2 := newEntity(2, 1, cls)

	e1.polySerializers[0] = modeA
	e2.polySerializers[0] = modeB

	// e1 resolves mode A's sub-fields only.
	assert.NotNil(t, e1.Property("m_pMode.m_aField"))
	assert.Nil(t, e1.Property("m_pMode.m_bField"))

	// e2 resolves mode B's sub-fields only.
	assert.Nil(t, e2.Property("m_pMode.m_aField"))
	assert.NotNil(t, e2.Property("m_pMode.m_bField"))
}

// TestPolyInactivePointerModelsUnresolvableBeforeActivation asserts that the
// sub-fields of a polymorphic pointer cannot be resolved while the pointer is
// inactive (no serializer selected).
func TestPolyInactivePointerModelsUnresolvableBeforeActivation(t *testing.T) {
	t.Parallel()

	cls, _, _ := polyTestClass(t)

	e := newEntity(1, 1, cls)

	assert.Nil(t, e.Property("m_pMode.m_aField"))
	assert.Nil(t, e.Property("m_pMode.m_bField"))
}

// TestPolyTypeSwitchInvalidatesCaches asserts that switching the active type
// invalidates name resolution and refreshes handlers: a handler bound to the
// old type's sub-field must no longer fire for the same numeric path under the
// new type, while a handler bound to the new type's sub-field must fire.
func TestPolyTypeSwitchInvalidatesCaches(t *testing.T) {
	t.Parallel()

	cls, modeA, modeB := polyTestClass(t)

	e := newEntity(1, 1, cls)

	// Activate mode A and bind a handler for its sub-field.
	e.polySerializers[0] = modeA

	var aFired, bFired bool
	e.Property("m_pMode.m_aField").OnUpdate(func(st.PropertyValue) {
		aFired = true
	})

	// A property of the old type must resolve before the switch.
	assert.NotNil(t, e.Property("m_pMode.m_aField"))

	// Switch to mode B via the poly update path.
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeB})

	// Old-type property is gone, new-type property resolves.
	assert.Nil(t, e.Property("m_pMode.m_aField"))
	assert.NotNil(t, e.Property("m_pMode.m_bField"))

	// Bind a handler for the new type's sub-field; this is only possible after
	// activation, mirroring real usage.
	e.Property("m_pMode.m_bField").OnUpdate(func(st.PropertyValue) {
		bFired = true
	})

	// Dispatching an update on the shared numeric path must fire only the
	// new type's handler.
	e.dispatchUpdate(polySubPath(), int32(7))

	assert.False(t, aFired, "old-type handler must not fire after type switch")
	assert.True(t, bFired, "new-type handler must fire after type switch")
}

// TestPolyNestedFieldPaths is a regression test for a pre-existing bug in
// getFieldPaths: tables nested under other tables were omitted from
// Properties()/Map()/String() because after descending, the state lookup used
// the full root-scoped path against the enclosing table's fieldState. The
// polymorphic pointer's sub-fields (a fixed table under m_pMode, itself a fixed
// table) were the first such field to appear in CS2 data.
//
// The active type's m_cfg table must be enumerated as m_pMode.m_cfg.m_cfgX.
func TestPolyNestedFieldPaths(t *testing.T) {
	t.Parallel()

	cls, modeA, _ := polyTestClass(t)

	// Add a nested fixed table to modeA: CModeA { m_aField, m_cfg: CCfg }
	cfg := newSerializer("CCfg", 0)
	cfg.addField(simpleField("m_cfgX", ""))
	modeA.addField(&field{
		varName:          "m_cfg",
		name:             "m_cfg",
		varType:          "CCfg*",
		model:            fieldModelFixedTable,
		serializer:       cfg,
		polySerializerID: -1,
	})

	e := newEntity(1, 1, cls)
	e.polySerializers[0] = modeA

	// No poly sub-state written yet: only the top-level simple field appears.
	assert.Equal(t, []string{"m_iHealth"}, propertyNames(t, e))

	// Write the nested table's state: m_pMode (index 1) → m_cfg → m_cfgX.
	fp := newFieldPath()
	fp.path[0] = 1 // m_pMode
	fp.path[1] = 1 // m_cfg
	fp.path[2] = 0 // m_cfgX
	fp.last = 2
	e.state.set(fp, int32(9))

	// m_pMode now has a fieldState, so its simple field (m_aField) and the
	// nested table (m_cfg.m_cfgX) are both enumerated.
	names := propertyNames(t, e)
	assert.Equal(t, []string{"m_iHealth", "m_pMode.m_aField", "m_pMode.m_cfg.m_cfgX"}, names)
}

// TestPolyHandlerOrderRegistrationStable asserts that handlers sharing an fp key
// (here via a deprecated bare-name alias resolving to the same field as
// m_pMode.m_aField) fire in their original registration order and that this
// order is preserved across poly type-change rebuilds.
func TestPolyHandlerOrderRegistrationStable(t *testing.T) {
	t.Parallel()

	cls, modeA, modeB := polyTestClass(t)

	// Simulate the deprecated bare-name alias rebuildFieldIndexes creates for
	// disambiguated field names: both names resolve to the same field path.
	modeA.fieldIndexes["m_alias"] = modeA.fieldIndexes["m_aField"]

	e := newEntity(1, 1, cls)
	e.polySerializers[0] = modeA

	var fired []string

	// Registration order is alias-then-aField; both resolve to the same path.
	e.Property("m_pMode.m_alias").OnUpdate(func(st.PropertyValue) {
		fired = append(fired, "alias")
	})
	e.Property("m_pMode.m_aField").OnUpdate(func(st.PropertyValue) {
		fired = append(fired, "aField")
	})

	// Both handlers resolve to the same fp key.
	key, ok := fpFlatKey(polySubPath())
	require.True(t, ok)
	assert.Len(t, e.handlersByFP[key], 2)

	// Registration order is preserved before any rebuild has run.
	e.dispatchUpdate(polySubPath(), int32(1))
	assert.Equal(t, []string{"alias", "aField"}, fired)

	// After a type change the fp-keyed index is rebuilt; handlers must still
	// fire in registration order, not name-sorted or map-iteration order.
	fired = nil
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeB})
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})

	e.dispatchUpdate(polySubPath(), int32(1))
	assert.Equal(t, []string{"alias", "aField"}, fired)
}

// TestInitialHandlerDispatchOrder asserts that the initial handler dispatch on
// entity creation fires all registered handlers in registration order.
func TestInitialHandlerDispatchOrder(t *testing.T) {
	t.Parallel()

	cls, modeA, _ := polyTestClass(t)

	e := newEntity(1, 1, cls)
	e.polySerializers[0] = modeA

	// Write some state so the initial values can be read back.
	e.state.set(polySubPath(), int32(5))

	var fired []string

	// Register on two different names in a known order that is the reverse of
	// name-sorted order ("m_iHealth" < "m_pMode.m_aField").
	e.Property("m_pMode.m_aField").OnUpdate(func(st.PropertyValue) {
		fired = append(fired, "aField")
	})
	e.Property("m_iHealth").OnUpdate(func(st.PropertyValue) {
		fired = append(fired, "health")
	})

	// fireInitialUpdateHandlers must dispatch in registration order, not the
	// name-sorted order the created-handler dispatch previously used.
	e.fireInitialUpdateHandlers()

	assert.Equal(t, []string{"aField", "health"}, fired)
}

// propertyNames returns the generated names of all field paths in e's state.
func propertyNames(t *testing.T, e *Entity) []string {
	t.Helper()

	fps := e.class.getFieldPaths(newFieldPath(), e.state, e.polySerializers)
	names := make([]string, 0, len(fps))
	for _, fp := range fps {
		names = append(names, e.class.getNameForFieldPath(fp, e.polySerializers))
	}

	return names
}

// TestPolyFieldPaths asserts that getFieldPaths/Properties only expose the
// sub-fields of the currently active type.
func TestPolyFieldPaths(t *testing.T) {
	t.Parallel()

	cls, modeA, modeB := polyTestClass(t)

	e := newEntity(1, 1, cls)

	// No sub-state yet: no sub-field paths.
	assert.Equal(t, []string{"m_iHealth"}, propertyNames(t, e))

	// Activate mode A and write its sub-field state.
	e.polySerializers[0] = modeA
	e.state.set(polySubPath(), int32(5))

	assert.Equal(t, []string{"m_iHealth", "m_pMode.m_aField"}, propertyNames(t, e))

	// Switch to mode B. The state slot still holds a fieldState laid out under
	// mode A, which getFieldPaths interprets under the new active type's layout
	// (index 0 = m_bField). Types don't switch mid-match in practice, so this
	// stale-state behavior is accepted and documented here.
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeB})
	assert.Equal(t, []string{"m_iHealth", "m_pMode.m_bField"}, propertyNames(t, e))
}

// TestMaxPolyID verifies the reachability computation used to size per-entity
// polySerializers slices: direct poly fields, nested via sub-serializers, via
// polymorphic alternatives, and cycle safety.
func TestMaxPolyID(t *testing.T) {
	t.Parallel()

	// Cycle guard: A's sub-serializer references back into A.
	a := newSerializer("CMod", 0)
	b := newSerializer("CMod.Sub", 0)
	aField := &field{model: fieldModelFixedTable, serializer: a, polySerializerID: -1}
	subField := &field{model: fieldModelFixedTable, serializer: a, polySerializerID: -1}
	b.addField(subField)
	a.addField(&field{model: fieldModelFixedTable, serializer: b, polySerializerID: -1})

	// Direct poly field with ID 3 nested under a poly alternative (ID 1).
	alt := newSerializer("CAlt", 0)
	alt.addField(&field{model: fieldModelFixedTable, polySerializerID: 3})
	poly := &field{polyTypes: []*serializer{nil, alt}, polySerializerID: 1}

	root := newSerializer("CRoot", 0)
	root.addField(aField)
	root.addField(poly)

	cache := make(map[*serializer]int)
	assert.Equal(t, 3, root.maxPolyID(cache))

	// A fresh root without alternative nesting only reaches the cycle fields.
	root2 := newSerializer("CRoot2", 0)
	root2.addField(aField)
	assert.Equal(t, -1, root2.maxPolyID(make(map[*serializer]int)))

	// nil serializer is safe.
	var nilSer *serializer
	assert.Equal(t, -1, nilSer.maxPolyID(make(map[*serializer]int)))
}

// TestMaxPolyIDCycleDoesNotCacheUnderstated guards the memoization in
// maxPolyID: a serializer whose poly reachability flows only through a cycle
// back-edge must not be cached with an understated value, otherwise a later
// top-level call returns the stale result and a class backed by it would get
// an undersized polySerializers slice (index-out-of-range on the first poly
// update).
func TestMaxPolyIDCycleDoesNotCacheUnderstated(t *testing.T) {
	t.Parallel()

	a := newSerializer("CMod", 0)
	b := newSerializer("CMod.Sub", 0)
	a.addField(&field{model: fieldModelFixedTable, serializer: b, polySerializerID: -1})
	b.addField(&field{model: fieldModelFixedTable, serializer: a, polySerializerID: -1})
	a.addField(&field{model: fieldModelFixedTable, polySerializerID: 5})

	cache := make(map[*serializer]int)

	// Computing a first traverses the a↔b cycle; b's true value (5, via a)
	// must not be cached as if it had none.
	assert.Equal(t, 5, a.maxPolyID(cache))

	// b must recompute to the full reachable value, not a stale understated -1.
	assert.Equal(t, 5, b.maxPolyID(cache))
}

// TestParsePacketBackPatchesPolyCount covers the class-info-before-FSV ordering:
// ParsePacket must back-patch both the serializer and the poly slot count onto
// the class created by OnDemoClassInfo, so entities get a correctly sized
// polySerializers slice.
func TestParsePacketBackPatchesPolyCount(t *testing.T) {
	t.Parallel()

	p := NewParser(nil)

	err := p.OnDemoClassInfo(&msg.CDemoClassInfo{
		Classes: []*msg.CDemoClassInfoClassT{
			{ClassId: proto.Int32(1), NetworkName: proto.String("CEnt")},
		},
	})
	require.NoError(t, err)

	cls := p.classesByName["CEnt"]
	require.NotNil(t, cls)
	assert.Nil(t, cls.serializer) // not yet linked
	assert.Equal(t, 0, cls.polyCount)

	err = p.ParsePacket(polyFSVBytes(t))
	require.NoError(t, err)

	assert.NotNil(t, cls.serializer)
	assert.Equal(t, 1, cls.polyCount)
}

// TestNewFieldPanicsOnUnknownPolySerializer asserts that a polymorphic field
// referencing a serializer that is missing or forward-declared fails fast with
// a clear panic instead of silently decoding as an inactive pointer.
func TestNewFieldPanicsOnUnknownPolySerializer(t *testing.T) {
	t.Parallel()

	ser := &msg.CSVCMsg_FlattenedSerializer{
		Symbols: []string{"m_pMode", "CMode*", "CModeA", "Missing"},
	}

	// Symbols index: m_pMode=0, CMode*=1, CModeA=2, Missing=3.
	int32p := func(n int) *int32 { return proto.Int32(int32(n)) }

	cases := []struct {
		name        string
		serializers map[string]*serializer
		baseSym     *int32
		polySyms    []*msg.ProtoFlattenedSerializerFieldTPolymorphicFieldT
		wantPanic   string
	}{
		{
			name:        "missing base serializer",
			serializers: map[string]*serializer{},
			baseSym:     int32p(3),
			polySyms: []*msg.ProtoFlattenedSerializerFieldTPolymorphicFieldT{
				{PolymorphicFieldSerializerNameSym: int32p(2)}, // CModeA
			},
			wantPanic: `polymorphic field "m_pMode": unknown serializer "Missing" (missing or forward-declared)`,
		},
		{
			name:        "missing poly alternative serializer",
			serializers: map[string]*serializer{"CModeA": newSerializer("CModeA", 0)},
			baseSym:     int32p(2), // CModeA
			polySyms: []*msg.ProtoFlattenedSerializerFieldTPolymorphicFieldT{
				{PolymorphicFieldSerializerNameSym: int32p(3)}, // Missing
			},
			wantPanic: `polymorphic field "m_pMode": unknown serializer "Missing" (missing or forward-declared)`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &msg.ProtoFlattenedSerializerFieldT{
				VarNameSym:             int32p(0),
				VarTypeSym:             int32p(1),
				FieldSerializerNameSym: c.baseSym,
				PolymorphicTypes:       c.polySyms,
			}

			assert.PanicsWithValue(t, c.wantPanic, func() {
				newField(c.serializers, ser, f)
			})
		})
	}
}

// polyFSVBytes marshals the polyTestSchema as a FlattenedSerializer message,
// length-prefixed as ParsePacket expects.
func polyFSVBytes(t *testing.T) []byte {
	t.Helper()

	symbols := []string{"CEnt", "CModeA", "CModeB", "m_iHealth", "m_pMode", "m_aField", "m_bField"}
	symIdx := map[string]int32{}
	for i, s := range symbols {
		symIdx[s] = int32(i)
	}

	field := func(name, vartype, ser string, polyNames ...string) *msg.ProtoFlattenedSerializerFieldT {
		x := &msg.ProtoFlattenedSerializerFieldT{
			VarNameSym: proto.Int32(symIdx[name]),
			VarTypeSym: proto.Int32(symIdx[vartype]),
		}
		if ser != "" {
			x.FieldSerializerNameSym = proto.Int32(symIdx[ser])
		}
		for _, pn := range polyNames {
			x.PolymorphicTypes = append(x.PolymorphicTypes, &msg.ProtoFlattenedSerializerFieldTPolymorphicFieldT{
				PolymorphicFieldSerializerNameSym: proto.Int32(symIdx[pn]),
			})
		}

		return x
	}

	m := &msg.CSVCMsg_FlattenedSerializer{
		Symbols: symbols,
		Fields: []*msg.ProtoFlattenedSerializerFieldT{
			// Sub-fields use a simple varType so they decode as plain values;
			// only m_pMode's pointer varType plus its serializer references make
			// it a polymorphic fixed-table field.
			field("m_iHealth", "int32", ""),                // index 0
			field("m_pMode", "CMode*", "CModeA", "CModeB"), // index 1 (poly fixed-table)
			field("m_aField", "int32", ""),                 // index 2
			field("m_bField", "int32", ""),                 // index 3
		},
		Serializers: []*msg.ProtoFlattenedSerializerT{
			{SerializerNameSym: proto.Int32(symIdx["CModeA"]), FieldsIndex: []int32{2}},
			{SerializerNameSym: proto.Int32(symIdx["CModeB"]), FieldsIndex: []int32{3}},
			{
				SerializerNameSym: proto.Int32(symIdx["CEnt"]),
				FieldsIndex:       []int32{0, 1},
			},
		},
	}

	b, err := proto.Marshal(m)
	require.NoError(t, err)

	// Length-prefix as ParsePacket expects.
	var lenBuf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(lenBuf[:], uint64(len(b)))

	return append(lenBuf[:n], b...)
}

// TestPolyStateSlotSemantics codifies the documented state-slot behavior: the
// activation bool is readable while the pointer has no sub-fields, and is
// replaced by a *fieldState once sub-field updates arrive (same as regular
// fixed pointers).
func TestPolyStateSlotSemantics(t *testing.T) {
	t.Parallel()

	cls, modeA, _ := polyTestClass(t)

	e := newEntity(1, 1, cls)

	// Simulate activation: the decoder converts the polyUpdate to a bool, stored
	// at the pointer field's own path.
	e.applyPolyUpdate(&polyUpdate{id: 0, ser: modeA})
	modePath := newFieldPath()
	modePath.path[0] = 1 // m_pMode
	e.state.set(modePath, true)

	val := e.Get("m_pMode")
	require.NotNil(t, val)
	assert.Equal(t, true, val) // bool while no sub-fields

	// Writing a sub-field replaces the bool with a *fieldState.
	e.state.set(polySubPath(), int32(5))

	val = e.Get("m_pMode")
	require.NotNil(t, val)
	_, isFS := val.(*fieldState)
	assert.True(t, isFS, "expected *fieldState, got %T", val)
}

// TestPolyPropertyEntries asserts that PropertyEntries enumerates the
// sub-fields of every polymorphic alternative, not just the default serializer,
// so schema inspection sees all types a polymorphic pointer could activate.
func TestPolyPropertyEntries(t *testing.T) {
	t.Parallel()

	cls, _, _ := polyTestClass(t)

	assert.Equal(t, []string{"m_iHealth", "CModeA.m_aField", "CModeB.m_bField"}, cls.PropertyEntries())
}
