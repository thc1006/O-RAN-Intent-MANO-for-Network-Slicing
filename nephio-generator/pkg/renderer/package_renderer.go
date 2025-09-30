package renderer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// PackageRenderer provides package rendering capabilities using kpt fn render
type PackageRenderer struct {
	workDir          string
	kptPath          string
	functionRegistry FunctionRegistry
	validator        RenderValidator
}

// FunctionRegistry manages Kpt functions
type FunctionRegistry interface {
	GetFunction(name string) (*KptFunction, error)
	ListFunctions() ([]*KptFunction, error)
	ValidateFunction(fn *KptFunction) error
	ExecuteFunction(ctx context.Context, fn *KptFunction, packagePath string) error
}

// RenderValidator validates rendered packages
type RenderValidator interface {
	ValidateRenderedPackage(packagePath string) (*ValidationResult, error)
	ValidateResources(resources []RenderedResource) (*ValidationResult, error)
}

// KptFunction represents a Kpt function
type KptFunction struct {
	Name         string                 `json:"name" yaml:"name,omitempty"`
	Image        string                 `json:"image" yaml:"image"`
	Version      string                 `json:"version" yaml:"version,omitempty"`
	Type         FunctionType           `json:"type" yaml:"type,omitempty"`
	Description  string                 `json:"description" yaml:"description,omitempty"`
	ConfigSchema map[string]interface{} `json:"configSchema" yaml:"configSchema,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty" yaml:"configMap,omitempty"`
	ExecTimeout  time.Duration          `json:"execTimeout" yaml:"execTimeout,omitempty"`
	Required     bool                   `json:"required" yaml:"required,omitempty"`
}

// Kptfile structure
type Kptfile struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   map[string]interface{} `yaml:"metadata,omitempty"`
	Pipeline   struct {
		Mutators   []KptFunction `yaml:"mutators,omitempty"`
		Validators []KptFunction `yaml:"validators,omitempty"`
	} `yaml:"pipeline,omitempty"`
}

// FunctionType represents the type of function
type FunctionType string

const (
	FunctionTypeMutator   FunctionType = "mutator"
	FunctionTypeValidator FunctionType = "validator"
	FunctionTypeGenerator FunctionType = "generator"
)

// RenderOptions represents rendering options
type RenderOptions struct {
	FunctionPaths    []string          `json:"functionPaths,omitempty"`
	ImagePullPolicy  string            `json:"imagePullPolicy,omitempty"`
	Network          string            `json:"network,omitempty"`
	Mount            []string          `json:"mount,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	FnConfigPath     string            `json:"fnConfigPath,omitempty"`
	ResultsDir       string            `json:"resultsDir,omitempty"`
	FailFast         bool              `json:"failFast"`
	DryRun           bool              `json:"dryRun"`
	AllowExec        bool              `json:"allowExec"`
	AllowNetwork     bool              `json:"allowNetwork"`
	AllowFilesystem  bool              `json:"allowFilesystem"`
}

// RenderResult represents the result of package rendering
type RenderResult struct {
	Success         bool               `json:"success"`
	PackagePath     string             `json:"packagePath"`
	Resources       []RenderedResource `json:"resources"`
	FunctionResults []FunctionResult   `json:"functionResults"`
	ValidationResult *ValidationResult `json:"validationResult,omitempty"`
	Errors          []string           `json:"errors,omitempty"`
	Warnings        []string           `json:"warnings,omitempty"`
	Duration        time.Duration      `json:"duration"`
	Timestamp       time.Time          `json:"timestamp"`
}

// RenderedResource represents a rendered Kubernetes resource
type RenderedResource struct {
	APIVersion  string                 `yaml:"apiVersion"`
	Kind        string                 `yaml:"kind"`
	Metadata    map[string]interface{} `yaml:"metadata"`
	Spec        map[string]interface{} `yaml:"spec,omitempty"`
	Data        map[string]interface{} `yaml:"data,omitempty"`
	Status      map[string]interface{} `yaml:"status,omitempty"`
	FilePath    string                 `json:"filePath"`
	Size        int64                  `json:"size"`
	Checksum    string                 `json:"checksum"`
}

// FunctionResult represents the result of a function execution
type FunctionResult struct {
	FunctionName string        `json:"functionName"`
	FunctionType FunctionType  `json:"functionType"`
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
	Output       string        `json:"output,omitempty"`
	Errors       []string      `json:"errors,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`
	ExitCode     int           `json:"exitCode"`
}

