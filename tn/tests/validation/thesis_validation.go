package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	managerPkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/pkg"
)

// ThesisValidator validates the TN module against thesis requirements
type ThesisValidator struct {
	manager *managerPkg.TNManager
	logger  *log.Logger
	results *ValidationResults
}

// ValidationResults contains comprehensive validation results
type ValidationResults struct {
	TestSuiteID       string                    `json:"testSuiteId"`
	Timestamp         time.Time                 `json:"timestamp"`
	OverallCompliance float64                   `json:"overallCompliance"`
	SliceResults      []SliceValidationResult   `json:"sliceResults"`
	PerformanceTests  []PerformanceTestResult   `json:"performanceTests"`
	ThesisTargets     ThesisTargets             `json:"thesisTargets"`
	Summary           ValidationSummary         `json:"summary"`
	Issues            []ValidationIssue         `json:"issues"`
}

// SliceValidationResult contains results for a specific slice type
type SliceValidationResult struct {
	SliceType         string                  `json:"sliceType"`
	TargetThroughput  float64                 `json:"targetThroughput"`
	ActualThroughput  float64                 `json:"actualThroughput"`
	TargetLatency     float64                 `json:"targetLatency"`
	ActualLatency     float64                 `json:"actualLatency"`
	ThroughputMet     bool                    `json:"throughputMet"`
	LatencyMet        bool                    `json:"latencyMet"`
	CompliancePercent float64                 `json:"compliancePercent"`
	DeployTime        time.Duration           `json:"deployTime"`
	TestResults       []PerformanceTestResult `json:"testResults"`
}

// PerformanceTestResult contains individual test results
type PerformanceTestResult struct {
	TestID        string        `json:"testId"`
	TestType      string        `json:"testType"`
	SliceType     string        `json:"sliceType"`
	Duration      time.Duration `json:"duration"`
	Success       bool          `json:"success"`
	ThroughputMbps float64      `json:"throughputMbps"`
	LatencyMs     float64       `json:"latencyMs"`
	PacketLoss    float64       `json:"packetLoss"`
	Jitter        float64       `json:"jitter"`
	TCOverhead    float64       `json:"tcOverhead"`
	VXLANOverhead float64       `json:"vxlanOverhead"`
	Timestamp     time.Time     `json:"timestamp"`
	ErrorMessage  string        `json:"errorMessage,omitempty"`
}

// ThesisTargets defines the thesis performance targets
type ThesisTargets struct {
	URllcThroughput float64       `json:"urllcThroughput"` // 0.93 Mbps
	URllcLatency    float64       `json:"urllcLatency"`    // 6.3 ms
	MIoTThroughput  float64       `json:"miotThroughput"`  // 2.77 Mbps
	MIoTLatency     float64       `json:"miotLatency"`     // 15.7 ms
	EMBBThroughput  float64       `json:"embbThroughput"`  // 4.57 Mbps
	EMBBLatency     float64       `json:"embbLatency"`     // 16.1 ms
	DeployTimeMax   time.Duration `json:"deployTimeMax"`   // 10 minutes
	Tolerance       float64       `json:"tolerance"`       // 10%
}

// ValidationSummary provides summary statistics
type ValidationSummary struct {
	TotalTests        int     `json:"totalTests"`
	PassedTests       int     `json:"passedTests"`
	FailedTests       int     `json:"failedTests"`
	OverallCompliance float64 `json:"overallCompliance"`
	AvgDeployTime     float64 `json:"avgDeployTimeMs"`
	AvgThroughput     float64 `json:"avgThroughputMbps"`
	AvgLatency        float64 `json:"avgLatencyMs"`
	MaxOverhead       float64 `json:"maxOverheadPercent"`
}

