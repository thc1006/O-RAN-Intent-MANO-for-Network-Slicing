// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SyncManager manages multi-cluster package synchronization
type SyncManager struct {
	Config           SyncConfig
	ClusterClients   map[string]*ClusterClient
	PackageValidator *NephioValidator
	GitRepo          *GitRepository
	mutex            sync.RWMutex
	syncStatus       map[string]*PackageSyncStatus
}

// SyncConfig holds synchronization configuration
type SyncConfig struct {
	Enabled           bool                    `yaml:"enabled"`
	SyncInterval      time.Duration           `yaml:"syncInterval"`
	MaxRetries        int                     `yaml:"maxRetries"`
	RetryBackoff      time.Duration           `yaml:"retryBackoff"`
	ConflictStrategy  ConflictStrategy        `yaml:"conflictStrategy"`
	PackageGroups     []PackageGroup          `yaml:"packageGroups"`
	Dependencies      []PackageDependency     `yaml:"dependencies"`
	HealthChecks      []HealthCheck           `yaml:"healthChecks"`
}

// ConflictStrategy defines how to handle sync conflicts
type ConflictStrategy string

const (
	ConflictStrategyGitWins     ConflictStrategy = "git-wins"
	ConflictStrategyClusterWins ConflictStrategy = "cluster-wins"
	ConflictStrategyManual      ConflictStrategy = "manual"
	ConflictStrategyMerge       ConflictStrategy = "merge"
)

// PackageGroup defines a group of packages that should be synchronized together
type PackageGroup struct {
	Name         string   `yaml:"name"`
	Packages     []string `yaml:"packages"`
	Clusters     []string `yaml:"clusters"`
	Priority     int      `yaml:"priority"`
	Sequential   bool     `yaml:"sequential"`
	Dependencies []string `yaml:"dependencies"`
}

// PackageDependency defines dependencies between packages
type PackageDependency struct {
	Package      string   `yaml:"package"`
	DependsOn    []string `yaml:"dependsOn"`
	WaitTimeout  time.Duration `yaml:"waitTimeout"`
}

// HealthCheck defines health checks for synchronized packages
type HealthCheck struct {
	Name        string        `yaml:"name"`
	Package     string        `yaml:"package"`
	Type        string        `yaml:"type"` // endpoint, resource, command
	Target      string        `yaml:"target"`
	Interval    time.Duration `yaml:"interval"`
	Timeout     time.Duration `yaml:"timeout"`
	Retries     int           `yaml:"retries"`
}

// PackageSyncStatus represents the synchronization status of a package/cluster
type PackageSyncStatus struct {
	Package       string                `json:"package"`
	Cluster       string                `json:"cluster"`
	Status        SyncState             `json:"status"`
	LastSync      time.Time             `json:"lastSync"`
	LastSuccess   time.Time             `json:"lastSuccess"`
	Version       string                `json:"version"`
	Errors        []SyncError           `json:"errors,omitempty"`
	RetryCount    int                   `json:"retryCount"`
	NextRetry     time.Time             `json:"nextRetry,omitempty"`
	Dependencies  []DependencyStatus    `json:"dependencies,omitempty"`
	HealthStatus  PackageHealthStatus   `json:"healthStatus"`
}

// SyncState represents the state of synchronization
type SyncState string

const (
	SyncStateUnknown      SyncState = "unknown"
	SyncStatePending      SyncState = "pending"
	SyncStateInProgress   SyncState = "in_progress"
	SyncStateSynced       SyncState = "synced"
	SyncStateFailed       SyncState = "failed"
	SyncStateConflict     SyncState = "conflict"
	SyncStateWaiting      SyncState = "waiting"
)

// SyncError represents a synchronization error
type SyncError struct {
	Message   string    `json:"message"`
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	Resource  string    `json:"resource,omitempty"`
}

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Package   string    `json:"package"`
	Status    SyncState `json:"status"`
	Required  bool      `json:"required"`
	Satisfied bool      `json:"satisfied"`
}

