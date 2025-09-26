package o2dms

import (
	"sync"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/models"
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

// Event represents an O2 DMS event
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	ResourceID  string                 `json:"resource_id,omitempty"`
	Data        map[string]interface{} `json:"data"`
	Severity    EventSeverity          `json:"severity"`
}

// EventType represents different types of O2 DMS events
type EventType string

const (
	// Deployment Manager Events
	EventTypeDeploymentManagerListed EventType = "deployment_manager.listed"
	EventTypeDeploymentManagerCreated EventType = "deployment_manager.created"
	EventTypeDeploymentManagerUpdated EventType = "deployment_manager.updated"
	EventTypeDeploymentManagerDeleted EventType = "deployment_manager.deleted"

	// NF Deployment Events
	EventTypeDeploymentCreated      EventType = "deployment.created"
	EventTypeDeploymentUpdated      EventType = "deployment.updated"
	EventTypeDeploymentDeleted      EventType = "deployment.deleted"
	EventTypeDeploymentFailed       EventType = "deployment.failed"
	EventTypeDeploymentReady        EventType = "deployment.ready"
	EventTypeDeploymentDeleteFailed EventType = "deployment.delete_failed"

	// Network Slice Events
	EventTypeSliceDeployed EventType = "slice.deployed"
	EventTypeSliceFailed   EventType = "slice.failed"
	EventTypeSliceDeleted  EventType = "slice.deleted"

	// Health Events
	EventTypeHealthCheckSuccess EventType = "health.check.success"
	EventTypeHealthCheckFailed  EventType = "health.check.failed"

	// Subscription Events
	EventTypeSubscriptionCreated EventType = "subscription.created"
	EventTypeSubscriptionDeleted EventType = "subscription.deleted"
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

// NetworkSliceSpec represents the specification for deploying a network slice
type NetworkSliceSpec struct {
	SliceID           string                       `json:"sliceId"`
	SliceType         string                       `json:"sliceType"`
	QoSRequirements   *models.ORanQoSRequirements  `json:"qosRequirements"`
	Placement         *models.ORanPlacement        `json:"placement"`
	NetworkFunctions  []NetworkFunctionSpec        `json:"networkFunctions"`
	WaitForReady      bool                         `json:"waitForReady"`
	DeploymentTimeout time.Duration                `json:"deploymentTimeout"`
	Dependencies      map[string][]string          `json:"dependencies,omitempty"`
	Extensions        map[string]interface{}       `json:"extensions,omitempty"`
}

// NetworkFunctionSpec represents a network function to be deployed
type NetworkFunctionSpec struct {
	Type         string                 `json:"type"`
	DescriptorID string                 `json:"descriptorId"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// NetworkSliceDeployment represents the result of a network slice deployment
type NetworkSliceDeployment struct {
	SliceID             string                  `json:"sliceId"`
	Status              SliceStatus             `json:"status"`
	DeploymentManagerID string                  `json:"deploymentManagerId"`
	NFDeployments       []*models.NFDeployment  `json:"nfDeployments"`
	Error               string                  `json:"error,omitempty"`
	CreatedAt           time.Time               `json:"createdAt"`
	UpdatedAt           time.Time               `json:"updatedAt"`
	Extensions          map[string]interface{}  `json:"extensions,omitempty"`
}

// SliceStatus represents the status of a network slice deployment
type SliceStatus string

const (
	SliceStatusPending   SliceStatus = "pending"
	SliceStatusDeploying SliceStatus = "deploying"
	SliceStatusDeployed  SliceStatus = "deployed"
	SliceStatusFailed    SliceStatus = "failed"
	SliceStatusDeleting  SliceStatus = "deleting"
	SliceStatusDeleted   SliceStatus = "deleted"
)

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

// DeploymentStrategy defines how deployments should be managed
type DeploymentStrategy struct {
	Type                string                 `json:"type"`                // sequential, parallel, rolling
	MaxConcurrent       int                    `json:"maxConcurrent"`       // for parallel deployments
	RollingUpdateConfig *RollingUpdateConfig   `json:"rollingUpdate,omitempty"`
	RetryStrategy       *RetryStrategy         `json:"retryStrategy,omitempty"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`
}

// RollingUpdateConfig defines rolling update parameters
type RollingUpdateConfig struct {
	MaxUnavailable int           `json:"maxUnavailable"`
	MaxSurge       int           `json:"maxSurge"`
	UpdateInterval time.Duration `json:"updateInterval"`
}

// RetryStrategy defines retry behavior for deployments
type RetryStrategy struct {
	MaxAttempts      int           `json:"maxAttempts"`
	BackoffStrategy  string        `json:"backoffStrategy"`  // exponential, linear, fixed
	InitialDelay     time.Duration `json:"initialDelay"`
	MaxDelay         time.Duration `json:"maxDelay"`
	BackoffMultiplier float64      `json:"backoffMultiplier"`
}

// DeploymentContext provides additional context for deployments
type DeploymentContext struct {
	UserID               string                 `json:"userId"`
	TenantID             string                 `json:"tenantId"`
	RequestID            string                 `json:"requestId"`
	Priority             int                    `json:"priority"`
	DeploymentStrategy   *DeploymentStrategy    `json:"deploymentStrategy,omitempty"`
	ResourceConstraints  *ResourceConstraints   `json:"resourceConstraints,omitempty"`
	SecurityContext      *SecurityContext       `json:"securityContext,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceConstraints defines resource limits and requirements
type ResourceConstraints struct {
	CPU               string                 `json:"cpu,omitempty"`
	Memory            string                 `json:"memory,omitempty"`
	Storage           string                 `json:"storage,omitempty"`
	NetworkBandwidth  string                 `json:"networkBandwidth,omitempty"`
	GPUCount          int                    `json:"gpuCount,omitempty"`
	NodeSelector      map[string]string      `json:"nodeSelector,omitempty"`
	Tolerations       []Toleration           `json:"tolerations,omitempty"`
	Affinity          *Affinity              `json:"affinity,omitempty"`
	Extensions        map[string]interface{} `json:"extensions,omitempty"`
}

// Toleration represents a toleration for node taints
type Toleration struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
	Effect   string `json:"effect"`
}

// Affinity represents node affinity rules
type Affinity struct {
	NodeAffinity *NodeAffinity `json:"nodeAffinity,omitempty"`
	PodAffinity  *PodAffinity  `json:"podAffinity,omitempty"`
}

// NodeAffinity represents node affinity
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector   `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []NodeSelector  `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinity represents pod affinity
type PodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelector represents node selection criteria
type NodeSelector struct {
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty"`
	MatchLabels      map[string]string `json:"matchLabels,omitempty"`
}

// MatchExpression represents a label selector requirement
type MatchExpression struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// PodAffinityTerm represents pod affinity term
type PodAffinityTerm struct {
	LabelSelector *LabelSelector `json:"labelSelector,omitempty"`
	Namespaces    []string       `json:"namespaces,omitempty"`
	TopologyKey   string         `json:"topologyKey"`
}

// LabelSelector represents a label selector
type LabelSelector struct {
	MatchLabels      map[string]string `json:"matchLabels,omitempty"`
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty"`
}

// SecurityContext defines security settings for deployments
type SecurityContext struct {
	RunAsUser    *int64                 `json:"runAsUser,omitempty"`
	RunAsGroup   *int64                 `json:"runAsGroup,omitempty"`
	RunAsNonRoot *bool                  `json:"runAsNonRoot,omitempty"`
	ReadOnlyRootFilesystem *bool        `json:"readOnlyRootFilesystem,omitempty"`
	Capabilities *Capabilities          `json:"capabilities,omitempty"`
	SELinuxOptions *SELinuxOptions      `json:"seLinuxOptions,omitempty"`
	Extensions   map[string]interface{} `json:"extensions,omitempty"`
}

// Capabilities represents container capabilities
type Capabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// SELinuxOptions represents SELinux options
type SELinuxOptions struct {
	User  string `json:"user,omitempty"`
	Role  string `json:"role,omitempty"`
	Type  string `json:"type,omitempty"`
	Level string `json:"level,omitempty"`
}