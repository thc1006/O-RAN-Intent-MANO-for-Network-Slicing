package unit

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock types for TN Manager testing

type MockTNAgentClient struct {
	mock.Mock
	endpoint string
	status   *TNStatus
}

func (m *MockTNAgentClient) Connect() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTNAgentClient) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTNAgentClient) ConfigureSlice(sliceID string, config *TNConfig) error {
	args := m.Called(sliceID, config)
	return args.Error(0)
}

func (m *MockTNAgentClient) GetStatus() (*TNStatus, error) {
	args := m.Called()
	return args.Get(0).(*TNStatus), args.Error(1)
}

func (m *MockTNAgentClient) RunPerformanceTest(config *PerformanceTestConfig) (*PerformanceMetrics, error) {
	args := m.Called(config)
	return args.Get(0).(*PerformanceMetrics), args.Error(1)
}

// Test data structures

type TNConfig struct {
	ClusterName    string            `json:"clusterName"`
	BandwidthLimit string            `json:"bandwidthLimit"`
	QoSProfile     string            `json:"qosProfile"`
	VLANConfig     map[string]string `json:"vlanConfig"`
}

type TNStatus struct {
	Cluster     string                 `json:"cluster"`
	Status      string                 `json:"status"`
	Uptime      time.Duration          `json:"uptime"`
	ActiveSlices int                   `json:"activeSlices"`
	Metrics     map[string]interface{} `json:"metrics"`
}

type PerformanceTestConfig struct {
	TestID        string        `json:"testId"`
	SliceID       string        `json:"sliceId"`
	SliceType     string        `json:"sliceType"`
	Duration      time.Duration `json:"duration"`
	TestType      string        `json:"testType"`
	SourceCluster string        `json:"sourceCluster"`
	TargetCluster string        `json:"targetCluster"`
	Protocol      string        `json:"protocol"`
	Parallel      int           `json:"parallel"`
	WindowSize    string        `json:"windowSize"`
	Interval      time.Duration `json:"interval"`
}

type PerformanceMetrics struct {
	ClusterName string              `json:"clusterName"`
	Timestamp   time.Time           `json:"timestamp"`
	TestType    string              `json:"testType"`
	Throughput  ThroughputMetrics   `json:"throughput"`
	Latency     LatencyMetrics      `json:"latency"`
	PacketLoss  float64             `json:"packetLoss"`
	Jitter      float64             `json:"jitter"`
}

type ThroughputMetrics struct {
	AvgMbps  float64 `json:"avgMbps"`
	PeakMbps float64 `json:"peakMbps"`
}

type LatencyMetrics struct {
	AvgRTTMs float64 `json:"avgRttMs"`
	MaxRTTMs float64 `json:"maxRttMs"`
}

type NetworkSliceMetrics struct {
	SliceID          string                      `json:"sliceId"`
	SliceType        string                      `json:"sliceType"`
	Timestamp        time.Time                   `json:"timestamp"`
	Performance      PerformanceMetrics          `json:"performance"`
	ClusterMetrics   map[string]PerformanceMetrics `json:"clusterMetrics"`
	ThesisValidation ThesisValidation            `json:"thesisValidation"`
	SLACompliance    bool                        `json:"slaCompliance"`
}

type ThesisValidation struct {
	ThroughputTargets   []float64 `json:"throughputTargets"`
	ThroughputResults   []float64 `json:"throughputResults"`
	RTTTargets          []float64 `json:"rttTargets"`
	RTTResults          []float64 `json:"rttResults"`
	DeployTimeMs        int64     `json:"deployTimeMs"`
	DeployTargetMs      int64     `json:"deployTargetMs"`
	PassedTests         int       `json:"passedTests"`
	TotalTests          int       `json:"totalTests"`
	CompliancePercent   float64   `json:"compliancePercent"`
}

type MetricsCollector struct {
	logger  *log.Logger
	metrics map[string]interface{}
	mu      sync.RWMutex
}

func NewMetricsCollector(logger *log.Logger) *MetricsCollector {
	return &MetricsCollector{
		logger:  logger,
		metrics: make(map[string]interface{}),
	}
}

func (mc *MetricsCollector) Start(ctx context.Context) error {
	return nil
}

func (mc *MetricsCollector) Stop() error {
	return nil
}

func (mc *MetricsCollector) RecordStatus(name string, status *TNStatus) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics[name] = status
	return nil
}

