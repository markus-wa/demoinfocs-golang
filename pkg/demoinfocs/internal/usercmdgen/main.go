// Command usercmdgen generates s2_usercmd_typed.go of package demoinfocs: a typed deep copy and a typed delta merge
// for msg.CSGOUserCmdPB and every message it contains, so that user commands are reconstructed without protobuf
// reflection (see applyUserCmdDelta in s2_usercmd_delta.go).
//
// The generated functions behave like proto.Clone and like the reflective merge they replaced, which is kept as the
// reference of the differential tests (mergeUserCmdDeltaReflect in s2_usercmd_delta_reflect_test.go).
//
// The messages and their fields are read from the Go types of package msg (the protobuf struct tags) and from their
// protobuf descriptors (the full names used in error messages), so the file is regenerated when the protos change:
//
//	go generate ./pkg/demoinfocs
//
// TestGeneratedCodeIsCurrent fails while the generated file is stale.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"

	msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func main() {
	out := flag.String("out", "s2_usercmd_typed.go", "the file to write")

	flag.Parse()

	src, err := generate()
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(*out, src, 0o600); err != nil {
		log.Fatal(err)
	}
}

// generate returns the gofmt'd source of s2_usercmd_typed.go.
func generate() ([]byte, error) {
	messages, err := collectMessages(reflect.TypeOf(msg.CSGOUserCmdPB{}))
	if err != nil {
		return nil, err
	}

	g := &generator{}
	g.writeHeader(messages)

	for _, m := range messages {
		g.writeCopy(m)
		g.writeMerge(m)

		for _, f := range m.fields {
			if f.kind == kindMessageList {
				g.writeRepeated(m, f)
			}
		}
	}

	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting the generated code: %w\n%s", err, g.buf.Bytes())
	}

	return src, nil
}

type fieldKind int

const (
	kindScalar      fieldKind = iota // a pointer to a number, bool, enum or string
	kindBytes                        // []byte
	kindMessage                      // a pointer to a message
	kindMessageList                  // a slice of pointers to messages
)

type field struct {
	num       protowire.Number
	goName    string
	protoName string         // the field's name in the proto, for comments
	fullName  string         // the protobuf full name, for error messages
	wire      protowire.Type // the wire type of the field's values
	kind      fieldKind
	goType    reflect.Type // the Go type of the struct field
	elem      reflect.Type // kindMessage and kindMessageList: the message struct type
	bit       int          // kindScalar and kindBytes: the field's bit in the merge's set mask
}

type message struct {
	goType reflect.Type // the message struct type, e.g. msg.CBaseUserCmdPB
	fields []field      // by field number
	set    int          // the number of fields with a bit in the merge's set mask
}

func (m *message) name() string {
	return m.goType.Name()
}

// valuesName is the struct type that holds a copy's scalar values, see writeCopy.
func (m *message) valuesName() string {
	n := m.name()

	return strings.ToLower(n[:1]) + n[1:] + "Values"
}

var msgPkgPath = reflect.TypeOf(msg.CSGOUserCmdPB{}).PkgPath()

// collectMessages describes root and every message type reachable from its fields, root first.
func collectMessages(root reflect.Type) ([]*message, error) {
	var out []*message

	seen := map[reflect.Type]bool{root: true}
	queue := []reflect.Type{root}

	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]

		m, err := describeMessage(t)
		if err != nil {
			return nil, err
		}

		out = append(out, m)

		for _, f := range m.fields {
			if f.elem != nil && !seen[f.elem] {
				seen[f.elem] = true
				queue = append(queue, f.elem)
			}
		}
	}

	return out, nil
}

