// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// RollbackManager handles automated rollback operations
type RollbackManager struct {
	Config        RollbackConfig
	ClusterClient *ClusterClient
	GitRepo       *GitRepository
	Validator     *ValidationFramework
}

// RollbackState represents the state of a rollback operation
type RollbackState struct {
	ID               string                  `json:"id"`
	Timestamp        time.Time               `json:"timestamp"`
	Reason           string                  `json:"reason"`
	SourceCommit     string                  `json:"sourceCommit"`
	TargetCommit     string                  `json:"targetCommit"`
	Status           RollbackStatus          `json:"status"`
	Resources        []RollbackResource      `json:"resources"`
	ValidationResult *ValidationResult       `json:"validationResult,omitempty"`
	Duration         time.Duration           `json:"duration"`
	Errors           []string                `json:"errors,omitempty"`
}

// RollbackStatus represents rollback operation status
type RollbackStatus string

const (
	RollbackStatusPending    RollbackStatus = "pending"
	RollbackStatusInProgress RollbackStatus = "in_progress"
	RollbackStatusCompleted  RollbackStatus = "completed"
	RollbackStatusFailed     RollbackStatus = "failed"
	RollbackStatusCancelled  RollbackStatus = "cancelled"
)

// RollbackResource represents a resource involved in rollback
type RollbackResource struct {
	APIVersion    string            `json:"apiVersion"`
	Kind          string            `json:"kind"`
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace"`
	Action        RollbackAction    `json:"action"`
	Status        string            `json:"status"`
	PreviousState map[string]interface{} `json:"previousState,omitempty"`
	CurrentState  map[string]interface{} `json:"currentState,omitempty"`
	Error         string            `json:"error,omitempty"`
}

// RollbackAction represents the action taken on a resource during rollback
type RollbackAction string

const (
	RollbackActionRevert  RollbackAction = "revert"
	RollbackActionDelete  RollbackAction = "delete"
	RollbackActionCreate  RollbackAction = "create"
	RollbackActionUpdate  RollbackAction = "update"
	RollbackActionSkip    RollbackAction = "skip"
)

// RollbackHistory maintains history of rollback operations
type RollbackHistory struct {
	Operations []RollbackState `json:"operations"`
	MaxEntries int             `json:"maxEntries"`
}

// RollbackTrigger represents conditions that trigger a rollback
type RollbackTrigger struct {
	Type        string                 `json:"type"`
	Condition   string                 `json:"condition"`
	Threshold   map[string]interface{} `json:"threshold"`
	Enabled     bool                   `json:"enabled"`
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(config RollbackConfig, client *ClusterClient, gitRepo *GitRepository, validator *ValidationFramework) *RollbackManager {
	return &RollbackManager{
		Config:        config,
		ClusterClient: client,
		GitRepo:       gitRepo,
		Validator:     validator,
	}
}

// ExecuteRollback executes a rollback operation
func (rm *RollbackManager) ExecuteRollback(ctx context.Context, reason, targetCommit string) (*RollbackState, error) {
	if !rm.Config.Enabled {
		return nil, fmt.Errorf("rollback is disabled")
	}

	startTime := time.Now()
	rollbackID := fmt.Sprintf("rollback-%d", startTime.Unix())

	// Get current commit
	currentCommit, err := rm.GitRepo.GetLastCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get current commit: %w", err)
	}

	// Initialize rollback state
	rollbackState := &RollbackState{
		ID:           rollbackID,
		Timestamp:    startTime,
		Reason:       reason,
		SourceCommit: currentCommit,
		TargetCommit: targetCommit,
		Status:       RollbackStatusPending,
		Resources:    make([]RollbackResource, 0),
	}

	log.Printf("Starting rollback %s: %s -> %s (reason: %s)", rollbackID, currentCommit[:8], targetCommit[:8], reason)

	// Execute rollback steps
	if err := rm.executeRollbackSteps(ctx, rollbackState); err != nil {
		rollbackState.Status = RollbackStatusFailed
		rollbackState.Errors = append(rollbackState.Errors, err.Error())
		rollbackState.Duration = time.Since(startTime)
		return rollbackState, err
	}

	rollbackState.Status = RollbackStatusCompleted
	rollbackState.Duration = time.Since(startTime)

	log.Printf("Rollback %s completed successfully in %v", rollbackID, rollbackState.Duration)
	return rollbackState, nil
}

