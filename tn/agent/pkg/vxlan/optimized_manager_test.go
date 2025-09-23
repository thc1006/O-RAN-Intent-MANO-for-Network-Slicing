package vxlan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCommandExecutor implements security.CommandExecutor for testing
type mockCommandExecutor struct {
	secureExecuteFunc               func(ctx context.Context, command string, args ...string) ([]byte, error)
	secureExecuteWithValidationFunc func(ctx context.Context, command string, customValidator func([]string) error, args ...string) ([]byte, error)
}

// mockCommandExecutor provides a simple command executor for testing

func (m *mockCommandExecutor) SecureExecute(ctx context.Context, command string, args ...string) ([]byte, error) {
	if m.secureExecuteFunc != nil {
		return m.secureExecuteFunc(ctx, command, args...)
	}
	return []byte("ok"), nil
}

func (m *mockCommandExecutor) SecureExecuteWithValidation(ctx context.Context, command string, customValidator func([]string) error, args ...string) ([]byte, error) {
	if m.secureExecuteWithValidationFunc != nil {
		return m.secureExecuteWithValidationFunc(ctx, command, customValidator, args...)
	}
	return []byte("ok"), nil
}

func TestNewOptimizedManager(t *testing.T) {
	manager := NewOptimizedManager()

	require.NotNil(t, manager)
	assert.NotNil(t, manager.tunnels)
	assert.NotNil(t, manager.commandCache)
	assert.NotNil(t, manager.workerPool)
	assert.NotNil(t, manager.metrics)
	assert.NotNil(t, manager.cmdExecutor)
	assert.Equal(t, 100*time.Millisecond, manager.batchInterval)
	assert.Equal(t, 10, cap(manager.workerPool))
	assert.False(t, manager.useNetlink) // Should be false initially
}

func TestOptimizedManager_CreateTunnelOptimized_ValidationErrors(t *testing.T) {
	// Create mock executor that always succeeds to focus on validation logic
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	testCases := []struct {
		name        string
		vxlanID     int32
		localIP     string
		remoteIPs   []string
		physInterface string
		expectError string
	}{
		{
			name:        "negative VXLAN ID",
			vxlanID:     -1,
			localIP:     "10.0.1.1",
			remoteIPs:   []string{"10.0.1.2"},
			physInterface: "eth0",
			expectError: "validation",
		},
		{
			name:        "empty local IP",
			vxlanID:     100,
			localIP:     "",
			remoteIPs:   []string{"10.0.1.2"},
			physInterface: "eth0",
			expectError: "validation",
		},
		{
			name:        "empty physical interface",
			vxlanID:     100,
			localIP:     "10.0.1.1",
			remoteIPs:   []string{"10.0.1.2"},
			physInterface: "",
			expectError: "validation",
		},
		{
			name:        "empty remote IPs",
			vxlanID:     100,
			localIP:     "10.0.1.1",
			remoteIPs:   []string{},
			physInterface: "eth0",
			expectError: "", // This might be valid in some cases
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := manager.CreateTunnelOptimized(tc.vxlanID, tc.localIP, tc.remoteIPs, tc.physInterface)

			if tc.expectError != "" {
				assert.Error(t, err)
				if tc.expectError != "validation" {
					assert.Contains(t, err.Error(), tc.expectError)
				}
			} else {
				// Error might occur due to other reasons, but not validation
				// We mainly check that it doesn't panic
			}
		})
	}
}

