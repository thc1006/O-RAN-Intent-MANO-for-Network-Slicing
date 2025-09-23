package pkg

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

// MetricsCollector collects and aggregates performance metrics
type MetricsCollector struct {
	logger      *log.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	running     bool

	// Storage for metrics
	agentMetrics    map[string][]*PerformanceMetrics
	statusHistory   map[string][]*TNStatus
	testResults     []*NetworkSliceMetrics

	// Configuration
	retentionPeriod time.Duration
	maxSamples      int
}


// MetricsReport contains comprehensive performance analysis
type MetricsReport struct {
	GeneratedAt          time.Time                    `json:"generatedAt"`
	ReportPeriod         time.Duration                `json:"reportPeriod"`
	Summary              MetricsSummary               `json:"summary"`
	ThesisValidation     ThesisValidationReport       `json:"thesisValidation"`
	ClusterPerformance   map[string]ClusterMetrics    `json:"clusterPerformance"`
	NetworkSliceResults  []*NetworkSliceMetrics       `json:"networkSliceResults"`
	TrendAnalysis        TrendAnalysis                `json:"trendAnalysis"`
	Recommendations      []string                     `json:"recommendations"`
	QualityAssessment    QualityAssessment            `json:"qualityAssessment"`
}

// MetricsSummary contains aggregated metrics summary
type MetricsSummary struct {
	TotalTests           int                `json:"totalTests"`
	PassedTests          int                `json:"passedTests"`
	FailedTests          int                `json:"failedTests"`
	SuccessRate          float64            `json:"successRate"`
	AvgThroughputMbps    float64            `json:"avgThroughputMbps"`
	AvgLatencyMs         float64            `json:"avgLatencyMs"`
	AvgDeployTimeMs      int64              `json:"avgDeployTimeMs"`
	TotalClusters        int                `json:"totalClusters"`
	ActiveClusters       int                `json:"activeClusters"`
	TotalSlices          int                `json:"totalSlices"`
	SLACompliantSlices   int                `json:"slaCompliantSlices"`
	WorstPerformingSlice string             `json:"worstPerformingSlice"`
	BestPerformingSlice  string             `json:"bestPerformingSlice"`
}

// ThesisValidationReport validates against thesis targets
type ThesisValidationReport struct {
	ThroughputTargets      []float64            `json:"throughputTargets"`
	RTTTargets             []float64            `json:"rttTargets"`
	DeployTimeTargetMs     int64                `json:"deployTimeTargetMs"`
	ThroughputResults      []float64            `json:"throughputResults"`
	RTTResults             []float64            `json:"rttResults"`
	DeployTimeResults      []int64              `json:"deployTimeResults"`
	ThroughputCompliance   float64              `json:"throughputCompliance"`
	RTTCompliance          float64              `json:"rttCompliance"`
	DeployTimeCompliance   float64              `json:"deployTimeCompliance"`
	OverallCompliance      float64              `json:"overallCompliance"`
	TargetAchievements     map[string]bool      `json:"targetAchievements"`
	ComplianceDetails      []ComplianceDetail   `json:"complianceDetails"`
}

// ComplianceDetail contains detailed compliance information
type ComplianceDetail struct {
	TestType     string  `json:"testType"`
	Target       float64 `json:"target"`
	Achieved     float64 `json:"achieved"`
	Compliant    bool    `json:"compliant"`
	Variance     float64 `json:"variance"`
	VarianceType string  `json:"varianceType"` // "better", "worse", "within_tolerance"
}

// ClusterMetrics contains metrics for a specific cluster
type ClusterMetrics struct {
	ClusterName          string               `json:"clusterName"`
	TotalTests           int                  `json:"totalTests"`
	SuccessfulTests      int                  `json:"successfulTests"`
	FailureRate          float64              `json:"failureRate"`
	AvgThroughputMbps    float64              `json:"avgThroughputMbps"`
	AvgLatencyMs         float64              `json:"avgLatencyMs"`
	PeakThroughputMbps   float64              `json:"peakThroughputMbps"`
	MinLatencyMs         float64              `json:"minLatencyMs"`
	MaxLatencyMs         float64              `json:"maxLatencyMs"`
	AvgBandwidthUtil     float64              `json:"avgBandwidthUtilization"`
	ErrorCount           int                  `json:"errorCount"`
	HealthScore          float64              `json:"healthScore"`
	LastUpdateTime       time.Time            `json:"lastUpdateTime"`
	QoSCompliance        map[string]float64   `json:"qosCompliance"`
}

