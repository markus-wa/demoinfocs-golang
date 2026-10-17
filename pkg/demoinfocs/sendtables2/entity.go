package sendtables2

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/golang/geo/r3"
	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// Entity represents a single game entity in the replay
type Entity struct {
	index  int32
	serial int32
	class  *class
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
	updateHandlers   map[string][]st.PropertyUpdateHandler
	// handlerLog preserves the global registration order of update handlers.
	// updateHandlers remains the name-keyed index for lookup; this log is the
	// single source of truth for dispatch ordering (handlersByFP rebuilds and
	// the initial entity-creation dispatch). Handlers are never unregistered,
	// so the log is append-only and never stale.
	handlerLog   []handlerRegistration
	handlersByFP map[uint64][]st.PropertyUpdateHandler
	hasHandlers  bool // cached: len(updateHandlers) > 0
	propCache    map[string]*property
	// cacheGen is bumped whenever the per-entity caches are invalidated (see applyPolyUpdate), so that
	// properties handed out before can tell their cached field path is stale.
	cacheGen uint32
	// positionProps caches the six properties Position() reads, nil until resolved.
	positionProps *[6]*property
}

// handlerRegistration records a single update-handler registration so that
// dispatch order can follow registration order instead of map-iteration order.
type handlerRegistration struct {
	name    string
	handler st.PropertyUpdateHandler
}

func (e *Entity) ServerClass() st.ServerClass {
	return e.class
}

func (e *Entity) ID() int {
	return int(e.index)
}

func (e *Entity) SerialNum() int {
	return int(e.serial)
}

func (e *Entity) Properties() (out []st.Property) {
	for _, fp := range e.class.getFieldPaths(newFieldPath(), e.state, e.polySerializers) {
		prop := e.Property(e.class.getNameForFieldPath(fp, e.polySerializers))
		if prop == nil {
			continue // a generated name must always resolve; don't hand out nil properties if it doesn't
		}

		out = append(out, prop)
	}

	return
}

type property struct {
	entity *Entity
	name   string
	// fp is the field path resolved for name as of gen, see fieldPath().
	fp  *fieldPath
	gen uint32
}

func (p *property) Name() string {
	return p.name
}

// fieldPath returns the property's field path (nil if the property doesn't exist for the entity's active
// type), cached on the property so Value() doesn't look it up by name on every call.
func (p *property) fieldPath() *fieldPath {
	if p.fp == nil || p.gen != p.entity.cacheGen {
		p.fp = p.entity.fieldPathForName(p.name)
		p.gen = p.entity.cacheGen
	}

	return p.fp
}

func (p *property) Value() st.PropertyValue {
	var v any
	if fp := p.fieldPath(); fp != nil {
		v = p.entity.state.get(fp)
	}

	fs, ok := v.(*fieldState)
	if ok {
		v = fs.state
	}

	return st.PropertyValue{
		VectorVal: r3.Vector{},
		IntVal:    0,
		Int64Val:  0,
		ArrayVal:  nil,
		StringVal: "",
		FloatVal:  0,
		Any:       v,
		S2:        true,
	}
}

func (p *property) Type() st.PropertyType {
	panic("not implemented")
}

func (p *property) ArrayElementType() st.PropertyType {
	panic("not implemented")
}

func (p *property) OnUpdate(handler st.PropertyUpdateHandler) {
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], handler)
	p.entity.handlerLog = append(p.entity.handlerLog, handlerRegistration{name: p.name, handler: handler})
	p.entity.addHandlerByFP(p.name, handler)
	p.entity.hasHandlers = true
}