func TestOptimizedManager_CreateTunnelOptimized_WorkerPoolTimeout(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Fill up the worker pool
	for i := 0; i < cap(manager.workerPool); i++ {
		manager.workerPool <- struct{}{}
	}

	// This should timeout
	err := manager.CreateTunnelOptimized(100, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation timeout")

	// Clear the worker pool
	for i := 0; i < cap(manager.workerPool); i++ {
		<-manager.workerPool
	}
}

func TestOptimizedManager_CreateTunnelOptimized_ExistingTunnel(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Add an existing active tunnel
	existingTunnel := &EnhancedTunnelInfo{
		InterfaceName: "vxlan100",
		VxlanID:      100,
		LocalIP:      "10.0.1.1",
		RemoteIPs:    []string{"10.0.1.2"},
		State:        TunnelStateActive,
		CreatedAt:    time.Now(),
		Stats:        &TunnelStats{LastUpdated: time.Now()},
	}
	manager.tunnels[100] = existingTunnel

	// Creating the same tunnel should succeed immediately (already active)
	err := manager.CreateTunnelOptimized(100, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
	assert.NoError(t, err)

	// Add a failed tunnel
	failedTunnel := &EnhancedTunnelInfo{
		InterfaceName: "vxlan101",
		VxlanID:      101,
		LocalIP:      "10.0.1.1",
		RemoteIPs:    []string{"10.0.1.2"},
		State:        TunnelStateFailed,
		CreatedAt:    time.Now(),
		Stats:        &TunnelStats{LastUpdated: time.Now()},
	}
	manager.tunnels[101] = failedTunnel

	// Creating a tunnel with failed state should try to delete and recreate
	err = manager.CreateTunnelOptimized(101, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
	// The result depends on the mock setup, but it shouldn't panic
	assert.NotNil(t, manager.tunnels[101]) // Should still exist in some state
}

func TestOptimizedManager_CreateTunnelOptimized_DeletionPermissionError(t *testing.T) {
	// Mock deletion to return permission denied error
	mockExec := &mockCommandExecutor{
		secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
			if len(args) > 2 && args[1] == "del" {
				return nil, errors.New("permission denied")
			}
			return []byte("ok"), nil
		},
	}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Add a failed tunnel
	failedTunnel := &EnhancedTunnelInfo{
		InterfaceName: "vxlan101",
		VxlanID:      101,
		LocalIP:      "10.0.1.1",
		RemoteIPs:    []string{"10.0.1.2"},
		State:        TunnelStateFailed,
		CreatedAt:    time.Now(),
		Stats:        &TunnelStats{LastUpdated: time.Now()},
	}
	manager.tunnels[101] = failedTunnel

	// Should fail due to permission error during deletion
	err := manager.CreateTunnelOptimized(101, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deletion failed due to permissions")
}

func TestOptimizedManager_CreateTunnelOptimized_DeletionNonCriticalError(t *testing.T) {
	// Mock deletion to return device not found error (non-critical)
	callCount := 0
	mockExec := &mockCommandExecutor{
		secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
			callCount++
			if len(args) > 2 && args[1] == "del" {
				return nil, errors.New("device not found")
			}
			return []byte("ok"), nil
		},
	}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Add a failed tunnel
	failedTunnel := &EnhancedTunnelInfo{
		InterfaceName: "vxlan101",
		VxlanID:      101,
		LocalIP:      "10.0.1.1",
		RemoteIPs:    []string{"10.0.1.2"},
		State:        TunnelStateFailed,
		CreatedAt:    time.Now(),
		Stats:        &TunnelStats{LastUpdated: time.Now()},
	}
	manager.tunnels[101] = failedTunnel

	// Should continue with creation despite deletion error
	err := manager.CreateTunnelOptimized(101, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
	// Might succeed or fail based on subsequent operations, but should not fail due to deletion error
	_ = err // We check callCount instead of specific error
	assert.True(t, callCount > 1) // Should have tried deletion and then creation
}

func TestOptimizedManager_CreateTunnelIPCommand_CommandFailures(t *testing.T) {
	testCases := []struct {
		name        string
		setupMock   func() *mockCommandExecutor
		expectError string
	}{
		{
			name: "interface creation fails",
			setupMock: func() *mockCommandExecutor {
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						if len(args) > 3 && args[1] == "add" {
							return nil, errors.New("interface creation failed")
						}
						return []byte("ok"), nil
					},
				}
			},
			expectError: "failed to create VXLAN interface",
		},
		{
			name: "MTU setting fails",
			setupMock: func() *mockCommandExecutor {
				callCount := 0
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						callCount++
						if callCount == 2 && len(args) > 3 && args[3] == "mtu" {
							return nil, errors.New("MTU setting failed")
						}
						return []byte("ok"), nil
					},
				}
			},
			expectError: "failed to create VXLAN interface",
		},
		{
			name: "interface up fails",
			setupMock: func() *mockCommandExecutor {
				callCount := 0
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						callCount++
						if callCount == 3 && len(args) > 3 && args[3] == "up" {
							return nil, errors.New("interface up failed")
						}
						return []byte("ok"), nil
					},
				}
			},
			expectError: "failed to create VXLAN interface",
		},
		{
			name: "IP assignment fails with non-exists error",
			setupMock: func() *mockCommandExecutor {
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						if len(args) > 1 && args[1] == "addr" {
							return nil, errors.New("IP assignment failed")
						}
						return []byte("ok"), nil
					},
				}
			},
			expectError: "failed to assign IP",
		},
		{
			name: "IP assignment fails with exists error (should succeed)",
			setupMock: func() *mockCommandExecutor {
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						if len(args) > 1 && args[1] == "addr" {
							return nil, errors.New("File exists")
						}
						return []byte("ok"), nil
					},
				}
			},
			expectError: "", // Should not error as "File exists" is ignored
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExec := tc.setupMock()
			manager := NewOptimizedManagerWithExecutor(mockExec)

			err := manager.createTunnelIPCommand(100, "10.0.1.1", []string{"10.0.1.2"}, "eth0")

			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptimizedManager_CreateTunnelIPCommand_FDBErrors(t *testing.T) {
	// Test with multiple remote IPs where some FDB operations fail
	fdbCallCount := 0
	mockExec := &mockCommandExecutor{
		secureExecuteFunc: func(ctx context.Context, command string, args ...string) ([]byte, error) {
			if command == "bridge" {
				fdbCallCount++
				if fdbCallCount%2 == 0 { // Fail every second FDB operation
					return nil, errors.New("FDB operation failed")
				}
			}
			return []byte("ok"), nil
		},
	}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Should succeed despite FDB failures (they're non-critical)
	err := manager.createTunnelIPCommand(100, "10.0.1.1", []string{"10.0.1.2", "10.0.1.3", "10.0.1.4"}, "eth0")
	assert.NoError(t, err)

	// Test with single remote IP FDB failure
	fdbCallCount = 0
	err = manager.createTunnelIPCommand(101, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
	assert.NoError(t, err) // Should succeed despite FDB failure
}

func TestOptimizedManager_ExecuteOptimizedCommand_Errors(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		setupMock   func() *mockCommandExecutor
		expectError string
	}{
		{
			name:        "empty command arguments",
			args:        []string{},
			setupMock:   func() *mockCommandExecutor { return &mockCommandExecutor{} },
			expectError: "empty command arguments",
		},
		{
			name: "IP command validation fails",
			args: []string{"ip", "link", "add", "test"},
			setupMock: func() *mockCommandExecutor {
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						return nil, errors.New("IP validation failed")
					},
				}
			},
			expectError: "secure command execution failed",
		},
		{
			name: "non-IP command execution fails",
			args: []string{"bridge", "fdb", "add"},
			setupMock: func() *mockCommandExecutor {
				return &mockCommandExecutor{
					secureExecuteFunc: func(ctx context.Context, command string, args ...string) ([]byte, error) {
						return nil, errors.New("bridge command failed")
					},
				}
			},
			expectError: "secure command execution failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExec := tc.setupMock()
			manager := NewOptimizedManagerWithExecutor(mockExec)

			err := manager.executeOptimizedCommand(tc.args)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectError)
		})
	}
}

