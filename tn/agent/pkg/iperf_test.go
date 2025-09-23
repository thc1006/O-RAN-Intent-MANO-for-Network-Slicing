package pkg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// MockSecurityExecutor for testing secure execution calls
type MockSecurityExecutor struct {
	mock.Mock
}

func (m *MockSecurityExecutor) SecureExecute(ctx context.Context, command string, args ...string) ([]byte, error) {
	allArgs := append([]string{command}, args...)
	mockArgs := m.Called(ctx, allArgs)
	return mockArgs.Get(0).([]byte), mockArgs.Error(1)
}

func (m *MockSecurityExecutor) SecureExecuteWithValidation(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
	allArgs := append([]string{command}, args...)
	mockArgs := m.Called(ctx, allArgs, validator)
	return mockArgs.Get(0).([]byte), mockArgs.Error(1)
}

func TestNewIperfManager(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	require.NotNil(t, manager)
	assert.Equal(t, logger, manager.logger)
	assert.NotNil(t, manager.servers)
	assert.Empty(t, manager.servers)
}

func TestIperfManager_StartServer_ValidationErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name        string
		port        int
		expectError string
	}{
		{
			name:        "port too low",
			port:        0,
			expectError: "invalid port",
		},
		{
			name:        "port too high",
			port:        65536,
			expectError: "invalid port",
		},
		{
			name:        "negative port",
			port:        -1,
			expectError: "invalid port",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := manager.StartServer(tc.port)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
		})
	}
}

func TestIperfManager_StartServer_ExecutionErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	// Test execution failure
	t.Run("command execution fails", func(t *testing.T) {
		// Mock security.SecureExecute to return an error
		originalSecureExecute := security.SecureExecute
		defer func() {
			security.SecureExecute = originalSecureExecute
		}()

		security.SecureExecute = func(ctx context.Context, command string, args ...string) ([]byte, error) {
			if command == "iperf3" {
				return nil, errors.New("iperf3 not found")
			}
			return nil, errors.New("unexpected command")
		}

		err := manager.StartServer(5001)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start iperf3 server")
	})

	// Test argument validation failure
	t.Run("argument validation fails", func(t *testing.T) {
		originalValidateIPerfArgs := security.ValidateIPerfArgs
		defer func() {
			security.ValidateIPerfArgs = originalValidateIPerfArgs
		}()

		security.ValidateIPerfArgs = func(args []string) error {
			return errors.New("invalid iperf arguments")
		}

		err := manager.StartServer(5001)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "iperf3 argument validation failed")
	})

	// Test command argument validation failure
	t.Run("command argument validation fails", func(t *testing.T) {
		originalValidateCommandArgument := security.ValidateCommandArgument
		defer func() {
			security.ValidateCommandArgument = originalValidateCommandArgument
		}()

		security.ValidateCommandArgument = func(arg string) error {
			if arg == "5001" {
				return errors.New("invalid port argument")
			}
			return nil
		}

		err := manager.StartServer(5001)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port argument")
	})

	// Test server not listening after start
	t.Run("server not listening after start", func(t *testing.T) {
		originalSecureExecute := security.SecureExecute
		defer func() {
			security.SecureExecute = originalSecureExecute
		}()

		// Mock successful start but server not listening
		security.SecureExecute = func(ctx context.Context, command string, args ...string) ([]byte, error) {
			if command == "iperf3" {
				return []byte("started"), nil
			}
			if command == "pgrep" {
				return nil, errors.New("no process found")
			}
			if command == "pkill" {
				return []byte("killed"), nil
			}
			return nil, errors.New("unexpected command")
		}

		err := manager.StartServer(5001)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "iperf3 server failed to start listening")
	})
}

func TestIperfManager_StartServer_DuplicateServer(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	// Mock a server that's already listening
	originalSecureExecute := security.SecureExecute
	defer func() {
		security.SecureExecute = originalSecureExecute
	}()

	callCount := 0
	security.SecureExecute = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		callCount++
		if command == "iperf3" {
			return []byte("started"), nil
		}
		if command == "pgrep" {
			return []byte("12345"), nil
		}
		return []byte("ok"), nil
	}

	// First start should succeed
	err := manager.StartServer(5001)
	assert.NoError(t, err)

	// Reset call count
	callCount = 0

	// Second start on same port should return immediately (already running)
	err = manager.StartServer(5001)
	assert.NoError(t, err)

	// Should not execute commands if server already exists and is listening
	assert.Equal(t, 0, callCount)
}

