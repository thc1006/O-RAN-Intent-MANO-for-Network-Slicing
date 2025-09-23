package translator

import (
	"fmt"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
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
	case manov1alpha1.VNFTypeRAN:
		pkg.Resources = t.generateRANResources(vnf)
	case manov1alpha1.VNFTypeCN:
		pkg.Resources = t.generateCNResources(vnf)
	case manov1alpha1.VNFTypeTN:
		pkg.Resources = t.generateTNResources(vnf)
	case manov1alpha1.VNFTypeUPF:
		pkg.Resources = t.generateUPFResources(vnf)
	case manov1alpha1.VNFTypeAMF:
		pkg.Resources = t.generateAMFResources(vnf)
	case manov1alpha1.VNFTypeSMF:
		pkg.Resources = t.generateSMFResources(vnf)
	case manov1alpha1.VNFTypePCF:
		pkg.Resources = t.generatePCFResources(vnf)
	case manov1alpha1.VNFTypeUDM:
		pkg.Resources = t.generateUDMResources(vnf)
	case manov1alpha1.VNFTypeAUSF:
		pkg.Resources = t.generateAUSFResources(vnf)
	case manov1alpha1.VNFTypeNSSF:
		pkg.Resources = t.generateNSSFResources(vnf)
	case manov1alpha1.VNFTypeNEF:
		pkg.Resources = t.generateNEFResources(vnf)
	case manov1alpha1.VNFTypeNRF:
		pkg.Resources = t.generateNRFResources(vnf)
	case manov1alpha1.VNFTypegNB:
		pkg.Resources = t.generateGNBResources(vnf)
	case manov1alpha1.VNFTypeCU:
		pkg.Resources = t.generateCUResources(vnf)
	case manov1alpha1.VNFTypeDU:
		pkg.Resources = t.generateDUResources(vnf)
	case manov1alpha1.VNFTypeRU:
		pkg.Resources = t.generateRUResources(vnf)
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
							"image": fmt.Sprintf("%s:%s", vnf.Spec.Image.Repository, vnf.Spec.Image.Tag),
							"resources": t.generateResourceRequirements(vnf),
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
	qosConfig := t.generateQoSConfigMap(vnf)
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
							"image": fmt.Sprintf("%s:%s", vnf.Spec.Image.Repository, vnf.Spec.Image.Tag),
							"resources": t.generateResourceRequirements(vnf),
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
							"image": fmt.Sprintf("%s:%s", vnf.Spec.Image.Repository, vnf.Spec.Image.Tag),
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

// generateUPFResources creates resources for UPF VNF
func (t *PorchTranslator) generateUPFResources(vnf *manov1alpha1.VNF) []Resource {
	resources := []Resource{}

	// UPF StatefulSet
	statefulset := Resource{
		APIVersion: "apps/v1",
		Kind:       "StatefulSet",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-upf", vnf.Name),
			"namespace": vnf.Namespace,
			"labels": map[string]interface{}{
				"vnf-type": "UPF",
				"vnf-name": vnf.Name,
			},
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
							"image": fmt.Sprintf("%s:%s", vnf.Spec.Image.Repository, vnf.Spec.Image.Tag),
							"resources": t.generateResourceRequirements(vnf),
							"env": t.generateEnvVars(vnf),
							"ports": []map[string]interface{}{
								{
									"name":          "n4",
									"containerPort": 8805,
									"protocol":      "UDP",
								},
								{
									"name":          "n3",
									"containerPort": 2152,
									"protocol":      "UDP",
								},
								},
						},
					},
					"nodeSelector": t.generateNodeSelector(vnf),
				},
			},
		},
	}
	resources = append(resources, statefulset)

	// UPF Service
	service := Resource{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-upf-svc", vnf.Name),
			"namespace": vnf.Namespace,
		},
		Spec: map[string]interface{}{
			"selector": map[string]interface{}{
				"app": fmt.Sprintf("%s-upf", vnf.Name),
			},
			"ports": []map[string]interface{}{
				{
					"name":       "n4",
					"port":       8805,
					"targetPort": 8805,
					"protocol":   "UDP",
				},
				{
					"name":       "n3",
					"port":       2152,
					"targetPort": 2152,
					"protocol":   "UDP",
				},
			},
		},
	}
	resources = append(resources, service)

	// QoS ConfigMap
	qosConfig := t.generateQoSConfigMap(vnf)
	resources = append(resources, qosConfig)

	return resources
}