// HealthStatus represents health check status
type PackageHealthStatus struct {
	Status      string    `json:"status"`
	LastCheck   time.Time `json:"lastCheck"`
	CheckCount  int       `json:"checkCount"`
	FailCount   int       `json:"failCount"`
	Message     string    `json:"message,omitempty"`
}

// SyncOperationResult represents the result of a synchronization operation
type SyncOperationResult struct {
	SyncID       string                `json:"syncId"`
	Timestamp    time.Time             `json:"timestamp"`
	Duration     time.Duration         `json:"duration"`
	Success      bool                  `json:"success"`
	PackagesSynced int                 `json:"packagesSynced"`
	PackagesFailed int                 `json:"packagesFailed"`
	Results      []PackageSyncResult   `json:"results"`
	Conflicts    []SyncConflict        `json:"conflicts,omitempty"`
}

// PackageSyncResult represents the sync result for a single package
type PackageSyncResult struct {
	Package   string        `json:"package"`
	Cluster   string        `json:"cluster"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	Version   string        `json:"version"`
	Actions   []SyncAction  `json:"actions"`
	Errors    []string      `json:"errors,omitempty"`
}

// SyncAction represents an action taken during sync
type SyncAction struct {
	Type        string    `json:"type"`        // create, update, delete, skip
	Resource    string    `json:"resource"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

// SyncConflict represents a synchronization conflict
type SyncConflict struct {
	Package      string                 `json:"package"`
	Cluster      string                 `json:"cluster"`
	Resource     string                 `json:"resource"`
	ConflictType string                 `json:"conflictType"`
	GitVersion   map[string]interface{} `json:"gitVersion"`
	ClusterVersion map[string]interface{} `json:"clusterVersion"`
	Resolution   string                 `json:"resolution,omitempty"`
}

// NewSyncManager creates a new sync manager
func NewSyncManager(config SyncConfig, clusterClients map[string]*ClusterClient, packageValidator *NephioValidator, gitRepo *GitRepository) *SyncManager {
	return &SyncManager{
		Config:           config,
		ClusterClients:   clusterClients,
		PackageValidator: packageValidator,
		GitRepo:          gitRepo,
		syncStatus:       make(map[string]*PackageSyncStatus),
	}
}

// SynchronizeAll synchronizes all packages across all clusters
func (sm *SyncManager) SynchronizeAll(ctx context.Context) (*SyncOperationResult, error) {
	if !sm.Config.Enabled {
		return nil, fmt.Errorf("synchronization is disabled")
	}

	startTime := time.Now()
	syncID := fmt.Sprintf("sync-%d", startTime.Unix())

	log.Printf("Starting package synchronization %s", syncID)

	result := &SyncOperationResult{
		SyncID:    syncID,
		Timestamp: startTime,
		Results:   make([]PackageSyncResult, 0),
		Conflicts: make([]SyncConflict, 0),
	}

	// Process package groups in priority order
	groups := sm.getSortedPackageGroups()

	for _, group := range groups {
		groupResult, err := sm.synchronizePackageGroup(ctx, group)
		if err != nil {
			log.Printf("Failed to synchronize package group %s: %v", group.Name, err)
		}

		result.Results = append(result.Results, groupResult...)
	}

	// Calculate final status
	for _, pkgResult := range result.Results {
		if pkgResult.Success {
			result.PackagesSynced++
		} else {
			result.PackagesFailed++
		}
	}

	result.Success = result.PackagesFailed == 0
	result.Duration = time.Since(startTime)

	log.Printf("Package synchronization %s completed: %d succeeded, %d failed",
		syncID, result.PackagesSynced, result.PackagesFailed)

	return result, nil
}

// synchronizePackageGroup synchronizes packages in a group
func (sm *SyncManager) synchronizePackageGroup(ctx context.Context, group PackageGroup) ([]PackageSyncResult, error) {
	var results []PackageSyncResult

	log.Printf("Synchronizing package group %s (priority: %d)", group.Name, group.Priority)

	// Check group dependencies
	if err := sm.waitForGroupDependencies(ctx, group); err != nil {
		return results, fmt.Errorf("group dependencies not satisfied: %w", err)
	}

	if group.Sequential {
		// Synchronize packages sequentially
		for _, packageName := range group.Packages {
			pkgResults, err := sm.synchronizePackage(ctx, packageName, group.Clusters)
			if err != nil {
				log.Printf("Failed to synchronize package %s: %v", packageName, err)
				// Continue with next package
			}
			results = append(results, pkgResults...)
		}
	} else {
		// Synchronize packages in parallel
		resultsChan := make(chan []PackageSyncResult, len(group.Packages))
		var wg sync.WaitGroup

		for _, packageName := range group.Packages {
			wg.Add(1)
			go func(pkg string) {
				defer wg.Done()
				pkgResults, err := sm.synchronizePackage(ctx, pkg, group.Clusters)
				if err != nil {
					log.Printf("Failed to synchronize package %s: %v", pkg, err)
				}
				resultsChan <- pkgResults
			}(packageName)
		}

		wg.Wait()
		close(resultsChan)

		for pkgResults := range resultsChan {
			results = append(results, pkgResults...)
		}
	}

	return results, nil
}

// synchronizePackage synchronizes a package across specified clusters
func (sm *SyncManager) synchronizePackage(ctx context.Context, packageName string, clusters []string) ([]PackageSyncResult, error) {
	var results []PackageSyncResult

	log.Printf("Synchronizing package %s across %d clusters", packageName, len(clusters))

	// Check package dependencies
	if err := sm.waitForPackageDependencies(ctx, packageName); err != nil {
		return results, fmt.Errorf("package dependencies not satisfied: %w", err)
	}

	// Get package content from Git
	packageContent, err := sm.getPackageFromGit(ctx, packageName)
	if err != nil {
		return results, fmt.Errorf("failed to get package from Git: %w", err)
	}

	// Validate package
	if sm.PackageValidator != nil {
		if err := sm.PackageValidator.ValidatePackage(ctx, packageName); err != nil {
			return results, fmt.Errorf("package validation failed: %w", err)
		}
	}

	// Synchronize to each cluster
	for _, clusterName := range clusters {
		result := sm.synchronizePackageToCluster(ctx, packageName, clusterName, packageContent)
		results = append(results, result)
	}

	return results, nil
}

// synchronizePackageToCluster synchronizes a package to a specific cluster
func (sm *SyncManager) synchronizePackageToCluster(ctx context.Context, packageName, clusterName string, packageContent []unstructured.Unstructured) PackageSyncResult {
	startTime := time.Now()

	result := PackageSyncResult{
		Package:   packageName,
		Cluster:   clusterName,
		Actions:   make([]SyncAction, 0),
		Errors:    make([]string, 0),
	}

	client, exists := sm.ClusterClients[clusterName]
	if !exists {
		result.Errors = append(result.Errors, fmt.Sprintf("cluster client not found: %q", clusterName))
		result.Duration = time.Since(startTime)
		return result
	}

	log.Printf("Synchronizing package %s to cluster %s", packageName, clusterName)

	// Update sync status
	sm.updateSyncStatus(packageName, clusterName, SyncStateInProgress, "")

	// Process each resource in the package
	for _, resource := range packageContent {
		action, err := sm.syncResource(ctx, client, &resource)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to sync resource %s: %v", resource.GetName(), err))
			continue
		}

		result.Actions = append(result.Actions, SyncAction{
			Type:      action,
			Resource:  fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()),
			Timestamp: time.Now(),
		})
	}

	// Determine result status
	result.Success = len(result.Errors) == 0
	result.Duration = time.Since(startTime)

	// Update final sync status
	if result.Success {
		sm.updateSyncStatus(packageName, clusterName, SyncStateSynced, "")
		log.Printf("Successfully synchronized package %s to cluster %s", packageName, clusterName)
	} else {
		errorMsg := fmt.Sprintf("Sync failed with %d errors", len(result.Errors))
		sm.updateSyncStatus(packageName, clusterName, SyncStateFailed, errorMsg)
		log.Printf("Failed to synchronize package %s to cluster %s: %s", packageName, clusterName, errorMsg)
	}

	return result
}

