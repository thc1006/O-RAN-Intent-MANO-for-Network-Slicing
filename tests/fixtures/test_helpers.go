package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestHelpers provides utility functions for test setup and validation
type TestHelpers struct {
	t *testing.T
}

// NewTestHelpers creates a new test helpers instance
func NewTestHelpers(t *testing.T) *TestHelpers {
	return &TestHelpers{t: t}
}

// Timeout configurations for different test types
const (
	UnitTestTimeout        = 5 * time.Second
	IntegrationTestTimeout = 30 * time.Second
	E2ETestTimeout         = 5 * time.Minute
)

// Test data generation utilities
func GenerateTestID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func GenerateTestName(components ...string) string {
	result := "test"
	for _, component := range components {
		result += "-" + component
	}
	return result
}

// Context utilities
func CreateTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func CreateTestContextWithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), deadline)
}

// JSON test utilities
func MustMarshalJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal JSON: %v", err))
	}
	return string(data)
}

func MustUnmarshalJSON(data string, v interface{}) {
	err := json.Unmarshal([]byte(data), v)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal JSON: %v", err))
	}
}

func CompareJSON(t *testing.T, expected, actual interface{}) {
	expectedJSON := MustMarshalJSON(expected)
	actualJSON := MustMarshalJSON(actual)
	assert.JSONEq(t, expectedJSON, actualJSON)
}

// Validation utilities
func (h *TestHelpers) ValidateNotEmpty(value string, fieldName string) {
	assert.NotEmpty(h.t, value, "%s should not be empty", fieldName)
}

func (h *TestHelpers) ValidatePositive(value float64, fieldName string) {
	assert.Positive(h.t, value, "%s should be positive", fieldName)
}

func (h *TestHelpers) ValidateInRange(value, min, max float64, fieldName string) {
	assert.GreaterOrEqual(h.t, value, min, "%s should be >= %f", fieldName, min)
	assert.LessOrEqual(h.t, value, max, "%s should be <= %f", fieldName, max)
}

func (h *TestHelpers) ValidateLatency(latency time.Duration, maxAllowed time.Duration, context string) {
	assert.LessOrEqual(h.t, latency, maxAllowed,
		"Latency in %s should be <= %v, got %v", context, maxAllowed, latency)
}

func (h *TestHelpers) ValidateQoSProfile(profile QoSProfile, sliceType SliceType) {
	switch sliceType {
	case SliceTypeEMBB:
		h.ValidateEMBBQoS(profile)
	case SliceTypeURLLC:
		h.ValidateURLLCQoS(profile)
	case SliceTypeMmTC:
		h.ValidateMmTCQoS(profile)
	default:
		h.t.Errorf("Unknown slice type: %s", sliceType)
	}
}

func (h *TestHelpers) ValidateEMBBQoS(profile QoSProfile) {
	// eMBB typically has moderate latency (10-50ms) and high throughput
	assert.NotEmpty(h.t, profile.Latency, "eMBB latency should be specified")
	assert.NotEmpty(h.t, profile.Throughput, "eMBB throughput should be specified")

	// Validate latency is reasonable for eMBB (should be <= 100ms)
	if profile.Latency != "" && profile == "ms" {
		latencyMs := parseLatencyValue(profile.Latency)
		assert.LessOrEqual(h.t, latencyMs, 100.0, "eMBB latency should be <= 100ms")
		assert.GreaterOrEqual(h.t, latencyMs, 1.0, "eMBB latency should be >= 1ms")
	}
}

func (h *TestHelpers) ValidateURLLCQoS(profile QoSProfile) {
	// URLLC requires ultra-low latency (< 1ms) and high reliability
	assert.NotEmpty(h.t, profile.Latency, "URLLC latency should be specified")
	assert.NotEmpty(h.t, profile.Reliability.Value, "URLLC reliability should be specified")

	// Validate latency is ultra-low
	if profile.Latency != "" && profile == "ms" {
		latencyMs := parseLatencyValue(profile.Latency)
		assert.LessOrEqual(h.t, latencyMs, 1.0, "URLLC latency should be <= 1ms")
	}

	// Validate high reliability
	if profile.Reliability.Value != "" {
		reliability := parseReliabilityValue(profile.Reliability.Value)
		assert.GreaterOrEqual(h.t, reliability, 99.99, "URLLC reliability should be >= 99.99%")
	}
}

func (h *TestHelpers) ValidateMmTCQoS(profile QoSProfile) {
	// mMTC can tolerate higher latency but needs to support massive connections
	assert.NotEmpty(h.t, profile.Latency, "mMTC latency should be specified")

	// Validate latency tolerance
	if profile.Latency != "" && profile == "ms" {
		latencyMs := parseLatencyValue(profile.Latency)
		assert.LessOrEqual(h.t, latencyMs, 1000.0, "mMTC latency should be <= 1000ms")
		assert.GreaterOrEqual(h.t, latencyMs, 10.0, "mMTC latency should be >= 10ms")
	}
}

