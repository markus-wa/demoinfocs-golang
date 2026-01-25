package sendtablescs2

import (
	"math"
	"math/rand"
	"testing"
)

// generateTestData creates realistic test data for benchmarking
func generateTestData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// Original implementations for comparison
func noscaleDecoderOriginal(r *reader) interface{} {
	return math.Float32frombits(r.readLeUint32())
}

func qanglePreciseDecoderOriginal(r *reader) interface{} {
	v := make([]float32, 3)
	hasX := r.readBoolean()
	hasY := r.readBoolean()
	hasZ := r.readBoolean()

	if hasX {
		v[0] = readBitCoordPres(r)
	}

	if hasY {
		v[1] = readBitCoordPres(r)
	}

	if hasZ {
		v[2] = readBitCoordPres(r)
	}

	return v
}

// BenchmarkNoscaleDecoder benchmarks both original and optimized implementations
func BenchmarkNoscaleDecoder(b *testing.B) {
	testData := generateTestData(1024) // 1KB of test data
	
	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			r := newReader(testData)
			for r.remBytes() >= 4 { // need at least 4 bytes for readLeUint32
				_ = noscaleDecoderOriginal(r)
			}
		}
	})
	
	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			r := newReader(testData)
			for r.remBytes() >= 4 { // need at least 4 bytes for noscaleDecoder
				_ = noscaleDecoder(r)
			}
		}
	})
}

// BenchmarkQanglePreciseDecoder benchmarks both original and optimized implementations
func BenchmarkQanglePreciseDecoder(b *testing.B) {
	// Create test data with different bit patterns to simulate various qangle scenarios
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "AllComponents",
			data: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			name: "NoComponents", 
			data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "MixedComponents",
			data: []byte{0xA5, 0x5A, 0xC3, 0x3C, 0x96, 0x69, 0xF0, 0x0F, 0x55, 0xAA, 0x33, 0xCC, 0x77, 0x88, 0x22, 0xDD},
		},
	}
	
	for _, tc := range testCases {
		b.Run("Original_"+tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				r := newReader(tc.data)
				for r.remBits() >= 20*3+3 { // minimum bits needed for qanglePreciseDecoder
					_ = qanglePreciseDecoderOriginal(r)
				}
			}
		})
		
		b.Run("Optimized_"+tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				r := newReader(tc.data)
				for r.remBits() >= 20*3+3 { // minimum bits needed for qanglePreciseDecoder
					_ = qanglePreciseDecoder(r)
				}
			}
		})
	}
}

// BenchmarkMemoryAllocation specifically measures memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	testData := generateTestData(2048)
	
	b.Run("NoscaleDecoder_Original_Allocs", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			r := newReader(testData)
			for r.remBytes() >= 4 {
				result := noscaleDecoderOriginal(r)
				_ = result // prevent optimization
			}
		}
	})
	
	b.Run("NoscaleDecoder_Optimized_Allocs", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			r := newReader(testData)
			for r.remBytes() >= 4 {
				result := noscaleDecoder(r)
				_ = result // prevent optimization
			}
		}
	})
	
	b.Run("QanglePreciseDecoder_Original_Allocs", func(b *testing.B) {
		b.ResetTimer() 
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			r := newReader(testData)
			for r.remBits() >= 20*3+3 {
				result := qanglePreciseDecoderOriginal(r)
				_ = result // prevent optimization
			}
		}
	})
	
	b.Run("QanglePreciseDecoder_Optimized_Allocs", func(b *testing.B) {
		b.ResetTimer() 
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			r := newReader(testData)
			for r.remBits() >= 20*3+3 {
				result := qanglePreciseDecoder(r)
				_ = result // prevent optimization
			}
		}
	})
}

// BenchmarkConcurrent tests performance under concurrent access
func BenchmarkConcurrent(b *testing.B) {
	testData := generateTestData(4096)
	
	b.Run("NoscaleDecoder_Original_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				r := newReader(testData)
				for r.remBytes() >= 4 {
					_ = noscaleDecoderOriginal(r)
				}
			}
		})
	})
	
	b.Run("NoscaleDecoder_Optimized_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				r := newReader(testData)
				for r.remBytes() >= 4 {
					_ = noscaleDecoder(r)
				}
			}
		})
	})
	
	b.Run("QanglePreciseDecoder_Original_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				r := newReader(testData)
				for r.remBits() >= 20*3+3 {
					_ = qanglePreciseDecoderOriginal(r)
				}
			}
		})
	})
	
	b.Run("QanglePreciseDecoder_Optimized_Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				r := newReader(testData)
				for r.remBits() >= 20*3+3 {
					_ = qanglePreciseDecoder(r)
				}
			}
		})
	})
}
