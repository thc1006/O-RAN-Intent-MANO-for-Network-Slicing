package o2dms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/models"
)

// Client provides O2DMS (O-RAN Deployment Management Service) operations
type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
	timeout    time.Duration
}

// ClientOption configures the O2DMS client
type ClientOption func(*Client)

// WithTimeout sets the client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithAuthToken sets the authentication token
func WithAuthToken(token string) ClientOption {
	return func(c *Client) {
		c.authToken = token
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new O2DMS client
func NewClient(baseURL string, options ...ClientOption) *Client {
	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// DeploymentRequest represents a request to deploy an NF
type DeploymentRequest struct {
	Name                         string                 `json:"name"`
	Description                  string                 `json:"description,omitempty"`
	NFDeploymentDescriptorID     string                 `json:"nfDeploymentDescriptorId"`
	ParentDeploymentID           string                 `json:"parentDeploymentId,omitempty"`
	InputParams                  map[string]interface{} `json:"inputParams,omitempty"`
	LocationConstraints          []string               `json:"locationConstraints,omitempty"`
	Extensions                   map[string]interface{} `json:"extensions,omitempty"`
}

// ListQuery represents query parameters for list operations
type ListQuery struct {
	Limit          int      `json:"limit,omitempty"`
	Offset         int      `json:"offset,omitempty"`
	Marker         string   `json:"marker,omitempty"`
	Filter         string   `json:"filter,omitempty"`
	Sort           string   `json:"sort,omitempty"`
	AllFields      bool     `json:"all_fields,omitempty"`
	Fields         []string `json:"fields,omitempty"`
	ExcludeFields  []string `json:"exclude_fields,omitempty"`
	ExcludeDefault bool     `json:"exclude_default,omitempty"`
}

// Deployment Manager Operations

// ListDeploymentManagers retrieves available deployment managers
func (c *Client) ListDeploymentManagers(ctx context.Context, query *ListQuery) ([]*models.DeploymentManager, error) {
	var response models.ListResponse
	endpoint := "/o2dms/v1/deploymentManagers"

	if query != nil {
		params := c.buildQueryParams(query)
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	err := c.doRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployment managers: %w", err)
	}

	managers := make([]*models.DeploymentManager, len(response.Items))
	for i, item := range response.Items {
		managerData, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal deployment manager %d: %w", i, err)
		}

		var manager models.DeploymentManager
		if err := json.Unmarshal(managerData, &manager); err != nil {
			return nil, fmt.Errorf("failed to unmarshal deployment manager %d: %w", i, err)
		}
		managers[i] = &manager
	}

	return managers, nil
}

// GetDeploymentManager retrieves a specific deployment manager
func (c *Client) GetDeploymentManager(ctx context.Context, deploymentManagerID string) (*models.DeploymentManager, error) {
	var manager models.DeploymentManager
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s", deploymentManagerID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &manager)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment manager %s: %w", deploymentManagerID, err)
	}

	return &manager, nil
}

// NF Deployment Descriptor Operations

// ListNFDeploymentDescriptors retrieves available NF deployment descriptors
func (c *Client) ListNFDeploymentDescriptors(ctx context.Context, deploymentManagerID string, query *ListQuery) ([]*models.NFDeploymentDescriptor, error) {
	var response models.ListResponse
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeploymentDescriptors", deploymentManagerID)

	if query != nil {
		params := c.buildQueryParams(query)
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	err := c.doRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list NF deployment descriptors: %w", err)
	}

	descriptors := make([]*models.NFDeploymentDescriptor, len(response.Items))
	for i, item := range response.Items {
		descriptorData, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal NF deployment descriptor %d: %w", i, err)
		}

		var descriptor models.NFDeploymentDescriptor
		if err := json.Unmarshal(descriptorData, &descriptor); err != nil {
			return nil, fmt.Errorf("failed to unmarshal NF deployment descriptor %d: %w", i, err)
		}
		descriptors[i] = &descriptor
	}

	return descriptors, nil
}

// GetNFDeploymentDescriptor retrieves a specific NF deployment descriptor
func (c *Client) GetNFDeploymentDescriptor(ctx context.Context, deploymentManagerID, descriptorID string) (*models.NFDeploymentDescriptor, error) {
	var descriptor models.NFDeploymentDescriptor
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeploymentDescriptors/%s",
		deploymentManagerID, descriptorID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &descriptor)
	if err != nil {
		return nil, fmt.Errorf("failed to get NF deployment descriptor %s: %w", descriptorID, err)
	}

	return &descriptor, nil
}

