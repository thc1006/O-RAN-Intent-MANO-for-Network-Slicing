package dashboard

import (
	"fmt"
	"testing"
	"time"
)

func TestValidateHistoryLimit(t *testing.T) {
	tests := []struct {
		name             string
		limit            int
		availableHistory int
		expectedResult   int
		expectLog        bool
	}{
		{
			name:             "Valid normal limit",
			limit:            50,
			availableHistory: 100,
			expectedResult:   50,
			expectLog:        false,
		},
		{
			name:             "Zero limit with small history",
			limit:            0,
			availableHistory: 50,
			expectedResult:   50,
			expectLog:        false,
		},
		{
			name:             "Zero limit with large history",
			limit:            0,
			availableHistory: 500,
			expectedResult:   DefaultHistoryLimit,
			expectLog:        false,
		},
		{
			name:             "Negative limit",
			limit:            -10,
			availableHistory: 100,
			expectedResult:   DefaultHistoryLimit,
			expectLog:        false,
		},
		{
			name:             "Limit exceeding MaxHistoryLimit",
			limit:            2000,
			availableHistory: 3000,
			expectedResult:   MaxHistoryLimit,
			expectLog:        true,
		},
		{
			name:             "Limit exceeding AbsoluteMaxHistoryLimit",
			limit:            50000,
			availableHistory: 60000,
			expectedResult:   AbsoluteMaxHistoryLimit,
			expectLog:        true,
		},
		{
			name:             "Limit greater than available history",
			limit:            200,
			availableHistory: 100,
			expectedResult:   100,
			expectLog:        false,
		},
		{
			name:             "Edge case - exactly at MaxHistoryLimit",
			limit:            MaxHistoryLimit,
			availableHistory: MaxHistoryLimit + 100,
			expectedResult:   MaxHistoryLimit,
			expectLog:        false,
		},
		{
			name:             "Edge case - exactly at AbsoluteMaxHistoryLimit",
			limit:            AbsoluteMaxHistoryLimit,
			availableHistory: AbsoluteMaxHistoryLimit + 100,
			expectedResult:   AbsoluteMaxHistoryLimit,
			expectLog:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateHistoryLimit(tt.limit, tt.availableHistory)
			if result != tt.expectedResult {
				t.Errorf("validateHistoryLimit(%d, %d) = %d, want %d",
					tt.limit, tt.availableHistory, result, tt.expectedResult)
			}
		})
	}
}

func TestGetMetricsHistoryMemoryExhaustionProtection(t *testing.T) {
	// Create aggregator with test data
	aggregator := &MetricsAggregator{
		metricsHistory: make([]*TestMetrics, 0),
		maxHistorySize: DefaultHistoryLimit,
	}

	// Add some test metrics to history
	for i := 0; i < 500; i++ {
		metrics := &TestMetrics{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			TestSuiteResults: map[string]*TestSuiteResult{
				"test": {
					Name:        fmt.Sprintf("test-%d", i),
					TotalTests:  10,
					PassedTests: 9,
				},
			},
		}
		aggregator.metricsHistory = append(aggregator.metricsHistory, metrics)
	}

	tests := []struct {
		name           string
		limit          int
		expectedLength int
		shouldNotPanic bool
	}{
		{
			name:           "Normal request",
			limit:          10,
			expectedLength: 10,
			shouldNotPanic: true,
		},
		{
			name:           "Zero limit",
			limit:          0,
			expectedLength: DefaultHistoryLimit,
			shouldNotPanic: true,
		},
		{
			name:           "Negative limit",
			limit:          -100,
			expectedLength: DefaultHistoryLimit,
			shouldNotPanic: true,
		},
		{
			name:           "Attempt memory exhaustion with huge limit",
			limit:          1000000, // 1 million - should be capped
			expectedLength: AbsoluteMaxHistoryLimit,
			shouldNotPanic: true,
		},
		{
			name:           "Extreme attack scenario",
			limit:          999999999, // Nearly 1 billion - should be safely capped
			expectedLength: AbsoluteMaxHistoryLimit,
			shouldNotPanic: true,
		},
		{
			name:           "Request more than available",
			limit:          1000,
			expectedLength: 500, // Available history length
			shouldNotPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.shouldNotPanic {
					t.Errorf("GetMetricsHistory(%d) panicked: %v", tt.limit, r)
				}
			}()

			result := aggregator.GetMetricsHistory(tt.limit)

			if len(result) != tt.expectedLength {
				t.Errorf("GetMetricsHistory(%d) returned %d items, expected %d",
					tt.limit, len(result), tt.expectedLength)
			}

			// Verify the result is sorted by timestamp (newest first)
			for i := 1; i < len(result); i++ {
				if result[i-1].Timestamp.Before(result[i].Timestamp) {
					t.Errorf("Result not sorted correctly: timestamp at index %d is before timestamp at index %d",
						i-1, i)
				}
			}
		})
	}
}

