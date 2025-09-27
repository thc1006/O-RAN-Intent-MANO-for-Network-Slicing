package o2ims

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/models"
)

// Enhanced O2 IMS Client Methods with retry logic and event handling

// SetRetryConfig configures retry behavior
func (c *Client) SetRetryConfig(config RetryConfig) {
	c.retryConfig = config
}

// GetMetrics returns client performance metrics
func (c *Client) GetMetrics() (int64, int64, int64, time.Duration) {
	return c.metrics.GetMetrics()
}

// Event Management

// Subscribe adds an event handler for specific event types
func (c *Client) Subscribe(eventTypes []EventType, handler EventHandler) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	subscriptionID := fmt.Sprintf("sub_%d", time.Now().UnixNano())

	for _, eventType := range eventTypes {
		key := string(eventType)
		if c.subscribers[key] == nil {
			c.subscribers[key] = make([]EventHandler, 0)
		}
		c.subscribers[key] = append(c.subscribers[key], handler)
	}

	return subscriptionID
}

// StartEventProcessing starts processing events
func (c *Client) StartEventProcessing(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-c.eventChan:
				c.processEvent(event)
			}
		}
	}()
}

// publishEvent publishes an event to subscribers
func (c *Client) publishEvent(event Event) {
	select {
	case c.eventChan <- event:
	default:
		log.Printf("Event channel full, dropping event: %s", event.ID)
	}
}

// processEvent processes an event and notifies subscribers
func (c *Client) processEvent(event Event) {
	c.mutex.RLock()
	handlers := append([]EventHandler(nil), c.subscribers[string(event.Type)]...)
	c.mutex.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Event handler panic: %v", r)
				}
			}()
			h(event)
		}(handler)
	}
}

// Enhanced O2 IMS Operations with Retry Logic

// GetInventoryWithRetry retrieves the complete infrastructure inventory with retry
func (c *Client) GetInventoryWithRetry(ctx context.Context) (*models.InventoryInfo, error) {
	var inventory models.InventoryInfo
	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", "/o2ims-infrastructureInventory/v1/", nil, &inventory)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	return &inventory, nil
}

// GetResourceTypesWithRetry retrieves all available resource types with retry
func (c *Client) GetResourceTypesWithRetry(ctx context.Context, filter models.ResourceTypeFilter) (*models.ResourceTypeCollection, error) {
	params := c.buildResourceTypeParams(filter)
	var collection models.ResourceTypeCollection
	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", "/o2ims-infrastructureInventory/v1/resourceTypes", params, &collection)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource types: %w", err)
	}
	return &collection, nil
}

// GetResourcePoolsWithRetry retrieves all resource pools with retry
func (c *Client) GetResourcePoolsWithRetry(ctx context.Context, filter models.ResourcePoolFilter) (*models.ResourcePoolCollection, error) {
	params := c.buildResourcePoolParams(filter)
	var collection models.ResourcePoolCollection
	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", "/o2ims-infrastructureInventory/v1/resourcePools", params, &collection)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource pools: %w", err)
	}
	return &collection, nil
}

// GetResourcesWithRetry retrieves resources from a specific pool with retry
func (c *Client) GetResourcesWithRetry(ctx context.Context, resourcePoolID string, filter models.ResourceFilter) (*models.ResourceCollection, error) {
	params := c.buildResourceParams(filter)
	var collection models.ResourceCollection
	endpoint := fmt.Sprintf("/o2ims-infrastructureInventory/v1/resourcePools/%s/resources", resourcePoolID)
	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", endpoint, params, &collection)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get resources from pool %s: %w", resourcePoolID, err)
	}
	return &collection, nil
}

// Health Monitoring

// StartHealthMonitoring starts periodic health monitoring
func (c *Client) StartHealthMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.performHealthCheck(ctx)
			}
		}
	}()
}

// performHealthCheck performs a health check
func (c *Client) performHealthCheck(ctx context.Context) {
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.GetHealthInfo(healthCtx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		c.publishEvent(Event{
			ID:        fmt.Sprintf("health_%d", time.Now().UnixNano()),
			Type:      "health.check.failed",
			Source:    "o2ims.client",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"error": err.Error()},
			Severity:  SeverityError,
		})
	} else {
		c.publishEvent(Event{
			ID:        fmt.Sprintf("health_%d", time.Now().UnixNano()),
			Type:      "health.check.success",
			Source:    "o2ims.client",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{},
			Severity:  SeverityInfo,
		})
	}
}

// Helper methods for building query parameters

func (c *Client) buildResourceTypeParams(filter models.ResourceTypeFilter) map[string]string {
	params := make(map[string]string)
	if filter.Limit > 0 {
		params["limit"] = fmt.Sprintf("%d", filter.Limit)
	}
	if filter.Offset > 0 {
		params["offset"] = fmt.Sprintf("%d", filter.Offset)
	}
	if filter.Name != "" {
		params["filter"] = fmt.Sprintf("name eq '%s'", filter.Name)
	}
	if filter.Vendor != "" {
		if params["filter"] != "" {
			params["filter"] += fmt.Sprintf(" and vendor eq '%s'", filter.Vendor)
		} else {
			params["filter"] = fmt.Sprintf("vendor eq '%s'", filter.Vendor)
		}
	}
	return params
}

