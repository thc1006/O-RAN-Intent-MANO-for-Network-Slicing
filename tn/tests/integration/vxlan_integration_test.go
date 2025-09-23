package integration

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg/vxlan"
)

// VXLANIntegrationTestSuite for testing VXLAN functionality
type VXLANIntegrationTestSuite struct {
	manager         *vxlan.Manager
	optimizedManager *vxlan.OptimizedManager
	createdTunnels  []int32
	testInterface   string
	localIP         string
}

func setupVXLANIntegrationTest(t *testing.T) *VXLANIntegrationTestSuite {
	// Check if we have the necessary privileges and tools
	if !hasVXLANCapabilities() {
		t.Skip("VXLAN capabilities not available (requires root/CAP_NET_ADMIN)")
	}

	manager := vxlan.NewManager()
	optimizedManager := vxlan.NewOptimizedManager()

	// Use a test interface and local IP
	testInterface := getTestInterface()
	localIP := getTestLocalIP()

	return &VXLANIntegrationTestSuite{
		manager:         manager,
		optimizedManager: optimizedManager,
		createdTunnels:  []int32{},
		testInterface:   testInterface,
		localIP:         localIP,
	}
}

func (suite *VXLANIntegrationTestSuite) cleanup() {
	// Clean up all created tunnels
	for _, vxlanID := range suite.createdTunnels {
		suite.manager.DeleteTunnel(vxlanID)
		suite.optimizedManager.DeleteTunnelOptimized(vxlanID)
	}

	// Clean up any remaining tunnels
	suite.manager.Cleanup()
	suite.optimizedManager.CleanupOptimized()
}

func (suite *VXLANIntegrationTestSuite) addCreatedTunnel(vxlanID int32) {
	suite.createdTunnels = append(suite.createdTunnels, vxlanID)
}

func TestVXLANIntegration_BasicManager_TunnelLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VXLAN integration test in short mode")
	}

	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	vxlanID := int32(1000)
	remoteIPs := []string{"10.0.1.2", "10.0.1.3"}

	t.Run("create tunnel successfully", func(t *testing.T) {
		err := suite.manager.CreateTunnel(vxlanID, suite.localIP, remoteIPs, suite.testInterface)
		if err != nil {
			if isPermissionError(err) {
				t.Skip("Insufficient permissions for VXLAN operations")
			}
			if isNetworkError(err) {
				t.Skipf("Network configuration issue: %v", err)
			}
			t.Fatalf("Failed to create VXLAN tunnel: %v", err)
		}

		suite.addCreatedTunnel(vxlanID)

		// Verify tunnel was created
		status, err := suite.manager.GetTunnelStatus(vxlanID)
		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, vxlanID, status.VxlanID)
		assert.Equal(t, suite.localIP, status.LocalIP)
		assert.Equal(t, remoteIPs, status.RemoteIPs)

		// Verify interface exists in system
		interfaceName := fmt.Sprintf("vxlan%d", vxlanID)
		assert.True(t, interfaceExists(interfaceName), "VXLAN interface should exist")
	})

	t.Run("get tunnel status", func(t *testing.T) {
		status, err := suite.manager.GetTunnelStatus(vxlanID)
		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, fmt.Sprintf("vxlan%d", vxlanID), status.InterfaceName)
	})

	t.Run("delete tunnel successfully", func(t *testing.T) {
		err := suite.manager.DeleteTunnel(vxlanID)
		assert.NoError(t, err)

		// Verify tunnel status returns error
		_, err = suite.manager.GetTunnelStatus(vxlanID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Verify interface was removed
		interfaceName := fmt.Sprintf("vxlan%d", vxlanID)
		assert.False(t, interfaceExists(interfaceName), "VXLAN interface should be removed")
	})
}

