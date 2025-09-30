// Package e2e provides end-to-end integration tests for the O-RAN Intent MANO system.
// These tests validate the complete workflow from natural language intent to operational network slice.
//
// Test Coverage:
// - NLP intent parsing
// - Claude AI analysis
// - Orchestrator placement calculation
// - Nephio package generation
// - VNF deployment
// - Transport network configuration
// - End-to-end slice provisioning
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite encapsulates all end-to-end integration tests
type E2ETestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	testTimeout       time.Duration
	orchestratorURL   string
	vnfOperatorURL    string
	nephioRendererURL string
	tnAgentURL        string
}

// SetupSuite runs once before all tests
func (s *E2ETestSuite) SetupSuite() {
	s.testTimeout = 5 * time.Minute

	// Load environment configuration
	s.orchestratorURL = getEnvOrDefault("ORCHESTRATOR_URL", "http://localhost:8080")
	s.vnfOperatorURL = getEnvOrDefault("VNF_OPERATOR_URL", "http://localhost:8081")
	s.nephioRendererURL = getEnvOrDefault("NEPHIO_RENDERER_URL", "http://localhost:8082")
	s.tnAgentURL = getEnvOrDefault("TN_AGENT_URL", "http://localhost:8083")

	s.T().Logf("E2E Test Suite Configuration:")
	s.T().Logf("  Orchestrator: %s", s.orchestratorURL)
	s.T().Logf("  VNF Operator: %s", s.vnfOperatorURL)
	s.T().Logf("  Nephio Renderer: %s", s.nephioRendererURL)
	s.T().Logf("  TN Agent: %s", s.tnAgentURL)
}

// SetupTest runs before each test
func (s *E2ETestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), s.testTimeout)
}

// TearDownTest runs after each test
func (s *E2ETestSuite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
}

