package sendtables_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stfake"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

func TestEntity_PropertyValueMust(t *testing.T) {
	e := stfake.New(stfake.Spec{Class: "CFoo", ID: 7, Props: []stfake.Prop{stfake.P("m_x", int32(1))}})

	assert.Equal(t, 1, e.PropertyValueMust("m_x").Int())

	var r any

	func() {
		defer func() { r = recover() }()

		e.PropertyValueMust("m_missing")
	}()

	assert.Equal(t, &st.PropertyNotFoundError{Name: "m_missing", EntityID: 7, ServerClass: "CFoo"}, r)
}

func TestPropertyNotFoundError_Error(t *testing.T) {
	err := &st.PropertyNotFoundError{Name: "m_missing", EntityID: 7, ServerClass: "CFoo"}

	assert.EqualError(t, err, `property 'm_missing' not found on entity 7 (server-class "CFoo")`)
}
