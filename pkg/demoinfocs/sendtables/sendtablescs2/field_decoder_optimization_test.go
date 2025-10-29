package sendtablescs2

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// TestOptimizationCorrectness ensures the optimizations don't change behavior
func TestOptimizationCorrectness(t *testing.T) {
	// Set random seed for reproducible tests
	rand.Seed(42)
	
	// Generate many test cases with random data
	for i := 0; i < 1000; i++ {
		// Test noscaleDecoder
		t.Run("NoscaleDecoder_Random", func(t *testing.T) {
			data := make([]byte, 4)
			rand.Read(data)
			
			r1 := newReader(data)
			r2 := newReader(data)
			
			result1 := noscaleDecoderOriginal(r1)
			result2 := noscaleDecoder(r2)
			
			if result1 != result2 {
				t.Errorf("Results differ: original=%v, optimized=%v", result1, result2)
			}
		})
		
		// Test qanglePreciseDecoder
		t.Run("QanglePreciseDecoder_Random", func(t *testing.T) {
			data := make([]byte, 16) // Enough for worst case
			rand.Read(data)
			
			r1 := newReader(data)
			r2 := newReader(data)
			
			result1 := qanglePreciseDecoderOriginal(r1)
			result2 := qanglePreciseDecoder(r2)
			
			slice1 := result1.([]float32)
			slice2 := result2.([]float32)
			
			if len(slice1) != len(slice2) {
				t.Fatalf("Length mismatch: %d vs %d", len(slice1), len(slice2))
			}
			
			for j := 0; j < len(slice1); j++ {
				if math.Abs(float64(slice1[j]-slice2[j])) > 0.0001 {
					t.Errorf("Component %d differs: original=%f, optimized=%f", j, slice1[j], slice2[j])
				}
			}
		})
	}
}

// TestZeroCaseOptimization specifically tests the zero case optimization
func TestZeroCaseOptimization(t *testing.T) {
	// Create data that represents all-false booleans (000)
	data := []byte{0x00, 0x00, 0x00, 0x00}
	
	// Call the optimized version multiple times
	var results []interface{}
	for i := 0; i < 10; i++ {
		r := newReader(data)
		result := qanglePreciseDecoder(r)
		results = append(results, result)
		
		// Verify it's a zero slice
		slice := result.([]float32)
		for j, val := range slice {
			if val != 0.0 {
				t.Errorf("Expected zero value at position %d, got %f", j, val)
			}
		}
	}
	
	// The optimization should return the same shared slice instance for zero cases
	// But since we copy the slice, they should be equal but potentially different instances
	slice0 := results[0].([]float32)
	for i := 1; i < len(results); i++ {
		sliceI := results[i].([]float32)
		for j := 0; j < 3; j++ {
			if slice0[j] != sliceI[j] {
				t.Errorf("Zero slice inconsistency at position %d: %f != %f", j, slice0[j], sliceI[j])
			}
		}
	}
}

