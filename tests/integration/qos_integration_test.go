package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"../fixtures"
)

// TestValidVNFDeploymentFixture tests that ValidVNFDeployment creates valid objects
func TestValidVNFDeploymentFixture(t *testing.T) {
	vnf := fixtures.ValidVNFDeployment()
	require.NotNil(t, vnf)

	helpers := fixtures.NewTestHelpers(t)
	helpers.ValidateVNFDeployment(vnf)

	// Specific validation for the fixture
	assert.Equal(t, "test-vnf", vnf.Name)
	assert.Equal(t, "oran-system", vnf.Namespace)
	assert.Equal(t, "cucp", vnf.Spec.VNFType)
	assert.Equal(t, "eMBB", vnf.Spec.SliceType)
	assert.Equal(t, "2000m", vnf.Spec.Resources.CPU)
	assert.Equal(t, "4Gi", vnf.Spec.Resources.Memory)
	assert.Equal(t, "10ms", vnf.Spec.QoSProfile.Latency)
	assert.Equal(t, "1Gbps", vnf.Spec.QoSProfile.Throughput)
	assert.Equal(t, "99.99%", vnf.Spec.QoSProfile.Reliability)
}

// TestHelpersWithBothTypes tests that helpers work with both QoS types
func TestHelpersWithBothTypes(t *testing.T) {
	helpers := fixtures.NewTestHelpers(t)

	// Test with VNFQoSProfile (through conversion)
	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	qosProfile := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
	assert.NotPanics(t, func() {
		helpers.ValidateQoSProfile(qosProfile, fixtures.SliceTypeEMBB)
	})

	// Test with direct QoSProfile
	directProfile := fixtures.QoSProfile{
		Latency: fixtures.LatencyRequirement{
			Value: "20",
			Unit:  "ms",
			Type:  "end-to-end",
		},
		Throughput: fixtures.ThroughputRequirement{
			Downlink: "1Gbps",
			Uplink:   "100Mbps",
			Unit:     "bps",
		},
		Reliability: fixtures.ReliabilityRequirement{
			Value: "99.9",
			Unit:  "percentage",
		},
	}

	assert.NotPanics(t, func() {
		helpers.ValidateQoSProfile(directProfile, fixtures.SliceTypeEMBB)
	})
}

// TestVNFBuilderPattern tests that the builder pattern works correctly
func TestVNFBuilderPattern(t *testing.T) {
	vnf := fixtures.NewVNFBuilder().
		WithName("test-builder-vnf").
		WithVNFType("cuup").
		WithSliceType(fixtures.SliceTypeURLLC).
		WithLatency("1", "ms").
		WithThroughput("100Mbps", "50Mbps").
		WithResources("4000m", "8Gi").
		Build()

	require.NotNil(t, vnf)

	// Validate the built VNF
	assert.Equal(t, "test-builder-vnf", vnf.Name)
	assert.Equal(t, "cuup", vnf.Spec.VNFType)
	assert.Equal(t, "URLLC", vnf.Spec.SliceType)
	assert.Equal(t, "1ms", vnf.Spec.QoSProfile.Latency)
	assert.Equal(t, "100Mbps", vnf.Spec.QoSProfile.Throughput)
	assert.Equal(t, "4000m", vnf.Spec.Resources.CPU)
	assert.Equal(t, "8Gi", vnf.Spec.Resources.Memory)

	// Validate with helpers
	helpers := fixtures.NewTestHelpers(t)
	helpers.ValidateVNFDeployment(vnf)
}

// TestEMBBFixtureValidation tests eMBB fixture with new QoS structure
func TestEMBBFixtureValidation(t *testing.T) {
	vnf := fixtures.EMBBVNFDeployment()
	require.NotNil(t, vnf)

	helpers := fixtures.NewTestHelpers(t)

	// Convert VNF QoS to structured QoS for validation
	qosProfile := fixtures.ConvertVNFQoSProfileToQoSProfile(vnf.Spec.QoSProfile)

	// Validate eMBB specific requirements
	helpers.ValidateEMBBQoS(qosProfile)

	// Check eMBB characteristics
	assert.Equal(t, "eMBB", vnf.Spec.SliceType)

	// Latency should be moderate (10-50ms for eMBB)
	latencyMs := fixtures.ParseLatencyValue(qosProfile.Latency.Value)
	assert.LessOrEqual(t, latencyMs, 100.0)
	assert.GreaterOrEqual(t, latencyMs, 1.0)

	// Throughput should be high for eMBB
	assert.Contains(t, qosProfile.Throughput.Downlink, "Gbps")
}