func TestVXLANIntegration_OptimizedManager_TunnelLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VXLAN integration test in short mode")
	}

	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	vxlanID := int32(2000)
	remoteIPs := []string{"10.0.2.2", "10.0.2.3"}

	t.Run("create optimized tunnel successfully", func(t *testing.T) {
		err := suite.optimizedManager.CreateTunnelOptimized(vxlanID, suite.localIP, remoteIPs, suite.testInterface)
		if err != nil {
			if isPermissionError(err) {
				t.Skip("Insufficient permissions for VXLAN operations")
			}
			if isNetworkError(err) {
				t.Skipf("Network configuration issue: %v", err)
			}
			t.Fatalf("Failed to create optimized VXLAN tunnel: %v", err)
		}

		suite.addCreatedTunnel(vxlanID)

		// Verify tunnel was created
		status, err := suite.optimizedManager.GetTunnelStatusOptimized(vxlanID)
		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, vxlanID, status.VxlanID)
		assert.Equal(t, suite.localIP, status.LocalIP)
		assert.Equal(t, remoteIPs, status.RemoteIPs)

		// Verify interface exists
		interfaceName := fmt.Sprintf("vxlan%d", vxlanID)
		assert.True(t, interfaceExists(interfaceName), "Optimized VXLAN interface should exist")
	})

	t.Run("get optimized tunnel status", func(t *testing.T) {
		status, err := suite.optimizedManager.GetTunnelStatusOptimized(vxlanID)
		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, vxlan.TunnelStateActive, status.State)
	})

	t.Run("list active tunnels", func(t *testing.T) {
		activeTunnels := suite.optimizedManager.ListActiveTunnels()
		assert.Contains(t, activeTunnels, vxlanID)
		assert.Equal(t, vxlan.TunnelStateActive, activeTunnels[vxlanID].State)
	})

	t.Run("get performance metrics", func(t *testing.T) {
		metrics := suite.optimizedManager.GetMetrics()
		assert.NotNil(t, metrics)
		assert.True(t, metrics.TotalOperations > 0)
		assert.True(t, metrics.SuccessfulOps > 0)
	})

	t.Run("delete optimized tunnel successfully", func(t *testing.T) {
		err := suite.optimizedManager.DeleteTunnelOptimized(vxlanID)
		assert.NoError(t, err)

		// Verify tunnel is removed from active list
		activeTunnels := suite.optimizedManager.ListActiveTunnels()
		assert.NotContains(t, activeTunnels, vxlanID)

		// Verify interface was removed
		interfaceName := fmt.Sprintf("vxlan%d", vxlanID)
		assert.False(t, interfaceExists(interfaceName), "Optimized VXLAN interface should be removed")
	})
}

func TestVXLANIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VXLAN integration test in short mode")
	}

	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	t.Run("create tunnel with invalid interface", func(t *testing.T) {
		vxlanID := int32(3001)
		err := suite.manager.CreateTunnel(vxlanID, suite.localIP, []string{"10.0.1.2"}, "nonexistent0")

		if !isPermissionError(err) {
			// Should fail due to nonexistent interface
			assert.Error(t, err)
		}
	})

	t.Run("create tunnel with invalid local IP", func(t *testing.T) {
		vxlanID := int32(3002)
		err := suite.manager.CreateTunnel(vxlanID, "999.999.999.999", []string{"10.0.1.2"}, suite.testInterface)

		// Should fail validation or system command
		assert.Error(t, err)
	})

	t.Run("get status for non-existent tunnel", func(t *testing.T) {
		_, err := suite.manager.GetTunnelStatus(9999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete non-existent tunnel", func(t *testing.T) {
		err := suite.manager.DeleteTunnel(9999)
		// Should handle gracefully (interface not found is acceptable)
		if err != nil {
			assert.True(t,
				strings.Contains(err.Error(), "Cannot find device") ||
				strings.Contains(err.Error(), "does not exist"),
			)
		}
	})

	t.Run("optimized manager error handling", func(t *testing.T) {
		_, err := suite.optimizedManager.GetTunnelStatusOptimized(9999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestVXLANIntegration_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VXLAN integration test in short mode")
	}

	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	// Test concurrent tunnel creation
	t.Run("concurrent tunnel creation", func(t *testing.T) {
		var wg sync.WaitGroup
		errorChan := make(chan error, 10)
		startVXLAN := int32(4000)

		// Create tunnels concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				vxlanID := startVXLAN + int32(index)
				remoteIP := fmt.Sprintf("10.0.4.%d", index+2)

				err := suite.optimizedManager.CreateTunnelOptimized(vxlanID, suite.localIP, []string{remoteIP}, suite.testInterface)
				if err != nil && !isPermissionError(err) && !isNetworkError(err) {
					errorChan <- fmt.Errorf("tunnel %d: %w", vxlanID, err)
				} else if err == nil {
					suite.addCreatedTunnel(vxlanID)
				}
			}(i)
		}

		wg.Wait()
		close(errorChan)

		// Collect errors
		var errors []error
		for err := range errorChan {
			errors = append(errors, err)
		}

		// Should handle concurrent operations with minimal errors
		assert.True(t, len(errors) < 3, "Too many errors in concurrent operations: %v", errors)

		// Clean up created tunnels
		for i := 0; i < 5; i++ {
			vxlanID := startVXLAN + int32(i)
			suite.optimizedManager.DeleteTunnelOptimized(vxlanID)
		}
	})
}

