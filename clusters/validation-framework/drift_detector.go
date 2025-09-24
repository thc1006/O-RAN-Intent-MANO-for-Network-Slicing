// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// DriftDetector detects configuration drift between desired and actual state
type DriftDetector struct {
	Config        DriftDetectionConfig
	ClusterClient *ClusterClient
	GitRepo       *GitRepository
	BasePath      string
	validator     *security.FilePathValidator
	mutex         sync.RWMutex
	lastScan      time.Time
	driftCache    map[string]*DriftResult
}

// DriftResult represents the result of drift detection
type DriftResult struct {
	Resource     ResourceIdentifier     `json:"resource"`
	HasDrift     bool                   `json:"hasDrift"`
	DriftType    DriftType              `json:"driftType"`
	Changes      []FieldChange          `json:"changes"`
	Severity     DriftSeverity          `json:"severity"`
	DetectedAt   time.Time              `json:"detectedAt"`
	LastChecked  time.Time              `json:"lastChecked"`
	DesiredState map[string]interface{} `json:"desiredState,omitempty"`
	ActualState  map[string]interface{} `json:"actualState,omitempty"`
	Checksum     string                 `json:"checksum"`
}

// ResourceIdentifier uniquely identifies a Kubernetes resource
type ResourceIdentifier struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Source     string `json:"source"` // File path in Git repo
}

// DriftType represents the type of drift detected
type DriftType string

const (
	DriftTypeModified DriftType = "modified"
	DriftTypeAdded    DriftType = "added"
	DriftTypeDeleted  DriftType = "deleted"
	DriftTypeUnknown  DriftType = "unknown"
)

// DriftSeverity represents the severity of drift
type DriftSeverity string

const (
	DriftSeverityLow      DriftSeverity = "low"
	DriftSeverityMedium   DriftSeverity = "medium"
	DriftSeverityHigh     DriftSeverity = "high"
	DriftSeverityCritical DriftSeverity = "critical"
)

// FieldChange represents a change in a specific field
type FieldChange struct {
	Path         string      `json:"path"`
	DesiredValue interface{} `json:"desiredValue"`
	ActualValue  interface{} `json:"actualValue"`
	Action       string      `json:"action"` // added, removed, modified
}

// DriftScanResult represents the result of a complete drift scan
type DriftScanResult struct {
	ScanID           string        `json:"scanId"`
	Timestamp        time.Time     `json:"timestamp"`
	Duration         time.Duration `json:"duration"`
	TotalResources   int           `json:"totalResources"`
	DriftedResources int           `json:"driftedResources"`
	Results          []DriftResult `json:"results"`
	Summary          DriftSummary  `json:"summary"`
}

