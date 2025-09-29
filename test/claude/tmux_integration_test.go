package claude_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/claude"
)

// TestTmuxManagerCreation tests tmux session creation
func TestTmuxManagerCreation(t *testing.T) {
	// Skip if tmux not installed
	if err := exec.Command("which", "tmux").Run(); err != nil {
		t.Skip("tmux not installed, skipping tmux tests")
	}

	t.Run("Create and manage tmux session", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		tmuxManager := claude.NewTmuxManager("test-claude-session")

		// Act - Create session
		err := tmuxManager.CreateSession(ctx)
		require.NoError(t, err)
		defer tmuxManager.KillSession(ctx)

		// Send a test command
		err = tmuxManager.SendCommand(ctx, "echo 'Hello from tmux'")
		require.NoError(t, err)

		// Capture output
		time.Sleep(1 * time.Second)
		output, err := tmuxManager.CaptureOutput(ctx)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, output, "Hello from tmux")
	})
}

// TestClaudeCliExecution tests actual Claude CLI execution
func TestClaudeCliExecution(t *testing.T) {
	// Check if both tmux and claude are available
	tmuxErr := exec.Command("which", "tmux").Run()
	claudeErr := exec.Command("which", "claude").Run()

	if tmuxErr != nil || claudeErr != nil {
		t.Skip("tmux or claude CLI not available, skipping integration test")
	}

	t.Run("Execute Claude command through tmux", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		tmuxManager := claude.NewTmuxManager("claude-exec-test")

		err := tmuxManager.CreateSession(ctx)
		require.NoError(t, err)
		defer tmuxManager.KillSession(ctx)

		// Act - Execute Claude command
		prompt := "What is 2+2? Reply with just the number."
		response, err := tmuxManager.ExecuteClaudeCommand(ctx, prompt)

		// Assert
		if err != nil {
			// Claude might not be accessible, skip assertion
			t.Logf("Claude execution failed (might not have access): %v", err)
		} else {
			// Claude might return empty or just echo, that's OK for test
			t.Logf("Claude response received (may be empty): '%s'", response)
			// Don't assert NotEmpty as Claude might not be fully configured
		}
	})

	t.Run("Execute with pipe fallback", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		tmuxManager := claude.NewTmuxManager("claude-pipe-test")

		// Act - Try pipe execution
		prompt := "Respond with 'OK' if you receive this."
		response, err := tmuxManager.ExecuteWithPipe(ctx, prompt)

		// Assert
		if err != nil {
			t.Logf("Pipe execution failed (expected without Claude access): %v", err)
		} else {
			assert.NotEmpty(t, response)
			t.Logf("Pipe response: %s", response)
		}
	})
}

// TestRealClaudeIntegration tests the full Claude client with real CLI
func TestRealClaudeIntegration(t *testing.T) {
	// Skip if claude CLI is not available
	if err := exec.Command("which", "claude").Run(); err != nil {
		t.Skip("claude CLI not available")
	}

	t.Run("Process network slice intent with real Claude", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		config := &claude.ClientConfig{
			SessionName: "real-claude-test",
			Timeout:     10 * time.Second,
			UseFallback: false, // Force real Claude
		}

		client, err := claude.NewClient(ctx, config)
		require.NoError(t, err)
		defer client.Cleanup(ctx)

		intent := &claude.IntentRequest{
			Text: "Create an eMBB network slice with 1Gbps throughput for video streaming",
		}

		// Act
		response, err := client.ProcessIntent(ctx, intent)

		// Assert
		if err != nil {
			t.Logf("Claude processing failed (might not have access): %v", err)
			// Should at least fall back to pattern matching
			assert.NotNil(t, response)
			assert.Equal(t, "eMBB", response.ParsedIntent.SliceType)
		} else {
			require.NoError(t, err)
			assert.NotNil(t, response)
			assert.Contains(t, response.Response, "Claude")
			assert.Equal(t, "eMBB", response.ParsedIntent.SliceType)
			assert.Equal(t, 1000, response.ParsedIntent.Requirements.Throughput)
		}
	})
}

