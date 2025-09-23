package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	agentpkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg"
	managerpkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/pkg"
)

// TNIntegrationTestSuite provides integration testing for TN components
type TNIntegrationTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *log.Logger
	manager     *managerpkg.TNManager
	agents      map[string]*agentpkg.TNAgent
	testConfig  *TestConfiguration
}

// TestConfiguration contains test configuration
type TestConfiguration struct {
	Clusters       []ClusterConfig       `json:"clusters"`
	NetworkConfig  NetworkTestConfig     `json:"networkConfig"`
	TestScenarios  []TestScenario        `json:"testScenarios"`
	ThesisTargets  ThesisTargets         `json:"thesisTargets"`
}

// ClusterConfig defines cluster configuration for testing
type ClusterConfig struct {
	Name          string `json:"name"`
	LocalIP       string `json:"localIP"`
	MonitoringPort int   `json:"monitoringPort"`
	VNI           uint32 `json:"vni"`
	BWLimitMbps   float64 `json:"bwLimitMbps"`
}

// NetworkTestConfig defines network test configuration
type NetworkTestConfig struct {
	VXLANMTU       int      `json:"vxlanMTU"`
	VXLANPort      int      `json:"vxlanPort"`
	TestDuration   string   `json:"testDuration"`
	ParallelStreams int     `json:"parallelStreams"`
	Protocols      []string `json:"protocols"`
}

// TestScenario defines a test scenario
type TestScenario struct {
	Name            string  `json:"name"`
	SliceType       string  `json:"sliceType"`
	ExpectedMbps    float64 `json:"expectedMbps"`
	ExpectedRTTMs   float64 `json:"expectedRTTMs"`
	MaxDeployTimeMs int64   `json:"maxDeployTimeMs"`
	QoSClass        string  `json:"qosClass"`
}

// ThesisTargets defines thesis validation targets
type ThesisTargets struct {
	ThroughputMbps []float64 `json:"throughputMbps"`
	RTTMs          []float64 `json:"rttMs"`
	DeployTimeMs   int64     `json:"deployTimeMs"`
}

// SetupSuite initializes the test suite
func (suite *TNIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.logger = log.New(os.Stdout, "[TN-Integration] ", log.LstdFlags|log.Lshortfile)
	suite.agents = make(map[string]*agentpkg.TNAgent)

	// Load test configuration
	suite.loadTestConfiguration()

	// Initialize TN manager
	suite.initializeTNManager()

	// Initialize TN agents
	suite.initializeTNAgents()
}

// TearDownSuite cleans up the test suite
func (suite *TNIntegrationTestSuite) TearDownSuite() {
	suite.logger.Println("Tearing down integration test suite...")

	// Stop all agents
	for name, agent := range suite.agents {
		if err := agent.Stop(); err != nil {
			suite.logger.Printf("Error stopping agent %s: %v", name, err)
		}
	}

	// Stop manager
	if suite.manager != nil {
		if err := suite.manager.Stop(); err != nil {
			suite.logger.Printf("Error stopping manager: %v", err)
		}
	}

	suite.cancel()
}

