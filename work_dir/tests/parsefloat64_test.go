package tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseFloat64WithUnits tests parsing float values with unit suffixes
func TestParseFloat64WithUnits(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue float64
		wantUnit  string
		wantErr   bool
		errMsg    string
	}{
		// Bandwidth units
		{
			name:      "bandwidth in Mbps",
			input:     "4.5 Mbps",
			wantValue: 4.5,
			wantUnit:  "Mbps",
			wantErr:   false,
		},
		{
			name:      "bandwidth in Gbps",
			input:     "100 Gbps",
			wantValue: 100.0,
			wantUnit:  "Gbps",
			wantErr:   false,
		},
		{
			name:      "bandwidth in Kbps",
			input:     "512 Kbps",
			wantValue: 512.0,
			wantUnit:  "Kbps",
			wantErr:   false,
		},
		{
			name:      "bandwidth without space",
			input:     "10Mbps",
			wantValue: 10.0,
			wantUnit:  "Mbps",
			wantErr:   false,
		},

		// Latency units
		{
			name:      "latency in milliseconds",
			input:     "10ms",
			wantValue: 10.0,
			wantUnit:  "ms",
			wantErr:   false,
		},
		{
			name:      "latency in microseconds",
			input:     "500us",
			wantValue: 500.0,
			wantUnit:  "us",
			wantErr:   false,
		},
		{
			name:      "latency in seconds",
			input:     "1.5s",
			wantValue: 1.5,
			wantUnit:  "s",
			wantErr:   false,
		},

		// Percentage
		{
			name:      "percentage value",
			input:     "99.9%",
			wantValue: 99.9,
			wantUnit:  "%",
			wantErr:   false,
		},
		{
			name:      "integer percentage",
			input:     "100%",
			wantValue: 100.0,
			wantUnit:  "%",
			wantErr:   false,
		},

		// Memory units
		{
			name:      "memory in MB",
			input:     "256 MB",
			wantValue: 256.0,
			wantUnit:  "MB",
			wantErr:   false,
		},
		{
			name:      "memory in GB",
			input:     "8 GB",
			wantValue: 8.0,
			wantUnit:  "GB",
			wantErr:   false,
		},
		{
			name:      "memory in KB",
			input:     "1024 KB",
			wantValue: 1024.0,
			wantUnit:  "KB",
			wantErr:   false,
		},

		// No unit
		{
			name:      "plain number",
			input:     "42.5",
			wantValue: 42.5,
			wantUnit:  "",
			wantErr:   false,
		},
		{
			name:      "integer without unit",
			input:     "100",
			wantValue: 100.0,
			wantUnit:  "",
			wantErr:   false,
		},

		// Scientific notation
		{
			name:      "scientific notation",
			input:     "1.5e3 Mbps",
			wantValue: 1500.0,
			wantUnit:  "Mbps",
			wantErr:   false,
		},
		{
			name:      "negative exponent",
			input:     "1.5e-3 ms",
			wantValue: 0.0015,
			wantUnit:  "ms",
			wantErr:   false,
		},

		// Edge cases
		{
			name:      "zero value",
			input:     "0 Mbps",
			wantValue: 0.0,
			wantUnit:  "Mbps",
			wantErr:   false,
		},
		{
			name:      "negative value",
			input:     "-5.5 dB",
			wantValue: -5.5,
			wantUnit:  "dB",
			wantErr:   false,
		},
		{
			name:      "very large value",
			input:     "999999.99 Gbps",
			wantValue: 999999.99,
			wantUnit:  "Gbps",
			wantErr:   false,
		},
		{
			name:      "very small value",
			input:     "0.001 ms",
			wantValue: 0.001,
			wantUnit:  "ms",
			wantErr:   false,
		},

		// Whitespace variations
		{
			name:      "multiple spaces",
			input:     "10   Mbps",
			wantValue: 10.0,
			wantUnit:  "Mbps",
			wantErr:   false,
		},
		{
			name:      "leading whitespace",
			input:     "  50 Mbps",
			wantValue: 50.0,
			wantUnit:  "Mbps",
			wantErr:   false,
		},
		{
			name:      "trailing whitespace",
			input:     "50 Mbps  ",
			wantValue: 50.0,
			wantUnit:  "Mbps",
			wantErr:   false,
		},

		// Error cases
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "empty input",
		},
		{
			name:    "only unit",
			input:   "Mbps",
			wantErr: true,
			errMsg:  "missing value",
		},
		{
			name:    "multiple units",
			input:   "10 Mbps Gbps",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "invalid number",
			input:   "abc Mbps",
			wantErr: true,
			errMsg:  "invalid number",
		},
		{
			name:    "special characters",
			input:   "10@ Mbps",
			wantErr: true,
			errMsg:  "invalid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, unit, err := parseFloat64WithUnits(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.wantValue, value, 0.0001)
				assert.Equal(t, tt.wantUnit, unit)
			}
		})
	}
}

