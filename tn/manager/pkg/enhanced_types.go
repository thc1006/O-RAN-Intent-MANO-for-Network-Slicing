package pkg

import (
	"context"
	"fmt"
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

// NewNetworkTopology creates a new NetworkTopology instance
func NewNetworkTopology() *NetworkTopology {
	return &NetworkTopology{
		Nodes:       make(map[string]*TopologyNode),
		Links:       make(map[string]*TopologyLink),
		LastUpdated: time.Now(),
		Version:     "1.0.0",
	}
}

// GetNodes returns a slice of all topology nodes
func (nt *NetworkTopology) GetNodes() []*TopologyNode {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	nodes := make([]*TopologyNode, 0, len(nt.Nodes))
	for _, node := range nt.Nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// AddLink adds a link to the topology
func (nt *NetworkTopology) AddLink(link *TopologyLink) {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	linkID := fmt.Sprintf("%s-%s", link.SourceNode, link.TargetNode)
	nt.Links[linkID] = link
	nt.LastUpdated = time.Now()
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

// CompareAndNotifyChanges compares new topology against cached old topology and notifies of changes
func (td *TopologyDiscovery) CompareAndNotifyChanges(newTopology *NetworkTopology) {
	td.mutex.Lock()
	defer td.mutex.Unlock()

	// For now, just log the topology update
	// TODO: Implement actual change detection and notification
	if newTopology != nil {
		// Store for future comparison
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

// StartMonitoring starts the fault monitoring process
func (fd *FaultDetector) StartMonitoring(ctx context.Context, agents map[string]*TNAgentClient, faultCallback func(*NetworkFault)) {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()

	// Start background monitoring
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fd.mutex.Lock()
				// Monitor agents for faults
				for nodeID, agent := range agents {
					if !agent.IsConnected() {
						faultID := fmt.Sprintf("fault-%s-%d", nodeID, time.Now().Unix())
						fault := &NetworkFault{
							ID:          faultID,
							Type:        FaultTypeNodeUnreachable,
							Severity:    FaultSeverityHigh,
							NodeName:    nodeID,
							DetectedAt:  time.Now(),
							Description: fmt.Sprintf("Agent %s is disconnected", nodeID),
							Details:     map[string]interface{}{"reason": "agent_disconnected"},
						}

						fd.activeFaults[faultID] = fault
						fd.faultHistory = append(fd.faultHistory, fault)

						if faultCallback != nil {
							faultCallback(fault)
						}
					}
				}
				fd.mutex.Unlock()
			}
		}
	}()
}

// GetFaultsSummary returns a summary of active and recent faults
func (fd *FaultDetector) GetFaultsSummary() *FaultsSummary {
	fd.mutex.RLock()
	defer fd.mutex.RUnlock()

	faultsByType := make(map[FaultType]int)
	criticalCount := 0
	activeCount := len(fd.activeFaults)
	resolvedCount := 0

	recentFaults := make([]*NetworkFault, 0)
	for _, fault := range fd.faultHistory {
		faultsByType[fault.Type]++
		if fault.Severity == FaultSeverityCritical {
			criticalCount++
		}
		if fault.ResolvedAt != nil {
			resolvedCount++
		}
		if len(recentFaults) < 10 {
			recentFaults = append(recentFaults, fault)
		}
	}

	return &FaultsSummary{
		TotalFaults:    len(fd.faultHistory),
		CriticalFaults: criticalCount,
		ActiveFaults:   activeCount,
		ResolvedFaults: resolvedCount,
		FaultsByType:   faultsByType,
		RecentFaults:   recentFaults,
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

// UpdateVXLANConfig updates VXLAN configuration for a slice
func (ns *NetworkState) UpdateVXLANConfig(sliceID string, config *DynamicVXLANConfig) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	ns.sliceConfigs[sliceID] = config
	return nil
}

// GetVXLANConfig retrieves VXLAN configuration for a slice
func (ns *NetworkState) GetVXLANConfig(sliceID string) (*DynamicVXLANConfig, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	config, exists := ns.sliceConfigs[sliceID]
	if !exists {
		return nil, fmt.Errorf("VXLAN config not found for slice %s", sliceID)
	}
	return config, nil
}

// UpdateQoSStrategy updates QoS strategy for a slice
func (ns *NetworkState) UpdateQoSStrategy(sliceID string, strategy *QoSStrategy) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	ns.qosStrategies[sliceID] = strategy
	return nil
}

// GetQoSStrategy retrieves QoS strategy for a slice
func (ns *NetworkState) GetQoSStrategy(sliceID string) (*QoSStrategy, error) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	strategy, exists := ns.qosStrategies[sliceID]
	if !exists {
		return nil, fmt.Errorf("QoS strategy not found for slice %s", sliceID)
	}
	return strategy, nil
}

// UpdateTopology updates the network topology
func (ns *NetworkState) UpdateTopology(topology *NetworkTopology) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	ns.topology = topology
	return nil
}

