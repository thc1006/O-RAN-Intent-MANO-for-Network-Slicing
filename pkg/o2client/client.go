// Package o2client provides client for O-RAN O2 interface
package o2client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// O2Client represents an O2 interface client with HTTP operations
type O2Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	username   string
	password   string
	maxRetries int
	mu         sync.RWMutex
}

// Client represents an O2 interface client (legacy alias)
type Client struct {
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new O2 client (legacy)
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
	}
}

// NewO2Client creates a new O2 client with HTTP operations
func NewO2Client(baseURL, username, password string) *O2Client {
	return &O2Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		maxRetries: 3,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

// SetTimeout sets the HTTP client timeout
func (c *O2Client) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient.Timeout = timeout
}

// SetMaxRetries sets the maximum number of retries
func (c *O2Client) SetMaxRetries(maxRetries int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxRetries = maxRetries
}

// GetToken returns the current authentication token
func (c *O2Client) GetToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// SetToken sets the authentication token
func (c *O2Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// Authenticate performs authentication with the O2 IMS
func (c *O2Client) Authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate credentials
	if c.username == "" || c.password == "" {
		return fmt.Errorf("credentials cannot be empty")
	}

	authURL := fmt.Sprintf("%s/auth/token", c.baseURL)

	payload := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	resp, err := c.httpClient.Post(authURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	// Handle non-OK status codes
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed: unauthorized")
	}

	if resp.StatusCode == http.StatusInternalServerError {
		return fmt.Errorf("authentication failed: server error")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	token, ok := result["token"]
	if !ok {
		return fmt.Errorf("no token in response")
	}

	c.token = token
	return nil
}

// GetResource retrieves a specific resource by ID
func (c *O2Client) GetResource(resourceID string) (interface{}, error) {
	if resourceID == "" {
		return nil, fmt.Errorf("resource ID cannot be empty")
	}

	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	resourceURL := fmt.Sprintf("%s/resources/%s", c.baseURL, resourceID)

	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req, c.maxRetries)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: token missing or invalid")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("resource not found: %s", resourceID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetResourceWithContext retrieves a specific resource with context
func (c *O2Client) GetResourceWithContext(ctx context.Context, resourceID string) error {
	if resourceID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}

	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	resourceURL := fmt.Sprintf("%s/resources/%s", c.baseURL, resourceID)

	req, err := http.NewRequestWithContext(ctx, "GET", resourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req, c.maxRetries)
	if err != nil {
		// Check if it's a context deadline exceeded error
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("request failed: context deadline exceeded")
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: token missing or invalid")
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("resource not found: %s", resourceID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// ListResources lists resources with optional filters
func (c *O2Client) ListResources(filter map[string]string) ([]interface{}, error) {
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	resourceURL := fmt.Sprintf("%s/resources", c.baseURL)

	// Add query parameters
	if filter != nil && len(filter) > 0 {
		params := url.Values{}
		for k, v := range filter {
			params.Add(k, v)
		}
		resourceURL = fmt.Sprintf("%s?%s", resourceURL, params.Encode())
	}

	req, err := http.NewRequest("GET", resourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req, c.maxRetries)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: status %d", resp.StatusCode)
	}

	var result []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// CreateResource creates a new resource
func (c *O2Client) CreateResource(resource map[string]interface{}) (map[string]interface{}, error) {
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	resourceURL := fmt.Sprintf("%s/resources", c.baseURL)

	jsonData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}

	req, err := http.NewRequest("POST", resourceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req, c.maxRetries)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("request failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// UpdateResource updates an existing resource
func (c *O2Client) UpdateResource(resourceID string, resource map[string]interface{}) (map[string]interface{}, error) {
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	resourceURL := fmt.Sprintf("%s/resources/%s", c.baseURL, resourceID)

	jsonData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}

	req, err := http.NewRequest("PUT", resourceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req, c.maxRetries)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// DeleteResource deletes a resource
func (c *O2Client) DeleteResource(resourceID string) error {
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	resourceURL := fmt.Sprintf("%s/resources/%s", c.baseURL, resourceID)

	req, err := http.NewRequest("DELETE", resourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doWithRetry(req, c.maxRetries)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("request failed: status %d", resp.StatusCode)
	}

	return nil
}

// doWithRetry executes an HTTP request with exponential backoff retry
func (c *O2Client) doWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	c.mu.RLock()
	currentMaxRetries := c.maxRetries
	c.mu.RUnlock()

	// Use the instance's maxRetries if maxRetries parameter is the default
	if maxRetries == 3 {
		maxRetries = currentMaxRetries
	}

	var resp *http.Response
	var err error
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone the request for retry
		var reqClone *http.Request
		if req.Body != nil {
			// For requests with body, we need to read and restore
			bodyBytes, readErr := io.ReadAll(req.Body)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read request body: %w", readErr)
			}
			req.Body.Close()

			// Create new request with fresh body
			reqClone, err = http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), bytes.NewBuffer(bodyBytes))
			if err != nil {
				return nil, fmt.Errorf("failed to clone request: %w", err)
			}
			reqClone.Header = req.Header.Clone()

			// Restore original body for potential future use
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		} else {
			reqClone = req
		}

		resp, err = c.httpClient.Do(reqClone)
		lastErr = err

		// Success or non-retryable error
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// Close response body before retry
		if resp != nil {
			resp.Body.Close()
		}

		// Don't sleep after last attempt
		if attempt < maxRetries {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoff)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
	}
	return nil, fmt.Errorf("max retries exceeded with status: %d", resp.StatusCode)
}