// TestConvertToBaseUnit tests unit conversion to base units
func TestConvertToBaseUnit(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		fromUnit  string
		toUnit    string
		wantValue float64
		wantErr   bool
	}{
		// Bandwidth conversions
		{
			name:      "Gbps to Mbps",
			value:     1.0,
			fromUnit:  "Gbps",
			toUnit:    "Mbps",
			wantValue: 1000.0,
			wantErr:   false,
		},
		{
			name:      "Mbps to Kbps",
			value:     1.0,
			fromUnit:  "Mbps",
			toUnit:    "Kbps",
			wantValue: 1000.0,
			wantErr:   false,
		},
		{
			name:      "Kbps to Mbps",
			value:     1000.0,
			fromUnit:  "Kbps",
			toUnit:    "Mbps",
			wantValue: 1.0,
			wantErr:   false,
		},

		// Time conversions
		{
			name:      "seconds to milliseconds",
			value:     1.0,
			fromUnit:  "s",
			toUnit:    "ms",
			wantValue: 1000.0,
			wantErr:   false,
		},
		{
			name:      "milliseconds to microseconds",
			value:     1.0,
			fromUnit:  "ms",
			toUnit:    "us",
			wantValue: 1000.0,
			wantErr:   false,
		},

		// Memory conversions
		{
			name:      "GB to MB",
			value:     1.0,
			fromUnit:  "GB",
			toUnit:    "MB",
			wantValue: 1024.0,
			wantErr:   false,
		},
		{
			name:      "MB to KB",
			value:     1.0,
			fromUnit:  "MB",
			toUnit:    "KB",
			wantValue: 1024.0,
			wantErr:   false,
		},

		// Error cases
		{
			name:     "incompatible units",
			value:    1.0,
			fromUnit: "Mbps",
			toUnit:   "ms",
			wantErr:  true,
		},
		{
			name:     "unknown source unit",
			value:    1.0,
			fromUnit: "unknown",
			toUnit:   "Mbps",
			wantErr:  true,
		},
		{
			name:     "unknown target unit",
			value:    1.0,
			fromUnit: "Mbps",
			toUnit:   "unknown",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToBaseUnit(tt.value, tt.fromUnit, tt.toUnit)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.wantValue, result, 0.0001)
			}
		})
	}
}

// TestParseFloat64Precision tests floating point precision
func TestParseFloat64Precision(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue float64
	}{
		{
			name:      "high precision decimal",
			input:     "3.141592653589793 Mbps",
			wantValue: 3.141592653589793,
		},
		{
			name:      "repeating decimal",
			input:     "0.333333333333 ms",
			wantValue: 0.333333333333,
		},
		{
			name:      "max float64",
			input:     "1.7976931348623157e+308",
			wantValue: math.MaxFloat64,
		},
		{
			name:      "min positive float64",
			input:     "2.2250738585072014e-308",
			wantValue: math.SmallestNonzeroFloat64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _, err := parseFloat64WithUnits(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

// TestNormalizeUnit tests unit normalization
func TestNormalizeUnit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantUnit string
	}{
		{
			name:     "lowercase to standard",
			input:    "mbps",
			wantUnit: "Mbps",
		},
		{
			name:     "uppercase to standard",
			input:    "MBPS",
			wantUnit: "Mbps",
		},
		{
			name:     "mixed case",
			input:    "MbPs",
			wantUnit: "Mbps",
		},
		{
			name:     "milliseconds variants",
			input:    "MS",
			wantUnit: "ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeUnit(tt.input)
			assert.Equal(t, tt.wantUnit, normalized)
		})
	}
}

// Functions are now implemented in parsefloat64.go