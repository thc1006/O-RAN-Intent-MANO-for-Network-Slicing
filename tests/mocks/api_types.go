// Package mocks provides mock API types for testing
package mocks

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mock API types for orchestrator
type Intent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              IntentSpec   `json:"spec,omitempty"`
	Status            IntentStatus `json:"status,omitempty"`
}

type IntentSpec struct {
	Description               string                    `json:"description,omitempty"`
	SliceType                 string                    `json:"sliceType,omitempty"`
	Requirements              Requirements              `json:"requirements,omitempty"`
	QoSRequirements           QoSRequirements           `json:"qosRequirements,omitempty"`
	Coverage                  CoverageRequirement       `json:"coverage,omitempty"`
	SliceComposition          []SliceRequirement        `json:"sliceComposition,omitempty"`
	DomainRequirements        DomainRequirements        `json:"domainRequirements,omitempty"`
	Placement                 PlacementPolicy           `json:"placement,omitempty"`
	ValidationRequirements    ValidationRequirements    `json:"validationRequirements,omitempty"`
	MultiSiteRequirements     MultiSiteRequirements     `json:"multiSiteRequirements,omitempty"`
	ResiliencyRequirements    ResiliencyRequirements    `json:"resiliencyRequirements,omitempty"`
}

type IntentStatus struct {
	Phase                string                `json:"phase,omitempty"`
	QoSMapping           *QoSMapping           `json:"qosMapping,omitempty"`
	PlacementDecision    *PlacementDecision    `json:"placementDecision,omitempty"`
	AllocatedResources   *AllocatedResources   `json:"allocatedResources,omitempty"`
	DeployedSlices       []DeployedSlice       `json:"deployedSlices,omitempty"`
	ConnectivityStatus   string                `json:"connectivityStatus,omitempty"`
	PerformanceMetrics   *PerformanceMetrics   `json:"performanceMetrics,omitempty"`
	RANStatus            string                `json:"ranStatus,omitempty"`
	TNStatus             string                `json:"tnStatus,omitempty"`
	CNStatus             string                `json:"cnStatus,omitempty"`
	DeployedSites        []string              `json:"deployedSites,omitempty"`
	SiteCoordination     string                `json:"siteCoordination,omitempty"`
	RecoveryStatus       string                `json:"recoveryStatus,omitempty"`
	FailureEvents        []FailureEvent        `json:"failureEvents,omitempty"`
}

type Requirements struct {
	ServiceType string `json:"serviceType,omitempty"`
	Quality     string `json:"quality,omitempty"`
}

type QoSRequirements struct {
	Throughput     string `json:"throughput,omitempty"`
	Latency        string `json:"latency,omitempty"`
	Reliability    string `json:"reliability,omitempty"`
	PacketLoss     string `json:"packetLoss,omitempty"`
	DeviceDensity  string `json:"deviceDensity,omitempty"`
}

type CoverageRequirement struct {
	Type     string   `json:"type,omitempty"`
	Sites    []string `json:"sites,omitempty"`
	Mobility string   `json:"mobility,omitempty"`
}

type SliceRequirement struct {
	SliceType       string          `json:"sliceType,omitempty"`
	Weight          float64         `json:"weight,omitempty"`
	QoSRequirements QoSRequirements `json:"qosRequirements,omitempty"`
}

type DomainRequirements struct {
	RAN RANRequirements `json:"ran,omitempty"`
	TN  TNRequirements  `json:"tn,omitempty"`
	CN  CNRequirements  `json:"cn,omitempty"`
}

type RANRequirements struct {
	CoverageType          string `json:"coverageType,omitempty"`
	BeamFormingRequired   bool   `json:"beamFormingRequired,omitempty"`
	CarrierAggregation    bool   `json:"carrierAggregation,omitempty"`
}

type TNRequirements struct {
	BandwidthGuarantee bool `json:"bandwidthGuarantee,omitempty"`
	PathDiversity      bool `json:"pathDiversity,omitempty"`
	QoSPolicing        bool `json:"qosPolicing,omitempty"`
}

type CNRequirements struct {
	Architecture    string `json:"architecture,omitempty"`
	EdgeComputing   bool   `json:"edgeComputing,omitempty"`
	SliceIsolation  string `json:"sliceIsolation,omitempty"`
}

type PlacementPolicy struct {
	PreferredSites []string              `json:"preferredSites,omitempty"`
	Constraints    []PlacementConstraint `json:"constraints,omitempty"`
}