func describeMessage(t reflect.Type) (*message, error) {
	if t.PkgPath() != msgPkgPath {
		return nil, fmt.Errorf("%s is from package %s, only package msg is supported", t, t.PkgPath())
	}

	pm, ok := reflect.New(t).Interface().(protoreflect.ProtoMessage)
	if !ok {
		return nil, fmt.Errorf("%s is not a protobuf message", t)
	}

	desc := pm.ProtoReflect().Descriptor()

	// proto.Unmarshal validates the UTF-8 of a proto3 string, which the merge doesn't
	if desc.Syntax() != protoreflect.Proto2 {
		return nil, fmt.Errorf("%s is %s, only proto2 messages are supported", t.Name(), desc.Syntax())
	}

	m := &message{goType: t}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Tag.Get("protobuf_oneof") != "" {
			return nil, fmt.Errorf("%s.%s: oneof fields aren't supported", t.Name(), sf.Name)
		}

		tag, ok := sf.Tag.Lookup("protobuf")
		if !ok {
			continue // state, unknownFields, sizeCache
		}

		f, err := describeField(sf, tag, desc)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", t.Name(), sf.Name, err)
		}

		m.fields = append(m.fields, f)
	}

	sort.Slice(m.fields, func(i, j int) bool { return m.fields[i].num < m.fields[j].num })

	for i := range m.fields {
		f := &m.fields[i]
		if f.kind == kindScalar || f.kind == kindBytes {
			f.bit = m.set
			m.set++
		}
	}

	if m.set > 64 {
		return nil, fmt.Errorf("%s: %d scalar fields don't fit the set mask", t.Name(), m.set)
	}

	return m, nil
}

// describeField reads a struct field's protobuf tag, "<encoding>,<number>,<opt|rep|req>,...".
func describeField(sf reflect.StructField, tag string, desc protoreflect.MessageDescriptor) (field, error) {
	parts := strings.Split(tag, ",")
	if len(parts) < 3 {
		return field{}, fmt.Errorf("unexpected protobuf tag %q", tag)
	}

	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return field{}, fmt.Errorf("unexpected field number in protobuf tag %q: %w", tag, err)
	}

	// proto.Unmarshal fails a message whose required field is unset, which the merge doesn't
	if parts[2] == "req" {
		return field{}, fmt.Errorf("required fields aren't supported")
	}

	fd := desc.Fields().ByNumber(protowire.Number(num))
	if fd == nil {
		return field{}, fmt.Errorf("field %d isn't in the descriptor of %s", num, desc.FullName())
	}

	wire, err := wireType(parts[0])
	if err != nil {
		return field{}, err
	}

	f := field{
		num:       protowire.Number(num),
		goName:    sf.Name,
		protoName: string(fd.Name()),
		fullName:  string(fd.FullName()),
		wire:      wire,
		goType:    sf.Type,
	}

	if err := classify(&f, parts[2] == "rep"); err != nil {
		return field{}, err
	}

	if f.elem != nil && f.elem.PkgPath() != msgPkgPath {
		return field{}, fmt.Errorf("%s is from package %s, only package msg is supported", f.elem, f.elem.PkgPath())
	}

	return f, nil
}

func wireType(encoding string) (protowire.Type, error) {
	switch encoding {
	case "varint":
		return protowire.VarintType, nil
	case "fixed32":
		return protowire.Fixed32Type, nil
	case "fixed64":
		return protowire.Fixed64Type, nil
	case "bytes":
		return protowire.BytesType, nil
	default:
		return 0, fmt.Errorf("the %s encoding isn't supported", encoding)
	}
}

// classify sets the field's kind (and elem) from its Go type.
func classify(f *field, repeated bool) error {
	t := f.goType

	switch {
	case t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 && !repeated:
		if f.wire != protowire.BytesType {
			return fmt.Errorf("a %s field can't be a %s", t, wireName(f.wire))
		}

		f.kind = kindBytes
	case t.Kind() == reflect.Slice:
		if !repeated || t.Elem().Kind() != reflect.Pointer || t.Elem().Elem().Kind() != reflect.Struct || f.wire != protowire.BytesType {
			return fmt.Errorf("only repeated message fields are supported, not %s", t)
		}

		f.kind = kindMessageList
		f.elem = t.Elem().Elem()
	case repeated:
		return fmt.Errorf("only repeated message fields are supported, not %s", t)
	case t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct:
		if f.wire != protowire.BytesType {
			return fmt.Errorf("a %s field can't be a %s", t, wireName(f.wire))
		}

		f.kind = kindMessage
		f.elem = t.Elem()
	case t.Kind() == reflect.Pointer:
		if _, err := scalarExpr(*f, "v"); err != nil {
			return err
		}

		f.kind = kindScalar
	default:
		return fmt.Errorf("%s fields aren't supported", t)
	}

	return nil
}