func TestIperfManager_StopServer_ValidationErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name        string
		port        int
		expectError string
	}{
		{
			name:        "port too low",
			port:        0,
			expectError: "invalid port",
		},
		{
			name:        "port too high",
			port:        65536,
			expectError: "invalid port",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := manager.StopServer(tc.port)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
		})
	}
}

func TestIperfManager_StopServer_ServerNotFound(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	err := manager.StopServer(5001)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no iperf3 server running on port 5001")
}

func TestIperfManager_StopServer_SecurityValidationErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	// Add a mock server
	server := &IperfServer{
		Port:    5001,
		PID:     12345,
		Started: time.Now(),
		Context: context.Background(),
		Cancel:  func() {},
	}
	manager.servers["port_5001"] = server

	testCases := []struct {
		name        string
		setupMock   func()
		expectError string
	}{
		{
			name: "command argument validation fails",
			setupMock: func() {
				originalValidateCommandArgument := security.ValidateCommandArgument
				security.ValidateCommandArgument = func(arg string) error {
					if arg == "5001" {
						return errors.New("invalid port argument")
					}
					return originalValidateCommandArgument(arg)
				}
			},
			expectError: "invalid port argument",
		},
		{
			name: "port out of range validation",
			setupMock: func() {
				// This case is handled by changing the server port to invalid range
				server.Port = 70000
			},
			expectError: "port out of valid range",
		},
		{
			name: "safe process pattern fails",
			setupMock: func() {
				originalCreateSafeProcessPattern := security.CreateSafeProcessPattern
				security.CreateSafeProcessPattern = func(process, flag, value string) string {
					return "" // Force fallback
				}
				originalIsValidPortString := security.IsValidPortString
				security.IsValidPortString = func(port string) bool {
					return false // Force validation failure
				}
			},
			expectError: "invalid port format for process killing",
		},
		{
			name: "pkill pattern validation fails",
			setupMock: func() {
				originalValidatePkillPattern := security.ValidatePkillPattern
				security.ValidatePkillPattern = func(pattern string) error {
					return errors.New("unsafe pkill pattern")
				}
			},
			expectError: "unsafe pkill pattern",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset server state
			server.Port = 5001
			manager.servers["port_5001"] = server

			// Setup mocks
			tc.setupMock()

			err := manager.StopServer(5001)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)

			// Cleanup - restore functions (simplified for test)
			security.ValidateCommandArgument = func(arg string) error { return nil }
			security.CreateSafeProcessPattern = func(process, flag, value string) string { return "pattern" }
			security.IsValidPortString = func(port string) bool { return true }
			security.ValidatePkillPattern = func(pattern string) error { return nil }
		})
	}
}

func TestIperfManager_RunTest_ValidationErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name        string
		config      *IperfTestConfig
		expectError string
	}{
		{
			name: "invalid server IP",
			config: &IperfTestConfig{
				ServerIP: "invalid_ip",
				Port:     5001,
			},
			expectError: "invalid server IP",
		},
		{
			name: "invalid port",
			config: &IperfTestConfig{
				ServerIP: "10.0.1.1",
				Port:     0,
			},
			expectError: "invalid port",
		},
		{
			name: "duration too long",
			config: &IperfTestConfig{
				ServerIP: "10.0.1.1",
				Port:     5001,
				Duration: 2 * time.Hour,
			},
			expectError: "duration too long",
		},
		{
			name: "invalid parallel streams - too low",
			config: &IperfTestConfig{
				ServerIP: "10.0.1.1",
				Port:     5001,
				Parallel: 0,
			},
			expectError: "invalid parallel streams",
		},
		{
			name: "invalid parallel streams - too high",
			config: &IperfTestConfig{
				ServerIP: "10.0.1.1",
				Port:     5001,
				Parallel: 200,
			},
			expectError: "invalid parallel streams",
		},
		{
			name: "invalid bandwidth",
			config: &IperfTestConfig{
				ServerIP:  "10.0.1.1",
				Port:      5001,
				Bandwidth: "invalid",
			},
			expectError: "invalid bandwidth",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks for validation functions
			originalValidateIPAddress := security.ValidateIPAddress
			originalValidatePort := security.ValidatePort
			originalValidateBandwidth := security.ValidateBandwidth

			security.ValidateIPAddress = func(ip string) error {
				if ip == "invalid_ip" {
					return errors.New("invalid IP")
				}
				return nil
			}
			security.ValidatePort = func(port int) error {
				if port <= 0 || port > 65535 {
					return errors.New("invalid port")
				}
				return nil
			}
			security.ValidateBandwidth = func(bw string) error {
				if bw == "invalid" {
					return errors.New("invalid bandwidth format")
				}
				return nil
			}

			defer func() {
				security.ValidateIPAddress = originalValidateIPAddress
				security.ValidatePort = originalValidatePort
				security.ValidateBandwidth = originalValidateBandwidth
			}()

			result, err := manager.RunTest(tc.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
			if result != nil {
				assert.NotEmpty(t, result.ErrorMessages)
			}
		})
	}
}