// addHandlerByFP registers handler in the fast field-path-keyed lookup table.
// Called alongside every updateHandlers insertion so the two maps stay in sync.
func (e *Entity) addHandlerByFP(name string, handler st.PropertyUpdateHandler) {
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
		e.handlersByFP = make(map[uint64][]st.PropertyUpdateHandler)
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
	e.cacheGen++
	e.positionProps = nil

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

	e.handlersByFP = make(map[uint64][]st.PropertyUpdateHandler, len(e.updateHandlers))

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

type bindFactory func(variable any) st.PropertyUpdateHandler

var bindFactoryByType = map[st.PropertyValueType]bindFactory{
	st.ValTypeVector: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*r3.Vector) = v.R3Vec()
		}
	},
	st.ValTypeInt: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*int) = v.Int()
		}
	},
	st.ValTypeArray: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*[]st.PropertyValue) = v.ArrayVal
		}
	},
	st.ValTypeString: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*string) = v.String()
		}
	},
	st.ValTypeBoolInt: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*bool) = v.BoolVal()
		}
	},
	st.ValTypeFloat32: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*float32) = v.Float()
		}
	},
	st.ValTypeFloat64: func(variable any) st.PropertyUpdateHandler {
		return func(v st.PropertyValue) {
			*variable.(*float64) = float64(v.Float())
		}
	},
}

func (p *property) Bind(variable any, t st.PropertyValueType) {
	h := bindFactoryByType[t](variable)
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], h)
	p.entity.handlerLog = append(p.entity.handlerLog, handlerRegistration{name: p.name, handler: h})
	p.entity.addHandlerByFP(p.name, h)
	p.entity.hasHandlers = true
}

func (e *Entity) Property(name string) st.Property {
	if prop := e.property(name); prop != nil {
		return prop
	}

	return nil
}

// property is Property() returning the concrete type, nil if the entity has no property of that name.
func (e *Entity) property(name string) *property {
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

	prop := &property{
		entity: e,
		name:   name,
	}
	e.propCache[name] = prop

	return prop
}

func (e *Entity) BindProperty(prop string, variable any, t st.PropertyValueType) {
	e.Property(prop).Bind(variable, t)
}

func (e *Entity) PropertyValue(name string) (st.PropertyValue, bool) {
	prop := e.Property(name)
	if prop == nil {
		return st.PropertyValue{S2: true}, false
	}

	v := prop.Value()

	return v, true
}

func (e *Entity) PropertyValueMust(name string) st.PropertyValue {
	val, ok := e.PropertyValue(name)
	if !ok {
		panic(fmt.Sprintf("property '%s' not found", name))
	}

	return val
}

func (e *Entity) ApplyUpdate(reader *bit.BitReader) {
	panic("not implemented")
}

// CS2 networks an entity's position through its body component. The cell and
// offset fields sit under a send-node group whose name depends on the
// body-component type: m_skeletonInstance.m_vecOrigin for animated entities
// (CBodyComponentBaseAnimGraph, CBodyComponentBaseModelEntity) and
// m_sceneNode.m_vecOrigin for point entities (CBodyComponentPoint).
const (
	propCellX = "CBodyComponent.m_skeletonInstance.m_vecOrigin.m_cellX"
	propCellY = "CBodyComponent.m_skeletonInstance.m_vecOrigin.m_cellY"
	propCellZ = "CBodyComponent.m_skeletonInstance.m_vecOrigin.m_cellZ"
	propVecX  = "CBodyComponent.m_skeletonInstance.m_vecOrigin.m_vecX"
	propVecY  = "CBodyComponent.m_skeletonInstance.m_vecOrigin.m_vecY"
	propVecZ  = "CBodyComponent.m_skeletonInstance.m_vecOrigin.m_vecZ"

	propCellXPoint = "CBodyComponent.m_sceneNode.m_vecOrigin.m_cellX"
	propCellYPoint = "CBodyComponent.m_sceneNode.m_vecOrigin.m_cellY"
	propCellZPoint = "CBodyComponent.m_sceneNode.m_vecOrigin.m_cellZ"
	propVecXPoint  = "CBodyComponent.m_sceneNode.m_vecOrigin.m_vecX"
	propVecYPoint  = "CBodyComponent.m_sceneNode.m_vecOrigin.m_vecY"
	propVecZPoint  = "CBodyComponent.m_sceneNode.m_vecOrigin.m_vecZ"
)

