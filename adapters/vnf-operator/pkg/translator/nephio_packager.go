package translator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
)

// NephioPackager generates Nephio packages from VNF specifications
type NephioPackager struct {
	logger     *slog.Logger
	workingDir string
	registry   string
}

// NephioPackage represents a complete Nephio package
type NephioPackage struct {
	Metadata      *PackageMetadata   `yaml:"metadata"`
	Kptfile       *KptfileSpec       `yaml:"kptfile"`
	Resources     []runtime.Object   `yaml:"resources"`
	Kustomization *KustomizationSpec `yaml:"kustomization"`
	Functions     []FunctionSpec     `yaml:"functions,omitempty"`
	Validation    *ValidationSpec    `yaml:"validation,omitempty"`
}

// PackageMetadata contains package identification and versioning
type PackageMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Keywords    []string          `yaml:"keywords,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// KptfileSpec represents the Kpt package specification
type KptfileSpec struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   map[string]interface{} `yaml:"metadata"`
	Info       *PackageInfo           `yaml:"info"`
	Pipeline   *Pipeline              `yaml:"pipeline,omitempty"`
	Inventory  *Inventory             `yaml:"inventory,omitempty"`
}

// PackageInfo contains package information
type PackageInfo struct {
	Site        string   `yaml:"site,omitempty"`
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords,omitempty"`
	Man         string   `yaml:"man,omitempty"`
	License     string   `yaml:"license,omitempty"`
}

// Pipeline defines the package processing pipeline
type Pipeline struct {
	Mutators   []FunctionSpec `yaml:"mutators,omitempty"`
	Validators []FunctionSpec `yaml:"validators,omitempty"`
}

// FunctionSpec defines a Kpt function
type FunctionSpec struct {
	Image      string                 `yaml:"image"`
	ConfigPath string                 `yaml:"configPath,omitempty"`
	Config     map[string]interface{} `yaml:"config,omitempty"`
	Name       string                 `yaml:"name,omitempty"`
	Selectors  []Selector             `yaml:"selectors,omitempty"`
}

// Selector defines resource selection criteria
type Selector struct {
	APIVersion string            `yaml:"apiVersion,omitempty"`
	Kind       string            `yaml:"kind,omitempty"`
	Name       string            `yaml:"name,omitempty"`
	Namespace  string            `yaml:"namespace,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
}

// Inventory tracks package resources
type Inventory struct {
	Namespace   string `yaml:"namespace,omitempty"`
	Name        string `yaml:"name,omitempty"`
	InventoryID string `yaml:"inventoryID,omitempty"`
}

// KustomizationSpec represents Kustomize configuration
type KustomizationSpec struct {
	APIVersion            string               `yaml:"apiVersion"`
	Kind                  string               `yaml:"kind"`
	Namespace             string               `yaml:"namespace,omitempty"`
	NamePrefix            string               `yaml:"namePrefix,omitempty"`
	NameSuffix            string               `yaml:"nameSuffix,omitempty"`
	Resources             []string             `yaml:"resources"`
	Components            []string             `yaml:"components,omitempty"`
	ConfigMapGenerator    []ConfigMapGenerator `yaml:"configMapGenerator,omitempty"`
	SecretGenerator       []SecretGenerator    `yaml:"secretGenerator,omitempty"`
	PatchesStrategicMerge []string             `yaml:"patchesStrategicMerge,omitempty"`
	Patches               []Patch              `yaml:"patches,omitempty"`
	Images                []ImageTransform     `yaml:"images,omitempty"`
	Replicas              []ReplicaTransform   `yaml:"replicas,omitempty"`
}

// ConfigMapGenerator generates ConfigMaps
type ConfigMapGenerator struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Files     []string          `yaml:"files,omitempty"`
	Literals  []string          `yaml:"literals,omitempty"`
	EnvFiles  []string          `yaml:"envs,omitempty"`
	Options   *GeneratorOptions `yaml:"options,omitempty"`
}

// SecretGenerator generates Secrets
type SecretGenerator struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Type      string            `yaml:"type,omitempty"`
	Files     []string          `yaml:"files,omitempty"`
	Literals  []string          `yaml:"literals,omitempty"`
	EnvFiles  []string          `yaml:"envs,omitempty"`
	Options   *GeneratorOptions `yaml:"options,omitempty"`
}

// GeneratorOptions configure resource generation
type GeneratorOptions struct {
	Labels                map[string]string `yaml:"labels,omitempty"`
	Annotations           map[string]string `yaml:"annotations,omitempty"`
	DisableNameSuffixHash bool              `yaml:"disableNameSuffixHash,omitempty"`
}

// Patch defines strategic merge patches
type Patch struct {
	Path    string          `yaml:"path,omitempty"`
	Patch   string          `yaml:"patch,omitempty"`
	Target  *PatchTarget    `yaml:"target,omitempty"`
	Options map[string]bool `yaml:"options,omitempty"`
}

// PatchTarget specifies patch targets
type PatchTarget struct {
	Group       string            `yaml:"group,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Kind        string            `yaml:"kind,omitempty"`
	Name        string            `yaml:"name,omitempty"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labelSelector,omitempty"`
	Annotations map[string]string `yaml:"annotationSelector,omitempty"`
}