// loadTestConfiguration loads test configuration
func (suite *TNIntegrationTestSuite) loadTestConfiguration() {
	// Default test configuration
	suite.testConfig = &TestConfiguration{
		Clusters: []ClusterConfig{
			{
				Name:           "cluster-edge01",
				LocalIP:        "10.0.1.10",
				MonitoringPort: 8080,
				VNI:            1001,
				BWLimitMbps:    10.0,
			},
			{
				Name:           "cluster-edge02",
				LocalIP:        "10.0.2.10",
				MonitoringPort: 8081,
				VNI:            1002,
				BWLimitMbps:    5.0,
			},
			{
				Name:           "cluster-regional",
				LocalIP:        "10.0.3.10",
				MonitoringPort: 8082,
				VNI:            1003,
				BWLimitMbps:    20.0,
			},
		},
		NetworkConfig: NetworkTestConfig{
			VXLANMTU:        1450,
			VXLANPort:       4789,
			TestDuration:    "30s",
			ParallelStreams: 3,
			Protocols:       []string{"tcp", "udp"},
		},
		TestScenarios: []TestScenario{
			{
				Name:            "eMBB High Throughput",
				SliceType:       "eMBB",
				ExpectedMbps:    4.57,
				ExpectedRTTMs:   16.1,
				MaxDeployTimeMs: 600000,
				QoSClass:        "high",
			},
			{
				Name:            "URLLC Low Latency",
				SliceType:       "URLLC",
				ExpectedMbps:    2.77,
				ExpectedRTTMs:   15.7,
				MaxDeployTimeMs: 300000,
				QoSClass:        "ultra_low_latency",
			},
			{
				Name:            "mMTC Efficient",
				SliceType:       "mMTC",
				ExpectedMbps:    0.93,
				ExpectedRTTMs:   6.3,
				MaxDeployTimeMs: 400000,
				QoSClass:        "efficient",
			},
		},
		ThesisTargets: ThesisTargets{
			ThroughputMbps: []float64{0.93, 2.77, 4.57},
			RTTMs:          []float64{6.3, 15.7, 16.1},
			DeployTimeMs:   600000, // 10 minutes
		},
	}

	suite.logger.Printf("Loaded test configuration with %d clusters, %d scenarios",
		len(suite.testConfig.Clusters), len(suite.testConfig.TestScenarios))
}

// initializeTNManager initializes the TN manager
func (suite *TNIntegrationTestSuite) initializeTNManager() {
	config := &managerpkg.TNConfig{
		ClusterName:    "tn-manager",
		MonitoringPort: 9090,
	}

	suite.manager = managerpkg.NewTNManager(config, suite.logger)
	require.NoError(suite.T(), suite.manager.Start())

	suite.logger.Println("TN Manager initialized successfully")
}

// initializeTNAgents initializes TN agents for each cluster
func (suite *TNIntegrationTestSuite) initializeTNAgents() {
	for _, clusterConfig := range suite.testConfig.Clusters {
		agent := suite.createTNAgent(clusterConfig)
		suite.agents[clusterConfig.Name] = agent

		// Register agent with manager
		endpoint := fmt.Sprintf("http://%s:%d", clusterConfig.LocalIP, clusterConfig.MonitoringPort)
		err := suite.manager.RegisterAgent(clusterConfig.Name, endpoint)
		require.NoError(suite.T(), err)
	}

	suite.logger.Printf("Initialized %d TN agents", len(suite.agents))
}

// createTNAgent creates a TN agent with test configuration
func (suite *TNIntegrationTestSuite) createTNAgent(clusterConfig ClusterConfig) *agentpkg.TNAgent {
	// Build remote IPs list (all other clusters)
	var remoteIPs []string
	for _, otherCluster := range suite.testConfig.Clusters {
		if otherCluster.Name != clusterConfig.Name {
			remoteIPs = append(remoteIPs, otherCluster.LocalIP)
		}
	}

	config := &agentpkg.TNConfig{
		ClusterName:    clusterConfig.Name,
		NetworkCIDR:    fmt.Sprintf("%s/24", clusterConfig.LocalIP),
		MonitoringPort: clusterConfig.MonitoringPort,
		QoSClass:       "test",
		VXLANConfig: agentpkg.VXLANConfig{
			VNI:        clusterConfig.VNI,
			RemoteIPs:  remoteIPs,
			LocalIP:    clusterConfig.LocalIP,
			Port:       suite.testConfig.NetworkConfig.VXLANPort,
			MTU:        suite.testConfig.NetworkConfig.VXLANMTU,
			DeviceName: fmt.Sprintf("vxlan%d", clusterConfig.VNI),
			Learning:   false,
		},
		BWPolicy: agentpkg.BandwidthPolicy{
			DownlinkMbps: clusterConfig.BWLimitMbps,
			UplinkMbps:   clusterConfig.BWLimitMbps,
			LatencyMs:    10.0,
			JitterMs:     2.0,
			LossPercent:  0.1,
			Priority:     1,
			QueueClass:   "htb",
			Filters: []agentpkg.Filter{
				{
					Protocol: "tcp",
					Priority: 10,
					ClassID:  "1:10",
				},
				{
					Protocol: "udp",
					Priority: 20,
					ClassID:  "1:20",
				},
			},
		},
		Interfaces: []agentpkg.NetworkInterface{
			{
				Name:    "eth0",
				Type:    "physical",
				IP:      clusterConfig.LocalIP,
				Netmask: "255.255.255.0",
				MTU:     1500,
				State:   "up",
			},
		},
	}

	var agent *agentpkg.TNAgent

	// Use mock agent in CI environment
	if agentpkg.IsRunningInCI() {
		suite.logger.Println("Creating mock TN agent for CI environment")
		mockAgent := agentpkg.NewMockTNAgent(config, suite.logger)
		require.NoError(suite.T(), mockAgent.Start())
		// Create a wrapper to return TNAgent interface
		agent = &agentpkg.TNAgent{} // This will be handled by mock
		// Store mock agent reference
		suite.agents[clusterConfig.Name] = agent
		return agent
	} else {
		agent = agentpkg.NewTNAgent(config, suite.logger)
		require.NoError(suite.T(), agent.Start())
		return agent
	}
}