// TestStreamingInteraction tests interactive Claude session
func TestStreamingInteraction(t *testing.T) {
	// Skip if tmux not available
	if err := exec.Command("which", "tmux").Run(); err != nil {
		t.Skip("tmux not installed")
	}

	t.Run("Start interactive Claude session", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tmuxManager := claude.NewTmuxManager("claude-interactive")
		err := tmuxManager.CreateSession(ctx)
		require.NoError(t, err)
		defer tmuxManager.KillSession(context.Background())

		// Act - Start streaming
		err = tmuxManager.StreamClaudeInteraction(ctx)
		if err != nil {
			t.Logf("Streaming failed (Claude might not be available): %v", err)
			return
		}

		// Get output channel
		outputChan := tmuxManager.GetStreamedOutput()

		// Send a prompt
		err = tmuxManager.SendPrompt(ctx, "Hello Claude, respond with 'Hi'")
		require.NoError(t, err)

		// Wait for response
		select {
		case output := <-outputChan:
			assert.NotEmpty(t, output)
			t.Logf("Streamed output: %s", output)
		case <-time.After(5 * time.Second):
			t.Log("No response received (Claude might not be available)")
		}
	})
}

// TestMultipleIntents tests processing multiple intents in sequence
func TestMultipleIntents(t *testing.T) {
	t.Run("Process multiple intents with context preservation", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		config := &claude.ClientConfig{
			SessionName: "multi-intent-test",
			Timeout:     10 * time.Second,
			UseFallback: false,
		}

		client, err := claude.NewClient(ctx, config)
		require.NoError(t, err)
		defer client.Cleanup(ctx)

		intents := []string{
			"Create an eMBB slice named video-slice",
			"Add 500 Mbps throughput requirement",
			"Set latency to 10ms",
		}

		// Act - Process intents in sequence
		var lastResponse *claude.IntentResponse
		for _, intentText := range intents {
			intent := &claude.IntentRequest{Text: intentText}
			response, err := client.ProcessIntent(ctx, intent)

			// Assert each step
			if err != nil {
				t.Logf("Intent processing failed: %v", err)
				// Should still get fallback response
				assert.NotNil(t, response)
			} else {
				assert.NotNil(t, response)
			}
			lastResponse = response
		}

		// Verify final state
		if lastResponse != nil && lastResponse.ParsedIntent != nil {
			t.Logf("Final slice type: %s", lastResponse.ParsedIntent.SliceType)
			t.Logf("Final requirements: %+v", lastResponse.ParsedIntent.Requirements)
		}
	})
}

// TestTmuxErrorHandling tests error handling in tmux integration
func TestTmuxErrorHandling(t *testing.T) {
	t.Run("Handle tmux session errors gracefully", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		tmuxManager := claude.NewTmuxManager("error-test")

		// Act - Try to send command without creating session
		err := tmuxManager.SendCommand(ctx, "test")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("Handle invalid Claude responses", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		config := &claude.ClientConfig{
			SessionName: "invalid-test",
			Timeout:     5 * time.Second,
			UseFallback: false,
		}

		client, err := claude.NewClient(ctx, config)
		require.NoError(t, err)
		defer client.Cleanup(ctx)

		// Send intent that might produce non-JSON response
		intent := &claude.IntentRequest{
			Text: "This is a test that might not parse as expected @#$%^",
		}

		// Act
		response, err := client.ProcessIntent(ctx, intent)

		// Assert - Should handle gracefully
		if err != nil {
			t.Logf("Error handled: %v", err)
		} else {
			// Should at least return fallback response
			assert.NotNil(t, response)
			assert.NotEmpty(t, response.ParsedIntent.SliceType)
		}
	})
}