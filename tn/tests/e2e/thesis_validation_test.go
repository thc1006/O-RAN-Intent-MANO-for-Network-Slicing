package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	agentpkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg"
	managerpkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/pkg"
)

// ThesisValidationSuite provides end-to-end thesis validation testing
type ThesisValidationSuite struct {
	suite.Suite
	ctx             context.Context
	cancel          context.CancelFunc
	logger          *log.Logger
	manager         *managerpkg.TNManager
	agents          map[string]*agentpkg.TNAgent
	testResults     []*managerpkg.NetworkSliceMetrics
	reportDir       string
	startTime       time.Time
}

// ThesisTargets defines the exact thesis targets to validate
var ThesisTargets = struct {
	ThroughputMbps   []float64
	RTTMs            []float64
	DeployTimeMs     int64
	TolerancePercent float64
}{
	ThroughputMbps:   []float64{0.93, 2.77, 4.57}, // DL throughput targets
	RTTMs:            []float64{6.3, 15.7, 16.1},  // Ping RTT targets
	DeployTimeMs:     600000,                       // 10 minutes = 600,000ms
	TolerancePercent: 10.0,                         // 10% tolerance
}

// SliceConfiguration defines network slice configurations for testing
var SliceConfigurations = []struct {
	Name          string
	Type          string
	QoSClass      string
	BWLimitMbps   float64
	LatencyMs     float64
	LossPercent   float64
	Priority      int
	ExpectedIndex int // Index in thesis targets arrays
}{
	{
		Name:          "mMTC_Efficient_Slice",
		Type:          "mMTC",
		QoSClass:      "efficient",
		BWLimitMbps:   1.0,
		LatencyMs:     8.0,
		LossPercent:   0.1,
		Priority:      3,
		ExpectedIndex: 0, // 0.93 Mbps, 6.3 ms
	},
	{
		Name:          "URLLC_LowLatency_Slice",
		Type:          "URLLC",
		QoSClass:      "ultra_low_latency",
		BWLimitMbps:   3.0,
		LatencyMs:     5.0,
		LossPercent:   0.01,
		Priority:      1,
		ExpectedIndex: 1, // 2.77 Mbps, 15.7 ms
	},
	{
		Name:          "eMBB_HighThroughput_Slice",
		Type:          "eMBB",
		QoSClass:      "high_throughput",
		BWLimitMbps:   5.0,
		LatencyMs:     10.0,
		LossPercent:   0.05,
		Priority:      2,
		ExpectedIndex: 2, // 4.57 Mbps, 16.1 ms
	},
}

// SetupSuite initializes the thesis validation test suite
func (suite *ThesisValidationSuite) SetupSuite() {
	suite.startTime = time.Now()
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.logger = log.New(os.Stdout, "[Thesis-Validation] ", log.LstdFlags|log.Lshortfile)
	suite.agents = make(map[string]*agentpkg.TNAgent)
	suite.testResults = make([]*managerpkg.NetworkSliceMetrics, 0)

	// Create report directory
	suite.reportDir = filepath.Join("reports", fmt.Sprintf("thesis_validation_%d", time.Now().Unix()))
	err := os.MkdirAll(suite.reportDir, 0755)
	require.NoError(suite.T(), err)

	suite.logger.Printf("Starting thesis validation suite - Report dir: %s", suite.reportDir)

	// Initialize test environment
	suite.initializeTestEnvironment()
}

// TearDownSuite cleans up and generates final report
func (suite *ThesisValidationSuite) TearDownSuite() {
	totalDuration := time.Since(suite.startTime)
	suite.logger.Printf("Thesis validation completed in %v", totalDuration)

	// Generate comprehensive report
	suite.generateFinalReport(totalDuration)

	// Cleanup resources
	suite.cleanupTestEnvironment()
	suite.cancel()
}

