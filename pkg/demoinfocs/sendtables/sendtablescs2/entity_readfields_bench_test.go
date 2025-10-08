package sendtablescs2

import (
	"runtime"
	"testing"

	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

// BenchmarkReadFields_Current benchmarks the current implementation
func BenchmarkReadFields_Current(b *testing.B) {
	// Create a simple entity with minimal setup
	entity := &Entity{
		index:          1,
		serial:         1,
		class:          nil, // Will need to handle nil checks
		active:         true,
		state:          newFieldState(),
		fpCache:        make(map[string]*fieldPath),
		fpNoop:         make(map[string]bool),
		updateHandlers: make(map[string][]st.PropertyUpdateHandler),
		propCache:      make(map[string]st.Property),
	}

	// Create mock reader with dummy data
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	reader := newReader(data)

	// Create simple field paths
	paths := make([]*fieldPath, 10)
	for i := 0; i < 10; i++ {
		fp := newFieldPath()
		fp.path[0] = i
		fp.last = 0
		fp.done = false
		paths[i] = fp
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Reset reader position
		reader.pos = 0
		reader.size = uint32(len(reader.buf))
		
		// Use the current readFields implementation with simplified logic
		entity.readFieldsCurrentSimplified(reader, &paths)
	}
}

// Simplified current implementation for comparison
func (e *Entity) readFieldsCurrentSimplified(r *reader, paths *[]*fieldPath) {
	// Simulate the core loop without the complex class/serializer dependencies
	n := len(*paths)

	for i := 0; i < n; i++ {
		fp := (*paths)[i]
		
		// Mock decoder that just reads a uint32
		val := r.readLeUint32()
		
		// Simple state setting
		e.state.set(fp, val)
		
		// Mock property value creation (the allocation we want to optimize)
		pv := st.PropertyValue{Any: val}
		_ = pv // Prevent optimization
	}
}

// BenchmarkReadFields_MemoryProfile focuses on memory allocation patterns
func BenchmarkReadFields_MemoryProfile(b *testing.B) {
	entity := &Entity{
		index:          1,
		serial:         1,
		state:          newFieldState(),
		fpCache:        make(map[string]*fieldPath),
		fpNoop:         make(map[string]bool),
		updateHandlers: make(map[string][]st.PropertyUpdateHandler),
		propCache:      make(map[string]st.Property),
	}

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	reader := newReader(data)

	paths := make([]*fieldPath, 10)
	for i := 0; i < 10; i++ {
		fp := newFieldPath()
		fp.path[0] = i
		fp.last = 0
		fp.done = false
		paths[i] = fp
	}
	
	var m1, m2 runtime.MemStats
	
	b.ResetTimer()
	
	// Measure memory before
	runtime.ReadMemStats(&m1)
	
	for i := 0; i < b.N; i++ {
		reader.pos = 0
		reader.size = uint32(len(reader.buf))
		
		entity.readFieldsCurrentSimplified(reader, &paths)
	}
	
	// Measure memory after
	runtime.ReadMemStats(&m2)
	
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "allocs/op")
}

// BenchmarkReadFields_OptimizedMinimal benchmarks the minimal optimization version
func BenchmarkReadFields_OptimizedMinimal(b *testing.B) {
	entity := &Entity{
		index:          1,
		serial:         1,
		state:          newFieldState(),
		fpCache:        make(map[string]*fieldPath),
		fpNoop:         make(map[string]bool),
		updateHandlers: make(map[string][]st.PropertyUpdateHandler),
		propCache:      make(map[string]st.Property),
	}

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	reader := newReader(data)

	paths := make([]*fieldPath, 10)
	for i := 0; i < 10; i++ {
		fp := newFieldPath()
		fp.path[0] = i
		fp.last = 0
		fp.done = false
		paths[i] = fp
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		reader.pos = 0
		reader.size = uint32(len(reader.buf))
		
		entity.readFieldsOptimizedMinimalSimplified(reader, &paths)
	}
}

// Simplified optimized implementation for benchmarking
func (e *Entity) readFieldsOptimizedMinimalSimplified(r *reader, paths *[]*fieldPath) {
	n := len(*paths)
	
	if n == 0 {
		return
	}

	// Single PropertyValue reuse - the biggest optimization win
	reusablePV := st.PropertyValue{}
	
	for i := 0; i < n; i++ {
		fp := (*paths)[i]
		
		// Mock decoder that just reads a uint32
		val := r.readLeUint32()
		
		// Simple state setting
		e.state.set(fp, val)
		
		// Reuse PropertyValue instead of allocating new one each time
		reusablePV.Any = val
		_ = reusablePV // Prevent optimization
	}
}

// BenchmarkReadFields_Comparison runs both implementations side by side
func BenchmarkReadFields_Comparison(b *testing.B) {
	entity := &Entity{
		index:          1,
		serial:         1,
		state:          newFieldState(),
		fpCache:        make(map[string]*fieldPath),
		fpNoop:         make(map[string]bool),
		updateHandlers: make(map[string][]st.PropertyUpdateHandler),
		propCache:      make(map[string]st.Property),
	}

	// Add some handlers to make the test more realistic
	dummyHandler := func(pv st.PropertyValue) {
		_ = pv.Any // Just access the value
	}
	entity.updateHandlers["field0"] = []st.PropertyUpdateHandler{dummyHandler}
	entity.updateHandlers["field1"] = []st.PropertyUpdateHandler{dummyHandler}
	entity.updateHandlers["field2"] = []st.PropertyUpdateHandler{dummyHandler}

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	reader := newReader(data)

	paths := make([]*fieldPath, 10)
	for i := 0; i < 10; i++ {
		fp := newFieldPath()
		fp.path[0] = i
		fp.last = 0
		fp.done = false
		paths[i] = fp
	}

	b.Run("Current", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader.pos = 0
			reader.size = uint32(len(reader.buf))
			entity.readFieldsCurrentWithHandlers(reader, &paths)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader.pos = 0
			reader.size = uint32(len(reader.buf))
			entity.readFieldsOptimizedWithHandlers(reader, &paths)
		}
	})
}

// Current implementation with handlers to test PropertyValue allocation
func (e *Entity) readFieldsCurrentWithHandlers(r *reader, paths *[]*fieldPath) {
	n := len(*paths)

	for i := 0; i < n; i++ {
		fp := (*paths)[i]
		
		val := r.readLeUint32()
		e.state.set(fp, val)
		
		// Simulate handler calls with realistic field names
		fieldName := "field" + string(rune('0'+i%10))
		handlers := e.updateHandlers[fieldName]
		
		// This creates a new PropertyValue struct for each handler call
		for _, h := range handlers {
			h(st.PropertyValue{Any: val}) // NEW ALLOCATION EACH TIME
		}
	}
}

// Optimized implementation with PropertyValue reuse
func (e *Entity) readFieldsOptimizedWithHandlers(r *reader, paths *[]*fieldPath) {
	n := len(*paths)
	
	if n == 0 {
		return
	}

	// OPTIMIZATION: Single PropertyValue reuse
	reusablePV := st.PropertyValue{}

	for i := 0; i < n; i++ {
		fp := (*paths)[i]
		
		val := r.readLeUint32()
		e.state.set(fp, val)
		
		fieldName := "field" + string(rune('0'+i%10))
		handlers := e.updateHandlers[fieldName]
		
		// OPTIMIZATION: Reuse PropertyValue instead of allocating new one
		if len(handlers) > 0 {
			reusablePV.Any = val
			for _, h := range handlers {
				h(reusablePV) // REUSE SAME STRUCT
			}
		}
	}
}
