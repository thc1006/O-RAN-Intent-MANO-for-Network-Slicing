// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

// Package validation provides comprehensive GitOps validation framework
// for O-RAN Intent-based MANO system with Nephio/Porch integration
package validation

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// ValidationFramework provides comprehensive GitOps validation
type ValidationFramework struct {
	Config           *ValidationConfig
	KubeClients      map[string]*ClusterClient
	GitRepo          *GitRepository
	PackageValidator *NephioValidator
	MetricsCollector *MetricsCollector
	mu               sync.RWMutex
}

// ValidationConfig holds validation framework configuration
type ValidationConfig struct {
	Clusters       []ClusterConfig      `yaml:"clusters"`
	Git            GitConfig            `yaml:"git"`
	Nephio         NephioConfig         `yaml:"nephio"`
	Validation     ValidationRules      `yaml:"validation"`
	Monitoring     MonitoringConfig     `yaml:"monitoring"`
	Rollback       RollbackConfig       `yaml:"rollback"`
	DriftDetection DriftDetectionConfig `yaml:"driftDetection"`
	Performance    PerformanceConfig    `yaml:"performance"`
}

// ClusterConfig represents cluster-specific configuration
type ClusterConfig struct {
	Name         string            `yaml:"name"`
	Type         string            `yaml:"type"` // edge01, edge02, regional, central
	KubeConfig   string            `yaml:"kubeconfig"`
	Context      string            `yaml:"context"`
	Packages     []string          `yaml:"packages"`
	Capabilities []string          `yaml:"capabilities"`
	Labels       map[string]string `yaml:"labels"`
	Environment  string            `yaml:"environment"`
}

// ClusterClient wraps Kubernetes clients for a cluster
type ClusterClient struct {
	Config        *rest.Config
	Clientset     *kubernetes.Clientset
	DynamicClient dynamic.Interface
	Client        client.Client
	Discovery     discovery.DiscoveryInterface
	Context       string
}

// GitConfig holds Git repository configuration
type GitConfig struct {
	RepoURL    string `yaml:"repoUrl"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
	AuthToken  string `yaml:"authToken,omitempty"`
	SSHKeyPath string `yaml:"sshKeyPath,omitempty"`
}

// NephioConfig holds Nephio/Porch specific configuration
type NephioConfig struct {
	PorchServer   string            `yaml:"porchServer"`
	Repositories  []string          `yaml:"repositories"`
	PackagePaths  map[string]string `yaml:"packagePaths"`
	RenderTimeout time.Duration     `yaml:"renderTimeout"`
}

// ValidationRules defines validation criteria
type ValidationRules struct {
	RequiredResources     []ResourceRule        `yaml:"requiredResources"`
	ReadinessTimeout      time.Duration         `yaml:"readinessTimeout"`
	DriftTolerance        DriftTolerance        `yaml:"driftTolerance"`
	PerformanceThresholds PerformanceThresholds `yaml:"performanceThresholds"`
}

// ResourceRule defines validation rules for Kubernetes resources
type ResourceRule struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Name       string            `yaml:"name,omitempty"`
	Namespace  string            `yaml:"namespace,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
	Fields     []FieldRule       `yaml:"fields,omitempty"`
}

// FieldRule defines validation for specific resource fields
type FieldRule struct {
	Path      string `yaml:"path"`
	Value     string `yaml:"value,omitempty"`
	Condition string `yaml:"condition"` // exists, equals, contains, matches
}

// DriftTolerance defines acceptable drift parameters
type DriftTolerance struct {
	MaxDriftPercentage float64       `yaml:"maxDriftPercentage"`
	CheckInterval      time.Duration `yaml:"checkInterval"`
	AutoCorrect        bool          `yaml:"autoCorrect"`
}

// PerformanceThresholds defines expected performance metrics
type PerformanceThresholds struct {
	DeploymentTime    time.Duration `yaml:"deploymentTime"` // <10 min
	ThroughputMbps    []float64     `yaml:"throughputMbps"` // [4.57, 2.77, 0.93]
	PingRTTMs         []float64     `yaml:"pingRttMs"`      // [16.1, 15.7, 6.3]
	CPUUtilization    float64       `yaml:"cpuUtilization"`
	MemoryUtilization float64       `yaml:"memoryUtilization"`
}

