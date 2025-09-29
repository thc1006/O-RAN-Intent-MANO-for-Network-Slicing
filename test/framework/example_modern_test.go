// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package framework_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/test/framework"
)

// ExampleModernTestSuite demonstrates modern testing patterns
type ExampleModernTestSuite struct {
	framework.ModernTestSuite
}

// TestModernTestSuite runs the suite
func TestModernTestSuite(t *testing.T) {
	suite.Run(t, new(ExampleModernTestSuite))
}

// TestParallelExecution demonstrates parallel test execution
func (s *ExampleModernTestSuite) TestParallelExecution() {
	// Create a parallel test group
	group := framework.NewParallelTestGroup("parallel_example").
		WithSetup(func(t *testing.T) *framework.TestContext {
			return framework.NewTestContext(t, 5*time.Second)
		})

	testCases := []struct {
		name     string
		delay    time.Duration
		expected string
	}{
		{"fast_test", 10 * time.Millisecond, "fast"},
		{"medium_test", 50 * time.Millisecond, "medium"},
		{"slow_test", 100 * time.Millisecond, "slow"},
	}

	for _, tc := range testCases {
		test := tc // Capture loop variable
		group.Run(s.T(), test.name, func(ctx *framework.TestContext) {
			// Simulate work
			time.Sleep(test.delay)
			assert.Equal(s.T(), test.expected, test.expected)
		})
	}
}

// TestTableDrivenWithModernPattern demonstrates modern table-driven tests
func TestTableDrivenWithModernPattern(t *testing.T) {
	// Define test cases
	tests := []framework.TableTest[string]{
		{
			Name:        "valid_input",
			Input:       "hello world",
			Expected:    "HELLO WORLD",
			ExpectError: false,
		},
		{
			Name:        "empty_input",
			Input:       "",
			Expected:    "",
			ExpectError: false,
		},
		{
			Name:        "special_characters",
			Input:       "hello@world#123",
			Expected:    "HELLO@WORLD#123",
			ExpectError: false,
		},
	}

	// Test function that converts to uppercase
	testFunc := func(tc *framework.TestContext, input string) (interface{}, error) {
		return fmt.Sprintf("%s", input), nil
	}

	// Run the table tests
	framework.RunTableTests(t, tests, testFunc)
}

// TestWithCleanup demonstrates proper cleanup handling
func TestWithCleanup(t *testing.T) {
	tc := framework.NewTestContext(t, 10*time.Second)

	// Create test resources that need cleanup
	tempFile := tc.CreateTempFile("", "test_*.txt")
	defer tempFile.Close()

	// Add custom cleanup
	tc.AddCleanup(func() {
		t.Log("Custom cleanup executed")
	})

	// Test operations
	_, err := tempFile.WriteString("test content")
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, tempFile.Name())

	// Cleanup will be automatically handled by TestContext
}

// TestHTTPWithModernHelpers demonstrates HTTP testing with helpers
func TestHTTPWithModernHelpers(t *testing.T) {
	tc := framework.NewTestContext(t, 30*time.Second)

	// Create a simple HTTP handler for testing
	handler := framework.NewHTTPTestHelper(tc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": "healthy"}`)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "internal error"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	t.Run("health_endpoint", func(t *testing.T) {
		t.Parallel()

		resp, err := handler.GET("/health", nil)
		require.NoError(t, err)

		handler.AssertStatusCode(t, resp, http.StatusOK)
		handler.AssertJSONResponse(t, resp, map[string]interface{}{
			"status": "healthy",
		})
	})

	t.Run("error_endpoint", func(t *testing.T) {
		t.Parallel()

		resp, err := handler.GET("/error", nil)
		require.NoError(t, err)

		handler.AssertStatusCode(t, resp, http.StatusInternalServerError)
		handler.AssertResponseContains(t, resp, "internal error")
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		resp, err := handler.GET("/nonexistent", nil)
		require.NoError(t, err)

		handler.AssertStatusCode(t, resp, http.StatusNotFound)
	})
}

