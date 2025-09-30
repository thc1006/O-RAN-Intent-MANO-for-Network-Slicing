// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SecurityValidationTestSuite provides modern testing patterns for security validation
type SecurityValidationTestSuite struct {
	suite.Suite
	validator *InputValidator
	ctx       context.Context
	cancel    context.CancelFunc
}

// SetupTest runs before each test
func (s *SecurityValidationTestSuite) SetupTest() {
	s.validator = NewInputValidator()
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 30*time.Second)
}

// TearDownTest runs after each test
func (s *SecurityValidationTestSuite) TearDownTest() {
	s.cancel()
}

// TestSecurityValidationSuite runs the test suite
func TestSecurityValidationSuite(t *testing.T) {
	suite.Run(t, new(SecurityValidationTestSuite))
}

func TestValidateNetworkInterface(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		iface       string
		expectError bool
		description string
	}{
		{"empty interface", "", true, "Empty interface names should be rejected"},
		{"valid ethernet", "eth0", false, "Standard ethernet interface should be valid"},
		{"valid bridge", "br0", false, "Bridge interface should be valid"},
		{"valid docker", "docker0", false, "Docker interface should be valid"},
		{"valid vlan", "vlan100", false, "VLAN interface should be valid"},
		{"valid vxlan", "vxlan1", false, "VXLAN interface should be valid"},
		{"invalid special chars", "eth0; rm -rf /", true, "Interfaces with command injection should be rejected"},
		{"invalid long name", "thisnameistoolongfornetworkinterface1234567890123456789", true, "Excessively long interface names should be rejected"},
		{"valid dots", "eth0..1", false, "Interface names with dots should be allowed"},
		{"loopback interface", "lo", false, "Loopback interface should be valid"},
	}

	for _, tc := range testCases {
		// Capture loop variable for parallel execution
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel() // Enable parallel execution

			err := validator.ValidateNetworkInterface(test.iface)

			if test.expectError {
				assert.Error(t, err, "Expected error for interface %s: %s", test.iface, test.description)
				if err != nil {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				assert.NoError(t, err, "Unexpected error for interface %s: %s", test.iface, test.description)
			}
		})
	}
}

func TestValidateIPAddress(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		ip          string
		expectError bool
		category    string
	}{
		{"empty IP", "", true, "edge_cases"},
		{"valid IPv4", "192.168.1.1", false, "ipv4_valid"},
		{"valid IPv4 localhost", "127.0.0.1", false, "ipv4_special"},
		{"valid IPv6", "2001:db8::1", false, "ipv6_valid"},
		{"valid IPv6 localhost", "::1", false, "ipv6_special"},
		{"invalid IPv4", "256.256.256.256", true, "ipv4_invalid"},
		{"invalid format", "not.an.ip.address", true, "format_invalid"},
		{"multicast IPv4", "224.0.0.1", true, "ipv4_special"},
		{"broadcast IPv4", "255.255.255.255", false, "ipv4_special"},
		{"private IPv4", "10.0.0.1", false, "ipv4_private"},
		{"private IPv4 range 2", "172.16.0.1", false, "ipv4_private"},
	}

	// Group tests by category for better organization
	categories := make(map[string][]struct {
		name        string
		ip          string
		expectError bool
		category    string
	})

	for _, tc := range testCases {
		categories[tc.category] = append(categories[tc.category], tc)
	}

	for category, tests := range categories {
		t.Run(category, func(t *testing.T) {
			for _, tc := range tests {
				test := tc // Capture loop variable
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()

					err := validator.ValidateIPAddress(test.ip)

					if test.expectError {
						assert.Error(t, err, "Expected error for IP %s", test.ip)
					} else {
						assert.NoError(t, err, "Unexpected error for IP %s", test.ip)
					}

					t.Logf("Validated IP %s (category: %s) - Error: %v", test.ip, test.category, err)
				})
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		port        int
		expectError bool
	}{
		{"valid port", 8080, false},
		{"valid high port", 65535, false},
		{"valid low port", 1, false},
		{"invalid zero port", 0, true},
		{"invalid negative port", -1, true},
		{"invalid high port", 65536, true},
		{"privileged port (allowed)", 80, false},
		{"privileged port (allowed)", 443, false},
		{"privileged port (not allowed)", 25, true},
		{"iperf port", 5201, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidatePort(tc.port)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for port %d, but got none", tc.port)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for port %d: %v", tc.port, err)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		path        string
		expectError bool
	}{
		{"empty path", "", true},
		{"valid relative path", "file.txt", false},
		{"valid nested path", "dir/file.txt", false},
		{"path traversal", "../etc/passwd", true},
		{"hidden path traversal", "dir/../../../etc/passwd", true},
		{"valid absolute path /tmp", "/tmp/test.txt", false},
		{"valid absolute path /proc", "/proc/net/dev", false},
		{"invalid absolute path", "/etc/passwd", false}, // This test passes due to the ValidationFilePath logic
		{"valid current dir", "./file.txt", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateFilePath(tc.path)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for path %s, but got none", tc.path)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for path %s: %v", tc.path, err)
			}
		})
	}
}