func TestValidateAggregatorConfigSecurity(t *testing.T) {
	tests := []struct {
		name        string
		config      *AggregatorConfig
		expectError bool
		expectLog   bool
	}{
		{
			name: "Valid configuration",
			config: &AggregatorConfig{
				MaxHistorySize:  100,
				UpdateInterval:  time.Minute,
				OutputDirectory: "valid/path",
			},
			expectError: false,
			expectLog:   false,
		},
		{
			name: "Invalid MaxHistorySize - zero",
			config: &AggregatorConfig{
				MaxHistorySize:  0,
				UpdateInterval:  time.Minute,
				OutputDirectory: "valid/path",
			},
			expectError: false, // Should be auto-corrected
			expectLog:   true,
		},
		{
			name: "Invalid MaxHistorySize - negative",
			config: &AggregatorConfig{
				MaxHistorySize:  -100,
				UpdateInterval:  time.Minute,
				OutputDirectory: "valid/path",
			},
			expectError: false, // Should be auto-corrected
			expectLog:   true,
		},
		{
			name: "MaxHistorySize exceeding AbsoluteMaxHistoryLimit",
			config: &AggregatorConfig{
				MaxHistorySize:  50000,
				UpdateInterval:  time.Minute,
				OutputDirectory: "valid/path",
			},
			expectError: false, // Should be auto-corrected
			expectLog:   true,
		},
		{
			name: "MaxHistorySize exceeding MaxHistoryLimit",
			config: &AggregatorConfig{
				MaxHistorySize:  5000,
				UpdateInterval:  time.Minute,
				OutputDirectory: "valid/path",
			},
			expectError: false, // Should log warning
			expectLog:   true,
		},
		{
			name: "Path traversal attack in output directory",
			config: &AggregatorConfig{
				MaxHistorySize:  100,
				UpdateInterval:  time.Minute,
				OutputDirectory: "../../../etc/passwd",
			},
			expectError: true,
			expectLog:   false,
		},
		{
			name: "Another path traversal variant",
			config: &AggregatorConfig{
				MaxHistorySize:  100,
				UpdateInterval:  time.Minute,
				OutputDirectory: "valid/path/../../../sensitive",
			},
			expectError: true,
			expectLog:   false,
		},
		{
			name: "Very frequent update interval",
			config: &AggregatorConfig{
				MaxHistorySize:  100,
				UpdateInterval:  time.Second * 5, // Too frequent
				OutputDirectory: "valid/path",
			},
			expectError: false, // Should be auto-corrected
			expectLog:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAggregatorConfig(tt.config)

			if tt.expectError && err == nil {
				t.Errorf("validateAggregatorConfig() expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("validateAggregatorConfig() unexpected error: %v", err)
			}

			// Verify that dangerous values were corrected
			if !tt.expectError {
				if tt.config.MaxHistorySize <= 0 {
					if tt.config.MaxHistorySize != DefaultHistoryLimit {
						t.Errorf("Expected MaxHistorySize to be corrected to %d, got %d",
							DefaultHistoryLimit, tt.config.MaxHistorySize)
					}
				} else if tt.config.MaxHistorySize > AbsoluteMaxHistoryLimit {
					if tt.config.MaxHistorySize != AbsoluteMaxHistoryLimit {
						t.Errorf("Expected MaxHistorySize to be capped to %d, got %d",
							AbsoluteMaxHistoryLimit, tt.config.MaxHistorySize)
					}
				}

				if tt.config.UpdateInterval < time.Second*10 {
					if tt.config.UpdateInterval != time.Second*10 {
						t.Errorf("Expected UpdateInterval to be corrected to %v, got %v",
							time.Second*10, tt.config.UpdateInterval)
					}
				}
			}
		})
	}
}

func BenchmarkGetMetricsHistoryLargeLimit(b *testing.B) {
	// Create aggregator with large history
	aggregator := &MetricsAggregator{
		metricsHistory: make([]*TestMetrics, 0),
		maxHistorySize: AbsoluteMaxHistoryLimit,
	}

	// Add maximum allowed history
	for i := 0; i < AbsoluteMaxHistoryLimit; i++ {
		metrics := &TestMetrics{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
		aggregator.metricsHistory = append(aggregator.metricsHistory, metrics)
	}

	b.ResetTimer()

	// Benchmark with various limits to ensure performance doesn't degrade with attack scenarios
	limits := []int{100, 1000, 10000, 100000, 1000000, 999999999}

	for _, limit := range limits {
		b.Run(fmt.Sprintf("limit_%d", limit), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				result := aggregator.GetMetricsHistory(limit)
				// Verify the result is reasonable
				if len(result) > AbsoluteMaxHistoryLimit {
					b.Errorf("Returned too many results: %d", len(result))
				}
			}
		})
	}
}

func TestMemoryAllocationLimits(t *testing.T) {
	// Test that our limits prevent excessive memory allocation
	testCases := []struct {
		requestedLimit int
		expectedMax    int
	}{
		{100, 100},
		{MaxHistoryLimit, MaxHistoryLimit},
		{MaxHistoryLimit + 1, MaxHistoryLimit + 1}, // Should log warning but allow up to absolute max
		{AbsoluteMaxHistoryLimit, AbsoluteMaxHistoryLimit},
		{AbsoluteMaxHistoryLimit + 1, AbsoluteMaxHistoryLimit},
		{1000000, AbsoluteMaxHistoryLimit},   // 1 million request capped
		{999999999, AbsoluteMaxHistoryLimit}, // Nearly 1 billion request capped
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("limit_%d", tc.requestedLimit), func(t *testing.T) {
			validatedLimit := validateHistoryLimit(tc.requestedLimit, 50000)
			if validatedLimit > tc.expectedMax {
				t.Errorf("validateHistoryLimit(%d) = %d, should not exceed %d",
					tc.requestedLimit, validatedLimit, tc.expectedMax)
			}
		})
	}
}
