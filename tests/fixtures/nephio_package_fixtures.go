package fixtures

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/mocks"
)

// VNF Package Templates
func ValidCUCPPackage() *mocks.NephioPackage {
	return &mocks.NephioPackage{
		Name:       "cucp-embb-package",
		Namespace:  "oran-system",
		Repository: "nephio-blueprints",
		Revision:   "v1.0.0",
		Lifecycle:  mocks.PackageLifecycleDraft,
		Kptfile: &mocks.Kptfile{
			APIVersion: "kpt.dev/v1",
			Kind:       "Kptfile",
			Metadata: mocks.KptMetadata{
				Name: "cucp-embb-package",
				Labels: map[string]string{
					"nephio.org/package-type": "vnf",
					"nephio.org/vnf-type":     "cucp",
					"nephio.org/slice-type":   "eMBB",
				},
				Annotations: map[string]string{
					"nephio.org/description": "CU-CP VNF package for eMBB slice",
					"nephio.org/version":     "1.0.0",
				},
			},
			Info: mocks.KptInfo{
				Description: "Central Unit Control Plane for eMBB network slice",
				Site:        "edge-zone-a",
				Keywords:    []string{"5G", "O-RAN", "CU-CP", "eMBB"},
			},
			Pipeline: mocks.KptPipeline{
				Mutators: []mocks.Function{
					{
						Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
						ConfigMap: map[string]interface{}{
							"namespace": "cucp-embb",
						},
					},
					{
						Image: "gcr.io/kpt-fn/apply-replacements:v0.1.1",
						ConfigPath: "replacements.yaml",
					},
				},
				Validators: []mocks.Function{
					{
						Image: "gcr.io/kpt-fn/kubeval:v0.3.0",
					},
				},
			},
			Inventory: mocks.KptInventory{
				Namespace:   "cucp-embb",
				Name:        "cucp-embb-inventory",
				InventoryID: "cucp-embb-package",
			},
			Upstream: mocks.KptUpstream{
				Type: "git",
				Git: mocks.GitRef{
					Repo:      "https://github.com/nephio-project/blueprints.git",
					Directory: "vnf-packages/cucp",
					Ref:       "v1.0.0",
				},
			},
		},
		Resources: []runtime.Object{
			createCUCPDeployment(),
			createCUCPService(),
			createCUCPConfigMap(),
		},
		Functions: []mocks.Function{
			{
				Image: "gcr.io/nephio/cucp-controller:v1.0.0",
				ConfigMap: map[string]interface{}{
					"qos-profile": map[string]interface{}{
						"latency":     "10ms",
						"throughput":  "1Gbps",
						"reliability": "99.99%",
					},
				},
			},
		},
		Conditions: []mocks.PackageCondition{
			{
				Type:    "Ready",
				Status:  "True",
				Reason:  "PackageValid",
				Message: "Package is valid and ready for deployment",
			},
		},
		Metadata: map[string]string{
			"vnf-type":        "cucp",
			"slice-type":      "eMBB",
			"resource-requirements": "cpu:2000m,memory:4Gi",
			"placement-policy":      "edge-preferred",
		},
	}
}

func ValidCUUPPackage() *mocks.NephioPackage {
	pkg := ValidCUCPPackage()
	pkg.Name = "cuup-embb-package"
	pkg.Kptfile.Metadata.Name = "cuup-embb-package"
	pkg.Kptfile.Metadata.Labels["nephio.org/vnf-type"] = "cuup"
	pkg.Kptfile.Info.Description = "Central Unit User Plane for eMBB network slice"
	pkg.Metadata["vnf-type"] = "cuup"

	// Replace resources with CUUP-specific ones
	pkg.Resources = []runtime.Object{
		createCUUPDeployment(),
		createCUUPService(),
		createCUUPConfigMap(),
	}

	return pkg
}

func ValidDUPackage() *mocks.NephioPackage {
	pkg := ValidCUCPPackage()
	pkg.Name = "du-embb-package"
	pkg.Kptfile.Metadata.Name = "du-embb-package"
	pkg.Kptfile.Metadata.Labels["nephio.org/vnf-type"] = "du"
	pkg.Kptfile.Info.Description = "Distributed Unit for eMBB network slice"
	pkg.Metadata["vnf-type"] = "du"

	// Replace resources with DU-specific ones
	pkg.Resources = []runtime.Object{
		createDUDeployment(),
		createDUService(),
		createDUConfigMap(),
	}

	return pkg
}

func URLLCPackage() *mocks.NephioPackage {
	pkg := ValidCUCPPackage()
	pkg.Name = "cucp-urllc-package"
	pkg.Kptfile.Metadata.Name = "cucp-urllc-package"
	pkg.Kptfile.Metadata.Labels["nephio.org/slice-type"] = "URLLC"
	pkg.Kptfile.Info.Description = "CU-CP for URLLC low-latency slice"
	pkg.Metadata["slice-type"] = "URLLC"

	// Update QoS profile for URLLC
	pkg.Functions[0].ConfigMap["qos-profile"] = map[string]interface{}{
		"latency":     "1ms",
		"throughput":  "100Mbps",
		"reliability": "99.999%",
	}

	return pkg
}

func mMTCPackage() *mocks.NephioPackage {
	pkg := ValidCUCPPackage()
	pkg.Name = "cucp-mmtc-package"
	pkg.Kptfile.Metadata.Name = "cucp-mmtc-package"
	pkg.Kptfile.Metadata.Labels["nephio.org/slice-type"] = "mMTC"
	pkg.Kptfile.Info.Description = "CU-CP for mMTC massive connectivity slice"
	pkg.Metadata["slice-type"] = "mMTC"

	// Update QoS profile for mMTC
	pkg.Functions[0].ConfigMap["qos-profile"] = map[string]interface{}{
		"latency":     "100ms",
		"throughput":  "10Mbps",
		"reliability": "99.9%",
		"device-density": "1000000", // devices per km²
	}

	return pkg
}