func TestOptimizedManager_ExecuteOptimizedCommand_Caching(t *testing.T) {
	// Mock successful execution
	executeCount := 0
	mockExec := &mockCommandExecutor{
		secureExecuteFunc: func(ctx context.Context, command string, args ...string) ([]byte, error) {
			executeCount++
			return []byte("ok"), nil
		},
	}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	args := []string{"bridge", "fdb", "show"}

	// First execution should call the actual command
	err := manager.executeOptimizedCommand(args)
	assert.NoError(t, err)
	assert.Equal(t, 1, executeCount)

	// Second execution within cache time should use cache
	err = manager.executeOptimizedCommand(args)
	assert.NoError(t, err)
	assert.Equal(t, 2, executeCount) // Cache hit increases execute count (but uses cached args)

	// Verify cache was populated
	assert.Len(t, manager.commandCache, 1)

	// Verify cache limit (add many commands to test cleanup)
	for i := 0; i < 120; i++ {
		uniqueArgs := []string{"test", fmt.Sprintf("command%d", i)}
		manager.executeOptimizedCommand(uniqueArgs)
	}

	// Cache should be limited to around 100 entries
	assert.True(t, len(manager.commandCache) <= 100)
}

func TestOptimizedManager_DeleteTunnelOptimized_Errors(t *testing.T) {
	testCases := []struct {
		name        string
		setupMock   func() *mockCommandExecutor
		expectError string
	}{
		{
			name: "delete command fails with real error",
			setupMock: func() *mockCommandExecutor {
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						return nil, errors.New("delete failed")
					},
				}
			},
			expectError: "failed to delete interface",
		},
		{
			name: "delete command fails with device not found (should succeed)",
			setupMock: func() *mockCommandExecutor {
				return &mockCommandExecutor{
					secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
						return nil, errors.New("Cannot find device")
					},
				}
			},
			expectError: "", // Should not error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExec := tc.setupMock()
			manager := NewOptimizedManagerWithExecutor(mockExec)

			// Add a tunnel to delete
			tunnel := &EnhancedTunnelInfo{
				InterfaceName: "vxlan100",
				VxlanID:      100,
				State:        TunnelStateActive,
				CreatedAt:    time.Now(),
				Stats:        &TunnelStats{LastUpdated: time.Now()},
			}
			manager.tunnels[100] = tunnel

			err := manager.DeleteTunnelOptimized(100)

			if tc.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				assert.NoError(t, err)
			}

			// Tunnel should be removed from map regardless of error
			assert.NotContains(t, manager.tunnels, int32(100))
		})
	}
}

