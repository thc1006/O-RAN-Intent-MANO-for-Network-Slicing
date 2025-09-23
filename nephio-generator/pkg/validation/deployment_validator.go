package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentValidator validates Nephio package deployments
type DeploymentValidator struct {
	client           client.Client
	clientset        kubernetes.Interface
	validationConfig *ValidationConfig
}

// ValidationConfig represents validation configuration
type ValidationConfig struct {
	// Timeout for validation operations
	Timeout time.Duration `json:"timeout"`

	// RetryInterval for polling operations
	RetryInterval time.Duration `json:"retryInterval"`

	// EnableNetworkValidation enables network connectivity validation
	EnableNetworkValidation bool `json:"enableNetworkValidation"`

	// EnableResourceValidation enables resource utilization validation
	EnableResourceValidation bool `json:"enableResourceValidation"`

	// EnableQoSValidation enables QoS validation
	EnableQoSValidation bool `json:"enableQoSValidation"`

	// EnableSecurityValidation enables security validation
	EnableSecurityValidation bool `json:"enableSecurityValidation"`

	// QoSThresholds defines QoS validation thresholds
	QoSThresholds QoSThresholds `json:"qosThresholds"`

	// ResourceThresholds defines resource validation thresholds
	ResourceThresholds ResourceThresholds `json:"resourceThresholds"`

	// SecurityPolicies defines security validation policies
	SecurityPolicies SecurityPolicies `json:"securityPolicies"`
}

// QoSThresholds defines QoS validation thresholds
type QoSThresholds struct {
	// MaxLatencyMs maximum allowed latency in milliseconds
	MaxLatencyMs float64 `json:"maxLatencyMs"`

	// MaxJitterMs maximum allowed jitter in milliseconds
	MaxJitterMs float64 `json:"maxJitterMs"`

	// MaxPacketLossPercent maximum allowed packet loss percentage
	MaxPacketLossPercent float64 `json:"maxPacketLossPercent"`

	// MinBandwidthMbps minimum required bandwidth in Mbps
	MinBandwidthMbps float64 `json:"minBandwidthMbps"`

	// MinReliabilityPercent minimum required reliability percentage
	MinReliabilityPercent float64 `json:"minReliabilityPercent"`
}

// ResourceThresholds defines resource validation thresholds
type ResourceThresholds struct {
	// MaxCPUUtilizationPercent maximum allowed CPU utilization
	MaxCPUUtilizationPercent float64 `json:"maxCpuUtilizationPercent"`

	// MaxMemoryUtilizationPercent maximum allowed memory utilization
	MaxMemoryUtilizationPercent float64 `json:"maxMemoryUtilizationPercent"`

	// MaxStorageUtilizationPercent maximum allowed storage utilization
	MaxStorageUtilizationPercent float64 `json:"maxStorageUtilizationPercent"`

	// MinAvailableReplicas minimum required available replicas
	MinAvailableReplicas int32 `json:"minAvailableReplicas"`
}

// SecurityPolicies defines security validation policies
type SecurityPolicies struct {
	// RequireNonRoot requires containers to run as non-root
	RequireNonRoot bool `json:"requireNonRoot"`

	// RequireReadOnlyRootFS requires read-only root filesystem
	RequireReadOnlyRootFS bool `json:"requireReadOnlyRootFS"`

	// RequireResourceLimits requires resource limits
	RequireResourceLimits bool `json:"requireResourceLimits"`

	// RequireNetworkPolicies requires network policies
	RequireNetworkPolicies bool `json:"requireNetworkPolicies"`

	// AllowedCapabilities defines allowed Linux capabilities
	AllowedCapabilities []string `json:"allowedCapabilities"`

	// ForbiddenCapabilities defines forbidden Linux capabilities
	ForbiddenCapabilities []string `json:"forbiddenCapabilities"`
}

