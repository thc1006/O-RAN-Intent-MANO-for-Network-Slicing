package dms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
)

// Client interface for O2 DMS operations
type Client interface {
	CreateDeployment(ctx context.Context, vnf *manov1alpha1.VNF) (string, error)
	GetDeploymentStatus(ctx context.Context, deploymentID string) (string, error)
	UpdateDeployment(ctx context.Context, deploymentID string, vnf *manov1alpha1.VNF) error
	DeleteDeployment(ctx context.Context, deploymentID string) error
}

// MockDMSClient provides a mock implementation for testing
type MockDMSClient struct {
	Deployments map[string]*DeploymentInfo
}

// DeploymentInfo stores deployment information
type DeploymentInfo struct {
	ID     string
	VNF    *manov1alpha1.VNF
	Status string
}

// NewMockDMSClient creates a new mock DMS client
func NewMockDMSClient() *MockDMSClient {
	return &MockDMSClient{
		Deployments: make(map[string]*DeploymentInfo),
	}
}

// CreateDeployment creates a new DMS deployment
func (c *MockDMSClient) CreateDeployment(_ context.Context, vnf *manov1alpha1.VNF) (string, error) {
	// Simulate deployment creation
	deploymentID := fmt.Sprintf("dms-%s-%s", vnf.Name, vnf.Spec.Type)

	c.Deployments[deploymentID] = &DeploymentInfo{
		ID:     deploymentID,
		VNF:    vnf,
		Status: "Creating",
	}

	return deploymentID, nil
}

// GetDeploymentStatus gets the status of a deployment
func (c *MockDMSClient) GetDeploymentStatus(_ context.Context, deploymentID string) (string, error) {
	deployment, exists := c.Deployments[deploymentID]
	if !exists {
		return "", fmt.Errorf("deployment %s not found", deploymentID)
	}

	// Simulate status progression
	if deployment.Status == "Creating" {
		deployment.Status = "Running"
	}

	return deployment.Status, nil
}

// UpdateDeployment updates an existing deployment
func (c *MockDMSClient) UpdateDeployment(_ context.Context, deploymentID string, vnf *manov1alpha1.VNF) error {
	deployment, exists := c.Deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	deployment.VNF = vnf
	deployment.Status = "Updating"

	return nil
}

// DeleteDeployment deletes a deployment
func (c *MockDMSClient) DeleteDeployment(_ context.Context, deploymentID string) error {
	_, exists := c.Deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	delete(c.Deployments, deploymentID)
	return nil
}

// O2DeploymentRequest represents a deployment request to O2 DMS
type O2DeploymentRequest struct {
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Version        string                 `json:"version"`
	TargetClusters []string               `json:"target_clusters"`
	Resources      map[string]interface{} `json:"resources"`
	QoSProfile     map[string]interface{} `json:"qos_profile"`
	ConfigData     map[string]string      `json:"config_data,omitempty"`
}

// O2DeploymentResponse represents a deployment response from O2 DMS
type O2DeploymentResponse struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

// O2DeploymentStatus represents deployment status response
type O2DeploymentStatus struct {
	DeploymentID string            `json:"deployment_id"`
	Status       string            `json:"status"`
	Phase        string            `json:"phase"`
	Message      string            `json:"message,omitempty"`
	Clusters     map[string]string `json:"clusters,omitempty"`
	LastUpdate   time.Time         `json:"last_update"`
}

// HTTPError represents HTTP error responses
type HTTPError struct {
	StatusCode int
	Message    string
	Details    map[string]interface{}
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// O2DMSClient implements the real O2 DMS client
type O2DMSClient struct {
	Endpoint   string
	Token      string
	httpClient *http.Client
	logger     *slog.Logger
	maxRetries int
	retryDelay time.Duration
}

// NewO2DMSClient creates a new O2 DMS client with configuration
func NewO2DMSClient(endpoint, token string) *O2DMSClient {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &O2DMSClient{
		Endpoint: endpoint,
		Token:    token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: true,
			},
		},
		logger:     logger,
		maxRetries: 3,
		retryDelay: time.Second,
	}
}

// CreateDeployment creates a deployment via O2 DMS API with retry logic
func (c *O2DMSClient) CreateDeployment(ctx context.Context, vnf *manov1alpha1.VNF) (string, error) {
	req := &O2DeploymentRequest{
		Name:           vnf.Name,
		Type:           string(vnf.Spec.Type),
		Version:        vnf.Spec.Version,
		TargetClusters: vnf.Spec.TargetClusters,
		Resources: map[string]interface{}{
			"cpu_cores": vnf.Spec.Resources.CPUCores,
			"memory_gb": vnf.Spec.Resources.MemoryGB,
		},
		QoSProfile: map[string]interface{}{
			"bandwidth": vnf.Spec.QoS.Bandwidth,
			"latency":   vnf.Spec.QoS.Latency,
			"jitter":    vnf.Spec.QoS.Jitter,
		},
		ConfigData: vnf.Spec.ConfigData,
	}

	var resp *O2DeploymentResponse
	err := c.retryOperation(ctx, func() error {
		var retryErr error
		resp, retryErr = c.createDeploymentRequest(ctx, req)
		return retryErr
	})

	if err != nil {
		c.logger.Error("Failed to create deployment", "vnf", vnf.Name, "error", err)
		return "", fmt.Errorf("failed to create O2 DMS deployment: %w", err)
	}

	c.logger.Info("Created deployment", "vnf", vnf.Name, "deployment_id", resp.DeploymentID)
	return resp.DeploymentID, nil
}

