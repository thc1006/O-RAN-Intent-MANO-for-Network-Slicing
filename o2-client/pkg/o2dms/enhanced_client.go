package o2dms

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/models"
)

// Enhanced O2 DMS Client with retry logic, event notifications, and advanced features

// EnhancedClient provides enhanced O2DMS operations with retry and event handling
type EnhancedClient struct {
	*Client
	retryConfig RetryConfig
	eventChan   chan Event
	subscribers map[string][]EventHandler
	mutex       sync.RWMutex
	metrics     *ClientMetrics
	state       *ClientState
}

// NewEnhancedClient creates a new enhanced O2DMS client
func NewEnhancedClient(baseURL string, options ...ClientOption) *EnhancedClient {
	base := NewClient(baseURL, options...)
	return &EnhancedClient{
		Client:      base,
		retryConfig: DefaultRetryConfig(),
		eventChan:   make(chan Event, 1000),
		subscribers: make(map[string][]EventHandler),
		metrics:     NewClientMetrics(),
		state:       &ClientState{Connection: ConnectionStateDisconnected},
	}
}

// SetRetryConfig configures retry behavior
func (c *EnhancedClient) SetRetryConfig(config RetryConfig) {
	c.retryConfig = config
}

// Enhanced Deployment Manager Operations with Retry

// GetDeploymentManagersWithRetry retrieves deployment managers with retry logic
func (c *EnhancedClient) GetDeploymentManagersWithRetry(ctx context.Context, filter models.DeploymentManagerFilter) (*models.DeploymentManagerCollection, error) {
	params := c.buildDeploymentManagerParams(filter)
	var collection models.DeploymentManagerCollection

	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", "/o2dms/v1/deploymentManagers", params, &collection)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get deployment managers: %w", err)
	}

	c.publishEvent(Event{
		ID:        fmt.Sprintf("dm_list_%d", time.Now().UnixNano()),
		Type:      EventTypeDeploymentManagerListed,
		Source:    "o2dms.client",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"count": len(collection.Items)},
		Severity:  SeverityInfo,
	})

	return &collection, nil
}

// GetDeploymentManagerWithRetry retrieves a specific deployment manager with retry
func (c *EnhancedClient) GetDeploymentManagerWithRetry(ctx context.Context, deploymentManagerID string) (*models.DeploymentManager, error) {
	var manager models.DeploymentManager
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s", deploymentManagerID)

	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", endpoint, nil, &manager)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get deployment manager %s: %w", deploymentManagerID, err)
	}

	return &manager, nil
}

// Enhanced NF Deployment Operations

// CreateNFDeploymentWithRetry creates a new NF deployment with retry logic
func (c *EnhancedClient) CreateNFDeploymentWithRetry(ctx context.Context, deploymentManagerID string, request *DeploymentRequest) (*models.NFDeployment, error) {
	var deployment models.NFDeployment
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments", deploymentManagerID)

	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "POST", endpoint, request, &deployment)
	})

	if err != nil {
		c.publishEvent(Event{
			ID:        fmt.Sprintf("deploy_failed_%d", time.Now().UnixNano()),
			Type:      EventTypeDeploymentFailed,
			Source:    "o2dms.client",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"deployment_manager_id": deploymentManagerID,
				"request":               request,
				"error":                 err.Error(),
			},
			Severity: SeverityError,
		})
		return nil, fmt.Errorf("failed to create NF deployment: %w", err)
	}

	c.publishEvent(Event{
		ID:        deployment.ID,
		Type:      EventTypeDeploymentCreated,
		Source:    "o2dms.client",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"deployment_id":         deployment.ID,
			"deployment_manager_id": deploymentManagerID,
			"status":                deployment.Status,
		},
		Severity: SeverityInfo,
	})

	return &deployment, nil
}

// GetNFDeploymentsWithRetry retrieves NF deployments with retry logic
func (c *EnhancedClient) GetNFDeploymentsWithRetry(ctx context.Context, deploymentManagerID string, filter models.NFDeploymentFilter) (*models.NFDeploymentCollection, error) {
	params := c.buildNFDeploymentParams(filter)
	var collection models.NFDeploymentCollection
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments", deploymentManagerID)

	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", endpoint, params, &collection)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get NF deployments: %w", err)
	}

	return &collection, nil
}

