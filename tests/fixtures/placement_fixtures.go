package fixtures

import (
	"time"

	"github.com/O-RAN-Intent-MANO-for-Network-Slicing/tests/mocks"
)

// PlacementRequest represents a request for optimal resource placement
type PlacementRequest struct {
	ID            string                    `json:"id"`
	VNFSpec       *VNFDeployment            `json:"vnfSpec"`
	QoSProfile    QoSProfile                `json:"qosProfile"`
	Constraints   PlacementConstraints      `json:"constraints"`
	Objectives    []OptimizationObjective   `json:"objectives"`
	Timestamp     time.Time                 `json:"timestamp"`
	Priority      Priority                  `json:"priority"`
	Metadata      map[string]string         `json:"metadata,omitempty"`
}

type PlacementConstraints struct {
	HardConstraints []Constraint `json:"hardConstraints"`
	SoftConstraints []Constraint `json:"softConstraints"`
	Resources       ResourceConstraints `json:"resources"`
	Geographic      GeographicConstraints `json:"geographic"`
	Network         NetworkConstraints `json:"network"`
	Compliance      ComplianceConstraints `json:"compliance"`
}

type Constraint struct {
	Type        string      `json:"type"`
	Field       string      `json:"field"`
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value"`
	Weight      float64     `json:"weight,omitempty"`
	Description string      `json:"description,omitempty"`
}

type ResourceConstraints struct {
	MinCPU       string   `json:"minCpu"`
	MaxCPU       string   `json:"maxCpu"`
	MinMemory    string   `json:"minMemory"`
	MaxMemory    string   `json:"maxMemory"`
	RequiredGPU  bool     `json:"requiredGpu"`
	StorageType  string   `json:"storageType"`
	NetworkBW    string   `json:"networkBandwidth"`
	NodeTypes    []string `json:"nodeTypes"`
}

type GeographicConstraints struct {
	Zones             []string  `json:"zones"`
	ExcludedZones     []string  `json:"excludedZones"`
	MaxLatencyToUser  string    `json:"maxLatencyToUser"`
	DataSovereignty   []string  `json:"dataSovereignty"`
	DisasterRecovery  bool      `json:"disasterRecovery"`
	Proximity         ProximityConstraint `json:"proximity"`
}

type ProximityConstraint struct {
	ToServices   []string `json:"toServices"`
	MaxDistance  string   `json:"maxDistance"`
	MinDistance  string   `json:"minDistance"`
}

type NetworkConstraints struct {
	MaxLatency     string   `json:"maxLatency"`
	MinBandwidth   string   `json:"minBandwidth"`
	MaxJitter      string   `json:"maxJitter"`
	MaxPacketLoss  string   `json:"maxPacketLoss"`
	QoSClass       string   `json:"qosClass"`
	Isolation      string   `json:"isolation"`
	ConnectedTo    []string `json:"connectedTo"`
}

type ComplianceConstraints struct {
	Regulations    []string `json:"regulations"`
	SecurityLevel  string   `json:"securityLevel"`
	DataClassification string `json:"dataClassification"`
	Certifications []string `json:"certifications"`
}

type OptimizationObjective struct {
	Type        string  `json:"type"`
	Target      string  `json:"target"`
	Direction   string  `json:"direction"` // minimize, maximize
	Weight      float64 `json:"weight"`
	Threshold   float64 `json:"threshold,omitempty"`
}

// PlacementSolution represents the optimal placement solution
type PlacementSolution struct {
	ID            string                `json:"id"`
	RequestID     string                `json:"requestId"`
	Placements    []ResourcePlacement   `json:"placements"`
	Score         OptimizationScore     `json:"score"`
	Constraints   ConstraintResults     `json:"constraints"`
	Alternatives  []AlternativePlacement `json:"alternatives,omitempty"`
	Timestamp     time.Time             `json:"timestamp"`
	ValidUntil    time.Time             `json:"validUntil"`
	Metadata      map[string]string     `json:"metadata,omitempty"`
}

type ResourcePlacement struct {
	VNFComponent  string            `json:"vnfComponent"`
	NodeID        string            `json:"nodeId"`
	Zone          string            `json:"zone"`
	Region        string            `json:"region"`
	Resources     AllocatedResources `json:"resources"`
	NetworkPaths  []NetworkPath     `json:"networkPaths"`
	Affinity      AffinityRules     `json:"affinity"`
	Score         float64           `json:"score"`
	Reason        string            `json:"reason"`
}