// TrendAnalysis provides trend analysis over time
type TrendAnalysis struct {
	ThroughputTrend      string               `json:"throughputTrend"`      // "improving", "degrading", "stable"
	LatencyTrend         string               `json:"latencyTrend"`
	ErrorRateTrend       string               `json:"errorRateTrend"`
	ThroughputChange     float64              `json:"throughputChange"`     // percentage change
	LatencyChange        float64              `json:"latencyChange"`
	ErrorRateChange      float64              `json:"errorRateChange"`
	TrendPeriod          time.Duration        `json:"trendPeriod"`
	DataPoints           int                  `json:"dataPoints"`
	Confidence           float64              `json:"confidence"`           // confidence in trend analysis
}

// QualityAssessment provides overall quality assessment
type QualityAssessment struct {
	OverallGrade         string               `json:"overallGrade"`         // A, B, C, D, F
	PerformanceScore     float64              `json:"performanceScore"`     // 0-100
	ReliabilityScore     float64              `json:"reliabilityScore"`
	EfficiencyScore      float64              `json:"efficiencyScore"`
	SLAComplianceScore   float64              `json:"slaComplianceScore"`
	CriticalIssues       []string             `json:"criticalIssues"`
	Warnings             []string             `json:"warnings"`
	Strengths            []string             `json:"strengths"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *log.Logger) *MetricsCollector {
	return &MetricsCollector{
		logger:          logger,
		agentMetrics:    make(map[string][]*PerformanceMetrics),
		statusHistory:   make(map[string][]*TNStatus),
		testResults:     make([]*NetworkSliceMetrics, 0),
		retentionPeriod: 24 * time.Hour,
		maxSamples:      1000,
		running:         false,
	}
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return fmt.Errorf("metrics collector already running")
	}

	mc.ctx, mc.cancel = context.WithCancel(ctx)
	mc.running = true

	// Start cleanup goroutine
	go mc.cleanupLoop()

	mc.logger.Println("Metrics collector started")
	return nil
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return nil
	}

	mc.cancel()
	mc.running = false

	mc.logger.Println("Metrics collector stopped")
	return nil
}

// RecordMetrics records performance metrics from an agent
func (mc *MetricsCollector) RecordMetrics(agentName string, metrics *PerformanceMetrics) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return fmt.Errorf("metrics collector not running")
	}

	// Add to agent metrics
	mc.agentMetrics[agentName] = append(mc.agentMetrics[agentName], metrics)

	// Trim if too many samples
	if len(mc.agentMetrics[agentName]) > mc.maxSamples {
		mc.agentMetrics[agentName] = mc.agentMetrics[agentName][1:]
	}

	return nil
}

// RecordStatus records status information from an agent
func (mc *MetricsCollector) RecordStatus(agentName string, status *TNStatus) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return fmt.Errorf("metrics collector not running")
	}

	// Add to status history
	mc.statusHistory[agentName] = append(mc.statusHistory[agentName], status)

	// Trim if too many samples
	if len(mc.statusHistory[agentName]) > mc.maxSamples {
		mc.statusHistory[agentName] = mc.statusHistory[agentName][1:]
	}

	return nil
}

// RecordTestResult records a network slice test result
func (mc *MetricsCollector) RecordTestResult(result *NetworkSliceMetrics) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return fmt.Errorf("metrics collector not running")
	}

	mc.testResults = append(mc.testResults, result)

	// Trim if too many results
	if len(mc.testResults) > mc.maxSamples {
		mc.testResults = mc.testResults[1:]
	}

	mc.logger.Printf("Recorded test result for slice %s: %.2f%% compliance",
		result.SliceID, result.ThesisValidation.CompliancePercent)

	return nil
}

// GenerateReport generates a comprehensive metrics report
func (mc *MetricsCollector) GenerateReport() (*MetricsReport, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	report := &MetricsReport{
		GeneratedAt:         time.Now(),
		ReportPeriod:        mc.retentionPeriod,
		ClusterPerformance:  make(map[string]ClusterMetrics),
		NetworkSliceResults: make([]*NetworkSliceMetrics, len(mc.testResults)),
	}

	copy(report.NetworkSliceResults, mc.testResults)

	// Generate summary
	report.Summary = mc.generateSummary()

	// Generate thesis validation report
	report.ThesisValidation = mc.generateThesisValidation()

	// Generate cluster performance
	for agentName := range mc.agentMetrics {
		report.ClusterPerformance[agentName] = mc.generateClusterMetrics(agentName)
	}

	// Generate trend analysis
	report.TrendAnalysis = mc.generateTrendAnalysis()

	// Generate quality assessment
	report.QualityAssessment = mc.generateQualityAssessment()

	// Generate recommendations
	report.Recommendations = mc.generateRecommendations(report)

	mc.logger.Printf("Generated metrics report with %d test results", len(report.NetworkSliceResults))

	return report, nil
}

// generateSummary generates metrics summary
func (mc *MetricsCollector) generateSummary() MetricsSummary {
	summary := MetricsSummary{
		TotalClusters: len(mc.agentMetrics),
	}

	var totalThroughput, totalLatency float64
	var totalDeployTime int64
	var throughputCount, latencyCount, deployTimeCount int

	// Count active clusters
	for _, status := range mc.statusHistory {
		if len(status) > 0 && status[len(status)-1].Healthy {
			summary.ActiveClusters++
		}
	}

	// Process test results
	summary.TotalTests = len(mc.testResults)
	summary.TotalSlices = len(mc.testResults)

	for _, result := range mc.testResults {
		if result.SLACompliance {
			summary.PassedTests++
			summary.SLACompliantSlices++
		} else {
			summary.FailedTests++
		}

		if result.Performance.Throughput.AvgMbps > 0 {
			totalThroughput += result.Performance.Throughput.AvgMbps
			throughputCount++
		}

		if result.Performance.Latency.AvgRTTMs > 0 {
			totalLatency += result.Performance.Latency.AvgRTTMs
			latencyCount++
		}

		if result.ThesisValidation.DeployTimeMs > 0 {
			totalDeployTime += result.ThesisValidation.DeployTimeMs
			deployTimeCount++
		}
	}

	if summary.TotalTests > 0 {
		summary.SuccessRate = float64(summary.PassedTests) / float64(summary.TotalTests) * 100
	}

	if throughputCount > 0 {
		summary.AvgThroughputMbps = totalThroughput / float64(throughputCount)
	}

	if latencyCount > 0 {
		summary.AvgLatencyMs = totalLatency / float64(latencyCount)
	}

	if deployTimeCount > 0 {
		summary.AvgDeployTimeMs = totalDeployTime / int64(deployTimeCount)
	}

	// Find best and worst performing slices
	var bestCompliance, worstCompliance float64 = -1, 101
	for _, result := range mc.testResults {
		compliance := result.ThesisValidation.CompliancePercent
		if compliance > bestCompliance {
			bestCompliance = compliance
			summary.BestPerformingSlice = result.SliceID
		}
		if compliance < worstCompliance {
			worstCompliance = compliance
			summary.WorstPerformingSlice = result.SliceID
		}
	}

	return summary
}

// generateThesisValidation generates thesis validation report
func (mc *MetricsCollector) generateThesisValidation() ThesisValidationReport {
	report := ThesisValidationReport{
		ThroughputTargets:   []float64{0.93, 2.77, 4.57},
		RTTTargets:          []float64{6.3, 15.7, 16.1},
		DeployTimeTargetMs:  600000, // 10 minutes
		TargetAchievements:  make(map[string]bool),
		ComplianceDetails:   make([]ComplianceDetail, 0),
	}

	var throughputResults, rttResults []float64
	var deployTimeResults []int64
	var throughputCompliant, rttCompliant, deployTimeCompliant int

	// Collect results from test data
	for _, result := range mc.testResults {
		if len(result.ThesisValidation.ThroughputResults) > 0 {
			throughputResults = append(throughputResults, result.ThesisValidation.ThroughputResults...)
		}
		if len(result.ThesisValidation.RTTResults) > 0 {
			rttResults = append(rttResults, result.ThesisValidation.RTTResults...)
		}
		if result.ThesisValidation.DeployTimeMs > 0 {
			deployTimeResults = append(deployTimeResults, result.ThesisValidation.DeployTimeMs)
		}
	}

	report.ThroughputResults = throughputResults
	report.RTTResults = rttResults
	report.DeployTimeResults = deployTimeResults

	// Validate throughput targets
	for i, target := range report.ThroughputTargets {
		targetKey := fmt.Sprintf("throughput_%.2f_mbps", target)
		achieved := false

		if i < len(throughputResults) {
			achieved = throughputResults[i] >= target*0.9 // 10% tolerance
			if achieved {
				throughputCompliant++
			}

			// Add compliance detail
			detail := ComplianceDetail{
				TestType: fmt.Sprintf("Throughput %.2f Mbps", target),
				Target:   target,
				Achieved: throughputResults[i],
				Compliant: achieved,
				Variance: (throughputResults[i] - target) / target * 100,
			}

			if detail.Variance > 10 {
				detail.VarianceType = "better"
			} else if detail.Variance < -10 {
				detail.VarianceType = "worse"
			} else {
				detail.VarianceType = "within_tolerance"
			}

			report.ComplianceDetails = append(report.ComplianceDetails, detail)
		}

		report.TargetAchievements[targetKey] = achieved
	}

	// Validate RTT targets
	for i, target := range report.RTTTargets {
		targetKey := fmt.Sprintf("rtt_%.1f_ms", target)
		achieved := false

		if i < len(rttResults) {
			achieved = rttResults[i] <= target*1.1 // 10% tolerance
			if achieved {
				rttCompliant++
			}

			// Add compliance detail
			detail := ComplianceDetail{
				TestType: fmt.Sprintf("RTT %.1f ms", target),
				Target:   target,
				Achieved: rttResults[i],
				Compliant: achieved,
				Variance: (rttResults[i] - target) / target * 100,
			}

			if detail.Variance < -10 {
				detail.VarianceType = "better"
			} else if detail.Variance > 10 {
				detail.VarianceType = "worse"
			} else {
				detail.VarianceType = "within_tolerance"
			}

			report.ComplianceDetails = append(report.ComplianceDetails, detail)
		}

		report.TargetAchievements[targetKey] = achieved
	}

	// Validate deploy time
	deployTimeKey := "deploy_time_10min"
	deployTimeAchieved := false
	if len(deployTimeResults) > 0 {
		avgDeployTime := int64(0)
		for _, dt := range deployTimeResults {
			avgDeployTime += dt
		}
		avgDeployTime /= int64(len(deployTimeResults))

		deployTimeAchieved = avgDeployTime <= report.DeployTimeTargetMs
		if deployTimeAchieved {
			deployTimeCompliant = len(deployTimeResults)
		}

		// Add compliance detail
		detail := ComplianceDetail{
			TestType: "Deploy Time 10 min",
			Target:   float64(report.DeployTimeTargetMs),
			Achieved: float64(avgDeployTime),
			Compliant: deployTimeAchieved,
			Variance: (float64(avgDeployTime) - float64(report.DeployTimeTargetMs)) / float64(report.DeployTimeTargetMs) * 100,
		}

		if detail.Variance < 0 {
			detail.VarianceType = "better"
		} else if detail.Variance > 20 {
			detail.VarianceType = "worse"
		} else {
			detail.VarianceType = "within_tolerance"
		}

		report.ComplianceDetails = append(report.ComplianceDetails, detail)
	}

	report.TargetAchievements[deployTimeKey] = deployTimeAchieved

	// Calculate compliance percentages
	if len(report.ThroughputTargets) > 0 {
		report.ThroughputCompliance = float64(throughputCompliant) / float64(len(report.ThroughputTargets)) * 100
	}

	if len(report.RTTTargets) > 0 {
		report.RTTCompliance = float64(rttCompliant) / float64(len(report.RTTTargets)) * 100
	}

	if len(deployTimeResults) > 0 {
		report.DeployTimeCompliance = float64(deployTimeCompliant) / float64(len(deployTimeResults)) * 100
	}

	// Calculate overall compliance
	totalTargets := len(report.ThroughputTargets) + len(report.RTTTargets) + 1 // +1 for deploy time
	totalCompliant := throughputCompliant + rttCompliant + deployTimeCompliant
	report.OverallCompliance = float64(totalCompliant) / float64(totalTargets) * 100

	return report
}

// generateClusterMetrics generates metrics for a specific cluster
func (mc *MetricsCollector) generateClusterMetrics(agentName string) ClusterMetrics {
	metrics := ClusterMetrics{
		ClusterName:    agentName,
		QoSCompliance:  make(map[string]float64),
		LastUpdateTime: time.Now(),
	}

	agentMetrics := mc.agentMetrics[agentName]
	if len(agentMetrics) == 0 {
		return metrics
	}

	var totalThroughput, totalLatency, totalBandwidthUtil float64
	var maxThroughput, minLatency, maxLatency float64
	var errorCount int

	metrics.TotalTests = len(agentMetrics)
	maxLatency = 0
	minLatency = math.MaxFloat64

	for _, m := range agentMetrics {
		if len(m.ErrorDetails) == 0 {
			metrics.SuccessfulTests++
		} else {
			errorCount += len(m.ErrorDetails)
		}

		totalThroughput += m.Throughput.AvgMbps
		totalLatency += m.Latency.AvgRTTMs
		totalBandwidthUtil += m.BandwidthUtilization

		if m.Throughput.PeakMbps > maxThroughput {
			maxThroughput = m.Throughput.PeakMbps
		}

		if m.Latency.MinRTTMs < minLatency && m.Latency.MinRTTMs > 0 {
			minLatency = m.Latency.MinRTTMs
		}

		if m.Latency.MaxRTTMs > maxLatency {
			maxLatency = m.Latency.MaxRTTMs
		}

		// Update last update time
		if m.Timestamp.After(metrics.LastUpdateTime) {
			metrics.LastUpdateTime = m.Timestamp
		}
	}

	if metrics.TotalTests > 0 {
		metrics.AvgThroughputMbps = totalThroughput / float64(metrics.TotalTests)
		metrics.AvgLatencyMs = totalLatency / float64(metrics.TotalTests)
		metrics.AvgBandwidthUtil = totalBandwidthUtil / float64(metrics.TotalTests)
		metrics.FailureRate = float64(metrics.TotalTests-metrics.SuccessfulTests) / float64(metrics.TotalTests) * 100
	}

	metrics.PeakThroughputMbps = maxThroughput
	metrics.MinLatencyMs = minLatency
	metrics.MaxLatencyMs = maxLatency
	metrics.ErrorCount = errorCount

	// Calculate health score (0-100)
	healthScore := 100.0
	healthScore -= metrics.FailureRate                    // Subtract failure rate
	healthScore -= math.Min(metrics.AvgLatencyMs/10, 50)  // Penalize high latency
	healthScore += math.Min(metrics.AvgThroughputMbps, 20) // Reward high throughput

	if healthScore < 0 {
		healthScore = 0
	}
	metrics.HealthScore = healthScore

	return metrics
}

// generateTrendAnalysis generates trend analysis
func (mc *MetricsCollector) generateTrendAnalysis() TrendAnalysis {
	analysis := TrendAnalysis{
		TrendPeriod: time.Hour * 24, // Analyze last 24 hours
		Confidence:  0.0,
	}

	if len(mc.testResults) < 2 {
		return analysis
	}

	// Sort results by timestamp
	sortedResults := make([]*NetworkSliceMetrics, len(mc.testResults))
	copy(sortedResults, mc.testResults)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].Timestamp.Before(sortedResults[j].Timestamp)
	})

	// Split into two halves for comparison
	mid := len(sortedResults) / 2
	firstHalf := sortedResults[:mid]
	secondHalf := sortedResults[mid:]

	// Calculate averages for each half
	var firstThroughput, firstLatency, firstErrorRate float64
	var secondThroughput, secondLatency, secondErrorRate float64

	for _, result := range firstHalf {
		firstThroughput += result.Performance.Throughput.AvgMbps
		firstLatency += result.Performance.Latency.AvgRTTMs
		if !result.SLACompliance {
			firstErrorRate += 1
		}
	}

	for _, result := range secondHalf {
		secondThroughput += result.Performance.Throughput.AvgMbps
		secondLatency += result.Performance.Latency.AvgRTTMs
		if !result.SLACompliance {
			secondErrorRate += 1
		}
	}

	if len(firstHalf) > 0 {
		firstThroughput /= float64(len(firstHalf))
		firstLatency /= float64(len(firstHalf))
		firstErrorRate = firstErrorRate / float64(len(firstHalf)) * 100
	}

	if len(secondHalf) > 0 {
		secondThroughput /= float64(len(secondHalf))
		secondLatency /= float64(len(secondHalf))
		secondErrorRate = secondErrorRate / float64(len(secondHalf)) * 100
	}

	// Calculate changes
	if firstThroughput > 0 {
		analysis.ThroughputChange = (secondThroughput - firstThroughput) / firstThroughput * 100
	}

	if firstLatency > 0 {
		analysis.LatencyChange = (secondLatency - firstLatency) / firstLatency * 100
	}

	analysis.ErrorRateChange = secondErrorRate - firstErrorRate

	// Determine trends
	if analysis.ThroughputChange > 5 {
		analysis.ThroughputTrend = "improving"
	} else if analysis.ThroughputChange < -5 {
		analysis.ThroughputTrend = "degrading"
	} else {
		analysis.ThroughputTrend = "stable"
	}

	if analysis.LatencyChange > 5 {
		analysis.LatencyTrend = "degrading"
	} else if analysis.LatencyChange < -5 {
		analysis.LatencyTrend = "improving"
	} else {
		analysis.LatencyTrend = "stable"
	}

	if analysis.ErrorRateChange > 5 {
		analysis.ErrorRateTrend = "degrading"
	} else if analysis.ErrorRateChange < -5 {
		analysis.ErrorRateTrend = "improving"
	} else {
		analysis.ErrorRateTrend = "stable"
	}

	analysis.DataPoints = len(sortedResults)
	analysis.Confidence = math.Min(float64(analysis.DataPoints)/10.0, 1.0) // Higher confidence with more data points

	return analysis
}

// generateQualityAssessment generates quality assessment
func (mc *MetricsCollector) generateQualityAssessment() QualityAssessment {
	assessment := QualityAssessment{
		CriticalIssues: make([]string, 0),
		Warnings:       make([]string, 0),
		Strengths:      make([]string, 0),
	}

	if len(mc.testResults) == 0 {
		assessment.OverallGrade = "F"
		assessment.CriticalIssues = append(assessment.CriticalIssues, "No test results available")
		return assessment
	}

	// Calculate scores
	var totalCompliance, totalThroughput, totalLatency float64
	var compliantResults, reliableResults int

	for _, result := range mc.testResults {
		totalCompliance += result.ThesisValidation.CompliancePercent
		totalThroughput += result.Performance.Throughput.AvgMbps
		totalLatency += result.Performance.Latency.AvgRTTMs

		if result.SLACompliance {
			compliantResults++
		}

		if len(result.Performance.ErrorDetails) == 0 {
			reliableResults++
		}
	}

	numResults := float64(len(mc.testResults))
	avgCompliance := totalCompliance / numResults
	avgThroughput := totalThroughput / numResults
	avgLatency := totalLatency / numResults

	// Performance score (0-100)
	assessment.PerformanceScore = avgCompliance

	// Reliability score based on success rate
	assessment.ReliabilityScore = float64(reliableResults) / numResults * 100

	// Efficiency score based on throughput and latency
	efficiencyScore := 50.0 // Base score
	if avgThroughput > 2.0 {
		efficiencyScore += 25
	}
	if avgLatency < 20.0 {
		efficiencyScore += 25
	}
	assessment.EfficiencyScore = math.Min(efficiencyScore, 100)

	// SLA compliance score
	assessment.SLAComplianceScore = float64(compliantResults) / numResults * 100

	// Overall grade
	overallScore := (assessment.PerformanceScore + assessment.ReliabilityScore +
					assessment.EfficiencyScore + assessment.SLAComplianceScore) / 4

	if overallScore >= 90 {
		assessment.OverallGrade = "A"
	} else if overallScore >= 80 {
		assessment.OverallGrade = "B"
	} else if overallScore >= 70 {
		assessment.OverallGrade = "C"
	} else if overallScore >= 60 {
		assessment.OverallGrade = "D"
	} else {
		assessment.OverallGrade = "F"
	}

	// Generate issues and strengths
	if assessment.SLAComplianceScore < 80 {
		assessment.CriticalIssues = append(assessment.CriticalIssues, "SLA compliance below 80%")
	}

	if avgLatency > 50 {
		assessment.CriticalIssues = append(assessment.CriticalIssues, "High average latency detected")
	}

	if avgThroughput < 1.0 {
		assessment.Warnings = append(assessment.Warnings, "Low average throughput")
	}

	if assessment.ReliabilityScore < 90 {
		assessment.Warnings = append(assessment.Warnings, "Reliability issues detected")
	}

	if avgThroughput > 3.0 {
		assessment.Strengths = append(assessment.Strengths, "High throughput performance")
	}

	if avgLatency < 10 {
		assessment.Strengths = append(assessment.Strengths, "Excellent latency performance")
	}

	if assessment.SLAComplianceScore > 95 {
		assessment.Strengths = append(assessment.Strengths, "Excellent SLA compliance")
	}

	return assessment
}

// generateRecommendations generates recommendations based on analysis
func (mc *MetricsCollector) generateRecommendations(report *MetricsReport) []string {
	recommendations := make([]string, 0)

	// Performance-based recommendations
	if report.Summary.SuccessRate < 80 {
		recommendations = append(recommendations, "Investigate and address test failures to improve success rate")
	}

	if report.Summary.AvgLatencyMs > 30 {
		recommendations = append(recommendations, "Optimize network configuration to reduce latency")
	}

	if report.Summary.AvgThroughputMbps < 2.0 {
		recommendations = append(recommendations, "Review bandwidth allocation and traffic shaping policies")
	}

	// Trend-based recommendations
	if report.TrendAnalysis.ThroughputTrend == "degrading" {
		recommendations = append(recommendations, "Investigate throughput degradation trend")
	}

	if report.TrendAnalysis.LatencyTrend == "degrading" {
		recommendations = append(recommendations, "Monitor network congestion and optimize routing")
	}

	if report.TrendAnalysis.ErrorRateTrend == "degrading" {
		recommendations = append(recommendations, "Review error logs and implement preventive measures")
	}

	// Compliance-based recommendations
	if report.ThesisValidation.OverallCompliance < 80 {
		recommendations = append(recommendations, "Review thesis requirements and adjust system configuration")
	}

	if report.ThesisValidation.DeployTimeCompliance < 90 {
		recommendations = append(recommendations, "Optimize deployment process to meet 10-minute target")
	}

	// Cluster-specific recommendations
	for clusterName, metrics := range report.ClusterPerformance {
		if metrics.HealthScore < 70 {
			recommendations = append(recommendations,
				fmt.Sprintf("Investigate health issues in cluster %s (score: %.1f)", clusterName, metrics.HealthScore))
		}

		if metrics.FailureRate > 20 {
			recommendations = append(recommendations,
				fmt.Sprintf("Address high failure rate in cluster %s (%.1f%%)", clusterName, metrics.FailureRate))
		}
	}

	// Quality-based recommendations
	if len(report.QualityAssessment.CriticalIssues) > 0 {
		recommendations = append(recommendations, "Address critical issues identified in quality assessment")
	}

	if report.QualityAssessment.OverallGrade == "F" || report.QualityAssessment.OverallGrade == "D" {
		recommendations = append(recommendations, "Comprehensive system review and optimization required")
	}

	return recommendations
}

// cleanupLoop periodically cleans up old metrics
func (mc *MetricsCollector) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.cleanupOldMetrics()
		}
	}
}

// cleanupOldMetrics removes metrics older than retention period
func (mc *MetricsCollector) cleanupOldMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.retentionPeriod)

	// Clean agent metrics
	for agentName, metrics := range mc.agentMetrics {
		var filtered []*PerformanceMetrics
		for _, m := range metrics {
			if m.Timestamp.After(cutoff) {
				filtered = append(filtered, m)
			}
		}
		mc.agentMetrics[agentName] = filtered
	}

	// Clean status history
	for agentName, statuses := range mc.statusHistory {
		var filtered []*TNStatus
		for _, s := range statuses {
			if s.LastUpdate.After(cutoff) {
				filtered = append(filtered, s)
			}
		}
		mc.statusHistory[agentName] = filtered
	}

	// Clean test results
	var filteredResults []*NetworkSliceMetrics
	for _, result := range mc.testResults {
		if result.Timestamp.After(cutoff) {
			filteredResults = append(filteredResults, result)
		}
	}
	mc.testResults = filteredResults

	mc.logger.Println("Cleaned up old metrics data")
}

// Export exports all collected metrics
func (mc *MetricsCollector) Export() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	export := map[string]interface{}{
		"agent_metrics":    mc.agentMetrics,
		"status_history":   mc.statusHistory,
		"test_results":     mc.testResults,
		"collection_time":  time.Now(),
		"retention_period": mc.retentionPeriod,
	}

	return export
}

