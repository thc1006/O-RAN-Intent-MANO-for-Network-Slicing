// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"
)

// TestSanitizeForLog tests the basic log sanitization function
func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Normal string",
			input:    "normal log message",
			expected: "normal log message",
		},
		{
			name:     "String with newlines",
			input:    "line1\nline2\r\nline3",
			expected: "line1\\nline2\\r\\nline3",
		},
		{
			name:     "String with tabs",
			input:    "column1\tcolumn2\tcolumn3",
			expected: "column1\\tcolumn2\\tcolumn3",
		},
		{
			name:     "String with control characters",
			input:    "test\x00\x01\x1b[31mred\x1b[0m",
			expected: "test\\x00\\x01\\e[31mred\\e[0m",
		},
		{
			name:     "String with format specifiers",
			input:    "test %s %d %v",
			expected: "test %%s %%d %%v",
		},
		{
			name:     "String with ANSI escape sequences",
			input:    "\x1b[31;1mBold Red\x1b[0m",
			expected: "\\e[31;1mBold Red\\e[0m",
		},
		{
			name:     "Long string truncation",
			input:    strings.Repeat("a", 600),
			expected: strings.Repeat("a", 509) + "...",
		},
		{
			name:     "Mixed dangerous characters",
			input:    "user\ninput\r\n\twith\x1b[0m\x00null",
			expected: "user\\ninput\\r\\n\\twith\\e[0m\\x00null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForLog(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForLog() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSanitizeErrorForLog tests error sanitization
func TestSanitizeErrorForLog(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: "<nil>",
		},
		{
			name:     "Simple error",
			err:      errors.New("simple error"),
			expected: "simple error",
		},
		{
			name:     "Error with newlines",
			err:      errors.New("error\nwith\nnewlines"),
			expected: "error\\nwith\\nnewlines",
		},
		{
			name:     "Error with control characters",
			err:      errors.New("error\x1b[31mwith\x1b[0mcolor"),
			expected: "error\\e[31mwith\\e[0mcolor",
		},
		{
			name:     "Error with format specifiers",
			err:      errors.New("error with %s %d"),
			expected: "error with %%s %%d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorForLog(tt.err)
			if result != tt.expected {
				t.Errorf("SanitizeErrorForLog() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSanitizeIPForLog tests IP address sanitization
func TestSanitizeIPForLog(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "Empty IP",
			ip:       "",
			expected: "<empty>",
		},
		{
			name:     "Valid IPv4",
			ip:       "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "Valid IPv6",
			ip:       "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "Invalid IP",
			ip:       "999.999.999.999",
			expected: "<invalid-ip:999.999.999.999>",
		},
		{
			name:     "IP with injection attempt",
			ip:       "192.168.1.1\nmalicious",
			expected: "<invalid-ip:192.168.1.1\\nmalicious>",
		},
		{
			name:     "IP with control characters",
			ip:       "192.168.1.1\x1b[0m",
			expected: "<invalid-ip:192.168.1.1\\e[0m>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIPForLog(tt.ip)
			if result != tt.expected {
				t.Errorf("SanitizeIPForLog() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestSanitizeStringForLog tests string sanitization with suspicious pattern detection
func TestSanitizeStringForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // Check if result contains this substring
	}{
		{
			name:     "Empty string",
			input:    "",
			contains: "<empty>",
		},
		{
			name:     "Normal string",
			input:    "normal string",
			contains: "normal string",
		},
		{
			name:     "String with ANSI escape",
			input:    "text\x1b[31mred\x1b[0m",
			contains: "text\\e[31mred\\e[0m",
		},
		{
			name:     "String with shell expansion",
			input:    "test ${SHELL}",
			contains: "<suspicious-input:",
		},
		{
			name:     "String with command substitution",
			input:    "test `whoami`",
			contains: "<suspicious-input:",
		},
		{
			name:     "String with exec",
			input:    "exec(malicious)",
			contains: "<suspicious-input:",
		},
		{
			name:     "String with eval",
			input:    "eval(dangerous)",
			contains: "<suspicious-input:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeStringForLog(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("SanitizeStringForLog() = %q, expected to contain %q", result, tt.contains)
			}
		})
	}
}

// TestSecureLogger tests the SecureLogger functionality
func TestSecureLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	secureLogger := NewSecureLogger(logger)

	tests := []struct {
		name         string
		action       func()
		expectedLogs string
	}{
		{
			name: "SafeLogf with normal format",
			action: func() {
				secureLogger.SafeLogf("User %s logged in", "alice")
			},
			expectedLogs: "User alice logged in",
		},
		{
			name: "SafeLogf with dangerous input",
			action: func() {
				secureLogger.SafeLogf("User %s logged in", "alice\nADMIN")
			},
			expectedLogs: "User alice\\nADMIN logged in",
		},
		{
			name: "SafeLogError with normal error",
			action: func() {
				secureLogger.SafeLogError("Operation failed", errors.New("connection timeout"))
			},
			expectedLogs: "Operation failed: connection timeout",
		},
		{
			name: "SafeLogError with malicious error",
			action: func() {
				secureLogger.SafeLogError("Operation failed", errors.New("error\nFAKE LOG ENTRY"))
			},
			expectedLogs: "Operation failed: error\\nFAKE LOG ENTRY",
		},
		{
			name: "SafeLogIP with valid IP",
			action: func() {
				secureLogger.SafeLogIP("Connecting to", "192.168.1.1")
			},
			expectedLogs: "Connecting to: 192.168.1.1",
		},
		{
			name: "SafeLogIP with invalid IP",
			action: func() {
				secureLogger.SafeLogIP("Connecting to", "invalid\nip")
			},
			expectedLogs: "Connecting to: <invalid-ip:invalid\\nip>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.action()
			output := buf.String()
			if !strings.Contains(output, tt.expectedLogs) {
				t.Errorf("Expected log to contain %q, got %q", tt.expectedLogs, output)
			}
		})
	}
}

// TestIsValidLogFormat tests format string validation
func TestIsValidLogFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected bool
	}{
		{
			name:     "Valid simple format",
			format:   "User %s logged in",
			expected: true,
		},
		{
			name:     "Valid multiple formats",
			format:   "User %s connected from %s at %d",
			expected: true,
		},
		{
			name:     "Invalid %n format specifier",
			format:   "User %s %n",
			expected: false,
		},
		{
			name:     "Invalid %x format specifier",
			format:   "Memory dump: %x",
			expected: false,
		},
		{
			name:     "Too many format specifiers",
			format:   "%s %s %s %s %s %s %s %s %s %s %s %s",
			expected: false,
		},
		{
			name:     "Escaped percent signs",
			format:   "Progress: %d%% complete",
			expected: true,
		},
		{
			name:     "Valid complex format",
			format:   "Error %d: %s (code=%v)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidLogFormat(tt.format)
			if result != tt.expected {
				t.Errorf("isValidLogFormat(%q) = %v, want %v", tt.format, result, tt.expected)
			}
		})
	}
}