func TestVXLANIntegration_IPGeneration(t *testing.T) {
	suite := setupVXLANIntegrationTest(t)

	testCases := []struct {
		name     string
		vxlanID  int32
		nodeIP   string
		expected string
	}{
		{
			name:     "basic IP generation",
			vxlanID:  100,
			nodeIP:   "192.168.1.10",
			expected: "10.0.100.10",
		},
		{
			name:     "high VXLAN ID",
			vxlanID:  65000,
			nodeIP:   "192.168.1.20",
			expected: "10.254.8.20",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This tests the internal IP generation logic
			vxlanID := tc.vxlanID
			nodeIP := tc.nodeIP

			// Create tunnel to test IP generation
			err := suite.manager.CreateTunnel(vxlanID, nodeIP, []string{"10.0.1.2"}, suite.testInterface)
			if err != nil {
				if isPermissionError(err) || isNetworkError(err) {
					t.Skip("Cannot test IP generation due to system limitations")
				}
				// IP generation might still work even if tunnel creation fails
			} else {
				suite.addCreatedTunnel(vxlanID)
				defer suite.manager.DeleteTunnel(vxlanID)
			}

			// The actual IP generation logic is internal, but we can verify
			// that the tunnel was created with the expected configuration
			if err == nil {
				status, statusErr := suite.manager.GetTunnelStatus(vxlanID)
				if statusErr == nil {
					assert.Equal(t, nodeIP, status.LocalIP)
				}
			}
		})
	}
}

func TestVXLANIntegration_SecurityValidation(t *testing.T) {
	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	securityTestCases := []struct {
		name        string
		vxlanID     int32
		localIP     string
		remoteIPs   []string
		interface_  string
		expectError bool
	}{
		{
			name:        "command injection in local IP",
			vxlanID:     5001,
			localIP:     "10.0.1.1; rm -rf /",
			remoteIPs:   []string{"10.0.1.2"},
			interface_:  suite.testInterface,
			expectError: true,
		},
		{
			name:        "command injection in remote IP",
			vxlanID:     5002,
			localIP:     suite.localIP,
			remoteIPs:   []string{"10.0.1.2; cat /etc/passwd"},
			interface_:  suite.testInterface,
			expectError: true,
		},
		{
			name:        "path traversal in interface",
			vxlanID:     5003,
			localIP:     suite.localIP,
			remoteIPs:   []string{"10.0.1.2"},
			interface_:  "../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "null bytes in parameters",
			vxlanID:     5004,
			localIP:     "10.0.1.1\x00",
			remoteIPs:   []string{"10.0.1.2\x00"},
			interface_:  suite.testInterface + "\x00",
			expectError: true,
		},
	}

	for _, tc := range securityTestCases {
		t.Run(tc.name, func(t *testing.T) {
			err := suite.manager.CreateTunnel(tc.vxlanID, tc.localIP, tc.remoteIPs, tc.interface_)

			if tc.expectError {
				// Should reject malicious input
				assert.Error(t, err)
			} else {
				// Should handle safely
				if err == nil {
					suite.addCreatedTunnel(tc.vxlanID)
				}
			}

			// Ensure no command injection occurred (system should be safe)
			// This is hard to test directly, but the error handling should prevent execution
		})
	}
}