// scalarExpr converts v, a field value as decoded from the wire (uint64 for varints and fixed64, uint32 for fixed32,
// []byte for length-delimited values), to the type the field points to.
func scalarExpr(f field, v string) (string, error) {
	e := f.goType.Elem()

	name, err := typeName(e)
	if err != nil {
		return "", err
	}

	kind := e.Kind()

	switch {
	case f.wire == protowire.VarintType && (kind == reflect.Int32 || kind == reflect.Int64 || kind == reflect.Uint32):
		return name + "(" + v + ")", nil
	case f.wire == protowire.VarintType && kind == reflect.Uint64:
		return convert(name, "uint64", v), nil
	case f.wire == protowire.VarintType && kind == reflect.Bool:
		return v + " != 0", nil
	case f.wire == protowire.Fixed32Type && kind == reflect.Float32:
		return "math.Float32frombits(" + v + ")", nil
	case f.wire == protowire.Fixed32Type && kind == reflect.Uint32:
		return convert(name, "uint32", v), nil
	case f.wire == protowire.Fixed32Type && kind == reflect.Int32:
		return name + "(" + v + ")", nil
	case f.wire == protowire.Fixed64Type && kind == reflect.Float64:
		return "math.Float64frombits(" + v + ")", nil
	case f.wire == protowire.Fixed64Type && kind == reflect.Uint64:
		return convert(name, "uint64", v), nil
	case f.wire == protowire.Fixed64Type && kind == reflect.Int64:
		return name + "(" + v + ")", nil
	case f.wire == protowire.BytesType && kind == reflect.String:
		return name + "(" + v + ")", nil
	default:
		return "", fmt.Errorf("a %s field can't be a %s", f.goType, wireName(f.wire))
	}
}

// convert returns v converted to name, unless v already has that type.
func convert(name, vType, v string) string {
	if name == vType {
		return v
	}

	return name + "(" + v + ")"
}

func typeName(t reflect.Type) (string, error) {
	switch t.PkgPath() {
	case "":
		return t.Name(), nil
	case msgPkgPath:
		return "msg." + t.Name(), nil
	default:
		return "", fmt.Errorf("%s is from package %s, only package msg and the builtin types are supported", t, t.PkgPath())
	}
}

func mustTypeName(t reflect.Type) string {
	name, err := typeName(t)
	if err != nil {
		panic(err) // classify has checked every type
	}

	return name
}

var wireNames = map[protowire.Type]string{
	protowire.VarintType:     "varint",
	protowire.Fixed32Type:    "fixed32",
	protowire.Fixed64Type:    "fixed64",
	protowire.BytesType:      "length-delimited value",
	protowire.StartGroupType: "group",
	protowire.EndGroupType:   "group",
}

func wireName(typ protowire.Type) string {
	if name, ok := wireNames[typ]; ok {
		return name
	}

	return fmt.Sprintf("wire type %d", typ)
}

type generator struct {
	buf bytes.Buffer
}

// pf writes one line. format is a Printf format, so % is written as %%.
func (g *generator) pf(format string, args ...any) {
	fmt.Fprintf(&g.buf, format, args...)
	g.buf.WriteByte('\n')
}

func (g *generator) writeHeader(messages []*message) {
	var needErrors, needMath bool

	for _, m := range messages {
		for _, f := range m.fields {
			if f.kind == kindMessageList {
				needErrors = true
			}

			if f.kind == kindScalar && (f.goType.Elem().Kind() == reflect.Float32 || f.goType.Elem().Kind() == reflect.Float64) {
				needMath = true
			}
		}
	}

	g.pf("// Code generated by go run ./internal/usercmdgen; DO NOT EDIT.")
	g.pf("")
	g.pf("package demoinfocs")
	g.pf("")
	g.pf("import (")

	if needErrors {
		g.pf(`"errors"`)
	}

	g.pf(`"fmt"`)

	if needMath {
		g.pf(`"math"`)
	}

	g.pf("")
	g.pf(`"google.golang.org/protobuf/encoding/protowire"`)
	g.pf("")
	g.pf(`msg "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"`)
	g.pf(")")
}