func TestIperfManager_RunTest_ExecutionErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	config := &IperfTestConfig{
		ServerIP: "10.0.1.1",
		Port:     5001,
		Duration: 10 * time.Second,
		Protocol: "tcp",
		Parallel: 1,
		JSON:     true,
	}

	testCases := []struct {
		name        string
		setupMock   func()
		expectError string
	}{
		{
			name: "command argument validation fails",
			setupMock: func() {
				originalValidateCommandArgument := security.ValidateCommandArgument
				security.ValidateCommandArgument = func(arg string) error {
					if arg == "10.0.1.1" {
						return errors.New("invalid argument")
					}
					return nil
				}
			},
			expectError: "invalid iperf3 argument",
		},
		{
			name: "secure execute fails",
			setupMock: func() {
				originalSecureExecuteWithValidation := security.SecureExecuteWithValidation
				security.SecureExecuteWithValidation = func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
					return nil, errors.New("command execution failed")
				}
			},
			expectError: "iperf3 test failed",
		},
		{
			name: "window size validation fails",
			setupMock: func() {
				config.WindowSize = "invalid_size"
				originalValidateCommandArgument := security.ValidateCommandArgument
				security.ValidateCommandArgument = func(arg string) error {
					if arg == "invalid_size" {
						return errors.New("invalid window size")
					}
					return nil
				}
			},
			expectError: "invalid window size",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup base validation mocks
			originalValidateIPAddress := security.ValidateIPAddress
			originalValidatePort := security.ValidatePort
			originalValidateBandwidth := security.ValidateBandwidth
			originalValidateCommandArgument := security.ValidateCommandArgument
			originalSecureExecuteWithValidation := security.SecureExecuteWithValidation

			security.ValidateIPAddress = func(ip string) error { return nil }
			security.ValidatePort = func(port int) error { return nil }
			security.ValidateBandwidth = func(bw string) error { return nil }
			security.ValidateCommandArgument = func(arg string) error { return nil }
			security.SecureExecuteWithValidation = func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
				return []byte(`{"end": {"sum_received": {"bits_per_second": 1000000}}}`), nil
			}

			// Apply test-specific mock
			tc.setupMock()

			defer func() {
				security.ValidateIPAddress = originalValidateIPAddress
				security.ValidatePort = originalValidatePort
				security.ValidateBandwidth = originalValidateBandwidth
				security.ValidateCommandArgument = originalValidateCommandArgument
				security.SecureExecuteWithValidation = originalSecureExecuteWithValidation
				// Reset config
				config.WindowSize = ""
			}()

			result, err := manager.RunTest(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
			assert.NotNil(t, result)
		})
	}
}

func TestIperfManager_RunThroughputTest_ValidationErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name        string
		serverIP    string
		port        int
		duration    time.Duration
		expectError string
	}{
		{
			name:        "invalid server IP",
			serverIP:    "invalid",
			port:        5001,
			duration:    10 * time.Second,
			expectError: "invalid server IP",
		},
		{
			name:        "invalid port",
			serverIP:    "10.0.1.1",
			port:        0,
			duration:    10 * time.Second,
			expectError: "invalid port",
		},
		{
			name:        "duration too long",
			serverIP:    "10.0.1.1",
			port:        5001,
			duration:    2 * time.Hour,
			expectError: "duration too long",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup validation mocks
			originalValidateIPAddress := security.ValidateIPAddress
			originalValidatePort := security.ValidatePort

			security.ValidateIPAddress = func(ip string) error {
				if ip == "invalid" {
					return errors.New("invalid IP")
				}
				return nil
			}
			security.ValidatePort = func(port int) error {
				if port <= 0 {
					return errors.New("invalid port")
				}
				return nil
			}

			defer func() {
				security.ValidateIPAddress = originalValidateIPAddress
				security.ValidatePort = originalValidatePort
			}()

			result, err := manager.RunThroughputTest(tc.serverIP, tc.port, tc.duration)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
			assert.NotNil(t, result)
		})
	}
}