// syncResource synchronizes a single resource
func (sm *SyncManager) syncResource(ctx context.Context, client *ClusterClient, resource *unstructured.Unstructured) (string, error) {
	gvr := sm.getGVRFromResource(resource)

	// Check if resource exists in cluster
	var existing *unstructured.Unstructured
	var err error

	if resource.GetNamespace() != "" {
		existing, err = client.DynamicClient.Resource(gvr).Namespace(resource.GetNamespace()).Get(ctx, resource.GetName(), metav1.GetOptions{})
	} else {
		existing, err = client.DynamicClient.Resource(gvr).Get(ctx, resource.GetName(), metav1.GetOptions{})
	}

	if errors.IsNotFound(err) {
		// Resource doesn't exist - create it
		return sm.createResource(ctx, client, gvr, resource)
	} else if err != nil {
		return "", fmt.Errorf("failed to get existing resource: %w", err)
	}

	// Resource exists - check if update is needed
	if sm.resourceNeedsUpdate(existing, resource) {
		return sm.updateResource(ctx, client, gvr, resource, existing)
	}

	return "skip", nil
}

// createResource creates a new resource
func (sm *SyncManager) createResource(ctx context.Context, client *ClusterClient, gvr schema.GroupVersionResource, resource *unstructured.Unstructured) (string, error) {
	if resource.GetNamespace() != "" {
		_, err := client.DynamicClient.Resource(gvr).Namespace(resource.GetNamespace()).Create(ctx, resource, metav1.CreateOptions{})
		return "create", err
	}

	_, err := client.DynamicClient.Resource(gvr).Create(ctx, resource, metav1.CreateOptions{})
	return "create", err
}

