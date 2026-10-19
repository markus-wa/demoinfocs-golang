package sendtables

import (
	"fmt"
	"strings"

	"github.com/golang/geo/r3"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/internal/panics"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

// Entity stores an entity in the game (e.g. players etc.) with its properties.
//
// Entities are created by the parser, see demoinfocs.GameState.Entities() and ServerClass.OnEntityCreated();
// the zero value is not usable.
type Entity struct {
	index  int32
	serial int32
	class  *ServerClass
	active bool
	// inPVS is cleared when the entity leaves the snapshot (PVS) or is destroyed, see InPVS().
	inPVS bool
	// replaced is set if the entity was destroyed because a new entity was created at its index, see Replaced().
	replaced bool
	state    *fieldState
	fpCache  map[string]*fieldPath
	fpNoop   map[string]bool
	// polySerializers holds the currently active serializer for each polymorphic
	// pointer field, indexed by field.polySerializerID. Nil entries mean the
	// pointer is inactive. This slice is per-entity so that two entities of the
	// same class can hold different active types simultaneously.
	polySerializers []*serializer
	// budgetSlots are the live slots charged for the entity, see entityBudget.
	budgetSlots int64

	onCreateFinished []func()
	onUpdateFinished []func()
	onDestroy        []func()
	onLeavePVS       []func()
	onEnterPVS       []func()
	updateHandlers   map[string][]PropertyUpdateHandler
	// handlerLog preserves the global registration order of update handlers.
	// updateHandlers remains the name-keyed index for lookup; this log is the
	// single source of truth for dispatch ordering (handlersByFP rebuilds and
	// the initial entity-creation dispatch). Handlers are never unregistered,
	// so the log is append-only and never stale.
	handlerLog   []handlerRegistration
	handlersByFP map[uint64][]PropertyUpdateHandler
	hasHandlers  bool // cached: len(updateHandlers) > 0
	propCache    map[string]*Property
}

// handlerRegistration records a single update-handler registration so that
// dispatch order can follow registration order instead of map-iteration order.
type handlerRegistration struct {
	name    string
	handler PropertyUpdateHandler
}

// ServerClass returns the entity's server-class.
func (e *Entity) ServerClass() *ServerClass {
	return e.class
}

// ID returns the entity's ID (its index in the entity list).
func (e *Entity) ID() int {
	return int(e.index)
}

// SerialNum returns the entity's serial number.
func (e *Entity) SerialNum() int {
	return int(e.serial)
}

// Properties returns all properties of the entity.
func (e *Entity) Properties() (out []*Property) {
	for _, fp := range e.class.getFieldPaths(newFieldPath(), e.state, e.polySerializers) {
		prop := e.Property(e.class.getNameForFieldPath(fp, e.polySerializers))
		if prop == nil {
			continue // a generated name must always resolve; don't hand out nil properties if it doesn't
		}

		out = append(out, prop)
	}

	return
}

// Property is a property of an Entity and allows registering handlers
// that are triggered on updates of the property.
//
// Properties are obtained from Entity.Property() and Entity.Properties(); the zero value is not usable.
type Property struct {
	entity *Entity
	name   string
}

// Name returns the property's name.
func (p *Property) Name() string {
	return p.name
}

// Value returns the current value of the property.
func (p *Property) Value() PropertyValue {
	v := p.entity.get(p.name)

	fs, ok := v.(*fieldState)
	if ok {
		v = fs.state
	}

	return PropertyValue{
		Any: v,
	}
}

// OnUpdate registers a handler for updates of the property's value.
//
// Handlers registered before the entity is fully created (e.g. in a ServerClass.OnEntityCreated() handler)
// are called once with the initial value after the creation; handlers registered later are only
// called on subsequent updates.
func (p *Property) OnUpdate(handler PropertyUpdateHandler) {
	p.onUpdate(guardUpdateHandler(handler))
}

// guardUpdateHandler tags the panics of a user's update handler as user-code panics.
func guardUpdateHandler(handler PropertyUpdateHandler) PropertyUpdateHandler {
	return func(v PropertyValue) {
		defer panics.GuardUser()

		handler(v)
	}
}

// onUpdate is OnUpdate without tagging the handler's panics as user-code panics.
func (p *Property) onUpdate(handler PropertyUpdateHandler) {
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], handler)
	p.entity.handlerLog = append(p.entity.handlerLog, handlerRegistration{name: p.name, handler: handler})
	p.entity.addHandlerByFP(p.name, handler)
	p.entity.hasHandlers = true
}

// addHandlerByFP registers handler in the fast field-path-keyed lookup table.
// Called alongside every updateHandlers insertion so the two maps stay in sync.
func (e *Entity) addHandlerByFP(name string, handler PropertyUpdateHandler) {
	fp := newFieldPath()
	defer fp.release()

	if !e.class.getFieldPathForName(fp, name, e.polySerializers) {
		return
	}

	key, ok := fpFlatKey(fp)
	if !ok {
		return
	}

	if e.handlersByFP == nil {
		e.handlersByFP = make(map[uint64][]PropertyUpdateHandler)
	}

	e.handlersByFP[key] = append(e.handlersByFP[key], handler)
}

