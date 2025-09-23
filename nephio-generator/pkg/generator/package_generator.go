package generator

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

// PackageGenerator generates Nephio packages from VNF specifications
type PackageGenerator struct {
	templateRegistry TemplateRegistry
	outputDir        string
	packagePrefix    string
}

// TemplateRegistry provides access to package templates
type TemplateRegistry interface {
	GetTemplate(vnfType, templateType string) (*PackageTemplate, error)
	ListTemplates() ([]TemplateInfo, error)
}

// PackageTemplate represents a Nephio package template
type PackageTemplate struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	VNFType      string                 `json:"vnfType"`
	Type         TemplateType           `json:"type"`
	Files        []TemplateFile         `json:"files"`
	Variables    map[string]Variable    `json:"variables"`
	Dependencies []string               `json:"dependencies"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TemplateType represents the type of package template
type TemplateType string

const (
	TemplateTypeKustomize TemplateType = "kustomize"
	TemplateTypeHelm      TemplateType = "helm"
	TemplateTypeKpt       TemplateType = "kpt"
)

// TemplateFile represents a file in the package template
type TemplateFile struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	IsTemplate  bool   `json:"isTemplate"`
	Executable  bool   `json:"executable"`
}

// Variable represents a template variable
type Variable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	Default      interface{} `json:"default"`
	Required     bool        `json:"required"`
	Constraints  []string    `json:"constraints,omitempty"`
}

// TemplateInfo provides information about available templates
type TemplateInfo struct {
	Name        string       `json:"name"`
	VNFType     string       `json:"vnfType"`
	Type        TemplateType `json:"type"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
}

// VNFSpec represents VNF specification for package generation
type VNFSpec struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	QoS       QoSRequirements   `json:"qos"`
	Placement PlacementSpec     `json:"placement"`
	Resources ResourceSpec      `json:"resources"`
	Config    map[string]string `json:"config"`
	Image     ImageSpec         `json:"image"`
}

// QoSRequirements defines QoS parameters
type QoSRequirements struct {
	Bandwidth   float64 `json:"bandwidth"`
	Latency     float64 `json:"latency"`
	Jitter      float64 `json:"jitter,omitempty"`
	PacketLoss  float64 `json:"packetLoss,omitempty"`
	Reliability float64 `json:"reliability,omitempty"`
	SliceType   string  `json:"sliceType,omitempty"`
}

// PlacementSpec defines placement requirements
type PlacementSpec struct {
	CloudType string   `json:"cloudType"`
	Region    string   `json:"region,omitempty"`
	Zone      string   `json:"zone,omitempty"`
	Site      string   `json:"site,omitempty"`
	Labels    []string `json:"labels,omitempty"`
}

// ResourceSpec defines resource requirements
type ResourceSpec struct {
	CPUCores  int `json:"cpuCores,omitempty"`
	MemoryGB  int `json:"memoryGB,omitempty"`
	StorageGB int `json:"storageGB,omitempty"`
}

// ImageSpec defines container image configuration
type ImageSpec struct {
	Repository string   `json:"repository"`
	Tag        string   `json:"tag"`
	PullPolicy string   `json:"pullPolicy,omitempty"`
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

// GeneratedPackage represents a generated Nephio package
type GeneratedPackage struct {
	Name         string            `json:"name"`
	Path         string            `json:"path"`
	Type         TemplateType      `json:"type"`
	VNFSpec      VNFSpec           `json:"vnfSpec"`
	Files        []GeneratedFile   `json:"files"`
	Dependencies []string          `json:"dependencies"`
	Metadata     map[string]string `json:"metadata"`
}

// GeneratedFile represents a file in the generated package
type GeneratedFile struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Size       int64  `json:"size"`
	Executable bool   `json:"executable"`
}

// NewPackageGenerator creates a new package generator
func NewPackageGenerator(registry TemplateRegistry, outputDir, packagePrefix string) *PackageGenerator {
	return &PackageGenerator{
		templateRegistry: registry,
		outputDir:        outputDir,
		packagePrefix:    packagePrefix,
	}
}