type PlacementConstraint struct {
	Type   string  `json:"type,omitempty"`
	Value  string  `json:"value,omitempty"`
	Weight float64 `json:"weight,omitempty"`
}

type ValidationRequirements struct {
	PerformanceValidation bool `json:"performanceValidation,omitempty"`
	MetricsCollection     bool `json:"metricsCollection,omitempty"`
	ComplianceChecking    bool `json:"complianceChecking,omitempty"`
}

type MultiSiteRequirements struct {
	EdgeSites     []string `json:"edgeSites,omitempty"`
	CloudSites    []string `json:"cloudSites,omitempty"`
	Coordination  string   `json:"coordination,omitempty"`
	LoadBalancing string   `json:"loadBalancing,omitempty"`
}

type ResiliencyRequirements struct {
	FailoverEnabled     bool     `json:"failoverEnabled,omitempty"`
	BackupSites         []string `json:"backupSites,omitempty"`
	RecoveryTimeout     string   `json:"recoveryTimeout,omitempty"`
	HealthCheckInterval string   `json:"healthCheckInterval,omitempty"`
}

type QoSMapping struct {
	MappedRequirements map[string]string `json:"mappedRequirements,omitempty"`
}

type PlacementDecision struct {
	SelectedSites []string `json:"selectedSites,omitempty"`
	Score         float64  `json:"score,omitempty"`
}

type AllocatedResources struct {
	CPU     string `json:"cpu,omitempty"`
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage,omitempty"`
}

type DeployedSlice struct {
	SliceID   string `json:"sliceId,omitempty"`
	SliceType string `json:"sliceType,omitempty"`
	Site      string `json:"site,omitempty"`
	Status    string `json:"status,omitempty"`
}

type PerformanceMetrics struct {
	Throughput float64 `json:"throughput,omitempty"`
	Latency    float64 `json:"latency,omitempty"`
}

type FailureEvent struct {
	Timestamp   time.Time `json:"timestamp,omitempty"`
	Type        string    `json:"type,omitempty"`
	Description string    `json:"description,omitempty"`
	Recovered   bool      `json:"recovered,omitempty"`
}

// Mock API types for RAN-DMS
type RANResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RANResourceSpec   `json:"spec,omitempty"`
	Status            RANResourceStatus `json:"status,omitempty"`
}

type RANResourceSpec struct {
	Type          string                `json:"type,omitempty"`
	Location      string                `json:"location,omitempty"`
	Capacity      ResourceCapacity      `json:"capacity,omitempty"`
	RadioConfig   RadioConfiguration    `json:"radioConfig,omitempty"`
	HAConfig      HAConfig              `json:"haConfig,omitempty"`
}

type RANResourceStatus struct {
	Phase      string `json:"phase,omitempty"`
	ActiveSite string `json:"activeSite,omitempty"`
}

type ResourceCapacity struct {
	CPU     string `json:"cpu,omitempty"`
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage,omitempty"`
}

type RadioConfiguration struct {
	Frequency string `json:"frequency,omitempty"`
	Bandwidth string `json:"bandwidth,omitempty"`
	TxPower   string `json:"txPower,omitempty"`
}

type HAConfig struct {
	Enabled       bool     `json:"enabled,omitempty"`
	BackupSites   []string `json:"backupSites,omitempty"`
	FailoverTime  string   `json:"failoverTime,omitempty"`
	SyncInterval  string   `json:"syncInterval,omitempty"`
}

// RAN Slice types
type RANSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RANSliceSpec   `json:"spec,omitempty"`
	Status            RANSliceStatus `json:"status,omitempty"`
}

type RANSliceSpec struct {
	SliceID            string             `json:"sliceId,omitempty"`
	SliceType          string             `json:"sliceType,omitempty"`
	QoSProfile         QoSProfile         `json:"qosProfile,omitempty"`
	ResourceAllocation ResourceAllocation `json:"resourceAllocation,omitempty"`
}

type RANSliceStatus struct {
	Phase               string               `json:"phase,omitempty"`
	ConfiguredPRBs      int                  `json:"configuredPRBs,omitempty"`
	AllocatedResources  *AllocatedRANResources `json:"allocatedResources,omitempty"`
}

type QoSProfile struct {
	Priority    int    `json:"priority,omitempty"`
	Throughput  string `json:"throughput,omitempty"`
	Latency     string `json:"latency,omitempty"`
	Reliability string `json:"reliability,omitempty"`
}

type ResourceAllocation struct {
	PRBs     int            `json:"prbs,omitempty"`
	Antennas int            `json:"antennas,omitempty"`
	Carriers []CarrierConfig `json:"carriers,omitempty"`
}

