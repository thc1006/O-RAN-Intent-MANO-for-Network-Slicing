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

// Constants for commonly used strings
const (
	// Status constants
	StatusReady = "Ready"
)

// ArgoCDValidator validates ArgoCD resources and applications
type ArgoCDValidator struct {
	ClusterClient *ClusterClient
	Config        ArgoCDConfig
}

// ArgoCDConfig holds ArgoCD configuration
type ArgoCDConfig struct {
	Namespace            string        `yaml:"namespace"`
	ServerName           string        `yaml:"serverName"`
	ApplicationTimeout   time.Duration `yaml:"applicationTimeout"`
	SyncTimeout          time.Duration `yaml:"syncTimeout"`
	HealthCheckTimeout   time.Duration `yaml:"healthCheckTimeout"`
	ExpectedApplications []string      `yaml:"expectedApplications"`
	ProjectNamespaces    []string      `yaml:"projectNamespaces"`
}

// ArgoCDStatus represents the status of ArgoCD
type ArgoCDStatus struct {
	Healthy               bool                `json:"healthy"`
	ServerStatus          string              `json:"serverStatus"`
	Applications          []ApplicationStatus `json:"applications"`
	Projects              []ProjectStatus     `json:"projects"`
	Repositories          []RepositoryStatus  `json:"repositories"`
	Clusters              []ClusterStatus     `json:"clusters"`
	SyncedApplications    int                 `json:"syncedApplications"`
	OutOfSyncApplications int                 `json:"outOfSyncApplications"`
	ErroredApplications   int                 `json:"erroredApplications"`
	Errors                []string            `json:"errors,omitempty"`
	Warnings              []string            `json:"warnings,omitempty"`
	LastUpdate            time.Time           `json:"lastUpdate"`
}

// ApplicationStatus represents ArgoCD application status
type ApplicationStatus struct {
	Name           string                 `json:"name"`
	Namespace      string                 `json:"namespace"`
	Project        string                 `json:"project"`
	SyncStatus     SyncStatusInfo         `json:"syncStatus"`
	HealthStatus   HealthStatusInfo       `json:"healthStatus"`
	Source         ApplicationSource      `json:"source"`
	Destination    ApplicationDestination `json:"destination"`
	Conditions     []ApplicationCondition `json:"conditions"`
	OperationState *OperationState        `json:"operationState,omitempty"`
	Resources      []ResourceStatus       `json:"resources"`
	CreatedAt      time.Time              `json:"createdAt"`
	LastSyncedAt   time.Time              `json:"lastSyncedAt"`
}

// ProjectStatus represents ArgoCD project status
type ProjectStatus struct {
	Name         string               `json:"name"`
	Namespace    string               `json:"namespace"`
	Description  string               `json:"description"`
	Destinations []ProjectDestination `json:"destinations"`
	Sources      []string             `json:"sources"`
	Roles        []ProjectRole        `json:"roles"`
	Conditions   []ProjectCondition   `json:"conditions"`
}

// RepositoryStatus represents ArgoCD repository status
type RepositoryStatus struct {
	Name            string          `json:"name"`
	Repo            string          `json:"repo"`
	Type            string          `json:"type"`
	ConnectionState ConnectionState `json:"connectionState"`
	Credentials     string          `json:"credentials,omitempty"`
	InsecureIgnore  bool            `json:"insecureIgnore"`
}

// ClusterStatus represents ArgoCD cluster status
type ClusterStatus struct {
	Name            string          `json:"name"`
	Server          string          `json:"server"`
	ConnectionState ConnectionState `json:"connectionState"`
	ServerVersion   string          `json:"serverVersion"`
	Config          ClusterConfig   `json:"config"`
}

// SyncStatusInfo represents application sync status
type SyncStatusInfo struct {
	Status     string     `json:"status"`
	Revision   string     `json:"revision"`
	Revisions  []string   `json:"revisions,omitempty"`
	ComparedTo ComparedTo `json:"comparedTo"`
}

