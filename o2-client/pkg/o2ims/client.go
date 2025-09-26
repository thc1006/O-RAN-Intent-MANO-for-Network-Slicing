package o2ims

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/models"
)

// Client provides O2IMS (O-RAN Infrastructure Management Service) operations
type Client struct {
	baseURL      string
	httpClient   *http.Client
	authToken    string
	timeout      time.Duration
	retryConfig  RetryConfig
	eventChan    chan Event
	subscribers  map[string][]EventHandler
	mutex        sync.RWMutex
	metrics      *ClientMetrics
}

// ClientOption configures the O2IMS client
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

// NewClient creates a new O2IMS client
func NewClient(baseURL string, options ...ClientOption) *Client {
	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout:      30 * time.Second,
		retryConfig:  DefaultRetryConfig(),
		eventChan:    make(chan Event, 1000),
		subscribers:  make(map[string][]EventHandler),
		metrics:      NewClientMetrics(),
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// ListQuery represents query parameters for list operations
type ListQuery struct {
	Limit        int               `json:"limit,omitempty"`
	Offset       int               `json:"offset,omitempty"`
	Marker       string            `json:"marker,omitempty"`
	Filter       string            `json:"filter,omitempty"`
	Sort         string            `json:"sort,omitempty"`
	AllFields    bool              `json:"all_fields,omitempty"`
	Fields       []string          `json:"fields,omitempty"`
	ExcludeFields []string         `json:"exclude_fields,omitempty"`
	ExcludeDefault bool            `json:"exclude_default,omitempty"`
}

// O-Cloud Operations

// GetOCloudInfo retrieves information about the O-Cloud
func (c *Client) GetOCloudInfo(ctx context.Context) (*models.OCloudInfo, error) {
	var ocloud models.OCloudInfo
	err := c.doRequest(ctx, "GET", "/o2ims-infrastructureInventory/v1/", nil, &ocloud)
	if err != nil {
		return nil, fmt.Errorf("failed to get O-Cloud info: %w", err)
	}
	return &ocloud, nil
}

// Resource Pool Operations

// ListResourcePools retrieves a list of resource pools
func (c *Client) ListResourcePools(ctx context.Context, query *ListQuery) ([]*models.O2CloudResourcePool, error) {
	var response models.ListResponse
	endpoint := "/o2ims-infrastructureInventory/v1/resourcePools"

	if query != nil {
		params := c.buildQueryParams(query)
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	err := c.doRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource pools: %w", err)
	}

	pools := make([]*models.O2CloudResourcePool, len(response.Items))
	for i, item := range response.Items {
		poolData, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource pool %d: %w", i, err)
		}

		var pool models.O2CloudResourcePool
		if err := json.Unmarshal(poolData, &pool); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource pool %d: %w", i, err)
		}
		pools[i] = &pool
	}

	return pools, nil
}

// GetResourcePool retrieves a specific resource pool by ID
func (c *Client) GetResourcePool(ctx context.Context, resourcePoolID string) (*models.O2CloudResourcePool, error) {
	var pool models.O2CloudResourcePool
	endpoint := fmt.Sprintf("/o2ims-infrastructureInventory/v1/resourcePools/%s", resourcePoolID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &pool)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource pool %s: %w", resourcePoolID, err)
	}

	return &pool, nil
}

// Resource Operations

// ListResources retrieves resources from a specific resource pool
func (c *Client) ListResources(ctx context.Context, resourcePoolID string, query *ListQuery) ([]*models.Resource, error) {
	var response models.ListResponse
	endpoint := fmt.Sprintf("/o2ims-infrastructureInventory/v1/resourcePools/%s/resources", resourcePoolID)

	if query != nil {
		params := c.buildQueryParams(query)
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	err := c.doRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources for pool %s: %w", resourcePoolID, err)
	}

	resources := make([]*models.Resource, len(response.Items))
	for i, item := range response.Items {
		resourceData, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource %d: %w", i, err)
		}

		var resource models.Resource
		if err := json.Unmarshal(resourceData, &resource); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource %d: %w", i, err)
		}
		resources[i] = &resource
	}

	return resources, nil
}

// GetResource retrieves a specific resource
func (c *Client) GetResource(ctx context.Context, resourcePoolID, resourceID string) (*models.Resource, error) {
	var resource models.Resource
	endpoint := fmt.Sprintf("/o2ims-infrastructureInventory/v1/resourcePools/%s/resources/%s",
		resourcePoolID, resourceID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s/%s: %w", resourcePoolID, resourceID, err)
	}

	return &resource, nil
}

// Resource Type Operations

