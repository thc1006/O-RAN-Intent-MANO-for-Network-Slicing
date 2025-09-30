// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

// Package framework provides modern testing utilities for the O-RAN Intent MANO project
// This package implements 2025 testing standards including:
// - testify v1.11.1 features
// - Parallel test execution
// - Proper test cleanup
// - Mock and fixture management
// - Table-driven test helpers
// - Performance benchmarking utilities
package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestContext provides a standardized test context with cleanup and timeout management
type TestContext struct {
	ctx        context.Context
	cancel     context.CancelFunc
	t          *testing.T
	cleanup    []func()
	tempDirs   []string
	tempFiles  []string
	startTime  time.Time
}

// NewTestContext creates a new test context with timeout and cleanup management
func NewTestContext(t *testing.T, timeout time.Duration) *TestContext {
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	tc := &TestContext{
		ctx:       ctx,
		cancel:    cancel,
		t:         t,
		cleanup:   make([]func(), 0),
		tempDirs:  make([]string, 0),
		tempFiles: make([]string, 0),
		startTime: time.Now(),
	}

	// Register cleanup with testing.T
	t.Cleanup(func() {
		tc.Cleanup()
	})

	return tc
}

// Context returns the test context
func (tc *TestContext) Context() context.Context {
	return tc.ctx
}

// AddCleanup adds a cleanup function to be called when the test completes
func (tc *TestContext) AddCleanup(fn func()) {
	tc.cleanup = append(tc.cleanup, fn)
}

// CreateTempDir creates a temporary directory and registers it for cleanup
func (tc *TestContext) CreateTempDir(pattern string) string {
	dir, err := os.MkdirTemp("", pattern)
	require.NoError(tc.t, err, "Failed to create temp directory")

	tc.tempDirs = append(tc.tempDirs, dir)
	return dir
}

// CreateTempFile creates a temporary file and registers it for cleanup
func (tc *TestContext) CreateTempFile(dir, pattern string) *os.File {
	file, err := os.CreateTemp(dir, pattern)
	require.NoError(tc.t, err, "Failed to create temp file")

	tc.tempFiles = append(tc.tempFiles, file.Name())
	return file
}

// Cleanup performs all registered cleanup operations
func (tc *TestContext) Cleanup() {
	// Cancel context first
	tc.cancel()

	// Execute cleanup functions in reverse order
	for i := len(tc.cleanup) - 1; i >= 0; i-- {
		tc.cleanup[i]()
	}

	// Clean up temp files
	for _, file := range tc.tempFiles {
		if err := os.Remove(file); err != nil {
			tc.t.Logf("Warning: Failed to remove temp file %s: %v", file, err)
		}
	}

	// Clean up temp directories
	for _, dir := range tc.tempDirs {
		if err := os.RemoveAll(dir); err != nil {
			tc.t.Logf("Warning: Failed to remove temp directory %s: %v", dir, err)
		}
	}

	tc.t.Logf("Test completed in %v", time.Since(tc.startTime))
}

// TableTest represents a single test case for table-driven tests
type TableTest[T any] struct {
	Name        string
	Input       T
	Expected    interface{}
	ExpectError bool
	Setup       func(*testing.T) *TestContext
	Validate    func(*testing.T, *TestContext, interface{}, error)
}

// RunTableTests executes a table of test cases with proper setup and cleanup
func RunTableTests[T any](t *testing.T, tests []TableTest[T], testFunc func(*TestContext, T) (interface{}, error)) {
	for _, tt := range tests {
		// Capture loop variable for closure
		test := tt
		t.Run(test.Name, func(t *testing.T) {
			// Enable parallel execution for table tests
			t.Parallel()

			var tc *TestContext
			if test.Setup != nil {
				tc = test.Setup(t)
			} else {
				tc = NewTestContext(t, 10*time.Second)
			}

			// Execute the test function
			result, err := testFunc(tc, test.Input)

			// Validate error expectation
			if test.ExpectError {
				assert.Error(t, err, "Expected error for test case: %s", test.Name)
			} else {
				assert.NoError(t, err, "Unexpected error for test case: %s", test.Name)
			}

			// Custom validation if provided
			if test.Validate != nil {
				test.Validate(t, tc, result, err)
			} else if !test.ExpectError && test.Expected != nil {
				assert.Equal(t, test.Expected, result, "Result mismatch for test case: %s", test.Name)
			}
		})
	}
}

