package placement

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/O-RAN-Intent-MANO-for-Network-Slicing/tests/fixtures"
	"github.com/O-RAN-Intent-MANO-for-Network-Slicing/tests/mocks"
)

// Optimizer interfaces that need to be implemented
type ResourcePlacementOptimizer interface {
	OptimizePlacement(ctx context.Context, request fixtures.PlacementRequest) (*fixtures.PlacementSolution, error)
	ValidateConstraints(request fixtures.PlacementRequest) error
	CalculateScore(placement []fixtures.ResourcePlacement, objectives []fixtures.OptimizationObjective) fixtures.OptimizationScore
	FindAlternatives(ctx context.Context, request fixtures.PlacementRequest, primary *fixtures.PlacementSolution) ([]fixtures.AlternativePlacement, error)
}

type TopologyManager interface {
	GetInfrastructure(ctx context.Context) (*fixtures.InfrastructureTopology, error)
	GetNodeMetrics(ctx context.Context, nodeID string) (*mocks.ResourceMetrics, error)
	GetNetworkLatency(ctx context.Context, source, destination string) (time.Duration, error)
	UpdateTopology(ctx context.Context, updates []TopologyUpdate) error
}

type ConstraintSolver interface {
	SolveConstraints(constraints fixtures.PlacementConstraints, topology *fixtures.InfrastructureTopology) ([]fixtures.ResourcePlacement, error)
	ValidateHardConstraints(placement fixtures.ResourcePlacement, constraints []fixtures.Constraint) error
	EvaluateSoftConstraints(placement fixtures.ResourcePlacement, constraints []fixtures.Constraint) float64
}

type PerformancePredictor interface {
	PredictLatency(ctx context.Context, placement fixtures.ResourcePlacement) (time.Duration, error)
	PredictThroughput(ctx context.Context, placement fixtures.ResourcePlacement) (float64, error)
	PredictResourceUtilization(ctx context.Context, placement fixtures.ResourcePlacement) (*mocks.ResourceMetrics, error)
}

// Supporting types
type TopologyUpdate struct {
	Type   string      `json:"type"`
	NodeID string      `json:"nodeId"`
	Data   interface{} `json:"data"`
}

type OptimizationConfig struct {
	Algorithm      string        `json:"algorithm"`
	MaxIterations  int           `json:"maxIterations"`
	Timeout        time.Duration `json:"timeout"`
	Tolerance      float64       `json:"tolerance"`
	ParallelSearch bool          `json:"parallelSearch"`
}

// PlacementOptimizer - the optimizer we're testing (not implemented yet)
type PlacementOptimizer struct {
	MetricsCollector mocks.MetricsCollector
	TopologyManager  TopologyManager
	ConstraintSolver ConstraintSolver
	Predictor        PerformancePredictor
	Config           OptimizationConfig
}

// NewPlacementOptimizer creates a new placement optimizer (not implemented yet)
func NewPlacementOptimizer(config OptimizationConfig) *PlacementOptimizer {
	// Intentionally not implemented to cause test failure (RED phase)
	return nil
}

// Interface methods that need to be implemented
func (p *PlacementOptimizer) OptimizePlacement(ctx context.Context, request fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *PlacementOptimizer) ValidateConstraints(request fixtures.PlacementRequest) error {
	// Not implemented yet - will cause tests to fail
	return nil
}

func (p *PlacementOptimizer) CalculateScore(placement []fixtures.ResourcePlacement, objectives []fixtures.OptimizationObjective) fixtures.OptimizationScore {
	// Not implemented yet - will cause tests to fail
	return fixtures.OptimizationScore{}
}