type AllocatedRANResources struct {
	PRBs     int `json:"prbs,omitempty"`
	StartPRB int `json:"startPRB,omitempty"`
}

type CarrierConfig struct {
	Frequency string `json:"frequency,omitempty"`
	Bandwidth string `json:"bandwidth,omitempty"`
}

// gNodeB types
type GNodeB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GNodeBSpec   `json:"spec,omitempty"`
	Status            GNodeBStatus `json:"status,omitempty"`
}

type GNodeBSpec struct {
	NodeID         string          `json:"nodeId,omitempty"`
	PLMNs          []PLMN          `json:"plmns,omitempty"`
	Cells          []CellConfig    `json:"cells,omitempty"`
	AMFConnections []AMFConnection `json:"amfConnections,omitempty"`
}

type GNodeBStatus struct {
	Phase          string `json:"phase,omitempty"`
	ConnectedCells int    `json:"connectedCells,omitempty"`
}

type PLMN struct {
	MCC string `json:"mcc,omitempty"`
	MNC string `json:"mnc,omitempty"`
}

type CellConfig struct {
	CellID    int    `json:"cellId,omitempty"`
	PCI       int    `json:"pci,omitempty"`
	TAC       string `json:"tac,omitempty"`
	Frequency string `json:"frequency,omitempty"`
	Bandwidth string `json:"bandwidth,omitempty"`
	TxPower   string `json:"txPower,omitempty"`
}

type AMFConnection struct {
	AMF_IP   string `json:"amf_ip,omitempty"`
	AMF_Port int    `json:"amf_port,omitempty"`
}

// Performance Monitor types
type PerformanceMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PerformanceMonitorSpec   `json:"spec,omitempty"`
	Status            PerformanceMonitorStatus `json:"status,omitempty"`
}

type PerformanceMonitorSpec struct {
	Targets  []MonitorTarget `json:"targets,omitempty"`
	Metrics  []MetricConfig  `json:"metrics,omitempty"`
	Duration string          `json:"duration,omitempty"`
}

type PerformanceMonitorStatus struct {
	Phase            string                   `json:"phase,omitempty"`
	CollectedMetrics []CollectedMetric        `json:"collectedMetrics,omitempty"`
}

type MonitorTarget struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

type MetricConfig struct {
	Name     string `json:"name,omitempty"`
	Interval string `json:"interval,omitempty"`
	Unit     string `json:"unit,omitempty"`
}

type CollectedMetric struct {
	Name      string    `json:"name,omitempty"`
	Value     float64   `json:"value,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// Mock API types for CN-DMS
type CoreNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CoreNetworkSpec   `json:"spec,omitempty"`
	Status            CoreNetworkStatus `json:"status,omitempty"`
}

type CoreNetworkSpec struct {
	Release          string             `json:"release,omitempty"`
	Deployment       DeploymentConfig   `json:"deployment,omitempty"`
	NetworkFunctions []NetworkFunction  `json:"networkFunctions,omitempty"`
}

type CoreNetworkStatus struct {
	Phase       string                  `json:"phase,omitempty"`
	DeployedNFs []DeployedNetworkFunction `json:"deployedNFs,omitempty"`
}

type DeploymentConfig struct {
	Architecture string `json:"architecture,omitempty"`
	Mode         string `json:"mode,omitempty"`
	Scale        string `json:"scale,omitempty"`
}

type NetworkFunction struct {
	Type          string                `json:"type,omitempty"`
	Version       string                `json:"version,omitempty"`
	Replicas      int                   `json:"replicas,omitempty"`
	Resources     ResourceRequirements  `json:"resources,omitempty"`
	Configuration map[string]string     `json:"configuration,omitempty"`
}

type DeployedNetworkFunction struct {
	Type   string `json:"type,omitempty"`
	Status string `json:"status,omitempty"`
}

type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// Network Slice types
type NetworkSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NetworkSliceSpec   `json:"spec,omitempty"`
	Status            NetworkSliceStatus `json:"status,omitempty"`
}

type NetworkSliceSpec struct {
	SNSSAI         SNSSAI         `json:"snssai,omitempty"`
	SliceType      string         `json:"sliceType,omitempty"`
	QoSProfile     CNQoSProfile   `json:"qosProfile,omitempty"`
	Coverage       []CoverageArea `json:"coverage,omitempty"`
	IsolationLevel string         `json:"isolationLevel,omitempty"`
	Priority       int            `json:"priority,omitempty"`
	TenantID       string         `json:"tenantId,omitempty"`
}

type NetworkSliceStatus struct {
	Phase               string                  `json:"phase,omitempty"`
	AllocatedResources  *CNAllocatedResources   `json:"allocatedResources,omitempty"`
	TenantID            string                  `json:"tenantId,omitempty"`
	IsolationConfig     map[string]string       `json:"isolationConfig,omitempty"`
	LastReconfigured    metav1.Time             `json:"lastReconfigured,omitempty"`
}

type SNSSAI struct {
	SST int    `json:"sst,omitempty"`
	SD  string `json:"sd,omitempty"`
}

type CNQoSProfile struct {
	ULThroughput string `json:"ulThroughput,omitempty"`
	DLThroughput string `json:"dlThroughput,omitempty"`
	Latency      string `json:"latency,omitempty"`
	Reliability  string `json:"reliability,omitempty"`
	PacketLoss   string `json:"packetLoss,omitempty"`
}

type CoverageArea struct {
	Type        string                  `json:"type,omitempty"`
	Name        string                  `json:"name,omitempty"`
	Coordinates GeographicCoordinates   `json:"coordinates,omitempty"`
}

type GeographicCoordinates struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Radius    int     `json:"radius,omitempty"`
}

type CNAllocatedResources struct {
	CPU     string `json:"cpu,omitempty"`
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage,omitempty"`
}

