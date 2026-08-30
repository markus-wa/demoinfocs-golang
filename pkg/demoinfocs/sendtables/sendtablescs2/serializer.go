package sendtablescs2

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

func (s *serializer) rebuildFieldIndexes() {
	s.fieldIndexes = make(map[string]*fieldIndex, len(s.fields))

	for i, f := range s.fields {
		s.fieldIndexes[f.name] = &fieldIndex{
			index: i,
			field: f,
		}
	}

	// Deprecated aliases: the bare varName of a disambiguated field keeps
	// resolving to the same declaration it resolved to before the rename
	// (the last one registered), so existing name-based lookups keep working.
	// Use the sendNode-prefixed names instead; these aliases may be removed
	// in a future major version.
	for i, f := range s.fields {
		if f.name == f.varName {
			continue
		}

		if owner := s.fieldIndexes[f.varName]; owner != nil && owner.field.name == f.varName {
			continue // a field legitimately owns the bare name
		}

		s.fieldIndexes[f.varName] = &fieldIndex{
			index: i,
			field: f,
		}
	}
}

// resolveFieldNameCollisions disambiguates fields that share a varName within
// the same serializer and are distinguished on the wire only by their
// send_node — e.g. CCSPlayerPawn declares m_vecX/m_vecY/m_vecZ twice, once
// under m_vecVelocity and once under m_vecViewOffset, so name-based lookups
// would otherwise resolve to whichever was registered last and shadow the
// other. Affected fields are renamed to sendNode+"."+varName; that is the
// canonical name used for generated names, while the bare varName remains
// available as a deprecated lookup alias resolving to the same declaration
// it resolved to before the rename.
//
// Field objects are shared between serializers (a class serializer re-lists
// its parents' fields), so renames are applied globally after a full detection
// pass and every serializer containing a renamed field re-keys its index.
// serializers must contain every instance created for the message, including
// ones shadowed by a later version of the same serializer name.
func resolveFieldNameCollisions(serializers []*serializer) {
	renamed := make(map[*field]bool)

	for _, s := range serializers {
		byName := make(map[string]*field, len(s.fields))

		for _, f := range s.fields {
			if prev, ok := byName[f.name]; ok && prev != f && prev.sendNode != f.sendNode {
				if prev.sendNode != "" {
					renamed[prev] = true
				}

				if f.sendNode != "" {
					renamed[f] = true
				}
			}

			byName[f.name] = f
		}
	}

	if len(renamed) == 0 {
		return
	}

	for f := range renamed {
		f.name = f.sendNode + "." + f.varName
	}

	for _, s := range serializers {
		for _, f := range s.fields {
			if renamed[f] {
				s.rebuildFieldIndexes()
				break
			}
		}
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
