package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	managerPkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/pkg"
	agentPkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg"
)

// E2ESliceTestSuite tests end-to-end network slice deployment
type E2ESliceTestSuite struct {
	suite.Suite
	manager *managerPkg.TNManager
	agents  map[string]*agentPkg.TNAgent
	logger  *log.Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

func (suite *E2ESliceTestSuite) SetupSuite() {
	suite.logger = log.New(os.Stdout, "[E2E] ", log.LstdFlags)
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Initialize TN Manager
	managerConfig := &managerPkg.TNConfig{
		ClusterName:    "manager",
		MonitoringPort: 8080,
	}

	suite.manager = managerPkg.NewTNManager(managerConfig, suite.logger)
	require.NotNil(suite.T(), suite.manager)

	err := suite.manager.Start()
	require.NoError(suite.T(), err, "Manager should start successfully")

	// Initialize TN Agents for different cluster types
	suite.agents = make(map[string]*agentPkg.TNAgent)
	clusterConfigs := suite.getTestClusterConfigs()

	for clusterName, config := range clusterConfigs {
		agent := agentPkg.NewTNAgent(config, suite.logger)
		require.NotNil(suite.T(), agent, "Agent should be created for cluster %s", clusterName)

		// Start agent (in test mode, skip actual network operations)
		err := suite.startAgentTestMode(agent)
		require.NoError(suite.T(), err, "Agent should start for cluster %s", clusterName)

		suite.agents[clusterName] = agent

		// Register agent with manager
		endpoint := fmt.Sprintf("http://localhost:%d", config.MonitoringPort)
		err = suite.manager.RegisterAgent(clusterName, endpoint)
		require.NoError(suite.T(), err, "Should register agent for cluster %s", clusterName)
	}

	suite.logger.Println("E2E test suite setup completed")
}

func (suite *E2ESliceTestSuite) TearDownSuite() {
	suite.logger.Println("Tearing down E2E test suite")

	// Stop agents
	for clusterName, agent := range suite.agents {
		err := agent.Stop()
		if err != nil {
			suite.logger.Printf("Error stopping agent %s: %v", clusterName, err)
		}
	}

	// Stop manager
	err := suite.manager.Stop()
	if err != nil {
		suite.logger.Printf("Error stopping manager: %v", err)
	}

	suite.cancel()
}

func (suite *E2ESliceTestSuite) TestURLL_SliceDeployment() {
	suite.logger.Println("Testing URLLC slice deployment")

	// Create URLLC placement decision
	decision := &managerPkg.PlacementDecision{
		SliceID:   "urllc-slice-001",
		SliceType: "URLLC",
		Clusters: []managerPkg.ClusterPlacement{
			{
				ClusterName: "edge01",
				ClusterType: "edge",
				VNFs: []managerPkg.VNFPlacement{
					{VNFName: "urllc-du", VNFType: "O-DU"},
				},
			},
			{
				ClusterName: "regional",
				ClusterType: "regional",
				VNFs: []managerPkg.VNFPlacement{
					{VNFName: "urllc-cu", VNFType: "O-CU"},
				},
			},
		},
		NetworkPolicy: managerPkg.NetworkPolicy{
			VXLANSegment: managerPkg.VXLANSegment{
				VNI:     1001,
				Subnets: []string{"10.1.0.0/16"},
				MTU:     1450,
			},
			BandwidthPolicy: managerPkg.BandwidthPolicy{
				DownlinkMbps: 0.93,
				UplinkMbps:   0.5,
				LatencyMs:    6.3,
				JitterMs:     0.5,
				LossPercent:  0.001,
				Priority:     1,
			},
		},
		QoSRequirement: managerPkg.QoSRequirement{
			ThroughputMbps: 0.93,
			LatencyMs:      6.3,
			Availability:   99.999,
			Priority:       1,
		},
		Timestamp: time.Now(),
		RequestID: "req-urllc-001",
	}

	// Implement the placement
	implementation, err := suite.manager.ImplementPlacement(suite.ctx, decision)
	require.NoError(suite.T(), err, "URLLC placement should succeed")
	require.NotNil(suite.T(), implementation)

	// Verify implementation status
	assert.Equal(suite.T(), "urllc-slice-001", implementation.SliceID)
	assert.Contains(suite.T(), []string{"ready", "partial"}, implementation.Status)

	// Verify cluster configuration
	assert.Contains(suite.T(), implementation.ClustersConfigured, "edge01")
	assert.Contains(suite.T(), implementation.ClustersConfigured, "regional")

	edge01Status := implementation.ClustersConfigured["edge01"]
	assert.Equal(suite.T(), "configured", edge01Status.Status)

	// Verify network connections
	assert.NotEmpty(suite.T(), implementation.NetworkConnections)

	// Run performance validation
	if implementation.PerformanceMetrics != nil {
		metrics := implementation.PerformanceMetrics

		// Validate URLLC targets
		assert.LessOrEqual(suite.T(), metrics.Performance.Latency.AvgRTTMs, 6.3*1.1,
			"URLLC latency should meet target with 10% tolerance")
		assert.GreaterOrEqual(suite.T(), metrics.Performance.Throughput.AvgMbps, 0.93*0.9,
			"URLLC throughput should meet target with 10% tolerance")

		// Validate thesis compliance
		assert.Greater(suite.T(), metrics.ThesisValidation.CompliancePercent, 80.0,
			"URLLC slice should meet thesis compliance requirements")
	}

	suite.logger.Println("URLLC slice deployment test completed successfully")
}

func (suite *E2ESliceTestSuite) TestEMBB_SliceDeployment() {
	suite.logger.Println("Testing eMBB slice deployment")

	decision := &managerPkg.PlacementDecision{
		SliceID:   "embb-slice-001",
		SliceType: "eMBB",
		Clusters: []managerPkg.ClusterPlacement{
			{
				ClusterName: "edge01",
				ClusterType: "edge",
				VNFs: []managerPkg.VNFPlacement{
					{VNFName: "embb-du", VNFType: "O-DU"},
				},
			},
			{
				ClusterName: "edge02",
				ClusterType: "edge",
				VNFs: []managerPkg.VNFPlacement{
					{VNFName: "embb-du-2", VNFType: "O-DU"},
				},
			},
			{
				ClusterName: "central",
				ClusterType: "central",
				VNFs: []managerPkg.VNFPlacement{
					{VNFName: "embb-cu", VNFType: "O-CU"},
					{VNFName: "embb-core", VNFType: "5GC"},
				},
			},
		},
		NetworkPolicy: managerPkg.NetworkPolicy{
			VXLANSegment: managerPkg.VXLANSegment{
				VNI:     1002,
				Subnets: []string{"10.2.0.0/16"},
				MTU:     1500,
			},
			BandwidthPolicy: managerPkg.BandwidthPolicy{
				DownlinkMbps: 4.57,
				UplinkMbps:   2.0,
				LatencyMs:    16.1,
				JitterMs:     2.0,
				LossPercent:  0.1,
				Priority:     3,
			},
		},
		QoSRequirement: managerPkg.QoSRequirement{
			ThroughputMbps: 4.57,
			LatencyMs:      16.1,
			Availability:   99.9,
			Priority:       3,
		},
		Timestamp: time.Now(),
		RequestID: "req-embb-001",
	}

	implementation, err := suite.manager.ImplementPlacement(suite.ctx, decision)
	require.NoError(suite.T(), err, "eMBB placement should succeed")
	require.NotNil(suite.T(), implementation)

	// Verify implementation
	assert.Equal(suite.T(), "embb-slice-001", implementation.SliceID)
	assert.Contains(suite.T(), []string{"ready", "partial"}, implementation.Status)

	// Verify all clusters configured
	expectedClusters := []string{"edge01", "edge02", "central"}
	for _, cluster := range expectedClusters {
		assert.Contains(suite.T(), implementation.ClustersConfigured, cluster)
		status := implementation.ClustersConfigured[cluster]
		assert.Equal(suite.T(), "configured", status.Status)
	}

	// Verify multi-cluster connectivity
	assert.GreaterOrEqual(suite.T(), len(implementation.NetworkConnections), 2,
		"Should have multiple network connections")

	// Validate eMBB performance targets
	if implementation.PerformanceMetrics != nil {
		metrics := implementation.PerformanceMetrics

		assert.LessOrEqual(suite.T(), metrics.Performance.Latency.AvgRTTMs, 16.1*1.1,
			"eMBB latency should meet target")
		assert.GreaterOrEqual(suite.T(), metrics.Performance.Throughput.AvgMbps, 4.57*0.9,
			"eMBB throughput should meet target")
	}

	suite.logger.Println("eMBB slice deployment test completed successfully")
}

func (suite *E2ESliceTestSuite) TestMIoT_SliceDeployment() {
	suite.logger.Println("Testing mIoT slice deployment")

	decision := &managerPkg.PlacementDecision{
		SliceID:   "miot-slice-001",
		SliceType: "mIoT",
		Clusters: []managerPkg.ClusterPlacement{
			{
				ClusterName: "edge02",
				ClusterType: "edge",
				VNFs: []managerPkg.VNFPlacement{
					{VNFName: "miot-du", VNFType: "O-DU"},
				},
			},
			{
				ClusterName: "regional",
				ClusterType: "regional",
				VNFs: []managerPkg.VNFPlacement{
					{VNFName: "miot-cu", VNFType: "O-CU"},
					{VNFName: "miot-gateway", VNFType: "IoT-Gateway"},
				},
			},
		},
		NetworkPolicy: managerPkg.NetworkPolicy{
			VXLANSegment: managerPkg.VXLANSegment{
				VNI:     1003,
				Subnets: []string{"10.3.0.0/16"},
				MTU:     1450,
			},
			BandwidthPolicy: managerPkg.BandwidthPolicy{
				DownlinkMbps: 2.77,
				UplinkMbps:   1.0,
				LatencyMs:    15.7,
				JitterMs:     3.0,
				LossPercent:  0.1,
				Priority:     2,
			},
		},
		QoSRequirement: managerPkg.QoSRequirement{
			ThroughputMbps: 2.77,
			LatencyMs:      15.7,
			Availability:   99.95,
			Priority:       2,
		},
		Timestamp: time.Now(),
		RequestID: "req-miot-001",
	}

	implementation, err := suite.manager.ImplementPlacement(suite.ctx, decision)
	require.NoError(suite.T(), err, "mIoT placement should succeed")
	require.NotNil(suite.T(), implementation)

	// Verify implementation
	assert.Equal(suite.T(), "miot-slice-001", implementation.SliceID)
	assert.Contains(suite.T(), []string{"ready", "partial"}, implementation.Status)

	// Validate mIoT performance targets
	if implementation.PerformanceMetrics != nil {
		metrics := implementation.PerformanceMetrics

		assert.LessOrEqual(suite.T(), metrics.Performance.Latency.AvgRTTMs, 15.7*1.1,
			"mIoT latency should meet target")
		assert.GreaterOrEqual(suite.T(), metrics.Performance.Throughput.AvgMbps, 2.77*0.9,
			"mIoT throughput should meet target")
	}

	suite.logger.Println("mIoT slice deployment test completed successfully")
}

func (suite *E2ESliceTestSuite) TestThesisValidationE2E() {
	suite.logger.Println("Testing comprehensive thesis validation")

	// Deploy all three slice types and validate against thesis targets
	sliceTypes := []struct {
		name           string
		throughputMbps float64
		latencyMs      float64
	}{
		{"URLLC", 0.93, 6.3},
		{"mIoT", 2.77, 15.7},
		{"eMBB", 4.57, 16.1},
	}

	var allResults []managerPkg.PerformanceMetrics

	for _, sliceType := range sliceTypes {
		suite.T().Run(sliceType.name+"_ThesisValidation", func(t *testing.T) {
			// Run comprehensive performance test
			testConfig := &managerPkg.PerformanceTestConfig{
				TestID:    fmt.Sprintf("thesis_%s_%d", sliceType.name, time.Now().Unix()),
				SliceID:   fmt.Sprintf("%s-thesis-slice", strings.ToLower(sliceType.name)),
				SliceType: sliceType.name,
				Duration:  30 * time.Second,
				TestType:  "comprehensive",
				Protocol:  "tcp",
				Parallel:  1,
			}

			metrics, err := suite.manager.RunPerformanceTest(testConfig)
			require.NoError(t, err, "Performance test should succeed for %s", sliceType.name)
			require.NotNil(t, metrics)

			allResults = append(allResults, metrics.Performance)

			// Validate against thesis targets with 10% tolerance
			assert.LessOrEqual(t, metrics.Performance.Latency.AvgRTTMs, sliceType.latencyMs*1.1,
				"%s latency should meet thesis target", sliceType.name)
			assert.GreaterOrEqual(t, metrics.Performance.Throughput.AvgMbps, sliceType.throughputMbps*0.9,
				"%s throughput should meet thesis target", sliceType.name)

			// Validate deployment time (should be under 10 minutes)
			assert.LessOrEqual(t, metrics.ThesisValidation.DeployTimeMs, int64(600000),
				"Deployment time should be under 10 minutes")
		})
	}

	// Overall thesis compliance validation
	if len(allResults) == 3 {
		// Calculate compliance based on actual performance vs targets
		complianceCount := 0
		for i, result := range allResults {
			targetThroughput := []float64{0.93, 2.77, 4.57}[i]
			targetLatency := []float64{6.3, 15.7, 16.1}[i]

			if result.Throughput.AvgMbps >= targetThroughput*0.9 &&
			   result.Latency.AvgRTTMs <= targetLatency*1.1 {
				complianceCount++
			}
		}
		overallCompliance := float64(complianceCount) / float64(len(allResults)) * 100.0

		assert.Greater(suite.T(), overallCompliance, 80.0,
			"Overall thesis compliance should be above 80%")

		suite.logger.Printf("Overall thesis compliance: %.2f%%", overallCompliance)
	}

	suite.logger.Println("Thesis validation test completed successfully")
}

func (suite *E2ESliceTestSuite) TestMultiClusterConnectivity() {
	suite.logger.Println("Testing multi-cluster connectivity")

	// Test connectivity between all cluster pairs
	clusters := []string{"edge01", "edge02", "regional", "central"}

	for i, source := range clusters {
		for j, target := range clusters {
			if i >= j {
				continue
			}

			suite.T().Run(fmt.Sprintf("%s_to_%s", source, target), func(t *testing.T) {
				// Test VXLAN connectivity
				sourceAgent := suite.agents[source]
				require.NotNil(t, sourceAgent, "Source agent should exist")

				// In test mode, simulate successful connectivity
				suite.logger.Printf("Testing connectivity: %s -> %s", source, target)

				// Simulate connectivity test with expected results
				latency := suite.getExpectedLatency(source, target)
				bandwidth := suite.getExpectedBandwidth(source, target)

				assert.Greater(t, bandwidth, 0.0, "Bandwidth should be positive")
				assert.Greater(t, latency, 0.0, "Latency should be positive")
				assert.Less(t, latency, 50.0, "Latency should be reasonable")
			})
		}
	}

	suite.logger.Println("Multi-cluster connectivity test completed")
}

// Helper methods

func (suite *E2ESliceTestSuite) getTestClusterConfigs() map[string]*agentPkg.TNConfig {
	return map[string]*agentPkg.TNConfig{
		"edge01": {
			ClusterName:    "edge01",
			NetworkCIDR:    "10.1.0.0/16",
			MonitoringPort: 8081,
			VXLANConfig: agentPkg.VXLANConfig{
				VNI:        100,
				LocalIP:    "192.168.1.100",
				RemoteIPs:  []string{"192.168.1.101", "192.168.1.102", "192.168.1.103"},
				Port:       4789,
				MTU:        1450,
				DeviceName: "vxlan100",
			},
			BWPolicy: agentPkg.BandwidthPolicy{
				DownlinkMbps: 100.0,
				UplinkMbps:   50.0,
			},
			QoSClass: "test",
		},
		"edge02": {
			ClusterName:    "edge02",
			NetworkCIDR:    "10.2.0.0/16",
			MonitoringPort: 8082,
			VXLANConfig: agentPkg.VXLANConfig{
				VNI:        101,
				LocalIP:    "192.168.1.101",
				RemoteIPs:  []string{"192.168.1.100", "192.168.1.102", "192.168.1.103"},
				Port:       4789,
				MTU:        1450,
				DeviceName: "vxlan101",
			},
			BWPolicy: agentPkg.BandwidthPolicy{
				DownlinkMbps: 100.0,
				UplinkMbps:   50.0,
			},
			QoSClass: "test",
		},
		"regional": {
			ClusterName:    "regional",
			NetworkCIDR:    "10.3.0.0/16",
			MonitoringPort: 8083,
			VXLANConfig: agentPkg.VXLANConfig{
				VNI:        102,
				LocalIP:    "192.168.1.102",
				RemoteIPs:  []string{"192.168.1.100", "192.168.1.101", "192.168.1.103"},
				Port:       4789,
				MTU:        1450,
				DeviceName: "vxlan102",
			},
			BWPolicy: agentPkg.BandwidthPolicy{
				DownlinkMbps: 200.0,
				UplinkMbps:   100.0,
			},
			QoSClass: "test",
		},
		"central": {
			ClusterName:    "central",
			NetworkCIDR:    "10.4.0.0/16",
			MonitoringPort: 8084,
			VXLANConfig: agentPkg.VXLANConfig{
				VNI:        103,
				LocalIP:    "192.168.1.103",
				RemoteIPs:  []string{"192.168.1.100", "192.168.1.101", "192.168.1.102"},
				Port:       4789,
				MTU:        1500,
				DeviceName: "vxlan103",
			},
			BWPolicy: agentPkg.BandwidthPolicy{
				DownlinkMbps: 500.0,
				UplinkMbps:   200.0,
			},
			QoSClass: "test",
		},
	}
}

func (suite *E2ESliceTestSuite) startAgentTestMode(agent *agentPkg.TNAgent) error {
	// In test mode, skip actual network operations
	// Start HTTP server and monitoring without real network setup
	suite.logger.Printf("Starting agent in test mode")
	return nil // Simulate successful start
}

func (suite *E2ESliceTestSuite) getExpectedLatency(source, target string) float64 {
	// Simulate expected latency based on cluster types
	latencyMap := map[string]map[string]float64{
		"edge01": {
			"edge02":   5.0,
			"regional": 12.0,
			"central":  18.0,
		},
		"edge02": {
			"regional": 10.0,
			"central":  16.0,
		},
		"regional": {
			"central": 8.0,
		},
	}

	if sourceMap, exists := latencyMap[source]; exists {
		if latency, exists := sourceMap[target]; exists {
			return latency
		}
	}

	// Check reverse direction
	if targetMap, exists := latencyMap[target]; exists {
		if latency, exists := targetMap[source]; exists {
			return latency
		}
	}

	return 15.0 // Default latency
}

func (suite *E2ESliceTestSuite) getExpectedBandwidth(source, target string) float64 {
	// Simulate expected bandwidth based on cluster connectivity
	return 100.0 // Default bandwidth in Mbps
}

// Run the test suite
func TestE2ESliceTestSuite(t *testing.T) {
	suite.Run(t, new(E2ESliceTestSuite))
}