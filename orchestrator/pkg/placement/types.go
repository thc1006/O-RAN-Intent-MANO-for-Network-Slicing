// Package placement provides intelligent network function placement policies
// for O-RAN Intent-based MANO framework
package placement

import (
	"fmt"
	"time"
)

// CloudType represents the type of O-Cloud infrastructure
type CloudType string

const (
	// CloudTypeEdge represents edge cloud with ultra-low latency
	CloudTypeEdge CloudType = "edge"
	// CloudTypeRegional represents regional cloud with balanced resources
	CloudTypeRegional CloudType = "regional"
	// CloudTypeCentral represents central cloud with high capacity
	CloudTypeCentral CloudType = "central"
)

// Site represents an O-Cloud deployment site
type Site struct {
	// Unique identifier for the site
	ID string `json:"id"`
	// Human-readable name
	Name string `json:"name"`
	// Type of cloud infrastructure
	Type CloudType `json:"type"`
	// Geographic location
	Location Location `json:"location"`
	// Resource capacity
	Capacity ResourceCapacity `json:"capacity"`
	// Network characteristics
	NetworkProfile NetworkProfile `json:"network_profile"`
	// Current resource utilization
	Metrics *SiteMetrics `json:"metrics,omitempty"`
	// Availability status
	Available bool `json:"available"`
}

// Location represents geographic coordinates
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Region    string  `json:"region"`
	Zone      string  `json:"zone"`
}

// ResourceCapacity defines the total resources available at a site
type ResourceCapacity struct {
	// CPU cores available
	CPUCores int `json:"cpu_cores"`
	// Memory in GB
	MemoryGB int `json:"memory_gb"`
	// Storage in GB
	StorageGB int `json:"storage_gb"`
	// Network bandwidth in Mbps
	BandwidthMbps float64 `json:"bandwidth_mbps"`
}

// NetworkProfile defines network characteristics of a site
type NetworkProfile struct {
	// Base latency to reach this site (ms)
	BaseLatencyMs float64 `json:"base_latency_ms"`
	// Maximum throughput in Mbps
	MaxThroughputMbps float64 `json:"max_throughput_mbps"`
	// Packet loss rate (0-1)
	PacketLossRate float64 `json:"packet_loss_rate"`
	// Jitter in ms
	JitterMs float64 `json:"jitter_ms"`
}

// SiteMetrics represents live metrics from a site
type SiteMetrics struct {
	// Timestamp of metrics collection
	Timestamp time.Time `json:"timestamp"`
	// CPU utilization (0-100)
	CPUUtilization float64 `json:"cpu_utilization"`
	// Memory utilization (0-100)
	MemoryUtilization float64 `json:"memory_utilization"`
	// Available bandwidth in Mbps
	AvailableBandwidthMbps float64 `json:"available_bandwidth_mbps"`
	// Current latency in ms
	CurrentLatencyMs float64 `json:"current_latency_ms"`
	// Active network functions count
	ActiveNFs int `json:"active_nfs"`
}

// NetworkFunction represents a VNF/CNF to be placed
type NetworkFunction struct {
	// Unique identifier
	ID string `json:"id"`
	// Type of network function (e.g., "UPF", "AMF", "SMF")
	Type string `json:"type"`
	// Resource requirements
	Requirements ResourceRequirements `json:"requirements"`
	// QoS requirements
	QoSRequirements QoSRequirements `json:"qos_requirements"`
	// Preferred placement hints
	PlacementHints []PlacementHint `json:"placement_hints,omitempty"`
}

// ResourceRequirements defines resources needed by a network function
type ResourceRequirements struct {
	// Minimum CPU cores required
	MinCPUCores int `json:"min_cpu_cores"`
	// Minimum memory in GB
	MinMemoryGB int `json:"min_memory_gb"`
	// Minimum storage in GB
	MinStorageGB int `json:"min_storage_gb"`
	// Minimum bandwidth in Mbps
	MinBandwidthMbps float64 `json:"min_bandwidth_mbps"`
}

// QoSRequirements defines QoS constraints for a network function
type QoSRequirements struct {
	// Maximum tolerable latency in ms
	MaxLatencyMs float64 `json:"max_latency_ms"`
	// Minimum required throughput in Mbps
	MinThroughputMbps float64 `json:"min_throughput_mbps"`
	// Maximum tolerable packet loss rate (0-1)
	MaxPacketLossRate float64 `json:"max_packet_loss_rate"`
	// Maximum tolerable jitter in ms
	MaxJitterMs float64 `json:"max_jitter_ms"`
}

// PlacementHint provides hints for placement decisions
type PlacementHint struct {
	// Type of hint
	Type HintType `json:"type"`
	// Value associated with the hint
	Value string `json:"value"`
	// Weight/priority of the hint (0-100)
	Weight int `json:"weight"`
}

// HintType represents types of placement hints
type HintType string

const (
	// HintTypeAffinity prefers placement near specified resources
	HintTypeAffinity HintType = "affinity"
	// HintTypeAntiAffinity avoids placement near specified resources
	HintTypeAntiAffinity HintType = "anti-affinity"
	// HintTypeLocation prefers specific geographic locations
	HintTypeLocation HintType = "location"
	// HintTypeCloudType prefers specific cloud types
	HintTypeCloudType HintType = "cloud-type"
)

// PlacementDecision represents the output of placement policy
type PlacementDecision struct {
	// Network function being placed
	NetworkFunction *NetworkFunction `json:"network_function"`
	// Selected site for placement
	Site *Site `json:"site"`
	// Score indicating quality of placement (0-100)
	Score float64 `json:"score"`
	// Reason for the placement decision
	Reason string `json:"reason"`
	// Alternative sites considered
	Alternatives []SiteScore `json:"alternatives,omitempty"`
	// Timestamp of decision
	Timestamp time.Time `json:"timestamp"`
}

// SiteScore represents a site with its placement score
type SiteScore struct {
	Site  *Site   `json:"site"`
	Score float64 `json:"score"`
}

// PlacementPolicy defines the interface for placement algorithms
type PlacementPolicy interface {
	// Place determines optimal placement for a network function
	Place(nf *NetworkFunction, sites []*Site) (*PlacementDecision, error)
	// PlaceMultiple handles batch placement with dependencies
	PlaceMultiple(nfs []*NetworkFunction, sites []*Site) ([]*PlacementDecision, error)
	// Rebalance optimizes existing placements
	Rebalance(decisions []*PlacementDecision, sites []*Site) ([]*PlacementDecision, error)
}

// MetricsProvider defines interface for retrieving site metrics
type MetricsProvider interface {
	// GetMetrics retrieves current metrics for a site
	GetMetrics(siteID string) (*SiteMetrics, error)
	// GetAllMetrics retrieves metrics for all sites
	GetAllMetrics() (map[string]*SiteMetrics, error)
	// Subscribe to metric updates
	Subscribe(siteID string, callback func(*SiteMetrics))
}

// PlacementError represents placement-specific errors
type PlacementError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *PlacementError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrNoSuitableSite     = "NO_SUITABLE_SITE"
	ErrInsufficientResources = "INSUFFICIENT_RESOURCES"
	ErrQoSViolation      = "QOS_VIOLATION"
	ErrSiteUnavailable   = "SITE_UNAVAILABLE"
)