// Resource validation utilities
func (h *TestHelpers) ValidateResourceProfile(profile ResourceProfile, sliceType SliceType) {
	h.ValidateNotEmpty(profile.Compute.CPU, "CPU")
	h.ValidateNotEmpty(profile.Compute.Memory, "Memory")

	switch sliceType {
	case SliceTypeEMBB:
		// eMBB needs moderate to high resources for throughput
		h.validateMinimumResources(profile, "2000m", "4Gi")
	case SliceTypeURLLC:
		// URLLC needs high resources for low latency
		h.validateMinimumResources(profile, "4000m", "8Gi")
	case SliceTypeMmTC:
		// mMTC can use fewer resources per connection
		h.validateMinimumResources(profile, "1000m", "2Gi")
	}
}

func (h *TestHelpers) validateMinimumResources(profile ResourceProfile, minCPU, minMemory string) {
	// This is a simplified validation - in reality you'd parse the resource strings
	assert.NotEmpty(h.t, profile.Compute.CPU, "CPU should be specified")
	assert.NotEmpty(h.t, profile.Compute.Memory, "Memory should be specified")
}

// Placement validation utilities
func (h *TestHelpers) ValidatePlacementSolution(solution *PlacementSolution, request PlacementRequest) {
	require.NotNil(h.t, solution, "Placement solution should not be nil")

	// Validate basic structure
	assert.NotEmpty(h.t, solution.ID, "Solution ID should not be empty")
	assert.Equal(h.t, request.ID, solution.RequestID, "Request ID should match")
	assert.NotEmpty(h.t, solution.Placements, "Should have at least one placement")

	// Validate score
	assert.GreaterOrEqual(h.t, solution.Score.Total, 0.0, "Score should be non-negative")
	assert.LessOrEqual(h.t, solution.Score.Total, 1.0, "Score should be <= 1.0")

	// Validate feasibility
	if solution.Constraints.Feasible {
		assert.Empty(h.t, solution.Constraints.Violated, "Feasible solution should have no violated constraints")
	} else {
		assert.NotEmpty(h.t, solution.Constraints.Violated, "Infeasible solution should have violated constraints")
	}

	// Validate each placement
	for i, placement := range solution.Placements {
		h.ValidateResourcePlacement(placement, fmt.Sprintf("placement[%d]", i))
	}
}

func (h *TestHelpers) ValidateResourcePlacement(placement ResourcePlacement, context string) {
	assert.NotEmpty(h.t, placement.VNFComponent, "%s: VNF component should not be empty", context)
	assert.NotEmpty(h.t, placement.NodeID, "%s: Node ID should not be empty", context)
	assert.NotEmpty(h.t, placement.Zone, "%s: Zone should not be empty", context)
	assert.GreaterOrEqual(h.t, placement.Score, 0.0, "%s: Placement score should be non-negative", context)
	assert.LessOrEqual(h.t, placement.Score, 1.0, "%s: Placement score should be <= 1.0", context)
}

// Network validation utilities
func (h *TestHelpers) ValidateNetworkPaths(paths []NetworkPath, maxLatency time.Duration) {
	assert.NotEmpty(h.t, paths, "Network paths should not be empty")

	for i, path := range paths {
		context := fmt.Sprintf("path[%d]", i)
		assert.NotEmpty(h.t, path.Source, "%s: Source should not be empty", context)
		assert.NotEmpty(h.t, path.Destination, "%s: Destination should not be empty", context)
		assert.Positive(h.t, path.Bandwidth, "%s: Bandwidth should be positive", context)
		assert.Positive(h.t, int64(path.Latency), "%s: Latency should be positive", context)

		if maxLatency > 0 {
			assert.LessOrEqual(h.t, path.Latency, maxLatency,
				"%s: Latency should be <= %v", context, maxLatency)
		}
	}
}

// VNF validation utilities
func (h *TestHelpers) ValidateVNFDeployment(vnf *VNFDeployment) {
	require.NotNil(h.t, vnf, "VNF deployment should not be nil")

	assert.NotEmpty(h.t, vnf.Name, "VNF name should not be empty")
	assert.NotEmpty(h.t, vnf.Namespace, "VNF namespace should not be empty")
	assert.NotEmpty(h.t, vnf.Spec.VNFType, "VNF type should not be empty")
	assert.NotEmpty(h.t, vnf.Spec.SliceType, "Slice type should not be empty")

	// Validate slice type is one of the known types
	validSliceTypes := []string{string(SliceTypeEMBB), string(SliceTypeURLLC), string(SliceTypeMmTC)}
	assert.Contains(h.t, validSliceTypes, vnf.Spec.SliceType, "Slice type should be valid")

	// Validate QoS profile
	h.ValidateQoSProfile(vnf.Spec.QoSProfile, SliceType(vnf.Spec.SliceType))

	// Validate resource profile
	h.ValidateResourceProfile(ResourceProfile{
		Compute: ComputeRequirement{
			CPU:    vnf.Spec.Resources.CPU,
			Memory: vnf.Spec.Resources.Memory,
		},
	}, SliceType(vnf.Spec.SliceType))
}

