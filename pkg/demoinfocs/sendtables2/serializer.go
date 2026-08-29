package sendtables2

import (
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

func (s *serializer) getNameForFieldPath(fp *fieldPath, pos int) []string {
	return s.fields[fp.path[pos]].getNameForFieldPath(fp, pos+1)
}

// getDecoderAndCollection is a single-pass alternative to calling
// getFieldForFieldPath + getDecoderForFieldPath2 separately.
// Returns the decoder and whether this update requires fieldState handling.
func (s *serializer) getDecoderAndCollection(fp *fieldPath, pos int) (fieldDecoder, bool) {
	return s.fields[fp.path[pos]].getDecoderAndCollection(fp, pos+1)
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
