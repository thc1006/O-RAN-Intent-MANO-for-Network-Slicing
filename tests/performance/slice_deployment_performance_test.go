package performance

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Performance test requirements based on thesis targets
const (
	// Deployment time target: <60s (as per project requirement)
	MaxDeploymentTime = 60 * time.Second

	// eMBB throughput target: ≥4.57 Mbps
	EMBBThroughputTarget = 4.57

	// URLLC latency target: ≤6.3 ms
	URLLCLatencyTarget = 6.3

	// mMTC connection density target: 10,000 devices/km²
	MMTCConnectionTarget = 10000

	// Concurrent slice handling target: 10+ slices
	ConcurrentSliceTarget = 10

	// API response time target: <100ms
	APIResponseTarget = 100 * time.Millisecond
)

// Performance test data structures
type SliceDeploymentMetrics struct {
	SliceID          string        `json:"sliceId"`
	SliceType        string        `json:"sliceType"`
	StartTime        time.Time     `json:"startTime"`
	EndTime          time.Time     `json:"endTime"`
	DeploymentTime   time.Duration `json:"deploymentTime"`
	ValidationTime   time.Duration `json:"validationTime"`
	PlanningTime     time.Duration `json:"planningTime"`
	ExecutionTime    time.Duration `json:"executionTime"`
	ActivationTime   time.Duration `json:"activationTime"`
	Success          bool          `json:"success"`
	ErrorMessage     string        `json:"errorMessage"`
	ResourceMetrics  ResourceMetrics `json:"resourceMetrics"`
}

type ResourceMetrics struct {
	CPUUsage       float64 `json:"cpuUsage"`
	MemoryUsage    float64 `json:"memoryUsage"`
	NetworkUsage   float64 `json:"networkUsage"`
	StorageUsage   float64 `json:"storageUsage"`
	PodCount       int     `json:"podCount"`
	NodeCount      int     `json:"nodeCount"`
}

type PerformanceTestResult struct {
	TestName           string                    `json:"testName"`
	StartTime          time.Time                 `json:"startTime"`
	EndTime            time.Time                 `json:"endTime"`
	TotalDuration      time.Duration             `json:"totalDuration"`
	SliceMetrics       []SliceDeploymentMetrics  `json:"sliceMetrics"`
	SuccessRate        float64                   `json:"successRate"`
	AverageDeployTime  time.Duration             `json:"averageDeployTime"`
	MedianDeployTime   time.Duration             `json:"medianDeployTime"`
	P95DeployTime      time.Duration             `json:"p95DeployTime"`
	P99DeployTime      time.Duration             `json:"p99DeployTime"`
	ThroughputMetrics  ThroughputPerformance     `json:"throughputMetrics"`
	LatencyMetrics     LatencyPerformance        `json:"latencyMetrics"`
	ThesisCompliance   ThesisComplianceMetrics   `json:"thesisCompliance"`
}

type ThroughputPerformance struct {
	EMBBMbps          float64 `json:"embbMbps"`
	URLLCMbps         float64 `json:"urllcMbps"`
	MMTCMbps          float64 `json:"mmtcMbps"`
	AggregatedMbps    float64 `json:"aggregatedMbps"`
	PeakMbps          float64 `json:"peakMbps"`
}

type LatencyPerformance struct {
	EMBBLatencyMs     float64 `json:"embbLatencyMs"`
	URLLCLatencyMs    float64 `json:"urllcLatencyMs"`
	MMTCLatencyMs     float64 `json:"mmtcLatencyMs"`
	AverageLatencyMs  float64 `json:"averageLatencyMs"`
	P99LatencyMs      float64 `json:"p99LatencyMs"`
}

type ThesisComplianceMetrics struct {
	DeploymentTimeCompliant bool    `json:"deploymentTimeCompliant"`
	ThroughputCompliant     bool    `json:"throughputCompliant"`
	LatencyCompliant        bool    `json:"latencyCompliant"`
	OverallCompliance       float64 `json:"overallCompliance"`
	ComplianceDetails       map[string]bool `json:"complianceDetails"`
}

// Mock slice deployment simulator
type SliceDeploymentSimulator struct {
	orchestratorLatency time.Duration
	vnfOperatorLatency  time.Duration
	dmsLatency          time.Duration
	networkLatency      time.Duration
	errorRate           float64
	mu                  sync.Mutex
	deployedSlices      map[string]*SliceDeploymentMetrics
}

