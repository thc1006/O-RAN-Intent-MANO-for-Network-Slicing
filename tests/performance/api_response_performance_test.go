package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// API response time performance requirements
const (
	// API response time targets
	FastAPITarget    = 50 * time.Millisecond  // Critical operations (health, status)
	NormalAPITarget  = 100 * time.Millisecond // Normal operations (CRUD)
	SlowAPITarget    = 500 * time.Millisecond // Complex operations (deployment, analysis)

	// Throughput targets
	MinThroughputRPS = 100 // Minimum requests per second
	MaxConcurrency   = 50  // Maximum concurrent requests to test
)

// API performance test data structures
type APIPerformanceMetrics struct {
	Endpoint        string            `json:"endpoint"`
	Method          string            `json:"method"`
	ResponseTime    time.Duration     `json:"responseTime"`
	StatusCode      int               `json:"statusCode"`
	RequestSize     int64             `json:"requestSize"`
	ResponseSize    int64             `json:"responseSize"`
	Timestamp       time.Time         `json:"timestamp"`
	Success         bool              `json:"success"`
	ErrorMessage    string            `json:"errorMessage"`
	Headers         map[string]string `json:"headers"`
}

type APIPerformanceResult struct {
	TestName          string                  `json:"testName"`
	StartTime         time.Time               `json:"startTime"`
	EndTime           time.Time               `json:"endTime"`
	TotalDuration     time.Duration           `json:"totalDuration"`
	Metrics           []APIPerformanceMetrics `json:"metrics"`
	AverageResponseTime time.Duration         `json:"averageResponseTime"`
	MedianResponseTime  time.Duration         `json:"medianResponseTime"`
	P95ResponseTime     time.Duration         `json:"p95ResponseTime"`
	P99ResponseTime     time.Duration         `json:"p99ResponseTime"`
	MinResponseTime     time.Duration         `json:"minResponseTime"`
	MaxResponseTime     time.Duration         `json:"maxResponseTime"`
	TotalRequests       int                   `json:"totalRequests"`
	SuccessfulRequests  int                   `json:"successfulRequests"`
	FailedRequests      int                   `json:"failedRequests"`
	RequestsPerSecond   float64               `json:"requestsPerSecond"`
	SuccessRate         float64               `json:"successRate"`
	ThroughputMBps      float64               `json:"throughputMBps"`
}

// Mock API server for performance testing
type PerformanceAPIServer struct {
	server     *httptest.Server
	router     *gin.Engine
	latencies  map[string]time.Duration
	errorRates map[string]float64
	mu         sync.RWMutex
	metrics    []APIPerformanceMetrics
}

func NewPerformanceAPIServer() *PerformanceAPIServer {
	gin.SetMode(gin.TestMode)

	server := &PerformanceAPIServer{
		router:     gin.New(),
		latencies:  make(map[string]time.Duration),
		errorRates: make(map[string]float64),
		metrics:    make([]APIPerformanceMetrics, 0),
	}

	server.setupRoutes()
	server.server = httptest.NewServer(server.router)

	return server
}

func (s *PerformanceAPIServer) setupRoutes() {
	// Add middleware for metrics collection
	s.router.Use(s.metricsMiddleware())

	// Health and status endpoints (should be fast)
	s.router.GET("/health", s.simulateLatency(10*time.Millisecond), s.healthHandler)
	s.router.GET("/ready", s.simulateLatency(20*time.Millisecond), s.readinessHandler)
	s.router.GET("/version", s.simulateLatency(5*time.Millisecond), s.versionHandler)

	// Slice management endpoints (normal speed)
	api := s.router.Group("/api/v1")
	{
		api.GET("/slices", s.simulateLatency(80*time.Millisecond), s.getSlicesHandler)
		api.GET("/slices/:id", s.simulateLatency(60*time.Millisecond), s.getSliceHandler)
		api.POST("/slices", s.simulateLatency(150*time.Millisecond), s.createSliceHandler)
		api.PUT("/slices/:id", s.simulateLatency(120*time.Millisecond), s.updateSliceHandler)
		api.DELETE("/slices/:id", s.simulateLatency(100*time.Millisecond), s.deleteSliceHandler)

		// VNF management endpoints (normal speed)
		api.GET("/vnfs", s.simulateLatency(90*time.Millisecond), s.getVNFsHandler)
		api.POST("/vnfs/deploy", s.simulateLatency(300*time.Millisecond), s.deployVNFHandler)
		api.DELETE("/vnfs/:id", s.simulateLatency(200*time.Millisecond), s.undeployVNFHandler)

		// Status and monitoring endpoints (fast)
		api.GET("/status", s.simulateLatency(30*time.Millisecond), s.statusHandler)
		api.GET("/metrics", s.simulateLatency(40*time.Millisecond), s.metricsHandler)

		// Complex analysis endpoints (slower)
		api.POST("/analyze/performance", s.simulateLatency(400*time.Millisecond), s.performanceAnalysisHandler)
		api.POST("/optimize/placement", s.simulateLatency(450*time.Millisecond), s.placementOptimizationHandler)
		api.GET("/reports/detailed", s.simulateLatency(350*time.Millisecond), s.detailedReportHandler)
	}
}