// GetSliceVXLANConfigs returns VXLAN configurations for all slices on a specific node
func (ns *NetworkState) GetSliceVXLANConfigs(nodeID string) map[string]*DynamicVXLANConfig {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	configs := make(map[string]*DynamicVXLANConfig)
	for sliceID, config := range ns.sliceConfigs {
		// Check if this slice uses the specified node
		for _, endpoint := range config.Endpoints {
			if endpoint.NodeName == nodeID {
				configs[sliceID] = config
				break
			}
		}
	}
	return configs
}

// GetSliceQoSStrategies returns QoS strategies for all slices on a specific node
func (ns *NetworkState) GetSliceQoSStrategies(nodeID string) map[string]*QoSStrategy {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	strategies := make(map[string]*QoSStrategy)
	for sliceID, strategy := range ns.qosStrategies {
		// Check if this slice uses the specified node
		if config, exists := ns.sliceConfigs[sliceID]; exists {
			for _, endpoint := range config.Endpoints {
				if endpoint.NodeName == nodeID {
					strategies[sliceID] = strategy
					break
				}
			}
		}
	}
	return strategies
}

// GetSlicesUsingNode returns all slice IDs that use a specific node
func (ns *NetworkState) GetSlicesUsingNode(nodeID string) []string {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	slices := make([]string, 0)
	for sliceID, config := range ns.sliceConfigs {
		for _, endpoint := range config.Endpoints {
			if endpoint.NodeName == nodeID {
				slices = append(slices, sliceID)
				break
			}
		}
	}
	return slices
}

// GetTopology returns the current network topology
func (ns *NetworkState) GetTopology() *NetworkTopology {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	return ns.topology
}

// GetActiveSlices returns the list of active slices
func (ns *NetworkState) GetActiveSlices() map[string]*SliceState {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	// Return a copy to avoid concurrent modification
	slices := make(map[string]*SliceState, len(ns.activeSlices))
	for sliceID, state := range ns.activeSlices {
		slices[sliceID] = state
	}
	return slices
}

// GetVXLANStatus returns VXLAN status for all slices
func (ns *NetworkState) GetVXLANStatus() *VXLANStatusSummary {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()

	totalTunnels := len(ns.sliceConfigs)
	activeTunnels := 0
	failedTunnels := 0
	tunnelsBySlice := make(map[string]int)

	for sliceID, config := range ns.sliceConfigs {
		tunnelsBySlice[sliceID] = len(config.Endpoints)
		// Assume active if config exists (proper status tracking would be more complex)
		activeTunnels++
	}

	return &VXLANStatusSummary{
		TotalTunnels:   totalTunnels,
		ActiveTunnels:  activeTunnels,
		FailedTunnels:  failedTunnels,
		TunnelsBySlice: tunnelsBySlice,
	}
}
