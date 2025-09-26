// Package models defines the data types for CN-DMS
package models

import (
	"time"
)

// Slice represents a 5G network slice instance
type Slice struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Type             SliceType         `json:"type"`
	Status           SliceStatus       `json:"status"`
	QoSProfile       QoSProfile        `json:"qos_profile"`
	NetworkFunctions []NetworkFunction `json:"network_functions,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// SliceType represents the type of network slice
type SliceType string

const (
	SliceTypeEMBB  SliceType = "eMBB"  // Enhanced Mobile Broadband
	SliceTypeURLLC SliceType = "URLLC" // Ultra-Reliable Low-Latency Communications
	SliceTypeMMTC  SliceType = "mMTC"  // Massive Machine-Type Communications
)

// SliceStatus represents the current status of a slice
type SliceStatus string

const (
	SliceStatusPending     SliceStatus = "pending"
	SliceStatusActivating  SliceStatus = "activating"
	SliceStatusActive      SliceStatus = "active"
	SliceStatusDeactivated SliceStatus = "deactivated"
	SliceStatusError       SliceStatus = "error"
)

// QoSProfile defines Quality of Service requirements
type QoSProfile struct {
	MaxLatencyMs      float64 `json:"max_latency_ms"`
	MinThroughputMbps float64 `json:"min_throughput_mbps"`
	MaxPacketLoss     float64 `json:"max_packet_loss"`
	Reliability       float64 `json:"reliability"`
}

// NetworkFunction represents a 5G Core Network Function
type NetworkFunction struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Type            NFType                `json:"type"`
	Status          NFStatus              `json:"status"`
	SliceID         string                `json:"slice_id,omitempty"`
	Resources       ResourceRequirements  `json:"resources"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`
	DeploymentSite  string                `json:"deployment_site,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// NFType represents the type of Network Function
type NFType string

const (
	NFTypeAMF NFType = "AMF" // Access and Mobility Management Function
	NFTypeSMF NFType = "SMF" // Session Management Function
	NFTypeUPF NFType = "UPF" // User Plane Function
	NFTypePCF NFType = "PCF" // Policy Control Function
	NFTypeUDM NFType = "UDM" // Unified Data Management
	NFTypeNRF NFType = "NRF" // Network Repository Function
)

// NFStatus represents the current status of a Network Function
type NFStatus string

const (
	NFStatusPending     NFStatus = "pending"
	NFStatusDeploying   NFStatus = "deploying"
	NFStatusRunning     NFStatus = "running"
	NFStatusStopped     NFStatus = "stopped"
	NFStatusError       NFStatus = "error"
	NFStatusTerminating NFStatus = "terminating"
)

// ResourceRequirements defines the resource needs for a Network Function
type ResourceRequirements struct {
	CPUCores    int     `json:"cpu_cores"`
	MemoryGB    int     `json:"memory_gb"`
	StorageGB   int     `json:"storage_gb"`
	NetworkMbps float64 `json:"network_mbps"`
}

// CNStatus represents the overall status of the Core Network Domain
type CNStatus struct {
	Status      string            `json:"status"`
	Uptime      int64             `json:"uptime"`
	SliceCount  int               `json:"slice_count"`
	NFCount     int               `json:"nf_count"`
	Health      HealthStatus      `json:"health"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	LastUpdated time.Time         `json:"last_updated"`
}

// HealthStatus represents health check results
type HealthStatus struct {
	Overall   string             `json:"overall"`
	Services  map[string]string  `json:"services"`
	Checks    []HealthCheck      `json:"checks"`
	Timestamp time.Time          `json:"timestamp"`
}

// HealthCheck represents individual health check
type HealthCheck struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}

// CNCapabilities represents the capabilities of the CN Domain
type CNCapabilities struct {
	MaxSlices         int      `json:"max_slices"`
	SupportedNFTypes  []NFType `json:"supported_nf_types"`
	SliceTypes        []SliceType `json:"slice_types"`
	APIVersion        string   `json:"api_version"`
	SupportedFeatures []string `json:"supported_features"`
}

// SliceRequest represents a request to create a new slice
type SliceRequest struct {
	Name       string            `json:"name" binding:"required"`
	Type       SliceType         `json:"type" binding:"required"`
	QoSProfile QoSProfile        `json:"qos_profile" binding:"required"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// NFDeploymentRequest represents a request to deploy a Network Function
type NFDeploymentRequest struct {
	Name          string                `json:"name" binding:"required"`
	Type          NFType                `json:"type" binding:"required"`
	SliceID       string                `json:"slice_id,omitempty"`
	Resources     ResourceRequirements  `json:"resources" binding:"required"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	TargetSite    string                `json:"target_site,omitempty"`
}