func TestIperfManager_RunLatencyTest_ValidationErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name        string
		serverIP    string
		port        int
		duration    time.Duration
		expectError string
	}{
		{
			name:        "invalid server IP",
			serverIP:    "invalid",
			port:        5001,
			duration:    10 * time.Second,
			expectError: "invalid server IP",
		},
		{
			name:        "invalid port",
			serverIP:    "10.0.1.1",
			port:        0,
			duration:    10 * time.Second,
			expectError: "invalid port",
		},
		{
			name:        "duration too long",
			serverIP:    "10.0.1.1",
			port:        5001,
			duration:    15 * time.Minute,
			expectError: "duration too long",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup validation mocks
			originalValidateIPAddress := security.ValidateIPAddress
			originalValidatePort := security.ValidatePort

			security.ValidateIPAddress = func(ip string) error {
				if ip == "invalid" {
					return errors.New("invalid IP")
				}
				return nil
			}
			security.ValidatePort = func(port int) error {
				if port <= 0 {
					return errors.New("invalid port")
				}
				return nil
			}

			defer func() {
				security.ValidateIPAddress = originalValidateIPAddress
				security.ValidatePort = originalValidatePort
			}()

			result, err := manager.RunLatencyTest(tc.serverIP, tc.port, tc.duration)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
			assert.NotNil(t, result)
		})
	}
}

func TestIperfManager_RunLatencyTest_PingErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name        string
		setupMock   func()
		expectError string
	}{
		{
			name: "ping argument validation fails",
			setupMock: func() {
				originalValidateCommandArgument := security.ValidateCommandArgument
				security.ValidateCommandArgument = func(arg string) error {
					if arg == "10.0.1.1" {
						return errors.New("invalid ping argument")
					}
					return nil
				}
			},
			expectError: "invalid ping argument",
		},
		{
			name: "ping execution fails",
			setupMock: func() {
				originalSecureExecute := security.SecureExecute
				security.SecureExecute = func(ctx context.Context, command string, args ...string) ([]byte, error) {
					if command == "ping" {
						return nil, errors.New("ping failed")
					}
					return nil, nil
				}
			},
			expectError: "ping test failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup base validation mocks
			originalValidateIPAddress := security.ValidateIPAddress
			originalValidatePort := security.ValidatePort
			originalValidateCommandArgument := security.ValidateCommandArgument
			originalSecureExecute := security.SecureExecute

			security.ValidateIPAddress = func(ip string) error { return nil }
			security.ValidatePort = func(port int) error { return nil }
			security.ValidateCommandArgument = func(arg string) error { return nil }
			security.SecureExecute = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				return []byte("PING 10.0.1.1: 56 data bytes\ntime=1.234 ms\ntime=2.345 ms"), nil
			}

			// Apply test-specific mock
			tc.setupMock()

			defer func() {
				security.ValidateIPAddress = originalValidateIPAddress
				security.ValidatePort = originalValidatePort
				security.ValidateCommandArgument = originalValidateCommandArgument
				security.SecureExecute = originalSecureExecute
			}()

			result, err := manager.RunLatencyTest("10.0.1.1", 5001, 10*time.Second)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
			assert.NotNil(t, result)
		})
	}
}

func TestIperfManager_StopAllServers_Errors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	// Add servers with different error conditions
	servers := map[string]*IperfServer{
		"port_5001": {Port: 5001, PID: 12345, Started: time.Now(), Context: context.Background(), Cancel: func() {}},
		"port_70000": {Port: 70000, PID: 12346, Started: time.Now(), Context: context.Background(), Cancel: func() {}}, // Invalid port
		"port_5003": {Port: 5003, PID: 12347, Started: time.Now(), Context: context.Background(), Cancel: func() {}},
	}

	for key, server := range servers {
		manager.servers[key] = server
	}

	// Mock validation functions to trigger different error paths
	originalValidateCommandArgument := security.ValidateCommandArgument
	originalCreateSafeProcessPattern := security.CreateSafeProcessPattern
	originalIsValidPortString := security.IsValidPortString
	originalValidatePkillPattern := security.ValidatePkillPattern
	originalSecureExecute := security.SecureExecute

	security.ValidateCommandArgument = func(arg string) error {
		if arg == "70000" {
			return errors.New("invalid port argument")
		}
		return nil
	}

	security.CreateSafeProcessPattern = func(process, flag, value string) string {
		return "pattern"
	}

	security.IsValidPortString = func(port string) bool {
		return port != "70000"
	}

	security.ValidatePkillPattern = func(pattern string) error {
		return nil
	}

	security.SecureExecute = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		return []byte("killed"), nil
	}

	defer func() {
		security.ValidateCommandArgument = originalValidateCommandArgument
		security.CreateSafeProcessPattern = originalCreateSafeProcessPattern
		security.IsValidPortString = originalIsValidPortString
		security.ValidatePkillPattern = originalValidatePkillPattern
		security.SecureExecute = originalSecureExecute
	}()

	err := manager.StopAllServers()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "errors stopping servers")

	// Verify some servers were removed despite errors
	assert.Len(t, manager.servers, 0) // All should be removed from map even if errors occurred
}