var positionPropPairs = [...][2]string{
	{propCellX, propCellXPoint},
	{propCellY, propCellYPoint},
	{propCellZ, propCellZPoint},
	{propVecX, propVecXPoint},
	{propVecY, propVecYPoint},
	{propVecZ, propVecZPoint},
}

// positionProp returns the position-related property for this entity's
// body-component type, or nil if the entity has neither variant.
func (e *Entity) positionProp(skeleton string, point string) *property {
	if prop := e.property(skeleton); prop != nil {
		return prop
	}

	return e.property(point)
}

// resolvePositionProps returns the six properties Position() reads, nil if the entity has no body component.
// They are resolved once per entity, Position() is called a lot (e.g. for every player on every frame).
func (e *Entity) resolvePositionProps() *[6]*property {
	if e.positionProps != nil {
		return e.positionProps
	}

	var props [6]*property

	for i, names := range positionPropPairs {
		prop := e.positionProp(names[0], names[1])
		if prop == nil {
			return nil
		}

		props[i] = prop
	}

	e.positionProps = &props

	return e.positionProps
}

// Returns a coordinate from a cell + offset
func coordFromCell(cell uint64, offset float32) float64 {
	const (
		cellBits    = 9
		maxCoordInt = 16384
	)

	cellCoord := float64(cell)*float64(1<<cellBits) - maxCoordInt

	return cellCoord + float64(offset)
}

func (e *Entity) Position() r3.Vector {
	props := e.resolvePositionProps()
	if props == nil {
		return r3.Vector{} // entity without a body component
	}

	cellXVal := props[0].Value()
	cellYVal := props[1].Value()
	cellZVal := props[2].Value()
	offsetXVal := props[3].Value()
	offsetYVal := props[4].Value()
	offsetZVal := props[5].Value()

	if cellXVal.Any == nil || cellYVal.Any == nil || cellZVal.Any == nil || offsetXVal.Any == nil || offsetYVal.Any == nil || offsetZVal.Any == nil {
		return r3.Vector{} // CS2 POV demos
	}

	cellX := cellXVal.S2UInt64()
	cellY := cellYVal.S2UInt64()
	cellZ := cellZVal.S2UInt64()
	offsetX := offsetXVal.Float()
	offsetY := offsetYVal.Float()
	offsetZ := offsetZVal.Float()

	return r3.Vector{
		X: coordFromCell(cellX, offsetX),
		Y: coordFromCell(cellY, offsetY),
		Z: coordFromCell(cellZ, offsetZ),
	}
}

func (e *Entity) OnPositionUpdate(h func(pos r3.Vector)) {
	pos := new(r3.Vector)
	firePosUpdate := func(st.PropertyValue) {
		newPos := e.Position()
		if *pos != newPos {
			h(newPos)
			*pos = newPos
		}
	}

	for _, names := range positionPropPairs {
		if prop := e.positionProp(names[0], names[1]); prop != nil {
			prop.OnUpdate(firePosUpdate)
		}
	}
}

func (e *Entity) OnDestroy(delegate func()) {
	e.onDestroy = append(e.onDestroy, delegate)
}

// Replaced returns true if the entity was destroyed because the server created a new entity (new serial) at its index
// without an explicit delete for it - usually because entities are re-created on a round restart or backup restore.
// The entity may or may not have gone away in-game the usual way (e.g. a flashbang may not have detonated),
// so OnDestroy handlers can use this to skip events derived from a regular deletion.
func (e *Entity) Replaced() bool {
	return e.replaced
}

// Destroy marks the entity as inactive and fires all OnDestroy handlers.
func (e *Entity) Destroy() {
	e.active = false

	for _, delegate := range e.onDestroy {
		delegate()
	}
}

// OnUpdateFinished registers a function that is called once all property updates of the entity in an
// entity packet have been applied and their update handlers fired - including the initial firing of
// all update handlers after the entity was created.
//
// Update handlers fire per property in field-path order while the entity's data is read, so a handler
// can see some of the packet's updates of the entity but not others. Use this to act on a consistent state.
func (e *Entity) OnUpdateFinished(delegate func()) {
	e.onUpdateFinished = append(e.onUpdateFinished, delegate)
}