// executeRollbackSteps executes the rollback process
func (rm *RollbackManager) executeRollbackSteps(ctx context.Context, rollbackState *RollbackState) error {
	rollbackState.Status = RollbackStatusInProgress

	// Step 1: Validate target commit exists
	if err := rm.validateTargetCommit(ctx, rollbackState.TargetCommit); err != nil {
		return fmt.Errorf("target commit validation failed: %w", err)
	}

	// Step 2: Get changed resources between commits
	changedResources, err := rm.getChangedResources(ctx, rollbackState.TargetCommit, rollbackState.SourceCommit)
	if err != nil {
		return fmt.Errorf("failed to get changed resources: %w", err)
	}

	// Step 3: Plan rollback actions
	rollbackPlan, err := rm.planRollbackActions(ctx, changedResources)
	if err != nil {
		return fmt.Errorf("failed to plan rollback actions: %w", err)
	}
	rollbackState.Resources = rollbackPlan

	// Step 4: Execute Git rollback
	if err := rm.executeGitRollback(ctx, rollbackState.TargetCommit); err != nil {
		return fmt.Errorf("Git rollback failed: %w", err)
	}

	// Step 5: Execute Kubernetes resource rollback
	if err := rm.executeResourceRollback(ctx, rollbackState); err != nil {
		return fmt.Errorf("resource rollback failed: %w", err)
	}

	// Step 6: Validate rollback success
	if err := rm.validateRollbackSuccess(ctx, rollbackState); err != nil {
		return fmt.Errorf("rollback validation failed: %w", err)
	}

	return nil
}

// validateTargetCommit validates that the target commit exists
func (rm *RollbackManager) validateTargetCommit(ctx context.Context, targetCommit string) error {
	// Check if commit exists in repository
	commits, err := rm.GitRepo.GetCommitHistory(100) // Check last 100 commits
	if err != nil {
		return fmt.Errorf("failed to get commit history: %w", err)
	}

	for _, commit := range commits {
		if commit.Hash == targetCommit || commit.Hash[:8] == targetCommit[:8] {
			return nil
		}
	}

	return fmt.Errorf("target commit %s not found in repository history", targetCommit)
}