// ModernTestSuite provides a base test suite with modern testing patterns
type ModernTestSuite struct {
	suite.Suite
	TestContext *TestContext
}

// SetupTest runs before each test in the suite
func (s *ModernTestSuite) SetupTest() {
	s.TestContext = NewTestContext(s.T(), 30*time.Second)
}

// TearDownTest runs after each test in the suite
func (s *ModernTestSuite) TearDownTest() {
	if s.TestContext != nil {
		s.TestContext.Cleanup()
	}
}

// BenchmarkConfig provides configuration for benchmark tests
type BenchmarkConfig struct {
	MinTime     time.Duration
	MaxTime     time.Duration
	MemoryLimit int64 // in bytes
	SetupFunc   func(*testing.B) func() // Returns cleanup function
}

// DefaultBenchmarkConfig returns a default benchmark configuration
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		MinTime:     100 * time.Millisecond,
		MaxTime:     10 * time.Second,
		MemoryLimit: 100 * 1024 * 1024, // 100MB
	}
}

// RunBenchmarkWithConfig runs a benchmark with the specified configuration
func RunBenchmarkWithConfig(b *testing.B, config BenchmarkConfig, benchFunc func(*testing.B)) {
	var cleanup func()
	if config.SetupFunc != nil {
		cleanup = config.SetupFunc(b)
		if cleanup != nil {
			defer cleanup()
		}
	}

	// Track memory usage
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	start := time.Now()

	benchFunc(b)

	elapsed := time.Since(start)
	b.StopTimer()

	runtime.ReadMemStats(&m2)
	memUsed := m2.TotalAlloc - m1.TotalAlloc

	// Report metrics
	b.ReportMetric(float64(elapsed.Nanoseconds())/float64(b.N), "ns/op")
	b.ReportMetric(float64(memUsed)/float64(b.N), "B/op")

	// Check constraints
	if elapsed < config.MinTime {
		b.Logf("Warning: Benchmark ran for only %v, consider increasing work", elapsed)
	}

	// Safe integer overflow check: ensure memUsed fits in int64 before comparison
	if config.MemoryLimit > 0 {
		// Check if memUsed would overflow int64
		const maxInt64 = int64(^uint64(0) >> 1)
		if memUsed > uint64(maxInt64) {
			b.Errorf("Memory usage %d bytes exceeds int64 maximum and limit %d bytes", memUsed, config.MemoryLimit)
		} else if int64(memUsed) > config.MemoryLimit {
			b.Errorf("Memory usage %d bytes exceeds limit %d bytes", memUsed, config.MemoryLimit)
		}
	}
}

// MockHelper provides utilities for creating and managing mocks
type MockHelper struct {
	t     *testing.T
	mocks []interface{ AssertExpectations(t *testing.T) bool }
}

// NewMockHelper creates a new mock helper
func NewMockHelper(t *testing.T) *MockHelper {
	mh := &MockHelper{t: t}

	// Register cleanup to assert all expectations
	t.Cleanup(func() {
		mh.AssertAllExpectations()
	})

	return mh
}

// RegisterMock registers a mock for automatic expectation assertion
func (mh *MockHelper) RegisterMock(mock interface{ AssertExpectations(t *testing.T) bool }) {
	mh.mocks = append(mh.mocks, mock)
}

// AssertAllExpectations asserts expectations on all registered mocks
func (mh *MockHelper) AssertAllExpectations() {
	for _, mock := range mh.mocks {
		mock.AssertExpectations(mh.t)
	}
}

// TestDataManager helps manage test data and fixtures
type TestDataManager struct {
	tc       *TestContext
	dataDir  string
	fixtures map[string]interface{}
}

// NewTestDataManager creates a new test data manager
func NewTestDataManager(tc *TestContext) *TestDataManager {
	dataDir := tc.CreateTempDir("testdata_*")

	return &TestDataManager{
		tc:       tc,
		dataDir:  dataDir,
		fixtures: make(map[string]interface{}),
	}
}

