package pkg

import (
	"log"
	"sync"
	"time"

	tnv1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/api/v1alpha1"
)

// Enhanced TN Manager Types

// DynamicVXLANConfig represents dynamic VXLAN configuration
type DynamicVXLANConfig struct {
	VxlanID        int32               `json:"vxlanId"`
	Endpoints      []TNEndpoint        `json:"endpoints"`
	ClusterMapping map[string]string   `json:"clusterMapping"` // IP to cluster mapping
	MTU            int                 `json:"mtu"`
	Port           int                 `json:"port"`
	Encryption     *VXLANEncryption    `json:"encryption,omitempty"`
	QoS            *VXLANQoS           `json:"qos,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// TNEndpoint represents a transport network endpoint
type TNEndpoint struct {
	tnv1alpha1.Endpoint
	Capabilities []string               `json:"capabilities,omitempty"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// VXLANEncryption defines encryption settings for VXLAN tunnels
type VXLANEncryption struct {
	Enabled    bool   `json:"enabled"`
	Algorithm  string `json:"algorithm"`  // aes256, chacha20
	KeyRotation int   `json:"keyRotation"` // rotation interval in hours
}

// VXLANQoS defines QoS settings for VXLAN tunnels
type VXLANQoS struct {
	DSCP     int     `json:"dscp"`
	Priority int     `json:"priority"`
	Bandwidth string `json:"bandwidth"`
}

// VXLANUpdateConfig represents updates to VXLAN configuration
type VXLANUpdateConfig struct {
	AddEndpoints    []TNEndpoint `json:"addEndpoints,omitempty"`
	RemoveEndpoints []TNEndpoint `json:"removeEndpoints,omitempty"`
	MTU             int          `json:"mtu,omitempty"`
	UpdateQoS       *VXLANQoS    `json:"updateQos,omitempty"`
}

// QoS Strategy Management Types

// QoSStrategy defines a comprehensive QoS strategy
type QoSStrategy struct {
	Type            QoSStrategyType        `json:"type"`
	Priority        int                    `json:"priority"`
	BandwidthLimits map[string]string      `json:"bandwidthLimits"` // direction -> limit
	LatencyTargets  map[string]float64     `json:"latencyTargets"`  // metric -> target
	JitterLimits    map[string]float64     `json:"jitterLimits"`
	PacketLossLimits map[string]float64    `json:"packetLossLimits"`
	TrafficClasses  []TrafficClass         `json:"trafficClasses"`
	SchedulingPolicy SchedulingPolicy      `json:"schedulingPolicy"`
	CongestionControl CongestionControl    `json:"congestionControl"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// QoSStrategyType defines different QoS strategy types
type QoSStrategyType string

const (
	QoSStrategyTypeULLC     QoSStrategyType = "uRLLC"     // Ultra-Reliable Low Latency
	QoSStrategyTypeEMBB     QoSStrategyType = "eMBB"      // Enhanced Mobile Broadband
	QoSStrategyTypeMIOT     QoSStrategyType = "mIoT"      // Massive IoT
	QoSStrategyTypeCustom   QoSStrategyType = "custom"
)

// TrafficClass defines traffic classification and handling
type TrafficClass struct {
	Name        string                 `json:"name"`
	Selector    TrafficSelector        `json:"selector"`
	Priority    int                    `json:"priority"`
	Bandwidth   string                 `json:"bandwidth"`
	Latency     float64                `json:"latency"`
	Actions     []TrafficAction        `json:"actions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TrafficSelector defines how to identify traffic for a class
type TrafficSelector struct {
	Protocol    string            `json:"protocol,omitempty"`
	SourceIP    string            `json:"sourceIp,omitempty"`
	DestIP      string            `json:"destIp,omitempty"`
	SourcePort  int               `json:"sourcePort,omitempty"`
	DestPort    int               `json:"destPort,omitempty"`
	DSCP        int               `json:"dscp,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// TrafficAction defines actions to apply to classified traffic
type TrafficAction struct {
	Type       string                 `json:"type"`       // mark, police, shape, drop
	Parameters map[string]interface{} `json:"parameters"`
}

// SchedulingPolicy defines packet scheduling behavior
type SchedulingPolicy struct {
	Algorithm   string                 `json:"algorithm"`   // fifo, fair, priority, cbq
	Parameters  map[string]interface{} `json:"parameters"`
	Queues      []QueueConfig          `json:"queues"`
}

// QueueConfig defines queue configuration
type QueueConfig struct {
	ID          string  `json:"id"`
	Weight      int     `json:"weight"`
	Bandwidth   string  `json:"bandwidth"`
	BurstSize   string  `json:"burstSize"`
	Priority    int     `json:"priority"`
}

// CongestionControl defines congestion control mechanisms
type CongestionControl struct {
	Algorithm   string                 `json:"algorithm"`   // red, wred, codel, fq_codel
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
}

// QoSUpdates represents updates to QoS strategy
type QoSUpdates struct {
	BandwidthChanges map[string]string     `json:"bandwidthChanges,omitempty"`
	LatencyChanges   map[string]float64    `json:"latencyChanges,omitempty"`
	PriorityChanges  map[string]int        `json:"priorityChanges,omitempty"`
	AddTrafficClasses []TrafficClass       `json:"addTrafficClasses,omitempty"`
	RemoveTrafficClasses []string          `json:"removeTrafficClasses,omitempty"`
	UpdateScheduling *SchedulingPolicy     `json:"updateScheduling,omitempty"`
}

// Network Topology Types

// NetworkTopology represents the complete network topology
type NetworkTopology struct {
	Nodes       map[string]*TopologyNode `json:"nodes"`
	Links       map[string]*TopologyLink `json:"links"`
	LastUpdated time.Time                `json:"lastUpdated"`
	Version     string                   `json:"version"`
	mutex       sync.RWMutex
}

// TopologyNode represents a node in the network
type TopologyNode struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`         // compute, network, storage
	Capabilities []string               `json:"capabilities"`
	Interfaces   []NodeInterface        `json:"interfaces"`
	Status       string                 `json:"status"`       // healthy, degraded, failed
	Location     *NodeLocation          `json:"location,omitempty"`
	Resources    *NodeResources         `json:"resources,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	LastUpdated  time.Time              `json:"lastUpdated"`
}

// NodeInterface represents a network interface on a node
type NodeInterface struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`         // physical, virtual, bridge
	MAC          string                 `json:"mac"`
	IP           string                 `json:"ip"`
	Speed        string                 `json:"speed"`        // 1Gbps, 10Gbps, etc.
	Status       string                 `json:"status"`       // up, down, unknown
	Utilization  float64                `json:"utilization"`  // percentage
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NodeLocation represents the physical or logical location of a node
type NodeLocation struct {
	Region       string  `json:"region"`
	Zone         string  `json:"zone"`
	DataCenter   string  `json:"dataCenter"`
	Rack         string  `json:"rack"`
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
}

// NodeResources represents available resources on a node
type NodeResources struct {
	CPU         NodeResource `json:"cpu"`
	Memory      NodeResource `json:"memory"`
	Storage     NodeResource `json:"storage"`
	Network     NodeResource `json:"network"`
	GPU         *NodeResource `json:"gpu,omitempty"`
}

// NodeResource represents a specific resource
type NodeResource struct {
	Total     string  `json:"total"`
	Available string  `json:"available"`
	Used      string  `json:"used"`
	Unit      string  `json:"unit"`
	Utilization float64 `json:"utilization"`
}

// TopologyLink represents a link between nodes
type TopologyLink struct {
	ID          string                 `json:"id"`
	SourceNode  string                 `json:"sourceNode"`
	TargetNode  string                 `json:"targetNode"`
	Type        string                 `json:"type"`         // network, storage, management
	Status      string                 `json:"status"`       // up, down, degraded
	Metrics     LinkMetrics            `json:"metrics"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	LastUpdated time.Time              `json:"lastUpdated"`
}

// LinkMetrics represents link performance metrics
type LinkMetrics struct {
	Bandwidth   float64 `json:"bandwidth"`   // Mbps
	Latency     float64 `json:"latency"`     // ms
	PacketLoss  float64 `json:"packetLoss"`  // percentage
	Jitter      float64 `json:"jitter"`      // ms
	Utilization float64 `json:"utilization"` // percentage
}

// Fault Detection Types

// FaultDetector manages fault detection and reporting
type FaultDetector struct {
	logger          *log.Logger
	activeFaults    map[string]*NetworkFault
	faultHistory    []*NetworkFault
	detectionRules  []FaultDetectionRule
	mutex           sync.RWMutex
}

// NetworkFault represents a detected network fault
type NetworkFault struct {
	ID          string                 `json:"id"`
	Type        FaultType              `json:"type"`
	Severity    FaultSeverity          `json:"severity"`
	NodeName    string                 `json:"nodeName"`
	SliceID     string                 `json:"sliceId,omitempty"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	DetectedAt  time.Time              `json:"detectedAt"`
	ResolvedAt  *time.Time             `json:"resolvedAt,omitempty"`
	Actions     []RecoveryAction       `json:"actions,omitempty"`
}

// FaultType represents different types of network faults
type FaultType string

const (
	FaultTypeVXLANDown     FaultType = "vxlan_down"
	FaultTypeQoSViolation  FaultType = "qos_violation"
	FaultTypeLinkDown      FaultType = "link_down"
	FaultTypeHighLatency   FaultType = "high_latency"
	FaultTypePacketLoss    FaultType = "packet_loss"
	FaultTypeBandwidthUsage FaultType = "bandwidth_usage"
	FaultTypeNodeUnreachable FaultType = "node_unreachable"
)

// FaultSeverity represents fault severity levels
type FaultSeverity string

const (
	FaultSeverityLow      FaultSeverity = "low"
	FaultSeverityMedium   FaultSeverity = "medium"
	FaultSeverityHigh     FaultSeverity = "high"
	FaultSeverityCritical FaultSeverity = "critical"
)

// FaultDetectionRule defines rules for fault detection
type FaultDetectionRule struct {
	Name        string                 `json:"name"`
	Type        FaultType              `json:"type"`
	Condition   string                 `json:"condition"`   // expression to evaluate
	Threshold   map[string]interface{} `json:"threshold"`
	Interval    time.Duration          `json:"interval"`
	Enabled     bool                   `json:"enabled"`
}

// RecoveryAction represents an automated recovery action
type RecoveryAction struct {
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	ExecutedAt  time.Time              `json:"executedAt"`
	Result      string                 `json:"result"`
	Error       string                 `json:"error,omitempty"`
}

// Event Management Types

// TNEvent represents a Transport Network event
type TNEvent struct {
	Type      TNEventType            `json:"type"`
	SliceID   string                 `json:"sliceId,omitempty"`
	NodeName  string                 `json:"nodeName,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Severity  string                 `json:"severity"`
}

// TNEventType represents different types of TN events
type TNEventType string

const (
	EventTypeVXLANConfigured        TNEventType = "vxlan_configured"
	EventTypeVXLANRecovered         TNEventType = "vxlan_recovered"
	EventTypeQoSConfigured          TNEventType = "qos_configured"
	EventTypeQoSRecovered           TNEventType = "qos_recovered"
	EventTypeTopologyDiscovered     TNEventType = "topology_discovered"
	EventTypeTopologyChanged        TNEventType = "topology_changed"
	EventTypeFaultDetectionStarted  TNEventType = "fault_detection_started"
	EventTypeFaultDetected          TNEventType = "fault_detected"
	EventTypeFaultResolved          TNEventType = "fault_resolved"
	EventTypeSliceConfigured        TNEventType = "slice_configured"
	EventTypeSliceTerminated        TNEventType = "slice_terminated"
)

// TNEventHandler processes TN events
type TNEventHandler func(event TNEvent)

// Network State Management Types

// NetworkState manages the current state of the network
type NetworkState struct {
	topology        *NetworkTopology
	sliceConfigs    map[string]*DynamicVXLANConfig
	qosStrategies   map[string]*QoSStrategy
	activeSlices    map[string]*SliceState
	mutex           sync.RWMutex
}

// SliceState represents the state of a network slice
type SliceState struct {
	SliceID      string                 `json:"sliceId"`
	Status       string                 `json:"status"`
	VXLANConfig  *DynamicVXLANConfig    `json:"vxlanConfig,omitempty"`
	QoSStrategy  *QoSStrategy           `json:"qosStrategy,omitempty"`
	Nodes        []string               `json:"nodes"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Enhanced Status Types

// EnhancedTNStatus provides comprehensive TN status
type EnhancedTNStatus struct {
	BaseStatus      map[string]*TNStatus   `json:"baseStatus"`
	NetworkTopology *NetworkTopology       `json:"networkTopology"`
	ActiveSlices    map[string]*SliceState `json:"activeSlices"`
	FaultsSummary   *FaultsSummary         `json:"faultsSummary"`
	QoSCompliance   *QoSComplianceSummary  `json:"qosCompliance"`
	VXLANStatus     *VXLANStatusSummary    `json:"vxlanStatus"`
	LastUpdated     time.Time              `json:"lastUpdated"`
}

// FaultsSummary provides a summary of network faults
type FaultsSummary struct {
	TotalFaults     int                        `json:"totalFaults"`
	CriticalFaults  int                        `json:"criticalFaults"`
	ActiveFaults    int                        `json:"activeFaults"`
	ResolvedFaults  int                        `json:"resolvedFaults"`
	FaultsByType    map[FaultType]int          `json:"faultsByType"`
	RecentFaults    []*NetworkFault            `json:"recentFaults"`
}

// QoSComplianceSummary provides QoS compliance information
type QoSComplianceSummary struct {
	OverallCompliance float64                    `json:"overallCompliance"`
	SliceCompliance   map[string]float64         `json:"sliceCompliance"`
	Violations        []QoSViolation             `json:"violations"`
	LastUpdated       time.Time                  `json:"lastUpdated"`
}

// QoSViolation represents a QoS policy violation
type QoSViolation struct {
	SliceID     string                 `json:"sliceId"`
	MetricType  string                 `json:"metricType"`
	Expected    interface{}            `json:"expected"`
	Actual      interface{}            `json:"actual"`
	Severity    string                 `json:"severity"`
	DetectedAt  time.Time              `json:"detectedAt"`
	Details     map[string]interface{} `json:"details"`
}

// VXLANStatusSummary provides VXLAN status summary
type VXLANStatusSummary struct {
	TotalTunnels    int                    `json:"totalTunnels"`
	ActiveTunnels   int                    `json:"activeTunnels"`
	FailedTunnels   int                    `json:"failedTunnels"`
	TunnelsBySlice  map[string]int         `json:"tunnelsBySlice"`
	OverheadStats   *OverheadStats         `json:"overheadStats"`
	LastUpdated     time.Time              `json:"lastUpdated"`
}

// OverheadStats provides VXLAN overhead statistics
type OverheadStats struct {
	AverageOverhead float64 `json:"averageOverhead"` // percentage
	MaxOverhead     float64 `json:"maxOverhead"`
	MinOverhead     float64 `json:"minOverhead"`
	TotalBytes      int64   `json:"totalBytes"`
	OverheadBytes   int64   `json:"overheadBytes"`
}

// Discovery Types

// NodeDiscoveryInfo represents discovered node information
type NodeDiscoveryInfo struct {
	Type         string                 `json:"type"`
	Capabilities []string               `json:"capabilities"`
	Interfaces   []NodeInterface        `json:"interfaces"`
	Status       string                 `json:"status"`
	Resources    *NodeResources         `json:"resources,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
// TopologyDiscovery manages network topology discovery
type TopologyDiscovery struct {
	logger *log.Logger
	mutex  sync.RWMutex
}

// NewTopologyDiscovery creates a new topology discovery instance
func NewTopologyDiscovery(logger *log.Logger) *TopologyDiscovery {
	return &TopologyDiscovery{
		logger: logger,
	}
}

// NewFaultDetector creates a new fault detector instance
func NewFaultDetector(logger *log.Logger) *FaultDetector {
	return &FaultDetector{
		logger:         logger,
		activeFaults:   make(map[string]*NetworkFault),
		faultHistory:   make([]*NetworkFault, 0),
		detectionRules: make([]FaultDetectionRule, 0),
	}
}

// NewNetworkState creates a new network state instance
func NewNetworkState() *NetworkState {
	return &NetworkState{
		topology:      &NetworkTopology{},
		sliceConfigs:  make(map[string]*DynamicVXLANConfig),
		qosStrategies: make(map[string]*QoSStrategy),
		activeSlices:  make(map[string]*SliceState),
	}
}
