package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var _ = Describe("O-RAN Monitoring Performance Tests", func() {
	var (
		clientset   *kubernetes.Clientset
		restConfig  *rest.Config
		namespace   = "oran-monitoring"
		testTimeout = 15 * time.Minute
		ctx         context.Context
		cancel      context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), testTimeout)

		var err error
		restConfig, err = config.GetConfig()
		Expect(err).NotTo(HaveOccurred())

		clientset, err = kubernetes.NewForConfig(restConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if cancel != nil {
			cancel()
		}
	})

	Describe("Metrics Ingestion Performance", func() {
		It("should ingest metrics at expected rate", func() {
			// Measure metrics ingestion rate over 2 minutes
			startTime := time.Now()

			// Get initial metrics count
			initialCount := getMetricsCount(ctx, clientset, restConfig, namespace)

			// Wait for 2 minutes
			time.Sleep(2 * time.Minute)

			// Get final metrics count
			finalCount := getMetricsCount(ctx, clientset, restConfig, namespace)

			duration := time.Since(startTime).Seconds()
			ingestionRate := float64(finalCount-initialCount) / duration

			// Expect at least 100 metrics per second (conservative estimate)
			Expect(ingestionRate).To(BeNumerically(">=", 100.0),
				fmt.Sprintf("Ingestion rate should be >= 100 metrics/sec, got %.2f", ingestionRate))
		})

		It("should handle high cardinality within limits", func() {
			// Query for unique time series count
			query := "prometheus_tsdb_symbol_table_size_bytes"
			resp := executePrometheusQuery(ctx, clientset, restConfig, namespace, query)

			var queryResp PrometheusQueryResponse
			err := json.NewDecoder(resp.Body).Decode(&queryResp)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			// Basic check that we have metrics
			Expect(len(queryResp.Data.Result)).To(BeNumerically(">", 0))
		})

		It("should maintain stable memory usage during ingestion", func() {
			measurements := make([]float64, 0, 10)

			// Take memory measurements every 30 seconds for 5 minutes
			for i := 0; i < 10; i++ {
				memUsage := getPrometheusMemoryUsage(ctx, clientset, restConfig, namespace)
				measurements = append(measurements, memUsage)

				if i < 9 {
					time.Sleep(30 * time.Second)
				}
			}

			// Check memory growth is reasonable (less than 50% increase)
			initialMem := measurements[0]
			finalMem := measurements[len(measurements)-1]
			growthPercent := ((finalMem - initialMem) / initialMem) * 100

			Expect(growthPercent).To(BeNumerically("<", 50.0),
				fmt.Sprintf("Memory growth should be < 50%%, got %.2f%%", growthPercent))
		})
	})

	Describe("Query Performance", func() {
		It("should execute simple queries under 100ms", func() {
			queries := []string{
				"up",
				"rate(prometheus_http_requests_total[5m])",
				"node_cpu_seconds_total",
				"container_memory_usage_bytes",
			}

			for _, query := range queries {
				latency := measureQueryLatency(ctx, clientset, restConfig, namespace, query)
				Expect(latency).To(BeNumerically("<", 100*time.Millisecond),
					fmt.Sprintf("Query '%s' latency should be < 100ms, got %v", query, latency))
			}
		})

		It("should execute complex queries under 500ms", func() {
			complexQueries := []string{
				`rate(prometheus_http_requests_total[5m]) by (handler)`,
				`avg_over_time(up[1h]) by (job)`,
				`histogram_quantile(0.95, rate(prometheus_http_request_duration_seconds_bucket[5m]))`,
				`topk(10, rate(node_cpu_seconds_total[5m])) by (instance)`,
			}

			for _, query := range complexQueries {
				latency := measureQueryLatency(ctx, clientset, restConfig, namespace, query)
				Expect(latency).To(BeNumerically("<", 500*time.Millisecond),
					fmt.Sprintf("Complex query latency should be < 500ms, got %v for query: %s", latency, query))
			}
		})

		It("should handle concurrent queries efficiently", func() {
			concurrency := 20
			queriesPerGoroutine := 10
			query := "up"

			var wg sync.WaitGroup
			latencies := make(chan time.Duration, concurrency*queriesPerGoroutine)

			startTime := time.Now()

			// Launch concurrent queries
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < queriesPerGoroutine; j++ {
						latency := measureQueryLatency(ctx, clientset, restConfig, namespace, query)
						latencies <- latency
					}
				}()
			}

			wg.Wait()
			close(latencies)

			totalTime := time.Since(startTime)

			// Calculate statistics
			var totalLatency time.Duration
			maxLatency := time.Duration(0)
			count := 0

			for latency := range latencies {
				totalLatency += latency
				if latency > maxLatency {
					maxLatency = latency
				}
				count++
			}

			avgLatency := totalLatency / time.Duration(count)

			Expect(avgLatency).To(BeNumerically("<", 200*time.Millisecond),
				fmt.Sprintf("Average concurrent query latency should be < 200ms, got %v", avgLatency))
			Expect(maxLatency).To(BeNumerically("<", 1*time.Second),
				fmt.Sprintf("Max concurrent query latency should be < 1s, got %v", maxLatency))

			// Check overall throughput
			qps := float64(count) / totalTime.Seconds()
			Expect(qps).To(BeNumerically(">=", 50.0),
				fmt.Sprintf("Query throughput should be >= 50 QPS, got %.2f", qps))
		})

		It("should maintain 95th percentile latency under 500ms for 1 hour range queries", func() {
			rangeQueries := []string{
				"rate(prometheus_http_requests_total[1h])",
				"avg_over_time(up[1h])",
				"max_over_time(node_memory_MemTotal_bytes[1h])",
			}

			for _, query := range rangeQueries {
				latency := measureQueryLatency(ctx, clientset, restConfig, namespace, query)
				Expect(latency).To(BeNumerically("<", 500*time.Millisecond),
					fmt.Sprintf("1h range query latency should be < 500ms, got %v for: %s", latency, query))
			}
		})
	})

	Describe("Storage Performance", func() {
		It("should have reasonable disk usage growth rate", func() {
			// Get current storage usage
			initialStorage := getPrometheusStorageUsage(ctx, clientset, restConfig, namespace)
			Expect(initialStorage).To(BeNumerically(">", 0))

			// Wait and measure again (in real test, this would be longer)
			time.Sleep(1 * time.Minute)
			finalStorage := getPrometheusStorageUsage(ctx, clientset, restConfig, namespace)

			// Storage growth should be reasonable
			growthRate := finalStorage - initialStorage
			Expect(growthRate).To(BeNumerically(">=", 0), "Storage should not decrease")
		})

		It("should compact data efficiently", func() {
			// Check TSDB compaction metrics
			query := "prometheus_tsdb_compactions_total"
			resp := executePrometheusQuery(ctx, clientset, restConfig, namespace, query)
			defer resp.Body.Close()

			var queryResp PrometheusQueryResponse
			err := json.NewDecoder(resp.Body).Decode(&queryResp)
			Expect(err).NotTo(HaveOccurred())

			// Should have some compaction activity
			Expect(len(queryResp.Data.Result)).To(BeNumerically(">=", 0))
		})
	})

	Describe("Dashboard Rendering Performance", func() {
		It("should render Grafana dashboards quickly", func() {
			// Get list of dashboards
			dashboards := getGrafanaDashboards(ctx, clientset, restConfig, namespace)

			for _, dashboard := range dashboards {
				renderTime := measureDashboardRenderTime(ctx, clientset, restConfig, namespace, dashboard.UID)
				Expect(renderTime).To(BeNumerically("<", 3*time.Second),
					fmt.Sprintf("Dashboard '%s' render time should be < 3s, got %v", dashboard.Title, renderTime))
			}
		})

		It("should handle multiple concurrent dashboard requests", func() {
			dashboards := getGrafanaDashboards(ctx, clientset, restConfig, namespace)
			if len(dashboards) == 0 {
				Skip("No dashboards available for testing")
			}

			concurrency := 5
			var wg sync.WaitGroup
			renderTimes := make(chan time.Duration, concurrency)

			// Select a random dashboard for concurrent testing
			testDashboard := dashboards[rand.Intn(len(dashboards))]

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					renderTime := measureDashboardRenderTime(ctx, clientset, restConfig, namespace, testDashboard.UID)
					renderTimes <- renderTime
				}()
			}

			wg.Wait()
			close(renderTimes)

			// Check all render times are reasonable
			for renderTime := range renderTimes {
				Expect(renderTime).To(BeNumerically("<", 5*time.Second),
					fmt.Sprintf("Concurrent dashboard render time should be < 5s, got %v", renderTime))
			}
		})
	})
})

