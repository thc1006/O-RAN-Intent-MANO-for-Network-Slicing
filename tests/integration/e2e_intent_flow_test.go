package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/O-RAN-Intent-MANO-for-Network-Slicing/tests/fixtures"
	"github.com/O-RAN-Intent-MANO-for-Network-Slicing/tests/mocks"
)

// E2EIntentFlowTestSuite tests the complete flow from intent to deployment
type E2EIntentFlowTestSuite struct {
	suite.Suite

	// System under test components (not implemented yet)
	IntentProcessor   IntentProcessorInterface
	PlacementEngine   PlacementEngineInterface
	VNFOrchestrator   VNFOrchestratorInterface
	DMSClient         DMSClientInterface
	GitOpsClient      GitOpsClientInterface
	K8sClient         client.Client

	// Mock dependencies
	MockMetrics       *mocks.MockMetricsCollector
	MockHTTP          *mocks.MockHTTPClient
	MockPorch         *mocks.MockPorchClient

	// Test infrastructure
	TestInfrastructure *fixtures.InfrastructureTopology
	TestTimeout        time.Duration
}

// Interfaces for system components (not implemented yet)
type IntentProcessorInterface interface {
	ProcessIntent(ctx context.Context, intent fixtures.Intent) (*fixtures.ParsedIntent, error)
	ValidateIntent(intent fixtures.Intent) error
}

type PlacementEngineInterface interface {
	FindOptimalPlacement(ctx context.Context, request fixtures.PlacementRequest) (*fixtures.PlacementSolution, error)
	ValidatePlacement(solution *fixtures.PlacementSolution) error
}

type VNFOrchestratorInterface interface {
	DeployVNF(ctx context.Context, vnfSpec *fixtures.VNFDeployment, placement *fixtures.PlacementSolution) (*VNFDeploymentResult, error)
	ScaleVNF(ctx context.Context, vnfID string, scaleFactor float64) error
	DeleteVNF(ctx context.Context, vnfID string) error
}

type DMSClientInterface interface {
	RegisterVNF(ctx context.Context, vnfSpec interface{}) error
	GetVNFStatus(ctx context.Context, vnfID string) (string, error)
	ConfigureVNF(ctx context.Context, vnfID string, config map[string]interface{}) error
}

type GitOpsClientInterface interface {
	CreatePackage(ctx context.Context, pkg interface{}) error
	GetPackageStatus(ctx context.Context, packageName string) (string, error)
	SyncPackage(ctx context.Context, packageName string) error
}

// Supporting types
type VNFDeploymentResult struct {
	VNFDeploymentID string                 `json:"vnfDeploymentId"`
	Status          string                 `json:"status"`
	Resources       fixtures.AllocatedResources `json:"resources"`
	NetworkPaths    []fixtures.NetworkPath      `json:"networkPaths"`
	Metrics         *DeploymentMetrics          `json:"metrics,omitempty"`
	Timestamp       time.Time                   `json:"timestamp"`
}

type DeploymentMetrics struct {
	LatencyP99    time.Duration `json:"latencyP99"`
	Throughput    float64       `json:"throughput"`
	Availability  float64       `json:"availability"`
	ResourceUsage ResourceUsage `json:"resourceUsage"`
}

type ResourceUsage struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Network float64 `json:"network"`
}

// Test setup and teardown
func (suite *E2EIntentFlowTestSuite) SetupSuite() {
	suite.TestTimeout = time.Minute * 5
	suite.TestInfrastructure = func() *fixtures.InfrastructureTopology {
		infra := fixtures.CreateTestInfrastructure()
		return &infra
	}()

	// Setup mocks
	suite.MockMetrics = &mocks.MockMetricsCollector{}
	suite.MockHTTP = &mocks.MockHTTPClient{}
	suite.MockPorch = &mocks.MockPorchClient{}
	suite.K8sClient = &mocks.MockK8sClient{}

	// Setup default mock behaviors
	suite.setupDefaultMockBehaviors()

	// Initialize system components (these don't exist yet - will cause tests to fail)
	suite.initializeSystemComponents()
}

func (suite *E2EIntentFlowTestSuite) TearDownSuite() {
	// Cleanup test resources
	suite.cleanupTestResources()
}

