// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// HTTPTestHelper provides utilities for HTTP testing with modern patterns
type HTTPTestHelper struct {
	Server   *httptest.Server
	Client   *http.Client
	TestCtx  *TestContext
	recorder *httptest.ResponseRecorder
}

// NewHTTPTestHelper creates a new HTTP test helper
func NewHTTPTestHelper(tc *TestContext, handler http.Handler) *HTTPTestHelper {
	server := httptest.NewServer(handler)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	helper := &HTTPTestHelper{
		Server:  server,
		Client:  client,
		TestCtx: tc,
	}

	// Register cleanup
	tc.AddCleanup(func() {
		server.Close()
	})

	return helper
}

// NewHTTPRecorderHelper creates a helper for testing handlers without a server
func NewHTTPRecorderHelper(tc *TestContext) *HTTPTestHelper {
	return &HTTPTestHelper{
		TestCtx:  tc,
		recorder: httptest.NewRecorder(),
	}
}

// GET performs a GET request and returns the response
func (h *HTTPTestHelper) GET(path string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(h.TestCtx.Context(), "GET", h.Server.URL+path, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.Client.Do(req)
}

// POST performs a POST request with JSON body
func (h *HTTPTestHelper) POST(path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequestWithContext(h.TestCtx.Context(), "POST", h.Server.URL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return h.Client.Do(req)
}

// AssertStatusCode asserts the HTTP status code
func (h *HTTPTestHelper) AssertStatusCode(t *testing.T, resp *http.Response, expectedStatus int) {
	assert.Equal(t, expectedStatus, resp.StatusCode,
		"HTTP status mismatch. Response body: %s", h.getResponseBody(resp))
}

// AssertJSONResponse asserts the JSON response body
func (h *HTTPTestHelper) AssertJSONResponse(t *testing.T, resp *http.Response, expected interface{}) {
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	defer resp.Body.Close()

	var actual interface{}
	err = json.Unmarshal(body, &actual)
	require.NoError(t, err, "Failed to unmarshal JSON response: %s", string(body))

	assert.Equal(t, expected, actual, "JSON response mismatch")
}

// AssertResponseContains asserts the response body contains specific text
func (h *HTTPTestHelper) AssertResponseContains(t *testing.T, resp *http.Response, expectedText string) {
	body := h.getResponseBody(resp)
	assert.Contains(t, body, expectedText, "Response body does not contain expected text")
}

// getResponseBody reads and returns the response body as string
func (h *HTTPTestHelper) getResponseBody(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading body: %v", err)
	}
	resp.Body.Close()
	return string(body)
}

// DatabaseTestHelper provides utilities for database testing
type DatabaseTestHelper struct {
	TestCtx   *TestContext
	TempDBDir string
}

// NewDatabaseTestHelper creates a new database test helper
func NewDatabaseTestHelper(tc *TestContext) *DatabaseTestHelper {
	tempDir := tc.CreateTempDir("test_db_*")

	return &DatabaseTestHelper{
		TestCtx:   tc,
		TempDBDir: tempDir,
	}
}

// MockServiceHelper provides utilities for mocking services
type MockServiceHelper struct {
	TestCtx *TestContext
	Mocks   map[string]*mock.Mock
}

// NewMockServiceHelper creates a new mock service helper
func NewMockServiceHelper(tc *TestContext) *MockServiceHelper {
	return &MockServiceHelper{
		TestCtx: tc,
		Mocks:   make(map[string]*mock.Mock),
	}
}

// RegisterMock registers a mock with a name for later reference
func (m *MockServiceHelper) RegisterMock(name string, mockObj *mock.Mock) {
	m.Mocks[name] = mockObj
}

// AssertAllExpectations asserts expectations on all registered mocks
func (m *MockServiceHelper) AssertAllExpectations(t *testing.T) {
	for name, mockObj := range m.Mocks {
		mockObj.AssertExpectations(t)
		t.Logf("Mock '%s' expectations satisfied", name)
	}
}

// FileSystemTestHelper provides utilities for filesystem testing
type FileSystemTestHelper struct {
	TestCtx  *TestContext
	TestRoot string
}

// NewFileSystemTestHelper creates a new filesystem test helper
func NewFileSystemTestHelper(tc *TestContext) *FileSystemTestHelper {
	testRoot := tc.CreateTempDir("fs_test_*")

	return &FileSystemTestHelper{
		TestCtx:  tc,
		TestRoot: testRoot,
	}
}

// CreateTestFile creates a test file with specified content
func (fs *FileSystemTestHelper) CreateTestFile(relativePath string, content string) string {
	fullPath := fmt.Sprintf("%s/%s", fs.TestRoot, relativePath)

	// Create directory if it doesn't exist
	dir := fmt.Sprintf("%s/%s", fs.TestRoot, strings.Split(relativePath, "/")[0])
	os.MkdirAll(dir, 0755)

	file, err := os.Create(fullPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to create test file %s: %v", fullPath, err))
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		panic(fmt.Sprintf("Failed to write to test file %s: %v", fullPath, err))
	}

	return fullPath
}

// AssertFileExists asserts that a file exists
func (fs *FileSystemTestHelper) AssertFileExists(t *testing.T, relativePath string) {
	fullPath := fmt.Sprintf("%s/%s", fs.TestRoot, relativePath)
	_, err := os.Stat(fullPath)
	assert.NoError(t, err, "File should exist: %s", fullPath)
}

// AssertFileContent asserts the content of a file
func (fs *FileSystemTestHelper) AssertFileContent(t *testing.T, relativePath string, expectedContent string) {
	fullPath := fmt.Sprintf("%s/%s", fs.TestRoot, relativePath)
	content, err := os.ReadFile(fullPath)
	require.NoError(t, err, "Failed to read file: %s", fullPath)
	assert.Equal(t, expectedContent, string(content), "File content mismatch: %s", fullPath)
}

// ValidationTestHelper provides utilities for validation testing
type ValidationTestHelper struct {
	TestCtx *TestContext
}

// NewValidationTestHelper creates a new validation test helper
func NewValidationTestHelper(tc *TestContext) *ValidationTestHelper {
	return &ValidationTestHelper{
		TestCtx: tc,
	}
}

// ValidateStruct validates a struct using reflection and custom rules
func (v *ValidationTestHelper) ValidateStruct(t *testing.T, obj interface{}, rules map[string]func(interface{}) error) {
	value := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)

	// Handle pointers
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
		typ = typ.Elem()
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name

		if rule, exists := rules[fieldName]; exists {
			err := rule(field.Interface())
			assert.NoError(t, err, "Validation failed for field %s", fieldName)
		}
	}
}