// updateResource updates an existing resource
func (sm *SyncManager) updateResource(ctx context.Context, client *ClusterClient, gvr schema.GroupVersionResource, desired, existing *unstructured.Unstructured) (string, error) {
	// Handle conflicts based on strategy
	if sm.hasConflict(existing, desired) {
		resolved, err := sm.resolveConflict(existing, desired)
		if err != nil {
			return "", fmt.Errorf("failed to resolve conflict: %w", err)
		}
		desired = resolved
	}

	// Preserve certain fields from existing resource
	sm.preserveClusterFields(desired, existing)

	if desired.GetNamespace() != "" {
		_, err := client.DynamicClient.Resource(gvr).Namespace(desired.GetNamespace()).Update(ctx, desired, metav1.UpdateOptions{})
		return "update", err
	}

	_, err := client.DynamicClient.Resource(gvr).Update(ctx, desired, metav1.UpdateOptions{})
	return "update", err
}

// resourceNeedsUpdate determines if a resource needs to be updated
func (sm *SyncManager) resourceNeedsUpdate(existing, desired *unstructured.Unstructured) bool {
	// Compare spec sections
	existingSpec, existsInExisting, _ := unstructured.NestedMap(existing.Object, "spec")
	desiredSpec, existsInDesired, _ := unstructured.NestedMap(desired.Object, "spec")

	if existsInExisting != existsInDesired {
		return true
	}

	if existsInExisting && existsInDesired {
		return !sm.deepEqual(existingSpec, desiredSpec)
	}

	// Compare labels and annotations
	if !sm.deepEqual(existing.GetLabels(), desired.GetLabels()) {
		return true
	}

	// Filter out system annotations for comparison
	existingAnnotations := sm.filterSystemAnnotations(existing.GetAnnotations())
	desiredAnnotations := sm.filterSystemAnnotations(desired.GetAnnotations())

	return !sm.deepEqual(existingAnnotations, desiredAnnotations)
}