// TestEndToEndSliceProvisioning validates the complete slice provisioning workflow
func (s *E2ETestSuite) TestEndToEndSliceProvisioning() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	// Natural language intent from user
	intent := "Create a network slice for video streaming with 100 Mbps throughput and 20ms latency"
	sliceID := ""

	s.Run("Step1_NLP_Processing", func() {
		s.T().Log("Step 1: Processing natural language intent with NLP module")

		parsedIntent, err := s.processNLPIntent(intent)
		require.NoError(s.T(), err, "NLP processing should succeed")
		require.NotNil(s.T(), parsedIntent, "Parsed intent should not be nil")

		// Validate parsed intent structure
		assert.Equal(s.T(), "video", parsedIntent.ServiceType, "Service type should be video")
		assert.Equal(s.T(), "100 Mbps", parsedIntent.Throughput, "Throughput should be 100 Mbps")
		assert.Equal(s.T(), "20ms", parsedIntent.Latency, "Latency should be 20ms")
		assert.NotEmpty(s.T(), parsedIntent.IntentID, "Intent ID should be generated")

		s.T().Logf("✓ NLP parsed intent: %+v", parsedIntent)
	})

	s.Run("Step2_Claude_Analysis", func() {
		s.T().Log("Step 2: Analyzing intent with Claude AI")

		analysis, err := s.analyzeWithClaude(s.ctx, intent)
		require.NoError(s.T(), err, "Claude analysis should succeed")
		require.NotNil(s.T(), analysis, "Analysis should not be nil")

		// Validate analysis contains required fields
		assert.NotEmpty(s.T(), analysis.Requirements, "Requirements should not be empty")
		assert.Contains(s.T(), analysis.Requirements, "throughput", "Should contain throughput requirement")
		assert.Contains(s.T(), analysis.Requirements, "latency", "Should contain latency requirement")
		assert.NotEmpty(s.T(), analysis.RecommendedVNFs, "Should recommend VNFs")
		assert.Greater(s.T(), analysis.ConfidenceScore, 0.7, "Confidence should be high")

		s.T().Logf("✓ Claude analysis: %d requirements, %d VNFs recommended",
			len(analysis.Requirements), len(analysis.RecommendedVNFs))
	})

	s.Run("Step3_Orchestrator_Placement", func() {
		s.T().Log("Step 3: Calculating optimal resource placement")

		placement, err := s.calculatePlacement(s.ctx)
		require.NoError(s.T(), err, "Placement calculation should succeed")
		require.NotNil(s.T(), placement, "Placement should not be nil")

		// Validate placement results
		assert.NotEmpty(s.T(), placement.CNFPlacements, "Should have CNF placements")
		assert.NotEmpty(s.T(), placement.TransportLinks, "Should have transport links")
		assert.NotEmpty(s.T(), placement.PlacementID, "Placement ID should be generated")

		// Verify resource constraints are met
		for _, cnf := range placement.CNFPlacements {
			assert.NotEmpty(s.T(), cnf.NodeID, "Node ID should be assigned")
			assert.NotEmpty(s.T(), cnf.CNFID, "CNF ID should be assigned")
			assert.True(s.T(), cnf.CPUAllocated > 0, "CPU should be allocated")
			assert.True(s.T(), cnf.MemoryAllocated > 0, "Memory should be allocated")
		}

		s.T().Logf("✓ Placement calculated: %d CNFs, %d transport links",
			len(placement.CNFPlacements), len(placement.TransportLinks))
	})

	s.Run("Step4_Nephio_PackageGeneration", func() {
		s.T().Log("Step 4: Generating Nephio packages")

		packages, err := s.generateNephioPackages(s.ctx)
		require.NoError(s.T(), err, "Package generation should succeed")
		require.NotEmpty(s.T(), packages, "Should generate at least one package")

		// Validate each package structure
		for i, pkg := range packages {
			s.T().Logf("Validating package %d: %s", i+1, pkg.Name)

			// Check Kptfile exists
			kptfilePath := fmt.Sprintf("%s/Kptfile", pkg.Path)
			assert.FileExists(s.T(), kptfilePath, "Kptfile should exist")

			// Check resources directory
			resourcesPath := fmt.Sprintf("%s/resources", pkg.Path)
			assert.DirExists(s.T(), resourcesPath, "Resources directory should exist")

			// Validate package metadata
			assert.NotEmpty(s.T(), pkg.Name, "Package name should not be empty")
			assert.NotEmpty(s.T(), pkg.Version, "Package version should not be empty")
			assert.NotEmpty(s.T(), pkg.Path, "Package path should not be empty")
		}

		s.T().Logf("✓ Generated %d Nephio packages", len(packages))
	})

	s.Run("Step5_VNF_Deployment", func() {
		s.T().Log("Step 5: Deploying VNFs to Kubernetes")

		deploymentID, err := s.deployVNFs(s.ctx)
		require.NoError(s.T(), err, "VNF deployment should succeed")
		require.NotEmpty(s.T(), deploymentID, "Deployment ID should be returned")

		sliceID = deploymentID

		// Wait for deployment to become ready
		s.T().Log("Waiting for VNF deployment to become ready...")
		err = s.waitForDeploymentReady(s.ctx, deploymentID, 2*time.Minute)
		require.NoError(s.T(), err, "Deployment should become ready within timeout")

		// Verify deployment status
		status, err := s.getDeploymentStatus(s.ctx, deploymentID)
		require.NoError(s.T(), err, "Should get deployment status")
		assert.Equal(s.T(), "Ready", status.State, "Deployment should be in Ready state")
		assert.True(s.T(), status.AllPodsRunning, "All pods should be running")

		s.T().Logf("✓ VNFs deployed successfully: %s", deploymentID)
	})

	s.Run("Step6_TN_Configuration", func() {
		s.T().Log("Step 6: Configuring transport network")

		err := s.configureTN(s.ctx)
		require.NoError(s.T(), err, "TN configuration should succeed")

		// Verify VXLAN interfaces are created
		interfaces, err := s.listVXLANInterfaces(s.ctx)
		require.NoError(s.T(), err, "Should list VXLAN interfaces")
		assert.NotEmpty(s.T(), interfaces, "VXLAN interfaces should be created")

		// Verify each interface is properly configured
		for _, iface := range interfaces {
			assert.NotEmpty(s.T(), iface.Name, "Interface name should not be empty")
			assert.NotEmpty(s.T(), iface.VNI, "VNI should be assigned")
			assert.Equal(s.T(), "up", iface.Status, "Interface should be up")
		}

		s.T().Logf("✓ Transport network configured: %d VXLAN interfaces", len(interfaces))
	})

	s.Run("Step7_Slice_Verification", func() {
		s.T().Log("Step 7: Verifying network slice is operational")

		require.NotEmpty(s.T(), sliceID, "Slice ID should be available from deployment")

		// Get slice operational status
		sliceStatus, err := s.getSliceStatus(s.ctx, sliceID)
		require.NoError(s.T(), err, "Should get slice status")
		require.NotNil(s.T(), sliceStatus, "Slice status should not be nil")

		// Verify slice is active
		assert.Equal(s.T(), "Active", sliceStatus.State, "Slice should be in Active state")
		assert.True(s.T(), sliceStatus.HealthCheck, "Health check should pass")

		// Verify QoS parameters meet requirements
		assert.GreaterOrEqual(s.T(), sliceStatus.Throughput, 100.0,
			"Throughput should meet or exceed 100 Mbps")
		assert.LessOrEqual(s.T(), sliceStatus.Latency, 20.0,
			"Latency should be 20ms or less")

		// Verify slice components
		assert.NotEmpty(s.T(), sliceStatus.ActiveVNFs, "Should have active VNFs")
		assert.NotEmpty(s.T(), sliceStatus.TransportLinks, "Should have transport links")

		s.T().Logf("✓ Network slice operational:")
		s.T().Logf("  - State: %s", sliceStatus.State)
		s.T().Logf("  - Throughput: %.2f Mbps", sliceStatus.Throughput)
		s.T().Logf("  - Latency: %.2f ms", sliceStatus.Latency)
		s.T().Logf("  - Active VNFs: %d", len(sliceStatus.ActiveVNFs))
	})
}