// DeleteNFDeploymentWithRetry deletes an NF deployment with retry logic
func (c *EnhancedClient) DeleteNFDeploymentWithRetry(ctx context.Context, deploymentManagerID, deploymentID string) error {
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s", deploymentManagerID, deploymentID)

	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "DELETE", endpoint, nil, nil)
	})

	if err != nil {
		c.publishEvent(Event{
			ID:        fmt.Sprintf("delete_failed_%d", time.Now().UnixNano()),
			Type:      EventTypeDeploymentDeleteFailed,
			Source:    "o2dms.client",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"deployment_id":         deploymentID,
				"deployment_manager_id": deploymentManagerID,
				"error":                 err.Error(),
			},
			Severity: SeverityError,
		})
		return fmt.Errorf("failed to delete NF deployment %s: %w", deploymentID, err)
	}

	c.publishEvent(Event{
		ID:        deploymentID,
		Type:      EventTypeDeploymentDeleted,
		Source:    "o2dms.client",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"deployment_id":         deploymentID,
			"deployment_manager_id": deploymentManagerID,
		},
		Severity: SeverityInfo,
	})

	return nil
}

// Advanced Deployment Management

// DeployNetworkSlice deploys a complete network slice with multiple NFs
func (c *EnhancedClient) DeployNetworkSlice(ctx context.Context, deploymentManagerID string, sliceSpec *NetworkSliceSpec) (*NetworkSliceDeployment, error) {
	log.Printf("Deploying network slice: %s", sliceSpec.SliceID)

	sliceDeployment := &NetworkSliceDeployment{
		SliceID:              sliceSpec.SliceID,
		Status:               SliceStatusDeploying,
		DeploymentManagerID:  deploymentManagerID,
		NFDeployments:        make([]*models.NFDeployment, 0),
		CreatedAt:            time.Now(),
	}

	// Deploy each NF in the slice
	for _, nfSpec := range sliceSpec.NetworkFunctions {
		deployReq := &DeploymentRequest{
			Name:                     fmt.Sprintf("%s-%s", sliceSpec.SliceID, nfSpec.Type),
			Description:              fmt.Sprintf("NF deployment for slice %s", sliceSpec.SliceID),
			NFDeploymentDescriptorID: nfSpec.DescriptorID,
			InputParams: map[string]interface{}{
				"qosRequirements": sliceSpec.QoSRequirements,
				"placement":       sliceSpec.Placement,
				"sliceId":         sliceSpec.SliceID,
				"nfType":          nfSpec.Type,
			},
			Extensions: map[string]interface{}{
				"oran.io/slice-id":       sliceSpec.SliceID,
				"oran.io/nf-type":        nfSpec.Type,
				"oran.io/slice-type":     sliceSpec.QoSRequirements.SliceType,
				"oran.io/deployment-seq": len(sliceDeployment.NFDeployments),
			},
		}

		deployment, err := c.CreateNFDeploymentWithRetry(ctx, deploymentManagerID, deployReq)
		if err != nil {
			sliceDeployment.Status = SliceStatusFailed
			sliceDeployment.Error = err.Error()
			return sliceDeployment, fmt.Errorf("failed to deploy NF %s for slice %s: %w", nfSpec.Type, sliceSpec.SliceID, err)
		}

		sliceDeployment.NFDeployments = append(sliceDeployment.NFDeployments, deployment)

		// Wait for deployment to be ready if specified
		if sliceSpec.WaitForReady {
			_, err = c.WaitForDeploymentReadyWithRetry(ctx, deploymentManagerID, deployment.ID, sliceSpec.DeploymentTimeout)
			if err != nil {
				sliceDeployment.Status = SliceStatusFailed
				sliceDeployment.Error = err.Error()
				return sliceDeployment, fmt.Errorf("NF deployment %s failed to become ready: %w", deployment.ID, err)
			}
		}
	}

	sliceDeployment.Status = SliceStatusDeployed
	sliceDeployment.UpdatedAt = time.Now()

	c.publishEvent(Event{
		ID:        sliceSpec.SliceID,
		Type:      EventTypeSliceDeployed,
		Source:    "o2dms.client",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"slice_id":              sliceSpec.SliceID,
			"deployment_manager_id": deploymentManagerID,
			"nf_count":              len(sliceDeployment.NFDeployments),
		},
		Severity: SeverityInfo,
	})

	return sliceDeployment, nil
}

