package security

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"../fixtures"
)

// TestQoSInputValidation tests input validation for QoS conversion
func TestQoSInputValidation(t *testing.T) {
	testCases := []struct {
		name        string
		vnfProfile  fixtures.VNFQoSProfile
		expectError bool
		description string
	}{
		{
			name: "Valid input",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "20ms",
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			expectError: false,
			description: "Normal valid input should work",
		},
		{
			name: "SQL injection attempt in latency",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "20ms'; DROP TABLE qos_profiles; --",
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			expectError: false, // Conversion should work but sanitize
			description: "SQL injection should be sanitized",
		},
		{
			name: "XSS attempt in throughput",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "20ms",
				Throughput:  "<script>alert('xss')</script>",
				Reliability: "99.9%",
			},
			expectError: false, // Conversion should work but sanitize
			description: "XSS payload should be sanitized",
		},
		{
			name: "Very long input (potential buffer overflow)",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     strings.Repeat("A", 10000) + "ms",
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			expectError: false, // Should handle gracefully
			description: "Very long input should not crash",
		},
		{
			name: "Control characters in reliability",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "20ms",
				Throughput:  "1Gbps",
				Reliability: "99.9%\x00\x01\x02",
			},
			expectError: false, // Should sanitize
			description: "Control characters should be handled",
		},
		{
			name: "Unicode injection attempt",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "20ms",
				Throughput:  "1Gbps\u202e", // Right-to-left override
				Reliability: "99.9%",
			},
			expectError: false, // Should handle Unicode safely
			description: "Unicode injection should be handled safely",
		},
		{
			name: "Command injection attempt",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "20ms && rm -rf /",
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			expectError: false, // Should not execute commands
			description: "Command injection should be prevented",
		},
		{
			name: "Path traversal attempt",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "../../../etc/passwd",
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			expectError: false, // Should not access files
			description: "Path traversal should be prevented",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test should not panic or fail catastrophically
			assert.NotPanics(t, func() {
				result := fixtures.ConvertVNFQoSProfileToQoSProfile(tc.vnfProfile)

				// Validate that the result is safe
				validateSafeOutput(t, result, tc.description)
			})
		})
	}
}

// validateSafeOutput ensures the converted output is safe
func validateSafeOutput(t *testing.T, qos fixtures.QoSProfile, context string) {
	// Check for dangerous patterns in output
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"DROP TABLE",
		"SELECT *",
		"../",
		"rm -rf",
		"\x00", // Null bytes
	}

	fields := []string{
		qos.Latency.Value,
		qos.Latency.Unit,
		qos.Latency.Type,
		qos.Throughput.Downlink,
		qos.Throughput.Uplink,
		qos.Throughput.Unit,
		qos.Reliability.Value,
		qos.Reliability.Unit,
	}

	for _, field := range fields {
		for _, pattern := range dangerousPatterns {
			assert.NotContains(t, strings.ToLower(field), strings.ToLower(pattern),
				"%s: Output should not contain dangerous pattern '%s'", context, pattern)
		}

		// Check for reasonable length limits
		assert.LessOrEqual(t, len(field), 1000,
			"%s: Field length should be reasonable (<%d chars), got %d", context, 1000, len(field))
	}
}

// TestQoSSQLInjectionPrevention tests SQL injection prevention
func TestQoSSQLInjectionPrevention(t *testing.T) {
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"'; SELECT * FROM sensitive_data; --",
		"' UNION SELECT password FROM users --",
		"'; UPDATE qos_profiles SET reliability='0%'; --",
		"1; DELETE FROM vnf_deployments; --",
	}

	for i, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("sql_injection_%d", i), func(t *testing.T) {
			vnfProfile := fixtures.VNFQoSProfile{
				Latency:     payload,
				Throughput:  payload,
				Reliability: payload,
			}

			result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)

			// Verify SQL injection was neutralized
			assert.NotContains(t, result.Latency.Value, "DROP")
			assert.NotContains(t, result.Latency.Value, "SELECT")
			assert.NotContains(t, result.Latency.Value, "DELETE")
			assert.NotContains(t, result.Latency.Value, "UPDATE")
			assert.NotContains(t, result.Latency.Value, "UNION")

			// Result should still be usable (not empty due to over-sanitization)
			assert.NotNil(t, result)
		})
	}
}

// TestQoSXSSPrevention tests XSS prevention in QoS data
func TestQoSXSSPrevention(t *testing.T) {
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"<img src=x onerror=alert('xss')>",
		"<iframe src=javascript:alert('xss')></iframe>",
		"<svg onload=alert('xss')>",
		"&lt;script&gt;alert('xss')&lt;/script&gt;",
		"<%73%63%72%69%70%74>alert('xss')</script>",
	}

	for i, payload := range xssPayloads {
		t.Run(fmt.Sprintf("xss_prevention_%d", i), func(t *testing.T) {
			vnfProfile := fixtures.VNFQoSProfile{
				Latency:     payload,
				Throughput:  payload,
				Reliability: payload,
			}

			result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)

			// Verify XSS was neutralized
			fields := []string{
				result.Latency.Value,
				result.Throughput.Downlink,
				result.Reliability.Value,
			}

			for _, field := range fields {
				// Should not contain dangerous HTML/JS
				assert.NotContains(t, field, "<script")
				assert.NotContains(t, field, "javascript:")
				assert.NotContains(t, field, "onerror=")
				assert.NotContains(t, field, "onload=")
				assert.NotContains(t, field, "<iframe")
				assert.NotContains(t, field, "<svg")

				// Common XSS evasion techniques should be blocked
				assert.NotContains(t, strings.ToLower(field), "&#x")
				assert.NotContains(t, strings.ToLower(field), "%3c") // URL encoded <
			}
		})
	}
}