// TestURLLCFixtureValidation tests URLLC fixture with new QoS structure
func TestURLLCFixtureValidation(t *testing.T) {
	vnf := fixtures.URLLCVNFDeployment()
	require.NotNil(t, vnf)

	helpers := fixtures.NewTestHelpers(t)

	// Convert VNF QoS to structured QoS for validation
	qosProfile := fixtures.ConvertVNFQoSProfileToQoSProfile(vnf.Spec.QoSProfile)

	// Validate URLLC specific requirements
	helpers.ValidateURLLCQoS(qosProfile)

	// Check URLLC characteristics
	assert.Equal(t, "URLLC", vnf.Spec.SliceType)

	// Latency should be ultra-low (≤1ms for URLLC)
	latencyMs := fixtures.ParseLatencyValue(qosProfile.Latency.Value)
	assert.LessOrEqual(t, latencyMs, 1.0)

	// Reliability should be very high (≥99.99%)
	reliability := fixtures.ParseReliabilityValue(qosProfile.Reliability.Value)
	assert.GreaterOrEqual(t, reliability, 99.99)
}

// TestMmTCFixtureValidation tests mMTC fixture with new QoS structure
func TestMmTCFixtureValidation(t *testing.T) {
	vnf := fixtures.MMTCVNFDeployment()
	require.NotNil(t, vnf)

	helpers := fixtures.NewTestHelpers(t)

	// Convert VNF QoS to structured QoS for validation
	qosProfile := fixtures.ConvertVNFQoSProfileToQoSProfile(vnf.Spec.QoSProfile)

	// Validate mMTC specific requirements
	helpers.ValidateMmTCQoS(qosProfile)

	// Check mMTC characteristics
	assert.Equal(t, "mMTC", vnf.Spec.SliceType)

	// Latency can be higher for mMTC (10-1000ms)
	latencyMs := fixtures.ParseLatencyValue(qosProfile.Latency.Value)
	assert.LessOrEqual(t, latencyMs, 1000.0)
	assert.GreaterOrEqual(t, latencyMs, 10.0)

	// Throughput can be lower for mMTC
	assert.True(t, len(qosProfile.Throughput.Downlink) > 0)
}

// TestQoSProfileValidation tests QoS profile validation across slice types
func TestQoSProfileValidation(t *testing.T) {
	testCases := []struct {
		sliceType fixtures.SliceType
		qosProfile fixtures.QoSProfile
		shouldPass bool
	}{
		{
			sliceType: fixtures.SliceTypeEMBB,
			qosProfile: fixtures.QoSProfile{
				Latency: fixtures.LatencyRequirement{
					Value: "50",
					Unit:  "ms",
					Type:  "end-to-end",
				},
				Throughput: fixtures.ThroughputRequirement{
					Downlink: "1Gbps",
					Uplink:   "100Mbps",
					Unit:     "bps",
				},
				Reliability: fixtures.ReliabilityRequirement{
					Value: "99.9",
					Unit:  "percentage",
				},
			},
			shouldPass: true,
		},
		{
			sliceType: fixtures.SliceTypeURLLC,
			qosProfile: fixtures.QoSProfile{
				Latency: fixtures.LatencyRequirement{
					Value: "1",
					Unit:  "ms",
					Type:  "end-to-end",
				},
				Throughput: fixtures.ThroughputRequirement{
					Downlink: "100Mbps",
					Uplink:   "50Mbps",
					Unit:     "bps",
				},
				Reliability: fixtures.ReliabilityRequirement{
					Value: "99.999",
					Unit:  "percentage",
				},
			},
			shouldPass: true,
		},
		{
			sliceType: fixtures.SliceTypeMmTC,
			qosProfile: fixtures.QoSProfile{
				Latency: fixtures.LatencyRequirement{
					Value: "100",
					Unit:  "ms",
					Type:  "end-to-end",
				},
				Throughput: fixtures.ThroughputRequirement{
					Downlink: "10Mbps",
					Uplink:   "1Mbps",
					Unit:     "bps",
				},
				Reliability: fixtures.ReliabilityRequirement{
					Value: "99.9",
					Unit:  "percentage",
				},
			},
			shouldPass: true,
		},
	}

	helpers := fixtures.NewTestHelpers(t)

	for _, tc := range testCases {
		t.Run(string(tc.sliceType), func(t *testing.T) {
			if tc.shouldPass {
				assert.NotPanics(t, func() {
					helpers.ValidateQoSProfile(tc.qosProfile, tc.sliceType)
				})
			} else {
				// Test would fail validation
				assert.Panics(t, func() {
					helpers.ValidateQoSProfile(tc.qosProfile, tc.sliceType)
				})
			}
		})
	}
}

