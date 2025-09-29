package e2e

import "time"

// E2EConfig configuration for E2E orchestrator
type E2EConfig struct {
	PorchEndpoint   string
	Namespace       string
	Repository      string
	GitRepo         string
	ArgoCDNamespace string
	PrometheusURL   string
}

// E2EResult result of complete E2E flow
type E2EResult struct {
	Intent           string            `json:"intent"`
	Success          bool              `json:"success"`
	SliceID          string            `json:"sliceId"`
	Steps            []StepResult      `json:"steps"`
	DeploymentStatus *DeploymentStatus `json:"deploymentStatus"`
	Timestamp        time.Time         `json:"timestamp"`
	Error            string            `json:"error,omitempty"`
}

// StepResult result of individual step
type StepResult struct {
	Name      string      `json:"name"`
	Status    string      `json:"status"`
	Details   interface{} `json:"details"`
	Timestamp time.Time   `json:"timestamp"`
	Error     string      `json:"error,omitempty"`
}

// DeploymentStatus Kubernetes deployment status
type DeploymentStatus struct {
	AppName   string `json:"appName"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Health    string `json:"health"`
	Sync      string `json:"sync"`
	Ready     bool   `json:"ready"`
}

// Metrics collected metrics
type Metrics struct {
	SliceType      string    `json:"sliceType"`
	Throughput     string    `json:"throughput"`
	Latency        string    `json:"latency"`
	ActiveSessions string    `json:"activeSessions"`
	Timestamp      time.Time `json:"timestamp"`
}

// Nephio Package Types

// KptFile represents Kptfile structure
type KptFile struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   KptMetadata  `yaml:"metadata"`
	Info       KptInfo      `yaml:"info"`
	Pipeline   KptPipeline  `yaml:"pipeline"`
}

type KptMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Annotations map[string]string `yaml:"annotations"`
}

type KptInfo struct {
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
}

type KptPipeline struct {
	Mutators []KptFunction `yaml:"mutators"`
}

type KptFunction struct {
	Image     string                 `yaml:"image"`
	ConfigMap map[string]interface{} `yaml:"configMap"`
}

// NetworkSliceCR represents NetworkSlice custom resource
type NetworkSliceCR struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   Metadata         `yaml:"metadata"`
	Spec       NetworkSliceSpec `yaml:"spec"`
}

type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

type NetworkSliceSpec struct {
	SliceType        string            `yaml:"sliceType"`
	QoS              QoSSpec           `yaml:"qos"`
	NetworkFunctions []NetworkFunction `yaml:"networkFunctions"`
	Capacity         CapacitySpec      `yaml:"capacity"`
}

type QoSSpec struct {
	Throughput  int     `yaml:"throughput"`
	Latency     float64 `yaml:"latency"`
	Reliability float64 `yaml:"reliability"`
}

type NetworkFunction struct {
	Name     string `yaml:"name"`
	Replicas int    `yaml:"replicas"`
}

type CapacitySpec struct {
	MaxSessions int `yaml:"maxSessions"`
}

// ArgoCD Types

// ArgoCDApplication represents ArgoCD Application
type ArgoCDApplication struct {
	APIVersion string        `yaml:"apiVersion"`
	Kind       string        `yaml:"kind"`
	Metadata   Metadata      `yaml:"metadata"`
	Spec       ArgoCDAppSpec `yaml:"spec"`
}

type ArgoCDAppSpec struct {
	Project     string             `yaml:"project"`
	Source      ArgoCDSource       `yaml:"source"`
	Destination ArgoCDDestination  `yaml:"destination"`
	SyncPolicy  *ArgoCDSyncPolicy  `yaml:"syncPolicy,omitempty"`
}

type ArgoCDSource struct {
	RepoURL        string `yaml:"repoURL"`
	Path           string `yaml:"path"`
	TargetRevision string `yaml:"targetRevision"`
}

type ArgoCDDestination struct {
	Server    string `yaml:"server"`
	Namespace string `yaml:"namespace"`
}

type ArgoCDSyncPolicy struct {
	Automated   *ArgoCDAutomated `yaml:"automated,omitempty"`
	SyncOptions []string         `yaml:"syncOptions,omitempty"`
	Retry       *ArgoCDRetry     `yaml:"retry,omitempty"`
}

type ArgoCDAutomated struct {
	Prune      bool `yaml:"prune"`
	SelfHeal   bool `yaml:"selfHeal"`
	AllowEmpty bool `yaml:"allowEmpty"`
}

type ArgoCDRetry struct {
	Limit   int            `yaml:"limit"`
	Backoff *ArgoCDBackoff `yaml:"backoff,omitempty"`
}

type ArgoCDBackoff struct {
	Duration    string `yaml:"duration"`
	Factor      int    `yaml:"factor"`
	MaxDuration string `yaml:"maxDuration"`
}