func TestIperfManager_ConcurrentOperations(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	// Mock successful operations
	originalSecureExecute := security.SecureExecute
	originalValidateIPAddress := security.ValidateIPAddress
	originalValidatePort := security.ValidatePort

	security.SecureExecute = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		if command == "iperf3" {
			return []byte("started"), nil
		}
		if command == "pgrep" {
			return []byte("12345"), nil
		}
		if command == "pkill" {
			return []byte("killed"), nil
		}
		return []byte("ok"), nil
	}

	security.ValidateIPAddress = func(ip string) error { return nil }
	security.ValidatePort = func(port int) error { return nil }

	defer func() {
		security.SecureExecute = originalSecureExecute
		security.ValidateIPAddress = originalValidateIPAddress
		security.ValidatePort = originalValidatePort
	}()

	t.Run("concurrent server start/stop", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 20)

		// Start 10 servers concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(port int) {
				defer wg.Done()
				err := manager.StartServer(5000 + port)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		// Stop 10 servers concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(port int) {
				defer wg.Done()
				// Add small delay to ensure servers are started first
				time.Sleep(50 * time.Millisecond)
				err := manager.StopServer(5000 + port)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Collect any errors
		var errorList []error
		for err := range errors {
			errorList = append(errorList, err)
		}

		// Should handle concurrent operations without major issues
		assert.True(t, len(errorList) < 5, "Too many errors in concurrent operations: %v", errorList)
	})
}

func TestIperfManager_ParseJSONOutput_Errors(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name   string
		output string
		result *IperfResult
	}{
		{
			name:   "invalid JSON",
			output: `{invalid json}`,
			result: &IperfResult{},
		},
		{
			name:   "missing fields",
			output: `{"start": {}}`,
			result: &IperfResult{},
		},
		{
			name:   "empty output",
			output: ``,
			result: &IperfResult{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := manager.parseJSONOutput(tc.output, tc.result)
			// parseJSONOutput should handle errors gracefully and not panic
			if tc.output == "" || tc.output == `{invalid json}` {
				assert.Error(t, err)
			} else {
				// For other cases, it might succeed with partial data
				// The test ensures no panic occurs
			}
		})
	}
}

func TestIperfManager_ParseTextOutput_EdgeCases(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	testCases := []struct {
		name   string
		output string
	}{
		{
			name:   "empty output",
			output: "",
		},
		{
			name:   "malformed lines",
			output: "malformed line\nanother bad line\n",
		},
		{
			name:   "partial data",
			output: "Connecting to host\nsender incomplete\nreceiver incomplete",
		},
		{
			name:   "unexpected format",
			output: "Unexpected output format\nwith various\nrandom lines",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &IperfResult{}
			err := manager.parseTextOutput(tc.output, result)

			// parseTextOutput should not return errors but handle malformed input gracefully
			assert.NoError(t, err)

			// Result should be initialized properly even with bad input
			assert.NotNil(t, result)
		})
	}
}

func TestIperfManager_GetActiveServers(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	// Test with empty servers
	servers := manager.GetActiveServers()
	assert.Empty(t, servers)

	// Add some servers
	testServers := map[string]*IperfServer{
		"port_5001": {Port: 5001, PID: 12345},
		"port_5002": {Port: 5002, PID: 12346},
	}

	for key, server := range testServers {
		manager.servers[key] = server
	}

	// Test with servers present
	servers = manager.GetActiveServers()
	assert.Len(t, servers, 2)
	assert.Contains(t, servers, "port_5001")
	assert.Contains(t, servers, "port_5002")

	// Verify it returns copies, not references
	servers["port_5001"].PID = 99999
	assert.NotEqual(t, 99999, manager.servers["port_5001"].PID)
}

