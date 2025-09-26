package mocks

import (
	"context"
	"time"
)

// MetricsCollector interface for collecting performance metrics
type MetricsCollector interface {
	CollectLatencyMetrics(ctx context.Context, target string) (*LatencyMetrics, error)
	CollectThroughputMetrics(ctx context.Context, target string) (*ThroughputMetrics, error)
	CollectResourceMetrics(ctx context.Context, target string) (*ResourceMetrics, error)
	CollectNetworkMetrics(ctx context.Context, target string) (*NetworkMetrics, error)
	GetHistoricalMetrics(ctx context.Context, target string, duration time.Duration) (*HistoricalMetrics, error)
}

// MockMetricsCollector provides a mock implementation for testing
type MockMetricsCollector struct {
	CollectLatencyFunc    func(ctx context.Context, target string) (*LatencyMetrics, error)
	CollectThroughputFunc func(ctx context.Context, target string) (*ThroughputMetrics, error)
	CollectResourceFunc   func(ctx context.Context, target string) (*ResourceMetrics, error)
	CollectNetworkFunc    func(ctx context.Context, target string) (*NetworkMetrics, error)
	GetHistoricalFunc     func(ctx context.Context, target string, duration time.Duration) (*HistoricalMetrics, error)

	// Call tracking
	LatencyCalls    []MetricsCall
	ThroughputCalls []MetricsCall
	ResourceCalls   []MetricsCall
	NetworkCalls    []MetricsCall
	HistoricalCalls []HistoricalMetricsCall
}

type MetricsCall struct {
	Target string
}

type HistoricalMetricsCall struct {
	Target   string
	Duration time.Duration
}

// Metrics data structures
type LatencyMetrics struct {
	Average    time.Duration            `json:"average"`
	P50        time.Duration            `json:"p50"`
	P95        time.Duration            `json:"p95"`
	P99        time.Duration            `json:"p99"`
	Min        time.Duration            `json:"min"`
	Max        time.Duration            `json:"max"`
	Timestamp  time.Time                `json:"timestamp"`
	Source     string                   `json:"source"`
	Breakdown  map[string]time.Duration `json:"breakdown,omitempty"`
}

type ThroughputMetrics struct {
	Downlink    float64            `json:"downlink"`
	Uplink      float64            `json:"uplink"`
	Unit        string             `json:"unit"`
	Timestamp   time.Time          `json:"timestamp"`
	Source      string             `json:"source"`
	Utilization map[string]float64 `json:"utilization,omitempty"`
}