// getChangedResources gets resources changed between two commits
func (rm *RollbackManager) getChangedResources(ctx context.Context, fromCommit, toCommit string) ([]string, error) {
	changedFiles, err := rm.GitRepo.GetChangedFiles(fromCommit, toCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	// Filter for Kubernetes resource files
	var resourceFiles []string
	for _, file := range changedFiles {
		if rm.isKubernetesResourceFile(file) {
			resourceFiles = append(resourceFiles, file)
		}
	}

	return resourceFiles, nil
}

// isKubernetesResourceFile checks if a file is a Kubernetes resource file
func (rm *RollbackManager) isKubernetesResourceFile(filename string) bool {
	// Check file extension
	if !(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
		return false
	}

	// Skip certain directories/files
	skipPaths := []string{
		".git/",
		"docs/",
		"scripts/",
		"tests/",
	}

	for _, skipPath := range skipPaths {
		if strings.Contains(filename, skipPath) {
			return false
		}
	}

	return true
}

// planRollbackActions plans the rollback actions for each resource
func (rm *RollbackManager) planRollbackActions(ctx context.Context, resourceFiles []string) ([]RollbackResource, error) {
	var rollbackResources []RollbackResource

	for _, file := range resourceFiles {
		// Parse current version of the file
		currentBranch, _ := rm.GitRepo.GetCurrentBranch()
		currentResources, err := rm.parseResourceFile(ctx, file, currentBranch)
		if err != nil {
			log.Printf("Warning: failed to parse current version of %s: %v", file, err)
			continue
		}

		// Parse target version of the file
		targetResources, err := rm.parseResourceFile(ctx, file, "target-commit")
		if err != nil {
			log.Printf("Warning: failed to parse target version of %s: %v", file, err)
			continue
		}

		// Plan actions for each resource
		for _, currentRes := range currentResources {
			rollbackRes := RollbackResource{
				APIVersion:    currentRes.GetAPIVersion(),
				Kind:          currentRes.GetKind(),
				Name:          currentRes.GetName(),
				Namespace:     currentRes.GetNamespace(),
				CurrentState:  currentRes.Object,
			}

			// Find corresponding resource in target
			var targetRes *unstructured.Unstructured
			for _, tr := range targetResources {
				if rm.resourcesMatch(currentRes, tr) {
					targetRes = tr
					break
				}
			}

			if targetRes != nil {
				// Resource exists in both - plan update/revert
				rollbackRes.Action = RollbackActionRevert
				rollbackRes.PreviousState = targetRes.Object
			} else {
				// Resource only exists in current - plan delete
				rollbackRes.Action = RollbackActionDelete
			}

			rollbackResources = append(rollbackResources, rollbackRes)
		}

		// Handle resources that only exist in target (need to be created)
		for _, targetRes := range targetResources {
			found := false
			for _, currentRes := range currentResources {
				if rm.resourcesMatch(targetRes, currentRes) {
					found = true
					break
				}
			}

			if !found {
				rollbackRes := RollbackResource{
					APIVersion:    targetRes.GetAPIVersion(),
					Kind:          targetRes.GetKind(),
					Name:          targetRes.GetName(),
					Namespace:     targetRes.GetNamespace(),
					Action:        RollbackActionCreate,
					PreviousState: targetRes.Object,
				}
				rollbackResources = append(rollbackResources, rollbackRes)
			}
		}
	}

	return rollbackResources, nil
}

// parseResourceFile parses Kubernetes resources from a file
func (rm *RollbackManager) parseResourceFile(ctx context.Context, filename, commit string) ([]*unstructured.Unstructured, error) {
	// Get file content at specific commit
	var content string
	var err error

	currentBranch, _ := rm.GitRepo.GetCurrentBranch()
	if commit == currentBranch {
		// Read current file
		fullPath := filepath.Join(rm.GitRepo.LocalPath, filename)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		content = string(data)
	} else {
		// Get file content at specific commit (would need git show command)
		content, err = rm.getFileAtCommit(filename, commit)
		if err != nil {
			return nil, err
		}
	}

	// Parse YAML documents
	var resources []*unstructured.Unstructured
	docs := strings.Split(content, "---")

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var resource unstructured.Unstructured
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			continue // Skip invalid documents
		}

		if resource.GetAPIVersion() != "" && resource.GetKind() != "" {
			resources = append(resources, &resource)
		}
	}

	return resources, nil
}

// getFileAtCommit gets file content at a specific commit
func (rm *RollbackManager) getFileAtCommit(filename, commit string) (string, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commit, filename))
	cmd.Dir = rm.GitRepo.LocalPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get file %s at commit %s: %w", filename, commit, err)
	}

	return string(output), nil
}

// resourcesMatch checks if two resources represent the same object
func (rm *RollbackManager) resourcesMatch(res1, res2 *unstructured.Unstructured) bool {
	return res1.GetAPIVersion() == res2.GetAPIVersion() &&
		res1.GetKind() == res2.GetKind() &&
		res1.GetName() == res2.GetName() &&
		res1.GetNamespace() == res2.GetNamespace()
}