// initializeTestEnvironment sets up the test environment
func (suite *ThesisValidationSuite) initializeTestEnvironment() {
	suite.logger.Println("Initializing thesis validation test environment...")

	// Initialize TN Manager
	managerConfig := &managerpkg.TNConfig{
		ClusterName:    "thesis-validation-manager",
		MonitoringPort: 9000,
	}

	suite.manager = managerpkg.NewTNManager(managerConfig, suite.logger)
	require.NoError(suite.T(), suite.manager.Start())

	// Initialize test clusters
	clusters := []struct {
		Name    string
		IP      string
		Port    int
		VNI     uint32
	}{
		{"edge-cluster-01", "192.168.100.10", 8080, 2001},
		{"edge-cluster-02", "192.168.100.11", 8081, 2002},
		{"regional-cluster", "192.168.100.12", 8082, 2003},
		{"central-cluster", "192.168.100.13", 8083, 2004},
	}

	// Create and register agents
	for _, cluster := range clusters {
		agent := suite.createTestAgent(cluster.Name, cluster.IP, cluster.Port, cluster.VNI, clusters)
		suite.agents[cluster.Name] = agent

		// Register with manager
		endpoint := fmt.Sprintf("http://%s:%d", cluster.IP, cluster.Port)
		err := suite.manager.RegisterAgent(cluster.Name, endpoint)
		require.NoError(suite.T(), err)
	}

	suite.logger.Printf("Test environment initialized with %d clusters", len(suite.agents))
}

// createTestAgent creates a test agent with specific configuration
func (suite *ThesisValidationSuite) createTestAgent(name, ip string, port int, vni uint32, allClusters []struct {
	Name string
	IP   string
	Port int
	VNI  uint32
}) *agentpkg.TNAgent {
	// Build remote IPs (all other clusters)
	var remoteIPs []string
	for _, cluster := range allClusters {
		if cluster.Name != name {
			remoteIPs = append(remoteIPs, cluster.IP)
		}
	}

	config := &agentpkg.TNConfig{
		ClusterName:    name,
		NetworkCIDR:    fmt.Sprintf("%s/24", ip),
		MonitoringPort: port,
		QoSClass:       "thesis_validation",
		VXLANConfig: agentpkg.VXLANConfig{
			VNI:        vni,
			RemoteIPs:  remoteIPs,
			LocalIP:    ip,
			Port:       4789,
			MTU:        1450,
			DeviceName: fmt.Sprintf("vxlan%d", vni),
			Learning:   false,
		},
		BWPolicy: agentpkg.BandwidthPolicy{
			DownlinkMbps: 10.0, // Will be updated per slice
			UplinkMbps:   10.0,
			LatencyMs:    15.0,
			JitterMs:     2.0,
			LossPercent:  0.1,
			Priority:     2,
			QueueClass:   "htb",
		},
		Interfaces: []agentpkg.NetworkInterface{
			{
				Name:    "eth0",
				Type:    "physical",
				IP:      ip,
				Netmask: "255.255.255.0",
				MTU:     1500,
				State:   "up",
			},
		},
	}

	agent := agentpkg.NewTNAgent(config, suite.logger)
	require.NoError(suite.T(), agent.Start())

	return agent
}

// TestThesisThroughputTargets tests the three thesis throughput targets
func (suite *ThesisValidationSuite) TestThesisThroughputTargets() {
	suite.logger.Println("=== Testing Thesis Throughput Targets ===")

	for i, sliceConfig := range SliceConfigurations {
		suite.logger.Printf("Testing slice %d: %s", i+1, sliceConfig.Name)

		// Configure the network slice
		result := suite.runSliceTest(sliceConfig, "throughput")

		// Validate against thesis target
		expectedMbps := ThesisTargets.ThroughputMbps[sliceConfig.ExpectedIndex]
		actualMbps := result.Performance.Throughput.AvgMbps

		tolerance := expectedMbps * ThesisTargets.TolerancePercent / 100
		minAcceptable := expectedMbps - tolerance
		maxAcceptable := expectedMbps + tolerance

		suite.logger.Printf("Slice %s - Throughput: Expected %.2f Mbps, Got %.2f Mbps (Range: %.2f-%.2f)",
			sliceConfig.Name, expectedMbps, actualMbps, minAcceptable, maxAcceptable)

		// Assert with tolerance
		assert.GreaterOrEqual(suite.T(), actualMbps, minAcceptable,
			"Throughput for %s should be at least %.2f Mbps", sliceConfig.Name, minAcceptable)

		// Log detailed results
		suite.logDetailedThroughputResults(sliceConfig, result, expectedMbps)

		// Store result for final report
		suite.testResults = append(suite.testResults, result)
	}
}