// applyPolyUpdate stores the newly selected serializer of a polymorphic pointer
// and invalidates the per-entity caches whenever the active type changes.
//
// Note on entity state: the activation bool stored at the pointer field's slot
// is replaced by a *fieldState once sub-field updates arrive (fieldState.set
// semantics, same as for regular fixed pointers), so the value is only readable
// as a bool while the pointer has no sub-fields. The source of truth for the
// active type is e.polySerializers, not the stored value.
func (e *Entity) applyPolyUpdate(pu *polyUpdate) {
	if e.polySerializers[pu.id] == pu.ser {
		// Same active type, nothing to invalidate.
		return
	}

	e.polySerializers[pu.id] = pu.ser

	// Invalidate per-entity caches: the active type changed, so cached field
	// paths, name-to-path resolutions and property lookups may no longer be valid.
	clear(e.fpCache)
	clear(e.fpNoop)
	clear(e.propCache)

	e.rebuildHandlersByFP()
}

// rebuildHandlersByFP re-resolves the field paths of all registered update
// handlers after the active polymorphic type changed, so fp-keyed dispatch
// keeps matching the fields of the now-active serializer.
//
// Name-keyed handlers (e.updateHandlers) are preserved as-is: handlers bound to
// sub-fields of a type that is no longer active simply stop firing until the
// type switches back. The fp-keyed index is rebuilt because those keys are
// numeric paths, which map to different fields under a different active type.
//
// The rebuild walks handlerLog in registration order, so handlers sharing an fp
// key (e.g. via deprecated bare-name aliases resolving to the same field) keep
// firing in their original registration order across rebuilds.
func (e *Entity) rebuildHandlersByFP() {
	if len(e.handlerLog) == 0 {
		e.handlersByFP = nil

		return
	}

	e.handlersByFP = make(map[uint64][]PropertyUpdateHandler, len(e.updateHandlers))

	for _, r := range e.handlerLog {
		e.addHandlerByFP(r.name, r.handler)
	}
}

// fireInitialUpdateHandlers fires every registered update handler once with the
// entity's current value for its property, in registration order. It is called
// right after a newly created entity has been fully read (baseline + initial
// update), so update handlers registered in created-handlers observe the
// initial state immediately.
func (e *Entity) fireInitialUpdateHandlers() {
	for _, r := range e.handlerLog {
		r.handler(e.PropertyValueMust(r.name))
	}
}

type bindFactory func(variable any) PropertyUpdateHandler

var bindFactoryByType = map[PropertyValueType]bindFactory{
	ValTypeVector: func(variable any) PropertyUpdateHandler {
		return func(v PropertyValue) {
			*variable.(*r3.Vector) = v.R3Vec()
		}
	},
	ValTypeInt: func(variable any) PropertyUpdateHandler {
		return func(v PropertyValue) {
			*variable.(*int) = v.Int()
		}
	},
	ValTypeArray: func(variable any) PropertyUpdateHandler {
		return func(v PropertyValue) {
			*variable.(*[]any) = v.Array()
		}
	},
	ValTypeString: func(variable any) PropertyUpdateHandler {
		return func(v PropertyValue) {
			*variable.(*string) = v.String()
		}
	},
	ValTypeBoolInt: func(variable any) PropertyUpdateHandler {
		return func(v PropertyValue) {
			*variable.(*bool) = v.BoolVal()
		}
	},
	ValTypeFloat32: func(variable any) PropertyUpdateHandler {
		return func(v PropertyValue) {
			*variable.(*float32) = v.Float()
		}
	},
	ValTypeFloat64: func(variable any) PropertyUpdateHandler {
		return func(v PropertyValue) {
			*variable.(*float64) = float64(v.Float())
		}
	},
}

/*
Bind binds a property's value to a pointer.

Example:

	var i int
	property.Bind(&i, ValTypeInt)

This will bind the property's value to i so every time it's updated i is updated as well.

The valueType indicates which field of the PropertyValue to use for the binding.
*/
func (p *Property) Bind(variable any, t PropertyValueType) {
	p.onUpdate(guardUpdateHandler(bindFactoryByType[t](variable)))
}

// bind is Bind without tagging the binding's panics (e.g. a wrong pointer type) as user-code panics.
func (p *Property) bind(variable any, t PropertyValueType) {
	p.onUpdate(bindFactoryByType[t](variable))
}

