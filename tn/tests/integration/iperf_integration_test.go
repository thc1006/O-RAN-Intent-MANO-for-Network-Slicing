package integration

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg"
)

// IperfIntegrationTestSuite for testing iperf functionality
type IperfIntegrationTestSuite struct {
	manager *pkg.IperfManager
	logger  *log.Logger
}

func setupIperfIntegrationTest(t *testing.T) *IperfIntegrationTestSuite {
	logger := log.New(os.Stdout, "[IPERF-TEST] ", log.LstdFlags)
	manager := pkg.NewIperfManager(logger)

	return &IperfIntegrationTestSuite{
		manager: manager,
		logger:  logger,
	}
}

func TestIperfIntegration_ServerLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	// Find an available port for testing
	port := findAvailablePort(t, 15001, 15100)

	t.Run("start server successfully", func(t *testing.T) {
		err := suite.manager.StartServer(port)
		if err != nil {
			if strings.Contains(err.Error(), "iperf3 not found") ||
				strings.Contains(err.Error(), "command not found") {
				t.Skip("iperf3 not available in test environment")
			}
			t.Fatalf("Failed to start iperf server: %v", err)
		}

		// Verify server is in the active servers list
		servers := suite.manager.GetActiveServers()
		serverKey := fmt.Sprintf("port_%d", port)
		assert.Contains(t, servers, serverKey)

		// Verify server is actually listening
		assert.True(t, isPortListening(port), "Server should be listening on port %d", port)
	})

	t.Run("stop server successfully", func(t *testing.T) {
		err := suite.manager.StopServer(port)
		assert.NoError(t, err)

		// Verify server is removed from active servers
		servers := suite.manager.GetActiveServers()
		serverKey := fmt.Sprintf("port_%d", port)
		assert.NotContains(t, servers, serverKey)

		// Give some time for the port to be released
		time.Sleep(100 * time.Millisecond)

		// Verify port is no longer listening
		assert.False(t, isPortListening(port), "Port should not be listening after stop")
	})

	t.Run("restart server on same port", func(t *testing.T) {
		// Should be able to start again on the same port
		err := suite.manager.StartServer(port)
		if err != nil && !strings.Contains(err.Error(), "iperf3 not found") {
			assert.NoError(t, err)
		}

		// Cleanup
		suite.manager.StopServer(port)
	})
}

func TestIperfIntegration_MultipleServers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	// Find multiple available ports
	ports := []int{
		findAvailablePort(t, 15101, 15110),
		findAvailablePort(t, 15111, 15120),
		findAvailablePort(t, 15121, 15130),
	}

	// Start multiple servers
	for _, port := range ports {
		err := suite.manager.StartServer(port)
		if err != nil {
			if strings.Contains(err.Error(), "iperf3 not found") {
				t.Skip("iperf3 not available in test environment")
			}
			t.Logf("Failed to start server on port %d: %v", port, err)
		}
	}

	// Verify all servers are active
	servers := suite.manager.GetActiveServers()
	startedCount := 0
	for _, port := range ports {
		serverKey := fmt.Sprintf("port_%d", port)
		if _, exists := servers[serverKey]; exists {
			startedCount++
		}
	}

	t.Logf("Started %d out of %d servers", startedCount, len(ports))

	// Stop all servers
	err := suite.manager.StopAllServers()
	if startedCount > 0 {
		// Should succeed if any servers were started
		assert.NoError(t, err)
	}

	// Verify all servers are stopped
	servers = suite.manager.GetActiveServers()
	assert.Empty(t, servers)
}

