package gitops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/translator"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client interface for GitOps operations
type Client interface {
	PushPackage(ctx context.Context, pkg *translator.PorchPackage) (string, error)
	GetPackageRevision(ctx context.Context, revision string) (*translator.PorchPackage, error)
	UpdatePackage(ctx context.Context, revision string, pkg *translator.PorchPackage) (string, error)
	DeletePackage(ctx context.Context, revision string) error
}

// MockGitOpsClient provides a mock implementation for testing
type MockGitOpsClient struct {
	Packages map[string]*translator.PorchPackage
}

// NewMockGitOpsClient creates a new mock GitOps client
func NewMockGitOpsClient() *MockGitOpsClient {
	return &MockGitOpsClient{
		Packages: make(map[string]*translator.PorchPackage),
	}
}

// PushPackage pushes a package to the repository
func (c *MockGitOpsClient) PushPackage(ctx context.Context, pkg *translator.PorchPackage) (string, error) {
	revision := fmt.Sprintf("rev-%s-001", pkg.Name)
	c.Packages[revision] = pkg
	return revision, nil
}

// GetPackageRevision gets a specific package revision
func (c *MockGitOpsClient) GetPackageRevision(ctx context.Context, revision string) (*translator.PorchPackage, error) {
	pkg, exists := c.Packages[revision]
	if !exists {
		return nil, fmt.Errorf("package revision %s not found", revision)
	}
	return pkg, nil
}

// UpdatePackage updates an existing package
func (c *MockGitOpsClient) UpdatePackage(ctx context.Context, revision string, pkg *translator.PorchPackage) (string, error) {
	if _, exists := c.Packages[revision]; !exists {
		return "", fmt.Errorf("package revision %s not found", revision)
	}

	newRevision := fmt.Sprintf("%s-updated", revision)
	c.Packages[newRevision] = pkg
	return newRevision, nil
}

// DeletePackage deletes a package
func (c *MockGitOpsClient) DeletePackage(ctx context.Context, revision string) error {
	if _, exists := c.Packages[revision]; !exists {
		return fmt.Errorf("package revision %s not found", revision)
	}

	delete(c.Packages, revision)
	return nil
}

// Porch API types for PackageRevision resources

// PackageRevisionSpec represents the spec of a Porch PackageRevision
type PackageRevisionSpec struct {
	PackageName    string            `json:"packageName"`
	RepositoryName string            `json:"repositoryName"`
	Lifecycle      string            `json:"lifecycle,omitempty"`
	Tasks          []Task            `json:"tasks,omitempty"`
	Resources      map[string]string `json:"resources,omitempty"`
	Revision       string            `json:"revision,omitempty"`
}

// Task represents a Porch package task
type Task struct {
	Type  string     `json:"type"`
	Clone *CloneTask `json:"clone,omitempty"`
	Init  *InitTask  `json:"init,omitempty"`
}

// CloneTask represents a clone operation
type CloneTask struct {
	Upstream PackageRef `json:"upstream"`
}

// InitTask represents an init operation
type InitTask struct {
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
}

// PackageRef represents a reference to another package
type PackageRef struct {
	Name string `json:"name"`
}

// PackageRevisionStatus represents the status of a PackageRevision
type PackageRevisionStatus struct {
	Lifecycle           string `json:"lifecycle,omitempty"`
	PublishedBy         string `json:"publishedBy,omitempty"`
	PublishedAt         string `json:"publishedAt,omitempty"`
	UpstreamLock        string `json:"upstreamLock,omitempty"`
	UpstreamSource      string `json:"upstreamSource,omitempty"`
	UpstreamSourceType  string `json:"upstreamSourceType,omitempty"`
}

// PorchClient implements the real Porch GitOps client
type PorchClient struct {
	Client    client.Client
	RepoURL   string
	Namespace string
}

var (
	// PackageRevisionGVK is the GroupVersionKind for Porch PackageRevisions
	PackageRevisionGVK = schema.GroupVersionKind{
		Group:   "porch.kpt.dev",
		Version: "v1alpha1",
		Kind:    "PackageRevision",
	}
)

// NewPorchClient creates a new Porch client
func NewPorchClient(k8sClient client.Client, repoURL, namespace string) *PorchClient {
	return &PorchClient{
		Client:    k8sClient,
		RepoURL:   repoURL,
		Namespace: namespace,
	}
}