// Property finds a property on the entity by name.
//
// Returns nil if the property wasn't found.
func (e *Entity) Property(name string) *Property {
	if prop := e.propCache[name]; prop != nil {
		return prop
	}

	var ok bool
	if len(e.polySerializers) == 0 {
		// Fast path for entities without polymorphic pointer fields.
		ok = e.class.serializer.checkFieldName(name)
	} else {
		// Poly-aware path: use the active serializers to check existence.
		// The predicate is identical to Get's (the same resolver call with the
		// same active serializers), so fpNoop doubles as the negative property
		// cache; it is cleared together with propCache whenever the active type
		// changes.
		if e.fpNoop[name] {
			return nil
		}

		fp := newFieldPath()
		ok = e.class.getFieldPathForName(fp, name, e.polySerializers)
		fp.release()

		if !ok {
			e.fpNoop[name] = true

			return nil
		}
	}

	if !ok {
		return nil
	}

	prop := &Property{
		entity: e,
		name:   name,
	}
	e.propCache[name] = prop

	return prop
}

// BindProperty combines Property() & Property.Bind() into one.
// Essentially binds a property's value to a pointer.
// See the docs of the two individual functions for more info.
func (e *Entity) BindProperty(prop string, variable any, t PropertyValueType) {
	e.Property(prop).Bind(variable, t)
}

// bindProperty is BindProperty without tagging the binding's panics as user-code panics.
func (e *Entity) bindProperty(prop string, variable any, t PropertyValueType) {
	e.Property(prop).bind(variable, t)
}

// PropertyValue finds a property on the entity by name and returns its value.
//
// Returns false as second value if the property was not found.
func (e *Entity) PropertyValue(name string) (PropertyValue, bool) {
	prop := e.Property(name)
	if prop == nil {
		return PropertyValue{}, false
	}

	v := prop.Value()

	return v, true
}

// PropertyValueMust finds a property on the entity by name and returns its value.
//
// Panics with a *PropertyNotFoundError if the property was not found.
// When this happens during parsing (also in a handler), parsing stops and demoinfocs.Parser.ParseToEnd()
// returns a demoinfocs.DecodeError with the *PropertyNotFoundError as cause, since a missing property means
// the demo's schema differs from what the code expects (e.g. after a game update).
// Use PropertyValue() for properties that may be missing.
func (e *Entity) PropertyValueMust(name string) PropertyValue {
	val, ok := e.PropertyValue(name)
	if !ok {
		panic(newPropertyNotFoundError(e, name))
	}

	return val
}

// PropertyNotFoundError is the panic value of Entity.PropertyValueMust() if the entity has no property with the given name.
// This usually means the demo's schema differs from what the caller expects (e.g. after a game update).
// It's returned by demoinfocs.Parser.ParseToEnd() as the cause of a demoinfocs.DecodeError (see errors.As()).
type PropertyNotFoundError struct {
	Name        string // Name of the property
	EntityID    int    // ID of the entity, see Entity.ID()
	ServerClass string // Name of the entity's server-class, see ServerClass.Name()
}

func newPropertyNotFoundError(e *Entity, name string) *PropertyNotFoundError {
	err := &PropertyNotFoundError{
		Name:     name,
		EntityID: int(e.index),
	}

	if e.class != nil {
		err.ServerClass = e.class.name
	}

	return err
}

// Error implements error.
func (e *PropertyNotFoundError) Error() string {
	return fmt.Sprintf("property '%s' not found on entity %d (server-class %q)", e.Name, e.EntityID, e.ServerClass)
}

const (
	propCellX = "CBodyComponent.m_cellX"
	propCellY = "CBodyComponent.m_cellY"
	propCellZ = "CBodyComponent.m_cellZ"
	propVecX  = "CBodyComponent.m_vecX"
	propVecY  = "CBodyComponent.m_vecY"
	propVecZ  = "CBodyComponent.m_vecZ"
)

// Returns a coordinate from a cell + offset
func coordFromCell(cell uint64, offset float32) float64 {
	const (
		cellBits    = 9
		maxCoordInt = 16384
	)

	cellCoord := float64(cell)*float64(1<<cellBits) - maxCoordInt

	return cellCoord + float64(offset)
}

// Position returns the entity's position in world coordinates.
func (e *Entity) Position() r3.Vector {
	cellXVal := e.Property(propCellX).Value()
	cellYVal := e.Property(propCellY).Value()
	cellZVal := e.Property(propCellZ).Value()
	offsetXVal := e.Property(propVecX).Value()
	offsetYVal := e.Property(propVecY).Value()
	offsetZVal := e.Property(propVecZ).Value()

	if cellXVal.Any == nil || cellYVal.Any == nil || cellZVal.Any == nil || offsetXVal.Any == nil || offsetYVal.Any == nil || offsetZVal.Any == nil {
		return r3.Vector{} // CS2 POV demos
	}

	cellX := cellXVal.UInt64()
	cellY := cellYVal.UInt64()
	cellZ := cellZVal.UInt64()
	offsetX := offsetXVal.Float()
	offsetY := offsetYVal.Float()
	offsetZ := offsetZVal.Float()

	return r3.Vector{
		X: coordFromCell(cellX, offsetX),
		Y: coordFromCell(cellY, offsetY),
		Z: coordFromCell(cellZ, offsetZ),
	}
}

