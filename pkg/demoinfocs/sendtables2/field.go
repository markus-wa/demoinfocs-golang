package sendtables2

import (
	"fmt"
	"strconv"

	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msgs2"
)

const (
	fieldModelSimple = iota
	fieldModelFixedArray
	fieldModelFixedTable
	fieldModelVariableArray
	fieldModelVariableTable
)

type field struct {
	parentName        string
	varName           string
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
	value             interface{}
	model             int
	polyTypes         map[uint32]*serializer

	decoder      fieldDecoder
	baseDecoder  fieldDecoder
	childDecoder fieldDecoder
}

func (f *field) modelString() string {
	switch f.model {
	case fieldModelFixedArray:
		return "fixed-array"
	case fieldModelFixedTable:
		return "fixed-table"
	case fieldModelVariableArray:
		return "variable-array"
	case fieldModelVariableTable:
		return "variable-table"
	case fieldModelSimple:
		return "simple"
	default:
		return "other"
	}
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
			x.polyTypes[uint32(i+1)] = serializers[resolve(t.PolymorphicFieldSerializerNameSym)]
		}
	}

	if x.sendNode == "(root)" {
		x.sendNode = ""
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
			f.baseDecoder = func(r *reader) interface{} {
				b := r.readBoolean()
				f.serializer = f.polyTypes[r.readUBitVar()]

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

func (f *field) getName() string {
	return f.varName
}

func (f *field) getFieldForFieldPath(fp *fieldPath, pos int) *field {
	switch f.model {
	case fieldModelFixedArray:
		return f

	case fieldModelFixedTable:
		if fp.last != pos-1 {
			return f.serializer.getFieldForFieldPath(fp, pos)
		}

	case fieldModelVariableArray:
		return f

	case fieldModelVariableTable:
		if fp.last >= pos+1 {
			return f.serializer.getFieldForFieldPath(fp, pos+1)
		}
	}

	return f
}

func (f *field) getNameForFieldPath(fp *fieldPath, pos int) []string {
	x := []string{f.varName}

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

func (f *field) getTypeForFieldPath(fp *fieldPath, pos int) *fieldType {
	switch f.model {
	case fieldModelFixedArray:
		return f.fieldType

	case fieldModelFixedTable:
		if fp.last != pos-1 {
			return f.serializer.getTypeForFieldPath(fp, pos)
		}

	case fieldModelVariableArray:
		if fp.last == pos {
			return f.fieldType.genericType
		}

	case fieldModelVariableTable:
		if fp.last >= pos+1 {
			return f.serializer.getTypeForFieldPath(fp, pos+1)
		}
	}

	return f.fieldType
}

func (f *field) getDecoderForFieldPath(fp *fieldPath, pos int) fieldDecoder {
	switch f.model {
	case fieldModelFixedArray:
		return f.decoder

	case fieldModelFixedTable:
		if fp.last == pos-1 {
			return f.baseDecoder
		}
		return f.serializer.getDecoderForFieldPath(fp, pos)

	case fieldModelVariableArray:
		if fp.last == pos {
			return f.childDecoder
		}
		return f.baseDecoder

	case fieldModelVariableTable:
		if fp.last >= pos+1 {
			return f.serializer.getDecoderForFieldPath(fp, pos+1)
		}
		return f.baseDecoder
	}

	return f.decoder
}

func (f *field) getFieldPathForName(fp *fieldPath, name string) bool {
	switch f.model {
	case fieldModelFixedArray:
		assertLen(name, 4)
		fp.path[fp.last] = mustAtoi(name)
		return true

	case fieldModelFixedTable:
		return f.serializer.getFieldPathForName(fp, name)

	case fieldModelVariableArray:
		assertLen(name, 4)
		fp.path[fp.last] = mustAtoi(name)
		return true

	case fieldModelVariableTable:
		assertLenMin(name, 6)
		fp.path[fp.last] = mustAtoi(name[:4])
		fp.last++
		return f.serializer.getFieldPathForName(fp, name[5:])

	case fieldModelSimple:
		_panicf("not supported")
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

func mustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		_panicf("assertion failed: '%s' not a number", s)
	}
	return n
}

func assertLen(s string, n int) {
	if len(s) != n {
		_panicf("assertion failed: '%s' is not %d long", s, n)
	}
}

func assertLenMin(s string, n int) {
	if len(s) < n {
		_panicf("assertion failed: '%s' is less than %d long", s, n)
	}
}
