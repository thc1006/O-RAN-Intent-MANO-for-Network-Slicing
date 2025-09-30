package slices

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// SliceConfiguration holds the configuration for a network slice
type SliceConfiguration struct {
	SliceID   string
	SliceType string // eMBB, URLLC, mMTC, mMTC-RedCap
	Namespace string
	QoS       QoSProfile
}

// QoSProfile defines Quality of Service parameters
type QoSProfile struct {
	Throughput  int     // Mbps
	Latency     int     // milliseconds
	Reliability float64 // percentage (0-100)
	Connections int     // for mMTC (optional)
}

// Validate checks if QoS profile is valid
func (q *QoSProfile) Validate() error {
	if q.Throughput < 0 {
		return fmt.Errorf("throughput cannot be negative")
	}
	if q.Latency <= 0 {
		return fmt.Errorf("latency must be positive")
	}
	if q.Reliability < 0 || q.Reliability > 100 {
		return fmt.Errorf("reliability must be between 0 and 100")
	}
	return nil
}

// SliceManifests contains all Kubernetes manifests for a slice
type SliceManifests struct {
	Namespace     *corev1.Namespace
	QoSConfigMap  *corev1.ConfigMap
	Deployment    *appsv1.Deployment
	Service       *corev1.Service
	Resources     []interface{} // All resources combined
}

// ToYAML serializes all manifests to YAML
func (sm *SliceManifests) ToYAML() ([]byte, error) {
	var allYAML []string

	// Serialize each resource
	resources := []interface{}{
		sm.Namespace,
		sm.QoSConfigMap,
		sm.Deployment,
		sm.Service,
	}

	for _, res := range resources {
		if res == nil {
			continue
		}
		yamlBytes, err := yaml.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource: %w", err)
		}
		allYAML = append(allYAML, string(yamlBytes))
	}

	// Join with YAML document separator
	combined := strings.Join(allYAML, "---\n")
	return []byte(combined), nil
}

// GenerateSliceManifests generates Kubernetes manifests for a network slice
func GenerateSliceManifests(config SliceConfiguration) (*SliceManifests, error) {
	// Validate slice type
	validTypes := map[string]bool{
		"eMBB":        true,
		"URLLC":       true,
		"mMTC":        true,
		"mMTC-RedCap": true,
	}
	if !validTypes[config.SliceType] {
		return nil, fmt.Errorf("invalid slice type: %s", config.SliceType)
	}

	// Validate QoS
	if err := config.QoS.Validate(); err != nil {
		return nil, fmt.Errorf("invalid QoS profile: %w", err)
	}

	manifests := &SliceManifests{}

	// Generate Namespace
	manifests.Namespace = generateNamespace(config)

	// Generate QoS ConfigMap
	manifests.QoSConfigMap = generateQoSConfigMap(config)

	// Generate Deployment
	manifests.Deployment = generateDeployment(config)

	// Generate Service
	manifests.Service = generateService(config)

	// Populate Resources slice
	manifests.Resources = []interface{}{
		manifests.Namespace,
		manifests.QoSConfigMap,
		manifests.Deployment,
		manifests.Service,
	}

	return manifests, nil
}

// generateNamespace creates a Namespace manifest
func generateNamespace(config SliceConfiguration) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Namespace,
			Labels: map[string]string{
				"slice-id":   config.SliceID,
				"slice-type": strings.ToLower(config.SliceType),
				"managed-by": "oran-orchestrator",
			},
		},
	}
}

// generateQoSConfigMap creates a ConfigMap with QoS configuration
func generateQoSConfigMap(config SliceConfiguration) *corev1.ConfigMap {
	qosYAML := fmt.Sprintf(`qos_profile:
  slice_id: %s
  slice_type: %s
  parameters:
    throughput: %d  # Mbps
    latency: %d     # ms
    reliability: %.2f  # percentage
`,
		config.SliceID,
		config.SliceType,
		config.QoS.Throughput,
		config.QoS.Latency,
		config.QoS.Reliability,
	)

	if config.QoS.Connections > 0 {
		qosYAML += fmt.Sprintf("    connections: %d  # for mMTC\n", config.QoS.Connections)
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-qos", config.SliceID),
			Namespace: config.Namespace,
			Labels: map[string]string{
				"slice-id": config.SliceID,
				"type":     "qos-config",
			},
		},
		Data: map[string]string{
			"qos.yaml": qosYAML,
		},
	}
}

// getResourceRequirements returns resource requirements based on slice type
func getResourceRequirements(sliceType string) corev1.ResourceRequirements {
	switch sliceType {
	case "eMBB":
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2000m"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4000m"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		}
	case "URLLC":
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4000m"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4000m"), // Guaranteed
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		}
	case "mMTC", "mMTC-RedCap":
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		}
	default:
		// Default resources
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		}
	}
}

// generateDeployment creates a Deployment manifest
func generateDeployment(config SliceConfiguration) *appsv1.Deployment {
	replicas := int32(1)
	if config.SliceType == "URLLC" {
		replicas = 2 // High availability for URLLC
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SliceID,
			Namespace: config.Namespace,
			Labels: map[string]string{
				"slice-id":   config.SliceID,
				"slice-type": strings.ToLower(config.SliceType),
				"component":  "slice-controller",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"slice-id": config.SliceID,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"slice-id":   config.SliceID,
						"slice-type": strings.ToLower(config.SliceType),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  fmt.Sprintf("slice-%s", strings.ToLower(config.SliceType)),
							Image: fmt.Sprintf("oran/slice-%s:latest", strings.ToLower(config.SliceType)),
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 9090,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "SLICE_ID",
									Value: config.SliceID,
								},
								{
									Name:  "SLICE_TYPE",
									Value: config.SliceType,
								},
								{
									Name:  "QOS_THROUGHPUT",
									Value: fmt.Sprintf("%d", config.QoS.Throughput),
								},
								{
									Name:  "QOS_LATENCY",
									Value: fmt.Sprintf("%d", config.QoS.Latency),
								},
							},
							Resources: getResourceRequirements(config.SliceType),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "qos-config",
									MountPath: "/etc/oran/qos",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "qos-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-qos", config.SliceID),
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

// generateService creates a Service manifest
func generateService(config SliceConfiguration) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-svc", config.SliceID),
			Namespace: config.Namespace,
			Labels: map[string]string{
				"slice-id": config.SliceID,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"slice-id": config.SliceID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     9090,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}