package apis

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Application is a definition of Application resource in Argo CD
// This is a simplified version to avoid Argo CD v2 dependency conflicts
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec   `json:"spec"`
	Status            ApplicationStatus `json:"status,omitempty"`
}

// ApplicationSpec represents desired application state
type ApplicationSpec struct {
	// Source is a reference to the location of the application's manifests
	Source *ApplicationSource `json:"source,omitempty"`
	// Destination is a reference to the target Kubernetes server and namespace
	Destination ApplicationDestination `json:"destination"`
	// Project is a reference to the project this application belongs to
	Project string `json:"project"`
	// SyncPolicy controls when and how a sync will be performed
	SyncPolicy *SyncPolicy `json:"syncPolicy,omitempty"`
}

// ApplicationSource contains all required information about the source of an application
type ApplicationSource struct {
	// RepoURL is the URL to the repository (Git, Helm, etc)
	RepoURL string `json:"repoURL"`
	// Path is a directory path within the repository
	Path string `json:"path,omitempty"`
	// Plugin holds config management plugin specific options
	Plugin *ApplicationSourcePlugin `json:"plugin,omitempty"`
}

// ApplicationSourcePlugin holds config management plugin specific options
type ApplicationSourcePlugin struct {
	// Name of the plugin
	Name string `json:"name,omitempty"`
	// Env is a list of environment variable entries
	Env []EnvEntry `json:"env,omitempty"`
}

// EnvEntry represents an entry in the application's environment
type EnvEntry struct {
	// Name is the name of the variable
	Name string `json:"name"`
	// Value is the value of the variable
	Value string `json:"value"`
}

// ApplicationDestination holds information about the destination of application deployment
type ApplicationDestination struct {
	// Server overrides the environment server value in the application spec
	Server string `json:"server,omitempty"`
	// Namespace specifies the target namespace for the application's resources
	Namespace string `json:"namespace,omitempty"`
}

// SyncPolicy controls when a sync will be performed in response to updates in git
type SyncPolicy struct {
	// Automated will keep an application synced to the target revision
	Automated *SyncPolicyAutomated `json:"automated,omitempty"`
	// Options allow you to specify whole app sync-options
	SyncOptions []string `json:"syncOptions,omitempty"`
}

// SyncPolicyAutomated controls the behavior of an automated sync
type SyncPolicyAutomated struct {
	// Prune specifies whether to delete resources from the cluster that are not found in the sources anymore as part of automated sync
	Prune *bool `json:"prune,omitempty"`
	// SelfHeal specifies whether to revert resources back to their desired state upon modification in the cluster
	SelfHeal *bool `json:"selfHeal,omitempty"`
}

// ApplicationStatus contains status information for the application
type ApplicationStatus struct {
	// Sync contains information about the application's current sync status
	Sync SyncStatus `json:"sync,omitempty"`
	// Health contains information about the application's current health status
	Health HealthStatus `json:"health,omitempty"`
}

// SyncStatus contains information about the currently observed live and desired states of an application
type SyncStatus struct {
	// Status is the sync state of the comparison
	Status SyncStatusCode `json:"status"`
}

// SyncStatusCode is a type which represents possible comparison results
type SyncStatusCode string

const (
	// SyncStatusCodeUnknown indicates that the status of a sync could not be reliably determined
	SyncStatusCodeUnknown SyncStatusCode = "Unknown"
	// SyncStatusCodeSynced indicates that the desired and live states match
	SyncStatusCodeSynced SyncStatusCode = "Synced"
	// SyncStatusCodeOutOfSync indicates that there is a drift between desired and live states
	SyncStatusCodeOutOfSync SyncStatusCode = "OutOfSync"
)

// HealthStatus contains information about the currently observed health state of an application
type HealthStatus struct {
	// Status holds the status code of the application or resource
	Status HealthStatusCode `json:"status,omitempty"`
}

// HealthStatusCode is a type which represents possible health states
type HealthStatusCode string

const (
	// HealthStatusUnknown indicates that health assessment failed and actual health status is unknown
	HealthStatusUnknown HealthStatusCode = "Unknown"
	// HealthStatusProgressing indicates that the resource is not healthy but still making progress
	HealthStatusProgressing HealthStatusCode = "Progressing"
	// HealthStatusHealthy indicates resource is 100% healthy
	HealthStatusHealthy HealthStatusCode = "Healthy"
	// HealthStatusDegraded indicates resource status is degraded
	HealthStatusDegraded HealthStatusCode = "Degraded"
)