// writeCopy writes copyM, the deep copy of a message. The scalars of one message are copied into one struct that the
// copy's fields point into, which is one allocation for all of them instead of one per field.
func (g *generator) writeCopy(m *message) {
	n := m.name()

	// m.set counts the bytes fields too, which the copy doesn't keep in the values struct
	scalars := 0

	for _, f := range m.fields {
		if f.kind == kindScalar {
			scalars++
		}
	}

	if scalars > 0 {
		g.pf("")
		g.pf("// %s holds the scalar values of a copy of %s, see copy%s.", m.valuesName(), n, n)
		g.pf("type %s struct {", m.valuesName())

		for _, f := range m.fields {
			if f.kind == kindScalar {
				g.pf("%s %s", f.goName, mustTypeName(f.goType.Elem()))
			}
		}

		g.pf("}")
	}

	g.pf("")
	g.pf("// copy%s returns a deep copy of src, like proto.Clone.", n)
	g.pf("func copy%s(src *msg.%s) *msg.%s {", n, n, n)
	g.pf("if src == nil {")
	g.pf("return nil")
	g.pf("}")
	g.pf("")
	g.pf("dst := &msg.%s{}", n)

	if scalars > 0 {
		g.pf("vals := &%s{}", m.valuesName())
	}

	for _, f := range m.fields {
		g.pf("")
		g.writeCopyField(f)
	}

	g.pf("")
	g.pf("userCmdCopyUnknown(dst, src)")
	g.pf("")
	g.pf("return dst")
	g.pf("}")
}

const copyListTemplate = `if len(src.%[1]s) > 0 {
	dst.%[1]s = make([]*msg.%[2]s, len(src.%[1]s))

	for i, e := range src.%[1]s {
		if e == nil {
			dst.%[1]s[i] = &msg.%[2]s{} // proto.Clone copies a nil element as an empty message
		} else {
			dst.%[1]s[i] = copy%[2]s(e)
		}
	}
}`

func (g *generator) writeCopyField(f field) {
	switch f.kind {
	case kindScalar:
		g.pf("if src.%s != nil {", f.goName)
		g.pf("vals.%s = *src.%s", f.goName, f.goName)
		g.pf("dst.%s = &vals.%s", f.goName, f.goName)
		g.pf("}")
	case kindBytes:
		g.pf("if src.%s != nil {", f.goName)
		g.pf("dst.%s = append([]byte{}, src.%s...)", f.goName, f.goName)
		g.pf("}")
	case kindMessage:
		g.pf("dst.%s = copy%s(src.%s)", f.goName, f.elem.Name(), f.goName)
	case kindMessageList:
		g.pf(copyListTemplate, f.goName, f.elem.Name())
	}
}

// wireBranch is one case of the merge's switch on the wire type, for the wire types with a plain value.
type wireBranch struct {
	typ     protowire.Type
	name    string // the protowire constant
	consume string // the protowire function decoding a value
}

var wireBranches = []wireBranch{
	{protowire.VarintType, "protowire.VarintType", "protowire.ConsumeVarint"},
	{protowire.Fixed32Type, "protowire.Fixed32Type", "protowire.ConsumeFixed32"},
	{protowire.Fixed64Type, "protowire.Fixed64Type", "protowire.ConsumeFixed64"},
}

const mergeHeadTemplate = `
// merge%[1]sDelta applies a user command delta to m, see applyUserCmdDelta.
func merge%[1]sDelta(m *msg.%[1]s, data []byte) error {`

const mergeSetTemplate = `	// set has a bit for every scalar field this delta has set so far: a reset marker that follows the field's value
	// in the same delta doesn't undo it (the values of a delta are applied after its reset markers).
	var set uint64
`

const mergeLoopTemplate = `	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("failed to read delta field tag: %%w", protowire.ParseError(n))
		}

		data = data[n:]

		switch typ {`

// The start group, end group and reserved wire types: ConsumeFieldValue consumes a well-formed group and rejects the
// others, like the reflective merge did.
const mergeDefaultTemplate = `		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("failed to read delta field value: %%w", protowire.ParseError(n))
			}
`

const mergeTailTemplate = `
			data = data[n:]
		}
	}

	return nil
}`