// executeGitRollback executes the Git rollback
func (rm *RollbackManager) executeGitRollback(ctx context.Context, targetCommit string) error {
	log.Printf("Executing Git rollback to commit %s", targetCommit)

	// Create a backup branch before rollback
	backupBranch := fmt.Sprintf("backup-%d", time.Now().Unix())
	if err := rm.GitRepo.CreateBranch(backupBranch); err != nil {
		log.Printf("Warning: failed to create backup branch: %v", err)
	}

	// Reset to target commit
	if err := rm.GitRepo.Reset(targetCommit, true); err != nil {
		return fmt.Errorf("failed to reset to target commit: %w", err)
	}

	return nil
}

// executeResourceRollback executes the Kubernetes resource rollback
func (rm *RollbackManager) executeResourceRollback(ctx context.Context, rollbackState *RollbackState) error {
	log.Printf("Executing Kubernetes resource rollback for %d resources", len(rollbackState.Resources))

	// Sort resources by priority (deletions first, then updates, then creates)
	sort.Slice(rollbackState.Resources, func(i, j int) bool {
		return rm.getActionPriority(rollbackState.Resources[i].Action) <
			   rm.getActionPriority(rollbackState.Resources[j].Action)
	})

	for i := range rollbackState.Resources {
		resource := &rollbackState.Resources[i]
		if err := rm.executeResourceAction(ctx, resource); err != nil {
			resource.Status = "failed"
			resource.Error = err.Error()
			log.Printf("Failed to rollback resource %s/%s: %v", resource.Kind, resource.Name, err)

			if rm.Config.PreserveData {
				continue // Continue with other resources
			} else {
				return fmt.Errorf("resource rollback failed for %s/%s: %w", resource.Kind, resource.Name, err)
			}
		} else {
			resource.Status = "success"
		}
	}

	return nil
}

// getActionPriority returns priority for rollback actions
func (rm *RollbackManager) getActionPriority(action RollbackAction) int {
	switch action {
	case RollbackActionDelete:
		return 1
	case RollbackActionUpdate, RollbackActionRevert:
		return 2
	case RollbackActionCreate:
		return 3
	default:
		return 4
	}
}

// executeResourceAction executes a rollback action on a resource
func (rm *RollbackManager) executeResourceAction(ctx context.Context, resource *RollbackResource) error {
	gvr := rm.getGVR(resource.APIVersion, resource.Kind)

	switch resource.Action {
	case RollbackActionDelete:
		return rm.deleteResource(ctx, gvr, resource)
	case RollbackActionCreate:
		return rm.createResource(ctx, gvr, resource)
	case RollbackActionRevert, RollbackActionUpdate:
		return rm.updateResource(ctx, gvr, resource)
	default:
		return fmt.Errorf("unknown rollback action: %s", resource.Action)
	}
}

