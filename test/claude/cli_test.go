package claude_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/claude"
)

// TestClaudeClientInitialization verifies Claude client initialization
func TestClaudeClientInitialization(t *testing.T) {
	t.Run("Initialize Claude client with tmux session", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		config := &claude.ClientConfig{
			SessionName: "claude-test",
			Timeout:     30 * time.Second,
		}

		// Act
		client, err := claude.NewClient(ctx, config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, client.IsInitialized())
		defer client.Cleanup(ctx)
	})

	t.Run("Handle initialization without tmux", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		config := &claude.ClientConfig{
			SessionName: "claude-fallback",
			UseFallback: true,
		}

		// Act
		client, err := claude.NewClient(ctx, config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, client.IsFallbackMode())
	})
}

// TestNaturalLanguageProcessing verifies NL intent processing
func TestNaturalLanguageProcessing(t *testing.T) {
	t.Run("Process simple intent", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		intent := &claude.IntentRequest{
			Text: "Create a network slice for mobile broadband with 100 Mbps throughput",
			Context: claude.Context{
				Domain: "network-slicing",
				Type:   "creation",
			},
		}

		// Act
		response, err := client.ProcessIntent(ctx, intent)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ParsedIntent)
		assert.Contains(t, response.ParsedIntent.SliceType, "eMBB")
		assert.Equal(t, 100, response.ParsedIntent.Requirements.Throughput)
	})

	t.Run("Process complex multi-slice intent", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		intent := &claude.IntentRequest{
			Text: "Deploy three slices: one for video streaming with high bandwidth, " +
				"one for autonomous vehicles with ultra-low latency, and one for IoT sensors",
			Context: claude.Context{
				Domain: "network-slicing",
				Type:   "multi-deployment",
			},
		}

		// Act
		response, err := client.ProcessIntent(ctx, intent)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.ParsedSlices, 3)

		// Verify each slice type
		sliceTypes := []string{}
		for _, slice := range response.ParsedSlices {
			sliceTypes = append(sliceTypes, slice.Type)
		}
		assert.Contains(t, sliceTypes, "eMBB")  // video streaming
		assert.Contains(t, sliceTypes, "URLLC") // autonomous vehicles
		assert.Contains(t, sliceTypes, "mIoT")  // IoT sensors
	})

	t.Run("Process QoS modification intent", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		intent := &claude.IntentRequest{
			Text: "Increase the latency requirement to 1ms and add 99.999% reliability for the URLLC slice",
			Context: claude.Context{
				Domain:  "network-slicing",
				Type:    "modification",
				SliceID: "urllc-slice-001",
			},
		}

		// Act
		response, err := client.ProcessIntent(ctx, intent)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "modification", response.ActionType)
		assert.Equal(t, 1.0, response.QoSUpdate.Latency)
		assert.Equal(t, 99.999, response.QoSUpdate.Reliability)
	})
}

// TestBatchProcessing verifies batch intent processing
func TestBatchProcessing(t *testing.T) {
	t.Run("Process batch of intents", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		intents := []string{
			"Create an eMBB slice for video streaming",
			"Deploy URLLC slice for industrial automation",
			"Setup mIoT slice for smart city sensors",
		}

		// Act
		results, err := client.ProcessBatch(ctx, intents)

		// Assert
		require.NoError(t, err)
		assert.Len(t, results, 3)
		for _, result := range results {
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.Response)
		}
	})

	t.Run("Handle partial batch failures gracefully", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		intents := []string{
			"Create an eMBB slice",
			"Invalid intent that should fail parsing",
			"Deploy URLLC slice",
		}

		// Act
		results, err := client.ProcessBatch(ctx, intents)

		// Assert
		require.NoError(t, err) // Should not fail entire batch
		assert.Len(t, results, 3)
		assert.True(t, results[0].Success)
		assert.False(t, results[1].Success)
		assert.True(t, results[2].Success)
	})
}

// TestPromptEngineering verifies prompt generation
func TestPromptEngineering(t *testing.T) {
	t.Run("Generate slice creation prompt", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)

		template := &claude.PromptTemplate{
			Type: "slice-creation",
			Parameters: map[string]interface{}{
				"sliceType":  "eMBB",
				"throughput": 1000,
				"latency":    20,
			},
		}

		// Act
		prompt, err := client.GeneratePrompt(ctx, template)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "eMBB")
		assert.Contains(t, prompt, "1000")
	})

	t.Run("Generate optimization prompt", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)

		template := &claude.PromptTemplate{
			Type: "optimization",
			Parameters: map[string]interface{}{
				"metric":    "throughput",
				"target":    "maximize",
				"sliceType": "eMBB",
			},
		}

		// Act
		prompt, err := client.GeneratePrompt(ctx, template)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, prompt, "optimize")
		assert.Contains(t, prompt, "throughput")
	})
}

