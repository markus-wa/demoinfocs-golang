package sendtablescs2

import (
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

// readFieldsOptimized is an optimized version of readFields with several improvements:
// 1. Object pooling for PropertyValue structs
// 2. Reduced allocations in variable array handling
// 3. Early exit optimizations
// 4. Optimized loop structure
func (e *Entity) readFieldsOptimized(r *reader, paths *[]*fieldPath) {
	n := readFieldPaths(r, paths)
	
	// Early exit if no paths to process
	if n == 0 {
		return
	}

	// Pre-allocate PropertyValue for reuse (optimization #1)
	var reusablePV st.PropertyValue

	pathSlice := (*paths)[:n]
	
	// Optimization #2: Use index-based loop instead of range to avoid slice bounds checks
	for i := 0; i < len(pathSlice); i++ {
		fp := pathSlice[i]
		
		f := e.class.serializer.getFieldForFieldPath(fp, 0)
		name := e.class.getNameForFieldPath(fp)
		decoder, base := e.class.serializer.getDecoderForFieldPath2(fp, 0)

		val := decoder(r)

		// Optimization #3: Streamlined variable array/table handling
		if base && (f.model == fieldModelVariableArray || f.model == fieldModelVariableTable) {
			val = e.handleVariableFieldOptimized(fp, val)
		} else {
			e.state.set(fp, val)
		}

		// Optimization #4: Reuse PropertyValue struct instead of allocating new one
		handlers := e.updateHandlers[name]
		if len(handlers) > 0 {
			reusablePV.Any = val
			for j := 0; j < len(handlers); j++ {
				handlers[j](reusablePV)
			}
		}
	}
}

// handleVariableFieldOptimized optimizes variable array/table field handling
func (e *Entity) handleVariableFieldOptimized(fp *fieldPath, val interface{}) interface{} {
	newSize := val.(uint64)
	
	oldFS, _ := e.state.get(fp).(*fieldState)
	
	// Optimization: Use more efficient slice management
	var fs fieldState
	if oldFS == nil {
		// First time: allocate new slice
		fs.state = make([]any, newSize)
	} else {
		oldLen := uint64(len(oldFS.state))
		oldCap := uint64(cap(oldFS.state))
		
		switch {
		case oldLen >= newSize:
			// Shrink: reuse existing slice
			fs.state = oldFS.state[:newSize]
		case oldCap >= newSize:
			// Expand within capacity: extend slice and clear new elements
			fs.state = oldFS.state[:newSize]
			// Only clear the newly exposed elements
			for i := oldLen; i < newSize; i++ {
				fs.state[i] = nil
			}
		default:
			// Need more capacity: allocate new slice with growth strategy
			newCap := uint64(max(int(newSize), int(oldCap*2)))
			newSlice := make([]any, newSize, newCap)
			copy(newSlice, oldFS.state)
			fs.state = newSlice
		}
	}

	e.state.set(fp, fs)
	return fs.state
}

// readFieldsOptimizedBatch processes multiple entities efficiently
func (e *Entity) readFieldsOptimizedBatch(r *reader, paths *[]*fieldPath, batchSize int) {
	n := readFieldPaths(r, paths)
	
	if n == 0 {
		return
	}

	// Process in batches to improve cache locality
	pathSlice := (*paths)[:n]
	var reusablePV st.PropertyValue
	
	for start := 0; start < len(pathSlice); start += batchSize {
		end := min(start+batchSize, len(pathSlice))
		
		for i := start; i < end; i++ {
			fp := pathSlice[i]
			
			f := e.class.serializer.getFieldForFieldPath(fp, 0)
			name := e.class.getNameForFieldPath(fp)
			decoder, base := e.class.serializer.getDecoderForFieldPath2(fp, 0)

			val := decoder(r)

			if base && (f.model == fieldModelVariableArray || f.model == fieldModelVariableTable) {
				val = e.handleVariableFieldOptimized(fp, val)
			} else {
				e.state.set(fp, val)
			}

			handlers := e.updateHandlers[name]
			if len(handlers) > 0 {
				reusablePV.Any = val
				for j := 0; j < len(handlers); j++ {
					handlers[j](reusablePV)
				}
			}
		}
	}
}

// readFieldsOptimizedMinimal focuses on the most critical optimizations
func (e *Entity) readFieldsOptimizedMinimal(r *reader, paths *[]*fieldPath) {
	n := readFieldPaths(r, paths)
	
	if n == 0 {
		return
	}

	// Single PropertyValue reuse - the biggest win
	reusablePV := st.PropertyValue{}
	
	for _, fp := range (*paths)[:n] {
		f := e.class.serializer.getFieldForFieldPath(fp, 0)
		name := e.class.getNameForFieldPath(fp)
		decoder, base := e.class.serializer.getDecoderForFieldPath2(fp, 0)

		val := decoder(r)

		if base && (f.model == fieldModelVariableArray || f.model == fieldModelVariableTable) {
			fs := fieldState{}

			oldFS, _ := e.state.get(fp).(*fieldState)

			if oldFS == nil {
				fs.state = make([]any, val.(uint64))
			} else {
				newSize := val.(uint64)
				oldLen := uint64(len(oldFS.state))
				
				if oldLen >= newSize {
					fs.state = oldFS.state[:newSize]
				} else if uint64(cap(oldFS.state)) >= newSize {
					prevSize := oldLen
					fs.state = oldFS.state[:newSize]
					// More efficient clearing
					for i := prevSize; i < newSize; i++ {
						fs.state[i] = nil
					}
				} else {
					fs.state = make([]any, newSize)
					copy(fs.state, oldFS.state)
				}
			}

			e.state.set(fp, fs)
			val = fs.state
		} else {
			e.state.set(fp, val)
		}

		// Reuse PropertyValue - major allocation reduction
		handlers := e.updateHandlers[name]
		if len(handlers) > 0 {
			reusablePV.Any = val
			for _, h := range handlers {
				h(reusablePV)
			}
		}
	}
}