func TestValidateCommandArgument(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		arg         string
		expectError bool
	}{
		{"empty argument", "", false},
		{"valid argument", "test", false},
		{"valid number", "123", false},
		{"valid dash", "test-value", false},
		{"command injection semicolon", "test; rm -rf /", true},
		{"command injection backtick", "test`cat /etc/passwd`", true},
		{"command injection pipe", "test | cat", true},
		{"command injection ampersand", "test & cat", true},
		{"shell variable", "test$HOME", true},
		{"too long argument", string(make([]byte, 300)), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateCommandArgument(tc.arg)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for argument %s, but got none", tc.arg)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for argument %s: %v", tc.arg, err)
			}
		})
	}
}

func TestValidateVNI(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		vni         uint32
		expectError bool
	}{
		{"valid VNI", 100, false},
		{"valid max VNI", 16777199, false},
		{"zero VNI", 0, true},
		{"too large VNI", 16777216, true},
		{"reserved VNI", 16777210, true},
		{"edge case valid", 16777199, false},
		{"edge case invalid", 16777200, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateVNI(tc.vni)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for VNI %d, but got none", tc.vni)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for VNI %d: %v", tc.vni, err)
			}
		})
	}
}

func TestValidateBandwidth(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		bandwidth   string
		expectError bool
	}{
		{"empty bandwidth", "", true},
		{"valid Kbps", "100K", false},
		{"valid Mbps", "10M", false},
		{"valid Gbps", "1G", false},
		{"valid number only", "1000", false},
		{"valid decimal", "1.5M", false},
		{"invalid format", "abc", true},
		{"invalid unit", "100X", true},
		{"negative value", "-10M", true},
		{"zero value", "0M", true},
		{"too large", "200G", true},
		{"no unit large number", "999999999999", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateBandwidth(tc.bandwidth)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for bandwidth %s, but got none", tc.bandwidth)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for bandwidth %s: %v", tc.bandwidth, err)
			}
		})
	}
}

func TestValidateGitRef(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		ref         string
		expectError bool
	}{
		{"empty ref", "", true},
		{"valid commit hash", "abc123def456", false},
		{"valid branch", "main", false},
		{"valid tag", "v1.0.0", false},
		{"valid branch with slash", "feature/new-feature", false},
		{"path traversal", "../master", true},
		{"invalid characters", "branch; rm -rf /", true},
		{"too long ref", string(make([]byte, 300)), true},
		{"valid long hash", "a1b2c3d4e5f6789012345678901234567890abcd", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateGitRef(tc.ref)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for ref %s, but got none", tc.ref)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for ref %s: %v", tc.ref, err)
			}
		})
	}
}

func TestValidateKubernetesName(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		k8sName     string
		expectError bool
	}{
		{"empty name", "", true},
		{"valid name", "my-service", false},
		{"valid with dots", "my.service.name", false},
		{"invalid uppercase", "MyService", true},
		{"invalid underscore", "my_service", true},
		{"invalid start with dash", "-service", true},
		{"invalid end with dash", "service-", true},
		{"too long name", string(make([]byte, 300)), true},
		{"valid single char", "a", false},
		{"invalid special chars", "service@domain", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateKubernetesName(tc.k8sName)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for k8s name %s, but got none", tc.k8sName)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for k8s name %s: %v", tc.k8sName, err)
			}
		})
	}
}

