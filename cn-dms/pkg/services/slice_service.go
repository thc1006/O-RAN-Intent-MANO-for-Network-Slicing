package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/cn-dms/pkg/models"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/errors"
)

// sliceServiceImpl implements the SliceService interface
type sliceServiceImpl struct {
	repo SliceRepository
}

// NewSliceService creates a new slice service instance
func NewSliceService(repo SliceRepository) SliceService {
	return &sliceServiceImpl{
		repo: repo,
	}
}

// CreateSlice creates a new network slice
func (s *sliceServiceImpl) CreateSlice(ctx context.Context, req *models.SliceRequest) (*models.Slice, error) {
	if err := s.validateSliceRequest(req); err != nil {
		return nil, errors.NewValidationError("slice_request", err.Error())
	}

	slice := &models.Slice{
		ID:         fmt.Sprintf("cn-slice-%s", uuid.New().String()[:8]),
		Name:       req.Name,
		Type:       req.Type,
		Status:     models.SliceStatusPending,
		QoSProfile: req.QoSProfile,
		Metadata:   req.Metadata,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, slice); err != nil {
		return nil, fmt.Errorf("failed to create slice: %w", err)
	}

	// Initiate slice activation process
	go s.activateSliceAsync(slice.ID)

	return slice, nil
}

// GetSlice retrieves a slice by ID
func (s *sliceServiceImpl) GetSlice(ctx context.Context, id string) (*models.Slice, error) {
	if id == "" {
		return nil, errors.NewValidationError("id", "slice ID is required")
	}

	slice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get slice %s: %w", id, err)
	}

	return slice, nil
}

// ListSlices retrieves all slices with optional filtering
func (s *sliceServiceImpl) ListSlices(ctx context.Context, filters map[string]interface{}) ([]*models.Slice, error) {
	slices, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list slices: %w", err)
	}

	return slices, nil
}

// UpdateSlice updates an existing slice
func (s *sliceServiceImpl) UpdateSlice(ctx context.Context, id string, updates map[string]interface{}) (*models.Slice, error) {
	slice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get slice for update: %w", err)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		slice.Name = name
	}
	if metadata, ok := updates["metadata"].(map[string]string); ok {
		slice.Metadata = metadata
	}

	slice.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, slice); err != nil {
		return nil, fmt.Errorf("failed to update slice: %w", err)
	}

	return slice, nil
}

// DeleteSlice removes a slice
func (s *sliceServiceImpl) DeleteSlice(ctx context.Context, id string) error {
	slice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get slice for deletion: %w", err)
	}

	// Only allow deletion if slice is deactivated
	if slice.Status != models.SliceStatusDeactivated {
		return errors.NewValidationError("status", "slice must be deactivated before deletion")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete slice: %w", err)
	}

	return nil
}

// ActivateSlice activates a slice
func (s *sliceServiceImpl) ActivateSlice(ctx context.Context, id string) error {
	slice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get slice for activation: %w", err)
	}

	if slice.Status == models.SliceStatusActive {
		return nil // Already active
	}

	slice.Status = models.SliceStatusActivating
	slice.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, slice); err != nil {
		return fmt.Errorf("failed to update slice status: %w", err)
	}

	// Perform activation logic asynchronously
	go s.activateSliceAsync(id)

	return nil
}

// DeactivateSlice deactivates a slice
func (s *sliceServiceImpl) DeactivateSlice(ctx context.Context, id string) error {
	slice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get slice for deactivation: %w", err)
	}

	slice.Status = models.SliceStatusDeactivated
	slice.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, slice); err != nil {
		return fmt.Errorf("failed to deactivate slice: %w", err)
	}

	return nil
}

// validateSliceRequest validates the slice creation request
func (s *sliceServiceImpl) validateSliceRequest(req *models.SliceRequest) error {
	if req.Name == "" {
		return fmt.Errorf("slice name is required")
	}

	if req.Type == "" {
		return fmt.Errorf("slice type is required")
	}

	if req.QoSProfile.MaxLatencyMs <= 0 {
		return fmt.Errorf("QoS profile must specify maximum latency")
	}

	if req.QoSProfile.MinThroughputMbps <= 0 {
		return fmt.Errorf("QoS profile must specify minimum throughput")
	}

	return nil
}

// activateSliceAsync performs slice activation in the background
func (s *sliceServiceImpl) activateSliceAsync(sliceID string) {
	ctx := context.Background()

	// Simulate activation process
	time.Sleep(2 * time.Second)

	slice, err := s.repo.GetByID(ctx, sliceID)
	if err != nil {
		return
	}

	slice.Status = models.SliceStatusActive
	slice.UpdatedAt = time.Now()

	s.repo.Update(ctx, slice)
}