// ConcurrencyTestHelper provides utilities for testing concurrent operations
type ConcurrencyTestHelper struct {
	TestCtx *TestContext
}

// NewConcurrencyTestHelper creates a new concurrency test helper
func NewConcurrencyTestHelper(tc *TestContext) *ConcurrencyTestHelper {
	return &ConcurrencyTestHelper{
		TestCtx: tc,
	}
}

// RunConcurrent runs multiple functions concurrently and waits for completion
func (c *ConcurrencyTestHelper) RunConcurrent(t *testing.T, funcs []func() error, timeout time.Duration) {
	resultCh := make(chan error, len(funcs))

	// Start all functions concurrently
	for _, f := range funcs {
		go func(fn func() error) {
			resultCh <- fn()
		}(f)
	}

	// Collect results with timeout
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for i := 0; i < len(funcs); i++ {
		select {
		case err := <-resultCh:
			assert.NoError(t, err, "Concurrent function %d failed", i)
		case <-timer.C:
			t.Fatalf("Timeout waiting for concurrent operations to complete")
		}
	}
}

// PerformanceTestHelper provides utilities for performance testing
type PerformanceTestHelper struct {
	TestCtx *TestContext
}

// NewPerformanceTestHelper creates a new performance test helper
func NewPerformanceTestHelper(tc *TestContext) *PerformanceTestHelper {
	return &PerformanceTestHelper{
		TestCtx: tc,
	}
}