// Other CN types
type NetworkService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NetworkServiceSpec   `json:"spec,omitempty"`
	Status            NetworkServiceStatus `json:"status,omitempty"`
}

type NetworkServiceSpec struct {
	ServiceID  string                     `json:"serviceId,omitempty"`
	Type       string                     `json:"type,omitempty"`
	Template   string                     `json:"template,omitempty"`
	Parameters map[string]string          `json:"parameters,omitempty"`
	Endpoints  []ServiceEndpoint          `json:"endpoints,omitempty"`
}

type NetworkServiceStatus struct {
	Phase           string            `json:"phase,omitempty"`
	ActiveEndpoints []ServiceEndpoint `json:"activeEndpoints,omitempty"`
}

type ServiceEndpoint struct {
	Name     string `json:"name,omitempty"`
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// Service Chain types
type ServiceChain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceChainSpec   `json:"spec,omitempty"`
	Status            ServiceChainStatus `json:"status,omitempty"`
}

type ServiceChainSpec struct {
	ChainID         string            `json:"chainId,omitempty"`
	Services        []ServiceFunction `json:"services,omitempty"`
	TrafficPolicies []TrafficPolicy   `json:"trafficPolicies,omitempty"`
}

type ServiceChainStatus struct {
	Phase            string                  `json:"phase,omitempty"`
	DeployedServices []DeployedServiceFunction `json:"deployedServices,omitempty"`
}

type ServiceFunction struct {
	Name          string            `json:"name,omitempty"`
	Type          string            `json:"type,omitempty"`
	Version       string            `json:"version,omitempty"`
	Order         int               `json:"order,omitempty"`
	Configuration map[string]string `json:"configuration,omitempty"`
}

type DeployedServiceFunction struct {
	Name  string `json:"name,omitempty"`
	Order int    `json:"order,omitempty"`
}

type TrafficPolicy struct {
	Type      string `json:"type,omitempty"`
	Selector  string `json:"selector,omitempty"`
	ChainRule string `json:"chainRule,omitempty"`
}

// CN Performance Monitor types
type CNPerformanceMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CNPerformanceMonitorSpec   `json:"spec,omitempty"`
	Status            CNPerformanceMonitorStatus `json:"status,omitempty"`
}

type CNPerformanceMonitorSpec struct {
	Targets          []MonitoringTarget `json:"targets,omitempty"`
	SamplingInterval string             `json:"samplingInterval,omitempty"`
	RetentionPeriod  string             `json:"retentionPeriod,omitempty"`
	Thresholds       []Threshold        `json:"thresholds,omitempty"`
}

type CNPerformanceMonitorStatus struct {
	Phase          string      `json:"phase,omitempty"`
	LastSampleTime metav1.Time `json:"lastSampleTime,omitempty"`
}

type MonitoringTarget struct {
	Type    string   `json:"type,omitempty"`
	Name    string   `json:"name,omitempty"`
	Metrics []string `json:"metrics,omitempty"`
}

type Threshold struct {
	Metric    string `json:"metric,omitempty"`
	Condition string `json:"condition,omitempty"`
	Value     string `json:"value,omitempty"`
	Action    string `json:"action,omitempty"`
}