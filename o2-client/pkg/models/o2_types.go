package models

import (
	"time"
)

// O-RAN O2 Interface Models
// Based on O-RAN.WG6.O2IMS-INTERFACE-R003-v05.00

// ResourceType represents the type of O-Cloud resource
type ResourceType string

const (
	ResourceTypeNode        ResourceType = "Node"
	ResourceTypeCPU         ResourceType = "CPU"
	ResourceTypeMemory      ResourceType = "Memory"
	ResourceTypeStorage     ResourceType = "Storage"
	ResourceTypeNetwork     ResourceType = "Network"
	ResourceTypeAccelerator ResourceType = "Accelerator"
)

// ResourcePoolState represents the operational state of a resource pool
type ResourcePoolState string

const (
	ResourcePoolStateEnabled  ResourcePoolState = "enabled"
	ResourcePoolStateDisabled ResourcePoolState = "disabled"
)

// DeploymentManagerState represents the state of a deployment manager
type DeploymentManagerState string

const (
	DeploymentManagerStateEnabled  DeploymentManagerState = "enabled"
	DeploymentManagerStateDisabled DeploymentManagerState = "disabled"
)

// O2IMS Models

// OCloudInfo represents information about an O-Cloud
type OCloudInfo struct {
	OCloudID          string                 `json:"oCloudId"`
	GlobalCloudID     string                 `json:"globalCloudId,omitempty"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	ServiceURI        string                 `json:"serviceUri"`
	SupportedFeatures []string               `json:"supportedFeatures,omitempty"`
	Extensions        map[string]interface{} `json:"extensions,omitempty"`
}

// O2CloudResourcePool represents a pool of resources in the O-Cloud
type O2CloudResourcePool struct {
	ResourcePoolID string                 `json:"resourcePoolId"`
	OCloudID       string                 `json:"oCloudId"`
	GlobalCloudID  string                 `json:"globalCloudId,omitempty"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Location       string                 `json:"location,omitempty"`
	State          ResourcePoolState      `json:"state"`
	Resources      []Resource             `json:"resources,omitempty"`
	Extensions     map[string]interface{} `json:"extensions,omitempty"`
}

// Resource represents a specific resource within a resource pool
type Resource struct {
	ResourceID     string                 `json:"resourceId"`
	ResourcePoolID string                 `json:"resourcePoolId"`
	OCloudID       string                 `json:"oCloudId"`
	GlobalCloudID  string                 `json:"globalCloudId,omitempty"`
	ResourceTypeID string                 `json:"resourceTypeId"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Elements       []ResourceElement      `json:"elements,omitempty"`
	Extensions     map[string]interface{} `json:"extensions,omitempty"`
}

// ResourceElement represents individual elements of a resource
type ResourceElement struct {
	ElementID   string                 `json:"elementId"`
	ResourceID  string                 `json:"resourceId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
}

// ResourceTypeInfo describes a type of resource
type ResourceTypeInfo struct {
	ResourceTypeID   string                 `json:"resourceTypeId"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Vendor           string                 `json:"vendor,omitempty"`
	Model            string                 `json:"model,omitempty"`
	Version          string                 `json:"version,omitempty"`
	AlarmDictionary  AlarmDictionary        `json:"alarmDictionary,omitempty"`
	ResourceKind     string                 `json:"resourceKind,omitempty"`
	ResourceClass    string                 `json:"resourceClass,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`
}

// AlarmDictionary defines alarm types for resources
type AlarmDictionary struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	EntityType         string      `json:"entityType"`
	AlarmDefinition    []AlarmDef  `json:"alarmDefinition,omitempty"`
	AlarmLastChange    string      `json:"alarmLastChange,omitempty"`
	AlarmChangeType    []string    `json:"alarmChangeType,omitempty"`
	AlarmDescription   string      `json:"alarmDescription,omitempty"`
	ProposedRepairActions string   `json:"proposedRepairActions,omitempty"`
	ClearingType       []string    `json:"clearingType,omitempty"`
}

// AlarmDef defines an alarm
type AlarmDef struct {
	AlarmCode        string `json:"alarmCode"`
	AlarmName        string `json:"alarmName"`
	AlarmDescription string `json:"alarmDescription"`
	ProposedRepairActions string `json:"proposedRepairActions"`
	ClearingType     string `json:"clearingType"`
}

// Infrastructure Models