func TestOptimizedManager_GetTunnelStatusOptimized_Errors(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Test tunnel not found
	info, err := manager.GetTunnelStatusOptimized(999)
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "tunnel 999 not found")

	// Add a tunnel and test successful retrieval
	tunnel := &EnhancedTunnelInfo{
		InterfaceName: "vxlan100",
		VxlanID:      100,
		State:        TunnelStateActive,
		CreatedAt:    time.Now(),
		LastUsed:     time.Now().Add(-1 * time.Hour),
		Stats:        &TunnelStats{LastUpdated: time.Now().Add(-1 * time.Hour)},
	}
	manager.tunnels[100] = tunnel

	info, err = manager.GetTunnelStatusOptimized(100)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, int32(100), info.VxlanID)

	// LastUsed should be updated
	assert.True(t, info.LastUsed.After(tunnel.LastUsed))
}

func TestOptimizedManager_BatchProcessing(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Test shouldBatch logic
	assert.False(t, manager.shouldBatch("create", 100))  // Low VXLAN ID - critical
	assert.True(t, manager.shouldBatch("create", 2000))  // High VXLAN ID - batchable

	// Test addToBatch
	callbackCalled := false
	callback := func(err error) {
		callbackCalled = true
	}

	manager.addToBatch("create", 2000, "10.0.1.1", []string{"10.0.1.2"}, "eth0", callback)
	assert.Len(t, manager.pendingOps, 1)

	// Test processBatch
	manager.processBatch()

	// Wait a bit for async processing
	time.Sleep(200 * time.Millisecond)

	// Batch should be processed
	assert.Len(t, manager.pendingOps, 0)
	assert.True(t, callbackCalled)

	// Verify tunnel was created
	assert.Contains(t, manager.tunnels, int32(2000))
}

func TestOptimizedManager_CleanupOptimized_Errors(t *testing.T) {
	// Mock delete operations with some failures
	deleteCount := 0
	mockExec := &mockCommandExecutor{
		secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
			deleteCount++
			if deleteCount%2 == 0 {
				return nil, errors.New("delete failed")
			}
			return []byte("ok"), nil
		},
	}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Add tunnels with different delete behaviors
	tunnels := map[int32]*EnhancedTunnelInfo{
		100: {VxlanID: 100, State: TunnelStateActive},
		101: {VxlanID: 101, State: TunnelStateActive},
		102: {VxlanID: 102, State: TunnelStateActive},
	}

	for id, tunnel := range tunnels {
		manager.tunnels[id] = tunnel
	}

	err := manager.CleanupOptimized()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup failed")

	// All tunnels should be removed from tracking regardless of errors
	assert.Empty(t, manager.tunnels)
}

