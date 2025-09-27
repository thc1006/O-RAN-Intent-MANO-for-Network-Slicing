package fixtures

import (
	"encoding/json"
	"time"
)

// Intent represents a natural language intent for network slicing
type Intent struct {
	ID          string                 `json:"id"`
	Text        string                 `json:"text"`
	Type        IntentType             `json:"type"`
	Priority    Priority               `json:"priority"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     IntentContext          `json:"context"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

type IntentType string

const (
	IntentTypeSliceCreate   IntentType = "slice-create"
	IntentTypeSliceModify   IntentType = "slice-modify"
	IntentTypeSliceDelete   IntentType = "slice-delete"
	IntentTypeQoSOptimize   IntentType = "qos-optimize"
	IntentTypeResourceScale IntentType = "resource-scale"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityCritical Priority = "critical"
)

type IntentContext struct {
	User        string    `json:"user"`
	Application string    `json:"application"`
	Region      string    `json:"region"`
	TimeWindow  TimeWindow `json:"timeWindow,omitempty"`
}

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ParsedIntent represents the structured output after parsing natural language
type ParsedIntent struct {
	Intent      Intent             `json:"intent"`
	SliceType   SliceType          `json:"sliceType"`
	QoSProfile  QoSProfile         `json:"qosProfile"`
	Resources   ResourceProfile    `json:"resources"`
	Placement   PlacementProfile   `json:"placement"`
	Confidence  float64            `json:"confidence"`
	Validation  ValidationResult   `json:"validation"`
}

type SliceType string

const (
	SliceTypeEMBB  SliceType = "eMBB"
	SliceTypeURLLC SliceType = "URLLC"
	SliceTypeMmTC  SliceType = "mMTC"
)

type QoSProfile struct {
	Latency        LatencyRequirement     `json:"latency"`
	Throughput     ThroughputRequirement  `json:"throughput"`
	Reliability    ReliabilityRequirement `json:"reliability"`
	Availability   AvailabilityRequirement `json:"availability"`
	PacketLoss     PacketLossRequirement  `json:"packetLoss,omitempty"`
	Jitter         JitterRequirement      `json:"jitter,omitempty"`
}

type LatencyRequirement struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
	Type  string `json:"type"` // end-to-end, one-way, round-trip
}

type ThroughputRequirement struct {
	Downlink string `json:"downlink"`
	Uplink   string `json:"uplink"`
	Unit     string `json:"unit"`
}

type ReliabilityRequirement struct {
	Value string `json:"value"`
	Unit  string `json:"unit"` // percentage, nines
}

type AvailabilityRequirement struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

type PacketLossRequirement struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

type JitterRequirement struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

type ResourceProfile struct {
	Compute  ComputeRequirement `json:"compute"`
	Network  NetworkRequirement `json:"network"`
	Storage  StorageRequirement `json:"storage,omitempty"`
	GPU      GPURequirement     `json:"gpu,omitempty"`
}

type ComputeRequirement struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Cores  int    `json:"cores,omitempty"`
}

type NetworkRequirement struct {
	Bandwidth string `json:"bandwidth"`
	Ports     []int  `json:"ports,omitempty"`
}

type StorageRequirement struct {
	Size string `json:"size"`
	Type string `json:"type"`
	IOPS string `json:"iops,omitempty"`
}

type GPURequirement struct {
	Count int    `json:"count"`
	Type  string `json:"type"`
	Memory string `json:"memory,omitempty"`
}

type PlacementProfile struct {
	Zones         []string                `json:"zones,omitempty"`
	Affinity      map[string]string       `json:"affinity,omitempty"`
	AntiAffinity  map[string]string       `json:"antiAffinity,omitempty"`
	NodeSelector  map[string]string       `json:"nodeSelector,omitempty"`
	Constraints   []PlacementConstraint   `json:"constraints,omitempty"`
}

type PlacementConstraint struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Test fixtures for various intent scenarios
func ValidEMBBIntent() Intent {
	return Intent{
		ID:        "intent-embb-001",
		Text:      "Create an enhanced mobile broadband slice for video streaming with 50ms latency and 1Gbps throughput in edge zone A",
		Type:      IntentTypeSliceCreate,
		Priority:  PriorityHigh,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "network-admin",
			Application: "video-streaming",
			Region:      "edge-zone-a",
		},
		Constraints: map[string]interface{}{
			"max-latency":   "50ms",
			"min-throughput": "1Gbps",
			"availability":   "99.9%",
		},
		Metadata: map[string]string{
			"slice-type": "eMBB",
			"use-case":   "video-streaming",
			"priority":   "high",
		},
	}
}

