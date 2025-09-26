package statemachine

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

// NewStateMachine creates a new state machine
func NewStateMachine(id string, initialState State, config Config) *StateMachine {
	sm := &StateMachine{
		ID:           id,
		CurrentState: initialState,
		InitialState: initialState,
		transitions:  make(map[State]map[Event]*Transition),
		eventHistory: make([]EventRecord, 0),
		stateHistory: make([]StateRecord, 0),
		maxRetries:   config.MaxRetries,
		metadata:     make(map[string]interface{}),
		listeners:    make([]StateListener, 0),
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	// Set up default transitions
	sm.setupDefaultTransitions()
	return sm
}

// setupDefaultTransitions configures the standard deployment lifecycle transitions
func (sm *StateMachine) setupDefaultTransitions() {
	// Initialization flow
	sm.AddTransition(StateInitializing, EventValidate, StateValidating, sm.validateAction, nil)
	sm.AddTransition(StateValidating, EventValidationSuccess, StatePending, nil, nil)
	sm.AddTransition(StateValidating, EventValidationFailure, StateValidationFailed, sm.handleValidationFailure, nil)

	// Planning flow
	sm.AddTransition(StatePending, EventPlan, StatePlanning, sm.planAction, nil)
	sm.AddTransition(StatePlanning, EventPlanningSuccess, StatePlanned, nil, nil)
	sm.AddTransition(StatePlanning, EventPlanningFailure, StatePlanningFailed, sm.handlePlanningFailure, nil)

	// Deployment flow
	sm.AddTransition(StatePlanned, EventDeploy, StateDeploying, sm.deployAction, nil)
	sm.AddTransition(StateDeploying, EventDeploymentSuccess, StateDeployed, nil, nil)
	sm.AddTransition(StateDeploying, EventDeploymentFailure, StateDeploymentFailed, sm.handleDeploymentFailure, nil)

	// Activation flow
	sm.AddTransition(StateDeployed, EventActivate, StateActive, sm.activateAction, nil)

	// Recovery flows
	sm.AddTransition(StateValidationFailed, EventRetry, StateValidating, sm.retryValidation, sm.canRetry)
	sm.AddTransition(StatePlanningFailed, EventRetry, StatePlanning, sm.retryPlanning, sm.canRetry)
	sm.AddTransition(StateDeploymentFailed, EventRetry, StateDeploying, sm.retryDeployment, sm.canRetry)
	sm.AddTransition(StateDeploymentFailed, EventRollback, StateRollingBack, sm.rollbackAction, nil)

	// Rollback flows
	sm.AddTransition(StateRollingBack, EventRollbackSuccess, StateRolledBack, nil, nil)
	sm.AddTransition(StateRollingBack, EventRollbackFailure, StateFailed, sm.handleRollbackFailure, nil)

	// Termination flows
	sm.AddTransition(StateActive, EventTerminate, StateTerminating, sm.terminateAction, nil)
	sm.AddTransition(StateDeployed, EventTerminate, StateTerminating, sm.terminateAction, nil)
	sm.AddTransition(StateTerminating, EventActivationSuccess, StateTerminated, nil, nil)

	// Error handling
	sm.AddTransition(StateValidationFailed, EventRecover, StateRecovering, sm.recoverAction, nil)
	sm.AddTransition(StatePlanningFailed, EventRecover, StateRecovering, sm.recoverAction, nil)
	sm.AddTransition(StateDeploymentFailed, EventRecover, StateRecovering, sm.recoverAction, nil)
	sm.AddTransition(StateRecovering, EventRecoverySuccess, StatePending, nil, nil)
	sm.AddTransition(StateRecovering, EventRecoveryFailure, StateFailed, sm.handleRecoveryFailure, nil)
}

// AddTransition adds a new transition to the state machine
func (sm *StateMachine) AddTransition(from State, event Event, to State, action ActionFunc, guard GuardFunc) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.transitions[from] == nil {
		sm.transitions[from] = make(map[Event]*Transition)
	}

	sm.transitions[from][event] = &Transition{
		From:   from,
		To:     to,
		Event:  event,
		Action: action,
		Guard:  guard,
	}
}

