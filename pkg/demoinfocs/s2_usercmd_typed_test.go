package demoinfocs

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

// The typed merge (s2_usercmd_typed.go) is tested against the protobuf-reflection merge it replaced
// (s2_usercmd_delta_reflect_test.go): on random baselines and random deltas, valid and malformed, both must return the
// same error, and when they don't fail, the same message, byte for byte.

var userCmdMergeImpls = map[string]func(*msg.CSGOUserCmdPB, []byte) error{
	"typed":   applyUserCmdDelta,
	"reflect": applyUserCmdDeltaReflect,
}

func errText(err error) string {
	if err == nil {
		return "<nil>"
	}

	return err.Error()
}

func TestUserCmdTypedMergeMatchesReflect(t *testing.T) {
	const cases = 20000

	r := rand.New(rand.NewPCG(2026, 11))
	g := &deltaGen{r: r}

	var ok, failed, withUnknown int

	errKinds := map[string]int{}
	numbers := regexp.MustCompile(`\d+`)

	for i := 0; i < cases; i++ {
		base := randomUserCmd(r)
		baseBytes := mustMarshal(t, base)
		delta := g.message(base.ProtoReflect().Descriptor(), 3)

		ref := proto.Clone(base).(*msg.CSGOUserCmdPB)
		typed := copyUserCmd(base)

		errRef := applyUserCmdDeltaReflect(ref, delta)
		errTyped := applyUserCmdDelta(typed, delta)
		require.Equal(t, errText(errRef), errText(errTyped), "case %d: the errors differ, delta %x on %x", i, delta, baseBytes)

		if errRef != nil {
			failed++
			errKinds[numbers.ReplaceAllString(errRef.Error(), "#")]++
		} else {
			ok++

			require.Equal(t, mustMarshal(t, ref), mustMarshal(t, typed), "case %d: the results differ, delta %x on %x", i, delta, baseBytes)
			require.True(t, proto.Equal(ref, typed), "case %d: proto.Equal is false, delta %x on %x", i, delta, baseBytes)

			if hasUnknownFields(typed.ProtoReflect()) {
				withUnknown++
			}
		}

		require.Equal(t, baseBytes, mustMarshal(t, base), "case %d: the baseline was modified, delta %x", i, delta)
	}

	t.Logf("%d cases: %d merged (%d with unknown fields), %d failed with the same error", cases, ok, withUnknown, failed)

	for kind, n := range errKinds {
		t.Logf("  %6d  %s", n, kind)
	}

	assert.Greater(t, ok, cases/2, "most deltas should be valid")
	assert.Greater(t, withUnknown, cases/100, "values with the wrong wire type should be kept as unknown fields")

	// every error of the merge has come up (the numbers in the messages are #; protowire's messages are matched
	// without their "proto:" prefix, whose spacing protobuf-go varies from build to build on purpose)
	for _, want := range []string{
		"failed to read delta field tag: unexpected EOF",
		"failed to read delta field tag: ",
		"invalid field number",
		"failed to read delta field value: unexpected EOF",
		"cannot parse reserved wire type",
		"mismatching end group marker",
		"failed to read length-delimited delta field: ",
		"wire type # references unknown field #",
		"failed to read repeated delta index: ",
		"repeated delta index # exceeds limit #",
		"repeated delta index # is out of bounds for length #",
		"failed to read repeated delta message: ",
		"unsupported repeated delta wire type #",
		"failed to merge repeated field com.github.markus_wa.demoinfocs_golang.v#.CBaseUserCmdPB.subtick_moves[#]: ",
		"failed to merge repeated field com.github.markus_wa.demoinfocs_golang.v#.CSGOUserCmdPB.input_history[#]: ",
	} {
		var n int

		for kind, count := range errKinds {
			if strings.Contains(kind, want) {
				n += count
			}
		}

		assert.Positive(t, n, "no delta failed with %q", want)
	}
}