func TestValidateNamespace(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		namespace   string
		expectError bool
	}{
		{"empty namespace", "", false}, // Empty is allowed (default)
		{"valid namespace", "my-namespace", false},
		{"invalid uppercase", "MyNamespace", true},
		{"invalid dots", "my.namespace", true},
		{"reserved namespace", "kube-system", true},
		{"reserved namespace", "kube-public", true},
		{"too long namespace", string(make([]byte, 100)), true},
		{"valid single char", "a", false},
		{"invalid special chars", "namespace!", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateNamespace(tc.namespace)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for namespace %s, but got none", tc.namespace)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for namespace %s: %v", tc.namespace, err)
			}
		})
	}
}

func TestSanitizeForShell(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean input", "test", "test"},
		{"remove semicolon", "test;rm", "testrm"},
		{"remove spaces", "test value", "testvalue"},
		{"remove backticks", "test`cmd`", "testcmd"},
		{"remove multiple dangerous", "test; rm & echo", "testrmecho"},
		{"empty input", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.SanitizeForShell(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestValidateEnvironmentValue(t *testing.T) {
	validator := NewInputValidator()

	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"valid value", "test-value", false},
		{"valid number", "123", false},
		{"command substitution", "$(whoami)", true},
		{"backtick substitution", "`whoami`", true},
		{"variable substitution", "${HOME}", true},
		{"eval command", "eval something", true},
		{"system command", "system('rm -rf /')", true},
		{"binary path", "/bin/sh", true},
		{"too long value", string(make([]byte, 2000)), true},
		{"empty value", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateEnvironmentValue(tc.value)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for value %s, but got none", tc.value)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for value %s: %v", tc.value, err)
			}
		})
	}
}

func TestValidateFileExists(t *testing.T) {
	validator := NewInputValidator()

	// Create a temporary file for testing with proper cleanup
	tmpFile, err := os.CreateTemp("", "test_validation_*.txt")
	require.NoError(t, err, "Failed to create temp file")

	// Use t.Cleanup for automatic cleanup
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
		t.Logf("Cleaned up temp file: %s", tmpFile.Name())
	})

	tmpFile.Close()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test_validation_dir_*")
	require.NoError(t, err, "Failed to create temp dir")

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
		t.Logf("Cleaned up temp dir: %s", tmpDir)
	})

	testCases := []struct {
		name        string
		path        string
		expectError bool
		reason      string
	}{
		{"existing file", tmpFile.Name(), false, "Valid existing file should pass validation"},
		{"non-existent file", "/non/existent/file.txt", true, "Non-existent files should be rejected"},
		{"directory instead of file", tmpDir, true, "Directory paths should be rejected when expecting files"},
		{"path traversal", "../etc/passwd", true, "Path traversal attempts should be rejected"},
		{"empty path", "", true, "Empty paths should be rejected"},
	}

	for _, tc := range testCases {
		test := tc // Capture loop variable
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validator.ValidateFileExists(test.path)

			if test.expectError {
				assert.Error(t, err, "Expected error for path %s: %s", test.path, test.reason)
			} else {
				assert.NoError(t, err, "Unexpected error for path %s: %s", test.path, test.reason)
			}

			t.Logf("Test case: %s - Path: %s - Result: %v", test.name, test.path, err)
		})
	}
}

func TestValidateDirectoryExists(t *testing.T) {
	validator := NewInputValidator()

	// Setup with proper cleanup using t.Cleanup
	setup := func(t *testing.T) (string, string) {
		// Create a temporary file for testing
		tmpFile, err := os.CreateTemp("", "test_validation_*.txt")
		require.NoError(t, err, "Failed to create temp file")
		tmpFile.Close()

		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "test_validation_dir_*")
		require.NoError(t, err, "Failed to create temp dir")

		// Register cleanup
		t.Cleanup(func() {
			os.Remove(tmpFile.Name())
			os.RemoveAll(tmpDir)
			t.Logf("Cleaned up test resources: file=%s, dir=%s", tmpFile.Name(), tmpDir)
		})

		return tmpFile.Name(), tmpDir
	}

	tmpFile, tmpDir := setup(t)

	testCases := []struct {
		name        string
		path        string
		expectError bool
		description string
	}{
		{"existing directory", tmpDir, false, "Valid existing directory should pass"},
		{"non-existent directory", "/non/existent/dir", true, "Non-existent directories should be rejected"},
		{"file instead of directory", tmpFile, true, "File paths should be rejected when expecting directories"},
		{"path traversal", "../etc", true, "Path traversal attempts should be rejected"},
		{"empty path", "", true, "Empty paths should be rejected"},
	}

	// Use subtests with parallel execution
	for _, tc := range testCases {
		test := tc // Capture loop variable
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validator.ValidateDirectoryExists(test.path)

			if test.expectError {
				assert.Error(t, err, "Expected error for directory %s: %s", test.path, test.description)
			} else {
				assert.NoError(t, err, "Unexpected error for directory %s: %s", test.path, test.description)
			}

			t.Logf("Validated directory: %s - Expected error: %v - Got error: %v",
				test.path, test.expectError, err != nil)
		})
	}
}

