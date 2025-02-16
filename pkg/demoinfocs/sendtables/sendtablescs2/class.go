package sendtablescs2

import (
	"fmt"
	"strings"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type fpNameTreeCache struct {
	next map[int]*fpNameTreeCache
	name string
}

type class struct {
	classId         int32
	name            string
	serializer      *serializer
	createdHandlers []st.EntityCreatedHandler
	fpNameCache     *fpNameTreeCache
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
		props = append(props, fmt.Sprintf("%s: %s", f.varName, f.varType))
	}

	return fmt.Sprintf("%d %s\n %s", c.classId, c.name, strings.Join(props, "\n "))
}

func (c *class) collectFieldsEntries(fields []*field, prefix string) []string {
	paths := make([]string, 0)

	for _, field := range fields {
		if field.serializer != nil {
			subPaths := c.collectFieldsEntries(field.serializer.fields, prefix+field.serializer.name+".")
			paths = append(paths, subPaths...)
		} else {
			paths = append(paths, prefix+field.varName)
		}
	}

	return paths
}

func (c *class) getNameForFieldPath(fp *fieldPath) string {
	currentCacheNode := c.fpNameCache

	for i := 0; i <= fp.last; i++ {
		if currentCacheNode.next == nil {
			currentCacheNode.next = make(map[int]*fpNameTreeCache)
		}

		pos := fp.path[i]
		next, exists := currentCacheNode.next[pos]
		if !exists {
			next = &fpNameTreeCache{}
			currentCacheNode.next[pos] = next
		}
		currentCacheNode = next
	}

	if currentCacheNode.name == "" {
		currentCacheNode.name = strings.Join(c.serializer.getNameForFieldPath(fp, 0), ".")
	}

	return currentCacheNode.name
}

func (c *class) getTypeForFieldPath(fp *fieldPath) *fieldType {
	return c.serializer.getTypeForFieldPath(fp, 0)
}

func (c *class) getDecoderForFieldPath(fp *fieldPath) fieldDecoder {
	return c.serializer.getDecoderForFieldPath(fp, 0)
}

func (c *class) getFieldPathForName(fp *fieldPath, name string) bool {
	return c.serializer.getFieldPathForName(fp, name)
}

func (c *class) getFieldPaths(fp *fieldPath, state *fieldState) []*fieldPath {
	return c.serializer.getFieldPaths(fp, state)
}
