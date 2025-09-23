package tc

import (
	"strings"
	"testing"
)

func TestShaper_ApplyRules_InputValidation(t *testing.T) {
	shaper := NewShaper()

	tests := []struct {
		name        string
		iface       string
		rules       []Rule
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid interface and rules",
			iface:       "eth0",
			rules:       []Rule{{Priority: 1, Rate: 1000, Burst: 100, Latency: 10.5}},
			expectError: true, // Will fail due to actual tc command, but validation should pass
			errorSubstr: "failed to clear interface", // Expected since we can't actually run tc
		},
		{
			name:        "invalid interface name",
			iface:       "eth0; rm -rf /",
			rules:       []Rule{{Priority: 1, Rate: 1000, Burst: 100, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid interface name",
		},
		{
			name:        "empty interface name",
			iface:       "",
			rules:       []Rule{{Priority: 1, Rate: 1000, Burst: 100, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid interface name",
		},
		{
			name:        "invalid priority - zero",
			iface:       "eth0",
			rules:       []Rule{{Priority: 0, Rate: 1000, Burst: 100, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid priority",
		},
		{
			name:        "invalid priority - too high",
			iface:       "eth0",
			rules:       []Rule{{Priority: 70000, Rate: 1000, Burst: 100, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid priority",
		},
		{
			name:        "invalid rate - zero",
			iface:       "eth0",
			rules:       []Rule{{Priority: 1, Rate: 0, Burst: 100, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid rate",
		},
		{
			name:        "invalid rate - negative",
			iface:       "eth0",
			rules:       []Rule{{Priority: 1, Rate: -1000, Burst: 100, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid rate",
		},
		{
			name:        "invalid burst - zero",
			iface:       "eth0",
			rules:       []Rule{{Priority: 1, Rate: 1000, Burst: 0, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid burst",
		},
		{
			name:        "invalid burst - negative",
			iface:       "eth0",
			rules:       []Rule{{Priority: 1, Rate: 1000, Burst: -100, Latency: 10.5}},
			expectError: true,
			errorSubstr: "invalid burst",
		},
		{
			name:        "invalid latency - negative",
			iface:       "eth0",
			rules:       []Rule{{Priority: 1, Rate: 1000, Burst: 100, Latency: -10.5}},
			expectError: true,
			errorSubstr: "invalid latency",
		},
		{
			name:        "valid latency - zero",
			iface:       "eth0",
			rules:       []Rule{{Priority: 1, Rate: 1000, Burst: 100, Latency: 0}},
			expectError: true, // Will fail due to actual tc command, but validation should pass
			errorSubstr: "failed to clear interface", // Expected since we can't actually run tc
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shaper.ApplyRules(tt.iface, tt.rules)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestShaper_GetStatistics_InputValidation(t *testing.T) {
	shaper := NewShaper()

	tests := []struct {
		name        string
		iface       string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid interface",
			iface:       "eth0",
			expectError: true, // Will fail due to actual tc command, but validation should pass
			errorSubstr: "failed to get statistics", // Expected since we can't actually run tc
		},
		{
			name:        "invalid interface with command injection",
			iface:       "eth0; rm -rf /",
			expectError: true,
			errorSubstr: "invalid interface name",
		},
		{
			name:        "empty interface",
			iface:       "",
			expectError: true,
			errorSubstr: "invalid interface name",
		},
		{
			name:        "interface with special characters",
			iface:       "eth0$test",
			expectError: true,
			errorSubstr: "invalid interface name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := shaper.GetStatistics(tt.iface)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRule_ValidationBoundaries(t *testing.T) {
	shaper := NewShaper()

	// Test boundary values
	tests := []struct {
		name  string
		rule  Rule
		valid bool
	}{
		{"minimum valid priority", Rule{Priority: 1, Rate: 1, Burst: 1, Latency: 0}, true},
		{"maximum valid priority", Rule{Priority: 65535, Rate: 1, Burst: 1, Latency: 0}, true},
		{"minimum valid rate", Rule{Priority: 1, Rate: 1, Burst: 1, Latency: 0}, true},
		{"minimum valid burst", Rule{Priority: 1, Rate: 1, Burst: 1, Latency: 0}, true},
		{"minimum valid latency", Rule{Priority: 1, Rate: 1, Burst: 1, Latency: 0}, true},

		{"invalid priority - boundary", Rule{Priority: 0, Rate: 1, Burst: 1, Latency: 0}, false},
		{"invalid priority - upper boundary", Rule{Priority: 65536, Rate: 1, Burst: 1, Latency: 0}, false},
		{"invalid rate - boundary", Rule{Priority: 1, Rate: 0, Burst: 1, Latency: 0}, false},
		{"invalid burst - boundary", Rule{Priority: 1, Rate: 1, Burst: 0, Latency: 0}, false},
		{"invalid latency - boundary", Rule{Priority: 1, Rate: 1, Burst: 1, Latency: -0.1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shaper.ApplyRules("eth0", []Rule{tt.rule})

			if tt.valid {
				// Should only fail on actual tc execution, not validation
				if err != nil && !strings.Contains(err.Error(), "failed to clear interface") {
					t.Errorf("Unexpected validation error: %v", err)
				}
			} else {
				// Should fail on validation
				if err == nil {
					t.Errorf("Expected validation error but got none")
				} else if strings.Contains(err.Error(), "failed to clear interface") {
					t.Errorf("Should have failed on validation, not tc execution")
				}
			}
		})
	}
}

func TestNewShaper(t *testing.T) {
	shaper := NewShaper()

	if shaper == nil {
		t.Fatal("NewShaper returned nil")
	}

	if shaper.currentConfigs == nil {
		t.Error("currentConfigs map not initialized")
	}

	if len(shaper.currentConfigs) != 0 {
		t.Error("currentConfigs should be empty initially")
	}
}

func TestCleanup(t *testing.T) {
	shaper := NewShaper()

	// Add a mock configuration
	shaper.currentConfigs["eth0"] = &Config{
		Interface: "eth0",
		Rules: []Rule{{Priority: 1, Rate: 1000, Burst: 100, Latency: 10}},
	}

	initialConfigs := len(shaper.currentConfigs)
	if initialConfigs == 0 {
		t.Fatal("Test setup failed: no configs added")
	}

	err := shaper.Cleanup()

	// In test environment, tc commands will fail, so cleanup should fail and configs remain
	// This is correct behavior - we don't want to claim configs are cleared if tc fails
	if err != nil {
		t.Logf("Cleanup failed as expected in test environment: %v", err)
		// Configs should remain since clearInterface failed
		if len(shaper.currentConfigs) != initialConfigs {
			t.Errorf("Configs should remain unchanged when cleanup fails, had %d, now has %d",
				initialConfigs, len(shaper.currentConfigs))
		}
	} else {
		t.Log("Cleanup succeeded (unexpected in test environment)")
		// If cleanup succeeded, configs should be cleared
		if len(shaper.currentConfigs) != 0 {
			t.Error("Configs should be cleared when cleanup succeeds")
		}
	}
}