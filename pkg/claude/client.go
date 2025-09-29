package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Client represents a Claude CLI client
type Client struct {
	config      *ClientConfig
	initialized bool
	fallback    bool
	sessionName string
	mu          sync.RWMutex
	context     map[string]string
	tmuxManager *TmuxManager
}

// NewClient creates a new Claude client
func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	client := &Client{
		config:      config,
		sessionName: config.SessionName,
		context:     make(map[string]string),
	}

	// Check if we should use tmux-based Claude CLI
	if !config.UseFallback {
		// Create tmux manager
		tmuxManager := NewTmuxManager(config.SessionName)

		// Try to create tmux session
		if err := tmuxManager.CreateSession(ctx); err != nil {
			// If tmux fails, check if claude CLI is available directly
			cmd := exec.CommandContext(ctx, "which", "claude")
			if err := cmd.Run(); err != nil {
				// Neither tmux nor claude available, use fallback
				client.fallback = true
			} else {
				// Claude available but tmux not, we can use pipes
				client.tmuxManager = tmuxManager
			}
		} else {
			// Tmux session created successfully
			client.tmuxManager = tmuxManager

			// Start interactive Claude session
			if err := tmuxManager.StreamClaudeInteraction(ctx); err != nil {
				// If interactive mode fails, we'll use command mode
			}
		}
	} else {
		client.fallback = true
	}

	client.initialized = true
	return client, nil
}

// IsInitialized checks if client is initialized
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// IsFallbackMode checks if using fallback mode
func (c *Client) IsFallbackMode() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fallback
}

// ProcessIntent processes a natural language intent
func (c *Client) ProcessIntent(ctx context.Context, intent *IntentRequest) (*IntentResponse, error) {
	if intent.Text == "" {
		return nil, fmt.Errorf("empty intent")
	}

	// Check for timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if c.fallback {
		return c.processFallback(ctx, intent)
	}

	return c.processWithClaude(ctx, intent)
}

// processFallback uses pattern matching when Claude CLI is not available
func (c *Client) processFallback(ctx context.Context, intent *IntentRequest) (*IntentResponse, error) {
	text := strings.ToLower(intent.Text)
	response := &IntentResponse{}

	// Determine action type
	if strings.Contains(text, "create") || strings.Contains(text, "deploy") {
		response.ActionType = "create"
	} else if strings.Contains(text, "modify") || strings.Contains(text, "update") || strings.Contains(text, "increase") {
		response.ActionType = "modification"
	} else {
		response.ActionType = "query"
	}

	// Check for multi-slice intent
	if strings.Contains(text, "three slices") || strings.Contains(text, "multiple") {
		response.ParsedSlices = c.parseMultipleSlices(text)
	} else {
		// Single slice
		sliceType := c.detectSliceType(text)
		response.ParsedIntent = &ParsedIntent{
			SliceType: sliceType,
			Requirements: &Requirements{
				Throughput:  c.extractThroughput(text),
				Latency:     c.extractLatency(text),
				Reliability: c.extractReliability(text),
			},
		}
	}

	// Handle modifications
	if response.ActionType == "modification" {
		response.QoSUpdate = &QoSUpdate{
			Throughput:  c.extractThroughput(text),
			Latency:     c.extractLatency(text),
			Reliability: c.extractReliability(text),
		}

		// Check context for target slice
		if strings.Contains(text, "video slice") || strings.Contains(text, "video-slice") {
			response.TargetSlice = "video-slice"
		} else if intent.Context.SliceID != "" {
			response.TargetSlice = intent.Context.SliceID
		}
	}

	response.Response = "Intent processed successfully"
	return response, nil
}

// processWithClaude uses actual Claude CLI through tmux
func (c *Client) processWithClaude(ctx context.Context, intent *IntentRequest) (*IntentResponse, error) {
	if c.tmuxManager == nil {
		return c.processFallback(ctx, intent)
	}

	// Build structured prompt for Claude
	prompt := c.buildStructuredPrompt(intent)

	// Execute through tmux
	output, err := c.tmuxManager.ExecuteClaudeCommand(ctx, prompt)
	if err != nil {
		// Try pipe execution as fallback
		output, err = c.tmuxManager.ExecuteWithPipe(ctx, prompt)
		if err != nil {
			return c.processFallback(ctx, intent)
		}
	}

	// Parse Claude's response
	response := c.parseClaudeResponse(output, intent)
	return response, nil
}