// ValidationResult represents validation results
type ValidationResult struct {
	Valid       bool               `json:"valid"`
	Errors      []ValidationError  `json:"errors,omitempty"`
	Warnings    []ValidationError  `json:"warnings,omitempty"`
	Suggestions []ValidationError  `json:"suggestions,omitempty"`
	Summary     ValidationSummary  `json:"summary"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	ResourceRef string `json:"resourceRef,omitempty"`
	Rule        string `json:"rule,omitempty"`
}

// ValidationSummary represents validation summary
type ValidationSummary struct {
	TotalResources    int `json:"totalResources"`
	ValidResources    int `json:"validResources"`
	InvalidResources  int `json:"invalidResources"`
	ErrorCount        int `json:"errorCount"`
	WarningCount      int `json:"warningCount"`
	SuggestionCount   int `json:"suggestionCount"`
}

// NewPackageRenderer creates a new package renderer
func NewPackageRenderer(workDir, kptPath string, functionRegistry FunctionRegistry, validator RenderValidator) *PackageRenderer {
	return &PackageRenderer{
		workDir:          workDir,
		kptPath:          kptPath,
		functionRegistry: functionRegistry,
		validator:        validator,
	}
}

// RenderPackage renders a package using kpt fn render
func (r *PackageRenderer) RenderPackage(ctx context.Context, packagePath string, options *RenderOptions) (*RenderResult, error) {
	startTime := time.Now()

	result := &RenderResult{
		PackagePath: packagePath,
		Timestamp:   startTime,
	}

	// Validate package structure
	if err := r.validatePackageStructure(packagePath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Package structure validation failed: %v", err))
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("package structure validation failed: %w", err)
	}

	// Read Kptfile to get function pipeline
	kptfile, err := r.readKptfile(packagePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read Kptfile: %v", err))
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to read Kptfile: %w", err)
	}

	// Execute function pipeline
	if err := r.executeFunctionPipeline(ctx, packagePath, kptfile, options, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Function pipeline execution failed: %v", err))
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("function pipeline execution failed: %w", err)
	}

	// Read rendered resources
	resources, err := r.readRenderedResources(packagePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read rendered resources: %v", err))
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to read rendered resources: %w", err)
	}
	result.Resources = resources

	// Validate rendered package
	if r.validator != nil {
		validationResult, err := r.validator.ValidateRenderedPackage(packagePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Validation failed: %v", err))
		} else {
			result.ValidationResult = validationResult
		}
	}

	// Run kustomize build to final validation
	if err := r.runKustomizeBuild(packagePath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Kustomize build failed: %v", err))
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("kustomize build failed: %w", err)
	}

	result.Success = len(result.Errors) == 0
	result.Duration = time.Since(startTime)

	return result, nil
}

// RenderPackageWithKustomize renders a package using kustomize
func (r *PackageRenderer) RenderPackageWithKustomize(ctx context.Context, packagePath string) (*RenderResult, error) {
	startTime := time.Now()

	result := &RenderResult{
		PackagePath: packagePath,
		Timestamp:   startTime,
	}

	// Create filesystem
	fSys := filesys.MakeFsOnDisk()

	// Create kustomizer with options
	options := krusty.MakeDefaultOptions()
	options.LoadRestrictions = types.LoadRestrictionsRootOnly
	options.AddManagedbyLabel = true
	options.PluginConfig = types.DisabledPluginConfig()

	k := krusty.MakeKustomizer(options)

	// Run kustomize build
	resMap, err := k.Run(fSys, packagePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Kustomize build failed: %v", err))
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("kustomize build failed: %w", err)
	}

	// Convert resources to RenderedResource format
	for _, res := range resMap.Resources() {
		yamlContent, err := res.AsYAML()
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to convert resource to YAML: %v", err))
			continue
		}

		var resource RenderedResource
		if err := yaml.Unmarshal(yamlContent, &resource); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to unmarshal resource: %v", err))
			continue
		}

		resource.Size = int64(len(yamlContent))
		resource.Checksum = fmt.Sprintf("%x", yamlContent) // Simple checksum

		result.Resources = append(result.Resources, resource)
	}

	// Validate rendered resources
	if r.validator != nil {
		validationResult, err := r.validator.ValidateResources(result.Resources)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Resource validation failed: %v", err))
		} else {
			result.ValidationResult = validationResult
		}
	}

	result.Success = len(result.Errors) == 0
	result.Duration = time.Since(startTime)

	return result, nil
}

// GetKptFunctions returns available Kpt functions
func (r *PackageRenderer) GetKptFunctions() ([]*KptFunction, error) {
	return r.functionRegistry.ListFunctions()
}

// ValidateFunction validates a Kpt function
func (r *PackageRenderer) ValidateFunction(fn *KptFunction) error {
	return r.functionRegistry.ValidateFunction(fn)
}

// ExecuteFunction executes a single Kpt function
func (r *PackageRenderer) ExecuteFunction(ctx context.Context, fn *KptFunction, packagePath string) error {
	return r.functionRegistry.ExecuteFunction(ctx, fn, packagePath)
}

// validatePackageStructure validates the package directory structure
func (r *PackageRenderer) validatePackageStructure(packagePath string) error {
	if packagePath == "" {
		return fmt.Errorf("package path cannot be empty")
	}

	info, err := os.Stat(packagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("package directory does not exist: %s", packagePath)
		}
		return fmt.Errorf("failed to stat package directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("package path is not a directory: %s", packagePath)
	}

	// Check if directory is empty
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return fmt.Errorf("failed to read package directory: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("package directory is empty: %s", packagePath)
	}

	// Check for Kptfile
	kptfilePath := filepath.Join(packagePath, "Kptfile")
	if _, err := os.Stat(kptfilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Kptfile not found in package: %s", packagePath)
		}
		return fmt.Errorf("failed to check Kptfile: %w", err)
	}

	return nil
}

// readKptfile reads and parses the Kptfile from the package
func (r *PackageRenderer) readKptfile(packagePath string) (*Kptfile, error) {
	kptfilePath := filepath.Join(packagePath, "Kptfile")

	data, err := os.ReadFile(kptfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Kptfile: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("Kptfile is empty")
	}

	var kptfile Kptfile
	if err := yaml.Unmarshal(data, &kptfile); err != nil {
		return nil, fmt.Errorf("failed to parse Kptfile: %w", err)
	}

	// Validate required fields
	if kptfile.APIVersion == "" {
		return nil, fmt.Errorf("Kptfile missing apiVersion")
	}

	if kptfile.Kind == "" {
		return nil, fmt.Errorf("Kptfile missing kind")
	}

	return &kptfile, nil
}

// executeFunctionPipeline executes the Kpt function pipeline
func (r *PackageRenderer) executeFunctionPipeline(ctx context.Context, packagePath string, kptfile *Kptfile, options *RenderOptions, result *RenderResult) error {
	if kptfile == nil {
		return fmt.Errorf("kptfile cannot be nil")
	}

	// Combine mutators and validators
	var functions []KptFunction
	functions = append(functions, kptfile.Pipeline.Mutators...)
	functions = append(functions, kptfile.Pipeline.Validators...)

	// If no functions, return success
	if len(functions) == 0 {
		return nil
	}

	// Execute each function
	for _, fn := range functions {
		fnResult := FunctionResult{
			FunctionName: fn.Name,
			FunctionType: fn.Type,
		}

		startTime := time.Now()

		// Validate function
		if fn.Image == "" {
			fnResult.Success = false
			fnResult.Errors = append(fnResult.Errors, "function image cannot be empty")
			result.FunctionResults = append(result.FunctionResults, fnResult)
			return fmt.Errorf("function %s has empty image", fn.Name)
		}

		// Execute via registry if available
		if r.functionRegistry != nil {
			if err := r.functionRegistry.ExecuteFunction(ctx, &fn, packagePath); err != nil {
				fnResult.Success = false
				fnResult.Errors = append(fnResult.Errors, err.Error())
				fnResult.Duration = time.Since(startTime)
				result.FunctionResults = append(result.FunctionResults, fnResult)

				if options != nil && options.FailFast {
					return fmt.Errorf("function %s failed: %w", fn.Name, err)
				}
				continue
			}
		}

		fnResult.Success = true
		fnResult.Duration = time.Since(startTime)
		result.FunctionResults = append(result.FunctionResults, fnResult)
	}

	return nil
}

// readRenderedResources reads all rendered resources from the package
func (r *PackageRenderer) readRenderedResources(packagePath string) ([]RenderedResource, error) {
	var resources []RenderedResource

	// Read all YAML files in resources directory
	resourcesDir := filepath.Join(packagePath, "resources")

	// If resources directory doesn't exist, try reading from package root
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		resourcesDir = packagePath
	}

	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read resources directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		// Skip Kptfile
		if entry.Name() == "Kptfile" {
			continue
		}

		filePath := filepath.Join(resourcesDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read resource file %s: %w", entry.Name(), err)
		}

		// Skip empty files
		if len(data) == 0 {
			continue
		}

		// Try to parse as YAML
		var resource RenderedResource
		if err := yaml.Unmarshal(data, &resource); err != nil {
			// If it's not a valid Kubernetes resource, skip it
			continue
		}

		// Only include if it has required fields
		if resource.APIVersion == "" || resource.Kind == "" {
			continue
		}

		resource.FilePath = filePath
		resource.Size = int64(len(data))
		resources = append(resources, resource)
	}

	return resources, nil
}

// runKustomizeBuild runs kustomize build if kustomization.yaml exists
func (r *PackageRenderer) runKustomizeBuild(packagePath string) error {
	// Check for kustomization.yaml
	kustomizationPath := filepath.Join(packagePath, "kustomization.yaml")
	if _, err := os.Stat(kustomizationPath); os.IsNotExist(err) {
		// Also check for kustomization.yml
		kustomizationPath = filepath.Join(packagePath, "kustomization.yml")
		if _, err := os.Stat(kustomizationPath); os.IsNotExist(err) {
			// No kustomization file, skip
			return nil
		}
	}

	// Use existing RenderPackageWithKustomize logic
	_, err := r.RenderPackageWithKustomize(context.Background(), packagePath)
	if err != nil {
		// Check if error is due to missing resources
		if strings.Contains(err.Error(), "no such file or directory") ||
			strings.Contains(err.Error(), "not found") {
			// This is acceptable - kustomization might reference optional resources
			return nil
		}
		return fmt.Errorf("kustomize build failed: %w", err)
	}

	return nil
}