// TestThesisRTTTargets tests the three thesis RTT targets
func (suite *ThesisValidationSuite) TestThesisRTTTargets() {
	suite.logger.Println("=== Testing Thesis RTT Targets ===")

	for i, sliceConfig := range SliceConfigurations {
		suite.logger.Printf("Testing RTT for slice %d: %s", i+1, sliceConfig.Name)

		// Run latency-focused test
		result := suite.runSliceTest(sliceConfig, "latency")

		// Validate against thesis target
		expectedRTT := ThesisTargets.RTTMs[sliceConfig.ExpectedIndex]
		actualRTT := result.Performance.Latency.AvgRTTMs

		tolerance := expectedRTT * ThesisTargets.TolerancePercent / 100
		maxAcceptable := expectedRTT + tolerance

		suite.logger.Printf("Slice %s - RTT: Expected %.2f ms, Got %.2f ms (Max Acceptable: %.2f)",
			sliceConfig.Name, expectedRTT, actualRTT, maxAcceptable)

		// Assert RTT is within acceptable range
		assert.LessOrEqual(suite.T(), actualRTT, maxAcceptable,
			"RTT for %s should be at most %.2f ms", sliceConfig.Name, maxAcceptable)

		// Log detailed results
		suite.logDetailedRTTResults(sliceConfig, result, expectedRTT)

		// Update existing result or add new one
		suite.updateTestResult(result)
	}
}

// TestThesisDeploymentTime tests the deployment time target
func (suite *ThesisValidationSuite) TestThesisDeploymentTime() {
	suite.logger.Println("=== Testing Thesis Deployment Time Target ===")

	deployStartTime := time.Now()

	// Deploy all slices and measure total time
	for i, sliceConfig := range SliceConfigurations {
		sliceDeployStart := time.Now()

		suite.logger.Printf("Deploying slice %d: %s", i+1, sliceConfig.Name)

		// Configure slice on all agents
		tnConfig := suite.buildTNConfig(sliceConfig)
		err := suite.manager.ConfigureNetworkSlice(sliceConfig.Name, tnConfig)
		require.NoError(suite.T(), err)

		sliceDeployTime := time.Since(sliceDeployStart)
		suite.logger.Printf("Slice %s deployed in %v", sliceConfig.Name, sliceDeployTime)
	}

	totalDeployTime := time.Since(deployStartTime)
	totalDeployTimeMs := totalDeployTime.Milliseconds()

	suite.logger.Printf("Total deployment time: %v (%d ms)", totalDeployTime, totalDeployTimeMs)
	suite.logger.Printf("Target deployment time: %d ms (%.1f minutes)", ThesisTargets.DeployTimeMs, float64(ThesisTargets.DeployTimeMs)/60000)

	// Assert deployment time meets thesis target
	assert.LessOrEqual(suite.T(), totalDeployTimeMs, ThesisTargets.DeployTimeMs,
		"Total deployment time should be within thesis target of %d ms", ThesisTargets.DeployTimeMs)

	// Log deployment efficiency
	efficiency := float64(ThesisTargets.DeployTimeMs-totalDeployTimeMs) / float64(ThesisTargets.DeployTimeMs) * 100
	if efficiency > 0 {
		suite.logger.Printf("Deployment efficiency: %.1f%% faster than target", efficiency)
	} else {
		suite.logger.Printf("Deployment time exceeded target by %.1f%%", -efficiency)
	}
}

