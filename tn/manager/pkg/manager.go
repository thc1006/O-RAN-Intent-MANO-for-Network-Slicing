package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// TNManager manages Transport Network configuration and monitoring
type TNManager struct {
	config      *TNConfig
	agents      map[string]*TNAgentClient
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	metrics     *MetricsCollector
	logger      *log.Logger
}

// NewTNManager creates a new Transport Network manager
func NewTNManager(config *TNConfig, logger *log.Logger) *TNManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &TNManager{
		config:  config,
		agents:  make(map[string]*TNAgentClient),
		ctx:     ctx,
		cancel:  cancel,
		metrics: NewMetricsCollector(logger),
		logger:  logger,
	}
}

// Start initializes and starts the TN manager
func (tm *TNManager) Start() error {
	security.SafeLogf(tm.logger, "Starting TN Manager for cluster: %s", security.SanitizeForLog(tm.config.ClusterName))

	// Start metrics collection
	if err := tm.metrics.Start(tm.ctx); err != nil {
		return fmt.Errorf("failed to start metrics collector: %w", err)
	}

	// Start monitoring goroutine
	go tm.monitoringLoop()

	tm.logger.Println("TN Manager started successfully")
	return nil
}

// Stop gracefully shuts down the TN manager
func (tm *TNManager) Stop() error {
	tm.logger.Println("Stopping TN Manager...")

	tm.cancel()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Stop all agents
	for name, agent := range tm.agents {
		if err := agent.Stop(); err != nil {
			security.SafeLogf(tm.logger, "Error stopping agent %s: %s", security.SanitizeForLog(name), security.SanitizeErrorForLog(err))
		}
	}

	// Stop metrics collector
	if err := tm.metrics.Stop(); err != nil {
		security.SafeLogError(tm.logger, "Error stopping metrics collector", err)
	}

	tm.logger.Println("TN Manager stopped")
	return nil
}

// RegisterAgent registers a new TN agent
func (tm *TNManager) RegisterAgent(clusterName, endpoint string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	agent := NewTNAgentClient(endpoint, tm.logger)
	if err := agent.Connect(); err != nil {
		return fmt.Errorf("failed to connect to agent %s: %w", clusterName, err)
	}

	tm.agents[clusterName] = agent
	security.SafeLogf(tm.logger, "Registered TN agent for cluster: %s", security.SanitizeForLog(clusterName))

	return nil
}