func NewSliceDeploymentSimulator() *SliceDeploymentSimulator {
	return &SliceDeploymentSimulator{
		orchestratorLatency: 500 * time.Millisecond,
		vnfOperatorLatency:  2 * time.Second,
		dmsLatency:          1 * time.Second,
		networkLatency:      200 * time.Millisecond,
		errorRate:           0.05, // 5% error rate
		deployedSlices:      make(map[string]*SliceDeploymentMetrics),
	}
}

func (s *SliceDeploymentSimulator) DeploySlice(ctx context.Context, sliceID, sliceType string) (*SliceDeploymentMetrics, error) {
	metrics := &SliceDeploymentMetrics{
		SliceID:   sliceID,
		SliceType: sliceType,
		StartTime: time.Now(),
		Success:   true,
	}

	// Simulate random failure
	if rand.Float64() < s.errorRate {
		metrics.Success = false
		metrics.ErrorMessage = "Simulated deployment failure"
		metrics.EndTime = time.Now()
		metrics.DeploymentTime = metrics.EndTime.Sub(metrics.StartTime)
		return metrics, fmt.Errorf("deployment failed: %s", metrics.ErrorMessage)
	}

	// Simulate deployment phases with realistic timings
	validationStart := time.Now()
	time.Sleep(s.orchestratorLatency + time.Duration(rand.Intn(200))*time.Millisecond)
	metrics.ValidationTime = time.Since(validationStart)

	planningStart := time.Now()
	time.Sleep(s.orchestratorLatency/2 + time.Duration(rand.Intn(300))*time.Millisecond)
	metrics.PlanningTime = time.Since(planningStart)

	executionStart := time.Now()
	// VNF deployment varies by slice type
	var execTime time.Duration
	switch sliceType {
	case "eMBB":
		execTime = s.vnfOperatorLatency + time.Duration(rand.Intn(1000))*time.Millisecond
	case "URLLC":
		execTime = s.vnfOperatorLatency/2 + time.Duration(rand.Intn(500))*time.Millisecond // Faster for URLLC
	case "mMTC":
		execTime = s.vnfOperatorLatency*2 + time.Duration(rand.Intn(2000))*time.Millisecond // Slower for mMTC
	default:
		execTime = s.vnfOperatorLatency + time.Duration(rand.Intn(1000))*time.Millisecond
	}
	time.Sleep(execTime)
	metrics.ExecutionTime = time.Since(executionStart)

	activationStart := time.Now()
	time.Sleep(s.dmsLatency + s.networkLatency + time.Duration(rand.Intn(300))*time.Millisecond)
	metrics.ActivationTime = time.Since(activationStart)

	metrics.EndTime = time.Now()
	metrics.DeploymentTime = metrics.EndTime.Sub(metrics.StartTime)

	// Simulate resource usage
	metrics.ResourceMetrics = ResourceMetrics{
		CPUUsage:     30 + rand.Float64()*40,     // 30-70%
		MemoryUsage:  40 + rand.Float64()*30,     // 40-70%
		NetworkUsage: 10 + rand.Float64()*20,     // 10-30%
		StorageUsage: 20 + rand.Float64()*15,     // 20-35%
		PodCount:     3 + rand.Intn(7),           // 3-9 pods
		NodeCount:    1 + rand.Intn(3),           // 1-3 nodes
	}

	s.mu.Lock()
	s.deployedSlices[sliceID] = metrics
	s.mu.Unlock()

	return metrics, nil
}

func (s *SliceDeploymentSimulator) GetPerformanceMetrics(sliceType string) (ThroughputPerformance, LatencyPerformance) {
	// Simulate realistic performance based on slice type
	throughput := ThroughputPerformance{}
	latency := LatencyPerformance{}

	switch sliceType {
	case "eMBB":
		throughput.EMBBMbps = 4.5 + rand.Float64()*1.0 // 4.5-5.5 Mbps
		latency.EMBBLatencyMs = 10 + rand.Float64()*5   // 10-15 ms
	case "URLLC":
		throughput.URLLCMbps = 1.0 + rand.Float64()*0.5 // 1.0-1.5 Mbps
		latency.URLLCLatencyMs = 1 + rand.Float64()*5    // 1-6 ms
	case "mMTC":
		throughput.MMTCMbps = 0.1 + rand.Float64()*0.2 // 0.1-0.3 Mbps
		latency.MMTCLatencyMs = 50 + rand.Float64()*50  // 50-100 ms
	}

	return throughput, latency
}