// DriftSummary provides a summary of drift detection results
type DriftSummary struct {
	BySeverity map[DriftSeverity]int `json:"bySeverity"`
	ByType     map[DriftType]int     `json:"byType"`
	ByResource map[string]int        `json:"byResource"`
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(config DriftDetectionConfig, client *ClusterClient, gitRepo *GitRepository, basePath string) *DriftDetector {
	// Create secure file path validator for Kubernetes files
	validator := security.CreateValidatorForKubernetes(basePath)

	return &DriftDetector{
		Config:        config,
		ClusterClient: client,
		GitRepo:       gitRepo,
		BasePath:      basePath,
		validator:     validator,
		driftCache:    make(map[string]*DriftResult),
	}
}

// ScanForDrift performs a comprehensive drift scan
func (dd *DriftDetector) ScanForDrift(ctx context.Context) (*DriftScanResult, error) {
	if !dd.Config.Enabled {
		return nil, fmt.Errorf("drift detection is disabled")
	}

	startTime := time.Now()
	scanID := fmt.Sprintf("drift-scan-%d", startTime.Unix())

	log.Printf("Starting drift scan %s", scanID)

	result := &DriftScanResult{
		ScanID:    scanID,
		Timestamp: startTime,
		Results:   make([]DriftResult, 0),
		Summary: DriftSummary{
			BySeverity: make(map[DriftSeverity]int),
			ByType:     make(map[DriftType]int),
			ByResource: make(map[string]int),
		},
	}

	// Get desired state from Git repository
	desiredResources, err := dd.getDesiredState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get desired state: %w", err)
	}

	// Get actual state from Kubernetes cluster
	actualResources, err := dd.getActualState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get actual state: %w", err)
	}

	// Compare states and detect drift
	driftResults := dd.compareStates(desiredResources, actualResources)

	// Update cache and results
	dd.mutex.Lock()
	dd.lastScan = startTime
	for _, drift := range driftResults {
		dd.driftCache[dd.getResourceKey(drift.Resource)] = &drift
		result.Results = append(result.Results, drift)

		if drift.HasDrift {
			result.DriftedResources++
			result.Summary.BySeverity[drift.Severity]++
			result.Summary.ByType[drift.DriftType]++
			result.Summary.ByResource[drift.Resource.Kind]++
		}
	}
	dd.mutex.Unlock()

	result.TotalResources = len(driftResults)
	result.Duration = time.Since(startTime)

	log.Printf("Drift scan %s completed: %d/%d resources drifted", scanID, result.DriftedResources, result.TotalResources)

	// Trigger remediation if configured
	if dd.Config.Remediation != "alert" && result.DriftedResources > 0 {
		if err := dd.remediateDrift(ctx, driftResults); err != nil {
			log.Printf("Drift remediation failed: %v", err)
		}
	}

	return result, nil
}

// getDesiredState retrieves the desired state from Git repository
func (dd *DriftDetector) getDesiredState(_ context.Context) (map[string]*unstructured.Unstructured, error) {
	desiredResources := make(map[string]*unstructured.Unstructured)

	// Walk through Git repository and find Kubernetes resource files
	err := filepath.Walk(dd.BasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if dd.isKubernetesResourceFile(path) {
			resources, err := dd.parseResourceFile(path)
			if err != nil {
				log.Printf("Warning: failed to parse resource file %s: %v", path, err)
				return nil
			}

			for _, resource := range resources {
				key := dd.getResourceKeyFromObject(resource)
				// Store relative path from base
				relPath, _ := filepath.Rel(dd.BasePath, path)
				resource.SetAnnotations(map[string]string{
					"gitops.oran.io/source-file": relPath,
				})
				desiredResources[key] = resource
			}
		}

		return nil
	})

	return desiredResources, err
}

// getActualState retrieves the actual state from Kubernetes cluster
func (dd *DriftDetector) getActualState(ctx context.Context) (map[string]*unstructured.Unstructured, error) {
	actualResources := make(map[string]*unstructured.Unstructured)

	// Get all API resources
	apiResources, err := dd.ClusterClient.Discovery.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get API resources: %w", err)
	}

	// Query each resource type
	for _, apiResourceList := range apiResources {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			if !dd.shouldMonitorResource(apiResource.Kind) {
				continue
			}

			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			resources, err := dd.listResources(ctx, gvr, apiResource.Namespaced)
			if err != nil {
				log.Printf("Warning: failed to list %s: %v", apiResource.Kind, err)
				continue
			}

			for _, resource := range resources {
				key := dd.getResourceKeyFromObject(resource)
				actualResources[key] = resource
			}
		}
	}

	return actualResources, nil
}

// listResources lists resources of a specific type
func (dd *DriftDetector) listResources(ctx context.Context, gvr schema.GroupVersionResource, namespaced bool) ([]*unstructured.Unstructured, error) {
	var resources []*unstructured.Unstructured

	if namespaced {
		// List resources in all namespaces
		namespaces, err := dd.ClusterClient.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, ns := range namespaces.Items {
			list, err := dd.ClusterClient.DynamicClient.Resource(gvr).Namespace(ns.Name).List(ctx, metav1.ListOptions{})
			if errors.IsNotFound(err) || errors.IsForbidden(err) {
				continue
			}
			if err != nil {
				return nil, err
			}

			for _, item := range list.Items {
				resources = append(resources, &item)
			}
		}
	} else {
		// List cluster-scoped resources
		list, err := dd.ClusterClient.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
		if errors.IsNotFound(err) || errors.IsForbidden(err) {
			return resources, nil
		}
		if err != nil {
			return nil, err
		}

		for _, item := range list.Items {
			resources = append(resources, &item)
		}
	}

	return resources, nil
}

