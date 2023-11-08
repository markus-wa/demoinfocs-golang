package fake

import (
	"github.com/stretchr/testify/mock"

	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

var _ st.Property = new(Property)

// Property is a mock for of sendtables.Property.
type Property struct {
	mock.Mock
}

// Name is a mock-implementation of Property.Name().
func (p *Property) Name() string {
	return p.Called().Get(0).(string)
}

// Value is a mock-implementation of Property.Value().
func (p *Property) Value() st.PropertyValue {
	return p.Called().Get(0).(st.PropertyValue)
}

// Type is a mock-implementation of Property.Type().
func (p *Property) Type() st.PropertyType {
	return p.Called().Get(0).(st.PropertyType)
}

// OnUpdateWithId is a mock-implementation of Property.OnUpdateWithId().
func (p *Property) OnUpdateWithId(handler st.PropertyUpdateHandler, id int64) {
	p.Called(handler, id)
}

// OnUpdate is a mock-implementation of Property.OnUpdate().
func (p *Property) OnUpdate(handler st.PropertyUpdateHandler) int64 {
	return p.Called(handler).Get(0).(int64)
}

// Bind is a mock-implementation of Property.Bind().
func (p *Property) Bind(variable any, valueType st.PropertyValueType) int64 {
	return p.Called(variable, valueType).Get(0).(int64)
}

// ArrayElementType is a mock-implementation of Property.ArrayElementType().
func (p *Property) ArrayElementType() st.PropertyType {
	return p.Called().Get(0).(st.PropertyType)
}
