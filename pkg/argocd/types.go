package argocd

import (
	"time"
)

// ApplicationSpec defines the specification for an ArgoCD application
type ApplicationSpec struct {
	Name        string
	Namespace   string
	Project     string
	Source      ApplicationSource
	Destination ApplicationDestination
	SyncPolicy  *SyncPolicy
	Info        []ApplicationInfo
}

// ApplicationSource defines the source of an application
type ApplicationSource struct {
	RepoURL        string
	Path           string
	TargetRevision string
	Chart          string
	Helm           *HelmParameters
	Kustomize      *KustomizeOptions
}

// HelmParameters defines Helm-specific options
type HelmParameters struct {
	ValueFiles []string
	Parameters []HelmParameter
	Values     string
	ReleaseName string
}

// HelmParameter defines a Helm parameter
type HelmParameter struct {
	Name  string
	Value string
}

// KustomizeOptions defines Kustomize-specific options
type KustomizeOptions struct {
	NamePrefix string
	NameSuffix string
	Images     []string
}

// ApplicationDestination defines the destination cluster and namespace
type ApplicationDestination struct {
	Server    string
	Namespace string
	Name      string
}

// SyncPolicy defines automated sync policy
type SyncPolicy struct {
	Automated   *AutomatedSyncPolicy
	SyncOptions []string
	Retry       *RetryStrategy
}

// AutomatedSyncPolicy defines automated sync behavior
type AutomatedSyncPolicy struct {
	Prune      bool
	SelfHeal   bool
	AllowEmpty bool
}

// RetryStrategy defines retry behavior
type RetryStrategy struct {
	Limit   int
	Backoff *Backoff
}

// Backoff defines backoff strategy
type Backoff struct {
	Duration    string
	Factor      int
	MaxDuration string
}

