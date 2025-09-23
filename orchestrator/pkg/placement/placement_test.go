package placement

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MockMetricsProvider is provided by metrics_mock.go - removed duplicate implementation

// IntelligentPlacementPolicy is a mock implementation for testing
type IntelligentPlacementPolicy struct {
	metricsProvider MetricsProvider
}

func NewIntelligentPlacementPolicy(provider MetricsProvider) *IntelligentPlacementPolicy {
	return &IntelligentPlacementPolicy{
		metricsProvider: provider,
	}
}

func (p *IntelligentPlacementPolicy) Place(nf *NetworkFunction, sites []*Site) (*PlacementDecision, error) {
	if len(sites) == 0 {
		return nil, &PlacementError{
			Code:    ErrNoSuitableSite,
			Message: "No sites available for placement",
		}
	}

	var bestSite *Site
	var bestScore float64 = -1
	var alternatives []SiteScore

	for _, site := range sites {
		if !site.Available {
			continue
		}

		score := p.calculatePlacementScore(nf, site)
		alternatives = append(alternatives, SiteScore{Site: site, Score: score})

		if score > bestScore {
			bestScore = score
			bestSite = site
		}
	}

	if bestSite == nil {
		return nil, &PlacementError{
			Code:    ErrNoSuitableSite,
			Message: "No suitable site found for placement",
		}
	}

	// Sort alternatives by score in descending order
	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].Score > alternatives[j].Score
	})

	return &PlacementDecision{
		NetworkFunction: nf,
		Site:           bestSite,
		Score:          bestScore,
		Reason:         fmt.Sprintf("Selected site %s with score %.2f", bestSite.Name, bestScore),
		Alternatives:   alternatives,
		Timestamp:      time.Now(),
	}, nil
}

func (p *IntelligentPlacementPolicy) PlaceMultiple(nfs []*NetworkFunction, sites []*Site) ([]*PlacementDecision, error) {
	decisions := make([]*PlacementDecision, 0, len(nfs))

	for _, nf := range nfs {
		decision, err := p.Place(nf, sites)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}

	return decisions, nil
}

func (p *IntelligentPlacementPolicy) Rebalance(decisions []*PlacementDecision, sites []*Site) ([]*PlacementDecision, error) {
	// Mock rebalancing logic
	return decisions, nil
}

func (p *IntelligentPlacementPolicy) calculatePlacementScore(nf *NetworkFunction, site *Site) float64 {
	score := 0.0

	// Resource availability score (0-30 points)
	resourceScore := p.calculateResourceScore(nf, site)
	score += resourceScore * 0.3

	// QoS compliance score (0-40 points)
	qosScore := p.calculateQoSScore(nf, site)
	score += qosScore * 0.4

	// If site can't meet basic resource or QoS requirements, penalize heavily
	if resourceScore == 0.0 || qosScore == 0.0 {
		return score * 0.1 // Only 10% of the score for impossible placements
	}

	// Placement hints score (0-20 points)
	hintsScore := p.calculateHintsScore(nf, site)
	score += hintsScore * 0.2

	// Site efficiency score (0-10 points)
	efficiencyScore := p.calculateEfficiencyScore(site)
	score += efficiencyScore * 0.1

	return score
}

func (p *IntelligentPlacementPolicy) calculateResourceScore(nf *NetworkFunction, site *Site) float64 {
	// Check if site has sufficient resources
	if site.Capacity.CPUCores < nf.Requirements.MinCPUCores ||
		site.Capacity.MemoryGB < nf.Requirements.MinMemoryGB ||
		site.Capacity.StorageGB < nf.Requirements.MinStorageGB ||
		site.Capacity.BandwidthMbps < nf.Requirements.MinBandwidthMbps {
		return 0.0
	}

	// Calculate resource utilization and availability
	metrics, _ := p.metricsProvider.GetMetrics(site.ID)
	if metrics == nil {
		return 50.0 // Default score if no metrics available
	}

	// Score based on available resources (higher availability = higher score)
	cpuScore := 100.0 - metrics.CPUUtilization
	memoryScore := 100.0 - metrics.MemoryUtilization

	return (cpuScore + memoryScore) / 2.0
}

