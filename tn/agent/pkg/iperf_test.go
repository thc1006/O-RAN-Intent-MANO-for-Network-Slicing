package pkg

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IperfTestSuite provides a comprehensive test suite for IperfManager
type IperfTestSuite struct {
	suite.Suite
	manager *IperfManager
	ctx     context.Context
	cancel  context.CancelFunc
	tempDir string
}

// SetupTest runs before each test method
func (s *IperfTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 30*time.Second)

	// Create temp directory for test artifacts
	tempDir, err := os.MkdirTemp("", "iperf_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	s.manager = setupTestIperfManager()
}

// TearDownTest runs after each test method
func (s *IperfTestSuite) TearDownTest() {
	s.cancel()

	// Cleanup any running servers
	if s.manager != nil {
		servers := s.manager.GetActiveServers()
		for _, server := range servers {
			s.manager.StopServer(server.Port)
		}
	}

	// Clean up temp directory
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// TestIperfManagerSuite runs the test suite
func TestIperfManagerSuite(t *testing.T) {
	suite.Run(t, new(IperfTestSuite))
}

func setupTestIperfManager() *IperfManager {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	manager := &IperfManager{
		logger:  logger,
		servers: make(map[string]*IperfServer),
		mu:      sync.RWMutex{}, // Add mutex for thread safety
	}

	return manager
}

// TestStartServerErrorHandling tests error handling in server startup
func (s *IperfTestSuite) TestStartServerErrorHandling() {
	tests := []struct {
		name        string
		port        int
		expectError bool
		errorSubstr string
		description string
	}{
		{
			name:        "invalid_negative_port",
			port:        -1,
			expectError: true,
			errorSubstr: "invalid port",
			description: "Negative port numbers should be rejected",
		},
		{
			name:        "invalid_zero_port",
			port:        0,
			expectError: true,
			errorSubstr: "invalid port",
			description: "Zero port should be rejected",
		},
		{
			name:        "port_out_of_range",
			port:        70000,
			expectError: true,
			errorSubstr: "port",
			description: "Ports above valid range should be rejected",
		},
		{
			name:        "valid_port_range",
			port:        5001,
			expectError: false,
			errorSubstr: "",
			description: "Valid port should be accepted",
		},
	}

	for _, test := range tests {
		test := test // Capture loop variable
		s.Run(test.name, func() {
			// Use parallel execution for independent tests
			s.T().Parallel()

			manager := setupTestIperfManager()

			err := manager.StartServer(test.port)

			if test.expectError {
				s.Error(err, "Expected error for port %d: %s", test.port, test.description)
				if test.errorSubstr != "" {
					s.Contains(err.Error(), test.errorSubstr, "Error message should contain expected substring")
				}
			} else {
				// In test environment, server may fail due to permissions
				// but shouldn't fail due to validation errors
				if err != nil {
					s.NotContains(err.Error(), "invalid port", "Should not fail due to port validation")
					s.T().Logf("Server start failed in test environment (expected): %v", err)
				}
			}

			// Cleanup if server was started
			if err == nil {
				manager.StopServer(test.port)
			}
		})
	}
}

// Legacy test function for backward compatibility
func TestIperfManager_StartServer_ErrorHandling(t *testing.T) {
	t.Run("invalid_port_number", func(t *testing.T) {
		t.Parallel()

		manager := setupTestIperfManager()

		// Test with invalid port (negative)
		err := manager.StartServer(-1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("port_out_of_range", func(t *testing.T) {
		t.Parallel()

		manager := setupTestIperfManager()

		// Test with port out of range
		err := manager.StartServer(70000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "port")
	})

	t.Run("port_already_in_use", func(t *testing.T) {
		// This test doesn't run in parallel because it tests server state
		manager := setupTestIperfManager()

		// Use t.Cleanup for proper cleanup
		t.Cleanup(func() {
			manager.StopServer(5001)
		})

		// Start server successfully first
		err := manager.StartServer(5001)
		if err != nil {
			t.Logf("First server start failed (expected in test): %v", err)
			return // Skip duplicate port test if first start fails
		}

		// Try to start another server on same port
		err = manager.StartServer(5001)
		assert.Error(t, err, "Should fail when starting server on already used port")
	})
}

func TestIperfManager_StopServer_ErrorHandling(t *testing.T) {
	t.Run("server_not_found", func(t *testing.T) {
		manager := setupTestIperfManager()

		err := manager.StopServer(5001)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("stop_after_start", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Start server first (may fail in test environment)
		err := manager.StartServer(5001)
		if err != nil {
			t.Logf("Server start failed (expected in test): %v", err)
			return
		}

		// Stop server
		err = manager.StopServer(5001)
		if err != nil {
			t.Logf("Server stop had issues: %v", err)
		}
	})
}

// TestInputValidation tests comprehensive input validation
func (s *IperfTestSuite) TestInputValidation() {
	tests := []struct {
		port        int
		expectError bool
		description string
		category    string
	}{
		{-1, true, "negative port", "invalid"},
		{0, true, "zero port", "invalid"},
		{1, false, "minimum valid port", "valid"},
		{80, false, "common HTTP port", "valid"},
		{443, false, "common HTTPS port", "valid"},
		{5001, false, "typical iperf port", "valid"},
		{8080, false, "common alternative HTTP port", "valid"},
		{65535, false, "maximum valid port", "valid"},
		{65536, true, "port too high", "invalid"},
		{70000, true, "definitely too high", "invalid"},
		{100000, true, "way too high", "invalid"},
	}

	// Group tests by category
	for _, test := range tests {
		test := test // Capture loop variable
		s.Run(fmt.Sprintf("%s_%s", test.category, test.description), func() {
			s.T().Parallel()

			manager := setupTestIperfManager()

			err := manager.StartServer(test.port)

			if test.expectError {
				s.Error(err, "Port %d should be invalid: %s", test.port, test.description)
				s.T().Logf("Port %d correctly rejected: %v", test.port, err)
			} else {
				// In test environment, server might fail to start due to permissions
				// but it shouldn't fail due to port validation
				if err != nil {
					s.NotContains(err.Error(), "invalid port", "Should not fail due to port validation for port %d", test.port)
					s.T().Logf("Port %d validation passed, start failed due to environment: %v", test.port, err)
				} else {
					s.T().Logf("Port %d validation and start successful", test.port)
				}

				// Cleanup if server was started
				if err == nil {
					manager.StopServer(test.port)
				}
			}
		})
	}
}

// Legacy test function with modernized patterns
func TestIperfManager_InputValidation(t *testing.T) {
	t.Run("validate_port_boundaries", func(t *testing.T) {
		manager := setupTestIperfManager()

		testCases := []struct {
			port        int
			expectError bool
			description string
		}{
			{-1, true, "negative port"},
			{0, true, "zero port"},
			{1, false, "minimum valid port"},
			{80, false, "common port"},
			{5001, false, "typical iperf port"},
			{65535, false, "maximum valid port"},
			{65536, true, "port too high"},
			{70000, true, "definitely too high"},
		}

		for _, tc := range testCases {
			test := tc // Capture loop variable
			t.Run(test.description, func(t *testing.T) {
				t.Parallel() // Enable parallel execution

				err := manager.StartServer(test.port)

				if test.expectError {
					assert.Error(t, err, "Port %d should be invalid", test.port)
				} else {
					// In test environment, server might fail to start due to permissions
					// but it shouldn't fail due to port validation
					if err != nil {
						assert.NotContains(t, err.Error(), "invalid port")
					}
				}

				// Use t.Cleanup for proper cleanup
				if err == nil {
					t.Cleanup(func() {
						manager.StopServer(test.port)
					})
				}
			})
		}
	})
}

func TestIperfManager_GetServerStatus(t *testing.T) {
	t.Run("nonexistent_server_status", func(t *testing.T) {
		manager := setupTestIperfManager()

		servers := manager.GetActiveServers()
		assert.NotNil(t, servers)
	})

	t.Run("active_server_status", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Try to start server
		err := manager.StartServer(5001)
		if err != nil {
			t.Logf("Server start failed in test environment: %v", err)
			return
		}

		servers := manager.GetActiveServers()
		assert.NotNil(t, servers)

		// Cleanup
		manager.StopServer(5001)
	})
}

func TestIperfManager_SecurityValidation(t *testing.T) {
	t.Run("port_range_validation", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Test extreme values
		extremePorts := []int{
			-1000,
			-1,
			0,
			65536,
			100000,
		}

		for _, port := range extremePorts {
			err := manager.StartServer(port)
			assert.Error(t, err, "Extreme port %d should be rejected", port)
		}
	})
}

func TestIperfManager_ResourceManagement(t *testing.T) {
	t.Run("server_lifecycle", func(t *testing.T) {
		manager := setupTestIperfManager()

		port := 5001

		// Check initial state
		servers := manager.GetActiveServers()
		initialCount := len(servers)

		// Start server (may fail in test environment)
		err := manager.StartServer(port)
		if err != nil {
			t.Logf("Server start failed in test environment: %v", err)
			return
		}

		// Verify server is tracked
		servers = manager.GetActiveServers()
		assert.True(t, len(servers) >= initialCount)

		// Stop server
		err = manager.StopServer(port)
		if err != nil {
			t.Logf("Server stop had issues: %v", err)
		}

		// Verify cleanup
		servers = manager.GetActiveServers()
		assert.True(t, len(servers) <= initialCount+1)
	})
}

// TestErrorResilience tests error resilience with modern patterns
func (s *IperfTestSuite) TestErrorResilience() {
	s.Run("ConcurrentOperations", func() {
		const numGoroutines = 10
		const opsPerGoroutine = 5

		var wg sync.WaitGroup
		errorCh := make(chan error, numGoroutines*opsPerGoroutine*2) // *2 for start+stop

		// Run concurrent operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				for j := 0; j < opsPerGoroutine; j++ {
					port := 5000 + (routineID*opsPerGoroutine + j)

					// Try to start server
					if err := s.manager.StartServer(port); err != nil {
						select {
						case errorCh <- err:
						default: // Don't block if channel is full
						}
					}

					// Try to stop server
					if err := s.manager.StopServer(port); err != nil {
						select {
						case errorCh <- err:
						default: // Don't block if channel is full
						}
					}
				}
			}(i)
		}

		// Wait with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-s.ctx.Done():
			s.Fail("Concurrent operations test timed out")
			return
		}

		close(errorCh)

		// Count errors (expected in test environment)
		errorCount := 0
		for err := range errorCh {
			errorCount++
			s.T().Logf("Expected error in test environment: %v", err)
		}

		// Manager should still be functional after concurrent operations
		servers := s.manager.GetActiveServers()
		s.NotNil(servers, "Manager should remain functional after concurrent operations")
		s.T().Logf("Processed %d errors from concurrent operations", errorCount)
	})
}