// TestComprehensiveThesisValidation runs comprehensive validation against all thesis targets
func (suite *ThesisValidationSuite) TestComprehensiveThesisValidation() {
	suite.logger.Println("=== Comprehensive Thesis Validation ===")

	// Run comprehensive tests for all slices
	comprehensiveResults := make([]*managerpkg.NetworkSliceMetrics, 0)

	for i, sliceConfig := range SliceConfigurations {
		suite.logger.Printf("Running comprehensive test for slice %d: %s", i+1, sliceConfig.Name)

		result := suite.runSliceTest(sliceConfig, "comprehensive")
		comprehensiveResults = append(comprehensiveResults, result)
	}

	// Analyze overall compliance
	suite.analyzeOverallCompliance(comprehensiveResults)

	// Generate compliance report
	suite.generateComplianceReport(comprehensiveResults)
}

// runSliceTest runs a performance test for a specific slice configuration
func (suite *ThesisValidationSuite) runSliceTest(sliceConfig struct {
	Name          string
	Type          string
	QoSClass      string
	BWLimitMbps   float64
	LatencyMs     float64
	LossPercent   float64
	Priority      int
	ExpectedIndex int
}, testType string) *managerpkg.NetworkSliceMetrics {

	// Configure the slice
	tnConfig := suite.buildTNConfig(sliceConfig)
	err := suite.manager.ConfigureNetworkSlice(sliceConfig.Name, tnConfig)
	require.NoError(suite.T(), err)

	// Wait for configuration to take effect
	time.Sleep(5 * time.Second)

	// Determine test duration based on test type
	var duration time.Duration
	switch testType {
	case "latency":
		duration = 15 * time.Second
	case "throughput":
		duration = 45 * time.Second
	case "comprehensive":
		duration = 60 * time.Second
	default:
		duration = 30 * time.Second
	}

	// Create test configuration
	testConfig := &managerpkg.PerformanceTestConfig{
		TestID:    fmt.Sprintf("thesis_%s_%s_%d", sliceConfig.Type, testType, time.Now().Unix()),
		SliceID:   sliceConfig.Name,
		SliceType: sliceConfig.Type,
		Duration:  duration,
		TestType:  testType,
		Protocol:  "tcp",
		Parallel:  3,
		Interval:  time.Second,
	}

	// Run the test
	result, err := suite.manager.RunPerformanceTest(testConfig)
	require.NoError(suite.T(), err)

	suite.logger.Printf("Test completed for %s: Throughput=%.2f Mbps, RTT=%.2f ms",
		sliceConfig.Name, result.Performance.Throughput.AvgMbps, result.Performance.Latency.AvgRTTMs)

	return result
}

// buildTNConfig builds TN configuration for a slice
func (suite *ThesisValidationSuite) buildTNConfig(sliceConfig struct {
	Name          string
	Type          string
	QoSClass      string
	BWLimitMbps   float64
	LatencyMs     float64
	LossPercent   float64
	Priority      int
	ExpectedIndex int
}) *managerpkg.TNConfig {

	return &managerpkg.TNConfig{
		ClusterName: sliceConfig.Name,
		QoSClass:    sliceConfig.QoSClass,
		BWPolicy: managerpkg.BandwidthPolicy{
			DownlinkMbps: sliceConfig.BWLimitMbps,
			UplinkMbps:   sliceConfig.BWLimitMbps,
			LatencyMs:    sliceConfig.LatencyMs,
			LossPercent:  sliceConfig.LossPercent,
			Priority:     sliceConfig.Priority,
			QueueClass:   "htb",
			Filters: []managerpkg.Filter{
				{
					Protocol: "tcp",
					SrcIP:    "",
					DstIP:    "",
					SrcPort:  0,
					DstPort:  0,
				},
			},
		},
	}
}

