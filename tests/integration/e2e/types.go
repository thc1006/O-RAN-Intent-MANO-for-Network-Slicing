// Package e2e provides type definitions for end-to-end integration tests
package e2e

import "time"

// ParsedIntent represents the result of NLP intent parsing
type ParsedIntent struct {
	IntentID    string            `json:"intent_id"`
	ServiceType string            `json:"service_type"`
	Throughput  string            `json:"throughput"`
	Latency     string            `json:"latency"`
	Reliability string            `json:"reliability,omitempty"`
	Coverage    string            `json:"coverage,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Analysis represents Claude AI analysis results
type Analysis struct {
	AnalysisID       string              `json:"analysis_id"`
	Requirements     map[string]string   `json:"requirements"`
	RecommendedVNFs  []string            `json:"recommended_vnfs"`
	ConfidenceScore  float64             `json:"confidence_score"`
	Recommendations  []string            `json:"recommendations,omitempty"`
	RiskAssessment   []string            `json:"risk_assessment,omitempty"`
	EstimatedCost    float64             `json:"estimated_cost,omitempty"`
	EstimatedLatency float64             `json:"estimated_latency,omitempty"`
	Timestamp        time.Time           `json:"timestamp"`
}

// Placement represents orchestrator placement calculation results
type Placement struct {
	PlacementID    string           `json:"placement_id"`
	CNFPlacements  []CNFPlacement   `json:"cnf_placements"`
	TransportLinks []TransportLink  `json:"transport_links"`
	TotalCPU       float64          `json:"total_cpu"`
	TotalMemory    float64          `json:"total_memory"`
	TotalBandwidth float64          `json:"total_bandwidth"`
	Score          float64          `json:"score"`
	Timestamp      time.Time        `json:"timestamp"`
}

// CNFPlacement represents placement of a single CNF
type CNFPlacement struct {
	CNFID           string            `json:"cnf_id"`
	CNFName         string            `json:"cnf_name"`
	NodeID          string            `json:"node_id"`
	NodeName        string            `json:"node_name"`
	CPUAllocated    float64           `json:"cpu_allocated"`
	MemoryAllocated float64           `json:"memory_allocated"`
	Affinity        map[string]string `json:"affinity,omitempty"`
}

// TransportLink represents a network transport link
type TransportLink struct {
	LinkID        string            `json:"link_id"`
	SourceNode    string            `json:"source_node"`
	DestNode      string            `json:"dest_node"`
	Bandwidth     float64           `json:"bandwidth"`
	Latency       float64           `json:"latency"`
	VXLANConfig   *VXLANConfig      `json:"vxlan_config,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// VXLANConfig represents VXLAN configuration for a link
type VXLANConfig struct {
	VNI         string `json:"vni"`
	LocalIP     string `json:"local_ip"`
	RemoteIP    string `json:"remote_ip"`
	MTU         int    `json:"mtu"`
	UDPPort     int    `json:"udp_port,omitempty"`
}

// Package represents a Nephio package
type Package struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Path        string            `json:"path"`
	Resources   []string          `json:"resources"`
	Dependencies []string         `json:"dependencies,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// DeploymentStatus represents VNF deployment status
type DeploymentStatus struct {
	DeploymentID   string            `json:"deployment_id"`
	State          string            `json:"state"` // Pending, Running, Ready, Failed, RolledBack
	AllPodsRunning bool              `json:"all_pods_running"`
	ReadyReplicas  int               `json:"ready_replicas"`
	TotalReplicas  int               `json:"total_replicas"`
	Conditions     []StatusCondition `json:"conditions,omitempty"`
	Message        string            `json:"message,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// StatusCondition represents a deployment condition
type StatusCondition struct {
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	Reason  string    `json:"reason,omitempty"`
	Message string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"last_transition_time"`
}

// VXLANInterface represents a VXLAN network interface
type VXLANInterface struct {
	Name       string `json:"name"`
	VNI        string `json:"vni"`
	LocalIP    string `json:"local_ip"`
	RemoteIP   string `json:"remote_ip"`
	Status     string `json:"status"` // up, down
	MTU        int    `json:"mtu"`
	TXBytes    int64  `json:"tx_bytes"`
	RXBytes    int64  `json:"rx_bytes"`
	CreatedAt  time.Time `json:"created_at"`
}

// SliceStatus represents operational status of a network slice
type SliceStatus struct {
	SliceID        string              `json:"slice_id"`
	State          string              `json:"state"` // Provisioning, Active, Degraded, Failed
	HealthCheck    bool                `json:"health_check"`
	Throughput     float64             `json:"throughput"` // Mbps
	Latency        float64             `json:"latency"`    // ms
	PacketLoss     float64             `json:"packet_loss,omitempty"` // percentage
	Jitter         float64             `json:"jitter,omitempty"`      // ms
	ActiveVNFs     []ActiveVNF         `json:"active_vnfs"`
	TransportLinks []string            `json:"transport_links"`
	Metrics        *SliceMetrics       `json:"metrics,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// ActiveVNF represents an active VNF in the slice
type ActiveVNF struct {
	VNFID      string    `json:"vnf_id"`
	VNFName    string    `json:"vnf_name"`
	Status     string    `json:"status"` // Running, Degraded, Failed
	CPUUsage   float64   `json:"cpu_usage"`
	MemoryUsage float64  `json:"memory_usage"`
	LastCheck  time.Time `json:"last_check"`
}

// SliceMetrics contains detailed metrics for a slice
type SliceMetrics struct {
	AvgThroughput    float64 `json:"avg_throughput"`
	MaxThroughput    float64 `json:"max_throughput"`
	MinThroughput    float64 `json:"min_throughput"`
	AvgLatency       float64 `json:"avg_latency"`
	MaxLatency       float64 `json:"max_latency"`
	MinLatency       float64 `json:"min_latency"`
	TotalDataTransfer int64  `json:"total_data_transfer"` // bytes
	Uptime           int64   `json:"uptime"`              // seconds
	SuccessRate      float64 `json:"success_rate"`        // percentage
}

// ResourceConstraints represents resource constraints for placement
type ResourceConstraints struct {
	MaxCPU      float64           `json:"max_cpu"`
	MaxMemory   float64           `json:"max_memory"`
	MaxBandwidth float64          `json:"max_bandwidth,omitempty"`
	NodeLabels  map[string]string `json:"node_labels,omitempty"`
	Affinity    []string          `json:"affinity,omitempty"`
	AntiAffinity []string         `json:"anti_affinity,omitempty"`
}

// InvalidDeploymentConfig represents an invalid deployment configuration for testing
type InvalidDeploymentConfig struct {
	MissingFields  bool   `json:"missing_fields"`
	InvalidValues  bool   `json:"invalid_values"`
	ConflictingConfig bool `json:"conflicting_config"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// TestScenario represents a test scenario configuration
type TestScenario struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Intent          string                 `json:"intent"`
	ExpectedResult  string                 `json:"expected_result"`
	ShouldFail      bool                   `json:"should_fail"`
	Constraints     *ResourceConstraints   `json:"constraints,omitempty"`
	CustomConfig    map[string]interface{} `json:"custom_config,omitempty"`
	TimeoutSeconds  int                    `json:"timeout_seconds"`
}

// MockResponse represents a mock HTTP response for testing
type MockResponse struct {
	StatusCode int                    `json:"status_code"`
	Body       map[string]interface{} `json:"body"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Delay      time.Duration          `json:"delay,omitempty"`
}

// TestMetrics represents metrics collected during test execution
type TestMetrics struct {
	TestName           string        `json:"test_name"`
	StartTime          time.Time     `json:"start_time"`
	EndTime            time.Time     `json:"end_time"`
	Duration           time.Duration `json:"duration"`
	Success            bool          `json:"success"`
	ErrorMessage       string        `json:"error_message,omitempty"`
	ComponentMetrics   map[string]ComponentMetric `json:"component_metrics"`
	ResourceUsage      *ResourceUsage `json:"resource_usage,omitempty"`
}

// ComponentMetric represents metrics for a single component
type ComponentMetric struct {
	ComponentName    string        `json:"component_name"`
	ResponseTime     time.Duration `json:"response_time"`
	RequestCount     int           `json:"request_count"`
	ErrorCount       int           `json:"error_count"`
	SuccessRate      float64       `json:"success_rate"`
}

// ResourceUsage represents resource usage during test
type ResourceUsage struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsageMB      float64 `json:"memory_usage_mb"`
	NetworkBandwidthMB float64 `json:"network_bandwidth_mb"`
	DiskIOPS           int64   `json:"disk_iops"`
}