// GeneratePackage generates a Nephio package from VNF specification
func (g *PackageGenerator) GeneratePackage(spec *VNFSpec, templateType TemplateType) (*GeneratedPackage, error) {
	// Get template for VNF type
	template, err := g.templateRegistry.GetTemplate(spec.Type, string(templateType))
	if err != nil {
		return nil, fmt.Errorf("failed to get template for VNF type %s: %w", spec.Type, err)
	}

	// Generate package name
	packageName := g.generatePackageName(spec)

	// Create package structure
	pkg := &GeneratedPackage{
		Name:         packageName,
		Path:         filepath.Join(g.outputDir, packageName),
		Type:         templateType,
		VNFSpec:      *spec,
		Dependencies: template.Dependencies,
		Metadata: map[string]string{
			"generated.by":      "nephio-generator",
			"generated.at":      time.Now().Format(time.RFC3339),
			"vnf.type":          spec.Type,
			"vnf.name":          spec.Name,
			"template.name":     template.Name,
			"template.version":  template.Version,
		},
	}

	// Generate template context
	context := g.buildTemplateContext(spec)

	// Process template files
	for _, templateFile := range template.Files {
		generatedFile, err := g.processTemplateFile(&templateFile, context, packageName)
		if err != nil {
			return nil, fmt.Errorf("failed to process template file %s: %w", templateFile.Path, err)
		}
		pkg.Files = append(pkg.Files, *generatedFile)
	}

	// Generate package-specific files based on template type
	switch templateType {
	case TemplateTypeKustomize:
		kustomizeFiles, err := g.generateKustomizeFiles(spec, context)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Kustomize files: %w", err)
		}
		pkg.Files = append(pkg.Files, kustomizeFiles...)

	case TemplateTypeKpt:
		kptFiles, err := g.generateKptFiles(spec, context)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Kpt files: %w", err)
		}
		pkg.Files = append(pkg.Files, kptFiles...)

	case TemplateTypeHelm:
		helmFiles, err := g.generateHelmFiles(spec, context)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Helm files: %w", err)
		}
		pkg.Files = append(pkg.Files, helmFiles...)
	}

	return pkg, nil
}