// logDetailedThroughputResults logs detailed throughput analysis
func (suite *ThesisValidationSuite) logDetailedThroughputResults(sliceConfig struct {
	Name          string
	Type          string
	QoSClass      string
	BWLimitMbps   float64
	LatencyMs     float64
	LossPercent   float64
	Priority      int
	ExpectedIndex int
}, result *managerpkg.NetworkSliceMetrics, expectedMbps float64) {

	throughput := result.Performance.Throughput

	suite.logger.Printf("=== Detailed Throughput Analysis for %s ===", sliceConfig.Name)
	suite.logger.Printf("  Target: %.2f Mbps", expectedMbps)
	suite.logger.Printf("  Achieved: %.2f Mbps", throughput.AvgMbps)
	suite.logger.Printf("  Peak: %.2f Mbps", throughput.PeakMbps)
	suite.logger.Printf("  Min: %.2f Mbps", throughput.MinMbps)
	suite.logger.Printf("  Downlink: %.2f Mbps", throughput.DownlinkMbps)
	suite.logger.Printf("  Uplink: %.2f Mbps", throughput.UplinkMbps)

	if expectedMbps > 0 {
		achievementRatio := throughput.AvgMbps / expectedMbps * 100
		suite.logger.Printf("  Achievement Ratio: %.1f%%", achievementRatio)
	}

	// Check for VXLAN and TC overhead
	suite.logger.Printf("  VXLAN Overhead: %.2f%%", result.Performance.VXLANOverhead)
	suite.logger.Printf("  TC Overhead: %.2f%%", result.Performance.TCOverhead)
}

// logDetailedRTTResults logs detailed RTT analysis
func (suite *ThesisValidationSuite) logDetailedRTTResults(sliceConfig struct {
	Name          string
	Type          string
	QoSClass      string
	BWLimitMbps   float64
	LatencyMs     float64
	LossPercent   float64
	Priority      int
	ExpectedIndex int
}, result *managerpkg.NetworkSliceMetrics, expectedRTT float64) {

	latency := result.Performance.Latency

	suite.logger.Printf("=== Detailed RTT Analysis for %s ===", sliceConfig.Name)
	suite.logger.Printf("  Target: %.2f ms", expectedRTT)
	suite.logger.Printf("  Achieved: %.2f ms", latency.AvgRTTMs)
	suite.logger.Printf("  Min RTT: %.2f ms", latency.MinRTTMs)
	suite.logger.Printf("  Max RTT: %.2f ms", latency.MaxRTTMs)
	suite.logger.Printf("  Std Dev: %.2f ms", latency.StdDevMs)
	suite.logger.Printf("  P50: %.2f ms", latency.P50Ms)
	suite.logger.Printf("  P95: %.2f ms", latency.P95Ms)
	suite.logger.Printf("  P99: %.2f ms", latency.P99Ms)

	if expectedRTT > 0 {
		performanceRatio := expectedRTT / latency.AvgRTTMs * 100
		suite.logger.Printf("  Performance Ratio: %.1f%%", performanceRatio)
	}
}

// updateTestResult updates an existing test result or adds a new one
func (suite *ThesisValidationSuite) updateTestResult(newResult *managerpkg.NetworkSliceMetrics) {
	// Find existing result with same slice ID
	for i, existing := range suite.testResults {
		if existing.SliceID == newResult.SliceID {
			// Update existing result
			suite.testResults[i] = newResult
			return
		}
	}

	// Add new result
	suite.testResults = append(suite.testResults, newResult)
}