func ValidURLLCIntent() Intent {
	return Intent{
		ID:        "intent-urllc-001",
		Text:      "Deploy ultra-reliable low latency slice for autonomous driving with 1ms latency and 99.999% reliability",
		Type:      IntentTypeSliceCreate,
		Priority:  PriorityCritical,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "autonomous-systems",
			Application: "autonomous-driving",
			Region:      "city-center",
		},
		Constraints: map[string]interface{}{
			"max-latency":   "1ms",
			"reliability":   "99.999%",
			"packet-loss":   "0.001%",
		},
		Metadata: map[string]string{
			"slice-type": "URLLC",
			"use-case":   "autonomous-driving",
			"priority":   "critical",
		},
	}
}

func ValidMmTCIntent() Intent {
	return Intent{
		ID:        "intent-mmtc-001",
		Text:      "Create massive machine type communication slice for IoT sensors supporting 1 million devices per square kilometer",
		Type:      IntentTypeSliceCreate,
		Priority:  PriorityMedium,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "iot-platform",
			Application: "smart-city",
			Region:      "metropolitan-area",
		},
		Constraints: map[string]interface{}{
			"device-density": "1000000",
			"power-efficiency": "high",
			"coverage":       "wide-area",
		},
		Metadata: map[string]string{
			"slice-type":     "mMTC",
			"use-case":       "smart-city",
			"device-type":    "sensors",
		},
	}
}

func ComplexMultiSliceIntent() Intent {
	return Intent{
		ID:        "intent-multi-001",
		Text:      "Deploy mixed slice supporting both video streaming (eMBB) and industrial automation (URLLC) with dynamic resource allocation",
		Type:      IntentTypeSliceCreate,
		Priority:  PriorityHigh,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "enterprise-admin",
			Application: "industrial-campus",
			Region:      "industrial-zone",
		},
		Constraints: map[string]interface{}{
			"embb-throughput": "2Gbps",
			"urllc-latency":   "5ms",
			"shared-resources": true,
			"isolation":       "guaranteed",
		},
		Metadata: map[string]string{
			"slice-types": "eMBB,URLLC",
			"use-case":    "mixed-enterprise",
		},
	}
}

func QoSOptimizationIntent() Intent {
	return Intent{
		ID:        "intent-qos-001",
		Text:      "Optimize existing slice performance by reducing latency to 10ms and increasing reliability to 99.99%",
		Type:      IntentTypeQoSOptimize,
		Priority:  PriorityHigh,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "network-operator",
			Application: "existing-service",
			Region:      "edge-zone-b",
		},
		Constraints: map[string]interface{}{
			"target-latency":    "10ms",
			"target-reliability": "99.99%",
			"existing-slice-id": "slice-embb-001",
		},
		Metadata: map[string]string{
			"operation": "optimize",
			"target":    "qos-improvement",
		},
	}
}

func ResourceScalingIntent() Intent {
	return Intent{
		ID:        "intent-scale-001",
		Text:      "Scale up compute resources for existing slice to handle 50% more traffic during peak hours",
		Type:      IntentTypeResourceScale,
		Priority:  PriorityMedium,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "operations-team",
			Application: "traffic-management",
			Region:      "metro-core",
			TimeWindow: TimeWindow{
				Start: time.Now().Add(time.Hour),
				End:   time.Now().Add(4 * time.Hour),
			},
		},
		Constraints: map[string]interface{}{
			"scale-factor":      "1.5",
			"resource-type":     "compute",
			"temporary":         true,
			"existing-slice-id": "slice-embb-002",
		},
		Metadata: map[string]string{
			"operation":  "scale",
			"direction":  "up",
			"temporary":  "true",
		},
	}
}

func InvalidIntent() Intent {
	return Intent{
		ID:        "", // Invalid: empty ID
		Text:      "", // Invalid: empty text
		Type:      "", // Invalid: empty type
		Priority:  "", // Invalid: empty priority
		Timestamp: time.Time{}, // Invalid: zero time
		Context:   IntentContext{}, // Invalid: empty context
	}
}

