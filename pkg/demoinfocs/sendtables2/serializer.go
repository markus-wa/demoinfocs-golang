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

func (s *serializer) getNameForFieldPath(fp *fieldPath, pos int, ps []*serializer) []string {
	return s.fields[fp.path[pos]].getNameForFieldPath(fp, pos+1, ps)
}

// getDecoderAndCollection is a single-pass alternative to calling
// getFieldForFieldPath + getDecoderForFieldPath2 separately.
// Returns the decoder and whether this update requires fieldState handling.
func (s *serializer) getDecoderAndCollection(fp *fieldPath, pos int, ps []*serializer) (fieldDecoder, bool) {
	return s.fields[fp.path[pos]].getDecoderAndCollection(fp, pos+1, ps)
}

func (s *serializer) getFieldPathForName(fp *fieldPath, name string, ps []*serializer) bool {
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

			if fi.field.getFieldPathForName(fp, name[dotIndex+1:], ps) {
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

func (s *serializer) getFieldPaths(fp *fieldPath, state *fieldState, ps []*serializer) []*fieldPath {
	results := make([]*fieldPath, 0, 4)

	for i, f := range s.fields {
		fp.path[fp.last] = i
		results = append(results, f.getFieldPaths(fp, state, ps)...)
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
		ok = s.getFieldPathForName(newFieldPath(), name, nil)
		s.fieldNameChecks[name] = ok
	}

	return ok
}

// maxPolyID returns the highest polymorphic serializer ID reachable from s,
// including through polymorphic type alternatives, or -1 if none.
//
// It is used to size the per-entity polySerializers slice of classes so that
// only classes that can actually reach polymorphic pointer fields pay for the
// per-entity tracking and lose the shared fast paths.
//
// cache memoizes results across calls; back-edges (serializers referencing
// themselves through fixed tables) contribute nothing, which is safe for a
// max-aggregation since every node of a cycle is reachable without them.
func (s *serializer) maxPolyID(cache map[*serializer]int) int {
	if s == nil {
		return -1
	}

	if v, ok := cache[s]; ok {
		return v
	}

	cache[s] = -1 // cycle guard

	maxID := -1
	for _, f := range s.fields {
		if f.polySerializerID > maxID {
			maxID = f.polySerializerID
		}

		if m := f.serializer.maxPolyID(cache); m > maxID {
			maxID = m
		}

		for _, pt := range f.polyTypes {
			if m := pt.maxPolyID(cache); m > maxID {
				maxID = m
			}
		}
	}

	cache[s] = maxID

	return maxID
}
