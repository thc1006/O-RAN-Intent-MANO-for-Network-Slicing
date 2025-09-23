package pkg

import (
	"time"
)

// TNConfig represents Transport Network configuration
type TNConfig struct {
	ClusterName     string            `json:"clusterName" yaml:"clusterName"`
	NetworkCIDR     string            `json:"networkCIDR" yaml:"networkCIDR"`
	VXLANConfig     VXLANConfig       `json:"vxlan" yaml:"vxlan"`
	BWPolicy        BandwidthPolicy   `json:"bandwidthPolicy" yaml:"bandwidthPolicy"`
	QoSClass        string            `json:"qosClass" yaml:"qosClass"`
	Interfaces      []NetworkInterface `json:"interfaces" yaml:"interfaces"`
	MonitoringPort  int               `json:"monitoringPort" yaml:"monitoringPort"`
}

// VXLANConfig defines VXLAN tunnel configuration
type VXLANConfig struct {
	VNI         uint32   `json:"vni" yaml:"vni"`
	RemoteIPs   []string `json:"remoteIPs" yaml:"remoteIPs"`
	LocalIP     string   `json:"localIP" yaml:"localIP"`
	Port        int      `json:"port" yaml:"port"`
	MTU         int      `json:"mtu" yaml:"mtu"`
	DeviceName  string   `json:"deviceName" yaml:"deviceName"`
	Learning    bool     `json:"learning" yaml:"learning"`
}

// BandwidthPolicy defines traffic shaping parameters
type BandwidthPolicy struct {
	DownlinkMbps float64   `json:"downlinkMbps" yaml:"downlinkMbps"`
	UplinkMbps   float64   `json:"uplinkMbps" yaml:"uplinkMbps"`
	LatencyMs    float64   `json:"latencyMs" yaml:"latencyMs"`
	JitterMs     float64   `json:"jitterMs" yaml:"jitterMs"`
	LossPercent  float64   `json:"lossPercent" yaml:"lossPercent"`
	Priority     int       `json:"priority" yaml:"priority"`
	QueueClass   string    `json:"queueClass" yaml:"queueClass"`
	Burst        string    `json:"burst" yaml:"burst"`
	Filters      []Filter  `json:"filters" yaml:"filters"`
}

// Filter defines traffic classification rules
type Filter struct {
	Protocol   string `json:"protocol" yaml:"protocol"`
	SrcIP      string `json:"srcIP" yaml:"srcIP"`
	DstIP      string `json:"dstIP" yaml:"dstIP"`
	SrcPort    int    `json:"srcPort" yaml:"srcPort"`
	DstPort    int    `json:"dstPort" yaml:"dstPort"`
	FlowID     string `json:"flowID" yaml:"flowID"`
	Action     string `json:"action" yaml:"action"`
	ClassID    string `json:"classID" yaml:"classID"`
	Priority   int    `json:"priority" yaml:"priority"`
}

// NetworkInterface represents a network interface configuration
type NetworkInterface struct {
	Name      string `json:"name" yaml:"name"`
	Type      string `json:"type" yaml:"type"`
	IP        string `json:"ip" yaml:"ip"`
	Netmask   string `json:"netmask" yaml:"netmask"`
	Gateway   string `json:"gateway" yaml:"gateway"`
	MTU       int    `json:"mtu" yaml:"mtu"`
	State     string `json:"state" yaml:"state"`
}

// PerformanceMetrics contains network performance measurements
type PerformanceMetrics struct {
	Timestamp        time.Time       `json:"timestamp"`
	ClusterName      string          `json:"clusterName"`
	TestID           string          `json:"testId"`
	TestType         string          `json:"testType"`
	Duration         time.Duration   `json:"duration"`
	Throughput       ThroughputMetrics `json:"throughput"`
	Latency          LatencyMetrics    `json:"latency"`
	PacketLoss       float64         `json:"packetLoss"`
	Jitter           float64         `json:"jitter"`
	BandwidthUtilization float64     `json:"bandwidthUtilization"`
	QoSClass         string          `json:"qosClass"`
	VXLANOverhead    float64         `json:"vxlanOverhead"`
	TCOverhead       float64         `json:"tcOverhead"`
	NetworkPath      []string        `json:"networkPath"`
	ErrorDetails     []string        `json:"errorDetails,omitempty"`
}