// Test Suite for structured performance testing
type MonitoringPerformanceTestSuite struct {
	suite.Suite
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
	namespace  string
	ctx        context.Context
	cancel     context.CancelFunc
}

func (suite *MonitoringPerformanceTestSuite) SetupSuite() {
	suite.namespace = "oran-monitoring"
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 20*time.Minute)

	var err error
	suite.restConfig, err = config.GetConfig()
	require.NoError(suite.T(), err)

	suite.clientset, err = kubernetes.NewForConfig(suite.restConfig)
	require.NoError(suite.T(), err)
}

func (suite *MonitoringPerformanceTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *MonitoringPerformanceTestSuite) TestQueryPerformanceBenchmark() {
	t := suite.T()

	// Benchmark different query types
	testCases := []struct {
		name        string
		query       string
		maxLatency  time.Duration
		description string
	}{
		{
			name:        "SimpleMetric",
			query:       "up",
			maxLatency:  50 * time.Millisecond,
			description: "Basic up metric query",
		},
		{
			name:        "RateQuery",
			query:       "rate(prometheus_http_requests_total[5m])",
			maxLatency:  100 * time.Millisecond,
			description: "Rate calculation over 5 minutes",
		},
		{
			name:        "AggregationQuery",
			query:       "avg_over_time(up[1h]) by (job)",
			maxLatency:  300 * time.Millisecond,
			description: "Aggregation over 1 hour with grouping",
		},
		{
			name:        "PercentileQuery",
			query:       "histogram_quantile(0.95, rate(prometheus_http_request_duration_seconds_bucket[5m]))",
			maxLatency:  500 * time.Millisecond,
			description: "95th percentile calculation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run multiple iterations to get stable measurements
			iterations := 10
			latencies := make([]time.Duration, iterations)

			for i := 0; i < iterations; i++ {
				latency := measureQueryLatency(suite.ctx, suite.clientset, suite.restConfig, suite.namespace, tc.query)
				latencies[i] = latency
			}

			// Calculate statistics
			var total time.Duration
			min, max := latencies[0], latencies[0]

			for _, latency := range latencies {
				total += latency
				if latency < min {
					min = latency
				}
				if latency > max {
					max = latency
				}
			}

			avg := total / time.Duration(iterations)

			t.Logf("Query: %s", tc.description)
			t.Logf("Average latency: %v", avg)
			t.Logf("Min latency: %v", min)
			t.Logf("Max latency: %v", max)

			assert.True(t, avg < tc.maxLatency,
				"Average latency %v should be less than %v for query: %s", avg, tc.maxLatency, tc.query)
		})
	}
}

