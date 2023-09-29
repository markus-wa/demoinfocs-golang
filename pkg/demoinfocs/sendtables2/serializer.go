package sendtables2

import (
	"fmt"
	"strings"
)

type FieldIndex struct {
	index int
	field *field
}

type serializer struct {
	name         string
	version      int32
	fields       []*field
	fieldIndexes map[string]*FieldIndex
}

func (s *serializer) id() string {
	return serializerId(s.name, s.version)
}

func (s *serializer) getNameForFieldPath(fp *fieldPath, pos int) []string {
	return s.fields[fp.path[pos]].getNameForFieldPath(fp, pos+1)
}

func (s *serializer) getTypeForFieldPath(fp *fieldPath, pos int) *fieldType {
	return s.fields[fp.path[pos]].getTypeForFieldPath(fp, pos+1)
}

func (s *serializer) getDecoderForFieldPath(fp *fieldPath, pos int) fieldDecoder {
	index := fp.path[pos]
	if len(s.fields) <= index {
		_panicf("serializer %s: field path %s has no field (%d)", s.name, fp, index)
	}
	return s.fields[index].getDecoderForFieldPath(fp, pos+1)
}

func (s *serializer) getFieldForFieldPath(fp *fieldPath, pos int) *field {
	return s.fields[fp.path[pos]].getFieldForFieldPath(fp, pos+1)
}

func (s *serializer) getFieldPathForName(fp *fieldPath, name string) bool {
	if s.fieldIndexes[name] != nil {
		fp.path[fp.last] = s.fieldIndexes[name].index
		return true
	}

	dotIndex := strings.Index(name, ".")
	if dotIndex != -1 {
		nameBeforeDot := name[:dotIndex]
		if s.fieldIndexes[nameBeforeDot] != nil {
			fp.path[fp.last] = s.fieldIndexes[nameBeforeDot].index
			fp.last++
			f := s.fieldIndexes[nameBeforeDot].field
			return f.getFieldPathForName(fp, name[len(f.varName)+1:])
		}
	}

	return false
}

func (s *serializer) getFieldPaths(fp *fieldPath, state *fieldState) []*fieldPath {
	results := make([]*fieldPath, 0, 4)
	for i, f := range s.fields {
		fp.path[fp.last] = i
		results = append(results, f.getFieldPaths(fp, state)...)
	}
	return results
}

func serializerId(name string, version int32) string {
	return fmt.Sprintf("%s(%d)", name, version)
}

func (s *serializer) addField(f *field) {
	s.fields = append(s.fields, f)
	if s.fieldIndexes == nil {
		s.fieldIndexes = make(map[string]*FieldIndex)
	}
	s.fieldIndexes[f.varName] = &FieldIndex{
		index: len(s.fields) - 1,
		field: f,
	}
}