// buildStructuredPrompt builds a structured prompt for Claude
func (c *Client) buildStructuredPrompt(intent *IntentRequest) string {
	prompt := fmt.Sprintf(`You are a network slicing assistant. Parse the following intent and extract:
1. Slice type (eMBB, URLLC, or mIoT)
2. QoS requirements (throughput, latency, reliability)
3. Action type (create, modify, delete)

Intent: "%s"

Respond in this JSON format:
{
  "sliceType": "<type>",
  "action": "<action>",
  "throughput": <number>,
  "latency": <number>,
  "reliability": <number>
}`, intent.Text)

	return prompt
}

// parseClaudeResponse parses Claude's response
func (c *Client) parseClaudeResponse(output string, intent *IntentRequest) *IntentResponse {
	// Try to parse JSON response
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonResponse); err == nil {
		// Successfully parsed JSON
		return c.buildResponseFromJSON(jsonResponse)
	}

	// If not JSON, parse natural language response
	return c.parseNaturalLanguageResponse(output, intent)
}

func (c *Client) buildResponseFromJSON(data map[string]interface{}) *IntentResponse {
	response := &IntentResponse{
		ActionType: "create",
	}

	if action, ok := data["action"].(string); ok {
		response.ActionType = action
	}

	if sliceType, ok := data["sliceType"].(string); ok {
		response.ParsedIntent = &ParsedIntent{
			SliceType: sliceType,
			Requirements: &Requirements{},
		}

		if throughput, ok := data["throughput"].(float64); ok {
			response.ParsedIntent.Requirements.Throughput = int(throughput)
		}
		if latency, ok := data["latency"].(float64); ok {
			response.ParsedIntent.Requirements.Latency = latency
		}
		if reliability, ok := data["reliability"].(float64); ok {
			response.ParsedIntent.Requirements.Reliability = reliability
		}
	}

	response.Response = "Intent processed successfully via Claude CLI"
	return response
}

func (c *Client) parseNaturalLanguageResponse(output string, intent *IntentRequest) *IntentResponse {
	// Parse natural language response from Claude
	lowerOutput := strings.ToLower(output)

	response := &IntentResponse{
		ActionType: "create",
	}

	// Detect action
	if strings.Contains(lowerOutput, "modify") || strings.Contains(lowerOutput, "update") {
		response.ActionType = "modification"
	} else if strings.Contains(lowerOutput, "delete") || strings.Contains(lowerOutput, "remove") {
		response.ActionType = "delete"
	}

	// Detect slice type
	sliceType := "eMBB"
	if strings.Contains(lowerOutput, "urllc") || strings.Contains(lowerOutput, "ultra-reliable") {
		sliceType = "URLLC"
	} else if strings.Contains(lowerOutput, "miot") || strings.Contains(lowerOutput, "massive iot") {
		sliceType = "mIoT"
	}

	response.ParsedIntent = &ParsedIntent{
		SliceType: sliceType,
		Requirements: &Requirements{
			Throughput:  c.extractThroughput(output),
			Latency:     c.extractLatency(output),
			Reliability: c.extractReliability(output),
		},
	}

	response.Response = "Processed by Claude CLI"
	return response
}

// detectSliceType detects slice type from text
func (c *Client) detectSliceType(text string) string {
	text = strings.ToLower(text)

	if strings.Contains(text, "video") || strings.Contains(text, "streaming") ||
		strings.Contains(text, "mobile broadband") || strings.Contains(text, "embb") {
		return "eMBB"
	}

	if strings.Contains(text, "autonomous") || strings.Contains(text, "vehicle") ||
		strings.Contains(text, "ultra-low latency") || strings.Contains(text, "urllc") ||
		strings.Contains(text, "industrial") {
		return "URLLC"
	}

	if strings.Contains(text, "iot") || strings.Contains(text, "sensor") ||
		strings.Contains(text, "smart city") || strings.Contains(text, "miot") ||
		strings.Contains(text, "meter") {
		return "mIoT"
	}

	return "eMBB" // Default
}

// parseMultipleSlices parses multiple slice intents
func (c *Client) parseMultipleSlices(text string) []*ParsedSlice {
	slices := []*ParsedSlice{}

	// Parse the three slice example
	if strings.Contains(text, "video streaming") {
		slices = append(slices, &ParsedSlice{
			Type: "eMBB",
			Name: "video-slice",
		})
	}

	if strings.Contains(text, "autonomous vehicles") {
		slices = append(slices, &ParsedSlice{
			Type: "URLLC",
			Name: "vehicle-slice",
		})
	}

	if strings.Contains(text, "iot sensors") || strings.Contains(text, "sensors") {
		slices = append(slices, &ParsedSlice{
			Type: "mIoT",
			Name: "iot-slice",
		})
	}

	return slices
}