func (p *PlacementOptimizer) FindAlternatives(ctx context.Context, request fixtures.PlacementRequest, primary *fixtures.PlacementSolution) ([]fixtures.AlternativePlacement, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *PlacementOptimizer) OptimizeForLatency(ctx context.Context, request fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *PlacementOptimizer) OptimizeForThroughput(ctx context.Context, request fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *PlacementOptimizer) OptimizeForCost(ctx context.Context, request fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *PlacementOptimizer) SolveMultiObjective(ctx context.Context, request fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

// Table-driven tests for resource placement optimization
func TestPlacementOptimizer_OptimizePlacement(t *testing.T) {
	tests := []struct {
		name              string
		request           fixtures.PlacementRequest
		infrastructureSetup func() *fixtures.InfrastructureTopology
		metricsSetup      func(*mocks.MockMetricsCollector)
		expectedSolution  *fixtures.PlacementSolution
		expectedError     bool
		minScore          float64
		validateSolution  func(t *testing.T, solution *fixtures.PlacementSolution)
	}{
		{
			name:    "optimize_embb_video_streaming",
			request: fixtures.ValidEMBBPlacementRequest(),
			infrastructureSetup: func() *fixtures.InfrastructureTopology {
				topology := fixtures.CreateTestInfrastructure()
				return &topology
			},
			metricsSetup: func(mockMetrics *mocks.MockMetricsCollector) {
				mockMetrics.CollectLatencyFunc = func(ctx context.Context, target string) (*mocks.LatencyMetrics, error) {
					return mocks.CreateLowLatencyMetrics(target), nil
				}
				mockMetrics.CollectThroughputFunc = func(ctx context.Context, target string) (*mocks.ThroughputMetrics, error) {
					return mocks.CreateHighThroughputMetrics(target), nil
				}
				mockMetrics.CollectResourceFunc = func(ctx context.Context, target string) (*mocks.ResourceMetrics, error) {
					return mocks.CreateLowResourceUsageMetrics(target), nil
				}
			},
			expectedError: false,
			minScore:      0.8,
			validateSolution: func(t *testing.T, solution *fixtures.PlacementSolution) {
				assert.NotEmpty(t, solution.Placements)
				assert.True(t, solution.Constraints.Feasible)
				assert.GreaterOrEqual(t, solution.Score.Total, 0.8)

				// Verify eMBB-specific requirements
				for _, placement := range solution.Placements {
					assert.NotEmpty(t, placement.NetworkPaths)
					for _, path := range placement.NetworkPaths {
						assert.LessOrEqual(t, path.Latency, 20*time.Millisecond)
						assert.GreaterOrEqual(t, path.Bandwidth, 1000.0) // 1 Gbps
					}
				}
			},
		},
		{
			name:    "optimize_urllc_autonomous_driving",
			request: fixtures.ValidURLLCPlacementRequest(),
			infrastructureSetup: func() *fixtures.InfrastructureTopology {
				topology := fixtures.CreateOptimalInfrastructure()
				return &topology
			},
			metricsSetup: func(mockMetrics *mocks.MockMetricsCollector) {
				mockMetrics.CollectLatencyFunc = func(ctx context.Context, target string) (*mocks.LatencyMetrics, error) {
					return mocks.CreateLowLatencyMetrics(target), nil
				}
				mockMetrics.CollectResourceFunc = func(ctx context.Context, target string) (*mocks.ResourceMetrics, error) {
					return mocks.CreateLowResourceUsageMetrics(target), nil
				}
			},
			expectedError: false,
			minScore:      0.85,
			validateSolution: func(t *testing.T, solution *fixtures.PlacementSolution) {
				assert.NotEmpty(t, solution.Placements)
				assert.True(t, solution.Constraints.Feasible)
				assert.GreaterOrEqual(t, solution.Score.Total, 0.85)

				// Verify URLLC ultra-low latency requirements
				for _, placement := range solution.Placements {
					for _, path := range placement.NetworkPaths {
						assert.LessOrEqual(t, path.Latency, 1*time.Millisecond)
					}
				}
			},
		},
		{
			name:    "optimize_mmtc_iot_sensors",
			request: fixtures.ValidMmTCPlacementRequest(),
			infrastructureSetup: func() *fixtures.InfrastructureTopology {
				topology := fixtures.CreateTestInfrastructure()
				return &topology
			},
			metricsSetup: func(mockMetrics *mocks.MockMetricsCollector) {
				mockMetrics.CollectResourceFunc = func(ctx context.Context, target string) (*mocks.ResourceMetrics, error) {
					return mocks.CreateLowResourceUsageMetrics(target), nil
				}
			},
			expectedError: false,
			minScore:      0.75,
			validateSolution: func(t *testing.T, solution *fixtures.PlacementSolution) {
				assert.NotEmpty(t, solution.Placements)
				assert.True(t, solution.Constraints.Feasible)

				// Verify mMTC resource efficiency
				for _, placement := range solution.Placements {
					assert.Contains(t, placement.NodeID, "edge") // Prefer edge for mMTC
				}
			},
		},
		{
			name:    "multi_constraint_optimization",
			request: fixtures.MultiConstraintPlacementRequest(),
			infrastructureSetup: func() *fixtures.InfrastructureTopology {
				topology := fixtures.CreateOptimalInfrastructure()
				return &topology
			},
			metricsSetup: func(mockMetrics *mocks.MockMetricsCollector) {
				mockMetrics.CollectLatencyFunc = func(ctx context.Context, target string) (*mocks.LatencyMetrics, error) {
					return mocks.CreateDefaultLatencyMetrics(target), nil
				}
			},
			expectedError: false,
			minScore:      0.7,
			validateSolution: func(t *testing.T, solution *fixtures.PlacementSolution) {
				assert.NotEmpty(t, solution.Placements)
				assert.True(t, solution.Constraints.Feasible)

				// Verify compliance constraints are satisfied
				assert.NotEmpty(t, solution.Constraints.Satisfied)
			},
		},
		{
			name:    "conflicting_constraints_scenario",
			request: fixtures.ConflictingConstraintsPlacementRequest(),
			infrastructureSetup: func() *fixtures.InfrastructureTopology {
				topology := fixtures.CreateTestInfrastructure()
				return &topology
			},
			metricsSetup: func(mockMetrics *mocks.MockMetricsCollector) {
				mockMetrics.CollectLatencyFunc = func(ctx context.Context, target string) (*mocks.LatencyMetrics, error) {
					return mocks.CreateHighLatencyMetrics(target), nil
				}
			},
			expectedError: false, // Should find best compromise solution
			minScore:      0.3,   // Low score due to conflicting constraints
			validateSolution: func(t *testing.T, solution *fixtures.PlacementSolution) {
				// Should find a compromise solution
				assert.False(t, solution.Constraints.Feasible) // Not all constraints can be satisfied
				assert.NotEmpty(t, solution.Constraints.Violated)
				assert.NotEmpty(t, solution.Alternatives) // Should provide alternatives
			},
		},
		{
			name:    "resource_constrained_infrastructure",
			request: fixtures.ValidEMBBPlacementRequest(),
			infrastructureSetup: func() *fixtures.InfrastructureTopology {
				topology := fixtures.CreateCongestedInfrastructure()
				return &topology
			},
			metricsSetup: func(mockMetrics *mocks.MockMetricsCollector) {
				mockMetrics.CollectResourceFunc = func(ctx context.Context, target string) (*mocks.ResourceMetrics, error) {
					return mocks.CreateHighResourceUsageMetrics(target), nil
				}
				mockMetrics.CollectLatencyFunc = func(ctx context.Context, target string) (*mocks.LatencyMetrics, error) {
					return mocks.CreateHighLatencyMetrics(target), nil
				}
			},
			expectedError: false,
			minScore:      0.4, // Lower score due to resource constraints
			validateSolution: func(t *testing.T, solution *fixtures.PlacementSolution) {
				// Should still find a solution but with compromises
				assert.NotEmpty(t, solution.Placements)
				assert.LessOrEqual(t, solution.Score.Total, 0.6)
			},
		},
		{
			name:    "invalid_placement_request",
			request: fixtures.InvalidPlacementRequest(),
			infrastructureSetup: func() *fixtures.InfrastructureTopology {
				topology := fixtures.CreateTestInfrastructure()
				return &topology
			},
			metricsSetup: func(mockMetrics *mocks.MockMetricsCollector) {
				// No setup needed for invalid request
			},
			expectedError: true,
			validateSolution: func(t *testing.T, solution *fixtures.PlacementSolution) {
				assert.Nil(t, solution)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockMetrics := &mocks.MockMetricsCollector{}
			if tt.metricsSetup != nil {
				tt.metricsSetup(mockMetrics)
			}

			// Create optimizer
			optimizer := &PlacementOptimizer{
				MetricsCollector: mockMetrics,
				Config: OptimizationConfig{
					Algorithm:     "multi-objective-genetic",
					MaxIterations: 100,
					Timeout:       time.Minute,
					Tolerance:     0.01,
				},
			}

			// Execute test
			result, err := optimizer.OptimizePlacement(context.Background(), tt.request)

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				if tt.minScore > 0 {
					assert.GreaterOrEqual(t, result.Score.Total, tt.minScore)
				}
			}

			// Run custom validations
			if tt.validateSolution != nil {
				tt.validateSolution(t, result)
			}
		})
	}
}