// shouldMonitorResource determines if a resource type should be monitored for drift
func (dd *DriftDetector) shouldMonitorResource(kind string) bool {
	// Monitor core O-RAN resources and common Kubernetes resources
	monitoredResources := []string{
		"Deployment", "Service", "ConfigMap", "Secret", "ServiceAccount",
		"ClusterRole", "ClusterRoleBinding", "Role", "RoleBinding",
		"PersistentVolume", "PersistentVolumeClaim", "StorageClass",
		"Ingress", "NetworkPolicy", "VirtualService", "DestinationRule",
		"VNF", "WorkloadCluster", "PackageRevision", "Repository",
	}

	for _, monitored := range monitoredResources {
		if kind == monitored {
			return true
		}
	}

	return false
}

// compareStates compares desired and actual states to detect drift
func (dd *DriftDetector) compareStates(desired, actual map[string]*unstructured.Unstructured) []DriftResult {
	var results []DriftResult
	now := time.Now()

	// Check for modified or deleted resources
	for key, desiredResource := range desired {
		actualResource, exists := actual[key]

		result := DriftResult{
			Resource: ResourceIdentifier{
				APIVersion: desiredResource.GetAPIVersion(),
				Kind:       desiredResource.GetKind(),
				Name:       desiredResource.GetName(),
				Namespace:  desiredResource.GetNamespace(),
				Source:     desiredResource.GetAnnotations()["gitops.oran.io/source-file"],
			},
			DetectedAt:  now,
			LastChecked: now,
		}

		if !exists {
			// Resource exists in desired but not in actual - deleted
			result.HasDrift = true
			result.DriftType = DriftTypeDeleted
			result.Severity = dd.calculateSeverity(result.Resource, DriftTypeDeleted, nil)
			result.DesiredState = desiredResource.Object
		} else {
			// Resource exists in both - check for modifications
			changes := dd.compareResources(desiredResource, actualResource)
			if len(changes) > 0 {
				result.HasDrift = true
				result.DriftType = DriftTypeModified
				result.Changes = changes
				result.Severity = dd.calculateSeverity(result.Resource, DriftTypeModified, changes)
				result.DesiredState = desiredResource.Object
				result.ActualState = actualResource.Object
			}
		}

		// Calculate checksum for change tracking
		result.Checksum = dd.calculateChecksum(result)
		results = append(results, result)
	}

	// Check for added resources (exist in actual but not in desired)
	for key, actualResource := range actual {
		if _, exists := desired[key]; !exists {
			// Skip resources that are not managed by GitOps
			if !dd.isManagedResource(actualResource) {
				continue
			}

			result := DriftResult{
				Resource: ResourceIdentifier{
					APIVersion: actualResource.GetAPIVersion(),
					Kind:       actualResource.GetKind(),
					Name:       actualResource.GetName(),
					Namespace:  actualResource.GetNamespace(),
				},
				HasDrift:    true,
				DriftType:   DriftTypeAdded,
				DetectedAt:  now,
				LastChecked: now,
				ActualState: actualResource.Object,
			}
			result.Severity = dd.calculateSeverity(result.Resource, DriftTypeAdded, nil)
			result.Checksum = dd.calculateChecksum(result)
			results = append(results, result)
		}
	}

	return results
}

