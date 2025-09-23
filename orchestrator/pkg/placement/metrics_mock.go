package placement

import (
	"math/rand"
	"sync"
	"time"
)

// MockMetricsProvider provides simulated metrics for testing
type MockMetricsProvider struct {
	mu            sync.RWMutex
	metrics       map[string]*SiteMetrics
	subscriptions map[string][]func(*SiteMetrics)
	scenarios     map[string]MetricsScenario
	randomSeed    int64
}

// MetricsScenario defines a specific load pattern for a site
type MetricsScenario struct {
	BaseCPU       float64
	BaseMemory    float64
	BaseBandwidth float64
	BaseLatency   float64
	Variability   float64 // 0-1, how much metrics vary
	TrendUp       bool    // Whether utilization is trending up
}

// NewMockMetricsProvider creates a new mock metrics provider
func NewMockMetricsProvider() *MockMetricsProvider {
	return &MockMetricsProvider{
		metrics:       make(map[string]*SiteMetrics),
		subscriptions: make(map[string][]func(*SiteMetrics)),
		scenarios:     make(map[string]MetricsScenario),
		randomSeed:    time.Now().UnixNano(),
	}
}

// NewMockMetricsProviderWithScenarios creates provider with predefined scenarios
func NewMockMetricsProviderWithScenarios(scenarios map[string]MetricsScenario) *MockMetricsProvider {
	p := NewMockMetricsProvider()
	p.scenarios = scenarios
	return p
}

// SetScenario sets a specific metrics scenario for a site
func (m *MockMetricsProvider) SetScenario(siteID string, scenario MetricsScenario) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scenarios[siteID] = scenario
}

// SetMetrics directly sets metrics for a site (for testing)
func (m *MockMetricsProvider) SetMetrics(siteID string, metrics *SiteMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics[siteID] = metrics

	// Notify subscribers
	if callbacks, ok := m.subscriptions[siteID]; ok {
		for _, callback := range callbacks {
			go callback(metrics)
		}
	}
}

// GetMetrics retrieves current metrics for a site
func (m *MockMetricsProvider) GetMetrics(siteID string) (*SiteMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return existing metrics if available
	if metrics, ok := m.metrics[siteID]; ok {
		return metrics, nil
	}

	// Generate new metrics based on scenario
	metrics := m.generateMetrics(siteID)
	m.metrics[siteID] = metrics
	return metrics, nil
}

// GetAllMetrics retrieves metrics for all sites
func (m *MockMetricsProvider) GetAllMetrics() (map[string]*SiteMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate metrics for any sites with scenarios but no metrics
	for siteID := range m.scenarios {
		if _, hasMetrics := m.metrics[siteID]; !hasMetrics {
			m.metrics[siteID] = m.generateMetrics(siteID)
		}
	}

	// Return copy of metrics map
	result := make(map[string]*SiteMetrics)
	for k, v := range m.metrics {
		result[k] = v
	}
	return result, nil
}

// Subscribe to metric updates
func (m *MockMetricsProvider) Subscribe(siteID string, callback func(*SiteMetrics)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[siteID] = append(m.subscriptions[siteID], callback)
}

// generateMetrics creates simulated metrics for a site
func (m *MockMetricsProvider) generateMetrics(siteID string) *SiteMetrics {
	rand.Seed(m.randomSeed)

	// Get scenario or use defaults
	scenario, hasScenario := m.scenarios[siteID]
	if !hasScenario {
		// Default scenario based on site type inference
		scenario = m.inferScenario(siteID)
	}

	// Generate metrics with variability
	variability := scenario.Variability
	if variability == 0 {
		variability = 0.1 // Default 10% variability
	}

	cpu := scenario.BaseCPU + (rand.Float64()-0.5)*2*variability*scenario.BaseCPU
	memory := scenario.BaseMemory + (rand.Float64()-0.5)*2*variability*scenario.BaseMemory
	bandwidth := scenario.BaseBandwidth + (rand.Float64()-0.5)*2*variability*scenario.BaseBandwidth
	latency := scenario.BaseLatency + (rand.Float64()-0.5)*2*variability*scenario.BaseLatency

	// Apply trends
	if scenario.TrendUp {
		timeFactor := float64(time.Now().Unix()%300) / 300.0 // 5-minute cycle
		cpu += timeFactor * 10
		memory += timeFactor * 10
	}

	// Ensure values are within valid ranges
	cpu = min(max(cpu, 0), 100)
	memory = min(max(memory, 0), 100)
	bandwidth = max(bandwidth, 0)
	latency = max(latency, 0)

	return &SiteMetrics{
		Timestamp:              time.Now(),
		CPUUtilization:        cpu,
		MemoryUtilization:     memory,
		AvailableBandwidthMbps: bandwidth,
		CurrentLatencyMs:       latency,
		ActiveNFs:             int(cpu / 10), // Rough approximation
	}
}

// inferScenario creates a default scenario based on site naming patterns
func (m *MockMetricsProvider) inferScenario(siteID string) MetricsScenario {
	// Edge sites: low utilization, low latency, moderate bandwidth
	if containsAny(siteID, []string{"edge", "ran", "access"}) {
		return MetricsScenario{
			BaseCPU:       30,
			BaseMemory:    35,
			BaseBandwidth: 100,
			BaseLatency:   5,
			Variability:   0.15,
			TrendUp:       false,
		}
	}

	// Regional sites: moderate utilization, moderate latency, high bandwidth
	if containsAny(siteID, []string{"regional", "metro", "aggregation"}) {
		return MetricsScenario{
			BaseCPU:       50,
			BaseMemory:    55,
			BaseBandwidth: 1000,
			BaseLatency:   15,
			Variability:   0.1,
			TrendUp:       false,
		}
	}

	// Central sites: higher utilization, higher latency, very high bandwidth
	if containsAny(siteID, []string{"central", "core", "datacenter", "dc"}) {
		return MetricsScenario{
			BaseCPU:       65,
			BaseMemory:    70,
			BaseBandwidth: 10000,
			BaseLatency:   25,
			Variability:   0.05,
			TrendUp:       true,
		}
	}

	// Default scenario
	return MetricsScenario{
		BaseCPU:       40,
		BaseMemory:    45,
		BaseBandwidth: 500,
		BaseLatency:   10,
		Variability:   0.1,
		TrendUp:       false,
	}
}

// SimulateTimeSeriesMetrics generates a time series of metrics for testing
func (m *MockMetricsProvider) SimulateTimeSeriesMetrics(siteID string, duration time.Duration, interval time.Duration) []*SiteMetrics {
	var series []*SiteMetrics

	startTime := time.Now()
	currentTime := startTime

	for currentTime.Sub(startTime) < duration {
		// Generate metrics for this time point
		metrics := m.generateMetrics(siteID)
		metrics.Timestamp = currentTime
		series = append(series, metrics)

		// Update stored metrics
		m.SetMetrics(siteID, metrics)

		// Advance time
		currentTime = currentTime.Add(interval)

		// Simulate time-based changes
		if scenario, ok := m.scenarios[siteID]; ok && scenario.TrendUp {
			// Gradually increase utilization
			scenario.BaseCPU += 1
			scenario.BaseMemory += 1
			m.scenarios[siteID] = scenario
		}
	}

	return series
}

// ResetMetrics clears all stored metrics
func (m *MockMetricsProvider) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = make(map[string]*SiteMetrics)
}

// containsAny checks if string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains is a simple substring check (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(stringContains(toLowerCase(s), toLowerCase(substr)))
}

// Simple string utility functions to avoid external dependencies
func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func stringContains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// min returns the smaller of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}