func TestCopyUserCmd_MatchesClone(t *testing.T) {
	assert.Nil(t, copyUserCmd(nil))

	r := rand.New(rand.NewPCG(7, 42))

	for i := 0; i < 5000; i++ {
		base := randomUserCmd(r)

		// what the random messages don't produce: nil list elements (proto.Clone copies them as empty messages),
		// an empty but present bytes field, an empty but present list
		switch r.IntN(5) {
		case 0:
			if base.Base != nil && len(base.Base.SubtickMoves) > 0 {
				base.Base.SubtickMoves[r.IntN(len(base.Base.SubtickMoves))] = nil
			}
		case 1:
			if len(base.InputHistory) > 0 {
				base.InputHistory[r.IntN(len(base.InputHistory))] = nil
			}
		case 2:
			if base.Base != nil {
				base.Base.MoveCrc = []byte{}
			}
		case 3:
			base.InputHistory = []*msg.CSGOInputHistoryEntryPB{}
		}

		want := proto.Clone(base).(*msg.CSGOUserCmdPB)
		got := copyUserCmd(base)

		require.True(t, proto.Equal(want, got), "case %d: %v != %v", i, want, got)
		require.Equal(t, mustMarshal(t, want), mustMarshal(t, got), "case %d", i)
	}
}

// A copy shares nothing with the command it was copied from: changing every field of the copy leaves the original as
// it was, as proto.Clone does, so the ring's baselines are safe from the deltas applied to their copies.
func TestCopyUserCmd_IsIndependent(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	g := &deltaGen{r: r}

	for i := 0; i < 2000; i++ {
		base := randomUserCmd(r)
		baseBytes := mustMarshal(t, base)

		c := copyUserCmd(base)
		if base.Base != nil && base.Base.ClientTick != nil {
			assert.NotSame(t, base.Base.ClientTick, c.Base.ClientTick, "case %d: a scalar is shared", i)
		}

		for j := 0; j < 3; j++ {
			_ = applyUserCmdDelta(c, g.message(c.ProtoReflect().Descriptor(), 3))
		}

		if c.Base != nil && c.Base.MoveCrc != nil {
			for k := range c.Base.MoveCrc {
				c.Base.MoveCrc[k]++
			}
		}

		require.Equal(t, baseBytes, mustMarshal(t, base), "case %d: the original changed", i)
	}
}

// A reset marker that follows the field's value in one delta doesn't undo the value: the reflective merge collected the
// values and had the protobuf decoder apply them after the loop that handled the markers, and the typed merge keeps that
// order. A nested message is not a value, so its reset marker always applies.
func TestApplyUserCmdDelta_ValueThenReset(t *testing.T) {
	tick := func(v uint64) []byte { return varintField(fieldBaseUserCmdClientTick, v) }
	reset := resetMarker(fieldBaseUserCmdClientTick)
	buttons := bytesField(fieldBaseUserCmdButtonsPb, varintField(fieldButtonsState1, 8))

	for name, apply := range userCmdMergeImpls {
		t.Run(name, func(t *testing.T) {
			m := baseline()
			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, append(tick(7), reset...))))
			assert.EqualValues(t, 7, m.GetBase().GetClientTick(), "the value survives a later reset marker")

			m = baseline()
			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, append(reset, tick(7)...))))
			assert.EqualValues(t, 7, m.GetBase().GetClientTick(), "the value follows the reset marker")

			m = baseline()
			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, concat(tick(7), reset, tick(9)))))
			assert.EqualValues(t, 9, m.GetBase().GetClientTick(), "the last value wins")

			m = baseline()
			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, append(buttons, resetMarker(fieldBaseUserCmdButtonsPb)...))))
			assert.Nil(t, m.GetBase().GetButtonsPb(), "a nested message's reset marker applies after its delta")

			m = baseline()
			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, append(resetMarker(fieldBaseUserCmdButtonsPb), buttons...))))
			assert.EqualValues(t, 8, m.GetBase().GetButtonsPb().GetButtonstate1())
			assert.Nil(t, m.GetBase().GetButtonsPb().Buttonstate2, "the delta merges into a fresh message after the reset marker")
		})
	}
}

