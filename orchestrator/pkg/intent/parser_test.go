package intent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/fixtures"
)

// IntentParser - the parser we're testing (not implemented yet)
type IntentParser struct {
	NLPEngine    NLPEngine
	Validator    IntentValidator
	QoSMapper    QoSMapper
	Confidence   ConfidenceCalculator
}

// Interfaces that need to be implemented
type NLPEngine interface {
	ParseText(ctx context.Context, text string) (NLPResult, error)
	ExtractEntities(ctx context.Context, text string) ([]Entity, error)
	ClassifyIntent(ctx context.Context, text string) (IntentClassification, error)
}

type IntentValidator interface {
	ValidateIntent(intent fixtures.Intent) fixtures.ValidationResult
	ValidateQoSProfile(profile fixtures.QoSProfile) error
	ValidateResourceProfile(profile fixtures.ResourceProfile) error
	ValidatePlacementProfile(profile fixtures.PlacementProfile) error
}

type QoSMapper interface {
	MapToQoSProfile(sliceType fixtures.SliceType, constraints map[string]interface{}) (fixtures.QoSProfile, error)
	MapToResourceProfile(sliceType fixtures.SliceType, qos fixtures.QoSProfile) (fixtures.ResourceProfile, error)
	MapToPlacementProfile(constraints map[string]interface{}) (fixtures.PlacementProfile, error)
}

type ConfidenceCalculator interface {
	CalculateConfidence(nlpResult NLPResult, validation fixtures.ValidationResult) float64
}

