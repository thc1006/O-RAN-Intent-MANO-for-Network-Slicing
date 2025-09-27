package performance

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"../fixtures"
)

// BenchmarkQoSConversion benchmarks the QoS conversion performance
func BenchmarkQoSConversion(b *testing.B) {
	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
		_ = result // Prevent optimization
	}
}

// BenchmarkQoSConversionMemory benchmarks memory allocation during conversion
func BenchmarkQoSConversionMemory(b *testing.B) {
	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
		_ = result // Prevent optimization
	}
}

// BenchmarkQoSConversionParallel benchmarks parallel conversion performance
func BenchmarkQoSConversionParallel(b *testing.B) {
	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
			_ = result // Prevent optimization
		}
	})
}

// BenchmarkQoSValidation benchmarks QoS validation performance
func BenchmarkQoSValidation(b *testing.B) {
	qosProfile := fixtures.QoSProfile{
		Latency: fixtures.LatencyRequirement{
			Value: "20",
			Unit:  "ms",
			Type:  "end-to-end",
		},
		Throughput: fixtures.ThroughputRequirement{
			Downlink: "1Gbps",
			Uplink:   "100Mbps",
			Unit:     "bps",
		},
		Reliability: fixtures.ReliabilityRequirement{
			Value: "99.9",
			Unit:  "percentage",
		},
	}

	// Note: We'd need a mock testing.T for this benchmark
	// For now, just measure the conversion part
	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
		// Simplified validation check
		if result.Latency.Value == "" {
			b.Fatal("Invalid conversion result")
		}
	}
}

// TestConcurrentQoSConversion tests concurrent access to QoS conversion
func TestConcurrentQoSConversion(t *testing.T) {
	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	numGoroutines := 1000
	numIterations := 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	results := make(chan fixtures.QoSProfile, numGoroutines*numIterations)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)

				// Validate result
				if result.Latency.Value != "20" {
					errors <- fmt.Errorf("goroutine %d iteration %d: expected latency 20, got %s", id, j, result.Latency.Value)
					return
				}
				if result.Throughput.Downlink != "1Gbps" {
					errors <- fmt.Errorf("goroutine %d iteration %d: expected throughput 1Gbps, got %s", id, j, result.Throughput.Downlink)
					return
				}

				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(results)

	duration := time.Since(start)
	totalOperations := numGoroutines * numIterations

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Count successful results
	successCount := 0
	for range results {
		successCount++
	}

	assert.Equal(t, totalOperations, successCount, "All operations should succeed")

	// Performance assertions
	opsPerSecond := float64(totalOperations) / duration.Seconds()
	t.Logf("Completed %d operations in %v (%.2f ops/sec)", totalOperations, duration, opsPerSecond)

	// Should be able to handle at least 10k ops/sec
	assert.Greater(t, opsPerSecond, 10000.0, "Should handle at least 10k operations per second")
}

// TestLargeDatasetValidation tests validation with large datasets
func TestLargeDatasetValidation(t *testing.T) {
	datasetSizes := []int{100, 1000, 10000}

	for _, size := range datasetSizes {
		t.Run(fmt.Sprintf("dataset_size_%d", size), func(t *testing.T) {
			start := time.Now()

			// Generate large dataset
			vnfProfiles := make([]fixtures.VNFQoSProfile, size)
			for i := 0; i < size; i++ {
				vnfProfiles[i] = fixtures.VNFQoSProfile{
					Latency:     fmt.Sprintf("%dms", 10+i%90),
					Throughput:  fmt.Sprintf("%dMbps", 100+i%1000),
					Reliability: fmt.Sprintf("99.%d%%", 9+i%10),
				}
			}

			generationTime := time.Since(start)

			// Convert all profiles
			start = time.Now()
			qosProfiles := make([]fixtures.QoSProfile, size)
			for i, vnfProfile := range vnfProfiles {
				qosProfiles[i] = fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
			}
			conversionTime := time.Since(start)

			// Validate conversion was successful
			for i, qosProfile := range qosProfiles {
				assert.NotEmpty(t, qosProfile.Latency.Value, "Profile %d should have latency", i)
				assert.NotEmpty(t, qosProfile.Throughput.Downlink, "Profile %d should have throughput", i)
				assert.NotEmpty(t, qosProfile.Reliability.Value, "Profile %d should have reliability", i)
			}

			validationTime := time.Since(start) - conversionTime

			t.Logf("Dataset size: %d", size)
			t.Logf("Generation time: %v", generationTime)
			t.Logf("Conversion time: %v (%.2f μs/item)", conversionTime, float64(conversionTime.Nanoseconds()/1000)/float64(size))
			t.Logf("Validation time: %v", validationTime)

			// Performance requirements
			avgConversionTimePerItem := conversionTime.Nanoseconds() / int64(size)
			assert.Less(t, avgConversionTimePerItem, int64(10000), "Average conversion time should be < 10μs per item")
		})
	}
}