// hasConflict checks if there's a conflict between existing and desired state
func (sm *SyncManager) hasConflict(existing, desired *unstructured.Unstructured) bool {
	// For now, consider any difference as a potential conflict
	// In a real implementation, this would be more sophisticated
	return sm.resourceNeedsUpdate(existing, desired)
}

// resolveConflict resolves conflicts based on the configured strategy
func (sm *SyncManager) resolveConflict(existing, desired *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	switch sm.Config.ConflictStrategy {
	case ConflictStrategyGitWins:
		return desired, nil
	case ConflictStrategyClusterWins:
		return existing, nil
	case ConflictStrategyMerge:
		return sm.mergeResources(existing, desired)
	case ConflictStrategyManual:
		return nil, fmt.Errorf("manual conflict resolution required")
	default:
		return desired, nil
	}
}

// mergeResources merges two resources
func (sm *SyncManager) mergeResources(existing, desired *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Simple merge strategy - take desired spec but preserve existing metadata
	merged := desired.DeepCopy()

	// Preserve certain metadata fields
	merged.SetResourceVersion(existing.GetResourceVersion())
	merged.SetUID(existing.GetUID())
	merged.SetCreationTimestamp(existing.GetCreationTimestamp())
	merged.SetGeneration(existing.GetGeneration())

	return merged, nil
}

// preserveClusterFields preserves certain cluster-managed fields
func (sm *SyncManager) preserveClusterFields(desired, existing *unstructured.Unstructured) {
	desired.SetResourceVersion(existing.GetResourceVersion())
	desired.SetUID(existing.GetUID())
	desired.SetCreationTimestamp(existing.GetCreationTimestamp())

	// Preserve managed fields
	if managedFields := existing.GetManagedFields(); managedFields != nil {
		desired.SetManagedFields(managedFields)
	}
}

// getPackageFromGit retrieves package content from Git repository
// TODO: Implement actual Git package retrieval logic
func (sm *SyncManager) getPackageFromGit(_ context.Context, packageName string) ([]unstructured.Unstructured, error) {
	// This is a placeholder implementation - actual implementation would:
	// 1. Clone or pull from Git repository
	// 2. Navigate to package directory
	// 3. Parse YAML/JSON files into unstructured objects
	// 4. Return the resources

	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	// For now, return empty slice but this should be implemented
	log.Printf("Warning: getPackageFromGit is not implemented, returning empty package for %s", packageName)
	return []unstructured.Unstructured{}, nil
}

// getGVRFromResource extracts GroupVersionResource from a resource
func (sm *SyncManager) getGVRFromResource(resource *unstructured.Unstructured) schema.GroupVersionResource {
	gv, _ := schema.ParseGroupVersion(resource.GetAPIVersion())

	// Simple resource name derivation
	resource_name := strings.ToLower(resource.GetKind())
	if !strings.HasSuffix(resource_name, "s") {
		resource_name = resource_name + "s"
	}

	return schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource_name,
	}
}

// updateSyncStatus updates the synchronization status
func (sm *SyncManager) updateSyncStatus(packageName, clusterName string, state SyncState, message string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	key := fmt.Sprintf("%s/%s", packageName, clusterName)
	status := sm.syncStatus[key]
	if status == nil {
		status = &PackageSyncStatus{
			Package: packageName,
			Cluster: clusterName,
		}
		sm.syncStatus[key] = status
	}

	status.Status = state
	status.LastSync = time.Now()
	if state == SyncStateSynced {
		status.LastSuccess = time.Now()
		status.RetryCount = 0
	}

	if message != "" {
		status.Errors = append(status.Errors, SyncError{
			Message:   message,
			Timestamp: time.Now(),
		})
	}
}