// Application represents an ArgoCD application
type Application struct {
	Name      string
	Namespace string
	UID       string
	Spec      ApplicationSpec
	Status    ApplicationStatus
	Health    HealthStatus
	Sync      SyncStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ApplicationStatus represents the status of an application
type ApplicationStatus struct {
	Resources      []ResourceStatus
	Sync           SyncStatus
	Health         HealthStatus
	History        []RevisionHistory
	Conditions     []ApplicationCondition
	ReconciledAt   *time.Time
	OperationState *OperationState
}

// ResourceStatus represents the status of a resource
type ResourceStatus struct {
	Group     string
	Kind      string
	Namespace string
	Name      string
	Status    SyncStatusCode
	Health    *HealthStatus
	Version   string
}

// HealthStatus represents health status
type HealthStatus struct {
	Status  HealthStatusCode
	Message string
}

// HealthStatusCode represents health status codes
type HealthStatusCode string

const (
	HealthStatusUnknown     HealthStatusCode = "Unknown"
	HealthStatusProgressing HealthStatusCode = "Progressing"
	HealthStatusHealthy     HealthStatusCode = "Healthy"
	HealthStatusSuspended   HealthStatusCode = "Suspended"
	HealthStatusDegraded    HealthStatusCode = "Degraded"
	HealthStatusMissing     HealthStatusCode = "Missing"
)

// SyncStatus represents sync status
type SyncStatus struct {
	Status     SyncStatusCode
	ComparedTo ComparedTo
	Revision   string
}

// SyncStatusCode represents sync status codes
type SyncStatusCode string

const (
	SyncStatusCodeUnknown   SyncStatusCode = "Unknown"
	SyncStatusCodeSynced    SyncStatusCode = "Synced"
	SyncStatusCodeOutOfSync SyncStatusCode = "OutOfSync"
)

// ComparedTo contains revision comparison info
type ComparedTo struct {
	Source      ApplicationSource
	Destination ApplicationDestination
}

// RevisionHistory contains history information
type RevisionHistory struct {
	Revision   string
	DeployedAt time.Time
	ID         int64
	Source     ApplicationSource
}

// ApplicationCondition contains condition information
type ApplicationCondition struct {
	Type               string
	Message            string
	LastTransitionTime *time.Time
}

// OperationState contains operation state
type OperationState struct {
	Operation  Operation
	Phase      OperationPhase
	Message    string
	SyncResult *SyncResult
	StartedAt  time.Time
	FinishedAt *time.Time
}

// Operation represents an operation
type Operation struct {
	Sync *SyncOperation
}

// SyncOperation represents a sync operation
type SyncOperation struct {
	Revision      string
	Prune         bool
	DryRun        bool
	SyncStrategy  *SyncStrategy
	Resources     []SyncResource
	Source        *ApplicationSource
	Manifests     []string
	SyncOptions   []string
}

// SyncStrategy represents sync strategy
type SyncStrategy struct {
	Apply *ApplyStrategy
	Hook  *HookStrategy
}

// ApplyStrategy represents apply strategy
type ApplyStrategy struct {
	Force bool
}

// HookStrategy represents hook strategy
type HookStrategy struct {
	Force bool
}

// OperationPhase represents operation phase
type OperationPhase string

const (
	OperationPhaseRunning    OperationPhase = "Running"
	OperationPhaseTerminating OperationPhase = "Terminating"
	OperationPhaseFailed     OperationPhase = "Failed"
	OperationPhaseSucceeded  OperationPhase = "Succeeded"
	OperationPhaseError      OperationPhase = "Error"
	OperationPhaseDryRun     OperationPhase = "DryRun"
)

// SyncResult represents sync result
type SyncResult struct {
	Resources []ResourceResult
	Revision  string
	Source    ApplicationSource
}

// ResourceResult represents resource result
type ResourceResult struct {
	Group     string
	Version   string
	Kind      string
	Namespace string
	Name      string
	Status    ResultCode
	Message   string
	HookPhase OperationPhase
	SyncPhase SyncPhase
}

// ResultCode represents result code
type ResultCode string

const (
	ResultCodeSynced    ResultCode = "Synced"
	ResultCodePruned    ResultCode = "Pruned"
	ResultCodeSyncFailed ResultCode = "SyncFailed"
	ResultCodePruneFailed ResultCode = "PruneFailed"
)

// SyncPhase represents sync phase
type SyncPhase string

const (
	SyncPhasePreSync  SyncPhase = "PreSync"
	SyncPhaseSync     SyncPhase = "Sync"
	SyncPhasePostSync SyncPhase = "PostSync"
)

// ApplicationInfo contains application metadata
type ApplicationInfo struct {
	Name  string
	Value string
}

// SyncOptions defines sync options
type SyncOptions struct {
	Prune     bool
	DryRun    bool
	Strategy  SyncStrategyType
	Resources []SyncResource
}

// SyncStrategyType represents sync strategy type
type SyncStrategyType string

const (
	SyncStrategyApply SyncStrategyType = "Apply"
	SyncStrategyHook  SyncStrategyType = "Hook"
)

// SyncResource represents a resource to sync
type SyncResource struct {
	Group string
	Kind  string
	Name  string
}

// RollbackOptions defines rollback options
type RollbackOptions struct {
	Revision  string
	HistoryID int64
	DryRun    bool
	Prune     bool
}

// ClusterConfig defines cluster configuration
type ClusterConfig struct {
	Name       string
	Server     string
	Region     string
	Config     string
	Namespaces []string
}

// MultiClusterDeploymentResult represents multi-cluster deployment result
type MultiClusterDeploymentResult struct {
	Applications  []*Application
	SuccessCount  int
	FailureCount  int
	Errors        []error
	DeploymentID  string
}

// ProgressiveDeliveryStrategy defines progressive delivery strategy
type ProgressiveDeliveryStrategy struct {
	Type      ProgressiveDeliveryType
	Steps     []CanaryStep
	Analysis  *AnalysisTemplate
	BlueGreen *BlueGreenStrategy
}

// ProgressiveDeliveryType represents progressive delivery type
type ProgressiveDeliveryType string

const (
	CanaryDeployment    ProgressiveDeliveryType = "Canary"
	BlueGreenDeployment ProgressiveDeliveryType = "BlueGreen"
)

// CanaryStep defines a canary rollout step
type CanaryStep struct {
	Weight   int
	Pause    *PauseDuration
	Analysis *AnalysisRun
}

// PauseDuration defines pause duration
type PauseDuration struct {
	Duration time.Duration
}

// AnalysisRun defines an analysis run
type AnalysisRun struct {
	Template string
	Args     []AnalysisArgument
}

// AnalysisArgument defines analysis argument
type AnalysisArgument struct {
	Name  string
	Value string
}

// AnalysisTemplate defines analysis template
type AnalysisTemplate struct {
	Metrics []Metric
}

// Metric defines a metric
type Metric struct {
	Name             string
	Interval         string
	Count            int
	SuccessCondition string
	FailureCondition string
	Provider         MetricProvider
}

// MetricProvider defines metric provider
type MetricProvider struct {
	Prometheus *PrometheusMetric
}

// PrometheusMetric defines Prometheus metric
type PrometheusMetric struct {
	Address string
	Query   string
}

// BlueGreenStrategy defines blue-green strategy
type BlueGreenStrategy struct {
	ActiveService         string
	PreviewService        string
	AutoPromotionEnabled  bool
	ScaleDownDelaySeconds int
	PrePromotionAnalysis  *AnalysisRun
}

// ProgressiveDeliveryResult represents progressive delivery result
type ProgressiveDeliveryResult struct {
	Status           ProgressiveDeliveryStatus
	CurrentStep      int
	CurrentWeight    int
	RolloutID        string
	BlueGreenActive  bool
	Message          string
	HistoryID        int64
	Revision         string
	Phase            OperationPhase
	Resources        []ResourceResult // For test verification
	StartedAt        time.Time
	FinishedAt       *time.Time
}

// ProgressiveDeliveryStatus represents progressive delivery status
type ProgressiveDeliveryStatus string

const (
	ProgressiveDeliveryStatusInProgress ProgressiveDeliveryStatus = "InProgress"
	ProgressiveDeliveryStatusSucceeded  ProgressiveDeliveryStatus = "Succeeded"
	ProgressiveDeliveryStatusFailed     ProgressiveDeliveryStatus = "Failed"
	ProgressiveDeliveryStatusAborted    ProgressiveDeliveryStatus = "Aborted"
	ProgressiveDeliveryStatusPaused     ProgressiveDeliveryStatus = "Paused"
)

// PromoteOptions defines promotion options
type PromoteOptions struct {
	Full         bool
	SkipAnalysis bool
}

// ApplicationEvent represents an application event
type ApplicationEvent struct {
	Type      string
	Reason    string
	Message   string
	Timestamp time.Time
}