// ImageTransform modifies container images
type ImageTransform struct {
	Name    string `yaml:"name"`
	NewName string `yaml:"newName,omitempty"`
	NewTag  string `yaml:"newTag,omitempty"`
	Digest  string `yaml:"digest,omitempty"`
}

// ReplicaTransform modifies replica counts
type ReplicaTransform struct {
	Name  string `yaml:"name"`
	Count int    `yaml:"count"`
}

// ValidationSpec defines package validation rules
type ValidationSpec struct {
	OpenAPISchema string           `yaml:"openapi,omitempty"`
	Rules         []ValidationRule `yaml:"rules,omitempty"`
}

// ValidationRule defines validation constraints
type ValidationRule struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Rule        string `yaml:"rule"`
	Severity    string `yaml:"severity"`
}

// NewNephioPackager creates a new Nephio package generator
func NewNephioPackager(workingDir, registry string) *NephioPackager {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if workingDir == "" {
		workingDir = "/tmp/nephio-packages"
	}

	return &NephioPackager{
		logger:     logger,
		workingDir: workingDir,
		registry:   registry,
	}
}

// GeneratePackage creates a complete Nephio package from VNF specification
func (np *NephioPackager) GeneratePackage(ctx context.Context, vnf *manov1alpha1.VNF) (*NephioPackage, error) {
	np.logger.Info("Generating Nephio package", "vnf", vnf.Name, "type", vnf.Spec.Type)

	pkg := &NephioPackage{
		Metadata: np.generateMetadata(vnf),
	}

	// Generate Kptfile
	kptfile, err := np.generateKptfile(vnf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Kptfile: %w", err)
	}
	pkg.Kptfile = kptfile

	// Generate Kubernetes resources
	resources, err := np.generateResources(vnf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate resources: %w", err)
	}
	pkg.Resources = resources

	// Generate Kustomization
	kustomization, err := np.generateKustomization(vnf, resources)
	if err != nil {
		return nil, fmt.Errorf("failed to generate kustomization: %w", err)
	}
	pkg.Kustomization = kustomization

	// Generate pipeline functions
	functions, err := np.generateFunctions(vnf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate functions: %w", err)
	}
	pkg.Functions = functions

	// Generate validation rules
	validation, err := np.generateValidation(vnf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate validation: %w", err)
	}
	pkg.Validation = validation

	np.logger.Info("Generated Nephio package", "vnf", vnf.Name, "resources", len(resources))
	return pkg, nil
}

// WritePackage persists the package to filesystem
func (np *NephioPackager) WritePackage(ctx context.Context, pkg *NephioPackage, outputDir string) error {
	if outputDir == "" {
		outputDir = filepath.Join(np.workingDir, pkg.Metadata.Name)
	}

	// Create package directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Write Kptfile
	if err := np.writeKptfile(pkg.Kptfile, outputDir); err != nil {
		return fmt.Errorf("failed to write Kptfile: %w", err)
	}

	// Write resources
	if err := np.writeResources(pkg.Resources, outputDir); err != nil {
		return fmt.Errorf("failed to write resources: %w", err)
	}

	// Write Kustomization
	if err := np.writeKustomization(pkg.Kustomization, outputDir); err != nil {
		return fmt.Errorf("failed to write kustomization: %w", err)
	}

	// Write function configs
	if err := np.writeFunctions(pkg.Functions, outputDir); err != nil {
		return fmt.Errorf("failed to write functions: %w", err)
	}

	np.logger.Info("Package written to filesystem", "path", outputDir)
	return nil
}