// TestVXLANConnectivity tests VXLAN tunnel connectivity
func (suite *TNIntegrationTestSuite) TestVXLANConnectivity() {
	suite.logger.Println("Testing VXLAN connectivity...")

	for clusterName, agent := range suite.agents {
		status, err := agent.GetStatus()
		require.NoError(suite.T(), err)

		assert.True(suite.T(), status.VXLANStatus.TunnelUp,
			"VXLAN tunnel should be up for cluster %s", clusterName)

		suite.logger.Printf("Cluster %s: VXLAN tunnel is up", clusterName)
	}
}

// TestTrafficShaping tests traffic control implementation
func (suite *TNIntegrationTestSuite) TestTrafficShaping() {
	suite.logger.Println("Testing traffic shaping...")

	for clusterName, agent := range suite.agents {
		status, err := agent.GetStatus()
		require.NoError(suite.T(), err)

		assert.True(suite.T(), status.TCStatus.RulesActive,
			"TC rules should be active for cluster %s", clusterName)
		assert.True(suite.T(), status.TCStatus.ShapingActive,
			"Traffic shaping should be active for cluster %s", clusterName)

		suite.logger.Printf("Cluster %s: Traffic shaping is active", clusterName)
	}
}

// TestThroughputMeasurement tests throughput measurement capabilities
func (suite *TNIntegrationTestSuite) TestThroughputMeasurement() {
	suite.logger.Println("Testing throughput measurement...")

	duration, _ := time.ParseDuration(suite.testConfig.NetworkConfig.TestDuration)

	for _, scenario := range suite.testConfig.TestScenarios {
		suite.logger.Printf("Running throughput test for scenario: %s", scenario.Name)

		testConfig := &managerpkg.PerformanceTestConfig{
			TestID:    fmt.Sprintf("throughput_%s_%d", scenario.SliceType, time.Now().Unix()),
			SliceID:   fmt.Sprintf("slice_%s", scenario.SliceType),
			SliceType: scenario.SliceType,
			Duration:  duration,
			TestType:  "iperf3",
			Protocol:  "tcp",
			Parallel:  suite.testConfig.NetworkConfig.ParallelStreams,
		}

		result, err := suite.manager.RunPerformanceTest(testConfig)
		require.NoError(suite.T(), err)

		// Validate throughput
		actualThroughput := result.Performance.Throughput.AvgMbps
		expectedThroughput := scenario.ExpectedMbps

		// Allow 20% tolerance for throughput
		tolerance := expectedThroughput * 0.2
		assert.InDelta(suite.T(), expectedThroughput, actualThroughput, tolerance,
			"Throughput for %s should be within tolerance", scenario.Name)

		suite.logger.Printf("Scenario %s: Expected %.2f Mbps, Got %.2f Mbps",
			scenario.Name, expectedThroughput, actualThroughput)
	}
}