// OnPositionUpdate registers a handler for the entity's position update.
// The handler is called with the new position every time a position-relevant property is updated.
//
// See also Position()
func (e *Entity) OnPositionUpdate(h func(pos r3.Vector)) {
	// Only the user's handler is guarded, firing it is library code.
	e.onPositionUpdate(func(pos r3.Vector) {
		defer panics.GuardUser()

		h(pos)
	})
}

// onPositionUpdate is OnPositionUpdate without tagging the handler's panics as user-code panics.
func (e *Entity) onPositionUpdate(h func(pos r3.Vector)) {
	pos := new(r3.Vector)
	firePosUpdate := func(PropertyValue) {
		newPos := e.Position()
		if *pos != newPos {
			h(newPos)
			*pos = newPos
		}
	}

	e.Property(propCellX).onUpdate(firePosUpdate)
	e.Property(propCellY).onUpdate(firePosUpdate)
	e.Property(propCellZ).onUpdate(firePosUpdate)
	e.Property(propVecX).onUpdate(firePosUpdate)
	e.Property(propVecY).onUpdate(firePosUpdate)
	e.Property(propVecZ).onUpdate(firePosUpdate)
}

// OnDestroy registers a function to be called on the entity's destruction.
func (e *Entity) OnDestroy(delegate func()) {
	e.addOnDestroy(guardDelegate(delegate))
}

// guardDelegate tags the panics of a user's delegate as user-code panics.
func guardDelegate(delegate func()) func() {
	return func() {
		defer panics.GuardUser()

		delegate()
	}
}

// addOnDestroy is OnDestroy without tagging the delegate's panics as user-code panics.
func (e *Entity) addOnDestroy(delegate func()) {
	e.onDestroy = append(e.onDestroy, delegate)
}

// Replaced returns true if the entity was destroyed because the server created a new entity (new serial) at its index
// without an explicit delete for it - usually because entities are re-created on a round restart or backup restore.
// It's also set if an entity that left the PVS (see OnLeavePVS) is re-created at its index with the same serial -
// usually how entities re-enter the PVS. The new object shows up via ServerClass.OnEntityCreated().
// The entity may or may not have gone away in-game the usual way (e.g. a flashbang may not have detonated),
// so OnDestroy handlers can use this to skip events derived from a regular deletion.
func (e *Entity) Replaced() bool {
	return e.replaced
}

// destroy marks the entity as inactive and triggers all via OnDestroy() registered functions.
func (e *Entity) destroy() {
	e.active = false
	e.inPVS = false

	for _, delegate := range e.onDestroy {
		delegate()
	}
}

// InPVS returns false after the entity left the snapshot (its PVS on POV demos) without being deleted
// (see OnLeavePVS) and after its destruction, otherwise true - i.e. it's true unless a leave op was seen.
// On POV demos the server sometimes re-creates entities (same serial) without a leave op first,
// the old object then stays InPVS() == true although it isn't updated anymore.
func (e *Entity) InPVS() bool {
	return e.inPVS
}

// OnLeavePVS registers a function to be called when the entity leaves the snapshot (the PVS) without being deleted.
// The entity isn't destroyed yet, its properties keep their last values.
// Afterwards exactly one of OnEnterPVS (it comes back with an update - rare) or OnDestroy (it's deleted, or
// re-created at its index as a new object, see Replaced(), which shows up via ServerClass.OnEntityCreated())
// follows, unless the demo ends first.
// It isn't called when the entity is deleted, only OnDestroy is.
//
// Like OnDestroy, it's called while the entity packet is parsed: other entities of the packet may not be updated yet.
func (e *Entity) OnLeavePVS(delegate func()) {
	e.onLeavePVS = append(e.onLeavePVS, guardDelegate(delegate))
}

// leavePVS marks the entity as not in the PVS and triggers all via OnLeavePVS() registered functions.
func (e *Entity) leavePVS() {
	e.inPVS = false

	for _, delegate := range e.onLeavePVS {
		delegate()
	}
}

// OnEnterPVS registers a function to be called when an entity that left the snapshot (see OnLeavePVS)
// comes back with an update. This is rare: entities usually come back as a new object, see OnLeavePVS.
// It's called after the update's property update handlers and OnUpdateFinished delegates,
// while the entity packet is parsed: entities later in the packet aren't updated yet.
func (e *Entity) OnEnterPVS(delegate func()) {
	e.onEnterPVS = append(e.onEnterPVS, guardDelegate(delegate))
}

func (e *Entity) fireEnterPVS() {
	for _, delegate := range e.onEnterPVS {
		delegate()
	}
}