// Test multi-constraint satisfaction
func TestPlacementOptimizer_MultiConstraintSatisfaction(t *testing.T) {
	tests := []struct {
		name              string
		constraints       fixtures.PlacementConstraints
		expectedFeasible  bool
		expectedViolations int
	}{
		{
			name: "satisfiable_hard_constraints",
			constraints: fixtures.PlacementConstraints{
				HardConstraints: []fixtures.Constraint{
					{
						Type:     "resource",
						Field:    "cpu",
						Operator: ">=",
						Value:    "2000m",
					},
					{
						Type:     "latency",
						Field:    "end-to-end",
						Operator: "<=",
						Value:    "20ms",
					},
				},
			},
			expectedFeasible:   true,
			expectedViolations: 0,
		},
		{
			name: "unsatisfiable_hard_constraints",
			constraints: fixtures.PlacementConstraints{
				HardConstraints: []fixtures.Constraint{
					{
						Type:     "latency",
						Field:    "end-to-end",
						Operator: "<=",
						Value:    "0.1ms", // Impossible latency
					},
					{
						Type:     "cost",
						Field:    "budget",
						Operator: "<=",
						Value:    1, // Impossible budget
					},
				},
			},
			expectedFeasible:   false,
			expectedViolations: 2,
		},
		{
			name: "mixed_constraint_types",
			constraints: fixtures.PlacementConstraints{
				HardConstraints: []fixtures.Constraint{
					{
						Type:     "geographic",
						Field:    "zone",
						Operator: "in",
						Value:    []string{"edge-zone-a"},
					},
				},
				SoftConstraints: []fixtures.Constraint{
					{
						Type:     "performance",
						Field:    "latency",
						Operator: "<=",
						Value:    "10ms",
						Weight:   0.8,
					},
				},
			},
			expectedFeasible:   true,
			expectedViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizer := &PlacementOptimizer{}

			request := fixtures.PlacementRequest{
				ID:          "test-request",
				VNFSpec:     fixtures.ValidVNFDeployment(),
				Constraints: tt.constraints,
			}

			err := optimizer.ValidateConstraints(request)

			if tt.expectedFeasible {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Test performance optimization goals
func TestPlacementOptimizer_PerformanceOptimization(t *testing.T) {
	tests := []struct {
		name           string
		optimizationFunc func(*PlacementOptimizer, context.Context, fixtures.PlacementRequest) (*fixtures.PlacementSolution, error)
		request        fixtures.PlacementRequest
		validateResult func(t *testing.T, solution *fixtures.PlacementSolution)
	}{
		{
			name: "optimize_for_latency",
			optimizationFunc: func(p *PlacementOptimizer, ctx context.Context, req fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
				return p.OptimizeForLatency(ctx, req)
			},
			request: fixtures.ValidURLLCPlacementRequest(),
			validateResult: func(t *testing.T, solution *fixtures.PlacementSolution) {
				// Should prioritize low-latency placements
				assert.NotNil(t, solution)
				// Verify latency optimization in score components
				assert.GreaterOrEqual(t, solution.Score.Components["latency"], 0.9)
			},
		},
		{
			name: "optimize_for_throughput",
			optimizationFunc: func(p *PlacementOptimizer, ctx context.Context, req fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
				return p.OptimizeForThroughput(ctx, req)
			},
			request: fixtures.ValidEMBBPlacementRequest(),
			validateResult: func(t *testing.T, solution *fixtures.PlacementSolution) {
				// Should prioritize high-throughput placements
				assert.NotNil(t, solution)
				assert.GreaterOrEqual(t, solution.Score.Components["throughput"], 0.9)
			},
		},
		{
			name: "optimize_for_cost",
			optimizationFunc: func(p *PlacementOptimizer, ctx context.Context, req fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
				return p.OptimizeForCost(ctx, req)
			},
			request: fixtures.ValidMmTCPlacementRequest(),
			validateResult: func(t *testing.T, solution *fixtures.PlacementSolution) {
				// Should prioritize cost-effective placements
				assert.NotNil(t, solution)
				assert.GreaterOrEqual(t, solution.Score.Components["cost"], 0.8)
			},
		},
		{
			name: "multi_objective_optimization",
			optimizationFunc: func(p *PlacementOptimizer, ctx context.Context, req fixtures.PlacementRequest) (*fixtures.PlacementSolution, error) {
				return p.SolveMultiObjective(ctx, req)
			},
			request: fixtures.MultiConstraintPlacementRequest(),
			validateResult: func(t *testing.T, solution *fixtures.PlacementSolution) {
				// Should balance multiple objectives
				assert.NotNil(t, solution)
				assert.GreaterOrEqual(t, solution.Score.Total, 0.7)
				// Verify all objectives are considered
				assert.NotEmpty(t, solution.Score.Components)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizer := &PlacementOptimizer{}

			result, err := tt.optimizationFunc(optimizer, context.Background(), tt.request)

			assert.NoError(t, err)
			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

// Test score calculation
func TestPlacementOptimizer_CalculateScore(t *testing.T) {
	tests := []struct {
		name        string
		placements  []fixtures.ResourcePlacement
		objectives  []fixtures.OptimizationObjective
		expectedMin float64
		expectedMax float64
	}{
		{
			name: "balanced_objectives",
			placements: []fixtures.ResourcePlacement{
				{
					VNFComponent: "cucp",
					NodeID:       "edge-node-1",
					Score:        0.9,
				},
			},
			objectives: []fixtures.OptimizationObjective{
				{
					Type:      "performance",
					Target:    "latency",
					Direction: "minimize",
					Weight:    0.5,
				},
				{
					Type:      "cost",
					Target:    "resource_cost",
					Direction: "minimize",
					Weight:    0.5,
				},
			},
			expectedMin: 0.7,
			expectedMax: 1.0,
		},
		{
			name: "latency_priority",
			placements: []fixtures.ResourcePlacement{
				{
					VNFComponent: "cucp",
					NodeID:       "edge-node-1",
					Score:        0.95,
				},
			},
			objectives: []fixtures.OptimizationObjective{
				{
					Type:      "performance",
					Target:    "latency",
					Direction: "minimize",
					Weight:    0.9,
				},
				{
					Type:      "cost",
					Target:    "resource_cost",
					Direction: "minimize",
					Weight:    0.1,
				},
			},
			expectedMin: 0.8,
			expectedMax: 1.0,
		},
		{
			name:        "no_objectives",
			placements:  []fixtures.ResourcePlacement{},
			objectives:  []fixtures.OptimizationObjective{},
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizer := &PlacementOptimizer{}

			result := optimizer.CalculateScore(tt.placements, tt.objectives)

			assert.GreaterOrEqual(t, result.Total, tt.expectedMin)
			assert.LessOrEqual(t, result.Total, tt.expectedMax)
			assert.Equal(t, len(tt.objectives), len(result.Weights))
		})
	}
}

// Test alternative solution finding
func TestPlacementOptimizer_FindAlternatives(t *testing.T) {
	tests := []struct {
		name                string
		request             fixtures.PlacementRequest
		primarySolution     *fixtures.PlacementSolution
		expectedAlternatives int
		validateAlternatives func(t *testing.T, alternatives []fixtures.AlternativePlacement)
	}{
		{
			name:            "embb_alternatives",
			request:         fixtures.ValidEMBBPlacementRequest(),
			primarySolution: func() *fixtures.PlacementSolution { s := fixtures.ExpectedEMBBPlacementSolution(); return &s }(),
			expectedAlternatives: 2,
			validateAlternatives: func(t *testing.T, alternatives []fixtures.AlternativePlacement) {
				assert.Len(t, alternatives, 2)
				for _, alt := range alternatives {
					assert.NotEmpty(t, alt.Placements)
					assert.NotEmpty(t, alt.Tradeoffs)
				}
			},
		},
		{
			name:            "urllc_alternatives",
			request:         fixtures.ValidURLLCPlacementRequest(),
			primarySolution: func() *fixtures.PlacementSolution { s := fixtures.ExpectedURLLCPlacementSolution(); return &s }(),
			expectedAlternatives: 1,
			validateAlternatives: func(t *testing.T, alternatives []fixtures.AlternativePlacement) {
				assert.Len(t, alternatives, 1)
				// URLLC has fewer alternatives due to strict constraints
			},
		},
		{
			name:            "infeasible_primary",
			request:         fixtures.ConflictingConstraintsPlacementRequest(),
			primarySolution: func() *fixtures.PlacementSolution { s := fixtures.ExpectedInfeasibleSolution(); return &s }(),
			expectedAlternatives: 3,
			validateAlternatives: func(t *testing.T, alternatives []fixtures.AlternativePlacement) {
				// Should provide alternatives even when primary is infeasible
				assert.Len(t, alternatives, 3)
				for _, alt := range alternatives {
					assert.GreaterOrEqual(t, alt.Score.Total, 0.3)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizer := &PlacementOptimizer{}

			result, err := optimizer.FindAlternatives(context.Background(), tt.request, tt.primarySolution)

			assert.NoError(t, err)
			if tt.validateAlternatives != nil {
				tt.validateAlternatives(t, result)
			}
		})
	}
}

// Test edge cases and error conditions
func TestPlacementOptimizer_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		testFunc      func(*PlacementOptimizer) error
		expectedError bool
		errorContains string
	}{
		{
			name: "context_timeout",
			testFunc: func(p *PlacementOptimizer) error {
				ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
				defer cancel()
				time.Sleep(time.Millisecond) // Ensure timeout
				_, err := p.OptimizePlacement(ctx, fixtures.ValidEMBBPlacementRequest())
				return err
			},
			expectedError: true,
			errorContains: "context",
		},
		{
			name: "empty_infrastructure",
			testFunc: func(p *PlacementOptimizer) error {
				request := fixtures.ValidEMBBPlacementRequest()
				_, err := p.OptimizePlacement(context.Background(), request)
				return err
			},
			expectedError: true,
			errorContains: "no available nodes",
		},
		{
			name: "nil_vnf_spec",
			testFunc: func(p *PlacementOptimizer) error {
				request := fixtures.ValidEMBBPlacementRequest()
				request.VNFSpec = nil
				_, err := p.OptimizePlacement(context.Background(), request)
				return err
			},
			expectedError: true,
			errorContains: "vnf spec cannot be nil",
		},
		{
			name: "invalid_resource_format",
			testFunc: func(p *PlacementOptimizer) error {
				request := fixtures.ValidEMBBPlacementRequest()
				request.Constraints.Resources.MinCPU = "invalid-cpu"
				_, err := p.OptimizePlacement(context.Background(), request)
				return err
			},
			expectedError: true,
			errorContains: "invalid resource format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizer := &PlacementOptimizer{
				Config: OptimizationConfig{
					Timeout: time.Second,
				},
			}

			err := tt.testFunc(optimizer)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkPlacementOptimizer_OptimizePlacement(b *testing.B) {
	optimizer := &PlacementOptimizer{
		Config: OptimizationConfig{
			Algorithm:     "genetic",
			MaxIterations: 50,
			Timeout:       time.Second * 10,
		},
	}

	request := fixtures.ValidEMBBPlacementRequest()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.OptimizePlacement(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPlacementOptimizer_MultiObjective(b *testing.B) {
	optimizer := &PlacementOptimizer{
		Config: OptimizationConfig{
			Algorithm:     "multi-objective-genetic",
			MaxIterations: 100,
		},
	}

	request := fixtures.MultiConstraintPlacementRequest()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.SolveMultiObjective(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
	}
}