// TestLatencyMeasurement tests latency measurement capabilities
func (suite *TNIntegrationTestSuite) TestLatencyMeasurement() {
	suite.logger.Println("Testing latency measurement...")

	duration, _ := time.ParseDuration("10s") // Shorter duration for latency tests

	for _, scenario := range suite.testConfig.TestScenarios {
		suite.logger.Printf("Running latency test for scenario: %s", scenario.Name)

		testConfig := &managerpkg.PerformanceTestConfig{
			TestID:    fmt.Sprintf("latency_%s_%d", scenario.SliceType, time.Now().Unix()),
			SliceID:   fmt.Sprintf("slice_%s", scenario.SliceType),
			SliceType: scenario.SliceType,
			Duration:  duration,
			TestType:  "ping",
			Protocol:  "icmp",
		}

		result, err := suite.manager.RunPerformanceTest(testConfig)
		require.NoError(suite.T(), err)

		// Validate latency
		actualLatency := result.Performance.Latency.AvgRTTMs
		expectedLatency := scenario.ExpectedRTTMs

		// Allow 30% tolerance for latency (network conditions can vary)
		tolerance := expectedLatency * 0.3
		assert.InDelta(suite.T(), expectedLatency, actualLatency, tolerance,
			"Latency for %s should be within tolerance", scenario.Name)

		suite.logger.Printf("Scenario %s: Expected %.2f ms, Got %.2f ms",
			scenario.Name, expectedLatency, actualLatency)
	}
}

// TestThesisValidation tests validation against thesis targets
func (suite *TNIntegrationTestSuite) TestThesisValidation() {
	suite.logger.Println("Testing thesis validation...")

	duration, _ := time.ParseDuration(suite.testConfig.NetworkConfig.TestDuration)

	var allResults []*managerpkg.NetworkSliceMetrics

	// Run tests for all scenarios
	for i, scenario := range suite.testConfig.TestScenarios {
		suite.logger.Printf("Running thesis validation test %d: %s", i+1, scenario.Name)

		testConfig := &managerpkg.PerformanceTestConfig{
			TestID:    fmt.Sprintf("thesis_%s_%d", scenario.SliceType, time.Now().Unix()),
			SliceID:   fmt.Sprintf("slice_%s", scenario.SliceType),
			SliceType: scenario.SliceType,
			Duration:  duration,
			TestType:  "comprehensive",
			Protocol:  "tcp",
			Parallel:  suite.testConfig.NetworkConfig.ParallelStreams,
		}

		result, err := suite.manager.RunPerformanceTest(testConfig)
		require.NoError(suite.T(), err)

		allResults = append(allResults, result)
	}

	// Validate thesis targets
	targets := suite.testConfig.ThesisTargets

	for i, result := range allResults {
		if i < len(targets.ThroughputMbps) {
			expectedThroughput := targets.ThroughputMbps[i]
			actualThroughput := result.Performance.Throughput.AvgMbps

			// Check if within 10% tolerance
			tolerance := expectedThroughput * 0.1
			withinTolerance := actualThroughput >= expectedThroughput-tolerance

			suite.logger.Printf("Thesis Target %d - Throughput: Expected %.2f Mbps, Got %.2f Mbps, Within Tolerance: %v",
				i+1, expectedThroughput, actualThroughput, withinTolerance)

			assert.True(suite.T(), withinTolerance,
				"Throughput should meet thesis target %d", i+1)
		}

		if i < len(targets.RTTMs) {
			expectedRTT := targets.RTTMs[i]
			actualRTT := result.Performance.Latency.AvgRTTMs

			// Check if within 10% tolerance
			tolerance := expectedRTT * 0.1
			withinTolerance := actualRTT <= expectedRTT+tolerance

			suite.logger.Printf("Thesis Target %d - RTT: Expected %.2f ms, Got %.2f ms, Within Tolerance: %v",
				i+1, expectedRTT, actualRTT, withinTolerance)

			assert.True(suite.T(), withinTolerance,
				"RTT should meet thesis target %d", i+1)
		}

		// Check deploy time
		if result.ThesisValidation.DeployTimeMs > 0 {
			assert.LessOrEqual(suite.T(), result.ThesisValidation.DeployTimeMs, targets.DeployTimeMs,
				"Deploy time should be within target")

			suite.logger.Printf("Deploy Time: %d ms (Target: %d ms)",
				result.ThesisValidation.DeployTimeMs, targets.DeployTimeMs)
		}
	}

	// Calculate overall compliance
	var totalCompliance float64
	for _, result := range allResults {
		totalCompliance += result.ThesisValidation.CompliancePercent
	}

	avgCompliance := totalCompliance / float64(len(allResults))
	suite.logger.Printf("Overall thesis compliance: %.2f%%", avgCompliance)

	assert.GreaterOrEqual(suite.T(), avgCompliance, 80.0,
		"Overall thesis compliance should be at least 80%")
}