func (e *Entity) fireUpdateFinished() {
	for _, f := range e.onUpdateFinished {
		f()
	}
}

// OnCreateFinished registers a handler that is called after entity creation is complete.
func (e *Entity) OnCreateFinished(delegate func()) {
	e.onCreateFinished = append(e.onCreateFinished, delegate)
}

// newEntity returns a new entity for the given index, serial and class
func newEntity(index, serial int32, class *class) *Entity {
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
		updateHandlers:   make(map[string][]st.PropertyUpdateHandler),
		propCache:        map[string]*property{},
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

// Map returns a map of current entity state as key-value pairs
func (e *Entity) Map() map[string]any {
	values := make(map[string]any)
	for _, fp := range e.class.getFieldPaths(newFieldPath(), e.state, e.polySerializers) {
		values[e.class.getNameForFieldPath(fp, e.polySerializers)] = e.state.get(fp)
	}

	return values
}

// Get returns the current value of the Entity state for the given key
func (e *Entity) Get(name string) any {
	fp := e.fieldPathForName(name)
	if fp == nil {
		return nil
	}

	return e.state.get(fp)
}

// fieldPathForName returns the field path of the named property for the entity's active type,
// nil if there is none. Resolutions are cached per entity until the active type changes.
func (e *Entity) fieldPathForName(name string) *fieldPath {
	if fp, ok := e.fpCache[name]; ok {
		return fp
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

	return fp
}

// Exists returns true if the given key exists in the Entity state
func (e *Entity) Exists(name string) bool {
	return e.Get(name) != nil
}

// GetInt32 gets given key as an int32
func (e *Entity) GetInt32(name string) (int32, bool) {
	x, ok := e.Get(name).(int32)
	return x, ok
}

// GetUint32 gets given key as a uint32
func (e *Entity) GetUint32(name string) (uint32, bool) {
	if v := e.Get(name); v != nil {
		switch x := v.(type) {
		case uint32:
			return x, true
		case uint64:
			return uint32(x), true
		}
	}

	return 0, false
}

// GetUint64 gets given key as a uint64
func (e *Entity) GetUint64(name string) (uint64, bool) {
	x, ok := e.Get(name).(uint64)
	return x, ok
}

// GetFloat32 gets given key as an float32
func (e *Entity) GetFloat32(name string) (float32, bool) {
	x, ok := e.Get(name).(float32)
	return x, ok
}

// GetString gets given key as a string
func (e *Entity) GetString(name string) (string, bool) {
	x, ok := e.Get(name).(string)
	return x, ok
}

// GetBool gets given key as a bool
func (e *Entity) GetBool(name string) (bool, bool) {
	x, ok := e.Get(name).(bool)
	return x, ok
}

// GetSerial return the serial of the class associated with this Entity
func (e *Entity) GetSerial() int32 {
	return e.serial
}

// GetClassId returns the id of the class associated with this Entity
func (e *Entity) GetClassId() int32 { //nolint:revive
	return e.class.classID
}

// GetClassName returns the name of the class associated with this Entity
func (e *Entity) GetClassName() string {
	return e.class.name
}

// GetIndex returns the index of this Entity
func (e *Entity) GetIndex() int32 {
	return e.index
}

// FindEntity finds a given Entity by index
func (p *Parser) FindEntity(index int32) *Entity {
	return p.entities[index]
}

func handle2idx(handle uint64) int32 {
	return int32(handle & constants.EntityHandleIndexMaskSource2)
}

func serialForHandle(handle uint64) int32 {
	return int32(handle >> constants.MaxEdictBitsSource2)
}

// FindEntityByHandle finds a given Entity by handle
func (p *Parser) FindEntityByHandle(handle uint64) *Entity {
	idx := handle2idx(handle)

	e := p.FindEntity(idx)
	if e != nil && e.GetSerial() != serialForHandle(handle) {
		return nil
	}

	return e
}

// FilterEntity finds entities by callback
func (p *Parser) FilterEntity(fb func(*Entity) bool) []*Entity {
	entities := make([]*Entity, 0)

	for _, et := range p.entities {
		if fb(et) {
			entities = append(entities, et)
		}
	}

	return entities
}

//nolint:gocognit
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
				h(st.PropertyValue{Any: val, S2: true})
			}

			return
		}
	}
	// Fallback: deep/large path — look up by name
	name := e.class.getNameForFieldPath(fp, e.polySerializers)
	for _, h := range e.updateHandlers[name] {
		h(st.PropertyValue{Any: val, S2: true})
	}
}