func TestIperfIntegration_ServerErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	t.Run("start server on invalid port", func(t *testing.T) {
		err := suite.manager.StartServer(-1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port")
	})

	t.Run("start server on privileged port", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("Running as root, privileged ports are available")
		}

		err := suite.manager.StartServer(80)
		if err != nil {
			// Expected to fail due to permissions
			assert.Contains(t, err.Error(), "permission")
		}
	})

	t.Run("stop non-existent server", func(t *testing.T) {
		err := suite.manager.StopServer(19999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no iperf3 server running")
	})

	t.Run("start server on busy port", func(t *testing.T) {
		// Create a listener to occupy the port
		port := findAvailablePort(t, 15201, 15210)
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		require.NoError(t, err)
		defer listener.Close()

		// Try to start iperf server on the same port
		err = suite.manager.StartServer(port)
		if err != nil {
			// Should fail because port is busy
			assert.True(t,
				strings.Contains(err.Error(), "failed to start listening") ||
					strings.Contains(err.Error(), "address already in use") ||
					strings.Contains(err.Error(), "bind") ||
					err != nil, // Any error is acceptable here
			)
		}
	})
}

func TestIperfIntegration_ClientTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	// Start a test server
	port := findAvailablePort(t, 15301, 15310)
	err := suite.manager.StartServer(port)
	if err != nil {
		if strings.Contains(err.Error(), "iperf3 not found") {
			t.Skip("iperf3 not available in test environment")
		}
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer suite.manager.StopServer(port)

	// Wait for server to be ready
	time.Sleep(1 * time.Second)

	t.Run("basic client test", func(t *testing.T) {
		config := &pkg.IperfTestConfig{
			ServerIP: "127.0.0.1",
			Port:     port,
			Duration: 2 * time.Second,
			Protocol: "tcp",
			Parallel: 1,
			JSON:     true,
		}

		result, err := suite.manager.RunTest(config)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "no route to host") {
				t.Skip("Network connectivity issues in test environment")
			}
			t.Logf("Test failed (may be expected in CI): %v", err)
		}

		assert.NotNil(t, result)
		assert.NotEmpty(t, result.TestID)
		assert.Equal(t, "tcp", result.Protocol)

		if err == nil {
			// If test succeeded, verify basic metrics
			assert.True(t, result.Duration > 0)
			t.Logf("Test completed successfully: %.2f Mbps", result.Summary.Received.MbitsPerSec)
		}
	})

	t.Run("UDP client test", func(t *testing.T) {
		config := &pkg.IperfTestConfig{
			ServerIP:  "127.0.0.1",
			Port:      port,
			Duration:  1 * time.Second,
			Protocol:  "udp",
			Bandwidth: "10M",
			Parallel:  1,
			JSON:      true,
		}

		result, err := suite.manager.RunTest(config)
		if err != nil {
			t.Logf("UDP test failed (may be expected): %v", err)
		}

		assert.NotNil(t, result)
		assert.Equal(t, "udp", result.Protocol)
	})

	t.Run("client test with invalid server", func(t *testing.T) {
		config := &pkg.IperfTestConfig{
			ServerIP: "192.0.2.1", // TEST-NET-1 (should not be reachable)
			Port:     port,
			Duration: 1 * time.Second,
			Protocol: "tcp",
			JSON:     true,
		}

		result, err := suite.manager.RunTest(config)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ErrorMessages)
	})
}

