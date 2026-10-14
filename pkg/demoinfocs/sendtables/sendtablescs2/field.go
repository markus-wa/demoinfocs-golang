package sendtablescs2

import (
	"fmt"
	"strconv"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

const (
	fieldModelSimple = iota
	fieldModelFixedArray
	fieldModelFixedTable
	fieldModelVariableArray
	fieldModelVariableTable
)

// polyUpdate is returned by the base decoder of a polymorphic fixed-table field.
// id is the field's polySerializerID; ser is the newly selected serializer
// (nil when the pointer is inactive / boolean was false).
type polyUpdate struct {
	id  int
	ser *serializer
}

type field struct {
	varName string
	// name is the canonical field name: the varName, prefixed with the sendNode
	// when another field in the same serializer shares the varName and only the
	// send_node distinguishes them on the wire
	// (e.g. m_vecVelocity.m_vecX vs m_vecViewOffset.m_vecX).
	name              string
	varType           string
	sendNode          string
	serializerName    string
	serializerVersion int32
	encoder           string
	encodeFlags       *int32
	bitCount          *int32
	lowValue          *float32
	highValue         *float32
	fieldType         *fieldType
	serializer        *serializer
	model             int
	polyTypes         []*serializer
	// polySerializerID is ≥ 0 for polymorphic fixed-table fields; -1 otherwise.
	// It indexes into the per-entity polySerializers slice, so the active
	// serializer is tracked per entity rather than on the shared field object.
	polySerializerID int

	decoder      fieldDecoder
	baseDecoder  fieldDecoder
	childDecoder fieldDecoder
}

func newField(serializers map[string]*serializer, ser *msg.CSVCMsg_FlattenedSerializer, f *msg.ProtoFlattenedSerializerFieldT) *field {
	resolve := func(p *int32) string {
		if p == nil {
			return ""
		}

		return ser.GetSymbols()[*p]
	}

	x := &field{
		varName:           resolve(f.VarNameSym),
		varType:           resolve(f.VarTypeSym),
		sendNode:          resolve(f.SendNodeSym),
		serializerName:    resolve(f.FieldSerializerNameSym),
		serializerVersion: f.GetFieldSerializerVersion(),
		encoder:           resolve(f.VarEncoderSym),
		encodeFlags:       f.EncodeFlags,
		bitCount:          f.BitCount,
		lowValue:          f.LowValue,
		highValue:         f.HighValue,
		model:             fieldModelSimple,
		polySerializerID:  -1,
	}

	if len(f.PolymorphicTypes) > 0 {
		// Build combined slice: [0] = default/field serializer, [1..N] = polymorphic alternatives.
		// The ubitvar read from the bitstream is a direct index into this slice, where
		// 0 selects the field's own serializer and 1..N select the polymorphic variants.
		x.polyTypes = make([]*serializer, len(f.PolymorphicTypes)+1)
		x.polyTypes[0] = serializers[x.serializerName]

		if x.polyTypes[0] == nil {
			_panicf("polymorphic field %q: unknown serializer %q (missing or forward-declared)", x.varName, x.serializerName)
		}

		for i, t := range f.PolymorphicTypes {
			name := resolve(t.PolymorphicFieldSerializerNameSym)
			x.polyTypes[i+1] = serializers[name]

			if x.polyTypes[i+1] == nil {
				_panicf("polymorphic field %q: unknown serializer %q (missing or forward-declared)", x.varName, name)
			}
		}
	}

	if x.sendNode == "(root)" {
		x.sendNode = ""
	}

	x.name = x.varName

	return x
}

func (f *field) setModel(model int) {
	f.model = model

	switch model {
	case fieldModelFixedArray:
		f.decoder = findDecoder(f)

	case fieldModelFixedTable:
		if len(f.polyTypes) == 0 {
			// Fixed pointer: single serializer, never changes type.
			// Only a boolean is read from the stream; serializer is on the field.
			f.baseDecoder = booleanDecoder
		} else {
			// Polymorphic pointer: bool then (if true) a ubitvar type index.
			// Returns a *polyUpdate so the active serializer can be stored
			// per entity rather than on this shared field object.
			polyTypes := f.polyTypes
			polyID := f.polySerializerID
			f.baseDecoder = func(r *reader) any {
				if r.readBoolean() {
					r.polyScratch = polyUpdate{id: polyID, ser: polyTypes[r.readUBitVar()]}
				} else {
					r.polyScratch = polyUpdate{id: polyID}
				}

				return &r.polyScratch
			}
		}

	case fieldModelVariableArray:
		if f.fieldType.genericType == nil {
			_panicf("no generic type for variable array field %#v", f)
		}

		f.baseDecoder = unsignedDecoder
		f.childDecoder = findDecoderByBaseType(f)

	case fieldModelVariableTable:
		f.baseDecoder = unsignedDecoder

	case fieldModelSimple:
		f.decoder = findDecoder(f)
	}
}

func (f *field) getNameForFieldPath(fp *fieldPath, pos int, ps []*serializer) []string {
	x := []string{f.name}

	switch f.model {
	case fieldModelFixedArray:
		if fp.last == pos {
			x = append(x, fmt.Sprintf("%04d", fp.path[pos]))
		}

	case fieldModelFixedTable:
		if fp.last >= pos {
			ser := f.serializer
			if f.polySerializerID >= 0 && ps != nil {
				ser = ps[f.polySerializerID]
			}

			if ser != nil {
				x = append(x, ser.getNameForFieldPath(fp, pos, ps)...)
			}
		}

	case fieldModelVariableArray:
		if fp.last == pos {
			x = append(x, fmt.Sprintf("%04d", fp.path[pos]))
		}

	case fieldModelVariableTable:
		if fp.last != pos-1 {
			x = append(x, fmt.Sprintf("%04d", fp.path[pos]))
			if fp.last != pos {
				x = append(x, f.serializer.getNameForFieldPath(fp, pos+1, ps)...)
			}
		}
	}

	return x
}

// getDecoderAndCollection returns the decoder and whether this field path is a
// variable-length collection update that requires fieldState handling.
// This encodes the (base && variableArray|variableTable) check directly,
// avoiding a separate getFieldForFieldPath traversal.
func (f *field) getDecoderAndCollection(fp *fieldPath, pos int, ps []*serializer) (fieldDecoder, bool) {
	switch f.model {
	case fieldModelFixedArray:
		return f.decoder, false

	case fieldModelFixedTable:
		if fp.last == pos-1 {
			return f.baseDecoder, false // base decoder but fixed, no fieldState update
		}

		ser := f.serializer
		if f.polySerializerID >= 0 && ps != nil {
			ser = ps[f.polySerializerID]
		}

		if ser == nil {
			return nil, false // polymorphic pointer not yet activated
		}

		return ser.getDecoderAndCollection(fp, pos, ps)

	case fieldModelVariableArray:
		if fp.last == pos {
			return f.childDecoder, false
		}

		return f.baseDecoder, true // variable collection update

	case fieldModelVariableTable:
		if fp.last >= pos+1 {
			return f.serializer.getDecoderAndCollection(fp, pos+1, ps)
		}

		return f.baseDecoder, true // variable collection update
	}

	return f.decoder, false
}

// getFieldPathForName resolves the remainder of a dotted name below this field.
// What the remainder looks like depends on the field's model:
//
//	array:          "0014"                    (m_pWeaponServices.m_iAmmo.0014 after m_iAmmo)
//	fixed table:    "m_iAmmo.0014"            (after m_pWeaponServices; nothing is consumed here, the sub-serializer resolves the remainder from its own fields)
//	variable table: "0007.m_nCount"           (after m_weaponPurchasesThisMatch.m_weaponPurchases)
//	simple:         nothing may remain, so any remainder fails
//
// It must not panic on names that don't resolve: since disambiguated field
// names contain dots themselves, the serializer-level walk tries each dot as
// a possible boundary and backtracks through here on failure — e.g. it may
// descend into a field "m_x" with the remainder "m_sub.0001.m_y2", which must
// return false so the walk can retry with the longer prefix "m_x.m_sub".
func (f *field) getFieldPathForName(fp *fieldPath, name string, ps []*serializer) bool {
	switch f.model {
	case fieldModelFixedArray, fieldModelVariableArray:
		i, ok := atoi4(name)
		if !ok {
			return false
		}

		fp.path[fp.last] = i

		return true

	case fieldModelFixedTable:
		ser := f.serializer
		if f.polySerializerID >= 0 && ps != nil {
			ser = ps[f.polySerializerID]
		}

		if ser == nil {
			return false
		}

		return ser.getFieldPathForName(fp, name, ps)

	case fieldModelVariableTable:
		if len(name) < 6 || name[4] != '.' {
			return false
		}

		i, ok := atoi4(name[:4])
		if !ok {
			return false
		}

		fp.path[fp.last] = i
		fp.last++

		return f.serializer.getFieldPathForName(fp, name[5:], ps)
	}

	return false
}

//nolint:gocognit,funlen
func (f *field) getFieldPaths(fp *fieldPath, state *fieldState, ps []*serializer) []*fieldPath {
	x := make([]*fieldPath, 0, 1)

	switch f.model {
	case fieldModelFixedArray:
		if sub, ok := stateAtField(state, fp); ok {
			fp.last++

			for i, v := range sub.state {
				if v != nil {
					fp.path[fp.last] = i
					x = append(x, fp.copy())
				}
			}

			fp.last--
		}

	case fieldModelFixedTable:
		if sub, ok := stateAtField(state, fp); ok {
			ser := f.serializer
			if f.polySerializerID >= 0 && ps != nil {
				ser = ps[f.polySerializerID]
			}

			if ser != nil {
				fp.last++
				x = append(x, ser.getFieldPaths(fp, sub, ps)...)
				fp.last--
			}
		}

	case fieldModelVariableArray:
		if sub, ok := stateAtField(state, fp); ok {
			fp.last++

			for i, v := range sub.state {
				if v != nil {
					fp.path[fp.last] = i
					x = append(x, fp.copy())
				}
			}

			fp.last--
		}

	case fieldModelVariableTable:
		if sub, ok := stateAtField(state, fp); ok {
			fp.last += 2

			for i, v := range sub.state {
				if vv, ok := v.(*fieldState); ok {
					fp.path[fp.last-1] = i
					x = append(x, f.serializer.getFieldPaths(fp, vv, ps)...)
				}
			}

			fp.last -= 2
		}

	case fieldModelSimple:
		x = append(x, fp.copy())
	}

	return x
}

// stateAtField returns the fieldState stored at the state slot of the current
// field during a getFieldPaths traversal.
//
// state is the fieldState of the serializer currently being enumerated, and the
// current field's component within it is fp.path[fp.last] (set by
// serializer.getFieldPaths before calling into this field). Once a table has
// been descended, state is the enclosing table's fieldState while fp keeps the
// full path from the serializer root, so a full-path state lookup would
// mis-index (this previously broke enumeration of tables nested under tables,
// e.g. polymorphic pointer sub-fields).
func stateAtField(state *fieldState, fp *fieldPath) (*fieldState, bool) {
	i := fp.path[fp.last]

	if i < 0 || i >= len(state.state) {
		return nil, false
	}

	sub, ok := state.state[i].(*fieldState)

	return sub, ok
}

// atoi4 parses a 4-digit array-index segment (e.g. "0007") as generated by
// getNameForFieldPath.
func atoi4(s string) (int, bool) {
	if len(s) != 4 {
		return 0, false
	}

	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}

	return n, true
}