// GenerateMultiClusterPackage generates packages for multi-cluster deployment
func (g *PackageGenerator) GenerateMultiClusterPackage(spec *VNFSpec, clusters []string, templateType TemplateType) ([]*GeneratedPackage, error) {
	var packages []*GeneratedPackage

	for _, cluster := range clusters {
		// Create cluster-specific spec
		clusterSpec := *spec
		clusterSpec.Name = fmt.Sprintf("%s-%s", spec.Name, cluster)

		// Add cluster-specific placement
		clusterSpec.Placement.Site = cluster

		// Add cluster label
		if clusterSpec.Placement.Labels == nil {
			clusterSpec.Placement.Labels = []string{}
		}
		clusterSpec.Placement.Labels = append(clusterSpec.Placement.Labels, fmt.Sprintf("cluster=%s", cluster))

		// Generate package for this cluster
		pkg, err := g.GeneratePackage(&clusterSpec, templateType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate package for cluster %s: %w", cluster, err)
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// generatePackageName creates a package name from VNF spec
func (g *PackageGenerator) generatePackageName(spec *VNFSpec) string {
	name := strings.ToLower(spec.Name)
	vnfType := strings.ToLower(spec.Type)
	cloudType := strings.ToLower(spec.Placement.CloudType)

	if g.packagePrefix != "" {
		return fmt.Sprintf("%s-%s-%s-%s", g.packagePrefix, name, vnfType, cloudType)
	}
	return fmt.Sprintf("%s-%s-%s", name, vnfType, cloudType)
}

// buildTemplateContext creates template substitution context
func (g *PackageGenerator) buildTemplateContext(spec *VNFSpec) map[string]interface{} {
	return map[string]interface{}{
		"vnf": map[string]interface{}{
			"name":    spec.Name,
			"type":    spec.Type,
			"version": spec.Version,
		},
		"qos": map[string]interface{}{
			"bandwidth":   spec.QoS.Bandwidth,
			"latency":     spec.QoS.Latency,
			"jitter":      spec.QoS.Jitter,
			"packetLoss":  spec.QoS.PacketLoss,
			"reliability": spec.QoS.Reliability,
			"sliceType":   spec.QoS.SliceType,
		},
		"placement": map[string]interface{}{
			"cloudType": spec.Placement.CloudType,
			"region":    spec.Placement.Region,
			"zone":      spec.Placement.Zone,
			"site":      spec.Placement.Site,
			"labels":    spec.Placement.Labels,
		},
		"resources": map[string]interface{}{
			"cpuCores":  spec.Resources.CPUCores,
			"memoryGB":  spec.Resources.MemoryGB,
			"storageGB": spec.Resources.StorageGB,
		},
		"image": map[string]interface{}{
			"repository": spec.Image.Repository,
			"tag":        spec.Image.Tag,
			"pullPolicy": spec.Image.PullPolicy,
			"pullSecrets": spec.Image.PullSecrets,
		},
		"config": spec.Config,
		"timestamp": time.Now().Format(time.RFC3339),
		"packageName": g.generatePackageName(spec),
	}
}

// processTemplateFile processes a single template file
func (g *PackageGenerator) processTemplateFile(templateFile *TemplateFile, context map[string]interface{}, packageName string) (*GeneratedFile, error) {
	content := templateFile.Content

	// Apply template substitution if needed
	if templateFile.IsTemplate {
		var err error
		content, err = g.applyTemplate(content, context)
		if err != nil {
			return nil, fmt.Errorf("failed to apply template substitution: %w", err)
		}
	}

	return &GeneratedFile{
		Path:       templateFile.Path,
		Content:    content,
		Size:       int64(len(content)),
		Executable: templateFile.Executable,
	}, nil
}

// generateKustomizeFiles creates Kustomize-specific files
func (g *PackageGenerator) generateKustomizeFiles(spec *VNFSpec, context map[string]interface{}) ([]GeneratedFile, error) {
	var files []GeneratedFile

	// Generate kustomization.yaml
	kustomization := map[string]interface{}{
		"apiVersion": "kustomize.config.k8s.io/v1beta1",
		"kind":       "Kustomization",
		"metadata": map[string]interface{}{
			"name": spec.Name,
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		"resources": []string{
			"deployment.yaml",
			"service.yaml",
			"configmap.yaml",
		},
		"commonLabels": map[string]string{
			"app":                         spec.Name,
			"app.kubernetes.io/name":      spec.Name,
			"app.kubernetes.io/component": spec.Type,
			"oran.io/vnf-type":           spec.Type,
			"oran.io/cloud-type":         spec.Placement.CloudType,
		},
		"commonAnnotations": map[string]string{
			"oran.io/qos-bandwidth": fmt.Sprintf("%.2f", spec.QoS.Bandwidth),
			"oran.io/qos-latency":   fmt.Sprintf("%.2f", spec.QoS.Latency),
		},
		"images": []map[string]interface{}{
			{
				"name":    spec.Name,
				"newName": spec.Image.Repository,
				"newTag":  spec.Image.Tag,
			},
		},
	}

	// Add patches for QoS and placement
	if spec.QoS.Jitter > 0 || spec.QoS.PacketLoss > 0 {
		kustomization["patchesStrategicMerge"] = []string{
			"qos-patch.yaml",
		}

		// Generate QoS patch
		qosPatch := map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": spec.Name,
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]string{
							"oran.io/qos-jitter":      fmt.Sprintf("%.2f", spec.QoS.Jitter),
							"oran.io/qos-packet-loss": fmt.Sprintf("%.4f", spec.QoS.PacketLoss),
						},
					},
				},
			},
		}

		qosPatchContent, err := yaml.Marshal(qosPatch)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal QoS patch: %w", err)
		}

		files = append(files, GeneratedFile{
			Path:    "qos-patch.yaml",
			Content: string(qosPatchContent),
			Size:    int64(len(qosPatchContent)),
		})
	}

	kustomizationContent, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kustomization: %w", err)
	}

	files = append(files, GeneratedFile{
		Path:    "kustomization.yaml",
		Content: string(kustomizationContent),
		Size:    int64(len(kustomizationContent)),
	})

	return files, nil
}