// ValidationResult represents the result of deployment validation
type ValidationResult struct {
	// Overall validation status
	Valid bool `json:"valid"`

	// Validation summary
	Summary ValidationSummary `json:"summary"`

	// Individual validation results
	Results []IndividualValidationResult `json:"results"`

	// Errors encountered during validation
	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings generated during validation
	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// Timestamp of validation
	Timestamp time.Time `json:"timestamp"`

	// Duration of validation
	Duration time.Duration `json:"duration"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	// Total number of validations performed
	TotalValidations int `json:"totalValidations"`

	// Number of successful validations
	SuccessfulValidations int `json:"successfulValidations"`

	// Number of failed validations
	FailedValidations int `json:"failedValidations"`

	// Number of warnings
	WarningCount int `json:"warningCount"`

	// Validation score (percentage)
	ValidationScore float64 `json:"validationScore"`
}

// IndividualValidationResult represents the result of an individual validation
type IndividualValidationResult struct {
	// Name of the validation
	Name string `json:"name"`

	// Type of validation
	Type ValidationType `json:"type"`

	// Result of the validation
	Result ValidationResultType `json:"result"`

	// Message describing the result
	Message string `json:"message"`

	// Resource being validated
	Resource ResourceReference `json:"resource"`

	// Details of the validation
	Details map[string]interface{} `json:"details,omitempty"`

	// Duration of this validation
	Duration time.Duration `json:"duration"`
}

// ValidationType represents the type of validation
type ValidationType string

const (
	ValidationTypeDeployment     ValidationType = "deployment"
	ValidationTypeService        ValidationType = "service"
	ValidationTypeNetworking     ValidationType = "networking"
	ValidationTypeResource       ValidationType = "resource"
	ValidationTypeQoS            ValidationType = "qos"
	ValidationTypeSecurity       ValidationType = "security"
	ValidationTypeConfiguration  ValidationType = "configuration"
	ValidationTypeHealthCheck    ValidationType = "healthcheck"
)

// ValidationResultType represents the result type of validation
type ValidationResultType string

const (
	ValidationResultPass    ValidationResultType = "pass"
	ValidationResultFail    ValidationResultType = "fail"
	ValidationResultWarning ValidationResultType = "warning"
	ValidationResultSkipped ValidationResultType = "skipped"
)

// ValidationError represents a validation error
type ValidationError struct {
	// Type of error
	Type string `json:"type"`

	// Error message
	Message string `json:"message"`

	// Resource related to the error
	Resource *ResourceReference `json:"resource,omitempty"`

	// Code of the error
	Code string `json:"code,omitempty"`

	// Severity of the error
	Severity string `json:"severity"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	// Type of warning
	Type string `json:"type"`

	// Warning message
	Message string `json:"message"`

	// Resource related to the warning
	Resource *ResourceReference `json:"resource,omitempty"`

	// Code of the warning
	Code string `json:"code,omitempty"`
}

// ResourceReference represents a reference to a Kubernetes resource
type ResourceReference struct {
	// API version of the resource
	APIVersion string `json:"apiVersion"`

	// Kind of the resource
	Kind string `json:"kind"`

	// Name of the resource
	Name string `json:"name"`

	// Namespace of the resource
	Namespace string `json:"namespace,omitempty"`
}

// DeploymentValidationSpec represents the specification for deployment validation
type DeploymentValidationSpec struct {
	// Namespace to validate
	Namespace string `json:"namespace"`

	// Labels to filter resources
	LabelSelector map[string]string `json:"labelSelector,omitempty"`

	// Annotations to filter resources
	AnnotationSelector map[string]string `json:"annotationSelector,omitempty"`

	// VNF type to validate
	VNFType string `json:"vnfType,omitempty"`

	// Cloud type to validate
	CloudType string `json:"cloudType,omitempty"`

	// Specific validations to run
	Validations []string `json:"validations,omitempty"`

	// Timeout for validation
	Timeout time.Duration `json:"timeout,omitempty"`
}

// NewDeploymentValidator creates a new deployment validator
func NewDeploymentValidator(config *rest.Config, validationConfig *ValidationConfig) (*DeploymentValidator, error) {
	// Create controller-runtime client
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add core/v1 to scheme: %w", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add apps/v1 to scheme: %w", err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add networking/v1 to scheme: %w", err)
	}

	runtimeClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime client: %w", err)
	}

	// Create kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	// Set default validation config if not provided
	if validationConfig == nil {
		validationConfig = DefaultValidationConfig()
	}

	return &DeploymentValidator{
		client:           runtimeClient,
		clientset:        clientset,
		validationConfig: validationConfig,
	}, nil
}