// TestQoSConversionProfiling tests for CPU and memory profiling
func TestQoSConversionProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping profiling test in short mode")
	}

	// Test data
	vnfProfiles := []fixtures.VNFQoSProfile{
		{Latency: "1ms", Throughput: "100Mbps", Reliability: "99.999%"},   // URLLC
		{Latency: "20ms", Throughput: "1Gbps", Reliability: "99.9%"},     // eMBB
		{Latency: "100ms", Throughput: "10Mbps", Reliability: "99.9%"},   // mMTC
	}

	numIterations := 10000

	// Measure initial memory
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	start := time.Now()

	// Perform conversions
	for i := 0; i < numIterations; i++ {
		vnfProfile := vnfProfiles[i%len(vnfProfiles)]
		result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
		_ = result // Prevent optimization
	}

	duration := time.Since(start)

	// Force garbage collection and measure final memory
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate metrics
	memoryUsed := m2.TotalAlloc - m1.TotalAlloc
	opsPerSecond := float64(numIterations) / duration.Seconds()

	t.Logf("Completed %d conversions in %v", numIterations, duration)
	t.Logf("Operations per second: %.2f", opsPerSecond)
	t.Logf("Total memory allocated: %d bytes", memoryUsed)
	t.Logf("Memory per operation: %.2f bytes", float64(memoryUsed)/float64(numIterations))
	t.Logf("Final heap size: %d bytes", m2.HeapInuse)
	t.Logf("GC runs: %d", m2.NumGC-m1.NumGC)

	// Performance assertions
	assert.Greater(t, opsPerSecond, 50000.0, "Should handle at least 50k ops/sec")
	assert.Less(t, float64(memoryUsed)/float64(numIterations), 1000.0, "Should use less than 1KB per operation")
}

// TestMemoryLeakDetection tests for memory leaks during conversion
func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	// Warm up
	for i := 0; i < 1000; i++ {
		result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
		_ = result
	}

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform many conversions
	numIterations := 100000
	for i := 0; i < numIterations; i++ {
		result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
		_ = result
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	heapGrowth := int64(m2.HeapInuse) - int64(m1.HeapInuse)

	t.Logf("Heap before: %d bytes", m1.HeapInuse)
	t.Logf("Heap after: %d bytes", m2.HeapInuse)
	t.Logf("Heap growth: %d bytes", heapGrowth)
	t.Logf("Iterations: %d", numIterations)

	// Memory should not grow significantly
	maxAcceptableGrowth := int64(1024 * 1024) // 1MB
	assert.Less(t, heapGrowth, maxAcceptableGrowth,
		"Heap should not grow more than %d bytes, grew %d bytes", maxAcceptableGrowth, heapGrowth)
}

// TestConversionWithDifferentSliceTypes tests performance across slice types
func TestConversionWithDifferentSliceTypes(t *testing.T) {
	sliceTypes := map[string]fixtures.VNFQoSProfile{
		"eMBB": {
			Latency:     "50ms",
			Throughput:  "10Gbps",
			Reliability: "99.9%",
		},
		"URLLC": {
			Latency:     "1ms",
			Throughput:  "100Mbps",
			Reliability: "99.999%",
		},
		"mMTC": {
			Latency:     "1000ms",
			Throughput:  "1Mbps",
			Reliability: "99.9%",
		},
	}

	for sliceType, vnfProfile := range sliceTypes {
		t.Run(sliceType, func(t *testing.T) {
			iterations := 10000
			start := time.Now()

			for i := 0; i < iterations; i++ {
				result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
				_ = result
			}

			duration := time.Since(start)
			opsPerSecond := float64(iterations) / duration.Seconds()

			t.Logf("%s: %d conversions in %v (%.2f ops/sec)",
				sliceType, iterations, duration, opsPerSecond)

			// All slice types should have similar performance
			assert.Greater(t, opsPerSecond, 10000.0,
				"%s conversions should handle at least 10k ops/sec", sliceType)
		})
	}
}