// Performance tests

func TestSliceDeploymentLatencyPerformance(t *testing.T) {
	t.Run("Single slice deployment latency", func(t *testing.T) {
		simulator := NewSliceDeploymentSimulator()
		ctx := context.Background()

		sliceTypes := []string{"eMBB", "URLLC", "mMTC"}

		for _, sliceType := range sliceTypes {
			t.Run(fmt.Sprintf("%s_deployment", sliceType), func(t *testing.T) {
				sliceID := fmt.Sprintf("perf-test-%s-%d", sliceType, time.Now().UnixNano())

				startTime := time.Now()
				metrics, err := simulator.DeploySlice(ctx, sliceID, sliceType)
				deployTime := time.Since(startTime)

				if err != nil {
					t.Logf("Deployment failed (acceptable for performance test): %v", err)
					return
				}

				require.NotNil(t, metrics)
				assert.True(t, metrics.Success)
				assert.Equal(t, sliceID, metrics.SliceID)
				assert.Equal(t, sliceType, metrics.SliceType)

				// Verify deployment time meets target
				assert.Less(t, deployTime, MaxDeploymentTime,
					"Deployment time %v exceeds target %v for slice type %s",
					deployTime, MaxDeploymentTime, sliceType)

				// Log performance metrics
				t.Logf("Slice %s (%s) deployed in %v", sliceID, sliceType, deployTime)
				t.Logf("  Validation: %v", metrics.ValidationTime)
				t.Logf("  Planning: %v", metrics.PlanningTime)
				t.Logf("  Execution: %v", metrics.ExecutionTime)
				t.Logf("  Activation: %v", metrics.ActivationTime)
				t.Logf("  CPU Usage: %.1f%%", metrics.ResourceMetrics.CPUUsage)
				t.Logf("  Memory Usage: %.1f%%", metrics.ResourceMetrics.MemoryUsage)
			})
		}
	})
}

func TestConcurrentSliceDeploymentPerformance(t *testing.T) {
	t.Run("Concurrent slice deployment", func(t *testing.T) {
		simulator := NewSliceDeploymentSimulator()
		ctx := context.Background()

		const numSlices = ConcurrentSliceTarget
		var wg sync.WaitGroup
		results := make(chan *SliceDeploymentMetrics, numSlices)
		errors := make(chan error, numSlices)

		startTime := time.Now()

		// Deploy slices concurrently
		for i := 0; i < numSlices; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				sliceTypes := []string{"eMBB", "URLLC", "mMTC"}
				sliceType := sliceTypes[id%len(sliceTypes)]
				sliceID := fmt.Sprintf("concurrent-slice-%d", id)

				metrics, err := simulator.DeploySlice(ctx, sliceID, sliceType)
				if err != nil {
					errors <- err
					return
				}
				results <- metrics
			}(i)
		}

		wg.Wait()
		close(results)
		close(errors)

		totalTime := time.Since(startTime)

		// Collect results
		var successfulDeployments []*SliceDeploymentMetrics
		var failedCount int

		for metrics := range results {
			successfulDeployments = append(successfulDeployments, metrics)
		}

		for range errors {
			failedCount++
		}

		// Calculate success rate
		successRate := float64(len(successfulDeployments)) / float64(numSlices) * 100

		// Verify performance requirements
		assert.GreaterOrEqual(t, len(successfulDeployments), int(numSlices*0.8),
			"Success rate should be at least 80%")

		assert.Less(t, totalTime, MaxDeploymentTime*2,
			"Total concurrent deployment time %v exceeds reasonable threshold", totalTime)

		// Calculate statistics
		var totalDeployTime time.Duration
		for _, metrics := range successfulDeployments {
			totalDeployTime += metrics.DeploymentTime
		}

		avgDeployTime := totalDeployTime / time.Duration(len(successfulDeployments))

		// Log performance metrics
		t.Logf("Concurrent deployment results:")
		t.Logf("  Total slices: %d", numSlices)
		t.Logf("  Successful: %d", len(successfulDeployments))
		t.Logf("  Failed: %d", failedCount)
		t.Logf("  Success rate: %.1f%%", successRate)
		t.Logf("  Total time: %v", totalTime)
		t.Logf("  Average deployment time: %v", avgDeployTime)
		t.Logf("  Concurrent efficiency: %.1f", float64(len(successfulDeployments))/totalTime.Seconds())
	})
}