func (p *IntelligentPlacementPolicy) calculateQoSScore(nf *NetworkFunction, site *Site) float64 {
	score := 0.0

	// Latency score
	if site.NetworkProfile.BaseLatencyMs <= nf.QoSRequirements.MaxLatencyMs {
		latencyRatio := site.NetworkProfile.BaseLatencyMs / nf.QoSRequirements.MaxLatencyMs
		score += (2.0 - latencyRatio) * 20.0 // Better latency = higher score, max 40 points
	}

	// Throughput score
	if site.NetworkProfile.MaxThroughputMbps >= nf.QoSRequirements.MinThroughputMbps {
		throughputRatio := nf.QoSRequirements.MinThroughputMbps / site.NetworkProfile.MaxThroughputMbps
		score += (1.5 - throughputRatio) * 20.0 // Better throughput = higher score, max 30 points
	}

	// Packet loss score
	if site.NetworkProfile.PacketLossRate <= nf.QoSRequirements.MaxPacketLossRate {
		score += 20.0
	}

	// Jitter score
	if site.NetworkProfile.JitterMs <= nf.QoSRequirements.MaxJitterMs {
		score += 10.0
	}

	return score
}

func (p *IntelligentPlacementPolicy) calculateHintsScore(nf *NetworkFunction, site *Site) float64 {
	score := 0.0
	totalWeight := 0

	for _, hint := range nf.PlacementHints {
		totalWeight += hint.Weight

		switch hint.Type {
		case HintTypeLocation:
			if site.Location.Region == hint.Value {
				score += float64(hint.Weight)
			}
		case HintTypeCloudType:
			if string(site.Type) == hint.Value {
				score += float64(hint.Weight)
			}
		}
	}

	if totalWeight > 0 {
		return (score / float64(totalWeight)) * 100.0
	}

	return 50.0 // Default score if no hints
}

func (p *IntelligentPlacementPolicy) calculateEfficiencyScore(site *Site) float64 {
	metrics, _ := p.metricsProvider.GetMetrics(site.ID)
	if metrics == nil {
		return 50.0
	}

	// Prefer sites with balanced utilization (not too empty, not too full)
	avgUtilization := (metrics.CPUUtilization + metrics.MemoryUtilization) / 2.0

	// Optimal utilization range: 30-70%
	if avgUtilization >= 30.0 && avgUtilization <= 70.0 {
		return 100.0
	} else if avgUtilization < 30.0 {
		return 50.0 + (avgUtilization / 30.0) * 50.0
	} else {
		return 100.0 - ((avgUtilization - 70.0) / 30.0) * 50.0
	}
}

// PlacementTestSuite provides comprehensive test suite for placement policies
type PlacementTestSuite struct {
	suite.Suite
	ctx             context.Context
	cancel          context.CancelFunc
	placement       PlacementPolicy
	testSites       []*Site
	testNFs         []*NetworkFunction
	metricsProvider *MockMetricsProvider
}

// SetupSuite initializes the test suite
func (suite *PlacementTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
	suite.metricsProvider = NewMockMetricsProvider()
	suite.placement = NewIntelligentPlacementPolicy(suite.metricsProvider)
	suite.setupTestData()
}

// TearDownSuite cleans up after the test suite
func (suite *PlacementTestSuite) TearDownSuite() {
	suite.cancel()
}

