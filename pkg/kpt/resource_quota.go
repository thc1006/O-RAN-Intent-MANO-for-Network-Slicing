package kpt

import (
	"context"
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceQuotaFunction applies resource quotas to containers
type ResourceQuotaFunction struct{}

// NewResourceQuotaFunction creates a new ResourceQuotaFunction
func NewResourceQuotaFunction() *ResourceQuotaFunction {
	return &ResourceQuotaFunction{}
}

// Run executes the resource quota function
func (f *ResourceQuotaFunction) Run(ctx context.Context, input *FunctionInput) (*FunctionOutput, error) {
	if input == nil || input.ResourceList == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Extract quota configuration
	var quotas, limits map[string]string
	if input.ResourceList.FunctionConfig != nil {
		quotasNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("quotas"))
		if quotasNode != nil {
			quotas = extractResourceMap(quotasNode)
		}

		limitsNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("limits"))
		if limitsNode != nil {
			limits = extractResourceMap(limitsNode)
		}
	}

	// Apply quotas to deployments
	for _, resource := range input.ResourceList.Items {
		if err := f.applyQuotaToResource(resource, quotas, limits); err != nil {
			return nil, err
		}
	}

	return &FunctionOutput{
		ResourceList: input.ResourceList,
	}, nil
}

func (f *ResourceQuotaFunction) applyQuotaToResource(resource *yaml.RNode, quotas, limits map[string]string) error {
	kind, _ := resource.Pipe(yaml.Lookup("kind"))
	if kind == nil || kind.YNode().Value != "Deployment" {
		return nil
	}

	// Find containers path
	spec, err := resource.Pipe(yaml.Lookup("spec", "template", "spec"))
	if err != nil || spec == nil {
		return nil
	}

	// Get or create containers array
	containers, err := spec.Pipe(yaml.Lookup("containers"))
	if err != nil || containers == nil {
		return nil
	}

	// Get the list of container elements
	elements, err := containers.Elements()
	if err != nil {
		return nil
	}

	// Apply resources to each container
	for _, container := range elements {
		// Create resources structure
		resources := yaml.NewMapRNode(nil)

		if len(quotas) > 0 {
			requests := yaml.NewMapRNode(nil)
			for key, value := range quotas {
				requests.PipeE(yaml.SetField(key, yaml.NewScalarRNode(value)))
			}
			resources.PipeE(yaml.SetField("requests", requests))
		}

		if len(limits) > 0 {
			limitsNode := yaml.NewMapRNode(nil)
			for key, value := range limits {
				limitsNode.PipeE(yaml.SetField(key, yaml.NewScalarRNode(value)))
			}
			resources.PipeE(yaml.SetField("limits", limitsNode))
		}

		// Set resources on container
		container.PipeE(yaml.SetField("resources", resources))
	}

	return nil
}

func extractResourceMap(node *yaml.RNode) map[string]string {
	result := make(map[string]string)
	if node == nil || node.YNode() == nil {
		return result
	}

	content := node.YNode().Content
	for i := 0; i < len(content)-1; i += 2 {
		key := content[i].Value
		value := content[i+1].Value
		result[key] = value
	}

	return result
}
