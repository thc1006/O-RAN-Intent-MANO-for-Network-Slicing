// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// Constants for commonly used strings
const (
	// File constants
	KptfileString = "Kptfile"
)

// NephioValidator provides validation for Nephio/Porch packages
type NephioValidator struct {
	Config      NephioConfig
	httpClient  *http.Client
	kptPath     string
	porchClient *PorchClient
	validator   *security.FilePathValidator
}

// PorchClient represents a client for Porch API
type PorchClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// PackageRevision represents a Porch package revision
type PackageRevision struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   PackageRevisionMeta   `json:"metadata"`
	Spec       PackageRevisionSpec   `json:"spec"`
	Status     PackageRevisionStatus `json:"status"`
}

// PackageRevisionMeta contains package revision metadata
type PackageRevisionMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// PackageRevisionSpec contains package revision specification
type PackageRevisionSpec struct {
	PackageName    string          `json:"packageName"`
	Repository     string          `json:"repository"`
	Revision       string          `json:"revision"`
	Lifecycle      string          `json:"lifecycle"`
	WorkspaceName  string          `json:"workspaceName,omitempty"`
	Tasks          []Task          `json:"tasks,omitempty"`
	ReadinessGates []ReadinessGate `json:"readinessGates,omitempty"`
}

// PackageRevisionStatus contains package revision status
type PackageRevisionStatus struct {
	Conditions   []Condition   `json:"conditions,omitempty"`
	UpstreamLock *UpstreamLock `json:"upstreamLock,omitempty"`
	PublishedBy  string        `json:"publishedBy,omitempty"`
	PublishedAt  *time.Time    `json:"publishedAt,omitempty"`
	Deployment   bool          `json:"deployment,omitempty"`
}

