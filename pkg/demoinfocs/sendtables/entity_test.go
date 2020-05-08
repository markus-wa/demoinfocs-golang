package sendtables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testData = struct {
	entity entity
}{
	entity: entity{
		props: []property{
			{value: PropertyValue{IntVal: 10}},
			{value: PropertyValue{IntVal: 20}},
			{value: PropertyValue{IntVal: 30}},
		},
		serverClass: &ServerClass{propNameToIndex: map[string]int{
			"myProp":     0,
			"test":       1,
			"anotherOne": 2,
		}},
	},
}

func TestEntity_Properties(t *testing.T) {
	ent := entity{props: []property{{value: PropertyValue{IntVal: 1}}}}

	assert.Equal(t, &ent.props[0], ent.Properties()[0])
}

func TestEntity_ServerClass(t *testing.T) {
	assert.Equal(t, testData.entity.serverClass, testData.entity.ServerClass())
}

func TestEntity_Property(t *testing.T) {
	assert.Equal(t, &testData.entity.props[1], testData.entity.Property("test"))
}

func TestEntity_Property_Nil(t *testing.T) {
	assert.Nil(t, testData.entity.Property("not_found"))
}

func TestEntity_Property_Value(t *testing.T) {
	val, ok := testData.entity.PropertyValue("test")

	assert.True(t, ok)
	assert.Equal(t, PropertyValue{IntVal: 20}, val)
}

func TestEntity_PropertyValue_NotFound(t *testing.T) {
	val, ok := testData.entity.PropertyValue("not_found")

	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestEntity_PropertyValueMust_NotFound_Panics(t *testing.T) {
	f := func() {
		testData.entity.PropertyValueMust("not_found")
	}

	assert.Panics(t, f)
}

func TestProperty_Name(t *testing.T) {
	prop := property{entry: &flattenedPropEntry{name: "test"}}

	assert.Equal(t, "test", prop.Name())
}
