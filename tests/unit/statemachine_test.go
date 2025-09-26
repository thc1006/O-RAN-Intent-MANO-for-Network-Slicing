package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/statemachine"
)

// Test state machine creation and initialization
func TestStateMachineCreation(t *testing.T) {
	config := statemachine.DefaultConfig()

	sm := statemachine.NewStateMachine("test-sm", statemachine.StateInitializing, config)

	assert.NotNil(t, sm)
	assert.Equal(t, "test-sm", sm.ID)
	assert.Equal(t, statemachine.StateInitializing, sm.GetCurrentState())
	assert.Equal(t, statemachine.StateInitializing, sm.InitialState)
}

// Test state transitions
func TestStateMachineTransitions(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-transitions", statemachine.StateInitializing, config)
	ctx := context.Background()

	tests := []struct {
		name          string
		event         statemachine.Event
		expectedState statemachine.State
		shouldError   bool
	}{
		{
			name:          "Initialize to validate",
			event:         statemachine.EventValidate,
			expectedState: statemachine.StateValidating,
			shouldError:   false,
		},
		{
			name:          "Validation success",
			event:         statemachine.EventValidationSuccess,
			expectedState: statemachine.StatePending,
			shouldError:   false,
		},
		{
			name:          "Plan",
			event:         statemachine.EventPlan,
			expectedState: statemachine.StatePlanning,
			shouldError:   false,
		},
		{
			name:          "Planning success",
			event:         statemachine.EventPlanningSuccess,
			expectedState: statemachine.StatePlanned,
			shouldError:   false,
		},
		{
			name:          "Deploy",
			event:         statemachine.EventDeploy,
			expectedState: statemachine.StateDeploying,
			shouldError:   false,
		},
		{
			name:          "Deployment success",
			event:         statemachine.EventDeploymentSuccess,
			expectedState: statemachine.StateDeployed,
			shouldError:   false,
		},
		{
			name:          "Activate",
			event:         statemachine.EventActivate,
			expectedState: statemachine.StateActive,
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.SendEvent(ctx, tt.event, nil)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedState, sm.GetCurrentState())
			}
		})
	}
}

// Test error handling and failure states
func TestStateMachineErrorHandling(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-errors", statemachine.StateInitializing, config)
	ctx := context.Background()

	// Start validation
	err := sm.SendEvent(ctx, statemachine.EventValidate, nil)
	require.NoError(t, err)
	assert.Equal(t, statemachine.StateValidating, sm.GetCurrentState())

	// Trigger validation failure
	err = sm.SendEvent(ctx, statemachine.EventValidationFailure, nil)
	require.NoError(t, err)
	assert.Equal(t, statemachine.StateValidationFailed, sm.GetCurrentState())

	// Test retry capability
	canRetry := sm.CanTransition(statemachine.EventRetry)
	assert.True(t, canRetry)

	// Retry validation
	err = sm.SendEvent(ctx, statemachine.EventRetry, nil)
	require.NoError(t, err)
	assert.Equal(t, statemachine.StateValidating, sm.GetCurrentState())
}

// Test rollback functionality
func TestStateMachineRollback(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-rollback", statemachine.StateInitializing, config)
	ctx := context.Background()

	// Progress through states to deployment
	events := []statemachine.Event{
		statemachine.EventValidate,
		statemachine.EventValidationSuccess,
		statemachine.EventPlan,
		statemachine.EventPlanningSuccess,
		statemachine.EventDeploy,
	}

	for _, event := range events {
		err := sm.SendEvent(ctx, event, nil)
		require.NoError(t, err)
	}

	assert.Equal(t, statemachine.StateDeploying, sm.GetCurrentState())

	// Trigger deployment failure
	err := sm.SendEvent(ctx, statemachine.EventDeploymentFailure, nil)
	require.NoError(t, err)
	assert.Equal(t, statemachine.StateDeploymentFailed, sm.GetCurrentState())

	// Initiate rollback
	err = sm.SendEvent(ctx, statemachine.EventRollback, nil)
	require.NoError(t, err)
	assert.Equal(t, statemachine.StateRollingBack, sm.GetCurrentState())

	// Complete rollback
	err = sm.SendEvent(ctx, statemachine.EventRollbackSuccess, nil)
	require.NoError(t, err)
	assert.Equal(t, statemachine.StateRolledBack, sm.GetCurrentState())
}