// HealthStatusInfo represents application health status
type HealthStatusInfo struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ApplicationSource represents application source
type ApplicationSource struct {
	RepoURL        string           `json:"repoURL"`
	Path           string           `json:"path"`
	TargetRevision string           `json:"targetRevision"`
	Helm           *HelmSource      `json:"helm,omitempty"`
	Kustomize      *KustomizeSource `json:"kustomize,omitempty"`
	Directory      *DirectorySource `json:"directory,omitempty"`
	Plugin         *PluginSource    `json:"plugin,omitempty"`
}

// ApplicationDestination represents application destination
type ApplicationDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
	Name      string `json:"name,omitempty"`
}

// ApplicationCondition represents application condition
type ApplicationCondition struct {
	Type               string    `json:"type"`
	Message            string    `json:"message"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
}

// OperationState represents operation state
type OperationState struct {
	Operation  Operation   `json:"operation"`
	Phase      string      `json:"phase"`
	Message    string      `json:"message,omitempty"`
	SyncResult *SyncResult `json:"syncResult,omitempty"`
	StartedAt  time.Time   `json:"startedAt"`
	FinishedAt *time.Time  `json:"finishedAt,omitempty"`
}

// ResourceStatus represents resource status
type ResourceStatus struct {
	Group           string        `json:"group,omitempty"`
	Version         string        `json:"version"`
	Kind            string        `json:"kind"`
	Namespace       string        `json:"namespace,omitempty"`
	Name            string        `json:"name"`
	Status          string        `json:"status"`
	Health          *HealthStatus `json:"health,omitempty"`
	Hook            bool          `json:"hook,omitempty"`
	RequiresPruning bool          `json:"requiresPruning,omitempty"`
}

// Additional types for ArgoCD structures
type ProjectDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
}

type ProjectRole struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Policies    []string `json:"policies"`
	Groups      []string `json:"groups"`
}

type ProjectCondition struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type ConnectionState struct {
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	AttemptedAt time.Time `json:"attemptedAt"`
}

type ComparedTo struct {
	Source      ApplicationSource      `json:"source"`
	Destination ApplicationDestination `json:"destination"`
}

type HelmSource struct {
	ValueFiles []string        `json:"valueFiles,omitempty"`
	Parameters []HelmParameter `json:"parameters,omitempty"`
	Values     string          `json:"values,omitempty"`
}

type KustomizeSource struct {
	NamePrefix   string            `json:"namePrefix,omitempty"`
	NameSuffix   string            `json:"nameSuffix,omitempty"`
	Images       []KustomizeImage  `json:"images,omitempty"`
	CommonLabels map[string]string `json:"commonLabels,omitempty"`
}

type DirectorySource struct {
	Recurse bool          `json:"recurse,omitempty"`
	Jsonnet JsonnetSource `json:"jsonnet,omitempty"`
}

type PluginSource struct {
	Name string      `json:"name"`
	Env  []PluginEnv `json:"env,omitempty"`
}

type HelmParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type KustomizeImage struct {
	Name    string `json:"name"`
	NewName string `json:"newName,omitempty"`
	NewTag  string `json:"newTag,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

type JsonnetSource struct {
	ExtVars []JsonnetVar `json:"extVars,omitempty"`
	TLAs    []JsonnetVar `json:"tlas,omitempty"`
}

type PluginEnv struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type JsonnetVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Code  bool   `json:"code,omitempty"`
}

type Operation struct {
	Sync *SyncOperation `json:"sync,omitempty"`
}

type SyncOperation struct {
	Revision    string             `json:"revision,omitempty"`
	Prune       bool               `json:"prune,omitempty"`
	DryRun      bool               `json:"dryRun,omitempty"`
	SyncOptions []string           `json:"syncOptions,omitempty"`
	Source      *ApplicationSource `json:"source,omitempty"`
	Manifests   []string           `json:"manifests,omitempty"`
}