// ValidatePackage validates package structure and contents
func (np *NephioPackager) ValidatePackage(ctx context.Context, pkg *NephioPackage) error {
	// Validate metadata
	if pkg.Metadata == nil || pkg.Metadata.Name == "" {
		return fmt.Errorf("package metadata is required")
	}

	// Validate Kptfile
	if pkg.Kptfile == nil {
		return fmt.Errorf("Kptfile is required")
	}

	// Validate resources
	if len(pkg.Resources) == 0 {
		return fmt.Errorf("package must contain at least one resource")
	}

	// Validate resource names and types
	for i, resource := range pkg.Resources {
		if resource == nil {
			return fmt.Errorf("resource %d is nil", i)
		}

		// Convert to unstructured to check required fields
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
		if err != nil {
			return fmt.Errorf("failed to convert resource %d: %w", i, err)
		}

		obj := &unstructured.Unstructured{Object: unstructuredObj}
		if obj.GetAPIVersion() == "" {
			return fmt.Errorf("resource %d missing apiVersion", i)
		}
		if obj.GetKind() == "" {
			return fmt.Errorf("resource %d missing kind", i)
		}
		if obj.GetName() == "" {
			return fmt.Errorf("resource %d missing name", i)
		}
	}

	// Validate Kustomization
	if pkg.Kustomization == nil {
		return fmt.Errorf("kustomization is required")
	}

	np.logger.Info("Package validation successful", "name", pkg.Metadata.Name)
	return nil
}

// generateMetadata creates package metadata from VNF specification
func (np *NephioPackager) generateMetadata(vnf *manov1alpha1.VNF) *PackageMetadata {
	return &PackageMetadata{
		Name:        fmt.Sprintf("%s-%s", vnf.Name, strings.ToLower(string(vnf.Spec.Type))),
		Namespace:   vnf.Namespace,
		Version:     vnf.Spec.Version,
		Description: fmt.Sprintf("Nephio package for %s VNF of type %s", vnf.Name, vnf.Spec.Type),
		Keywords:    []string{"oran", "vnf", strings.ToLower(string(vnf.Spec.Type)), "5g"},
		Labels: map[string]string{
			"vnf.oran.io/name":    vnf.Name,
			"vnf.oran.io/type":    string(vnf.Spec.Type),
			"vnf.oran.io/version": vnf.Spec.Version,
			"nephio.org/package":  "true",
		},
		Annotations: map[string]string{
			"config.kubernetes.io/local-config": "true",
			"nephio.org/generated-by":           "O-RAN-MANO-VNF-Operator",
			"nephio.org/generated-at":           time.Now().Format(time.RFC3339),
		},
	}
}

