package errors

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"strings"
	"time"
)

// ErrorCode represents standard error codes for the Nephio generator
type ErrorCode string

const (
	// Package generation errors
	ErrorCodePackageGeneration    ErrorCode = "PACKAGE_GENERATION_FAILED"
	ErrorCodeTemplateNotFound     ErrorCode = "TEMPLATE_NOT_FOUND"
	ErrorCodeTemplateValidation   ErrorCode = "TEMPLATE_VALIDATION_FAILED"
	ErrorCodeResourceGeneration   ErrorCode = "RESOURCE_GENERATION_FAILED"
	ErrorCodeKptfileGeneration    ErrorCode = "KPTFILE_GENERATION_FAILED"

	// Porch repository errors
	ErrorCodeRepositoryCreation   ErrorCode = "REPOSITORY_CREATION_FAILED"
	ErrorCodeRepositoryAccess     ErrorCode = "REPOSITORY_ACCESS_FAILED"
	ErrorCodePackageRevision      ErrorCode = "PACKAGE_REVISION_FAILED"
	ErrorCodePackageRendering     ErrorCode = "PACKAGE_RENDERING_FAILED"
	ErrorCodeFunctionExecution    ErrorCode = "FUNCTION_EXECUTION_FAILED"

	// ConfigSync errors
	ErrorCodeConfigSyncCreation   ErrorCode = "CONFIGSYNC_CREATION_FAILED"
	ErrorCodeConfigSyncSync       ErrorCode = "CONFIGSYNC_SYNC_FAILED"
	ErrorCodeRootSyncFailure      ErrorCode = "ROOTSYNC_FAILURE"
	ErrorCodeRepoSyncFailure      ErrorCode = "REPOSYNC_FAILURE"

	// Validation errors
	ErrorCodeValidationFailed     ErrorCode = "VALIDATION_FAILED"
	ErrorCodeDeploymentValidation ErrorCode = "DEPLOYMENT_VALIDATION_FAILED"
	ErrorCodeQoSValidation        ErrorCode = "QOS_VALIDATION_FAILED"
	ErrorCodeSecurityValidation   ErrorCode = "SECURITY_VALIDATION_FAILED"
	ErrorCodeResourceValidation   ErrorCode = "RESOURCE_VALIDATION_FAILED"

	// Workload API errors
	ErrorCodeWorkloadCreation     ErrorCode = "WORKLOAD_CREATION_FAILED"
	ErrorCodeWorkloadUpdate       ErrorCode = "WORKLOAD_UPDATE_FAILED"
	ErrorCodeWorkloadValidation   ErrorCode = "WORKLOAD_VALIDATION_FAILED"
	ErrorCodeClusterNotFound      ErrorCode = "CLUSTER_NOT_FOUND"
	ErrorCodePlacementFailure     ErrorCode = "PLACEMENT_FAILURE"

	// Infrastructure errors
	ErrorCodeKubernetesAPI        ErrorCode = "KUBERNETES_API_ERROR"
	ErrorCodeNetworkConnectivity  ErrorCode = "NETWORK_CONNECTIVITY_ERROR"
	ErrorCodeResourceExhaustion   ErrorCode = "RESOURCE_EXHAUSTION"
	ErrorCodeTimeout              ErrorCode = "OPERATION_TIMEOUT"
	ErrorCodePermissionDenied     ErrorCode = "PERMISSION_DENIED"

	// Configuration errors
	ErrorCodeInvalidConfiguration ErrorCode = "INVALID_CONFIGURATION"
	ErrorCodeMissingRequirement   ErrorCode = "MISSING_REQUIREMENT"
	ErrorCodeVersionMismatch      ErrorCode = "VERSION_MISMATCH"
	ErrorCodeDependencyFailure    ErrorCode = "DEPENDENCY_FAILURE"

	// Generic errors
	ErrorCodeInternal             ErrorCode = "INTERNAL_ERROR"
	ErrorCodeUnknown              ErrorCode = "UNKNOWN_ERROR"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityCritical ErrorSeverity = "critical"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityInfo     ErrorSeverity = "info"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategoryValidation    ErrorCategory = "validation"
	ErrorCategoryConfiguration ErrorCategory = "configuration"
	ErrorCategoryInfrastructure ErrorCategory = "infrastructure"
	ErrorCategoryNetworking    ErrorCategory = "networking"
	ErrorCategorySecurity      ErrorCategory = "security"
	ErrorCategoryGeneration    ErrorCategory = "generation"
	ErrorCategoryDeployment    ErrorCategory = "deployment"
	ErrorCategoryIntegration   ErrorCategory = "integration"
	ErrorCategoryInternal      ErrorCategory = "internal"
)

