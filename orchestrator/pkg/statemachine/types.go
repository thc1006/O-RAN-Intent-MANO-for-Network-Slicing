package statemachine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State represents the current state of a deployment
type State string

const (
	// Initial states
	StateInitializing    State = "initializing"
	StatePending         State = "pending"
	StateValidating      State = "validating"

	// Planning states
	StatePlanning        State = "planning"
	StatePlanned         State = "planned"
	StatePlanValidated   State = "plan_validated"

	// Deployment states
	StateDeploying       State = "deploying"
	StatePartiallyDeployed State = "partially_deployed"
	StateDeployed        State = "deployed"
	StateActive          State = "active"

	// Error states
	StateValidationFailed State = "validation_failed"
	StatePlanningFailed   State = "planning_failed"
	StateDeploymentFailed State = "deployment_failed"
	StateError           State = "error"

	// Recovery states
	StateRecovering      State = "recovering"
	StateRollingBack     State = "rolling_back"
	StateRolledBack      State = "rolled_back"

	// Termination states
	StateTerminating     State = "terminating"
	StateTerminated      State = "terminated"
	StateFailed          State = "failed"
)

// Event represents an event that can trigger a state transition
type Event string

const (
	// Control events
	EventInitialize     Event = "initialize"
	EventValidate       Event = "validate"
	EventPlan           Event = "plan"
	EventDeploy         Event = "deploy"
	EventActivate       Event = "activate"
	EventTerminate      Event = "terminate"
	EventRollback       Event = "rollback"
	EventRecover        Event = "recover"
	EventRetry          Event = "retry"

	// Success events
	EventValidationSuccess Event = "validation_success"
	EventPlanningSuccess   Event = "planning_success"
	EventDeploymentSuccess Event = "deployment_success"
	EventActivationSuccess Event = "activation_success"
	EventRecoverySuccess   Event = "recovery_success"
	EventRollbackSuccess   Event = "rollback_success"

	// Failure events
	EventValidationFailure Event = "validation_failure"
	EventPlanningFailure   Event = "planning_failure"
	EventDeploymentFailure Event = "deployment_failure"
	EventActivationFailure Event = "activation_failure"
	EventRecoveryFailure   Event = "recovery_failure"
	EventRollbackFailure   Event = "rollback_failure"
	EventSystemFailure     Event = "system_failure"
)

// Transition represents a state transition
type Transition struct {
	From   State
	To     State
	Event  Event
	Action ActionFunc
	Guard  GuardFunc
}

// ActionFunc is executed during state transition
type ActionFunc func(ctx context.Context, sm *StateMachine, data interface{}) error

// GuardFunc determines if a transition is allowed
type GuardFunc func(ctx context.Context, sm *StateMachine, data interface{}) bool

// StateMachine manages the lifecycle of a deployment
type StateMachine struct {
	ID              string
	CurrentState    State
	PreviousState   State
	InitialState    State
	transitions     map[State]map[Event]*Transition
	eventHistory    []EventRecord
	stateHistory    []StateRecord
	retryCount      int
	maxRetries      int
	lastError       error
	metadata        map[string]interface{}
	listeners       []StateListener
	mutex           sync.RWMutex
	createdAt       time.Time
	updatedAt       time.Time
}

// EventRecord tracks events in the state machine
type EventRecord struct {
	Event     Event
	Timestamp time.Time
	State     State
	Data      interface{}
	Success   bool
	Error     error
}

// StateRecord tracks state changes
type StateRecord struct {
	From      State
	To        State
	Event     Event
	Timestamp time.Time
	Duration  time.Duration
}

// StateListener receives notifications of state changes
type StateListener interface {
	OnStateChange(ctx context.Context, sm *StateMachine, from, to State, event Event)
	OnError(ctx context.Context, sm *StateMachine, err error)
}

// Config holds configuration for state machine behavior
type Config struct {
	MaxRetries        int
	RetryDelay        time.Duration
	StateTimeout      time.Duration
	EnableHistory     bool
	EnableMetrics     bool
	FailureThreshold  int
	RecoveryStrategy  RecoveryStrategy
}

// RecoveryStrategy defines how to handle failures
type RecoveryStrategy string

const (
	RecoveryStrategyRetry    RecoveryStrategy = "retry"
	RecoveryStrategyRollback RecoveryStrategy = "rollback"
	RecoveryStrategyManual   RecoveryStrategy = "manual"
	RecoveryStrategyIgnore   RecoveryStrategy = "ignore"
)

// Manager manages multiple state machines
type Manager struct {
	machines    map[string]*StateMachine
	config      Config
	listeners   []StateListener
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// DeploymentContext provides context for deployment operations
type DeploymentContext struct {
	SliceID     string
	Intent      interface{}
	Resources   interface{}
	Placement   interface{}
	Timeout     time.Duration
	RetryPolicy RetryPolicy
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid   bool
	Errors  []error
	Warnings []string
	Metadata map[string]interface{}
}

// PlanningResult represents the result of planning
type PlanningResult struct {
	Plan        interface{}
	Resources   interface{}
	Dependencies []string
	Timeline    time.Duration
	Metadata    map[string]interface{}
}

// DeploymentResult represents the result of deployment
type DeploymentResult struct {
	Success    bool
	Resources  interface{}
	Endpoints  []string
	Metrics    map[string]interface{}
	Rollback   RollbackInfo
}

// RollbackInfo contains information needed for rollback
type RollbackInfo struct {
	Enabled       bool
	Snapshot      interface{}
	PreviousState interface{}
	Resources     []string
	Strategy      string
}

// MetricsCollector defines interface for collecting state machine metrics
type MetricsCollector interface {
	RecordTransition(sm *StateMachine, from, to State, duration time.Duration)
	RecordError(sm *StateMachine, err error)
	RecordRetry(sm *StateMachine, attempt int)
	GetMetrics(smID string) map[string]interface{}
}

// Error types
var (
	ErrInvalidTransition    = fmt.Errorf("invalid state transition")
	ErrGuardConditionFailed = fmt.Errorf("guard condition failed")
	ErrActionFailed         = fmt.Errorf("action execution failed")
	ErrStateMachineNotFound = fmt.Errorf("state machine not found")
	ErrMaxRetriesExceeded   = fmt.Errorf("maximum retries exceeded")
	ErrTimeout              = fmt.Errorf("operation timeout")
	ErrInvalidConfiguration = fmt.Errorf("invalid configuration")
)

// DefaultRetryPolicy provides sensible defaults
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// DefaultConfig provides sensible defaults for state machine configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries:        3,
		RetryDelay:        5 * time.Second,
		StateTimeout:      30 * time.Minute,
		EnableHistory:     true,
		EnableMetrics:     true,
		FailureThreshold:  3,
		RecoveryStrategy:  RecoveryStrategyRetry,
	}
}