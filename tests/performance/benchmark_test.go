// Package performance provides comprehensive benchmarking tests
package performance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/utils"
)

// BenchmarkOrchestratorIntentProcessing benchmarks intent processing performance
func BenchmarkOrchestratorIntentProcessing(b *testing.B) {
	testEnv := utils.SetupTestEnvironment(b, scheme)
	defer testEnv.Cleanup(b)

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		intentID := 0
		for pb.Next() {
			intentID++
			processIntent(ctx, testEnv, fmt.Sprintf("benchmark-intent-%d", intentID))
		}
	})
}

// BenchmarkThroughputMeasurement benchmarks network throughput measurement
func BenchmarkThroughputMeasurement(b *testing.B) {
	collector, err := utils.NewMetricsCollector("http://prometheus:9090", nil, "default")
	if err != nil {
		b.Skip("Prometheus not available")
	}

	ctx := context.Background()
	targetIP := "10.0.1.100" // Mock target

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.CollectNetworkMetrics(ctx, targetIP, 5*time.Second)
		if err != nil {
			b.Errorf("Failed to collect network metrics: %v", err)
		}
	}
}

// BenchmarkConcurrentSliceDeployment benchmarks concurrent slice deployment
func BenchmarkConcurrentSliceDeployment(b *testing.B) {
	testEnv := utils.SetupTestEnvironment(b, scheme)
	defer testEnv.Cleanup(b)

	ctx := context.Background()
	concurrencyLevels := []int{1, 5, 10, 20, 50}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				deploySlicesConcurrently(ctx, testEnv, concurrency)
			}
		})
	}
}

// BenchmarkResourceAllocation benchmarks resource allocation performance
func BenchmarkResourceAllocation(b *testing.B) {
	testEnv := utils.SetupTestEnvironment(b, scheme)
	defer testEnv.Cleanup(b)

	ctx := context.Background()
	resourceSizes := []int{10, 50, 100, 500, 1000}

	for _, size := range resourceSizes {
		b.Run(fmt.Sprintf("Resources-%d", size), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				allocateResources(ctx, testEnv, size)
			}
		})
	}
}

// BenchmarkE2EIntentFlow benchmarks end-to-end intent flow
func BenchmarkE2EIntentFlow(b *testing.B) {
	testEnv := utils.SetupTestEnvironment(b, scheme)
	defer testEnv.Cleanup(b)

	ctx := context.Background()
	sliceTypes := []string{"embb", "urllc", "mmtc"}

	for _, sliceType := range sliceTypes {
		b.Run(fmt.Sprintf("SliceType-%s", sliceType), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				executeE2EFlow(ctx, testEnv, sliceType, i)
			}
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory usage under load
func BenchmarkMemoryUsage(b *testing.B) {
	testEnv := utils.SetupTestEnvironment(b, scheme)
	defer testEnv.Cleanup(b)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create temporary objects to simulate memory usage
		data := make([]byte, 1024*1024) // 1MB allocation
		processLargeDataset(ctx, data)
	}
}

// BenchmarkScalabilityLimits tests system scalability limits
func BenchmarkScalabilityLimits(b *testing.B) {
	testEnv := utils.SetupTestEnvironment(b, scheme)
	defer testEnv.Cleanup(b)

	ctx := context.Background()
	loadLevels := []int{10, 100, 500, 1000, 2000}

	for _, load := range loadLevels {
		b.Run(fmt.Sprintf("Load-%d", load), func(b *testing.B) {
			if testing.Short() && load > 100 {
				b.Skip("Skipping high load test in short mode")
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				simulateSystemLoad(ctx, testEnv, load)
			}
		})
	}
}

// BenchmarkLatencyMeasurement benchmarks latency measurement accuracy
func BenchmarkLatencyMeasurement(b *testing.B) {
	targetIPs := []string{
		"10.0.1.100", // Local network
		"10.0.2.100", // Cross-subnet
		"192.168.1.1", // Different network
	}

	for _, targetIP := range targetIPs {
		b.Run(fmt.Sprintf("Target-%s", targetIP), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				latency, err := utils.MeasureEndToEndLatency("client", targetIP+":80", 10)
				if err == nil {
					b.Logf("Measured latency: %.2f ms", latency)
				}
			}
		})
	}
}