// ValidateDeployment validates a complete deployment
func (dv *DeploymentValidator) ValidateDeployment(ctx context.Context, spec *DeploymentValidationSpec) (*ValidationResult, error) {
	startTime := time.Now()

	result := &ValidationResult{
		Timestamp: startTime,
		Results:   []IndividualValidationResult{},
		Errors:    []ValidationError{},
		Warnings:  []ValidationWarning{},
	}

	// Set timeout context
	timeout := dv.validationConfig.Timeout
	if spec.Timeout > 0 {
		timeout = spec.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Run deployment validations
	if err := dv.validateDeployments(ctx, spec, result); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "deployment-validation",
			Message:  fmt.Sprintf("Deployment validation failed: %v", err),
			Severity: "error",
		})
	}

	// Run service validations
	if err := dv.validateServices(ctx, spec, result); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "service-validation",
			Message:  fmt.Sprintf("Service validation failed: %v", err),
			Severity: "error",
		})
	}

	// Run networking validations
	if dv.validationConfig.EnableNetworkValidation {
		if err := dv.validateNetworking(ctx, spec, result); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "networking-validation",
				Message:  fmt.Sprintf("Networking validation failed: %v", err),
				Severity: "warning",
			})
		}
	}

	// Run resource validations
	if dv.validationConfig.EnableResourceValidation {
		if err := dv.validateResources(ctx, spec, result); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "resource-validation",
				Message:  fmt.Sprintf("Resource validation failed: %v", err),
				Severity: "warning",
			})
		}
	}

	// Run QoS validations
	if dv.validationConfig.EnableQoSValidation {
		if err := dv.validateQoS(ctx, spec, result); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "qos-validation",
				Message:  fmt.Sprintf("QoS validation failed: %v", err),
				Severity: "warning",
			})
		}
	}

	// Run security validations
	if dv.validationConfig.EnableSecurityValidation {
		if err := dv.validateSecurity(ctx, spec, result); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "security-validation",
				Message:  fmt.Sprintf("Security validation failed: %v", err),
				Severity: "error",
			})
		}
	}

	// Calculate results
	result.Duration = time.Since(startTime)
	result.Summary = dv.calculateSummary(result)
	result.Valid = result.Summary.FailedValidations == 0

	return result, nil
}

// validateDeployments validates deployment resources
func (dv *DeploymentValidator) validateDeployments(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// List deployments
	deploymentList := &appsv1.DeploymentList{}
	listOptions := []client.ListOption{
		client.InNamespace(spec.Namespace),
	}

	if len(spec.LabelSelector) > 0 {
		listOptions = append(listOptions, client.MatchingLabels(spec.LabelSelector))
	}

	if err := dv.client.List(ctx, deploymentList, listOptions...); err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deploymentList.Items {
		// Validate deployment readiness
		dv.validateDeploymentReadiness(&deployment, result)

		// Validate deployment configuration
		dv.validateDeploymentConfiguration(&deployment, result)

		// Validate deployment replicas
		dv.validateDeploymentReplicas(&deployment, result)

		// Validate deployment strategy
		dv.validateDeploymentStrategy(&deployment, result)
	}

	return nil
}

// validateServices validates service resources
func (dv *DeploymentValidator) validateServices(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// List services
	serviceList := &corev1.ServiceList{}
	listOptions := []client.ListOption{
		client.InNamespace(spec.Namespace),
	}

	if len(spec.LabelSelector) > 0 {
		listOptions = append(listOptions, client.MatchingLabels(spec.LabelSelector))
	}

	if err := dv.client.List(ctx, serviceList, listOptions...); err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range serviceList.Items {
		// Validate service endpoints
		dv.validateServiceEndpoints(ctx, &service, result)

		// Validate service configuration
		dv.validateServiceConfiguration(&service, result)

		// Validate service ports
		dv.validateServicePorts(&service, result)
	}

	return nil
}

