package fake

import (
	"github.com/stretchr/testify/mock"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
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

// OnUpdate is a mock-implementation of Property.OnUpdate().
func (p *Property) OnUpdate(handler st.PropertyUpdateHandler) {
	p.Called(handler)
}

// Bind is a mock-implementation of Property.Bind().
func (p *Property) Bind(variable any, valueType st.PropertyValueType) {
	p.Called(variable, valueType)
}
