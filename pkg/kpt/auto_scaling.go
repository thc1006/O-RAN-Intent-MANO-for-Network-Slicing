package kpt

import (
	"context"
	"fmt"
	"strconv"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// AutoScalingFunction generates HPA resources for deployments
type AutoScalingFunction struct{}

// NewAutoScalingFunction creates a new AutoScalingFunction
func NewAutoScalingFunction() *AutoScalingFunction {
	return &AutoScalingFunction{}
}

// Run executes the auto-scaling function
func (f *AutoScalingFunction) Run(ctx context.Context, input *FunctionInput) (*FunctionOutput, error) {
	if input == nil || input.ResourceList == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Extract configuration
	var minReplicas, maxReplicas, targetCPU int

	if input.ResourceList.FunctionConfig != nil {
		minNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("minReplicas"))
		if minNode != nil {
			minReplicas, _ = strconv.Atoi(minNode.YNode().Value)
		}

		maxNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("maxReplicas"))
		if maxNode != nil {
			maxReplicas, _ = strconv.Atoi(maxNode.YNode().Value)
		}

		targetNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("targetCPUUtilizationPercentage"))
		if targetNode != nil {
			targetCPU, _ = strconv.Atoi(targetNode.YNode().Value)
		}
	}

	if minReplicas == 0 {
		minReplicas = 1
	}
	if maxReplicas == 0 {
		maxReplicas = 10
	}
	if targetCPU == 0 {
		targetCPU = 80
	}

	// Generate HPAs for deployments
	newResources := []*yaml.RNode{}
	for _, resource := range input.ResourceList.Items {
		newResources = append(newResources, resource)

		kind, _ := resource.Pipe(yaml.Lookup("kind"))
		if kind != nil && kind.YNode().Value == "Deployment" {
			hpa := f.generateHPA(resource, minReplicas, maxReplicas, targetCPU)
			if hpa != nil {
				newResources = append(newResources, hpa)
			}
		}
	}

	return &FunctionOutput{
		ResourceList: &ResourceList{
			Items:          newResources,
			FunctionConfig: input.ResourceList.FunctionConfig,
		},
	}, nil
}

func (f *AutoScalingFunction) generateHPA(deployment *yaml.RNode, minReplicas, maxReplicas, targetCPU int) *yaml.RNode {
	name, _ := deployment.Pipe(yaml.Lookup("metadata", "name"))
	if name == nil {
		return nil
	}

	namespace, _ := deployment.Pipe(yaml.Lookup("metadata", "namespace"))
	nsValue := "default"
	if namespace != nil {
		nsValue = namespace.YNode().Value
	}

	hpaYAML := fmt.Sprintf(`
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s-hpa
  namespace: %s
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: %s
  minReplicas: %d
  maxReplicas: %d
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: %d
`, name.YNode().Value, nsValue, name.YNode().Value, minReplicas, maxReplicas, targetCPU)

	return yaml.MustParse(hpaYAML)
}