// setupTestData creates test sites and network functions
func (suite *PlacementTestSuite) setupTestData() {
	// Create test sites with different characteristics
	suite.testSites = []*Site{
		{
			ID:   "central-site-01",
			Name: "Central Data Center US-East",
			Type: CloudTypeCentral,
			Location: Location{
				Latitude:  39.0458,
				Longitude: -76.6413,
				Region:    "us-east-1",
				Zone:      "us-east-1a",
			},
			Capacity: ResourceCapacity{
				CPUCores:      1000,
				MemoryGB:      4000,
				StorageGB:     10000,
				BandwidthMbps: 10000,
			},
			NetworkProfile: NetworkProfile{
				BaseLatencyMs:     10.0,
				MaxThroughputMbps: 10000,
				PacketLossRate:    0.001,
				JitterMs:          1.0,
			},
			Available: true,
		},
		{
			ID:   "edge-site-01",
			Name: "Edge Site Boston",
			Type: CloudTypeEdge,
			Location: Location{
				Latitude:  42.3601,
				Longitude: -71.0589,
				Region:    "us-east-1",
				Zone:      "us-east-1b",
			},
			Capacity: ResourceCapacity{
				CPUCores:      500,
				MemoryGB:      2000,
				StorageGB:     5000,
				BandwidthMbps: 5000,
			},
			NetworkProfile: NetworkProfile{
				BaseLatencyMs:     1.0,
				MaxThroughputMbps: 5000,
				PacketLossRate:    0.0001,
				JitterMs:          0.5,
			},
			Available: true,
		},
		{
			ID:   "edge-site-02",
			Name: "Edge Site San Francisco",
			Type: CloudTypeEdge,
			Location: Location{
				Latitude:  37.7749,
				Longitude: -122.4194,
				Region:    "us-west-1",
				Zone:      "us-west-1a",
			},
			Capacity: ResourceCapacity{
				CPUCores:      300,
				MemoryGB:      1500,
				StorageGB:     3000,
				BandwidthMbps: 3000,
			},
			NetworkProfile: NetworkProfile{
				BaseLatencyMs:     2.0,
				MaxThroughputMbps: 3000,
				PacketLossRate:    0.0005,
				JitterMs:          1.0,
			},
			Available: true,
		},
		{
			ID:   "regional-site-01",
			Name: "Regional Site Chicago",
			Type: CloudTypeRegional,
			Location: Location{
				Latitude:  41.8781,
				Longitude: -87.6298,
				Region:    "us-central-1",
				Zone:      "us-central-1a",
			},
			Capacity: ResourceCapacity{
				CPUCores:      800,
				MemoryGB:      3000,
				StorageGB:     8000,
				BandwidthMbps: 8000,
			},
			NetworkProfile: NetworkProfile{
				BaseLatencyMs:     5.0,
				MaxThroughputMbps: 8000,
				PacketLossRate:    0.0002,
				JitterMs:          0.8,
			},
			Available: true,
		},
	}

	// Setup metrics for test sites
	for _, site := range suite.testSites {
		metrics := &SiteMetrics{
			Timestamp:              time.Now(),
			CPUUtilization:         20.0, // 20% utilization
			MemoryUtilization:      25.0, // 25% utilization
			AvailableBandwidthMbps: site.Capacity.BandwidthMbps * 0.8,
			CurrentLatencyMs:       site.NetworkProfile.BaseLatencyMs,
			ActiveNFs:              2,
		}
		suite.metricsProvider.SetMetrics(site.ID, metrics)
	}

	// Create test network functions with different requirements
	suite.testNFs = []*NetworkFunction{
		{
			ID:   "emergency-upf",
			Type: "UPF",
			Requirements: ResourceRequirements{
				MinCPUCores:      50,
				MinMemoryGB:      200,
				MinStorageGB:     500,
				MinBandwidthMbps: 1000,
			},
			QoSRequirements: QoSRequirements{
				MaxLatencyMs:      6.3, // Thesis target for emergency services
				MinThroughputMbps: 4.57, // Thesis target for emergency services
				MaxPacketLossRate: 0.001, // 99.9% reliability
				MaxJitterMs:       0.5,
			},
			PlacementHints: []PlacementHint{
				{
					Type:   HintTypeLocation,
					Value:  "us-east-1",
					Weight: 80,
				},
				{
					Type:   HintTypeCloudType,
					Value:  string(CloudTypeEdge),
					Weight: 90,
				},
			},
		},
		{
			ID:   "video-streaming-upf",
			Type: "UPF",
			Requirements: ResourceRequirements{
				MinCPUCores:      100,
				MinMemoryGB:      400,
				MinStorageGB:     1000,
				MinBandwidthMbps: 2000,
			},
			QoSRequirements: QoSRequirements{
				MaxLatencyMs:      15.7, // Thesis target for video
				MinThroughputMbps: 2.77, // Thesis target
				MaxPacketLossRate: 0.001,
				MaxJitterMs:       2.0,
			},
			PlacementHints: []PlacementHint{
				{
					Type:   HintTypeCloudType,
					Value:  string(CloudTypeRegional),
					Weight: 70,
				},
			},
		},
		{
			ID:   "iot-aggregator",
			Type: "SMF",
			Requirements: ResourceRequirements{
				MinCPUCores:      20,
				MinMemoryGB:      100,
				MinStorageGB:     200,
				MinBandwidthMbps: 100,
			},
			QoSRequirements: QoSRequirements{
				MaxLatencyMs:      16.1, // Thesis target for IoT
				MinThroughputMbps: 0.93, // Thesis target
				MaxPacketLossRate: 0.01,  // 99% reliability
				MaxJitterMs:       5.0,
			},
			PlacementHints: []PlacementHint{
				{
					Type:   HintTypeCloudType,
					Value:  string(CloudTypeCentral),
					Weight: 60,
				},
			},
		},
	}
}