func (suite *E2EIntentFlowTestSuite) SetupTest() {
	// Reset mock call tracking before each test
	suite.MockMetrics = &mocks.MockMetricsCollector{}
	suite.MockHTTP = &mocks.MockHTTPClient{}
	suite.MockPorch = &mocks.MockPorchClient{}
}

// Test complete eMBB slice creation flow
func (suite *E2EIntentFlowTestSuite) TestEMBBSliceCreationFlow() {
	ctx, cancel := context.WithTimeout(context.Background(), suite.TestTimeout)
	defer cancel()

	// Step 1: Process natural language intent
	intent := fixtures.ValidEMBBIntent()

	parsedIntent, err := suite.IntentProcessor.ProcessIntent(ctx, intent)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsedIntent)

	// Validate parsed intent
	assert.Equal(suite.T(), fixtures.SliceTypeEMBB, parsedIntent.SliceType)
	assert.Equal(suite.T(), "50", parsedIntent.QoSProfile.Latency.Value)
	assert.Equal(suite.T(), "1Gbps", parsedIntent.QoSProfile.Throughput.Downlink)
	assert.True(suite.T(), parsedIntent.Validation.Valid)

	// Step 2: Find optimal placement
	placementRequest := fixtures.PlacementRequest{
		ID:         "e2e-embb-placement",
		VNFSpec:    fixtures.eMBBVNFDeployment(),
		QoSProfile: parsedIntent.QoSProfile,
		Constraints: fixtures.PlacementConstraints{
			Resources: fixtures.ResourceConstraints{
				MinCPU:    "2000m",
				MinMemory: "4Gi",
			},
			Geographic: fixtures.GeographicConstraints{
				Zones: []string{"edge-zone-a"},
			},
		},
		Objectives: []fixtures.OptimizationObjective{
			{Type: "performance", Target: "latency", Direction: "minimize", Weight: 0.6},
			{Type: "cost", Target: "resource_cost", Direction: "minimize", Weight: 0.4},
		},
		Priority: fixtures.PriorityHigh,
	}

	placement, err := suite.PlacementEngine.FindOptimalPlacement(ctx, placementRequest)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), placement)

	// Validate placement solution
	assert.True(suite.T(), placement.Constraints.Feasible)
	assert.NotEmpty(suite.T(), placement.Placements)
	assert.GreaterOrEqual(suite.T(), placement.Score.Total, 0.8)

	// Step 3: Deploy VNF
	vnfSpec := fixtures.eMBBVNFDeployment()
	deploymentResult, err := suite.VNFOrchestrator.DeployVNF(ctx, vnfSpec, placement)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), deploymentResult)

	// Validate deployment
	assert.NotEmpty(suite.T(), deploymentResult.VNFDeploymentID)
	assert.Equal(suite.T(), "deployed", deploymentResult.Status)
	assert.NotEmpty(suite.T(), deploymentResult.Resources.Nodes)

	// Step 4: Register with DMS (O2 interface)
	err = suite.DMSClient.RegisterVNF(ctx, vnfSpec)
	require.NoError(suite.T(), err)

	// Step 5: Create GitOps package (Nephio integration)
	nephioPackage := fixtures.ValidCUCPPackage()
	err = suite.GitOpsClient.CreatePackage(ctx, nephioPackage)
	require.NoError(suite.T(), err)

	// Step 6: Validate end-to-end metrics
	suite.validateE2EMetrics(ctx, deploymentResult, parsedIntent.QoSProfile)

	// Step 7: Verify system state consistency
	suite.verifySystemStateConsistency(ctx, deploymentResult.VNFDeploymentID)
}