// DataDir returns the test data directory
func (tdm *TestDataManager) DataDir() string {
	return tdm.dataDir
}

// LoadFixture loads a test fixture by name
func (tdm *TestDataManager) LoadFixture(name string, target interface{}) error {
	if fixture, exists := tdm.fixtures[name]; exists {
		// Implement fixture loading logic based on type
		_ = fixture
		_ = target
		return nil
	}

	// Load from file system
	fixturePath := filepath.Join(tdm.dataDir, fmt.Sprintf("%s.json", name))
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		return fmt.Errorf("fixture %s not found", name)
	}

	// TODO: Implement JSON/YAML loading
	return nil
}

// SaveFixture saves a test fixture
func (tdm *TestDataManager) SaveFixture(name string, data interface{}) error {
	tdm.fixtures[name] = data

	// Also save to filesystem for persistence across test runs
	fixturePath := filepath.Join(tdm.dataDir, fmt.Sprintf("%s.json", name))

	// TODO: Implement JSON/YAML saving
	_ = fixturePath

	return nil
}

// ParallelTestGroup manages a group of parallel tests with shared setup
type ParallelTestGroup struct {
	name     string
	setupFn  func(*testing.T) *TestContext
	cleanupFn func(*TestContext)
}

// NewParallelTestGroup creates a new parallel test group
func NewParallelTestGroup(name string) *ParallelTestGroup {
	return &ParallelTestGroup{
		name: name,
	}
}

// WithSetup sets the setup function for the test group
func (ptg *ParallelTestGroup) WithSetup(fn func(*testing.T) *TestContext) *ParallelTestGroup {
	ptg.setupFn = fn
	return ptg
}

// WithCleanup sets the cleanup function for the test group
func (ptg *ParallelTestGroup) WithCleanup(fn func(*TestContext)) *ParallelTestGroup {
	ptg.cleanupFn = fn
	return ptg
}

// Run executes a test within this parallel group
func (ptg *ParallelTestGroup) Run(t *testing.T, name string, testFn func(*TestContext)) {
	t.Run(fmt.Sprintf("%s/%s", ptg.name, name), func(t *testing.T) {
		t.Parallel()

		var tc *TestContext
		if ptg.setupFn != nil {
			tc = ptg.setupFn(t)
		} else {
			tc = NewTestContext(t, 30*time.Second)
		}

		if ptg.cleanupFn != nil {
			tc.AddCleanup(func() {
				ptg.cleanupFn(tc)
			})
		}

		testFn(tc)
	})
}

// IntegrationTestConfig provides configuration for integration tests
type IntegrationTestConfig struct {
	SkipIfShort     bool
	RequireNetwork  bool
	RequireDocker   bool
	RequireKubectl  bool
	Timeout         time.Duration
	SetupTimeout    time.Duration
	CleanupTimeout  time.Duration
}

// DefaultIntegrationConfig returns default integration test configuration
func DefaultIntegrationConfig() IntegrationTestConfig {
	return IntegrationTestConfig{
		SkipIfShort:    true,
		RequireNetwork: false,
		RequireDocker:  false,
		RequireKubectl: false,
		Timeout:        5 * time.Minute,
		SetupTimeout:   30 * time.Second,
		CleanupTimeout: 30 * time.Second,
	}
}

// RunIntegrationTest runs an integration test with proper prerequisites checking
func RunIntegrationTest(t *testing.T, config IntegrationTestConfig, testFn func(*TestContext)) {
	if config.SkipIfShort && testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check prerequisites
	if config.RequireDocker {
		if _, err := os.Stat("/var/run/docker.sock"); err != nil {
			t.Skip("Docker not available, skipping integration test")
		}
	}

	if config.RequireKubectl {
		// Simple check for kubectl
		if _, err := os.Stat("/usr/local/bin/kubectl"); err != nil {
			if _, err := os.Stat("/usr/bin/kubectl"); err != nil {
				t.Skip("kubectl not available, skipping integration test")
			}
		}
	}

	tc := NewTestContext(t, config.Timeout)
	testFn(tc)
}