// Package generator implements the Nephio package generation logic
// integrating with the existing VNF operator and placement decisions
package nephio

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
)

// DefaultPackageGenerator implements PackageGenerator interface
type DefaultPackageGenerator struct {
	PackageCatalog    PackageCatalog
	TemplateRenderer  TemplateRenderer
	ResourceValidator ResourceValidator
}

// Package represents a complete Nephio package
type Package struct {
	Metadata     PackageMetadata                `json:"metadata"`
	Resources    []unstructured.Unstructured    `json:"resources"`
	Dependencies []PackageDependency            `json:"dependencies"`
	Targets      []DeploymentTarget             `json:"targets"`
	Kustomize    *Kustomization                 `json:"kustomize,omitempty"`
}

// PackageMetadata contains package identification and versioning
type PackageMetadata struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Vendor       string            `json:"vendor"`
	Category     string            `json:"category"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// PackageDependency represents a dependency on another package
type PackageDependency struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Optional bool   `json:"optional"`
	Scope    string `json:"scope"` // "runtime", "build", "test"
}

// DeploymentTarget specifies where the package should be deployed
type DeploymentTarget struct {
	ClusterName string            `json:"clusterName"`
	Namespace   string            `json:"namespace"`
	Site        string            `json:"site"`
	CloudType   string            `json:"cloudType"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// PackageCatalog interface for package template management
type PackageCatalog interface {
	GetTemplate(vnfType, version string) (*PackageTemplate, error)
	ListTemplates(category string) ([]*PackageTemplate, error)
	ValidateTemplate(template *PackageTemplate) error
}

// PackageTemplate represents a reusable package template
type PackageTemplate struct {
	Metadata     PackageMetadata           `json:"metadata"`
	ManifestPath string                   `json:"manifestPath"`
	ConfigSchema map[string]interface{}   `json:"configSchema"`
	Resources    []ResourceTemplate       `json:"resources"`
	Dependencies []PackageDependency      `json:"dependencies"`
}

// ResourceTemplate represents a template for Kubernetes resources
type ResourceTemplate struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Template   map[string]interface{} `json:"template"`
	ConfigPath string                 `json:"configPath,omitempty"`
}

// TemplateRenderer interface for template rendering
type TemplateRenderer interface {
	RenderTemplate(template *PackageTemplate, config TemplateConfig) ([]unstructured.Unstructured, error)
	ValidateConfig(schema map[string]interface{}, config TemplateConfig) error
}

// TemplateConfig contains configuration values for template rendering
type TemplateConfig struct {
	VNFSpec       manov1alpha1.VNFSpec `json:"vnfSpec"`
	PlacementInfo PlacementInfo        `json:"placementInfo"`
	QoSProfile    QoSProfile          `json:"qosProfile"`
	ClusterInfo   ClusterInfo         `json:"clusterInfo"`
	NetworkConfig NetworkConfig       `json:"networkConfig"`
}

// PlacementInfo contains placement decision information
type PlacementInfo struct {
	SiteID      string            `json:"siteId"`
	ClusterName string            `json:"clusterName"`
	CloudType   string            `json:"cloudType"`
	Region      string            `json:"region"`
	Zone        string            `json:"zone"`
	Coordinates Coordinates       `json:"coordinates"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// Coordinates represents geographic coordinates
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ClusterInfo contains target cluster information
type ClusterInfo struct {
	Name         string            `json:"name"`
	Endpoint     string            `json:"endpoint"`
	Version      string            `json:"version"`
	CNI          string            `json:"cni"`
	StorageClass string            `json:"storageClass"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// NetworkConfig contains network-specific configuration
type NetworkConfig struct {
	Interfaces []NetworkInterface    `json:"interfaces"`
	Routes     []Route              `json:"routes,omitempty"`
	QoSPolicies []QoSPolicy         `json:"qosPolicies,omitempty"`
	SecurityGroups []SecurityGroup  `json:"securityGroups,omitempty"`
}

// NetworkInterface represents a network interface configuration
type NetworkInterface struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "management", "data", "signaling"
	CIDR     string `json:"cidr"`
	VLAN     int    `json:"vlan,omitempty"`
	MTU      int    `json:"mtu,omitempty"`
}

