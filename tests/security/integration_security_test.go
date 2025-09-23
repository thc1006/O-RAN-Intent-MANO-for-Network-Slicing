// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// IntegrationSecurityTestSuite provides end-to-end security testing
type IntegrationSecurityTestSuite struct {
	executor     *security.SecureSubprocessExecutor
	validator    *security.InputValidator
	httpServer   *SecureHTTPServer
	logger       *TestSecureLogger
	errorHandler *SecureErrorHandler
	tempDir      string
	serverAddr   string
	cleanup      []func()
}

func NewIntegrationSecurityTestSuite(t *testing.T) *IntegrationSecurityTestSuite {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "security_integration_test_*")
	require.NoError(t, err)

	logger := NewTestSecureLogger()
	errorHandler := NewSecureErrorHandler(logger)

	suite := &IntegrationSecurityTestSuite{
		executor:     security.NewSecureSubprocessExecutor(),
		validator:    security.NewInputValidator(),
		httpServer:   NewSecureHTTPServer(nil),
		logger:       logger,
		errorHandler: errorHandler,
		tempDir:      tempDir,
		cleanup:      []func(){},
	}

	// Setup cleanup
	suite.cleanup = append(suite.cleanup, func() {
		os.RemoveAll(tempDir)
	})

	// Start HTTP server for integration tests
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	suite.serverAddr = listener.Addr().String()

	go func() {
		suite.httpServer.server.Serve(listener)
	}()

	suite.cleanup = append(suite.cleanup, func() {
		listener.Close()
	})

	return suite
}

func (suite *IntegrationSecurityTestSuite) Cleanup() {
	for _, cleanup := range suite.cleanup {
		cleanup()
	}
}

// TestEndToEndSecurityWorkflow tests complete security workflows
func TestEndToEndSecurityWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	suite := NewIntegrationSecurityTestSuite(t)
	defer suite.Cleanup()

	t.Run("secure_command_execution_workflow", func(t *testing.T) {
		suite.testSecureCommandExecutionWorkflow(t)
	})

	t.Run("http_request_validation_workflow", func(t *testing.T) {
		suite.testHTTPRequestValidationWorkflow(t)
	})

	t.Run("file_operation_security_workflow", func(t *testing.T) {
		suite.testFileOperationSecurityWorkflow(t)
	})

	t.Run("error_handling_integration_workflow", func(t *testing.T) {
		suite.testErrorHandlingIntegrationWorkflow(t)
	})

	t.Run("concurrent_security_operations", func(t *testing.T) {
		suite.testConcurrentSecurityOperations(t)
	})
}

func (suite *IntegrationSecurityTestSuite) testSecureCommandExecutionWorkflow(t *testing.T) {
	ctx := context.Background()

	// Test workflow: validate input -> execute command -> handle errors -> log securely
	testCases := []struct {
		name        string
		command     string
		args        []string
		shouldPass  bool
		description string
	}{
		{
			name:        "legitimate_ping_command",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1"},
			shouldPass:  true,
			description: "Legitimate ping should pass through entire workflow",
		},
		{
			name:        "command_injection_attempt",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1; cat /etc/passwd"},
			shouldPass:  false,
			description: "Command injection should be blocked at validation stage",
		},
		{
			name:        "privilege_escalation_attempt",
			command:     "sudo",
			args:        []string{"cat", "/etc/passwd"},
			shouldPass:  false,
			description: "Privilege escalation should be blocked at allowlist stage",
		},
		{
			name:        "path_traversal_in_args",
			command:     "cat",
			args:        []string{"../../../etc/passwd"},
			shouldPass:  false,
			description: "Path traversal should be blocked at validation stage",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite.logger.Clear()

			// Step 1: Validate all arguments
			var validationErrors []error
			for _, arg := range tc.args {
				if err := suite.validator.ValidateCommandArgument(arg); err != nil {
					validationErrors = append(validationErrors, err)
				}
			}

			// Step 2: Execute command if validation passes
			var execError error
			if len(validationErrors) == 0 {
				_, execError = suite.executor.SecureExecute(ctx, tc.command, tc.args...)
			}

			// Step 3: Handle any errors
			var handledError error
			if len(validationErrors) > 0 {
				handledError = suite.errorHandler.HandleError(validationErrors[0], map[string]interface{}{
					"command": tc.command,
					"args":    tc.args,
					"stage":   "validation",
				})
			} else if execError != nil {
				handledError = suite.errorHandler.HandleError(execError, map[string]interface{}{
					"command": tc.command,
					"args":    tc.args,
					"stage":   "execution",
				})
			}

			// Step 4: Verify workflow behavior
			if tc.shouldPass {
				// For legitimate commands, validation should pass
				assert.Empty(t, validationErrors, "Legitimate command should pass validation")

				// Execution might fail due to system constraints, but not due to security
				if execError != nil {
					assert.False(t, strings.Contains(execError.Error(), "command not in allowlist"),
						"Legitimate command should not be blocked by allowlist")
					assert.False(t, strings.Contains(execError.Error(), "argument validation failed"),
						"Legitimate command should not fail argument validation")
				}
			} else {
				// For malicious commands, should be blocked somewhere in the workflow
				assert.True(t, len(validationErrors) > 0 || execError != nil,
					"Malicious command should be blocked somewhere in workflow")

				// Error should be handled securely
				if handledError != nil {
					// Ensure sensitive information is not leaked
					errorMsg := handledError.Error()
					assert.NotContains(t, errorMsg, "/etc/passwd")
					assert.NotContains(t, errorMsg, "sudo")
					assert.NotContains(t, errorMsg, "cat")
				}
			}

			// Step 5: Verify secure logging
			errorEntries := suite.logger.GetEntriesByLevel(ERROR)
			for _, entry := range errorEntries {
				// Ensure no sensitive data in logs
				logContent := fmt.Sprintf("%v %v", entry.Message, entry.Fields)
				assert.NotContains(t, logContent, "/etc/passwd")
				assert.NotContains(t, logContent, "rm -rf")
			}
		})
	}
}

