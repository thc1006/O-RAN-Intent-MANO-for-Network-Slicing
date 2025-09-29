package deployment

import (
	"time"
)

// Component types
type ComponentType string

const (
	ComponentTypeCUCP ComponentType = "CU-CP"
	ComponentTypeCUUP ComponentType = "CU-UP"
	ComponentTypeDU   ComponentType = "DU"
	ComponentTypeRU   ComponentType = "RU"
)

// Deployment status
type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "Pending"
	DeploymentStatusRunning   DeploymentStatus = "Running"
	DeploymentStatusFailed    DeploymentStatus = "Failed"
	DeploymentStatusCompleted DeploymentStatus = "Completed"
)

// RANComponentSpec defines RAN component specification
type RANComponentSpec struct {
	Type          ComponentType
	Name          string
	Namespace     string
	SiteID        string
	Cells         []string
	Resources     ResourceRequirements
	NetworkConfig NetworkConfig
	QoS           QoSRequirements
	Accelerator   *AcceleratorConfig
	Fronthaul     *FronthaulConfig
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	CPU     string
	Memory  string
	Storage string
	GPU     string
}

// NetworkConfig defines network configuration
type NetworkConfig struct {
	N2Interface string
	N3Interface string
	F1Interface string
	E1Interface string
	XnInterface string
}

// QoSRequirements defines QoS requirements
type QoSRequirements struct {
	Bandwidth   int
	Latency     int
	Reliability float64
	Jitter      int
}

// AcceleratorConfig defines hardware accelerator config
type AcceleratorConfig struct {
	Type   string
	Vendor string
	Model  string
	Count  int
}

// FronthaulConfig defines fronthaul configuration
type FronthaulConfig struct {
	Protocol    string
	VLANID      int
	MTU         int
	Compression string
}

// DeploymentResult represents deployment result
type DeploymentResult struct {
	DeploymentName     string
	Status             DeploymentStatus
	PodIPs             []string
	ServiceEndpoint    string
	AcceleratorEnabled bool
	FronthaulConfig    *FronthaulConfig
	Timestamp          time.Time
}

// CoreNetworkSpec defines 5G Core specification
type CoreNetworkSpec struct {
	Name       string
	Namespace  string
	Version    string
	Components []CoreComponent
	Database   DatabaseConfig
}

// CoreComponent defines a core network component
type CoreComponent struct {
	Type     string
	Replicas int
}

// DatabaseConfig defines database configuration
type DatabaseConfig struct {
	Type     string
	Replicas int
	Storage  string
}

// CoreDeploymentResult represents core deployment result
type CoreDeploymentResult struct {
	DeploymentName    string
	ComponentStatuses []ComponentStatus
}

// ComponentStatus represents component status
type ComponentStatus struct {
	Name      string
	Type      string
	Status    DeploymentStatus
	Endpoints []string
}

// UPFSpec defines UPF specification
type UPFSpec struct {
	Name       string
	Namespace  string
	Location   string
	MEC        *MECIntegration
	Throughput string
}

// MECIntegration defines MEC integration
type MECIntegration struct {
	Enabled  bool
	Platform string
	Apps     []string
}

// UPFDeploymentResult represents UPF deployment result
type UPFDeploymentResult struct {
	DeploymentName string
	MECEnabled     bool
	Location       string
	Status         DeploymentStatus
}

// TransportNetworkSpec defines transport network specification
type TransportNetworkSpec struct {
	SliceID   string
	Type      string
	Endpoints []TNEndpoint
	QoS       TNQoS
	SDN       *SDNConfig
}

// TNEndpoint defines transport network endpoint
type TNEndpoint struct {
	Site      string
	VLANID    int
	Bandwidth string
}

// TNQoS defines transport network QoS
type TNQoS struct {
	Class    string
	Priority int
	Jitter   int
	Loss     float64
}

// SDNConfig defines SDN configuration
type SDNConfig struct {
	Controller string
	Protocol   string
	Version    string
}

// TNStatus represents transport network status
type TNStatus string

const (
	TNStatusActive   TNStatus = "Active"
	TNStatusInactive TNStatus = "Inactive"
	TNStatusFailed   TNStatus = "Failed"
)

// TNDeploymentResult represents TN deployment result
type TNDeploymentResult struct {
	SliceID   string
	Status    TNStatus
	FlowRules []string
}

// TSNSpec defines Time-Sensitive Networking specification
type TSNSpec struct {
	NetworkID string
	Domains   []TSNDomain
	Streams   []TSNStream
}

// TSNDomain defines TSN domain
type TSNDomain struct {
	ID       string
	Bridges  []string
	Schedule string
}

// TSNStream defines TSN stream
type TSNStream struct {
	ID         string
	Priority   int
	MaxLatency int
	MaxJitter  int
	Redundancy string
}