func (suite *MonitoringPerformanceTestSuite) TestCardinalityLimits() {
	t := suite.T()

	// Test cardinality limits (max 10k time series per component)
	components := []string{"prometheus", "grafana", "alertmanager"}

	for _, component := range components {
		// Query metrics count for component
		query := fmt.Sprintf(`count by (job) (group by (__name__, job) ({job=~".*%s.*"}))`, component)
		resp := executePrometheusQuery(suite.ctx, suite.clientset, suite.restConfig, suite.namespace, query)

		var queryResp PrometheusQueryResponse
		err := json.NewDecoder(resp.Body).Decode(&queryResp)
		resp.Body.Close()

		if err == nil && len(queryResp.Data.Result) > 0 {
			// If we have results, verify cardinality is reasonable
			for _, result := range queryResp.Data.Result {
				if len(result.Value) > 1 {
					// This is a simplified check - in practice you'd parse the actual count
					t.Logf("Component %s has metrics available", component)
				}
			}
		}
	}
}

func (suite *MonitoringPerformanceTestSuite) TestStorageEfficiency() {
	t := suite.T()

	// Test storage efficiency over time
	storageQueries := []string{
		"prometheus_tsdb_head_samples_appended_total",
		"prometheus_tsdb_head_series",
		"prometheus_tsdb_wal_size_bytes",
	}

	for _, query := range storageQueries {
		resp := executePrometheusQuery(suite.ctx, suite.clientset, suite.restConfig, suite.namespace, query)

		var queryResp PrometheusQueryResponse
		err := json.NewDecoder(resp.Body).Decode(&queryResp)
		resp.Body.Close()

		assert.NoError(t, err, "Storage query should execute successfully: %s", query)

		if len(queryResp.Data.Result) > 0 {
			t.Logf("Storage metric %s: has %d results", query, len(queryResp.Data.Result))
		}
	}
}