// The value of a known field with another wire type than the field's is kept as an unknown field, as proto.Unmarshal
// does, and the field keeps its value.
func TestApplyUserCmdDelta_WrongWireTypeIsUnknownField(t *testing.T) {
	tag := protowire.AppendTag(nil, fieldBaseUserCmdClientTick, protowire.Fixed32Type)
	record := protowire.AppendFixed32(tag, 0x01020304)

	for name, apply := range userCmdMergeImpls {
		t.Run(name, func(t *testing.T) {
			m := baseline()
			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, record)))

			assert.EqualValues(t, 100, m.GetBase().GetClientTick(), "the field keeps its value")
			assert.Equal(t, record, []byte(m.GetBase().ProtoReflect().GetUnknown()), "the record is kept as an unknown field")
			assert.Empty(t, m.ProtoReflect().GetUnknown(), "on the message that has the field, not its parent")
		})
	}
}

// A field number the message doesn't have is skipped whatever its wire type, except as a reset marker, which is an
// error.
func TestApplyUserCmdDelta_UnknownFieldIgnored(t *testing.T) {
	unknown := concat(
		varintField(99, 5),
		bytesField(98, []byte("ab")),
		protowire.AppendFixed32(protowire.AppendTag(nil, 97, protowire.Fixed32Type), 1),
		protowire.AppendFixed64(protowire.AppendTag(nil, 96, protowire.Fixed64Type), 1),
	)

	for name, apply := range userCmdMergeImpls {
		t.Run(name, func(t *testing.T) {
			m := baseline()
			want := mustMarshal(t, m)

			require.NoError(t, apply(m, concat(unknown, bytesField(fieldCSGOUserCmdBase, unknown))))
			assert.Equal(t, want, mustMarshal(t, m), "nothing changes, no unknown fields are kept")

			err := apply(m, bytesField(fieldCSGOUserCmdBase, resetMarker(99)))
			assert.EqualError(t, err, "wire type 7 references unknown field 99")
		})
	}
}

// A nested message's delta creates the message when the baseline doesn't have it, even an empty delta.
func TestApplyUserCmdDelta_NestedDeltaCreatesMessage(t *testing.T) {
	for name, apply := range userCmdMergeImpls {
		t.Run(name, func(t *testing.T) {
			m := &msg.CSGOUserCmdPB{}
			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, nil)))
			require.NotNil(t, m.Base)
			assert.True(t, proto.Equal(&msg.CBaseUserCmdPB{}, m.Base))

			require.NoError(t, apply(m, bytesField(fieldCSGOUserCmdBase, bytesField(fieldBaseUserCmdButtonsPb, varintField(fieldButtonsState1, 8)))))
			assert.EqualValues(t, 8, m.GetBase().GetButtonsPb().GetButtonstate1())
		})
	}
}

// A nil element of a repeated field is allocated when a delta addresses it (the reflective merge couldn't merge into
// it). The parser never has nil elements, proto.Unmarshal doesn't produce them.
func TestApplyUserCmdDelta_NilListElement(t *testing.T) {
	m := &msg.CSGOUserCmdPB{Base: &msg.CBaseUserCmdPB{SubtickMoves: []*msg.CSubtickMoveStep{nil}}}
	delta := bytesField(fieldCSGOUserCmdBase, bytesField(fieldBaseUserCmdSubtick, repeatedPatch(0, varintField(fieldSubtickButton, 3))))

	require.NoError(t, applyUserCmdDelta(m, delta))
	assert.EqualValues(t, 3, m.GetBase().GetSubtickMoves()[0].GetButton())
}

// The benchmark's fixture is a valid delta that both merges read the same way.
func TestBenchmarkDeltaIsValid(t *testing.T) {
	ref := benchmarkCommand()
	typed := benchmarkCommand()

	require.NoError(t, applyUserCmdDeltaReflect(ref, benchmarkDelta()))
	require.NoError(t, applyUserCmdDelta(typed, benchmarkDelta()))
	assert.Equal(t, mustMarshal(t, ref), mustMarshal(t, typed))

	assert.EqualValues(t, 2417, typed.GetBase().GetClientTick())
	assert.InDelta(t, 83.40295, typed.GetBase().GetViewangles().GetY(), 0.0001)
	assert.EqualValues(t, -2, typed.GetBase().GetMousedy())
	assert.Len(t, typed.GetInputHistory(), 3)
	assert.EqualValues(t, 2418, typed.GetInputHistory()[2].GetRenderTickCount())
}

