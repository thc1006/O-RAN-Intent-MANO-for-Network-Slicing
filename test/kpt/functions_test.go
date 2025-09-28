package kpt_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/kpt"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestSetNamespaceFunction verifies the set-namespace KPT function
func TestSetNamespaceFunction(t *testing.T) {
	t.Run("Set namespace on all resources", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewSetNamespaceFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{
					yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
`),
					yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
`),
				},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: SetNamespace
metadata:
  name: set-namespace
namespace: network-slices
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.ResourceList.Items, 2)

		// Check that namespace was set on all resources
		for _, resource := range output.ResourceList.Items {
			namespace, err := resource.Pipe(yaml.Lookup("metadata", "namespace"))
			require.NoError(t, err)
			assert.Equal(t, "network-slices", namespace.YNode().Value)
		}
	})

	t.Run("Skip namespace-scoped resources like Namespace itself", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewSetNamespaceFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{
					yaml.MustParse(`
apiVersion: v1
kind: Namespace
metadata:
  name: test-namespace
`),
					yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
`),
				},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: SetNamespace
metadata:
  name: set-namespace
namespace: network-slices
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.Len(t, output.ResourceList.Items, 2)

		// Check Namespace resource doesn't have namespace field
		namespaceResource := output.ResourceList.Items[0]
		namespace, _ := namespaceResource.Pipe(yaml.Lookup("metadata", "namespace"))
		assert.Nil(t, namespace)

		// Check ConfigMap has namespace set
		configMapResource := output.ResourceList.Items[1]
		namespace, _ = configMapResource.Pipe(yaml.Lookup("metadata", "namespace"))
		assert.Equal(t, "network-slices", namespace.YNode().Value)
	})
}

// TestResourceQuotaFunction verifies resource quota application
func TestResourceQuotaFunction(t *testing.T) {
	t.Run("Apply resource quotas to deployments", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewResourceQuotaFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{
					yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec:
      containers:
      - name: app
        image: nginx:latest
`),
				},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: ResourceQuota
metadata:
  name: apply-quotas
quotas:
  cpu: "100m"
  memory: "128Mi"
limits:
  cpu: "500m"
  memory: "512Mi"
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)

		// Check resources were applied
		deployment := output.ResourceList.Items[0]

		// Get containers array
		containers, err := deployment.Pipe(yaml.Lookup("spec", "template", "spec", "containers"))
		require.NoError(t, err)
		require.NotNil(t, containers)

		// Get the first container from the array
		elements, err := containers.Elements()
		require.NoError(t, err)
		require.NotEmpty(t, elements)

		container := elements[0]

		// Check if resources field exists on container
		resources, err := container.Pipe(yaml.Lookup("resources"))
		require.NoError(t, err)
		assert.NotNil(t, resources, "Resources should be set on container")

		if resources != nil {
			// Check requests
			requests, _ := resources.Pipe(yaml.Lookup("requests"))
			assert.NotNil(t, requests, "Requests should be set")

			// Check limits
			limits, _ := resources.Pipe(yaml.Lookup("limits"))
			assert.NotNil(t, limits, "Limits should be set")
		}
	})
}

// TestNetworkPolicyFunction verifies network policy generation
func TestNetworkPolicyFunction(t *testing.T) {
	t.Run("Generate network policies for services", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewNetworkPolicyFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{
					yaml.MustParse(`
apiVersion: v1
kind: Service
metadata:
  name: test-service
  labels:
    app: test
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 8080
`),
				},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: NetworkPolicyConfig
metadata:
  name: generate-policies
mode: strict
allowedNamespaces:
  - network-slices
  - monitoring
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		// Should have original service plus generated NetworkPolicy
		assert.Len(t, output.ResourceList.Items, 2)

		// Find the NetworkPolicy
		var networkPolicy *yaml.RNode
		for _, item := range output.ResourceList.Items {
			kind, _ := item.Pipe(yaml.Lookup("kind"))
			if kind != nil && kind.YNode().Value == "NetworkPolicy" {
				networkPolicy = item
				break
			}
		}

		assert.NotNil(t, networkPolicy, "NetworkPolicy should be generated")
	})
}