// compareResources compares two resources and returns field changes
func (dd *DriftDetector) compareResources(desired, actual *unstructured.Unstructured) []FieldChange {
	var changes []FieldChange

	// Remove fields that should be ignored
	desiredFiltered := dd.filterIgnoredFields(desired.Object)
	actualFiltered := dd.filterIgnoredFields(actual.Object)

	// Compare spec section (most important for drift detection)
	if desiredSpec, exists := desiredFiltered["spec"]; exists {
		if actualSpec, exists := actualFiltered["spec"]; exists {
			specChanges := dd.compareObjects("spec", desiredSpec, actualSpec)
			changes = append(changes, specChanges...)
		}
	}

	// Compare metadata labels and annotations
	if desiredMeta, exists := desiredFiltered["metadata"]; exists {
		if actualMeta, exists := actualFiltered["metadata"]; exists {
			metaChanges := dd.compareMetadata(desiredMeta, actualMeta)
			changes = append(changes, metaChanges...)
		}
	}

	return changes
}

// compareObjects recursively compares two objects
func (dd *DriftDetector) compareObjects(basePath string, desired, actual interface{}) []FieldChange {
	var changes []FieldChange

	if reflect.TypeOf(desired) != reflect.TypeOf(actual) {
		return []FieldChange{{
			Path:         basePath,
			DesiredValue: desired,
			ActualValue:  actual,
			Action:       "modified",
		}}
	}

	switch d := desired.(type) {
	case map[string]interface{}:
		a := actual.(map[string]interface{})

		// Check for added/modified fields
		for key, dValue := range d {
			path := fmt.Sprintf("%s.%s", basePath, key)
			if aValue, exists := a[key]; exists {
				subChanges := dd.compareObjects(path, dValue, aValue)
				changes = append(changes, subChanges...)
			} else {
				changes = append(changes, FieldChange{
					Path:         path,
					DesiredValue: dValue,
					Action:       "added",
				})
			}
		}

		// Check for removed fields
		for key, aValue := range a {
			if _, exists := d[key]; !exists {
				path := fmt.Sprintf("%s.%s", basePath, key)
				changes = append(changes, FieldChange{
					Path:        path,
					ActualValue: aValue,
					Action:      "removed",
				})
			}
		}

	case []interface{}:
		a := actual.([]interface{})
		if len(d) != len(a) {
			return []FieldChange{{
				Path:         basePath,
				DesiredValue: desired,
				ActualValue:  actual,
				Action:       "modified",
			}}
		}

		for i, dItem := range d {
			if i < len(a) {
				path := fmt.Sprintf("%s[%d]", basePath, i)
				subChanges := dd.compareObjects(path, dItem, a[i])
				changes = append(changes, subChanges...)
			}
		}

	default:
		if !reflect.DeepEqual(desired, actual) {
			changes = append(changes, FieldChange{
				Path:         basePath,
				DesiredValue: desired,
				ActualValue:  actual,
				Action:       "modified",
			})
		}
	}

	return changes
}

// compareMetadata compares metadata sections
func (dd *DriftDetector) compareMetadata(desired, actual interface{}) []FieldChange {
	var changes []FieldChange

	dMeta, ok1 := desired.(map[string]interface{})
	aMeta, ok2 := actual.(map[string]interface{})

	if !ok1 || !ok2 {
		return changes
	}

	// Compare labels
	if dLabels, exists := dMeta["labels"]; exists {
		if aLabels, exists := aMeta["labels"]; exists {
			labelChanges := dd.compareObjects("metadata.labels", dLabels, aLabels)
			changes = append(changes, labelChanges...)
		}
	}

	// Compare annotations (excluding system annotations)
	if dAnnotations, exists := dMeta["annotations"]; exists {
		if aAnnotations, exists := aMeta["annotations"]; exists {
			filteredDAnnotations := dd.filterSystemAnnotations(dAnnotations)
			filteredAAnnotations := dd.filterSystemAnnotations(aAnnotations)
			annotationChanges := dd.compareObjects("metadata.annotations", filteredDAnnotations, filteredAAnnotations)
			changes = append(changes, annotationChanges...)
		}
	}

	return changes
}