// Test complete URLLC slice creation flow
func (suite *E2EIntentFlowTestSuite) TestURLLCSliceCreationFlow() {
	ctx, cancel := context.WithTimeout(context.Background(), suite.TestTimeout)
	defer cancel()

	// Step 1: Process URLLC intent
	intent := fixtures.ValidURLLCIntent()

	parsedIntent, err := suite.IntentProcessor.ProcessIntent(ctx, intent)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsedIntent)

	// Validate URLLC-specific requirements
	assert.Equal(suite.T(), fixtures.SliceTypeURLLC, parsedIntent.SliceType)
	assert.Equal(suite.T(), "1", parsedIntent.QoSProfile.Latency.Value)
	assert.Equal(suite.T(), "99.999", parsedIntent.QoSProfile.Reliability.Value)

	// Step 2: Find ultra-low latency placement
	placementRequest := fixtures.ValidURLLCPlacementRequest()
	placement, err := suite.PlacementEngine.FindOptimalPlacement(ctx, placementRequest)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), placement)

	// Validate URLLC placement constraints
	assert.True(suite.T(), placement.Constraints.Feasible)
	for _, p := range placement.Placements {
		for _, path := range p.NetworkPaths {
			assert.LessOrEqual(suite.T(), path.Latency, 1*time.Millisecond)
		}
	}

	// Step 3: Deploy URLLC VNF with strict QoS
	vnfSpec := fixtures.URLLCVNFDeployment()
	deploymentResult, err := suite.VNFOrchestrator.DeployVNF(ctx, vnfSpec, placement)
	require.NoError(suite.T(), err)

	// Validate URLLC deployment metrics
	assert.Equal(suite.T(), "deployed", deploymentResult.Status)
	if deploymentResult.Metrics != nil {
		assert.LessOrEqual(suite.T(), deploymentResult.Metrics.LatencyP99, 1*time.Millisecond)
		assert.GreaterOrEqual(suite.T(), deploymentResult.Metrics.Availability, 99.999)
	}

	// Step 4: Validate critical application requirements
	suite.validateCriticalApplicationRequirements(ctx, deploymentResult)
}

// Test complete mMTC slice creation flow
func (suite *E2EIntentFlowTestSuite) TestMmTCSliceCreationFlow() {
	ctx, cancel := context.WithTimeout(context.Background(), suite.TestTimeout)
	defer cancel()

	// Step 1: Process mMTC intent
	intent := fixtures.ValidMmTCIntent()

	parsedIntent, err := suite.IntentProcessor.ProcessIntent(ctx, intent)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsedIntent)

	// Validate mMTC-specific requirements
	assert.Equal(suite.T(), fixtures.SliceTypeMmTC, parsedIntent.SliceType)
	assert.Contains(suite.T(), intent.Constraints, "device-density")

	// Step 2: Find massive connectivity placement
	placementRequest := fixtures.ValidMmTCPlacementRequest()
	placement, err := suite.PlacementEngine.FindOptimalPlacement(ctx, placementRequest)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), placement)

	// Step 3: Deploy mMTC VNF optimized for device density
	vnfSpec := fixtures.mMTCVNFDeployment()
	deploymentResult, err := suite.VNFOrchestrator.DeployVNF(ctx, vnfSpec, placement)
	require.NoError(suite.T(), err)

	// Validate mMTC deployment characteristics
	assert.Equal(suite.T(), "deployed", deploymentResult.Status)

	// Step 4: Validate massive device connectivity
	suite.validateMassiveConnectivity(ctx, deploymentResult)
}

