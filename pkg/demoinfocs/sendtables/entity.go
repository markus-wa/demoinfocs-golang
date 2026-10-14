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

	onCreateFinished []func()
	onUpdateFinished []func()
	onDestroy        []func()
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
	// The bound handler is library code, its panics aren't tagged as user-code panics.
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
	e.Property(prop).onUpdate(guardUpdateHandler(bindFactoryByType[t](variable)))
}

// bindProperty is BindProperty without tagging the binding's panics as user-code panics.
func (e *Entity) bindProperty(prop string, variable any, t PropertyValueType) {
	e.Property(prop).Bind(variable, t)
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
// The entity may or may not have gone away in-game the usual way (e.g. a flashbang may not have detonated),
// so OnDestroy handlers can use this to skip events derived from a regular deletion.
func (e *Entity) Replaced() bool {
	return e.replaced
}

// destroy marks the entity as inactive and triggers all via OnDestroy() registered functions.
func (e *Entity) destroy() {
	e.active = false

	for _, delegate := range e.onDestroy {
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
		state:            &fieldState{state: make([]any, 0, 16)},
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
func (e *Entity) readFields(r *reader, paths *[]*fieldPath) {
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
				// Use 2× initial capacity so small incremental growths don't reallocate.
				initCap := newLen * 2
				if initCap < 8 {
					initCap = 8
				}

				fs = &fieldState{state: make([]any, newLen, initCap)}
				e.state.set(fp, fs)
			} else {
				// Subsequent updates: resize the existing slice in place.
				curLen := uint64(len(fs.state))
				if newLen < curLen {
					clear(fs.state[newLen:curLen])
					fs.state = fs.state[:newLen]
				} else if newLen > curLen {
					if newLen <= uint64(cap(fs.state)) {
						fs.state = fs.state[:newLen]
					} else {
						// Exponential growth to avoid repeated reallocations.
						newCap := uint64(cap(fs.state)) * 2
						if newCap < newLen {
							newCap = newLen
						}

						newState := make([]any, newLen, newCap)
						copy(newState, fs.state)
						fs.state = newState
					}
				}
			}

			val = fs.state
		} else {
			e.state.set(fp, val)
		}

		if e.hasHandlers {
			e.dispatchUpdate(fp, val)
		}
	}
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

// OnPacketEntities parses a CSVCMsg_PacketEntities message, applying its entity updates.
//
// If decoding fails (e.g. corrupt data or a missing property, see PropertyNotFoundError), the error is returned
// as a *panics.Failure. Panics of user callbacks (see panics.Classify()) are re-panicked as *panics.Captured.
// The entities may be partially updated in both cases.
//
// Used internally by package demoinfocs; not covered by API stability guarantees.
//
//nolint:gocognit,funlen
func (p *Parser) OnPacketEntities(m *msg.CSVCMsg_PacketEntities) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		if panics.Classify[*PropertyNotFoundError](r) == panics.KindUser {
			panic(panics.Capture(r))
		}

		err = panics.NewFailure(r)
	}()

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
				if old := p.entities[index]; old != nil && old.active && old.serial != serial {
					old.replaced = true
					old.destroy()

					p.tuplesCache = append(p.tuplesCache, tuple{old, EntityOpLeft | EntityOpDeleted})
				}

				e = newEntity(index, serial, class)
				p.entities[index] = e

				baseline := p.classBaselines[classID]

				if baseline != nil {
					// POV demos are missing some baselines?
					br := newReader(baseline)
					e.readFields(br, &p.pathCache)
					br.release()
				}

				e.readFields(r, &p.pathCache)

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

				if !e.active {
					e.active = true
					op |= EntityOpEntered
				}

				e.readFields(r, &p.pathCache)
				e.fireUpdateFinished()
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
			if cmd&0x02 != 0 {
				op |= EntityOpDeleted

				e.destroy()
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
