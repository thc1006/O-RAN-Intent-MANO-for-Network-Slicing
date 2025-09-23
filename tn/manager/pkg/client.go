package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// TNAgentClient represents a client for communicating with TN agents
type TNAgentClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *log.Logger
	connected  bool
}

// NewTNAgentClient creates a new TN agent client
func NewTNAgentClient(endpoint string, logger *log.Logger) *TNAgentClient {
	return &TNAgentClient{
		baseURL: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:    logger,
		connected: false,
	}
}

// Connect establishes a connection to the TN agent
func (client *TNAgentClient) Connect() error {
	client.logger.Printf("Connecting to TN agent at %s", client.baseURL)

	// Test connectivity with health check
	resp, err := client.httpClient.Get(client.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent health check failed: status %d", resp.StatusCode)
	}

	client.connected = true
	client.logger.Printf("Successfully connected to TN agent at %s", client.baseURL)
	return nil
}

// Stop closes the connection to the TN agent
func (client *TNAgentClient) Stop() error {
	client.connected = false
	client.logger.Printf("Disconnected from TN agent at %s", client.baseURL)
	return nil
}

// ConfigureSlice configures a network slice on the agent
func (client *TNAgentClient) ConfigureSlice(sliceID string, config *TNConfig) error {
	if !client.connected {
		return fmt.Errorf("client not connected")
	}

	client.logger.Printf("Configuring slice %s on agent %s", sliceID, client.baseURL)

	payload := map[string]interface{}{
		"sliceId": sliceID,
		"config":  config,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	resp, err := client.httpClient.Post(
		client.baseURL+"/api/v1/slices/configure",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return fmt.Errorf("failed to send configuration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("configuration failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	client.logger.Printf("Successfully configured slice %s on agent", sliceID)
	return nil
}

// RunPerformanceTest executes a performance test on the agent
func (client *TNAgentClient) RunPerformanceTest(config *PerformanceTestConfig) (*PerformanceMetrics, error) {
	if !client.connected {
		return nil, fmt.Errorf("client not connected")
	}

	client.logger.Printf("Running performance test %s on agent %s", config.TestID, client.baseURL)

	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal test configuration: %w", err)
	}

	resp, err := client.httpClient.Post(
		client.baseURL+"/api/v1/test/performance",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("performance test failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var metrics PerformanceMetrics
	if err := json.Unmarshal(body, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	client.logger.Printf("Performance test completed: %.2f Mbps throughput, %.2f ms latency",
		metrics.Throughput.AvgMbps, metrics.Latency.AvgRTTMs)

	return &metrics, nil
}

// GetStatus retrieves the current status from the agent
func (client *TNAgentClient) GetStatus() (*TNStatus, error) {
	if !client.connected {
		return nil, fmt.Errorf("client not connected")
	}

	resp, err := client.httpClient.Get(client.baseURL + "/api/v1/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status request failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var status TNStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

// GetMetrics retrieves metrics from the agent
func (client *TNAgentClient) GetMetrics() (map[string]interface{}, error) {
	if !client.connected {
		return nil, fmt.Errorf("client not connected")
	}

	resp, err := client.httpClient.Get(client.baseURL + "/api/v1/metrics")
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics request failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var metrics map[string]interface{}
	if err := json.Unmarshal(body, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return metrics, nil
}

// SendCommand sends a generic command to the agent
func (client *TNAgentClient) SendCommand(command string, payload interface{}) (interface{}, error) {
	if !client.connected {
		return nil, fmt.Errorf("client not connected")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := client.httpClient.Post(
		client.baseURL+"/api/v1/command/"+command,
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("command failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// IsConnected returns true if the client is connected to the agent
func (client *TNAgentClient) IsConnected() bool {
	return client.connected
}

// Ping sends a ping to test connectivity
func (client *TNAgentClient) Ping() error {
	if !client.connected {
		return fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", client.baseURL+"/ping", nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed: status %d", resp.StatusCode)
	}

	return nil
}