// filterIgnoredFields removes fields that should be ignored during drift detection
func (dd *DriftDetector) filterIgnoredFields(obj map[string]interface{}) map[string]interface{} {
	filtered := make(map[string]interface{})

	for key, value := range obj {
		// Skip status and other runtime fields
		if dd.shouldIgnoreField(key) {
			continue
		}
		filtered[key] = value
	}

	// Filter metadata
	if metadata, exists := obj["metadata"]; exists {
		if metaMap, ok := metadata.(map[string]interface{}); ok {
			filteredMeta := make(map[string]interface{})
			for key, value := range metaMap {
				if !dd.shouldIgnoreMetadataField(key) {
					filteredMeta[key] = value
				}
			}
			filtered["metadata"] = filteredMeta
		}
	}

	return filtered
}

// shouldIgnoreField determines if a field should be ignored
func (dd *DriftDetector) shouldIgnoreField(field string) bool {
	ignoredFields := []string{
		"status", "metadata.resourceVersion", "metadata.generation",
		"metadata.managedFields", "metadata.uid", "metadata.creationTimestamp",
		"metadata.selfLink", "metadata.deletionTimestamp", "metadata.deletionGracePeriodSeconds",
	}

	for _, ignored := range ignoredFields {
		if field == ignored {
			return true
		}
	}

	// Check configured ignore fields
	for _, configIgnored := range dd.Config.IgnoreFields {
		if field == configIgnored {
			return true
		}
	}

	return false
}

// shouldIgnoreMetadataField determines if a metadata field should be ignored
func (dd *DriftDetector) shouldIgnoreMetadataField(field string) bool {
	ignoredMetadataFields := []string{
		"resourceVersion", "generation", "managedFields", "uid",
		"creationTimestamp", "selfLink", "deletionTimestamp",
		"deletionGracePeriodSeconds",
	}

	for _, ignored := range ignoredMetadataFields {
		if field == ignored {
			return true
		}
	}

	return false
}

// filterSystemAnnotations filters out system-managed annotations
func (dd *DriftDetector) filterSystemAnnotations(annotations interface{}) interface{} {
	if annotMap, ok := annotations.(map[string]interface{}); ok {
		filtered := make(map[string]interface{})
		for key, value := range annotMap {
			// Keep only user-managed annotations
			if !strings.HasPrefix(key, "kubectl.kubernetes.io/") &&
				!strings.HasPrefix(key, "deployment.kubernetes.io/") &&
				!strings.HasPrefix(key, "pv.kubernetes.io/") {
				filtered[key] = value
			}
		}
		return filtered
	}
	return annotations
}

// isManagedResource determines if a resource is managed by GitOps
func (dd *DriftDetector) isManagedResource(resource *unstructured.Unstructured) bool {
	annotations := resource.GetAnnotations()
	if annotations == nil {
		return false
	}

	// Check for GitOps annotations
	gitopsAnnotations := []string{
		"config.kubernetes.io/local-config",
		"config.k8s.io/local-config",
		"argocd.argoproj.io/tracking-id",
		"configsync.gke.io/resource-id",
		"gitops.oran.io/managed",
	}

	for _, annotation := range gitopsAnnotations {
		if _, exists := annotations[annotation]; exists {
			return true
		}
	}

	// Check for managed labels
	labels := resource.GetLabels()
	if labels != nil {
		if _, exists := labels["app.kubernetes.io/managed-by"]; exists {
			return true
		}
	}

	return false
}

// calculateSeverity calculates the severity of drift
func (dd *DriftDetector) calculateSeverity(resource ResourceIdentifier, driftType DriftType, changes []FieldChange) DriftSeverity {
	// Critical resources
	if dd.isCriticalResource(resource) {
		return DriftSeverityCritical
	}

	// Check drift type
	switch driftType {
	case DriftTypeDeleted:
		return DriftSeverityHigh
	case DriftTypeAdded:
		return DriftSeverityMedium
	case DriftTypeModified:
		return dd.calculateModificationSeverity(changes)
	}

	return DriftSeverityLow
}