// MonitoringConfig defines monitoring integration
type MonitoringConfig struct {
	Enabled        bool          `yaml:"enabled"`
	PrometheusURL  string        `yaml:"prometheusUrl"`
	GrafanaURL     string        `yaml:"grafanaUrl"`
	AlertManager   string        `yaml:"alertManager"`
	MetricsPath    string        `yaml:"metricsPath"`
	ScrapeInterval time.Duration `yaml:"scrapeInterval"`
}

// RollbackConfig defines automated rollback behavior
type RollbackConfig struct {
	Enabled            bool          `yaml:"enabled"`
	MaxRollbacks       int           `yaml:"maxRollbacks"`
	RollbackTimeout    time.Duration `yaml:"rollbackTimeout"`
	HealthCheckTimeout time.Duration `yaml:"healthCheckTimeout"`
	PreserveData       bool          `yaml:"preserveData"`
}

// DriftDetectionConfig configures drift detection
type DriftDetectionConfig struct {
	Enabled      bool          `yaml:"enabled"`
	ScanInterval time.Duration `yaml:"scanInterval"`
	Remediation  string        `yaml:"remediation"` // alert, correct, rollback
	IgnoreFields []string      `yaml:"ignoreFields"`
}

// PerformanceConfig defines performance monitoring
type PerformanceConfig struct {
	Enabled            bool               `yaml:"enabled"`
	CollectionInterval time.Duration      `yaml:"collectionInterval"`
	RetentionPeriod    time.Duration      `yaml:"retentionPeriod"`
	AlertThresholds    map[string]float64 `yaml:"alertThresholds"`
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Timestamp   time.Time                  `json:"timestamp"`
	Cluster     string                     `json:"cluster"`
	Success     bool                       `json:"success"`
	Errors      []string                   `json:"errors,omitempty"`
	Warnings    []string                   `json:"warnings,omitempty"`
	Resources   []ResourceValidationResult `json:"resources"`
	Performance *PerformanceResult         `json:"performance,omitempty"`
	GitState    *GitValidationResult       `json:"gitState,omitempty"`
	Duration    time.Duration              `json:"duration"`
}

// ResourceValidationResult represents validation result for a specific resource
type ResourceValidationResult struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	Kind       string    `json:"kind"`
	Ready      bool      `json:"ready"`
	Status     string    `json:"status"`
	Conditions []string  `json:"conditions,omitempty"`
	LastUpdate time.Time `json:"lastUpdate"`
}

// PerformanceResult holds performance validation results
type PerformanceResult struct {
	DeploymentTime    time.Duration `json:"deploymentTime"`
	ThroughputMbps    []float64     `json:"throughputMbps"`
	PingRTTMs         []float64     `json:"pingRttMs"`
	CPUUtilization    float64       `json:"cpuUtilization"`
	MemoryUtilization float64       `json:"memoryUtilization"`
	WithinThresholds  bool          `json:"withinThresholds"`
}

// GitValidationResult represents Git repository validation result
type GitValidationResult struct {
	Branch     string    `json:"branch"`
	LastCommit string    `json:"lastCommit"`
	CleanState bool      `json:"cleanState"`
	SyncStatus string    `json:"syncStatus"`
	LastSync   time.Time `json:"lastSync"`
}

// NewValidationFramework creates a new validation framework instance
func NewValidationFramework(configPath string) (*ValidationFramework, error) {
	config, err := LoadValidationConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load validation config: %w", err)
	}

	framework := &ValidationFramework{
		Config:      config,
		KubeClients: make(map[string]*ClusterClient),
	}

	// Initialize cluster clients
	for _, cluster := range config.Clusters {
		client, err := framework.createClusterClient(cluster)
		if err != nil {
			log.Printf("Warning: failed to create client for cluster %s: %v", cluster.Name, err)
			continue
		}
		framework.KubeClients[cluster.Name] = client
	}

	// Initialize Git repository
	framework.GitRepo, err = NewGitRepository(config.Git)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Initialize Nephio validator
	framework.PackageValidator, err = NewNephioValidator(config.Nephio)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Nephio validator: %w", err)
	}

	// Initialize metrics collector
	framework.MetricsCollector, err = NewMetricsCollector(config.Monitoring, config.Performance)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics collector: %w", err)
	}

	return framework, nil
}