func FuzzApplyUserCmdDelta(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{0xff},
		{0x00},
		benchmarkDelta(),
		buttonsDelta(8),
		clientTickDelta(5),
		bytesField(fieldCSGOUserCmdBase, resetMarker(fieldBaseUserCmdButtonsPb)),
		bytesField(fieldCSGOUserCmdBase, bytesField(fieldBaseUserCmdSubtick, protowire.AppendTag(nil, 2, wireTypeReset))),
		bytesField(fieldCSGOUserCmdHistory, repeatedPatch(0, varintField(fieldHistoryFrame, 11))),
	} {
		f.Add(seed)
	}

	base := benchmarkCommand()
	base.ProtoReflect().SetUnknown(protowire.AppendVarint(protowire.AppendTag(nil, 1000, protowire.VarintType), 1))

	f.Fuzz(func(t *testing.T, delta []byte) {
		ref := proto.Clone(base).(*msg.CSGOUserCmdPB)
		typed := copyUserCmd(base)

		errRef := applyUserCmdDeltaReflect(ref, delta)
		errTyped := applyUserCmdDelta(typed, delta)

		if errText(errRef) != errText(errTyped) {
			t.Fatalf("the errors differ: reflect %q, typed %q", errText(errRef), errText(errTyped))
		}

		if errRef == nil && !proto.Equal(ref, typed) {
			t.Fatalf("the results differ:\nreflect %v\ntyped   %v", ref, typed)
		}
	})
}

// benchmarkCommand is shaped like the full commands of a CS2 demo: a base with buttons, view angles and movement, one
// subtick move, and three input history entries with view angles and tick fractions.
func benchmarkCommand() *msg.CSGOUserCmdPB {
	history := func(tick int32, frac float32) *msg.CSGOInputHistoryEntryPB {
		return &msg.CSGOInputHistoryEntryPB{
			ViewAngles:         &msg.CMsgQAngle{X: protoFloat(0.8246081), Y: protoFloat(83.52075), Z: protoFloat(0)},
			RenderTickCount:    proto32(tick),
			RenderTickFraction: protoFloat(frac),
			PlayerTickCount:    proto32(tick + 8),
			PlayerTickFraction: protoFloat(frac / 2),
		}
	}

	return &msg.CSGOUserCmdPB{
		Base: &msg.CBaseUserCmdPB{
			ClientTick:                proto32(2416),
			PredictionOffsetTicksX256: protoUint32(2023),
			ButtonsPb:                 &msg.CInButtonStatePB{Buttonstate1: proto64(1024), Buttonstate2: proto64(1)},
			Viewangles:                &msg.CMsgQAngle{X: protoFloat(0.8246081), Y: protoFloat(83.52075)},
			Forwardmove:               protoFloat(1),
			Leftmove:                  protoFloat(-1),
			RandomSeed:                proto32(1197119452),
			PawnEntityHandle:          protoUint32(3342632),
			SubtickMoves:              []*msg.CSubtickMoveStep{{Button: proto64(1), Pressed: protoBool(false), When: protoFloat(0.21875)}},
		},
		InputHistory:             []*msg.CSGOInputHistoryEntryPB{history(2414, 0.8747724), history(2415, 0.112846024), history(2415, 0.40918523)},
		IsPredictingBodyShotFx:   protoBool(false),
		IsPredictingHeadShotFx:   protoBool(false),
		IsPredictingKillRagdolls: protoBool(false),
	}
}