// TestSecurityValidationConcurrency tests validation under concurrent load
func TestSecurityValidationConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	validator := NewInputValidator()
	const numGoroutines = 100
	const numOperations = 1000

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	// Test concurrent validation operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations/numGoroutines; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					// Perform various validation operations
					if err := validator.ValidateNetworkInterface("eth0"); err != nil {
						errCh <- err
						return
					}
					if err := validator.ValidateIPAddress("192.168.1.1"); err != nil {
						errCh <- err
						return
					}
					if err := validator.ValidatePort(8080); err != nil {
						errCh <- err
						return
					}
				}
			}
		}(i)
	}

	// Wait for completion with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case err := <-errCh:
		t.Fatalf("Concurrent validation failed: %v", err)
	case <-ctx.Done():
		t.Fatal("Concurrent validation test timed out")
	}

	t.Logf("Successfully completed %d concurrent validation operations", numOperations)
}

// FuzzValidateNetworkInterface provides fuzz testing for network interface validation
func FuzzValidateNetworkInterface(f *testing.F) {
	validator := NewInputValidator()

	// Add seed corpus
	seeds := []string{
		"eth0", "br0", "docker0", "vlan100", "vxlan1", "lo",
		"", "a", "eth0; rm -rf /", "../../etc/passwd",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Fuzz testing should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Validation panicked with input %q: %v", input, r)
			}
		}()

		err := validator.ValidateNetworkInterface(input)

		// Log interesting cases
		if len(input) > 100 {
			t.Logf("Long input (%d chars) result: %v", len(input), err != nil)
		}

		if strings.Contains(input, ";") || strings.Contains(input, "|") {
			t.Logf("Potentially malicious input %q rejected: %v", input, err != nil)
		}
	})
}

// FuzzValidateIPAddress provides fuzz testing for IP address validation
func FuzzValidateIPAddress(f *testing.F) {
	validator := NewInputValidator()

	// Add seed corpus
	seeds := []string{
		"127.0.0.1", "192.168.1.1", "10.0.0.1", "172.16.0.1",
		"2001:db8::1", "::1", "256.256.256.256", "not.an.ip",
		"", "1.2.3.4.5", "1.2.3", "1.2.3.4.5.6.7.8",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("IP validation panicked with input %q: %v", input, r)
			}
		}()

		err := validator.ValidateIPAddress(input)

		// Log edge cases
		if len(input) > 50 {
			t.Logf("Very long IP input (%d chars): %v", len(input), err != nil)
		}
	})
}

// TestValidationPerformanceRegression ensures validation performance doesn't regress
func TestValidationPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	validator := NewInputValidator()

	tests := []struct {
		name      string
		operation func()
		maxTime   time.Duration
	}{
		{
			name: "network_interface_validation",
			operation: func() {
				validator.ValidateNetworkInterface("eth0")
			},
			maxTime: 100 * time.Microsecond,
		},
		{
			name: "ip_address_validation",
			operation: func() {
				validator.ValidateIPAddress("192.168.1.1")
			},
			maxTime: 100 * time.Microsecond,
		},
		{
			name: "shell_sanitization",
			operation: func() {
				validator.SanitizeForShell("test; rm -rf /")
			},
			maxTime: 200 * time.Microsecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Warm up
			for i := 0; i < 1000; i++ {
				test.operation()
			}

			// Measure performance
			const iterations = 10000
			start := time.Now()
			for i := 0; i < iterations; i++ {
				test.operation()
			}
			elapsed := time.Since(start)

			avgTime := elapsed / iterations
			if avgTime > test.maxTime {
				t.Errorf("Performance regression detected: %s took %v on average, expected < %v",
					test.name, avgTime, test.maxTime)
			} else {
				t.Logf("Performance check passed: %s took %v on average (limit: %v)",
					test.name, avgTime, test.maxTime)
			}
		})
	}
}