// writeMerge writes mergeMDelta, the typed delta merge of a message, see applyUserCmdDelta for what it does.
func (g *generator) writeMerge(m *message) {
	g.pf(mergeHeadTemplate, m.name())

	if m.set > 0 {
		g.pf(mergeSetTemplate)
	}

	g.pf(mergeLoopTemplate)

	for _, w := range wireBranches {
		g.writeValueBranch(m, w)
	}

	g.writeBytesBranch(m)
	g.writeResetBranch(m)
	g.pf(mergeDefaultTemplate)

	if nums := fieldNumbers(m, func(f field) bool { return f.kind != kindMessageList }); nums != "" {
		g.pf("switch num {")
		g.pf("case %s: // known fields, which never have this wire type", nums)
		g.pf("userCmdAppendUnknown(m, num, typ, data[:n])")
		g.pf("}")
	}

	g.pf(mergeTailTemplate)
}

// writeValueBranch writes the merge's case for a wire type with a plain value (varint, fixed32 or fixed64): it sets
// the scalar fields of that wire type and stores the values of every other known field as unknown fields, as
// proto.Unmarshal does with a mismatched wire type.
func (g *generator) writeValueBranch(m *message, w wireBranch) {
	isSet := func(f field) bool { return f.kind == kindScalar && f.wire == w.typ }
	sets := fieldNumbers(m, isSet)
	others := fieldNumbers(m, func(f field) bool { return f.kind != kindMessageList && !isSet(f) })

	g.pf("case %s:", w.name)

	if sets != "" {
		g.pf("v, n := %s(data)", w.consume)
	} else {
		g.pf("_, n := %s(data)", w.consume)
	}

	g.pf("if n < 0 {")
	g.pf(`return fmt.Errorf("failed to read delta field value: %%w", protowire.ParseError(n))`)
	g.pf("}")

	if sets != "" || others != "" {
		g.pf("")
		g.pf("switch num {")

		for _, f := range m.fields {
			if isSet(f) {
				g.writeSetScalar(f)
			}
		}

		if others != "" {
			g.pf("case %s: // known fields of another wire type", others)
			g.pf("userCmdAppendUnknown(m, num, typ, data[:n])")
		}

		g.pf("}")
	}

	g.pf("")
	g.pf("data = data[n:]")
}

// writeSetScalar writes the case setting a scalar field from v, the decoded value.
func (g *generator) writeSetScalar(f field) {
	expr, err := scalarExpr(f, "v")
	if err != nil {
		panic(err) // classify has checked every scalar
	}

	g.pf("case %d: // %s", f.num, f.protoName)
	g.pf("if m.%s == nil {", f.goName)
	g.pf("m.%s = new(%s)", f.goName, mustTypeName(f.goType.Elem()))
	g.pf("}")
	g.pf("")
	g.pf("*m.%s = %s", f.goName, expr)
	g.pf("set |= 1 << %d", f.bit)
}

// writeBytesBranch writes the merge's case for length-delimited values: nested messages are merged recursively,
// repeated fields take the indexed repeated operations, strings and bytes are set, and the values of the other known
// fields become unknown fields, as proto.Unmarshal does with a mismatched wire type.
func (g *generator) writeBytesBranch(m *message) {
	uses := func(f field) bool { return f.kind != kindScalar || f.wire == protowire.BytesType }
	others := fieldNumbers(m, func(f field) bool { return !uses(f) })

	g.pf("case protowire.BytesType:")

	if fieldNumbers(m, uses) != "" {
		g.pf("v, n := protowire.ConsumeBytes(data)")
	} else {
		g.pf("_, n := protowire.ConsumeBytes(data)")
	}

	g.pf("if n < 0 {")
	g.pf(`return fmt.Errorf("failed to read length-delimited delta field: %%w", protowire.ParseError(n))`)
	g.pf("}")

	if len(m.fields) > 0 {
		g.pf("")
		g.pf("switch num {")

		for _, f := range m.fields {
			g.writeBytesCase(m, f)
		}

		if others != "" {
			g.pf("case %s: // known fields of another wire type", others)
			g.pf("userCmdAppendUnknown(m, num, typ, data[:n])")
		}

		g.pf("}")
	}

	g.pf("")
	g.pf("data = data[n:]")
}