// generateAMFResources creates resources for AMF VNF
func (t *PorchTranslator) generateAMFResources(vnf *manov1alpha1.VNF) []Resource {
	resources := []Resource{}

	// AMF Deployment
	deployment := Resource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-amf", vnf.Name),
			"namespace": vnf.Namespace,
			"labels": map[string]interface{}{
				"vnf-type": "AMF",
				"vnf-name": vnf.Name,
			},
		},
		Spec: map[string]interface{}{
			"replicas": 1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": fmt.Sprintf("%s-amf", vnf.Name),
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": fmt.Sprintf("%s-amf", vnf.Name),
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "amf",
							"image": fmt.Sprintf("%s:%s", vnf.Spec.Image.Repository, vnf.Spec.Image.Tag),
							"resources": t.generateResourceRequirements(vnf),
							"env": t.generateEnvVars(vnf),
							"ports": []map[string]interface{}{
								{
									"name":          "n1-n2",
									"containerPort": 38412,
									"protocol":      "SCTP",
								},
								{
									"name":          "sbi",
									"containerPort": 8080,
									"protocol":      "TCP",
								},
								},
						},
					},
					"nodeSelector": t.generateNodeSelector(vnf),
				},
			},
		},
	}
	resources = append(resources, deployment)

	// AMF Service
	service := Resource{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-amf-svc", vnf.Name),
			"namespace": vnf.Namespace,
		},
		Spec: map[string]interface{}{
			"selector": map[string]interface{}{
				"app": fmt.Sprintf("%s-amf", vnf.Name),
			},
			"ports": []map[string]interface{}{
				{
					"name":       "n1-n2",
					"port":       38412,
					"targetPort": 38412,
					"protocol":   "SCTP",
				},
				{
					"name":       "sbi",
					"port":       8080,
					"targetPort": 8080,
					"protocol":   "TCP",
				},
			},
		},
	}
	resources = append(resources, service)

	// QoS ConfigMap
	qosConfig := t.generateQoSConfigMap(vnf)
	resources = append(resources, qosConfig)

	return resources
}

// Simplified generators for other VNF types
func (t *PorchTranslator) generateSMFResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "smf", []map[string]interface{}{
		{"name": "sbi", "port": 8080, "protocol": "TCP"},
		{"name": "n4", "port": 8805, "protocol": "UDP"},
	})
}

func (t *PorchTranslator) generatePCFResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "pcf", []map[string]interface{}{
		{"name": "sbi", "port": 8080, "protocol": "TCP"},
	})
}

func (t *PorchTranslator) generateUDMResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "udm", []map[string]interface{}{
		{"name": "sbi", "port": 8080, "protocol": "TCP"},
	})
}

func (t *PorchTranslator) generateAUSFResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "ausf", []map[string]interface{}{
		{"name": "sbi", "port": 8080, "protocol": "TCP"},
	})
}

func (t *PorchTranslator) generateNSSFResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "nssf", []map[string]interface{}{
		{"name": "sbi", "port": 8080, "protocol": "TCP"},
	})
}

func (t *PorchTranslator) generateNEFResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "nef", []map[string]interface{}{
		{"name": "sbi", "port": 8080, "protocol": "TCP"},
	})
}

func (t *PorchTranslator) generateNRFResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "nrf", []map[string]interface{}{
		{"name": "sbi", "port": 8080, "protocol": "TCP"},
	})
}

func (t *PorchTranslator) generateGNBResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "gnb", []map[string]interface{}{
		{"name": "n2", "port": 38412, "protocol": "SCTP"},
		{"name": "n3", "port": 2152, "protocol": "UDP"},
	})
}

func (t *PorchTranslator) generateCUResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "cu", []map[string]interface{}{
		{"name": "f1", "port": 38470, "protocol": "SCTP"},
	})
}

func (t *PorchTranslator) generateDUResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "du", []map[string]interface{}{
		{"name": "f1", "port": 38470, "protocol": "SCTP"},
	})
}

func (t *PorchTranslator) generateRUResources(vnf *manov1alpha1.VNF) []Resource {
	return t.generateGenericNFResources(vnf, "ru", []map[string]interface{}{
		{"name": "fronthaul", "port": 7777, "protocol": "UDP"},
	})
}