// LoadValidationConfig loads validation configuration from file
func LoadValidationConfig(configPath string) (*ValidationConfig, error) {
	// Create validator for configuration files
	validator := security.CreateValidatorForConfig(".")

	// Validate file path for security
	if err := validator.ValidateFilePathAndExtension(configPath, []string{".yaml", ".yml", ".json", ".toml", ".conf", ".cfg"}); err != nil {
		return nil, fmt.Errorf("config file path validation failed: %w", err)
	}

	data, err := validator.SafeReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ValidationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Validation.ReadinessTimeout == 0 {
		config.Validation.ReadinessTimeout = 10 * time.Minute
	}
	if config.Nephio.RenderTimeout == 0 {
		config.Nephio.RenderTimeout = 5 * time.Minute
	}
	if config.Validation.DriftTolerance.CheckInterval == 0 {
		config.Validation.DriftTolerance.CheckInterval = 30 * time.Second
	}

	return &config, nil
}

// createClusterClient creates a Kubernetes client for the specified cluster
func (vf *ValidationFramework) createClusterClient(cluster ClusterConfig) (*ClusterClient, error) {
	var config *rest.Config
	var err error

	if cluster.KubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", cluster.KubeConfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	runtimeClient, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}

	return &ClusterClient{
		Config:        config,
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		Client:        runtimeClient,
		Discovery:     discovery,
		Context:       cluster.Context,
	}, nil
}

