package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
	stfake "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables/fake"
)

func TestCachedProps_ResolvesOncePerEntity(t *testing.T) {
	entity := entityWithProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{Any: int32(100), S2: true}},
	})

	var c cachedProps

	for i := 0; i < 3; i++ {
		v, ok := c.value(entity, propHealth)

		assert.True(t, ok)
		assert.Equal(t, 100, v.Int())
	}

	entity.AssertNumberOfCalls(t, "Property", 1)
}

func TestCachedProps_ResetsOnNewEntity(t *testing.T) {
	first := entityWithProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{Any: int32(100), S2: true}},
	})
	second := entityWithProperties([]fakeProp{
		{propName: "m_iHealth", value: st.PropertyValue{Any: int32(37), S2: true}},
	})

	var c cachedProps

	v, _ := c.value(first, propHealth)
	assert.Equal(t, 100, v.Int())

	v, _ = c.value(second, propHealth)
	assert.Equal(t, 37, v.Int())

	v, _ = c.value(first, propHealth)
	assert.Equal(t, 100, v.Int())

	first.AssertNumberOfCalls(t, "Property", 2)
	second.AssertNumberOfCalls(t, "Property", 1)
}

func TestCachedProps_MissingPropertyIsRetried(t *testing.T) {
	entity := new(stfake.Entity)
	entity.On("Property", mock.Anything).Return(nil)

	var c cachedProps

	for i := 0; i < 2; i++ {
		_, ok := c.value(entity, propFlashbangAmmo)

		assert.False(t, ok)
	}

	entity.AssertNumberOfCalls(t, "Property", 2)

	assert.PanicsWithValue(t, "property 'm_pWeaponServices.m_iAmmo.0014' not found", func() {
		c.mustValue(entity, propFlashbangAmmo)
	})
}

func TestCachedProps_AllNamesSet(t *testing.T) {
	for idx, name := range cachedPropNames {
		assert.NotEmpty(t, name, "cachedProp %d has no name", idx)
	}
}