// generateGenericNFResources creates resources for generic network functions
func (t *PorchTranslator) generateGenericNFResources(vnf *manov1alpha1.VNF, nfType string, ports []map[string]interface{}) []Resource {
	resources := []Resource{}

	// Generic Deployment
	deployment := Resource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-%s", vnf.Name, nfType),
			"namespace": vnf.Namespace,
			"labels": map[string]interface{}{
				"vnf-type": string(vnf.Spec.Type),
				"vnf-name": vnf.Name,
			},
		},
		Spec: map[string]interface{}{
			"replicas": 1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": fmt.Sprintf("%s-%s", vnf.Name, nfType),
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": fmt.Sprintf("%s-%s", vnf.Name, nfType),
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  nfType,
							"image": fmt.Sprintf("%s:%s", vnf.Spec.Image.Repository, vnf.Spec.Image.Tag),
							"resources": t.generateResourceRequirements(vnf),
							"env": t.generateEnvVars(vnf),
							"ports": t.generateContainerPorts(ports),
						},
					},
					"nodeSelector": t.generateNodeSelector(vnf),
				},
			},
		},
	}
	resources = append(resources, deployment)

	// Generic Service
	service := Resource{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-%s-svc", vnf.Name, nfType),
			"namespace": vnf.Namespace,
		},
		Spec: map[string]interface{}{
			"selector": map[string]interface{}{
				"app": fmt.Sprintf("%s-%s", vnf.Name, nfType),
			},
			"ports": t.generateServicePorts(ports),
		},
	}
	resources = append(resources, service)

	// QoS ConfigMap
	qosConfig := t.generateQoSConfigMap(vnf)
	resources = append(resources, qosConfig)

	return resources
}

// generateQoSConfigMap creates a QoS ConfigMap for any VNF
func (t *PorchTranslator) generateQoSConfigMap(vnf *manov1alpha1.VNF) Resource {
	return Resource{
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
				"sliceType": vnf.Spec.QoS.SliceType,
			},
		},
	}
}

// generateResourceRequirements creates resource requirements
func (t *PorchTranslator) generateResourceRequirements(vnf *manov1alpha1.VNF) map[string]interface{} {
	resources := map[string]interface{}{
		"requests": map[string]interface{}{},
		"limits":   map[string]interface{}{},
	}

	// Use new CPUCores and MemoryGB fields if available
	if vnf.Spec.Resources.CPUCores > 0 {
		resources["requests"].(map[string]interface{})["cpu"] = fmt.Sprintf("%dm", vnf.Spec.Resources.CPUCores*1000)
		resources["limits"].(map[string]interface{})["cpu"] = fmt.Sprintf("%dm", vnf.Spec.Resources.CPUCores*1000)
	} else if vnf.Spec.Resources.CPU != "" {
		// Fallback to legacy CPU field
		resources["requests"].(map[string]interface{})["cpu"] = vnf.Spec.Resources.CPU
		resources["limits"].(map[string]interface{})["cpu"] = vnf.Spec.Resources.CPU
	}

	if vnf.Spec.Resources.MemoryGB > 0 {
		resources["requests"].(map[string]interface{})["memory"] = fmt.Sprintf("%dGi", vnf.Spec.Resources.MemoryGB)
		resources["limits"].(map[string]interface{})["memory"] = fmt.Sprintf("%dGi", vnf.Spec.Resources.MemoryGB)
	} else if vnf.Spec.Resources.Memory != "" {
		// Fallback to legacy Memory field
		resources["requests"].(map[string]interface{})["memory"] = vnf.Spec.Resources.Memory
		resources["limits"].(map[string]interface{})["memory"] = vnf.Spec.Resources.Memory
	}

	return resources
}

// generateContainerPorts creates container port configurations
func (t *PorchTranslator) generateContainerPorts(ports []map[string]interface{}) []map[string]interface{} {
	containerPorts := []map[string]interface{}{}
	for _, port := range ports {
		containerPorts = append(containerPorts, map[string]interface{}{
			"name":          port["name"],
			"containerPort": port["port"],
			"protocol":      port["protocol"],
		})
	}
	return containerPorts
}

// generateServicePorts creates service port configurations
func (t *PorchTranslator) generateServicePorts(ports []map[string]interface{}) []map[string]interface{} {
	servicePorts := []map[string]interface{}{}
	for _, port := range ports {
		servicePorts = append(servicePorts, map[string]interface{}{
			"name":       port["name"],
			"port":       port["port"],
			"targetPort": port["port"],
			"protocol":   port["protocol"],
		})
	}
	return servicePorts
}