// Route represents a network route
type Route struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Interface   string `json:"interface"`
	Metric      int    `json:"metric,omitempty"`
}

// QoSPolicy represents a Quality of Service policy
type QoSPolicy struct {
	Name        string  `json:"name"`
	Class       string  `json:"class"`
	Bandwidth   string  `json:"bandwidth"`
	Priority    int     `json:"priority"`
	MatchRules  []string `json:"matchRules"`
}

// SecurityGroup represents network security rules
type SecurityGroup struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Rules       []SecurityRule `json:"rules"`
}

// SecurityRule represents a security rule
type SecurityRule struct {
	Direction string `json:"direction"` // "ingress", "egress"
	Protocol  string `json:"protocol"`  // "tcp", "udp", "icmp", "all"
	Port      string `json:"port,omitempty"`
	Source    string `json:"source,omitempty"`
	Target    string `json:"target,omitempty"`
	Action    string `json:"action"` // "allow", "deny"
}

// ResourceValidator interface for resource validation
type ResourceValidator interface {
	ValidateResources(resources []unstructured.Unstructured) error
	ValidateDeployment(target DeploymentTarget, resources []unstructured.Unstructured) error
}

// GeneratePackages generates Nephio packages from NetworkSliceIntent
func (g *DefaultPackageGenerator) GeneratePackages(ctx context.Context, intent *NetworkSliceIntent) ([]*Package, error) {
	packages := make([]*Package, 0)

	// Generate individual VNF packages
	for _, nfSpec := range intent.Spec.NetworkFunctions {
		pkg, err := g.generateVNFPackage(ctx, nfSpec, intent)
		if err != nil {
			return nil, fmt.Errorf("failed to generate package for %s: %w", nfSpec.Type, err)
		}
		packages = append(packages, pkg)
	}

	// Generate network slice orchestration package
	slicePackage, err := g.generateSliceOrchestrationPackage(ctx, intent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate slice orchestration package: %w", err)
	}
	packages = append(packages, slicePackage)

	// Generate ConfigSync packages for each target cluster
	for _, clusterName := range intent.Spec.TargetClusters {
		configSyncPkg, err := g.generateConfigSyncPackage(ctx, intent, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ConfigSync package for %s: %w", clusterName, err)
		}
		packages = append(packages, configSyncPkg)
	}

	return packages, nil
}