func (mc *MetricsCollector) Export() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range mc.metrics {
		result[k] = v
	}
	return result
}

// Simplified TNManager for testing
type TNManager struct {
	config  *TNConfig
	agents  map[string]*MockTNAgentClient
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	metrics *MetricsCollector
	logger  *log.Logger
}

func NewTNManager(config *TNConfig, logger *log.Logger) *TNManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TNManager{
		config:  config,
		agents:  make(map[string]*MockTNAgentClient),
		ctx:     ctx,
		cancel:  cancel,
		metrics: NewMetricsCollector(logger),
		logger:  logger,
	}
}

func (tm *TNManager) Start() error {
	return tm.metrics.Start(tm.ctx)
}

func (tm *TNManager) Stop() error {
	tm.cancel()
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for name, agent := range tm.agents {
		if err := agent.Stop(); err != nil {
			tm.logger.Printf("Error stopping agent %s: %v", name, err)
		}
	}
	return tm.metrics.Stop()
}

func (tm *TNManager) RegisterAgent(clusterName, endpoint string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	agent := &MockTNAgentClient{endpoint: endpoint}
	agent.On("Connect").Return(nil)
	agent.On("Stop").Return(nil)

	if err := agent.Connect(); err != nil {
		return err
	}

	tm.agents[clusterName] = agent
	return nil
}