func TestSliceTypeSpecificPerformance(t *testing.T) {
	simulator := NewSliceDeploymentSimulator()
	ctx := context.Background()

	testCases := []struct {
		sliceType           string
		maxDeploymentTime   time.Duration
		minThroughput       float64
		maxLatency          float64
		description         string
	}{
		{
			sliceType:         "eMBB",
			maxDeploymentTime: 45 * time.Second,
			minThroughput:     EMBBThroughputTarget,
			maxLatency:        20,
			description:       "Enhanced Mobile Broadband",
		},
		{
			sliceType:         "URLLC",
			maxDeploymentTime: 30 * time.Second, // Faster deployment for URLLC
			minThroughput:     1.0,
			maxLatency:        URLLCLatencyTarget,
			description:       "Ultra-Reliable Low-Latency Communications",
		},
		{
			sliceType:         "mMTC",
			maxDeploymentTime: 90 * time.Second, // Slower deployment acceptable for mMTC
			minThroughput:     0.1,
			maxLatency:        100,
			description:       "Massive Machine-Type Communications",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.sliceType, func(t *testing.T) {
			sliceID := fmt.Sprintf("type-perf-%s-%d", tc.sliceType, time.Now().UnixNano())

			// Deploy slice
			metrics, err := simulator.DeploySlice(ctx, sliceID, tc.sliceType)
			require.NoError(t, err)
			require.NotNil(t, metrics)

			// Verify deployment time
			assert.Less(t, metrics.DeploymentTime, tc.maxDeploymentTime,
				"%s deployment time %v exceeds target %v",
				tc.description, metrics.DeploymentTime, tc.maxDeploymentTime)

			// Get performance metrics
			throughput, latency := simulator.GetPerformanceMetrics(tc.sliceType)

			// Verify throughput based on slice type
			switch tc.sliceType {
			case "eMBB":
				assert.GreaterOrEqual(t, throughput.EMBBMbps, tc.minThroughput,
					"eMBB throughput %.2f Mbps below target %.2f Mbps",
					throughput.EMBBMbps, tc.minThroughput)
			case "URLLC":
				assert.LessOrEqual(t, latency.URLLCLatencyMs, tc.maxLatency,
					"URLLC latency %.2f ms exceeds target %.2f ms",
					latency.URLLCLatencyMs, tc.maxLatency)
			case "mMTC":
				// mMTC focuses on connection density rather than individual performance
				assert.GreaterOrEqual(t, throughput.MMTCMbps, tc.minThroughput,
					"mMTC throughput %.2f Mbps below minimum %.2f Mbps",
					throughput.MMTCMbps, tc.minThroughput)
			}

			t.Logf("%s Performance:", tc.description)
			t.Logf("  Deployment time: %v (target: <%v)", metrics.DeploymentTime, tc.maxDeploymentTime)
			t.Logf("  Throughput: %.2f Mbps (target: >%.2f)", getThroughputForType(throughput, tc.sliceType), tc.minThroughput)
			t.Logf("  Latency: %.2f ms (target: <%.2f)", getLatencyForType(latency, tc.sliceType), tc.maxLatency)
		})
	}
}