func (s *PerformanceAPIServer) simulateLatency(baseLatency time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add some random variation (±20%)
		variation := time.Duration(float64(baseLatency) * 0.2 * (rand.Float64() - 0.5))
		latency := baseLatency + variation

		// Store expected latency for endpoint
		endpoint := c.Request.Method + " " + c.FullPath()
		s.mu.Lock()
		s.latencies[endpoint] = latency
		s.mu.Unlock()

		time.Sleep(latency)
		c.Next()
	}
}

func (s *PerformanceAPIServer) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		responseTime := time.Since(startTime)

		metrics := APIPerformanceMetrics{
			Endpoint:     c.Request.Method + " " + c.FullPath(),
			Method:       c.Request.Method,
			ResponseTime: responseTime,
			StatusCode:   c.Writer.Status(),
			RequestSize:  c.Request.ContentLength,
			ResponseSize: int64(c.Writer.Size()),
			Timestamp:    startTime,
			Success:      c.Writer.Status() < 400,
			Headers: map[string]string{
				"Content-Type": c.Writer.Header().Get("Content-Type"),
			},
		}

		if !metrics.Success {
			if errors := c.Errors; len(errors) > 0 {
				metrics.ErrorMessage = errors[0].Error()
			}
		}

		s.mu.Lock()
		s.metrics = append(s.metrics, metrics)
		s.mu.Unlock()
	}
}

// Handler implementations
func (s *PerformanceAPIServer) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func (s *PerformanceAPIServer) readinessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"timestamp": time.Now(),
		"checks": map[string]bool{
			"database": true,
			"storage":  true,
			"network":  true,
		},
	})
}

func (s *PerformanceAPIServer) versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": "v1.0.0",
		"build":   "test-build",
	})
}

func (s *PerformanceAPIServer) getSlicesHandler(c *gin.Context) {
	slices := make([]map[string]interface{}, 10)
	for i := 0; i < 10; i++ {
		slices[i] = map[string]interface{}{
			"id":        fmt.Sprintf("slice-%d", i),
			"type":      []string{"eMBB", "URLLC", "mMTC"}[i%3],
			"status":    "active",
			"created":   time.Now().Add(-time.Duration(i)*time.Hour),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"slices": slices,
		"count":  len(slices),
		"total":  100,
	})
}

func (s *PerformanceAPIServer) getSliceHandler(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":       id,
		"type":     "eMBB",
		"status":   "active",
		"created":  time.Now().Add(-2*time.Hour),
		"resources": map[string]interface{}{
			"cpu":    "4",
			"memory": "8Gi",
			"vnfs":   []string{"amf", "smf", "upf"},
		},
	})
}

func (s *PerformanceAPIServer) createSliceHandler(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      fmt.Sprintf("slice-%d", time.Now().Unix()),
		"status":  "creating",
		"message": "Slice creation initiated",
		"request": request,
	})
}

func (s *PerformanceAPIServer) updateSliceHandler(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid updates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"status":  "updated",
		"message": fmt.Sprintf("Slice %s updated successfully", id),
		"updates": updates,
	})
}

func (s *PerformanceAPIServer) deleteSliceHandler(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"status":  "deleting",
		"message": fmt.Sprintf("Slice %s deletion initiated", id),
	})
}