// analyzeOverallCompliance analyzes overall thesis compliance
func (suite *ThesisValidationSuite) analyzeOverallCompliance(results []*managerpkg.NetworkSliceMetrics) {
	suite.logger.Println("=== Overall Thesis Compliance Analysis ===")

	var totalCompliancePercent float64
	var compliantSlices int
	var totalSlices = len(results)

	for i, result := range results {
		compliancePercent := result.ThesisValidation.CompliancePercent
		totalCompliancePercent += compliancePercent

		if compliancePercent >= 80.0 { // Consider 80%+ as compliant
			compliantSlices++
		}

		suite.logger.Printf("Slice %d (%s): %.1f%% compliant", i+1, result.SliceType, compliancePercent)
	}

	avgCompliance := totalCompliancePercent / float64(totalSlices)
	sliceComplianceRate := float64(compliantSlices) / float64(totalSlices) * 100

	suite.logger.Printf("Average Compliance: %.1f%%", avgCompliance)
	suite.logger.Printf("Slice Compliance Rate: %.1f%% (%d/%d slices)", sliceComplianceRate, compliantSlices, totalSlices)

	// Overall assessment
	if avgCompliance >= 90 {
		suite.logger.Println("ASSESSMENT: Excellent thesis compliance")
	} else if avgCompliance >= 80 {
		suite.logger.Println("ASSESSMENT: Good thesis compliance")
	} else if avgCompliance >= 70 {
		suite.logger.Println("ASSESSMENT: Acceptable thesis compliance with room for improvement")
	} else {
		suite.logger.Println("ASSESSMENT: Poor thesis compliance - significant improvements needed")
	}

	// Assert overall compliance
	assert.GreaterOrEqual(suite.T(), avgCompliance, 70.0,
		"Overall thesis compliance should be at least 70%%")
}

// generateComplianceReport generates a detailed compliance report
func (suite *ThesisValidationSuite) generateComplianceReport(results []*managerpkg.NetworkSliceMetrics) {
	reportFile := filepath.Join(suite.reportDir, "thesis_compliance_report.json")

	report := map[string]interface{}{
		"timestamp":         time.Now(),
		"thesis_targets": map[string]interface{}{
			"throughput_mbps": ThesisTargets.ThroughputMbps,
			"rtt_ms":          ThesisTargets.RTTMs,
			"deploy_time_ms":  ThesisTargets.DeployTimeMs,
			"tolerance_percent": ThesisTargets.TolerancePercent,
		},
		"test_results":      results,
		"slice_configurations": SliceConfigurations,
		"summary": map[string]interface{}{
			"total_slices":    len(results),
			"test_duration":   time.Since(suite.startTime).String(),
		},
	}

	// Calculate summary metrics
	var totalCompliance, totalThroughput, totalRTT float64
	var compliantSlices int

	for _, result := range results {
		totalCompliance += result.ThesisValidation.CompliancePercent
		totalThroughput += result.Performance.Throughput.AvgMbps
		totalRTT += result.Performance.Latency.AvgRTTMs

		if result.ThesisValidation.CompliancePercent >= 80.0 {
			compliantSlices++
		}
	}

	summary := report["summary"].(map[string]interface{})
	summary["avg_compliance_percent"] = totalCompliance / float64(len(results))
	summary["avg_throughput_mbps"] = totalThroughput / float64(len(results))
	summary["avg_rtt_ms"] = totalRTT / float64(len(results))
	summary["compliant_slices"] = compliantSlices
	summary["compliance_rate_percent"] = float64(compliantSlices) / float64(len(results)) * 100

	// Write report to file
	reportData, err := json.MarshalIndent(report, "", "  ")
	require.NoError(suite.T(), err)

	err = os.WriteFile(reportFile, reportData, 0644)
	require.NoError(suite.T(), err)

	suite.logger.Printf("Compliance report saved to: %s", reportFile)
}

