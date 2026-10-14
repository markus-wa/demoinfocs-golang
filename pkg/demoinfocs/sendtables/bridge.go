package sendtables

import (
	"github.com/golang/geo/r3"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/stbridge"
)

// Registration of the parser's own entity callbacks, see stbridge.
// These skip tagging the callbacks' panics as user-code panics, unlike the public registration methods.
func init() {
	stbridge.OnUpdate = func(prop, handler any) {
		prop.(*Property).onUpdate(handler.(PropertyUpdateHandler))
	}
	stbridge.OnEntityCreated = func(class, handler any) {
		class.(*ServerClass).onEntityCreated(handler.(EntityCreatedHandler))
	}
	stbridge.BindProperty = func(entity any, prop string, variable, t any) {
		entity.(*Entity).bindProperty(prop, variable, t.(PropertyValueType))
	}
	stbridge.OnPositionUpdate = func(entity any, handler func(pos r3.Vector)) {
		entity.(*Entity).onPositionUpdate(handler)
	}
	stbridge.OnDestroy = func(entity any, delegate func()) {
		entity.(*Entity).addOnDestroy(delegate)
	}
	stbridge.OnCreateFinished = func(entity any, delegate func()) {
		entity.(*Entity).addOnCreateFinished(delegate)
	}
	stbridge.OnUpdateFinished = func(entity any, delegate func()) {
		entity.(*Entity).addOnUpdateFinished(delegate)
	}
}