func TestThesisValidationPerformance(t *testing.T) {
	t.Run("Thesis compliance validation", func(t *testing.T) {
		simulator := NewSliceDeploymentSimulator()
		ctx := context.Background()

		result := &PerformanceTestResult{
			TestName:  "Thesis Validation Test",
			StartTime: time.Now(),
		}

		// Test each slice type according to thesis requirements
		sliceTypes := []string{"eMBB", "URLLC", "mMTC"}
		const testsPerType = 5

		for _, sliceType := range sliceTypes {
			for i := 0; i < testsPerType; i++ {
				sliceID := fmt.Sprintf("thesis-%s-%d", sliceType, i)

				metrics, err := simulator.DeploySlice(ctx, sliceID, sliceType)
				if err != nil {
					continue // Skip failed deployments for thesis validation
				}

				result.SliceMetrics = append(result.SliceMetrics, *metrics)

				// Get performance metrics
				throughput, latency := simulator.GetPerformanceMetrics(sliceType)

				// Store in aggregated metrics
				switch sliceType {
				case "eMBB":
					result.ThroughputMetrics.EMBBMbps = throughput.EMBBMbps
					result.LatencyMetrics.EMBBLatencyMs = latency.EMBBLatencyMs
				case "URLLC":
					result.ThroughputMetrics.URLLCMbps = throughput.URLLCMbps
					result.LatencyMetrics.URLLCLatencyMs = latency.URLLCLatencyMs
				case "mMTC":
					result.ThroughputMetrics.MMTCMbps = throughput.MMTCMbps
					result.LatencyMetrics.MMTCLatencyMs = latency.MMTCLatencyMs
				}
			}
		}

		result.EndTime = time.Now()
		result.TotalDuration = result.EndTime.Sub(result.StartTime)

		// Calculate compliance metrics
		compliance := calculateThesisCompliance(result)
		result.ThesisCompliance = compliance

		// Calculate success rate
		successfulDeployments := 0
		totalDeployTime := time.Duration(0)
		for _, metrics := range result.SliceMetrics {
			if metrics.Success {
				successfulDeployments++
				totalDeployTime += metrics.DeploymentTime
			}
		}

		result.SuccessRate = float64(successfulDeployments) / float64(len(result.SliceMetrics)) * 100
		if successfulDeployments > 0 {
			result.AverageDeployTime = totalDeployTime / time.Duration(successfulDeployments)
		}

		// Verify thesis compliance requirements
		assert.GreaterOrEqual(t, compliance.OverallCompliance, 80.0,
			"Overall thesis compliance %.1f%% below 80%% target", compliance.OverallCompliance)

		assert.True(t, compliance.DeploymentTimeCompliant,
			"Deployment time not compliant with thesis requirements")

		if result.ThroughputMetrics.EMBBMbps > 0 {
			assert.GreaterOrEqual(t, result.ThroughputMetrics.EMBBMbps, EMBBThroughputTarget,
				"eMBB throughput %.2f Mbps below thesis target %.2f Mbps",
				result.ThroughputMetrics.EMBBMbps, EMBBThroughputTarget)
		}

		if result.LatencyMetrics.URLLCLatencyMs > 0 {
			assert.LessOrEqual(t, result.LatencyMetrics.URLLCLatencyMs, URLLCLatencyTarget,
				"URLLC latency %.2f ms exceeds thesis target %.2f ms",
				result.LatencyMetrics.URLLCLatencyMs, URLLCLatencyTarget)
		}

		// Log detailed thesis validation results
		t.Logf("Thesis Validation Results:")
		t.Logf("  Overall compliance: %.1f%%", compliance.OverallCompliance)
		t.Logf("  Success rate: %.1f%%", result.SuccessRate)
		t.Logf("  Average deployment time: %v", result.AverageDeployTime)
		t.Logf("  eMBB throughput: %.2f Mbps (target: ≥%.2f)", result.ThroughputMetrics.EMBBMbps, EMBBThroughputTarget)
		t.Logf("  URLLC latency: %.2f ms (target: ≤%.2f)", result.LatencyMetrics.URLLCLatencyMs, URLLCLatencyTarget)
		t.Logf("  Deployment time compliant: %v", compliance.DeploymentTimeCompliant)
		t.Logf("  Throughput compliant: %v", compliance.ThroughputCompliant)
		t.Logf("  Latency compliant: %v", compliance.LatencyCompliant)
	})
}

// Benchmark tests for performance measurement

func BenchmarkSliceDeployment(b *testing.B) {
	simulator := NewSliceDeploymentSimulator()
	ctx := context.Background()

	b.Run("eMBB", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sliceID := fmt.Sprintf("bench-embb-%d", i)
			simulator.DeploySlice(ctx, sliceID, "eMBB")
		}
	})

	b.Run("URLLC", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sliceID := fmt.Sprintf("bench-urllc-%d", i)
			simulator.DeploySlice(ctx, sliceID, "URLLC")
		}
	})

	b.Run("mMTC", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sliceID := fmt.Sprintf("bench-mmtc-%d", i)
			simulator.DeploySlice(ctx, sliceID, "mMTC")
		}
	})
}

func BenchmarkConcurrentSliceDeployment(b *testing.B) {
	simulator := NewSliceDeploymentSimulator()
	ctx := context.Background()

	concurrencyLevels := []int{1, 5, 10, 20}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for j := 0; j < concurrency; j++ {
					wg.Add(1)
					go func(id int) {
						defer wg.Done()
						sliceID := fmt.Sprintf("bench-concurrent-%d-%d", i, id)
						simulator.DeploySlice(ctx, sliceID, "eMBB")
					}(j)
				}
				wg.Wait()
			}
		})
	}
}

// Helper functions