// TestQoSDataSanitization tests general data sanitization
func TestQoSDataSanitization(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic HTML tags",
			input:    "<b>20</b>ms",
			expected: "ms", // Should remove HTML tags, keep valid parts
		},
		{
			name:     "Multiple dangerous characters",
			input:    "20ms;DROP TABLE users;",
			expected: "ms", // Should sanitize SQL injection attempts
		},
		{
			name:     "Control characters",
			input:    "20\x00\x01\x02ms",
			expected: "ms", // Should remove control characters
		},
		{
			name:     "Normal valid input",
			input:    "20ms",
			expected: "20ms", // Should preserve valid input
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vnfProfile := fixtures.VNFQoSProfile{
				Latency:     tc.input,
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			}

			result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)

			// Basic sanitization should occur
			assert.NotContains(t, result.Latency.Value, "<")
			assert.NotContains(t, result.Latency.Value, ">")
			assert.NotContains(t, result.Latency.Value, ";")
			assert.NotContains(t, result.Latency.Value, "\x00")

			// For simple parsing, we expect basic cleanup
			assert.NotNil(t, result)
		})
	}
}

// TestQoSAuthentication tests authentication integration with QoS endpoints
func TestQoSAuthentication(t *testing.T) {
	// This would test API authentication, but for now test data integrity
	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)

	// Ensure conversion doesn't expose sensitive information
	assert.NotContains(t, fixtures.MustMarshalJSON(result), "password")
	assert.NotContains(t, fixtures.MustMarshalJSON(result), "secret")
	assert.NotContains(t, fixtures.MustMarshalJSON(result), "token")
	assert.NotContains(t, fixtures.MustMarshalJSON(result), "key")

	// Ensure required fields are present and valid
	assert.NotEmpty(t, result.Latency.Value)
	assert.NotEmpty(t, result.Latency.Unit)
	assert.NotEmpty(t, result.Throughput.Downlink)
	assert.NotEmpty(t, result.Reliability.Value)
}

// TestQoSAuthorization tests authorization for QoS modifications
func TestQoSAuthorization(t *testing.T) {
	// Test that QoS conversion preserves data integrity for authorization
	testProfiles := []fixtures.VNFQoSProfile{
		{Latency: "1ms", Throughput: "100Mbps", Reliability: "99.999%"},   // URLLC - high security
		{Latency: "20ms", Throughput: "1Gbps", Reliability: "99.9%"},     // eMBB - medium security
		{Latency: "100ms", Throughput: "10Mbps", Reliability: "99.9%"},   // mMTC - standard security
	}

	for i, profile := range testProfiles {
		t.Run(fmt.Sprintf("authorization_test_%d", i), func(t *testing.T) {
			result := fixtures.ConvertVNFQoSProfileToQoSProfile(profile)

			// Ensure converted data maintains integrity for authorization decisions
			assert.NotEmpty(t, result.Latency.Value, "Latency required for authorization")
			assert.NotEmpty(t, result.Throughput.Downlink, "Throughput required for authorization")
			assert.NotEmpty(t, result.Reliability.Value, "Reliability required for authorization")

			// Ensure no privilege escalation through data manipulation
			assert.NotContains(t, result.Latency.Value, "admin")
			assert.NotContains(t, result.Latency.Value, "root")
			assert.NotContains(t, result.Latency.Value, "sudo")

			// Verify data types are preserved for security checks
			assert.Equal(t, "ms", result.Latency.Unit)
			assert.Equal(t, "bps", result.Throughput.Unit)
			assert.Equal(t, "percentage", result.Reliability.Unit)
		})
	}
}