// TestEdgeCases tests various edge cases for both functions
func TestEdgeCases(t *testing.T) {
	t.Run("NoscaleDecoder_EdgeCases", func(t *testing.T) {
		testCases := []struct {
			name     string
			data     []byte
			expected float32
		}{
			{"PositiveInfinity", []byte{0x00, 0x00, 0x80, 0x7F}, float32(math.Inf(1))},
			{"NegativeInfinity", []byte{0x00, 0x00, 0x80, 0xFF}, float32(math.Inf(-1))},
			{"MaxFloat32", []byte{0xFF, 0xFF, 0x7F, 0x7F}, math.MaxFloat32},
			{"SmallestFloat32", []byte{0x01, 0x00, 0x00, 0x00}, math.SmallestNonzeroFloat32},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				r := newReader(tc.data)
				result := noscaleDecoder(r)
				
				if math.IsInf(float64(tc.expected), 0) {
					if !math.IsInf(float64(result.(float32)), 0) {
						t.Errorf("Expected infinity, got %v", result)
					}
				} else if math.IsNaN(float64(tc.expected)) {
					if !math.IsNaN(float64(result.(float32))) {
						t.Errorf("Expected NaN, got %v", result)
					}
				} else if result != tc.expected {
					t.Errorf("Expected %v, got %v", tc.expected, result)
				}
			})
		}
	})
	
	t.Run("QanglePreciseDecoder_EdgeCases", func(t *testing.T) {
		testCases := []struct {
			name string
			data []byte
			desc string
		}{
			{"AllTrue", []byte{0x07, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, "All components true with max values"},
			{"OnlyX", []byte{0x01, 0xFF, 0xFF, 0x0F}, "Only X component"},
			{"OnlyY", []byte{0x02, 0xFF, 0xFF, 0x0F}, "Only Y component"},
			{"OnlyZ", []byte{0x04, 0xFF, 0xFF, 0x0F}, "Only Z component"},
			{"XAndY", []byte{0x03, 0xFF, 0xFF, 0xFF, 0xFF, 0x0F}, "X and Y components"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				r1 := newReader(tc.data)
				r2 := newReader(tc.data)
				
				result1 := qanglePreciseDecoderOriginal(r1)
				result2 := qanglePreciseDecoder(r2)
				
				slice1 := result1.([]float32)
				slice2 := result2.([]float32)
				
				for i := 0; i < 3; i++ {
					if math.Abs(float64(slice1[i]-slice2[i])) > 0.0001 {
						t.Errorf("Component %d differs: original=%f, optimized=%f", i, slice1[i], slice2[i])
					}
				}
			})
		}
	})
}

// TestConcurrentSafety ensures the optimizations are thread-safe
func TestConcurrentSafety(t *testing.T) {
	const numGoroutines = 100
	const numIterations = 1000
	
	// Test data
	noscaleData := []byte{0x00, 0x00, 0x80, 0x3F} // 1.0
	qangleData := []byte{0x00, 0x00, 0x00, 0x00}  // All zeros
	
	// Channel to collect results
	results := make(chan bool, numGoroutines*2)
	
	// Launch concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in goroutine: %v", r)
					results <- false
					return
				}
				results <- true
			}()
			
			// Test noscaleDecoder
			for j := 0; j < numIterations; j++ {
				r := newReader(noscaleData)
				result := noscaleDecoder(r)
				if result.(float32) != 1.0 {
					t.Errorf("Unexpected noscale result: %v", result)
					return
				}
			}
		}()
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in goroutine: %v", r)
					results <- false
					return
				}
				results <- true
			}()
			
			// Test qanglePreciseDecoder
			for j := 0; j < numIterations; j++ {
				r := newReader(qangleData)
				result := qanglePreciseDecoder(r)
				slice := result.([]float32)
				for k, val := range slice {
					if val != 0.0 {
						t.Errorf("Unexpected qangle result at %d: %v", k, val)
						return
					}
				}
			}
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines*2; i++ {
		select {
		case success := <-results:
			if !success {
				t.Fatal("Goroutine failed")
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}

// TestMemoryLeaks tests for potential memory leaks in optimizations
func TestMemoryLeaks(t *testing.T) {
	// This test ensures that the shared zero slice doesn't accumulate references
	data := []byte{0x00, 0x00, 0x00, 0x00} // All zeros
	
	// Call the function many times to potentially expose memory leaks
	for i := 0; i < 10000; i++ {
		r := newReader(data)
		result := qanglePreciseDecoder(r)
		
		// Verify the result but don't hold references
		slice := result.([]float32)
		if len(slice) != 3 {
			t.Fatalf("Unexpected slice length: %d", len(slice))
		}
		
		// Modify the returned slice to ensure it doesn't affect the shared zero slice
		slice[0] = float32(i)
		slice[1] = float32(i + 1)
		slice[2] = float32(i + 2)
	}
	
	// Verify the shared zero slice is still zero
	r := newReader(data)
	result := qanglePreciseDecoder(r)
	slice := result.([]float32)
	for i, val := range slice {
		if val != 0.0 {
			t.Errorf("Shared zero slice corrupted at position %d: %f", i, val)
		}
	}
}