// Supporting types
type NLPResult struct {
	Entities       []Entity               `json:"entities"`
	Classification IntentClassification   `json:"classification"`
	Confidence     float64                `json:"confidence"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type Entity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
}

type IntentClassification struct {
	Type       fixtures.IntentType `json:"type"`
	SliceType  fixtures.SliceType  `json:"sliceType"`
	Confidence float64             `json:"confidence"`
}

// ParserConfig for intent parser configuration
type ParserConfig struct {
	NLPModel        string  `json:"nlpModel"`
	ConfidenceThreshold float64 `json:"confidenceThreshold"`
	StrictValidation    bool    `json:"strictValidation"`
}

// NewIntentParser creates a new intent parser (not implemented yet)
func NewIntentParser(config ParserConfig) *IntentParser {
	// Intentionally not implemented to cause test failure (RED phase)
	return nil
}

// Interface methods that need to be implemented
func (p *IntentParser) ParseIntent(ctx context.Context, intent fixtures.Intent) (*fixtures.ParsedIntent, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *IntentParser) ValidateIntent(intent fixtures.Intent) fixtures.ValidationResult {
	// Not implemented yet - will cause tests to fail
	return fixtures.ValidationResult{Valid: false}
}

func (p *IntentParser) ExtractQoSRequirements(intent fixtures.Intent) (fixtures.QoSProfile, error) {
	// Not implemented yet - will cause tests to fail
	return fixtures.QoSProfile{}, nil
}

func (p *IntentParser) GenerateResourceProfile(sliceType fixtures.SliceType, qos fixtures.QoSProfile) (fixtures.ResourceProfile, error) {
	// Not implemented yet - will cause tests to fail
	return fixtures.ResourceProfile{}, nil
}

func (p *IntentParser) OptimizeQoSProfile(profile fixtures.QoSProfile, constraints map[string]interface{}) (fixtures.QoSProfile, error) {
	// Not implemented yet - will cause tests to fail
	return fixtures.QoSProfile{}, nil
}

// Table-driven tests for natural language intent parsing
func TestIntentParser_ParseIntent(t *testing.T) {
	tests := []struct {
		name            string
		intent          fixtures.Intent
		expectedResult  *fixtures.ParsedIntent
		expectedError   bool
		minConfidence   float64
		validateResult  func(t *testing.T, result *fixtures.ParsedIntent)
	}{
		{
			name:           "parse_embb_video_streaming_intent",
			intent:         fixtures.ValidEMBBIntent(),
			expectedResult: func() *fixtures.ParsedIntent { r := fixtures.ExpectedEMBBParsedIntent(); return &r }(),
			expectedError:  false,
			minConfidence:  0.9,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.Equal(t, fixtures.SliceTypeEMBB, result.SliceType)
				assert.Equal(t, "50", result.QoSProfile.Latency.Value)
				assert.Equal(t, "ms", result.QoSProfile.Latency.Unit)
				assert.Equal(t, "1Gbps", result.QoSProfile.Throughput.Downlink)
				assert.Contains(t, result.Placement.Zones, "edge-zone-a")
				assert.GreaterOrEqual(t, result.Confidence, 0.9)
			},
		},
		{
			name:           "parse_urllc_autonomous_driving_intent",
			intent:         fixtures.ValidURLLCIntent(),
			expectedResult: func() *fixtures.ParsedIntent { r := fixtures.ExpectedURLLCParsedIntent(); return &r }(),
			expectedError:  false,
			minConfidence:  0.95,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.Equal(t, fixtures.SliceTypeURLLC, result.SliceType)
				assert.Equal(t, "1", result.QoSProfile.Latency.Value)
				assert.Equal(t, "ms", result.QoSProfile.Latency.Unit)
				assert.Equal(t, "99.999", result.QoSProfile.Reliability.Value)
				assert.NotNil(t, result.QoSProfile.PacketLoss)
				assert.Equal(t, "0.001", result.QoSProfile.PacketLoss.Value)
				assert.GreaterOrEqual(t, result.Confidence, 0.95)
			},
		},
		{
			name:           "parse_mmtc_iot_sensors_intent",
			intent:         fixtures.ValidMmTCIntent(),
			expectedResult: func() *fixtures.ParsedIntent { r := fixtures.ExpectedMmTCParsedIntent(); return &r }(),
			expectedError:  false,
			minConfidence:  0.85,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.Equal(t, fixtures.SliceTypeMmTC, result.SliceType)
				assert.Contains(t, result.Intent.Constraints, "device-density")
				assert.Equal(t, "1000000", result.Intent.Constraints["device-density"])
				assert.GreaterOrEqual(t, result.Confidence, 0.85)
			},
		},
		{
			name:           "parse_complex_multi_slice_intent",
			intent:         fixtures.ComplexMultiSliceIntent(),
			expectedError:  false,
			minConfidence:  0.8,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				// Complex intent should be parsed with multiple slice types considered
				assert.NotEmpty(t, result.Intent.Constraints)
				assert.Contains(t, result.Intent.Constraints, "embb-throughput")
				assert.Contains(t, result.Intent.Constraints, "urllc-latency")
				assert.GreaterOrEqual(t, result.Confidence, 0.8)
			},
		},
		{
			name:           "parse_qos_optimization_intent",
			intent:         fixtures.QoSOptimizationIntent(),
			expectedError:  false,
			minConfidence:  0.9,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.Equal(t, fixtures.IntentTypeQoSOptimize, result.Intent.Type)
				assert.Contains(t, result.Intent.Constraints, "target-latency")
				assert.Contains(t, result.Intent.Constraints, "existing-slice-id")
			},
		},
		{
			name:           "parse_resource_scaling_intent",
			intent:         fixtures.ResourceScalingIntent(),
			expectedError:  false,
			minConfidence:  0.85,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.Equal(t, fixtures.IntentTypeResourceScale, result.Intent.Type)
				assert.Contains(t, result.Intent.Constraints, "scale-factor")
				assert.NotZero(t, result.Intent.Context.TimeWindow.Start)
			},
		},
		{
			name:          "parse_invalid_intent",
			intent:        fixtures.InvalidIntent(),
			expectedError: true,
			minConfidence: 0.0,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.Nil(t, result)
			},
		},
		{
			name:           "parse_ambiguous_intent",
			intent:         fixtures.AmbiguousIntent(),
			expectedError:  false,
			minConfidence:  0.3, // Low confidence expected for ambiguous intents
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.Less(t, result.Confidence, 0.5)
				assert.False(t, result.Validation.Valid)
				assert.NotEmpty(t, result.Validation.Warnings)
			},
		},
		{
			name:           "parse_conflicting_constraints_intent",
			intent:         fixtures.ConflictingConstraintsIntent(),
			expectedError:  false,
			minConfidence:  0.4,
			validateResult: func(t *testing.T, result *fixtures.ParsedIntent) {
				assert.False(t, result.Validation.Valid)
				assert.NotEmpty(t, result.Validation.Errors)
				assert.Contains(t, result.Validation.Errors[0].Message, "conflicting")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &IntentParser{
				// Mock dependencies would be set up here
			}

			result, err := parser.ParseIntent(context.Background(), tt.intent)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.GreaterOrEqual(t, result.Confidence, tt.minConfidence)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

// Test intent validation
func TestIntentParser_ValidateIntent(t *testing.T) {
	tests := []struct {
		name           string
		intent         fixtures.Intent
		expectedValid  bool
		expectedErrors int
		errorCodes     []string
	}{
		{
			name:           "validate_valid_embb_intent",
			intent:         fixtures.ValidEMBBIntent(),
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:           "validate_valid_urllc_intent",
			intent:         fixtures.ValidURLLCIntent(),
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:           "validate_valid_mmtc_intent",
			intent:         fixtures.ValidMmTCIntent(),
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name:           "validate_invalid_intent",
			intent:         fixtures.InvalidIntent(),
			expectedValid:  false,
			expectedErrors: 3,
			errorCodes:     []string{"EMPTY_ID", "EMPTY_TEXT", "INVALID_TYPE"},
		},
		{
			name:           "validate_conflicting_constraints",
			intent:         fixtures.ConflictingConstraintsIntent(),
			expectedValid:  false,
			expectedErrors: 1,
			errorCodes:     []string{"CONFLICTING_CONSTRAINTS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &IntentParser{}

			result := parser.ValidateIntent(tt.intent)

			assert.Equal(t, tt.expectedValid, result.Valid)
			assert.Len(t, result.Errors, tt.expectedErrors)

			if tt.errorCodes != nil {
				for i, expectedCode := range tt.errorCodes {
					if i < len(result.Errors) {
						assert.Equal(t, expectedCode, result.Errors[i].Code)
					}
				}
			}
		})
	}
}

// Test QoS profile generation
func TestIntentParser_ExtractQoSRequirements(t *testing.T) {
	tests := []struct {
		name            string
		intent          fixtures.Intent
		expectedLatency string
		expectedThroughput string
		expectedReliability string
		expectedError   bool
	}{
		{
			name:               "extract_embb_qos",
			intent:             fixtures.ValidEMBBIntent(),
			expectedLatency:    "50ms",
			expectedThroughput: "1Gbps",
			expectedReliability: "99.9%",
			expectedError:      false,
		},
		{
			name:               "extract_urllc_qos",
			intent:             fixtures.ValidURLLCIntent(),
			expectedLatency:    "1ms",
			expectedReliability: "99.999%",
			expectedError:      false,
		},
		{
			name:               "extract_mmtc_qos",
			intent:             fixtures.ValidMmTCIntent(),
			expectedError:      false,
		},
		{
			name:          "extract_invalid_qos",
			intent:        fixtures.InvalidIntent(),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &IntentParser{}

			result, err := parser.ExtractQoSRequirements(tt.intent)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedLatency != "" {
					assert.Contains(t, tt.expectedLatency, result.Latency.Value)
				}
				if tt.expectedThroughput != "" {
					assert.Contains(t, result.Throughput.Downlink, tt.expectedThroughput[:len(tt.expectedThroughput)-3])
				}
				if tt.expectedReliability != "" {
					assert.Contains(t, tt.expectedReliability, result.Reliability.Value)
				}
			}
		})
	}
}

// Test resource profile generation
func TestIntentParser_GenerateResourceProfile(t *testing.T) {
	tests := []struct {
		name          string
		sliceType     fixtures.SliceType
		qosProfile    fixtures.QoSProfile
		expectedCPU   string
		expectedMemory string
		expectedError bool
	}{
		{
			name:      "generate_embb_resources",
			sliceType: fixtures.SliceTypeEMBB,
			qosProfile: fixtures.QoSProfile{
				Latency: fixtures.LatencyRequirement{Value: "50", Unit: "ms"},
				Throughput: fixtures.ThroughputRequirement{Downlink: "1Gbps"},
			},
			expectedCPU:    "4000m",
			expectedMemory: "8Gi",
			expectedError:  false,
		},
		{
			name:      "generate_urllc_resources",
			sliceType: fixtures.SliceTypeURLLC,
			qosProfile: fixtures.QoSProfile{
				Latency: fixtures.LatencyRequirement{Value: "1", Unit: "ms"},
				Reliability: fixtures.ReliabilityRequirement{Value: "99.999"},
			},
			expectedCPU:    "8000m",
			expectedMemory: "16Gi",
			expectedError:  false,
		},
		{
			name:      "generate_mmtc_resources",
			sliceType: fixtures.SliceTypeMmTC,
			qosProfile: fixtures.QoSProfile{
				Latency: fixtures.LatencyRequirement{Value: "100", Unit: "ms"},
			},
			expectedCPU:    "2000m",
			expectedMemory: "4Gi",
			expectedError:  false,
		},
		{
			name:          "generate_invalid_slice_type",
			sliceType:     "",
			qosProfile:    fixtures.QoSProfile{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &IntentParser{}

			result, err := parser.GenerateResourceProfile(tt.sliceType, tt.qosProfile)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCPU, result.Compute.CPU)
				assert.Equal(t, tt.expectedMemory, result.Compute.Memory)
			}
		})
	}
}

// Test edge cases and error conditions
func TestIntentParser_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		testFunc      func(*IntentParser) error
		expectedError bool
		errorContains string
	}{
		{
			name: "nil_intent",
			testFunc: func(p *IntentParser) error {
				_, err := p.ParseIntent(context.Background(), fixtures.Intent{})
				return err
			},
			expectedError: true,
			errorContains: "empty intent",
		},
		{
			name: "context_timeout",
			testFunc: func(p *IntentParser) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				_, err := p.ParseIntent(ctx, fixtures.ValidEMBBIntent())
				return err
			},
			expectedError: true,
			errorContains: "context",
		},
		{
			name: "extremely_long_text",
			testFunc: func(p *IntentParser) error {
				intent := fixtures.ValidEMBBIntent()
				intent.Text = string(make([]byte, 100000)) // Very long text
				_, err := p.ParseIntent(context.Background(), intent)
				return err
			},
			expectedError: true,
			errorContains: "text too long",
		},
		{
			name: "special_characters_text",
			testFunc: func(p *IntentParser) error {
				intent := fixtures.ValidEMBBIntent()
				intent.Text = "Create slice with 特殊字符 and émojis 🚀 and symbols #@$%"
				_, err := p.ParseIntent(context.Background(), intent)
				return err
			},
			expectedError: false, // Should handle special characters gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &IntentParser{}

			err := tt.testFunc(parser)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test QoS profile optimization
func TestIntentParser_OptimizeQoSProfile(t *testing.T) {
	tests := []struct {
		name          string
		profile       fixtures.QoSProfile
		constraints   map[string]interface{}
		expectedError bool
		validate      func(t *testing.T, optimized fixtures.QoSProfile)
	}{
		{
			name: "optimize_for_cost",
			profile: fixtures.QoSProfile{
				Latency:    fixtures.LatencyRequirement{Value: "10", Unit: "ms"},
				Throughput: fixtures.ThroughputRequirement{Downlink: "1Gbps"},
			},
			constraints: map[string]interface{}{
				"optimization-target": "cost",
				"max-budget":         "1000",
			},
			expectedError: false,
			validate: func(t *testing.T, optimized fixtures.QoSProfile) {
				// Should optimize for cost while maintaining requirements
				assert.NotEmpty(t, optimized.Latency.Value)
				assert.NotEmpty(t, optimized.Throughput.Downlink)
			},
		},
		{
			name: "optimize_for_performance",
			profile: fixtures.QoSProfile{
				Latency:    fixtures.LatencyRequirement{Value: "50", Unit: "ms"},
				Reliability: fixtures.ReliabilityRequirement{Value: "99.9"},
			},
			constraints: map[string]interface{}{
				"optimization-target": "performance",
				"priority":           "latency",
			},
			expectedError: false,
			validate: func(t *testing.T, optimized fixtures.QoSProfile) {
				// Should improve latency while maintaining other requirements
				assert.NotEmpty(t, optimized.Latency.Value)
			},
		},
		{
			name: "invalid_optimization_constraints",
			profile: fixtures.QoSProfile{
				Latency: fixtures.LatencyRequirement{Value: "10", Unit: "ms"},
			},
			constraints: map[string]interface{}{
				"optimization-target": "invalid-target",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &IntentParser{}

			result, err := parser.OptimizeQoSProfile(tt.profile, tt.constraints)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}