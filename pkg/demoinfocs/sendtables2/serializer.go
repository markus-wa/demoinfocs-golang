package sendtables2

import (
	"fmt"
	"strings"
)

type fieldIndex struct {
	index int
	field *field
}

type serializer struct {
	name            string
	version         int32
	fields          []*field
	fieldIndexes    map[string]*fieldIndex
	fieldNameChecks map[string]bool
}

func newSerializer(name string, version int32) *serializer {
	return &serializer{
		name:            name,
		version:         version,
		fields:          []*field{},
		fieldIndexes:    make(map[string]*fieldIndex),
		fieldNameChecks: make(map[string]bool),
	}
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

	dec, _ := s.fields[index].getDecoderForFieldPath(fp, pos+1)

	return dec
}

func (s *serializer) getDecoderForFieldPath2(fp *fieldPath, pos int) (fieldDecoder, bool) {
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

	// Fields disambiguated with their sendNode contain dots in their own name
	// (e.g. m_AttributeList.m_Attributes), so the prefix before any dot may be
	// the field name. Try each dot position and backtrack on failure.
	for dotIndex := strings.Index(name, "."); dotIndex != -1; {
		nameBeforeDot := name[:dotIndex]
		if fi := s.fieldIndexes[nameBeforeDot]; fi != nil {
			last := fp.last
			fp.path[fp.last] = fi.index
			fp.last++

			if fi.field.getFieldPathForName(fp, name[dotIndex+1:]) {
				return true
			}

			fp.last = last
		}

		next := strings.Index(name[dotIndex+1:], ".")
		if next == -1 {
			break
		}
		dotIndex += 1 + next
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
	newFieldIndex := len(s.fields)
	s.fields = append(s.fields, f)

	s.fieldIndexes[f.name] = &fieldIndex{
		index: newFieldIndex,
		field: f,
	}
}

func (s *serializer) checkFieldName(name string) bool {
	ok, exists := s.fieldNameChecks[name]
	if !exists {
		ok = s.getFieldPathForName(newFieldPath(), name)
		s.fieldNameChecks[name] = ok
	}

	return ok
}