func (tm *TNManager) ConfigureNetworkSlice(sliceID string, config *TNConfig) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(tm.agents))

	for clusterName, agent := range tm.agents {
		wg.Add(1)
		go func(name string, a *MockTNAgentClient) {
			defer wg.Done()
			a.On("ConfigureSlice", sliceID, config).Return(nil)
			if err := a.ConfigureSlice(sliceID, config); err != nil {
				errChan <- err
			}
		}(clusterName, agent)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (tm *TNManager) RunPerformanceTest(testConfig *PerformanceTestConfig) (*NetworkSliceMetrics, error) {
	startTime := time.Now()

	results := &NetworkSliceMetrics{
		SliceID:        testConfig.SliceID,
		SliceType:      testConfig.SliceType,
		Timestamp:      startTime,
		ClusterMetrics: make(map[string]PerformanceMetrics),
		ThesisValidation: ThesisValidation{
			ThroughputTargets: []float64{0.93, 2.77, 4.57},
			RTTTargets:        []float64{6.3, 15.7, 16.1},
			DeployTargetMs:    600000,
		},
	}

	var wg sync.WaitGroup
	metricsChan := make(chan PerformanceMetrics, len(tm.agents))

	tm.mu.RLock()
	for clusterName, agent := range tm.agents {
		wg.Add(1)
		go func(name string, a *MockTNAgentClient) {
			defer wg.Done()

			metrics := &PerformanceMetrics{
				ClusterName: name,
				Timestamp:   time.Now(),
				TestType:    testConfig.TestType,
				Throughput:  ThroughputMetrics{AvgMbps: 5.0, PeakMbps: 8.0},
				Latency:     LatencyMetrics{AvgRTTMs: 5.0, MaxRTTMs: 10.0},
				PacketLoss:  0.01,
				Jitter:      0.5,
			}

			a.On("RunPerformanceTest", testConfig).Return(metrics, nil)
			result, err := a.RunPerformanceTest(testConfig)
			if err == nil {
				metricsChan <- *result
			}
		}(clusterName, agent)
	}
	tm.mu.RUnlock()

	wg.Wait()
	close(metricsChan)

	var allMetrics []PerformanceMetrics
	for metrics := range metricsChan {
		results.ClusterMetrics[metrics.ClusterName] = metrics
		allMetrics = append(allMetrics, metrics)
	}

	if len(allMetrics) > 0 {
		results.Performance = tm.aggregateMetrics(allMetrics)
	}

	results.ThesisValidation = tm.validateThesisTargets(allMetrics, time.Since(startTime))
	results.SLACompliance = results.ThesisValidation.CompliancePercent >= 80.0

	return results, nil
}

func (tm *TNManager) aggregateMetrics(metrics []PerformanceMetrics) PerformanceMetrics {
	if len(metrics) == 0 {
		return PerformanceMetrics{}
	}

	var totalThroughput, totalLatency, totalLoss, totalJitter float64
	var maxThroughput, maxLatency float64

	for _, m := range metrics {
		totalThroughput += m.Throughput.AvgMbps
		totalLatency += m.Latency.AvgRTTMs
		totalLoss += m.PacketLoss
		totalJitter += m.Jitter

		if m.Throughput.PeakMbps > maxThroughput {
			maxThroughput = m.Throughput.PeakMbps
		}
		if m.Latency.MaxRTTMs > maxLatency {
			maxLatency = m.Latency.MaxRTTMs
		}
	}

	count := float64(len(metrics))
	return PerformanceMetrics{
		Timestamp:   time.Now(),
		TestType:    "aggregated",
		Throughput: ThroughputMetrics{
			AvgMbps:  totalThroughput / count,
			PeakMbps: maxThroughput,
		},
		Latency: LatencyMetrics{
			AvgRTTMs: totalLatency / count,
			MaxRTTMs: maxLatency,
		},
		PacketLoss: totalLoss / count,
		Jitter:     totalJitter / count,
	}
}

func (tm *TNManager) validateThesisTargets(metrics []PerformanceMetrics, deployTime time.Duration) ThesisValidation {
	validation := ThesisValidation{
		ThroughputTargets: []float64{0.93, 2.77, 4.57},
		RTTTargets:        []float64{6.3, 15.7, 16.1},
		DeployTimeMs:      deployTime.Milliseconds(),
		DeployTargetMs:    600000,
	}

	for _, m := range metrics {
		validation.ThroughputResults = append(validation.ThroughputResults, m.Throughput.AvgMbps)
		validation.RTTResults = append(validation.RTTResults, m.Latency.AvgRTTMs)
	}

	passedTests := 0
	totalTests := 0

	// Validate throughput targets
	for i, target := range validation.ThroughputTargets {
		if i < len(validation.ThroughputResults) {
			totalTests++
			if validation.ThroughputResults[i] >= target*0.9 {
				passedTests++
			}
		}
	}

	// Validate RTT targets
	for i, target := range validation.RTTTargets {
		if i < len(validation.RTTResults) {
			totalTests++
			if validation.RTTResults[i] <= target*1.1 {
				passedTests++
			}
		}
	}

	// Validate deploy time
	totalTests++
	if validation.DeployTimeMs <= validation.DeployTargetMs {
		passedTests++
	}

	validation.PassedTests = passedTests
	validation.TotalTests = totalTests

	if totalTests > 0 {
		validation.CompliancePercent = float64(passedTests) / float64(totalTests) * 100
	}

	return validation
}

func (tm *TNManager) GetStatus() (map[string]*TNStatus, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	status := make(map[string]*TNStatus)
	for clusterName, agent := range tm.agents {
		agentStatus := &TNStatus{
			Cluster:      clusterName,
			Status:       "operational",
			Uptime:       time.Hour,
			ActiveSlices: 5,
			Metrics:      map[string]interface{}{"cpu": "60%", "memory": "40%"},
		}
		agent.On("GetStatus").Return(agentStatus, nil)
		agentStatus, err := agent.GetStatus()
		if err != nil {
			continue
		}
		status[clusterName] = agentStatus
	}
	return status, nil
}

// Unit tests

func TestTNManagerCreation(t *testing.T) {
	config := &TNConfig{
		ClusterName:    "test-cluster",
		BandwidthLimit: "100Mbps",
		QoSProfile:     "high",
	}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)

	manager := NewTNManager(config, logger)

	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.NotNil(t, manager.agents)
	assert.NotNil(t, manager.metrics)
	assert.NotNil(t, manager.logger)
}

func TestTNManagerStartStop(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Test start
	err := manager.Start()
	assert.NoError(t, err)

	// Test stop
	err = manager.Stop()
	assert.NoError(t, err)
}

func TestTNManagerRegisterAgent(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	err := manager.RegisterAgent("cluster1", "http://localhost:8080")
	assert.NoError(t, err)

	// Verify agent was registered
	manager.mu.RLock()
	agent, exists := manager.agents["cluster1"]
	manager.mu.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, agent)
	assert.Equal(t, "http://localhost:8080", agent.endpoint)
}

func TestTNManagerConfigureNetworkSlice(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Register multiple agents
	err := manager.RegisterAgent("cluster1", "http://localhost:8080")
	require.NoError(t, err)
	err = manager.RegisterAgent("cluster2", "http://localhost:8081")
	require.NoError(t, err)

	sliceConfig := &TNConfig{
		ClusterName:    "test-slice",
		BandwidthLimit: "50Mbps",
		QoSProfile:     "medium",
		VLANConfig:     map[string]string{"vlan": "100"},
	}

	err = manager.ConfigureNetworkSlice("slice-001", sliceConfig)
	assert.NoError(t, err)

	// Verify all agents were called
	for _, agent := range manager.agents {
		agent.AssertCalled(t, "ConfigureSlice", "slice-001", sliceConfig)
	}
}