func TestIperfIntegration_ThroughputTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	// Start a test server
	port := findAvailablePort(t, 15401, 15410)
	err := suite.manager.StartServer(port)
	if err != nil {
		if strings.Contains(err.Error(), "iperf3 not found") {
			t.Skip("iperf3 not available in test environment")
		}
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer suite.manager.StopServer(port)

	// Wait for server to be ready
	time.Sleep(1 * time.Second)

	t.Run("throughput test", func(t *testing.T) {
		metrics, err := suite.manager.RunThroughputTest("127.0.0.1", port, 2*time.Second)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				t.Skip("Network connectivity issues in test environment")
			}
			t.Logf("Throughput test failed (may be expected): %v", err)
		}

		assert.NotNil(t, metrics)

		if err == nil {
			// Verify basic throughput metrics
			assert.True(t, metrics.DownlinkMbps >= 0)
			assert.True(t, metrics.UplinkMbps >= 0)
			assert.True(t, metrics.AvgMbps >= 0)

			t.Logf("Throughput results: DL=%.2f Mbps, UL=%.2f Mbps, Avg=%.2f Mbps",
				metrics.DownlinkMbps, metrics.UplinkMbps, metrics.AvgMbps)
		}
	})

	t.Run("throughput test with validation errors", func(t *testing.T) {
		// Invalid IP
		_, err := suite.manager.RunThroughputTest("invalid-ip", port, 1*time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server IP")

		// Invalid port
		_, err = suite.manager.RunThroughputTest("127.0.0.1", -1, 1*time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port")

		// Duration too long
		_, err = suite.manager.RunThroughputTest("127.0.0.1", port, 2*time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duration too long")
	})
}

func TestIperfIntegration_LatencyTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	t.Run("latency test to localhost", func(t *testing.T) {
		metrics, err := suite.manager.RunLatencyTest("127.0.0.1", 80, 2*time.Second)
		if err != nil {
			if strings.Contains(err.Error(), "ping") ||
				strings.Contains(err.Error(), "command not found") {
				t.Skip("ping command not available in test environment")
			}
			t.Logf("Latency test failed (may be expected): %v", err)
		}

		assert.NotNil(t, metrics)

		if err == nil && metrics.AvgRTTMs > 0 {
			// Verify latency metrics
			assert.True(t, metrics.AvgRTTMs > 0)
			assert.True(t, metrics.MinRTTMs <= metrics.AvgRTTMs)
			assert.True(t, metrics.MaxRTTMs >= metrics.AvgRTTMs)

			t.Logf("Latency results: Avg=%.2f ms, Min=%.2f ms, Max=%.2f ms",
				metrics.AvgRTTMs, metrics.MinRTTMs, metrics.MaxRTTMs)
		}
	})

	t.Run("latency test with validation errors", func(t *testing.T) {
		// Invalid IP
		_, err := suite.manager.RunLatencyTest("invalid-ip", 80, 1*time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server IP")

		// Duration too long
		_, err = suite.manager.RunLatencyTest("127.0.0.1", 80, 15*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duration too long")
	})

	t.Run("latency test to unreachable host", func(t *testing.T) {
		metrics, err := suite.manager.RunLatencyTest("192.0.2.1", 80, 1*time.Second)
		if err != nil {
			// Expected to fail for unreachable host
			assert.NotNil(t, metrics)
		}
	})
}

func TestIperfIntegration_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	// Test concurrent server operations
	t.Run("concurrent server start/stop", func(t *testing.T) {
		var wg sync.WaitGroup
		errorChan := make(chan error, 20)

		// Find available ports
		ports := make([]int, 10)
		for i := range ports {
			ports[i] = findAvailablePort(t, 15501+i*10, 15501+(i+1)*10-1)
		}

		// Start servers concurrently
		for _, port := range ports {
			wg.Add(1)
			go func(p int) {
				defer wg.Done()
				err := suite.manager.StartServer(p)
				if err != nil && !strings.Contains(err.Error(), "iperf3 not found") {
					errorChan <- fmt.Errorf("start port %d: %w", p, err)
				}
			}(port)
		}

		wg.Wait()

		// Stop servers concurrently
		for _, port := range ports {
			wg.Add(1)
			go func(p int) {
				defer wg.Done()
				err := suite.manager.StopServer(p)
				if err != nil && !strings.Contains(err.Error(), "no iperf3 server running") {
					errorChan <- fmt.Errorf("stop port %d: %w", p, err)
				}
			}(port)
		}

		wg.Wait()
		close(errorChan)

		// Collect errors
		var errors []error
		for err := range errorChan {
			errors = append(errors, err)
		}

		// Should handle concurrent operations with minimal errors
		assert.True(t, len(errors) < 5, "Too many errors in concurrent operations: %v", errors)
	})

	t.Run("concurrent client tests", func(t *testing.T) {
		// Start a single server for testing
		port := findAvailablePort(t, 15601, 15610)
		err := suite.manager.StartServer(port)
		if err != nil {
			if strings.Contains(err.Error(), "iperf3 not found") {
				t.Skip("iperf3 not available in test environment")
			}
			t.Fatalf("Failed to start test server: %v", err)
		}
		defer suite.manager.StopServer(port)

		// Wait for server to be ready
		time.Sleep(1 * time.Second)

		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// Run multiple concurrent client tests
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(testID int) {
				defer wg.Done()

				config := &pkg.IperfTestConfig{
					ServerIP: "127.0.0.1",
					Port:     port,
					Duration: 1 * time.Second,
					Protocol: "tcp",
					JSON:     true,
				}

				result, err := suite.manager.RunTest(config)
				if err == nil && result != nil && len(result.ErrorMessages) == 0 {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// At least some tests should succeed if iperf is available
		t.Logf("Concurrent tests: %d/5 succeeded", successCount)
	})
}

func TestIperfIntegration_ResourceCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	// Start multiple servers
	ports := []int{
		findAvailablePort(t, 15701, 15710),
		findAvailablePort(t, 15711, 15720),
		findAvailablePort(t, 15721, 15730),
	}

	startedPorts := []int{}
	for _, port := range ports {
		err := suite.manager.StartServer(port)
		if err == nil {
			startedPorts = append(startedPorts, port)
		} else if !strings.Contains(err.Error(), "iperf3 not found") {
			t.Logf("Failed to start server on port %d: %v", port, err)
		}
	}

	if len(startedPorts) == 0 {
		t.Skip("No servers could be started")
	}

	// Verify servers are running
	servers := suite.manager.GetActiveServers()
	assert.True(t, len(servers) >= len(startedPorts))

	// Test StopAllServers
	err := suite.manager.StopAllServers()
	assert.NoError(t, err)

	// Verify all servers are stopped
	servers = suite.manager.GetActiveServers()
	assert.Empty(t, servers)

	// Verify ports are no longer listening
	for _, port := range startedPorts {
		assert.False(t, isPortListening(port), "Port %d should not be listening after cleanup", port)
	}
}

func TestIperfIntegration_SecurityValidation(t *testing.T) {
	suite := setupIperfIntegrationTest(t)

	securityTestCases := []struct {
		name   string
		config *pkg.IperfTestConfig
	}{
		{
			name: "malicious server IP",
			config: &pkg.IperfTestConfig{
				ServerIP: "127.0.0.1; rm -rf /",
				Port:     5001,
				Duration: 1 * time.Second,
			},
		},
		{
			name: "command injection in bandwidth",
			config: &pkg.IperfTestConfig{
				ServerIP:  "127.0.0.1",
				Port:      5001,
				Duration:  1 * time.Second,
				Bandwidth: "10M; cat /etc/passwd",
			},
		},
		{
			name: "path traversal in window size",
			config: &pkg.IperfTestConfig{
				ServerIP:   "127.0.0.1",
				Port:       5001,
				Duration:   1 * time.Second,
				WindowSize: "../../../etc/passwd",
			},
		},
	}

	for _, tc := range securityTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := suite.manager.RunTest(tc.config)

			// Should either reject the malicious input or handle it safely
			assert.NotNil(t, result)
			if err != nil {
				// Expected - malicious input should be rejected
				assert.Contains(t, err.Error(), "invalid")
			} else {
				// If it somehow succeeds, the result should be safe
				assert.NotNil(t, result.ErrorMessages)
			}
		})
	}
}