// generateKptfile creates the Kptfile specification
func (np *NephioPackager) generateKptfile(vnf *manov1alpha1.VNF) (*KptfileSpec, error) {
	packageName := fmt.Sprintf("%s-%s", vnf.Name, strings.ToLower(string(vnf.Spec.Type)))

	return &KptfileSpec{
		APIVersion: "kpt.dev/v1",
		Kind:       "Kptfile",
		Metadata: map[string]interface{}{
			"name":      packageName,
			"namespace": vnf.Namespace,
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		Info: &PackageInfo{
			Description: fmt.Sprintf("Nephio package for %s VNF", vnf.Name),
			Keywords:    []string{"oran", "vnf", strings.ToLower(string(vnf.Spec.Type))},
			License:     "Apache-2.0",
		},
		Pipeline: &Pipeline{
			Mutators: []FunctionSpec{
				{
					Image: "gcr.io/kpt-fn/apply-setters:v0.2",
					Config: map[string]interface{}{
						"vnf-name":    vnf.Name,
						"vnf-type":    string(vnf.Spec.Type),
						"vnf-version": vnf.Spec.Version,
						"namespace":   vnf.Namespace,
					},
				},
				{
					Image: "gcr.io/kpt-fn/set-namespace:v0.4",
					Config: map[string]interface{}{
						"namespace": vnf.Namespace,
					},
				},
			},
			Validators: []FunctionSpec{
				{
					Image: "gcr.io/kpt-fn/kubeval:v0.3",
				},
				{
					Image: "gcr.io/kpt-fn/gatekeeper:v0.2",
				},
			},
		},
		Inventory: &Inventory{
			Namespace:   vnf.Namespace,
			Name:        packageName + "-inventory",
			InventoryID: packageName,
		},
	}, nil
}

// generateResources creates Kubernetes resources from VNF specification
func (np *NephioPackager) generateResources(vnf *manov1alpha1.VNF) ([]runtime.Object, error) {
	translator := NewPorchTranslator()

	porchPkg, err := translator.TranslateVNF(vnf)
	if err != nil {
		return nil, fmt.Errorf("failed to translate VNF: %w", err)
	}

	var resources []runtime.Object

	// Convert PorchPackage resources to runtime.Objects
	for _, resource := range porchPkg.Resources {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": resource.APIVersion,
				"kind":       resource.Kind,
				"metadata":   resource.Metadata,
				"spec":       resource.Spec,
			},
		}
		resources = append(resources, obj)
	}

	return resources, nil
}

// generateKustomization creates Kustomize configuration
func (np *NephioPackager) generateKustomization(vnf *manov1alpha1.VNF, resources []runtime.Object) (*KustomizationSpec, error) {
	var resourceFiles []string
	for i, resource := range resources {
		obj := &unstructured.Unstructured{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			resource.(*unstructured.Unstructured).Object, obj); err != nil {
			return nil, fmt.Errorf("failed to convert resource %d: %w", i, err)
		}

		fileName := fmt.Sprintf("%s-%s.yaml",
			strings.ToLower(obj.GetKind()),
			strings.ToLower(obj.GetName()))
		resourceFiles = append(resourceFiles, fileName)
	}

	return &KustomizationSpec{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Namespace:  vnf.Namespace,
		NamePrefix: vnf.Name + "-",
		Resources:  resourceFiles,
		ConfigMapGenerator: []ConfigMapGenerator{
			{
				Name: vnf.Name + "-config",
				Literals: []string{
					fmt.Sprintf("vnf-name=%s", vnf.Name),
					fmt.Sprintf("vnf-type=%s", vnf.Spec.Type),
					fmt.Sprintf("vnf-version=%s", vnf.Spec.Version),
				},
			},
		},
		Images: []ImageTransform{
			{
				Name:    vnf.Spec.Image.Repository,
				NewName: vnf.Spec.Image.Repository,
				NewTag:  vnf.Spec.Image.Tag,
			},
		},
	}, nil
}

// generateFunctions creates pipeline function specifications
func (np *NephioPackager) generateFunctions(vnf *manov1alpha1.VNF) ([]FunctionSpec, error) {
	functions := []FunctionSpec{
		{
			Name:  "apply-vnf-config",
			Image: "gcr.io/kpt-fn/apply-setters:v0.2",
			Config: map[string]interface{}{
				"vnf-name":        vnf.Name,
				"vnf-type":        string(vnf.Spec.Type),
				"vnf-version":     vnf.Spec.Version,
				"target-clusters": strings.Join(vnf.Spec.TargetClusters, ","),
			},
		},
		{
			Name:  "validate-qos",
			Image: "gcr.io/kpt-fn/starlark:v0.4",
			Config: map[string]interface{}{
				"source": np.generateQoSValidationScript(vnf),
			},
		},
	}

	// Add VNF-type-specific functions
	switch vnf.Spec.Type {
	case manov1alpha1.VNFTypeRAN:
		functions = append(functions, FunctionSpec{
			Name:  "validate-ran-config",
			Image: "gcr.io/kpt-fn/kubeval:v0.3",
			Selectors: []Selector{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Labels: map[string]string{
						"vnf-type": "RAN",
					},
				},
			},
		})
	case manov1alpha1.VNFTypeUPF:
		functions = append(functions, FunctionSpec{
			Name:  "validate-upf-config",
			Image: "gcr.io/kpt-fn/kubeval:v0.3",
			Selectors: []Selector{
				{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
					Labels: map[string]string{
						"vnf-type": "UPF",
					},
				},
			},
		})
	}

	return functions, nil
}