type ResourceMetrics struct {
	CPU      CPUMetrics     `json:"cpu"`
	Memory   MemoryMetrics  `json:"memory"`
	Network  NetworkMetrics `json:"network"`
	Storage  StorageMetrics `json:"storage,omitempty"`
	GPU      GPUMetrics     `json:"gpu,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Source    string        `json:"source"`
}

type CPUMetrics struct {
	Usage       float64 `json:"usage"`       // Percentage
	Cores       int     `json:"cores"`
	Frequency   float64 `json:"frequency"`   // GHz
	Temperature float64 `json:"temperature"` // Celsius
}

type MemoryMetrics struct {
	Used      int64   `json:"used"`      // Bytes
	Available int64   `json:"available"` // Bytes
	Total     int64   `json:"total"`     // Bytes
	Usage     float64 `json:"usage"`     // Percentage
}

type NetworkMetrics struct {
	BytesIn     int64   `json:"bytesIn"`
	BytesOut    int64   `json:"bytesOut"`
	PacketsIn   int64   `json:"packetsIn"`
	PacketsOut  int64   `json:"packetsOut"`
	ErrorsIn    int64   `json:"errorsIn"`
	ErrorsOut   int64   `json:"errorsOut"`
	Bandwidth   float64 `json:"bandwidth"` // Mbps
	Utilization float64 `json:"utilization"` // Percentage
}

type StorageMetrics struct {
	Used      int64   `json:"used"`      // Bytes
	Available int64   `json:"available"` // Bytes
	Total     int64   `json:"total"`     // Bytes
	Usage     float64 `json:"usage"`     // Percentage
	IOPS      int64   `json:"iops"`
}

type GPUMetrics struct {
	Usage       float64 `json:"usage"`       // Percentage
	Memory      int64   `json:"memory"`      // Bytes
	Temperature float64 `json:"temperature"` // Celsius
	Power       float64 `json:"power"`       // Watts
}

type HistoricalMetrics struct {
	Latency    []LatencyDataPoint    `json:"latency"`
	Throughput []ThroughputDataPoint `json:"throughput"`
	Resource   []ResourceDataPoint   `json:"resource"`
	TimeRange  TimeRange             `json:"timeRange"`
	Interval   time.Duration         `json:"interval"`
}

type LatencyDataPoint struct {
	Timestamp time.Time     `json:"timestamp"`
	Value     time.Duration `json:"value"`
}

type ThroughputDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Downlink  float64   `json:"downlink"`
	Uplink    float64   `json:"uplink"`
}

type ResourceDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	CPU       float64   `json:"cpu"`
	Memory    float64   `json:"memory"`
	Network   float64   `json:"network"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Mock implementations
func (m *MockMetricsCollector) CollectLatencyMetrics(ctx context.Context, target string) (*LatencyMetrics, error) {
	m.LatencyCalls = append(m.LatencyCalls, MetricsCall{Target: target})
	if m.CollectLatencyFunc != nil {
		return m.CollectLatencyFunc(ctx, target)
	}
	return CreateDefaultLatencyMetrics(target), nil
}

func (m *MockMetricsCollector) CollectThroughputMetrics(ctx context.Context, target string) (*ThroughputMetrics, error) {
	m.ThroughputCalls = append(m.ThroughputCalls, MetricsCall{Target: target})
	if m.CollectThroughputFunc != nil {
		return m.CollectThroughputFunc(ctx, target)
	}
	return CreateDefaultThroughputMetrics(target), nil
}

func (m *MockMetricsCollector) CollectResourceMetrics(ctx context.Context, target string) (*ResourceMetrics, error) {
	m.ResourceCalls = append(m.ResourceCalls, MetricsCall{Target: target})
	if m.CollectResourceFunc != nil {
		return m.CollectResourceFunc(ctx, target)
	}
	return CreateDefaultResourceMetrics(target), nil
}

func (m *MockMetricsCollector) CollectNetworkMetrics(ctx context.Context, target string) (*NetworkMetrics, error) {
	m.NetworkCalls = append(m.NetworkCalls, MetricsCall{Target: target})
	if m.CollectNetworkFunc != nil {
		return m.CollectNetworkFunc(ctx, target)
	}
	return CreateDefaultNetworkMetrics(), nil
}

func (m *MockMetricsCollector) GetHistoricalMetrics(ctx context.Context, target string, duration time.Duration) (*HistoricalMetrics, error) {
	m.HistoricalCalls = append(m.HistoricalCalls, HistoricalMetricsCall{Target: target, Duration: duration})
	if m.GetHistoricalFunc != nil {
		return m.GetHistoricalFunc(ctx, target, duration)
	}
	return CreateDefaultHistoricalMetrics(target, duration), nil
}

// Helper functions to create default metrics for testing
func CreateDefaultLatencyMetrics(target string) *LatencyMetrics {
	return &LatencyMetrics{
		Average:   10 * time.Millisecond,
		P50:       8 * time.Millisecond,
		P95:       20 * time.Millisecond,
		P99:       50 * time.Millisecond,
		Min:       1 * time.Millisecond,
		Max:       100 * time.Millisecond,
		Timestamp: time.Now(),
		Source:    target,
		Breakdown: map[string]time.Duration{
			"processing": 5 * time.Millisecond,
			"network":    3 * time.Millisecond,
			"storage":    2 * time.Millisecond,
		},
	}
}

func CreateHighLatencyMetrics(target string) *LatencyMetrics {
	return &LatencyMetrics{
		Average:   100 * time.Millisecond,
		P50:       80 * time.Millisecond,
		P95:       200 * time.Millisecond,
		P99:       500 * time.Millisecond,
		Min:       10 * time.Millisecond,
		Max:       1000 * time.Millisecond,
		Timestamp: time.Now(),
		Source:    target,
	}
}

func CreateLowLatencyMetrics(target string) *LatencyMetrics {
	return &LatencyMetrics{
		Average:   1 * time.Millisecond,
		P50:       800 * time.Microsecond,
		P95:       2 * time.Millisecond,
		P99:       5 * time.Millisecond,
		Min:       100 * time.Microsecond,
		Max:       10 * time.Millisecond,
		Timestamp: time.Now(),
		Source:    target,
	}
}

func CreateDefaultThroughputMetrics(target string) *ThroughputMetrics {
	return &ThroughputMetrics{
		Downlink:  1000.0, // Mbps
		Uplink:    100.0,  // Mbps
		Unit:      "Mbps",
		Timestamp: time.Now(),
		Source:    target,
		Utilization: map[string]float64{
			"downlink": 75.0,
			"uplink":   60.0,
		},
	}
}

func CreateHighThroughputMetrics(target string) *ThroughputMetrics {
	return &ThroughputMetrics{
		Downlink:  10000.0, // Mbps
		Uplink:    1000.0,  // Mbps
		Unit:      "Mbps",
		Timestamp: time.Now(),
		Source:    target,
	}
}

func CreateLowThroughputMetrics(target string) *ThroughputMetrics {
	return &ThroughputMetrics{
		Downlink:  10.0, // Mbps
		Uplink:    1.0,  // Mbps
		Unit:      "Mbps",
		Timestamp: time.Now(),
		Source:    target,
	}
}

func CreateDefaultResourceMetrics(target string) *ResourceMetrics {
	return &ResourceMetrics{
		CPU: CPUMetrics{
			Usage:       75.0,
			Cores:       4,
			Frequency:   2.4,
			Temperature: 65.0,
		},
		Memory: MemoryMetrics{
			Used:      8 * 1024 * 1024 * 1024, // 8GB
			Available: 2 * 1024 * 1024 * 1024, // 2GB
			Total:     10 * 1024 * 1024 * 1024, // 10GB
			Usage:     80.0,
		},
		Network: *CreateDefaultNetworkMetrics(),
		Storage: StorageMetrics{
			Used:      500 * 1024 * 1024 * 1024, // 500GB
			Available: 500 * 1024 * 1024 * 1024, // 500GB
			Total:     1024 * 1024 * 1024 * 1024, // 1TB
			Usage:     50.0,
			IOPS:      10000,
		},
		Timestamp: time.Now(),
		Source:    target,
	}
}

func CreateHighResourceUsageMetrics(target string) *ResourceMetrics {
	metrics := CreateDefaultResourceMetrics(target)
	metrics.CPU.Usage = 95.0
	metrics.Memory.Usage = 95.0
	return metrics
}

func CreateLowResourceUsageMetrics(target string) *ResourceMetrics {
	metrics := CreateDefaultResourceMetrics(target)
	metrics.CPU.Usage = 20.0
	metrics.Memory.Usage = 30.0
	return metrics
}

func CreateDefaultNetworkMetrics() *NetworkMetrics {
	return &NetworkMetrics{
		BytesIn:     1024 * 1024 * 1024, // 1GB
		BytesOut:    500 * 1024 * 1024,  // 500MB
		PacketsIn:   1000000,
		PacketsOut:  500000,
		ErrorsIn:    10,
		ErrorsOut:   5,
		Bandwidth:   1000.0, // Mbps
		Utilization: 60.0,
	}
}

func CreateDefaultHistoricalMetrics(target string, duration time.Duration) *HistoricalMetrics {
	now := time.Now()
	start := now.Add(-duration)
	interval := duration / 100 // 100 data points

	latencyPoints := make([]LatencyDataPoint, 100)
	throughputPoints := make([]ThroughputDataPoint, 100)
	resourcePoints := make([]ResourceDataPoint, 100)

	for i := 0; i < 100; i++ {
		timestamp := start.Add(time.Duration(i) * interval)

		latencyPoints[i] = LatencyDataPoint{
			Timestamp: timestamp,
			Value:     time.Duration(10+i/10) * time.Millisecond,
		}

		throughputPoints[i] = ThroughputDataPoint{
			Timestamp: timestamp,
			Downlink:  1000.0 + float64(i*10),
			Uplink:    100.0 + float64(i),
		}

		resourcePoints[i] = ResourceDataPoint{
			Timestamp: timestamp,
			CPU:       50.0 + float64(i%50),
			Memory:    60.0 + float64(i%40),
			Network:   40.0 + float64(i%30),
		}
	}

	return &HistoricalMetrics{
		Latency:    latencyPoints,
		Throughput: throughputPoints,
		Resource:   resourcePoints,
		TimeRange: TimeRange{
			Start: start,
			End:   now,
		},
		Interval: interval,
	}
}

// Helper functions for creating metrics with specific characteristics
func CreateEdgeZoneMetrics(zone string) *ResourceMetrics {
	metrics := CreateDefaultResourceMetrics(zone)
	// Edge zones typically have limited resources
	metrics.CPU.Cores = 2
	metrics.Memory.Total = 4 * 1024 * 1024 * 1024 // 4GB
	metrics.Storage.Total = 100 * 1024 * 1024 * 1024 // 100GB
	return metrics
}

func CreateCloudZoneMetrics(zone string) *ResourceMetrics {
	metrics := CreateDefaultResourceMetrics(zone)
	// Cloud zones have abundant resources
	metrics.CPU.Cores = 32
	metrics.Memory.Total = 128 * 1024 * 1024 * 1024 // 128GB
	metrics.Storage.Total = 10 * 1024 * 1024 * 1024 * 1024 // 10TB
	return metrics
}

func CreateCongestedNetworkMetrics(target string) *NetworkMetrics {
	metrics := CreateDefaultNetworkMetrics()
	metrics.Utilization = 95.0
	metrics.ErrorsIn = 1000
	metrics.ErrorsOut = 500
	return metrics
}

func CreateOptimalNetworkMetrics(target string) *NetworkMetrics {
	metrics := CreateDefaultNetworkMetrics()
	metrics.Utilization = 30.0
	metrics.ErrorsIn = 0
	metrics.ErrorsOut = 0
	return metrics
}