// TestGlobalSecureLoggingFunctions tests global convenience functions
func TestGlobalSecureLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	// Test SafeLogf global function
	buf.Reset()
	SafeLogf(logger, "Test %s", "message")
	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("SafeLogf global function failed, got: %q", output)
	}

	// Test SafeLogError global function
	buf.Reset()
	SafeLogError(logger, "Test error", errors.New("test\nerror"))
	output = buf.String()
	if !strings.Contains(output, "Test error: test\\nerror") {
		t.Errorf("SafeLogError global function failed, got: %q", output)
	}

	// Test SafeLogWarning global function
	buf.Reset()
	SafeLogWarning(logger, "Warning\nmessage")
	output = buf.String()
	if !strings.Contains(output, "[WARNING] Warning\\nmessage") {
		t.Errorf("SafeLogWarning global function failed, got: %q", output)
	}

	// Test SafeLogIP global function
	buf.Reset()
	SafeLogIP(logger, "IP address", "192.168.1.1")
	output = buf.String()
	if !strings.Contains(output, "IP address: 192.168.1.1") {
		t.Errorf("SafeLogIP global function failed, got: %q", output)
	}
}

// TestLogInjectionPrevention tests specific log injection attack scenarios
func TestLogInjectionPrevention(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	secureLogger := NewSecureLogger(logger)

	attackScenarios := []struct {
		name        string
		input       string
		shouldBlock bool
	}{
		{
			name:        "CRLF injection",
			input:       "user\r\nFAKE LOG ENTRY",
			shouldBlock: true,
		},
		{
			name:        "ANSI escape injection",
			input:       "user\x1b[2J\x1b[H\x1b[31mFAKE ERROR",
			shouldBlock: true,
		},
		{
			name:        "Format string injection",
			input:       "user%n%n%n%n",
			shouldBlock: true,
		},
		{
			name:        "Null byte injection",
			input:       "user\x00hidden",
			shouldBlock: true,
		},
		{
			name:        "Terminal control injection",
			input:       "user\x1b[2K\rFAKE: Authorized access",
			shouldBlock: true,
		},
	}

	for _, scenario := range attackScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			buf.Reset()
			secureLogger.SafeLogf("User logged in: %s", scenario.input)
			output := buf.String()

			if scenario.shouldBlock {
				// Check that original malicious content is not present
				if strings.Contains(output, scenario.input) {
					t.Errorf("Log injection not prevented for scenario %s: %q", scenario.name, output)
				}
				// Check that content was sanitized (% escaped to %%)
				if scenario.name == "Format string injection" {
					if !strings.Contains(output, "%%") {
						t.Errorf("Expected format specifiers to be escaped in output for scenario %s: %q", scenario.name, output)
					}
				} else {
					if !strings.Contains(output, "\\") {
						t.Errorf("Expected sanitization markers in output for scenario %s: %q", scenario.name, output)
					}
				}
			}
		})
	}
}