// MeasureLatency measures the latency of a function
func (p *PerformanceTestHelper) MeasureLatency(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// AssertLatency asserts that a function completes within expected time
func (p *PerformanceTestHelper) AssertLatency(t *testing.T, fn func(), maxLatency time.Duration, description string) {
	latency := p.MeasureLatency(fn)
	assert.LessOrEqual(t, latency, maxLatency,
		"%s took %v, expected <= %v", description, latency, maxLatency)
	t.Logf("%s completed in %v", description, latency)
}

// AssertThroughput measures and asserts minimum throughput
func (p *PerformanceTestHelper) AssertThroughput(t *testing.T, fn func(int), operations int,
	minThroughput float64, description string) {

	start := time.Now()
	fn(operations)
	duration := time.Since(start)

	actualThroughput := float64(operations) / duration.Seconds()
	assert.GreaterOrEqual(t, actualThroughput, minThroughput,
		"%s throughput %v ops/sec, expected >= %v ops/sec",
		description, actualThroughput, minThroughput)

	t.Logf("%s: %d operations in %v (%.2f ops/sec)",
		description, operations, duration, actualThroughput)
}

// SecurityTestHelper provides utilities for security testing
type SecurityTestHelper struct {
	TestCtx *TestContext
}

// NewSecurityTestHelper creates a new security test helper
func NewSecurityTestHelper(tc *TestContext) *SecurityTestHelper {
	return &SecurityTestHelper{
		TestCtx: tc,
	}
}

// TestInputSanitization tests input sanitization functions
func (s *SecurityTestHelper) TestInputSanitization(t *testing.T, sanitizeFn func(string) string) {
	maliciousInputs := []struct {
		input    string
		name     string
		mustNotContain []string
	}{
		{
			input: "'; DROP TABLE users; --",
			name:  "SQL injection",
			mustNotContain: []string{"'", ";", "--"},
		},
		{
			input: "<script>alert('xss')</script>",
			name:  "XSS script",
			mustNotContain: []string{"<script>", "alert"},
		},
		{
			input: "$(rm -rf /)",
			name:  "Command injection",
			mustNotContain: []string{"$(", "rm -rf"},
		},
		{
			input: "../../../etc/passwd",
			name:  "Path traversal",
			mustNotContain: []string{"../"},
		},
	}

	for _, test := range maliciousInputs {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizeFn(test.input)

			for _, forbidden := range test.mustNotContain {
				assert.NotContains(t, result, forbidden,
					"Sanitized output still contains dangerous pattern: %s", forbidden)
			}

			t.Logf("Input: %s -> Output: %s", test.input, result)
		})
	}
}

// ErrorTestHelper provides utilities for testing error handling
type ErrorTestHelper struct {
	TestCtx *TestContext
}

// NewErrorTestHelper creates a new error test helper
func NewErrorTestHelper(tc *TestContext) *ErrorTestHelper {
	return &ErrorTestHelper{
		TestCtx: tc,
	}
}

// AssertErrorType asserts that an error is of a specific type
func (e *ErrorTestHelper) AssertErrorType(t *testing.T, err error, expectedType interface{}) {
	require.Error(t, err, "Expected an error")
	assert.IsType(t, expectedType, err, "Error type mismatch")
}

// AssertErrorChain asserts that an error chain contains a specific error
func (e *ErrorTestHelper) AssertErrorChain(t *testing.T, err error, expectedErr error) {
	require.Error(t, err, "Expected an error")
	assert.ErrorIs(t, err, expectedErr, "Error chain does not contain expected error")
}

// TestErrorRecovery tests recovery from various error conditions
func (e *ErrorTestHelper) TestErrorRecovery(t *testing.T, setupFn func() interface{},
	errorFn func(interface{}) error, recoveryFn func(interface{}, error) error) {

	resource := setupFn()

	// Cause the error
	err := errorFn(resource)
	require.Error(t, err, "Expected error condition to occur")

	// Test recovery
	recoveryErr := recoveryFn(resource, err)
	assert.NoError(t, recoveryErr, "Recovery should succeed")
}