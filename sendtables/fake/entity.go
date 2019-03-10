// Package fake provides basic mocks for IEntity.
// See examples/mocking (https://github.com/markus-wa/demoinfocs-golang/tree/master/examples/mocking).
package fake

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/bitread"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
	"github.com/stretchr/testify/mock"
)

// Entity is a mock for of sendtables.IEntity.
type Entity struct {
	mock.Mock
}

// ServerClass is a mock-implementation of IEntity.ServerClass().
func (e *Entity) ServerClass() *st.ServerClass {
	return e.Called().Get(0).(*st.ServerClass)
}

// ID is a mock-implementation of IEntity.ID().
func (e *Entity) ID() int {
	return e.Called().Int(0)
}

// Properties is a mock-implementation of IEntity.Properties().
func (e *Entity) Properties() []st.Property {
	return e.Called().Get(0).([]st.Property)
}

// FindProperty is a mock-implementation of IEntity.FindProperty().
func (e *Entity) FindProperty(name string) *st.Property {
	return e.Called(name).Get(0).(*st.Property)
}

// BindProperty is a mock-implementation of IEntity.BindProperty().
func (e *Entity) BindProperty(name string, variable interface{}, valueType st.PropertyValueType) {
	e.Called(name, variable, valueType)
}

// ApplyUpdate is a mock-implementation of IEntity.ApplyUpdate().
func (e *Entity) ApplyUpdate(reader *bitread.BitReader) {
	e.Called(reader)
}

// Position is a mock-implementation of IEntity.Position().
func (e *Entity) Position() r3.Vector {
	return e.Called().Get(0).(r3.Vector)
}

// OnPositionUpdate is a mock-implementation of IEntity.OnPositionUpdate().
func (e *Entity) OnPositionUpdate(handler func(pos r3.Vector)) {
	e.Called(handler)
}

// BindPosition is a mock-implementation of IEntity.BindPosition().
func (e *Entity) BindPosition(pos *r3.Vector) {
	e.Called(pos)
}

// OnDestroy is a mock-implementation of IEntity.OnDestroy().
func (e *Entity) OnDestroy(delegate func()) {
	e.Called(delegate)
}

// Destroy is a mock-implementation of IEntity.Destroy().
func (e *Entity) Destroy() {
	e.Called()
}

// OnCreateFinished is a mock-implementation of IEntity.OnCreateFinished().
func (e *Entity) OnCreateFinished(delegate func()) {
	e.Called()
}