// generateVNFPackage creates a package for a specific VNF
func (g *DefaultPackageGenerator) generateVNFPackage(ctx context.Context, nfSpec NetworkFunctionSpec, intent *NetworkSliceIntent) (*Package, error) {
	// Get template from catalog
	template, err := g.PackageCatalog.GetTemplate(nfSpec.Type, "latest")
	if err != nil {
		return nil, fmt.Errorf("failed to get template for %s: %w", nfSpec.Type, err)
	}

	// Prepare template configuration
	config := TemplateConfig{
		VNFSpec: manov1alpha1.VNFSpec{
			Name: fmt.Sprintf("%s-%s", strings.ToLower(nfSpec.Type), intent.Name),
			Type: manov1alpha1.VNFType(nfSpec.Type),
			QoS: manov1alpha1.QoSRequirements{
				Bandwidth:   parseFloat64(intent.Spec.QoSProfile.Bandwidth),
				Latency:     parseFloat64(intent.Spec.QoSProfile.Latency),
				SliceType:   intent.Spec.QoSProfile.SliceType,
			},
			Placement: manov1alpha1.PlacementRequirements{
				CloudType: nfSpec.Placement.CloudType,
				Region:    nfSpec.Placement.Region,
				Zone:      nfSpec.Placement.Zone,
				Site:      nfSpec.Placement.SiteID,
			},
			Resources: manov1alpha1.ResourceRequirements{
				CPUCores:  nfSpec.Resources.CPUCores,
				MemoryGB:  nfSpec.Resources.MemoryGB,
				StorageGB: nfSpec.Resources.StorageGB,
			},
			Config: nfSpec.Config,
		},
		PlacementInfo: PlacementInfo{
			SiteID:      nfSpec.Placement.SiteID,
			ClusterName: g.getTargetCluster(nfSpec.Placement),
			CloudType:   nfSpec.Placement.CloudType,
			Region:      nfSpec.Placement.Region,
			Zone:        nfSpec.Placement.Zone,
		},
		QoSProfile: intent.Spec.QoSProfile,
		NetworkConfig: g.generateNetworkConfig(nfSpec.Type, intent.Spec.QoSProfile),
	}

	// Validate configuration
	if err := g.TemplateRenderer.ValidateConfig(template.ConfigSchema, config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Render template
	resources, err := g.TemplateRenderer.RenderTemplate(template, config)
	if err != nil {
		return nil, fmt.Errorf("template rendering failed: %w", err)
	}

	// Create package
	pkg := &Package{
		Metadata: PackageMetadata{
			Name:        fmt.Sprintf("%s-%s", strings.ToLower(nfSpec.Type), intent.Name),
			Version:     "v1.0.0",
			Description: fmt.Sprintf("%s network function for slice %s", nfSpec.Type, intent.Name),
			Vendor:      template.Metadata.Vendor,
			Category:    template.Metadata.Category,
			Labels: map[string]string{
				"slice-intent":     intent.Name,
				"vnf-type":         nfSpec.Type,
				"cloud-type":       nfSpec.Placement.CloudType,
				"generated-by":     "oran-mano-nephio-adapter",
			},
		},
		Resources:    resources,
		Dependencies: template.Dependencies,
		Targets: []DeploymentTarget{
			{
				ClusterName: config.PlacementInfo.ClusterName,
				Namespace:   fmt.Sprintf("slice-%s", intent.Name),
				Site:        nfSpec.Placement.SiteID,
				CloudType:   nfSpec.Placement.CloudType,
				Labels: map[string]string{
					"vnf-type": nfSpec.Type,
				},
			},
		},
		Kustomize: g.generateKustomization(nfSpec.Type, intent.Name),
	}

	// Validate package
	if err := g.ResourceValidator.ValidateResources(pkg.Resources); err != nil {
		return nil, fmt.Errorf("resource validation failed: %w", err)
	}

	return pkg, nil
}

// generateSliceOrchestrationPackage creates the top-level slice orchestration package
func (g *DefaultPackageGenerator) generateSliceOrchestrationPackage(ctx context.Context, intent *NetworkSliceIntent) (*Package, error) {
	// Create NetworkSliceIntent resource
	sliceIntent := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "nf.nephio.org/v1alpha1",
			"kind":       "NetworkSliceIntent",
			"metadata": map[string]interface{}{
				"name":      intent.Name,
				"namespace": "default",
				"labels": map[string]interface{}{
					"slice-type":   intent.Spec.QoSProfile.SliceType,
					"generated-by": "oran-mano-nephio-adapter",
				},
			},
			"spec": intent.Spec,
		},
	}

	// Create slice monitoring configuration
	monitoringConfig := g.generateSliceMonitoringConfig(intent)

	// Create slice network policy
	networkPolicy := g.generateSliceNetworkPolicy(intent)

	resources := []unstructured.Unstructured{
		*sliceIntent,
		*monitoringConfig,
		*networkPolicy,
	}

	pkg := &Package{
		Metadata: PackageMetadata{
			Name:        fmt.Sprintf("slice-orchestration-%s", intent.Name),
			Version:     "v1.0.0",
			Description: fmt.Sprintf("Orchestration package for network slice %s", intent.Name),
			Category:    "slice-orchestration",
			Labels: map[string]string{
				"slice-intent":   intent.Name,
				"slice-type":     intent.Spec.QoSProfile.SliceType,
				"generated-by":   "oran-mano-nephio-adapter",
			},
		},
		Resources: resources,
		Targets: []DeploymentTarget{
			{
				ClusterName: "management-cluster",
				Namespace:   "nephio-system",
				Site:        "management",
				CloudType:   "central",
			},
		},
	}

	return pkg, nil
}