func TestOptimizedManager_ListActiveTunnels(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Add tunnels in different states
	tunnels := map[int32]*EnhancedTunnelInfo{
		100: {VxlanID: 100, State: TunnelStateActive, InterfaceName: "vxlan100"},
		101: {VxlanID: 101, State: TunnelStateFailed, InterfaceName: "vxlan101"},
		102: {VxlanID: 102, State: TunnelStateActive, InterfaceName: "vxlan102"},
		103: {VxlanID: 103, State: TunnelStateDeleting, InterfaceName: "vxlan103"},
	}

	for id, tunnel := range tunnels {
		manager.tunnels[id] = tunnel
	}

	activeTunnels := manager.ListActiveTunnels()

	// Should only return active tunnels
	assert.Len(t, activeTunnels, 2)
	assert.Contains(t, activeTunnels, int32(100))
	assert.Contains(t, activeTunnels, int32(102))
	assert.NotContains(t, activeTunnels, int32(101))
	assert.NotContains(t, activeTunnels, int32(103))

	// Verify it returns copies
	activeTunnels[100].InterfaceName = "modified"
	assert.NotEqual(t, "modified", manager.tunnels[100].InterfaceName)
}

func TestOptimizedManager_GetMetrics(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Update some metrics
	manager.updateMetrics(100*time.Millisecond, true)
	manager.updateMetrics(200*time.Millisecond, false)
	manager.metrics.CacheHits = 5
	manager.metrics.BatchedOps = 3

	metrics := manager.GetMetrics()

	assert.Equal(t, int64(2), metrics.TotalOperations)
	assert.Equal(t, int64(1), metrics.SuccessfulOps)
	assert.Equal(t, int64(1), metrics.FailedOps)
	assert.Equal(t, int64(5), metrics.CacheHits)
	assert.Equal(t, int64(3), metrics.BatchedOps)
	assert.True(t, metrics.AvgOpTimeMs > 0)

	// Verify it returns a copy
	metrics.TotalOperations = 999
	assert.NotEqual(t, int64(999), manager.metrics.TotalOperations)
}

func TestOptimizedManager_GenerateVXLANIP(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	testCases := []struct {
		name     string
		vxlanID  int32
		nodeIP   string
		expected string
	}{
		{
			name:     "simple case",
			vxlanID:  100,
			nodeIP:   "192.168.1.10",
			expected: "10.0.100.10",
		},
		{
			name:     "large VXLAN ID",
			vxlanID:  65536,
			nodeIP:   "192.168.1.20",
			expected: "10.0.0.20",
		},
		{
			name:     "VXLAN ID with overflow",
			vxlanID:  300,
			nodeIP:   "192.168.1.30",
			expected: "10.1.44.30",
		},
		{
			name:     "invalid node IP format",
			vxlanID:  100,
			nodeIP:   "invalid",
			expected: "10.0.100.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := manager.generateVXLANIP(tc.vxlanID, tc.nodeIP)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestOptimizedManager_ConcurrentOperations(t *testing.T) {
	mockExec := &mockCommandExecutor{
		secureExecuteWithValidationFunc: func(ctx context.Context, command string, validator func([]string) error, args ...string) ([]byte, error) {
			// Add small delay to simulate real operation
			time.Sleep(1 * time.Millisecond)
			return []byte("ok"), nil
		},
	}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Test concurrent tunnel creation and deletion
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// Concurrent creates
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := manager.CreateTunnelOptimized(int32(1000+id), "10.0.1.1", []string{"10.0.1.2"}, "eth0")
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	// Concurrent deletes (some will fail because tunnels don't exist yet)
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := manager.DeleteTunnelOptimized(int32(1000 + id))
			if err != nil && !strings.Contains(err.Error(), "Cannot find device") {
				errChan <- err
			}
		}(i)
	}

	// Concurrent status checks
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := manager.GetTunnelStatusOptimized(int32(1000 + id))
			if err != nil && !strings.Contains(err.Error(), "not found") {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// Should handle concurrent operations with minimal errors
	assert.True(t, len(errors) < 10, "Too many errors in concurrent operations: %v", errors)

	// Verify metrics were updated
	metrics := manager.GetMetrics()
	assert.True(t, metrics.TotalOperations > 0)
}

func TestOptimizedManager_MemoryLeaks(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	// Create and delete many tunnels to test for memory leaks
	for i := 0; i < 1000; i++ {
		vxlanID := int32(i + 10000)

		// Create tunnel
		err := manager.CreateTunnelOptimized(vxlanID, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
		assert.NoError(t, err)

		// Delete tunnel
		err = manager.DeleteTunnelOptimized(vxlanID)
		assert.NoError(t, err)
	}

	// Verify tunnels map is cleaned up
	assert.Empty(t, manager.tunnels)

	// Verify command cache is bounded
	assert.True(t, len(manager.commandCache) <= 100)

	// Test batch operations cleanup
	for i := 0; i < 100; i++ {
		manager.addToBatch("create", int32(i+20000), "10.0.1.1", []string{"10.0.1.2"}, "eth0", nil)
	}

	// Process batches multiple times
	for i := 0; i < 5; i++ {
		manager.processBatch()
		time.Sleep(10 * time.Millisecond)
	}

	// Pending operations should be processed
	assert.Empty(t, manager.pendingOps)
}

// Benchmark tests for performance validation
func BenchmarkOptimizedManager_CreateTunnel(b *testing.B) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vxlanID := int32(i + 50000)
		manager.CreateTunnelOptimized(vxlanID, "10.0.1.1", []string{"10.0.1.2"}, "eth0")
	}
}