// generateFinalReport generates the final comprehensive report
func (suite *ThesisValidationSuite) generateFinalReport(totalDuration time.Duration) {
	suite.logger.Println("Generating final thesis validation report...")

	finalReportFile := filepath.Join(suite.reportDir, "final_thesis_validation_report.json")

	// Collect all test data
	finalReport := map[string]interface{}{
		"report_metadata": map[string]interface{}{
			"generated_at":    time.Now(),
			"total_duration":  totalDuration.String(),
			"start_time":      suite.startTime,
			"end_time":        time.Now(),
			"report_version":  "1.0",
		},
		"thesis_validation": map[string]interface{}{
			"targets": map[string]interface{}{
				"throughput_targets_mbps": ThesisTargets.ThroughputMbps,
				"rtt_targets_ms":         ThesisTargets.RTTMs,
				"deploy_time_target_ms":  ThesisTargets.DeployTimeMs,
				"tolerance_percent":      ThesisTargets.TolerancePercent,
			},
			"results":  suite.testResults,
			"summary":  suite.generateExecutiveSummary(),
		},
		"test_environment": map[string]interface{}{
			"clusters":          len(suite.agents),
			"slice_configurations": SliceConfigurations,
		},
		"performance_analysis": suite.generatePerformanceAnalysis(),
		"recommendations":      suite.generateRecommendations(),
	}

	// Write final report
	reportData, err := json.MarshalIndent(finalReport, "", "  ")
	require.NoError(suite.T(), err)

	err = os.WriteFile(finalReportFile, reportData, 0644)
	require.NoError(suite.T(), err)

	suite.logger.Printf("Final thesis validation report saved to: %s", finalReportFile)
	suite.logger.Printf("Total report size: %d bytes", len(reportData))
}

// generateExecutiveSummary generates executive summary
func (suite *ThesisValidationSuite) generateExecutiveSummary() map[string]interface{} {
	if len(suite.testResults) == 0 {
		return map[string]interface{}{"status": "no_test_results"}
	}

	var totalCompliance, totalThroughput, totalRTT float64
	var compliantSlices, throughputAchievedTargets, rttAchievedTargets int

	for i, result := range suite.testResults {
		totalCompliance += result.ThesisValidation.CompliancePercent
		totalThroughput += result.Performance.Throughput.AvgMbps
		totalRTT += result.Performance.Latency.AvgRTTMs

		if result.ThesisValidation.CompliancePercent >= 80.0 {
			compliantSlices++
		}

		// Check if individual targets were achieved
		if i < len(ThesisTargets.ThroughputMbps) {
			target := ThesisTargets.ThroughputMbps[i]
			tolerance := target * ThesisTargets.TolerancePercent / 100
			if result.Performance.Throughput.AvgMbps >= target-tolerance {
				throughputAchievedTargets++
			}
		}

		if i < len(ThesisTargets.RTTMs) {
			target := ThesisTargets.RTTMs[i]
			tolerance := target * ThesisTargets.TolerancePercent / 100
			if result.Performance.Latency.AvgRTTMs <= target+tolerance {
				rttAchievedTargets++
			}
		}
	}

	numResults := float64(len(suite.testResults))

	return map[string]interface{}{
		"overall_compliance_percent":    totalCompliance / numResults,
		"compliant_slices":             compliantSlices,
		"total_slices":                 len(suite.testResults),
		"slice_compliance_rate_percent": float64(compliantSlices) / numResults * 100,
		"avg_throughput_mbps":          totalThroughput / numResults,
		"avg_rtt_ms":                   totalRTT / numResults,
		"throughput_targets_achieved":   throughputAchievedTargets,
		"rtt_targets_achieved":         rttAchievedTargets,
		"throughput_success_rate_percent": float64(throughputAchievedTargets) / numResults * 100,
		"rtt_success_rate_percent":     float64(rttAchievedTargets) / numResults * 100,
		"status": suite.determineOverallStatus(totalCompliance/numResults),
	}
}

// determineOverallStatus determines overall test status
func (suite *ThesisValidationSuite) determineOverallStatus(avgCompliance float64) string {
	if avgCompliance >= 90 {
		return "excellent"
	} else if avgCompliance >= 80 {
		return "good"
	} else if avgCompliance >= 70 {
		return "acceptable"
	} else {
		return "needs_improvement"
	}
}