// generateConfigSyncPackage creates ConfigSync packages for cluster-specific deployments
func (g *DefaultPackageGenerator) generateConfigSyncPackage(ctx context.Context, intent *NetworkSliceIntent, clusterName string) (*Package, error) {
	// Create RootSync configuration
	rootSync := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "configsync.gke.io/v1beta1",
			"kind":       "RootSync",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("slice-%s-sync", intent.Name),
				"namespace": "config-management-system",
			},
			"spec": map[string]interface{}{
				"sourceFormat": "unstructured",
				"git": map[string]interface{}{
					"repo":   g.getGitRepository(clusterName),
					"branch": "main",
					"dir":    fmt.Sprintf("clusters/%s/slices/%s", clusterName, intent.Name),
					"auth":   "token",
					"secretRef": map[string]interface{}{
						"name": "git-creds",
					},
				},
				"override": g.generateClusterOverrides(clusterName, intent),
			},
		},
	}

	// Create RepoSync for namespace-scoped sync
	repoSync := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "configsync.gke.io/v1beta1",
			"kind":       "RepoSync",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("slice-%s-namespace-sync", intent.Name),
				"namespace": fmt.Sprintf("slice-%s", intent.Name),
			},
			"spec": map[string]interface{}{
				"sourceFormat": "unstructured",
				"git": map[string]interface{}{
					"repo":   g.getGitRepository(clusterName),
					"branch": "main",
					"dir":    fmt.Sprintf("namespaces/slice-%s", intent.Name),
					"auth":   "token",
					"secretRef": map[string]interface{}{
						"name": "git-creds",
					},
				},
			},
		},
	}

	resources := []unstructured.Unstructured{*rootSync, *repoSync}

	pkg := &Package{
		Metadata: PackageMetadata{
			Name:        fmt.Sprintf("configsync-%s-%s", clusterName, intent.Name),
			Version:     "v1.0.0",
			Description: fmt.Sprintf("ConfigSync package for slice %s on cluster %s", intent.Name, clusterName),
			Category:    "config-sync",
			Labels: map[string]string{
				"slice-intent":   intent.Name,
				"target-cluster": clusterName,
				"generated-by":   "oran-mano-nephio-adapter",
			},
		},
		Resources: resources,
		Targets: []DeploymentTarget{
			{
				ClusterName: clusterName,
				Namespace:   "config-management-system",
				CloudType:   g.getClusterCloudType(clusterName),
			},
		},
	}

	return pkg, nil
}

// Helper methods

func (g *DefaultPackageGenerator) generateNetworkConfig(vnfType string, qos QoSProfile) NetworkConfig {
	config := NetworkConfig{
		Interfaces: []NetworkInterface{
			{
				Name: "management",
				Type: "management",
				CIDR: "10.0.0.0/24",
				MTU:  1500,
			},
		},
		QoSPolicies: []QoSPolicy{
			{
				Name:      fmt.Sprintf("%s-qos", strings.ToLower(vnfType)),
				Class:     qos.SliceType,
				Bandwidth: qos.Bandwidth,
				Priority:  g.getQoSPriority(qos.SliceType),
			},
		},
	}

	// Add VNF-specific interfaces
	switch vnfType {
	case "gNB":
		config.Interfaces = append(config.Interfaces,
			NetworkInterface{Name: "n2", Type: "signaling", CIDR: "192.168.1.0/24"},
			NetworkInterface{Name: "n3", Type: "data", CIDR: "192.168.2.0/24"},
		)
	case "AMF":
		config.Interfaces = append(config.Interfaces,
			NetworkInterface{Name: "n2", Type: "signaling", CIDR: "192.168.1.0/24"},
			NetworkInterface{Name: "n11", Type: "signaling", CIDR: "192.168.3.0/24"},
		)
	case "UPF":
		config.Interfaces = append(config.Interfaces,
			NetworkInterface{Name: "n3", Type: "data", CIDR: "192.168.2.0/24"},
			NetworkInterface{Name: "n4", Type: "signaling", CIDR: "192.168.4.0/24"},
			NetworkInterface{Name: "n6", Type: "data", CIDR: "192.168.5.0/24"},
		)
	}

	return config
}