// Internal Callback for OnCSVCMsg_PacketEntities.
//
//nolint:gocognit,funlen
func (p *Parser) OnPacketEntities(m *msgs2.CSVCMsg_PacketEntities) error {
	defer func() {
		if p.packetEntitiesPanicWarnFunc == nil {
			return
		}

		r := recover()
		if r != nil {
			fmt.Fprintf(os.Stderr, "error in OnPacketEntities: %v\n", r)
		}
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

	p.setAlternateBaselines(m)
	p.copyEntityBaselines(m)

	for ; updates > 0; updates-- {
		var (
			e  *Entity
			op st.EntityOp
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
					old.Destroy()

					p.tuplesCache = append(p.tuplesCache, tuple{old, st.EntityOpLeft | st.EntityOpDeleted})
				}

				e = newEntity(index, serial, class)
				p.entities[index] = e

				p.readCreateBaseline(e, m)
				e.readFields(r, &p.pathCache)
				p.storeEntityBaseline(e, m)

				// Fire created-handlers so update-handlers can be registered
				for _, h := range class.createdHandlers {
					h(e)
				}

				// Fire all post-creation actions
				for _, f := range e.onCreateFinished {
					f()
				}

				op = st.EntityOpCreated | st.EntityOpEntered
			} else {
				if m.GetHasPvsVisBitsDeprecated() > 0 && r.readBits(2)&0x01 != 0 {
					continue
				}

				e = p.entities[index]
				if e == nil {
					_panicf("unable to find existing entity %d", index)
				}

				op = st.EntityOpUpdated

				if !e.active {
					e.active = true
					op |= st.EntityOpEntered
				}

				p.readAlternateBaseline(e)
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

			op = st.EntityOpLeft
			if cmd&0x02 != 0 {
				op |= st.EntityOpDeleted

				e.Destroy()
			}
		}

		p.tuplesCache = append(p.tuplesCache, tuple{e, op})
	}

	// Packets with non_transmitted_entities report deletions through it instead (not handled here).
	if m.GetLegacyIsDelta() && m.NonTransmittedEntities == nil {
		p.readExplicitDeletions(r)
	}

	for _, t := range p.tuplesCache {
		e := t.ent

		for _, h := range p.entityHandlers {
			if err := h(e, t.op); err != nil {
				return err
			}
		}

		if t.op&st.EntityOpCreated != 0 {
			e.fireInitialUpdateHandlers()
			e.fireUpdateFinished()
		}
	}

	if r.remBytes() > 1 || r.bitCount > 7 { //nolint:revive,staticcheck
		// FIXME: maybe we should panic("didn't consume all data")
	}

	return nil
}

// setAlternateBaselines collects the entities whose updates in this packet are deltas from an instance baseline
// instead of their current state (sv_enable_alternate_baselines, e.g. in POV demos).
// The engine only reads them along serialized_entities.
func (p *Parser) setAlternateBaselines(m *msgs2.CSVCMsg_PacketEntities) {
	clear(p.altBaselines)

	if m.SerializedEntities == nil {
		return
	}

	for _, ab := range m.GetAlternateBaselines() {
		p.altBaselines[ab.GetEntityIndex()] = ab.GetBaselineIndex()
	}
}