// ValidateAll performs comprehensive validation across all clusters
func (vf *ValidationFramework) ValidateAll(ctx context.Context) (map[string]*ValidationResult, error) {
	vf.mu.RLock()
	clusters := make([]string, 0, len(vf.KubeClients))
	for name := range vf.KubeClients {
		clusters = append(clusters, name)
	}
	vf.mu.RUnlock()

	results := make(map[string]*ValidationResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, clusterName := range clusters {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			result, err := vf.ValidateCluster(ctx, name)
			if err != nil {
				result = &ValidationResult{
					Timestamp: time.Now(),
					Cluster:   name,
					Success:   false,
					Errors:    []string{err.Error()},
				}
			}
			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(clusterName)
	}

	wg.Wait()
	return results, nil
}

// ValidateCluster performs validation for a specific cluster
func (vf *ValidationFramework) ValidateCluster(ctx context.Context, clusterName string) (*ValidationResult, error) {
	startTime := time.Now()

	client, exists := vf.KubeClients[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}

	result := &ValidationResult{
		Timestamp: startTime,
		Cluster:   clusterName,
		Success:   true,
		Resources: make([]ResourceValidationResult, 0),
	}

	// Validate Git state
	gitResult, err := vf.validateGitState(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Git validation failed: %v", err))
		result.Success = false
	}
	result.GitState = gitResult

	// Validate Nephio packages
	if err := vf.validateNephioPackages(ctx, clusterName); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Nephio package validation failed: %v", err))
		result.Success = false
	}

	// Validate resources
	resourceResults, err := vf.validateResources(ctx, client, clusterName)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Resource validation failed: %v", err))
		result.Success = false
	}
	result.Resources = resourceResults

	// Validate performance metrics
	perfResult, err := vf.validatePerformance(ctx, clusterName)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Performance validation warning: %v", err))
	}
	result.Performance = perfResult

	// Check drift detection
	if vf.Config.DriftDetection.Enabled {
		if err := vf.detectDrift(ctx, client, clusterName); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Drift detection warning: %v", err))
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// validateGitState validates the Git repository state
func (vf *ValidationFramework) validateGitState(_ context.Context) (*GitValidationResult, error) {
	if vf.GitRepo == nil {
		return nil, fmt.Errorf("git repository not initialized")
	}

	branch, err := vf.GitRepo.GetCurrentBranch()
	if err != nil {
		return nil, err
	}

	lastCommit, err := vf.GitRepo.GetLastCommit()
	if err != nil {
		return nil, err
	}

	cleanState, err := vf.GitRepo.IsClean()
	if err != nil {
		return nil, err
	}

	syncStatus, lastSync, err := vf.GitRepo.GetSyncStatus()
	if err != nil {
		return nil, err
	}

	return &GitValidationResult{
		Branch:     branch,
		LastCommit: lastCommit,
		CleanState: cleanState,
		SyncStatus: syncStatus,
		LastSync:   lastSync,
	}, nil
}

// validateNephioPackages validates Nephio/Porch packages
func (vf *ValidationFramework) validateNephioPackages(ctx context.Context, clusterName string) error {
	if vf.PackageValidator == nil {
		return fmt.Errorf("nephio validator not initialized")
	}

	cluster := vf.findClusterConfig(clusterName)
	if cluster == nil {
		return fmt.Errorf("cluster config not found for %s", clusterName)
	}

	for _, packagePath := range cluster.Packages {
		if err := vf.PackageValidator.ValidatePackage(ctx, packagePath); err != nil {
			return fmt.Errorf("package validation failed for %s: %w", packagePath, err)
		}
	}

	return nil
}

// validateResources validates Kubernetes resources
func (vf *ValidationFramework) validateResources(ctx context.Context, client *ClusterClient, _ string) ([]ResourceValidationResult, error) {
	var results []ResourceValidationResult

	for _, rule := range vf.Config.Validation.RequiredResources {
		gvr := schema.GroupVersionResource{
			Group:    strings.Split(rule.APIVersion, "/")[0],
			Version:  strings.Split(rule.APIVersion, "/")[1],
			Resource: strings.ToLower(rule.Kind) + "s", // Simple pluralization
		}

		// List resources
		list, err := client.DynamicClient.Resource(gvr).Namespace(rule.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list %s: %w", rule.Kind, err)
		}

		for _, item := range list.Items {
			result := vf.validateResource(&item, rule)
			results = append(results, result)
		}
	}

	return results, nil
}

// validateResource validates a single Kubernetes resource
func (vf *ValidationFramework) validateResource(resource *unstructured.Unstructured, _ ResourceRule) ResourceValidationResult {
	result := ResourceValidationResult{
		Name:       resource.GetName(),
		Namespace:  resource.GetNamespace(),
		Kind:       resource.GetKind(),
		LastUpdate: time.Now(),
	}

	// Check readiness
	ready, status := vf.checkResourceReadiness(resource)
	result.Ready = ready
	result.Status = status

	// Check conditions
	conditions := vf.extractConditions(resource)
	result.Conditions = conditions

	return result
}

// checkResourceReadiness checks if a resource is ready
func (vf *ValidationFramework) checkResourceReadiness(resource *unstructured.Unstructured) (bool, string) {
	// Implementation depends on resource type
	switch resource.GetKind() {
	case "Deployment":
		return vf.checkDeploymentReadiness(resource)
	case "Pod":
		return vf.checkPodReadiness(resource)
	case "Service":
		return vf.checkServiceReadiness(resource)
	default:
		return true, "Unknown"
	}
}

// checkDeploymentReadiness checks deployment readiness
func (vf *ValidationFramework) checkDeploymentReadiness(resource *unstructured.Unstructured) (bool, string) {
	status, found, err := unstructured.NestedMap(resource.Object, "status")
	if !found || err != nil {
		return false, "NoStatus"
	}

	readyReplicas, found, _ := unstructured.NestedInt64(status, "readyReplicas")
	replicas, found2, _ := unstructured.NestedInt64(status, "replicas")

	if found && found2 && readyReplicas == replicas && replicas > 0 {
		return true, "Ready"
	}

	return false, "NotReady"
}

// checkPodReadiness checks pod readiness
func (vf *ValidationFramework) checkPodReadiness(resource *unstructured.Unstructured) (bool, string) {
	status, found, err := unstructured.NestedMap(resource.Object, "status")
	if !found || err != nil {
		return false, "NoStatus"
	}

	phase, found, _ := unstructured.NestedString(status, "phase")
	if found && phase == "Running" {
		return true, "Running"
	}

	return false, phase
}

// checkServiceReadiness checks service readiness
func (vf *ValidationFramework) checkServiceReadiness(resource *unstructured.Unstructured) (bool, string) {
	// Services are generally ready when created
	return true, "Ready"
}

// extractConditions extracts conditions from a resource
func (vf *ValidationFramework) extractConditions(resource *unstructured.Unstructured) []string {
	var conditions []string

	status, found, err := unstructured.NestedMap(resource.Object, "status")
	if !found || err != nil {
		return conditions
	}

	conditionList, found, err := unstructured.NestedSlice(status, "conditions")
	if !found || err != nil {
		return conditions
	}

	for _, cond := range conditionList {
		if condMap, ok := cond.(map[string]interface{}); ok {
			if condType, found := condMap["type"]; found {
				if condStatus, found := condMap["status"]; found {
					conditions = append(conditions, fmt.Sprintf("%v=%v", condType, condStatus))
				}
			}
		}
	}

	return conditions
}

// validatePerformance validates performance metrics
func (vf *ValidationFramework) validatePerformance(ctx context.Context, clusterName string) (*PerformanceResult, error) {
	if !vf.Config.Performance.Enabled || vf.MetricsCollector == nil {
		return nil, nil
	}

	metrics, err := vf.MetricsCollector.CollectMetrics(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	thresholds := vf.Config.Validation.PerformanceThresholds

	result := &PerformanceResult{
		DeploymentTime:    metrics.DeploymentTime,
		ThroughputMbps:    metrics.ThroughputMbps,
		PingRTTMs:         metrics.PingRTTMs,
		CPUUtilization:    metrics.CPUUtilization,
		MemoryUtilization: metrics.MemoryUtilization,
	}

	// Check if within thresholds
	result.WithinThresholds = vf.checkPerformanceThresholds(result, thresholds)

	return result, nil
}

// checkPerformanceThresholds validates performance against thresholds
func (vf *ValidationFramework) checkPerformanceThresholds(result *PerformanceResult, thresholds PerformanceThresholds) bool {
	if result.DeploymentTime > thresholds.DeploymentTime {
		return false
	}

	if result.CPUUtilization > thresholds.CPUUtilization {
		return false
	}

	if result.MemoryUtilization > thresholds.MemoryUtilization {
		return false
	}

	// Check throughput thresholds
	for i, expected := range thresholds.ThroughputMbps {
		if i < len(result.ThroughputMbps) && result.ThroughputMbps[i] < expected*0.9 { // 10% tolerance
			return false
		}
	}

	// Check RTT thresholds
	for i, expected := range thresholds.PingRTTMs {
		if i < len(result.PingRTTMs) && result.PingRTTMs[i] > expected*1.1 { // 10% tolerance
			return false
		}
	}

	return true
}

// detectDrift detects configuration drift
// TODO: Implement actual drift detection logic
func (vf *ValidationFramework) detectDrift(ctx context.Context, client *ClusterClient, clusterName string) error {
	// Placeholder for drift detection logic
	// This would compare current cluster state with desired state from Git
	log.Printf("Drift detection for cluster %s: Not implemented", clusterName)

	// For now, we'll just log that drift detection is not implemented
	// In a real implementation, this would:
	// 1. Get desired state from Git repository
	// 2. Get actual state from Kubernetes cluster
	// 3. Compare the states
	// 4. Return error if significant drift is detected

	return nil // No drift detected in placeholder implementation
}

// findClusterConfig finds cluster configuration by name
func (vf *ValidationFramework) findClusterConfig(name string) *ClusterConfig {
	for _, cluster := range vf.Config.Clusters {
		if cluster.Name == name {
			return &cluster
		}
	}
	return nil
}
