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
		if len(x.state) < z+2 {
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
		if y := len(x.state); y < z+2 {
			z := make([]interface{}, max(z+2, y*2))
			copy(z, x.state)
			x.state = z
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