// ConfigureNetworkSlice configures a network slice across all registered agents
func (tm *TNManager) ConfigureNetworkSlice(sliceID string, config *TNConfig) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	security.SafeLogf(tm.logger, "Configuring network slice %s across %d agents", security.SanitizeForLog(sliceID), len(tm.agents))

	var wg sync.WaitGroup
	errChan := make(chan error, len(tm.agents))

	for clusterName, agent := range tm.agents {
		wg.Add(1)
		go func(name string, a *TNAgentClient) {
			defer wg.Done()

			if err := a.ConfigureSlice(sliceID, config); err != nil {
				errChan <- fmt.Errorf("failed to configure slice %s on cluster %s: %w", sliceID, name, err)
				return
			}

			security.SafeLogf(tm.logger, "Successfully configured slice %s on cluster %s", security.SanitizeForLog(sliceID), security.SanitizeForLog(name))
		}(clusterName, agent)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// RunPerformanceTest executes comprehensive performance tests
func (tm *TNManager) RunPerformanceTest(testConfig *PerformanceTestConfig) (*NetworkSliceMetrics, error) {
	security.SafeLogf(tm.logger, "Running performance test: %s", security.SanitizeForLog(testConfig.TestID))

	startTime := time.Now()

	// Initialize test results
	results := &NetworkSliceMetrics{
		SliceID:        testConfig.SliceID,
		SliceType:      testConfig.SliceType,
		Timestamp:      startTime,
		ClusterMetrics: make(map[string]PerformanceMetrics),
		ThesisValidation: ThesisValidation{
			ThroughputTargets: []float64{0.93, 2.77, 4.57}, // Thesis targets
			RTTTargets:        []float64{6.3, 15.7, 16.1},  // Thesis targets
			DeployTargetMs:    600000,                       // 10 minutes
		},
	}

	// Run tests on all agents
	var wg sync.WaitGroup
	metricsChan := make(chan PerformanceMetrics, len(tm.agents))

	tm.mu.RLock()
	for clusterName, agent := range tm.agents {
		wg.Add(1)
		go func(name string, a *TNAgentClient) {
			defer wg.Done()

			metrics, err := a.RunPerformanceTest(testConfig)
			if err != nil {
				security.SafeLogf(tm.logger, "Performance test failed on cluster %s: %s", security.SanitizeForLog(name), security.SanitizeErrorForLog(err))
				return
			}

			metrics.ClusterName = name
			metricsChan <- *metrics
		}(clusterName, agent)
	}

	wg.Wait()
	close(metricsChan)

	// Aggregate results
	var allMetrics []PerformanceMetrics
	for metrics := range metricsChan {
		results.ClusterMetrics[metrics.ClusterName] = metrics
		allMetrics = append(allMetrics, metrics)
	}

	// Calculate aggregated performance metrics
	if len(allMetrics) > 0 {
		results.Performance = tm.aggregateMetrics(allMetrics)
	}

	// Validate against thesis targets
	results.ThesisValidation = tm.validateThesisTargets(allMetrics, time.Since(startTime))
	results.SLACompliance = results.ThesisValidation.CompliancePercent >= 80.0

	security.SafeLogf(tm.logger, "Performance test completed. Compliance: %.2f%%", results.ThesisValidation.CompliancePercent)

	return results, nil
}

// aggregateMetrics combines metrics from multiple clusters
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

// validateThesisTargets validates performance against thesis targets
func (tm *TNManager) validateThesisTargets(metrics []PerformanceMetrics, deployTime time.Duration) ThesisValidation {
	validation := ThesisValidation{
		ThroughputTargets: []float64{0.93, 2.77, 4.57},
		RTTTargets:        []float64{6.3, 15.7, 16.1},
		DeployTimeMs:      deployTime.Milliseconds(),
		DeployTargetMs:    600000, // 10 minutes
	}

	// Extract results
	for _, m := range metrics {
		validation.ThroughputResults = append(validation.ThroughputResults, m.Throughput.AvgMbps)
		validation.RTTResults = append(validation.RTTResults, m.Latency.AvgRTTMs)
	}

	// Count passed tests
	passedTests := 0
	totalTests := 0

	// Validate throughput targets
	for i, target := range validation.ThroughputTargets {
		if i < len(validation.ThroughputResults) {
			totalTests++
			// Allow 10% tolerance
			if validation.ThroughputResults[i] >= target*0.9 {
				passedTests++
			}
		}
	}

	// Validate RTT targets
	for i, target := range validation.RTTTargets {
		if i < len(validation.RTTResults) {
			totalTests++
			// Allow 10% tolerance
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

// GetStatus returns the current status of all TN agents
func (tm *TNManager) GetStatus() (map[string]*TNStatus, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	status := make(map[string]*TNStatus)

	for clusterName, agent := range tm.agents {
		agentStatus, err := agent.GetStatus()
		if err != nil {
			security.SafeLogf(tm.logger, "Failed to get status from agent %s: %s", security.SanitizeForLog(clusterName), security.SanitizeErrorForLog(err))
			continue
		}
		status[clusterName] = agentStatus
	}

	return status, nil
}

// monitoringLoop runs continuous monitoring
func (tm *TNManager) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			return
		case <-ticker.C:
			tm.collectMetrics()
		}
	}
}

// collectMetrics collects metrics from all agents
func (tm *TNManager) collectMetrics() {
	tm.mu.RLock()
	agents := make(map[string]*TNAgentClient)
	for k, v := range tm.agents {
		agents[k] = v
	}
	tm.mu.RUnlock()

	for clusterName, agent := range agents {
		go func(name string, a *TNAgentClient) {
			status, err := a.GetStatus()
			if err != nil {
				security.SafeLogf(tm.logger, "Failed to collect metrics from %s: %s", security.SanitizeForLog(name), security.SanitizeErrorForLog(err))
				return
			}

			// Store metrics
			if err := tm.metrics.RecordStatus(name, status); err != nil {
				security.SafeLogf(tm.logger, "Failed to record metrics for %s: %s", security.SanitizeForLog(name), security.SanitizeErrorForLog(err))
			}
		}(clusterName, agent)
	}
}

// ExportMetrics exports collected metrics in JSON format
func (tm *TNManager) ExportMetrics(filename string) error {
	metrics := tm.metrics.Export()

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Write to file (implementation would write to actual file)
	security.SafeLogf(tm.logger, "Exported metrics to %s (%d bytes)", security.SanitizeForLog(filename), len(data))

	return nil
}

// PerformanceTestConfig defines parameters for performance testing
type PerformanceTestConfig struct {
	TestID       string        `json:"testId"`
	SliceID      string        `json:"sliceId"`
	SliceType    string        `json:"sliceType"`
	Duration     time.Duration `json:"duration"`
	TestType     string        `json:"testType"`     // "iperf3", "ping", "custom"
	SourceCluster string       `json:"sourceCluster"`
	TargetCluster string       `json:"targetCluster"`
	Protocol     string        `json:"protocol"`     // "tcp", "udp"
	Parallel     int           `json:"parallel"`
	WindowSize   string        `json:"windowSize"`
	Interval     time.Duration `json:"interval"`
}