// SendEvent processes an event and potentially triggers a state transition
func (sm *StateMachine) SendEvent(ctx context.Context, event Event, data interface{}) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	startTime := time.Now()
	sm.updatedAt = startTime

	// Record the event
	eventRecord := EventRecord{
		Event:     event,
		Timestamp: startTime,
		State:     sm.CurrentState,
		Data:      data,
	}

	// Find transition for current state and event
	stateTransitions, exists := sm.transitions[sm.CurrentState]
	if !exists {
		eventRecord.Success = false
		eventRecord.Error = fmt.Errorf("no transitions defined for state %s", sm.CurrentState)
		sm.eventHistory = append(sm.eventHistory, eventRecord)
		return eventRecord.Error
	}

	transition, exists := stateTransitions[event]
	if !exists {
		eventRecord.Success = false
		eventRecord.Error = fmt.Errorf("invalid transition from %s with event %s", sm.CurrentState, event)
		sm.eventHistory = append(sm.eventHistory, eventRecord)
		return ErrInvalidTransition
	}

	// Check guard condition
	if transition.Guard != nil && !transition.Guard(ctx, sm, data) {
		eventRecord.Success = false
		eventRecord.Error = ErrGuardConditionFailed
		sm.eventHistory = append(sm.eventHistory, eventRecord)
		return ErrGuardConditionFailed
	}

	// Execute action if defined
	var actionErr error
	if transition.Action != nil {
		actionErr = transition.Action(ctx, sm, data)
		if actionErr != nil {
			eventRecord.Success = false
			eventRecord.Error = fmt.Errorf("action failed: %w", actionErr)
			sm.eventHistory = append(sm.eventHistory, eventRecord)
			sm.lastError = actionErr
			return actionErr
		}
	}

	// Perform state transition
	previousState := sm.CurrentState
	sm.PreviousState = previousState
	sm.CurrentState = transition.To

	// Record state transition
	duration := time.Since(startTime)
	stateRecord := StateRecord{
		From:      previousState,
		To:        transition.To,
		Event:     event,
		Timestamp: startTime,
		Duration:  duration,
	}
	sm.stateHistory = append(sm.stateHistory, stateRecord)

	// Record successful event
	eventRecord.Success = true
	sm.eventHistory = append(sm.eventHistory, eventRecord)

	// Notify listeners
	sm.notifyStateChange(ctx, previousState, transition.To, event)

	return nil
}

// GetCurrentState returns the current state
func (sm *StateMachine) GetCurrentState() State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.CurrentState
}

// CanTransition checks if a transition is possible
func (sm *StateMachine) CanTransition(event Event) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stateTransitions, exists := sm.transitions[sm.CurrentState]
	if !exists {
		return false
	}

	_, exists = stateTransitions[event]
	return exists
}

// GetHistory returns the state transition history
func (sm *StateMachine) GetHistory() []StateRecord {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return append([]StateRecord(nil), sm.stateHistory...)
}

// GetEventHistory returns the event history
func (sm *StateMachine) GetEventHistory() []EventRecord {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return append([]EventRecord(nil), sm.eventHistory...)
}

// AddListener adds a state change listener
func (sm *StateMachine) AddListener(listener StateListener) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// SetMetadata sets metadata for the state machine
func (sm *StateMachine) SetMetadata(key string, value interface{}) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.metadata[key] = value
}

// GetMetadata gets metadata from the state machine
func (sm *StateMachine) GetMetadata(key string) (interface{}, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	value, exists := sm.metadata[key]
	return value, exists
}

// notifyStateChange notifies all listeners of state changes
func (sm *StateMachine) notifyStateChange(ctx context.Context, from, to State, event Event) {
	for _, listener := range sm.listeners {
		go func(l StateListener) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("State listener panic: %v", r)
				}
			}()
			l.OnStateChange(ctx, sm, from, to, event)
		}(listener)
	}
}

// notifyError notifies all listeners of errors
func (sm *StateMachine) notifyError(ctx context.Context, err error) {
	for _, listener := range sm.listeners {
		go func(l StateListener) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Error listener panic: %v", r)
				}
			}()
			l.OnError(ctx, sm, err)
		}(listener)
	}
}

// Default action implementations