func (g *DefaultPackageGenerator) generateKustomization(vnfType, intentName string) *Kustomization {
	return &Kustomization{
		NamePrefix: fmt.Sprintf("%s-%s-", strings.ToLower(vnfType), intentName),
		CommonLabels: map[string]string{
			"app.kubernetes.io/name":      strings.ToLower(vnfType),
			"app.kubernetes.io/instance":  intentName,
			"app.kubernetes.io/component": "network-function",
			"slice-intent":                intentName,
		},
		Images: []Image{
			{
				Name:   strings.ToLower(vnfType),
				NewTag: "latest",
			},
		},
	}
}

func (g *DefaultPackageGenerator) generateSliceMonitoringConfig(intent *NetworkSliceIntent) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "monitoring.coreos.com/v1",
			"kind":       "ServiceMonitor",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("slice-%s-monitor", intent.Name),
				"namespace": fmt.Sprintf("slice-%s", intent.Name),
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"slice-intent": intent.Name,
					},
				},
				"endpoints": []interface{}{
					map[string]interface{}{
						"port":     "metrics",
						"interval": "30s",
						"path":     "/metrics",
					},
				},
			},
		},
	}
}

func (g *DefaultPackageGenerator) generateSliceNetworkPolicy(intent *NetworkSliceIntent) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "NetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("slice-%s-network-policy", intent.Name),
				"namespace": fmt.Sprintf("slice-%s", intent.Name),
			},
			"spec": map[string]interface{}{
				"podSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"slice-intent": intent.Name,
					},
				},
				"policyTypes": []string{"Ingress", "Egress"},
				"ingress": []interface{}{
					map[string]interface{}{
						"from": []interface{}{
							map[string]interface{}{
								"namespaceSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"name": fmt.Sprintf("slice-%s", intent.Name),
									},
								},
							},
						},
					},
				},
				"egress": []interface{}{
					map[string]interface{}{
						"to": []interface{}{
							map[string]interface{}{
								"namespaceSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"name": fmt.Sprintf("slice-%s", intent.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (g *DefaultPackageGenerator) generateClusterOverrides(clusterName string, intent *NetworkSliceIntent) map[string]interface{} {
	return map[string]interface{}{
		"resources": []interface{}{
			map[string]interface{}{
				"group": "apps",
				"kind":  "Deployment",
				"operations": []interface{}{
					map[string]interface{}{
						"operation": "replace",
						"path":      "/spec/template/spec/nodeSelector",
						"value": map[string]interface{}{
							"kubernetes.io/arch": "amd64",
							"cluster-name":       clusterName,
						},
					},
				},
			},
		},
	}
}

func (g *DefaultPackageGenerator) getTargetCluster(placement PlacementSpec) string {
	// Logic to determine target cluster based on placement constraints
	if placement.SiteID != "" {
		return fmt.Sprintf("cluster-%s", placement.SiteID)
	}
	return fmt.Sprintf("cluster-%s", placement.CloudType)
}

func (g *DefaultPackageGenerator) getGitRepository(clusterName string) string {
	return fmt.Sprintf("https://github.com/thc1006/nephio-deployments-%s", clusterName)
}

func (g *DefaultPackageGenerator) getClusterCloudType(clusterName string) string {
	// Logic to determine cloud type from cluster name
	if strings.Contains(clusterName, "edge") {
		return "edge"
	}
	if strings.Contains(clusterName, "regional") {
		return "regional"
	}
	return "central"
}

func (g *DefaultPackageGenerator) getQoSPriority(sliceType string) int {
	switch sliceType {
	case "uRLLC":
		return 1 // Highest priority
	case "eMBB":
		return 2
	case "mIoT":
		return 3
	default:
		return 4 // Lowest priority
	}
}

func parseFloat64(s string) float64 {
	// Parse bandwidth/latency strings like "4.5Mbps", "10ms"
	// Simplified implementation - in production, use proper parsing
	s = strings.Replace(s, "Mbps", "", -1)
	s = strings.Replace(s, "ms", "", -1)
	// Return default values for now
	return 5.0
}