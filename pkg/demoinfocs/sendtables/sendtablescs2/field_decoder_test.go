package sendtablescs2

import (
	"math"
	"testing"
)

// TestNoscaleDecoder tests the noscaleDecoder function with various inputs
func TestNoscaleDecoder(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected float32
	}{
		{
			name:     "Zero",
			input:    []byte{0x00, 0x00, 0x00, 0x00},
			expected: 0.0,
		},
		{
			name:     "One",
			input:    []byte{0x00, 0x00, 0x80, 0x3F}, // 1.0 in IEEE 754
			expected: 1.0,
		},
		{
			name:     "MinusOne",
			input:    []byte{0x00, 0x00, 0x80, 0xBF}, // -1.0 in IEEE 754
			expected: -1.0,
		},
		{
			name:     "Pi",
			input:    []byte{0xDB, 0x0F, 0x49, 0x40}, // π in IEEE 754
			expected: math.Pi,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := newReader(tc.input)
			result := noscaleDecoder(r)
			
			if result != tc.expected {
				t.Errorf("Expected %f, got %f", tc.expected, result)
			}
		})
	}
}

// TestQanglePreciseDecoder tests the qanglePreciseDecoder function
func TestQanglePreciseDecoder(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected []float32
		minBits  uint32
	}{
		{
			name:     "NoComponents",
			input:    []byte{0x00, 0x00, 0x00}, // 000 (no x, y, z)
			expected: []float32{0.0, 0.0, 0.0},
			minBits:  3,
		},
		{
			name:     "XOnly",
			input:    []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00}, // 001 (x only) + 20 bits
			expected: []float32{-180.0, 0.0, 0.0}, // readBitCoordPres returns 0 - 180 = -180
			minBits:  3 + 20,
		},
		{
			name:     "YOnly", 
			input:    []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00}, // 010 (y only) + 20 bits
			expected: []float32{0.0, -180.0, 0.0},
			minBits:  3 + 20,
		},
		{
			name:     "ZOnly",
			input:    []byte{0x04, 0x00, 0x00, 0x00, 0x00, 0x00}, // 100 (z only) + 20 bits
			expected: []float32{0.0, 0.0, -180.0},
			minBits:  3 + 20,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := newReader(tc.input)
			result := qanglePreciseDecoder(r)
			
			resultSlice, ok := result.([]float32)
			if !ok {
				t.Fatalf("Expected []float32, got %T", result)
			}
			
			if len(resultSlice) != 3 {
				t.Fatalf("Expected slice of length 3, got %d", len(resultSlice))
			}
			
			for i := 0; i < 3; i++ {
				if math.Abs(float64(resultSlice[i]-tc.expected[i])) > 0.001 {
					t.Errorf("Component %d: expected %f, got %f", i, tc.expected[i], resultSlice[i])
				}
			}
		})
	}
}

// TestReadBitCoordPres tests the helper function used by qanglePreciseDecoder
func TestReadBitCoordPres(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected float32
	}{
		{
			name:     "Zero",
			input:    []byte{0x00, 0x00, 0x00}, // 20 bits of zero
			expected: -180.0,                   // 0.0 - 180.0
		},
		{
			name:     "Max",
			input:    []byte{0xFF, 0xFF, 0x0F}, // 20 bits of 1s (0xFFFFF)
			expected: float32(((1<<20)-1)*360.0/float64(1<<20)) - 180.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := newReader(tc.input)
			result := readBitCoordPres(r)
			
			if math.Abs(float64(result-tc.expected)) > 0.001 {
				t.Errorf("Expected %f, got %f", tc.expected, result)
			}
		})
	}
}

// TestDecoderConsistency ensures both functions produce consistent results across multiple calls
func TestDecoderConsistency(t *testing.T) {
	testData := []byte{0x3F, 0x80, 0x00, 0x00, 0x40, 0x00, 0x00, 0x00}
	
	// Test noscaleDecoder consistency
	t.Run("NoscaleConsistency", func(t *testing.T) {
		var results []interface{}
		for i := 0; i < 3; i++ {
			r := newReader(testData[:4])
			result := noscaleDecoder(r)
			results = append(results, result)
		}
		
		for i := 1; i < len(results); i++ {
			if results[0] != results[i] {
				t.Errorf("Inconsistent results: %v != %v", results[0], results[i])
			}
		}
	})
	
	// Test qanglePreciseDecoder consistency
	t.Run("QangleConsistency", func(t *testing.T) {
		qangleData := []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // all components
		var results []interface{}
		for i := 0; i < 3; i++ {
			r := newReader(qangleData)
			result := qanglePreciseDecoder(r)
			results = append(results, result)
		}
		
		for i := 1; i < len(results); i++ {
			slice0 := results[0].([]float32)
			sliceI := results[i].([]float32)
			for j := 0; j < 3; j++ {
				if slice0[j] != sliceI[j] {
					t.Errorf("Inconsistent results at position %d: %v != %v", j, slice0[j], sliceI[j])
				}
			}
		}
	})
}