// benchmarkDelta is shaped like the deltas of a CS2 demo: a new view angle, random seed, client tick and mouse
// movement, and three input history entries with their view angles and tick fractions, then the history's length.
func benchmarkDelta() []byte {
	entry := func(tick int32, frac float32) []byte {
		return concat(
			bytesField(2, fixed32Field(2, 83.40295)), // view_angles.y
			varintField(4, uint64(tick)),             // render_tick_count
			fixed32Field(5, frac),                    // render_tick_fraction
			varintField(6, uint64(tick+8)),           // player_tick_count
			fixed32Field(7, frac/2),                  // player_tick_fraction
		)
	}

	mousedy := int64(-2)
	base := concat(
		varintField(fieldBaseUserCmdClientTick, 2417),
		bytesField(4, fixed32Field(2, 83.40295)), // viewangles.y
		varintField(10, 1661920067),              // random_seed
		varintField(11, 3),                       // mousedx
		varintField(12, uint64(mousedy)),         // mousedy
	)
	history := concat(
		repeatedPatch(0, entry(2416, 0.024404433)),
		repeatedPatch(1, entry(2416, 0.29259643)),
		repeatedPatch(2, entry(2418, 0.50621563)),
		protowire.AppendVarint(nil, 3<<3|uint64(wireTypeReset)), // length 3
	)

	return concat(bytesField(fieldCSGOUserCmdBase, base), bytesField(fieldCSGOUserCmdHistory, history))
}

var benchmarkSink *msg.CSGOUserCmdPB

// BenchmarkUserCmdMerge applies the same delta to one command over and over: the merge alone, without the copy.
func BenchmarkUserCmdMerge(b *testing.B) {
	delta := benchmarkDelta()

	for name, apply := range userCmdMergeImpls {
		b.Run(name, func(b *testing.B) {
			m := benchmarkCommand()

			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				if err := apply(m, delta); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUserCmdCopy(b *testing.B) {
	base := benchmarkCommand()

	b.Run("typed", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchmarkSink = copyUserCmd(base)
		}
	})

	b.Run("proto.Clone", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchmarkSink = proto.Clone(base).(*msg.CSGOUserCmdPB)
		}
	})
}

// BenchmarkUserCmdDecode is one delta command as handleUserCommands decodes it: a copy of the baseline, then the merge.
func BenchmarkUserCmdDecode(b *testing.B) {
	base := benchmarkCommand()
	delta := benchmarkDelta()

	b.Run("typed", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchmarkSink = copyUserCmd(base)

			if err := applyUserCmdDelta(benchmarkSink, delta); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("reflect", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			benchmarkSink = proto.Clone(base).(*msg.CSGOUserCmdPB)

			if err := applyUserCmdDeltaReflect(benchmarkSink, delta); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func protoUint32(v uint32) *uint32 { return &v }

func fixed32Field(field protowire.Number, v float32) []byte {
	b := protowire.AppendTag(nil, field, protowire.Fixed32Type)

	return protowire.AppendFixed32(b, math.Float32bits(v))
}

func concat(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

// hasUnknownFields reports whether m or a message in it has unknown fields.
func hasUnknownFields(m protoreflect.Message) bool {
	if len(m.GetUnknown()) > 0 {
		return true
	}

	found := false

	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Message() != nil:
			list := v.List()
			for i := 0; i < list.Len() && !found; i++ {
				found = hasUnknownFields(list.Get(i).Message())
			}
		case fd.Message() != nil:
			found = hasUnknownFields(v.Message())
		}

		return !found
	})

	return found
}

// randomUserCmd returns a random command: each field is present with probability 2/3 (default values included, proto2
// fields have presence), nested messages and lists down to three levels, unknown fields on some messages.
func randomUserCmd(r *rand.Rand) *msg.CSGOUserCmdPB {
	m := &msg.CSGOUserCmdPB{}
	randomMessage(r, m.ProtoReflect(), 3)

	return m
}

func randomMessage(r *rand.Rand, m protoreflect.Message, depth int) {
	fields := m.Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if r.IntN(3) == 0 {
			continue
		}

		switch {
		case fd.IsList():
			if depth == 0 {
				continue
			}

			list := m.Mutable(fd).List()
			for n := r.IntN(4); n > 0; n-- {
				e := list.NewElement()
				randomMessage(r, e.Message(), depth-1)
				list.Append(e)
			}
		case fd.Message() != nil:
			if depth == 0 {
				continue
			}

			randomMessage(r, m.Mutable(fd).Message(), depth-1)
		default:
			m.Set(fd, randomScalar(r, fd))
		}
	}

	if r.IntN(8) == 0 {
		m.SetUnknown(randomUnknownFields(r, m.Descriptor()))
	}
}

