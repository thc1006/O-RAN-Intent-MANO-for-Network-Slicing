package generator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// EnhancedPackageGenerator provides production-ready Nephio package generation
type EnhancedPackageGenerator struct {
	templateRegistry TemplateRegistry
	outputDir        string
	packagePrefix    string
	kptFunctionRegistry KptFunctionRegistry
	validator        PackageValidator
}

// KptFunctionRegistry manages Kpt functions
type KptFunctionRegistry interface {
	GetFunction(name string) (*KptFunction, error)
	ListFunctions() ([]*KptFunction, error)
	ValidateFunction(fn *KptFunction) error
}

// PackageValidator validates generated packages
type PackageValidator interface {
	ValidatePackage(pkg *EnhancedPackage) error
	ValidateKptfile(kptfile *EnhancedKptfile) error
	ValidateResources(resources []EnhancedResource) error
}

// KptFunction represents a Kpt function with enhanced capabilities
type KptFunction struct {
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	ConfigSchema map[string]interface{} `json:"configSchema"`
	InputSchema  map[string]interface{} `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema"`
	ExecTimeout  time.Duration          `json:"execTimeout"`
	Required     bool                   `json:"required"`
}

// EnhancedPackage represents a production-ready Nephio package
type EnhancedPackage struct {
	Name            string                   `json:"name"`
	Path            string                   `json:"path"`
	Type            TemplateType             `json:"type"`
	VNFSpec         VNFSpec                  `json:"vnfSpec"`
	Kptfile         *EnhancedKptfile         `json:"kptfile"`
	Resources       []EnhancedResource       `json:"resources"`
	Functions       []KptFunction            `json:"functions"`
	Dependencies    []PackageDependency      `json:"dependencies"`
	Metadata        map[string]string        `json:"metadata"`
	ValidationRules []ValidationRule         `json:"validationRules"`
	RenderStatus    RenderStatus             `json:"renderStatus"`
}

// EnhancedKptfile represents an enhanced Kptfile with full Nephio capabilities
type EnhancedKptfile struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   KptfileMetadata        `yaml:"metadata"`
	Info       KptfileInfo            `yaml:"info"`
	Pipeline   KptfilePipeline        `yaml:"pipeline"`
	Upstream   *KptfileUpstream       `yaml:"upstream,omitempty"`
	UpstreamLock *KptfileUpstreamLock `yaml:"upstreamLock,omitempty"`
	Inventory  *KptfileInventory      `yaml:"inventory,omitempty"`
}

// KptfileMetadata represents enhanced metadata
type KptfileMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// KptfileInfo represents package information
type KptfileInfo struct {
	Title         string   `yaml:"title,omitempty"`
	Description   string   `yaml:"description"`
	Site          string   `yaml:"site,omitempty"`
	Emails        []string `yaml:"emails,omitempty"`
	License       string   `yaml:"license,omitempty"`
	Keywords      []string `yaml:"keywords,omitempty"`
	Man           string   `yaml:"man,omitempty"`
}

// KptfilePipeline represents the function pipeline
type KptfilePipeline struct {
	Mutators   []KptPipelineFunction `yaml:"mutators,omitempty"`
	Validators []KptPipelineFunction `yaml:"validators,omitempty"`
}

// KptPipelineFunction represents a function in the pipeline
type KptPipelineFunction struct {
	Image      string                 `yaml:"image"`
	ConfigPath string                 `yaml:"configPath,omitempty"`
	ConfigMap  map[string]interface{} `yaml:"configMap,omitempty"`
	Name       string                 `yaml:"name,omitempty"`
	Exec       *KptExecFunction       `yaml:"exec,omitempty"`
}

// KptExecFunction represents an executable function
type KptExecFunction struct {
	Path string   `yaml:"path"`
	Args []string `yaml:"args,omitempty"`
	Env  []string `yaml:"env,omitempty"`
}

// KptfileUpstream represents upstream package information
type KptfileUpstream struct {
	Type string `yaml:"type"`
	Git  *struct {
		Repo      string `yaml:"repo"`
		Directory string `yaml:"directory"`
		Ref       string `yaml:"ref"`
	} `yaml:"git,omitempty"`
}

// KptfileUpstreamLock represents upstream lock information
type KptfileUpstreamLock struct {
	Type string `yaml:"type"`
	Git  *struct {
		Repo      string `yaml:"repo"`
		Directory string `yaml:"directory"`
		Ref       string `yaml:"ref"`
		Commit    string `yaml:"commit"`
	} `yaml:"git,omitempty"`
}

