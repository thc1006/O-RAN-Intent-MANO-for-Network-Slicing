// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ConfigSyncValidator validates Config Sync resources and status
type ConfigSyncValidator struct {
	ClusterClient *ClusterClient
	Config        ConfigSyncConfig
}

// ConfigSyncConfig holds Config Sync configuration
type ConfigSyncConfig struct {
	Namespace              string        `yaml:"namespace"`
	RootSyncName           string        `yaml:"rootSyncName"`
	RepoSyncTimeout        time.Duration `yaml:"repoSyncTimeout"`
	ValidationTimeout      time.Duration `yaml:"validationTimeout"`
	ExpectedSyncResources  []string      `yaml:"expectedSyncResources"`
	HealthCheckInterval    time.Duration `yaml:"healthCheckInterval"`
}

// ConfigSyncStatus represents the status of Config Sync
type ConfigSyncStatus struct {
	Healthy             bool                    `json:"healthy"`
	SyncStatus          string                  `json:"syncStatus"`
	LastSync            time.Time               `json:"lastSync"`
	Errors              []string                `json:"errors,omitempty"`
	Warnings            []string                `json:"warnings,omitempty"`
	RootSync            *RootSyncStatus         `json:"rootSync,omitempty"`
	RepoSyncs           []RepoSyncStatus        `json:"repoSyncs,omitempty"`
	ResourceGroups      []ResourceGroupStatus   `json:"resourceGroups,omitempty"`
	SyncedResources     int                     `json:"syncedResources"`
	ErroredResources    int                     `json:"erroredResources"`
	ValidationErrors    []ValidationError       `json:"validationErrors,omitempty"`
}

// RootSyncStatus represents RootSync status
type RootSyncStatus struct {
	Name              string              `json:"name"`
	Namespace         string              `json:"namespace"`
	Source            SourceStatus        `json:"source"`
	Sync              SyncStatus          `json:"sync"`
	Rendering         RenderingStatus     `json:"rendering"`
	Conditions        []ConditionStatus   `json:"conditions"`
	ObservedGeneration int64              `json:"observedGeneration"`
}

// RepoSyncStatus represents RepoSync status
type RepoSyncStatus struct {
	Name              string              `json:"name"`
	Namespace         string              `json:"namespace"`
	Source            SourceStatus        `json:"source"`
	Sync              SyncStatus          `json:"sync"`
	Rendering         RenderingStatus     `json:"rendering"`
	Conditions        []ConditionStatus   `json:"conditions"`
	ObservedGeneration int64              `json:"observedGeneration"`
}

// SourceStatus represents source status
type SourceStatus struct {
	Git             GitSourceStatus     `json:"git,omitempty"`
	Oci             OciSourceStatus     `json:"oci,omitempty"`
	Helm            HelmSourceStatus    `json:"helm,omitempty"`
	Errors          []ErrorStatus       `json:"errors,omitempty"`
	LastUpdate      time.Time           `json:"lastUpdate"`
}

// GitSourceStatus represents Git source status
type GitSourceStatus struct {
	Repo     string `json:"repo"`
	Revision string `json:"revision"`
	Branch   string `json:"branch"`
	Dir      string `json:"dir"`
	Commit   string `json:"commit"`
}

// OciSourceStatus represents OCI source status
type OciSourceStatus struct {
	Image    string `json:"image"`
	Dir      string `json:"dir"`
}

// HelmSourceStatus represents Helm source status
type HelmSourceStatus struct {
	Repo    string `json:"repo"`
	Chart   string `json:"chart"`
	Version string `json:"version"`
}

// SyncStatus represents sync status
type SyncStatus struct {
	Status           string      `json:"status"`
	LastUpdate       time.Time   `json:"lastUpdate"`
	GitStatus        GitStatus   `json:"gitStatus,omitempty"`
	Import           string      `json:"import,omitempty"`
	Sync             string      `json:"sync,omitempty"`
	Errors           []ErrorStatus `json:"errors,omitempty"`
}

