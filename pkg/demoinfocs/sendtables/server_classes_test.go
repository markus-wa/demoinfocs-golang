package sendtables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerClasses_ZeroValue(t *testing.T) {
	var sc ServerClasses

	assert.Nil(t, sc.All())
	assert.Nil(t, sc.FindByName("CCSPlayerPawn"))
	assert.Empty(t, sc.String())
}

func TestServerClasses_ReflectsClassesAddedLater(t *testing.T) {
	p := NewParser(nil)
	sc := p.ServerClasses()

	assert.Nil(t, sc.All())

	c := newSyntheticClass(3, "CFoo")
	p.classesById[c.classId] = c
	p.classesByName[c.name] = c

	assert.Equal(t, []*ServerClass{c}, sc.All())
	assert.Same(t, c, sc.FindByName("CFoo"))
	assert.Equal(t, "CFoo", sc.String())
}