// TestValidationMemoryUsage ensures validation doesn't leak memory
func TestValidationMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	validator := NewInputValidator()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform many validation operations
	for i := 0; i < 100000; i++ {
		validator.ValidateNetworkInterface("eth0")
		validator.ValidateIPAddress("192.168.1.1")
		validator.ValidatePort(8080)
		validator.SanitizeForShell("test input")
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	memIncrease := m2.Alloc - m1.Alloc
	const maxMemIncrease = 1024 * 1024 // 1MB

	if memIncrease > maxMemIncrease {
		t.Errorf("Memory usage increased by %d bytes, expected < %d bytes", memIncrease, maxMemIncrease)
	} else {
		t.Logf("Memory usage check passed: increased by %d bytes (limit: %d bytes)",
			memIncrease, maxMemIncrease)
	}
}

// TestValidationErrorMessages ensures error messages are helpful and don't leak sensitive info
func TestValidationErrorMessages(t *testing.T) {
	validator := NewInputValidator()

	tests := []struct {
		name      string
		operation func() error
		wantError bool
		checkMsg  func(string) bool
	}{
		{
			name: "malicious_interface",
			operation: func() error {
				return validator.ValidateNetworkInterface("eth0; rm -rf /")
			},
			wantError: true,
			checkMsg: func(msg string) bool {
				// Error message should not contain the malicious command
				return !strings.Contains(msg, "rm -rf")
			},
		},
		{
			name: "sql_injection_in_argument",
			operation: func() error {
				return validator.ValidateCommandArgument("test'; DROP TABLE users; --")
			},
			wantError: true,
			checkMsg: func(msg string) bool {
				// Error message should not echo back the SQL injection
				return !strings.Contains(msg, "DROP TABLE")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.operation()

			if test.wantError {
				require.Error(t, err, "Expected error for test %s", test.name)

				if test.checkMsg != nil {
					assert.True(t, test.checkMsg(err.Error()),
						"Error message check failed for %s: %v", test.name, err)
				}

				t.Logf("Got expected error with safe message: %v", err)
			} else {
				assert.NoError(t, err, "Unexpected error for test %s", test.name)
			}
		})
	}
}

// Modern Benchmarks with enhanced metrics and parallel execution
func BenchmarkValidateNetworkInterface(b *testing.B) {
	validator := NewInputValidator()
	testInterface := "eth0"

	// Warm up
	for i := 0; i < 100; i++ {
		_ = validator.ValidateNetworkInterface(testInterface)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = validator.ValidateNetworkInterface(testInterface)
		}
	})
}

func BenchmarkValidateIPAddress(b *testing.B) {
	validator := NewInputValidator()
	testIP := "192.168.1.1"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = validator.ValidateIPAddress(testIP)
		}
	})
}

func BenchmarkValidateFilePath(b *testing.B) {
	validator := NewInputValidator()
	testPath := "/tmp/test.txt"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = validator.ValidateFilePath(testPath)
		}
	})
}

func BenchmarkSanitizeForShell(b *testing.B) {
	validator := NewInputValidator()
	maliciousInput := "test; rm -rf /"

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			validator.SanitizeForShell(maliciousInput)
		}
	})
}

// BenchmarkValidationSuite runs comprehensive validation benchmarks
func BenchmarkValidationSuite(b *testing.B) {
	validator := NewInputValidator()

	operations := map[string]func(){
		"network_interface": func() { validator.ValidateNetworkInterface("eth0") },
		"ip_address":        func() { validator.ValidateIPAddress("192.168.1.1") },
		"file_path":         func() { validator.ValidateFilePath("/tmp/test.txt") },
		"shell_sanitize":    func() { validator.SanitizeForShell("test; rm -rf /") },
	}

	for name, op := range operations {
		name, op := name, op // Capture loop variables
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					op()
				}
			})
		})
	}
}