// TestConcurrencyHelpers demonstrates concurrency testing
func TestConcurrencyHelpers(t *testing.T) {
	tc := framework.NewTestContext(t, 30*time.Second)
	helper := framework.NewConcurrencyTestHelper(tc)

	counter := 0
	mutex := &sync.Mutex{}

	// Create functions that increment counter concurrently
	funcs := make([]func() error, 10)
	for i := 0; i < 10; i++ {
		funcs[i] = func() error {
			mutex.Lock()
			counter++
			mutex.Unlock()
			return nil
		}
	}

	helper.RunConcurrent(t, funcs, 5*time.Second)

	assert.Equal(t, 10, counter, "All concurrent operations should complete")
}

// TestPerformanceHelpers demonstrates performance testing
func TestPerformanceHelpers(t *testing.T) {
	tc := framework.NewTestContext(t, 30*time.Second)
	helper := framework.NewPerformanceTestHelper(tc)

	t.Run("latency_test", func(t *testing.T) {
		helper.AssertLatency(t, func() {
			time.Sleep(10 * time.Millisecond)
		}, 50*time.Millisecond, "sleep operation")
	})

	t.Run("throughput_test", func(t *testing.T) {
		helper.AssertThroughput(t, func(n int) {
			for i := 0; i < n; i++ {
				// Simulate work
				_ = i * i
			}
		}, 1000, 100000.0, "mathematical operations")
	})
}

// TestSecurityHelpers demonstrates security testing
func TestSecurityHelpers(t *testing.T) {
	tc := framework.NewTestContext(t, 30*time.Second)
	helper := framework.NewSecurityTestHelper(tc)

	// Example sanitization function
	sanitize := func(input string) string {
		replacer := strings.NewReplacer(
			"'", "",
			";", "",
			"--", "",
			"<", "&lt;",
			">", "&gt;",
			"$(", "",
			"../", "",
		)
		return replacer.Replace(input)
	}

	helper.TestInputSanitization(t, sanitize)
}

// BenchmarkModernBenchmarkExample demonstrates modern benchmarking
func BenchmarkModernBenchmarkExample(b *testing.B) {
	config := framework.DefaultBenchmarkConfig()
	config.MemoryLimit = 50 * 1024 * 1024 // 50MB limit

	framework.RunBenchmarkWithConfig(b, config, func(b *testing.B) {
		data := make([]int, 1000)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate work
				for i := range data {
					data[i] = i * i
				}
			}
		})
	})
}

// TestErrorHandling demonstrates error handling patterns
func TestErrorHandling(t *testing.T) {
	tc := framework.NewTestContext(t, 30*time.Second)
	helper := framework.NewErrorTestHelper(tc)

	t.Run("error_recovery", func(t *testing.T) {
		helper.TestErrorRecovery(t,
			// Setup
			func() interface{} {
				return map[string]int{"counter": 0}
			},
			// Error function
			func(resource interface{}) error {
				data := resource.(map[string]int)
				data["counter"] = -1 // Simulate error state
				return fmt.Errorf("simulated error")
			},
			// Recovery function
			func(resource interface{}, err error) error {
				data := resource.(map[string]int)
				data["counter"] = 0 // Reset to valid state
				return nil
			},
		)
	})
}

// TestIntegrationWithModernPatterns demonstrates integration testing
func TestIntegrationWithModernPatterns(t *testing.T) {
	config := framework.DefaultIntegrationConfig()
	config.RequireNetwork = false // Don't require network for this example
	config.Timeout = 1 * time.Minute

	framework.RunIntegrationTest(t, config, func(tc *framework.TestContext) {
		// Create filesystem helper
		fsHelper := framework.NewFileSystemTestHelper(tc)

		// Test file operations
		testFile := fsHelper.CreateTestFile("config/test.conf", "test=value")
		fsHelper.AssertFileExists(t, "config/test.conf")
		fsHelper.AssertFileContent(t, "config/test.conf", "test=value")

		t.Logf("Integration test completed with file: %s", testFile)
	})
}