func (s *PerformanceAPIServer) getVNFsHandler(c *gin.Context) {
	vnfs := make([]map[string]interface{}, 5)
	for i := 0; i < 5; i++ {
		vnfs[i] = map[string]interface{}{
			"id":     fmt.Sprintf("vnf-%d", i),
			"type":   []string{"AMF", "SMF", "UPF", "PCF", "UDM"}[i],
			"status": "running",
			"slice":  fmt.Sprintf("slice-%d", i%3),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"vnfs":  vnfs,
		"count": len(vnfs),
	})
}

func (s *PerformanceAPIServer) deployVNFHandler(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      fmt.Sprintf("vnf-deploy-%d", time.Now().Unix()),
		"status":  "deploying",
		"message": "VNF deployment initiated",
		"request": request,
	})
}

func (s *PerformanceAPIServer) undeployVNFHandler(c *gin.Context) {
	id := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"status":  "undeploying",
		"message": fmt.Sprintf("VNF %s undeployment initiated", id),
	})
}

func (s *PerformanceAPIServer) statusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "operational",
		"uptime": "72h30m",
		"stats": map[string]interface{}{
			"active_slices": 15,
			"running_vnfs": 45,
			"cpu_usage":    "45%",
			"memory_usage": "62%",
		},
	})
}

func (s *PerformanceAPIServer) metricsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"metrics": map[string]interface{}{
			"requests_total":      12450,
			"requests_per_second": 15.7,
			"average_latency_ms":  85.3,
			"error_rate":          0.02,
			"throughput_mbps":     125.6,
		},
	})
}

func (s *PerformanceAPIServer) performanceAnalysisHandler(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Simulate complex analysis
	analysis := map[string]interface{}{
		"analysis_id":  fmt.Sprintf("analysis-%d", time.Now().Unix()),
		"status":       "completed",
		"duration":     "2.3s",
		"results": map[string]interface{}{
			"performance_score": 85.7,
			"bottlenecks":      []string{"cpu", "network"},
			"recommendations":  []string{"scale_up", "optimize_routing"},
		},
	}

	c.JSON(http.StatusOK, analysis)
}

func (s *PerformanceAPIServer) placementOptimizationHandler(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	optimization := map[string]interface{}{
		"optimization_id": fmt.Sprintf("opt-%d", time.Now().Unix()),
		"status":          "completed",
		"improvements": map[string]interface{}{
			"latency_reduction":    "15%",
			"throughput_increase":  "8%",
			"resource_efficiency": "12%",
		},
		"placement_plan": []map[string]interface{}{
			{"vnf": "amf", "node": "node-1", "zone": "zone-a"},
			{"vnf": "smf", "node": "node-2", "zone": "zone-a"},
			{"vnf": "upf", "node": "node-3", "zone": "zone-b"},
		},
	}

	c.JSON(http.StatusOK, optimization)
}

func (s *PerformanceAPIServer) detailedReportHandler(c *gin.Context) {
	report := map[string]interface{}{
		"report_id":   fmt.Sprintf("report-%d", time.Now().Unix()),
		"generated":   time.Now(),
		"period":      "24h",
		"summary": map[string]interface{}{
			"total_slices":      25,
			"successful_deployments": 23,
			"failed_deployments":     2,
			"average_deploy_time":    "45s",
			"system_uptime":          "99.8%",
		},
		"performance": map[string]interface{}{
			"api_response_avg":  "78ms",
			"throughput_peak":   "150Mbps",
			"latency_p99":       "12ms",
		},
		"resources": map[string]interface{}{
			"cpu_avg":    "42%",
			"memory_avg": "58%",
			"disk_usage": "35%",
		},
	}

	c.JSON(http.StatusOK, report)
}

func (s *PerformanceAPIServer) Close() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *PerformanceAPIServer) GetMetrics() []APIPerformanceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]APIPerformanceMetrics, len(s.metrics))
	copy(result, s.metrics)
	return result
}

func (s *PerformanceAPIServer) ClearMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = s.metrics[:0]
}

// Performance tests