// Test Placement Policy initialization
func (suite *PlacementTestSuite) TestPlacementPolicyInitialization() {
	assert.NotNil(suite.T(), suite.placement)
	assert.NotNil(suite.T(), suite.metricsProvider)
	assert.NotEmpty(suite.T(), suite.testSites)
	assert.NotEmpty(suite.T(), suite.testNFs)
}

// Test basic placement policy execution
func (suite *PlacementTestSuite) TestBasicPlacementPolicy() {
	nf := suite.testNFs[0] // emergency-upf
	sites := suite.testSites

	result, err := suite.placement.Place(nf, sites)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.NotNil(suite.T(), result.Site)
	assert.Greater(suite.T(), result.Score, 0.0)
	assert.NotEmpty(suite.T(), result.Reason)
}

// Test latency-based placement
func (suite *PlacementTestSuite) TestLatencyBasedPlacement() {
	// Create a workload with strict latency requirements
	nf := &NetworkFunction{
		ID:   "ultra-low-latency",
		Type: "UPF",
		Requirements: ResourceRequirements{
			MinCPUCores:      30,
			MinMemoryGB:      150,
			MinStorageGB:     300,
			MinBandwidthMbps: 500,
		},
		QoSRequirements: QoSRequirements{
			MaxLatencyMs:      0.5, // Very strict latency requirement
			MinThroughputMbps: 50,
			MaxPacketLossRate: 0.0001,
			MaxJitterMs:       0.2,
		},
	}

	result, err := suite.placement.Place(nf, suite.testSites)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result.Site)

	// Should select edge site with lowest latency
	assert.Equal(suite.T(), CloudTypeEdge, result.Site.Type)
	assert.LessOrEqual(suite.T(), result.Site.NetworkProfile.BaseLatencyMs, 1.0)
}

// Test throughput-based placement
func (suite *PlacementTestSuite) TestThroughputBasedPlacement() {
	// Create a workload with high throughput requirements
	nf := &NetworkFunction{
		ID:   "high-bandwidth",
		Type: "UPF",
		Requirements: ResourceRequirements{
			MinCPUCores:      200,
			MinMemoryGB:      800,
			MinStorageGB:     2000,
			MinBandwidthMbps: 8000,
		},
		QoSRequirements: QoSRequirements{
			MaxLatencyMs:      20.0,
			MinThroughputMbps: 8000, // High throughput requirement
			MaxPacketLossRate: 0.001,
			MaxJitterMs:       2.0,
		},
	}

	result, err := suite.placement.Place(nf, suite.testSites)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result.Site)

	// Should select site with sufficient throughput capacity
	assert.GreaterOrEqual(suite.T(), result.Site.NetworkProfile.MaxThroughputMbps, 8000.0)
}