//nolint:exhaustive // the kinds of the user command messages' fields
func randomScalar(r *rand.Rand, fd protoreflect.FieldDescriptor) protoreflect.Value {
	zero := r.IntN(4) == 0

	switch fd.Kind() {
	case protoreflect.Int32Kind:
		if zero {
			return protoreflect.ValueOfInt32(0)
		}

		return protoreflect.ValueOfInt32(int32(r.Uint32()))
	case protoreflect.Uint32Kind:
		if zero {
			return protoreflect.ValueOfUint32(0)
		}

		return protoreflect.ValueOfUint32(r.Uint32())
	case protoreflect.Uint64Kind:
		if zero {
			return protoreflect.ValueOfUint64(0)
		}

		return protoreflect.ValueOfUint64(r.Uint64())
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(!zero && r.IntN(2) == 0)
	case protoreflect.FloatKind:
		if zero {
			return protoreflect.ValueOfFloat32(0)
		}

		return protoreflect.ValueOfFloat32(math.Float32frombits(r.Uint32()))
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(string(randomBytes(r)))
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes(randomBytes(r))
	default:
		panic(fmt.Sprintf("randomScalar: a %s field, %s, isn't supported", fd.Kind(), fd.FullName()))
	}
}

func randomBytes(r *rand.Rand) []byte {
	b := make([]byte, r.IntN(6))
	for i := range b {
		b[i] = byte(r.IntN(256))
	}

	return b
}

// unknownNumber returns a field number md doesn't have.
func unknownNumber(r *rand.Rand, md protoreflect.MessageDescriptor) protowire.Number {
	for {
		var num protowire.Number

		switch r.IntN(10) {
		case 0:
			num = protowire.MaxValidNumber
		case 1:
			num = protowire.MaxValidNumber + 7 // beyond the valid numbers, ConsumeTag doesn't mind
		default:
			num = protowire.Number(70 + r.IntN(300))
		}

		if md.Fields().ByNumber(num) == nil {
			return num
		}
	}
}

func randomUnknownFields(r *rand.Rand, md protoreflect.MessageDescriptor) protoreflect.RawFields {
	var b protoreflect.RawFields

	for n := 1 + r.IntN(3); n > 0; n-- {
		b = appendRandomValue(r, b, unknownNumber(r, md), []protowire.Type{protowire.VarintType, protowire.Fixed32Type, protowire.Fixed64Type, protowire.BytesType}[r.IntN(4)])
	}

	return b
}

// appendRandomValue appends a record of the field with a random value of the wire type.
func appendRandomValue(r *rand.Rand, b []byte, num protowire.Number, typ protowire.Type) []byte {
	b = protowire.AppendTag(b, num, typ)

	return appendRandomValueOnly(r, b, typ)
}

//nolint:exhaustive // the wire types with a value: the groups have none
func appendRandomValueOnly(r *rand.Rand, b []byte, typ protowire.Type) []byte {
	switch typ {
	case protowire.VarintType:
		return protowire.AppendVarint(b, randomVarint(r))
	case protowire.Fixed32Type:
		return protowire.AppendFixed32(b, r.Uint32())
	case protowire.Fixed64Type:
		return protowire.AppendFixed64(b, r.Uint64())
	default:
		return protowire.AppendBytes(b, randomBytes(r))
	}
}

func randomVarint(r *rand.Rand) uint64 {
	switch r.IntN(6) {
	case 0:
		return 0
	case 1:
		return 1
	case 2:
		return uint64(r.IntN(128))
	case 3:
		return uint64(r.Uint32())
	case 4:
		return r.Uint64()
	default:
		return ^uint64(r.IntN(1000)) // a negative int32, sign-extended
	}
}

