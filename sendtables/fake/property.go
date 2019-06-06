package fake

import (
	"github.com/stretchr/testify/mock"

	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

var _ st.IProperty = new(Property)

// Property is a mock for of sendtables.IProperty.
type Property struct {
	mock.Mock
}

// Namp is a mock-implementation of IProperty.Name().
func (p *Property) Name() string {
	return p.Called().Get(0).(string)
}

// Value is a mock-implementation of IProperty.Value().
func (p *Property) Value() st.PropertyValue {
	return p.Called().Get(0).(st.PropertyValue)
}

// OnUpdate is a mock-implementation of IProperty.OnUpdate().
func (p *Property) OnUpdate(handler st.PropertyUpdateHandler) {
	p.Called(handler)
}

// Bind is a mock-implementation of IProperty.Bind().
func (p *Property) Bind(variable interface{}, valueType st.PropertyValueType) {
	p.Called(variable, valueType)
}