// Test resource-based placement
func (suite *PlacementTestSuite) TestResourceBasedPlacement() {
	// Create a workload with high resource requirements
	nf := &NetworkFunction{
		ID:   "resource-intensive",
		Type: "UPF",
		Requirements: ResourceRequirements{
			MinCPUCores:      600, // High CPU requirement
			MinMemoryGB:      2400, // High memory requirement
			MinStorageGB:     6000,
			MinBandwidthMbps: 3000,
		},
		QoSRequirements: QoSRequirements{
			MaxLatencyMs:      15.0,
			MinThroughputMbps: 300,
			MaxPacketLossRate: 0.001,
			MaxJitterMs:       2.0,
		},
	}

	result, err := suite.placement.Place(nf, suite.testSites)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result.Site)

	// Should select site with sufficient available resources
	assert.GreaterOrEqual(suite.T(), result.Site.Capacity.CPUCores, 600)
	assert.GreaterOrEqual(suite.T(), result.Site.Capacity.MemoryGB, 2400)
}

// Test constraint-based placement
func (suite *PlacementTestSuite) TestConstraintBasedPlacement() {
	// Create a workload with specific placement constraints
	nf := &NetworkFunction{
		ID:   "constrained-workload",
		Type: "UPF",
		Requirements: ResourceRequirements{
			MinCPUCores:      50,
			MinMemoryGB:      200,
			MinStorageGB:     500,
			MinBandwidthMbps: 1000,
		},
		QoSRequirements: QoSRequirements{
			MaxLatencyMs:      5.0,
			MinThroughputMbps: 100,
			MaxPacketLossRate: 0.0005,
			MaxJitterMs:       1.0,
		},
		PlacementHints: []PlacementHint{
			{
				Type:   HintTypeLocation,
				Value:  "us-east-1",
				Weight: 80,
			},
			{
				Type:   HintTypeCloudType,
				Value:  string(CloudTypeEdge),
				Weight: 90,
			},
		},
	}

	result, err := suite.placement.Place(nf, suite.testSites)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result.Site)

	// Should select site matching constraints
	assert.Equal(suite.T(), "us-east-1", result.Site.Location.Region)
	assert.Equal(suite.T(), CloudTypeEdge, result.Site.Type)
}

// Test multiple placement
func (suite *PlacementTestSuite) TestMultiplePlacement() {
	// Place multiple network functions
	results, err := suite.placement.PlaceMultiple(suite.testNFs, suite.testSites)

	require.NoError(suite.T(), err)
	assert.Len(suite.T(), results, len(suite.testNFs))

	// Verify all placements are valid
	for i, result := range results {
		assert.NotNil(suite.T(), result)
		assert.NotNil(suite.T(), result.Site)
		assert.Equal(suite.T(), suite.testNFs[i].ID, result.NetworkFunction.ID)
		assert.Greater(suite.T(), result.Score, 0.0)
	}
}

// Test placement with unavailable sites
func (suite *PlacementTestSuite) TestPlacementWithUnavailableSites() {
	// Mark some sites as unavailable
	unavailableSites := make([]*Site, len(suite.testSites))
	copy(unavailableSites, suite.testSites)

	// Mark edge sites as unavailable
	for _, site := range unavailableSites {
		if site.Type == CloudTypeEdge {
			site.Available = false
		}
	}

	nf := suite.testNFs[0] // emergency-upf (prefers edge)

	result, err := suite.placement.Place(nf, unavailableSites)
	require.NoError(suite.T(), err)

	// Should fall back to available sites
	assert.NotNil(suite.T(), result.Site)
	assert.True(suite.T(), result.Site.Available)
}

