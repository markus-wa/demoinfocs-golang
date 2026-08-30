package sendtables2

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/golang/geo/r3"
	"golang.org/x/exp/maps"

	bit "github.com/markus-wa/demoinfocs-golang/v4/internal/bitread"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/constants"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/msgs2"
	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// Entity represents a single game entity in the replay
type Entity struct {
	index   int32
	serial  int32
	class   *class
	active  bool
	state   *fieldState
	fpCache map[string]*fieldPath
	fpNoop  map[string]bool
	// polySerializers holds the currently active serializer for each polymorphic
	// pointer field, indexed by field.polySerializerID. Nil entries mean the
	// pointer is inactive. This slice is per-entity so that two entities of the
	// same class can hold different active types simultaneously.
	polySerializers []*serializer

	onCreateFinished []func()
	onDestroy        []func()
	updateHandlers   map[string][]st.PropertyUpdateHandler
	// handlersByFP stores the same handlers indexed by field-path uint64 key
	// (see fpFlatKey) for O(1) non-string dispatch in the hot readFields path.
	// Only populated for paths that fit in the flat key range.
	handlersByFP map[uint64][]st.PropertyUpdateHandler
	hasHandlers  bool // cached: len(updateHandlers) > 0
	propCache    map[string]st.Property
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
}

func (p property) Name() string {
	return p.name
}

func (p property) Value() st.PropertyValue {
	v := p.entity.Get(p.name)

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

func (p property) Type() st.PropertyType {
	panic("not implemented")
}

func (p property) ArrayElementType() st.PropertyType {
	panic("not implemented")
}

func (p property) OnUpdate(handler st.PropertyUpdateHandler) {
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], handler)
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
// Names are iterated in sorted order so that handlers sharing an fp key (e.g.
// via deprecated bare-name aliases resolving to the same field) fire in a
// stable order across rebuilds, mirroring the created-handler dispatch in
// OnPacketEntities. Within a single name, registration order is preserved.
func (e *Entity) rebuildHandlersByFP() {
	if len(e.updateHandlers) == 0 {
		e.handlersByFP = nil

		return
	}

	e.handlersByFP = make(map[uint64][]st.PropertyUpdateHandler, len(e.updateHandlers))

	names := maps.Keys(e.updateHandlers)
	slices.Sort(names)

	for _, name := range names {
		for _, h := range e.updateHandlers[name] {
			e.addHandlerByFP(name, h)
		}
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

func (p property) Bind(variable any, t st.PropertyValueType) {
	h := bindFactoryByType[t](variable)
	p.entity.updateHandlers[p.name] = append(p.entity.updateHandlers[p.name], h)
	p.entity.addHandlerByFP(p.name, h)
	p.entity.hasHandlers = true
}

func (e *Entity) Property(name string) st.Property {
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

	e.propCache[name] = property{
		entity: e,
		name:   name,
	}

	return e.propCache[name]
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
func (e *Entity) positionProp(skeleton string, point string) st.Property {
	if prop := e.Property(skeleton); prop != nil {
		return prop
	}

	return e.Property(point)
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
	cellXProp := e.positionProp(propCellX, propCellXPoint)
	cellYProp := e.positionProp(propCellY, propCellYPoint)
	cellZProp := e.positionProp(propCellZ, propCellZPoint)
	offsetXProp := e.positionProp(propVecX, propVecXPoint)
	offsetYProp := e.positionProp(propVecY, propVecYPoint)
	offsetZProp := e.positionProp(propVecZ, propVecZPoint)

	if cellXProp == nil || cellYProp == nil || cellZProp == nil || offsetXProp == nil || offsetYProp == nil || offsetZProp == nil {
		return r3.Vector{} // entity without a body component
	}

	cellXVal := cellXProp.Value()
	cellYVal := cellYProp.Value()
	cellZVal := cellZProp.Value()
	offsetXVal := offsetXProp.Value()
	offsetYVal := offsetYProp.Value()
	offsetZVal := offsetZProp.Value()

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

func (e *Entity) Destroy() {
	e.active = false

	for _, delegate := range e.onDestroy {
		delegate()
	}
}

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
		propCache:        map[string]st.Property{},
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

func (e *Entity) readFields(r *reader, paths *[]*fieldPath) {
	n := readFieldPaths(r, paths)

	for _, fp := range (*paths)[:n] {
		decoder, updateCollection := e.class.serializer.getDecoderAndCollection(fp, 0, e.polySerializers)

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

				e.readFields(r, &p.pathCache)
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

	for _, t := range p.tuplesCache {
		e := t.ent

		for _, h := range p.entityHandlers {
			if err := h(e, t.op); err != nil {
				return err
			}
		}

		if t.op&st.EntityOpCreated != 0 {
			for prop, hs := range e.updateHandlers {
				v := e.PropertyValueMust(prop)

				for _, h := range hs {
					h(v)
				}
			}
		}
	}

	if r.remBytes() > 1 || r.bitCount > 7 { //nolint:revive,staticcheck
		// FIXME: maybe we should panic("didn't consume all data")
	}

	return nil
}

// OnEntity registers an EntityHandler that will be called when an entity
// is created, updated, deleted, etc.
func (p *Parser) OnEntity(h st.EntityHandler) {
	p.entityHandlers = append(p.entityHandlers, h)
}