type NetworkPath struct {
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
	Latency     time.Duration `json:"latency"`
	Bandwidth   float64       `json:"bandwidth"`
	Hops        int           `json:"hops"`
	QoS         string        `json:"qos"`
	Path        []string      `json:"path"`
}

type AffinityRules struct {
	ColocatedWith    []string `json:"colocatedWith"`
	SeparatedFrom    []string `json:"separatedFrom"`
	PreferredNodes   []string `json:"preferredNodes"`
	AvoidedNodes     []string `json:"avoidedNodes"`
}

type OptimizationScore struct {
	Total        float64            `json:"total"`
	Components   map[string]float64 `json:"components"`
	Weights      map[string]float64 `json:"weights"`
	Normalized   bool               `json:"normalized"`
}

type ConstraintResults struct {
	Satisfied    []ConstraintResult `json:"satisfied"`
	Violated     []ConstraintResult `json:"violated"`
	Warnings     []ConstraintResult `json:"warnings"`
	Feasible     bool               `json:"feasible"`
}

type ConstraintResult struct {
	Constraint  Constraint  `json:"constraint"`
	Status      string      `json:"status"`
	ActualValue interface{} `json:"actualValue"`
	Message     string      `json:"message"`
}

type AlternativePlacement struct {
	Placements []ResourcePlacement `json:"placements"`
	Score      OptimizationScore   `json:"score"`
	Tradeoffs  []string            `json:"tradeoffs"`
}

// Infrastructure topology for testing
type InfrastructureTopology struct {
	Nodes    []InfraNode    `json:"nodes"`
	Networks []NetworkLink  `json:"networks"`
	Zones    []Zone         `json:"zones"`
	Metrics  TopologyMetrics `json:"metrics"`
}

type InfraNode struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Zone        string                   `json:"zone"`
	Region      string                   `json:"region"`
	Capacity    mocks.ResourceMetrics    `json:"capacity"`
	Available   mocks.ResourceMetrics    `json:"available"`
	Performance mocks.LatencyMetrics     `json:"performance"`
	Status      string                   `json:"status"`
	Labels      map[string]string        `json:"labels"`
	Taints      []Taint                  `json:"taints"`
}

type NetworkLink struct {
	ID          string        `json:"id"`
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
	Bandwidth   float64       `json:"bandwidth"`
	Latency     time.Duration `json:"latency"`
	Utilization float64       `json:"utilization"`
	Status      string        `json:"status"`
	QoS         string        `json:"qos"`
}