// InfrastructureResource represents infrastructure resource information
type InfrastructureResource struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Location string            `json:"location"`
	Status   string            `json:"status"`
	Capacity map[string]string `json:"capacity,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ResourcePoolSpec represents the specification for creating a resource pool
type ResourcePoolSpec struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Location    string                `json:"location,omitempty"`
	Resources   []ResourceRequirement `json:"resources,omitempty"`
}

// ResourceRequirement represents a resource requirement
type ResourceRequirement struct {
	Type      string `json:"type"`
	CPU       string `json:"cpu,omitempty"`
	Memory    string `json:"memory,omitempty"`
	Storage   string `json:"storage,omitempty"`
	Bandwidth string `json:"bandwidth,omitempty"`
	Latency   string `json:"latency,omitempty"`
}

// ResourcePool represents a created resource pool
type ResourcePool struct {
	ID        string            `json:"id"`
	Spec      ResourcePoolSpec  `json:"spec"`
	Status    string            `json:"status"`
	CreatedAt time.Time         `json:"createdAt"`
}

// VNF Deployment Models

// VNFDeploymentSpec represents the specification for VNF deployment
type VNFDeploymentSpec struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Version     string                 `json:"version,omitempty"`
	PackageURI  string                 `json:"packageUri,omitempty"`
	TargetSite  string                 `json:"targetSite"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// VNFDeployment represents a VNF deployment instance
type VNFDeployment struct {
	ID     string            `json:"id"`
	Spec   VNFDeploymentSpec `json:"spec"`
	Status string            `json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
}

// CNF Deployment Models

// CNFDeploymentSpec represents the specification for CNF deployment
type CNFDeploymentSpec struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Version    string                 `json:"version,omitempty"`
	HelmChart  string                 `json:"helmChart,omitempty"`
	TargetSite string                 `json:"targetSite"`
	Namespace  string                 `json:"namespace,omitempty"`
	Values     map[string]interface{} `json:"values,omitempty"`
}

// CNFDeployment represents a CNF deployment instance
type CNFDeployment struct {
	ID     string            `json:"id"`
	Spec   CNFDeploymentSpec `json:"spec"`
	Status string            `json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
}

// O2DMS Models