// TestContextManagement verifies context preservation
func TestContextManagement(t *testing.T) {
	t.Run("Maintain conversation context", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		// First intent
		firstIntent := &claude.IntentRequest{
			Text: "Create an eMBB slice called video-slice",
		}

		// Act - First request
		_, err := client.ProcessIntent(ctx, firstIntent)
		require.NoError(t, err)

		// Second intent referencing first
		secondIntent := &claude.IntentRequest{
			Text: "Increase the throughput of the video slice to 500 Mbps",
		}

		// Act - Second request
		secondResponse, err := client.ProcessIntent(ctx, secondIntent)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, secondResponse)
		assert.Equal(t, "video-slice", secondResponse.TargetSlice)
		assert.Equal(t, 500, secondResponse.QoSUpdate.Throughput)
	})

	t.Run("Clear context on demand", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		// Add context
		client.SetContext(ctx, "slice-001", "eMBB")

		// Act
		err := client.ClearContext(ctx)

		// Assert
		require.NoError(t, err)
		assert.False(t, client.HasContext())
	})
}

// TestErrorHandling verifies error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("Handle malformed intent gracefully", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)
		defer client.Cleanup(ctx)

		intent := &claude.IntentRequest{
			Text: "", // Empty intent
		}

		// Act
		response, err := client.ProcessIntent(ctx, intent)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "empty intent")
	})

	t.Run("Handle timeout", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		config := &claude.ClientConfig{
			SessionName: "timeout-test",
			Timeout:     1 * time.Millisecond,
		}
		client, _ := claude.NewClient(ctx, config)

		intent := &claude.IntentRequest{
			Text: "Complex intent requiring long processing",
		}

		// Act
		response, err := client.ProcessIntent(ctx, intent)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.True(t, strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline"))
	})
}

// TestValidation verifies input validation
func TestValidation(t *testing.T) {
	t.Run("Validate slice type extraction", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)

		testCases := []struct {
			input    string
			expected string
		}{
			{"video streaming service", "eMBB"},
			{"autonomous vehicle control", "URLLC"},
			{"smart meter monitoring", "mIoT"},
			{"mobile broadband", "eMBB"},
			{"industrial automation", "URLLC"},
			{"sensor network", "mIoT"},
		}

		// Act & Assert
		for _, tc := range testCases {
			sliceType, err := client.ExtractSliceType(ctx, tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, sliceType)
		}
	})

	t.Run("Validate QoS parameters", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)

		qos := &claude.QoSParameters{
			Latency:     -1,  // Invalid
			Throughput:  0,   // Invalid
			Reliability: 101, // Invalid (>100%)
		}

		// Act
		err := client.ValidateQoS(ctx, qos)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}

// TestExportCapabilities verifies export functionality
func TestExportCapabilities(t *testing.T) {
	t.Run("Export to YAML", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)

		response := &claude.IntentResponse{
			ParsedIntent: &claude.ParsedIntent{
				SliceType: "eMBB",
				Requirements: &claude.Requirements{
					Throughput: 100,
					Latency:    20,
				},
			},
		}

		// Act
		yaml, err := client.ExportToYAML(ctx, response)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, yaml)
		assert.Contains(t, yaml, "slicetype: eMBB")
		assert.Contains(t, yaml, "throughput: 100")
	})

	t.Run("Export to JSON", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := createTestClient(t)

		response := &claude.IntentResponse{
			ActionType: "create",
			ParsedIntent: &claude.ParsedIntent{
				SliceType: "URLLC",
			},
		}

		// Act
		json, err := client.ExportToJSON(ctx, response)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, json)
		assert.Contains(t, json, `"SliceType": "URLLC"`)
	})
}

// Helper functions

func createTestClient(t *testing.T) *claude.Client {
	ctx := context.Background()
	config := &claude.ClientConfig{
		SessionName: "test-session",
		Timeout:     5 * time.Second,
		UseFallback: true, // Use fallback for tests
	}

	client, err := claude.NewClient(ctx, config)
	require.NoError(t, err)
	return client
}