// NephioError represents a comprehensive error structure for the Nephio generator
type NephioError struct {
	// Error identification
	Code        ErrorCode     `json:"code"`
	Message     string        `json:"message"`
	Severity    ErrorSeverity `json:"severity"`
	Category    ErrorCategory `json:"category"`

	// Context information
	Component   string        `json:"component"`
	Operation   string        `json:"operation"`
	Resource    *ResourceRef  `json:"resource,omitempty"`
	Context     ErrorContext  `json:"context"`

	// Error details
	Details     ErrorDetails  `json:"details"`
	Cause       error         `json:"cause,omitempty"`
	StackTrace  []StackFrame  `json:"stackTrace,omitempty"`

	// Metadata
	Timestamp   time.Time     `json:"timestamp"`
	RequestID   string        `json:"requestId,omitempty"`
	UserID      string        `json:"userId,omitempty"`
	SessionID   string        `json:"sessionId,omitempty"`

	// Recovery information
	Retryable   bool          `json:"retryable"`
	RetryAfter  *time.Duration `json:"retryAfter,omitempty"`
	Recovery    RecoveryInfo  `json:"recovery,omitempty"`
}

// ResourceRef represents a reference to a Kubernetes resource
type ResourceRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	UID        string `json:"uid,omitempty"`
}

// ErrorContext provides contextual information about the error
type ErrorContext struct {
	VNFType     string            `json:"vnfType,omitempty"`
	CloudType   string            `json:"cloudType,omitempty"`
	ClusterName string            `json:"clusterName,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Version     string            `json:"version,omitempty"`
}

// ErrorDetails provides detailed error information
type ErrorDetails struct {
	// Technical details
	InternalCode    string                 `json:"internalCode,omitempty"`
	HTTPStatusCode  int                    `json:"httpStatusCode,omitempty"`
	ValidationErrors []ValidationError     `json:"validationErrors,omitempty"`

	// Diagnostic information
	Metrics         map[string]interface{} `json:"metrics,omitempty"`
	State           map[string]interface{} `json:"state,omitempty"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`

	// Related information
	RelatedErrors   []string              `json:"relatedErrors,omitempty"`
	CorrelationID   string                `json:"correlationId,omitempty"`
	TraceID         string                `json:"traceId,omitempty"`
	SpanID          string                `json:"spanId,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// StackFrame represents a stack trace frame
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package"`
}

// RecoveryInfo provides information about error recovery
type RecoveryInfo struct {
	Suggestions    []string          `json:"suggestions,omitempty"`
	Documentation  []string          `json:"documentation,omitempty"`
	SupportContact string            `json:"supportContact,omitempty"`
	AutoRecovery   bool              `json:"autoRecovery"`
	RecoverySteps  []RecoveryStep    `json:"recoverySteps,omitempty"`
}