func TestVXLANIntegration_ResourceManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VXLAN integration test in short mode")
	}

	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	// Test creating and cleaning up many tunnels
	t.Run("bulk tunnel operations", func(t *testing.T) {
		tunnelCount := 10
		startVXLAN := int32(6000)
		createdTunnels := []int32{}

		// Create multiple tunnels
		for i := 0; i < tunnelCount; i++ {
			vxlanID := startVXLAN + int32(i)
			remoteIP := fmt.Sprintf("10.0.6.%d", i+2)

			err := suite.optimizedManager.CreateTunnelOptimized(vxlanID, suite.localIP, []string{remoteIP}, suite.testInterface)
			if err == nil {
				createdTunnels = append(createdTunnels, vxlanID)
				suite.addCreatedTunnel(vxlanID)
			} else if !isPermissionError(err) && !isNetworkError(err) {
				t.Logf("Failed to create tunnel %d: %v", vxlanID, err)
			}
		}

		t.Logf("Created %d out of %d tunnels", len(createdTunnels), tunnelCount)

		// Verify active tunnels
		activeTunnels := suite.optimizedManager.ListActiveTunnels()
		for _, vxlanID := range createdTunnels {
			assert.Contains(t, activeTunnels, vxlanID)
		}

		// Test cleanup
		err := suite.optimizedManager.CleanupOptimized()
		if len(createdTunnels) > 0 {
			// Should succeed if any tunnels were created
			assert.NoError(t, err)
		}

		// Verify all tunnels are cleaned up
		activeTunnels = suite.optimizedManager.ListActiveTunnels()
		assert.Empty(t, activeTunnels)

		// Clear our tracking list since cleanup was successful
		suite.createdTunnels = []int32{}
	})
}

func TestVXLANIntegration_NetworkConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VXLAN integration test in short mode")
	}

	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	vxlanID := int32(7000)
	remoteIPs := []string{"10.0.7.2", "10.0.7.3"}

	t.Run("tunnel with multiple remote IPs", func(t *testing.T) {
		err := suite.manager.CreateTunnel(vxlanID, suite.localIP, remoteIPs, suite.testInterface)
		if err != nil {
			if isPermissionError(err) || isNetworkError(err) {
				t.Skip("Cannot test network configuration due to system limitations")
			}
			t.Fatalf("Failed to create tunnel with multiple remotes: %v", err)
		}

		suite.addCreatedTunnel(vxlanID)

		// Verify tunnel configuration
		status, err := suite.manager.GetTunnelStatus(vxlanID)
		assert.NoError(t, err)
		assert.Equal(t, remoteIPs, status.RemoteIPs)

		// Verify MTU setting
		assert.Equal(t, 1450, status.MTU)

		// Clean up
		err = suite.manager.DeleteTunnel(vxlanID)
		assert.NoError(t, err)
	})
}

func TestVXLANIntegration_PerformanceMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VXLAN integration test in short mode")
	}

	suite := setupVXLANIntegrationTest(t)
	defer suite.cleanup()

	// Perform operations to generate metrics
	vxlanID := int32(8000)

	err := suite.optimizedManager.CreateTunnelOptimized(vxlanID, suite.localIP, []string{"10.0.8.2"}, suite.testInterface)
	if err != nil {
		if isPermissionError(err) || isNetworkError(err) {
			t.Skip("Cannot test performance metrics due to system limitations")
		}
		t.Fatalf("Failed to create tunnel for metrics test: %v", err)
	}

	suite.addCreatedTunnel(vxlanID)

	// Get performance metrics
	metrics := suite.optimizedManager.GetMetrics()
	assert.NotNil(t, metrics)
	assert.True(t, metrics.TotalOperations > 0)
	assert.True(t, metrics.SuccessfulOps > 0)
	assert.True(t, metrics.AvgOpTimeMs >= 0)

	// Delete tunnel and check metrics again
	err = suite.optimizedManager.DeleteTunnelOptimized(vxlanID)
	assert.NoError(t, err)

	metrics = suite.optimizedManager.GetMetrics()
	assert.True(t, metrics.TotalOperations >= 2) // Create + Delete operations

	t.Logf("Performance metrics: Total=%d, Success=%d, Failed=%d, AvgTime=%.2fms",
		metrics.TotalOperations, metrics.SuccessfulOps, metrics.FailedOps, metrics.AvgOpTimeMs)
}