// TestAutoScalingFunction verifies HPA generation
func TestAutoScalingFunction(t *testing.T) {
	t.Run("Generate HPA for deployments", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewAutoScalingFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{
					yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 3
`),
				},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: AutoScalingConfig
metadata:
  name: auto-scale
minReplicas: 2
maxReplicas: 10
targetCPUUtilizationPercentage: 70
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		// Should have original deployment plus generated HPA
		assert.Len(t, output.ResourceList.Items, 2)

		// Find the HPA
		var hpa *yaml.RNode
		for _, item := range output.ResourceList.Items {
			kind, _ := item.Pipe(yaml.Lookup("kind"))
			if kind != nil && kind.YNode().Value == "HorizontalPodAutoscaler" {
				hpa = item
				break
			}
		}

		assert.NotNil(t, hpa, "HPA should be generated")
	})
}

// TestFunctionChaining verifies multiple functions can be chained
func TestFunctionChaining(t *testing.T) {
	t.Run("Chain multiple KPT functions", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		pipeline := kpt.NewFunctionPipeline()

		// Add functions to pipeline
		pipeline.AddFunction(kpt.NewSetNamespaceFunction())
		pipeline.AddFunction(kpt.NewResourceQuotaFunction())
		pipeline.AddFunction(kpt.NewNetworkPolicyFunction())

		input := &kpt.PipelineInput{
			Resources: []*yaml.RNode{
				yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec:
      containers:
      - name: app
        image: nginx:latest
`),
				yaml.MustParse(`
apiVersion: v1
kind: Service
metadata:
  name: test-app-svc
spec:
  selector:
    app: test-app
  ports:
  - port: 80
    targetPort: 8080
`),
			},
			Configs: map[string]*yaml.RNode{
				"namespace": yaml.MustParse(`namespace: network-slices`),
				"quotas": yaml.MustParse(`
quotas:
  cpu: "100m"
  memory: "128Mi"
`),
				"network": yaml.MustParse(`
mode: strict
allowedNamespaces:
  - network-slices
`),
			},
		}

		// Act
		output, err := pipeline.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		// Should have deployment, service, plus generated NetworkPolicy
		assert.GreaterOrEqual(t, len(output.Resources), 3)

		// Verify namespace was set
		deployment := output.Resources[0]
		namespace, _ := deployment.Pipe(yaml.Lookup("metadata", "namespace"))
		assert.Equal(t, "network-slices", namespace.YNode().Value)

		// Verify resources were set
		containers, _ := deployment.Pipe(yaml.Lookup("spec", "template", "spec", "containers"))
		if containers != nil {
			elements, _ := containers.Elements()
			if len(elements) > 0 {
				resources, _ := elements[0].Pipe(yaml.Lookup("resources"))
				assert.NotNil(t, resources)
				if resources != nil {
					requests, _ := resources.Pipe(yaml.Lookup("requests"))
					assert.NotNil(t, requests)
				}
			}
		}
	})
}

// TestApplyQoSPolicyFunction verifies QoS policy application for network slices
func TestApplyQoSPolicyFunction(t *testing.T) {
	t.Run("Apply QoS policies to eMBB slice", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewApplyQoSPolicyFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{
					yaml.MustParse(`
apiVersion: ran.o-ran.org/v1alpha1
kind: NetworkSlice
metadata:
  name: embb-slice
spec:
  sliceType: eMBB
  sites:
  - siteID: "site-001"
    cells: ["cell-001", "cell-002"]
`),
				},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: QoSPolicy
metadata:
  name: embb-qos
policies:
  eMBB:
    bandwidth: 1000
    latency: 20
    reliability: 99.99
    priority: high
  URLLC:
    bandwidth: 100
    latency: 1
    reliability: 99.999
    priority: critical
  mIoT:
    bandwidth: 10
    latency: 1000
    reliability: 99.9
    priority: normal
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)

		// Check QoS was applied to the slice
		slice := output.ResourceList.Items[0]
		qos, _ := slice.Pipe(yaml.Lookup("spec", "qos"))
		assert.NotNil(t, qos)

		bandwidth, _ := slice.Pipe(yaml.Lookup("spec", "qos", "bandwidth"))
		assert.Equal(t, "1000", bandwidth.YNode().Value)

		latency, _ := slice.Pipe(yaml.Lookup("spec", "qos", "latency"))
		assert.Equal(t, "20", latency.YNode().Value)
	})

	t.Run("Apply QoS policies to URLLC slice", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewApplyQoSPolicyFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{
					yaml.MustParse(`
apiVersion: ran.o-ran.org/v1alpha1
kind: NetworkSlice
metadata:
  name: urllc-slice
spec:
  sliceType: URLLC
`),
				},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: QoSPolicy
metadata:
  name: urllc-qos
policies:
  URLLC:
    bandwidth: 100
    latency: 1
    reliability: 99.999
    priority: critical
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)

		slice := output.ResourceList.Items[0]
		latency, _ := slice.Pipe(yaml.Lookup("spec", "qos", "latency"))
		assert.Equal(t, "1", latency.YNode().Value)

		priority, _ := slice.Pipe(yaml.Lookup("spec", "qos", "priority"))
		assert.Equal(t, "critical", priority.YNode().Value)
	})
}