func (c *Client) buildResourcePoolParams(filter models.ResourcePoolFilter) map[string]string {
	params := make(map[string]string)
	if filter.Limit > 0 {
		params["limit"] = fmt.Sprintf("%d", filter.Limit)
	}
	if filter.Offset > 0 {
		params["offset"] = fmt.Sprintf("%d", filter.Offset)
	}
	if filter.Name != "" {
		params["filter"] = fmt.Sprintf("name eq '%s'", filter.Name)
	}
	if filter.OCloudID != "" {
		if params["filter"] != "" {
			params["filter"] += fmt.Sprintf(" and oCloudId eq '%s'", filter.OCloudID)
		} else {
			params["filter"] = fmt.Sprintf("oCloudId eq '%s'", filter.OCloudID)
		}
	}
	return params
}

func (c *Client) buildResourceParams(filter models.ResourceFilter) map[string]string {
	params := make(map[string]string)
	if filter.Limit > 0 {
		params["limit"] = fmt.Sprintf("%d", filter.Limit)
	}
	if filter.Offset > 0 {
		params["offset"] = fmt.Sprintf("%d", filter.Offset)
	}
	if filter.ResourceTypeID != "" {
		params["filter"] = fmt.Sprintf("resourceTypeId eq '%s'", filter.ResourceTypeID)
	}
	return params
}

// retryWithBackoff implements exponential backoff retry logic
func (c *Client) retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateBackoffDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			c.metrics.RecordRetry()
		}

		startTime := time.Now()
		if err := fn(); err != nil {
			lastErr = err
			c.metrics.RecordError()

			if attempt == c.retryConfig.MaxRetries {
				break
			}

			// Check if error is retryable
			if !c.isRetryableError(err) {
				return err
			}

			log.Printf("Attempt %d failed, retrying: %v", attempt+1, err)
			continue
		}

		c.metrics.RecordRequest(time.Since(startTime))
		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateBackoffDelay calculates delay with exponential backoff and jitter
func (c *Client) calculateBackoffDelay(attempt int) time.Duration {
	delay := float64(c.retryConfig.InitialDelay) * math.Pow(c.retryConfig.BackoffFactor, float64(attempt-1))

	// Add jitter (±25%)
	jitter := delay * 0.25 * (2*rand.Float64() - 1)
	delay += jitter

	maxDelay := float64(c.retryConfig.MaxDelay)
	if delay > maxDelay {
		delay = maxDelay
	}

	return time.Duration(delay)
}

// isRetryableError determines if an error should trigger a retry
func (c *Client) isRetryableError(err error) bool {
	// In a real implementation, you would check specific error types
	// For now, retry on most errors
	return true
}

// Advanced Resource Discovery

// DiscoverResourcesByCapabilities discovers resources by their capabilities
func (c *Client) DiscoverResourcesByCapabilities(ctx context.Context, capabilities map[string]interface{}) ([]*models.Resource, error) {
	// Get all resource pools
	pools, err := c.GetResourcePoolsWithRetry(ctx, models.ResourcePoolFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource pools: %w", err)
	}

	var matchingResources []*models.Resource

	// Search each pool
	for _, pool := range pools.Items {
		resources, err := c.GetResourcesWithRetry(ctx, pool.ResourcePoolID, models.ResourceFilter{})
		if err != nil {
			log.Printf("Failed to get resources from pool %s: %v", pool.ResourcePoolID, err)
			continue
		}

		// Filter by capabilities
		for _, resource := range resources.Items {
			if c.resourceHasCapabilities(&resource, capabilities) {
				matchingResources = append(matchingResources, &resource)
			}
		}
	}

	return matchingResources, nil
}

// resourceHasCapabilities checks if a resource has the required capabilities
func (c *Client) resourceHasCapabilities(resource *models.Resource, requiredCapabilities map[string]interface{}) bool {
	if resource.Extensions == nil {
		return false
	}

	for key, requiredValue := range requiredCapabilities {
		resourceValue, exists := resource.Extensions[key]
		if !exists {
			return false
		}

		// Simple equality check - in practice, you might need more sophisticated comparison
		if resourceValue != requiredValue {
			return false
		}
	}

	return true
}

// Batch Operations

// BatchGetResources retrieves multiple resources concurrently
func (c *Client) BatchGetResources(ctx context.Context, requests []ResourceRequest) ([]ResourceResult, error) {
	resultChan := make(chan ResourceResult, len(requests))

	for _, req := range requests {
		go func(request ResourceRequest) {
			resource, err := c.GetResource(ctx, request.PoolID, request.ResourceID)
			resultChan <- ResourceResult{
				Request:  request,
				Resource: resource,
				Error:    err,
			}
		}(req)
	}

	results := make([]ResourceResult, 0, len(requests))
	for i := 0; i < len(requests); i++ {
		results = append(results, <-resultChan)
	}

	return results, nil
}

// ResourceRequest represents a request for a specific resource
type ResourceRequest struct {
	PoolID     string
	ResourceID string
}

// ResourceResult represents the result of a resource request
type ResourceResult struct {
	Request  ResourceRequest
	Resource *models.Resource
	Error    error
}