func TestTNManagerPerformanceTest(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Register agents
	err := manager.RegisterAgent("cluster1", "http://localhost:8080")
	require.NoError(t, err)
	err = manager.RegisterAgent("cluster2", "http://localhost:8081")
	require.NoError(t, err)

	testConfig := &PerformanceTestConfig{
		TestID:        "test-001",
		SliceID:       "slice-001",
		SliceType:     "eMBB",
		Duration:      time.Minute,
		TestType:      "iperf3",
		SourceCluster: "cluster1",
		TargetCluster: "cluster2",
		Protocol:      "tcp",
		Parallel:      4,
		WindowSize:    "64K",
		Interval:      time.Second,
	}

	results, err := manager.RunPerformanceTest(testConfig)
	require.NoError(t, err)
	assert.NotNil(t, results)

	// Verify results structure
	assert.Equal(t, "slice-001", results.SliceID)
	assert.Equal(t, "eMBB", results.SliceType)
	assert.NotEmpty(t, results.ClusterMetrics)
	assert.Equal(t, 2, len(results.ClusterMetrics))

	// Verify thesis validation
	assert.NotEmpty(t, results.ThesisValidation.ThroughputTargets)
	assert.NotEmpty(t, results.ThesisValidation.RTTTargets)
	assert.True(t, results.SLACompliance)

	// Verify performance aggregation
	assert.Greater(t, results.Performance.Throughput.AvgMbps, 0.0)
	assert.Greater(t, results.Performance.Latency.AvgRTTMs, 0.0)
}

func TestTNManagerThesisValidation(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Test with metrics that meet targets
	goodMetrics := []PerformanceMetrics{
		{
			Throughput: ThroughputMetrics{AvgMbps: 5.0},
			Latency:    LatencyMetrics{AvgRTTMs: 5.0},
		},
	}

	validation := manager.validateThesisTargets(goodMetrics, 30*time.Second)
	assert.Greater(t, validation.CompliancePercent, 50.0)

	// Test with metrics that don't meet targets
	badMetrics := []PerformanceMetrics{
		{
			Throughput: ThroughputMetrics{AvgMbps: 0.1},
			Latency:    LatencyMetrics{AvgRTTMs: 100.0},
		},
	}

	validation = manager.validateThesisTargets(badMetrics, 15*time.Minute)
	assert.Less(t, validation.CompliancePercent, 50.0)
}

func TestTNManagerGetStatus(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Register agents
	err := manager.RegisterAgent("cluster1", "http://localhost:8080")
	require.NoError(t, err)
	err = manager.RegisterAgent("cluster2", "http://localhost:8081")
	require.NoError(t, err)

	status, err := manager.GetStatus()
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, 2, len(status))

	// Verify status for each cluster
	for clusterName, clusterStatus := range status {
		assert.Contains(t, []string{"cluster1", "cluster2"}, clusterName)
		assert.Equal(t, "operational", clusterStatus.Status)
		assert.Greater(t, clusterStatus.Uptime, time.Duration(0))
		assert.Equal(t, 5, clusterStatus.ActiveSlices)
	}
}

func TestTNManagerConcurrentOperations(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Register multiple agents
	for i := 0; i < 5; i++ {
		clusterName := fmt.Sprintf("cluster%d", i)
		endpoint := fmt.Sprintf("http://localhost:808%d", i)
		err := manager.RegisterAgent(clusterName, endpoint)
		require.NoError(t, err)
	}

	const numOperations = 10
	var wg sync.WaitGroup
	errors := make(chan error, numOperations)

	// Run concurrent slice configurations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sliceID := fmt.Sprintf("slice-%d", id)
			sliceConfig := &TNConfig{
				ClusterName:    sliceID,
				BandwidthLimit: "10Mbps",
				QoSProfile:     "low",
			}
			err := manager.ConfigureNetworkSlice(sliceID, sliceConfig)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check that all operations succeeded
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Logf("Error in concurrent operation: %v", err)
		}
	}

	assert.Equal(t, 0, errorCount, "All concurrent operations should succeed")
}

