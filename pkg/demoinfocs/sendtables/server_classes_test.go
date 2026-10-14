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
	p := NewParser()
	sc := p.ServerClasses()

	assert.Nil(t, sc.All())

	c := newSyntheticClass(3, "CFoo")
	p.classesById[c.classId] = c
	p.classesByName[c.name] = c

	assert.Equal(t, []*ServerClass{c}, sc.All())
	assert.Same(t, c, sc.FindByName("CFoo"))
	assert.Equal(t, "CFoo", sc.String())
}

func TestServerClasses_AllAndStringSortedByID(t *testing.T) {
	p := NewParser()

	for _, c := range []*ServerClass{
		newSyntheticClass(7, "CSeven"),
		newSyntheticClass(0, "CZero"),
		newSyntheticClass(42, "CFortyTwo"),
		newSyntheticClass(3, "CThree"),
		newSyntheticClass(12, "CTwelve"),
	} {
		p.classesById[c.classId] = c
		p.classesByName[c.name] = c
	}

	sc := p.ServerClasses()

	for range 10 { // map iteration order is random, repeat to catch unsorted results
		all := sc.All()
		ids := make([]int, 0, len(all))

		for _, c := range all {
			ids = append(ids, c.ID())
		}

		assert.Equal(t, []int{0, 3, 7, 12, 42}, ids)
		assert.Equal(t, "CZero\nCThree\nCSeven\nCTwelve\nCFortyTwo", sc.String())
	}
}