// generateKptFiles creates Kpt-specific files
func (g *PackageGenerator) generateKptFiles(spec *VNFSpec, context map[string]interface{}) ([]GeneratedFile, error) {
	var files []GeneratedFile

	// Generate Kptfile
	kptfile := map[string]interface{}{
		"apiVersion": "kpt.dev/v1",
		"kind":       "Kptfile",
		"metadata": map[string]interface{}{
			"name": spec.Name,
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		"info": map[string]interface{}{
			"description": fmt.Sprintf("Nephio package for %s VNF (%s)", spec.Name, spec.Type),
			"site":        "https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing",
		},
		"pipeline": map[string]interface{}{
			"mutators": []map[string]interface{}{
				{
					"image": "gcr.io/kpt-fn/apply-replacements:v0.1.1",
					"configMap": map[string]interface{}{
						"replacements": []map[string]interface{}{
							{
								"source": map[string]interface{}{
									"objref": map[string]interface{}{
										"kind": "ConfigMap",
										"name": fmt.Sprintf("%s-config", spec.Name),
									},
									"fieldref": "data.vnf-name",
								},
								"targets": []map[string]interface{}{
									{
										"select": map[string]interface{}{
											"kind": "Deployment",
										},
										"fieldPaths": []string{"metadata.name"},
									},
								},
							},
						},
					},
				},
				{
					"image": "gcr.io/kpt-fn/set-labels:v0.2.0",
					"configMap": map[string]interface{}{
						"oran.io/vnf-type":   spec.Type,
						"oran.io/cloud-type": spec.Placement.CloudType,
					},
				},
			},
		},
	}

	kptfileContent, err := yaml.Marshal(kptfile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Kptfile: %w", err)
	}

	files = append(files, GeneratedFile{
		Path:    "Kptfile",
		Content: string(kptfileContent),
		Size:    int64(len(kptfileContent)),
	})

	return files, nil
}

// generateHelmFiles creates Helm-specific files
func (g *PackageGenerator) generateHelmFiles(spec *VNFSpec, context map[string]interface{}) ([]GeneratedFile, error) {
	var files []GeneratedFile

	// Generate Chart.yaml
	chart := map[string]interface{}{
		"apiVersion":  "v2",
		"name":        spec.Name,
		"description": fmt.Sprintf("Helm chart for %s VNF (%s)", spec.Name, spec.Type),
		"type":        "application",
		"version":     "0.1.0",
		"appVersion":  spec.Version,
		"keywords":    []string{"oran", "vnf", spec.Type, spec.Placement.CloudType},
		"maintainers": []map[string]string{
			{
				"name":  "O-RAN Intent MANO",
				"email": "maintainer@oran-mano.io",
			},
		},
		"annotations": map[string]string{
			"oran.io/vnf-type":       spec.Type,
			"oran.io/cloud-type":     spec.Placement.CloudType,
			"oran.io/qos-bandwidth":  fmt.Sprintf("%.2f", spec.QoS.Bandwidth),
			"oran.io/qos-latency":    fmt.Sprintf("%.2f", spec.QoS.Latency),
		},
	}

	chartContent, err := yaml.Marshal(chart)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Chart.yaml: %w", err)
	}

	files = append(files, GeneratedFile{
		Path:    "Chart.yaml",
		Content: string(chartContent),
		Size:    int64(len(chartContent)),
	})

	// Generate values.yaml
	values := map[string]interface{}{
		"nameOverride":     spec.Name,
		"fullnameOverride": spec.Name,
		"image": map[string]interface{}{
			"repository": spec.Image.Repository,
			"pullPolicy": spec.Image.PullPolicy,
			"tag":        spec.Image.Tag,
		},
		"imagePullSecrets": spec.Image.PullSecrets,
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
		"qos": map[string]interface{}{
			"bandwidth":   spec.QoS.Bandwidth,
			"latency":     spec.QoS.Latency,
			"jitter":      spec.QoS.Jitter,
			"packetLoss":  spec.QoS.PacketLoss,
			"reliability": spec.QoS.Reliability,
			"sliceType":   spec.QoS.SliceType,
		},
		"placement": map[string]interface{}{
			"cloudType": spec.Placement.CloudType,
			"region":    spec.Placement.Region,
			"zone":      spec.Placement.Zone,
			"site":      spec.Placement.Site,
		},
		"config": spec.Config,
	}

	valuesContent, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal values.yaml: %w", err)
	}

	files = append(files, GeneratedFile{
		Path:    "values.yaml",
		Content: string(valuesContent),
		Size:    int64(len(valuesContent)),
	})

	return files, nil
}

// applyTemplate applies template substitution to content
func (g *PackageGenerator) applyTemplate(content string, context map[string]interface{}) (string, error) {
	// Simple template substitution - in practice, you'd use a proper template engine
	result := content

	// Replace common template variables
	if vnfName, ok := context["packageName"].(string); ok {
		result = strings.ReplaceAll(result, "{{.PackageName}}", vnfName)
	}

	if vnf, ok := context["vnf"].(map[string]interface{}); ok {
		if name, ok := vnf["name"].(string); ok {
			result = strings.ReplaceAll(result, "{{.VNF.Name}}", name)
		}
		if vnfType, ok := vnf["type"].(string); ok {
			result = strings.ReplaceAll(result, "{{.VNF.Type}}", vnfType)
		}
	}

	if placement, ok := context["placement"].(map[string]interface{}); ok {
		if cloudType, ok := placement["cloudType"].(string); ok {
			result = strings.ReplaceAll(result, "{{.Placement.CloudType}}", cloudType)
		}
	}

	return result, nil
}