// TestResourceProfileCompatibility tests resource profile compatibility
func TestResourceProfileCompatibility(t *testing.T) {
	helpers := fixtures.NewTestHelpers(t)

	// Test different slice types have appropriate resource requirements
	testCases := []struct {
		sliceType fixtures.SliceType
		resources fixtures.ResourceProfile
	}{
		{
			sliceType: fixtures.SliceTypeEMBB,
			resources: fixtures.ResourceProfile{
				Compute: fixtures.ComputeRequirement{
					CPU:    "4000m",
					Memory: "8Gi",
					Cores:  4,
				},
				Network: fixtures.NetworkRequirement{
					Bandwidth: "1Gbps",
				},
			},
		},
		{
			sliceType: fixtures.SliceTypeURLLC,
			resources: fixtures.ResourceProfile{
				Compute: fixtures.ComputeRequirement{
					CPU:    "8000m",
					Memory: "16Gi",
					Cores:  8,
				},
				Network: fixtures.NetworkRequirement{
					Bandwidth: "100Mbps",
				},
			},
		},
		{
			sliceType: fixtures.SliceTypeMmTC,
			resources: fixtures.ResourceProfile{
				Compute: fixtures.ComputeRequirement{
					CPU:    "1000m",
					Memory: "2Gi",
					Cores:  1,
				},
				Network: fixtures.NetworkRequirement{
					Bandwidth: "10Mbps",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.sliceType), func(t *testing.T) {
			assert.NotPanics(t, func() {
				helpers.ValidateResourceProfile(tc.resources, tc.sliceType)
			})
		})
	}
}

// TestPlacementProfileCompatibility tests placement profile compatibility
func TestPlacementProfileCompatibility(t *testing.T) {
	// Test that placement validation works with the new types
	solution := &fixtures.PlacementSolution{
		ID:        "test-solution-1",
		RequestID: "test-request-1",
		Placements: []fixtures.ResourcePlacement{
			{
				VNFComponent: "cucp",
				NodeID:       "node-1",
				Zone:         "zone-a",
				Score:        0.95,
			},
		},
		Score: fixtures.PlacementScore{
			Total: 0.95,
		},
		Constraints: fixtures.ConstraintResult{
			Feasible: true,
			Violated: []string{},
		},
	}

	request := fixtures.PlacementRequest{
		ID: "test-request-1",
	}

	helpers := fixtures.NewTestHelpers(t)
	assert.NotPanics(t, func() {
		helpers.ValidatePlacementSolution(solution, request)
	})
}

// TestJSONSerialization tests JSON marshal/unmarshal with new types
func TestJSONSerialization(t *testing.T) {
	// Test VNF deployment serialization
	vnf := fixtures.ValidVNFDeployment()

	// Marshal to JSON
	jsonData := fixtures.MustMarshalJSON(vnf)
	assert.NotEmpty(t, jsonData)

	// Unmarshal back
	var unmarshaledVNF fixtures.VNFDeployment
	fixtures.MustUnmarshalJSON(jsonData, &unmarshaledVNF)

	// Compare
	assert.Equal(t, vnf.Name, unmarshaledVNF.Name)
	assert.Equal(t, vnf.Spec.VNFType, unmarshaledVNF.Spec.VNFType)
	assert.Equal(t, vnf.Spec.SliceType, unmarshaledVNF.Spec.SliceType)
	assert.Equal(t, vnf.Spec.QoSProfile.Latency, unmarshaledVNF.Spec.QoSProfile.Latency)

	// Test structured QoS profile serialization
	qosProfile := fixtures.QoSProfile{
		Latency: fixtures.LatencyRequirement{
			Value: "20",
			Unit:  "ms",
			Type:  "end-to-end",
		},
		Throughput: fixtures.ThroughputRequirement{
			Downlink: "1Gbps",
			Uplink:   "100Mbps",
			Unit:     "bps",
		},
		Reliability: fixtures.ReliabilityRequirement{
			Value: "99.9",
			Unit:  "percentage",
		},
	}

	qosJSON := fixtures.MustMarshalJSON(qosProfile)
	assert.NotEmpty(t, qosJSON)

	var unmarshaledQoS fixtures.QoSProfile
	fixtures.MustUnmarshalJSON(qosJSON, &unmarshaledQoS)

	assert.Equal(t, qosProfile.Latency.Value, unmarshaledQoS.Latency.Value)
	assert.Equal(t, qosProfile.Throughput.Downlink, unmarshaledQoS.Throughput.Downlink)
	assert.Equal(t, qosProfile.Reliability.Value, unmarshaledQoS.Reliability.Value)
}

// TestConcurrentQoSConversion tests concurrent access to QoS conversion
func TestConcurrentQoSConversion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vnfProfile := fixtures.VNFQoSProfile{
		Latency:     "20ms",
		Throughput:  "1Gbps",
		Reliability: "99.9%",
	}

	// Run conversions concurrently
	numGoroutines := 100
	results := make(chan fixtures.QoSProfile, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("panic: %v", r)
				}
			}()

			result := fixtures.ConvertVNFQoSProfileToQoSProfile(vnfProfile)
			results <- result
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			assert.Equal(t, "20", result.Latency.Value)
			assert.Equal(t, "1Gbps", result.Throughput.Downlink)
			successCount++
		case err := <-errors:
			t.Errorf("Concurrent conversion failed: %v", err)
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}

	assert.Equal(t, numGoroutines, successCount, "All concurrent conversions should succeed")
}