type SyncResult struct {
	Resources []ResourceResult  `json:"resources,omitempty"`
	Revision  string            `json:"revision"`
	Source    ApplicationSource `json:"source"`
}

type ResourceResult struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	HookType  string `json:"hookType,omitempty"`
	HookPhase string `json:"hookPhase,omitempty"`
	SyncPhase string `json:"syncPhase,omitempty"`
}

type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// NewArgoCDValidator creates a new ArgoCD validator
func NewArgoCDValidator(client *ClusterClient, config ArgoCDConfig) *ArgoCDValidator {
	// Set defaults
	if config.Namespace == "" {
		config.Namespace = "argocd"
	}
	if config.ServerName == "" {
		config.ServerName = "argocd-server"
	}
	if config.ApplicationTimeout == 0 {
		config.ApplicationTimeout = 5 * time.Minute
	}
	if config.SyncTimeout == 0 {
		config.SyncTimeout = 10 * time.Minute
	}
	if config.HealthCheckTimeout == 0 {
		config.HealthCheckTimeout = 5 * time.Minute
	}

	return &ArgoCDValidator{
		ClusterClient: client,
		Config:        config,
	}
}

// ValidateArgoCD validates ArgoCD deployment and applications
func (acv *ArgoCDValidator) ValidateArgoCD(ctx context.Context) (*ArgoCDStatus, error) {
	status := &ArgoCDStatus{
		Healthy:      true,
		LastUpdate:   time.Now(),
		Applications: make([]ApplicationStatus, 0),
		Projects:     make([]ProjectStatus, 0),
		Repositories: make([]RepositoryStatus, 0),
		Clusters:     make([]ClusterStatus, 0),
	}

	// Check if ArgoCD is installed
	if err := acv.checkArgoCDInstallation(ctx); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("ArgoCD installation check failed: %v", err))
		return status, nil
	}

	// Validate ArgoCD server
	serverStatus, err := acv.validateArgoCDServer(ctx)
	if err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("ArgoCD server validation failed: %v", err))
	}
	status.ServerStatus = serverStatus

	// Validate applications
	apps := acv.validateApplications(ctx)
	status.Applications = apps
	acv.calculateApplicationStats(status)

	// Validate projects
	projects, err := acv.validateProjects(ctx)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Project validation warning: %v", err))
	} else {
		status.Projects = projects
	}

	// Validate repositories
	repos, err := acv.validateRepositories(ctx)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Repository validation warning: %v", err))
	} else {
		status.Repositories = repos
	}

	// Validate clusters
	clusters, err := acv.validateClusters(ctx)
	if err != nil {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Cluster validation warning: %v", err))
	} else {
		status.Clusters = clusters
	}

	return status, nil
}