// deleteResource deletes a Kubernetes resource
func (rm *RollbackManager) deleteResource(ctx context.Context, gvr schema.GroupVersionResource, resource *RollbackResource) error {
	if resource.Namespace != "" {
		return rm.ClusterClient.DynamicClient.Resource(gvr).Namespace(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
	}
	return rm.ClusterClient.DynamicClient.Resource(gvr).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}

// createResource creates a Kubernetes resource
func (rm *RollbackManager) createResource(ctx context.Context, gvr schema.GroupVersionResource, resource *RollbackResource) error {
	obj := &unstructured.Unstructured{Object: resource.PreviousState}

	if resource.Namespace != "" {
		_, err := rm.ClusterClient.DynamicClient.Resource(gvr).Namespace(resource.Namespace).Create(ctx, obj, metav1.CreateOptions{})
		return err
	}
	_, err := rm.ClusterClient.DynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	return err
}

// updateResource updates a Kubernetes resource
func (rm *RollbackManager) updateResource(ctx context.Context, gvr schema.GroupVersionResource, resource *RollbackResource) error {
	obj := &unstructured.Unstructured{Object: resource.PreviousState}

	if resource.Namespace != "" {
		_, err := rm.ClusterClient.DynamicClient.Resource(gvr).Namespace(resource.Namespace).Update(ctx, obj, metav1.UpdateOptions{})
		return err
	}
	_, err := rm.ClusterClient.DynamicClient.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
	return err
}

// getGVR converts apiVersion and kind to GroupVersionResource
func (rm *RollbackManager) getGVR(apiVersion, kind string) schema.GroupVersionResource {
	parts := strings.Split(apiVersion, "/")
	var group, version string

	if len(parts) == 1 {
		group = ""
		version = parts[0]
	} else {
		group = parts[0]
		version = parts[1]
	}

	// Simple resource name derivation
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

// validateRollbackSuccess validates that the rollback was successful
func (rm *RollbackManager) validateRollbackSuccess(ctx context.Context, rollbackState *RollbackState) error {
	if rm.Validator == nil {
		return nil // Skip validation if validator is not available
	}

	log.Printf("Validating rollback success...")

	// Wait for resources to stabilize
	time.Sleep(30 * time.Second)

	// Run validation
	clusterName := "rollback-validation" // Would be determined from context
	validationResult, err := rm.Validator.ValidateCluster(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	rollbackState.ValidationResult = validationResult

	if !validationResult.Success {
		return fmt.Errorf("rollback validation failed: %v", validationResult.Errors)
	}

	log.Printf("Rollback validation successful")
	return nil
}

// GetRollbackHistory returns the rollback history
func (rm *RollbackManager) GetRollbackHistory() (*RollbackHistory, error) {
	// This would typically be persisted to storage
	// For now, return empty history
	return &RollbackHistory{
		Operations: make([]RollbackState, 0),
		MaxEntries: rm.Config.MaxRollbacks,
	}, nil
}

// CanRollback checks if rollback is possible
func (rm *RollbackManager) CanRollback(targetCommit string) (bool, string) {
	if !rm.Config.Enabled {
		return false, "Rollback is disabled"
	}

	// Check rollback count limit
	history, err := rm.GetRollbackHistory()
	if err != nil {
		return false, fmt.Sprintf("Failed to get rollback history: %v", err)
	}

	if len(history.Operations) >= rm.Config.MaxRollbacks {
		return false, "Maximum rollback limit reached"
	}

	// Check if target commit exists
	if err := rm.validateTargetCommit(context.Background(), targetCommit); err != nil {
		return false, fmt.Sprintf("Target commit validation failed: %v", err)
	}

	return true, ""
}

// TriggerRollback triggers rollback based on validation results
func (rm *RollbackManager) TriggerRollback(ctx context.Context, validationResult *ValidationResult) (*RollbackState, error) {
	if !rm.shouldTriggerRollback(validationResult) {
		return nil, nil // No rollback needed
	}

	// Get previous stable commit
	previousCommit, err := rm.getPreviousStableCommit(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous stable commit: %w", err)
	}

	reason := fmt.Sprintf("Validation failed: %v", validationResult.Errors)
	return rm.ExecuteRollback(ctx, reason, previousCommit)
}

// shouldTriggerRollback determines if rollback should be triggered
func (rm *RollbackManager) shouldTriggerRollback(validationResult *ValidationResult) bool {
	if validationResult.Success {
		return false
	}

	// Check if errors are critical enough to trigger rollback
	criticalErrors := []string{
		"deployment failed",
		"service unavailable",
		"health check failed",
		"performance threshold exceeded",
	}

	for _, error := range validationResult.Errors {
		for _, critical := range criticalErrors {
			if strings.Contains(strings.ToLower(error), critical) {
				return true
			}
		}
	}

	return false
}

// getPreviousStableCommit gets the previous stable commit
func (rm *RollbackManager) getPreviousStableCommit(ctx context.Context) (string, error) {
	commits, err := rm.GitRepo.GetCommitHistory(10)
	if err != nil {
		return "", err
	}

	// For now, return the previous commit
	// In a real implementation, this would check for commits with successful validations
	if len(commits) > 1 {
		return commits[1].Hash, nil
	}

	return "", fmt.Errorf("no previous commit available")
}