// TestValidationPerformance tests QoS validation performance
func TestQoSValidationPerformance(t *testing.T) {
	// Create test profiles for each slice type
	profiles := map[fixtures.SliceType]fixtures.QoSProfile{
		fixtures.SliceTypeEMBB: {
			Latency: fixtures.LatencyRequirement{
				Value: "50",
				Unit:  "ms",
				Type:  "end-to-end",
			},
			Throughput: fixtures.ThroughputRequirement{
				Downlink: "10Gbps",
				Uplink:   "1Gbps",
				Unit:     "bps",
			},
			Reliability: fixtures.ReliabilityRequirement{
				Value: "99.9",
				Unit:  "percentage",
			},
		},
		fixtures.SliceTypeURLLC: {
			Latency: fixtures.LatencyRequirement{
				Value: "1",
				Unit:  "ms",
				Type:  "end-to-end",
			},
			Throughput: fixtures.ThroughputRequirement{
				Downlink: "100Mbps",
				Uplink:   "50Mbps",
				Unit:     "bps",
			},
			Reliability: fixtures.ReliabilityRequirement{
				Value: "99.999",
				Unit:  "percentage",
			},
		},
		fixtures.SliceTypeMmTC: {
			Latency: fixtures.LatencyRequirement{
				Value: "1000",
				Unit:  "ms",
				Type:  "end-to-end",
			},
			Throughput: fixtures.ThroughputRequirement{
				Downlink: "1Mbps",
				Uplink:   "500Kbps",
				Unit:     "bps",
			},
			Reliability: fixtures.ReliabilityRequirement{
				Value: "99.9",
				Unit:  "percentage",
			},
		},
	}

	for sliceType, profile := range profiles {
		t.Run(string(sliceType), func(t *testing.T) {
			iterations := 1000 // Fewer iterations for validation tests
			start := time.Now()

			for i := 0; i < iterations; i++ {
				// Note: We'd need a way to create a mock testing.T for benchmarking
				// For now, just validate the profile structure
				assert.NotEmpty(t, profile.Latency.Value)
				assert.NotEmpty(t, profile.Throughput.Downlink)
				assert.NotEmpty(t, profile.Reliability.Value)
			}

			duration := time.Since(start)
			opsPerSecond := float64(iterations) / duration.Seconds()

			t.Logf("%s validation: %d operations in %v (%.2f ops/sec)",
				sliceType, iterations, duration, opsPerSecond)

			// Validation should be fast
			assert.Greater(t, opsPerSecond, 1000.0,
				"%s validation should handle at least 1k ops/sec", sliceType)
		})
	}
}

// BenchmarkJSONSerialization benchmarks JSON serialization performance
func BenchmarkJSONSerialization(b *testing.B) {
	qosProfile := fixtures.QoSProfile{
		Latency: fixtures.LatencyRequirement{
			Value: "20",
			Unit:  "ms",
			Type:  "end-to-end",
		},
		Throughput: fixtures.ThroughputRequirement{
			Downlink: "1Gbps",
			Uplink:   "100Mbps",
			Unit:     "bps",
		},
		Reliability: fixtures.ReliabilityRequirement{
			Value: "99.9",
			Unit:  "percentage",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonData := fixtures.MustMarshalJSON(qosProfile)
		_ = jsonData
	}
}

// BenchmarkJSONDeserialization benchmarks JSON deserialization performance
func BenchmarkJSONDeserialization(b *testing.B) {
	qosProfile := fixtures.QoSProfile{
		Latency: fixtures.LatencyRequirement{
			Value: "20",
			Unit:  "ms",
			Type:  "end-to-end",
		},
		Throughput: fixtures.ThroughputRequirement{
			Downlink: "1Gbps",
			Uplink:   "100Mbps",
			Unit:     "bps",
		},
		Reliability: fixtures.ReliabilityRequirement{
			Value: "99.9",
			Unit:  "percentage",
		},
	}

	jsonData := fixtures.MustMarshalJSON(qosProfile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result fixtures.QoSProfile
		fixtures.MustUnmarshalJSON(jsonData, &result)
	}
}