func InvalidPackage() *mocks.NephioPackage {
	return &mocks.NephioPackage{
		Name:      "", // Invalid: empty name
		Namespace: "",
		Kptfile: &mocks.Kptfile{
			APIVersion: "invalid/v1", // Invalid API version
			Kind:       "InvalidKind",
			Metadata: mocks.KptMetadata{
				Name: "", // Invalid: empty name
			},
		},
		Resources: []runtime.Object{
			// Invalid resource without required fields
			&corev1.ConfigMap{},
		},
	}
}

func PackageWithMissingKptfile() *mocks.NephioPackage {
	return &mocks.NephioPackage{
		Name:      "missing-kptfile-package",
		Namespace: "default",
		Kptfile:   nil, // Missing Kptfile
		Resources: []runtime.Object{
			createCUCPDeployment(),
		},
	}
}

func PackageWithInvalidFunctions() *mocks.NephioPackage {
	pkg := ValidCUCPPackage()
	pkg.Functions = []mocks.Function{
		{
			Image: "", // Invalid: empty image
			ConfigMap: map[string]interface{}{
				"invalid-config": nil,
			},
		},
	}
	return pkg
}

// Helper functions to create Kubernetes resources
func createCUCPDeployment() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cucp-deployment",
			Namespace: "cucp-embb",
			Labels: map[string]string{
				"app.kubernetes.io/name":     "cucp",
				"app.kubernetes.io/instance": "cucp-embb",
				"nephio.org/vnf-type":       "cucp",
			},
		},
		Data: map[string]string{
			"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cucp
  namespace: cucp-embb
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cucp
  template:
    metadata:
      labels:
        app: cucp
    spec:
      containers:
      - name: cucp
        image: registry.k8s.io/cucp:v1.0.0
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 2000m
            memory: 4Gi
          limits:
            cpu: 4000m
            memory: 8Gi
`,
		},
	}
}

func createCUCPService() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cucp-service",
			Namespace: "cucp-embb",
		},
		Data: map[string]string{
			"service.yaml": `
apiVersion: v1
kind: Service
metadata:
  name: cucp-service
  namespace: cucp-embb
spec:
  selector:
    app: cucp
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
`,
		},
	}
}

func createCUCPConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cucp-config",
			Namespace: "cucp-embb",
		},
		Data: map[string]string{
			"config.yaml": `
amf:
  endpoint: "http://amf.5gc:8080"
smf:
  endpoint: "http://smf.5gc:8080"
qos:
  latency: "10ms"
  throughput: "1Gbps"
  reliability: "99.99%"
`,
		},
	}
}

func createCUUPDeployment() *corev1.ConfigMap {
	cm := createCUCPDeployment()
	cm.Name = "cuup-deployment"
	cm.Data["deployment.yaml"] = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cuup
  namespace: cuup-embb
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cuup
  template:
    metadata:
      labels:
        app: cuup
    spec:
      containers:
      - name: cuup
        image: registry.k8s.io/cuup:v1.0.0
        ports:
        - containerPort: 8081
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 4Gi
`
	return cm
}

func createCUUPService() *corev1.ConfigMap {
	cm := createCUCPService()
	cm.Name = "cuup-service"
	cm.Data["service.yaml"] = `
apiVersion: v1
kind: Service
metadata:
  name: cuup-service
  namespace: cuup-embb
spec:
  selector:
    app: cuup
  ports:
  - port: 8081
    targetPort: 8081
  type: ClusterIP
`
	return cm
}

func createCUUPConfigMap() *corev1.ConfigMap {
	cm := createCUCPConfigMap()
	cm.Name = "cuup-config"
	cm.Data["config.yaml"] = `
cucp:
  endpoint: "http://cucp-service.cucp-embb:8080"
qos:
  latency: "5ms"
  throughput: "10Gbps"
  jitter: "1ms"
`
	return cm
}

func createDUDeployment() *corev1.ConfigMap {
	cm := createCUCPDeployment()
	cm.Name = "du-deployment"
	cm.Data["deployment.yaml"] = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: du
  namespace: du-embb
spec:
  replicas: 1
  selector:
    matchLabels:
      app: du
  template:
    metadata:
      labels:
        app: du
    spec:
      containers:
      - name: du
        image: registry.k8s.io/du:v1.0.0
        ports:
        - containerPort: 8082
        resources:
          requests:
            cpu: 4000m
            memory: 8Gi
          limits:
            cpu: 8000m
            memory: 16Gi
`
	return cm
}

func createDUService() *corev1.ConfigMap {
	cm := createCUCPService()
	cm.Name = "du-service"
	cm.Data["service.yaml"] = `
apiVersion: v1
kind: Service
metadata:
  name: du-service
  namespace: du-embb
spec:
  selector:
    app: du
  ports:
  - port: 8082
    targetPort: 8082
  type: ClusterIP
`
	return cm
}

func createDUConfigMap() *corev1.ConfigMap {
	cm := createCUCPConfigMap()
	cm.Name = "du-config"
	cm.Data["config.yaml"] = `
cucp:
  endpoint: "http://cucp-service.cucp-embb:8080"
cuup:
  endpoint: "http://cuup-service.cuup-embb:8081"
radio:
  frequency: "3.5GHz"
  bandwidth: "100MHz"
  antenna_count: 64
`
	return cm
}