// writeBytesCase writes the case of one field taking a length-delimited value, nothing for a field that doesn't.
func (g *generator) writeBytesCase(m *message, f field) {
	switch {
	case f.kind == kindMessage:
		g.pf("case %d: // %s", f.num, f.protoName)
		g.pf("if m.%s == nil {", f.goName)
		g.pf("m.%s = &msg.%s{}", f.goName, f.elem.Name())
		g.pf("}")
		g.pf("")
		g.pf("if err := merge%sDelta(m.%s, v); err != nil {", f.elem.Name(), f.goName)
		g.pf("return err")
		g.pf("}")
	case f.kind == kindMessageList:
		g.pf("case %d: // %s", f.num, f.protoName)
		g.pf("if err := merge%s%s(m, v); err != nil {", m.name(), f.goName)
		g.pf("return err")
		g.pf("}")
	case f.kind == kindBytes:
		g.pf("case %d: // %s", f.num, f.protoName)
		g.pf("m.%s = append([]byte{}, v...)", f.goName)
		g.pf("set |= 1 << %d", f.bit)
	case f.kind == kindScalar && f.wire == protowire.BytesType:
		g.writeSetScalar(f)
	}
}

// writeResetBranch writes the merge's case for the wire-type-7 reset markers of Valve's delta encoder: a known field
// goes back to its default (unset), an unknown one is an error.
func (g *generator) writeResetBranch(m *message) {
	g.pf("case wireTypeReset:")
	g.pf("switch num {")

	for _, f := range m.fields {
		g.pf("case %d: // %s", f.num, f.protoName)

		if f.kind == kindScalar || f.kind == kindBytes {
			g.pf("if set&(1<<%d) == 0 {", f.bit)
			g.pf("m.%s = nil", f.goName)
			g.pf("}")
		} else {
			g.pf("m.%s = nil", f.goName)
		}
	}

	g.pf("default:")
	g.pf(`return fmt.Errorf("wire type 7 references unknown field %%d", num)`)
	g.pf("}")
}

// repeatedTemplate is mergeMField, which applies the indexed repeated operations of Valve's delta encoder to a
// repeated message field: wire type 7 with an index sets the length (truncating, or appending empty messages), a
// length-delimited value with an index is a delta of that element.
// %[1]s is the message, %[2]s the field, %[3]s the element type and %[4]q the field's protobuf full name.
const repeatedTemplate = `
// merge%[1]s%[2]s applies repeated delta operations to m.%[2]s, see applyUserCmdDelta.
func merge%[1]s%[2]s(m *msg.%[1]s, data []byte) error {
	for len(data) > 0 {
		// Repeated operations are not protobuf tags: the high bits are a zero-based index, so index zero
		// legitimately has protobuf field number zero. Read the raw varint instead of ConsumeTag.
		key, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return fmt.Errorf("failed to read repeated delta index: %%w", protowire.ParseError(n))
		}

		if key>>3 > uint64(^uint(0)>>1) {
			return errors.New("repeated delta index does not fit in int")
		}

		index := int(key >> 3)
		typ := protowire.Type(key & 0x07)
		data = data[n:]

		switch typ {
		case wireTypeReset:
			if index > userCmdRepeatedLimit {
				return fmt.Errorf("repeated delta index %%d exceeds limit %%d", index, userCmdRepeatedLimit)
			}

			if index > len(m.%[2]s) {
				for len(m.%[2]s) < index {
					m.%[2]s = append(m.%[2]s, &msg.%[3]s{})
				}
			} else {
				m.%[2]s = m.%[2]s[:index]
			}
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("failed to read repeated delta message: %%w", protowire.ParseError(n))
			}

			if index >= len(m.%[2]s) {
				return fmt.Errorf("repeated delta index %%d is out of bounds for length %%d", index, len(m.%[2]s))
			}

			if m.%[2]s[index] == nil {
				m.%[2]s[index] = &msg.%[3]s{}
			}

			if err := merge%[3]sDelta(m.%[2]s[index], v); err != nil {
				return fmt.Errorf("failed to merge repeated field %%s[%%d]: %%w", %[4]q, index, err)
			}

			data = data[n:]
		default:
			return fmt.Errorf("unsupported repeated delta wire type %%d", typ)
		}
	}

	return nil
}`

func (g *generator) writeRepeated(m *message, f field) {
	g.pf(repeatedTemplate, m.name(), f.goName, f.elem.Name(), f.fullName)
}

// fieldNumbers returns the numbers of the fields matching keep as a case list, "" when none does.
func fieldNumbers(m *message, keep func(field) bool) string {
	var nums []string

	for _, f := range m.fields {
		if keep(f) {
			nums = append(nums, strconv.Itoa(int(f.num)))
		}
	}

	return strings.Join(nums, ", ")
}