// Task represents a package task
type Task struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Image  string                 `json:"image,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// ReadinessGate represents a readiness gate
type ReadinessGate struct {
	ConditionType string `json:"conditionType"`
}

// Condition represents a condition
type Condition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
}

// UpstreamLock represents upstream lock information
type UpstreamLock struct {
	Type string                 `json:"type"`
	Ref  map[string]interface{} `json:"ref,omitempty"`
}

// PackageValidationResult represents validation result for a package
type PackageValidationResult struct {
	PackageName     string             `json:"packageName"`
	Repository      string             `json:"repository"`
	Revision        string             `json:"revision"`
	Valid           bool               `json:"valid"`
	RenderSuccess   bool               `json:"renderSuccess"`
	DeploymentReady bool               `json:"deploymentReady"`
	Errors          []string           `json:"errors,omitempty"`
	Warnings        []string           `json:"warnings,omitempty"`
	Resources       []RenderedResource `json:"resources,omitempty"`
	RenderTime      time.Duration      `json:"renderTime"`
}

// RenderedResource represents a rendered Kubernetes resource
type RenderedResource struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	Valid      bool     `json:"valid"`
	Issues     []string `json:"issues,omitempty"`
}

// NewNephioValidator creates a new Nephio validator
func NewNephioValidator(config NephioConfig) (*NephioValidator, error) {
	// Create secure file path validator for Kubernetes/Nephio packages
	fileValidator := security.CreateValidatorForKubernetes(".")

	validator := &NephioValidator{
		Config:    config,
		validator: fileValidator,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Find kpt binary
	kptPath, err := exec.LookPath("kpt")
	if err != nil {
		return nil, fmt.Errorf("kpt binary not found in PATH: %w", err)
	}
	validator.kptPath = kptPath

	// Initialize Porch client if server is configured
	if config.PorchServer != "" {
		validator.porchClient = &PorchClient{
			BaseURL:    config.PorchServer,
			HTTPClient: validator.httpClient,
		}
	}

	return validator, nil
}

// ValidatePackage validates a Nephio package
func (nv *NephioValidator) ValidatePackage(ctx context.Context, packagePath string) error {
	result, err := nv.ValidatePackageDetailed(ctx, packagePath)
	if err != nil {
		return err
	}

	if !result.Valid {
		return fmt.Errorf("package validation failed: %v", result.Errors)
	}

	return nil
}

// ValidatePackageDetailed performs detailed package validation
func (nv *NephioValidator) ValidatePackageDetailed(ctx context.Context, packagePath string) (*PackageValidationResult, error) {
	startTime := time.Now()

	result := &PackageValidationResult{
		PackageName: filepath.Base(packagePath),
		Valid:       true,
		Resources:   make([]RenderedResource, 0),
	}

	// Check if package directory exists
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("package directory does not exist: %q", packagePath))
		return result, nil
	}

	// Validate package structure
	if err := nv.validatePackageStructure(packagePath); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid package structure: %v", err))
	}

	// Validate Kptfile
	if err := nv.validateKptfile(packagePath); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid Kptfile: %v", err))
	}

	// Render package using kpt
	renderResult, err := nv.renderPackage(ctx, packagePath)
	if err != nil {
		result.Valid = false
		result.RenderSuccess = false
		result.Errors = append(result.Errors, fmt.Sprintf("package rendering failed: %v", err))
	} else {
		result.RenderSuccess = true
		result.Resources = renderResult
	}

	// Validate rendered resources
	if result.RenderSuccess {
		if err := nv.validateRenderedResources(result.Resources); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("rendered resource validation failed: %v", err))
		}
	}

	// Check deployment readiness if Porch client is available
	if nv.porchClient != nil {
		deploymentReady, err := nv.checkDeploymentReadiness(ctx, result.PackageName)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("deployment readiness check failed: %v", err))
		} else {
			result.DeploymentReady = deploymentReady
		}
	}

	result.RenderTime = time.Since(startTime)
	return result, nil
}

// validatePackageStructure validates the package directory structure
func (nv *NephioValidator) validatePackageStructure(packagePath string) error {
	// Check for required files
	requiredFiles := []string{KptfileString, "package-context.yaml"}

	for _, file := range requiredFiles {
		filePath := filepath.Join(packagePath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("required file missing: %s", file)
		}
	}

	// Check for at least one YAML resource file
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return fmt.Errorf("failed to read package directory: %w", err)
	}

	hasResourceFiles := false
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
			if entry.Name() != KptfileString && entry.Name() != "package-context.yaml" {
				hasResourceFiles = true
				break
			}
		}
	}

	if !hasResourceFiles {
		return fmt.Errorf("no resource files found in package")
	}

	return nil
}

// validateKptfile validates the Kptfile
func (nv *NephioValidator) validateKptfile(packagePath string) error {
	kptfilePath := filepath.Join(packagePath, KptfileString)

	// Validate file path for security
	if err := nv.validator.ValidateFilePath(kptfilePath); err != nil {
		return fmt.Errorf("kptfile path validation failed: %w", err)
	}

	data, err := nv.validator.SafeReadFile(kptfilePath)
	if err != nil {
		return fmt.Errorf("failed to read Kptfile: %w", err)
	}

	var kptfile map[string]interface{}
	if err := yaml.Unmarshal(data, &kptfile); err != nil {
		return fmt.Errorf("invalid YAML in Kptfile: %w", err)
	}

	// Validate required fields
	requiredFields := []string{"apiVersion", "kind", "metadata"}
	for _, field := range requiredFields {
		if _, exists := kptfile[field]; !exists {
			return fmt.Errorf("required field missing in Kptfile: %s", field)
		}
	}

	// Validate kind
	if kind, ok := kptfile["kind"].(string); !ok || kind != KptfileString {
		return fmt.Errorf("invalid kind in Kptfile, expected 'Kptfile'")
	}

	// Validate apiVersion
	if apiVersion, ok := kptfile["apiVersion"].(string); !ok || !strings.HasPrefix(apiVersion, "kpt.dev/") {
		return fmt.Errorf("invalid apiVersion in Kptfile, expected 'kpt.dev/*'")
	}

	return nil
}

// renderPackage renders the package using kpt
func (nv *NephioValidator) renderPackage(ctx context.Context, packagePath string) ([]RenderedResource, error) {
	// Create temporary directory for rendering
	tempDir, err := os.MkdirTemp("", "kpt-render-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Copy package to temp directory
	if err := nv.copyDirectory(packagePath, tempDir); err != nil {
		return nil, fmt.Errorf("failed to copy package: %w", err)
	}

	// Run kpt fn render using secure execution
	args := []string{"fn", "render", tempDir}
	output, err := security.SecureExecuteWithValidation(ctx, "kpt", security.ValidateKptArgs, args...)
	if err != nil {
		return nil, fmt.Errorf("kpt fn render failed: %w, output: %s", err, string(output))
	}

	// Parse rendered resources
	resources, err := nv.parseRenderedResources(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rendered resources: %w", err)
	}

	return resources, nil
}

// copyDirectory copies a directory recursively
func (nv *NephioValidator) copyDirectory(src, dst string) error {
	// Validate paths for security
	if err := security.ValidateFilePath(src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := security.ValidateFilePath(dst); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	// Use secure execution for copy command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{"-r", src + "/.", dst}
	_, err := security.SecureExecute(ctx, "cp", args...)
	return err
}

// parseRenderedResources parses rendered Kubernetes resources
func (nv *NephioValidator) parseRenderedResources(packagePath string) ([]RenderedResource, error) {
	var resources []RenderedResource

	err := filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}

		// Skip special files
		basename := filepath.Base(path)
		if basename == KptfileString || basename == "package-context.yaml" {
			return nil
		}

		// Validate file path for security
		if err := nv.validator.ValidateFilePathAndExtension(path, []string{".yaml", ".yml"}); err != nil {
			log.Printf("Skipping file due to path validation failure: %s: %v", path, err)
			return nil
		}

		data, err := nv.validator.SafeReadFile(path)
		if err != nil {
			return err
		}

		// Parse YAML documents
		docs := strings.Split(string(data), "---")
		for _, doc := range docs {
			doc = strings.TrimSpace(doc)
			if doc == "" {
				continue
			}

			var resource unstructured.Unstructured
			if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
				log.Printf("Warning: failed to parse resource in %s: %v", path, err)
				continue
			}

			if resource.GetAPIVersion() == "" || resource.GetKind() == "" {
				continue // Skip non-Kubernetes resources
			}

			renderedResource := RenderedResource{
				APIVersion: resource.GetAPIVersion(),
				Kind:       resource.GetKind(),
				Name:       resource.GetName(),
				Namespace:  resource.GetNamespace(),
				Valid:      true,
			}

			// Validate resource
			if err := nv.validateResource(&resource); err != nil {
				renderedResource.Valid = false
				renderedResource.Issues = append(renderedResource.Issues, err.Error())
			}

			resources = append(resources, renderedResource)
		}

		return nil
	})

	return resources, err
}

// validateResource validates a single Kubernetes resource
func (nv *NephioValidator) validateResource(resource *unstructured.Unstructured) error {
	// Basic validation
	if resource.GetName() == "" {
		return fmt.Errorf("resource name is required")
	}

	if resource.GetAPIVersion() == "" {
		return fmt.Errorf("resource apiVersion is required")
	}

	if resource.GetKind() == "" {
		return fmt.Errorf("resource kind is required")
	}

	// Additional validation based on resource type
	switch resource.GetKind() {
	case "Deployment":
		return nv.validateDeployment(resource)
	case "Service":
		return nv.validateService(resource)
	case "ConfigMap":
		return nv.validateConfigMap(resource)
	case "Secret":
		return nv.validateSecret(resource)
	}

	return nil
}

// validateDeployment validates a Deployment resource
func (nv *NephioValidator) validateDeployment(resource *unstructured.Unstructured) error {
	spec, found, err := unstructured.NestedMap(resource.Object, "spec")
	if !found || err != nil {
		return fmt.Errorf("deployment spec is required")
	}

	if err := nv.validateDeploymentReplicas(spec); err != nil {
		return err
	}

	if err := nv.validateDeploymentSelector(spec); err != nil {
		return err
	}

	if err := nv.validateDeploymentTemplate(spec); err != nil {
		return err
	}

	return nil
}

// validateDeploymentReplicas validates deployment replicas
func (nv *NephioValidator) validateDeploymentReplicas(spec map[string]interface{}) error {
	if replicas, found, _ := unstructured.NestedInt64(spec, "replicas"); found && replicas < 0 {
		return fmt.Errorf("deployment replicas cannot be negative")
	}
	return nil
}

// validateDeploymentSelector validates deployment selector
func (nv *NephioValidator) validateDeploymentSelector(spec map[string]interface{}) error {
	selector, found, err := unstructured.NestedMap(spec, "selector")
	if !found || err != nil {
		return fmt.Errorf("deployment selector is required")
	}

	if matchLabels, found, _ := unstructured.NestedMap(selector, "matchLabels"); !found || len(matchLabels) == 0 {
		return fmt.Errorf("deployment selector.matchLabels is required")
	}

	return nil
}

// validateDeploymentTemplate validates deployment template
func (nv *NephioValidator) validateDeploymentTemplate(spec map[string]interface{}) error {
	template, found, err := unstructured.NestedMap(spec, "template")
	if !found || err != nil {
		return fmt.Errorf("deployment template is required")
	}

	templateSpec, found, err := unstructured.NestedMap(template, "spec")
	if !found || err != nil {
		return fmt.Errorf("deployment template.spec is required")
	}

	containers, found, err := unstructured.NestedSlice(templateSpec, "containers")
	if !found || err != nil || len(containers) == 0 {
		return fmt.Errorf("deployment template.spec.containers is required")
	}

	return nil
}

// validateService validates a Service resource
func (nv *NephioValidator) validateService(resource *unstructured.Unstructured) error {
	spec, found, err := unstructured.NestedMap(resource.Object, "spec")
	if !found || err != nil {
		return fmt.Errorf("service spec is required")
	}

	ports, found, err := unstructured.NestedSlice(spec, "ports")
	if !found || err != nil || len(ports) == 0 {
		return fmt.Errorf("service ports are required")
	}

	return nil
}

// validateConfigMap validates a ConfigMap resource
func (nv *NephioValidator) validateConfigMap(resource *unstructured.Unstructured) error {
	// ConfigMaps are generally valid if they have the basic required fields
	return nil
}

// validateSecret validates a Secret resource
func (nv *NephioValidator) validateSecret(resource *unstructured.Unstructured) error {
	// Secrets are generally valid if they have the basic required fields
	return nil
}

// validateRenderedResources validates all rendered resources
func (nv *NephioValidator) validateRenderedResources(resources []RenderedResource) error {
	var errors []string

	for _, resource := range resources {
		if !resource.Valid {
			errors = append(errors, fmt.Sprintf("resource %s/%s is invalid: %v", resource.Kind, resource.Name, resource.Issues))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("resource validation errors: %v", errors)
	}

	return nil
}

// checkDeploymentReadiness checks if a package is deployed and ready
func (nv *NephioValidator) checkDeploymentReadiness(ctx context.Context, packageName string) (bool, error) {
	// This would interact with Porch API to check package deployment status
	// Placeholder implementation
	log.Printf("Checking deployment readiness for package %s", packageName)
	return true, nil
}

// RenderPackageWithKustomize renders a package using Kustomize
func (nv *NephioValidator) RenderPackageWithKustomize(packagePath string) ([]runtime.Object, error) {
	fSys := filesys.MakeFsOnDisk()

	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())

	resMap, err := kustomizer.Run(fSys, packagePath)
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}

	resources := resMap.Resources()
	objects := make([]runtime.Object, 0, len(resources))

	for _, resource := range resources {
		obj := &unstructured.Unstructured{}
		if err := obj.UnmarshalJSON([]byte(resource.MustYaml())); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

// GetPackageRevisions retrieves package revisions from Porch
func (pc *PorchClient) GetPackageRevisions(ctx context.Context, namespace string) ([]PackageRevision, error) {
	// This would make HTTP requests to Porch API
	// Placeholder implementation
	return []PackageRevision{}, nil
}

// GetPackageRevision retrieves a specific package revision
func (pc *PorchClient) GetPackageRevision(ctx context.Context, namespace, name string) (*PackageRevision, error) {
	// This would make HTTP requests to Porch API
	// Placeholder implementation
	return &PackageRevision{}, nil
}

// CreatePackageRevision creates a new package revision
func (pc *PorchClient) CreatePackageRevision(ctx context.Context, pr *PackageRevision) error {
	// This would make HTTP requests to Porch API
	// Placeholder implementation
	return nil
}

// UpdatePackageRevision updates a package revision
func (pc *PorchClient) UpdatePackageRevision(ctx context.Context, pr *PackageRevision) error {
	// This would make HTTP requests to Porch API
	// Placeholder implementation
	return nil
}

// DeletePackageRevision deletes a package revision
func (pc *PorchClient) DeletePackageRevision(ctx context.Context, namespace, name string) error {
	// This would make HTTP requests to Porch API
	// Placeholder implementation
	return nil
}