func getThroughputForType(throughput ThroughputPerformance, sliceType string) float64 {
	switch sliceType {
	case "eMBB":
		return throughput.EMBBMbps
	case "URLLC":
		return throughput.URLLCMbps
	case "mMTC":
		return throughput.MMTCMbps
	default:
		return 0
	}
}

func getLatencyForType(latency LatencyPerformance, sliceType string) float64 {
	switch sliceType {
	case "eMBB":
		return latency.EMBBLatencyMs
	case "URLLC":
		return latency.URLLCLatencyMs
	case "mMTC":
		return latency.MMTCLatencyMs
	default:
		return 0
	}
}

func calculateThesisCompliance(result *PerformanceTestResult) ThesisComplianceMetrics {
	compliance := ThesisComplianceMetrics{
		ComplianceDetails: make(map[string]bool),
	}

	complianceCount := 0
	totalChecks := 0

	// Check deployment time compliance
	if result.AverageDeployTime > 0 {
		compliance.DeploymentTimeCompliant = result.AverageDeployTime <= MaxDeploymentTime
		compliance.ComplianceDetails["deployment_time"] = compliance.DeploymentTimeCompliant
		if compliance.DeploymentTimeCompliant {
			complianceCount++
		}
		totalChecks++
	}

	// Check throughput compliance
	if result.ThroughputMetrics.EMBBMbps > 0 {
		compliance.ThroughputCompliant = result.ThroughputMetrics.EMBBMbps >= EMBBThroughputTarget
		compliance.ComplianceDetails["embb_throughput"] = compliance.ThroughputCompliant
		if compliance.ThroughputCompliant {
			complianceCount++
		}
		totalChecks++
	}

	// Check latency compliance
	if result.LatencyMetrics.URLLCLatencyMs > 0 {
		compliance.LatencyCompliant = result.LatencyMetrics.URLLCLatencyMs <= URLLCLatencyTarget
		compliance.ComplianceDetails["urllc_latency"] = compliance.LatencyCompliant
		if compliance.LatencyCompliant {
			complianceCount++
		}
		totalChecks++
	}

	// Calculate overall compliance
	if totalChecks > 0 {
		compliance.OverallCompliance = float64(complianceCount) / float64(totalChecks) * 100
	}

	return compliance
}

// Load testing functions for stress testing

func TestSliceDeploymentLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	t.Run("High load slice deployment", func(t *testing.T) {
		simulator := NewSliceDeploymentSimulator()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		const loadDuration = 2 * time.Minute
		const targetRate = 2 // slices per second

		startTime := time.Now()
		ticker := time.NewTicker(time.Second / targetRate)
		defer ticker.Stop()

		var deploymentCount int
		var successCount int
		var failureCount int

		for {
			select {
			case <-ctx.Done():
				t.Logf("Load test completed due to context timeout")
				return
			case <-ticker.C:
				if time.Since(startTime) >= loadDuration {
					ticker.Stop()
					goto results
				}

				deploymentCount++
				sliceID := fmt.Sprintf("load-test-%d", deploymentCount)
				sliceType := []string{"eMBB", "URLLC", "mMTC"}[deploymentCount%3]

				go func(id, sType string) {
					_, err := simulator.DeploySlice(ctx, id, sType)
					if err != nil {
						failureCount++
					} else {
						successCount++
					}
				}(sliceID, sliceType)
			}
		}

	results:
		time.Sleep(10 * time.Second) // Wait for pending deployments

		actualRate := float64(deploymentCount) / loadDuration.Seconds()
		successRate := float64(successCount) / float64(deploymentCount) * 100

		t.Logf("Load Test Results:")
		t.Logf("  Duration: %v", loadDuration)
		t.Logf("  Total deployments: %d", deploymentCount)
		t.Logf("  Successful: %d", successCount)
		t.Logf("  Failed: %d", failureCount)
		t.Logf("  Target rate: %.1f slices/sec", float64(targetRate))
		t.Logf("  Actual rate: %.1f slices/sec", actualRate)
		t.Logf("  Success rate: %.1f%%", successRate)

		// Verify load test requirements
		assert.GreaterOrEqual(t, actualRate, float64(targetRate)*0.8,
			"Actual deployment rate %.1f below 80%% of target %.1f", actualRate, float64(targetRate))

		assert.GreaterOrEqual(t, successRate, 70.0,
			"Success rate %.1f%% below 70%% under load", successRate)
	})
}