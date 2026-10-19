package sendtables

type fieldState struct {
	state []any
}

// newFieldStateSlots is the number of slots newFieldState() allocates.
const newFieldStateSlots = 16

func newFieldState() *fieldState {
	return &fieldState{
		state: make([]any, 8, newFieldStateSlots),
	}
}

// maxFieldIndex bounds decoded field-path components and variable-collection sizes,
// so a single update allocates at most 1 MiB of state (see also entityBudget).
// Anything larger means a corrupt or desynced bitstream.
// The largest index / size across 50 real demos is ~3k (CSmokeGrenadeProjectile.m_VoxelFrameData),
// the limit leaves room for larger voxel data.
const maxFieldIndex = 1 << 16

func validateFieldIndex(z int, fp *fieldPath) {
	if z < 0 || z > maxFieldIndex {
		_panicf("field path component %d out of range [0, %d] (path %v): corrupt or desynced demo bitstream", z, maxFieldIndex, fp.path[:fp.last+1])
	}
}

func (s *fieldState) get(fp *fieldPath) any {
	x := s
	z := 0

	for i := 0; i <= fp.last; i++ {
		z = fp.path[i]
		if len(x.state) < z+1 {
			return nil
		}

		if i == fp.last {
			return x.state[z]
		}

		if _, ok := x.state[z].(*fieldState); !ok {
			return nil
		}

		x = x.state[z].(*fieldState)
	}

	return nil
}

// set sets the value at fp.
// It returns the number of slots it allocated (incl. nested field states) and the number of slots
// of the slices it replaced (which are freed), see entityBudget.
// It never replaces nested field states: a value isn't stored where a *fieldState is already.
func (s *fieldState) set(fp *fieldPath, v any) (allocated, freed int) {
	// Fast path for the common single-level case (fp.last == 0)
	if fp.last == 0 { //nolint:nestif
		z := fp.path[0]
		validateFieldIndex(z, fp)

		if y := len(s.state); y <= z {
			if z+2 > cap(s.state) {
				newSlice := make([]any, z+1, grownCap(cap(s.state), z+2))
				copy(newSlice, s.state)
				allocated, freed = cap(newSlice), cap(s.state)
				s.state = newSlice
			} else {
				s.state = s.state[:z+1]
			}
		}

		if _, ok := s.state[z].(*fieldState); !ok {
			s.state[z] = v
		}

		return allocated, freed
	}

	x := s
	z := 0

	for i := 0; i <= fp.last; i++ {
		z = fp.path[i]
		validateFieldIndex(z, fp)

		if y := len(x.state); y <= z {
			if z+2 > cap(x.state) {
				newSlice := make([]any, z+1, grownCap(cap(x.state), z+2))
				copy(newSlice, x.state)
				allocated += cap(newSlice)
				freed += cap(x.state)
				x.state = newSlice
			} else {
				// Re-slice to update the length without allocating new memory
				x.state = x.state[:z+1]
			}
		}

		if i == fp.last {
			if _, ok := x.state[z].(*fieldState); !ok {
				x.state[z] = v
			}

			return allocated, freed
		}

		// replaces a scalar value (it has no slots of its own)
		if _, ok := x.state[z].(*fieldState); !ok {
			x.state[z] = newFieldState()
			allocated += newFieldStateSlots
		}

		x = x.state[z].(*fieldState)
	}

	return allocated, freed
}

// slots returns the number of slots of s, incl. its nested field states, see entityBudget.
func (s *fieldState) slots() int {
	n := cap(s.state)

	for _, v := range s.state {
		if nested, ok := v.(*fieldState); ok {
			n += nested.slots()
		}
	}

	return n
}

// slotsOf returns the number of slots of the field states in vs, see fieldState.slots().
func slotsOf(vs []any) (n int) {
	for _, v := range vs {
		if nested, ok := v.(*fieldState); ok {
			n += nested.slots()
		}
	}

	return n
}

// maxStateCap bounds the capacity grownCap() doubles to: no index or collection size is larger than maxFieldIndex,
// set() needs at most one more slot.
const maxStateCap = maxFieldIndex + 2

// grownCap returns the capacity of a state slice that grows from capacity old and must hold need slots.
// It doubles like append(), so slices that grow one slot at a time are reallocated O(log n) times,
// but not beyond maxStateCap (unless need is larger).
func grownCap(old, need int) int {
	return max(need, min(2*old, maxStateCap))
}

func max(a, b int) int { //nolint:revive
	if a > b {
		return a
	}

	return b
}