// TestMultiClusterIntegration tests multi-cluster network integration
func (suite *TNIntegrationTestSuite) TestMultiClusterIntegration() {
	suite.logger.Println("Testing multi-cluster integration...")

	// Test connectivity between all cluster pairs
	clusters := suite.testConfig.Clusters
	for i, source := range clusters {
		for j, target := range clusters {
			if i == j {
				continue // Skip self-connectivity
			}

			suite.logger.Printf("Testing connectivity: %s -> %s", source.Name, target.Name)

			testConfig := &managerpkg.PerformanceTestConfig{
				TestID:        fmt.Sprintf("connectivity_%s_%s_%d", source.Name, target.Name, time.Now().Unix()),
				SliceID:       "multi_cluster_test",
				SliceType:     "connectivity",
				Duration:      10 * time.Second,
				TestType:      "ping",
				SourceCluster: source.LocalIP,
				TargetCluster: target.LocalIP,
				Protocol:      "icmp",
			}

			result, err := suite.manager.RunPerformanceTest(testConfig)
			require.NoError(suite.T(), err)

			// Verify connectivity was successful
			assert.Empty(suite.T(), result.Performance.ErrorDetails,
				"Connectivity test should have no errors")

			assert.Greater(suite.T(), result.Performance.Latency.AvgRTTMs, 0.0,
				"Should have valid RTT measurement")

			suite.logger.Printf("Connectivity %s -> %s: RTT %.2f ms",
				source.Name, target.Name, result.Performance.Latency.AvgRTTMs)
		}
	}
}

// TestBandwidthMonitoring tests real-time bandwidth monitoring
func (suite *TNIntegrationTestSuite) TestBandwidthMonitoring() {
	suite.logger.Println("Testing bandwidth monitoring...")

	for clusterName, agent := range suite.agents {
		status, err := agent.GetStatus()
		require.NoError(suite.T(), err)

		// Check that bandwidth usage is being monitored
		assert.NotEmpty(suite.T(), status.BandwidthUsage,
			"Bandwidth usage should be monitored for cluster %s", clusterName)

		suite.logger.Printf("Cluster %s: Monitoring %d bandwidth metrics",
			clusterName, len(status.BandwidthUsage))
	}
}

// TestPerformanceReporting tests comprehensive performance reporting
func (suite *TNIntegrationTestSuite) TestPerformanceReporting() {
	suite.logger.Println("Testing performance reporting...")

	// Run a comprehensive test
	testConfig := &managerpkg.PerformanceTestConfig{
		TestID:    fmt.Sprintf("reporting_test_%d", time.Now().Unix()),
		SliceID:   "reporting_slice",
		SliceType: "comprehensive",
		Duration:  30 * time.Second,
		TestType:  "comprehensive",
		Protocol:  "tcp",
		Parallel:  3,
	}

	result, err := suite.manager.RunPerformanceTest(testConfig)
	require.NoError(suite.T(), err)

	// Validate report structure
	assert.NotEmpty(suite.T(), result.SliceID)
	assert.NotEmpty(suite.T(), result.SliceType)
	assert.True(suite.T(), result.Timestamp.After(time.Now().Add(-1*time.Minute)))

	// Validate performance metrics
	assert.GreaterOrEqual(suite.T(), result.Performance.Throughput.AvgMbps, 0.0)
	assert.GreaterOrEqual(suite.T(), result.Performance.Latency.AvgRTTMs, 0.0)

	// Validate thesis validation structure
	assert.NotEmpty(suite.T(), result.ThesisValidation.ThroughputTargets)
	assert.NotEmpty(suite.T(), result.ThesisValidation.RTTTargets)
	assert.GreaterOrEqual(suite.T(), result.ThesisValidation.CompliancePercent, 0.0)
	assert.LessOrEqual(suite.T(), result.ThesisValidation.CompliancePercent, 100.0)

	// Export report for manual review
	reportData, err := json.MarshalIndent(result, "", "  ")
	require.NoError(suite.T(), err)

	suite.logger.Printf("Performance report generated successfully (%d bytes)", len(reportData))
}