func (suite *IntegrationSecurityTestSuite) testHTTPRequestValidationWorkflow(t *testing.T) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	baseURL := fmt.Sprintf("http://%s", suite.serverAddr)

	testCases := []struct {
		name           string
		method         string
		path           string
		body           string
		headers        map[string]string
		expectedStatus int
		description    string
	}{
		{
			name:           "legitimate_health_check",
			method:         "GET",
			path:           "/health",
			body:           "",
			expectedStatus: 200,
			description:    "Legitimate health check should work",
		},
		{
			name:           "path_traversal_attack",
			method:         "GET",
			path:           "/../../../etc/passwd",
			expectedStatus: 404, // Should be treated as not found
			description:    "Path traversal should be handled safely",
		},
		{
			name:   "oversized_request_attack",
			method: "PUT",
			path:   "/config",
			body:   strings.Repeat("A", 10*1024*1024), // 10MB
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectedStatus: 413, // Request Entity Too Large
			description:    "Oversized request should be rejected",
		},
		{
			name:   "malicious_json_payload",
			method: "PUT",
			path:   "/config",
			body:   `{"config": "; rm -rf /", "password": "secret123"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectedStatus: 400, // Bad Request
			description:    "Malicious JSON should be rejected",
		},
		{
			name:   "header_injection_attack",
			method: "GET",
			path:   "/health",
			headers: map[string]string{
				"X-Test": "value\r\nX-Injected: malicious",
			},
			expectedStatus: 200, // Should handle gracefully
			description:    "Header injection should be handled safely",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite.logger.Clear()

			// Create HTTP request
			req, err := http.NewRequest(tc.method, baseURL+tc.path, strings.NewReader(tc.body))
			require.NoError(t, err)

			// Set headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			// Execute request
			resp, err := client.Do(req)
			if err != nil {
				// Network errors are acceptable for security tests
				t.Logf("Network error (expected for some security tests): %v", err)
				return
			}
			defer resp.Body.Close()

			// Verify response status
			assert.Equal(t, tc.expectedStatus, resp.StatusCode,
				"HTTP status should match expected for %s", tc.description)

			// Verify security headers are present
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
			assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))

			// Verify no sensitive information in response headers
			for name, values := range resp.Header {
				for _, value := range values {
					assert.NotContains(t, value, "password")
					assert.NotContains(t, value, "secret")
					assert.NotContains(t, value, "/etc/passwd")
				}
				assert.NotContains(t, name, "Injected")
				assert.NotContains(t, name, "Malicious")
			}
		})
	}
}

func (suite *IntegrationSecurityTestSuite) testFileOperationSecurityWorkflow(t *testing.T) {
	// Test secure file operations workflow
	testCases := []struct {
		name         string
		operation    string
		filePath     string
		shouldPass   bool
		description  string
	}{
		{
			name:        "legitimate_temp_file",
			operation:   "create",
			filePath:    filepath.Join(suite.tempDir, "test.txt"),
			shouldPass:  true,
			description: "Creating file in temp directory should be allowed",
		},
		{
			name:        "path_traversal_file_access",
			operation:   "read",
			filePath:    filepath.Join(suite.tempDir, "../../../etc/passwd"),
			shouldPass:  false,
			description: "Path traversal should be blocked",
		},
		{
			name:        "system_file_access",
			operation:   "read",
			filePath:    "/etc/passwd",
			shouldPass:  false,
			description: "Direct system file access should be blocked",
		},
		{
			name:        "hidden_file_access",
			operation:   "read",
			filePath:    "/home/user/.ssh/id_rsa",
			shouldPass:  false,
			description: "Hidden system file access should be blocked",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite.logger.Clear()

			// Step 1: Validate file path
			validationErr := suite.validator.ValidateFilePath(tc.filePath)

			// Step 2: Perform operation if validation passes
			var operationErr error
			if validationErr == nil {
				switch tc.operation {
				case "create":
					file, err := os.Create(tc.filePath)
					if err != nil {
						operationErr = err
					} else {
						file.Close()
					}
				case "read":
					_, err := os.ReadFile(tc.filePath)
					operationErr = err
				}
			}

			// Step 3: Handle errors securely
			var handledError error
			if validationErr != nil {
				handledError = suite.errorHandler.HandleError(validationErr, map[string]interface{}{
					"operation": tc.operation,
					"path":      tc.filePath,
					"stage":     "validation",
				})
			} else if operationErr != nil {
				handledError = suite.errorHandler.HandleError(operationErr, map[string]interface{}{
					"operation": tc.operation,
					"path":      tc.filePath,
					"stage":     "execution",
				})
			}

			// Step 4: Verify workflow behavior
			if tc.shouldPass {
				assert.NoError(t, validationErr, "Legitimate file operation should pass validation")
			} else {
				assert.True(t, validationErr != nil || operationErr != nil,
					"Malicious file operation should be blocked")

				// Ensure error doesn't leak sensitive path information
				if handledError != nil {
					errorMsg := handledError.Error()
					assert.NotContains(t, errorMsg, "/etc/passwd")
					assert.NotContains(t, errorMsg, ".ssh/id_rsa")
					assert.NotContains(t, errorMsg, "/home/user")
				}
			}

			// Step 5: Verify secure logging
			errorEntries := suite.logger.GetEntriesByLevel(ERROR)
			for _, entry := range errorEntries {
				logContent := fmt.Sprintf("%v %v", entry.Message, entry.Fields)
				// Sensitive paths should be sanitized in logs
				assert.NotContains(t, logContent, "/etc/passwd")
				assert.NotContains(t, logContent, ".ssh/id_rsa")
			}
		})
	}
}

func (suite *IntegrationSecurityTestSuite) testErrorHandlingIntegrationWorkflow(t *testing.T) {
	ctx := context.Background()

	// Test various error scenarios and ensure they're handled securely end-to-end
	errorScenarios := []struct {
		name        string
		trigger     func() error
		description string
	}{
		{
			name: "command_not_found_error",
			trigger: func() error {
				_, err := suite.executor.SecureExecute(ctx, "nonexistent_command", "arg1")
				return err
			},
			description: "Command not found should be handled securely",
		},
		{
			name: "validation_error",
			trigger: func() error {
				return suite.validator.ValidateCommandArgument("malicious; rm -rf /")
			},
			description: "Validation error should be handled securely",
		},
		{
			name: "file_permission_error",
			trigger: func() error {
				return suite.validator.ValidateFilePath("/etc/shadow")
			},
			description: "File permission error should be handled securely",
		},
		{
			name: "network_validation_error",
			trigger: func() error {
				return suite.validator.ValidateIPAddress("malicious.ip; cat /etc/passwd")
			},
			description: "Network validation error should be handled securely",
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			suite.logger.Clear()

			// Trigger error
			originalErr := scenario.trigger()

			if originalErr != nil {
				// Handle error through secure error handler
				context := map[string]interface{}{
					"scenario":  scenario.name,
					"timestamp": time.Now(),
					"user":      "test_user",
				}

				handledErr := suite.errorHandler.HandleError(originalErr, context)

				// Verify error is sanitized
				assert.Error(t, handledErr)
				assert.NotEqual(t, originalErr.Error(), handledErr.Error(),
					"Error should be sanitized")

				// Verify no sensitive information leaked
				errorMsg := handledErr.Error()
				assert.NotContains(t, errorMsg, "rm -rf")
				assert.NotContains(t, errorMsg, "/etc/passwd")
				assert.NotContains(t, errorMsg, "/etc/shadow")
				assert.NotContains(t, errorMsg, "cat")

				// Verify secure logging
				errorEntries := suite.logger.GetEntriesByLevel(ERROR)
				assert.True(t, len(errorEntries) > 0, "Error should be logged")

				for _, entry := range errorEntries {
					assert.False(t, entry.Sensitive, "Error log should not be marked as sensitive")

					logContent := fmt.Sprintf("%v %v", entry.Message, entry.Fields)
					assert.NotContains(t, logContent, "rm -rf")
					assert.NotContains(t, logContent, "/etc/passwd")
				}
			}
		})
	}
}

func (suite *IntegrationSecurityTestSuite) testConcurrentSecurityOperations(t *testing.T) {
	// Test that security controls work correctly under concurrent load
	const numWorkers = 50
	const opsPerWorker = 20

	var wg sync.WaitGroup
	results := make(chan SecurityOperationResult, numWorkers*opsPerWorker)

	// Launch concurrent workers performing various security operations
	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for op := 0; op < opsPerWorker; op++ {
				result := SecurityOperationResult{
					WorkerID:    workerID,
					OperationID: op,
					Timestamp:   time.Now(),
				}

				// Perform different types of operations
				switch op % 4 {
				case 0:
					// Command validation
					err := suite.validator.ValidateCommandArgument(fmt.Sprintf("test%d", workerID))
					result.Operation = "command_validation"
					result.Success = err == nil
					result.Error = err

				case 1:
					// File path validation
					path := filepath.Join(suite.tempDir, fmt.Sprintf("worker%d_file%d.txt", workerID, op))
					err := suite.validator.ValidateFilePath(path)
					result.Operation = "file_validation"
					result.Success = err == nil
					result.Error = err

				case 2:
					// Network validation
					ip := fmt.Sprintf("192.168.1.%d", (workerID%254)+1)
					err := suite.validator.ValidateIPAddress(ip)
					result.Operation = "network_validation"
					result.Success = err == nil
					result.Error = err

				case 3:
					// Command execution
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
					_, err := suite.executor.SecureExecute(ctx, "ping", "-c", "1", "127.0.0.1")
					cancel()
					result.Operation = "command_execution"
					result.Success = err == nil
					result.Error = err
				}

				results <- result
			}
		}(worker)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Analyze results
	operationStats := make(map[string]OperationStats)
	totalOperations := 0

	for result := range results {
		totalOperations++
		stats := operationStats[result.Operation]
		stats.Total++

		if result.Success {
			stats.Successful++
		} else {
			stats.Failed++
			if result.Error != nil {
				stats.Errors = append(stats.Errors, result.Error.Error())
			}
		}

		operationStats[result.Operation] = stats
	}

	// Verify concurrent operations worked correctly
	assert.Equal(t, numWorkers*opsPerWorker, totalOperations,
		"Should have performed all expected operations")

	for operation, stats := range operationStats {
		t.Logf("Operation %s: Total=%d, Successful=%d, Failed=%d",
			operation, stats.Total, stats.Successful, stats.Failed)

		// Most operations should succeed (except command execution which might fail due to system constraints)
		if operation != "command_execution" {
			successRate := float64(stats.Successful) / float64(stats.Total)
			assert.True(t, successRate > 0.8,
				"Success rate for %s should be high under concurrent load: %.2f", operation, successRate)
		}

		// Check for any security-related failures
		for _, errorMsg := range stats.Errors {
			assert.NotContains(t, errorMsg, "race condition")
			assert.NotContains(t, errorMsg, "concurrent access")
			assert.NotContains(t, errorMsg, "deadlock")
		}
	}

	// Verify logging system handled concurrent operations correctly
	allEntries := suite.logger.GetEntries()
	t.Logf("Total log entries from concurrent operations: %d", len(allEntries))

	// Should not have any race condition indicators in logs
	for _, entry := range allEntries {
		logContent := fmt.Sprintf("%v %v", entry.Message, entry.Fields)
		assert.NotContains(t, logContent, "race")
		assert.NotContains(t, logContent, "deadlock")
	}
}

// Helper types for concurrent testing
type SecurityOperationResult struct {
	WorkerID    int
	OperationID int
	Operation   string
	Success     bool
	Error       error
	Timestamp   time.Time
}

type OperationStats struct {
	Total      int
	Successful int
	Failed     int
	Errors     []string
}

// TestSecurityMetricsIntegration tests security metrics collection and analysis
func TestSecurityMetricsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping metrics integration tests in short mode")
	}

	suite := NewIntegrationSecurityTestSuite(t)
	defer suite.Cleanup()

	t.Run("security_violation_metrics", func(t *testing.T) {
		suite.testSecurityViolationMetrics(t)
	})

	t.Run("performance_metrics_under_attack", func(t *testing.T) {
		suite.testPerformanceMetricsUnderAttack(t)
	})
}

func (suite *IntegrationSecurityTestSuite) testSecurityViolationMetrics(t *testing.T) {
	// Generate various security violations and collect metrics
	violations := []struct {
		category string
		action   func() error
	}{
		{
			category: "command_injection",
			action: func() error {
				return suite.validator.ValidateCommandArgument("test; rm -rf /")
			},
		},
		{
			category: "path_traversal",
			action: func() error {
				return suite.validator.ValidateFilePath("../../../etc/passwd")
			},
		},
		{
			category: "privilege_escalation",
			action: func() error {
				ctx := context.Background()
				_, err := suite.executor.SecureExecute(ctx, "sudo", "whoami")
				return err
			},
		},
		{
			category: "network_injection",
			action: func() error {
				return suite.validator.ValidateIPAddress("127.0.0.1; cat /etc/passwd")
			},
		},
	}

	violationCounts := make(map[string]int)

	// Execute violations multiple times
	for i := 0; i < 10; i++ {
		for _, violation := range violations {
			suite.logger.Clear()
			err := violation.action()

			if err != nil {
				violationCounts[violation.category]++

				// Ensure violation is logged appropriately
				errorEntries := suite.logger.GetEntriesByLevel(ERROR)
				if len(errorEntries) > 0 {
					// Verify that violations are logged but sanitized
					for _, entry := range errorEntries {
						logContent := fmt.Sprintf("%v", entry.Fields)
						assert.NotContains(t, logContent, "rm -rf")
						assert.NotContains(t, logContent, "/etc/passwd")
					}
				}
			}
		}
	}

	// Verify metrics collection
	for category, count := range violationCounts {
		assert.Equal(t, 10, count,
			"Should have detected all violations for category: %s", category)
	}

	t.Logf("Security violation metrics: %+v", violationCounts)
}

func (suite *IntegrationSecurityTestSuite) testPerformanceMetricsUnderAttack(t *testing.T) {
	// Measure performance degradation under simulated attack
	baseline := suite.measureBaselinePerformance(t)

	// Simulate attack load
	attackLoad := suite.simulateAttackLoad(t)

	// Compare performance
	degradation := (attackLoad - baseline) / baseline
	t.Logf("Performance degradation under attack: %.2f%% (baseline: %v, under attack: %v)",
		degradation*100, baseline, attackLoad)

	// Performance degradation should be reasonable (less than 50%)
	assert.True(t, degradation < 0.5,
		"Performance degradation under attack should be reasonable: %.2f%%", degradation*100)
}

func (suite *IntegrationSecurityTestSuite) measureBaselinePerformance(t *testing.T) time.Duration {
	const numOps = 1000

	start := time.Now()
	for i := 0; i < numOps; i++ {
		suite.validator.ValidateCommandArgument(fmt.Sprintf("legitimate_arg_%d", i))
		suite.validator.ValidateIPAddress(fmt.Sprintf("192.168.1.%d", (i%254)+1))
		suite.validator.ValidateFilePath(fmt.Sprintf("/tmp/file_%d.txt", i))
	}
	return time.Since(start)
}

func (suite *IntegrationSecurityTestSuite) simulateAttackLoad(t *testing.T) time.Duration {
	const numOps = 1000

	maliciousInputs := []string{
		"arg; rm -rf /",
		"192.168.1.1; cat /etc/passwd",
		"../../../etc/passwd",
	}

	start := time.Now()
	for i := 0; i < numOps; i++ {
		maliciousInput := maliciousInputs[i%len(maliciousInputs)]

		suite.validator.ValidateCommandArgument(maliciousInput)
		suite.validator.ValidateIPAddress(maliciousInput)
		suite.validator.ValidateFilePath(maliciousInput)
	}
	return time.Since(start)
}

// TestSecurityComplianceIntegration tests compliance with security standards
func TestSecurityComplianceIntegration(t *testing.T) {
	suite := NewIntegrationSecurityTestSuite(t)
	defer suite.Cleanup()

	t.Run("owasp_top_10_compliance", func(t *testing.T) {
		suite.testOWASPTop10Compliance(t)
	})

	t.Run("cwe_top_25_compliance", func(t *testing.T) {
		suite.testCWETop25Compliance(t)
	})
}

func (suite *IntegrationSecurityTestSuite) testOWASPTop10Compliance(t *testing.T) {
	// Test against OWASP Top 10 security risks
	owaspTests := []struct {
		name        string
		risk        string
		testFunc    func() bool
		description string
	}{
		{
			name: "injection_prevention",
			risk: "A03:2021 – Injection",
			testFunc: func() bool {
				// Test command injection prevention
				err := suite.validator.ValidateCommandArgument("test; rm -rf /")
				return err != nil
			},
			description: "Should prevent injection attacks",
		},
		{
			name: "security_logging",
			risk: "A09:2021 – Security Logging and Monitoring Failures",
			testFunc: func() bool {
				// Test security logging
				suite.logger.Clear()
				suite.validator.ValidateCommandArgument("malicious; attack")

				entries := suite.logger.GetEntriesByLevel(ERROR)
				return len(entries) > 0
			},
			description: "Should log security events",
		},
		{
			name: "server_side_request_forgery",
			risk: "A10:2021 – Server-Side Request Forgery (SSRF)",
			testFunc: func() bool {
				// Test SSRF prevention
				err := suite.validator.ValidateIPAddress("169.254.169.254") // AWS metadata service
				return err != nil
			},
			description: "Should prevent SSRF attacks",
		},
	}

	for _, test := range owaspTests {
		t.Run(test.name, func(t *testing.T) {
			passed := test.testFunc()
			assert.True(t, passed, "OWASP compliance test failed: %s - %s", test.risk, test.description)
		})
	}
}

func (suite *IntegrationSecurityTestSuite) testCWETop25Compliance(t *testing.T) {
	// Test against CWE Top 25 Most Dangerous Software Weaknesses
	cweTests := []struct {
		name        string
		cwe         string
		testFunc    func() bool
		description string
	}{
		{
			name: "command_injection_cwe78",
			cwe:  "CWE-78: Improper Neutralization of Special Elements used in an OS Command",
			testFunc: func() bool {
				err := suite.validator.ValidateCommandArgument("test`whoami`")
				return err != nil
			},
			description: "Should prevent OS command injection",
		},
		{
			name: "path_traversal_cwe22",
			cwe:  "CWE-22: Improper Limitation of a Pathname to a Restricted Directory",
			testFunc: func() bool {
				err := suite.validator.ValidateFilePath("../../../etc/passwd")
				return err != nil
			},
			description: "Should prevent path traversal",
		},
		{
			name: "buffer_overflow_cwe120",
			cwe:  "CWE-120: Buffer Copy without Checking Size of Input",
			testFunc: func() bool {
				longInput := strings.Repeat("A", 10000)
				err := suite.validator.ValidateCommandArgument(longInput)
				return err != nil
			},
			description: "Should prevent buffer overflow",
		},
	}

	for _, test := range cweTests {
		t.Run(test.name, func(t *testing.T) {
			passed := test.testFunc()
			assert.True(t, passed, "CWE compliance test failed: %s - %s", test.cwe, test.description)
		})
	}
}

// BenchmarkIntegrationSecurity benchmarks the entire security workflow
func BenchmarkIntegrationSecurity(b *testing.B) {
	suite := NewIntegrationSecurityTestSuite(&testing.T{})
	defer suite.Cleanup()

	b.Run("end_to_end_validation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.validator.ValidateCommandArgument("test_arg")
			suite.validator.ValidateIPAddress("192.168.1.1")
			suite.validator.ValidateFilePath("/tmp/test.txt")
		}
	})

	b.Run("security_violation_handling", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.validator.ValidateCommandArgument("malicious; rm -rf /")
			suite.validator.ValidateIPAddress("192.168.1.1; cat /etc/passwd")
			suite.validator.ValidateFilePath("../../../etc/passwd")
		}
	})

	b.Run("error_handling_workflow", func(b *testing.B) {
		testErr := fmt.Errorf("test error with sensitive data: password=secret123")
		context := map[string]interface{}{
			"user":   "test",
			"action": "benchmark",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.errorHandler.HandleError(testErr, context)
		}
	})
}