// isCriticalResource determines if a resource is critical
func (dd *DriftDetector) isCriticalResource(resource ResourceIdentifier) bool {
	criticalResources := []string{
		"Deployment", "Service", "Secret", "ClusterRole", "ClusterRoleBinding",
	}

	for _, critical := range criticalResources {
		if resource.Kind == critical {
			return true
		}
	}

	// Check if it's an O-RAN component
	if strings.Contains(resource.Name, "ran-") ||
		strings.Contains(resource.Name, "cn-") ||
		strings.Contains(resource.Name, "tn-") ||
		strings.Contains(resource.Name, "orchestrator") {
		return true
	}

	return false
}

// calculateModificationSeverity calculates severity for modifications
func (dd *DriftDetector) calculateModificationSeverity(changes []FieldChange) DriftSeverity {
	if len(changes) == 0 {
		return DriftSeverityLow
	}

	highImpactFields := []string{
		"spec.replicas", "spec.image", "spec.ports", "spec.env",
		"spec.resources", "spec.nodeSelector", "spec.tolerations",
	}

	for _, change := range changes {
		for _, highImpact := range highImpactFields {
			if strings.Contains(change.Path, highImpact) {
				return DriftSeverityHigh
			}
		}
	}

	if len(changes) > 10 {
		return DriftSeverityMedium
	}

	return DriftSeverityLow
}

// calculateChecksum calculates a checksum for the drift result
func (dd *DriftDetector) calculateChecksum(result DriftResult) string {
	data := fmt.Sprintf("%s-%s-%s-%v",
		result.Resource.APIVersion,
		result.Resource.Kind,
		result.Resource.Name,
		result.HasDrift)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // First 8 bytes
}

// remediateDrift performs drift remediation
func (dd *DriftDetector) remediateDrift(ctx context.Context, driftResults []DriftResult) error {
	switch dd.Config.Remediation {
	case "correct":
		return dd.correctDrift(ctx, driftResults)
	case "rollback":
		return dd.triggerRollback(ctx, driftResults)
	default:
		return nil // No remediation
	}
}

// correctDrift corrects detected drift by applying desired state
func (dd *DriftDetector) correctDrift(ctx context.Context, driftResults []DriftResult) error {
	log.Printf("Correcting drift for %d resources", len(driftResults))

	var correctionErrors []string
	for _, drift := range driftResults {
		if !drift.HasDrift {
			continue
		}

		if err := dd.correctResourceDrift(ctx, drift); err != nil {
			errorMsg := fmt.Sprintf("Failed to correct drift for %s/%s: %v", drift.Resource.Kind, drift.Resource.Name, err)
			log.Printf("%s", errorMsg)
			correctionErrors = append(correctionErrors, errorMsg)
		}
	}

	if len(correctionErrors) > 0 {
		return fmt.Errorf("drift correction failed for some resources: %s", strings.Join(correctionErrors, "; "))
	}

	return nil
}

// correctResourceDrift corrects drift for a single resource
func (dd *DriftDetector) correctResourceDrift(ctx context.Context, drift DriftResult) error {
	switch drift.DriftType {
	case DriftTypeModified:
		return dd.updateResource(ctx, drift)
	case DriftTypeDeleted:
		return dd.createResource(ctx, drift)
	case DriftTypeAdded:
		return dd.deleteResource(ctx, drift)
	}

	return nil
}

// updateResource updates a resource to match desired state
func (dd *DriftDetector) updateResource(ctx context.Context, drift DriftResult) error {
	if drift.DesiredState == nil {
		return fmt.Errorf("no desired state available")
	}

	gvr := dd.getGVR(drift.Resource.APIVersion, drift.Resource.Kind)
	obj := &unstructured.Unstructured{Object: drift.DesiredState}

	if drift.Resource.Namespace != "" {
		_, err := dd.ClusterClient.DynamicClient.Resource(gvr).Namespace(drift.Resource.Namespace).Update(ctx, obj, metav1.UpdateOptions{})
		return err
	}

	_, err := dd.ClusterClient.DynamicClient.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
	return err
}