// Test concurrent slice requests
func (suite *E2EIntentFlowTestSuite) TestConcurrentSliceRequests() {
	ctx, cancel := context.WithTimeout(context.Background(), suite.TestTimeout)
	defer cancel()

	// Prepare multiple slice requests
	requests := []struct {
		name   string
		intent fixtures.Intent
	}{
		{"embb-video", fixtures.ValidEMBBIntent()},
		{"urllc-auto", fixtures.ValidURLLCIntent()},
		{"mmtc-iot", fixtures.ValidMmTCIntent()},
	}

	// Process requests concurrently
	results := make(chan struct {
		name   string
		result *VNFDeploymentResult
		err    error
	}, len(requests))

	for _, req := range requests {
		go func(name string, intent fixtures.Intent) {
			defer func() {
				if r := recover(); r != nil {
					results <- struct {
						name   string
						result *VNFDeploymentResult
						err    error
					}{name, nil, fmt.Errorf("panic: %v", r)}
				}
			}()

			// Process intent
			parsedIntent, err := suite.IntentProcessor.ProcessIntent(ctx, intent)
			if err != nil {
				results <- struct {
					name   string
					result *VNFDeploymentResult
					err    error
				}{name, nil, err}
				return
			}

			// Create placement request based on slice type
			var placementRequest fixtures.PlacementRequest
			switch parsedIntent.SliceType {
			case fixtures.SliceTypeEMBB:
				placementRequest = fixtures.ValidEMBBPlacementRequest()
			case fixtures.SliceTypeURLLC:
				placementRequest = fixtures.ValidURLLCPlacementRequest()
			case fixtures.SliceTypeMmTC:
				placementRequest = fixtures.ValidMmTCPlacementRequest()
			}
			placementRequest.ID = fmt.Sprintf("concurrent-%s", name)

			// Find placement
			placement, err := suite.PlacementEngine.FindOptimalPlacement(ctx, placementRequest)
			if err != nil {
				results <- struct {
					name   string
					result *VNFDeploymentResult
					err    error
				}{name, nil, err}
				return
			}

			// Deploy VNF
			var vnfSpec *fixtures.VNFDeployment
			switch parsedIntent.SliceType {
			case fixtures.SliceTypeEMBB:
				vnfSpec = fixtures.eMBBVNFDeployment()
			case fixtures.SliceTypeURLLC:
				vnfSpec = fixtures.URLLCVNFDeployment()
			case fixtures.SliceTypeMmTC:
				vnfSpec = fixtures.mMTCVNFDeployment()
			}

			result, err := suite.VNFOrchestrator.DeployVNF(ctx, vnfSpec, placement)
			results <- struct {
				name   string
				result *VNFDeploymentResult
				err    error
			}{name, result, err}
		}(req.name, req.intent)
	}

	// Collect and validate results
	successful := 0
	for i := 0; i < len(requests); i++ {
		select {
		case result := <-results:
			if result.err != nil {
				suite.T().Logf("Request %s failed: %v", result.name, result.err)
			} else {
				assert.NotNil(suite.T(), result.result)
				assert.Equal(suite.T(), "deployed", result.result.Status)
				successful++
			}
		case <-ctx.Done():
			suite.T().Fatal("Timeout waiting for concurrent requests")
		}
	}

	// Validate that at least some requests succeeded
	assert.GreaterOrEqual(suite.T(), successful, len(requests)/2, "At least half of concurrent requests should succeed")

	// Validate no resource conflicts
	suite.validateNoResourceConflicts(ctx)
}

// Test slice lifecycle management
func (suite *E2EIntentFlowTestSuite) TestSliceLifecycleManagement() {
	ctx, cancel := context.WithTimeout(context.Background(), suite.TestTimeout)
	defer cancel()

	// Step 1: Create slice
	intent := fixtures.ValidEMBBIntent()
	parsedIntent, err := suite.IntentProcessor.ProcessIntent(ctx, intent)
	require.NoError(suite.T(), err)

	placementRequest := fixtures.ValidEMBBPlacementRequest()
	placement, err := suite.PlacementEngine.FindOptimalPlacement(ctx, placementRequest)
	require.NoError(suite.T(), err)

	vnfSpec := fixtures.eMBBVNFDeployment()
	deploymentResult, err := suite.VNFOrchestrator.DeployVNF(ctx, vnfSpec, placement)
	require.NoError(suite.T(), err)

	vnfID := deploymentResult.VNFDeploymentID

	// Step 2: Scale slice
	err = suite.VNFOrchestrator.ScaleVNF(ctx, vnfID, 1.5) // Scale up by 50%
	require.NoError(suite.T(), err)

	// Validate scaling
	suite.validateScaling(ctx, vnfID, 1.5)

	// Step 3: Update QoS (optimization intent)
	optimizationIntent := fixtures.QoSOptimizationIntent()
	optimizationIntent.Constraints["existing-slice-id"] = vnfID

	_, err = suite.IntentProcessor.ProcessIntent(ctx, optimizationIntent)
	require.NoError(suite.T(), err)

	// Step 4: Delete slice
	err = suite.VNFOrchestrator.DeleteVNF(ctx, vnfID)
	require.NoError(suite.T(), err)

	// Validate deletion
	suite.validateDeletion(ctx, vnfID)
}