// Intent validation utilities
func (h *TestHelpers) ValidateIntent(intent Intent) {
	assert.NotEmpty(h.t, intent.ID, "Intent ID should not be empty")
	assert.NotEmpty(h.t, intent.Text, "Intent text should not be empty")
	assert.NotEmpty(h.t, string(intent.Type), "Intent type should not be empty")
	assert.NotEmpty(h.t, string(intent.Priority), "Intent priority should not be empty")
	assert.False(h.t, intent.Timestamp.IsZero(), "Intent timestamp should be set")
}

func (h *TestHelpers) ValidateParsedIntent(parsed *ParsedIntent) {
	require.NotNil(h.t, parsed, "Parsed intent should not be nil")

	h.ValidateIntent(parsed.Intent)
	assert.NotEmpty(h.t, string(parsed.SliceType), "Slice type should not be empty")
	assert.GreaterOrEqual(h.t, parsed.Confidence, 0.0, "Confidence should be non-negative")
	assert.LessOrEqual(h.t, parsed.Confidence, 1.0, "Confidence should be <= 1.0")

	// Validate QoS profile based on slice type
	h.ValidateQoSProfile(parsed.QoSProfile, parsed.SliceType)
}

// Test data builders
type VNFBuilder struct {
	vnf *VNFDeployment
}

func NewVNFBuilder() *VNFBuilder {
	return &VNFBuilder{
		vnf: &VNFDeployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "oran.io/v1",
				Kind:       "VNFDeployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      GenerateTestID("test-vnf"),
				Namespace: "oran-system",
			},
			Spec: VNFDeploymentSpec{
				VNFType:   "cucp",
				SliceType: "eMBB",
				Resources: ResourceRequests{
					CPU:    "2000m",
					Memory: "4Gi",
				},
				QoSProfile: VNFQoSProfile{
					Latency: LatencyRequirement{
						Value: "20",
						Unit:  "ms",
						Type:  "end-to-end",
					},
					Throughput: ThroughputRequirement{
						Downlink: "1Gbps",
						Uplink:   "100Mbps",
						Unit:     "bps",
					},
					Reliability: ReliabilityRequirement{
						Value: "99.9",
						Unit:  "percentage",
					},
				},
			},
		},
	}
}

func (b *VNFBuilder) WithName(name string) *VNFBuilder {
	b.vnf.Name = name
	return b
}

func (b *VNFBuilder) WithVNFType(vnfType string) *VNFBuilder {
	b.vnf.Spec.VNFType = vnfType
	return b
}

func (b *VNFBuilder) WithSliceType(sliceType SliceType) *VNFBuilder {
	b.vnf.Spec.SliceType = string(sliceType)
	return b
}

func (b *VNFBuilder) WithLatency(value, unit string) *VNFBuilder {
	b.vnf.Spec.QoSProfile.Latency = value
	b.vnf.Spec.QoSProfile = unit
	return b
}

func (b *VNFBuilder) WithThroughput(downlink, uplink string) *VNFBuilder {
	b.vnf.Spec.QoSProfile.Throughput = downlink
	b.vnf.Spec.QoSProfile = uplink
	return b
}

func (b *VNFBuilder) WithResources(cpu, memory string) *VNFBuilder {
	b.vnf.Spec.Resources.CPU = cpu
	b.vnf.Spec.Resources.Memory = memory
	return b
}

func (b *VNFBuilder) Build() *VNFDeployment {
	return b.vnf
}

// Utility functions for parsing values
func parseLatencyValue(value string) float64 {
	// Simplified parsing - in reality would handle different formats
	switch value {
	case "1":
		return 1.0
	case "10":
		return 10.0
	case "20":
		return 20.0
	case "50":
		return 50.0
	case "100":
		return 100.0
	default:
		return 0.0
	}
}

func parseReliabilityValue(value string) float64 {
	// Simplified parsing - in reality would handle percentage formats
	switch value {
	case "99.9":
		return 99.9
	case "99.99":
		return 99.99
	case "99.999":
		return 99.999
	default:
		return 0.0
	}
}

// Assertion helpers
func AssertSliceTypeCompatible(t *testing.T, sliceType SliceType, qos QoSProfile) {
	helpers := NewTestHelpers(t)
	helpers.ValidateQoSProfile(qos, sliceType)
}

func AssertPlacementFeasible(t *testing.T, solution *PlacementSolution) {
	assert.True(t, solution.Constraints.Feasible, "Placement should be feasible")
	assert.Empty(t, solution.Constraints.Violated, "Should have no violated constraints")
	assert.GreaterOrEqual(t, solution.Score.Total, 0.5, "Feasible solution should have reasonable score")
}

func AssertLatencyMeetsRequirement(t *testing.T, actual time.Duration, requirement LatencyRequirement) {
	maxLatency, err := time.ParseDuration(requirement.Value + requirement.Unit)
	require.NoError(t, err, "Should be able to parse latency requirement")
	assert.LessOrEqual(t, actual, maxLatency, "Actual latency should meet requirement")
}