func TestAPIResponseTimePerformance(t *testing.T) {
	server := NewPerformanceAPIServer()
	defer server.Close()

	testCases := []struct {
		name           string
		method         string
		endpoint       string
		body           interface{}
		expectedTarget time.Duration
		category       string
	}{
		// Fast endpoints (health, status)
		{"Health check", "GET", "/health", nil, FastAPITarget, "fast"},
		{"Readiness check", "GET", "/ready", nil, FastAPITarget, "fast"},
		{"Version info", "GET", "/version", nil, FastAPITarget, "fast"},

		// Normal endpoints (CRUD operations)
		{"List slices", "GET", "/api/v1/slices", nil, NormalAPITarget, "normal"},
		{"Get slice", "GET", "/api/v1/slices/test-slice", nil, NormalAPITarget, "normal"},
		{"Update slice", "PUT", "/api/v1/slices/test-slice", map[string]string{"status": "updated"}, NormalAPITarget, "normal"},
		{"Delete slice", "DELETE", "/api/v1/slices/test-slice", nil, NormalAPITarget, "normal"},
		{"List VNFs", "GET", "/api/v1/vnfs", nil, NormalAPITarget, "normal"},
		{"System status", "GET", "/api/v1/status", nil, NormalAPITarget, "normal"},
		{"System metrics", "GET", "/api/v1/metrics", nil, NormalAPITarget, "normal"},

		// Slow endpoints (complex operations)
		{"Create slice", "POST", "/api/v1/slices", map[string]interface{}{
			"type": "eMBB", "resources": map[string]string{"cpu": "4", "memory": "8Gi"},
		}, SlowAPITarget, "slow"},
		{"Deploy VNF", "POST", "/api/v1/vnfs/deploy", map[string]interface{}{
			"type": "UPF", "slice": "test-slice",
		}, SlowAPITarget, "slow"},
		{"Performance analysis", "POST", "/api/v1/analyze/performance", map[string]interface{}{
			"slice_id": "test-slice", "duration": "1h",
		}, SlowAPITarget, "slow"},
		{"Placement optimization", "POST", "/api/v1/optimize/placement", map[string]interface{}{
			"vnfs": []string{"amf", "smf", "upf"},
		}, SlowAPITarget, "slow"},
		{"Detailed report", "GET", "/api/v1/reports/detailed", nil, SlowAPITarget, "slow"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server.ClearMetrics()

			// Make request and measure response time
			startTime := time.Now()
			resp := s.makeRequest(t, server, tc.method, tc.endpoint, tc.body)
			responseTime := time.Since(startTime)

			// Verify response
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected successful response")
			resp.Body.Close()

			// Verify response time meets target
			assert.Less(t, responseTime, tc.expectedTarget,
				"Response time %v exceeds target %v for %s endpoint",
				responseTime, tc.expectedTarget, tc.category)

			// Log performance metrics
			t.Logf("%s: %v (target: <%v)", tc.name, responseTime, tc.expectedTarget)

			// Verify metrics were collected
			metrics := server.GetMetrics()
			require.Len(t, metrics, 1)
			assert.True(t, metrics[0].Success)
			assert.Equal(t, tc.method, metrics[0].Method)
		})
	}
}

func TestAPIThroughputPerformance(t *testing.T) {
	server := NewPerformanceAPIServer()
	defer server.Close()

	t.Run("High throughput test", func(t *testing.T) {
		const duration = 10 * time.Second
		const targetRPS = MinThroughputRPS

		server.ClearMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		startTime := time.Now()
		requestCount := 0
		successCount := 0

		// Send requests at target rate
		ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				goto results
			case <-ticker.C:
				requestCount++

				go func() {
					resp := s.makeRequest(t, server, "GET", "/health", nil)
					if resp.StatusCode == http.StatusOK {
						successCount++
					}
					resp.Body.Close()
				}()
			}
		}

	results:
		actualDuration := time.Since(startTime)
		time.Sleep(100 * time.Millisecond) // Wait for pending requests

		actualRPS := float64(requestCount) / actualDuration.Seconds()
		successRate := float64(successCount) / float64(requestCount) * 100

		t.Logf("Throughput Test Results:")
		t.Logf("  Duration: %v", actualDuration)
		t.Logf("  Total requests: %d", requestCount)
		t.Logf("  Successful requests: %d", successCount)
		t.Logf("  Target RPS: %d", targetRPS)
		t.Logf("  Actual RPS: %.1f", actualRPS)
		t.Logf("  Success rate: %.1f%%", successRate)

		// Verify throughput requirements
		assert.GreaterOrEqual(t, actualRPS, float64(targetRPS)*0.9,
			"Actual RPS %.1f below 90%% of target %d", actualRPS, targetRPS)

		assert.GreaterOrEqual(t, successRate, 95.0,
			"Success rate %.1f%% below 95%%", successRate)

		// Verify collected metrics
		metrics := server.GetMetrics()
		assert.GreaterOrEqual(t, len(metrics), requestCount/2,
			"Expected at least half the requests to be recorded in metrics")
	})
}

