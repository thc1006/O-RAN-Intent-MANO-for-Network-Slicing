package statemachine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// NewManager creates a new state machine manager
func NewManager(config Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		machines:  make(map[string]*StateMachine),
		config:    config,
		listeners: make([]StateListener, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// CreateStateMachine creates a new state machine
func (m *Manager) CreateStateMachine(id string, initialState State) (*StateMachine, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.machines[id]; exists {
		return nil, fmt.Errorf("state machine with ID %s already exists", id)
	}

	sm := NewStateMachine(id, initialState, m.config)

	// Add manager listeners to the state machine
	for _, listener := range m.listeners {
		sm.AddListener(listener)
	}

	m.machines[id] = sm
	return sm, nil
}

// GetStateMachine retrieves a state machine by ID
func (m *Manager) GetStateMachine(id string) (*StateMachine, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sm, exists := m.machines[id]
	if !exists {
		return nil, ErrStateMachineNotFound
	}

	return sm, nil
}

// RemoveStateMachine removes a state machine
func (m *Manager) RemoveStateMachine(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.machines[id]; !exists {
		return ErrStateMachineNotFound
	}

	delete(m.machines, id)
	return nil
}

// ListStateMachines returns all state machines
func (m *Manager) ListStateMachines() map[string]*StateMachine {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*StateMachine)
	for id, sm := range m.machines {
		result[id] = sm
	}
	return result
}

// AddGlobalListener adds a listener to all state machines
func (m *Manager) AddGlobalListener(listener StateListener) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.listeners = append(m.listeners, listener)

	// Add to existing state machines
	for _, sm := range m.machines {
		sm.AddListener(listener)
	}
}

// SendEventToAll sends an event to all state machines
func (m *Manager) SendEventToAll(ctx context.Context, event Event, data interface{}) map[string]error {
	m.mutex.RLock()
	machines := make(map[string]*StateMachine)
	for id, sm := range m.machines {
		machines[id] = sm
	}
	m.mutex.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup

	for id, sm := range machines {
		wg.Add(1)
		go func(smID string, machine *StateMachine) {
			defer wg.Done()
			if err := machine.SendEvent(ctx, event, data); err != nil {
				results[smID] = err
			}
		}(id, sm)
	}

	wg.Wait()
	return results
}

// GetStateMachinesByState returns all state machines in a specific state
func (m *Manager) GetStateMachinesByState(state State) []*StateMachine {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*StateMachine
	for _, sm := range m.machines {
		if sm.GetCurrentState() == state {
			result = append(result, sm)
		}
	}
	return result
}

// GetStatistics returns statistics about managed state machines
func (m *Manager) GetStatistics() ManagerStatistics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := ManagerStatistics{
		TotalMachines:   len(m.machines),
		StateDistribution: make(map[State]int),
		CreatedAt:       time.Now(),
	}

	for _, sm := range m.machines {
		state := sm.GetCurrentState()
		stats.StateDistribution[state]++

		if sm.lastError != nil {
			stats.ErrorCount++
		}
	}

	return stats
}

// StartHealthChecks starts periodic health checking of state machines
func (m *Manager) StartHealthChecks(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.performHealthCheck()
			}
		}
	}()
}

// performHealthCheck checks the health of all state machines
func (m *Manager) performHealthCheck() {
	m.mutex.RLock()
	machines := make([]*StateMachine, 0, len(m.machines))
	for _, sm := range m.machines {
		machines = append(machines, sm)
	}
	m.mutex.RUnlock()

	for _, sm := range machines {
		sm.mutex.RLock()
		timeSinceUpdate := time.Since(sm.updatedAt)
		currentState := sm.CurrentState
		sm.mutex.RUnlock()

		// Check for stuck state machines
		if timeSinceUpdate > m.config.StateTimeout {
			log.Printf("State machine %s appears stuck in state %s for %v",
				sm.ID, currentState, timeSinceUpdate)

			// Attempt recovery based on strategy
			if err := m.attemptRecovery(sm); err != nil {
				log.Printf("Recovery failed for state machine %s: %v", sm.ID, err)
			}
		}
	}
}

// attemptRecovery attempts to recover a stuck state machine
func (m *Manager) attemptRecovery(sm *StateMachine) error {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	switch m.config.RecoveryStrategy {
	case RecoveryStrategyRetry:
		return sm.SendEvent(ctx, EventRetry, nil)
	case RecoveryStrategyRollback:
		return sm.SendEvent(ctx, EventRollback, nil)
	case RecoveryStrategyManual:
		log.Printf("Manual recovery required for state machine %s", sm.ID)
		return nil
	default:
		return sm.SendEvent(ctx, EventRecover, nil)
	}
}