// generateValidation creates validation specifications
func (np *NephioPackager) generateValidation(vnf *manov1alpha1.VNF) (*ValidationSpec, error) {
	rules := []ValidationRule{
		{
			Name:        "require-vnf-labels",
			Description: "Ensure all resources have required VNF labels",
			Rule:        `has(object.metadata.labels) && has(object.metadata.labels["vnf.oran.io/name"])`,
			Severity:    "error",
		},
		{
			Name:        "validate-resource-limits",
			Description: "Ensure containers have resource limits",
			Rule:        `object.kind == "Deployment" ==> has(object.spec.template.spec.containers[0].resources.limits)`,
			Severity:    "warning",
		},
		{
			Name:        "validate-qos-requirements",
			Description: "Ensure QoS requirements are within valid ranges",
			Rule:        fmt.Sprintf(`object.kind == "ConfigMap" && object.metadata.name.endsWith("-qos-config") ==> double(object.data.bandwidth) >= 1.0 && double(object.data.bandwidth) <= 5.0`),
			Severity:    "error",
		},
	}

	return &ValidationSpec{
		Rules: rules,
	}, nil
}

// generateQoSValidationScript creates a Starlark script for QoS validation
func (np *NephioPackager) generateQoSValidationScript(vnf *manov1alpha1.VNF) string {
	return fmt.Sprintf(`
def validate_qos(resource):
    if resource.get("kind") == "ConfigMap" and "qos-config" in resource.get("metadata", {}).get("name", ""):
        data = resource.get("data", {})
        bandwidth = float(data.get("bandwidth", "0"))
        latency = float(data.get("latency", "0"))

        if bandwidth < 1.0 or bandwidth > 5.0:
            fail("Bandwidth must be between 1.0 and 5.0 Mbps, got: %%f" %% bandwidth)

        if latency < 1.0 or latency > 10.0:
            fail("Latency must be between 1.0 and 10.0 ms, got: %%f" %% latency)

for resource in ctx.resource_list["items"]:
    validate_qos(resource)
`)
}

// Helper methods for writing package files
func (np *NephioPackager) writeKptfile(kptfile *KptfileSpec, outputDir string) error {
	data, err := yaml.Marshal(kptfile)
	if err != nil {
		return fmt.Errorf("failed to marshal Kptfile: %w", err)
	}

	path := filepath.Join(outputDir, "Kptfile")
	return os.WriteFile(path, data, 0644)
}

func (np *NephioPackager) writeResources(resources []runtime.Object, outputDir string) error {
	for i, resource := range resources {
		obj := resource.(*unstructured.Unstructured)

		data, err := yaml.Marshal(obj.Object)
		if err != nil {
			return fmt.Errorf("failed to marshal resource %d: %w", i, err)
		}

		fileName := fmt.Sprintf("%s-%s.yaml",
			strings.ToLower(obj.GetKind()),
			strings.ToLower(obj.GetName()))
		path := filepath.Join(outputDir, fileName)

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write resource %s: %w", fileName, err)
		}
	}
	return nil
}

func (np *NephioPackager) writeKustomization(kustomization *KustomizationSpec, outputDir string) error {
	data, err := yaml.Marshal(kustomization)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization: %w", err)
	}

	path := filepath.Join(outputDir, "kustomization.yaml")
	return os.WriteFile(path, data, 0644)
}

func (np *NephioPackager) writeFunctions(functions []FunctionSpec, outputDir string) error {
	if len(functions) == 0 {
		return nil
	}

	functionsDir := filepath.Join(outputDir, "functions")
	if err := os.MkdirAll(functionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create functions directory: %w", err)
	}

	for _, function := range functions {
		data, err := yaml.Marshal(function)
		if err != nil {
			return fmt.Errorf("failed to marshal function %s: %w", function.Name, err)
		}

		fileName := fmt.Sprintf("%s.yaml", function.Name)
		path := filepath.Join(functionsDir, fileName)

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write function %s: %w", function.Name, err)
		}
	}
	return nil
}