func AmbiguousIntent() Intent {
	return Intent{
		ID:        "intent-ambiguous-001",
		Text:      "Create a slice with good performance for users", // Ambiguous: no specific requirements
		Type:      IntentTypeSliceCreate,
		Priority:  PriorityMedium,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "unknown-user",
			Application: "general",
			Region:      "unspecified",
		},
		Metadata: map[string]string{
			"ambiguous": "true",
		},
	}
}

func ConflictingConstraintsIntent() Intent {
	return Intent{
		ID:        "intent-conflict-001",
		Text:      "Create low latency slice with minimal resource usage and maximum throughput",
		Type:      IntentTypeSliceCreate,
		Priority:  PriorityHigh,
		Timestamp: time.Now(),
		Context: IntentContext{
			User:        "test-user",
			Application: "test-app",
			Region:      "test-region",
		},
		Constraints: map[string]interface{}{
			"max-latency":     "1ms",    // URLLC requirement
			"min-throughput":  "10Gbps", // eMBB requirement
			"max-resources":   "minimal", // Conflicting with above
			"device-density":  "1000000", // mMTC requirement
		},
		Metadata: map[string]string{
			"conflicting": "true",
		},
	}
}

// Expected parsed results for test validation
func ExpectedEMBBParsedIntent() ParsedIntent {
	return ParsedIntent{
		Intent:    ValidEMBBIntent(),
		SliceType: SliceTypeEMBB,
		QoSProfile: QoSProfile{
			Latency: LatencyRequirement{
				Value: "50",
				Unit:  "ms",
				Type:  "end-to-end",
			},
			Throughput: ThroughputRequirement{
				Downlink: "1Gbps",
				Uplink:   "100Mbps",
				Unit:     "bps",
			},
			Reliability: ReliabilityRequirement{
				Value: "99.9",
				Unit:  "percentage",
			},
			Availability: AvailabilityRequirement{
				Value: "99.9",
				Unit:  "percentage",
			},
		},
		Resources: ResourceProfile{
			Compute: ComputeRequirement{
				CPU:    "4000m",
				Memory: "8Gi",
				Cores:  4,
			},
			Network: NetworkRequirement{
				Bandwidth: "1Gbps",
				Ports:     []int{8080, 8443},
			},
		},
		Placement: PlacementProfile{
			Zones: []string{"edge-zone-a"},
			Affinity: map[string]string{
				"node-type": "edge",
			},
		},
		Confidence: 0.95,
		Validation: ValidationResult{
			Valid: true,
		},
	}
}

func ExpectedURLLCParsedIntent() ParsedIntent {
	parsed := ExpectedEMBBParsedIntent()
	parsed.Intent = ValidURLLCIntent()
	parsed.SliceType = SliceTypeURLLC
	parsed.QoSProfile.Latency.Value = "1"
	parsed.QoSProfile.Reliability.Value = "99.999"
	parsed.QoSProfile.PacketLoss = PacketLossRequirement{
		Value: "0.001",
		Unit:  "percentage",
	}
	parsed.Confidence = 0.98
	return parsed
}

func ExpectedMmTCParsedIntent() ParsedIntent {
	parsed := ExpectedEMBBParsedIntent()
	parsed.Intent = ValidMmTCIntent()
	parsed.SliceType = SliceTypeMmTC
	parsed.QoSProfile.Latency.Value = "100"
	parsed.QoSProfile.Throughput.Downlink = "10Mbps"
	parsed.Resources.Compute.CPU = "1000m"
	parsed.Resources.Compute.Memory = "2Gi"
	parsed.Confidence = 0.92
	return parsed
}

func ExpectedInvalidParsedIntent() ParsedIntent {
	return ParsedIntent{
		Intent:     InvalidIntent(),
		Confidence: 0.0,
		Validation: ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Field:   "id",
					Message: "Intent ID cannot be empty",
					Code:    "EMPTY_ID",
				},
				{
					Field:   "text",
					Message: "Intent text cannot be empty",
					Code:    "EMPTY_TEXT",
				},
				{
					Field:   "type",
					Message: "Intent type must be specified",
					Code:    "INVALID_TYPE",
				},
			},
		},
	}
}

// Helper functions to convert to JSON for testing
func ToJSON(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

func FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}