// OnUpdateFinished registers a function to be called once all property updates of the entity
// in an entity packet have been applied and their update handlers fired.
//
// Update handlers fire per property in field-path order while the entity's data is read, so a
// handler can see some of the packet's updates of the entity but not others yet.
// Use this to act on the entity's consistent state.
//
// For updated entities it is called right after the entity's data is read - before entities
// later in the packet are read and before the parser's entity handlers run for this entity.
// For created entities it is called after the initial firing of all update handlers, which
// happens after the parser's entity handlers. It isn't called when an entity leaves the PVS
// or is deleted, for skipped updates, or if the packet fails to parse.
//
// The parser of package demoinfocs applies player controllers' changes (e.g. Player.Name, SteamID64, Team, IsConnected)
// here too, so inside a controller's own update handlers the bound Player isn't updated yet.
func (e *Entity) OnUpdateFinished(delegate func()) {
	e.addOnUpdateFinished(guardDelegate(delegate))
}

// addOnUpdateFinished is OnUpdateFinished without tagging the delegate's panics as user-code panics.
func (e *Entity) addOnUpdateFinished(delegate func()) {
	e.onUpdateFinished = append(e.onUpdateFinished, delegate)
}

func (e *Entity) fireUpdateFinished() {
	for _, f := range e.onUpdateFinished {
		f()
	}
}

// OnCreateFinished registers a function to be called once the entity is fully created -
// i.e. once all property updates have been sent out.
func (e *Entity) OnCreateFinished(delegate func()) {
	e.addOnCreateFinished(guardDelegate(delegate))
}

// addOnCreateFinished is OnCreateFinished without tagging the delegate's panics as user-code panics.
func (e *Entity) addOnCreateFinished(delegate func()) {
	e.onCreateFinished = append(e.onCreateFinished, delegate)
}

// newEntity returns a new entity for the given index, serial and class
func newEntity(index, serial int32, class *ServerClass) *Entity {
	// Only allocate the per-entity polySerializers slice for classes that can
	// reach polymorphic pointer fields. A nil slice keeps entities of all other
	// classes on the shared fast paths (cached field-name checks and the
	// class-level field-path-name caches).
	var polySerializers []*serializer
	if class.polyCount > 0 {
		polySerializers = make([]*serializer, class.polyCount)
	}

	return &Entity{
		index:            index,
		serial:           serial,
		class:            class,
		active:           true,
		inPVS:            true,
		state:            &fieldState{state: make([]any, 0, entityInitialStateSlots)},
		fpCache:          make(map[string]*fieldPath),
		fpNoop:           make(map[string]bool),
		polySerializers:  polySerializers,
		onCreateFinished: nil,
		onDestroy:        nil,
		updateHandlers:   make(map[string][]PropertyUpdateHandler),
		propCache:        map[string]*Property{},
	}
}

// String returns a human identifiable string for the Entity
func (e *Entity) String() string {
	paths := e.class.getFieldPaths(newFieldPath(), e.state, e.polySerializers)
	props := make([]string, 0, len(paths))

	for _, fp := range paths {
		props = append(props, fmt.Sprintf("%s: %v", e.class.getNameForFieldPath(fp, e.polySerializers), e.state.get(fp)))
	}

	return fmt.Sprintf("%d <%s>\n %s", e.index, e.class.name, strings.Join(props, "\n "))
}

// get returns the current value of the Entity state for the given key
func (e *Entity) get(name string) any {
	if fp, ok := e.fpCache[name]; ok {
		return e.state.get(fp)
	}

	if e.fpNoop[name] {
		return nil
	}

	fp := newFieldPath()
	if !e.class.getFieldPathForName(fp, name, e.polySerializers) {
		e.fpNoop[name] = true

		fp.release()

		return nil
	}

	e.fpCache[name] = fp

	return e.state.get(fp)
}