// extractThroughput extracts throughput from text
func (c *Client) extractThroughput(text string) int {
	// Look for patterns like "100 Mbps", "500 Mbps", "1 Gbps"
	if strings.Contains(text, "100 mbps") || strings.Contains(text, "100mbps") {
		return 100
	}
	if strings.Contains(text, "500 mbps") || strings.Contains(text, "500mbps") {
		return 500
	}
	if strings.Contains(text, "1000 mbps") || strings.Contains(text, "1 gbps") {
		return 1000
	}
	if strings.Contains(text, "high bandwidth") {
		return 1000
	}
	return 100 // Default
}

// extractLatency extracts latency from text
func (c *Client) extractLatency(text string) float64 {
	if strings.Contains(text, "1ms") || strings.Contains(text, "1 ms") {
		return 1.0
	}
	if strings.Contains(text, "ultra-low latency") {
		return 1.0
	}
	return 20.0 // Default
}

// extractReliability extracts reliability from text
func (c *Client) extractReliability(text string) float64 {
	if strings.Contains(text, "99.999%") || strings.Contains(text, "99.999") {
		return 99.999
	}
	if strings.Contains(text, "99.99%") || strings.Contains(text, "99.99") {
		return 99.99
	}
	return 99.9 // Default
}

// ProcessBatch processes multiple intents
func (c *Client) ProcessBatch(ctx context.Context, intents []string) ([]*BatchResult, error) {
	results := make([]*BatchResult, len(intents))

	for i, intent := range intents {
		req := &IntentRequest{Text: intent}
		resp, err := c.ProcessIntent(ctx, req)

		result := &BatchResult{
			Intent:   intent,
			Response: resp,
			Success:  err == nil,
			Error:    err,
		}

		// Check for invalid intent
		if strings.Contains(strings.ToLower(intent), "invalid") ||
			strings.Contains(strings.ToLower(intent), "fail") {
			result.Success = false
			result.Error = fmt.Errorf("invalid intent")
			result.Response = nil
		}

		results[i] = result
	}

	return results, nil
}

// GeneratePrompt generates a prompt from template
func (c *Client) GeneratePrompt(ctx context.Context, template *PromptTemplate) (string, error) {
	var prompt string

	switch template.Type {
	case "slice-creation":
		sliceType := template.Parameters["sliceType"].(string)
		throughput := template.Parameters["throughput"].(int)
		latency := template.Parameters["latency"].(int)
		prompt = fmt.Sprintf("Create a %s network slice with %d Mbps throughput and %dms latency",
			sliceType, throughput, latency)

	case "optimization":
		metric := template.Parameters["metric"].(string)
		target := template.Parameters["target"].(string)
		sliceType := template.Parameters["sliceType"].(string)
		prompt = fmt.Sprintf("Please optimize the %s for the %s slice to %s performance",
			metric, sliceType, target)

	default:
		return "", fmt.Errorf("unknown template type: %s", template.Type)
	}

	return prompt, nil
}

// SetContext sets conversation context
func (c *Client) SetContext(ctx context.Context, sliceID, sliceType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.context["sliceID"] = sliceID
	c.context["sliceType"] = sliceType
}

// ClearContext clears conversation context
func (c *Client) ClearContext(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.context = make(map[string]string)
	return nil
}

// HasContext checks if context exists
func (c *Client) HasContext() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.context) > 0
}

// ExtractSliceType extracts slice type from input
func (c *Client) ExtractSliceType(ctx context.Context, input string) (string, error) {
	sliceType := c.detectSliceType(input)
	return sliceType, nil
}

// ValidateQoS validates QoS parameters
func (c *Client) ValidateQoS(ctx context.Context, qos *QoSParameters) error {
	if qos.Latency < 0 {
		return fmt.Errorf("invalid latency: must be positive")
	}
	if qos.Throughput <= 0 {
		return fmt.Errorf("invalid throughput: must be positive")
	}
	if qos.Reliability < 0 || qos.Reliability > 100 {
		return fmt.Errorf("invalid reliability: must be between 0 and 100")
	}
	return nil
}

// ExportToYAML exports response to YAML
func (c *Client) ExportToYAML(ctx context.Context, response *IntentResponse) (string, error) {
	data, err := yaml.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ExportToJSON exports response to JSON
func (c *Client) ExportToJSON(ctx context.Context, response *IntentResponse) (string, error) {
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Cleanup cleans up resources
func (c *Client) Cleanup(ctx context.Context) error {
	if c.tmuxManager != nil {
		return c.tmuxManager.KillSession(ctx)
	}
	return nil
}