// PushPackage pushes a package to Porch by creating a PackageRevision
func (c *PorchClient) PushPackage(ctx context.Context, pkg *translator.PorchPackage) (string, error) {
	if pkg == nil {
		return "", fmt.Errorf("package cannot be nil")
	}
	if pkg.Name == "" {
		return "", fmt.Errorf("package name cannot be empty")
	}

	// Generate revision name
	revisionName := fmt.Sprintf("%s-v1", pkg.Name)

	// Convert package resources to map[string]string for Porch
	resources, err := c.convertPorchPackageToResources(pkg)
	if err != nil {
		return "", fmt.Errorf("failed to convert package to resources: %w", err)
	}

	// Create PackageRevision as unstructured object
	packageRevision := &unstructured.Unstructured{}
	packageRevision.SetGroupVersionKind(PackageRevisionGVK)
	packageRevision.SetName(revisionName)
	packageRevision.SetNamespace(c.Namespace)

	// Set spec
	spec := map[string]interface{}{
		"packageName":    pkg.Name,
		"repositoryName": c.extractRepoName(),
		"lifecycle":      "Draft",
		"tasks": []map[string]interface{}{
			{
				"type": "init",
				"init": map[string]interface{}{
					"description": fmt.Sprintf("Package for %s", pkg.Name),
					"keywords":    []string{"oran", "vnf"},
				},
			},
		},
		"resources": resources,
	}

	if err := unstructured.SetNestedMap(packageRevision.Object, spec, "spec"); err != nil {
		return "", fmt.Errorf("failed to set spec: %w", err)
	}

	// Create the PackageRevision via Kubernetes API
	if err := c.Client.Create(ctx, packageRevision); err != nil {
		return "", fmt.Errorf("failed to create package revision: %w", err)
	}

	return revisionName, nil
}

// GetPackageRevision gets a package revision from Porch
func (c *PorchClient) GetPackageRevision(ctx context.Context, revision string) (*translator.PorchPackage, error) {
	if revision == "" {
		return nil, fmt.Errorf("revision name cannot be empty")
	}

	// Create unstructured object for PackageRevision
	packageRevision := &unstructured.Unstructured{}
	packageRevision.SetGroupVersionKind(PackageRevisionGVK)

	// Get the PackageRevision
	key := types.NamespacedName{
		Name:      revision,
		Namespace: c.Namespace,
	}

	if err := c.Client.Get(ctx, key, packageRevision); err != nil {
		return nil, fmt.Errorf("failed to get package revision: %w", err)
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(packageRevision.Object, "spec")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to get spec from package revision: %w", err)
	}

	// Extract package name
	packageName, found, err := unstructured.NestedString(spec, "packageName")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to get packageName from spec: %w", err)
	}

	// Extract resources
	resourcesMap, found, err := unstructured.NestedStringMap(spec, "resources")
	if err != nil {
		return nil, fmt.Errorf("failed to get resources from spec: %w", err)
	}

	// Convert resources back to PorchPackage
	pkg, err := c.convertResourcesToPorchPackage(packageName, c.Namespace, resourcesMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert resources to package: %w", err)
	}

	return pkg, nil
}

// UpdatePackage updates a package in Porch
func (c *PorchClient) UpdatePackage(ctx context.Context, revision string, pkg *translator.PorchPackage) (string, error) {
	if revision == "" {
		return "", fmt.Errorf("revision name cannot be empty")
	}
	if pkg == nil {
		return "", fmt.Errorf("package cannot be nil")
	}

	// Get existing PackageRevision
	packageRevision := &unstructured.Unstructured{}
	packageRevision.SetGroupVersionKind(PackageRevisionGVK)

	key := types.NamespacedName{
		Name:      revision,
		Namespace: c.Namespace,
	}

	if err := c.Client.Get(ctx, key, packageRevision); err != nil {
		return "", fmt.Errorf("failed to get package revision: %w", err)
	}

	// Convert package resources to map[string]string
	resources, err := c.convertPorchPackageToResources(pkg)
	if err != nil {
		return "", fmt.Errorf("failed to convert package to resources: %w", err)
	}

	// Update the resources in the spec
	if err := unstructured.SetNestedStringMap(packageRevision.Object, resources, "spec", "resources"); err != nil {
		return "", fmt.Errorf("failed to update resources in spec: %w", err)
	}

	// Update the PackageRevision
	if err := c.Client.Update(ctx, packageRevision); err != nil {
		return "", fmt.Errorf("failed to update package revision: %w", err)
	}

	return revision, nil
}