// DeploymentManager represents a deployment manager instance
type DeploymentManager struct {
	DeploymentManagerID string                 `json:"deploymentManagerId"`
	OCloudID            string                 `json:"oCloudId"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	DeploymentManagementServiceEndpoint string `json:"deploymentManagementServiceEndpoint"`
	CapacityInfo        string                 `json:"capacityInfo,omitempty"`
	State               DeploymentManagerState `json:"state"`
	SupportedLocations  []string               `json:"supportedLocations,omitempty"`
	Capabilities        []string               `json:"capabilities,omitempty"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`
}

// NFDeploymentDescriptor describes how to deploy a Network Function
type NFDeploymentDescriptor struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	InputParams         []Parameter            `json:"inputParams,omitempty"`
	OutputParams        []Parameter            `json:"outputParams,omitempty"`
	ArtifactReferences  []ArtifactReference    `json:"artifactReferences,omitempty"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`
}

// Parameter represents input/output parameters
type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	IsArray     bool        `json:"isArray,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// ArtifactReference references deployment artifacts
type ArtifactReference struct {
	ArtifactName string `json:"artifactName"`
	ArtifactURI  string `json:"artifactURI"`
	ArtifactType string `json:"artifactType"`
	CheckSum     string `json:"checkSum,omitempty"`
}

// NFDeployment represents a deployed Network Function
type NFDeployment struct {
	ID                            string                 `json:"id"`
	Name                          string                 `json:"name"`
	Description                   string                 `json:"description,omitempty"`
	NFDeploymentDescriptorID      string                 `json:"nfDeploymentDescriptorId"`
	ParentDeploymentID            string                 `json:"parentDeploymentId,omitempty"`
	DeploymentManagerID           string                 `json:"deploymentManagerId"`
	Status                        NFDeploymentStatus     `json:"status"`
	InputParams                   map[string]interface{} `json:"inputParams,omitempty"`
	OutputParams                  map[string]interface{} `json:"outputParams,omitempty"`
	CreationTime                  time.Time              `json:"creationTime"`
	LastUpdateTime                time.Time              `json:"lastUpdateTime"`
	Extensions                    map[string]interface{} `json:"extensions,omitempty"`
}

// NFDeploymentStatus represents the status of an NF deployment
type NFDeploymentStatus string

const (
	NFDeploymentStatusNotInstantiated NFDeploymentStatus = "NOT_INSTANTIATED"
	NFDeploymentStatusInstantiating   NFDeploymentStatus = "INSTANTIATING"
	NFDeploymentStatusInstantiated    NFDeploymentStatus = "INSTANTIATED"
	NFDeploymentStatusFailed          NFDeploymentStatus = "FAILED"
)

// Subscription Models

// SubscriptionSpec represents the specification for creating a subscription
type SubscriptionSpec struct {
	Filter      EventFilter `json:"filter"`
	CallbackURL string      `json:"callbackUrl"`
	ExpiryTime  time.Time   `json:"expiryTime,omitempty"`
}

// EventFilter defines the filter criteria for subscription events
type EventFilter struct {
	EventTypes []string `json:"eventTypes"`
	Source     string   `json:"source,omitempty"`
}

// Subscription represents a subscription to O2 notifications
type Subscription struct {
	ID                   string                 `json:"id"`
	SubscriptionID       string                 `json:"subscriptionId"`
	Spec                 SubscriptionSpec       `json:"spec"`
	Status               string                 `json:"status"`
	Callback             string                 `json:"callback"`
	ConsumerSubscriptionID string               `json:"consumerSubscriptionId,omitempty"`
	Filter               string                 `json:"filter,omitempty"`
	SystemType           []string               `json:"systemType,omitempty"`
	CreatedAt            time.Time              `json:"createdAt"`
	Extensions           map[string]interface{} `json:"extensions,omitempty"`
}

// Notification represents an O2 notification
type Notification struct {
	NotificationID         string                 `json:"notificationId"`
	NotificationType       string                 `json:"notificationType"`
	EventType              string                 `json:"eventType"`
	ObjectRef              string                 `json:"objectRef"`
	UpdatedFields          []string               `json:"updatedFields,omitempty"`
	NotificationEventTime  time.Time              `json:"notificationEventTime"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
}

// Error Models

// APIError represents an API error response
type APIError struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Instance  string `json:"instance,omitempty"`
}

// Common Response Types

// APIResponse represents a generic API response
type APIResponse struct {
	Data       interface{} `json:"data,omitempty"`
	Error      *APIError   `json:"error,omitempty"`
	StatusCode int         `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Items        []interface{} `json:"items"`
	Total        int           `json:"total"`
	NextMarker   string        `json:"nextMarker,omitempty"`
	PrevMarker   string        `json:"prevMarker,omitempty"`
	HasMore      bool          `json:"hasMore"`
}

// Health Check Models

// HealthInfo represents system health information
type HealthInfo struct {
	Status        string                 `json:"status"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description,omitempty"`
	ApiVersions   []string               `json:"apiVersions,omitempty"`
	UriPrefix     string                 `json:"uriPrefix,omitempty"`
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
}

// O-RAN Specific Extensions

// ORanQoSRequirements represents O-RAN specific QoS requirements
type ORanQoSRequirements struct {
	Bandwidth        float64 `json:"bandwidth"`         // Mbps
	Latency          float64 `json:"latency"`           // ms
	Jitter           float64 `json:"jitter,omitempty"`  // ms
	PacketLoss       float64 `json:"packetLoss,omitempty"` // percentage
	Reliability      float64 `json:"reliability,omitempty"` // percentage
	SliceType        string  `json:"sliceType,omitempty"`   // eMBB, uRLLC, mIoT
	Priority         int     `json:"priority,omitempty"`    // 1-10
}

// ORanPlacement represents O-RAN specific placement requirements
type ORanPlacement struct {
	CloudType     string   `json:"cloudType"`           // edge, regional, central
	Region        string   `json:"region,omitempty"`
	Zone          string   `json:"zone,omitempty"`
	Site          string   `json:"site,omitempty"`
	AffinityRules []string `json:"affinityRules,omitempty"`
}

// ORanSliceInfo represents O-RAN network slice information
type ORanSliceInfo struct {
	SliceID       string                 `json:"sliceId"`
	ServiceType   string                 `json:"serviceType"`
	QoSRequirements ORanQoSRequirements  `json:"qosRequirements"`
	Placement     ORanPlacement          `json:"placement"`
	NetworkFunctions []string            `json:"networkFunctions"`
	Capacity      map[string]interface{} `json:"capacity,omitempty"`
	SLA           map[string]interface{} `json:"sla,omitempty"`
}