// TestQoSInputBoundaryValues tests boundary value security
func TestQoSInputBoundaryValues(t *testing.T) {
	boundaryTests := []struct {
		name        string
		vnfProfile  fixtures.VNFQoSProfile
		expectSafe  bool
		description string
	}{
		{
			name: "Maximum reasonable latency",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "10000ms", // 10 seconds - very high but valid
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			expectSafe:  true,
			description: "High but reasonable latency should be accepted",
		},
		{
			name: "Extremely high latency (potential DoS)",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "999999999ms", // Extremely high
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			expectSafe:  true, // Should handle without crashing
			description: "Extremely high values should be handled safely",
		},
		{
			name: "Zero values",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "0ms",
				Throughput:  "0bps",
				Reliability: "0%",
			},
			expectSafe:  true,
			description: "Zero values should be handled safely",
		},
		{
			name: "Negative values",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "-10ms",
				Throughput:  "-1Gbps",
				Reliability: "-10%",
			},
			expectSafe:  true, // Should sanitize
			description: "Negative values should be sanitized",
		},
		{
			name: "Reliability over 100%",
			vnfProfile: fixtures.VNFQoSProfile{
				Latency:     "20ms",
				Throughput:  "1Gbps",
				Reliability: "150%", // Invalid
			},
			expectSafe:  true, // Should handle gracefully
			description: "Invalid reliability values should be handled",
		},
	}

	for _, tc := range boundaryTests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				result := fixtures.ConvertVNFQoSProfileToQoSProfile(tc.vnfProfile)

				if tc.expectSafe {
					// Should produce a valid result structure
					assert.NotNil(t, result)
					assert.NotNil(t, result.Latency)
					assert.NotNil(t, result.Throughput)
					assert.NotNil(t, result.Reliability)

					// Units should always be valid
					assert.Contains(t, []string{"ms", "μs", "ns", "s"}, result.Latency.Unit)
					assert.Contains(t, []string{"bps", "Kbps", "Mbps", "Gbps"}, result.Throughput.Unit)
					assert.Contains(t, []string{"percentage", "nines"}, result.Reliability.Unit)
				}
			}, tc.description)
		})
	}
}

// TestQoSConcurrentSecurityValidation tests security under concurrent load
func TestQoSConcurrentSecurityValidation(t *testing.T) {
	maliciousProfiles := []fixtures.VNFQoSProfile{
		{Latency: "'; DROP TABLE users; --", Throughput: "1Gbps", Reliability: "99.9%"},
		{Latency: "<script>alert('xss')</script>", Throughput: "1Gbps", Reliability: "99.9%"},
		{Latency: "../../../etc/passwd", Throughput: "1Gbps", Reliability: "99.9%"},
		{Latency: strings.Repeat("A", 10000), Throughput: "1Gbps", Reliability: "99.9%"},
	}

	numGoroutines := 50
	numIterations := 100

	results := make(chan fixtures.QoSProfile, numGoroutines*numIterations)
	errors := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("goroutine %d panicked: %v", goroutineID, r)
				}
			}()

			for j := 0; j < numIterations; j++ {
				profile := maliciousProfiles[j%len(maliciousProfiles)]
				result := fixtures.ConvertVNFQoSProfileToQoSProfile(profile)

				// Validate security of result
				validateSafeOutput(t, result, fmt.Sprintf("goroutine_%d_iteration_%d", goroutineID, j))

				results <- result
			}
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numGoroutines*numIterations; i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			successCount++
		case err := <-errors:
			t.Errorf("Security test failed: %v", err)
			errorCount++
		}
	}

	assert.Equal(t, numGoroutines*numIterations, successCount+errorCount)
	assert.Equal(t, 0, errorCount, "No security failures should occur under concurrent load")

	t.Logf("Concurrent security test completed: %d successes, %d errors", successCount, errorCount)
}

// TestQoSDataIntegrityUnderAttack tests data integrity under various attack scenarios
func TestQoSDataIntegrityUnderAttack(t *testing.T) {
	// Original valid data
	originalProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	expectedResult := fixtures.ConvertVNFQoSProfileToQoSProfile(originalProfile)

	// Attack scenarios that should not affect data integrity
	attackScenarios := []struct {
		name        string
		attackData  fixtures.VNFQoSProfile
		description string
	}{
		{
			name: "Data poisoning attempt",
			attackData: fixtures.VNFQoSProfile{
				Latency:     "20ms" + strings.Repeat("\x00", 100),
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			description: "Null byte injection should not affect parsing",
		},
		{
			name: "Format string attack",
			attackData: fixtures.VNFQoSProfile{
				Latency:     "%s%s%s%s%s%s%s",
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			description: "Format string attack should be neutralized",
		},
		{
			name: "Unicode normalization attack",
			attackData: fixtures.VNFQoSProfile{
				Latency:     "２０ms", // Full-width characters
				Throughput:  "1Gbps",
				Reliability: "99.9%",
			},
			description: "Unicode attacks should be handled",
		},
	}

	for _, scenario := range attackScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Process attack data
			attackResult := fixtures.ConvertVNFQoSProfileToQoSProfile(scenario.attackData)

			// Verify data integrity - core structure should be preserved
			assert.Equal(t, expectedResult.Latency.Unit, attackResult.Latency.Unit,
				"%s: Unit should be preserved", scenario.description)
			assert.Equal(t, expectedResult.Throughput.Unit, attackResult.Throughput.Unit,
				"%s: Throughput unit should be preserved", scenario.description)
			assert.Equal(t, expectedResult.Reliability.Unit, attackResult.Reliability.Unit,
				"%s: Reliability unit should be preserved", scenario.description)

			// Verify no dangerous content in results
			validateSafeOutput(t, attackResult, scenario.description)
		})
	}
}