// generatePerformanceAnalysis generates detailed performance analysis
func (suite *ThesisValidationSuite) generatePerformanceAnalysis() map[string]interface{} {
	analysis := map[string]interface{}{
		"throughput_analysis": make(map[string]interface{}),
		"latency_analysis":    make(map[string]interface{}),
		"overhead_analysis":   make(map[string]interface{}),
	}

	if len(suite.testResults) == 0 {
		return analysis
	}

	// Throughput analysis
	var throughputs []float64
	var latencies []float64
	var vxlanOverheads []float64
	var tcOverheads []float64

	for _, result := range suite.testResults {
		throughputs = append(throughputs, result.Performance.Throughput.AvgMbps)
		latencies = append(latencies, result.Performance.Latency.AvgRTTMs)
		vxlanOverheads = append(vxlanOverheads, result.Performance.VXLANOverhead)
		tcOverheads = append(tcOverheads, result.Performance.TCOverhead)
	}

	analysis["throughput_analysis"] = map[string]interface{}{
		"min_mbps":    minFloat64(throughputs),
		"max_mbps":    maxFloat64(throughputs),
		"avg_mbps":    avgFloat64(throughputs),
		"std_dev_mbps": stdDevFloat64(throughputs),
	}

	analysis["latency_analysis"] = map[string]interface{}{
		"min_rtt_ms":    minFloat64(latencies),
		"max_rtt_ms":    maxFloat64(latencies),
		"avg_rtt_ms":    avgFloat64(latencies),
		"std_dev_rtt_ms": stdDevFloat64(latencies),
	}

	analysis["overhead_analysis"] = map[string]interface{}{
		"avg_vxlan_overhead_percent": avgFloat64(vxlanOverheads),
		"avg_tc_overhead_percent":    avgFloat64(tcOverheads),
	}

	return analysis
}

// generateRecommendations generates recommendations based on test results
func (suite *ThesisValidationSuite) generateRecommendations() []string {
	recommendations := make([]string, 0)

	if len(suite.testResults) == 0 {
		recommendations = append(recommendations, "No test results available for analysis")
		return recommendations
	}

	// Analyze results and generate recommendations
	var totalCompliance, avgThroughput, avgLatency float64
	var lowPerformingSlices []string

	for _, result := range suite.testResults {
		totalCompliance += result.ThesisValidation.CompliancePercent
		avgThroughput += result.Performance.Throughput.AvgMbps
		avgLatency += result.Performance.Latency.AvgRTTMs

		if result.ThesisValidation.CompliancePercent < 70 {
			lowPerformingSlices = append(lowPerformingSlices, result.SliceID)
		}
	}

	numResults := float64(len(suite.testResults))
	totalCompliance /= numResults
	avgThroughput /= numResults
	avgLatency /= numResults

	// Generate specific recommendations
	if totalCompliance < 80 {
		recommendations = append(recommendations, "Overall compliance below 80% - review system configuration and optimization")
	}

	if avgLatency > 20 {
		recommendations = append(recommendations, "High average latency detected - optimize network routing and reduce processing delays")
	}

	if avgThroughput < 2.0 {
		recommendations = append(recommendations, "Low average throughput - review bandwidth allocation and traffic shaping policies")
	}

	if len(lowPerformingSlices) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Low performing slices identified: %v - investigate specific configurations", lowPerformingSlices))
	}

	// Add positive recommendations if performance is good
	if totalCompliance >= 90 {
		recommendations = append(recommendations, "Excellent thesis compliance achieved - consider documenting best practices")
	}

	if avgLatency < 10 {
		recommendations = append(recommendations, "Excellent latency performance - current configuration optimized for low-latency applications")
	}

	return recommendations
}

// cleanupTestEnvironment cleans up test resources
func (suite *ThesisValidationSuite) cleanupTestEnvironment() {
	suite.logger.Println("Cleaning up test environment...")

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

	suite.logger.Println("Test environment cleanup completed")
}

// Helper functions for statistical analysis
func minFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func avgFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDevFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	avg := avgFloat64(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - avg) * (v - avg)
	}
	return sum / float64(len(values))
}

// TestSuite runner
func TestThesisValidationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping thesis validation tests in short mode")
	}

	suite.Run(t, new(ThesisValidationSuite))
}