// createResource creates a missing resource
func (dd *DriftDetector) createResource(ctx context.Context, drift DriftResult) error {
	if drift.DesiredState == nil {
		return fmt.Errorf("no desired state available")
	}

	gvr := dd.getGVR(drift.Resource.APIVersion, drift.Resource.Kind)
	obj := &unstructured.Unstructured{Object: drift.DesiredState}

	if drift.Resource.Namespace != "" {
		_, err := dd.ClusterClient.DynamicClient.Resource(gvr).Namespace(drift.Resource.Namespace).Create(ctx, obj, metav1.CreateOptions{})
		return err
	}

	_, err := dd.ClusterClient.DynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	return err
}

// deleteResource deletes an unexpected resource
func (dd *DriftDetector) deleteResource(ctx context.Context, drift DriftResult) error {
	gvr := dd.getGVR(drift.Resource.APIVersion, drift.Resource.Kind)

	if drift.Resource.Namespace != "" {
		return dd.ClusterClient.DynamicClient.Resource(gvr).Namespace(drift.Resource.Namespace).Delete(ctx, drift.Resource.Name, metav1.DeleteOptions{})
	}

	return dd.ClusterClient.DynamicClient.Resource(gvr).Delete(ctx, drift.Resource.Name, metav1.DeleteOptions{})
}

// triggerRollback triggers a rollback due to drift
func (dd *DriftDetector) triggerRollback(ctx context.Context, driftResults []DriftResult) error {
	log.Printf("Triggering rollback due to drift detection")
	// This would integrate with the rollback manager
	return fmt.Errorf("rollback remediation not implemented")
}

// getGVR converts apiVersion and kind to GroupVersionResource
func (dd *DriftDetector) getGVR(apiVersion, kind string) schema.GroupVersionResource {
	parts := strings.Split(apiVersion, "/")
	var group, version string

	if len(parts) == 1 {
		group = ""
		version = parts[0]
	} else {
		group = parts[0]
		version = parts[1]
	}

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

// GetDriftStatus returns the current drift status
func (dd *DriftDetector) GetDriftStatus() map[string]*DriftResult {
	dd.mutex.RLock()
	defer dd.mutex.RUnlock()

	result := make(map[string]*DriftResult)
	for key, drift := range dd.driftCache {
		result[key] = drift
	}

	return result
}

// getResourceKey generates a unique key for a resource
func (dd *DriftDetector) getResourceKey(resource ResourceIdentifier) string {
	return fmt.Sprintf("%s/%s/%s/%s", resource.APIVersion, resource.Kind, resource.Namespace, resource.Name)
}

// getResourceKeyFromObject generates a resource key from an unstructured object
func (dd *DriftDetector) getResourceKeyFromObject(obj *unstructured.Unstructured) string {
	return fmt.Sprintf("%s/%s/%s/%s", obj.GetAPIVersion(), obj.GetKind(), obj.GetNamespace(), obj.GetName())
}

// isKubernetesResourceFile checks if a file contains Kubernetes resources
func (dd *DriftDetector) isKubernetesResourceFile(filename string) bool {
	return strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")
}

// parseResourceFile parses Kubernetes resources from a file
func (dd *DriftDetector) parseResourceFile(filename string) ([]*unstructured.Unstructured, error) {
	// Validate file path for security
	if err := dd.validator.ValidateFilePathAndExtension(filename, []string{".yaml", ".yml"}); err != nil {
		return nil, fmt.Errorf("file path validation failed: %w", err)
	}

	data, err := dd.validator.SafeReadFile(filename)
	if err != nil {
		return nil, err
	}

	var resources []*unstructured.Unstructured
	docs := strings.Split(string(data), "---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var resource unstructured.Unstructured
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			continue
		}

		if resource.GetAPIVersion() != "" && resource.GetKind() != "" {
			resources = append(resources, &resource)
		}
	}

	return resources, nil
}
