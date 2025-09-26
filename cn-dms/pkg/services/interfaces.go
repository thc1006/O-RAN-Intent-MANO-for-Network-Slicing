// Package services defines the service interfaces for CN-DMS
package services

import (
	"context"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/cn-dms/pkg/models"
)

// SliceService defines the interface for slice management operations
type SliceService interface {
	// CreateSlice creates a new network slice
	CreateSlice(ctx context.Context, req *models.SliceRequest) (*models.Slice, error)

	// GetSlice retrieves a slice by ID
	GetSlice(ctx context.Context, id string) (*models.Slice, error)

	// ListSlices retrieves all slices with optional filtering
	ListSlices(ctx context.Context, filters map[string]interface{}) ([]*models.Slice, error)

	// UpdateSlice updates an existing slice
	UpdateSlice(ctx context.Context, id string, updates map[string]interface{}) (*models.Slice, error)

	// DeleteSlice removes a slice
	DeleteSlice(ctx context.Context, id string) error

	// ActivateSlice activates a slice
	ActivateSlice(ctx context.Context, id string) error

	// DeactivateSlice deactivates a slice
	DeactivateSlice(ctx context.Context, id string) error
}

// NetworkFunctionService defines the interface for NF management operations
type NetworkFunctionService interface {
	// DeployNF deploys a new network function
	DeployNF(ctx context.Context, req *models.NFDeploymentRequest) (*models.NetworkFunction, error)

	// GetNF retrieves a network function by ID
	GetNF(ctx context.Context, id string) (*models.NetworkFunction, error)

	// ListNFs retrieves all network functions with optional filtering
	ListNFs(ctx context.Context, filters map[string]interface{}) ([]*models.NetworkFunction, error)

	// UpdateNF updates an existing network function
	UpdateNF(ctx context.Context, id string, updates map[string]interface{}) (*models.NetworkFunction, error)

	// UndeployNF removes a network function
	UndeployNF(ctx context.Context, id string) error

	// ScaleNF scales a network function
	ScaleNF(ctx context.Context, id string, replicas int) error
}

// HealthService defines the interface for health monitoring
type HealthService interface {
	// GetHealth retrieves overall system health
	GetHealth(ctx context.Context) (*models.HealthStatus, error)

	// CheckReadiness performs readiness checks
	CheckReadiness(ctx context.Context) (bool, error)

	// GetMetrics retrieves system metrics
	GetMetrics(ctx context.Context) (map[string]float64, error)
}

// StatusService defines the interface for status reporting
type StatusService interface {
	// GetCNStatus retrieves CN domain status
	GetCNStatus(ctx context.Context) (*models.CNStatus, error)

	// GetCapabilities retrieves CN domain capabilities
	GetCapabilities(ctx context.Context) (*models.CNCapabilities, error)
}

// Repository interfaces for data persistence

// SliceRepository defines data persistence for slices
type SliceRepository interface {
	Create(ctx context.Context, slice *models.Slice) error
	GetByID(ctx context.Context, id string) (*models.Slice, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*models.Slice, error)
	Update(ctx context.Context, slice *models.Slice) error
	Delete(ctx context.Context, id string) error
}

// NetworkFunctionRepository defines data persistence for NFs
type NetworkFunctionRepository interface {
	Create(ctx context.Context, nf *models.NetworkFunction) error
	GetByID(ctx context.Context, id string) (*models.NetworkFunction, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*models.NetworkFunction, error)
	Update(ctx context.Context, nf *models.NetworkFunction) error
	Delete(ctx context.Context, id string) error
	ListBySliceID(ctx context.Context, sliceID string) ([]*models.NetworkFunction, error)
}

// MetricsRepository defines data persistence for metrics
type MetricsRepository interface {
	Store(ctx context.Context, metrics map[string]float64) error
	GetLatest(ctx context.Context) (map[string]float64, error)
	GetHistorical(ctx context.Context, metric string, duration string) ([]float64, error)
}