// GetDeploymentStatus gets deployment status via O2 DMS API with caching
func (c *O2DMSClient) GetDeploymentStatus(ctx context.Context, deploymentID string) (string, error) {
	var status *O2DeploymentStatus
	err := c.retryOperation(ctx, func() error {
		var retryErr error
		status, retryErr = c.getDeploymentStatusRequest(ctx, deploymentID)
		return retryErr
	})

	if err != nil {
		c.logger.Error("Failed to get deployment status", "deployment_id", deploymentID, "error", err)
		return "", fmt.Errorf("failed to get deployment status: %w", err)
	}

	return status.Status, nil
}

// UpdateDeployment updates deployment via O2 DMS API
func (c *O2DMSClient) UpdateDeployment(ctx context.Context, deploymentID string, vnf *manov1alpha1.VNF) error {
	req := &O2DeploymentRequest{
		Name:           vnf.Name,
		Type:           string(vnf.Spec.Type),
		Version:        vnf.Spec.Version,
		TargetClusters: vnf.Spec.TargetClusters,
		Resources: map[string]interface{}{
			"cpu_cores": vnf.Spec.Resources.CPUCores,
			"memory_gb": vnf.Spec.Resources.MemoryGB,
		},
		QoSProfile: map[string]interface{}{
			"bandwidth": vnf.Spec.QoS.Bandwidth,
			"latency":   vnf.Spec.QoS.Latency,
			"jitter":    vnf.Spec.QoS.Jitter,
		},
		ConfigData: vnf.Spec.ConfigData,
	}

	err := c.retryOperation(ctx, func() error {
		return c.updateDeploymentRequest(ctx, deploymentID, req)
	})

	if err != nil {
		c.logger.Error("Failed to update deployment", "deployment_id", deploymentID, "error", err)
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	c.logger.Info("Updated deployment", "deployment_id", deploymentID)
	return nil
}

// DeleteDeployment deletes deployment via O2 DMS API with graceful cleanup
func (c *O2DMSClient) DeleteDeployment(ctx context.Context, deploymentID string) error {
	err := c.retryOperation(ctx, func() error {
		return c.deleteDeploymentRequest(ctx, deploymentID)
	})

	if err != nil {
		c.logger.Error("Failed to delete deployment", "deployment_id", deploymentID, "error", err)
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	c.logger.Info("Deleted deployment", "deployment_id", deploymentID)
	return nil
}

// retryOperation implements exponential backoff retry logic
func (c *O2DMSClient) retryOperation(ctx context.Context, operation func() error) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * c.retryDelay
			c.logger.Debug("Retrying operation", "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		lastErr = operation()
		if lastErr == nil {
			return nil
		}

		// Don't retry on client errors (4xx)
		if httpErr, ok := lastErr.(*HTTPError); ok && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
			return lastErr
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// createDeploymentRequest makes the actual HTTP request to create deployment
func (c *O2DMSClient) createDeploymentRequest(ctx context.Context, req *O2DeploymentRequest) (*O2DeploymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint+"/api/v1/deployments", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, c.handleHTTPError(resp)
	}

	var result O2DeploymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// getDeploymentStatusRequest makes HTTP request to get deployment status
func (c *O2DMSClient) getDeploymentStatusRequest(ctx context.Context, deploymentID string) (*O2DeploymentStatus, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.Endpoint+"/api/v1/deployments/"+deploymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, c.handleHTTPError(resp)
	}

	var result O2DeploymentStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// updateDeploymentRequest makes HTTP request to update deployment
func (c *O2DMSClient) updateDeploymentRequest(ctx context.Context, deploymentID string, req *O2DeploymentRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", c.Endpoint+"/api/v1/deployments/"+deploymentID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.handleHTTPError(resp)
	}

	return nil
}

// deleteDeploymentRequest makes HTTP request to delete deployment
func (c *O2DMSClient) deleteDeploymentRequest(ctx context.Context, deploymentID string) error {
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", c.Endpoint+"/api/v1/deployments/"+deploymentID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.handleHTTPError(resp)
	}

	return nil
}

// setHeaders sets common HTTP headers for O2 DMS API requests
func (c *O2DMSClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("User-Agent", "O-RAN-MANO-VNF-Operator/1.0")
}

// handleHTTPError processes HTTP error responses
func (c *O2DMSClient) handleHTTPError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    "Failed to read error response",
		}
	}

	var errorResp map[string]interface{}
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	message := "Unknown error"
	if msg, ok := errorResp["message"].(string); ok {
		message = msg
	} else if msg, ok := errorResp["error"].(string); ok {
		message = msg
	}

	return &HTTPError{
		StatusCode: resp.StatusCode,
		Message:    message,
		Details:    errorResp,
	}
}
