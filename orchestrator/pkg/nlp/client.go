package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a client for the NLP service
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new NLP client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IntentRequest represents a request to parse an intent
type IntentRequest struct {
	Intent    string `json:"intent"`
	SessionID string `json:"session_id,omitempty"`
}

// QoSProfile represents QoS parameters from NLP parsing
type QoSProfile struct {
	SliceType           string   `json:"slice_type"`
	ThroughputMbps      float64  `json:"throughput_mbps"`
	LatencyMs           float64  `json:"latency_ms"`
	PacketLossRate      float64  `json:"packet_loss_rate"`
	Priority            int      `json:"priority"`
	JitterMs            *float64 `json:"jitter_ms,omitempty"`
	BandwidthGuarantee  *float64 `json:"bandwidth_guarantee,omitempty"`
	Reliability         *float64 `json:"reliability,omitempty"`
}

// IntentResponse represents the response from NLP service
type IntentResponse struct {
	Success          bool       `json:"success"`
	SliceType        string     `json:"slice_type"`
	QoSProfile       QoSProfile `json:"qos_profile"`
	RawIntent        string     `json:"raw_intent"`
	SessionID        string     `json:"session_id,omitempty"`
	Timestamp        string     `json:"timestamp"`
	ProcessingTimeMs float64    `json:"processing_time_ms"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status                 string  `json:"status"`
	Version                string  `json:"version"`
	UptimeSeconds          float64 `json:"uptime_seconds"`
	TotalIntentsProcessed  int     `json:"total_intents_processed"`
}

// ParseIntent sends a natural language intent to the NLP service for parsing
func (c *Client) ParseIntent(ctx context.Context, intent string, sessionID string) (*IntentResponse, error) {
	req := IntentRequest{
		Intent:    intent,
		SessionID: sessionID,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/v1/parse", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NLP service returned error: %s (status %d)", string(body), resp.StatusCode)
	}

	var intentResp IntentResponse
	if err := json.Unmarshal(body, &intentResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &intentResp, nil
}

// HealthCheck checks if the NLP service is healthy
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.Unmarshal(body, &healthResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &healthResp, nil
}