// RecoveryStep represents a recovery step
type RecoveryStep struct {
	Step        int    `json:"step"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Automated   bool   `json:"automated"`
}

// Error implements the error interface
func (e *NephioError) Error() string {
	if e.Resource != nil {
		return fmt.Sprintf("[%s] %s (%s/%s in %s): %s",
			e.Code, e.Message, e.Resource.Kind, e.Resource.Name, e.Resource.Namespace, e.Component)
	}
	return fmt.Sprintf("[%s] %s (%s): %s", e.Code, e.Message, e.Component, e.Operation)
}

// Unwrap returns the underlying cause error
func (e *NephioError) Unwrap() error {
	return e.Cause
}

// Is implements error equality check
func (e *NephioError) Is(target error) bool {
	if nephioErr, ok := target.(*NephioError); ok {
		return e.Code == nephioErr.Code
	}
	return false
}

// JSON returns the error as a JSON string
func (e *NephioError) JSON() string {
	data, _ := json.MarshalIndent(e, "", "  ")
	return string(data)
}

// ErrorBuilder provides a fluent interface for building NephioErrors
type ErrorBuilder struct {
	error *NephioError
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder(code ErrorCode, message string) *ErrorBuilder {
	return &ErrorBuilder{
		error: &NephioError{
			Code:       code,
			Message:    message,
			Timestamp:  time.Now(),
			Severity:   ErrorSeverityMedium,
			Category:   ErrorCategoryInternal,
			Retryable:  false,
			Context:    ErrorContext{},
			Details:    ErrorDetails{},
			Recovery:   RecoveryInfo{},
		},
	}
}

// WithSeverity sets the error severity
func (eb *ErrorBuilder) WithSeverity(severity ErrorSeverity) *ErrorBuilder {
	eb.error.Severity = severity
	return eb
}

// WithCategory sets the error category
func (eb *ErrorBuilder) WithCategory(category ErrorCategory) *ErrorBuilder {
	eb.error.Category = category
	return eb
}

// WithComponent sets the component that generated the error
func (eb *ErrorBuilder) WithComponent(component string) *ErrorBuilder {
	eb.error.Component = component
	return eb
}

// WithOperation sets the operation that failed
func (eb *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	eb.error.Operation = operation
	return eb
}

// WithResource sets the resource reference
func (eb *ErrorBuilder) WithResource(resource *ResourceRef) *ErrorBuilder {
	eb.error.Resource = resource
	return eb
}

// WithCause sets the underlying cause error
func (eb *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	eb.error.Cause = cause
	return eb
}

// WithContext sets the error context
func (eb *ErrorBuilder) WithContext(context ErrorContext) *ErrorBuilder {
	eb.error.Context = context
	return eb
}

// WithVNFType sets the VNF type in context
func (eb *ErrorBuilder) WithVNFType(vnfType string) *ErrorBuilder {
	eb.error.Context.VNFType = vnfType
	return eb
}

// WithCluster sets the cluster information in context
func (eb *ErrorBuilder) WithCluster(clusterName string) *ErrorBuilder {
	eb.error.Context.ClusterName = clusterName
	return eb
}

// WithNamespace sets the namespace in context
func (eb *ErrorBuilder) WithNamespace(namespace string) *ErrorBuilder {
	eb.error.Context.Namespace = namespace
	return eb
}

// WithDetails sets the error details
func (eb *ErrorBuilder) WithDetails(details ErrorDetails) *ErrorBuilder {
	eb.error.Details = details
	return eb
}

// WithValidationErrors sets validation errors
func (eb *ErrorBuilder) WithValidationErrors(validationErrors []ValidationError) *ErrorBuilder {
	eb.error.Details.ValidationErrors = validationErrors
	return eb
}

// WithHTTPStatus sets the HTTP status code
func (eb *ErrorBuilder) WithHTTPStatus(statusCode int) *ErrorBuilder {
	eb.error.Details.HTTPStatusCode = statusCode
	return eb
}

// WithMetrics adds metrics to the error details
func (eb *ErrorBuilder) WithMetrics(metrics map[string]interface{}) *ErrorBuilder {
	if eb.error.Details.Metrics == nil {
		eb.error.Details.Metrics = make(map[string]interface{})
	}
	for k, v := range metrics {
		eb.error.Details.Metrics[k] = v
	}
	return eb
}

// WithRequestID sets the request ID
func (eb *ErrorBuilder) WithRequestID(requestID string) *ErrorBuilder {
	eb.error.RequestID = requestID
	return eb
}

// WithUserID sets the user ID
func (eb *ErrorBuilder) WithUserID(userID string) *ErrorBuilder {
	eb.error.UserID = userID
	return eb
}

// WithSessionID sets the session ID
func (eb *ErrorBuilder) WithSessionID(sessionID string) *ErrorBuilder {
	eb.error.SessionID = sessionID
	return eb
}

// WithRetryable marks the error as retryable
func (eb *ErrorBuilder) WithRetryable(retryable bool) *ErrorBuilder {
	eb.error.Retryable = retryable
	return eb
}

// WithRetryAfter sets the retry delay
func (eb *ErrorBuilder) WithRetryAfter(duration time.Duration) *ErrorBuilder {
	eb.error.RetryAfter = &duration
	eb.error.Retryable = true
	return eb
}

// WithSuggestions adds recovery suggestions
func (eb *ErrorBuilder) WithSuggestions(suggestions []string) *ErrorBuilder {
	eb.error.Recovery.Suggestions = suggestions
	return eb
}

// WithRecoverySteps adds recovery steps
func (eb *ErrorBuilder) WithRecoverySteps(steps []RecoveryStep) *ErrorBuilder {
	eb.error.Recovery.RecoverySteps = steps
	return eb
}

// WithStackTrace captures the current stack trace
func (eb *ErrorBuilder) WithStackTrace() *ErrorBuilder {
	eb.error.StackTrace = captureStackTrace()
	return eb
}

// Build creates the final NephioError
func (eb *ErrorBuilder) Build() *NephioError {
	return eb.error
}

// Predefined error creation functions

// NewPackageGenerationError creates a package generation error
func NewPackageGenerationError(message string, cause error) *NephioError {
	return NewErrorBuilder(ErrorCodePackageGeneration, message).
		WithSeverity(ErrorSeverityHigh).
		WithCategory(ErrorCategoryGeneration).
		WithComponent("package-generator").
		WithCause(cause).
		WithRetryable(true).
		WithSuggestions([]string{
			"Check template configuration",
			"Verify VNF specification",
			"Ensure required dependencies are available",
		}).
		WithStackTrace().
		Build()
}

// NewTemplateNotFoundError creates a template not found error
func NewTemplateNotFoundError(templateName, vnfType string) *NephioError {
	return NewErrorBuilder(ErrorCodeTemplateNotFound,
		fmt.Sprintf("Template '%s' not found for VNF type '%s'", templateName, vnfType)).
		WithSeverity(ErrorSeverityHigh).
		WithCategory(ErrorCategoryConfiguration).
		WithComponent("template-registry").
		WithVNFType(vnfType).
		WithSuggestions([]string{
			"Check available templates with 'list-templates' command",
			"Verify VNF type is supported",
			"Create custom template if needed",
		}).
		Build()
}

// NewRepositoryAccessError creates a repository access error
func NewRepositoryAccessError(repoName string, cause error) *NephioError {
	return NewErrorBuilder(ErrorCodeRepositoryAccess,
		fmt.Sprintf("Failed to access repository '%s'", repoName)).
		WithSeverity(ErrorSeverityHigh).
		WithCategory(ErrorCategoryInfrastructure).
		WithComponent("porch-client").
		WithCause(cause).
		WithRetryable(true).
		WithRetryAfter(30 * time.Second).
		WithSuggestions([]string{
			"Check repository URL and credentials",
			"Verify network connectivity",
			"Ensure Porch server is accessible",
		}).
		Build()
}

// NewValidationError creates a validation error
func NewValidationError(message string, validationErrors []ValidationError) *NephioError {
	return NewErrorBuilder(ErrorCodeValidationFailed, message).
		WithSeverity(ErrorSeverityMedium).
		WithCategory(ErrorCategoryValidation).
		WithComponent("validator").
		WithValidationErrors(validationErrors).
		WithSuggestions([]string{
			"Review validation error details",
			"Fix configuration issues",
			"Consult documentation for requirements",
		}).
		Build()
}

// NewConfigSyncError creates a ConfigSync error
func NewConfigSyncError(operation string, cause error) *NephioError {
	return NewErrorBuilder(ErrorCodeConfigSyncSync,
		fmt.Sprintf("ConfigSync %s failed", operation)).
		WithSeverity(ErrorSeverityHigh).
		WithCategory(ErrorCategoryDeployment).
		WithComponent("configsync-manager").
		WithOperation(operation).
		WithCause(cause).
		WithRetryable(true).
		WithSuggestions([]string{
			"Check ConfigSync status",
			"Verify Git repository access",
			"Review sync errors in ConfigSync logs",
		}).
		Build()
}

// NewWorkloadError creates a workload-related error
func NewWorkloadError(code ErrorCode, message string, resource *ResourceRef, cause error) *NephioError {
	return NewErrorBuilder(code, message).
		WithSeverity(ErrorSeverityHigh).
		WithCategory(ErrorCategoryDeployment).
		WithComponent("workload-controller").
		WithResource(resource).
		WithCause(cause).
		WithRetryable(true).
		WithSuggestions([]string{
			"Check workload specification",
			"Verify cluster resources",
			"Review placement constraints",
		}).
		Build()
}

// NewKubernetesAPIError creates a Kubernetes API error
func NewKubernetesAPIError(operation string, cause error) *NephioError {
	severity := ErrorSeverityHigh
	retryable := true

	// Determine severity and retryability based on the error
	if cause != nil {
		errMsg := strings.ToLower(cause.Error())
		if strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "unauthorized") {
			severity = ErrorSeverityCritical
			retryable = false
		} else if strings.Contains(errMsg, "not found") {
			severity = ErrorSeverityMedium
			retryable = false
		}
	}

	return NewErrorBuilder(ErrorCodeKubernetesAPI,
		fmt.Sprintf("Kubernetes API operation '%s' failed", operation)).
		WithSeverity(severity).
		WithCategory(ErrorCategoryInfrastructure).
		WithComponent("kubernetes-client").
		WithOperation(operation).
		WithCause(cause).
		WithRetryable(retryable).
		WithSuggestions([]string{
			"Check Kubernetes cluster connectivity",
			"Verify authentication credentials",
			"Ensure required permissions",
		}).
		Build()
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string, timeout time.Duration) *NephioError {
	return NewErrorBuilder(ErrorCodeTimeout,
		fmt.Sprintf("Operation '%s' timed out after %v", operation, timeout)).
		WithSeverity(ErrorSeverityMedium).
		WithCategory(ErrorCategoryInfrastructure).
		WithOperation(operation).
		WithRetryable(true).
		WithRetryAfter(timeout).
		WithSuggestions([]string{
			"Increase timeout value if appropriate",
			"Check system performance",
			"Review resource constraints",
		}).
		Build()
}

// Error aggregation and handling

// ErrorCollector collects multiple errors
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// FirstError returns the first error or nil
func (ec *ErrorCollector) FirstError() error {
	if len(ec.errors) > 0 {
		return ec.errors[0]
	}
	return nil
}

// Combine combines all errors into a single error
func (ec *ErrorCollector) Combine() error {
	if len(ec.errors) == 0 {
		return nil
	}

	if len(ec.errors) == 1 {
		return ec.errors[0]
	}

	messages := make([]string, len(ec.errors))
	for i, err := range ec.errors {
		messages[i] = err.Error()
	}

	return NewErrorBuilder(ErrorCodeInternal,
		fmt.Sprintf("Multiple errors occurred: %s", strings.Join(messages, "; "))).
		WithSeverity(ErrorSeverityHigh).
		WithCategory(ErrorCategoryInternal).
		WithDetails(ErrorDetails{
			RelatedErrors: messages,
		}).
		Build()
}

// Error recovery and retry logic

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries    int           `json:"maxRetries"`
	InitialDelay  time.Duration `json:"initialDelay"`
	MaxDelay      time.Duration `json:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor"`
	Jitter        bool          `json:"jitter"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func() error

// Retry executes a function with retry logic
func Retry(ctx context.Context, config *RetryConfig, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateDelay(config, attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if nephioErr, ok := err.(*NephioError); ok {
			if !nephioErr.Retryable {
				return err
			}
		}
	}

	return lastErr
}

// calculateDelay calculates the delay for a retry attempt
func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	delay := time.Duration(float64(config.InitialDelay) *
		math.Pow(config.BackoffFactor, float64(attempt-1)))

	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	if config.Jitter {
		// Add up to 25% jitter using crypto/rand
		maxJitter := int64(float64(delay) * 0.25)
		if maxJitter > 0 {
			n, err := rand.Int(rand.Reader, big.NewInt(maxJitter))
			if err == nil {
				jitter := time.Duration(n.Int64())
				delay += jitter
			}
			// If crypto/rand fails, continue without jitter for security
		}
	}

	return delay
}

// Utility functions

// captureStackTrace captures the current stack trace
func captureStackTrace() []StackFrame {
	var frames []StackFrame

	// Skip the first few frames (captureStackTrace, WithStackTrace, etc.)
	skip := 3
	for {
		pc, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			break
		}

		funcName := fn.Name()
		packageName := ""
		if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
			packageName = funcName[:lastSlash]
			funcName = funcName[lastSlash+1:]
		}

		frames = append(frames, StackFrame{
			Function: funcName,
			File:     file,
			Line:     line,
			Package:  packageName,
		})

		skip++
		if len(frames) >= 10 { // Limit stack trace depth
			break
		}
	}

	return frames
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if nephioErr, ok := err.(*NephioError); ok {
		return nephioErr.Retryable
	}
	return false
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if nephioErr, ok := err.(*NephioError); ok {
		return nephioErr.Code
	}
	return ErrorCodeUnknown
}

// GetErrorSeverity extracts the error severity from an error
func GetErrorSeverity(err error) ErrorSeverity {
	if nephioErr, ok := err.(*NephioError); ok {
		return nephioErr.Severity
	}
	return ErrorSeverityMedium
}