// StartConcurrentDeployments starts multiple deployments concurrently
func (m *Manager) StartConcurrentDeployments(ctx context.Context, deployments []DeploymentContext) error {
	if len(deployments) == 0 {
		return fmt.Errorf("no deployments provided")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(deployments))
	semaphore := make(chan struct{}, m.config.MaxRetries) // Limit concurrent deployments

	for _, deployment := range deployments {
		wg.Add(1)
		go func(dep DeploymentContext) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := m.executeDeployment(ctx, dep); err != nil {
				errChan <- fmt.Errorf("deployment %s failed: %w", dep.SliceID, err)
			}
		}(deployment)
	}

	// Wait for all deployments to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("concurrent deployment failures: %v", errors)
	}

	return nil
}

// executeDeployment executes a single deployment
func (m *Manager) executeDeployment(ctx context.Context, deployment DeploymentContext) error {
	// Create state machine for this deployment
	sm, err := m.CreateStateMachine(deployment.SliceID, StateInitializing)
	if err != nil {
		return fmt.Errorf("failed to create state machine: %w", err)
	}

	// Set deployment metadata
	sm.SetMetadata("intent", deployment.Intent)
	sm.SetMetadata("resources", deployment.Resources)
	sm.SetMetadata("placement", deployment.Placement)
	sm.SetMetadata("timeout", deployment.Timeout)
	sm.SetMetadata("retry_policy", deployment.RetryPolicy)

	// Execute deployment workflow
	deploymentCtx := ctx
	if deployment.Timeout > 0 {
		var cancel context.CancelFunc
		deploymentCtx, cancel = context.WithTimeout(ctx, deployment.Timeout)
		defer cancel()
	}

	// Validation phase
	if err := sm.SendEvent(deploymentCtx, EventValidate, deployment); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := sm.SendEvent(deploymentCtx, EventValidationSuccess, nil); err != nil {
		return fmt.Errorf("validation transition failed: %w", err)
	}

	// Planning phase
	if err := sm.SendEvent(deploymentCtx, EventPlan, deployment); err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}

	if err := sm.SendEvent(deploymentCtx, EventPlanningSuccess, nil); err != nil {
		return fmt.Errorf("planning transition failed: %w", err)
	}

	// Deployment phase with retry
	deployErr := sm.RetryWithBackoff(deploymentCtx, deployment.RetryPolicy, func() error {
		if err := sm.SendEvent(deploymentCtx, EventDeploy, deployment); err != nil {
			return err
		}
		return sm.SendEvent(deploymentCtx, EventDeploymentSuccess, nil)
	})

	if deployErr != nil {
		// Attempt rollback on deployment failure
		if rollbackErr := sm.SendEvent(deploymentCtx, EventRollback, deployment); rollbackErr != nil {
			return fmt.Errorf("deployment failed and rollback failed: deploy=%w, rollback=%w",
				deployErr, rollbackErr)
		}
		return fmt.Errorf("deployment failed but rollback succeeded: %w", deployErr)
	}

	// Activation phase
	if err := sm.SendEvent(deploymentCtx, EventActivate, deployment); err != nil {
		return fmt.Errorf("activation failed: %w", err)
	}

	log.Printf("Deployment %s completed successfully", deployment.SliceID)
	return nil
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.cancel()

	// Wait for all state machines to reach a stable state
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		stable := true
		for _, sm := range m.machines {
			state := sm.GetCurrentState()
			if state == StateDeploying || state == StateRollingBack || state == StateRecovering {
				stable = false
				break
			}
		}

		if stable {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return nil
}

// ManagerStatistics provides statistics about the state machine manager
type ManagerStatistics struct {
	TotalMachines     int
	StateDistribution map[State]int
	ErrorCount        int
	CreatedAt         time.Time
}

// DefaultStateListener provides a basic state change listener implementation
type DefaultStateListener struct{}

func (l *DefaultStateListener) OnStateChange(ctx context.Context, sm *StateMachine, from, to State, event Event) {
	log.Printf("State machine %s: %s -> %s (event: %s)", sm.ID, from, to, event)
}

func (l *DefaultStateListener) OnError(ctx context.Context, sm *StateMachine, err error) {
	log.Printf("State machine %s error: %v", sm.ID, err)
}

// MetricsStateListener collects metrics about state machine operations
type MetricsStateListener struct {
	collector MetricsCollector
}

func NewMetricsStateListener(collector MetricsCollector) *MetricsStateListener {
	return &MetricsStateListener{collector: collector}
}

func (l *MetricsStateListener) OnStateChange(ctx context.Context, sm *StateMachine, from, to State, event Event) {
	// Calculate transition duration
	var duration time.Duration
	history := sm.GetHistory()
	if len(history) > 0 {
		duration = history[len(history)-1].Duration
	}

	l.collector.RecordTransition(sm, from, to, duration)
}

func (l *MetricsStateListener) OnError(ctx context.Context, sm *StateMachine, err error) {
	l.collector.RecordError(sm, err)
}