// RenderingStatus represents rendering status
type RenderingStatus struct {
	Status     string        `json:"status"`
	LastUpdate time.Time     `json:"lastUpdate"`
	Message    string        `json:"message,omitempty"`
	Errors     []ErrorStatus `json:"errors,omitempty"`
}

// ConditionStatus represents condition status
type ConditionStatus struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastUpdateTime     time.Time `json:"lastUpdateTime"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
}

// ErrorStatus represents error status
type ErrorStatus struct {
	Code         string    `json:"code"`
	Description  string    `json:"description"`
	Timestamp    time.Time `json:"timestamp"`
	Resources    []string  `json:"resources,omitempty"`
}

// ResourceGroupStatus represents resource group status
type ResourceGroupStatus struct {
	Group     string   `json:"group"`
	Version   string   `json:"version"`
	Kind      string   `json:"kind"`
	Count     int      `json:"count"`
	Errors    int      `json:"errors"`
	Resources []string `json:"resources,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Resource    string    `json:"resource"`
	Kind        string    `json:"kind"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewConfigSyncValidator creates a new Config Sync validator
func NewConfigSyncValidator(client *ClusterClient, config ConfigSyncConfig) *ConfigSyncValidator {
	// Set defaults
	if config.Namespace == "" {
		config.Namespace = "config-management-system"
	}
	if config.RootSyncName == "" {
		config.RootSyncName = "root-sync"
	}
	if config.RepoSyncTimeout == 0 {
		config.RepoSyncTimeout = 5 * time.Minute
	}
	if config.ValidationTimeout == 0 {
		config.ValidationTimeout = 10 * time.Minute
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}

	return &ConfigSyncValidator{
		ClusterClient: client,
		Config:        config,
	}
}

// ValidateConfigSync validates Config Sync deployment and status
func (csv *ConfigSyncValidator) ValidateConfigSync(ctx context.Context) (*ConfigSyncStatus, error) {
	status := &ConfigSyncStatus{
		Healthy:            true,
		SyncStatus:        "unknown",
		ResourceGroups:    make([]ResourceGroupStatus, 0),
		ValidationErrors:  make([]ValidationError, 0),
	}

	// Check if Config Sync is installed
	if err := csv.checkConfigSyncInstallation(ctx); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("Config Sync installation check failed: %v", err))
		return status, nil
	}

	// Validate RootSync
	rootSyncStatus, err := csv.validateRootSync(ctx)
	if err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("RootSync validation failed: %v", err))
	} else {
		status.RootSync = rootSyncStatus
		if rootSyncStatus.Sync.Status != "SYNCED" {
			status.Healthy = false
		}
	}

	// Validate RepoSyncs
	repoSyncs, err := csv.validateRepoSyncs(ctx)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("RepoSync validation warning: %v", err))
	} else {
		status.RepoSyncs = repoSyncs
		for _, rs := range repoSyncs {
			if rs.Sync.Status != "SYNCED" {
				status.Healthy = false
			}
		}
	}

	// Validate synced resources
	if err := csv.validateSyncedResources(ctx, status); err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Synced resource validation warning: %v", err))
	}

	// Set overall sync status
	status.SyncStatus = csv.calculateOverallSyncStatus(status)

	return status, nil
}

// checkConfigSyncInstallation checks if Config Sync is properly installed
func (csv *ConfigSyncValidator) checkConfigSyncInstallation(ctx context.Context) error {
	// Check Config Sync namespace
	_, err := csv.ClusterClient.Clientset.CoreV1().Namespaces().Get(ctx, csv.Config.Namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Config Sync namespace %s not found", csv.Config.Namespace)
		}
		return fmt.Errorf("failed to get Config Sync namespace: %w", err)
	}

	// Check for Config Sync operator deployment
	deployments, err := csv.ClusterClient.Clientset.AppsV1().Deployments(csv.Config.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments in Config Sync namespace: %w", err)
	}

	expectedDeployments := []string{"config-management-operator", "reconciler-manager"}
	foundDeployments := make(map[string]bool)

	for _, deployment := range deployments.Items {
		for _, expected := range expectedDeployments {
			if deployment.Name == expected {
				foundDeployments[expected] = true
				// Check if deployment is ready
				if deployment.Status.ReadyReplicas == 0 {
					return fmt.Errorf("deployment %s is not ready", deployment.Name)
				}
			}
		}
	}

	for _, expected := range expectedDeployments {
		if !foundDeployments[expected] {
			return fmt.Errorf("expected deployment %s not found", expected)
		}
	}

	return nil
}

// validateRootSync validates RootSync resource
func (csv *ConfigSyncValidator) validateRootSync(ctx context.Context) (*RootSyncStatus, error) {
	// Define RootSync GVR
	rootSyncGVR := schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "rootsyncs",
	}

	// Get RootSync resource
	rootSync, err := csv.ClusterClient.DynamicClient.Resource(rootSyncGVR).
		Namespace(csv.Config.Namespace).
		Get(ctx, csv.Config.RootSyncName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get RootSync %s: %w", csv.Config.RootSyncName, err)
	}

	return csv.parseRootSyncStatus(rootSync)
}

// parseRootSyncStatus parses RootSync status from unstructured object
func (csv *ConfigSyncValidator) parseRootSyncStatus(rootSync *unstructured.Unstructured) (*RootSyncStatus, error) {
	status := &RootSyncStatus{
		Name:      rootSync.GetName(),
		Namespace: rootSync.GetNamespace(),
	}

	// Parse status
	statusObj, found, err := unstructured.NestedMap(rootSync.Object, "status")
	if !found || err != nil {
		return nil, fmt.Errorf("RootSync status not found or invalid")
	}

	// Parse observed generation
	if obsGen, found, _ := unstructured.NestedInt64(statusObj, "observedGeneration"); found {
		status.ObservedGeneration = obsGen
	}

	// Parse source status
	if sourceObj, found, _ := unstructured.NestedMap(statusObj, "source"); found {
		status.Source = csv.parseSourceStatus(sourceObj)
	}

	// Parse sync status
	if syncObj, found, _ := unstructured.NestedMap(statusObj, "sync"); found {
		status.Sync = csv.parseSyncStatus(syncObj)
	}

	// Parse rendering status
	if renderObj, found, _ := unstructured.NestedMap(statusObj, "rendering"); found {
		status.Rendering = csv.parseRenderingStatus(renderObj)
	}

	// Parse conditions
	if conditionsObj, found, _ := unstructured.NestedSlice(statusObj, "conditions"); found {
		status.Conditions = csv.parseConditions(conditionsObj)
	}

	return status, nil
}

// validateRepoSyncs validates all RepoSync resources
func (csv *ConfigSyncValidator) validateRepoSyncs(ctx context.Context) ([]RepoSyncStatus, error) {
	// Define RepoSync GVR
	repoSyncGVR := schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "reposyncs",
	}

	// List all RepoSync resources
	repoSyncList, err := csv.ClusterClient.DynamicClient.Resource(repoSyncGVR).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list RepoSync resources: %w", err)
	}

	var repoSyncs []RepoSyncStatus
	for _, item := range repoSyncList.Items {
		repoSync, err := csv.parseRepoSyncStatus(&item)
		if err != nil {
			continue // Skip invalid RepoSync resources
		}
		repoSyncs = append(repoSyncs, *repoSync)
	}

	return repoSyncs, nil
}

// parseRepoSyncStatus parses RepoSync status from unstructured object
func (csv *ConfigSyncValidator) parseRepoSyncStatus(repoSync *unstructured.Unstructured) (*RepoSyncStatus, error) {
	status := &RepoSyncStatus{
		Name:      repoSync.GetName(),
		Namespace: repoSync.GetNamespace(),
	}

	// Parse status (similar to RootSync)
	statusObj, found, err := unstructured.NestedMap(repoSync.Object, "status")
	if !found || err != nil {
		return nil, fmt.Errorf("RepoSync status not found or invalid")
	}

	// Parse observed generation
	if obsGen, found, _ := unstructured.NestedInt64(statusObj, "observedGeneration"); found {
		status.ObservedGeneration = obsGen
	}

	// Parse source, sync, rendering status and conditions (same as RootSync)
	if sourceObj, found, _ := unstructured.NestedMap(statusObj, "source"); found {
		status.Source = csv.parseSourceStatus(sourceObj)
	}

	if syncObj, found, _ := unstructured.NestedMap(statusObj, "sync"); found {
		status.Sync = csv.parseSyncStatus(syncObj)
	}

	if renderObj, found, _ := unstructured.NestedMap(statusObj, "rendering"); found {
		status.Rendering = csv.parseRenderingStatus(renderObj)
	}

	if conditionsObj, found, _ := unstructured.NestedSlice(statusObj, "conditions"); found {
		status.Conditions = csv.parseConditions(conditionsObj)
	}

	return status, nil
}

// parseSourceStatus parses source status from map
func (csv *ConfigSyncValidator) parseSourceStatus(sourceObj map[string]interface{}) SourceStatus {
	status := SourceStatus{
		LastUpdate: time.Now(),
	}

	// Parse Git source
	if gitObj, found, _ := unstructured.NestedMap(sourceObj, "git"); found {
		status.Git = GitSourceStatus{}
		if repo, found, _ := unstructured.NestedString(gitObj, "repo"); found {
			status.Git.Repo = repo
		}
		if revision, found, _ := unstructured.NestedString(gitObj, "revision"); found {
			status.Git.Revision = revision
		}
		if branch, found, _ := unstructured.NestedString(gitObj, "branch"); found {
			status.Git.Branch = branch
		}
		if dir, found, _ := unstructured.NestedString(gitObj, "dir"); found {
			status.Git.Dir = dir
		}
		if commit, found, _ := unstructured.NestedString(gitObj, "commit"); found {
			status.Git.Commit = commit
		}
	}

	// Parse errors
	if errorsObj, found, _ := unstructured.NestedSlice(sourceObj, "errors"); found {
		status.Errors = csv.parseErrorStatuses(errorsObj)
	}

	return status
}

// parseSyncStatus parses sync status from map
func (csv *ConfigSyncValidator) parseSyncStatus(syncObj map[string]interface{}) SyncStatus {
	status := SyncStatus{
		LastUpdate: time.Now(),
	}

	if syncStatus, found, _ := unstructured.NestedString(syncObj, "status"); found {
		status.Status = syncStatus
	}

	if importStatus, found, _ := unstructured.NestedString(syncObj, "import"); found {
		status.Import = importStatus
	}

	if syncVal, found, _ := unstructured.NestedString(syncObj, "sync"); found {
		status.Sync = syncVal
	}

	// Parse errors
	if errorsObj, found, _ := unstructured.NestedSlice(syncObj, "errors"); found {
		status.Errors = csv.parseErrorStatuses(errorsObj)
	}

	return status
}

// parseRenderingStatus parses rendering status from map
func (csv *ConfigSyncValidator) parseRenderingStatus(renderObj map[string]interface{}) RenderingStatus {
	status := RenderingStatus{
		LastUpdate: time.Now(),
	}

	if renderStatus, found, _ := unstructured.NestedString(renderObj, "status"); found {
		status.Status = renderStatus
	}

	if message, found, _ := unstructured.NestedString(renderObj, "message"); found {
		status.Message = message
	}

	// Parse errors
	if errorsObj, found, _ := unstructured.NestedSlice(renderObj, "errors"); found {
		status.Errors = csv.parseErrorStatuses(errorsObj)
	}

	return status
}

// parseConditions parses conditions from slice
func (csv *ConfigSyncValidator) parseConditions(conditionsObj []interface{}) []ConditionStatus {
	var conditions []ConditionStatus

	for _, condObj := range conditionsObj {
		if condMap, ok := condObj.(map[string]interface{}); ok {
			condition := ConditionStatus{}

			if condType, found, _ := unstructured.NestedString(condMap, "type"); found {
				condition.Type = condType
			}
			if status, found, _ := unstructured.NestedString(condMap, "status"); found {
				condition.Status = status
			}
			if reason, found, _ := unstructured.NestedString(condMap, "reason"); found {
				condition.Reason = reason
			}
			if message, found, _ := unstructured.NestedString(condMap, "message"); found {
				condition.Message = message
			}

			conditions = append(conditions, condition)
		}
	}

	return conditions
}

// parseErrorStatuses parses error statuses from slice
func (csv *ConfigSyncValidator) parseErrorStatuses(errorsObj []interface{}) []ErrorStatus {
	var errors []ErrorStatus

	for _, errObj := range errorsObj {
		if errMap, ok := errObj.(map[string]interface{}); ok {
			errorStatus := ErrorStatus{
				Timestamp: time.Now(),
			}

			if code, found, _ := unstructured.NestedString(errMap, "code"); found {
				errorStatus.Code = code
			}
			if desc, found, _ := unstructured.NestedString(errMap, "description"); found {
				errorStatus.Description = desc
			}

			errors = append(errors, errorStatus)
		}
	}

	return errors
}

// validateSyncedResources validates that expected resources are synced
func (csv *ConfigSyncValidator) validateSyncedResources(ctx context.Context, status *ConfigSyncStatus) error {
	resourceCount := 0
	errorCount := 0

	for _, expectedResource := range csv.Config.ExpectedSyncResources {
		// Parse resource identifier (format: apiVersion:kind:namespace:name)
		parts := strings.Split(expectedResource, ":")
		if len(parts) < 3 {
			continue
		}

		apiVersion := parts[0]
		kind := parts[1]
		namespace := ""
		name := ""

		if len(parts) >= 3 {
			if len(parts) == 3 {
				name = parts[2]
			} else {
				namespace = parts[2]
				name = parts[3]
			}
		}

		// Check if resource exists
		exists, err := csv.checkResourceExists(ctx, apiVersion, kind, namespace, name)
		if err != nil {
			errorCount++
			status.ValidationErrors = append(status.ValidationErrors, ValidationError{
				Resource:  expectedResource,
				Kind:      kind,
				Namespace: namespace,
				Name:      name,
				Error:     err.Error(),
				Timestamp: time.Now(),
			})
		} else if exists {
			resourceCount++
		}
	}

	status.SyncedResources = resourceCount
	status.ErroredResources = errorCount

	return nil
}

// checkResourceExists checks if a resource exists in the cluster
func (csv *ConfigSyncValidator) checkResourceExists(ctx context.Context, apiVersion, kind, namespace, name string) (bool, error) {
	// Convert apiVersion and kind to GVR
	gvr := csv.apiVersionKindToGVR(apiVersion, kind)

	var err error
	if namespace != "" {
		_, err = csv.ClusterClient.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		_, err = csv.ClusterClient.DynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// apiVersionKindToGVR converts apiVersion and kind to GroupVersionResource
func (csv *ConfigSyncValidator) apiVersionKindToGVR(apiVersion, kind string) schema.GroupVersionResource {
	parts := strings.Split(apiVersion, "/")
	var group, version string

	if len(parts) == 1 {
		group = ""
		version = parts[0]
	} else {
		group = parts[0]
		version = parts[1]
	}

	// Simple resource name derivation (should be enhanced for accuracy)
	resource := strings.ToLower(kind)
	if !strings.HasSuffix(resource, "s") {
		resource = resource + "s"
	}

	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
}

// calculateOverallSyncStatus calculates the overall sync status
func (csv *ConfigSyncValidator) calculateOverallSyncStatus(status *ConfigSyncStatus) string {
	if !status.Healthy {
		return "ERROR"
	}

	if status.RootSync != nil && status.RootSync.Sync.Status == "SYNCED" {
		allRepoSyncsSynced := true
		for _, rs := range status.RepoSyncs {
			if rs.Sync.Status != "SYNCED" {
				allRepoSyncsSynced = false
				break
			}
		}
		if allRepoSyncsSynced {
			return "SYNCED"
		}
	}

	return "SYNCING"
}