// BenchmarkSanitizeForLog benchmarks the sanitization function
func BenchmarkSanitizeForLog(b *testing.B) {
	input := "Normal log message with\nsome\tcontrol\x1b[31mcharacters\x1b[0m"
	for i := 0; i < b.N; i++ {
		SanitizeForLog(input)
	}
}

// BenchmarkSecureLogger benchmarks the secure logger
func BenchmarkSecureLogger(b *testing.B) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	secureLogger := NewSecureLogger(logger)

	for i := 0; i < b.N; i++ {
		secureLogger.SafeLogf("Test message %d with %s", i, "parameters")
	}
}

// TestMaxLogLength tests log length limiting
func TestMaxLogLength(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	secureLogger := NewSecureLogger(logger)

	// Test with a very long message
	longMessage := strings.Repeat("a", 2000)
	secureLogger.SafeLogf("Long message: %s", longMessage)

	output := buf.String()
	if len(output) > secureLogger.maxLogLength+100 { // Allow some overhead for formatting
		t.Errorf("Log message not properly truncated, length: %d", len(output))
	}

	if !strings.Contains(output, "...") {
		t.Errorf("Expected truncation indicator (...) in output")
	}
}

// TestSanitizeLogArgument tests argument sanitization
func TestSanitizeLogArgument(t *testing.T) {
	tests := []struct {
		name     string
		arg      interface{}
		expected string
	}{
		{
			name:     "String argument",
			arg:      "test\nstring",
			expected: "test\\nstring",
		},
		{
			name:     "Error argument",
			arg:      errors.New("test\nerror"),
			expected: "test\\nerror",
		},
		{
			name:     "Integer argument",
			arg:      42,
			expected: "42",
		},
		{
			name:     "Stringer argument",
			arg:      fmt.Errorf("error\nwith\nnewlines"),
			expected: "error\\nwith\\nnewlines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLogArgument(tt.arg)
			if result != tt.expected {
				t.Errorf("sanitizeLogArgument() = %q, want %q", result, tt.expected)
			}
		})
	}
}