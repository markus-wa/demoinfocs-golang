package sendtables2

import (
	"fmt"
	"strconv"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
)

const (
	fieldModelSimple = iota
	fieldModelFixedArray
	fieldModelFixedTable
	fieldModelVariableArray
	fieldModelVariableTable
)

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
	polyTypes         map[uint32]*serializer

	decoder      fieldDecoder
	baseDecoder  fieldDecoder
	childDecoder fieldDecoder
}

func newField(serializers map[string]*serializer, ser *msgs2.CSVCMsg_FlattenedSerializer, f *msgs2.ProtoFlattenedSerializerFieldT) *field {
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
	}

	if len(f.PolymorphicTypes) > 0 {
		x.polyTypes = make(map[uint32]*serializer, len(f.PolymorphicTypes))

		for i, t := range f.PolymorphicTypes {
			x.polyTypes[uint32(i+1)] = serializers[resolve(t.PolymorphicFieldSerializerNameSym)] //nolint:gosec
		}
	}

	if x.sendNode == "(root)" {
		x.sendNode = ""
	}

	x.name = x.varName
	if x.sendNode != "" {
		x.name = x.sendNode + "." + x.varName
	}

	return x
}

func (f *field) setModel(model int) {
	f.model = model

	switch model {
	case fieldModelFixedArray:
		f.decoder = findDecoder(f)

	case fieldModelFixedTable:
		if len(f.polyTypes) > 0 {
			f.baseDecoder = func(r *reader) any {
				b := r.readBoolean()
				polyTypeIndex := r.readUBitVar()
				f.serializer = f.polyTypes[polyTypeIndex]

				return b
			}
		} else {
			f.baseDecoder = booleanDecoder
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

func (f *field) getNameForFieldPath(fp *fieldPath, pos int) []string {
	x := []string{f.name}

	switch f.model {
	case fieldModelFixedArray:
		if fp.last == pos {
			x = append(x, fmt.Sprintf("%04d", fp.path[pos]))
		}

	case fieldModelFixedTable:
		if fp.last >= pos {
			x = append(x, f.serializer.getNameForFieldPath(fp, pos)...)
		}

	case fieldModelVariableArray:
		if fp.last == pos {
			x = append(x, fmt.Sprintf("%04d", fp.path[pos]))
		}

	case fieldModelVariableTable:
		if fp.last != pos-1 {
			x = append(x, fmt.Sprintf("%04d", fp.path[pos]))
			if fp.last != pos {
				x = append(x, f.serializer.getNameForFieldPath(fp, pos+1)...)
			}
		}
	}

	return x
}

// getDecoderAndCollection returns the decoder and whether this field path is a
// variable-length collection update that requires fieldState handling.
// This encodes the (base && variableArray|variableTable) check directly,
// avoiding a separate getFieldForFieldPath traversal.
func (f *field) getDecoderAndCollection(fp *fieldPath, pos int) (fieldDecoder, bool) {
	switch f.model {
	case fieldModelFixedArray:
		return f.decoder, false

	case fieldModelFixedTable:
		if fp.last == pos-1 {
			return f.baseDecoder, false // base decoder but fixed, no fieldState update
		}

		return f.serializer.getDecoderAndCollection(fp, pos)

	case fieldModelVariableArray:
		if fp.last == pos {
			return f.childDecoder, false
		}

		return f.baseDecoder, true // variable collection update

	case fieldModelVariableTable:
		if fp.last >= pos+1 {
			return f.serializer.getDecoderAndCollection(fp, pos+1)
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
func (f *field) getFieldPathForName(fp *fieldPath, name string) bool {
	switch f.model {
	case fieldModelFixedArray, fieldModelVariableArray:
		i, ok := atoi4(name)
		if !ok {
			return false
		}
		fp.path[fp.last] = i
		return true

	case fieldModelFixedTable:
		return f.serializer.getFieldPathForName(fp, name)

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
		return f.serializer.getFieldPathForName(fp, name[5:])
	}

	return false
}

func (f *field) getFieldPaths(fp *fieldPath, state *fieldState) []*fieldPath {
	x := make([]*fieldPath, 0, 1)

	switch f.model {
	case fieldModelFixedArray:
		if sub, ok := state.get(fp).(*fieldState); ok {
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
		if sub, ok := state.get(fp).(*fieldState); ok {
			fp.last++
			x = append(x, f.serializer.getFieldPaths(fp, sub)...)
			fp.last--
		}

	case fieldModelVariableArray:
		if sub, ok := state.get(fp).(*fieldState); ok {
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
		if sub, ok := state.get(fp).(*fieldState); ok {
			fp.last += 2
			for i, v := range sub.state {
				if vv, ok := v.(*fieldState); ok {
					fp.path[fp.last-1] = i
					x = append(x, f.serializer.getFieldPaths(fp, vv)...)
				}
			}
			fp.last -= 2
		}

	case fieldModelSimple:
		x = append(x, fp.copy())
	}

	return x
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