//nolint:gocognit,funlen
func (e *Entity) readFields(r *reader, paths *[]*fieldPath, budget *entityBudget) {
	if len(e.polySerializers) < e.class.polyCount {
		e.growPolySerializers(budget)
	}

	n := readFieldPaths(r, paths)

	for _, fp := range (*paths)[:n] {
		decoder, updateCollection := e.class.serializer.getDecoderAndCollection(fp, 0, e.polySerializers)

		if decoder == nil {
			// A sub-field update under a polymorphic pointer that has no active
			// serializer. Field paths are emitted in ascending order, so the
			// pointer's own update (which selects the type) always precedes its
			// sub-fields within an entity's stream — reaching this means the
			// tracked per-entity type has desynced from the encoder's. Fail
			// loudly instead of skipping the value bits and misreading the
			// rest of the packet.
			_panicf("no serializer for polymorphic pointer at field path %v: corrupt or desynced demo bitstream", fp.path[:fp.last+1])
		}

		val := decoder(r)

		// Intercept polymorphic pointer base updates: store the newly selected
		// serializer per entity and convert to a bool for entity state storage.
		if pu, ok := val.(*polyUpdate); ok {
			e.applyPolyUpdate(pu)

			val = pu.ser != nil
		}

		if updateCollection { //nolint:nestif
			newLen := val.(uint64)
			if newLen > maxFieldIndex {
				_panicf("variable collection size %d out of range [0, %d] (path %v): corrupt or desynced demo bitstream", newLen, maxFieldIndex, fp.path[:fp.last+1])
			}

			// Retrieve the *fieldState pointer stored on the first update.
			// We store a pointer so we can resize in place on subsequent updates
			// without allocating a new fieldState each time.
			fs, _ := e.state.get(fp).(*fieldState)

			if fs == nil {
				// First update: allocate once and store the pointer.
				// Without headroom, growing reallocates with doubling capacity (see below).
				initCap := max(int(newLen), 8)

				budget.spendState(e, initCap, 0, fp)

				fs = &fieldState{state: make([]any, newLen, initCap)}

				// fs is stored: there's no *fieldState at fp yet
				if allocated, freed := e.state.set(fp, fs); allocated > 0 {
					budget.spendState(e, allocated, freed, fp)
				}
			} else {
				// Subsequent updates: resize the existing slice in place.
				curLen := uint64(len(fs.state))
				if newLen < curLen {
					// the nested field states of the removed elements are dropped
					budget.release(e, slotsOf(fs.state[newLen:curLen]))

					clear(fs.state[newLen:curLen])
					fs.state = fs.state[:newLen]
				} else if newLen > curLen {
					if newLen <= uint64(cap(fs.state)) {
						fs.state = fs.state[:newLen]
					} else {
						// Exponential growth to avoid repeated reallocations.
						newCap := grownCap(cap(fs.state), int(newLen))

						budget.spendState(e, newCap, cap(fs.state), fp)

						newState := make([]any, newLen, newCap)
						copy(newState, fs.state)
						fs.state = newState
					}
				}
			}

			val = fs.state
		} else if allocated, freed := e.state.set(fp, val); allocated > 0 {
			budget.spendState(e, allocated, freed, fp)
		}

		if e.hasHandlers {
			e.dispatchUpdate(fp, val)
		}
	}
}

// growPolySerializers grows e.polySerializers to the polyCount of its class,
// which grows if the class's serializer is replaced by one with more polymorphic fields (see Parser.ParsePacket()).
// The live budget is charged for the added slots, the allocated budget for the new slice.
func (e *Entity) growPolySerializers(budget *entityBudget) {
	allocated, freed := polySerializersSlots(e.class.polyCount), polySerializersSlots(len(e.polySerializers))

	if exceeded := budget.spendSlots(e, allocated, freed); exceeded != "" {
		_panicf("entities exceed the budget of %s per byte of entity data (class %d): corrupt demo", exceeded, e.class.classId)
	}

	ps := make([]*serializer, e.class.polyCount)
	copy(ps, e.polySerializers)
	e.polySerializers = ps
}

// dispatchUpdate fires any registered update handlers for the given field path.
// Uses handlersByFP (uint64 key) when available, falling back to the string map.
func (e *Entity) dispatchUpdate(fp *fieldPath, val any) {
	if e.handlersByFP != nil {
		if key, ok := fpFlatKey(fp); ok {
			for _, h := range e.handlersByFP[key] {
				h(PropertyValue{Any: val})
			}

			return
		}
	}
	// Fallback: deep/large path — look up by name
	name := e.class.getNameForFieldPath(fp, e.polySerializers)
	for _, h := range e.updateHandlers[name] {
		h(PropertyValue{Any: val})
	}
}

// entityBudget bounds entity decoding relative to the size of the entity data read so far
// (the bytes of the PacketEntities messages, see credit()),
// so a few bits of corrupt or crafted entity data can't make the parser allocate GBs or run for minutes.
// It has two parts, each bounded by base + perByte * (entity data bytes):
//
//   - live: the memory of the entities in Parser.entities, in state slots (16 bytes):
//     the capacity of their state (see fieldState.slots()), plus entityFixedSlots and the polySerializers of each entity.
//     Slices that are replaced by larger ones and nested states that are dropped (shrinking collections) are credited back,
//     so an entity's charge (Entity.budgetSlots) is what it holds.
//     It's refunded when an entity is dropped from Parser.entities (a new entity is created at its index),
//     so entity churn doesn't add up over long demos. Destroyed entities keep their state and stay in
//     Parser.entities until their index is reused, so they're still charged.
//   - work: what was decoded and allocated in total, never refunded - the instance baselines (in bytes)
//     and the allocated slots (the capacity of every allocation charged to live). It bounds the CPU time:
//     re-creating an entity at the same index costs a few bytes of entity data,
//     but can decode a large baseline and allocate a lot of state each time.
//
// The limits are far above what real demos need, see the constants.
type entityBudget struct {
	liveSlots      int64
	allocatedSlots int64
	baselineBytes  int64

	maxLiveSlots      int64
	maxAllocatedSlots int64
	maxBaselineBytes  int64
}