// ThroughputMetrics contains detailed throughput measurements
type ThroughputMetrics struct {
	DownlinkMbps  float64 `json:"downlinkMbps"`
	UplinkMbps    float64 `json:"uplinkMbps"`
	BiDirMbps     float64 `json:"biDirMbps"`
	TargetMbps    float64 `json:"targetMbps"`
	AchievedRatio float64 `json:"achievedRatio"`
	PeakMbps      float64 `json:"peakMbps"`
	MinMbps       float64 `json:"minMbps"`
	AvgMbps       float64 `json:"avgMbps"`
	StdDevMbps    float64 `json:"stdDevMbps"`
}

// LatencyMetrics contains detailed latency measurements
type LatencyMetrics struct {
	RTTMs     float64 `json:"rttMs"`
	MinRTTMs  float64 `json:"minRttMs"`
	MaxRTTMs  float64 `json:"maxRttMs"`
	AvgRTTMs  float64 `json:"avgRttMs"`
	StdDevMs  float64 `json:"stdDevMs"`
	P50Ms     float64 `json:"p50Ms"`
	P95Ms     float64 `json:"p95Ms"`
	P99Ms     float64 `json:"p99Ms"`
	TargetMs  float64 `json:"targetMs"`
}

// NetworkSliceMetrics aggregates metrics for network slice validation
type NetworkSliceMetrics struct {
	SliceID          string              `json:"sliceId"`
	SliceType        string              `json:"sliceType"`
	Timestamp        time.Time           `json:"timestamp"`
	SLACompliance    bool                `json:"slaCompliance"`
	Performance      PerformanceMetrics  `json:"performance"`
	ThesisValidation ThesisValidation    `json:"thesisValidation"`
	ClusterMetrics   map[string]PerformanceMetrics `json:"clusterMetrics"`
}

// ThesisValidation validates against thesis target metrics
type ThesisValidation struct {
	ThroughputTargets []float64 `json:"throughputTargets"` // [0.93, 2.77, 4.57] Mbps
	RTTTargets        []float64 `json:"rttTargets"`        // [6.3, 15.7, 16.1] ms
	ThroughputResults []float64 `json:"throughputResults"`
	RTTResults        []float64 `json:"rttResults"`
	PassedTests       int       `json:"passedTests"`
	TotalTests        int       `json:"totalTests"`
	CompliancePercent float64   `json:"compliancePercent"`
	DeployTimeMs      int64     `json:"deployTimeMs"`
	DeployTargetMs    int64     `json:"deployTargetMs"`   // 10 minutes = 600000ms
}

// TNStatus represents the current status of TN agent
type TNStatus struct {
	Healthy          bool                `json:"healthy"`
	LastUpdate       time.Time           `json:"lastUpdate"`
	ActiveConnections int                `json:"activeConnections"`
	BandwidthUsage   map[string]float64  `json:"bandwidthUsage"`
	VXLANStatus      VXLANStatus         `json:"vxlanStatus"`
	TCStatus         TCStatus            `json:"tcStatus"`
	ErrorMessages    []string            `json:"errorMessages,omitempty"`
}

// VXLANStatus represents VXLAN tunnel status
type VXLANStatus struct {
	TunnelUp      bool                `json:"tunnelUp"`
	RemotePeers   []string            `json:"remotePeers"`
	PacketStats   map[string]int64    `json:"packetStats"`
	LastHeartbeat time.Time           `json:"lastHeartbeat"`
}

// TCStatus represents Traffic Control status
type TCStatus struct {
	RulesActive   bool                `json:"rulesActive"`
	QueueStats    map[string]int64    `json:"queueStats"`
	ShapingActive bool                `json:"shapingActive"`
	Interfaces    []string            `json:"interfaces"`
}