func TestAPIConcurrencyPerformance(t *testing.T) {
	server := NewPerformanceAPIServer()
	defer server.Close()

	concurrencyLevels := []int{1, 5, 10, 20, MaxConcurrency}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(t *testing.T) {
			server.ClearMetrics()

			const requestsPerWorker = 10

			var wg sync.WaitGroup
			startTime := time.Now()

			// Launch concurrent workers
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for j := 0; j < requestsPerWorker; j++ {
						resp := s.makeRequest(t, server, "GET", "/api/v1/slices", nil)
						resp.Body.Close()
					}
				}(i)
			}

			wg.Wait()
			totalDuration := time.Since(startTime)

			// Analyze results
			metrics := server.GetMetrics()
			totalRequests := concurrency * requestsPerWorker
			successfulRequests := 0

			var totalResponseTime time.Duration
			for _, metric := range metrics {
				if metric.Success {
					successfulRequests++
					totalResponseTime += metric.ResponseTime
				}
			}

			averageResponseTime := totalResponseTime / time.Duration(len(metrics))
			requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
			successRate := float64(successfulRequests) / float64(totalRequests) * 100

			t.Logf("Concurrency %d Results:", concurrency)
			t.Logf("  Total requests: %d", totalRequests)
			t.Logf("  Successful requests: %d", successfulRequests)
			t.Logf("  Total duration: %v", totalDuration)
			t.Logf("  Average response time: %v", averageResponseTime)
			t.Logf("  Requests per second: %.1f", requestsPerSecond)
			t.Logf("  Success rate: %.1f%%", successRate)

			// Verify performance under concurrency
			assert.GreaterOrEqual(t, successRate, 90.0,
				"Success rate %.1f%% below 90%% under concurrency %d", successRate, concurrency)

			assert.Less(t, averageResponseTime, NormalAPITarget*2,
				"Average response time %v too high under concurrency %d", averageResponseTime, concurrency)

			// Verify metrics collection completeness
			assert.Equal(t, totalRequests, len(metrics),
				"Expected all requests to be recorded in metrics")
		})
	}
}

func TestAPIPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	server := NewPerformanceAPIServer()
	defer server.Close()

	t.Run("Sustained load test", func(t *testing.T) {
		const loadDuration = 30 * time.Second
		const targetRPS = 50
		const maxConcurrency = 20

		server.ClearMetrics()

		ctx, cancel := context.WithTimeout(context.Background(), loadDuration)
		defer cancel()

		startTime := time.Now()
		semaphore := make(chan struct{}, maxConcurrency)

		var requestCount int
		var mu sync.Mutex

		// Generate load at target rate
		ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				goto results
			case <-ticker.C:
				// Acquire semaphore (limit concurrency)
				select {
				case semaphore <- struct{}{}:
					mu.Lock()
					requestCount++
					mu.Unlock()

					go func() {
						defer func() { <-semaphore }()

						// Mix of different endpoint types
						endpoints := []struct {
							method string
							path   string
							body   interface{}
						}{
							{"GET", "/health", nil},
							{"GET", "/api/v1/slices", nil},
							{"GET", "/api/v1/status", nil},
							{"POST", "/api/v1/slices", map[string]string{"type": "eMBB"}},
						}

						endpoint := endpoints[requestCount%len(endpoints)]
						resp := s.makeRequest(t, server, endpoint.method, endpoint.path, endpoint.body)
						resp.Body.Close()
					}()
				default:
					// Skip request if max concurrency reached
				}
			}
		}

	results:
		actualDuration := time.Since(startTime)
		time.Sleep(500 * time.Millisecond) // Wait for pending requests

		// Analyze performance under load
		metrics := server.GetMetrics()

		result := analyzeAPIPerformance("Sustained Load Test", metrics, actualDuration)

		t.Logf("Sustained Load Test Results:")
		t.Logf("  Duration: %v", actualDuration)
		t.Logf("  Total requests: %d", result.TotalRequests)
		t.Logf("  Successful requests: %d", result.SuccessfulRequests)
		t.Logf("  Failed requests: %d", result.FailedRequests)
		t.Logf("  Requests per second: %.1f", result.RequestsPerSecond)
		t.Logf("  Success rate: %.1f%%", result.SuccessRate)
		t.Logf("  Average response time: %v", result.AverageResponseTime)
		t.Logf("  P95 response time: %v", result.P95ResponseTime)
		t.Logf("  P99 response time: %v", result.P99ResponseTime)

		// Verify sustained performance requirements
		assert.GreaterOrEqual(t, result.SuccessRate, 95.0,
			"Success rate %.1f%% below 95%% under sustained load", result.SuccessRate)

		assert.GreaterOrEqual(t, result.RequestsPerSecond, float64(targetRPS)*0.8,
			"Actual RPS %.1f below 80%% of target %d under load", result.RequestsPerSecond, targetRPS)

		assert.Less(t, result.P95ResponseTime, NormalAPITarget*3,
			"P95 response time %v too high under sustained load", result.P95ResponseTime)
	})
}