// ValidationIssue represents a validation issue
type ValidationIssue struct {
	Severity    string    `json:"severity"`
	Component   string    `json:"component"`
	SliceType   string    `json:"sliceType,omitempty"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Suggestion  string    `json:"suggestion"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewThesisValidator creates a new thesis validator
func NewThesisValidator(manager *managerPkg.TNManager, logger *log.Logger) *ThesisValidator {
	return &ThesisValidator{
		manager: manager,
		logger:  logger,
		results: &ValidationResults{
			TestSuiteID: fmt.Sprintf("thesis_validation_%d", time.Now().Unix()),
			Timestamp:   time.Now(),
			ThesisTargets: ThesisTargets{
				URllcThroughput: 0.93,
				URllcLatency:    6.3,
				MIoTThroughput:  2.77,
				MIoTLatency:     15.7,
				EMBBThroughput:  4.57,
				EMBBLatency:     16.1,
				DeployTimeMax:   10 * time.Minute,
				Tolerance:       0.10, // 10%
			},
			SliceResults:     make([]SliceValidationResult, 0),
			PerformanceTests: make([]PerformanceTestResult, 0),
			Issues:           make([]ValidationIssue, 0),
		},
	}
}

// RunCompleteValidation runs comprehensive thesis validation
func (tv *ThesisValidator) RunCompleteValidation(ctx context.Context) (*ValidationResults, error) {
	tv.logger.Println("Starting comprehensive thesis validation")

	// Test each slice type
	sliceTypes := []struct {
		name           string
		throughputMbps float64
		latencyMs      float64
	}{
		{"URLLC", tv.results.ThesisTargets.URllcThroughput, tv.results.ThesisTargets.URllcLatency},
		{"mIoT", tv.results.ThesisTargets.MIoTThroughput, tv.results.ThesisTargets.MIoTLatency},
		{"eMBB", tv.results.ThesisTargets.EMBBThroughput, tv.results.ThesisTargets.EMBBLatency},
	}

	for _, sliceType := range sliceTypes {
		result, err := tv.validateSliceType(ctx, sliceType.name, sliceType.throughputMbps, sliceType.latencyMs)
		if err != nil {
			tv.addIssue("error", "validation", sliceType.name,
				fmt.Sprintf("Failed to validate slice type %s: %v", sliceType.name, err),
				"Cannot validate slice performance", "Check network configuration and retry")
			continue
		}

		tv.results.SliceResults = append(tv.results.SliceResults, *result)
	}

	// Run comprehensive multi-slice test
	err := tv.runMultiSliceTest(ctx)
	if err != nil {
		tv.addIssue("warning", "multi-slice", "",
			fmt.Sprintf("Multi-slice test issues: %v", err),
			"May affect concurrent slice performance", "Review resource allocation")
	}

	// Calculate overall results
	tv.calculateOverallResults()

	// Generate recommendations
	tv.generateRecommendations()

	tv.logger.Printf("Validation completed. Overall compliance: %.2f%%", tv.results.OverallCompliance)

	return tv.results, nil
}

// validateSliceType validates a specific slice type
func (tv *ThesisValidator) validateSliceType(ctx context.Context, sliceType string, targetThroughput, targetLatency float64) (*SliceValidationResult, error) {
	tv.logger.Printf("Validating slice type: %s", sliceType)

	result := &SliceValidationResult{
		SliceType:        sliceType,
		TargetThroughput: targetThroughput,
		TargetLatency:    targetLatency,
		TestResults:      make([]PerformanceTestResult, 0),
	}

	startTime := time.Now()

	// Run multiple test iterations for statistical validity
	testIterations := 5
	var throughputResults []float64
	var latencyResults []float64

	for i := 0; i < testIterations; i++ {
		testConfig := &managerPkg.PerformanceTestConfig{
			TestID:    fmt.Sprintf("thesis_%s_%d_%d", sliceType, time.Now().Unix(), i),
			SliceID:   fmt.Sprintf("%s-thesis-slice", sliceType),
			SliceType: sliceType,
			Duration:  30 * time.Second,
			TestType:  "comprehensive",
			Protocol:  "tcp",
			Parallel:  1,
			Interval:  time.Second,
		}

		metrics, err := tv.manager.RunPerformanceTest(testConfig)
		if err != nil {
			testResult := PerformanceTestResult{
				TestID:       testConfig.TestID,
				TestType:     testConfig.TestType,
				SliceType:    sliceType,
				Duration:     time.Since(startTime),
				Success:      false,
				Timestamp:    time.Now(),
				ErrorMessage: err.Error(),
			}
			result.TestResults = append(result.TestResults, testResult)
			continue
		}

		testResult := PerformanceTestResult{
			TestID:        testConfig.TestID,
			TestType:      testConfig.TestType,
			SliceType:     sliceType,
			Duration:      testConfig.Duration,
			Success:       true,
			ThroughputMbps: metrics.Performance.Throughput.AvgMbps,
			LatencyMs:     metrics.Performance.Latency.AvgRTTMs,
			PacketLoss:    metrics.Performance.PacketLoss,
			Jitter:        metrics.Performance.Jitter,
			TCOverhead:    metrics.Performance.TCOverhead,
			VXLANOverhead: metrics.Performance.VXLANOverhead,
			Timestamp:     time.Now(),
		}

		result.TestResults = append(result.TestResults, testResult)
		tv.results.PerformanceTests = append(tv.results.PerformanceTests, testResult)

		throughputResults = append(throughputResults, testResult.ThroughputMbps)
		latencyResults = append(latencyResults, testResult.LatencyMs)
	}

	result.DeployTime = time.Since(startTime)

	// Calculate average results
	if len(throughputResults) > 0 {
		result.ActualThroughput = average(throughputResults)
		result.ActualLatency = average(latencyResults)

		// Check compliance with 10% tolerance
		throughputTolerance := targetThroughput * tv.results.ThesisTargets.Tolerance
		latencyTolerance := targetLatency * tv.results.ThesisTargets.Tolerance

		result.ThroughputMet = result.ActualThroughput >= (targetThroughput - throughputTolerance)
		result.LatencyMet = result.ActualLatency <= (targetLatency + latencyTolerance)

		// Calculate compliance percentage
		complianceTests := 0
		totalTests := 2 // throughput + latency

		if result.ThroughputMet {
			complianceTests++
		}
		if result.LatencyMet {
			complianceTests++
		}

		// Add deployment time check
		totalTests++
		if result.DeployTime <= tv.results.ThesisTargets.DeployTimeMax {
			complianceTests++
		}

		result.CompliancePercent = float64(complianceTests) / float64(totalTests) * 100
	}

	// Add specific slice type validations
	tv.validateSliceSpecificRequirements(result)

	return result, nil
}

// validateSliceSpecificRequirements validates slice-specific requirements
func (tv *ThesisValidator) validateSliceSpecificRequirements(result *SliceValidationResult) {
	switch result.SliceType {
	case "URLLC":
		// URLLC requires ultra-low latency and high reliability
		if result.ActualLatency > 10.0 {
			tv.addIssue("critical", "latency", "URLLC",
				fmt.Sprintf("URLLC latency %.2f ms exceeds acceptable range", result.ActualLatency),
				"Critical applications may fail", "Optimize network path and reduce processing overhead")
		}

		// Check packet loss for reliability
		for _, test := range result.TestResults {
			if test.PacketLoss > 0.001 { // 0.001% max for URLLC
				tv.addIssue("high", "reliability", "URLLC",
					fmt.Sprintf("URLLC packet loss %.4f%% exceeds 0.001%% target", test.PacketLoss),
					"Reliability requirements not met", "Check network stability and error correction")
			}
		}

	case "eMBB":
		// eMBB requires high throughput
		if result.ActualThroughput < 4.0 {
			tv.addIssue("high", "throughput", "eMBB",
				fmt.Sprintf("eMBB throughput %.2f Mbps below expected range", result.ActualThroughput),
				"High-bandwidth applications may be affected", "Increase bandwidth allocation or optimize traffic shaping")
		}

		// Check for consistent high performance
		throughputVariation := tv.calculateVariation(result.TestResults, "throughput")
		if throughputVariation > 20.0 { // 20% variation threshold
			tv.addIssue("medium", "consistency", "eMBB",
				fmt.Sprintf("eMBB throughput variation %.1f%% exceeds 20%% threshold", throughputVariation),
				"Inconsistent user experience", "Stabilize network conditions and traffic control")
		}

	case "mIoT":
		// mIoT requires balanced performance and efficiency
		if result.ActualLatency > 20.0 {
			tv.addIssue("medium", "latency", "mIoT",
				fmt.Sprintf("mIoT latency %.2f ms higher than optimal", result.ActualLatency),
				"IoT device responsiveness affected", "Optimize routing and reduce network hops")
		}

		// Check energy efficiency (approximated by overhead)
		for _, test := range result.TestResults {
			totalOverhead := test.TCOverhead + test.VXLANOverhead
			if totalOverhead > 15.0 { // 15% overhead threshold
				tv.addIssue("low", "efficiency", "mIoT",
					fmt.Sprintf("mIoT total overhead %.1f%% may impact battery life", totalOverhead),
					"Reduced device battery life", "Optimize protocol efficiency")
			}
		}
	}
}

// runMultiSliceTest runs concurrent slice tests
func (tv *ThesisValidator) runMultiSliceTest(ctx context.Context) error {
	tv.logger.Println("Running multi-slice concurrent test")

	// Simulate concurrent slice deployment and testing
	testConfig := &managerPkg.PerformanceTestConfig{
		TestID:    fmt.Sprintf("multi_slice_%d", time.Now().Unix()),
		SliceID:   "multi-slice-test",
		SliceType: "mixed",
		Duration:  60 * time.Second,
		TestType:  "concurrent",
		Protocol:  "tcp",
		Parallel:  3,
	}

	metrics, err := tv.manager.RunPerformanceTest(testConfig)
	if err != nil {
		return fmt.Errorf("multi-slice test failed: %w", err)
	}

	testResult := PerformanceTestResult{
		TestID:        testConfig.TestID,
		TestType:      "multi-slice",
		SliceType:     "mixed",
		Duration:      testConfig.Duration,
		Success:       true,
		ThroughputMbps: metrics.Performance.Throughput.AvgMbps,
		LatencyMs:     metrics.Performance.Latency.AvgRTTMs,
		PacketLoss:    metrics.Performance.PacketLoss,
		Timestamp:     time.Now(),
	}

	tv.results.PerformanceTests = append(tv.results.PerformanceTests, testResult)

	return nil
}

// calculateOverallResults calculates overall validation results
func (tv *ThesisValidator) calculateOverallResults() {
	tv.logger.Println("Calculating overall validation results")

	summary := &tv.results.Summary
	summary.TotalTests = len(tv.results.PerformanceTests)

	var totalCompliance float64
	var totalDeployTime float64
	var totalThroughput float64
	var totalLatency float64
	var maxOverhead float64

	for _, result := range tv.results.SliceResults {
		totalCompliance += result.CompliancePercent
		totalDeployTime += result.DeployTime.Seconds() * 1000 // Convert to ms

		for _, test := range result.TestResults {
			if test.Success {
				summary.PassedTests++
				totalThroughput += test.ThroughputMbps
				totalLatency += test.LatencyMs

				totalTestOverhead := test.TCOverhead + test.VXLANOverhead
				if totalTestOverhead > maxOverhead {
					maxOverhead = totalTestOverhead
				}
			} else {
				summary.FailedTests++
			}
		}
	}

	if len(tv.results.SliceResults) > 0 {
		tv.results.OverallCompliance = totalCompliance / float64(len(tv.results.SliceResults))
		summary.OverallCompliance = tv.results.OverallCompliance
		summary.AvgDeployTime = totalDeployTime / float64(len(tv.results.SliceResults))
	}

	if summary.PassedTests > 0 {
		summary.AvgThroughput = totalThroughput / float64(summary.PassedTests)
		summary.AvgLatency = totalLatency / float64(summary.PassedTests)
	}

	summary.MaxOverhead = maxOverhead
}

// generateRecommendations generates recommendations based on results
func (tv *ThesisValidator) generateRecommendations() {
	tv.logger.Println("Generating recommendations")

	if tv.results.OverallCompliance < 80.0 {
		tv.addIssue("critical", "overall", "",
			fmt.Sprintf("Overall compliance %.1f%% below 80%% threshold", tv.results.OverallCompliance),
			"System does not meet thesis requirements",
			"Review network configuration, resource allocation, and performance tuning")
	}

	if tv.results.Summary.AvgDeployTime > 600000 { // 10 minutes in ms
		tv.addIssue("high", "deployment", "",
			fmt.Sprintf("Average deployment time %.1f minutes exceeds 10-minute target", tv.results.Summary.AvgDeployTime/60000),
			"Deployment efficiency below requirements",
			"Optimize deployment automation and reduce provisioning overhead")
	}

	if tv.results.Summary.MaxOverhead > 20.0 {
		tv.addIssue("medium", "overhead", "",
			fmt.Sprintf("Maximum overhead %.1f%% may impact efficiency", tv.results.Summary.MaxOverhead),
			"Network efficiency could be improved",
			"Optimize protocol stack and reduce encapsulation overhead")
	}

	// Generate slice-specific recommendations
	for _, sliceResult := range tv.results.SliceResults {
		if sliceResult.CompliancePercent < 90.0 {
			tv.addIssue("medium", "slice-compliance", sliceResult.SliceType,
				fmt.Sprintf("%s slice compliance %.1f%% below 90%% target", sliceResult.SliceType, sliceResult.CompliancePercent),
				"Slice performance may not meet user expectations",
				fmt.Sprintf("Tune %s slice configuration for better performance", sliceResult.SliceType))
		}
	}
}

// ExportResults exports validation results to JSON file
func (tv *ThesisValidator) ExportResults(filename string) error {
	data, err := json.MarshalIndent(tv.results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	tv.logger.Printf("Validation results exported to %s", filename)
	return nil
}

// Helper methods

func (tv *ThesisValidator) addIssue(severity, component, sliceType, description, impact, suggestion string) {
	issue := ValidationIssue{
		Severity:    severity,
		Component:   component,
		SliceType:   sliceType,
		Description: description,
		Impact:      impact,
		Suggestion:  suggestion,
		Timestamp:   time.Now(),
	}

	tv.results.Issues = append(tv.results.Issues, issue)
	tv.logger.Printf("[%s] %s: %s", severity, component, description)
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (tv *ThesisValidator) calculateVariation(results []PerformanceTestResult, metric string) float64 {
	if len(results) < 2 {
		return 0
	}

	var values []float64
	for _, result := range results {
		switch metric {
		case "throughput":
			values = append(values, result.ThroughputMbps)
		case "latency":
			values = append(values, result.LatencyMs)
		}
	}

	if len(values) == 0 {
		return 0
	}

	mean := average(values)
	var variance float64
	for _, value := range values {
		variance += (value - mean) * (value - mean)
	}
	variance /= float64(len(values))

	stdDev := variance // Simplified calculation
	if mean == 0 {
		return 0
	}

	return (stdDev / mean) * 100 // Coefficient of variation as percentage
}