func (sm *StateMachine) validateAction(ctx context.Context, machine *StateMachine, data interface{}) error {
	// Placeholder for validation logic
	log.Printf("Validating deployment %s", sm.ID)
	// Simulate validation time
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (sm *StateMachine) planAction(ctx context.Context, machine *StateMachine, data interface{}) error {
	// Placeholder for planning logic
	log.Printf("Planning deployment %s", sm.ID)
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (sm *StateMachine) deployAction(ctx context.Context, machine *StateMachine, data interface{}) error {
	// Placeholder for deployment logic
	log.Printf("Deploying %s", sm.ID)
	time.Sleep(1 * time.Second)
	return nil
}

func (sm *StateMachine) activateAction(ctx context.Context, machine *StateMachine, data interface{}) error {
	// Placeholder for activation logic
	log.Printf("Activating deployment %s", sm.ID)
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (sm *StateMachine) rollbackAction(ctx context.Context, machine *StateMachine, data interface{}) error {
	// Placeholder for rollback logic
	log.Printf("Rolling back deployment %s", sm.ID)
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (sm *StateMachine) terminateAction(ctx context.Context, machine *StateMachine, data interface{}) error {
	// Placeholder for termination logic
	log.Printf("Terminating deployment %s", sm.ID)
	time.Sleep(300 * time.Millisecond)
	return nil
}

func (sm *StateMachine) recoverAction(ctx context.Context, machine *StateMachine, data interface{}) error {
	// Placeholder for recovery logic
	log.Printf("Recovering deployment %s", sm.ID)
	time.Sleep(800 * time.Millisecond)
	return nil
}

// Error handling actions

func (sm *StateMachine) handleValidationFailure(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Handling validation failure for %s", sm.ID)
	sm.incrementRetryCount()
	return nil
}

func (sm *StateMachine) handlePlanningFailure(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Handling planning failure for %s", sm.ID)
	sm.incrementRetryCount()
	return nil
}

func (sm *StateMachine) handleDeploymentFailure(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Handling deployment failure for %s", sm.ID)
	sm.incrementRetryCount()
	return nil
}

func (sm *StateMachine) handleRollbackFailure(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Handling rollback failure for %s", sm.ID)
	return nil
}

func (sm *StateMachine) handleRecoveryFailure(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Handling recovery failure for %s", sm.ID)
	return nil
}

// Retry actions

func (sm *StateMachine) retryValidation(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Retrying validation for %s (attempt %d)", sm.ID, sm.retryCount+1)
	return sm.validateAction(ctx, machine, data)
}

func (sm *StateMachine) retryPlanning(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Retrying planning for %s (attempt %d)", sm.ID, sm.retryCount+1)
	return sm.planAction(ctx, machine, data)
}

func (sm *StateMachine) retryDeployment(ctx context.Context, machine *StateMachine, data interface{}) error {
	log.Printf("Retrying deployment for %s (attempt %d)", sm.ID, sm.retryCount+1)
	return sm.deployAction(ctx, machine, data)
}

// Guard functions

func (sm *StateMachine) canRetry(ctx context.Context, machine *StateMachine, data interface{}) bool {
	return sm.retryCount < sm.maxRetries
}

// Helper methods

func (sm *StateMachine) incrementRetryCount() {
	sm.retryCount++
}

func (sm *StateMachine) resetRetryCount() {
	sm.retryCount = 0
}

// RetryWithBackoff executes a function with exponential backoff
func (sm *StateMachine) RetryWithBackoff(ctx context.Context, policy RetryPolicy, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := sm.calculateBackoffDelay(policy, attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateBackoffDelay calculates the delay for exponential backoff
func (sm *StateMachine) calculateBackoffDelay(policy RetryPolicy, attempt int) time.Duration {
	delay := float64(policy.InitialDelay) * math.Pow(policy.BackoffFactor, float64(attempt-1))

	if policy.Jitter {
		// Add random jitter up to 10%
		jitter := delay * 0.1 * rand.Float64()
		delay += jitter
	}

	maxDelay := float64(policy.MaxDelay)
	if delay > maxDelay {
		delay = maxDelay
	}

	return time.Duration(delay)
}