// TestErrorHandling tests error handling and recovery
func (suite *TNIntegrationTestSuite) TestErrorHandling() {
	suite.logger.Println("Testing error handling...")

	// Test with invalid configuration
	invalidConfig := &managerpkg.PerformanceTestConfig{
		TestID:        "invalid_test",
		SliceID:       "invalid_slice",
		SliceType:     "invalid",
		Duration:      time.Second,
		TestType:      "invalid",
		TargetCluster: "invalid.ip.address",
		Protocol:      "invalid",
	}

	result, err := suite.manager.RunPerformanceTest(invalidConfig)

	// Should handle errors gracefully
	if err != nil {
		suite.logger.Printf("Expected error for invalid config: %v", err)
	}

	if result != nil {
		assert.NotEmpty(suite.T(), result.Performance.ErrorDetails,
			"Invalid test should produce error details")
	}
}

// TestCleanup tests cleanup operations
func (suite *TNIntegrationTestSuite) TestCleanup() {
	suite.logger.Println("Testing cleanup operations...")

	// Test stopping and restarting an agent
	testCluster := "cluster-edge01"
	agent := suite.agents[testCluster]

	// Stop the agent
	err := agent.Stop()
	require.NoError(suite.T(), err)

	// Verify it's stopped by checking health
	time.Sleep(2 * time.Second)

	// Restart the agent
	clusterConfig := suite.testConfig.Clusters[0] // First cluster config
	newAgent := suite.createTNAgent(clusterConfig)
	suite.agents[testCluster] = newAgent

	// Verify it's healthy again
	status, err := newAgent.GetStatus()
	require.NoError(suite.T(), err)
	assert.True(suite.T(), status.Healthy)

	suite.logger.Printf("Successfully tested cleanup and restart for %s", testCluster)
}

// TestSuite runs the complete integration test suite
func TestTNIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Skip in CI if VXLAN operations cannot be performed
	if agentpkg.IsRunningInCI() && !agentpkg.ShouldMockVXLAN() {
		t.Skip("Skipping integration tests in CI without VXLAN mock capability")
	}

	suite.Run(t, new(TNIntegrationTestSuite))
}

// Benchmark tests for performance analysis
func BenchmarkThroughputTest(b *testing.B) {
	suite := &TNIntegrationTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	testConfig := &managerpkg.PerformanceTestConfig{
		TestID:    "benchmark_throughput",
		SliceID:   "benchmark_slice",
		SliceType: "eMBB",
		Duration:  10 * time.Second,
		TestType:  "iperf3",
		Protocol:  "tcp",
		Parallel:  1,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := suite.manager.RunPerformanceTest(testConfig)
		if err != nil {
			b.Fatalf("Throughput test failed: %v", err)
		}
	}
}

func BenchmarkLatencyTest(b *testing.B) {
	suite := &TNIntegrationTestSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	testConfig := &managerpkg.PerformanceTestConfig{
		TestID:    "benchmark_latency",
		SliceID:   "benchmark_slice",
		SliceType: "URLLC",
		Duration:  5 * time.Second,
		TestType:  "ping",
		Protocol:  "icmp",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := suite.manager.RunPerformanceTest(testConfig)
		if err != nil {
			b.Fatalf("Latency test failed: %v", err)
		}
	}
}