// The limits, measured across 50 real demos (2023-2026, incl. POV, deathmatch and long matches) at every point of the parse.
// The ratios peak at the first entity message (the full snapshot of ~6 KB of entity data), the base covers that:
//   - live: at most 465,768 slots (~7 MiB), at most 7.6 slots per byte (1.3 after 256 KB of entity data).
//   - allocated: at most 10.1 slots per byte (2.4 after 256 KB), 15.2M slots in total (6381816.dem).
//   - baselines: at most 6.3 bytes per byte (1.4 after 256 KB), 8.7 MB in total.
//
// With these constants, every demo stays at least 16.1x (live), 21.1x (allocated) and 33.5x (baselines)
// below the limit at every point of the parse.
const (
	entityBudgetBaseLiveSlots         = 1 << 20 // 16 MiB
	entityBudgetLiveSlotsPerByte      = 64
	entityBudgetBaseAllocatedSlots    = 1 << 20 // 16 MiB
	entityBudgetAllocatedSlotsPerByte = 128
	entityBudgetBaseBaselineBytes     = 4 << 20
	entityBudgetBaselineBytesPerByte  = 64

	// entityFixedSlots is charged for each entity (the Entity, its initial state and maps), in state slots.
	entityFixedSlots = 64

	// entityInitialStateSlots is the capacity of a new entity's state, part of entityFixedSlots.
	// It's credited back when the state grows (see fieldState.set()).
	entityInitialStateSlots = 16
)

func newEntityBudget() entityBudget {
	return entityBudget{
		maxLiveSlots:      entityBudgetBaseLiveSlots,
		maxAllocatedSlots: entityBudgetBaseAllocatedSlots,
		maxBaselineBytes:  entityBudgetBaseBaselineBytes,
	}
}

// credit adds the budget for n bytes of entity data.
func (b *entityBudget) credit(n int) {
	b.maxLiveSlots += int64(n) * entityBudgetLiveSlotsPerByte
	b.maxAllocatedSlots += int64(n) * entityBudgetAllocatedSlotsPerByte
	b.maxBaselineBytes += int64(n) * entityBudgetBaselineBytesPerByte
}

// spendState records that allocated state slots were allocated (or are about to be) for e's update at fp,
// replacing slices with freed slots.
func (b *entityBudget) spendState(e *Entity, allocated, freed int, fp *fieldPath) {
	if exceeded := b.spendSlots(e, allocated, freed); exceeded != "" {
		_panicf("entity state exceeds the budget of %s per byte of entity data (path %v): corrupt or desynced demo bitstream",
			exceeded, fp.path[:fp.last+1])
	}
}

// spendEntity records the allocation of the entity e (see entityFixedSlots), incl. its polySerializers of the given length.
func (b *entityBudget) spendEntity(e *Entity, polySerializers int) {
	if exceeded := b.spendSlots(e, entityFixedSlots+polySerializersSlots(polySerializers), 0); exceeded != "" {
		_panicf("entities exceed the budget of %s per byte of entity data (class %d): corrupt demo", exceeded, e.class.classId)
	}
}

// polySerializersSlots returns the size of a polySerializers slice of length n in state slots (rounded up).
func polySerializersSlots(n int) int {
	return (n + 1) / 2
}

// spendSlots charges the allocated slots to the allocated slots,
// and to e (live) without the freed slots of the slices they replace.
// It returns which of them exceeds the budget, or "".
func (b *entityBudget) spendSlots(e *Entity, allocated, freed int) (exceeded string) {
	e.budgetSlots += int64(allocated - freed)
	b.liveSlots += int64(allocated - freed)
	b.allocatedSlots += int64(allocated)

	switch {
	case b.liveSlots > b.maxLiveSlots:
		return "live slots"
	case b.allocatedSlots > b.maxAllocatedSlots:
		return "allocated slots"
	}

	return ""
}

// release credits n live slots of e that were freed.
func (b *entityBudget) release(e *Entity, n int) {
	e.budgetSlots -= int64(n)
	b.liveSlots -= int64(n)
}

// refund releases the live slots of e, which was dropped from Parser.entities.
func (b *entityBudget) refund(e *Entity) {
	b.liveSlots -= e.budgetSlots
	e.budgetSlots = 0
}

// spendBaseline records that a baseline of n bytes is decoded.
func (b *entityBudget) spendBaseline(n int, classID int32) {
	b.baselineBytes += int64(n)

	if b.baselineBytes > b.maxBaselineBytes {
		_panicf("decoded instance baselines exceed the budget of %d bytes per byte of entity data (class %d): corrupt demo",
			entityBudgetBaselineBytesPerByte, classID)
	}
}