// Test error handling and recovery
func (suite *E2EIntentFlowTestSuite) TestErrorHandlingAndRecovery() {
	ctx, cancel := context.WithTimeout(context.Background(), suite.TestTimeout)
	defer cancel()

	// Test 1: Invalid intent handling
	invalidIntent := fixtures.InvalidIntent()
	_, err := suite.IntentProcessor.ProcessIntent(ctx, invalidIntent)
	assert.Error(suite.T(), err)

	// Test 2: Placement failure recovery
	conflictingRequest := fixtures.ConflictingConstraintsPlacementRequest()
	placement, err := suite.PlacementEngine.FindOptimalPlacement(ctx, conflictingRequest)
	if err == nil {
		// Should find a compromise solution
		assert.False(suite.T(), placement.Constraints.Feasible)
		assert.NotEmpty(suite.T(), placement.Alternatives)
	}

	// Test 3: Deployment failure handling
	// Simulate deployment failure by using invalid VNF spec
	invalidVNF := fixtures.InvalidVNFDeployment()
	validPlacement := func() *fixtures.PlacementSolution {
		s := fixtures.ExpectedEMBBPlacementSolution()
		return &s
	}()

	_, err = suite.VNFOrchestrator.DeployVNF(ctx, invalidVNF, validPlacement)
	assert.Error(suite.T(), err)

	// Test 4: DMS communication failure recovery
	// Should retry and handle gracefully
	err = suite.DMSClient.RegisterVNF(ctx, fixtures.ValidVNFDeployment())
	// Error is acceptable, but should not cause system crash
}

// Helper methods for validation and setup
func (suite *E2EIntentFlowTestSuite) setupDefaultMockBehaviors() {
	// Setup metrics collector defaults
	suite.MockMetrics.CollectLatencyFunc = func(ctx context.Context, target string) (*mocks.LatencyMetrics, error) {
		return mocks.CreateDefaultLatencyMetrics(target), nil
	}

	suite.MockMetrics.CollectThroughputFunc = func(ctx context.Context, target string) (*mocks.ThroughputMetrics, error) {
		return mocks.CreateDefaultThroughputMetrics(target), nil
	}

	suite.MockMetrics.CollectResourceFunc = func(ctx context.Context, target string) (*mocks.ResourceMetrics, error) {
		return mocks.CreateDefaultResourceMetrics(target), nil
	}

	// Setup HTTP client defaults
	suite.MockHTTP.DoFunc = func(req *mocks.HTTPRequest) (*mocks.HTTPResponse, error) {
		return mocks.CreateHTTPResponse(200, fixtures.ValidInventoryResponse(), nil), nil
	}

	// Setup Porch client defaults
	suite.MockPorch.CreatePackageFunc = func(ctx context.Context, pkg *mocks.NephioPackage) error {
		return nil
	}
}

func (suite *E2EIntentFlowTestSuite) initializeSystemComponents() {
	// These components don't exist yet - will cause tests to fail (RED phase)
	suite.IntentProcessor = nil
	suite.PlacementEngine = nil
	suite.VNFOrchestrator = nil
	suite.DMSClient = nil
	suite.GitOpsClient = nil
}

func (suite *E2EIntentFlowTestSuite) validateE2EMetrics(ctx context.Context, result *VNFDeploymentResult, expectedQoS fixtures.QoSProfile) {
	// Validate that deployed VNF meets QoS requirements
	if result.Metrics != nil {
		// Parse expected latency
		expectedLatencyMs := 50 // Default for eMBB
		if expectedQoS.Latency.Value == "1" {
			expectedLatencyMs = 1 // URLLC
		} else if expectedQoS.Latency.Value == "100" {
			expectedLatencyMs = 100 // mMTC
		}

		expectedLatency := time.Duration(expectedLatencyMs) * time.Millisecond
		assert.LessOrEqual(suite.T(), result.Metrics.LatencyP99, expectedLatency)

		// Validate throughput if specified
		if expectedQoS.Throughput.Downlink != "" {
			assert.GreaterOrEqual(suite.T(), result.Metrics.Throughput, 100.0) // Minimum baseline
		}

		// Validate resource utilization is reasonable
		assert.LessOrEqual(suite.T(), result.Metrics.ResourceUsage.CPU, 90.0)
		assert.LessOrEqual(suite.T(), result.Metrics.ResourceUsage.Memory, 90.0)
	}
}