// TestGenerateNetworkSliceFunction verifies complete network slice generation
func TestGenerateNetworkSliceFunction(t *testing.T) {
	t.Run("Generate complete eMBB network slice configuration", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewGenerateNetworkSliceFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: NetworkSliceGenerator
metadata:
  name: generate-embb
sliceSpec:
  name: embb-slice-001
  type: eMBB
  tenant: enterprise-customer
  sites:
  - siteID: "site-001"
    cells: ["cell-001", "cell-002", "cell-003"]
  - siteID: "site-002"
    cells: ["cell-004", "cell-005"]
  qos:
    bandwidth: 1000
    latency: 20
    reliability: 99.99
  networkFunctions:
  - type: AMF
    replicas: 2
  - type: SMF
    replicas: 2
  - type: UPF
    replicas: 3
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		// Should generate multiple resources: NetworkSlice, Deployments for NFs, Services, ConfigMaps
		assert.GreaterOrEqual(t, len(output.ResourceList.Items), 5)

		// Find the NetworkSlice resource
		var networkSlice *yaml.RNode
		for _, item := range output.ResourceList.Items {
			kind, _ := item.Pipe(yaml.Lookup("kind"))
			if kind != nil && kind.YNode().Value == "NetworkSlice" {
				networkSlice = item
				break
			}
		}

		require.NotNil(t, networkSlice, "NetworkSlice should be generated")

		// Verify slice properties
		sliceType, _ := networkSlice.Pipe(yaml.Lookup("spec", "sliceType"))
		assert.Equal(t, "eMBB", sliceType.YNode().Value)

		// Count generated deployments
		deploymentCount := 0
		for _, item := range output.ResourceList.Items {
			kind, _ := item.Pipe(yaml.Lookup("kind"))
			if kind != nil && kind.YNode().Value == "Deployment" {
				deploymentCount++
			}
		}
		assert.Equal(t, 3, deploymentCount, "Should generate deployments for AMF, SMF, UPF")
	})

	t.Run("Generate URLLC slice with edge computing", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		function := kpt.NewGenerateNetworkSliceFunction()

		input := &kpt.FunctionInput{
			ResourceList: &kpt.ResourceList{
				Items: []*yaml.RNode{},
				FunctionConfig: yaml.MustParse(`
apiVersion: fn.kpt.dev/v1alpha1
kind: NetworkSliceGenerator
metadata:
  name: generate-urllc
sliceSpec:
  name: urllc-slice-001
  type: URLLC
  tenant: industrial-iot
  sites:
  - siteID: "edge-site-001"
    cells: ["cell-001"]
  qos:
    bandwidth: 100
    latency: 1
    reliability: 99.999
  edgeComputing:
    enabled: true
    mec:
      location: "edge-site-001"
      resources:
        cpu: "16"
        memory: "32Gi"
        gpu: "2"
`),
			},
		}

		// Act
		output, err := function.Run(ctx, input)

		// Assert
		require.NoError(t, err)

		// Find MEC deployment
		var mecDeployment *yaml.RNode
		for _, item := range output.ResourceList.Items {
			kind, _ := item.Pipe(yaml.Lookup("kind"))
			name, _ := item.Pipe(yaml.Lookup("metadata", "name"))
			if kind != nil && kind.YNode().Value == "Deployment" &&
				name != nil && name.YNode().Value == "mec-urllc-slice-001" {
				mecDeployment = item
				break
			}
		}

		assert.NotNil(t, mecDeployment, "MEC deployment should be generated for URLLC")
	})
}