// TestAPICompatibility tests that APIs work with new QoS types
func TestAPICompatibility(t *testing.T) {
	// This would test API endpoints, but for now just validate types
	vnf := fixtures.ValidVNFDeployment()
	qosProfile := fixtures.ConvertVNFQoSProfileToQoSProfile(vnf.Spec.QoSProfile)

	// Ensure API-compatible JSON structure
	jsonData := fixtures.MustMarshalJSON(qosProfile)
	assert.Contains(t, jsonData, "latency")
	assert.Contains(t, jsonData, "throughput")
	assert.Contains(t, jsonData, "reliability")
	assert.Contains(t, jsonData, "value")
	assert.Contains(t, jsonData, "unit")
}

// TestDBModelCompatibility tests database model compatibility
func TestDBModelCompatibility(t *testing.T) {
	// Test that models can be stored and retrieved from database
	// For now, just test serialization compatibility

	vnf := fixtures.ValidVNFDeployment()

	// Test that all required fields are present for database storage
	assert.NotEmpty(t, vnf.Name)
	assert.NotEmpty(t, vnf.Namespace)
	assert.NotEmpty(t, vnf.Spec.VNFType)
	assert.NotEmpty(t, vnf.Spec.SliceType)
	assert.NotEmpty(t, vnf.Spec.QoSProfile.Latency)
	assert.NotEmpty(t, vnf.Spec.QoSProfile.Throughput)
	assert.NotEmpty(t, vnf.Spec.QoSProfile.Reliability)

	// Test QoS profile structure
	qosProfile := fixtures.ConvertVNFQoSProfileToQoSProfile(vnf.Spec.QoSProfile)
	assert.NotEmpty(t, qosProfile.Latency.Value)
	assert.NotEmpty(t, qosProfile.Latency.Unit)
	assert.NotEmpty(t, qosProfile.Throughput.Downlink)
	assert.NotEmpty(t, qosProfile.Reliability.Value)

	// Ensure JSON compatibility for database storage
	jsonData := fixtures.MustMarshalJSON(qosProfile)
	var restored fixtures.QoSProfile
	fixtures.MustUnmarshalJSON(jsonData, &restored)

	assert.Equal(t, qosProfile.Latency.Value, restored.Latency.Value)
	assert.Equal(t, qosProfile.Throughput.Downlink, restored.Throughput.Downlink)
	assert.Equal(t, qosProfile.Reliability.Value, restored.Reliability.Value)
}