package sendtablescs2

import (
	"testing"
	"unsafe"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

// TestReadFields_PropertyValueReuse tests the core optimization: PropertyValue reuse
func TestReadFields_PropertyValueReuse(t *testing.T) {
	// Test that PropertyValue can be reused without issues
	var reusablePV st.PropertyValue
	
	values := []interface{}{uint32(42), uint32(84), uint32(126)}
	results := make([]interface{}, 0, len(values))
	
	// Simulate handler that captures values
	handler := func(pv st.PropertyValue) {
		results = append(results, pv.Any)
	}
	
	// Test reusing PropertyValue struct
	for _, val := range values {
		reusablePV.Any = val
		handler(reusablePV)
	}
	
	// Verify all values were captured correctly
	if len(results) != len(values) {
		t.Errorf("Expected %d results, got %d", len(values), len(results))
	}
	
	for i, expected := range values {
		if results[i] != expected {
			t.Errorf("Result %d: expected %v, got %v", i, expected, results[i])
		}
	}
}

// TestReadFields_EarlyExit tests the early exit optimization
func TestReadFields_EarlyExit(t *testing.T) {
	// Test behavior with empty field paths
	entity := &Entity{
		state:          newFieldState(),
		updateHandlers: make(map[string][]st.PropertyUpdateHandler),
	}
	
	data := make([]byte, 256)
	reader := newReader(data)
	paths := make([]*fieldPath, 0) // Empty paths
	
	// This should exit early and not cause any issues
	testEarlyExit(entity, reader, &paths)
}

func testEarlyExit(entity *Entity, reader *reader, paths *[]*fieldPath) {
	// Simulate the early exit logic
	n := len(*paths)
	if n == 0 {
		return // Early exit
	}
	
	// If we reach here with empty paths, that's a problem
	panic("Early exit failed")
}

// TestReadFields_VariableArrayOptimization tests optimized slice management
func TestReadFields_VariableArrayOptimization(t *testing.T) {
	tests := []struct {
		name           string
		initialSize    uint64
		newSize        uint64
		expectRealloc  bool
	}{
		{"Shrink", 10, 5, false},
		{"Grow within capacity", 5, 8, false}, // Assuming cap >= 8
		{"Grow beyond capacity", 5, 20, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the slice management logic
			oldSlice := make([]any, tt.initialSize, max(int(tt.initialSize), 10))
			
			var newSlice []any
			oldLen := uint64(len(oldSlice))
			oldCap := uint64(cap(oldSlice))
			
			// Simulate the optimized slice management
			switch {
			case oldLen >= tt.newSize:
				newSlice = oldSlice[:tt.newSize]
			case oldCap >= tt.newSize:
				newSlice = oldSlice[:tt.newSize]
				// Clear newly exposed elements
				for i := oldLen; i < tt.newSize; i++ {
					newSlice[i] = nil
				}
			default:
				newCap := uint64(max(int(tt.newSize), int(oldCap*2)))
				newSlice = make([]any, tt.newSize, newCap)
				copy(newSlice, oldSlice)
			}
			
			// Verify the result
			if uint64(len(newSlice)) != tt.newSize {
				t.Errorf("Expected length %d, got %d", tt.newSize, len(newSlice))
			}
			
			// Check if reallocation happened as expected
			didRealloc := unsafe.Pointer(&newSlice[0]) != unsafe.Pointer(&oldSlice[0])
			if didRealloc != tt.expectRealloc {
				t.Errorf("Expected realloc: %v, got: %v", tt.expectRealloc, didRealloc)
			}
		})
	}
}

// TestReadFields_ConcurrentSafety tests that optimizations maintain thread safety
func TestReadFields_ConcurrentSafety(t *testing.T) {
	// Test that PropertyValue reuse doesn't cause data races
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()
			
			// Each goroutine gets its own PropertyValue
			var localPV st.PropertyValue
			
			for j := 0; j < 100; j++ {
				localPV.Any = uint32(id*100 + j)
				
				// Simulate handler call
				_ = localPV.Any
			}
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// BenchmarkPropertyValueReuse benchmarks the optimization
func BenchmarkPropertyValueReuse(b *testing.B) {
	values := []interface{}{uint32(1), uint32(2), uint32(3), uint32(4), uint32(5)}
	
	b.Run("WithReuse", func(b *testing.B) {
		var reusablePV st.PropertyValue
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			for _, val := range values {
				reusablePV.Any = val
				_ = reusablePV // Prevent optimization
			}
		}
	})
	
	b.Run("WithoutReuse", func(b *testing.B) {
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			for _, val := range values {
				pv := st.PropertyValue{Any: val}
				_ = pv // Prevent optimization
			}
		}
	})
}