// TestEndToEndSliceProvisioningErrorHandling tests error scenarios
func (s *E2ETestSuite) TestEndToEndSliceProvisioningErrorHandling() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	s.Run("InvalidIntent_ShouldFail", func() {
		s.T().Log("Testing invalid intent handling")

		invalidIntent := "This is not a valid network slice intent"
		_, err := s.processNLPIntent(invalidIntent)

		assert.Error(s.T(), err, "Invalid intent should return error")
		s.T().Logf("✓ Invalid intent properly rejected")
	})

	s.Run("InsufficientResources_ShouldFail", func() {
		s.T().Log("Testing resource exhaustion scenario")

		// Simulate resource exhaustion
		_, err := s.calculatePlacementWithConstraints(s.ctx, ResourceConstraints{
			MaxCPU:    0.1, // Insufficient CPU
			MaxMemory: 10,  // Insufficient memory
		})

		assert.Error(s.T(), err, "Insufficient resources should return error")
		assert.Contains(s.T(), err.Error(), "insufficient resources",
			"Error should indicate resource constraint")
		s.T().Logf("✓ Resource constraints properly enforced")
	})

	s.Run("DeploymentFailure_ShouldRollback", func() {
		s.T().Log("Testing deployment failure and rollback")

		// Attempt deployment with invalid configuration
		deploymentID, err := s.deployVNFsWithConfig(s.ctx, &InvalidDeploymentConfig{})

		if err == nil {
			// If deployment started, verify rollback on failure
			err = s.waitForDeploymentReady(s.ctx, deploymentID, 30*time.Second)
			assert.Error(s.T(), err, "Invalid deployment should fail")

			// Verify rollback occurred
			status, _ := s.getDeploymentStatus(s.ctx, deploymentID)
			assert.Equal(s.T(), "RolledBack", status.State, "Should rollback on failure")
		}

		s.T().Logf("✓ Deployment failure properly handled")
	})
}