// Test placement failure scenarios
func (suite *PlacementTestSuite) TestPlacementFailureScenarios() {
	// Test with impossible requirements
	impossibleNF := &NetworkFunction{
		ID:   "impossible-nf",
		Type: "UPF",
		Requirements: ResourceRequirements{
			MinCPUCores:      2000, // More than any site has
			MinMemoryGB:      8000,
			MinStorageGB:     20000,
			MinBandwidthMbps: 50000,
		},
		QoSRequirements: QoSRequirements{
			MaxLatencyMs:      0.1, // Impossible latency
			MinThroughputMbps: 50000, // Impossible throughput
			MaxPacketLossRate: 0.00001,
			MaxJitterMs:       0.1,
		},
	}

	result, err := suite.placement.Place(impossibleNF, suite.testSites)

	// Should return an error or result with very low score
	if err == nil {
		assert.NotNil(suite.T(), result)
		assert.Less(suite.T(), result.Score, 10.0) // Score should be very low
	} else {
		assert.Error(suite.T(), err)
	}
}

// Test placement with empty sites list
func (suite *PlacementTestSuite) TestPlacementWithEmptySites() {
	nf := suite.testNFs[0]
	emptySites := []*Site{}

	result, err := suite.placement.Place(nf, emptySites)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)

	placementErr, ok := err.(*PlacementError)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), ErrNoSuitableSite, placementErr.Code)
}

// Test placement scoring mechanism
func (suite *PlacementTestSuite) TestPlacementScoring() {
	nf := suite.testNFs[1] // video-streaming-upf

	result, err := suite.placement.Place(nf, suite.testSites)
	require.NoError(suite.T(), err)

	// Verify placement has proper scoring
	assert.NotNil(suite.T(), result)
	assert.Greater(suite.T(), result.Score, 0.0)
	assert.LessOrEqual(suite.T(), result.Score, 100.0)
	assert.NotEmpty(suite.T(), result.Alternatives)

	// Verify alternatives are sorted by score
	for i := 1; i < len(result.Alternatives); i++ {
		assert.GreaterOrEqual(suite.T(), result.Alternatives[i-1].Score, result.Alternatives[i].Score)
	}
}

// Test concurrent placement requests
func (suite *PlacementTestSuite) TestConcurrentPlacement() {
	numConcurrent := 10
	nf := suite.testNFs[2] // iot-aggregator

	resultChan := make(chan *PlacementDecision, numConcurrent)
	errorChan := make(chan error, numConcurrent)

	// Launch concurrent placement requests
	for i := 0; i < numConcurrent; i++ {
		go func() {
			result, err := suite.placement.Place(nf, suite.testSites)
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}()
	}

	// Collect results
	results := make([]*PlacementDecision, 0, numConcurrent)
	errors := make([]error, 0, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
		case err := <-errorChan:
			errors = append(errors, err)
		case <-time.After(10 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent placement results")
		}
	}

	// Verify all requests completed successfully
	assert.Equal(suite.T(), numConcurrent, len(results))
	assert.Empty(suite.T(), errors)

	// Verify all results are consistent
	for _, result := range results {
		assert.NotNil(suite.T(), result)
		assert.Equal(suite.T(), nf.ID, result.NetworkFunction.ID)
	}
}

// Test placement metrics and observability
func (suite *PlacementTestSuite) TestPlacementMetrics() {
	nf := suite.testNFs[1] // video-streaming-upf

	startTime := time.Now()
	result, err := suite.placement.Place(nf, suite.testSites)
	duration := time.Since(startTime)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)

	// Verify placement completed within reasonable time
	assert.Less(suite.T(), duration, 5*time.Second, "Placement should complete quickly")

	// Verify placement decision includes proper metadata
	assert.NotNil(suite.T(), result.NetworkFunction)
	assert.NotNil(suite.T(), result.Site)
	assert.NotZero(suite.T(), result.Timestamp)
	assert.NotEmpty(suite.T(), result.Reason)
}

