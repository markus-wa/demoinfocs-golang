package sendtables2

type fieldState struct {
	state []interface{}
}

func newFieldState() *fieldState {
	return &fieldState{
		state: make([]interface{}, 8),
	}
}

func (s *fieldState) get(fp *fieldPath) interface{} {
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

func (s *fieldState) set(fp *fieldPath, v interface{}) {
	x := s
	z := 0

	for i := 0; i <= fp.last; i++ {
		z = fp.path[i]

		if y := len(x.state); y <= z {
			newCap := max(z+2, y*2)
			if newCap > cap(x.state) {
				newSlice := make([]interface{}, z+1, newCap)
				copy(newSlice, x.state)
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
			return
		}

		if _, ok := x.state[z].(*fieldState); !ok {
			x.state[z] = newFieldState()
		}

		x = x.state[z].(*fieldState)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