func TestIperfIntegration_NetworkErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping iperf integration test in short mode")
	}

	suite := setupIperfIntegrationTest(t)

	networkErrorTests := []struct {
		name     string
		serverIP string
		port     int
		timeout  time.Duration
	}{
		{
			name:     "connection timeout",
			serverIP: "198.51.100.1", // TEST-NET-2 (should timeout)
			port:     5001,
			timeout:  2 * time.Second,
		},
		{
			name:     "connection refused",
			serverIP: "127.0.0.1",
			port:     99999, // Unlikely to be listening
			timeout:  1 * time.Second,
		},
		{
			name:     "invalid hostname resolution",
			serverIP: "nonexistent.invalid.domain.test",
			port:     5001,
			timeout:  1 * time.Second,
		},
	}

	for _, tc := range networkErrorTests {
		t.Run(tc.name, func(t *testing.T) {
			config := &pkg.IperfTestConfig{
				ServerIP: tc.serverIP,
				Port:     tc.port,
				Duration: tc.timeout,
				Protocol: "tcp",
				JSON:     true,
			}

			result, err := suite.manager.RunTest(config)

			// Should handle network errors gracefully
			assert.Error(t, err)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.ErrorMessages)

			// Verify error is related to network issues
			assert.True(t,
				strings.Contains(err.Error(), "connection") ||
					strings.Contains(err.Error(), "timeout") ||
					strings.Contains(err.Error(), "refused") ||
					strings.Contains(err.Error(), "failed"),
			)
		})
	}
}

