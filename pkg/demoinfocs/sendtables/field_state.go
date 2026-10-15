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
// It returns the number of slots it allocated (incl. nested field states), see stateBudget.
func (s *fieldState) set(fp *fieldPath, v any) (allocated int) {
	// Fast path for the common single-level case (fp.last == 0)
	if fp.last == 0 { //nolint:nestif
		z := fp.path[0]
		validateFieldIndex(z, fp)

		if y := len(s.state); y <= z {
			if z+2 > cap(s.state) {
				newSlice := make([]any, z+1, max(z+2, y*2))
				copy(newSlice, s.state)
				s.state = newSlice
				allocated = cap(newSlice)
			} else {
				s.state = s.state[:z+1]
			}
		}

		if _, ok := s.state[z].(*fieldState); !ok {
			s.state[z] = v
		}

		return allocated
	}

	x := s
	z := 0

	for i := 0; i <= fp.last; i++ {
		z = fp.path[i]
		validateFieldIndex(z, fp)

		if y := len(x.state); y <= z {
			newCap := max(z+2, y*2)
			if z+2 > cap(x.state) {
				newSlice := make([]any, z+1, newCap)
				copy(newSlice, x.state)
				x.state = newSlice
				allocated += newCap
			} else {
				// Re-slice to update the length without allocating new memory
				x.state = x.state[:z+1]
			}
		}

		if i == fp.last {
			if _, ok := x.state[z].(*fieldState); !ok {
				x.state[z] = v
			}

			return allocated
		}

		if _, ok := x.state[z].(*fieldState); !ok {
			x.state[z] = newFieldState()
			allocated += newFieldStateSlots
		}

		x = x.state[z].(*fieldState)
	}

	return allocated
}

func max(a, b int) int { //nolint:revive
	if a > b {
		return a
	}

	return b
}
