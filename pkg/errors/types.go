// Package errors provides enhanced error handling for O-RAN Intent MANO
package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// ErrorCode represents specific error classifications
type ErrorCode string

const (
	// Validation errors
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"
	ErrCodeRequired   ErrorCode = "REQUIRED_FIELD"
	ErrCodeInvalid    ErrorCode = "INVALID_VALUE"

	// Resource errors
	ErrCodeNotFound   ErrorCode = "RESOURCE_NOT_FOUND"
	ErrCodeExists     ErrorCode = "RESOURCE_EXISTS"
	ErrCodeConflict   ErrorCode = "RESOURCE_CONFLICT"

	// Service errors
	ErrCodeService    ErrorCode = "SERVICE_ERROR"
	ErrCodeTimeout    ErrorCode = "SERVICE_TIMEOUT"
	ErrCodeUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	// Infrastructure errors
	ErrCodeDatabase   ErrorCode = "DATABASE_ERROR"
	ErrCodeNetwork    ErrorCode = "NETWORK_ERROR"
	ErrCodePlacement  ErrorCode = "PLACEMENT_ERROR"

	// Security errors
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeRateLimit    ErrorCode = "RATE_LIMITED"
)

// ErrorSeverity indicates the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// BaseError provides the foundation for all application errors
type BaseError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Severity   ErrorSeverity          `json:"severity"`
	Timestamp  time.Time              `json:"timestamp"`
	StackTrace []string               `json:"stack_trace,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
}

// Error implements the error interface
func (e *BaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *BaseError) Unwrap() error {
	return e.Cause
}

// MarshalJSON customizes JSON serialization
func (e *BaseError) MarshalJSON() ([]byte, error) {
	type Alias BaseError
	return json.Marshal(&struct {
		*Alias
		Cause string `json:"cause,omitempty"`
	}{
		Alias: (*Alias)(e),
		Cause: e.getCauseString(),
	})
}

func (e *BaseError) getCauseString() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return ""
}

// ValidationError represents input validation errors
type ValidationError struct {
	*BaseError
	Field string `json:"field"`
	Value interface{} `json:"value,omitempty"`
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		BaseError: &BaseError{
			Code:      ErrCodeValidation,
			Message:   message,
			Severity:  SeverityMedium,
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		},
		Field: field,
	}
}

// NotFoundError represents resource not found errors
type NotFoundError struct {
	*BaseError
	Resource string `json:"resource"`
	ID       string `json:"id"`
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{
		BaseError: &BaseError{
			Code:      ErrCodeNotFound,
			Message:   fmt.Sprintf("%s with ID %s not found", resource, id),
			Severity:  SeverityMedium,
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		},
		Resource: resource,
		ID:       id,
	}
}

// ConflictError represents resource conflict errors
type ConflictError struct {
	*BaseError
	Resource string `json:"resource"`
	Reason   string `json:"reason"`
}

// NewConflictError creates a new conflict error
func NewConflictError(resource, reason string) *ConflictError {
	return &ConflictError{
		BaseError: &BaseError{
			Code:      ErrCodeConflict,
			Message:   fmt.Sprintf("Conflict with %s: %s", resource, reason),
			Severity:  SeverityMedium,
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		},
		Resource: resource,
		Reason:   reason,
	}
}

// ServiceError represents service-level errors
type ServiceError struct {
	*BaseError
	Service   string `json:"service"`
	Operation string `json:"operation"`
}

// NewServiceError creates a new service error
func NewServiceError(service, operation, message string) *ServiceError {
	return &ServiceError{
		BaseError: &BaseError{
			Code:      ErrCodeService,
			Message:   message,
			Severity:  SeverityHigh,
			Timestamp: time.Now(),
			Details:   make(map[string]interface{}),
		},
		Service:   service,
		Operation: operation,
	}
}

// InternalError represents internal system errors
type InternalError struct {
	*BaseError
	Component string `json:"component"`
}

// NewInternalError creates a new internal error with stack trace
func NewInternalError(component, message string, cause error) *InternalError {
	err := &InternalError{
		BaseError: &BaseError{
			Code:       ErrCodeService,
			Message:    message,
			Cause:      cause,
			Severity:   SeverityCritical,
			Timestamp:  time.Now(),
			Details:    make(map[string]interface{}),
			StackTrace: captureStackTrace(),
		},
		Component: component,
	}
	return err
}

// Helper functions

// captureStackTrace captures the current stack trace
func captureStackTrace() []string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var trace []string
	for {
		frame, more := frames.Next()
		trace = append(trace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return trace
}

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	if baseErr, ok := err.(*BaseError); ok {
		return &BaseError{
			Code:      baseErr.Code,
			Message:   fmt.Sprintf("%s: %s", message, baseErr.Message),
			Cause:     baseErr.Cause,
			Severity:  baseErr.Severity,
			Timestamp: time.Now(),
			Details:   baseErr.Details,
		}
	}

	return &BaseError{
		Code:      ErrCodeService,
		Message:   message,
		Cause:     err,
		Severity:  SeverityMedium,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// Is checks if an error is of a specific type
func Is(err error, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	if baseErr, ok := err.(*BaseError); ok {
		if targetBase, ok := target.(*BaseError); ok {
			return baseErr.Code == targetBase.Code
		}
	}

	return err == target
}

// Type checking helper functions

// IsValidation checks if error is a validation error
func IsValidation(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ValidationError)
	return ok || hasCode(err, ErrCodeValidation)
}

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*NotFoundError)
	return ok || hasCode(err, ErrCodeNotFound)
}

// IsConflict checks if error is a conflict error
func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ConflictError)
	return ok || hasCode(err, ErrCodeConflict)
}

// IsService checks if error is a service error
func IsService(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ServiceError)
	return ok || hasCode(err, ErrCodeService)
}

// IsInternal checks if error is an internal error
func IsInternal(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*InternalError)
	return ok
}

// hasCode checks if an error has a specific error code
func hasCode(err error, code ErrorCode) bool {
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr.Code == code
	}
	return false
}

// GetCode extracts the error code from an error
func GetCode(err error) ErrorCode {
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr.Code
	}
	return ErrCodeService
}

// GetSeverity extracts the severity from an error
func GetSeverity(err error) ErrorSeverity {
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr.Severity
	}
	return SeverityMedium
}