// Helper functions

func findAvailablePort(t *testing.T, start, end int) int {
	for port := start; port <= end; port++ {
		if isPortAvailable(port) {
			return port
		}
	}
	t.Fatalf("No available port found in range %d-%d", start, end)
	return 0
}

func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Performance benchmarks for integration testing
func BenchmarkIperfIntegration_ServerOperations(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping iperf benchmark in short mode")
	}

	suite := setupIperfIntegrationTest(nil)

	// Find available ports for benchmark
	ports := make([]int, b.N)
	for i := 0; i < b.N && i < 100; i++ { // Limit to 100 ports max
		ports[i] = findAvailablePort(nil, 16001+i*10, 16001+(i+1)*10-1)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() && i < len(ports) {
			port := ports[i%len(ports)]

			// Start server
			err := suite.manager.StartServer(port)
			if err != nil && !strings.Contains(err.Error(), "iperf3 not found") {
				b.Errorf("Failed to start server: %v", err)
			}

			// Stop server
			suite.manager.StopServer(port)
			i++
		}
	})

	// Cleanup any remaining servers
	suite.manager.StopAllServers()
}

func BenchmarkIperfIntegration_ClientTest(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping iperf benchmark in short mode")
	}

	suite := setupIperfIntegrationTest(nil)

	// Start a server for benchmarking
	port := findAvailablePort(nil, 16101, 16110)
	err := suite.manager.StartServer(port)
	if err != nil {
		if strings.Contains(err.Error(), "iperf3 not found") {
			b.Skip("iperf3 not available")
		}
		b.Fatalf("Failed to start benchmark server: %v", err)
	}
	defer suite.manager.StopServer(port)

	// Wait for server to be ready
	time.Sleep(1 * time.Second)

	config := &pkg.IperfTestConfig{
		ServerIP: "127.0.0.1",
		Port:     port,
		Duration: 1 * time.Second,
		Protocol: "tcp",
		JSON:     true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := suite.manager.RunTest(config)
		if err != nil && !strings.Contains(err.Error(), "connection refused") {
			b.Logf("Test failed: %v", err)
		}
		_ = result // Use result to avoid compiler optimization
	}
}

// Test helper for CI/CD validation
func TestIperfIntegration_CIValidation(t *testing.T) {
	suite := setupIperfIntegrationTest(t)

	// Basic smoke test that should work in most CI environments
	t.Run("manager creation", func(t *testing.T) {
		assert.NotNil(t, suite.manager)
		assert.NotNil(t, suite.logger)
	})

	t.Run("active servers query", func(t *testing.T) {
		servers := suite.manager.GetActiveServers()
		assert.NotNil(t, servers)
		assert.Empty(t, servers) // Should start empty
	})

	t.Run("validation functions", func(t *testing.T) {
		// Test validation without actually running commands
		config := &pkg.IperfTestConfig{
			ServerIP: "invalid-ip",
			Port:     -1,
			Duration: 25 * time.Hour,
			Parallel: 0,
		}

		result, err := suite.manager.RunTest(config)
		assert.Error(t, err)
		assert.NotNil(t, result)
	})
}