// Test state machine metadata
func TestStateMachineMetadata(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-metadata", statemachine.StateInitializing, config)

	// Set metadata
	sm.SetMetadata("intent", "test-intent")
	sm.SetMetadata("resources", map[string]interface{}{"cpu": "2", "memory": "4Gi"})

	// Get metadata
	intent, exists := sm.GetMetadata("intent")
	assert.True(t, exists)
	assert.Equal(t, "test-intent", intent)

	resources, exists := sm.GetMetadata("resources")
	assert.True(t, exists)
	assert.IsType(t, map[string]interface{}{}, resources)

	// Non-existent metadata
	_, exists = sm.GetMetadata("non-existent")
	assert.False(t, exists)
}

// Test state machine history tracking
func TestStateMachineHistory(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-history", statemachine.StateInitializing, config)
	ctx := context.Background()

	// Execute some transitions
	events := []statemachine.Event{
		statemachine.EventValidate,
		statemachine.EventValidationSuccess,
		statemachine.EventPlan,
	}

	for _, event := range events {
		err := sm.SendEvent(ctx, event, nil)
		require.NoError(t, err)
	}

	// Check state history
	stateHistory := sm.GetHistory()
	assert.Len(t, stateHistory, 3)

	// Verify first transition
	assert.Equal(t, statemachine.StateInitializing, stateHistory[0].From)
	assert.Equal(t, statemachine.StateValidating, stateHistory[0].To)
	assert.Equal(t, statemachine.EventValidate, stateHistory[0].Event)

	// Check event history
	eventHistory := sm.GetEventHistory()
	assert.Len(t, eventHistory, 3)

	for _, record := range eventHistory {
		assert.True(t, record.Success)
		assert.NoError(t, record.Error)
	}
}

// Test retry mechanism with backoff
func TestStateMachineRetryWithBackoff(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-retry", statemachine.StateInitializing, config)
	ctx := context.Background()

	retryPolicy := statemachine.DefaultRetryPolicy()
	retryPolicy.MaxAttempts = 3
	retryPolicy.InitialDelay = 10 * time.Millisecond

	attempts := 0
	startTime := time.Now()

	err := sm.RetryWithBackoff(ctx, retryPolicy, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("simulated failure %d", attempts)
		}
		return nil
	})

	duration := time.Since(startTime)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	// Should have some delay due to backoff
	assert.Greater(t, duration, 20*time.Millisecond)
}

// Test invalid transitions
func TestStateMachineInvalidTransitions(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-invalid", statemachine.StateInitializing, config)
	ctx := context.Background()

	// Try invalid transition from initializing state
	err := sm.SendEvent(ctx, statemachine.EventDeploy, nil)
	assert.Error(t, err)
	assert.Equal(t, statemachine.ErrInvalidTransition, err)

	// State should remain unchanged
	assert.Equal(t, statemachine.StateInitializing, sm.GetCurrentState())

	// Try with completely invalid event
	err = sm.SendEvent(ctx, statemachine.Event("invalid-event"), nil)
	assert.Error(t, err)
}

// Test state machine listener functionality
func TestStateMachineListeners(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-listeners", statemachine.StateInitializing, config)
	ctx := context.Background()

	// Test listener implementation
	listener := &TestStateListener{
		stateChanges: make([]StateChangeRecord, 0),
		errors:       make([]error, 0),
	}

	sm.AddListener(listener)

	// Execute transition
	err := sm.SendEvent(ctx, statemachine.EventValidate, nil)
	require.NoError(t, err)

	// Give listener time to process (they run in goroutines)
	time.Sleep(10 * time.Millisecond)

	// Check listener was called
	assert.Len(t, listener.stateChanges, 1)
	assert.Equal(t, statemachine.StateInitializing, listener.stateChanges[0].From)
	assert.Equal(t, statemachine.StateValidating, listener.stateChanges[0].To)
	assert.Equal(t, statemachine.EventValidate, listener.stateChanges[0].Event)
}

// Test concurrent state machine operations
func TestStateMachineConcurrency(t *testing.T) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("test-concurrent", statemachine.StateInitializing, config)
	ctx := context.Background()

	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	// Try to send the same event concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			err := sm.SendEvent(ctx, statemachine.EventValidate, nil)
			results <- err
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	// Only one should succeed, others should fail with invalid transition
	assert.Equal(t, 1, successCount)
	assert.Equal(t, numGoroutines-1, errorCount)
	assert.Equal(t, statemachine.StateValidating, sm.GetCurrentState())
}