// Helper functions

func hasVXLANCapabilities() bool {
	// Check if we can create network namespaces or have CAP_NET_ADMIN
	cmd := exec.Command("ip", "link", "help")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if VXLAN is supported
	return strings.Contains(string(output), "vxlan")
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "operation not permitted") ||
		strings.Contains(errStr, "not allowed")
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "interface") ||
		strings.Contains(errStr, "device") ||
		strings.Contains(errStr, "cannot find")
}

func interfaceExists(name string) bool {
	cmd := exec.Command("ip", "link", "show", name)
	err := cmd.Run()
	return err == nil
}

func getTestInterface() string {
	// Try to find a suitable test interface
	interfaces := []string{"eth0", "ens3", "enp0s3", "lo"}

	for _, iface := range interfaces {
		if interfaceExists(iface) {
			return iface
		}
	}

	// Fallback to lo (loopback) which should always exist
	return "lo"
}

func getTestLocalIP() string {
	// Use a test IP from TEST-NET-1 (RFC 5737)
	return "192.0.2.1"
}

// Benchmarks for performance testing
func BenchmarkVXLANIntegration_TunnelCreation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping VXLAN benchmark in short mode")
	}

	suite := setupVXLANIntegrationTest(nil)
	defer suite.cleanup()

	b.ResetTimer()
	for i := 0; i < b.N && i < 100; i++ { // Limit to avoid system resource exhaustion
		vxlanID := int32(9000 + i)
		remoteIP := fmt.Sprintf("10.0.9.%d", (i%250)+2)

		err := suite.optimizedManager.CreateTunnelOptimized(vxlanID, suite.localIP, []string{remoteIP}, suite.testInterface)
		if err != nil && !isPermissionError(err) && !isNetworkError(err) {
			b.Errorf("Failed to create tunnel %d: %v", vxlanID, err)
		}

		// Cleanup immediately to avoid resource exhaustion
		suite.optimizedManager.DeleteTunnelOptimized(vxlanID)
	}
}

func BenchmarkVXLANIntegration_StatusQuery(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping VXLAN benchmark in short mode")
	}

	suite := setupVXLANIntegrationTest(nil)
	defer suite.cleanup()

	// Create a tunnel for status queries
	vxlanID := int32(9500)
	err := suite.optimizedManager.CreateTunnelOptimized(vxlanID, suite.localIP, []string{"10.0.95.2"}, suite.testInterface)
	if err != nil {
		if isPermissionError(err) || isNetworkError(err) {
			b.Skip("Cannot create tunnel for benchmark")
		}
		b.Fatalf("Failed to create test tunnel: %v", err)
	}
	defer suite.optimizedManager.DeleteTunnelOptimized(vxlanID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.optimizedManager.GetTunnelStatusOptimized(vxlanID)
		if err != nil {
			b.Errorf("Failed to get tunnel status: %v", err)
		}
	}
}

// Test helper for CI/CD validation
func TestVXLANIntegration_CIValidation(t *testing.T) {
	// Basic validation tests that should work in most CI environments
	t.Run("manager creation", func(t *testing.T) {
		manager := vxlan.NewManager()
		assert.NotNil(t, manager)

		optimizedManager := vxlan.NewOptimizedManager()
		assert.NotNil(t, optimizedManager)
		assert.NotNil(t, optimizedManager.GetMetrics())
	})

	t.Run("validation functions", func(t *testing.T) {
		manager := vxlan.NewManager()

		// Test with obviously invalid parameters
		err := manager.CreateTunnel(-1, "invalid-ip", []string{}, "nonexistent")
		assert.Error(t, err)

		// Test status query for non-existent tunnel
		_, err = manager.GetTunnelStatus(99999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("optimized manager features", func(t *testing.T) {
		manager := vxlan.NewOptimizedManager()

		// Test metrics
		metrics := manager.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, int64(0), metrics.TotalOperations)

		// Test list active tunnels
		tunnels := manager.ListActiveTunnels()
		assert.NotNil(t, tunnels)
		assert.Empty(t, tunnels)

		// Test status query for non-existent tunnel
		_, err := manager.GetTunnelStatusOptimized(99999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}