// NF Deployment Operations

// CreateNFDeployment creates a new NF deployment
func (c *Client) CreateNFDeployment(ctx context.Context, deploymentManagerID string, request *DeploymentRequest) (*models.NFDeployment, error) {
	var deployment models.NFDeployment
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments", deploymentManagerID)

	err := c.doRequest(ctx, "POST", endpoint, request, &deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to create NF deployment: %w", err)
	}

	return &deployment, nil
}

// ListNFDeployments retrieves NF deployments
func (c *Client) ListNFDeployments(ctx context.Context, deploymentManagerID string, query *ListQuery) ([]*models.NFDeployment, error) {
	var response models.ListResponse
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments", deploymentManagerID)

	if query != nil {
		params := c.buildQueryParams(query)
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	err := c.doRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list NF deployments: %w", err)
	}

	deployments := make([]*models.NFDeployment, len(response.Items))
	for i, item := range response.Items {
		deploymentData, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal NF deployment %d: %w", i, err)
		}

		var deployment models.NFDeployment
		if err := json.Unmarshal(deploymentData, &deployment); err != nil {
			return nil, fmt.Errorf("failed to unmarshal NF deployment %d: %w", i, err)
		}
		deployments[i] = &deployment
	}

	return deployments, nil
}

// GetNFDeployment retrieves a specific NF deployment
func (c *Client) GetNFDeployment(ctx context.Context, deploymentManagerID, deploymentID string) (*models.NFDeployment, error) {
	var deployment models.NFDeployment
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s",
		deploymentManagerID, deploymentID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get NF deployment %s: %w", deploymentID, err)
	}

	return &deployment, nil
}

// UpdateNFDeployment updates an existing NF deployment
func (c *Client) UpdateNFDeployment(ctx context.Context, deploymentManagerID, deploymentID string, request *DeploymentRequest) (*models.NFDeployment, error) {
	var deployment models.NFDeployment
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s",
		deploymentManagerID, deploymentID)

	err := c.doRequest(ctx, "PUT", endpoint, request, &deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to update NF deployment %s: %w", deploymentID, err)
	}

	return &deployment, nil
}

// DeleteNFDeployment deletes an NF deployment
func (c *Client) DeleteNFDeployment(ctx context.Context, deploymentManagerID, deploymentID string) error {
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s",
		deploymentManagerID, deploymentID)

	err := c.doRequest(ctx, "DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete NF deployment %s: %w", deploymentID, err)
	}

	return nil
}

// Subscription Operations

// CreateSubscription creates a new subscription for deployment notifications
func (c *Client) CreateSubscription(ctx context.Context, subscription *models.Subscription) (*models.Subscription, error) {
	var createdSubscription models.Subscription
	endpoint := "/o2dms/v1/subscriptions"

	err := c.doRequest(ctx, "POST", endpoint, subscription, &createdSubscription)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return &createdSubscription, nil
}

// GetSubscription retrieves a subscription by ID
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	var subscription models.Subscription
	endpoint := fmt.Sprintf("/o2dms/v1/subscriptions/%s", subscriptionID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription %s: %w", subscriptionID, err)
	}

	return &subscription, nil
}

// DeleteSubscription deletes a subscription
func (c *Client) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	endpoint := fmt.Sprintf("/o2dms/v1/subscriptions/%s", subscriptionID)

	err := c.doRequest(ctx, "DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete subscription %s: %w", subscriptionID, err)
	}

	return nil
}

// Health Check Operations

// GetHealthInfo retrieves health information for the DMS
func (c *Client) GetHealthInfo(ctx context.Context) (*models.HealthInfo, error) {
	var health models.HealthInfo
	endpoint := "/o2dms/v1/health"

	err := c.doRequest(ctx, "GET", endpoint, nil, &health)
	if err != nil {
		return nil, fmt.Errorf("failed to get health info: %w", err)
	}

	return &health, nil
}

// Helper methods