// readAlternateBaseline applies the instance baseline an entity update is a delta from, if there is one.
// Alternate baselines contain every field of the entity, so this overwrites its whole state.
func (p *Parser) readAlternateBaseline(e *Entity) {
	baselineIndex, ok := p.altBaselines[e.index]
	if !ok {
		return
	}

	baseline := p.baselinesByIndex[baselineIndex]
	if baseline == nil {
		_panicf("missing alternate baseline %d for entity %d", baselineIndex, e.index)
	}

	r := newReader(baseline)
	e.readFields(r, &p.pathCache)
	r.release()
}

// readExplicitDeletions reads the list of deleted entities that follows the entity updates of delta packets
// in demos without non_transmitted_entities (older demos and POV demos).
// It contains entities the update headers can't delete, e.g. ones that left the PVS before being deleted.
//
// Format (engine2 ReadPacketEntities): UBitVar count, then count UBitVar index deltas starting from index -1.
func (p *Parser) readExplicitDeletions(r *reader) {
	// The count takes at least 6 bits, anything shorter is padding
	if r.remBytes()*8+r.bitCount < 6 {
		return
	}

	index := int32(-1)

	for count := r.readUBitVar(); count > 0; count-- {
		delta := r.readUBitVar()
		if delta == 0 || int64(index)+int64(delta) >= 1<<constants.MaxEdictBitsSource2 {
			_panicf("invalid explicit entity deletion: delta %d after index %d", delta, index)
		}

		index += int32(delta)

		e := p.entities[index]
		if e == nil || !e.active {
			continue // usually already deleted by a header in the same packet
		}

		e.Destroy()

		p.tuplesCache = append(p.tuplesCache, tuple{e, st.EntityOpLeft | st.EntityOpDeleted})
	}
}

// entityBaseline is the state of an entity right after it was created by an update_baseline packet (POV demos).
type entityBaseline struct {
	class           *class
	state           *fieldState
	polySerializers []*serializer
}

// copyEntityBaselines starts an update of the entity baselines (update_baseline):
// the other slot becomes a copy of the current one, the entities created by the packet are then stored in it.
//
// Baselines are only ever replaced, the engine (engine2.dll CopyEntityBaseline, CopyNewEntity, ProcessBaselineAck)
// keeps the one of an entity that left the PVS or was deleted.
func (p *Parser) copyEntityBaselines(m *msgs2.CSVCMsg_PacketEntities) {
	if !m.GetUpdateBaseline() {
		return
	}

	p.entityBaselines[1-m.GetBaseline()&1] = maps.Clone(p.entityBaselines[m.GetBaseline()&1])
}

// readCreateBaseline sets the state a newly created entity's data is a delta from:
// the entity baseline at its index if there is one of its class (POV demos), otherwise its class baseline.
func (p *Parser) readCreateBaseline(e *Entity, m *msgs2.CSVCMsg_PacketEntities) {
	if m.GetLegacyIsDelta() {
		if b := p.entityBaselines[m.GetBaseline()&1][e.index]; b != nil && b.class == e.class {
			e.state = b.state.clone()
			copy(e.polySerializers, b.polySerializers)

			return
		}
	}

	baseline := p.classBaselines[e.class.classID]
	if baseline == nil {
		return // POV demos are missing some baselines?
	}

	r := newReader(baseline)
	e.readFields(r, &p.pathCache)
	r.release()
}

// storeEntityBaseline stores the state of an entity created by an update_baseline packet as the entity baseline
// at its index in the other slot, see copyEntityBaselines.
func (p *Parser) storeEntityBaseline(e *Entity, m *msgs2.CSVCMsg_PacketEntities) {
	if !m.GetUpdateBaseline() {
		return
	}

	p.entityBaselines[1-m.GetBaseline()&1][e.index] = &entityBaseline{
		class:           e.class,
		state:           e.state.clone(),
		polySerializers: slices.Clone(e.polySerializers),
	}
}

// OnEntity registers an EntityHandler that will be called when an entity
// is created, updated, deleted, etc.
func (p *Parser) OnEntity(h st.EntityHandler) {
	p.entityHandlers = append(p.entityHandlers, h)
}