// DeploymentManager represents an O2 DMS
type DeploymentManager struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Status      string `json:"status"`
}

// GetDeploymentManagers retrieves available deployment managers
func (c *Client) GetDeploymentManagers(_ context.Context) ([]DeploymentManager, error) {
	// Placeholder implementation for legacy client
	return []DeploymentManager{
		{
			ID:          "ran-dms",
			Name:        "RAN DMS",
			Description: "RAN deployment management service",
			URL:         fmt.Sprintf("%s/ran-dms", c.BaseURL),
			Status:      "active",
		},
		{
			ID:          "cn-dms",
			Name:        "CN DMS",
			Description: "Core network deployment management service",
			URL:         fmt.Sprintf("%s/cn-dms", c.BaseURL),
			Status:      "active",
		},
	}, nil
}

// DeployNetworkFunction deploys a network function via O2 DMS
func (c *Client) DeployNetworkFunction(_ context.Context, _ string, _ interface{}) error {
	// Placeholder implementation for legacy client
	return nil
}

// GetAvailableSites retrieves available deployment sites
func (c *Client) GetAvailableSites(_ context.Context) ([]string, error) {
	// Placeholder implementation for legacy client
	return []string{"edge-site-1", "edge-site-2", "regional-site-1"}, nil
}

// DeploymentStatus represents the status of a deployed function
type DeploymentStatus struct {
	Name      string
	Type      string
	Cluster   string
	Namespace string
	Status    string
	IPAddress string
	Metrics   map[string]float64
}

// GetDeploymentStatus retrieves the status of a deployment
func (c *Client) GetDeploymentStatus(_ context.Context, deploymentID string) ([]DeploymentStatus, error) {
	// Placeholder implementation for legacy client
	return []DeploymentStatus{
		{
			Name:      "ran-cu-" + deploymentID,
			Type:      "CU",
			Cluster:   "edge-cluster-1",
			Namespace: "ran-ns",
			Status:    "Ready",
			IPAddress: "10.0.1.10",
			Metrics:   map[string]float64{"cpu": 45.2, "memory": 62.1},
		},
		{
			Name:      "ran-du-" + deploymentID,
			Type:      "DU",
			Cluster:   "edge-cluster-1",
			Namespace: "ran-ns",
			Status:    "Ready",
			IPAddress: "10.0.1.11",
			Metrics:   map[string]float64{"cpu": 38.5, "memory": 55.3},
		},
	}, nil
}

// DeleteDeployment deletes a deployment
func (c *Client) DeleteDeployment(_ context.Context, _ string) error {
	// Placeholder implementation for legacy client
	return nil
}