// DeletePackage deletes a package from Porch
func (c *PorchClient) DeletePackage(ctx context.Context, revision string) error {
	if revision == "" {
		return fmt.Errorf("revision name cannot be empty")
	}

	// Create unstructured object for deletion
	packageRevision := &unstructured.Unstructured{}
	packageRevision.SetGroupVersionKind(PackageRevisionGVK)
	packageRevision.SetName(revision)
	packageRevision.SetNamespace(c.Namespace)

	// Delete the PackageRevision
	if err := c.Client.Delete(ctx, packageRevision); err != nil {
		return fmt.Errorf("failed to delete package revision: %w", err)
	}

	return nil
}

// Helper methods

// convertPorchPackageToResources converts a PorchPackage to a map of resource files
func (c *PorchClient) convertPorchPackageToResources(pkg *translator.PorchPackage) (map[string]string, error) {
	resources := make(map[string]string)

	// Add Kptfile
	if pkg.Kptfile != nil {
		kptfileBytes, err := json.MarshalIndent(pkg.Kptfile, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Kptfile: %w", err)
		}
		resources["Kptfile"] = string(kptfileBytes)
	}

	// Add Kustomization
	if pkg.Kustomization != nil {
		kustomizationBytes, err := json.MarshalIndent(pkg.Kustomization, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Kustomization: %w", err)
		}
		resources["kustomization.yaml"] = string(kustomizationBytes)
	}

	// Add each resource as a separate file
	for i, resource := range pkg.Resources {
		resourceMap := map[string]interface{}{
			"apiVersion": resource.APIVersion,
			"kind":       resource.Kind,
			"metadata":   resource.Metadata,
			"spec":       resource.Spec,
		}

		resourceBytes, err := json.MarshalIndent(resourceMap, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource %d: %w", i, err)
		}

		// Generate filename based on kind and index
		filename := fmt.Sprintf("%s-%d.yaml", resource.Kind, i)
		resources[filename] = string(resourceBytes)
	}

	return resources, nil
}

// convertResourcesToPorchPackage converts a map of resources back to a PorchPackage
func (c *PorchClient) convertResourcesToPorchPackage(name, namespace string, resourcesMap map[string]string) (*translator.PorchPackage, error) {
	pkg := &translator.PorchPackage{
		Name:      name,
		Namespace: namespace,
		Resources: []translator.Resource{},
	}

	// Parse each resource file
	for filename, content := range resourcesMap {
		// Skip Kptfile and Kustomization (handle them separately if needed)
		if filename == "Kptfile" {
			var kptfile translator.Kptfile
			if err := json.Unmarshal([]byte(content), &kptfile); err != nil {
				return nil, fmt.Errorf("failed to unmarshal Kptfile: %w", err)
			}
			pkg.Kptfile = &kptfile
			continue
		}

		if filename == "kustomization.yaml" {
			var kustomization translator.Kustomization
			if err := json.Unmarshal([]byte(content), &kustomization); err != nil {
				return nil, fmt.Errorf("failed to unmarshal Kustomization: %w", err)
			}
			pkg.Kustomization = &kustomization
			continue
		}

		// Parse resource YAML
		var resourceMap map[string]interface{}
		if err := json.Unmarshal([]byte(content), &resourceMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource %s: %w", filename, err)
		}

		// Extract fields
		apiVersion, _ := resourceMap["apiVersion"].(string)
		kind, _ := resourceMap["kind"].(string)
		metadata, _ := resourceMap["metadata"].(map[string]interface{})
		spec, _ := resourceMap["spec"].(map[string]interface{})

		resource := translator.Resource{
			APIVersion: apiVersion,
			Kind:       kind,
			Metadata:   metadata,
			Spec:       spec,
		}

		pkg.Resources = append(pkg.Resources, resource)
	}

	return pkg, nil
}

// extractRepoName extracts the repository name from RepoURL
func (c *PorchClient) extractRepoName() string {
	// Simple extraction: remove protocol and path, keep only the repo identifier
	// For URL like "https://github.com/org/repo.git", return "repo"
	repoURL := c.RepoURL

	// Remove .git suffix if present
	if len(repoURL) > 4 && repoURL[len(repoURL)-4:] == ".git" {
		repoURL = repoURL[:len(repoURL)-4]
	}

	// Extract last component of path
	lastSlash := -1
	for i := len(repoURL) - 1; i >= 0; i-- {
		if repoURL[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash >= 0 && lastSlash < len(repoURL)-1 {
		return repoURL[lastSlash+1:]
	}

	return "default-repo"
}
