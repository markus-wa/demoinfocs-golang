package sendtables

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func TestServerClassGetters(t *testing.T) {
	sc := ServerClass{
		id:            1,
		name:          "TestClass",
		dataTableID:   2,
		dataTableName: "ADataTable",
	}

	assert.Equal(t, sc.id, sc.ID(), "ID should return the id field")
	assert.Equal(t, sc.name, sc.Name(), "Name should return the name field")
	assert.Equal(t, sc.dataTableID, sc.DataTableID(), "DataTableID should return the dataTableID field")
	assert.Equal(t, sc.dataTableName, sc.DataTableName(), "DataTableName should return the dataTableName field")
}

func TestServerClassPropertyEntries(t *testing.T) {
	var sc ServerClass

	assert.Empty(t, sc.PropertyEntries())

	sc.flattenedProps = []flattenedPropEntry{{name: "prop1"}, {name: "prop2"}}

	assert.ElementsMatch(t, []string{"prop1", "prop2"}, sc.PropertyEntries())
}

func TestServerClassString(t *testing.T) {
	sc := ServerClass{
		id:            1,
		name:          "TestClass",
		dataTableID:   2,
		dataTableName: "ADataTable",
	}

	expectedString := `ServerClass: id=1 name=TestClass
	dataTableId=2
	dataTableName=ADataTable
	baseClasses:
		-
	properties:
		-`

	assert.Equal(t, expectedString, sc.String())

	sc.baseClasses = []*ServerClass{{name: "AnotherClass"}, {name: "YetAnotherClass"}}
	sc.flattenedProps = []flattenedPropEntry{{name: "prop1"}, {name: "prop2"}}

	expectedString = `ServerClass: id=1 name=TestClass
	dataTableId=2
	dataTableName=ADataTable
	baseClasses:
		AnotherClass
		YetAnotherClass
	properties:
		prop1
		prop2`

	assert.Equal(t, expectedString, sc.String())
}