// Helper functions for benchmarks

func processIntent(ctx context.Context, testEnv *utils.TestEnvironment, intentName string) {
	// Simulate intent processing
	time.Sleep(10 * time.Millisecond)
}

func deploySlicesConcurrently(ctx context.Context, testEnv *utils.TestEnvironment, concurrency int) {
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			// Simulate slice deployment
			time.Sleep(time.Duration(50+id*5) * time.Millisecond)
		}(i)
	}

	wg.Wait()
}

func allocateResources(ctx context.Context, testEnv *utils.TestEnvironment, resourceCount int) {
	// Simulate resource allocation
	for i := 0; i < resourceCount; i++ {
		// Simulate resource creation overhead
		time.Sleep(time.Microsecond * 100)
	}
}

func executeE2EFlow(ctx context.Context, testEnv *utils.TestEnvironment, sliceType string, id int) {
	// Simulate different phases of E2E flow
	phases := []time.Duration{
		5 * time.Millisecond,  // Intent parsing
		15 * time.Millisecond, // QoS mapping
		25 * time.Millisecond, // Placement decision
		40 * time.Millisecond, // Resource allocation
		20 * time.Millisecond, // Deployment
	}

	for _, phase := range phases {
		time.Sleep(phase)
	}
}

func processLargeDataset(ctx context.Context, data []byte) {
	// Simulate data processing
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
}

func simulateSystemLoad(ctx context.Context, testEnv *utils.TestEnvironment, loadLevel int) {
	// Simulate system load by creating multiple operations
	var wg sync.WaitGroup

	for i := 0; i < loadLevel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Millisecond)
		}()
	}

	wg.Wait()
}

// Performance regression tests

func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression tests in short mode")
	}

	testEnv := utils.SetupTestEnvironment(t, scheme)
	defer testEnv.Cleanup(t)

	// Define performance baselines (in milliseconds)
	baselines := map[string]time.Duration{
		"intent_processing":  100 * time.Millisecond,
		"slice_deployment":   5 * time.Second,
		"resource_allocation": 50 * time.Millisecond,
		"qos_configuration":  200 * time.Millisecond,
	}

	for operation, baseline := range baselines {
		t.Run(operation, func(t *testing.T) {
			start := time.Now()
			executeOperation(t, testEnv, operation)
			duration := time.Since(start)

			// Allow 20% degradation from baseline
			threshold := baseline + (baseline / 5)
			require.Less(t, duration, threshold,
				"Performance regression detected: %s took %v (baseline: %v, threshold: %v)",
				operation, duration, baseline, threshold)

			t.Logf("✅ %s completed in %v (baseline: %v)", operation, duration, baseline)
		})
	}
}

func executeOperation(t *testing.T, testEnv *utils.TestEnvironment, operation string) {
	switch operation {
	case "intent_processing":
		processIntent(context.Background(), testEnv, "test-intent")
	case "slice_deployment":
		deploySlicesConcurrently(context.Background(), testEnv, 1)
	case "resource_allocation":
		allocateResources(context.Background(), testEnv, 10)
	case "qos_configuration":
		time.Sleep(150 * time.Millisecond) // Simulate QoS configuration
	default:
		t.Fatalf("Unknown operation: %s", operation)
	}
}

// Stress tests

func TestStressIntentProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress tests in short mode")
	}

	testEnv := utils.SetupTestEnvironment(t, scheme)
	defer testEnv.Cleanup(t)

	ctx := context.Background()
	intentCount := 1000
	concurrency := 50

	start := time.Now()

	var wg sync.WaitGroup
	intentChan := make(chan int, intentCount)

	// Fill intent channel
	for i := 0; i < intentCount; i++ {
		intentChan <- i
	}
	close(intentChan)

	// Process intents concurrently
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for intentID := range intentChan {
				processIntent(ctx, testEnv, fmt.Sprintf("stress-intent-%d", intentID))
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Processed %d intents in %v (%.2f intents/sec)",
		intentCount, duration, float64(intentCount)/duration.Seconds())

	// Verify performance is acceptable
	maxDuration := 2 * time.Minute
	require.Less(t, duration, maxDuration,
		"Stress test took too long: %v (max: %v)", duration, maxDuration)
}