// TestEndToEndSliceProvisioningEdgeCases tests boundary conditions
func (s *E2ETestSuite) TestEndToEndSliceProvisioningEdgeCases() {
	if testing.Short() {
		s.T().Skip("skipping integration test in short mode")
	}

	s.Run("MinimalRequirements_ShouldSucceed", func() {
		s.T().Log("Testing minimal resource requirements")

		intent := "Create a basic network slice with 1 Mbps throughput"
		parsedIntent, err := s.processNLPIntent(intent)

		require.NoError(s.T(), err, "Minimal intent should be processed")
		assert.Equal(s.T(), "1 Mbps", parsedIntent.Throughput)
		s.T().Logf("✓ Minimal requirements handled")
	})

	s.Run("MaximalRequirements_ShouldSucceed", func() {
		s.T().Log("Testing maximum resource requirements")

		intent := "Create a network slice with 10 Gbps throughput and 1ms latency"
		parsedIntent, err := s.processNLPIntent(intent)

		require.NoError(s.T(), err, "Maximal intent should be processed")
		assert.Equal(s.T(), "10 Gbps", parsedIntent.Throughput)
		assert.Equal(s.T(), "1ms", parsedIntent.Latency)
		s.T().Logf("✓ Maximal requirements handled")
	})

	s.Run("ConcurrentDeployments_ShouldSucceed", func() {
		s.T().Log("Testing concurrent slice deployments")

		// Deploy multiple slices concurrently
		numSlices := 5
		deploymentIDs := make([]string, numSlices)
		errors := make([]error, numSlices)

		// Launch concurrent deployments
		done := make(chan bool)
		for i := 0; i < numSlices; i++ {
			go func(index int) {
				deploymentIDs[index], errors[index] = s.deployVNFs(s.ctx)
				done <- true
			}(i)
		}

		// Wait for all deployments
		for i := 0; i < numSlices; i++ {
			<-close(done)
		}

		// Verify all succeeded
		successCount := 0
		for i := 0; i < numSlices; i++ {
			if errors[i] == nil {
				successCount++
			}
		}

		assert.Greater(s.T(), successCount, 0, "At least some concurrent deployments should succeed")
		s.T().Logf("✓ Concurrent deployments: %d/%d succeeded", successCount, numSlices)
	})
}

// TestIntegrationWithMockedComponents tests individual component integrations
func (s *E2ETestSuite) TestIntegrationWithMockedComponents() {
	s.Run("Nephio_O2Client_Integration", func() {
		s.T().Log("Testing Nephio generator with O2 client")

		// Test Nephio package generation with mocked O2 inventory
		packages, err := s.generateNephioPackagesWithO2Mock(s.ctx)
		require.NoError(s.T(), err, "Nephio-O2 integration should succeed")
		assert.NotEmpty(s.T(), packages, "Should generate packages from O2 inventory")

		s.T().Logf("✓ Nephio-O2 integration validated")
	})

	s.Run("Orchestrator_VNF_Integration", func() {
		s.T().Log("Testing orchestrator with VNF operator")

		// Test orchestrator placement with mocked VNF operator
		placement, err := s.testOrchestratorVNFIntegration(s.ctx)
		require.NoError(s.T(), err, "Orchestrator-VNF integration should succeed")
		assert.NotNil(s.T(), placement, "Should get valid placement")

		s.T().Logf("✓ Orchestrator-VNF integration validated")
	})

	s.Run("TN_Agent_VXLAN_Integration", func() {
		s.T().Log("Testing TN agent with VXLAN manager")

		// Test TN agent with VXLAN configuration
		err := s.testTNAgentVXLANIntegration(s.ctx)
		require.NoError(s.T(), err, "TN-VXLAN integration should succeed")

		s.T().Logf("✓ TN-VXLAN integration validated")
	})
}

// TestSuite runs the E2E test suite
func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

// Helper function to get environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}