// Test state machine timeout handling
func TestStateMachineTimeout(t *testing.T) {
	config := statemachine.DefaultConfig()
	config.StateTimeout = 100 * time.Millisecond

	sm := statemachine.NewStateMachine("test-timeout", statemachine.StateInitializing, config)

	// Add action that times out
	sm.AddTransition(
		statemachine.StateInitializing,
		statemachine.EventValidate,
		statemachine.StateValidating,
		func(ctx context.Context, sm *statemachine.StateMachine, data interface{}) error {
			time.Sleep(200 * time.Millisecond) // Longer than timeout
			return nil
		},
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := sm.SendEvent(ctx, statemachine.EventValidate, nil)
	assert.Error(t, err)
}

// Helper types for testing

type StateChangeRecord struct {
	From  statemachine.State
	To    statemachine.State
	Event statemachine.Event
}

type TestStateListener struct {
	stateChanges []StateChangeRecord
	errors       []error
}

func (l *TestStateListener) OnStateChange(ctx context.Context, sm *statemachine.StateMachine, from, to statemachine.State, event statemachine.Event) {
	l.stateChanges = append(l.stateChanges, StateChangeRecord{
		From:  from,
		To:    to,
		Event: event,
	})
}

func (l *TestStateListener) OnError(ctx context.Context, sm *statemachine.StateMachine, err error) {
	l.errors = append(l.errors, err)
}

// Benchmark tests for performance validation

func BenchmarkStateMachineTransition(b *testing.B) {
	config := statemachine.DefaultConfig()
	sm := statemachine.NewStateMachine("bench-sm", statemachine.StateInitializing, config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset state for each iteration
		sm.CurrentState = statemachine.StateInitializing
		sm.SendEvent(ctx, statemachine.EventValidate, nil)
	}
}

func BenchmarkStateMachineCreation(b *testing.B) {
	config := statemachine.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		statemachine.NewStateMachine(fmt.Sprintf("bench-sm-%d", i), statemachine.StateInitializing, config)
	}
}

// Table-driven tests for comprehensive state coverage

func TestStateMachineStateTransitions(t *testing.T) {
	config := statemachine.DefaultConfig()

	tests := []struct {
		name         string
		initialState statemachine.State
		event        statemachine.Event
		expectedState statemachine.State
		shouldError  bool
	}{
		{"Initialize to validate", statemachine.StateInitializing, statemachine.EventValidate, statemachine.StateValidating, false},
		{"Validate success", statemachine.StateValidating, statemachine.EventValidationSuccess, statemachine.StatePending, false},
		{"Validate failure", statemachine.StateValidating, statemachine.EventValidationFailure, statemachine.StateValidationFailed, false},
		{"Plan from pending", statemachine.StatePending, statemachine.EventPlan, statemachine.StatePlanning, false},
		{"Plan success", statemachine.StatePlanning, statemachine.EventPlanningSuccess, statemachine.StatePlanned, false},
		{"Plan failure", statemachine.StatePlanning, statemachine.EventPlanningFailure, statemachine.StatePlanningFailed, false},
		{"Deploy from planned", statemachine.StatePlanned, statemachine.EventDeploy, statemachine.StateDeploying, false},
		{"Deploy success", statemachine.StateDeploying, statemachine.EventDeploymentSuccess, statemachine.StateDeployed, false},
		{"Deploy failure", statemachine.StateDeploying, statemachine.EventDeploymentFailure, statemachine.StateDeploymentFailed, false},
		{"Activate from deployed", statemachine.StateDeployed, statemachine.EventActivate, statemachine.StateActive, false},
		{"Terminate from active", statemachine.StateActive, statemachine.EventTerminate, statemachine.StateTerminating, false},
		{"Rollback from failed", statemachine.StateDeploymentFailed, statemachine.EventRollback, statemachine.StateRollingBack, false},
		{"Invalid transition", statemachine.StateInitializing, statemachine.EventDeploy, statemachine.StateInitializing, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := statemachine.NewStateMachine("test-"+tt.name, tt.initialState, config)
			ctx := context.Background()

			err := sm.SendEvent(ctx, tt.event, nil)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedState, sm.GetCurrentState())
			}
		})
	}
}