// getSortedPackageGroups returns package groups sorted by priority
func (sm *SyncManager) getSortedPackageGroups() []PackageGroup {
	groups := make([]PackageGroup, len(sm.Config.PackageGroups))
	copy(groups, sm.Config.PackageGroups)

	// Sort by priority (higher priority first)
	for i := 0; i < len(groups)-1; i++ {
		for j := i + 1; j < len(groups); j++ {
			if groups[i].Priority < groups[j].Priority {
				groups[i], groups[j] = groups[j], groups[i]
			}
		}
	}

	return groups
}

// waitForGroupDependencies waits for group dependencies to be satisfied
func (sm *SyncManager) waitForGroupDependencies(ctx context.Context, group PackageGroup) error {
	if len(group.Dependencies) == 0 {
		return nil
	}

	log.Printf("Waiting for dependencies of group %s: %v", group.Name, group.Dependencies)

	// Wait for each dependency
	for _, depGroup := range group.Dependencies {
		if err := sm.waitForGroupCompletion(ctx, depGroup); err != nil {
			return fmt.Errorf("dependency %s not satisfied: %w", depGroup, err)
		}
	}

	return nil
}

// waitForPackageDependencies waits for package dependencies to be satisfied
func (sm *SyncManager) waitForPackageDependencies(ctx context.Context, packageName string) error {
	for _, dep := range sm.Config.Dependencies {
		if dep.Package == packageName {
			for _, depPackage := range dep.DependsOn {
				if err := sm.waitForPackageCompletion(ctx, depPackage, dep.WaitTimeout); err != nil {
					return fmt.Errorf("dependency %s not satisfied: %w", depPackage, err)
				}
			}
			break
		}
	}

	return nil
}

// waitForGroupCompletion waits for a group to complete
func (sm *SyncManager) waitForGroupCompletion(ctx context.Context, groupName string) error {
	// Find the group
	var group *PackageGroup
	for _, g := range sm.Config.PackageGroups {
		if g.Name == groupName {
			group = &g
			break
		}
	}

	if group == nil {
		return fmt.Errorf("group %s not found", groupName)
	}

	// Wait for all packages in the group to be synced
	for _, packageName := range group.Packages {
		if err := sm.waitForPackageCompletion(ctx, packageName, 5*time.Minute); err != nil {
			return err
		}
	}

	return nil
}

// waitForPackageCompletion waits for a package to be synced
func (sm *SyncManager) waitForPackageCompletion(ctx context.Context, packageName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for package %s", packageName)
		case <-ticker.C:
			if sm.isPackageSynced(packageName) {
				return nil
			}
		}
	}
}

// isPackageSynced checks if a package is synced across all required clusters
func (sm *SyncManager) isPackageSynced(packageName string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Check sync status for all clusters
	for key, status := range sm.syncStatus {
		if strings.HasPrefix(key, packageName+"/") {
			if status.Status != SyncStateSynced {
				return false
			}
		}
	}

	return true
}

// GetSyncStatus returns the current synchronization status
func (sm *SyncManager) GetSyncStatus() map[string]*PackageSyncStatus {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result := make(map[string]*PackageSyncStatus)
	for key, status := range sm.syncStatus {
		result[key] = status
	}

	return result
}

// deepEqual performs deep comparison of two interfaces
func (sm *SyncManager) deepEqual(a, b interface{}) bool {
	// Simple implementation - in production, use reflect.DeepEqual or similar
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// filterSystemAnnotations filters out system-managed annotations
func (sm *SyncManager) filterSystemAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}

	filtered := make(map[string]string)
	for key, value := range annotations {
		// Skip system annotations
		if !strings.HasPrefix(key, "kubectl.kubernetes.io/") &&
		   !strings.HasPrefix(key, "deployment.kubernetes.io/") &&
		   !strings.HasPrefix(key, "pv.kubernetes.io/") {
			filtered[key] = value
		}
	}

	return filtered
}