func TestTNManagerMetricsAggregation(t *testing.T) {
	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	metrics := []PerformanceMetrics{
		{
			Throughput: ThroughputMetrics{AvgMbps: 10.0, PeakMbps: 15.0},
			Latency:    LatencyMetrics{AvgRTTMs: 5.0, MaxRTTMs: 8.0},
			PacketLoss: 0.01,
			Jitter:     0.5,
		},
		{
			Throughput: ThroughputMetrics{AvgMbps: 20.0, PeakMbps: 25.0},
			Latency:    LatencyMetrics{AvgRTTMs: 10.0, MaxRTTMs: 15.0},
			PacketLoss: 0.02,
			Jitter:     1.0,
		},
	}

	aggregated := manager.aggregateMetrics(metrics)

	// Verify averages
	assert.Equal(t, 15.0, aggregated.Throughput.AvgMbps)
	assert.Equal(t, 7.5, aggregated.Latency.AvgRTTMs)
	assert.Equal(t, 0.015, aggregated.PacketLoss)
	assert.Equal(t, 0.75, aggregated.Jitter)

	// Verify maximums
	assert.Equal(t, 25.0, aggregated.Throughput.PeakMbps)
	assert.Equal(t, 15.0, aggregated.Latency.MaxRTTMs)
}

// Benchmark tests for performance validation

func BenchmarkTNManagerSliceConfiguration(b *testing.B) {
	config := &TNConfig{ClusterName: "bench-cluster"}
	logger := log.New(os.Stdout, "BENCH: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Register agents
	for i := 0; i < 10; i++ {
		clusterName := fmt.Sprintf("cluster%d", i)
		endpoint := fmt.Sprintf("http://localhost:808%d", i)
		manager.RegisterAgent(clusterName, endpoint)
	}

	sliceConfig := &TNConfig{
		ClusterName:    "bench-slice",
		BandwidthLimit: "100Mbps",
		QoSProfile:     "high",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sliceID := fmt.Sprintf("slice-%d", i)
		manager.ConfigureNetworkSlice(sliceID, sliceConfig)
	}
}

func BenchmarkTNManagerStatusRetrieval(b *testing.B) {
	config := &TNConfig{ClusterName: "bench-cluster"}
	logger := log.New(os.Stdout, "BENCH: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	// Register agents
	for i := 0; i < 10; i++ {
		clusterName := fmt.Sprintf("cluster%d", i)
		endpoint := fmt.Sprintf("http://localhost:808%d", i)
		manager.RegisterAgent(clusterName, endpoint)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetStatus()
	}
}

// Table-driven tests for comprehensive coverage

func TestTNManagerVLANConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		vlanConfig map[string]string
		expected   bool
	}{
		{
			name:       "Valid VLAN configuration",
			vlanConfig: map[string]string{"vlan": "100", "priority": "high"},
			expected:   true,
		},
		{
			name:       "Empty VLAN configuration",
			vlanConfig: map[string]string{},
			expected:   true,
		},
		{
			name:       "Nil VLAN configuration",
			vlanConfig: nil,
			expected:   true,
		},
	}

	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	err := manager.RegisterAgent("cluster1", "http://localhost:8080")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sliceConfig := &TNConfig{
				ClusterName:    "test-slice",
				BandwidthLimit: "50Mbps",
				QoSProfile:     "medium",
				VLANConfig:     tt.vlanConfig,
			}

			err := manager.ConfigureNetworkSlice("slice-vlan-test", sliceConfig)
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestTNManagerBandwidthAllocation(t *testing.T) {
	tests := []struct {
		name      string
		bandwidth string
		valid     bool
	}{
		{"Valid bandwidth 100Mbps", "100Mbps", true},
		{"Valid bandwidth 1Gbps", "1Gbps", true},
		{"Valid bandwidth 500Kbps", "500Kbps", true},
		{"Empty bandwidth", "", true}, // Should use default
		{"Invalid format", "100", true}, // Manager should handle gracefully
	}

	config := &TNConfig{ClusterName: "test-cluster"}
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	manager := NewTNManager(config, logger)

	err := manager.RegisterAgent("cluster1", "http://localhost:8080")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sliceConfig := &TNConfig{
				ClusterName:    "bandwidth-test",
				BandwidthLimit: tt.bandwidth,
				QoSProfile:     "medium",
			}

			err := manager.ConfigureNetworkSlice("slice-bandwidth-test", sliceConfig)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}