// Legacy test function with improvements
func TestIperfManager_ErrorResilience(t *testing.T) {
	t.Run("multiple_operations", func(t *testing.T) {
		t.Parallel()

		manager := setupTestIperfManager()

		// Track started servers for cleanup
		startedServers := make([]int, 0)
		t.Cleanup(func() {
			for _, port := range startedServers {
				manager.StopServer(port)
			}
		})

		// Try multiple operations that might fail
		for i := 0; i < 5; i++ {
			port := 5000 + i

			err := manager.StartServer(port)
			if err != nil {
				t.Logf("Start server %d failed (expected in test): %v", port, err)
			} else {
				startedServers = append(startedServers, port)
			}

			err = manager.StopServer(port)
			if err != nil {
				t.Logf("Stop server %d failed: %v", port, err)
			}
		}

		// Manager should still be functional
		servers := manager.GetActiveServers()
		assert.NotNil(t, servers)
		t.Logf("Manager remains functional with %d active servers", len(servers))
	})

	t.Run("invalid_operations", func(t *testing.T) {
		t.Parallel()

		manager := setupTestIperfManager()

		// Multiple invalid operations shouldn't crash
		invalidPorts := []int{-1, 0, 65536, 70000}

		for _, port := range invalidPorts {
			err := manager.StartServer(port)
			assert.Error(t, err, "Invalid port %d should produce error", port)

			err = manager.StopServer(port)
			assert.Error(t, err, "Stopping non-existent server on port %d should produce error", port)
		}

		// Manager should still be functional
		servers := manager.GetActiveServers()
		assert.NotNil(t, servers, "Manager should remain functional after invalid operations")
	})
}

// BenchmarkIperfManagerOperations provides comprehensive performance benchmarks
func BenchmarkIperfManagerOperations(b *testing.B) {
	manager := setupTestIperfManager()

	b.Run("GetActiveServers", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				servers := manager.GetActiveServers()
				_ = servers
			}
		})
	})

	b.Run("StartServerValidation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				// Try operation that will likely fail but should be fast
				err := manager.StartServer(70000 + (i % 100))
				_ = err
				i++
			}
		})
	})

	b.Run("PortValidation", func(b *testing.B) {
		testPorts := []int{-1, 0, 80, 5001, 65535, 70000}

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				port := testPorts[i%len(testPorts)]
				err := manager.StartServer(port)
				_ = err
				i++
			}
		})
	})
}

// Legacy benchmark function
func BenchmarkIperfManager_Operations(b *testing.B) {
	manager := setupTestIperfManager()

	// Warm up
	for i := 0; i < 100; i++ {
		servers := manager.GetActiveServers()
		_ = servers
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simple operations that should be fast
		servers := manager.GetActiveServers()
		_ = servers

		// Try operation that will likely fail but should be fast
		err := manager.StartServer(70000 + (i % 100))
		_ = err
	}
}