// WaitForDeploymentReadyWithRetry waits for a deployment to reach ready state with retry
func (c *EnhancedClient) WaitForDeploymentReadyWithRetry(ctx context.Context, deploymentManagerID, deploymentID string, timeout time.Duration) (*models.NFDeployment, error) {
	deadline := time.Now().Add(timeout)
	var lastDeployment *models.NFDeployment

	for time.Now().Before(deadline) {
		var err error
		lastDeployment, err = c.GetNFDeploymentWithRetry(ctx, deploymentManagerID, deploymentID)
		if err != nil {
			log.Printf("Failed to get deployment status for %s: %v", deploymentID, err)
			// Continue trying
		} else {
			switch lastDeployment.Status {
			case models.NFDeploymentStatusInstantiated:
				c.publishEvent(Event{
					ID:        deploymentID,
					Type:      EventTypeDeploymentReady,
					Source:    "o2dms.client",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"deployment_id":         deploymentID,
						"deployment_manager_id": deploymentManagerID,
					},
					Severity: SeverityInfo,
				})
				return lastDeployment, nil
			case models.NFDeploymentStatusFailed:
				return nil, fmt.Errorf("deployment failed: %s", deploymentID)
			}
		}

		// Wait before next check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue loop
		}
	}

	return lastDeployment, fmt.Errorf("deployment did not become ready within timeout: %s", deploymentID)
}

// GetNFDeploymentWithRetry retrieves a specific NF deployment with retry
func (c *EnhancedClient) GetNFDeploymentWithRetry(ctx context.Context, deploymentManagerID, deploymentID string) (*models.NFDeployment, error) {
	var deployment models.NFDeployment
	endpoint := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s", deploymentManagerID, deploymentID)

	err := c.retryWithBackoff(ctx, func() error {
		return c.doRequest(ctx, "GET", endpoint, nil, &deployment)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get NF deployment %s: %w", deploymentID, err)
	}

	return &deployment, nil
}

// Event Management

// Subscribe adds an event handler for specific event types
func (c *EnhancedClient) Subscribe(eventTypes []EventType, handler EventHandler) string {
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
func (c *EnhancedClient) StartEventProcessing(ctx context.Context) {
	c.state.SetEventsEnabled(true)
	go func() {
		defer c.state.SetEventsEnabled(false)
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
func (c *EnhancedClient) publishEvent(event Event) {
	if !c.state.IsEventsEnabled() {
		return
	}

	select {
	case c.eventChan <- event:
	default:
		log.Printf("Event channel full, dropping event: %s", event.ID)
	}
}

// processEvent processes an event and notifies subscribers
func (c *EnhancedClient) processEvent(event Event) {
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

// Health Monitoring

// StartHealthMonitoring starts periodic health monitoring
func (c *EnhancedClient) StartHealthMonitoring(ctx context.Context, interval time.Duration) {
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
func (c *EnhancedClient) performHealthCheck(ctx context.Context) {
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.GetHealthInfo(healthCtx)
	c.state.UpdateHealthCheck(err)

	if err != nil {
		c.state.SetConnectionState(ConnectionStateError)
		c.publishEvent(Event{
			ID:        fmt.Sprintf("health_%d", time.Now().UnixNano()),
			Type:      EventTypeHealthCheckFailed,
			Source:    "o2dms.client",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"error": err.Error()},
			Severity:  SeverityError,
		})
	} else {
		c.state.SetConnectionState(ConnectionStateConnected)
		c.publishEvent(Event{
			ID:        fmt.Sprintf("health_%d", time.Now().UnixNano()),
			Type:      EventTypeHealthCheckSuccess,
			Source:    "o2dms.client",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{},
			Severity:  SeverityInfo,
		})
	}
}

// Helper methods for building query parameters

func (c *EnhancedClient) buildDeploymentManagerParams(filter models.DeploymentManagerFilter) map[string]string {
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

func (c *EnhancedClient) buildNFDeploymentParams(filter models.NFDeploymentFilter) map[string]string {
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
	if filter.Status != "" {
		if params["filter"] != "" {
			params["filter"] += fmt.Sprintf(" and status eq '%s'", filter.Status)
		} else {
			params["filter"] = fmt.Sprintf("status eq '%s'", filter.Status)
		}
	}
	return params
}

// retryWithBackoff implements exponential backoff retry logic
func (c *EnhancedClient) retryWithBackoff(ctx context.Context, fn func() error) error {
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
func (c *EnhancedClient) calculateBackoffDelay(attempt int) time.Duration {
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
func (c *EnhancedClient) isRetryableError(err error) bool {
	// Check for context cancellation
	if ctx.Err() != nil {
		return false
	}

	// In a real implementation, you would check specific error types
	// For now, retry on most errors
	return true
}

// GetMetrics returns client performance metrics
func (c *EnhancedClient) GetMetrics() (int64, int64, int64, time.Duration) {
	return c.metrics.GetMetrics()
}

// GetConnectionState returns the current connection state
func (c *EnhancedClient) GetConnectionState() ConnectionState {
	return c.state.GetConnectionState()
}