func (suite *E2EIntentFlowTestSuite) verifySystemStateConsistency(ctx context.Context, vnfID string) {
	// Verify VNF is registered in DMS
	status, err := suite.DMSClient.GetVNFStatus(ctx, vnfID)
	if err == nil {
		assert.Contains(suite.T(), []string{"active", "deployed", "running"}, status)
	}

	// Verify GitOps package is created and synced
	packageStatus, err := suite.GitOpsClient.GetPackageStatus(ctx, vnfID)
	if err == nil {
		assert.Contains(suite.T(), []string{"synced", "published", "ready"}, packageStatus)
	}
}

func (suite *E2EIntentFlowTestSuite) validateCriticalApplicationRequirements(ctx context.Context, result *VNFDeploymentResult) {
	// URLLC-specific validations
	assert.NotEmpty(suite.T(), result.NetworkPaths)

	for _, path := range result.NetworkPaths {
		// Ultra-low latency requirement
		assert.LessOrEqual(suite.T(), path.Latency, 1*time.Millisecond)
		// High reliability path
		assert.Equal(suite.T(), "ultra-low-latency", path.QoS)
	}
}

func (suite *E2EIntentFlowTestSuite) validateMassiveConnectivity(ctx context.Context, result *VNFDeploymentResult) {
	// mMTC-specific validations
	assert.NotEmpty(suite.T(), result.Resources.Nodes)

	// Should be optimized for many concurrent connections
	if result.Metrics != nil {
		// Lower resource per connection efficiency
		assert.LessOrEqual(suite.T(), result.Metrics.ResourceUsage.CPU, 60.0)
		assert.LessOrEqual(suite.T(), result.Metrics.ResourceUsage.Memory, 70.0)
	}
}

func (suite *E2EIntentFlowTestSuite) validateNoResourceConflicts(ctx context.Context) {
	// Verify that concurrent deployments don't have resource conflicts
	// This would involve checking resource allocation across all deployments

	// Check with metrics collector
	metrics, err := suite.MockMetrics.CollectResourceMetrics(ctx, "cluster")
	if err == nil {
		// Ensure cluster is not over-allocated
		assert.LessOrEqual(suite.T(), metrics.CPU.Usage, 95.0)
		assert.LessOrEqual(suite.T(), metrics.Memory.Usage, 95.0)
	}
}

func (suite *E2EIntentFlowTestSuite) validateScaling(ctx context.Context, vnfID string, scaleFactor float64) {
	// Validate that VNF was scaled appropriately
	status, err := suite.DMSClient.GetVNFStatus(ctx, vnfID)
	if err == nil {
		assert.Equal(suite.T(), "active", status)
	}

	// Additional scaling validations would go here
}

func (suite *E2EIntentFlowTestSuite) validateDeletion(ctx context.Context, vnfID string) {
	// Validate that VNF was properly deleted
	status, err := suite.DMSClient.GetVNFStatus(ctx, vnfID)
	if err == nil {
		assert.Contains(suite.T(), []string{"deleted", "terminated", "not_found"}, status)
	}
}

func (suite *E2EIntentFlowTestSuite) cleanupTestResources() {
	// Clean up any test resources
	// This would involve deleting test VNFs, cleaning up K8s resources, etc.
}

// Test runner
func TestE2EIntentFlowTestSuite(t *testing.T) {
	suite.Run(t, new(E2EIntentFlowTestSuite))
}

// Individual test functions for specific scenarios
func TestCompleteEMBBFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	suite := &E2EIntentFlowTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	suite.SetupTest()
	suite.TestEMBBSliceCreationFlow()
}

func TestCompleteURLLCFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	suite := &E2EIntentFlowTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	suite.SetupTest()
	suite.TestURLLCSliceCreationFlow()
}

func TestCompleteMmTCFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	suite := &E2EIntentFlowTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	suite.SetupTest()
	suite.TestMmTCSliceCreationFlow()
}