// Benchmark tests for API performance

func BenchmarkAPIEndpoints(b *testing.B) {
	server := NewPerformanceAPIServer()
	defer server.Close()

	benchmarks := []struct {
		name     string
		method   string
		endpoint string
		body     interface{}
	}{
		{"Health", "GET", "/health", nil},
		{"ListSlices", "GET", "/api/v1/slices", nil},
		{"GetSlice", "GET", "/api/v1/slices/test", nil},
		{"CreateSlice", "POST", "/api/v1/slices", map[string]string{"type": "eMBB"}},
		{"Status", "GET", "/api/v1/status", nil},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp := s.makeRequest(b, server, bm.method, bm.endpoint, bm.body)
				resp.Body.Close()
			}
		})
	}
}

func BenchmarkAPIConcurrency(b *testing.B) {
	server := NewPerformanceAPIServer()
	defer server.Close()

	concurrencyLevels := []int{1, 5, 10, 20}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					resp := s.makeRequest(b, server, "GET", "/api/v1/slices", nil)
					resp.Body.Close()
				}
			})
		})
	}
}

// Helper functions

func (s *PerformanceAPIServer) makeRequest(t testing.TB, server *PerformanceAPIServer, method, endpoint string, body interface{}) *http.Response {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, server.server.URL+endpoint, reqBody)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func analyzeAPIPerformance(testName string, metrics []APIPerformanceMetrics, duration time.Duration) APIPerformanceResult {
	result := APIPerformanceResult{
		TestName:    testName,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(duration),
		TotalDuration: duration,
		Metrics:     metrics,
		TotalRequests: len(metrics),
	}

	if len(metrics) == 0 {
		return result
	}

	// Calculate basic statistics
	var totalResponseTime time.Duration
	var totalBytes int64
	var responseTimes []time.Duration

	for _, metric := range metrics {
		if metric.Success {
			result.SuccessfulRequests++
		} else {
			result.FailedRequests++
		}

		totalResponseTime += metric.ResponseTime
		totalBytes += metric.ResponseSize
		responseTimes = append(responseTimes, metric.ResponseTime)
	}

	// Calculate derived metrics
	result.SuccessRate = float64(result.SuccessfulRequests) / float64(result.TotalRequests) * 100
	result.RequestsPerSecond = float64(result.TotalRequests) / duration.Seconds()
	result.ThroughputMBps = float64(totalBytes) / (1024 * 1024) / duration.Seconds()

	if len(responseTimes) > 0 {
		result.AverageResponseTime = totalResponseTime / time.Duration(len(responseTimes))
		result.MinResponseTime = responseTimes[0]
		result.MaxResponseTime = responseTimes[0]

		// Find min/max
		for _, rt := range responseTimes {
			if rt < result.MinResponseTime {
				result.MinResponseTime = rt
			}
			if rt > result.MaxResponseTime {
				result.MaxResponseTime = rt
			}
		}

		// Calculate percentiles (simplified)
		if len(responseTimes) >= 20 {
			// Sort for percentiles
			for i := 0; i < len(responseTimes)-1; i++ {
				for j := 0; j < len(responseTimes)-i-1; j++ {
					if responseTimes[j] > responseTimes[j+1] {
						responseTimes[j], responseTimes[j+1] = responseTimes[j+1], responseTimes[j]
					}
				}
			}

			result.MedianResponseTime = responseTimes[len(responseTimes)/2]
			result.P95ResponseTime = responseTimes[int(float64(len(responseTimes))*0.95)]
			result.P99ResponseTime = responseTimes[int(float64(len(responseTimes))*0.99)]
		}
	}

	return result
}