// Test thesis performance targets
func (suite *PlacementTestSuite) TestThesisPerformanceTargets() {
	// Test deployment time target: placement decision should be fast
	nf := suite.testNFs[0] // emergency-upf

	startTime := time.Now()
	result, err := suite.placement.Place(nf, suite.testSites)
	placementTime := time.Since(startTime)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)

	// Placement decision should be fast (part of the 10-minute target)
	assert.Less(suite.T(), placementTime, 30*time.Second, "Placement decision should be fast")

	// Verify QoS targets are met for thesis scenarios
	testCases := []struct {
		nfIndex          int
		maxLatencyTarget float64
		minThroughputTarget float64
	}{
		{0, 6.3, 4.57},   // emergency-upf: critical latency and throughput
		{1, 15.7, 2.77},  // video-streaming-upf: video streaming targets
		{2, 16.1, 0.93},  // iot-aggregator: IoT targets
	}

	for _, tc := range testCases {
		testNF := suite.testNFs[tc.nfIndex]
		result, err := suite.placement.Place(testNF, suite.testSites)
		require.NoError(suite.T(), err)

		// Check that selected site can meet thesis targets
		assert.LessOrEqual(suite.T(), result.Site.NetworkProfile.BaseLatencyMs, tc.maxLatencyTarget,
			"Site should meet latency target for %s", testNF.ID)
		assert.GreaterOrEqual(suite.T(), result.Site.NetworkProfile.MaxThroughputMbps, tc.minThroughputTarget,
			"Site should meet throughput target for %s", testNF.ID)
	}
}

// Benchmark placement performance
func (suite *PlacementTestSuite) TestPlacementPerformanceBenchmark() {
	nf := suite.testNFs[1] // video-streaming-upf
	iterations := 1000

	start := time.Now()

	for i := 0; i < iterations; i++ {
		result, err := suite.placement.Place(nf, suite.testSites)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
	}

	totalTime := time.Since(start)
	avgTime := totalTime / time.Duration(iterations)

	// Average placement time should be reasonable
	assert.Less(suite.T(), avgTime, 10*time.Millisecond,
		"Average placement time should be less than 10ms")

	suite.T().Logf("Placement performance: %d iterations in %v (avg: %v)",
		iterations, totalTime, avgTime)
}

// Test site metrics integration
func (suite *PlacementTestSuite) TestMetricsIntegration() {
	// Update metrics for one site to show high utilization
	highUtilizationMetrics := &SiteMetrics{
		Timestamp:              time.Now(),
		CPUUtilization:         90.0, // High utilization
		MemoryUtilization:      85.0, // High utilization
		AvailableBandwidthMbps: 1000,
		CurrentLatencyMs:       1.0,
		ActiveNFs:              10,
	}
	suite.metricsProvider.SetMetrics(suite.testSites[1].ID, highUtilizationMetrics)

	nf := suite.testNFs[0] // emergency-upf
	result, err := suite.placement.Place(nf, suite.testSites)

	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)

	// Should prefer sites with lower utilization
	selectedMetrics, _ := suite.metricsProvider.GetMetrics(result.Site.ID)
	if selectedMetrics != nil {
		avgUtilization := (selectedMetrics.CPUUtilization + selectedMetrics.MemoryUtilization) / 2.0
		assert.Less(suite.T(), avgUtilization, 90.0, "Should prefer less utilized sites")
	}
}

// Run the test suite
func TestPlacementTestSuite(t *testing.T) {
	suite.Run(t, new(PlacementTestSuite))
}