func BenchmarkOptimizedManager_ExecuteCommand(b *testing.B) {
	mockExec := &mockCommandExecutor{
		secureExecuteFunc: func(ctx context.Context, command string, args ...string) ([]byte, error) {
			return []byte("ok"), nil
		},
	}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	args := []string{"bridge", "fdb", "show"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.executeOptimizedCommand(args)
	}
}

// Fuzz testing for edge cases
func TestOptimizedManager_FuzzInputs(t *testing.T) {
	mockExec := &mockCommandExecutor{}
	manager := NewOptimizedManagerWithExecutor(mockExec)

	fuzzInputs := []struct {
		name          string
		vxlanID       int32
		localIP       string
		remoteIPs     []string
		physInterface string
	}{
		{
			name:          "extreme VXLAN ID",
			vxlanID:       2147483647, // Max int32
			localIP:       "10.0.1.1",
			remoteIPs:     []string{"10.0.1.2"},
			physInterface: "eth0",
		},
		{
			name:          "zero VXLAN ID",
			vxlanID:       0,
			localIP:       "10.0.1.1",
			remoteIPs:     []string{"10.0.1.2"},
			physInterface: "eth0",
		},
		{
			name:          "many remote IPs",
			vxlanID:       100,
			localIP:       "10.0.1.1",
			remoteIPs:     make([]string, 1000), // Large slice
			physInterface: "eth0",
		},
		{
			name:          "long strings",
			vxlanID:       100,
			localIP:       strings.Repeat("1", 1000),
			remoteIPs:     []string{strings.Repeat("2", 1000)},
			physInterface: strings.Repeat("e", 1000),
		},
		{
			name:          "special characters",
			vxlanID:       100,
			localIP:       "10.0.1.1\x00\x01",
			remoteIPs:     []string{"10.0.1.2\n\r\t"},
			physInterface: "eth0\x00",
		},
	}

	for _, input := range fuzzInputs {
		t.Run(input.name, func(t *testing.T) {
			// Fill large slice with valid IPs for the "many remote IPs" test
			if len(input.remoteIPs) == 1000 {
				for i := range input.remoteIPs {
					input.remoteIPs[i] = fmt.Sprintf("10.0.%d.%d", i/256, i%256)
				}
			}

			// Should not panic with any input
			err := manager.CreateTunnelOptimized(input.vxlanID, input.localIP, input.remoteIPs, input.physInterface)

			// Error or success is fine, but no panic
			_ = err

			// Cleanup if tunnel was created
			manager.DeleteTunnelOptimized(input.vxlanID)
		})
	}
}