// scalarWireType is the wire type of a scalar field's values.
//
//nolint:exhaustive // every scalar kind is covered by the groups
func scalarWireType(fd protoreflect.FieldDescriptor) protowire.Type {
	switch fd.Kind() {
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind, protoreflect.BoolKind, protoreflect.EnumKind:
		return protowire.VarintType
	case protoreflect.FloatKind, protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind:
		return protowire.Fixed32Type
	case protoreflect.DoubleKind, protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind:
		return protowire.Fixed64Type
	default:
		return protowire.BytesType
	}
}

// appendNonCanonicalVarint appends v as a varint with a redundant continuation byte, which protowire accepts.
func appendNonCanonicalVarint(b []byte, v uint64) []byte {
	b = protowire.AppendVarint(b, v)
	b[len(b)-1] |= 0x80

	return append(b, 0x00)
}

// deltaGen writes random deltas in the grammar applyUserCmdDelta reads: mostly valid, with some of every error.
type deltaGen struct {
	r *rand.Rand
}

// message returns a random delta of a message with nested deltas down to depth.
func (g *deltaGen) message(md protoreflect.MessageDescriptor, depth int) []byte {
	fields := md.Fields()

	var b []byte

	for ops := g.r.IntN(5); ops > 0; ops-- {
		fd := fields.Get(g.r.IntN(fields.Len()))

		switch p := g.r.IntN(1000); {
		case p < 450:
			b = g.value(b, fd, depth) // the field's own value: a scalar, a nested delta or repeated operations
		case p < 600:
			b = protowire.AppendTag(b, fd.Number(), wireTypeReset)
		case p < 720:
			b = g.wrongWireType(b, fd) // kept as an unknown field, ignored for a repeated field
		case p < 850:
			b = appendRandomValue(g.r, b, unknownNumber(g.r, md), []protowire.Type{protowire.VarintType, protowire.Fixed32Type, protowire.Fixed64Type, protowire.BytesType}[g.r.IntN(4)]) // ignored
		case p < 880:
			b = g.group(b, md) // a well-formed group: an unknown field of a known field, ignored otherwise
		case p < 900:
			b = protowire.AppendTag(b, unknownNumber(g.r, md), wireTypeReset) // error: a reset marker of an unknown field
		case p < 915:
			b = protowire.AppendTag(b, fd.Number(), protowire.Type(6)) // error: the reserved wire type
		case p < 930:
			b = protowire.AppendTag(b, fd.Number(), protowire.EndGroupType) // error: an end group without its start
		case p < 960:
			return g.truncated(b, fd) // error: the data ends inside a record
		default:
			b = g.nonCanonical(b, fd) // a value under a tag whose varint has a redundant byte
		}
	}

	return b
}

func (g *deltaGen) value(b []byte, fd protoreflect.FieldDescriptor, depth int) []byte {
	switch {
	case fd.IsList():
		b = protowire.AppendTag(b, fd.Number(), protowire.BytesType)

		return protowire.AppendBytes(b, g.repeated(fd, depth))
	case fd.Message() != nil:
		b = protowire.AppendTag(b, fd.Number(), protowire.BytesType)

		if depth == 0 {
			return protowire.AppendBytes(b, nil) // an empty delta still creates the message
		}

		return protowire.AppendBytes(b, g.message(fd.Message(), depth-1))
	default:
		return appendRandomValue(g.r, b, fd.Number(), scalarWireType(fd))
	}
}

// wrongWireType appends a record of the field with another wire type than the field's.
func (g *deltaGen) wrongWireType(b []byte, fd protoreflect.FieldDescriptor) []byte {
	want := protowire.BytesType
	if !fd.IsList() && fd.Message() == nil {
		want = scalarWireType(fd)
	}

	typ := want
	for typ == want {
		typ = []protowire.Type{protowire.VarintType, protowire.Fixed32Type, protowire.Fixed64Type, protowire.BytesType}[g.r.IntN(4)]
	}

	return appendRandomValue(g.r, b, fd.Number(), typ)
}