// validateNetworking validates networking resources and connectivity
func (dv *DeploymentValidator) validateNetworking(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// Validate network policies
	if err := dv.validateNetworkPolicies(ctx, spec, result); err != nil {
		return fmt.Errorf("network policy validation failed: %w", err)
	}

	// Validate ingress resources
	if err := dv.validateIngress(ctx, spec, result); err != nil {
		return fmt.Errorf("ingress validation failed: %w", err)
	}

	// Validate pod-to-pod connectivity
	if err := dv.validatePodConnectivity(ctx, spec, result); err != nil {
		return fmt.Errorf("pod connectivity validation failed: %w", err)
	}

	return nil
}

// validateResources validates resource utilization and limits
func (dv *DeploymentValidator) validateResources(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// Get pods
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(spec.Namespace),
	}

	if len(spec.LabelSelector) > 0 {
		listOptions = append(listOptions, client.MatchingLabels(spec.LabelSelector))
	}

	if err := dv.client.List(ctx, podList, listOptions...); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range podList.Items {
		// Validate pod resource requests and limits
		dv.validatePodResources(&pod, result)

		// Validate pod resource utilization
		if err := dv.validatePodResourceUtilization(ctx, &pod, result); err != nil {
			// Log warning but don't fail validation
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "resource-utilization",
				Message: fmt.Sprintf("Failed to validate resource utilization for pod %s: %v", pod.Name, err),
				Resource: &ResourceReference{
					APIVersion: pod.APIVersion,
					Kind:       pod.Kind,
					Name:       pod.Name,
					Namespace:  pod.Namespace,
				},
			})
		}
	}

	return nil
}

// validateQoS validates Quality of Service requirements
func (dv *DeploymentValidator) validateQoS(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// Get pods with QoS annotations
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(spec.Namespace),
	}

	if len(spec.LabelSelector) > 0 {
		listOptions = append(listOptions, client.MatchingLabels(spec.LabelSelector))
	}

	if err := dv.client.List(ctx, podList, listOptions...); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range podList.Items {
		// Check if pod has QoS annotations
		if dv.hasQoSAnnotations(&pod) {
			// Validate QoS requirements
			dv.validatePodQoS(&pod, result)

			// Validate QoS metrics (if available)
			if err := dv.validateQoSMetrics(ctx, &pod, result); err != nil {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Type:    "qos-metrics",
					Message: fmt.Sprintf("Failed to validate QoS metrics for pod %s: %v", pod.Name, err),
					Resource: &ResourceReference{
						APIVersion: pod.APIVersion,
						Kind:       pod.Kind,
						Name:       pod.Name,
						Namespace:  pod.Namespace,
					},
				})
			}
		}
	}

	return nil
}

// validateSecurity validates security policies and configurations
func (dv *DeploymentValidator) validateSecurity(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// Get pods
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(spec.Namespace),
	}

	if len(spec.LabelSelector) > 0 {
		listOptions = append(listOptions, client.MatchingLabels(spec.LabelSelector))
	}

	if err := dv.client.List(ctx, podList, listOptions...); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range podList.Items {
		// Validate pod security context
		dv.validatePodSecurityContext(&pod, result)

		// Validate container security context
		dv.validateContainerSecurityContext(&pod, result)

		// Validate security policies
		if err := dv.validatePodSecurityPolicies(ctx, &pod, result); err != nil {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "security-policy",
				Message: fmt.Sprintf("Failed to validate security policies for pod %s: %v", pod.Name, err),
				Resource: &ResourceReference{
					APIVersion: pod.APIVersion,
					Kind:       pod.Kind,
					Name:       pod.Name,
					Namespace:  pod.Namespace,
				},
			})
		}
	}

	// Validate network policies
	if dv.validationConfig.SecurityPolicies.RequireNetworkPolicies {
		if err := dv.validateNetworkPolicies(ctx, spec, result); err != nil {
			return fmt.Errorf("network policy validation failed: %w", err)
		}
	}

	return nil
}

// Individual validation methods

