package models

// Filter types for O2 IMS and O2 DMS operations

// ResourceTypeFilter provides filtering options for resource type queries
type ResourceTypeFilter struct {
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Name   string `json:"name,omitempty"`
	Vendor string `json:"vendor,omitempty"`
	Model  string `json:"model,omitempty"`
	Version string `json:"version,omitempty"`
}

// ResourcePoolFilter provides filtering options for resource pool queries
type ResourcePoolFilter struct {
	Limit     int                   `json:"limit,omitempty"`
	Offset    int                   `json:"offset,omitempty"`
	Name      string                `json:"name,omitempty"`
	OCloudID  string                `json:"oCloudId,omitempty"`
	State     ResourcePoolState     `json:"state,omitempty"`
	Location  string                `json:"location,omitempty"`
}

// ResourceFilter provides filtering options for resource queries
type ResourceFilter struct {
	Limit          int    `json:"limit,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	ResourceTypeID string `json:"resourceTypeId,omitempty"`
	OCloudID       string `json:"oCloudId,omitempty"`
	Name           string `json:"name,omitempty"`
}

// DeploymentManagerFilter provides filtering options for deployment manager queries
type DeploymentManagerFilter struct {
	Limit    int                     `json:"limit,omitempty"`
	Offset   int                     `json:"offset,omitempty"`
	Name     string                  `json:"name,omitempty"`
	OCloudID string                  `json:"oCloudId,omitempty"`
	State    DeploymentManagerState  `json:"state,omitempty"`
}

// NFDeploymentFilter provides filtering options for NF deployment queries
type NFDeploymentFilter struct {
	Limit                       int                `json:"limit,omitempty"`
	Offset                      int                `json:"offset,omitempty"`
	Name                        string             `json:"name,omitempty"`
	DeploymentManagerID         string             `json:"deploymentManagerId,omitempty"`
	NFDeploymentDescriptorID    string             `json:"nfDeploymentDescriptorId,omitempty"`
	Status                      NFDeploymentStatus `json:"status,omitempty"`
}

// SubscriptionFilter provides filtering options for subscription queries
type SubscriptionFilter struct {
	Limit       int      `json:"limit,omitempty"`
	Offset      int      `json:"offset,omitempty"`
	EventTypes  []string `json:"eventTypes,omitempty"`
	Source      string   `json:"source,omitempty"`
	CallbackURL string   `json:"callbackUrl,omitempty"`
}

// Collection types for paginated responses

// ResourceTypeCollection represents a collection of resource types
type ResourceTypeCollection struct {
	Items      []ResourceTypeInfo `json:"items"`
	Total      int                `json:"total"`
	NextMarker string             `json:"nextMarker,omitempty"`
	HasMore    bool               `json:"hasMore"`
}

// ResourcePoolCollection represents a collection of resource pools
type ResourcePoolCollection struct {
	Items      []O2CloudResourcePool `json:"items"`
	Total      int                   `json:"total"`
	NextMarker string                `json:"nextMarker,omitempty"`
	HasMore    bool                  `json:"hasMore"`
}

// ResourceCollection represents a collection of resources
type ResourceCollection struct {
	Items      []Resource `json:"items"`
	Total      int        `json:"total"`
	NextMarker string     `json:"nextMarker,omitempty"`
	HasMore    bool       `json:"hasMore"`
}

// DeploymentManagerCollection represents a collection of deployment managers
type DeploymentManagerCollection struct {
	Items      []DeploymentManager `json:"items"`
	Total      int                 `json:"total"`
	NextMarker string              `json:"nextMarker,omitempty"`
	HasMore    bool                `json:"hasMore"`
}

// NFDeploymentCollection represents a collection of NF deployments
type NFDeploymentCollection struct {
	Items      []NFDeployment `json:"items"`
	Total      int            `json:"total"`
	NextMarker string         `json:"nextMarker,omitempty"`
	HasMore    bool           `json:"hasMore"`
}

// NFDeploymentDescriptorCollection represents a collection of NF deployment descriptors
type NFDeploymentDescriptorCollection struct {
	Items      []NFDeploymentDescriptor `json:"items"`
	Total      int                      `json:"total"`
	NextMarker string                   `json:"nextMarker,omitempty"`
	HasMore    bool                     `json:"hasMore"`
}

// InventoryInfo represents the O2 IMS inventory information
type InventoryInfo struct {
	OCloudID          string                 `json:"oCloudId"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	ServiceURI        string                 `json:"serviceUri"`
	SupportedFeatures []string               `json:"supportedFeatures,omitempty"`
	Extensions        map[string]interface{} `json:"extensions,omitempty"`
}