// group appends a well-formed group of a known or an unknown field.
func (g *deltaGen) group(b []byte, md protoreflect.MessageDescriptor) []byte {
	num := unknownNumber(g.r, md)
	if g.r.IntN(2) == 0 {
		num = md.Fields().Get(g.r.IntN(md.Fields().Len())).Number()
	}

	b = protowire.AppendTag(b, num, protowire.StartGroupType)

	for n := g.r.IntN(3); n > 0; n-- {
		b = appendRandomValue(g.r, b, protowire.Number(1+g.r.IntN(20)), []protowire.Type{protowire.VarintType, protowire.Fixed32Type, protowire.BytesType}[g.r.IntN(3)])
	}

	return protowire.AppendTag(b, num, protowire.EndGroupType)
}

// truncated appends a record whose data ends early.
func (g *deltaGen) truncated(b []byte, fd protoreflect.FieldDescriptor) []byte {
	switch g.r.IntN(5) {
	case 0:
		return append(b, 0x80) // a tag that doesn't end
	case 1:
		return append(b, 0x00) // field number zero
	case 2:
		return append(protowire.AppendTag(b, fd.Number(), protowire.VarintType), 0x80)
	case 3:
		return protowire.AppendVarint(protowire.AppendTag(b, fd.Number(), protowire.BytesType), 20)
	default:
		return append(protowire.AppendTag(b, fd.Number(), protowire.Fixed32Type), 1, 2)
	}
}

// nonCanonical appends a record of a scalar field whose tag, and length when it has one, are varints with a redundant
// byte, with the field's wire type or another one.
func (g *deltaGen) nonCanonical(b []byte, fd protoreflect.FieldDescriptor) []byte {
	typ := scalarWireType(fd)
	if fd.IsList() || fd.Message() != nil || g.r.IntN(2) == 0 {
		typ = []protowire.Type{protowire.VarintType, protowire.Fixed32Type, protowire.Fixed64Type, protowire.BytesType}[g.r.IntN(4)]
	}

	b = appendNonCanonicalVarint(b, protowire.EncodeTag(fd.Number(), typ))

	if typ != protowire.BytesType {
		return appendRandomValueOnly(g.r, b, typ)
	}

	v := randomBytes(g.r)
	b = appendNonCanonicalVarint(b, uint64(len(v)))

	return append(b, v...)
}

// repeated returns random repeated operations for a repeated message field, with element deltas down to depth.
func (g *deltaGen) repeated(fd protoreflect.FieldDescriptor, depth int) []byte {
	var b []byte

	for ops := g.r.IntN(4); ops > 0; ops-- {
		index := g.r.IntN(6)

		switch p := g.r.IntN(100); {
		case p < 4:
			index = userCmdRepeatedLimit - 1 + g.r.IntN(4) // around the limit: 255 and 256 extend the list, 257 and 258 fail
		case p < 8:
			index = 1 << 20 // out of bounds, over the limit
		}

		key := uint64(index) << 3

		switch p := g.r.IntN(100); {
		case p < 35: // the length
			b = protowire.AppendVarint(b, key|uint64(wireTypeReset))
		case p < 85: // an element's delta
			b = protowire.AppendVarint(b, key|uint64(protowire.BytesType))

			if depth == 0 {
				b = protowire.AppendBytes(b, nil)
			} else {
				b = protowire.AppendBytes(b, g.message(fd.Message(), depth-1))
			}
		case p < 91: // error: another wire type
			return protowire.AppendVarint(b, key|uint64([]protowire.Type{protowire.VarintType, protowire.Fixed64Type, protowire.StartGroupType, protowire.EndGroupType, protowire.Fixed32Type, 6}[g.r.IntN(6)]))
		case p < 95: // error: the element's delta is cut short
			return append(protowire.AppendVarint(b, key|uint64(protowire.BytesType)), 0x09)
		case p < 97: // error: the index doesn't end
			return append(b, 0x80)
		default: // the length under a varint with a redundant byte
			b = appendNonCanonicalVarint(b, key|uint64(wireTypeReset))
		}
	}

	return b
}
