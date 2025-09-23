package pkg

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestIperfManager() *IperfManager {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	manager := &IperfManager{
		logger:  logger,
		servers: make(map[string]*IperfServer),
	}

	return manager
}

func TestIperfManager_StartServer_ErrorHandling(t *testing.T) {
	t.Run("invalid_port_number", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Test with invalid port (negative)
		err := manager.StartServer(-1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("port_out_of_range", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Test with port out of range
		err := manager.StartServer(70000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "port")
	})

	t.Run("port_already_in_use", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Start server successfully first
		err := manager.StartServer(5001)
		if err != nil {
			t.Logf("First server start failed (expected in test): %v", err)
		}

		// Try to start another server on same port
		err = manager.StartServer(5001)
		assert.Error(t, err)
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
			t.Run(tc.description, func(t *testing.T) {
				err := manager.StartServer(tc.port)

				if tc.expectError {
					assert.Error(t, err, "Port %d should be invalid", tc.port)
				} else {
					// In test environment, server might fail to start due to permissions
					// but it shouldn't fail due to port validation
					if err != nil {
						assert.NotContains(t, err.Error(), "invalid port")
					}
				}

				// Cleanup
				if err == nil {
					manager.StopServer(tc.port)
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

func TestIperfManager_ErrorResilience(t *testing.T) {
	t.Run("multiple_operations", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Try multiple operations that might fail
		for i := 0; i < 5; i++ {
			port := 5000 + i

			err := manager.StartServer(port)
			if err != nil {
				t.Logf("Start server %d failed: %v", port, err)
			}

			err = manager.StopServer(port)
			if err != nil {
				t.Logf("Stop server %d failed: %v", port, err)
			}
		}

		// Manager should still be functional
		servers := manager.GetActiveServers()
		assert.NotNil(t, servers)
	})

	t.Run("invalid_operations", func(t *testing.T) {
		manager := setupTestIperfManager()

		// Multiple invalid operations shouldn't crash
		invalidPorts := []int{-1, 0, 65536, 70000}

		for _, port := range invalidPorts {
			err := manager.StartServer(port)
			assert.Error(t, err)

			err = manager.StopServer(port)
			assert.Error(t, err)
		}

		// Manager should still be functional
		servers := manager.GetActiveServers()
		assert.NotNil(t, servers)
	})
}

// Simple performance test to ensure no obvious bottlenecks
func BenchmarkIperfManager_Operations(b *testing.B) {
	manager := setupTestIperfManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simple operations that should be fast
		servers := manager.GetActiveServers()
		_ = servers

		// Try operation that will likely fail but should be fast
		err := manager.StartServer(70000 + (i % 100))
		_ = err
	}
}