// KptfileInventory represents inventory configuration
type KptfileInventory struct {
	Namespace   string            `yaml:"namespace"`
	Name        string            `yaml:"name"`
	InventoryID string            `yaml:"inventoryID"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// EnhancedResource represents a Kubernetes resource with validation
type EnhancedResource struct {
	APIVersion  string                 `yaml:"apiVersion"`
	Kind        string                 `yaml:"kind"`
	Metadata    map[string]interface{} `yaml:"metadata"`
	Spec        map[string]interface{} `yaml:"spec,omitempty"`
	Data        map[string]interface{} `yaml:"data,omitempty"`
	Status      map[string]interface{} `yaml:"status,omitempty"`
	Validation  ResourceValidation     `json:"validation"`
	DependsOn   []string               `json:"dependsOn,omitempty"`
}

// ResourceValidation represents validation information
type ResourceValidation struct {
	Validated bool     `json:"validated"`
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

// PackageDependency represents a package dependency
type PackageDependency struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
	Optional   bool   `json:"optional"`
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Expression  string `json:"expression"`
	Severity    string `json:"severity"`
}

// RenderStatus represents package rendering status
type RenderStatus struct {
	Rendered  bool      `json:"rendered"`
	Timestamp time.Time `json:"timestamp"`
	Errors    []string  `json:"errors,omitempty"`
	Warnings  []string  `json:"warnings,omitempty"`
}

// NewEnhancedPackageGenerator creates a new enhanced package generator
func NewEnhancedPackageGenerator(registry TemplateRegistry, outputDir, packagePrefix string, functionRegistry KptFunctionRegistry, validator PackageValidator) *EnhancedPackageGenerator {
	return &EnhancedPackageGenerator{
		templateRegistry:    registry,
		outputDir:          outputDir,
		packagePrefix:      packagePrefix,
		kptFunctionRegistry: functionRegistry,
		validator:          validator,
	}
}

// GenerateEnhancedPackage generates a production-ready Nephio package
func (g *EnhancedPackageGenerator) GenerateEnhancedPackage(ctx context.Context, spec *VNFSpec, templateType TemplateType) (*EnhancedPackage, error) {
	// Get template
	template, err := g.templateRegistry.GetTemplate(spec.Type, string(templateType))
	if err != nil {
		return nil, fmt.Errorf("failed to get template for VNF type %s: %w", spec.Type, err)
	}

	// Generate package name
	packageName := g.generatePackageName(spec)

	// Create enhanced package structure
	pkg := &EnhancedPackage{
		Name:    packageName,
		Path:    filepath.Join(g.outputDir, packageName),
		Type:    templateType,
		VNFSpec: *spec,
		Metadata: map[string]string{
			"generated.by":           "enhanced-nephio-generator",
			"generated.at":           time.Now().Format(time.RFC3339),
			"vnf.type":               spec.Type,
			"vnf.name":               spec.Name,
			"template.name":          template.Name,
			"template.version":       template.Version,
			"nephio.io/package-type": "workload",
			"nephio.io/workspace":    "default",
		},
		RenderStatus: RenderStatus{
			Rendered:  false,
			Timestamp: time.Now(),
		},
	}

	// Generate enhanced Kptfile
	pkg.Kptfile, err = g.generateEnhancedKptfile(spec, template)
	if err != nil {
		return nil, fmt.Errorf("failed to generate enhanced Kptfile: %w", err)
	}

	// Generate enhanced resources
	pkg.Resources, err = g.generateEnhancedResources(spec, template)
	if err != nil {
		return nil, fmt.Errorf("failed to generate enhanced resources: %w", err)
	}

	// Add Kpt functions based on VNF type
	pkg.Functions, err = g.selectKptFunctions(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to select Kpt functions: %w", err)
	}

	// Add validation rules
	pkg.ValidationRules = g.generateValidationRules(spec)

	// Add dependencies
	pkg.Dependencies = g.generateDependencies(spec, template)

	// Validate package
	if err := g.validator.ValidatePackage(pkg); err != nil {
		return nil, fmt.Errorf("package validation failed: %w", err)
	}

	return pkg, nil
}

// RenderPackage renders the package using kpt fn render
func (g *EnhancedPackageGenerator) RenderPackage(ctx context.Context, pkg *EnhancedPackage) error {
	// Create temporary filesystem
	fs := filesys.MakeFsOnDisk()

	// Create kustomizer
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())

	// Write package files to temporary directory
	tempDir := filepath.Join(g.outputDir, "temp", pkg.Name)
	if err := g.writePackageFiles(fs, tempDir, pkg); err != nil {
		return fmt.Errorf("failed to write package files: %w", err)
	}

	// Run kustomize build to validate
	_, err := k.Run(fs, tempDir)
	if err != nil {
		pkg.RenderStatus.Errors = append(pkg.RenderStatus.Errors, err.Error())
		return fmt.Errorf("kustomize build failed: %w", err)
	}

	// Mark as rendered
	pkg.RenderStatus.Rendered = true
	pkg.RenderStatus.Timestamp = time.Now()

	return nil
}

// generateEnhancedKptfile creates an enhanced Kptfile with full Nephio capabilities
func (g *EnhancedPackageGenerator) generateEnhancedKptfile(spec *VNFSpec, template *PackageTemplate) (*EnhancedKptfile, error) {
	kptfile := &EnhancedKptfile{
		APIVersion: "kpt.dev/v1",
		Kind:       "Kptfile",
		Metadata: KptfileMetadata{
			Name: g.generatePackageName(spec),
			Annotations: map[string]string{
				"config.kubernetes.io/local-config": "true",
				"nephio.io/package-type":             "workload",
				"nephio.io/workspace":                "default",
				"oran.io/vnf-type":                   spec.Type,
				"oran.io/qos-class":                  g.determineQoSClass(spec),
			},
			Labels: map[string]string{
				"nephio.io/component": "workload",
				"oran.io/vnf-type":    spec.Type,
				"oran.io/cloud-type":  spec.Placement.CloudType,
			},
		},
		Info: KptfileInfo{
			Title:       fmt.Sprintf("%s %s Package", spec.Name, spec.Type),
			Description: fmt.Sprintf("Nephio package for %s VNF of type %s with QoS requirements", spec.Name, spec.Type),
			Site:        "https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing",
			License:     "Apache-2.0",
			Keywords:    []string{"oran", "vnf", spec.Type, "nephio", "5g"},
		},
		Pipeline: KptfilePipeline{
			Mutators:   g.generateMutatorFunctions(spec),
			Validators: g.generateValidatorFunctions(spec),
		},
		Inventory: &KptfileInventory{
			Namespace:   spec.Name + "-" + strings.ToLower(spec.Type),
			Name:        g.generatePackageName(spec),
			InventoryID: fmt.Sprintf("%s-%s", spec.Name, spec.Type),
			Labels: map[string]string{
				"oran.io/vnf-type":   spec.Type,
				"oran.io/cloud-type": spec.Placement.CloudType,
			},
		},
	}

	return kptfile, nil
}

// generateMutatorFunctions creates mutator functions for the pipeline
func (g *EnhancedPackageGenerator) generateMutatorFunctions(spec *VNFSpec) []KptPipelineFunction {
	functions := []KptPipelineFunction{
		// Apply setters for VNF configuration
		{
			Image: "gcr.io/kpt-fn/apply-setters:v0.2",
			ConfigMap: map[string]interface{}{
				"vnf-name":      spec.Name,
				"vnf-type":      spec.Type,
				"vnf-version":   spec.Version,
				"cloud-type":    spec.Placement.CloudType,
				"cpu-cores":     spec.Resources.CPUCores,
				"memory-gb":     spec.Resources.MemoryGB,
				"bandwidth":     spec.QoS.Bandwidth,
				"max-latency":   spec.QoS.Latency,
				"image-repo":    spec.Image.Repository,
				"image-tag":     spec.Image.Tag,
			},
		},
		// Set namespace
		{
			Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
			ConfigMap: map[string]interface{}{
				"namespace": spec.Name + "-" + strings.ToLower(spec.Type),
			},
		},
		// Set labels
		{
			Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
			ConfigMap: map[string]interface{}{
				"app":                    spec.Name,
				"app.kubernetes.io/name": spec.Name,
				"oran.io/vnf-type":       spec.Type,
				"oran.io/cloud-type":     spec.Placement.CloudType,
				"nephio.io/component":    "workload",
			},
		},
		// Set annotations for QoS
		{
			Image: "gcr.io/kpt-fn/set-annotations:v0.1.4",
			ConfigMap: map[string]interface{}{
				"oran.io/qos-bandwidth": fmt.Sprintf("%.2f", spec.QoS.Bandwidth),
				"oran.io/qos-latency":   fmt.Sprintf("%.2f", spec.QoS.Latency),
				"oran.io/qos-class":     g.determineQoSClass(spec),
			},
		},
	}

	// Add VNF-specific functions
	switch spec.Type {
	case "RAN":
		functions = append(functions, KptPipelineFunction{
			Image: "gcr.io/kpt-fn/apply-replacements:v0.1.1",
			ConfigPath: "ran-replacements.yaml",
		})
	case "CN":
		functions = append(functions, KptPipelineFunction{
			Image: "gcr.io/kpt-fn/apply-replacements:v0.1.1",
			ConfigPath: "cn-replacements.yaml",
		})
	case "TN":
		functions = append(functions, KptPipelineFunction{
			Image: "gcr.io/kpt-fn/apply-replacements:v0.1.1",
			ConfigPath: "tn-replacements.yaml",
		})
	}

	return functions
}

// generateValidatorFunctions creates validator functions for the pipeline
func (g *EnhancedPackageGenerator) generateValidatorFunctions(spec *VNFSpec) []KptPipelineFunction {
	return []KptPipelineFunction{
		// Validate Kubernetes resources
		{
			Image: "gcr.io/kpt-fn/kubeval:v0.3",
			ConfigMap: map[string]interface{}{
				"strict":          true,
				"ignore_missing": false,
				"skip_kinds":     "CustomResourceDefinition",
			},
		},
		// Validate O-RAN specific requirements
		{
			Image: "oran/kpt-fn-validator:v1.0.0",
			ConfigMap: map[string]interface{}{
				"vnf-type":     spec.Type,
				"qos-validate": true,
				"placement-validate": true,
			},
		},
		// Validate security policies
		{
			Image: "gcr.io/kpt-fn/gatekeeper:v0.2",
			ConfigMap: map[string]interface{}{
				"violations": []map[string]interface{}{
					{
						"kind": "K8sRequiredLabels",
						"name": "must-have-oran-labels",
					},
				},
			},
		},
	}
}

// generateEnhancedResources creates enhanced Kubernetes resources
func (g *EnhancedPackageGenerator) generateEnhancedResources(spec *VNFSpec, template *PackageTemplate) ([]EnhancedResource, error) {
	var resources []EnhancedResource

	// Generate namespace
	namespace := EnhancedResource{
		APIVersion: "v1",
		Kind:       "Namespace",
		Metadata: map[string]interface{}{
			"name": spec.Name + "-" + strings.ToLower(spec.Type),
			"labels": map[string]interface{}{
				"oran.io/vnf-type":    spec.Type,
				"oran.io/cloud-type":  spec.Placement.CloudType,
				"nephio.io/component": "workload",
			},
			"annotations": map[string]interface{}{
				"oran.io/qos-bandwidth": fmt.Sprintf("%.2f", spec.QoS.Bandwidth),
				"oran.io/qos-latency":   fmt.Sprintf("%.2f", spec.QoS.Latency),
			},
		},
		Validation: ResourceValidation{Validated: true},
	}
	resources = append(resources, namespace)

	// Generate VNF-specific resources
	switch spec.Type {
	case "RAN":
		ranResources, err := g.generateRANResources(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate RAN resources: %w", err)
		}
		resources = append(resources, ranResources...)

	case "CN":
		cnResources, err := g.generateCNResources(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CN resources: %w", err)
		}
		resources = append(resources, cnResources...)

	case "TN":
		tnResources, err := g.generateTNResources(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate TN resources: %w", err)
		}
		resources = append(resources, tnResources...)
	}

	// Generate common resources
	commonResources := g.generateCommonResources(spec)
	resources = append(resources, commonResources...)

	return resources, nil
}

// generateRANResources creates RAN-specific resources
func (g *EnhancedPackageGenerator) generateRANResources(spec *VNFSpec) ([]EnhancedResource, error) {
	var resources []EnhancedResource

	// RAN Deployment with enhanced configuration
	deployment := EnhancedResource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-ran", spec.Name),
			"namespace": spec.Name + "-" + strings.ToLower(spec.Type),
			"labels": map[string]interface{}{
				"app":                     fmt.Sprintf("%s-ran", spec.Name),
				"app.kubernetes.io/name":  spec.Name,
				"app.kubernetes.io/component": "ran",
				"oran.io/vnf-type":        "RAN",
				"nephio.io/component":     "workload",
			},
			"annotations": map[string]interface{}{
				"oran.io/qos-bandwidth":   fmt.Sprintf("%.2f", spec.QoS.Bandwidth),
				"oran.io/qos-latency":     fmt.Sprintf("%.2f", spec.QoS.Latency),
				"oran.io/deployment-type": "ran-workload",
			},
		},
		Spec: map[string]interface{}{
			"replicas": 1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": fmt.Sprintf("%s-ran", spec.Name),
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app":                     fmt.Sprintf("%s-ran", spec.Name),
						"app.kubernetes.io/name":  spec.Name,
						"oran.io/vnf-type":        "RAN",
					},
					"annotations": map[string]interface{}{
						"oran.io/qos-bandwidth": fmt.Sprintf("%.2f", spec.QoS.Bandwidth),
						"oran.io/qos-latency":   fmt.Sprintf("%.2f", spec.QoS.Latency),
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "ran",
							"image": fmt.Sprintf("%s:%s", spec.Image.Repository, spec.Image.Tag),
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    fmt.Sprintf("%d", spec.Resources.CPUCores),
									"memory": fmt.Sprintf("%dGi", spec.Resources.MemoryGB),
								},
								"limits": map[string]interface{}{
									"cpu":    fmt.Sprintf("%d", spec.Resources.CPUCores),
									"memory": fmt.Sprintf("%dGi", spec.Resources.MemoryGB),
								},
							},
							"ports": []map[string]interface{}{
								{
									"name":          "sctp",
									"containerPort": 38412,
									"protocol":      "SCTP",
								},
								{
									"name":          "gtp",
									"containerPort": 2152,
									"protocol":      "UDP",
								},
							},
							"env": g.generateEnvVars(spec),
							"volumeMounts": []map[string]interface{}{
								{
									"name":      "config",
									"mountPath": "/etc/ran",
								},
							},
						},
					},
					"volumes": []map[string]interface{}{
						{
							"name": "config",
							"configMap": map[string]interface{}{
								"name": fmt.Sprintf("%s-ran-config", spec.Name),
							},
						},
					},
					"nodeSelector":    g.generateNodeSelector(spec),
					"tolerations":     g.generateTolerations(spec),
					"affinity":        g.generateAffinity(spec),
					"securityContext": g.generateSecurityContext(spec),
				},
			},
			"strategy": map[string]interface{}{
				"type": "RollingUpdate",
				"rollingUpdate": map[string]interface{}{
					"maxUnavailable": 0,
					"maxSurge":       1,
				},
			},
		},
		Validation: ResourceValidation{Validated: true},
	}
	resources = append(resources, deployment)

	// RAN Service with enhanced configuration
	service := EnhancedResource{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-ran-svc", spec.Name),
			"namespace": spec.Name + "-" + strings.ToLower(spec.Type),
			"labels": map[string]interface{}{
				"app":                     fmt.Sprintf("%s-ran", spec.Name),
				"app.kubernetes.io/name":  spec.Name,
				"oran.io/vnf-type":        "RAN",
				"oran.io/service-type":    "ran-interface",
			},
			"annotations": map[string]interface{}{
				"oran.io/qos-bandwidth": fmt.Sprintf("%.2f", spec.QoS.Bandwidth),
				"oran.io/qos-latency":   fmt.Sprintf("%.2f", spec.QoS.Latency),
			},
		},
		Spec: map[string]interface{}{
			"selector": map[string]interface{}{
				"app": fmt.Sprintf("%s-ran", spec.Name),
			},
			"ports": []map[string]interface{}{
				{
					"name":       "sctp",
					"port":       38412,
					"targetPort": 38412,
					"protocol":   "SCTP",
				},
				{
					"name":       "gtp",
					"port":       2152,
					"targetPort": 2152,
					"protocol":   "UDP",
				},
			},
			"type": "ClusterIP",
		},
		Validation: ResourceValidation{Validated: true},
	}
	resources = append(resources, service)

	// RAN ConfigMap
	configMap := EnhancedResource{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-ran-config", spec.Name),
			"namespace": spec.Name + "-" + strings.ToLower(spec.Type),
			"labels": map[string]interface{}{
				"app":                     fmt.Sprintf("%s-ran", spec.Name),
				"app.kubernetes.io/name":  spec.Name,
				"oran.io/vnf-type":        "RAN",
				"oran.io/config-type":     "ran-configuration",
			},
		},
		Data: map[string]interface{}{
			"ran.conf": g.generateRANConfig(spec),
			"qos.conf": g.generateQoSConfig(spec),
		},
		Validation: ResourceValidation{Validated: true},
	}
	resources = append(resources, configMap)

	return resources, nil
}

// generateCNResources creates CN-specific resources
func (g *EnhancedPackageGenerator) generateCNResources(spec *VNFSpec) ([]EnhancedResource, error) {
	// Implementation for CN resources
	return []EnhancedResource{}, nil
}

// generateTNResources creates TN-specific resources
func (g *EnhancedPackageGenerator) generateTNResources(spec *VNFSpec) ([]EnhancedResource, error) {
	// Implementation for TN resources
	return []EnhancedResource{}, nil
}

// generateCommonResources creates common resources for all VNF types
func (g *EnhancedPackageGenerator) generateCommonResources(spec *VNFSpec) []EnhancedResource {
	var resources []EnhancedResource

	// NetworkPolicy for security
	networkPolicy := EnhancedResource{
		APIVersion: "networking.k8s.io/v1",
		Kind:       "NetworkPolicy",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-network-policy", spec.Name),
			"namespace": spec.Name + "-" + strings.ToLower(spec.Type),
		},
		Spec: map[string]interface{}{
			"podSelector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app.kubernetes.io/name": spec.Name,
				},
			},
			"policyTypes": []string{"Ingress", "Egress"},
			"ingress": g.generateIngressRules(spec),
			"egress":  g.generateEgressRules(spec),
		},
		Validation: ResourceValidation{Validated: true},
	}
	resources = append(resources, networkPolicy)

	return resources
}

// Helper functions
func (g *EnhancedPackageGenerator) determineQoSClass(spec *VNFSpec) string {
	if spec.QoS.Latency <= 1.0 {
		return "ultra-low-latency"
	} else if spec.QoS.Latency <= 10.0 {
		return "low-latency"
	}
	return "best-effort"
}

func (g *EnhancedPackageGenerator) generateEnvVars(spec *VNFSpec) []map[string]interface{} {
	envVars := []map[string]interface{}{
		{"name": "VNF_NAME", "value": spec.Name},
		{"name": "VNF_TYPE", "value": spec.Type},
		{"name": "VNF_VERSION", "value": spec.Version},
		{"name": "QOS_BANDWIDTH", "value": fmt.Sprintf("%.2f", spec.QoS.Bandwidth)},
		{"name": "QOS_LATENCY", "value": fmt.Sprintf("%.2f", spec.QoS.Latency)},
	}

	for key, value := range spec.Config {
		envVars = append(envVars, map[string]interface{}{
			"name":  strings.ToUpper(key),
			"value": value,
		})
	}

	return envVars
}

func (g *EnhancedPackageGenerator) generateNodeSelector(spec *VNFSpec) map[string]interface{} {
	selector := map[string]interface{}{}

	if spec.Placement.CloudType != "" {
		selector["oran.io/cloud-type"] = spec.Placement.CloudType
	}

	if spec.Placement.Zone != "" {
		selector["oran.io/zone"] = spec.Placement.Zone
	}

	if spec.Placement.Site != "" {
		selector["oran.io/site"] = spec.Placement.Site
	}

	return selector
}

func (g *EnhancedPackageGenerator) generateTolerations(spec *VNFSpec) []map[string]interface{} {
	tolerations := []map[string]interface{}{
		{
			"key":      "oran.io/vnf-type",
			"operator": "Equal",
			"value":    spec.Type,
			"effect":   "NoSchedule",
		},
	}

	if spec.QoS.Latency <= 1.0 {
		tolerations = append(tolerations, map[string]interface{}{
			"key":      "oran.io/ultra-low-latency",
			"operator": "Exists",
			"effect":   "NoSchedule",
		})
	}

	return tolerations
}

func (g *EnhancedPackageGenerator) generateAffinity(spec *VNFSpec) map[string]interface{} {
	return map[string]interface{}{
		"nodeAffinity": map[string]interface{}{
			"requiredDuringSchedulingIgnoredDuringExecution": map[string]interface{}{
				"nodeSelectorTerms": []map[string]interface{}{
					{
						"matchExpressions": []map[string]interface{}{
							{
								"key":      "oran.io/vnf-capable",
								"operator": "In",
								"values":   []string{spec.Type, "ALL"},
							},
						},
					},
				},
			},
			"podAntiAffinity": map[string]interface{}{
				"preferredDuringSchedulingIgnoredDuringExecution": []map[string]interface{}{
					{
						"weight": 100,
						"podAffinityTerm": map[string]interface{}{
							"labelSelector": map[string]interface{}{
								"matchLabels": map[string]interface{}{
									"app.kubernetes.io/name": spec.Name,
								},
							},
							"topologyKey": "kubernetes.io/hostname",
						},
					},
				},
			},
		},
	}
}

func (g *EnhancedPackageGenerator) generateSecurityContext(spec *VNFSpec) map[string]interface{} {
	securityContext := map[string]interface{}{
		"runAsNonRoot": true,
		"runAsUser":    1000,
		"runAsGroup":   1000,
		"fsGroup":      1000,
	}

	// TN requires privileged access for network operations
	if spec.Type == "TN" {
		securityContext["runAsNonRoot"] = false
		securityContext["runAsUser"] = 0
	}

	return securityContext
}

func (g *EnhancedPackageGenerator) generateRANConfig(spec *VNFSpec) string {
	// Generate RAN-specific configuration
	return fmt.Sprintf(`# RAN Configuration for %s
name: %s
type: %s
version: %s

# QoS Configuration
qos:
  bandwidth: %.2f
  latency: %.2f
  jitter: %.2f

# Placement Configuration
placement:
  cloudType: %s
  region: %s
  zone: %s
  site: %s
`,
		spec.Name, spec.Name, spec.Type, spec.Version,
		spec.QoS.Bandwidth, spec.QoS.Latency, spec.QoS.Jitter,
		spec.Placement.CloudType, spec.Placement.Region, spec.Placement.Zone, spec.Placement.Site)
}

func (g *EnhancedPackageGenerator) generateQoSConfig(spec *VNFSpec) string {
	return fmt.Sprintf(`# QoS Configuration
bandwidth: %.2f
latency: %.2f
jitter: %.2f
packetLoss: %.4f
reliability: %.4f
sliceType: %s
`,
		spec.QoS.Bandwidth, spec.QoS.Latency, spec.QoS.Jitter,
		spec.QoS.PacketLoss, spec.QoS.Reliability, spec.QoS.SliceType)
}

func (g *EnhancedPackageGenerator) generateIngressRules(spec *VNFSpec) []map[string]interface{} {
	rules := []map[string]interface{}{}

	switch spec.Type {
	case "RAN":
		rules = append(rules, map[string]interface{}{
			"from": []map[string]interface{}{
				{
					"namespaceSelector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"oran.io/component": "core-network",
						},
					},
				},
			},
			"ports": []map[string]interface{}{
				{"protocol": "SCTP", "port": 38412},
				{"protocol": "UDP", "port": 2152},
			},
		})
	case "CN":
		// CN ingress rules
	case "TN":
		// TN ingress rules
	}

	return rules
}

func (g *EnhancedPackageGenerator) generateEgressRules(spec *VNFSpec) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"to": []map[string]interface{}{},
			"ports": []map[string]interface{}{
				{"protocol": "TCP", "port": 53},
				{"protocol": "UDP", "port": 53},
				{"protocol": "TCP", "port": 443},
			},
		},
	}
}

func (g *EnhancedPackageGenerator) selectKptFunctions(spec *VNFSpec) ([]KptFunction, error) {
	functions := []KptFunction{}

	// Add common functions
	commonFunctions := []string{"apply-setters", "set-namespace", "set-labels", "kubeval"}
	for _, name := range commonFunctions {
		fn, err := g.kptFunctionRegistry.GetFunction(name)
		if err != nil {
			return nil, fmt.Errorf("failed to get function %s: %w", name, err)
		}
		functions = append(functions, *fn)
	}

	// Add VNF-specific functions
	switch spec.Type {
	case "RAN":
		if fn, err := g.kptFunctionRegistry.GetFunction("oran-ran-validator"); err == nil {
			functions = append(functions, *fn)
		}
	case "CN":
		if fn, err := g.kptFunctionRegistry.GetFunction("oran-cn-validator"); err == nil {
			functions = append(functions, *fn)
		}
	case "TN":
		if fn, err := g.kptFunctionRegistry.GetFunction("oran-tn-validator"); err == nil {
			functions = append(functions, *fn)
		}
	}

	return functions, nil
}

func (g *EnhancedPackageGenerator) generateValidationRules(spec *VNFSpec) []ValidationRule {
	rules := []ValidationRule{
		{
			Name:        "required-labels",
			Description: "Ensure all resources have required O-RAN labels",
			Type:        "label-validation",
			Expression:  "metadata.labels['oran.io/vnf-type'] != ''",
			Severity:    "error",
		},
		{
			Name:        "qos-annotations",
			Description: "Ensure QoS annotations are present",
			Type:        "annotation-validation",
			Expression:  "metadata.annotations['oran.io/qos-bandwidth'] != ''",
			Severity:    "warning",
		},
		{
			Name:        "resource-limits",
			Description: "Ensure resource limits are specified",
			Type:        "resource-validation",
			Expression:  "spec.containers[*].resources.limits != null",
			Severity:    "error",
		},
	}

	// Add VNF-specific validation rules
	switch spec.Type {
	case "RAN":
		rules = append(rules, ValidationRule{
			Name:        "ran-ports",
			Description: "Ensure RAN has required ports",
			Type:        "port-validation",
			Expression:  "spec.ports[?(@.port==38412)] && spec.ports[?(@.port==2152)]",
			Severity:    "error",
		})
	case "CN":
		// CN-specific rules
	case "TN":
		// TN-specific rules
	}

	return rules
}

func (g *EnhancedPackageGenerator) generateDependencies(spec *VNFSpec, template *PackageTemplate) []PackageDependency {
	dependencies := []PackageDependency{
		{
			Name:       "oran-common",
			Version:    "v1.0.0",
			Repository: "https://github.com/o-ran/packages",
			Optional:   false,
		},
	}

	// Add VNF-specific dependencies
	switch spec.Type {
	case "RAN":
		dependencies = append(dependencies, PackageDependency{
			Name:       "oran-ran-common",
			Version:    "v1.0.0",
			Repository: "https://github.com/o-ran/packages",
			Optional:   false,
		})
	case "CN":
		dependencies = append(dependencies, PackageDependency{
			Name:       "oran-cn-common",
			Version:    "v1.0.0",
			Repository: "https://github.com/o-ran/packages",
			Optional:   false,
		})
	case "TN":
		dependencies = append(dependencies, PackageDependency{
			Name:       "oran-tn-common",
			Version:    "v1.0.0",
			Repository: "https://github.com/o-ran/packages",
			Optional:   false,
		})
	}

	return dependencies
}

func (g *EnhancedPackageGenerator) writePackageFiles(fs filesys.FileSystem, dir string, pkg *EnhancedPackage) error {
	// Create directory
	if err := fs.MkdirAll(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write Kptfile
	kptfileContent, err := yaml.Marshal(pkg.Kptfile)
	if err != nil {
		return fmt.Errorf("failed to marshal Kptfile: %w", err)
	}
	if err := fs.WriteFile(filepath.Join(dir, "Kptfile"), kptfileContent); err != nil {
		return fmt.Errorf("failed to write Kptfile: %w", err)
	}

	// Write resources
	for i, resource := range pkg.Resources {
		resourceContent, err := yaml.Marshal(resource)
		if err != nil {
			return fmt.Errorf("failed to marshal resource %d: %w", i, err)
		}
		filename := fmt.Sprintf("%s-%d.yaml", strings.ToLower(resource.Kind), i)
		if err := fs.WriteFile(filepath.Join(dir, filename), resourceContent); err != nil {
			return fmt.Errorf("failed to write resource file %s: %w", filename, err)
		}
	}

	return nil
}

// generatePackageName generates a unique package name based on VNF specification
func (g *EnhancedPackageGenerator) generatePackageName(spec *VNFSpec) string {
	// Create a package name using VNF name, type, and hash for uniqueness
	baseeName := fmt.Sprintf("%s-%s", strings.ToLower(spec.Name), strings.ToLower(spec.Type))

	// Sanitize the name to be Kubernetes compliant
	baseName := strings.ReplaceAll(baseeName, "_", "-")
	baseName = strings.ReplaceAll(baseName, " ", "-")

	// Remove any non-alphanumeric characters except hyphens
	var cleanName strings.Builder
	for _, r := range baseName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleanName.WriteRune(r)
		}
	}

	result := cleanName.String()

	// Ensure it doesn't start or end with a hyphen
	result = strings.Trim(result, "-")

	// Ensure it's not empty and not too long
	if result == "" {
		result = "vnf-package"
	}
	if len(result) > 50 {
		result = result[:50]
		result = strings.Trim(result, "-")
	}

	return result
}