type Zone struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Region      string            `json:"region"`
	Coordinates Coordinates       `json:"coordinates"`
	Nodes       []string          `json:"nodes"`
	Capabilities []string         `json:"capabilities"`
	Regulations []string          `json:"regulations"`
	Metadata    map[string]string `json:"metadata"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

type TopologyMetrics struct {
	TotalNodes      int            `json:"totalNodes"`
	AvailableNodes  int            `json:"availableNodes"`
	TotalCapacity   CapacitySummary `json:"totalCapacity"`
	UsedCapacity    CapacitySummary `json:"usedCapacity"`
	AverageLatency  time.Duration  `json:"averageLatency"`
	NetworkHealth   float64        `json:"networkHealth"`
}

type CapacitySummary struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
	GPU     int    `json:"gpu"`
}

// Test fixtures for placement optimization scenarios
func ValidEMBBPlacementRequest() PlacementRequest {
	return PlacementRequest{
		ID:      "placement-embb-001",
		VNFSpec: eMBBVNFDeployment(),
		QoSProfile: QoSProfile{
			Latency: LatencyRequirement{
				Value: "20",
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
		},
		Constraints: PlacementConstraints{
			HardConstraints: []Constraint{
				{
					Type:     "resource",
					Field:    "cpu",
					Operator: ">=",
					Value:    "2000m",
				},
				{
					Type:     "geographic",
					Field:    "zone",
					Operator: "in",
					Value:    []string{"edge-zone-a", "edge-zone-b"},
				},
			},
			SoftConstraints: []Constraint{
				{
					Type:     "performance",
					Field:    "latency",
					Operator: "<=",
					Value:    "15ms",
					Weight:   0.8,
				},
			},
			Resources: ResourceConstraints{
				MinCPU:      "2000m",
				MinMemory:   "4Gi",
				NodeTypes:   []string{"edge", "compute"},
				NetworkBW:   "1Gbps",
			},
			Geographic: GeographicConstraints{
				Zones:            []string{"edge-zone-a", "edge-zone-b"},
				MaxLatencyToUser: "50ms",
			},
			Network: NetworkConstraints{
				MaxLatency:    "20ms",
				MinBandwidth:  "1Gbps",
				MaxJitter:     "5ms",
				MaxPacketLoss: "0.1%",
				QoSClass:      "high",
			},
		},
		Objectives: []OptimizationObjective{
			{
				Type:      "performance",
				Target:    "latency",
				Direction: "minimize",
				Weight:    0.6,
			},
			{
				Type:      "cost",
				Target:    "resource_cost",
				Direction: "minimize",
				Weight:    0.4,
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
		Metadata: map[string]string{
			"slice-type": "eMBB",
			"use-case":   "video-streaming",
		},
	}
}

func ValidURLLCPlacementRequest() PlacementRequest {
	req := ValidEMBBPlacementRequest()
	req.ID = "placement-urllc-001"
	req.VNFSpec = URLLCVNFDeployment()
	req.QoSProfile.Latency.Value = "1"
	req.QoSProfile.Reliability.Value = "99.999"
	req.Constraints.Network.MaxLatency = "1ms"
	req.Constraints.Network.QoSClass = "ultra-low-latency"
	req.Priority = PriorityCritical
	req.Metadata["slice-type"] = "URLLC"
	req.Metadata["use-case"] = "autonomous-driving"

	// URLLC requires stricter constraints
	req.Constraints.HardConstraints = append(req.Constraints.HardConstraints, Constraint{
		Type:     "latency",
		Field:    "end-to-end",
		Operator: "<=",
		Value:    "1ms",
	})

	return req
}

func ValidMmTCPlacementRequest() PlacementRequest {
	req := ValidEMBBPlacementRequest()
	req.ID = "placement-mmtc-001"
	req.VNFSpec = mMTCVNFDeployment()
	req.QoSProfile.Latency.Value = "100"
	req.QoSProfile.Throughput.Downlink = "10Mbps"
	req.Constraints.Resources.MinCPU = "1000m"
	req.Constraints.Resources.MinMemory = "2Gi"
	req.Priority = PriorityMedium
	req.Metadata["slice-type"] = "mMTC"
	req.Metadata["use-case"] = "iot-sensors"

	// mMTC focuses on device density
	req.Constraints.HardConstraints = append(req.Constraints.HardConstraints, Constraint{
		Type:     "capacity",
		Field:    "device_density",
		Operator: ">=",
		Value:    1000000,
	})

	return req
}

func MultiConstraintPlacementRequest() PlacementRequest {
	req := ValidEMBBPlacementRequest()
	req.ID = "placement-multi-001"
	req.Constraints.Geographic.DataSovereignty = []string{"EU"}
	req.Constraints.Geographic.DisasterRecovery = true
	req.Constraints.Compliance = ComplianceConstraints{
		Regulations:        []string{"GDPR", "SOC2"},
		SecurityLevel:      "high",
		DataClassification: "sensitive",
		Certifications:     []string{"ISO27001"},
	}
	req.Objectives = append(req.Objectives, OptimizationObjective{
		Type:      "compliance",
		Target:    "security_score",
		Direction: "maximize",
		Weight:    0.3,
	})
	return req
}

func ConflictingConstraintsPlacementRequest() PlacementRequest {
	req := ValidEMBBPlacementRequest()
	req.ID = "placement-conflict-001"

	// Add conflicting constraints
	req.Constraints.HardConstraints = append(req.Constraints.HardConstraints,
		Constraint{
			Type:     "performance",
			Field:    "latency",
			Operator: "<=",
			Value:    "1ms", // URLLC requirement
		},
		Constraint{
			Type:     "cost",
			Field:    "budget",
			Operator: "<=",
			Value:    100, // Very low budget
		},
		Constraint{
			Type:     "performance",
			Field:    "throughput",
			Operator: ">=",
			Value:    "10Gbps", // High throughput requirement
		},
	)

	return req
}

func InvalidPlacementRequest() PlacementRequest {
	return PlacementRequest{
		ID:      "", // Invalid: empty ID
		VNFSpec: nil, // Invalid: nil VNF spec
		Constraints: PlacementConstraints{
			Resources: ResourceConstraints{
				MinCPU:    "invalid-cpu", // Invalid format
				MinMemory: "invalid-memory", // Invalid format
			},
		},
		Objectives: []OptimizationObjective{}, // Invalid: no objectives
		Priority:   "", // Invalid: empty priority
	}
}

// Expected placement solutions for testing
func ExpectedEMBBPlacementSolution() PlacementSolution {
	return PlacementSolution{
		ID:        "solution-embb-001",
		RequestID: "placement-embb-001",
		Placements: []ResourcePlacement{
			{
				VNFComponent: "cucp",
				NodeID:       "edge-node-1",
				Zone:         "edge-zone-a",
				Region:       "region-1",
				Resources: AllocatedResources{
					Nodes:    []string{"edge-node-1"},
					Pods:     []string{"cucp-pod-1", "cucp-pod-2"},
					Services: []string{"cucp-service"},
				},
				NetworkPaths: []NetworkPath{
					{
						Source:      "edge-node-1",
						Destination: "user-equipment",
						Latency:     15 * time.Millisecond,
						Bandwidth:   1000.0,
						Hops:        2,
						QoS:         "high",
						Path:        []string{"edge-node-1", "edge-gw", "ue"},
					},
				},
				Affinity: AffinityRules{
					PreferredNodes: []string{"edge-node-1", "edge-node-2"},
				},
				Score:  0.95,
				Reason: "Optimal latency and bandwidth match",
			},
		},
		Score: OptimizationScore{
			Total: 0.92,
			Components: map[string]float64{
				"latency":      0.95,
				"throughput":   0.90,
				"cost":         0.85,
				"reliability":  0.98,
			},
			Weights: map[string]float64{
				"latency":      0.4,
				"throughput":   0.3,
				"cost":         0.2,
				"reliability":  0.1,
			},
			Normalized: true,
		},
		Constraints: ConstraintResults{
			Satisfied: []ConstraintResult{
				{
					Constraint: Constraint{
						Type:     "resource",
						Field:    "cpu",
						Operator: ">=",
						Value:    "2000m",
					},
					Status:      "satisfied",
					ActualValue: "4000m",
					Message:     "CPU requirement satisfied",
				},
			},
			Feasible: true,
		},
		Timestamp:  time.Now(),
		ValidUntil: time.Now().Add(time.Hour),
		Metadata: map[string]string{
			"algorithm": "multi-objective-genetic",
			"version":   "1.0.0",
		},
	}
}

func ExpectedURLLCPlacementSolution() PlacementSolution {
	solution := ExpectedEMBBPlacementSolution()
	solution.ID = "solution-urllc-001"
	solution.RequestID = "placement-urllc-001"
	solution.Placements[0].NetworkPaths[0].Latency = 800 * time.Microsecond
	solution.Score.Components["latency"] = 0.99
	solution.Score.Total = 0.96
	return solution
}

func ExpectedInfeasibleSolution() PlacementSolution {
	return PlacementSolution{
		ID:        "solution-infeasible-001",
		RequestID: "placement-conflict-001",
		Placements: []ResourcePlacement{}, // No valid placements
		Score: OptimizationScore{
			Total: 0.0,
		},
		Constraints: ConstraintResults{
			Violated: []ConstraintResult{
				{
					Constraint: Constraint{
						Type:     "performance",
						Field:    "latency",
						Operator: "<=",
						Value:    "1ms",
					},
					Status:      "violated",
					ActualValue: "5ms",
					Message:     "Cannot achieve 1ms latency with given budget constraints",
				},
			},
			Feasible: false,
		},
		Timestamp: time.Now(),
	}
}

// Infrastructure topology fixtures
func CreateTestInfrastructure() InfrastructureTopology {
	return InfrastructureTopology{
		Nodes: []InfraNode{
			{
				ID:       "edge-node-1",
				Name:     "Edge Node 1",
				Type:     "edge",
				Zone:     "edge-zone-a",
				Region:   "region-1",
				Capacity: *mocks.CreateEdgeZoneMetrics("edge-node-1"),
				Available: *mocks.CreateLowResourceUsageMetrics("edge-node-1"),
				Performance: *mocks.CreateLowLatencyMetrics("edge-node-1"),
				Status:   "ready",
				Labels: map[string]string{
					"node-type": "edge",
					"zone":      "edge-zone-a",
				},
			},
			{
				ID:       "cloud-node-1",
				Name:     "Cloud Node 1",
				Type:     "cloud",
				Zone:     "cloud-zone-1",
				Region:   "region-1",
				Capacity: *mocks.CreateCloudZoneMetrics("cloud-node-1"),
				Available: *mocks.CreateDefaultResourceMetrics("cloud-node-1"),
				Performance: *mocks.CreateDefaultLatencyMetrics("cloud-node-1"),
				Status:   "ready",
				Labels: map[string]string{
					"node-type": "cloud",
					"zone":      "cloud-zone-1",
				},
			},
		},
		Networks: []NetworkLink{
			{
				ID:          "link-1",
				Source:      "edge-node-1",
				Destination: "cloud-node-1",
				Bandwidth:   10000.0, // 10 Gbps
				Latency:     5 * time.Millisecond,
				Utilization: 30.0,
				Status:      "up",
				QoS:         "high",
			},
		},
		Zones: []Zone{
			{
				ID:     "edge-zone-a",
				Name:   "Edge Zone A",
				Type:   "edge",
				Region: "region-1",
				Coordinates: Coordinates{
					Latitude:  37.7749,
					Longitude: -122.4194,
				},
				Nodes:        []string{"edge-node-1"},
				Capabilities: []string{"low-latency", "5g"},
				Regulations:  []string{"FCC"},
			},
			{
				ID:     "cloud-zone-1",
				Name:   "Cloud Zone 1",
				Type:   "cloud",
				Region: "region-1",
				Coordinates: Coordinates{
					Latitude:  37.4419,
					Longitude: -122.1430,
				},
				Nodes:        []string{"cloud-node-1"},
				Capabilities: []string{"high-compute", "storage"},
				Regulations:  []string{"SOC2"},
			},
		},
		Metrics: TopologyMetrics{
			TotalNodes:     2,
			AvailableNodes: 2,
			TotalCapacity: CapacitySummary{
				CPU:     "36000m",
				Memory:  "132Gi",
				Storage: "10100Gi",
				GPU:     0,
			},
			UsedCapacity: CapacitySummary{
				CPU:     "6000m",
				Memory:  "24Gi",
				Storage: "600Gi",
				GPU:     0,
			},
			AverageLatency: 7 * time.Millisecond,
			NetworkHealth:  0.95,
		},
	}
}

func CreateCongestedInfrastructure() InfrastructureTopology {
	infra := CreateTestInfrastructure()
	// Mark nodes as heavily utilized
	for i := range infra.Nodes {
		infra.Nodes[i].Available = *mocks.CreateHighResourceUsageMetrics(infra.Nodes[i].ID)
		infra.Nodes[i].Performance = *mocks.CreateHighLatencyMetrics(infra.Nodes[i].ID)
	}
	// Mark networks as congested
	for i := range infra.Networks {
		infra.Networks[i].Utilization = 95.0
		infra.Networks[i].Latency = 50 * time.Millisecond
	}
	infra.Metrics.NetworkHealth = 0.3
	return infra
}

func CreateOptimalInfrastructure() InfrastructureTopology {
	infra := CreateTestInfrastructure()
	// Add more nodes for better distribution
	infra.Nodes = append(infra.Nodes,
		InfraNode{
			ID:       "edge-node-2",
			Name:     "Edge Node 2",
			Type:     "edge",
			Zone:     "edge-zone-b",
			Region:   "region-1",
			Capacity: *mocks.CreateEdgeZoneMetrics("edge-node-2"),
			Available: *mocks.CreateLowResourceUsageMetrics("edge-node-2"),
			Performance: *mocks.CreateLowLatencyMetrics("edge-node-2"),
			Status:   "ready",
		},
	)
	infra.Metrics.TotalNodes = 3
	infra.Metrics.AvailableNodes = 3
	infra.Metrics.NetworkHealth = 0.98
	return infra
}