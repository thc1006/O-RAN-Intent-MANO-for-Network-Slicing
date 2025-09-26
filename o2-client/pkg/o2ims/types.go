package o2ims

import (
	"sync"
	"time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Timeout       time.Duration
}

// DefaultRetryConfig provides sensible retry defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Timeout:       60 * time.Second,
	}
}

// Event represents an O2 IMS event
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	ResourceID  string                 `json:"resource_id,omitempty"`
	Data        map[string]interface{} `json:"data"`
	Severity    EventSeverity          `json:"severity"`
}

// EventType represents different types of O2 IMS events
type EventType string

const (
	EventTypeResourceCreated       EventType = "resource.created"
	EventTypeResourceUpdated       EventType = "resource.updated"
	EventTypeResourceDeleted       EventType = "resource.deleted"
	EventTypeResourcePoolChanged   EventType = "resource_pool.changed"
	EventTypeDeploymentCreated     EventType = "deployment.created"
	EventTypeDeploymentUpdated     EventType = "deployment.updated"
	EventTypeDeploymentDeleted     EventType = "deployment.deleted"
	EventTypeAlarmCreated          EventType = "alarm.created"
	EventTypeAlarmCleared          EventType = "alarm.cleared"
	EventTypeInventoryChanged      EventType = "inventory.changed"
	EventTypeSubscriptionCreated   EventType = "subscription.created"
	EventTypeSubscriptionDeleted   EventType = "subscription.deleted"
)

// EventSeverity represents event severity levels
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityError    EventSeverity = "error"
	SeverityCritical EventSeverity = "critical"
)

// EventHandler processes events
type EventHandler func(event Event)

// ClientMetrics tracks client performance metrics
type ClientMetrics struct {
	RequestCount    int64
	ErrorCount      int64
	RetryCount      int64
	AvgResponseTime time.Duration
	LastRequestTime time.Time
	mutex           sync.RWMutex
}

// NewClientMetrics creates a new metrics instance
func NewClientMetrics() *ClientMetrics {
	return &ClientMetrics{}
}

// RecordRequest records a successful request
func (m *ClientMetrics) RecordRequest(duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.RequestCount++
	m.LastRequestTime = time.Now()

	// Calculate moving average
	if m.AvgResponseTime == 0 {
		m.AvgResponseTime = duration
	} else {
		m.AvgResponseTime = (m.AvgResponseTime + duration) / 2
	}
}

// RecordError records an error
func (m *ClientMetrics) RecordError() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ErrorCount++
}

// RecordRetry records a retry attempt
func (m *ClientMetrics) RecordRetry() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.RetryCount++
}

// GetMetrics returns current metrics
func (m *ClientMetrics) GetMetrics() (int64, int64, int64, time.Duration) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.RequestCount, m.ErrorCount, m.RetryCount, m.AvgResponseTime
}

// ConnectionState represents the connection state
type ConnectionState string

const (
	ConnectionStateConnected    ConnectionState = "connected"
	ConnectionStateDisconnected ConnectionState = "disconnected"
	ConnectionStateConnecting   ConnectionState = "connecting"
	ConnectionStateError        ConnectionState = "error"
)

// ClientState holds the current state of the client
type ClientState struct {
	Connection       ConnectionState
	LastHealthCheck  time.Time
	HealthCheckError error
	EventsEnabled    bool
	mutex            sync.RWMutex
}

// GetConnectionState returns the current connection state
func (s *ClientState) GetConnectionState() ConnectionState {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Connection
}

// SetConnectionState sets the connection state
func (s *ClientState) SetConnectionState(state ConnectionState) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Connection = state
}

// GetLastHealthCheck returns the last health check time and error
func (s *ClientState) GetLastHealthCheck() (time.Time, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.LastHealthCheck, s.HealthCheckError
}

// UpdateHealthCheck updates the health check status
func (s *ClientState) UpdateHealthCheck(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.LastHealthCheck = time.Now()
	s.HealthCheckError = err
}

// IsEventsEnabled returns whether events are enabled
func (s *ClientState) IsEventsEnabled() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.EventsEnabled
}

// SetEventsEnabled sets the events enabled state
func (s *ClientState) SetEventsEnabled(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.EventsEnabled = enabled
}