// checkArgoCDInstallation checks if ArgoCD is properly installed
func (acv *ArgoCDValidator) checkArgoCDInstallation(ctx context.Context) error {
	// Check ArgoCD namespace
	_, err := acv.ClusterClient.Clientset.CoreV1().Namespaces().Get(ctx, acv.Config.Namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ArgoCD namespace %s not found", acv.Config.Namespace)
		}
		return fmt.Errorf("failed to get ArgoCD namespace: %w", err)
	}

	// Check for ArgoCD deployments
	deployments, err := acv.ClusterClient.Clientset.AppsV1().Deployments(acv.Config.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list deployments in ArgoCD namespace: %w", err)
	}

	expectedDeployments := []string{"argocd-server", "argocd-application-controller", "argocd-repo-server", "argocd-redis"}
	foundDeployments := make(map[string]bool)

	for _, deployment := range deployments.Items {
		for _, expected := range expectedDeployments {
			if strings.Contains(deployment.Name, expected) {
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

// validateArgoCDServer validates ArgoCD server status
func (acv *ArgoCDValidator) validateArgoCDServer(ctx context.Context) (string, error) {
	// Check server deployment
	deployment, err := acv.ClusterClient.Clientset.AppsV1().Deployments(acv.Config.Namespace).Get(ctx, acv.Config.ServerName, metav1.GetOptions{})
	if err != nil {
		return "NotFound", fmt.Errorf("ArgoCD server deployment not found: %w", err)
	}

	if deployment.Status.ReadyReplicas == 0 {
		return "NotReady", fmt.Errorf("ArgoCD server deployment has no ready replicas")
	}

	if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
		return "PartiallyReady", fmt.Errorf("ArgoCD server deployment is partially ready: %d/%d", deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
	}

	return StatusReady, nil
}

// validateApplications validates ArgoCD applications
// Returns error interface for future error handling capabilities
func (acv *ArgoCDValidator) validateApplications(ctx context.Context) []ApplicationStatus {
	// Define Application GVR
	appGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}

	var applications []ApplicationStatus

	// Search in ArgoCD namespace and project namespaces
	namespacesToSearch := []string{acv.Config.Namespace}
	namespacesToSearch = append(namespacesToSearch, acv.Config.ProjectNamespaces...)

	for _, ns := range namespacesToSearch {
		appList, err := acv.ClusterClient.DynamicClient.Resource(appGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue // Skip if namespace doesn't exist or access denied
		}

		for _, item := range appList.Items {
			app, err := acv.parseApplicationStatus(&item)
			if err != nil {
				continue // Skip invalid applications
			}
			applications = append(applications, *app)
		}
	}

	return applications
}

// parseApplicationStatus parses application status from unstructured object
// Returns error interface for future parsing error handling
func (acv *ArgoCDValidator) parseApplicationStatus(app *unstructured.Unstructured) (*ApplicationStatus, error) {
	status := &ApplicationStatus{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
		Resources: make([]ResourceStatus, 0),
	}

	// Parse metadata
	if creationTimestamp := app.GetCreationTimestamp(); !creationTimestamp.IsZero() {
		status.CreatedAt = creationTimestamp.Time
	}

	// Parse spec
	spec, found, err := unstructured.NestedMap(app.Object, "spec")
	if found && err == nil {
		// Parse project
		if project, found, _ := unstructured.NestedString(spec, "project"); found {
			status.Project = project
		}

		// Parse source
		if sourceObj, found, _ := unstructured.NestedMap(spec, "source"); found {
			status.Source = acv.parseApplicationSource(sourceObj)
		}

		// Parse destination
		if destObj, found, _ := unstructured.NestedMap(spec, "destination"); found {
			status.Destination = acv.parseApplicationDestination(destObj)
		}
	}

	// Parse status
	statusObj, found, err := unstructured.NestedMap(app.Object, "status")
	if found && err == nil {
		// Parse sync status
		if syncObj, found, _ := unstructured.NestedMap(statusObj, "sync"); found {
			status.SyncStatus = acv.parseSyncStatus(syncObj)
		}

		// Parse health status
		if healthObj, found, _ := unstructured.NestedMap(statusObj, "health"); found {
			status.HealthStatus = acv.parseHealthStatus(healthObj)
		}

		// Parse conditions
		if conditionsObj, found, _ := unstructured.NestedSlice(statusObj, "conditions"); found {
			status.Conditions = acv.parseApplicationConditions(conditionsObj)
		}

		// Parse operation state
		if opStateObj, found, _ := unstructured.NestedMap(statusObj, "operationState"); found {
			opState := acv.parseOperationState(opStateObj)
			status.OperationState = &opState
		}

		// Parse resources
		if resourcesObj, found, _ := unstructured.NestedSlice(statusObj, "resources"); found {
			status.Resources = acv.parseResourceStatuses(resourcesObj)
		}
	}

	return status, nil
}

// parseApplicationSource parses application source
func (acv *ArgoCDValidator) parseApplicationSource(sourceObj map[string]interface{}) ApplicationSource {
	source := ApplicationSource{}

	if repoURL, found, _ := unstructured.NestedString(sourceObj, "repoURL"); found {
		source.RepoURL = repoURL
	}
	if path, found, _ := unstructured.NestedString(sourceObj, "path"); found {
		source.Path = path
	}
	if targetRevision, found, _ := unstructured.NestedString(sourceObj, "targetRevision"); found {
		source.TargetRevision = targetRevision
	}

	return source
}

// parseApplicationDestination parses application destination
func (acv *ArgoCDValidator) parseApplicationDestination(destObj map[string]interface{}) ApplicationDestination {
	dest := ApplicationDestination{}

	if server, found, _ := unstructured.NestedString(destObj, "server"); found {
		dest.Server = server
	}
	if namespace, found, _ := unstructured.NestedString(destObj, "namespace"); found {
		dest.Namespace = namespace
	}
	if name, found, _ := unstructured.NestedString(destObj, "name"); found {
		dest.Name = name
	}

	return dest
}

// parseSyncStatus parses sync status
func (acv *ArgoCDValidator) parseSyncStatus(syncObj map[string]interface{}) SyncStatusInfo {
	syncStatus := SyncStatusInfo{}

	if status, found, _ := unstructured.NestedString(syncObj, "status"); found {
		syncStatus.Status = status
	}
	if revision, found, _ := unstructured.NestedString(syncObj, "revision"); found {
		syncStatus.Revision = revision
	}

	return syncStatus
}

// parseHealthStatus parses health status
func (acv *ArgoCDValidator) parseHealthStatus(healthObj map[string]interface{}) HealthStatusInfo {
	healthStatus := HealthStatusInfo{}

	if status, found, _ := unstructured.NestedString(healthObj, "status"); found {
		healthStatus.Status = status
	}
	if message, found, _ := unstructured.NestedString(healthObj, "message"); found {
		healthStatus.Message = message
	}

	return healthStatus
}

// parseApplicationConditions parses application conditions
func (acv *ArgoCDValidator) parseApplicationConditions(conditionsObj []interface{}) []ApplicationCondition {
	var conditions []ApplicationCondition

	for _, condObj := range conditionsObj {
		if condMap, ok := condObj.(map[string]interface{}); ok {
			condition := ApplicationCondition{}

			if condType, found, _ := unstructured.NestedString(condMap, "type"); found {
				condition.Type = condType
			}
			if message, found, _ := unstructured.NestedString(condMap, "message"); found {
				condition.Message = message
			}

			conditions = append(conditions, condition)
		}
	}

	return conditions
}

// parseOperationState parses operation state
func (acv *ArgoCDValidator) parseOperationState(opStateObj map[string]interface{}) OperationState {
	opState := OperationState{}

	if phase, found, _ := unstructured.NestedString(opStateObj, "phase"); found {
		opState.Phase = phase
	}
	if message, found, _ := unstructured.NestedString(opStateObj, "message"); found {
		opState.Message = message
	}

	return opState
}

// parseResourceStatuses parses resource statuses
func (acv *ArgoCDValidator) parseResourceStatuses(resourcesObj []interface{}) []ResourceStatus {
	var resources []ResourceStatus

	for _, resObj := range resourcesObj {
		if resMap, ok := resObj.(map[string]interface{}); ok {
			resource := ResourceStatus{}

			if group, found, _ := unstructured.NestedString(resMap, "group"); found {
				resource.Group = group
			}
			if version, found, _ := unstructured.NestedString(resMap, "version"); found {
				resource.Version = version
			}
			if kind, found, _ := unstructured.NestedString(resMap, "kind"); found {
				resource.Kind = kind
			}
			if namespace, found, _ := unstructured.NestedString(resMap, "namespace"); found {
				resource.Namespace = namespace
			}
			if name, found, _ := unstructured.NestedString(resMap, "name"); found {
				resource.Name = name
			}
			if status, found, _ := unstructured.NestedString(resMap, "status"); found {
				resource.Status = status
			}

			resources = append(resources, resource)
		}
	}

	return resources
}

// validateProjects validates ArgoCD projects
func (acv *ArgoCDValidator) validateProjects(ctx context.Context) ([]ProjectStatus, error) {
	// Define AppProject GVR
	projectGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "appprojects",
	}

	projectList, err := acv.ClusterClient.DynamicClient.Resource(projectGVR).Namespace(acv.Config.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ArgoCD projects: %w", err)
	}

	var projects []ProjectStatus
	for _, item := range projectList.Items {
		project := ProjectStatus{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		// Parse spec for description and destinations
		if spec, found, _ := unstructured.NestedMap(item.Object, "spec"); found {
			if description, found, _ := unstructured.NestedString(spec, "description"); found {
				project.Description = description
			}
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// validateRepositories validates ArgoCD repositories
func (acv *ArgoCDValidator) validateRepositories(ctx context.Context) ([]RepositoryStatus, error) {
	// Define Repository GVR
	repoGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "repositories",
	}

	repoList, err := acv.ClusterClient.DynamicClient.Resource(repoGVR).Namespace(acv.Config.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ArgoCD repositories: %w", err)
	}

	var repositories []RepositoryStatus
	for _, item := range repoList.Items {
		repo := RepositoryStatus{
			Name: item.GetName(),
		}

		// Parse spec for repo URL and type
		if spec, found, _ := unstructured.NestedMap(item.Object, "spec"); found {
			if repoURL, found, _ := unstructured.NestedString(spec, "repo"); found {
				repo.Repo = repoURL
			}
			if repoType, found, _ := unstructured.NestedString(spec, "type"); found {
				repo.Type = repoType
			}
		}

		repositories = append(repositories, repo)
	}

	return repositories, nil
}

// validateClusters validates ArgoCD clusters
func (acv *ArgoCDValidator) validateClusters(ctx context.Context) ([]ClusterStatus, error) {
	// Define Cluster GVR
	clusterGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "clusters",
	}

	clusterList, err := acv.ClusterClient.DynamicClient.Resource(clusterGVR).Namespace(acv.Config.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ArgoCD clusters: %w", err)
	}

	var clusters []ClusterStatus
	for _, item := range clusterList.Items {
		cluster := ClusterStatus{
			Name: item.GetName(),
		}

		// Parse spec for server
		if spec, found, _ := unstructured.NestedMap(item.Object, "spec"); found {
			if server, found, _ := unstructured.NestedString(spec, "server"); found {
				cluster.Server = server
			}
		}

		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// calculateApplicationStats calculates application statistics
func (acv *ArgoCDValidator) calculateApplicationStats(status *ArgoCDStatus) {
	for _, app := range status.Applications {
		switch app.SyncStatus.Status {
		case "Synced":
			status.SyncedApplications++
		case "OutOfSync":
			status.OutOfSyncApplications++
		default:
			if app.HealthStatus.Status == "Degraded" || app.HealthStatus.Status == "Missing" {
				status.ErroredApplications++
			}
		}
	}
}

// WaitForApplicationSync waits for an application to be synced
func (acv *ArgoCDValidator) WaitForApplicationSync(ctx context.Context, appName, namespace string) error {
	timeout := acv.Config.SyncTimeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	appGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for application %s to sync", appName)
		default:
			app, err := acv.ClusterClient.DynamicClient.Resource(appGVR).Namespace(namespace).Get(ctx, appName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get application %s: %w", appName, err)
			}

			if syncStatus, found, _ := unstructured.NestedString(app.Object, "status", "sync", "status"); found {
				if syncStatus == "Synced" {
					return nil
				}
			}

			time.Sleep(10 * time.Second)
		}
	}
}
