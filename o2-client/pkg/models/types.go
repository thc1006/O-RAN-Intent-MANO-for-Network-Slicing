package models

import "time"

// ResourcePoolSpecV1 defines the specification for a resource pool (v1 format)
type ResourcePoolSpecV1 struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	ClusterID   string                 `json:"clusterId"`
	Capacity    ResourceCapacity       `json:"capacity"`
	Resources   []ResourceDefinition   `json:"resources,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceCapacity defines the capacity of resources
type ResourceCapacity struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Storage string `json:"storage,omitempty"`
}

// ResourceDefinition defines individual resources
type ResourceDefinition struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

// O2ResourcePool represents a pool of resources in O2IMS
type O2ResourcePool struct {
	ID          string             `json:"id"`
	Spec        ResourcePoolSpecV1 `json:"spec"`
	Status      string             `json:"status"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}