// Table-driven tests for various placement scenarios
func TestPlacementScenarios(t *testing.T) {
	testCases := []struct {
		name        string
		nf          *NetworkFunction
		expectedErr bool
		assertions  func(t *testing.T, result *PlacementDecision)
	}{
		{
			name: "Emergency URLLC Service",
			nf: &NetworkFunction{
				ID:   "emergency",
				Type: "UPF",
				Requirements: ResourceRequirements{
					MinCPUCores:      50,
					MinMemoryGB:      200,
					MinStorageGB:     500,
					MinBandwidthMbps: 1000,
				},
				QoSRequirements: QoSRequirements{
					MaxLatencyMs:      1.0,
					MinThroughputMbps: 100,
					MaxPacketLossRate: 0.001,
					MaxJitterMs:       0.5,
				},
			},
			expectedErr: false,
			assertions: func(t *testing.T, result *PlacementDecision) {
				assert.NotNil(t, result.Site)
				assert.LessOrEqual(t, result.Site.NetworkProfile.BaseLatencyMs, 1.0)
				assert.GreaterOrEqual(t, result.Site.NetworkProfile.MaxThroughputMbps, 100.0)
			},
		},
		{
			name: "Video Streaming eMBB",
			nf: &NetworkFunction{
				ID:   "streaming",
				Type: "UPF",
				Requirements: ResourceRequirements{
					MinCPUCores:      100,
					MinMemoryGB:      400,
					MinStorageGB:     1000,
					MinBandwidthMbps: 2000,
				},
				QoSRequirements: QoSRequirements{
					MaxLatencyMs:      20.0,
					MinThroughputMbps: 200,
					MaxPacketLossRate: 0.001,
					MaxJitterMs:       2.0,
				},
			},
			expectedErr: false,
			assertions: func(t *testing.T, result *PlacementDecision) {
				assert.NotNil(t, result.Site)
				assert.GreaterOrEqual(t, result.Site.NetworkProfile.MaxThroughputMbps, 200.0)
			},
		},
		{
			name: "IoT mMTC Service",
			nf: &NetworkFunction{
				ID:   "iot",
				Type: "SMF",
				Requirements: ResourceRequirements{
					MinCPUCores:      10,
					MinMemoryGB:      50,
					MinStorageGB:     100,
					MinBandwidthMbps: 50,
				},
				QoSRequirements: QoSRequirements{
					MaxLatencyMs:      100.0,
					MinThroughputMbps: 5,
					MaxPacketLossRate: 0.01,
					MaxJitterMs:       5.0,
				},
			},
			expectedErr: false,
			assertions: func(t *testing.T, result *PlacementDecision) {
				assert.NotNil(t, result.Site)
				// Any site should be able to handle low-priority IoT workloads
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metricsProvider := NewMockMetricsProvider()
			placement := NewIntelligentPlacementPolicy(metricsProvider)

			// Setup test sites
			testSites := []*Site{
				{
					ID:   "test-edge",
					Name: "Test Edge Site",
					Type: CloudTypeEdge,
					Location: Location{Region: "us-east-1"},
					Capacity: ResourceCapacity{
						CPUCores: 500, MemoryGB: 2000, StorageGB: 5000, BandwidthMbps: 5000,
					},
					NetworkProfile: NetworkProfile{
						BaseLatencyMs: 1.0, MaxThroughputMbps: 5000, PacketLossRate: 0.0001, JitterMs: 0.5,
					},
					Available: true,
				},
				{
					ID:   "test-central",
					Name: "Test Central Site",
					Type: CloudTypeCentral,
					Location: Location{Region: "us-east-1"},
					Capacity: ResourceCapacity{
						CPUCores: 1000, MemoryGB: 4000, StorageGB: 10000, BandwidthMbps: 10000,
					},
					NetworkProfile: NetworkProfile{
						BaseLatencyMs: 10.0, MaxThroughputMbps: 10000, PacketLossRate: 0.001, JitterMs: 1.0,
					},
					Available: true,
				},
			}

			// Setup metrics
			for _, site := range testSites {
				metrics := &SiteMetrics{
					Timestamp: time.Now(), CPUUtilization: 20.0, MemoryUtilization: 25.0,
					AvailableBandwidthMbps: site.Capacity.BandwidthMbps * 0.8,
					CurrentLatencyMs: site.NetworkProfile.BaseLatencyMs, ActiveNFs: 2,
				}
				metricsProvider.SetMetrics(site.ID, metrics)
			}

			result, err := placement.Place(tc.nf, testSites)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tc.assertions != nil {
					tc.assertions(t, result)
				}
			}
		})
	}
}