func TestIperfManager_SecurityValidation_PIDFinding(t *testing.T) {
	testCases := []struct {
		name        string
		port        int
		setupMock   func()
		expectError string
	}{
		{
			name: "invalid port validation",
			port: -1,
			setupMock: func() {
				originalValidatePort := security.ValidatePort
				security.ValidatePort = func(port int) error {
					if port < 0 {
						return errors.New("invalid port")
					}
					return nil
				}
			},
			expectError: "invalid port",
		},
		{
			name: "invalid port string format",
			port: 5001,
			setupMock: func() {
				originalIsValidPortString := security.IsValidPortString
				security.IsValidPortString = func(port string) bool {
					return false
				}
			},
			expectError: "invalid port format",
		},
		{
			name: "pattern validation failure",
			port: 5001,
			setupMock: func() {
				originalValidatePgrepPattern := security.ValidatePgrepPattern
				security.ValidatePgrepPattern = func(pattern string) error {
					return errors.New("unsafe pattern")
				}
			},
			expectError: "unsafe pgrep pattern",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup base mocks
			originalValidatePort := security.ValidatePort
			originalIsValidPortString := security.IsValidPortString
			originalCreateSafeProcessPattern := security.CreateSafeProcessPattern
			originalValidatePgrepPattern := security.ValidatePgrepPattern

			security.ValidatePort = func(port int) error { return nil }
			security.IsValidPortString = func(port string) bool { return true }
			security.CreateSafeProcessPattern = func(process, flag, value string) string { return "pattern" }
			security.ValidatePgrepPattern = func(pattern string) error { return nil }

			// Apply test-specific mock
			tc.setupMock()

			defer func() {
				security.ValidatePort = originalValidatePort
				security.IsValidPortString = originalIsValidPortString
				security.CreateSafeProcessPattern = originalCreateSafeProcessPattern
				security.ValidatePgrepPattern = originalValidatePgrepPattern
			}()

			_, err := findIperfDaemonPID(tc.port)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
		})
	}
}

// Fuzz testing for robust error handling
func TestIperfManager_FuzzInputs(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	manager := NewIperfManager(logger)

	// Setup basic mocks
	originalValidateIPAddress := security.ValidateIPAddress
	originalValidatePort := security.ValidatePort
	originalValidateBandwidth := security.ValidateBandwidth

	security.ValidateIPAddress = func(ip string) error {
		if len(ip) > 100 || strings.Contains(ip, "\x00") {
			return errors.New("invalid IP")
		}
		return nil
	}
	security.ValidatePort = func(port int) error {
		if port <= 0 || port > 65535 {
			return errors.New("invalid port")
		}
		return nil
	}
	security.ValidateBandwidth = func(bw string) error {
		if len(bw) > 20 {
			return errors.New("invalid bandwidth")
		}
		return nil
	}

	defer func() {
		security.ValidateIPAddress = originalValidateIPAddress
		security.ValidatePort = originalValidatePort
		security.ValidateBandwidth = originalValidateBandwidth
	}()

	fuzzInputs := []struct {
		name   string
		config *IperfTestConfig
	}{
		{
			name: "very long IP",
			config: &IperfTestConfig{
				ServerIP: strings.Repeat("1", 200),
				Port:     5001,
			},
		},
		{
			name: "null bytes in IP",
			config: &IperfTestConfig{
				ServerIP: "10.0.1.1\x00evil",
				Port:     5001,
			},
		},
		{
			name: "extreme port values",
			config: &IperfTestConfig{
				ServerIP: "10.0.1.1",
				Port:     999999,
			},
		},
		{
			name: "very long bandwidth",
			config: &IperfTestConfig{
				ServerIP:  "10.0.1.1",
				Port:      5001,
				Bandwidth: strings.Repeat("M", 100),
			},
		},
		{
			name: "extreme parallel streams",
			config: &IperfTestConfig{
				ServerIP: "10.0.1.1",
				Port:     5001,
				Parallel: 999999,
			},
		},
	}

	for _, input := range fuzzInputs {
		t.Run(input.name, func(t *testing.T) {
			result, err := manager.RunTest(input.config)

			// Should either reject the input or handle it gracefully
			// Most importantly, should not panic
			if err != nil {
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.ErrorMessages)
			}
		})
	}
}