// ListResourceTypes retrieves available resource types
func (c *Client) ListResourceTypes(ctx context.Context, query *ListQuery) ([]*models.ResourceTypeInfo, error) {
	var response models.ListResponse
	endpoint := "/o2ims-infrastructureInventory/v1/resourceTypes"

	if query != nil {
		params := c.buildQueryParams(query)
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	err := c.doRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource types: %w", err)
	}

	resourceTypes := make([]*models.ResourceTypeInfo, len(response.Items))
	for i, item := range response.Items {
		typeData, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource type %d: %w", i, err)
		}

		var resourceType models.ResourceTypeInfo
		if err := json.Unmarshal(typeData, &resourceType); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource type %d: %w", i, err)
		}
		resourceTypes[i] = &resourceType
	}

	return resourceTypes, nil
}

// GetResourceType retrieves a specific resource type
func (c *Client) GetResourceType(ctx context.Context, resourceTypeID string) (*models.ResourceTypeInfo, error) {
	var resourceType models.ResourceTypeInfo
	endpoint := fmt.Sprintf("/o2ims-infrastructureInventory/v1/resourceTypes/%s", resourceTypeID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource type %s: %w", resourceTypeID, err)
	}

	return &resourceType, nil
}

// Subscription Operations

// CreateSubscription creates a new subscription for notifications
func (c *Client) CreateSubscription(ctx context.Context, subscription *models.Subscription) (*models.Subscription, error) {
	var createdSubscription models.Subscription
	endpoint := "/o2ims-infrastructureInventory/v1/subscriptions"

	err := c.doRequest(ctx, "POST", endpoint, subscription, &createdSubscription)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return &createdSubscription, nil
}

// GetSubscription retrieves a subscription by ID
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	var subscription models.Subscription
	endpoint := fmt.Sprintf("/o2ims-infrastructureInventory/v1/subscriptions/%s", subscriptionID)

	err := c.doRequest(ctx, "GET", endpoint, nil, &subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription %s: %w", subscriptionID, err)
	}

	return &subscription, nil
}

// DeleteSubscription deletes a subscription
func (c *Client) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	endpoint := fmt.Sprintf("/o2ims-infrastructureInventory/v1/subscriptions/%s", subscriptionID)

	err := c.doRequest(ctx, "DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete subscription %s: %w", subscriptionID, err)
	}

	return nil
}

// Health Check Operations

// GetHealthInfo retrieves health information
func (c *Client) GetHealthInfo(ctx context.Context) (*models.HealthInfo, error) {
	var health models.HealthInfo
	endpoint := "/o2ims-infrastructureInventory/v1/health"

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

// FindResourcesByQoS finds resources that meet specific QoS requirements
func (c *Client) FindResourcesByQoS(ctx context.Context, qosReq *models.ORanQoSRequirements) ([]*models.Resource, error) {
	// Build filter based on QoS requirements
	filter := fmt.Sprintf("bandwidth>=%f,latency<=%f", qosReq.Bandwidth, qosReq.Latency)

	query := &ListQuery{
		Filter: filter,
		Limit:  100,
	}

	// Get all resource pools
	pools, err := c.ListResourcePools(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource pools: %w", err)
	}

	var matchingResources []*models.Resource

	// Search resources in each pool
	for _, pool := range pools {
		resources, err := c.ListResources(ctx, pool.ResourcePoolID, query)
		if err != nil {
			continue // Skip pools that fail
		}

		// Filter resources based on QoS requirements
		for _, resource := range resources {
			if c.resourceMeetsQoS(resource, qosReq) {
				matchingResources = append(matchingResources, resource)
			}
		}
	}

	return matchingResources, nil
}

// FindResourcesByPlacement finds resources that meet placement requirements
func (c *Client) FindResourcesByPlacement(ctx context.Context, placement *models.ORanPlacement) ([]*models.Resource, error) {
	// Build filter based on placement requirements
	filter := fmt.Sprintf("cloudType==%s", placement.CloudType)
	if placement.Region != "" {
		filter += fmt.Sprintf(",region==%s", placement.Region)
	}
	if placement.Zone != "" {
		filter += fmt.Sprintf(",zone==%s", placement.Zone)
	}

	query := &ListQuery{
		Filter: filter,
		Limit:  100,
	}

	// Get resource pools matching placement criteria
	pools, err := c.ListResourcePools(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource pools: %w", err)
	}

	var matchingResources []*models.Resource

	// Collect resources from matching pools
	for _, pool := range pools {
		resources, err := c.ListResources(ctx, pool.ResourcePoolID, nil)
		if err != nil {
			continue // Skip pools that fail
		}
		matchingResources = append(matchingResources, resources...)
	}

	return matchingResources, nil
}

// resourceMeetsQoS checks if a resource meets QoS requirements
func (c *Client) resourceMeetsQoS(resource *models.Resource, qosReq *models.ORanQoSRequirements) bool {
	// This is a simplified check - in practice, you would examine
	// resource properties and capabilities

	// Check if resource has sufficient capacity
	if bandwidth, ok := resource.Extensions["bandwidth"].(float64); ok {
		if bandwidth < qosReq.Bandwidth {
			return false
		}
	}

	if latency, ok := resource.Extensions["latency"].(float64); ok {
		if latency > qosReq.Latency {
			return false
		}
	}

	return true
}