// OnPacketEntities parses a CSVCMsg_PacketEntities message, applying its entity updates.
//
// It returns an error if the message couldn't be decoded (e.g. corrupt data or a missing property,
// see PropertyNotFoundError). Panics of user callbacks (e.g. OnEntityCreated handlers) propagate,
// also if they are a PropertyNotFoundError of the callback's own lookups.
// The entities may be partially updated in both cases.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
//
//nolint:gocognit,funlen
func (p *Parser) OnPacketEntities(m *msg.CSVCMsg_PacketEntities) (err error) {
	defer panics.RecoverDecode(&err)

	r := newReader(m.GetEntityData())
	defer r.release()

	var (
		index   = int32(-1)
		updates = int(m.GetUpdatedEntries())
		cmd     uint32
		classID int32
		serial  int32
	)

	if !m.GetLegacyIsDelta() {
		if p.entityFullPackets > 0 {
			return nil
		}

		p.entityFullPackets++
	}

	// After the early return above, so data that's ignored doesn't add to the budget.
	p.entityBudget.credit(len(m.GetEntityData()))

	p.tuplesCache = p.tuplesCache[:0]

	for ; updates > 0; updates-- {
		var (
			e  *Entity
			op EntityOp
		)

		next := index + int32(r.readUBitVar()) + 1
		index = next

		cmd = r.readBits(2)

		if cmd&0x01 == 0 { //nolint:nestif
			if cmd&0x02 != 0 {
				classID = int32(r.readBits(p.classIdSize))
				serial = int32(r.readBits(17))
				r.readVarUint32()

				class := p.classesById[classID]
				if class == nil {
					_panicf("unable to find new class %d", classID)
				}

				// The server may reuse the index of an entity for a new one (new serial) without sending
				// a delete for the old one first (e.g. on round restarts). The old entity is gone either way -
				// destroy it so its OnDestroy handlers run and consumers don't keep using state bound to it
				// (e.g. a weapon's unique id, #601).
				// The same serial means the same entity is re-created, e.g. when it re-enters the PVS on POV demos.
				// If it left the PVS before (see OnLeavePVS), the old object is destroyed as well, as it's never updated again.
				// Same-serial re-creates of entities that are still in the PVS keep the old object untouched.
				if old := p.entities[index]; old != nil {
					if old.active && (old.serial != serial || !old.inPVS) {
						old.replaced = true
						old.destroy()

						p.tuplesCache = append(p.tuplesCache, tuple{old, EntityOpLeft | EntityOpDeleted})
					}

					p.entityBudget.refund(old)
				}

				e = newEntity(index, serial, class)
				p.entities[index] = e

				p.entityBudget.spendEntity(e, len(e.polySerializers))

				baseline := p.classBaselines[classID]

				if baseline != nil {
					// POV demos are missing some baselines?
					p.entityBudget.spendBaseline(len(baseline), classID)

					br := newReader(baseline)
					e.readFields(br, &p.pathCache, &p.entityBudget)
					br.release()
				}

				e.readFields(r, &p.pathCache, &p.entityBudget)

				// Fire created-handlers so update-handlers can be registered
				for _, h := range class.createdHandlers {
					h(e)
				}

				// Fire all post-creation actions
				for _, f := range e.onCreateFinished {
					f()
				}

				op = EntityOpCreated | EntityOpEntered
			} else {
				if m.GetHasPvsVisBitsDeprecated() > 0 && r.readBits(2)&0x01 != 0 {
					continue
				}

				e = p.entities[index]
				if e == nil {
					_panicf("unable to find existing entity %d", index)
				}

				op = EntityOpUpdated

				// The entity left the PVS or was destroyed (destroy() clears inPVS too).
				// OnEnterPVS only fires for entities that left without being deleted (OnLeavePVS fired),
				// not for destroyed entities that are re-activated.
				reentered := false

				if !e.inPVS {
					reentered = e.active
					e.active = true
					e.inPVS = true
					op |= EntityOpEntered
				}

				e.readFields(r, &p.pathCache, &p.entityBudget)
				e.fireUpdateFinished()

				if reentered {
					e.fireEnterPVS()
				}
			}
		} else {
			e = p.entities[index]
			if e == nil {
				_panicf("unable to find existing entity %d", index)
			}

			if !e.active {
				continue // entity has already been destroyed
			}

			op = EntityOpLeft

			switch {
			case cmd&0x02 != 0:
				// also if the entity already left the PVS, the delete still destroys it
				op |= EntityOpDeleted

				e.destroy()
			case !e.inPVS:
				continue // entity has already left the PVS
			default:
				// the entity isn't destroyed, it may re-enter the PVS with an update (see OnEnterPVS)
				e.leavePVS()
			}
		}

		p.tuplesCache = append(p.tuplesCache, tuple{e, op})
	}

	for _, t := range p.tuplesCache {
		e := t.ent

		for _, h := range p.entityHandlers {
			if err := h(e, t.op); err != nil {
				return err
			}
		}

		if t.op&EntityOpCreated != 0 {
			e.fireInitialUpdateHandlers()
			e.fireUpdateFinished()
		}
	}

	if r.remBytes() > 1 || r.bitCount > 7 { //nolint:revive,staticcheck
		//nolint:godox
		// FIXME: maybe we should panic("didn't consume all data")
	}

	return nil
}

// OnEntity registers an EntityHandler that will be called when an entity
// is created, updated, deleted, etc.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
func (p *Parser) OnEntity(h EntityHandler) {
	p.entityHandlers = append(p.entityHandlers, h)
}