// TSNDeploymentResult represents TSN deployment result
type TSNDeploymentResult struct {
	TSNEnabled      bool
	GateControlList []string
}

// NetworkSliceDeployment defines network slice deployment
type NetworkSliceDeployment struct {
	SliceID      string
	Type         string
	Clusters     []ClusterTarget
	Orchestrator OrchestrationConfig
}

// ClusterTarget defines target cluster
type ClusterTarget struct {
	Name       string
	Region     string
	Components []string
}

// OrchestrationConfig defines orchestration configuration
type OrchestrationConfig struct {
	Type     string
	Strategy string
	Mesh     string
}

// SliceStatus represents slice status
type SliceStatus string

const (
	SliceStatusActive   SliceStatus = "Active"
	SliceStatusInactive SliceStatus = "Inactive"
	SliceStatusFailed   SliceStatus = "Failed"
)

// SliceDeploymentResult represents slice deployment result
type SliceDeploymentResult struct {
	SliceID           string
	Status            SliceStatus
	DeployedClusters  int
	ServiceMeshConfig string
	ClusterStatuses   []ClusterStatus
}

// ClusterStatus represents cluster status
type ClusterStatus struct {
	Name       string
	Status     DeploymentStatus
	Components []string
}

// FailoverSpec defines failover specification
type FailoverSpec struct {
	SliceID       string
	FailedCluster string
	BackupCluster string
	Strategy      string
	MaxDowntime   time.Duration
}

// FailoverResult represents failover result
type FailoverResult struct {
	Success          bool
	DowntimeDuration time.Duration
}

// RollbackSpec defines rollback specification
type RollbackSpec struct {
	DeploymentID  string
	Reason        string
	TargetVersion string
	Strategy      string
	DataMigration *DataMigrationConfig
}

// DataMigrationConfig defines data migration configuration
type DataMigrationConfig struct {
	Required bool
	Strategy string
	Backup   bool
}

// RollbackResult represents rollback result
type RollbackResult struct {
	Success        bool
	CurrentVersion string
	DataMigrated   bool
	BackupID       string
}

// O2DeploymentIntent defines O2 deployment intent
type O2DeploymentIntent struct {
	Name        string
	Description string
	Profile     string
	Resources   []O2Resource
	Lifecycle   O2Lifecycle
}

// O2Resource defines O2 resource
type O2Resource struct {
	Type       string
	Name       string
	Descriptor string
	Cluster    string
}

// O2Lifecycle defines O2 lifecycle
type O2Lifecycle struct {
	Instantiate bool
	Configure   bool
	Activate    bool
}

// O2Status represents O2 status
type O2Status string

const (
	O2StatusActive   O2Status = "Active"
	O2StatusInactive O2Status = "Inactive"
	O2StatusFailed   O2Status = "Failed"
)

// O2DeploymentResult represents O2 deployment result
type O2DeploymentResult struct {
	DeploymentID string
	Status       O2Status
}

// O2DeploymentStatus represents O2 deployment status
type O2DeploymentStatus struct {
	State     string
	Resources []O2ResourceStatus
}

// O2ResourceStatus represents O2 resource status
type O2ResourceStatus struct {
	Name   string
	Status string
}

// ResourceState represents current resource state
type ResourceState struct {
	Nodes []NodeResource
	Pods  []PodResource
}

// NodeResource represents node resource
type NodeResource struct {
	Name       string
	CPUUsed    float64
	MemoryUsed float64
}

// PodResource represents pod resource
type PodResource struct {
	Name   string
	CPU    float64
	Memory float64
	Node   string
}

// OptimizationPlan represents optimization plan
type OptimizationPlan struct {
	Migrations          []Migration
	ExpectedImprovement float64
}

// Migration represents pod migration
type Migration struct {
	Pod        string
	FromNode   string
	ToNode     string
}

// LoadMetrics represents load metrics
type LoadMetrics struct {
	CPUUtilization    float64
	MemoryUtilization float64
	RequestRate       int
	ResponseTime      int
}

// ScalingPolicy defines scaling policy
type ScalingPolicy struct {
	MinReplicas        int
	MaxReplicas        int
	TargetCPU          int
	TargetMemory       int
	ScaleUpThreshold   int
	ScaleDownThreshold int
}

// ScalingAction represents scaling action
type ScalingAction string

const (
	ScaleUp   ScalingAction = "ScaleUp"
	ScaleDown ScalingAction = "ScaleDown"
	NoChange  ScalingAction = "NoChange"
)

// ScalingDecision represents scaling decision
type ScalingDecision struct {
	Action          ScalingAction
	CurrentReplicas int
	NewReplicas     int
	Reason          string
}
