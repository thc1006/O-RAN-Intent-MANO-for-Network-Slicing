package translator

import (
	"fmt"

	manov1alpha1 "github.com/o-ran/intent-mano/adapters/vnf-operator/api/v1alpha1"
)

// PorchPackage represents a Kpt package for Porch
type PorchPackage struct {
	Name         string
	Namespace    string
	Resources    []Resource
	Kptfile      *Kptfile
	Kustomization *Kustomization
}

// Resource represents a Kubernetes resource in the package
type Resource struct {
	APIVersion string
	Kind       string
	Metadata   map[string]interface{}
	Spec       map[string]interface{}
}

// Kptfile represents the Kpt package metadata
type Kptfile struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Info       map[string]interface{} `json:"info"`
	Pipeline   []Function            `json:"pipeline,omitempty"`
}

// Function represents a Kpt function in the pipeline
type Function struct {
	Image string                 `json:"image"`
	ConfigPath string          `json:"configPath,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// Kustomization for GitOps
type Kustomization struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Resources  []string `json:"resources"`
	Namespace  string   `json:"namespace,omitempty"`
}

// PorchTranslator translates VNF to Porch packages
type PorchTranslator struct{}

// NewPorchTranslator creates a new translator
func NewPorchTranslator() *PorchTranslator {
	return &PorchTranslator{}
}

// TranslateVNF converts a VNF CR to a Porch package
func (t *PorchTranslator) TranslateVNF(vnf *manov1alpha1.VNF) (*PorchPackage, error) {
	pkg := &PorchPackage{
		Name:      fmt.Sprintf("%s-%s", vnf.Name, vnf.Spec.Type),
		Namespace: vnf.Namespace,
		Resources: []Resource{},
	}

	// Create resources based on VNF type
	switch vnf.Spec.Type {
	case "RAN":
		pkg.Resources = t.generateRANResources(vnf)
	case "CN":
		pkg.Resources = t.generateCNResources(vnf)
	case "TN":
		pkg.Resources = t.generateTNResources(vnf)
	default:
		return nil, fmt.Errorf("unknown VNF type: %s", vnf.Spec.Type)
	}

	// Create Kptfile
	pkg.Kptfile = t.generateKptfile(vnf)

	// Create Kustomization
	pkg.Kustomization = t.generateKustomization(vnf, pkg.Resources)

	return pkg, nil
}

func (t *PorchTranslator) generateRANResources(vnf *manov1alpha1.VNF) []Resource {
	resources := []Resource{}

	// RAN Deployment
	deployment := Resource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-ran", vnf.Name),
			"namespace": vnf.Namespace,
			"labels": map[string]interface{}{
				"vnf-type": "RAN",
				"vnf-name": vnf.Name,
			},
		},
		Spec: map[string]interface{}{
			"replicas": 1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": fmt.Sprintf("%s-ran", vnf.Name),
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": fmt.Sprintf("%s-ran", vnf.Name),
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "ran",
							"image": fmt.Sprintf("oran/ran:%s", vnf.Spec.Version),
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    vnf.Spec.Resources.CPU,
									"memory": vnf.Spec.Resources.Memory,
								},
								"limits": map[string]interface{}{
									"cpu":    vnf.Spec.Resources.CPU,
									"memory": vnf.Spec.Resources.Memory,
								},
							},
							"env": t.generateEnvVars(vnf),
						},
					},
					"nodeSelector": t.generateNodeSelector(vnf),
				},
			},
		},
	}
	resources = append(resources, deployment)

	// RAN Service
	service := Resource{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-ran-svc", vnf.Name),
			"namespace": vnf.Namespace,
		},
		Spec: map[string]interface{}{
			"selector": map[string]interface{}{
				"app": fmt.Sprintf("%s-ran", vnf.Name),
			},
			"ports": []map[string]interface{}{
				{
					"name":       "sctp",
					"port":       38412,
					"targetPort": 38412,
					"protocol":   "SCTP",
				},
				{
					"name":       "udp",
					"port":       2152,
					"targetPort": 2152,
					"protocol":   "UDP",
				},
			},
		},
	}
	resources = append(resources, service)

	// QoS ConfigMap
	qosConfig := Resource{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-qos-config", vnf.Name),
			"namespace": vnf.Namespace,
		},
		Spec: map[string]interface{}{
			"data": map[string]interface{}{
				"bandwidth": fmt.Sprintf("%.2f", vnf.Spec.QoS.Bandwidth),
				"latency":   fmt.Sprintf("%.2f", vnf.Spec.QoS.Latency),
				"jitter":    getJitterString(vnf.Spec.QoS.Jitter),
			},
		},
	}
	resources = append(resources, qosConfig)

	return resources
}

func (t *PorchTranslator) generateCNResources(vnf *manov1alpha1.VNF) []Resource {
	resources := []Resource{}

	// Core Network StatefulSet (for UPF)
	statefulset := Resource{
		APIVersion: "apps/v1",
		Kind:       "StatefulSet",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-upf", vnf.Name),
			"namespace": vnf.Namespace,
		},
		Spec: map[string]interface{}{
			"serviceName": fmt.Sprintf("%s-upf", vnf.Name),
			"replicas":    1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": fmt.Sprintf("%s-upf", vnf.Name),
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": fmt.Sprintf("%s-upf", vnf.Name),
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "upf",
							"image": fmt.Sprintf("oran/upf:%s", vnf.Spec.Version),
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    vnf.Spec.Resources.CPU,
									"memory": vnf.Spec.Resources.Memory,
								},
							},
							"env": t.generateEnvVars(vnf),
						},
					},
					"nodeSelector": t.generateNodeSelector(vnf),
				},
			},
		},
	}
	resources = append(resources, statefulset)

	return resources
}

func (t *PorchTranslator) generateTNResources(vnf *manov1alpha1.VNF) []Resource {
	resources := []Resource{}

	// Transport Network DaemonSet
	daemonset := Resource{
		APIVersion: "apps/v1",
		Kind:       "DaemonSet",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-tn-agent", vnf.Name),
			"namespace": vnf.Namespace,
		},
		Spec: map[string]interface{}{
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": fmt.Sprintf("%s-tn-agent", vnf.Name),
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": fmt.Sprintf("%s-tn-agent", vnf.Name),
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "tn-agent",
							"image": fmt.Sprintf("oran/tn-agent:%s", vnf.Spec.Version),
							"securityContext": map[string]interface{}{
								"privileged": true,
							},
							"env": []map[string]interface{}{
								{
									"name":  "BANDWIDTH_MBPS",
									"value": fmt.Sprintf("%.2f", vnf.Spec.QoS.Bandwidth),
								},
								{
									"name":  "MAX_LATENCY_MS",
									"value": fmt.Sprintf("%.2f", vnf.Spec.QoS.Latency),
								},
							},
						},
					},
					"hostNetwork": true,
				},
			},
		},
	}
	resources = append(resources, daemonset)

	return resources
}

func (t *PorchTranslator) generateKptfile(vnf *manov1alpha1.VNF) *Kptfile {
	return &Kptfile{
		APIVersion: "kpt.dev/v1",
		Kind:       "Kptfile",
		Metadata: map[string]interface{}{
			"name": fmt.Sprintf("%s-%s-package", vnf.Name, vnf.Spec.Type),
		},
		Info: map[string]interface{}{
			"description": fmt.Sprintf("VNF package for %s type %s", vnf.Name, vnf.Spec.Type),
			"keywords": []string{
				"oran",
				"vnf",
				string(vnf.Spec.Type),
			},
		},
		Pipeline: []Function{
			{
				Image: "gcr.io/kpt-fn/apply-setters:v0.2",
				Config: map[string]interface{}{
					"vnf-name":    vnf.Name,
					"vnf-type":    string(vnf.Spec.Type),
					"vnf-version": vnf.Spec.Version,
				},
			},
			{
				Image: "gcr.io/kpt-fn/kubeval:v0.3",
			},
		},
	}
}

func (t *PorchTranslator) generateKustomization(vnf *manov1alpha1.VNF, resources []Resource) *Kustomization {
	resourceFiles := []string{}
	for i := range resources {
		resourceFiles = append(resourceFiles, fmt.Sprintf("%s-%d.yaml", resources[i].Kind, i))
	}

	return &Kustomization{
		APIVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources:  resourceFiles,
		Namespace:  vnf.Namespace,
	}
}

func (t *PorchTranslator) generateEnvVars(vnf *manov1alpha1.VNF) []map[string]interface{} {
	envVars := []map[string]interface{}{
		{
			"name":  "VNF_NAME",
			"value": vnf.Name,
		},
		{
			"name":  "VNF_TYPE",
			"value": string(vnf.Spec.Type),
		},
	}

	// Add custom config data as env vars
	for key, value := range vnf.Spec.ConfigData {
		envVars = append(envVars, map[string]interface{}{
			"name":  key,
			"value": value,
		})
	}

	return envVars
}

func (t *PorchTranslator) generateNodeSelector(vnf *manov1alpha1.VNF) map[string]interface{} {
	selector := map[string]interface{}{}

	// Add cloud type selector
	if vnf.Spec.Placement.CloudType != "" {
		selector["cloud-type"] = vnf.Spec.Placement.CloudType
	}

	// Add zone selector if specified
	if len(vnf.Spec.Placement.PreferredZones) > 0 {
		selector["zone"] = vnf.Spec.Placement.PreferredZones[0]
	}

	return selector
}
func getJitterString(jitter *float64) string {
	if jitter == nil {
		return "0.0"
	}
	return fmt.Sprintf("%.2f", *jitter)
}