// buildQueryParams converts ListQuery to URL parameters
func (c *Client) buildQueryParams(query *ListQuery) url.Values {
	params := url.Values{}

	if query.Limit > 0 {
		params.Add("limit", strconv.Itoa(query.Limit))
	}
	if query.Offset > 0 {
		params.Add("offset", strconv.Itoa(query.Offset))
	}
	if query.Marker != "" {
		params.Add("marker", query.Marker)
	}
	if query.Filter != "" {
		params.Add("filter", query.Filter)
	}
	if query.Sort != "" {
		params.Add("sort", query.Sort)
	}
	if query.AllFields {
		params.Add("all_fields", "true")
	}
	if len(query.Fields) > 0 {
		for _, field := range query.Fields {
			params.Add("fields", field)
		}
	}
	if len(query.ExcludeFields) > 0 {
		for _, field := range query.ExcludeFields {
			params.Add("exclude_fields", field)
		}
	}
	if query.ExcludeDefault {
		params.Add("exclude_default", "true")
	}

	return params
}

// doRequest performs HTTP requests with proper error handling
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	url := c.baseURL + endpoint

	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiError models.APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return fmt.Errorf("API error %d: %s - %s", apiError.Status, apiError.Title, apiError.Detail)
	}

	// Decode response if result is provided
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// O-RAN Specific Helper Methods

// DeployVNFWithQoS deploys a VNF with specific QoS requirements
func (c *Client) DeployVNFWithQoS(ctx context.Context, deploymentManagerID string, vnfType string, qosReq *models.ORanQoSRequirements, placement *models.ORanPlacement) (*models.NFDeployment, error) {
	// Find appropriate deployment descriptor for the VNF type
	descriptors, err := c.ListNFDeploymentDescriptors(ctx, deploymentManagerID, &ListQuery{
		Filter: fmt.Sprintf("vnfType==%s", vnfType),
		Limit:  10,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find deployment descriptor for VNF type %s: %w", vnfType, err)
	}

	if len(descriptors) == 0 {
		return nil, fmt.Errorf("no deployment descriptor found for VNF type %s", vnfType)
	}

	// Use the first matching descriptor
	descriptor := descriptors[0]

	// Build deployment request with QoS and placement requirements
	request := &DeploymentRequest{
		Name:                     fmt.Sprintf("%s-deployment-%d", vnfType, time.Now().Unix()),
		Description:              fmt.Sprintf("VNF deployment for %s with QoS requirements", vnfType),
		NFDeploymentDescriptorID: descriptor.ID,
		InputParams: map[string]interface{}{
			"qosRequirements": qosReq,
			"placement":       placement,
			"vnfType":         vnfType,
		},
		LocationConstraints: []string{placement.CloudType},
		Extensions: map[string]interface{}{
			"oran.io/qos-bandwidth":  qosReq.Bandwidth,
			"oran.io/qos-latency":    qosReq.Latency,
			"oran.io/cloud-type":     placement.CloudType,
			"oran.io/slice-type":     qosReq.SliceType,
		},
	}

	// Add location constraints based on placement
	if placement.Region != "" {
		request.LocationConstraints = append(request.LocationConstraints, placement.Region)
	}
	if placement.Zone != "" {
		request.LocationConstraints = append(request.LocationConstraints, placement.Zone)
	}

	// Create the deployment
	deployment, err := c.CreateNFDeployment(ctx, deploymentManagerID, request)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy VNF: %w", err)
	}

	return deployment, nil
}

// WaitForDeploymentReady waits for a deployment to reach ready state
func (c *Client) WaitForDeploymentReady(ctx context.Context, deploymentManagerID, deploymentID string, timeout time.Duration) (*models.NFDeployment, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		deployment, err := c.GetNFDeployment(ctx, deploymentManagerID, deploymentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment status: %w", err)
		}

		switch deployment.Status {
		case models.NFDeploymentStatusInstantiated:
			return deployment, nil
		case models.NFDeploymentStatusFailed:
			return nil, fmt.Errorf("deployment failed: %s", deploymentID)
		}

		// Wait before next check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue loop
		}
	}

	return nil, fmt.Errorf("deployment did not become ready within timeout: %s", deploymentID)
}

// GetDeploymentsBySliceType retrieves deployments for a specific slice type
func (c *Client) GetDeploymentsBySliceType(ctx context.Context, deploymentManagerID, sliceType string) ([]*models.NFDeployment, error) {
	query := &ListQuery{
		Filter: fmt.Sprintf("sliceType==%s", sliceType),
		Limit:  100,
	}

	deployments, err := c.ListNFDeployments(ctx, deploymentManagerID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments for slice type %s: %w", sliceType, err)
	}

	return deployments, nil
}