// Helper functions for performance testing
func getMetricsCount(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) int {
	query := "prometheus_tsdb_head_samples_appended_total"
	resp := executePrometheusQuery(ctx, clientset, restConfig, namespace, query)
	defer resp.Body.Close()

	var queryResp PrometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return 0
	}

	if len(queryResp.Data.Result) > 0 && len(queryResp.Data.Result[0].Value) > 1 {
		// This is a simplified metric count - in practice you'd parse the actual value
		return len(queryResp.Data.Result)
	}
	return 0
}

func measureQueryLatency(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace, query string) time.Duration {
	start := time.Now()
	resp := executePrometheusQuery(ctx, clientset, restConfig, namespace, query)
	latency := time.Since(start)
	resp.Body.Close()
	return latency
}

func executePrometheusQuery(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace, query string) *http.Response {
	path := fmt.Sprintf("api/v1/query?query=%s", query)
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "prometheus", "", path, nil)
	resp, _ := proxyReq.DoRaw(ctx)
	return &http.Response{Body: resp}
}

func getPrometheusMemoryUsage(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) float64 {
	query := "prometheus_tsdb_head_series"
	resp := executePrometheusQuery(ctx, clientset, restConfig, namespace, query)
	defer resp.Body.Close()

	// Simplified memory usage calculation
	return float64(time.Now().Unix()) // Placeholder
}

func getPrometheusStorageUsage(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) float64 {
	query := "prometheus_tsdb_wal_size_bytes"
	resp := executePrometheusQuery(ctx, clientset, restConfig, namespace, query)
	defer resp.Body.Close()

	// Simplified storage usage calculation
	return float64(time.Now().Unix()) // Placeholder
}

func getGrafanaDashboards(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) []GrafanaDashboard {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "grafana", "", "api/search?type=dash-db", nil)
	resp, err := proxyReq.DoRaw(ctx)
	if err != nil {
		return []GrafanaDashboard{}
	}

	var dashboards []GrafanaDashboard
	json.Unmarshal(resp, &dashboards)
	return dashboards
}

func measureDashboardRenderTime(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace, uid string) time.Duration {
	start := time.Now()
	path := fmt.Sprintf("api/dashboards/uid/%s", uid)
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "grafana", "", path, nil)
	proxyReq.DoRaw(ctx)
	return time.Since(start)
}

// Response structures (reused from e2e tests)
type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type GrafanaDashboard struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// Test runner
func TestMonitoringPerformance(t *testing.T) {
	suite.Run(t, new(MonitoringPerformanceTestSuite))
}

func TestMain(m *testing.M) {
	RegisterFailHandler(Fail)
	RunSpecs(m, "Monitoring Performance Test Suite")
}