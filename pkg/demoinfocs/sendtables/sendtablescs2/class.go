// Package sendtablescs2 implements the CS2 send-tables (FlattenedSerializer) parser.
package sendtablescs2

import (
	"fmt"
	"strings"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

// fpNameTreeCache is kept as fallback for rare deep/large field paths.
type fpNameTreeCache struct {
	next []*fpNameTreeCache
	name string
}

type class struct {
	classId         int32 //nolint:revive
	name            string
	serializer      *serializer
	polyCount       int // size of the per-entity polySerializers slice needed by entities of this class; 0 if no polymorphic pointer fields are reachable
	createdHandlers []st.EntityCreatedHandler
	fpNameCache     *fpNameTreeCache
	// fpFlatCache provides O(1) lookup for the common case: depth ≤ 3 and
	// all path components ≤ 16383. Paths outside this range fall back to
	// fpNameCache. Each key packs up to 4 path components (14 bits each)
	// plus the depth (8 bits) into a uint64.
	fpFlatCache map[uint64]string
}

func (c *class) ID() int {
	return int(c.classId)
}

func (c *class) Name() string {
	return c.name
}

func (c *class) PropertyEntries() []string {
	return c.collectFieldsEntries(c.serializer.fields, "")
}

func (c *class) OnEntityCreated(handler st.EntityCreatedHandler) {
	c.createdHandlers = append(c.createdHandlers, handler)
}

func (c *class) String() string {
	props := make([]string, 0, len(c.serializer.fields))

	for _, f := range c.serializer.fields {
		props = append(props, fmt.Sprintf("%s: %s", f.name, f.varType))
	}

	return fmt.Sprintf("%d %s\n %s", c.classId, c.name, strings.Join(props, "\n "))
}

func (c *class) collectFieldsEntries(fields []*field, prefix string) []string {
	paths := make([]string, 0)

	for _, field := range fields {
		if len(field.polyTypes) > 0 { //nolint:gocritic
			// Polymorphic pointer: each candidate serializer contributes its
			// own sub-fields. The active type varies per entity, so enumerate
			// every alternative (the field's own serializer is polyTypes[0]).
			paths = append(paths, c.collectPolyFieldEntries(field, prefix)...)
		} else if field.serializer != nil {
			subPaths := c.collectFieldsEntries(field.serializer.fields, prefix+field.serializer.name+".")
			paths = append(paths, subPaths...)
		} else {
			paths = append(paths, prefix+field.name)
		}
	}

	return paths
}

// collectPolyFieldEntries enumerates the property entries of every serializer a
// polymorphic pointer field could activate, deduplicated. The active type varies
// per entity, so all alternatives must be listed.
func (c *class) collectPolyFieldEntries(f *field, prefix string) []string {
	paths := make([]string, 0)
	seen := make(map[string]bool)

	for _, ser := range f.polyTypes {
		if ser == nil {
			continue
		}

		for _, sub := range c.collectFieldsEntries(ser.fields, prefix+ser.name+".") {
			if !seen[sub] {
				seen[sub] = true
				paths = append(paths, sub)
			}
		}
	}

	return paths
}

// fpFlatKey encodes a field path as a uint64 for O(1) map lookup.
// Returns (key, true) when depth ≤ 3 and all components fit in 14 bits.
// Falls back to the tree cache otherwise.
func fpFlatKey(fp *fieldPath) (uint64, bool) {
	if fp.last > 3 {
		return 0, false
	}

	var key uint64

	for i := 0; i <= fp.last; i++ {
		v := fp.path[i]
		if uint(v) > 0x3FFF {
			return 0, false
		}

		key |= uint64(v) << uint(i*14)
	}

	key |= uint64(fp.last) << 56

	return key, true
}

//nolint:nestif
func (c *class) getNameForFieldPath(fp *fieldPath, ps []*serializer) string {
	if ps == nil {
		// No polymorphic fields: use the shared class-level caches.
		if key, ok := fpFlatKey(fp); ok {
			if name, hit := c.fpFlatCache[key]; hit {
				return name
			}

			name := strings.Join(c.serializer.getNameForFieldPath(fp, 0, nil), ".")
			c.fpFlatCache[key] = name

			return name
		}

		// Slow path: deep or large-component path — use the pointer tree.
		currentCacheNode := c.fpNameCache

		for i := 0; i <= fp.last; i++ {
			pos := fp.path[i]
			if pos >= len(currentCacheNode.next) {
				needed := pos + 1
				if cap(currentCacheNode.next) >= needed {
					currentCacheNode.next = currentCacheNode.next[:needed]
				} else {
					newCap := needed * 2
					if newCap < 8 {
						newCap = 8
					}

					newNext := make([]*fpNameTreeCache, needed, newCap)
					copy(newNext, currentCacheNode.next)
					currentCacheNode.next = newNext
				}
			}

			if currentCacheNode.next[pos] == nil {
				currentCacheNode.next[pos] = &fpNameTreeCache{}
			}

			currentCacheNode = currentCacheNode.next[pos]
		}

		if currentCacheNode.name == "" {
			currentCacheNode.name = strings.Join(c.serializer.getNameForFieldPath(fp, 0, nil), ".")
		}

		return currentCacheNode.name
	}

	// Polymorphic entity: skip class-level caches because the active type
	// varies per entity and the same field path can resolve to different names.
	return strings.Join(c.serializer.getNameForFieldPath(fp, 0, ps), ".")
}

func (c *class) getFieldPathForName(fp *fieldPath, name string, ps []*serializer) bool {
	return c.serializer.getFieldPathForName(fp, name, ps)
}

func (c *class) getFieldPaths(fp *fieldPath, state *fieldState, ps []*serializer) []*fieldPath {
	return c.serializer.getFieldPaths(fp, state, ps)
}