func (dv *DeploymentValidator) validateDeploymentReadiness(deployment *appsv1.Deployment, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "deployment-readiness",
		Type: ValidationTypeDeployment,
		Resource: ResourceReference{
			APIVersion: deployment.APIVersion,
			Kind:       deployment.Kind,
			Name:       deployment.Name,
			Namespace:  deployment.Namespace,
		},
	}

	if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
		validationResult.Result = ValidationResultPass
		validationResult.Message = fmt.Sprintf("Deployment %s is ready with %d/%d replicas",
			deployment.Name, deployment.Status.ReadyReplicas, deployment.Status.Replicas)
	} else {
		validationResult.Result = ValidationResultFail
		validationResult.Message = fmt.Sprintf("Deployment %s is not ready: %d/%d replicas ready",
			deployment.Name, deployment.Status.ReadyReplicas, deployment.Status.Replicas)
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateDeploymentConfiguration(deployment *appsv1.Deployment, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "deployment-configuration",
		Type: ValidationTypeConfiguration,
		Resource: ResourceReference{
			APIVersion: deployment.APIVersion,
			Kind:       deployment.Kind,
			Name:       deployment.Name,
			Namespace:  deployment.Namespace,
		},
	}

	issues := []string{}

	// Check if deployment has required labels
	if deployment.Labels == nil {
		issues = append(issues, "missing labels")
	} else {
		requiredLabels := []string{"app", "app.kubernetes.io/name", "oran.io/vnf-type"}
		for _, label := range requiredLabels {
			if _, exists := deployment.Labels[label]; !exists {
				issues = append(issues, fmt.Sprintf("missing required label: %s", label))
			}
		}
	}

	// Check if deployment has resource limits
	containers := deployment.Spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Resources.Limits == nil || container.Resources.Requests == nil {
			issues = append(issues, fmt.Sprintf("container %s missing resource limits/requests", container.Name))
		}
	}

	if len(issues) == 0 {
		validationResult.Result = ValidationResultPass
		validationResult.Message = "Deployment configuration is valid"
	} else {
		validationResult.Result = ValidationResultWarning
		validationResult.Message = fmt.Sprintf("Deployment configuration issues: %s", strings.Join(issues, ", "))
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateDeploymentReplicas(deployment *appsv1.Deployment, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "deployment-replicas",
		Type: ValidationTypeDeployment,
		Resource: ResourceReference{
			APIVersion: deployment.APIVersion,
			Kind:       deployment.Kind,
			Name:       deployment.Name,
			Namespace:  deployment.Namespace,
		},
	}

	minReplicas := dv.validationConfig.ResourceThresholds.MinAvailableReplicas
	if deployment.Status.AvailableReplicas >= minReplicas {
		validationResult.Result = ValidationResultPass
		validationResult.Message = fmt.Sprintf("Deployment has sufficient replicas: %d (min: %d)",
			deployment.Status.AvailableReplicas, minReplicas)
	} else {
		validationResult.Result = ValidationResultFail
		validationResult.Message = fmt.Sprintf("Deployment has insufficient replicas: %d (min: %d)",
			deployment.Status.AvailableReplicas, minReplicas)
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateDeploymentStrategy(deployment *appsv1.Deployment, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "deployment-strategy",
		Type: ValidationTypeConfiguration,
		Resource: ResourceReference{
			APIVersion: deployment.APIVersion,
			Kind:       deployment.Kind,
			Name:       deployment.Name,
			Namespace:  deployment.Namespace,
		},
	}

	strategy := deployment.Spec.Strategy.Type
	if strategy == appsv1.RollingUpdateDeploymentStrategyType {
		validationResult.Result = ValidationResultPass
		validationResult.Message = "Deployment uses RollingUpdate strategy"
	} else {
		validationResult.Result = ValidationResultWarning
		validationResult.Message = fmt.Sprintf("Deployment uses %s strategy (consider RollingUpdate for zero downtime)", strategy)
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateServiceEndpoints(ctx context.Context, service *corev1.Service, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "service-endpoints",
		Type: ValidationTypeService,
		Resource: ResourceReference{
			APIVersion: service.APIVersion,
			Kind:       service.Kind,
			Name:       service.Name,
			Namespace:  service.Namespace,
		},
	}

	// Get endpoints for the service
	endpoints := &corev1.Endpoints{}
	if err := dv.client.Get(ctx, client.ObjectKey{
		Name:      service.Name,
		Namespace: service.Namespace,
	}, endpoints); err != nil {
		validationResult.Result = ValidationResultFail
		validationResult.Message = fmt.Sprintf("Failed to get endpoints for service: %v", err)
	} else {
		readyAddresses := 0
		for _, subset := range endpoints.Subsets {
			readyAddresses += len(subset.Addresses)
		}

		if readyAddresses > 0 {
			validationResult.Result = ValidationResultPass
			validationResult.Message = fmt.Sprintf("Service has %d ready endpoints", readyAddresses)
		} else {
			validationResult.Result = ValidationResultFail
			validationResult.Message = "Service has no ready endpoints"
		}
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateServiceConfiguration(service *corev1.Service, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "service-configuration",
		Type: ValidationTypeConfiguration,
		Resource: ResourceReference{
			APIVersion: service.APIVersion,
			Kind:       service.Kind,
			Name:       service.Name,
			Namespace:  service.Namespace,
		},
	}

	issues := []string{}

	// Check if service has selector
	if len(service.Spec.Selector) == 0 {
		issues = append(issues, "service has no selector")
	}

	// Check if service has ports
	if len(service.Spec.Ports) == 0 {
		issues = append(issues, "service has no ports")
	}

	if len(issues) == 0 {
		validationResult.Result = ValidationResultPass
		validationResult.Message = "Service configuration is valid"
	} else {
		validationResult.Result = ValidationResultWarning
		validationResult.Message = fmt.Sprintf("Service configuration issues: %s", strings.Join(issues, ", "))
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateServicePorts(service *corev1.Service, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "service-ports",
		Type: ValidationTypeService,
		Resource: ResourceReference{
			APIVersion: service.APIVersion,
			Kind:       service.Kind,
			Name:       service.Name,
			Namespace:  service.Namespace,
		},
	}

	// Check for VNF-specific ports based on service labels
	vnfType := ""
	if service.Labels != nil {
		vnfType = service.Labels["oran.io/vnf-type"]
	}

	var requiredPorts []int32
	switch vnfType {
	case "RAN":
		requiredPorts = []int32{38412, 2152} // SCTP, GTP
	case "CN":
		requiredPorts = []int32{8080, 8443}  // HTTP, HTTPS
	case "TN":
		requiredPorts = []int32{830, 6640}   // NETCONF, OVSDB
	}

	if len(requiredPorts) > 0 {
		foundPorts := make(map[int32]bool)
		for _, port := range service.Spec.Ports {
			foundPorts[port.Port] = true
		}

		missingPorts := []int32{}
		for _, required := range requiredPorts {
			if !foundPorts[required] {
				missingPorts = append(missingPorts, required)
			}
		}

		if len(missingPorts) == 0 {
			validationResult.Result = ValidationResultPass
			validationResult.Message = fmt.Sprintf("Service has all required ports for %s VNF", vnfType)
		} else {
			validationResult.Result = ValidationResultWarning
			validationResult.Message = fmt.Sprintf("Service missing required ports for %s VNF: %v", vnfType, missingPorts)
		}
	} else {
		validationResult.Result = ValidationResultPass
		validationResult.Message = "Service ports configuration is valid"
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateNetworkPolicies(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// List network policies
	networkPolicyList := &networkingv1.NetworkPolicyList{}
	listOptions := []client.ListOption{
		client.InNamespace(spec.Namespace),
	}

	if err := dv.client.List(ctx, networkPolicyList, listOptions...); err != nil {
		return fmt.Errorf("failed to list network policies: %w", err)
	}

	validationResult := IndividualValidationResult{
		Name: "network-policies",
		Type: ValidationTypeNetworking,
	}

	if len(networkPolicyList.Items) > 0 {
		validationResult.Result = ValidationResultPass
		validationResult.Message = fmt.Sprintf("Found %d network policies", len(networkPolicyList.Items))
	} else {
		if dv.validationConfig.SecurityPolicies.RequireNetworkPolicies {
			validationResult.Result = ValidationResultFail
			validationResult.Message = "No network policies found (required for security)"
		} else {
			validationResult.Result = ValidationResultWarning
			validationResult.Message = "No network policies found"
		}
	}

	result.Results = append(result.Results, validationResult)
	return nil
}

func (dv *DeploymentValidator) validateIngress(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// This would validate ingress resources if present
	// For now, just return success
	return nil
}

func (dv *DeploymentValidator) validatePodConnectivity(ctx context.Context, spec *DeploymentValidationSpec, result *ValidationResult) error {
	// This would perform actual connectivity tests between pods
	// For now, just return success
	return nil
}

func (dv *DeploymentValidator) validatePodResources(pod *corev1.Pod, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "pod-resources",
		Type: ValidationTypeResource,
		Resource: ResourceReference{
			APIVersion: pod.APIVersion,
			Kind:       pod.Kind,
			Name:       pod.Name,
			Namespace:  pod.Namespace,
		},
	}

	issues := []string{}

	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests == nil {
			issues = append(issues, fmt.Sprintf("container %s missing resource requests", container.Name))
		}

		if container.Resources.Limits == nil {
			issues = append(issues, fmt.Sprintf("container %s missing resource limits", container.Name))
		}
	}

	if len(issues) == 0 {
		validationResult.Result = ValidationResultPass
		validationResult.Message = "Pod has proper resource configuration"
	} else {
		if dv.validationConfig.SecurityPolicies.RequireResourceLimits {
			validationResult.Result = ValidationResultFail
		} else {
			validationResult.Result = ValidationResultWarning
		}
		validationResult.Message = strings.Join(issues, ", ")
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validatePodResourceUtilization(ctx context.Context, pod *corev1.Pod, result *ValidationResult) error {
	// This would fetch actual resource utilization metrics
	// For now, just simulate success
	return nil
}

func (dv *DeploymentValidator) hasQoSAnnotations(pod *corev1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}

	qosAnnotations := []string{
		"oran.io/qos-bandwidth",
		"oran.io/qos-latency",
		"oran.io/qos-class",
	}

	for _, annotation := range qosAnnotations {
		if _, exists := pod.Annotations[annotation]; exists {
			return true
		}
	}

	return false
}

func (dv *DeploymentValidator) validatePodQoS(pod *corev1.Pod, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "pod-qos",
		Type: ValidationTypeQoS,
		Resource: ResourceReference{
			APIVersion: pod.APIVersion,
			Kind:       pod.Kind,
			Name:       pod.Name,
			Namespace:  pod.Namespace,
		},
	}

	// Validate QoS annotations
	qosClass := pod.Annotations["oran.io/qos-class"]
	if qosClass != "" {
		validationResult.Result = ValidationResultPass
		validationResult.Message = fmt.Sprintf("Pod has QoS class: %s", qosClass)
	} else {
		validationResult.Result = ValidationResultWarning
		validationResult.Message = "Pod missing QoS class annotation"
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateQoSMetrics(ctx context.Context, pod *corev1.Pod, result *ValidationResult) error {
	// This would fetch and validate actual QoS metrics
	// For now, just return success
	return nil
}

func (dv *DeploymentValidator) validatePodSecurityContext(pod *corev1.Pod, result *ValidationResult) {
	validationResult := IndividualValidationResult{
		Name: "pod-security-context",
		Type: ValidationTypeSecurity,
		Resource: ResourceReference{
			APIVersion: pod.APIVersion,
			Kind:       pod.Kind,
			Name:       pod.Name,
			Namespace:  pod.Namespace,
		},
	}

	issues := []string{}

	if pod.Spec.SecurityContext == nil {
		issues = append(issues, "missing security context")
	} else {
		sc := pod.Spec.SecurityContext

		if dv.validationConfig.SecurityPolicies.RequireNonRoot {
			if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
				issues = append(issues, "should run as non-root")
			}
		}

		// Note: ReadOnlyRootFilesystem is a container-level setting, not pod-level
		// This validation would need to be moved to container security context validation
	}

	if len(issues) == 0 {
		validationResult.Result = ValidationResultPass
		validationResult.Message = "Pod security context is valid"
	} else {
		validationResult.Result = ValidationResultFail
		validationResult.Message = strings.Join(issues, ", ")
	}

	result.Results = append(result.Results, validationResult)
}

func (dv *DeploymentValidator) validateContainerSecurityContext(pod *corev1.Pod, result *ValidationResult) {
	for _, container := range pod.Spec.Containers {
		validationResult := IndividualValidationResult{
			Name: fmt.Sprintf("container-security-context-%s", container.Name),
			Type: ValidationTypeSecurity,
			Resource: ResourceReference{
				APIVersion: pod.APIVersion,
				Kind:       pod.Kind,
				Name:       pod.Name,
				Namespace:  pod.Namespace,
			},
		}

		issues := []string{}

		if container.SecurityContext == nil {
			issues = append(issues, "missing container security context")
		} else {
			sc := container.SecurityContext

			if sc.AllowPrivilegeEscalation != nil && *sc.AllowPrivilegeEscalation {
				issues = append(issues, "allows privilege escalation")
			}

			if sc.Privileged != nil && *sc.Privileged {
				issues = append(issues, "runs as privileged")
			}

			// Check capabilities
			if sc.Capabilities != nil {
				for _, cap := range sc.Capabilities.Add {
					if dv.isCapabilityForbidden(string(cap)) {
						issues = append(issues, fmt.Sprintf("forbidden capability: %s", cap))
					}
				}
			}
		}

		if len(issues) == 0 {
			validationResult.Result = ValidationResultPass
			validationResult.Message = fmt.Sprintf("Container %s security context is valid", container.Name)
		} else {
			validationResult.Result = ValidationResultFail
			validationResult.Message = fmt.Sprintf("Container %s: %s", container.Name, strings.Join(issues, ", "))
		}

		result.Results = append(result.Results, validationResult)
	}
}

func (dv *DeploymentValidator) validatePodSecurityPolicies(ctx context.Context, pod *corev1.Pod, result *ValidationResult) error {
	// This would validate against Pod Security Policies or Pod Security Standards
	// For now, just return success
	return nil
}

func (dv *DeploymentValidator) isCapabilityForbidden(capability string) bool {
	for _, forbidden := range dv.validationConfig.SecurityPolicies.ForbiddenCapabilities {
		if capability == forbidden {
			return true
		}
	}
	return false
}

func (dv *DeploymentValidator) calculateSummary(result *ValidationResult) ValidationSummary {
	summary := ValidationSummary{
		TotalValidations: len(result.Results),
		WarningCount:     len(result.Warnings),
	}

	for _, r := range result.Results {
		switch r.Result {
		case ValidationResultPass:
			summary.SuccessfulValidations++
		case ValidationResultFail:
			summary.FailedValidations++
		}
	}

	if summary.TotalValidations > 0 {
		summary.ValidationScore = float64(summary.SuccessfulValidations) / float64(summary.TotalValidations) * 100
	}

	return summary
}

// DefaultValidationConfig returns a default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		Timeout:                  5 * time.Minute,
		RetryInterval:            10 * time.Second,
		EnableNetworkValidation:  true,
		EnableResourceValidation: true,
		EnableQoSValidation:      true,
		EnableSecurityValidation: true,
		QoSThresholds: QoSThresholds{
			MaxLatencyMs:          100,
			MaxJitterMs:           10,
			MaxPacketLossPercent:  1.0,
			MinBandwidthMbps:      1.0,
			MinReliabilityPercent: 99.0,
		},
		ResourceThresholds: ResourceThresholds{
			MaxCPUUtilizationPercent:    80,
			MaxMemoryUtilizationPercent: 80,
			MaxStorageUtilizationPercent: 80,
			MinAvailableReplicas:        1,
		},
		SecurityPolicies: SecurityPolicies{
			RequireNonRoot:        true,
			RequireReadOnlyRootFS: false,
			RequireResourceLimits: true,
			RequireNetworkPolicies: false,
			AllowedCapabilities:   []string{"NET_BIND_SERVICE"},
			ForbiddenCapabilities: []string{"SYS_ADMIN", "NET_ADMIN", "SYS_TIME"},
		},
	}
}